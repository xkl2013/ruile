package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type serviceRepository struct {
	db *gorm.DB
}

func NewServiceRepository(db *gorm.DB) interfaces.ServiceRepository {
	return &serviceRepository{db: db}
}

func (r *serviceRepository) CreateWorkProfile(ctx context.Context, profile *types.UserWorkProfile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if profile.DefaultProfile {
			if err := clearDefaultWorkProfile(tx, profile.TenantID, profile.UserID, profile.ID); err != nil {
				return err
			}
		}
		return tx.Create(profile).Error
	})
}

func (r *serviceRepository) UpdateWorkProfile(ctx context.Context, profile *types.UserWorkProfile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if profile.DefaultProfile {
			if err := clearDefaultWorkProfile(tx, profile.TenantID, profile.UserID, profile.ID); err != nil {
				return err
			}
		}
		return tx.Model(&types.UserWorkProfile{}).
			Where("tenant_id = ? AND id = ?", profile.TenantID, profile.ID).
			Select("name", "role_type", "campus_scope", "course_scope", "memory_scope", "tone_preference", "default_profile", "enabled", "state", "updated_by", "updated_at").
			Updates(profile).Error
	})
}

func clearDefaultWorkProfile(tx *gorm.DB, tenantID uint64, userID, exceptID string) error {
	query := tx.Model(&types.UserWorkProfile{}).
		Where("tenant_id = ? AND user_id = ? AND default_profile = ?", tenantID, userID, true)
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	return query.Update("default_profile", false).Error
}

func (r *serviceRepository) GetWorkProfile(ctx context.Context, tenantID uint64, id string) (*types.UserWorkProfile, error) {
	var profile types.UserWorkProfile
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *serviceRepository) GetDefaultWorkProfile(ctx context.Context, tenantID uint64, userID string) (*types.UserWorkProfile, error) {
	var profile types.UserWorkProfile
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND default_profile = ? AND enabled = ? AND state = ?",
			tenantID, strings.TrimSpace(userID), true, true, types.ServiceWorkProfileStateEnabled).
		Order("updated_at DESC").
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *serviceRepository) ListWorkProfiles(ctx context.Context, tenantID uint64, userID string) ([]*types.UserWorkProfile, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID)
	if strings.TrimSpace(userID) != "" {
		query = query.Where("user_id = ?", strings.TrimSpace(userID))
	}
	var profiles []*types.UserWorkProfile
	err := query.Order("default_profile DESC").
		Order("updated_at DESC").
		Find(&profiles).Error
	return profiles, err
}

func (r *serviceRepository) ReplaceAgentSettings(ctx context.Context, tenantID uint64, profileID string, settings []*types.WorkProfileAgentSetting) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND profile_id = ?", tenantID, profileID).
			Delete(&types.WorkProfileAgentSetting{}).Error; err != nil {
			return err
		}
		if len(settings) == 0 {
			return nil
		}
		return tx.Create(&settings).Error
	})
}

func (r *serviceRepository) ListAgentSettings(ctx context.Context, tenantID uint64, profileID string, onlyEnabled bool) ([]*types.WorkProfileAgentSetting, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND profile_id = ?", tenantID, strings.TrimSpace(profileID))
	if onlyEnabled {
		query = query.Where("enabled = ?", true)
	}
	var settings []*types.WorkProfileAgentSetting
	err := query.Order("display_order ASC").
		Order("created_at ASC").
		Find(&settings).Error
	return settings, err
}

func (r *serviceRepository) FindSubjectByKey(ctx context.Context, tenantID uint64, ownerUserID, subjectKey string) (*types.ServiceSubject, error) {
	var subject types.ServiceSubject
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND owner_user_id = ? AND subject_key = ?", tenantID, ownerUserID, strings.TrimSpace(subjectKey)).
		First(&subject).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &subject, err
}

func (r *serviceRepository) GetSubject(ctx context.Context, tenantID uint64, ownerUserID, id string) (*types.ServiceSubject, error) {
	var subject types.ServiceSubject
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND owner_user_id = ? AND id = ?", tenantID, strings.TrimSpace(ownerUserID), strings.TrimSpace(id)).
		First(&subject).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &subject, err
}

