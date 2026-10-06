package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrOrganizeCourseInvalidSource  = errors.New("invalid course source")
	ErrOrganizeCourseNoValidFiles   = errors.New("no valid file found in the uploaded folder")
	ErrOrganizeCourseLessonNotReady = errors.New("course lesson content is not available")
	ErrOrganizeCourseInvalidCover   = errors.New("invalid course cover image")
)

const (
	organizeCourseMaxTitleLength   = 255
	organizeCourseMaxCategoryRunes = 64
	organizeCourseMaxTeacherRunes  = 64
	organizeCourseMaxTeacherTitle  = 128
	organizeCourseMaxCoverURLLen   = 512
	organizeCourseDefaultPageSize  = 20
)

// CreateCourseFromFolder turns one uploaded folder into one published course.
//
// Each usable file becomes an organize_outputs row (reusing the exact same
// extraction + AI-summary pipeline as a single upload) plus an
// organize_course_lessons row that owns only the ordering. The course row is
// written last, in one transaction with its lessons.
//
// A folder routinely contains noise: .DS_Store, __MACOSX sidecars, or a file
// type the pipeline cannot parse. Those are collected in Skipped instead of
// aborting the whole course, because losing a 12-lesson upload to one stray
// file would be the worst possible behaviour.
func (s *organizeService) CreateCourseFromFolder(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeCourseUploadInput,
	files []types.OrganizeCourseUploadFile,
) (*types.OrganizeCourseUploadResult, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	if s.fileService == nil && s.storageResolver == nil {
		return nil, fmt.Errorf("file service is not configured")
	}

	title := trimMax(strings.TrimSpace(input.Title), organizeCourseMaxTitleLength)
	if title == "" {
		title = trimMax(strings.TrimSpace(input.DirectoryName), organizeCourseMaxTitleLength)
	}
	if title == "" {
		return nil, ErrOrganizeTitleRequired
	}

	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = types.OrganizeCourseSourceOfficial
	}
	if !types.IsValidOrganizeCourseSource(source) {
		return nil, ErrOrganizeCourseInvalidSource
	}

	publicStatus := strings.TrimSpace(input.PublicStatus)
	if publicStatus == "" {
		publicStatus = types.OrganizePublicContentStatusPublished
	}
	if !types.IsValidOrganizePublicContentStatus(publicStatus) {
		return nil, ErrOrganizeInvalidPublicStatus
	}

	category := trimMax(strings.TrimSpace(input.Category), organizeCourseMaxCategoryRunes)
	if category != "" && !isKnownOrganizeDiscoverCategory(category) {
		return nil, ErrOrganizeInvalidCategory
	}
	visibilityScope, sharedSpaceIDs, err := s.normalizeCourseVisibility(
		ctx,
		input.VisibilityScope,
		input.SharedSpaceIDs,
	)
	if err != nil {
		return nil, err
	}

	accepted, skipped := prepareOrganizeCourseFiles(files)
	if len(accepted) == 0 {
		return nil, ErrOrganizeCourseNoValidFiles
	}

	fileService, err := s.resolveOrganizeFileService(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}

	// The course ID has to exist before the outputs do: every lesson output
	// carries series_id = course ID so the legacy series_* columns stay usable
	// for the single-post discover rendering.
	courseID := types.NewOrganizeCourseID()

	coverURL := trimMax(strings.TrimSpace(input.CoverURL), organizeCourseMaxCoverURLLen)
	coverPath := ""
	if len(input.CoverImageData) > 0 {
		coverURL, coverPath, err = s.saveOrganizeCourseCoverImage(ctx, tenantID, fileService, input)
		if err != nil {
			return nil, err
		}
	}

	lessons := make([]*types.OrganizeCourseLesson, 0, len(accepted))
	createdOutputs := make([]*types.OrganizeOutput, 0, len(accepted))
	for _, file := range accepted {
		order := len(lessons)
		output, lessonType, buildErr := s.buildOrganizeCourseLessonOutput(
			ctx, tenantID, userID, courseID, title, category, order, file, fileService,
		)
		if buildErr != nil {
			skipped = append(skipped, types.OrganizeCourseSkipped{
				FileName: file.FileName,
				Reason:   buildErr.Error(),
			})
			continue
		}
		createdOutputs = append(createdOutputs, output)
		lessons = append(lessons, &types.OrganizeCourseLesson{
			CourseID:   courseID,
			TenantID:   tenantID,
			OutputID:   output.ID,
			Title:      trimMax(organizeCourseLessonTitle(output.Title, file.FileName), organizeMaxTitleLength),
			LessonType: lessonType,
			SortOrder:  order,
		})
	}
	if len(lessons) == 0 {
		if coverPath != "" {
			_ = fileService.DeleteFile(ctx, coverPath)
		}
		return nil, ErrOrganizeCourseNoValidFiles
	}

	course := &types.OrganizeCourse{
		ID:              courseID,
		TenantID:        tenantID,
		UserID:          userID,
		Source:          source,
		Title:           title,
		Summary:         trimMax(strings.TrimSpace(input.Summary), 0),
		Category:        category,
		CoverURL:        coverURL,
		TeacherName:     trimMax(strings.TrimSpace(input.TeacherName), organizeCourseMaxTeacherRunes),
		TeacherTitle:    trimMax(strings.TrimSpace(input.TeacherTitle), organizeCourseMaxTeacherTitle),
		PublicStatus:    publicStatus,
		VisibilityScope: visibilityScope,
		SharedSpaceIDs:  sharedSpaceIDs,
	}
	if err := s.repo.CreateCourse(ctx, course, lessons); err != nil {
		// Best effort: the outputs are already persisted and are perfectly valid
		// standalone posts, but leaving them behind on a failed course creation
		// would make the folder look half-imported.
		for _, output := range createdOutputs {
			_ = s.deleteOrganizeStoredFile(ctx, tenantID, "course", output.ID, output.Metadata, "file_path", "storage_path")
			_ = s.repo.DeleteOutput(ctx, tenantID, userID, output.ID)
		}
		if coverPath != "" {
			_ = fileService.DeleteFile(ctx, coverPath)
		}
		return nil, err
	}
	course.Lessons = lessons

	return &types.OrganizeCourseUploadResult{
		Course:  course,
		Skipped: skipped,
	}, nil
}

