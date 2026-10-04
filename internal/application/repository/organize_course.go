package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateCourse inserts a course together with its ordered lessons in one
// transaction. The lesson body itself is never written here: each lesson only
// carries output_id, which the caller has already persisted as an
// organize_outputs row.
//
// The course ID is assigned by the model hook, so lessons built before the call
// may carry an empty CourseID; those are backfilled here rather than leaving
// orphan rows behind.
func (r *organizeRepository) CreateCourse(
	ctx context.Context,
	course *types.OrganizeCourse,
	lessons []*types.OrganizeCourseLesson,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(course).Error; err != nil {
			return err
		}
		if err := createCourseSharedSpaceRows(
			tx,
			course.ID,
			course.SharedSpaceIDs,
			course.UserID,
		); err != nil {
			return err
		}
		for _, lesson := range lessons {
			if lesson != nil && lesson.CourseID == "" {
				lesson.CourseID = course.ID
			}
		}
		if len(lessons) == 0 {
			course.LessonCount = 0
			return nil
		}
		if err := tx.Create(&lessons).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.OrganizeCourse{}).
			Where("id = ?", course.ID).
			Update("lesson_count", len(lessons)).Error; err != nil {
			return err
		}
		course.LessonCount = len(lessons)
		return nil
	})
}

// AppendCourseLesson adds one already-processed lesson to an existing course
// and increments its denormalized lesson count atomically.
func (r *organizeRepository) AppendCourseLesson(
	ctx context.Context,
	course *types.OrganizeCourse,
	lesson *types.OrganizeCourseLesson,
) error {
	if course == nil || lesson == nil {
		return errors.New("course and lesson are required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if lesson.CourseID == "" {
			lesson.CourseID = course.ID
		}
		if err := tx.Create(lesson).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		update := tx.Model(&types.OrganizeCourse{}).
			Where("tenant_id = ? AND id = ?", course.TenantID, course.ID).
			Updates(map[string]interface{}{
				"lesson_count": gorm.Expr("lesson_count + ?", 1),
				"updated_at":   now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		course.LessonCount++
		course.UpdatedAt = now
		return nil
	})
}

// UpdateCourseLesson changes the display title on both the lesson row and its
// backing output. The description is stored as the output summary so existing
// readers and the content-management view see the same value.
func (r *organizeRepository) UpdateCourseLesson(
	ctx context.Context,
	course *types.OrganizeCourse,
	lesson *types.OrganizeCourseLesson,
) error {
	if course == nil || lesson == nil {
		return errors.New("course and lesson are required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		lessonUpdate := tx.Model(&types.OrganizeCourseLesson{}).
			Where("tenant_id = ? AND course_id = ? AND id = ?", course.TenantID, course.ID, lesson.ID).
			Updates(map[string]interface{}{
				"title":      lesson.Title,
				"updated_at": now,
			})
		if lessonUpdate.Error != nil {
			return lessonUpdate.Error
		}
		if lessonUpdate.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if lesson.Output != nil && lesson.OutputID != "" {
			outputUpdate := tx.Model(&types.OrganizeOutput{}).
				Where("tenant_id = ? AND user_id = ? AND id = ?", course.TenantID, course.UserID, lesson.OutputID).
				Updates(map[string]interface{}{
					"title":          lesson.Output.Title,
					"source_summary": lesson.Output.SourceSummary,
					"updated_at":     now,
				})
			if outputUpdate.Error != nil {
				return outputUpdate.Error
			}
			if outputUpdate.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}

		return tx.Model(&types.OrganizeCourse{}).
			Where("tenant_id = ? AND id = ?", course.TenantID, course.ID).
			Update("updated_at", now).Error
	})
}

// DeleteCourseLesson removes one lesson and its backing output atomically.
// Remaining lessons are renumbered so the next append can safely use the
// course lesson count as its sort order.
func (r *organizeRepository) DeleteCourseLesson(
	ctx context.Context,
	course *types.OrganizeCourse,
	lesson *types.OrganizeCourseLesson,
) error {
	if course == nil || lesson == nil {
		return errors.New("course and lesson are required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lessonDelete := tx.
			Where("tenant_id = ? AND course_id = ? AND id = ?", course.TenantID, course.ID, lesson.ID).
			Delete(&types.OrganizeCourseLesson{})
		if lessonDelete.Error != nil {
			return lessonDelete.Error
		}
		if lessonDelete.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if lesson.OutputID != "" {
			if err := tx.Where("output_id = ?", lesson.OutputID).
				Delete(&types.OrganizeOutputMemory{}).Error; err != nil {
				return err
			}
			if err := tx.Where("tenant_id = ? AND user_id = ? AND id = ?", course.TenantID, course.UserID, lesson.OutputID).
				Delete(&types.OrganizeOutput{}).Error; err != nil {
				return err
			}
		}

		var remaining []*types.OrganizeCourseLesson
		if err := tx.Where("tenant_id = ? AND course_id = ?", course.TenantID, course.ID).
			Order("sort_order ASC").
			Order("created_at ASC").
			Find(&remaining).Error; err != nil {
			return err
		}
		for index, remainingLesson := range remaining {
			if remainingLesson == nil || remainingLesson.SortOrder == index {
				continue
			}
			if err := tx.Model(&types.OrganizeCourseLesson{}).
				Where("tenant_id = ? AND course_id = ? AND id = ?", course.TenantID, course.ID, remainingLesson.ID).
				Update("sort_order", index).Error; err != nil {
				return err
			}
		}

		now := time.Now().UTC()
		update := tx.Model(&types.OrganizeCourse{}).
			Where("tenant_id = ? AND id = ?", course.TenantID, course.ID).
			Updates(map[string]interface{}{
				"lesson_count": len(remaining),
				"updated_at":   now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		course.LessonCount = len(remaining)
		course.UpdatedAt = now
		return nil
	})
}

// GetCourse reads a single course by ID.
//
// There is deliberately no tenant parameter: course IDs are globally unique and
// both readers are platform-scoped (the admin console and the published-only
// discover page). Tenant scoping is enforced by the service layer, which
// re-checks publication for the public path.
func (r *organizeRepository) GetCourse(ctx context.Context, id string) (*types.OrganizeCourse, error) {
	var course types.OrganizeCourse
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&course).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *organizeRepository) UpdateCourse(ctx context.Context, course *types.OrganizeCourse) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeCourse{}).
		Where("tenant_id = ? AND id = ?", course.TenantID, course.ID).
		Select(
			"title", "summary", "category", "cover_url", "teacher_name", "teacher_title",
			"source", "public_status", "visibility_scope", "lesson_count", "learner_count", "updated_at",
		).
		Updates(course).Error
}