func (r *serviceRepository) ListSubjects(ctx context.Context, query types.ServiceCustomerSpaceListQuery) ([]*types.ServiceSubject, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.ServiceSubject{}).
		Where("service_subjects.tenant_id = ? AND service_subjects.owner_user_id = ?", query.TenantID, strings.TrimSpace(query.UserID))
	dbq = applySubjectProfileFilter(dbq, query.ProfileID)
	dbq = applySubjectKeyword(dbq, query.Keyword)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var subjects []*types.ServiceSubject
	err := dbq.
		Order("service_subjects.updated_at DESC").
		Order("service_subjects.created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&subjects).Error
	return subjects, total, err
}

func applySubjectProfileFilter(dbq *gorm.DB, profileID string) *gorm.DB {
	profileID = strings.TrimSpace(profileID)
	if profileID != "" {
		return dbq.Where(`(
			EXISTS (
				SELECT 1 FROM agent_work_docs d
				WHERE d.tenant_id = service_subjects.tenant_id
					AND d.profile_id = ?
					AND d.subject_id = service_subjects.id
					AND d.doc_type = ?
					AND d.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM service_reminders sr
				WHERE sr.tenant_id = service_subjects.tenant_id
					AND sr.profile_id = ?
					AND sr.subject_id = service_subjects.id
					AND sr.deleted_at IS NULL
			)
		)`, profileID, types.AgentWorkDocTypeCustomerWorkspace, profileID)
	}
	return dbq.Where(`(
		EXISTS (
			SELECT 1 FROM agent_work_docs d
			WHERE d.tenant_id = service_subjects.tenant_id
				AND d.subject_id = service_subjects.id
				AND d.doc_type = ?
				AND d.deleted_at IS NULL
		)
		OR EXISTS (
			SELECT 1 FROM service_reminders sr
			WHERE sr.tenant_id = service_subjects.tenant_id
				AND sr.subject_id = service_subjects.id
				AND sr.deleted_at IS NULL
		)
	)`, types.AgentWorkDocTypeCustomerWorkspace)
}

func applySubjectKeyword(dbq *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return dbq
	}
	like := "%" + strings.ToLower(keyword) + "%"
	return dbq.Where(
		"LOWER(service_subjects.display_name) LIKE ? OR LOWER(service_subjects.student_name) LIKE ? OR LOWER(service_subjects.subject_key) LIKE ?",
		like, like, like,
	)
}

func (r *serviceRepository) UpsertSubject(ctx context.Context, subject *types.ServiceSubject) error {
	if subject.ID == "" {
		existing, err := r.FindSubjectByKey(ctx, subject.TenantID, subject.OwnerUserID, subject.SubjectKey)
		if err != nil {
			return err
		}
		if existing != nil {
			subject.ID = existing.ID
		}
	}
	if subject.ID == "" {
		return r.db.WithContext(ctx).Create(subject).Error
	}
	return r.db.WithContext(ctx).Model(&types.ServiceSubject{}).
		Where("tenant_id = ? AND id = ?", subject.TenantID, subject.ID).
		Select("display_name", "student_name", "relation", "aliases", "external_refs", "visibility_scope", "confidence", "updated_at").
		Updates(subject).Error
}

func (r *serviceRepository) UpsertWorkDocWithLinks(ctx context.Context, doc *types.AgentWorkDoc, links []*types.AgentWorkDocMemoryLink) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if doc.ID == "" {
			var existing types.AgentWorkDoc
			err := tx.Where("tenant_id = ? AND profile_id = ? AND subject_id = ? AND agent_domain = ? AND doc_path = ?",
				doc.TenantID, doc.ProfileID, doc.SubjectID, doc.AgentDomain, doc.DocPath).
				First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil {
				doc.ID = existing.ID
			}
		}
		if doc.ID == "" {
			if err := tx.Create(doc).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&types.AgentWorkDoc{}).
			Where("tenant_id = ? AND id = ?", doc.TenantID, doc.ID).
			Select("title", "content", "status", "source_memory_ids", "metadata", "updated_at").
			Updates(doc).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND doc_id = ?", doc.TenantID, doc.ID).
			Delete(&types.AgentWorkDocMemoryLink{}).Error; err != nil {
			return err
		}
		for _, link := range links {
			link.DocID = doc.ID
			link.DocPath = doc.DocPath
			link.TenantID = doc.TenantID
		}
		if len(links) == 0 {
			return nil
		}
		return tx.Create(&links).Error
	})
}