// CreateCourse creates the course shell before any lesson body is uploaded.
// The admin UI uses this as the first step, then appends each lesson through
// AppendCourseLessonFromUpload.
func (s *organizeService) CreateCourse(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeCourseUploadInput,
) (*types.OrganizeCourse, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}

	title := trimMax(strings.TrimSpace(input.Title), organizeCourseMaxTitleLength)
	if title == "" {
		title = trimMax(strings.TrimSpace(input.DirectoryName), organizeCourseMaxTitleLength)
	}
	if title == "" {
		return nil, ErrOrganizeTitleRequired
	}

	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = types.OrganizeCourseSourceOfficial
	}
	if !types.IsValidOrganizeCourseSource(source) {
		return nil, ErrOrganizeCourseInvalidSource
	}

	publicStatus := strings.TrimSpace(input.PublicStatus)
	if publicStatus == "" {
		publicStatus = types.OrganizePublicContentStatusPublished
	}
	if !types.IsValidOrganizePublicContentStatus(publicStatus) {
		return nil, ErrOrganizeInvalidPublicStatus
	}

	category := trimMax(strings.TrimSpace(input.Category), organizeCourseMaxCategoryRunes)
	if category != "" && !isKnownOrganizeDiscoverCategory(category) {
		return nil, ErrOrganizeInvalidCategory
	}
	visibilityScope, sharedSpaceIDs, err := s.normalizeCourseVisibility(
		ctx,
		input.VisibilityScope,
		input.SharedSpaceIDs,
	)
	if err != nil {
		return nil, err
	}

	course := &types.OrganizeCourse{
		ID:              types.NewOrganizeCourseID(),
		TenantID:        tenantID,
		UserID:          userID,
		Source:          source,
		Title:           title,
		Summary:         trimMax(strings.TrimSpace(input.Summary), 0),
		Category:        category,
		CoverURL:        trimMax(strings.TrimSpace(input.CoverURL), organizeCourseMaxCoverURLLen),
		TeacherName:     trimMax(strings.TrimSpace(input.TeacherName), organizeCourseMaxTeacherRunes),
		TeacherTitle:    trimMax(strings.TrimSpace(input.TeacherTitle), organizeCourseMaxTeacherTitle),
		PublicStatus:    publicStatus,
		VisibilityScope: visibilityScope,
		SharedSpaceIDs:  sharedSpaceIDs,
	}

	var fileService interfaces.FileService
	var coverPath string
	if len(input.CoverImageData) > 0 {
		var err error
		fileService, err = s.resolveOrganizeFileService(ctx, tenantID, "")
		if err != nil {
			return nil, err
		}
		course.CoverURL, coverPath, err = s.saveOrganizeCourseCoverImage(
			ctx,
			tenantID,
			fileService,
			input,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := s.repo.CreateCourse(ctx, course, nil); err != nil {
		if coverPath != "" {
			_ = fileService.DeleteFile(ctx, coverPath)
		}
		return nil, err
	}
	return course, nil
}

// AppendCourseLessonFromUpload processes exactly one lesson and attaches it to
// an existing course. Keeping this request single-file is what allows large
// videos to upload independently instead of sharing one multipart body with
// every other lesson in the folder.
func (s *organizeService) AppendCourseLessonFromUpload(
	ctx context.Context,
	tenantID uint64,
	userID string,
	courseID string,
	file types.OrganizeCourseUploadFile,
) (*types.OrganizeCourseUploadResult, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, ErrOrganizeNotFound
	}

	course, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil || course.TenantID != tenantID || course.UserID != userID {
		return nil, ErrOrganizeNotFound
	}

	fileService, err := s.resolveOrganizeFileService(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}

	order := course.LessonCount
	output, lessonType, err := s.buildOrganizeCourseLessonOutput(
		ctx,
		tenantID,
		userID,
		course.ID,
		course.Title,
		course.Category,
		order,
		file,
		fileService,
	)
	if err != nil {
		return nil, err
	}

	lesson := &types.OrganizeCourseLesson{
		CourseID:   course.ID,
		TenantID:   tenantID,
		OutputID:   output.ID,
		Title:      trimMax(organizeCourseLessonTitle(output.Title, file.FileName), organizeMaxTitleLength),
		LessonType: lessonType,
		SortOrder:  order,
	}
	if err := s.repo.AppendCourseLesson(ctx, course, lesson); err != nil {
		_ = s.deleteOrganizeStoredFile(ctx, tenantID, "course", output.ID, output.Metadata, "file_path", "storage_path")
		_ = s.repo.DeleteOutput(ctx, tenantID, userID, output.ID)
		return nil, err
	}
	course.Lessons = []*types.OrganizeCourseLesson{lesson}

	return &types.OrganizeCourseUploadResult{
		Course:  course,
		Skipped: nil,
	}, nil
}

