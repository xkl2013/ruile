package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrOrganizeCourseInvalidSource  = errors.New("invalid course source")
	ErrOrganizeCourseNoValidFiles   = errors.New("no valid file found in the uploaded folder")
	ErrOrganizeCourseLessonNotReady = errors.New("course lesson content is not available")
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
// A folder routinely contains noise: .DS_Store, __MACOSX sidecars, a file type
// the pipeline cannot parse, or a video past the size ceiling. Those are
// collected in Skipped instead of aborting the whole course, because losing a
// 12-lesson upload to one stray file would be the worst possible behaviour.
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
		return nil, ErrOrganizeCourseNoValidFiles
	}

	course := &types.OrganizeCourse{
		ID:           courseID,
		TenantID:     tenantID,
		UserID:       userID,
		Source:       source,
		Title:        title,
		Summary:      trimMax(strings.TrimSpace(input.Summary), 0),
		Category:     category,
		CoverURL:     trimMax(strings.TrimSpace(input.CoverURL), organizeCourseMaxCoverURLLen),
		TeacherName:  trimMax(strings.TrimSpace(input.TeacherName), organizeCourseMaxTeacherRunes),
		TeacherTitle: trimMax(strings.TrimSpace(input.TeacherTitle), organizeCourseMaxTeacherTitle),
		PublicStatus: publicStatus,
	}
	if err := s.repo.CreateCourse(ctx, course, lessons); err != nil {
		// Best effort: the outputs are already persisted and are perfectly valid
		// standalone posts, but leaving them behind on a failed course creation
		// would make the folder look half-imported.
		for _, output := range createdOutputs {
			_ = s.deleteOrganizeStoredFile(ctx, tenantID, "course", output.ID, output.Metadata, "file_path", "storage_path")
			_ = s.repo.DeleteOutput(ctx, tenantID, userID, output.ID)
		}
		return nil, err
	}
	course.Lessons = lessons

	return &types.OrganizeCourseUploadResult{
		Course:  course,
		Skipped: skipped,
	}, nil
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
	maxSize := secutils.GetMaxFileSizeMBForUpload(cleanName, file.MimeType) * 1024 * 1024
	if maxSize > 0 && file.Size > maxSize {
		return nil, "", fmt.Errorf("file too large: %s", cleanName)
	}
	data, err := readOrganizeCourseUploadFile(file, maxSize)
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

	aiResult, aiModelID, aiStatus := s.generateOrganizeUploadAIResult(ctx, cleanName, outputType, content)
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

func readOrganizeCourseUploadFile(file types.OrganizeCourseUploadFile, maxSize int64) ([]byte, error) {
	if len(file.Data) > 0 {
		if maxSize > 0 && int64(len(file.Data)) > maxSize {
			return nil, fmt.Errorf("file too large: %s", strings.TrimSpace(file.FileName))
		}
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

	var source io.Reader = reader
	if maxSize > 0 {
		source = io.LimitReader(reader, maxSize+1)
	}
	data, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", strings.TrimSpace(file.FileName), err)
	}
	if maxSize > 0 && int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file too large: %s", strings.TrimSpace(file.FileName))
	}
	return data, nil
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
	lesson, err := s.repo.GetCourseLesson(ctx, course.ID, strings.TrimSpace(lessonID))
	if err != nil {
		return "", err
	}
	if lesson == nil || lesson.Output == nil || lesson.Output.Status == types.OrganizeOutputStatusArchived {
		return "", ErrOrganizeCourseLessonNotReady
	}
	return lesson.Output.Content, nil
}

// OpenPublishedCourseLessonMedia returns a reader only after verifying that
// the course is published and the lesson belongs to it. The storage path never
// leaves this method, so cross-tenant viewers receive the bytes without
// learning an internal locator.
func (s *organizeService) OpenPublishedCourseLessonMedia(
	ctx context.Context,
	courseID, lessonID string,
) (io.ReadCloser, string, string, error) {
	course, err := s.repo.GetCourse(ctx, strings.TrimSpace(courseID))
	if err != nil {
		return nil, "", "", err
	}
	if course == nil || course.PublicStatus != types.OrganizePublicContentStatusPublished {
		return nil, "", "", ErrOrganizeNotFound
	}
	lessons, err := s.repo.ListCourseLessonsWithOutputs(ctx, course.ID)
	if err != nil {
		return nil, "", "", err
	}
	var target *types.OrganizeCourseLesson
	for _, lesson := range lessons {
		if lesson != nil && lesson.ID == strings.TrimSpace(lessonID) {
			target = lesson
			break
		}
	}
	if target == nil || target.Output == nil || target.Output.Status == types.OrganizeOutputStatusArchived {
		return nil, "", "", ErrOrganizeCourseLessonNotReady
	}
	filePath := organizeStoredFilePath(target.Output.Metadata, "file_path", "storage_path")
	if filePath == "" {
		return nil, "", "", ErrOrganizeCourseLessonNotReady
	}
	fileService, err := s.resolveOrganizeFileService(ctx, course.TenantID, filePath)
	if err != nil {
		return nil, "", "", err
	}
	reader, err := fileService.GetFile(ctx, filePath)
	if err != nil {
		return nil, "", "", err
	}
	fileName := stringValue(target.Output.Metadata, "file_name")
	if fileName == "" {
		fileName = target.Title
	}
	return reader, fileName, stringValue(target.Output.Metadata, "mime_type"), nil
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
	return s.repo.ListCourses(ctx, query)
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
	return s.repo.DeleteCourse(ctx, course.TenantID, course.ID)
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