func (r *serviceRepository) ListWorkDocsBySubject(ctx context.Context, tenantID uint64, profileID, subjectID string) ([]*types.AgentWorkDoc, error) {
	var docs []*types.AgentWorkDoc
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND profile_id = ? AND subject_id = ?", tenantID, profileID, subjectID).
		Order("doc_path ASC").
		Find(&docs).Error
	return docs, err
}

func (r *serviceRepository) GetDailyReportDoc(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentWorkDoc, error) {
	var doc types.AgentWorkDoc
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND owner_user_id = ? AND id = ? AND agent_domain = ? AND doc_type = ?",
			tenantID, strings.TrimSpace(userID), strings.TrimSpace(id), types.ServiceAgentDomainDailyReview, types.AgentWorkDocTypeDailyReport).
		First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &doc, err
}

func (r *serviceRepository) ListDailyReportDocs(ctx context.Context, query types.ServiceDailyReportListQuery) ([]*types.AgentWorkDoc, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.AgentWorkDoc{}).
		Where("tenant_id = ? AND owner_user_id = ? AND agent_domain = ? AND doc_type = ?",
			query.TenantID, strings.TrimSpace(query.UserID), types.ServiceAgentDomainDailyReview, types.AgentWorkDocTypeDailyReport)
	if strings.TrimSpace(query.ProfileID) != "" {
		dbq = dbq.Where("profile_id = ?", strings.TrimSpace(query.ProfileID))
	}
	dbq = applyDailyReportDocRange(dbq, query.Range)
	dbq = applyServiceDocKeyword(dbq, query.Keyword)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var docs []*types.AgentWorkDoc
	err := dbq.
		Order("updated_at DESC").
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&docs).Error
	return docs, total, err
}

func applyDailyReportDocRange(dbq *gorm.DB, reportRange string) *gorm.DB {
	switch strings.TrimSpace(reportRange) {
	case types.ServiceDailyReportRangeDay:
		return dbq.Where("doc_path LIKE ? AND doc_path NOT LIKE ? AND doc_path NOT LIKE ?",
			"日报/%.md", "日报/week/%", "日报/month/%")
	case types.ServiceDailyReportRangeWeek:
		return dbq.Where("doc_path LIKE ?", "日报/week/%")
	case types.ServiceDailyReportRangeMonth:
		return dbq.Where("doc_path LIKE ?", "日报/month/%")
	default:
		return dbq
	}
}

func applyServiceDocKeyword(dbq *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return dbq
	}
	like := "%" + strings.ToLower(keyword) + "%"
	return dbq.Where(
		"LOWER(title) LIKE ? OR LOWER(content) LIKE ? OR LOWER(doc_path) LIKE ?",
		like, like, like,
	)
}

func (r *serviceRepository) UpsertReminder(ctx context.Context, reminder *types.ServiceReminder) error {
	if reminder.ID == "" {
		return errors.New("service reminder id is required")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "summary", "priority", "due_at", "due_text", "stage", "channel", "decision_role",
			"risk_label", "assist_reason", "primary_action", "next_action", "avoid_action", "context_items",
			"memory_signals", "source_memory_ids", "source_memory_count", "last_memory_at", "confidence",
			"sales_highlights", "write_back_draft", "reply_draft", "metadata", "updated_at",
		}),
	}).Create(reminder).Error
}

func (r *serviceRepository) GetReminder(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceReminder, error) {
	var reminder types.ServiceReminder
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, strings.TrimSpace(userID), strings.TrimSpace(id)).
		First(&reminder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.fillReminderAssociations(ctx, []*types.ServiceReminder{&reminder}); err != nil {
		return nil, err
	}
	return &reminder, nil
}