func (s *organizeService) getAdminCourseLesson(
	ctx context.Context,
	courseID string,
	lessonID string,
) (*types.OrganizeCourse, *types.OrganizeCourseLesson, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(courseID))
	if err != nil {
		return nil, nil, err
	}
	if course == nil {
		return nil, nil, ErrOrganizeNotFound
	}
	lesson, err := s.repo.GetCourseLesson(ctx, course.ID, strings.TrimSpace(lessonID))
	if err != nil {
		return nil, nil, err
	}
	if lesson == nil || lesson.CourseID != course.ID {
		return nil, nil, ErrOrganizeNotFound
	}
	return course, lesson, nil
}

// UpdateCourseLesson edits the administrator-facing title and description
// without rebuilding or replacing the uploaded lesson file.
func (s *organizeService) UpdateCourseLesson(
	ctx context.Context,
	courseID string,
	lessonID string,
	input types.OrganizeCourseLessonUpdateInput,
) (*types.OrganizeCourseLesson, error) {
	course, lesson, err := s.getAdminCourseLesson(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	title := trimMax(strings.TrimSpace(input.Title), organizeMaxTitleLength)
	if title == "" {
		return nil, ErrOrganizeTitleRequired
	}
	if lesson.Output == nil {
		return nil, ErrOrganizeCourseLessonNotReady
	}
	lesson.Title = title
	lesson.Output.Title = title
	lesson.Output.SourceSummary = trimMax(strings.TrimSpace(input.Description), organizeMaxShortText)
	if err := s.repo.UpdateCourseLesson(ctx, course, lesson); err != nil {
		return nil, err
	}
	return s.repo.GetCourseLesson(ctx, course.ID, lesson.ID)
}

// DeleteCourseLesson removes one lesson and its stored source file.
func (s *organizeService) DeleteCourseLesson(
	ctx context.Context,
	courseID string,
	lessonID string,
) error {
	course, lesson, err := s.getAdminCourseLesson(ctx, courseID, lessonID)
	if err != nil {
		return err
	}
	if lesson.Output != nil {
		if err := s.deleteOrganizeStoredFile(
			ctx,
			course.TenantID,
			"course",
			lesson.Output.ID,
			lesson.Output.Metadata,
			"file_path",
			"storage_path",
		); err != nil {
			return err
		}
	}
	return s.repo.DeleteCourseLesson(ctx, course, lesson)
}

// buildOrganizeCourseLessonOutput creates the organize_outputs row backing one
// lesson. It mirrors CreateOutputFromUpload on purpose: same validation, same
// storage naming, same AI metadata, only the course-related fields differ.
//
// The course's publication state is deliberately NOT passed down. A lesson is
// not a post and must never enter the public content pool on its own; see
// applyOrganizeLessonOutputState.
func (s *organizeService) buildOrganizeCourseLessonOutput(
	ctx context.Context,
	tenantID uint64,
	userID, courseID, courseTitle, category string,
	order int,
	file types.OrganizeCourseUploadFile,
	fileService interfaces.FileService,
) (*types.OrganizeOutput, string, error) {
	cleanName := strings.TrimSpace(file.FileName)
	if cleanName == "" {
		return nil, "", errors.New("file name is required")
	}
	if !isValidFileType(cleanName) {
		return nil, "", fmt.Errorf("unsupported file type: %s", strings.ToLower(filepath.Ext(cleanName)))
	}
	data, err := readOrganizeCourseUploadFile(file)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("file is empty: %s", cleanName)
	}

	contentKind, outputType, icon := organizeOutputKindInfo(cleanName, file.MimeType)
	baseName := strings.TrimSuffix(filepath.Base(cleanName), filepath.Ext(cleanName))
	if baseName == "" {
		baseName = cleanName
	}

	content, transcript, asrModelID, warnings, err := s.extractOrganizeUploadContent(
		ctx, cleanName, file.MimeType, data, contentKind,
	)
	if err != nil {
		return nil, "", err
	}

	aiResult, aiModelID, aiStatus := s.generateUploadedOutputAIResult(ctx, cleanName, outputType, content)
	if aiResult.Title == "" {
		aiResult.Title = baseName
	}
	if aiResult.Summary == "" {
		aiResult.Summary = organizeUploadFallbackSummary(cleanName, outputType, content)
	}
	aiResult.Tags = normalizeOrganizeUploadTags(append(aiResult.Tags, organizeUploadFallbackTags(outputType)...))
	if len(aiResult.Tags) == 0 {
		aiResult.Tags = normalizeOrganizeUploadTags(organizeUploadFallbackTags(outputType))
	}

	storageName := fmt.Sprintf("organize_course_%s%s", uuid.NewString()[:12], filepath.Ext(cleanName))
	storagePath, saveErr := fileService.SaveBytes(ctx, data, tenantID, storageName, false)
	if saveErr != nil {
		return nil, "", fmt.Errorf("save upload file: %w", saveErr)
	}

	metadata := types.JSONMap{
		"content_kind":       contentKind,
		"content_kind_label": outputType,
		"file_name":          trimMax(cleanName, 0),
		"file_type":          strings.TrimPrefix(strings.ToLower(filepath.Ext(cleanName)), "."),
		"file_path":          trimMax(storagePath, 0),
		"mime_type":          trimMax(file.MimeType, 0),
		"ai_status":          aiStatus,
		"ai_model_id":        aiModelID,
		"tags":               types.StringArray(aiResult.Tags),
		"upload_source":      "course_folder",
		"uploaded_at":        time.Now().UTC().Format(time.RFC3339),
		"course_id":          courseID,
		"course_title":       trimMax(courseTitle, organizeCourseMaxTitleLength),
		"lesson_order":       order + 1,
	}
	if file.RelativePath != "" {
		metadata["relative_path"] = trimMax(file.RelativePath, 0)
	}
	if category != "" {
		metadata["discover_category"] = category
	}
	if transcript != "" {
		metadata["transcript"] = transcript
	}
	if asrModelID != "" {
		metadata["asr_model_id"] = asrModelID
	}
	if len(warnings) > 0 {
		metadata["warnings"] = warnings
	}

	output := &types.OrganizeOutput{
		TenantID:          tenantID,
		UserID:            userID,
		Title:             trimMax(aiResult.Title, organizeMaxTitleLength),
		OutputType:        outputType,
		Content:           trimMax(content, 0),
		SourceSummary:     trimMax(aiResult.Summary, organizeMaxShortText),
		PublicContentType: types.OrganizePublicContentTypeCourse,
		SeriesID:          courseID,
		SeriesTitle:       trimMax(courseTitle, organizeCourseMaxTitleLength),
		SeriesOrder:       order + 1,
		Icon:              icon,
		Metadata:          normalizeJSONMap(metadata),
	}
	applyOrganizeLessonOutputState(output)

	if err := s.repo.CreateOutput(ctx, output, nil); err != nil {
		_ = fileService.DeleteFile(ctx, storagePath)
		return nil, "", err
	}
	return output, contentKind, nil
}