func (r *organizeRepository) UpdateCourseVisibility(
	ctx context.Context,
	id string,
	visibilityScope string,
	organizationIDs []string,
	createdBy string,
) (*types.OrganizeCourse, error) {
	var result types.OrganizeCourse
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var course types.OrganizeCourse
		if err := tx.Where("id = ?", id).First(&course).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&types.OrganizeCourse{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"visibility_scope": visibilityScope,
				"updated_at":       now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Where("course_id = ?", id).
			Delete(&types.OrganizeCourseSharedSpace{}).Error; err != nil {
			if !isMissingCourseSharedSpaceTable(err) || len(organizationIDs) > 0 {
				return err
			}
		}
		if err := createCourseSharedSpaceRows(tx, id, organizationIDs, createdBy); err != nil {
			return err
		}
		course.VisibilityScope = visibilityScope
		course.UpdatedAt = now
		course.SharedSpaceIDs = append([]string(nil), organizationIDs...)
		result = course
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplaceCourseSharedSpaces replaces the selected organization links in one
// transaction. Organization membership is validated by the service before
// this method is called; the repository only owns the relation rows.
func (r *organizeRepository) ReplaceCourseSharedSpaces(
	ctx context.Context,
	courseID string,
	organizationIDs []string,
	createdBy string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("course_id = ?", courseID).
			Delete(&types.OrganizeCourseSharedSpace{}).Error; err != nil {
			if !isMissingCourseSharedSpaceTable(err) {
				return err
			}
		}
		for _, organizationID := range organizationIDs {
			organizationID = strings.TrimSpace(organizationID)
			if organizationID == "" {
				continue
			}
			row := &types.OrganizeCourseSharedSpace{
				ID:             uuid.NewString(),
				CourseID:       courseID,
				OrganizationID: organizationID,
				CreatedBy:      strings.TrimSpace(createdBy),
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func createCourseSharedSpaceRows(
	tx *gorm.DB,
	courseID string,
	organizationIDs []string,
	createdBy string,
) error {
	for _, organizationID := range organizationIDs {
		organizationID = strings.TrimSpace(organizationID)
		if organizationID == "" {
			continue
		}
		row := &types.OrganizeCourseSharedSpace{
			ID:             uuid.NewString(),
			CourseID:       courseID,
			OrganizationID: organizationID,
			CreatedBy:      strings.TrimSpace(createdBy),
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *organizeRepository) ListCourseSharedSpaceIDs(
	ctx context.Context,
	courseID string,
) ([]string, error) {
	var rows []types.OrganizeCourseSharedSpace
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		if isMissingCourseSharedSpaceTable(err) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := strings.TrimSpace(row.OrganizationID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func isMissingCourseSharedSpaceTable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "organize_course_shared_spaces") &&
		(strings.Contains(message, "no such table") ||
			strings.Contains(message, "does not exist") ||
			strings.Contains(message, "undefined table"))
}

// UpdateCoursePublicStatus changes the course and all lesson bodies in one
// transaction. Lesson bodies remain public_status=draft because the course is
// their only public entry point.
func (r *organizeRepository) UpdateCoursePublicStatus(
	ctx context.Context,
	id, status string,
) (*types.OrganizeCourse, error) {
	var result types.OrganizeCourse
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var course types.OrganizeCourse
		if err := tx.Where("id = ?", id).First(&course).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&types.OrganizeCourse{}).
			Where("id = ? AND tenant_id = ?", course.ID, course.TenantID).
			Select("public_status", "updated_at").
			Updates(map[string]interface{}{
				"public_status": status,
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		var lessons []*types.OrganizeCourseLesson
		if err := tx.Where("tenant_id = ? AND course_id = ?", course.TenantID, course.ID).
			Order("sort_order ASC").
			Order("created_at ASC").
			Find(&lessons).Error; err != nil {
			return err
		}
		outputIDs := make([]string, 0, len(lessons))
		for _, lesson := range lessons {
			if lesson != nil && lesson.OutputID != "" {
				outputIDs = append(outputIDs, lesson.OutputID)
			}
		}
		if len(outputIDs) > 0 {
			outputStatus := types.OrganizeOutputStatusDraft
			switch status {
			case types.OrganizePublicContentStatusPublished:
				outputStatus = types.OrganizeOutputStatusReady
			case types.OrganizePublicContentStatusPendingReview:
				outputStatus = types.OrganizeOutputStatusReview
			case types.OrganizePublicContentStatusOffline, types.OrganizePublicContentStatusRejected:
				outputStatus = types.OrganizeOutputStatusArchived
			}
			if err := tx.Model(&types.OrganizeOutput{}).
				Where("tenant_id = ? AND id IN ?", course.TenantID, outputIDs).
				Updates(map[string]interface{}{
					"status":        outputStatus,
					"public_status": types.OrganizePublicContentStatusDraft,
					"published_at":  nil,
					"published_by":  "",
					"updated_at":    now,
				}).Error; err != nil {
				return err
			}
		}
		course.PublicStatus = status
		course.UpdatedAt = now
		course.Lessons = lessons
		result = course
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteCourse soft-deletes the course and hard-deletes its lesson rows, so the
// unique (output_id) index does not keep the underlying outputs locked out of a
// future course. The outputs and memory links are removed from the active
// application view in the same transaction.
func (r *organizeRepository) DeleteCourse(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lessons []*types.OrganizeCourseLesson
		if err := tx.Select("output_id").
			Where("tenant_id = ? AND course_id = ?", tenantID, id).
			Find(&lessons).Error; err != nil {
			return err
		}
		outputIDs := make([]string, 0, len(lessons))
		for _, lesson := range lessons {
			if lesson != nil && lesson.OutputID != "" {
				outputIDs = append(outputIDs, lesson.OutputID)
			}
		}
		if len(outputIDs) > 0 {
			if err := tx.Where("output_id IN ?", outputIDs).
				Delete(&types.OrganizeOutputMemory{}).Error; err != nil {
				return err
			}
			if err := tx.Where("tenant_id = ? AND id IN ?", tenantID, outputIDs).
				Delete(&types.OrganizeOutput{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("tenant_id = ? AND course_id = ?", tenantID, id).
			Delete(&types.OrganizeCourseLesson{}).Error; err != nil {
			return err
		}
		if err := tx.Where("course_id = ?", id).
			Delete(&types.OrganizeCourseSharedSpace{}).Error; err != nil {
			if !isMissingCourseSharedSpaceTable(err) {
				return err
			}
		}
		return tx.Where("tenant_id = ? AND id = ?", tenantID, id).
			Delete(&types.OrganizeCourse{}).Error
	})
}

func (r *organizeRepository) ListCourses(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) ([]*types.OrganizeCourse, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeCourse{})
	dbq = applyCourseVisibilityFilter(ctx, dbq, query)
	if query.TenantID > 0 {
		dbq = dbq.Where("tenant_id = ?", query.TenantID)
	}
	if query.UserID != "" {
		dbq = dbq.Where("user_id = ?", query.UserID)
	}
	if query.PublicStatus != "" {
		dbq = dbq.Where("public_status = ?", query.PublicStatus)
	}
	if query.Source != "" {
		dbq = dbq.Where("source = ?", query.Source)
	}
	if query.Category != "" {
		dbq = dbq.Where("category = ?", query.Category)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "summary", "category", "teacher_name")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dbq = dbq.Order("updated_at DESC").Order("created_at DESC")
	if query.PageSize > 0 {
		page := query.Page
		if page < 1 {
			page = 1
		}
		dbq = dbq.Limit(query.PageSize).Offset((page - 1) * query.PageSize)
	}

	courses := make([]*types.OrganizeCourse, 0)
	if err := dbq.Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

func (r *organizeRepository) GetCourseStats(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) (*types.OrganizeCourseStats, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeCourse{})
	dbq = applyCourseVisibilityFilter(ctx, dbq, query)
	if query.TenantID > 0 {
		dbq = dbq.Where("tenant_id = ?", query.TenantID)
	}
	if query.UserID != "" {
		dbq = dbq.Where("user_id = ?", query.UserID)
	}
	if query.PublicStatus != "" {
		dbq = dbq.Where("public_status = ?", query.PublicStatus)
	}
	if query.Source != "" {
		dbq = dbq.Where("source = ?", query.Source)
	}
	if query.Category != "" {
		dbq = dbq.Where("category = ?", query.Category)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "summary", "category", "teacher_name")

	var stats types.OrganizeCourseStats
	if err := dbq.Select(
		"COUNT(*) AS total, "+
			"COALESCE(SUM(CASE WHEN public_status = ? THEN 1 ELSE 0 END), 0) AS published, "+
			"COALESCE(SUM(CASE WHEN public_status = ? THEN 1 ELSE 0 END), 0) AS offline, "+
			"COALESCE(SUM(lesson_count), 0) AS lesson_total",
		types.OrganizePublicContentStatusPublished,
		types.OrganizePublicContentStatusOffline,
	).Scan(&stats).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func applyCourseVisibilityFilter(
	ctx context.Context,
	dbq *gorm.DB,
	query types.OrganizeCourseQuery,
) *gorm.DB {
	if !query.VisibilityFilter || types.IsSystemAdminFromContext(ctx) {
		return dbq
	}

	clauses := []string{"visibility_scope = ?"}
	args := []interface{}{types.OrganizeCourseVisibilitySystem}

	privateClauses := make([]string, 0, 2)
	privateArgs := make([]interface{}, 0, 2)
	if query.ViewerTenantID > 0 {
		privateClauses = append(privateClauses, "tenant_id = ?")
		privateArgs = append(privateArgs, query.ViewerTenantID)
	}
	if strings.TrimSpace(query.ViewerUserID) != "" {
		privateClauses = append(privateClauses, "user_id = ?")
		privateArgs = append(privateArgs, strings.TrimSpace(query.ViewerUserID))
	}
	if len(privateClauses) > 0 {
		clauses = append(
			clauses,
			"visibility_scope = ? AND ("+strings.Join(privateClauses, " OR ")+")",
		)
		args = append(args, types.OrganizeCourseVisibilityPrivate)
		args = append(args, privateArgs...)
	}
	if len(query.ViewerOrganizationIDs) > 0 {
		clauses = append(
			clauses,
			"visibility_scope = ? AND EXISTS ("+
				"SELECT 1 FROM organize_course_shared_spaces css "+
				"WHERE css.course_id = organize_courses.id "+
				"AND css.organization_id IN ?)",
		)
		args = append(args, types.OrganizeCourseVisibilitySharedSpace, query.ViewerOrganizationIDs)
	}

	return dbq.Where("("+strings.Join(clauses, " OR ")+")", args...)
}

// organizeCourseLessonOutputColumns is what an outline read needs from an
// output: identity plus availability plus storage metadata. Adding `content`
// back here would silently restore the multi-megabyte payload this deliberately
// avoids.
const organizeCourseLessonOutputColumns = "id, tenant_id, user_id, title, output_type, icon, " +
	"source_summary, status, public_status, public_content_type, series_id, series_title, series_order, metadata"

// GetCourseLesson reads one lesson together with its full output body.
//
// Use this for body reads only; outline reads go through
// ListCourseLessonsWithOutputs, which skips the body columns on purpose.
func (r *organizeRepository) GetCourseLesson(
	ctx context.Context,
	courseID, lessonID string,
) (*types.OrganizeCourseLesson, error) {
	var lesson types.OrganizeCourseLesson
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND id = ?", courseID, lessonID).
		First(&lesson).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lesson.OutputID == "" {
		return &lesson, nil
	}

	var output types.OrganizeOutput
	err = r.db.WithContext(ctx).Where("id = ?", lesson.OutputID).First(&output).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Keep the lesson: the caller still needs it to distinguish "no body"
		// from "no such lesson".
		return &lesson, nil
	}
	if err != nil {
		return nil, err
	}
	lesson.Output = &output
	return &lesson, nil
}

func (r *organizeRepository) ListCourseLessons(
	ctx context.Context,
	courseID string,
) ([]*types.OrganizeCourseLesson, error) {
	lessons := make([]*types.OrganizeCourseLesson, 0)
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order ASC").
		Order("created_at ASC").
		Find(&lessons).Error; err != nil {
		return nil, err
	}
	return lessons, nil
}

// ListCourseLessonsWithOutputs additionally hydrates each lesson's Output so
// callers can judge availability and read storage metadata. Outputs that were
// deleted fall back to a nil Output rather than an error.
//
// It deliberately does NOT load content/source_summary. A 14-lesson course
// carries several megabytes of markdown, and neither caller renders more than
// one chapter at a time — shipping every body through this path made a single
// course detail request ~16 MB. Body reads go through GetCourseLesson.
func (r *organizeRepository) ListCourseLessonsWithOutputs(
	ctx context.Context,
	courseID string,
) ([]*types.OrganizeCourseLesson, error) {
	lessons, err := r.ListCourseLessons(ctx, courseID)
	if err != nil || len(lessons) == 0 {
		return lessons, err
	}

	outputIDs := make([]string, 0, len(lessons))
	for _, lesson := range lessons {
		if lesson != nil && lesson.OutputID != "" {
			outputIDs = append(outputIDs, lesson.OutputID)
		}
	}
	if len(outputIDs) == 0 {
		return lessons, nil
	}

	outputs := make([]*types.OrganizeOutput, 0, len(outputIDs))
	if err := r.db.WithContext(ctx).
		Select(organizeCourseLessonOutputColumns).
		Where("id IN ?", outputIDs).
		Find(&outputs).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]*types.OrganizeOutput, len(outputs))
	for _, output := range outputs {
		if output != nil {
			byID[output.ID] = output
		}
	}
	for _, lesson := range lessons {
		if lesson == nil {
			continue
		}
		lesson.Output = byID[lesson.OutputID]
	}
	return lessons, nil
}