func (r *serviceRepository) ListReminders(ctx context.Context, query types.ServiceListQuery) ([]*types.ServiceReminder, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.ServiceReminder{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if strings.TrimSpace(query.ProfileID) != "" {
		dbq = dbq.Where("profile_id = ?", strings.TrimSpace(query.ProfileID))
	}
	if strings.TrimSpace(query.SubjectID) != "" {
		dbq = dbq.Where("subject_id = ?", strings.TrimSpace(query.SubjectID))
	}
	if strings.TrimSpace(query.MemoryID) != "" {
		memoryID := strings.TrimSpace(query.MemoryID)
		memoryIDPattern := "%\"" + memoryID + "\"%"
		if r.db != nil && r.db.Dialector != nil && r.db.Dialector.Name() == "postgres" {
			dbq = dbq.Where("source_memory_ids::text LIKE ?", memoryIDPattern)
		} else {
			dbq = dbq.Where("source_memory_ids LIKE ?", memoryIDPattern)
		}
	}
	if strings.TrimSpace(query.Status) != "" {
		dbq = dbq.Where("status = ?", strings.TrimSpace(query.Status))
	}
	if strings.TrimSpace(query.AgentDomain) != "" {
		dbq = dbq.Where("agent_domain = ?", strings.TrimSpace(query.AgentDomain))
	}
	dbq = applyServiceKeyword(dbq, query.Keyword)

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reminders []*types.ServiceReminder
	err := dbq.
		Order("CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END").
		Order("COALESCE(due_at, updated_at) ASC").
		Order("updated_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&reminders).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.fillReminderAssociations(ctx, reminders); err != nil {
		return nil, 0, err
	}
	return reminders, total, nil
}

func applyServiceKeyword(dbq *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return dbq
	}
	like := "%" + strings.ToLower(keyword) + "%"
	return dbq.Where(
		"LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(stage) LIKE ? OR LOWER(channel) LIKE ? OR LOWER(risk_label) LIKE ? OR LOWER(next_action) LIKE ?",
		like, like, like, like, like, like,
	)
}

func (r *serviceRepository) UpdateReminderStatus(ctx context.Context, tenantID uint64, userID, id, status string) error {
	return r.db.WithContext(ctx).Model(&types.ServiceReminder{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, strings.TrimSpace(userID), strings.TrimSpace(id)).
		Updates(map[string]any{
			"status":            status,
			"write_back_status": reminderWriteBackStatus(status),
			"updated_at":        time.Now().UTC(),
		}).Error
}

func reminderWriteBackStatus(status string) string {
	switch status {
	case types.ServiceReminderStatusConfirmed:
		return "已确认动作"
	case types.ServiceReminderStatusCompleted:
		return "已完成"
	case types.ServiceReminderStatusIgnored:
		return "已忽略"
	case types.ServiceReminderStatusSnoozed:
		return "稍后处理"
	case types.ServiceReminderStatusGenerated:
		return "已生成话术"
	default:
		return "待确认"
	}
}

func (r *serviceRepository) CountRemindersByStatus(ctx context.Context, tenantID uint64, userID, profileID string) (map[string]int64, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	query := r.db.WithContext(ctx).Model(&types.ServiceReminder{}).
		Select("status, COUNT(*) AS count").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID)
	if strings.TrimSpace(profileID) != "" {
		query = query.Where("profile_id = ?", strings.TrimSpace(profileID))
	}
	err := query.Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Count
	}
	return out, nil
}

func (r *serviceRepository) CreateActionDraft(ctx context.Context, draft *types.AgentActionDraft) error {
	return r.db.WithContext(ctx).Create(draft).Error
}

func (r *serviceRepository) GetActionDraft(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentActionDraft, error) {
	var draft types.AgentActionDraft
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, strings.TrimSpace(userID), strings.TrimSpace(id)).
		First(&draft).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &draft, err
}

func (r *serviceRepository) ListActionDrafts(ctx context.Context, tenantID uint64, userID, reminderID string) ([]*types.AgentActionDraft, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, strings.TrimSpace(userID))
	if strings.TrimSpace(reminderID) != "" {
		query = query.Where("reminder_id = ?", strings.TrimSpace(reminderID))
	}
	var drafts []*types.AgentActionDraft
	err := query.Order("created_at DESC").Find(&drafts).Error
	return drafts, err
}