func readOrganizeCourseUploadFile(file types.OrganizeCourseUploadFile) ([]byte, error) {
	if len(file.Data) > 0 {
		return file.Data, nil
	}
	if file.Open == nil {
		return nil, fmt.Errorf("file is empty: %s", strings.TrimSpace(file.FileName))
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", strings.TrimSpace(file.FileName), err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", strings.TrimSpace(file.FileName), err)
	}
	return data, nil
}

func (s *organizeService) saveOrganizeCourseCoverImage(
	ctx context.Context,
	tenantID uint64,
	fileService interfaces.FileService,
	input types.OrganizeCourseUploadInput,
) (string, string, error) {
	mimeType := detectOrganizeCourseCoverMIME(input.CoverImageData)
	if !isSupportedOrganizeCourseCoverImage(mimeType) {
		return "", "", ErrOrganizeCourseInvalidCover
	}
	ext := organizeCourseCoverImageExt(input.CoverImageFileName, mimeType)
	storageName := fmt.Sprintf("organize_course_cover_%s%s", uuid.NewString()[:12], ext)
	storagePath, err := fileService.SaveBytes(ctx, input.CoverImageData, tenantID, storageName, false)
	if err != nil {
		return "", "", fmt.Errorf("save course cover image: %w", err)
	}
	// Persist the stable storage reference, not a presigned URL. Presigned
	// OSS URLs and resource grants expire, while course rows are long-lived.
	return trimMax(storagePath, organizeCourseMaxCoverURLLen), storagePath, nil
}

func detectOrganizeCourseCoverMIME(data []byte) string {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	mimeType := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	if index := strings.Index(mimeType, ";"); index >= 0 {
		mimeType = mimeType[:index]
	}
	return mimeType
}

func isSupportedOrganizeCourseCoverImage(mimeType string) bool {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func organizeCourseCoverImageExt(fileName, mimeType string) string {
	switch ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}

// GetCourse reads a course together with its outline, platform-wide.
//
// Course identity is global ("crs_xxx"), and the two callers are the platform
// admin console and the published-only discover page. Scoping the read to the
// caller's own workspace would make the admin list (which is cross-tenant, like
// ListAdminPublicContents) show courses that then 404 on click. Access control
// lives in the guards, exactly as it does for ModeratePublicContent.
func (s *organizeService) GetCourse(ctx context.Context, id string) (*types.OrganizeCourse, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrOrganizeNotFound
	}
	lessons, err := s.repo.ListCourseLessonsWithOutputs(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	course.Lessons = lessons
	// Older unit-test schemas and rolling deployments may not have the
	// relation table yet. Course reads remain usable during that window; the
	// migration is still required before a shared-space setting is written.
	if sharedSpaceIDs, relationErr := s.repo.ListCourseSharedSpaceIDs(ctx, course.ID); relationErr == nil {
		course.SharedSpaceIDs = sharedSpaceIDs
	}
	return course, nil
}

// GetPublishedCourse serves the discover-side detail page: only for courses
// that are actually published. An unpublished course is reported as not found
// rather than forbidden, so the endpoint does not leak the existence of another
// workspace's drafts.
func (s *organizeService) GetPublishedCourse(ctx context.Context, id string) (*types.OrganizeCourse, error) {
	course, err := s.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if course.PublicStatus != types.OrganizePublicContentStatusPublished {
		return nil, ErrOrganizeNotFound
	}
	visible, err := s.courseVisibleToViewer(ctx, course)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, ErrOrganizeNotFound
	}
	for _, lesson := range course.Lessons {
		if lesson == nil || lesson.Output == nil {
			continue
		}
		// A lesson body deliberately carries no public status of its own, so its
		// visibility follows the course we just checked rather than any
		// post-level flag. Only an archived body is withheld — and the outline
		// slot survives so the numbering still matches what the administrator
		// authored.
		if lesson.Output.Status == types.OrganizeOutputStatusArchived {
			lesson.Output = nil
		}
	}
	return course, nil
}

// GetPublishedCourseLessonContent returns one lesson body.
//
// Bodies are kept out of the outline payload on purpose: a 14-lesson course
// ships several megabytes of markdown per detail request, while the reader only
// ever renders one chapter. This repeats the same two checks as the media
// endpoint so the body and the media can never disagree about access.
func (s *organizeService) GetPublishedCourseLessonContent(
	ctx context.Context,
	courseID, lessonID string,
) (string, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(courseID))
	if err != nil {
		return "", err
	}
	if course == nil || course.PublicStatus != types.OrganizePublicContentStatusPublished {
		return "", ErrOrganizeNotFound
	}
	visible, err := s.courseVisibleToViewer(ctx, course)
	if err != nil {
		return "", err
	}
	if !visible {
		return "", ErrOrganizeNotFound
	}
	lesson, err := s.repo.GetCourseLesson(ctx, course.ID, strings.TrimSpace(lessonID))
	if err != nil {
		return "", err
	}
	if lesson == nil || lesson.Output == nil || lesson.Output.Status == types.OrganizeOutputStatusArchived {
		return "", ErrOrganizeCourseLessonNotReady
	}
	return lesson.Output.Content, nil
}

func (s *organizeService) publishedCourseLessonMediaTarget(
	ctx context.Context,
	courseID, lessonID string,
) (*types.OrganizeCourse, *types.OrganizeCourseLesson, string, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(courseID))
	if err != nil {
		return nil, nil, "", err
	}
	if course == nil || course.PublicStatus != types.OrganizePublicContentStatusPublished {
		return nil, nil, "", ErrOrganizeNotFound
	}
	visible, err := s.courseVisibleToViewer(ctx, course)
	if err != nil {
		return nil, nil, "", err
	}
	if !visible {
		return nil, nil, "", ErrOrganizeNotFound
	}
	lessons, err := s.repo.ListCourseLessonsWithOutputs(ctx, course.ID)
	if err != nil {
		return nil, nil, "", err
	}
	var target *types.OrganizeCourseLesson
	for _, lesson := range lessons {
		if lesson != nil && lesson.ID == strings.TrimSpace(lessonID) {
			target = lesson
			break
		}
	}
	if target == nil || target.Output == nil || target.Output.Status == types.OrganizeOutputStatusArchived {
		return nil, nil, "", ErrOrganizeCourseLessonNotReady
	}
	filePath := organizeStoredFilePath(target.Output.Metadata, "file_path", "storage_path")
	if filePath == "" {
		return nil, nil, "", ErrOrganizeCourseLessonNotReady
	}
	return course, target, filePath, nil
}

// OpenPublishedCourseLessonMedia returns a reader only after verifying that
// the course is published and the lesson belongs to it. The storage path never
// leaves this method, so cross-tenant viewers receive the bytes without
// learning an internal locator.
func (s *organizeService) OpenPublishedCourseLessonMedia(
	ctx context.Context,
	courseID, lessonID string,
) (io.ReadCloser, string, string, error) {
	course, target, filePath, err := s.publishedCourseLessonMediaTarget(ctx, courseID, lessonID)
	if err != nil {
		return nil, "", "", err
	}
	// The course owns the stored object. A public/shared-space viewer may carry
	// a different effective tenant, but that tenant must not be used to resolve
	// or read the course's storage backend.
	ownerCtx := context.WithValue(ctx, types.TenantIDContextKey, course.TenantID)
	fileService, err := s.resolveOrganizeFileService(ownerCtx, course.TenantID, filePath)
	if err != nil {
		return nil, "", "", err
	}
	reader, err := fileService.GetFile(ownerCtx, filePath)
	if err != nil {
		return nil, "", "", err
	}
	fileName := stringValue(target.Output.Metadata, "file_name")
	if fileName == "" {
		fileName = target.Title
	}
	return reader, fileName, stringValue(target.Output.Metadata, "mime_type"), nil
}

// OpenPublishedCourseCover serves a course cover through the course visibility
// boundary. The storage locator stays server-side, so private OSS buckets do
// not need public-read ACLs or client-facing credentials.
func (s *organizeService) OpenPublishedCourseCover(
	ctx context.Context,
	courseID string,
) (io.ReadCloser, string, string, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(courseID))
	if err != nil {
		return nil, "", "", err
	}
	if course == nil || course.PublicStatus != types.OrganizePublicContentStatusPublished {
		return nil, "", "", ErrOrganizeNotFound
	}
	visible, err := s.courseVisibleToViewer(ctx, course)
	if err != nil {
		return nil, "", "", err
	}
	if !visible {
		return nil, "", "", ErrOrganizeNotFound
	}

	coverPath := strings.TrimSpace(course.CoverURL)
	if !isOrganizeStoredFileReference(coverPath) {
		return nil, "", "", ErrOrganizeNotFound
	}
	fileService, err := s.resolveOrganizeFileService(ctx, course.TenantID, coverPath)
	if err != nil {
		return nil, "", "", err
	}
	reader, err := fileService.GetFile(ctx, coverPath)
	if err != nil {
		return nil, "", "", err
	}

	fileName := filepath.Base(coverPath)
	mimeType := ""
	if s.resourceCatalog != nil {
		if resource, resolveErr := s.resourceCatalog.Resolve(ctx, coverPath); resolveErr == nil && resource != nil {
			if strings.TrimSpace(resource.OriginalName) != "" {
				fileName = resource.OriginalName
			}
			mimeType = strings.TrimSpace(resource.MimeType)
		}
	}
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "course-cover"
	}
	return reader, fileName, mimeType, nil
}