func (r *serviceRepository) UpdateActionDraftStatus(ctx context.Context, tenantID uint64, userID, id, status string) error {
	return r.db.WithContext(ctx).Model(&types.AgentActionDraft{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, strings.TrimSpace(userID), strings.TrimSpace(id)).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (r *serviceRepository) CreateActionLog(ctx context.Context, log *types.AgentActionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *serviceRepository) fillReminderAssociations(ctx context.Context, reminders []*types.ServiceReminder) error {
	if len(reminders) == 0 {
		return nil
	}

	memoryIDs := make([]string, 0)
	subjectIDs := make([]string, 0, len(reminders))
	profileIDs := make([]string, 0, len(reminders))
	reminderIDs := make([]string, 0, len(reminders))
	seenMemory := map[string]bool{}
	for _, reminder := range reminders {
		reminderIDs = append(reminderIDs, reminder.ID)
		subjectIDs = append(subjectIDs, reminder.SubjectID)
		profileIDs = append(profileIDs, reminder.ProfileID)
		for _, id := range reminder.SourceMemoryIDs {
			if id = strings.TrimSpace(id); id != "" && !seenMemory[id] {
				seenMemory[id] = true
				memoryIDs = append(memoryIDs, id)
			}
		}
	}

	memoriesByID := map[string]types.OrganizeMemory{}
	if len(memoryIDs) > 0 {
		var memories []types.OrganizeMemory
		tenantID := reminders[0].TenantID
		if err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND id IN ?", tenantID, memoryIDs).
			Find(&memories).Error; err != nil {
			return err
		}
		for _, memory := range memories {
			memoriesByID[memory.ID] = memory
		}
	}

	docsBySubject := map[string][]*types.AgentWorkDoc{}
	var docs []*types.AgentWorkDoc
	if len(subjectIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND profile_id IN ? AND subject_id IN ?", reminders[0].TenantID, uniqueStrings(profileIDs), uniqueStrings(subjectIDs)).
			Order("doc_path ASC").
			Find(&docs).Error; err != nil {
			return err
		}
		for _, doc := range docs {
			docsBySubject[doc.SubjectID] = append(docsBySubject[doc.SubjectID], doc)
		}
	}

	draftsByReminder := map[string][]*types.AgentActionDraft{}
	var drafts []*types.AgentActionDraft
	if len(reminderIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND user_id = ? AND reminder_id IN ?", reminders[0].TenantID, reminders[0].UserID, reminderIDs).
			Order("created_at DESC").
			Find(&drafts).Error; err != nil {
			return err
		}
		for _, draft := range drafts {
			draftsByReminder[draft.ReminderID] = append(draftsByReminder[draft.ReminderID], draft)
		}
	}

	for _, reminder := range reminders {
		reminder.MemoryEvidence = make([]types.ServiceMemoryEvidence, 0, len(reminder.SourceMemoryIDs))
		for _, id := range reminder.SourceMemoryIDs {
			memory, ok := memoriesByID[id]
			if !ok {
				continue
			}
			reminder.MemoryEvidence = append(reminder.MemoryEvidence, types.ServiceMemoryEvidence{
				ID:              memory.ID,
				Title:           memory.Title,
				Summary:         plainExcerpt(memory.Content, memory.Metadata),
				SourceLabel:     firstNonEmpty(memory.Source, "个人记忆"),
				OccurredAtLabel: formatMonthDay(memory.OccurredAt),
			})
			if len(reminder.MemoryEvidence) >= 3 {
				break
			}
		}
		reminder.WorkDocs = docsBySubject[reminder.SubjectID]
		reminder.ActionDrafts = draftsByReminder[reminder.ID]
	}
	return nil
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func plainExcerpt(content string, metadata types.JSONMap) string {
	if summary, ok := metadata["summary"].(string); ok && strings.TrimSpace(summary) != "" {
		return truncateText(strings.TrimSpace(summary), 86)
	}
	text := strings.TrimSpace(content)
	replacer := strings.NewReplacer(
		"<br>", " ", "<br/>", " ", "<br />", " ",
	)
	text = replacer.Replace(text)
	text = strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"#", " ", ">", " ", "*", " ", "_", " ", "`", " ", "~", " ",
	).Replace(stripTags(text))
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "暂无记忆内容"
	}
	return truncateText(text, 86)
}

func stripTags(value string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range value {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
			builder.WriteRune(' ')
		default:
			if !inTag {
				builder.WriteRune(r)
			}
		}
	}
	return builder.String()
}

func truncateText(value string, max int) string {
	runes := []rune(value)
	if max <= 0 || len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func formatMonthDay(t time.Time) string {
	if t.IsZero() {
		return "最近"
	}
	return t.Local().Format("1月2日")
}