// GetPublishedCourseLessonMediaURL returns a short-lived or presigned URL
// after applying the same course visibility checks as the protected media
// endpoint. Native media elements can use this URL to issue Range requests
// without exposing the API bearer token to the browser media loader.
func (s *organizeService) GetPublishedCourseLessonMediaURL(
	ctx context.Context,
	courseID, lessonID string,
) (string, string, string, error) {
	course, target, filePath, err := s.publishedCourseLessonMediaTarget(ctx, courseID, lessonID)
	if err != nil {
		return "", "", "", err
	}
	ownerCtx := context.WithValue(ctx, types.TenantIDContextKey, course.TenantID)
	fileService, err := s.resolveOrganizeFileService(ownerCtx, course.TenantID, filePath)
	if err != nil {
		return "", "", "", err
	}
	mediaURL, err := fileService.GetFileURL(ownerCtx, filePath)
	if err != nil {
		return "", "", "", err
	}
	fileName := stringValue(target.Output.Metadata, "file_name")
	if fileName == "" {
		fileName = target.Title
	}
	return mediaURL, fileName, stringValue(target.Output.Metadata, "mime_type"), nil
}

// ListPublishedCourses backs the course cards embedded in 推荐: the same
// cross-tenant published pool as the discover output listing.
func (s *organizeService) ListPublishedCourses(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) ([]*types.OrganizeCourse, int64, error) {
	query.TenantID = 0
	query.UserID = ""
	query.PublicStatus = types.OrganizePublicContentStatusPublished
	query, err := s.courseViewerQuery(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return s.ListCourses(ctx, query)
}

// ListCourses serves the discover page (no tenant scope, published only) and
// the platform admin console (platform-wide, any status). It deliberately does
// not validate a caller scope: each caller applies its own filter, and the
// guards decide who may reach it.
func (s *organizeService) ListCourses(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) ([]*types.OrganizeCourse, int64, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Category = strings.TrimSpace(query.Category)
	query.Source = strings.TrimSpace(query.Source)
	query.PublicStatus = strings.TrimSpace(query.PublicStatus)
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = organizeCourseDefaultPageSize
	}
	if query.PageSize > organizeMaxPageSize {
		query.PageSize = organizeMaxPageSize
	}
	courses, total, err := s.repo.ListCourses(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	if !query.VisibilityFilter {
		for _, course := range courses {
			if course == nil {
				continue
			}
			if sharedSpaceIDs, relationErr := s.repo.ListCourseSharedSpaceIDs(ctx, course.ID); relationErr == nil {
				course.SharedSpaceIDs = sharedSpaceIDs
			}
		}
	}
	return courses, total, nil
}

func (s *organizeService) GetCourseStats(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) (*types.OrganizeCourseStats, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Category = strings.TrimSpace(query.Category)
	query.Source = strings.TrimSpace(query.Source)
	query.PublicStatus = strings.TrimSpace(query.PublicStatus)
	return s.repo.GetCourseStats(ctx, query)
}

// UpdateCoursePublicStatus flips a course and, crucially, all of its lessons.
//
// The course row is loaded first so every write uses that course's own tenant,
// which keeps a platform admin able to moderate any workspace's course without
// handing the request a tenant it should not otherwise touch.
func (s *organizeService) UpdateCoursePublicStatus(
	ctx context.Context,
	id, status string,
) (*types.OrganizeCourse, error) {
	status = strings.TrimSpace(status)
	if !types.IsValidOrganizePublicContentStatus(status) {
		return nil, ErrOrganizeInvalidPublicStatus
	}
	course, err := s.repo.UpdateCoursePublicStatus(ctx, strings.TrimSpace(id), status)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrOrganizeNotFound
	}
	return course, nil
}

func (s *organizeService) UpdateCourseVisibility(
	ctx context.Context,
	id string,
	input types.OrganizeCourseVisibilityInput,
) (*types.OrganizeCourse, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, ErrOrganizeNotFound
	}
	visibilityScope, sharedSpaceIDs, err := s.normalizeCourseVisibility(
		ctx,
		input.VisibilityScope,
		input.SharedSpaceIDs,
	)
	if err != nil {
		return nil, err
	}
	createdBy := course.UserID
	if userID, ok := types.UserIDFromContext(ctx); ok && strings.TrimSpace(userID) != "" {
		createdBy = strings.TrimSpace(userID)
	}
	updated, err := s.repo.UpdateCourseVisibility(
		ctx,
		course.ID,
		visibilityScope,
		sharedSpaceIDs,
		createdBy,
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrOrganizeNotFound
	}
	lessons, err := s.repo.ListCourseLessonsWithOutputs(ctx, updated.ID)
	if err != nil {
		return nil, err
	}
	updated.Lessons = lessons
	updated.SharedSpaceIDs = sharedSpaceIDs
	return updated, nil
}

// DeleteCourse removes a course and its lesson rows, using the course's own
// tenant so the repository scope check still means something.
func (s *organizeService) DeleteCourse(ctx context.Context, id string) error {
	course, err := s.GetCourse(ctx, id)
	if err != nil {
		return err
	}
	for _, lesson := range course.Lessons {
		if lesson == nil || lesson.Output == nil {
			continue
		}
		if err := s.deleteOrganizeStoredFile(
			ctx,
			course.TenantID,
			"course",
			lesson.Output.ID,
			lesson.Output.Metadata,
			"file_path",
			"storage_path",
		); err != nil {
			return err
		}
	}
	if isOrganizeStoredFileReference(course.CoverURL) {
		fileService, resolveErr := s.resolveOrganizeFileService(ctx, course.TenantID, course.CoverURL)
		if resolveErr != nil {
			return resolveErr
		}
		if deleteErr := fileService.DeleteFile(ctx, course.CoverURL); deleteErr != nil {
			return deleteErr
		}
	}
	return s.repo.DeleteCourse(ctx, course.TenantID, course.ID)
}

func (s *organizeService) normalizeCourseVisibility(
	ctx context.Context,
	scope string,
	organizationIDs []string,
) (string, []string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = types.OrganizeCourseVisibilitySystem
	}
	if !types.IsValidOrganizeCourseVisibilityScope(scope) {
		return "", nil, ErrOrganizeInvalidVisibilityScope
	}

	uniqueIDs := make([]string, 0, len(organizationIDs))
	seen := make(map[string]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		organizationID = strings.TrimSpace(organizationID)
		if organizationID == "" {
			continue
		}
		if _, exists := seen[organizationID]; exists {
			continue
		}
		seen[organizationID] = struct{}{}
		uniqueIDs = append(uniqueIDs, organizationID)
	}
	if scope != types.OrganizeCourseVisibilitySharedSpace {
		return scope, nil, nil
	}
	if len(uniqueIDs) == 0 {
		return "", nil, ErrOrganizeCourseSharedSpaceRequired
	}
	if s.organizationRepo == nil {
		return "", nil, errors.New("organization repository is not configured")
	}
	for _, organizationID := range uniqueIDs {
		organization, err := s.organizationRepo.GetByID(ctx, organizationID)
		if err != nil {
			if errors.Is(err, repository.ErrOrganizationNotFound) {
				return "", nil, ErrOrganizeCourseSharedSpaceNotFound
			}
			return "", nil, err
		}
		if organization == nil {
			return "", nil, ErrOrganizeNotFound
		}
	}
	return scope, uniqueIDs, nil
}

func (s *organizeService) courseViewerQuery(
	ctx context.Context,
	query types.OrganizeCourseQuery,
) (types.OrganizeCourseQuery, error) {
	query.VisibilityFilter = true
	query.ViewerTenantID, _ = types.TenantIDFromContext(ctx)
	query.ViewerUserID, _ = types.UserIDFromContext(ctx)
	if types.IsSystemAdminFromContext(ctx) || s.organizationRepo == nil || query.ViewerTenantID == 0 {
		return query, nil
	}
	organizations, err := s.organizationRepo.ListByTenantID(ctx, query.ViewerTenantID)
	if err != nil {
		return query, err
	}
	query.ViewerOrganizationIDs = make([]string, 0, len(organizations))
	for _, organization := range organizations {
		if organization != nil &&
			teamSpaceAllowsActiveTenant(ctx, organization, query.ViewerTenantID, nil) &&
			strings.TrimSpace(organization.ID) != "" {
			query.ViewerOrganizationIDs = append(query.ViewerOrganizationIDs, strings.TrimSpace(organization.ID))
		}
	}
	return query, nil
}

func (s *organizeService) courseVisibleToViewer(
	ctx context.Context,
	course *types.OrganizeCourse,
) (bool, error) {
	if course == nil || types.IsSystemAdminFromContext(ctx) {
		return course != nil, nil
	}
	scope := strings.TrimSpace(course.VisibilityScope)
	if scope == "" || scope == types.OrganizeCourseVisibilitySystem {
		return true, nil
	}
	query, err := s.courseViewerQuery(ctx, types.OrganizeCourseQuery{})
	if err != nil {
		return false, err
	}
	if scope == types.OrganizeCourseVisibilityPrivate {
		return (query.ViewerTenantID > 0 && query.ViewerTenantID == course.TenantID) ||
			(strings.TrimSpace(query.ViewerUserID) != "" && query.ViewerUserID == course.UserID), nil
	}
	if scope == types.OrganizeCourseVisibilitySharedSpace {
		allowed := make(map[string]struct{}, len(query.ViewerOrganizationIDs))
		for _, organizationID := range query.ViewerOrganizationIDs {
			allowed[organizationID] = struct{}{}
		}
		for _, organizationID := range course.SharedSpaceIDs {
			if _, ok := allowed[organizationID]; ok {
				return true, nil
			}
		}
		// Direct detail reads may not have hydrated the relation IDs.
		if s.repo != nil {
			sharedSpaceIDs, relationErr := s.repo.ListCourseSharedSpaceIDs(ctx, course.ID)
			if relationErr != nil {
				return false, relationErr
			}
			for _, organizationID := range sharedSpaceIDs {
				if _, ok := allowed[organizationID]; ok {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// applyOrganizeLessonOutputState marks a freshly built lesson body as ready to
// read while keeping it out of the public content pool.
//
// A lesson is not a post. It is a part of a course, and its only public door is
// the course itself (GET /organize/courses/:id, which gates on the course's own
// public_status). Publishing the body as well would drop every lesson into the
// discover feed as an independent card next to the course that already contains
// it — so PublicStatus stays draft and the body stays invisible to
// ListPublicContents.
func applyOrganizeLessonOutputState(output *types.OrganizeOutput) {
	if output == nil {
		return
	}
	output.Status = types.OrganizeOutputStatusReady
	output.PublicStatus = types.OrganizePublicContentStatusDraft
	output.PublishedAt = nil
	output.PublishedBy = ""
}

// applyOrganizeLessonState re-syncs a lesson body when its course changes
// publication state: an offlined course must take its lessons' bodies down with
// it. The public status is pinned to draft in every case, because the lesson is
// never published independently of the course.
func applyOrganizeLessonState(output *types.OrganizeOutput, courseStatus string) {
	if output == nil {
		return
	}
	switch courseStatus {
	case types.OrganizePublicContentStatusPublished:
		output.Status = types.OrganizeOutputStatusReady
	case types.OrganizePublicContentStatusPendingReview:
		output.Status = types.OrganizeOutputStatusReview
	case types.OrganizePublicContentStatusOffline, types.OrganizePublicContentStatusRejected:
		output.Status = types.OrganizeOutputStatusArchived
	default:
		output.Status = types.OrganizeOutputStatusDraft
	}
	output.PublicStatus = types.OrganizePublicContentStatusDraft
	output.PublishedAt = nil
	output.PublishedBy = ""
}

// prepareOrganizeCourseFiles drops folder noise and returns the remaining files
// in the order they should become lessons.
//
// Ordering rule: files whose name starts with a number (01_, 02-, 3.) are
// ordered by that number first, because that is how a teacher authors a folder;
// everything else follows in byte order of the file name. Byte order is
// deterministic but is NOT pinyin order for Han characters, which is exactly why
// the folder convention is to number the files. This matches what the upload
// wizard previews, so the administrator sees the order they will publish.
func prepareOrganizeCourseFiles(
	files []types.OrganizeCourseUploadFile,
) (accepted []types.OrganizeCourseUploadFile, skipped []types.OrganizeCourseSkipped) {
	accepted = make([]types.OrganizeCourseUploadFile, 0, len(files))
	skipped = make([]types.OrganizeCourseSkipped, 0)

	type rankedFile struct {
		file    types.OrganizeCourseUploadFile
		index   int
		order   int
		hasRank bool
	}
	ranked := make([]rankedFile, 0, len(files))

	for i, file := range files {
		displayName := organizeCourseDisplayName(file)
		if isOrganizeCourseJunkFile(file) {
			skipped = append(skipped, types.OrganizeCourseSkipped{
				FileName: displayName,
				Reason:   "系统文件或隐藏文件，已跳过",
			})
			continue
		}
		order, hasRank := organizeCourseFileRank(filepath.Base(displayName))
		ranked = append(ranked, rankedFile{file: file, index: i, order: order, hasRank: hasRank})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if left.hasRank != right.hasRank {
			return left.hasRank
		}
		if left.hasRank && right.hasRank && left.order != right.order {
			return left.order < right.order
		}
		leftName := strings.ToLower(filepath.Base(organizeCourseDisplayName(left.file)))
		rightName := strings.ToLower(filepath.Base(organizeCourseDisplayName(right.file)))
		if leftName != rightName {
			return leftName < rightName
		}
		return left.index < right.index
	})

	for _, item := range ranked {
		accepted = append(accepted, item.file)
	}
	return accepted, skipped
}

func organizeCourseDisplayName(file types.OrganizeCourseUploadFile) string {
	if strings.TrimSpace(file.RelativePath) != "" {
		return strings.TrimSpace(file.RelativePath)
	}
	return strings.TrimSpace(file.FileName)
}

func isOrganizeCourseJunkFile(file types.OrganizeCourseUploadFile) bool {
	relative := organizeCourseDisplayName(file)
	base := filepath.Base(relative)
	if base == "" || strings.HasPrefix(base, ".") {
		return true
	}
	lower := strings.ToLower(strings.ReplaceAll(relative, "\\", "/"))
	return strings.HasPrefix(lower, "__macosx/") || strings.Contains(lower, "/__macosx/")
}

// organizeCourseFileRank extracts a leading numeric prefix such as "03" from
// "03_招生话术.mp4". Returns hasRank=false when the name does not start with a
// number, in which case the caller falls back to name ordering.
func organizeCourseFileRank(baseName string) (int, bool) {
	runes := []rune(strings.TrimSpace(baseName))
	end := 0
	for end < len(runes) && runes[end] >= '0' && runes[end] <= '9' {
		end++
	}
	if end == 0 || end > 4 {
		return 0, false
	}
	// The digits must be a position marker, not part of the word itself:
	// "3D打印" is a title, "03_xxx" is a position.
	if end < len(runes) && !isOrganizeCourseRankSeparator(runes[end]) {
		return 0, false
	}
	value, err := strconv.Atoi(string(runes[:end]))
	if err != nil {
		return 0, false
	}
	return value, true
}

func isOrganizeCourseRankSeparator(r rune) bool {
	switch r {
	case '_', '-', '.', ' ', '．', '、', '。':
		return true
	default:
		return false
	}
}

// organizeCourseLessonTitle prefers the AI-generated title but strips any
// leading ordering prefix so the outline reads "招生话术实操" and not
// "03_招生话术实操".
func organizeCourseLessonTitle(aiTitle, fileName string) string {
	fallback := strings.TrimSuffix(filepath.Base(strings.TrimSpace(fileName)), filepath.Ext(fileName))
	title := strings.TrimSpace(aiTitle)
	if title == "" {
		title = fallback
	}
	if stripped := strings.TrimSpace(stripOrganizeCourseRankPrefix(title)); stripped != "" {
		title = stripped
	}
	if title == "" {
		return fallback
	}
	return title
}

// stripOrganizeCourseRankPrefix removes a leading ordering marker such as
// "03_" / "03-" / "03." / "03 " from a name. Names that merely start with a
// digit ("3D打印") are left untouched.
func stripOrganizeCourseRankPrefix(name string) string {
	runes := []rune(name)
	end := 0
	for end < len(runes) && runes[end] >= '0' && runes[end] <= '9' {
		end++
	}
	if end == 0 || end > 4 || end >= len(runes) {
		return name
	}
	if !isOrganizeCourseRankSeparator(runes[end]) {
		return name
	}
	return string(runes[end+1:])
}

func isKnownOrganizeDiscoverCategory(category string) bool {
	for _, candidate := range types.OrganizeDiscoverCategories() {
		if candidate.Key == category || candidate.Label == category {
			return true
		}
	}
	return false
}
