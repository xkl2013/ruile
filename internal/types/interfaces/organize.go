package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type OrganizeRepository interface {
	ListTemplates(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeTemplate, error)
	GetTemplate(ctx context.Context, tenantID uint64, userID, key string) (*types.OrganizeTemplate, error)
	GetTemplateVersion(ctx context.Context, templateID, version string) (*types.OrganizeTemplateVersion, error)
	ListAdminTemplates(ctx context.Context, query types.OrganizeTemplateAdminQuery) ([]*types.OrganizeTemplate, int64, error)
	GetAdminTemplate(ctx context.Context, key string) (*types.OrganizeTemplate, error)
	CreateTemplate(ctx context.Context, template *types.OrganizeTemplate) error
	UpdateTemplate(ctx context.Context, template *types.OrganizeTemplate) error
	ListTemplateVersions(ctx context.Context, query types.OrganizeTemplateVersionQuery) ([]*types.OrganizeTemplateVersion, int64, error)
	CreateTemplateVersion(ctx context.Context, version *types.OrganizeTemplateVersion) error
	ListDiscoverCategories(ctx context.Context) ([]*types.OrganizeDiscoverCategoryRecord, error)
	ListAdminDiscoverCategories(ctx context.Context, query types.OrganizeDiscoverCategoryQuery) ([]*types.OrganizeDiscoverCategoryRecord, int64, error)
	CreateDiscoverCategory(ctx context.Context, category *types.OrganizeDiscoverCategoryRecord) error
	UpdateDiscoverCategory(ctx context.Context, category *types.OrganizeDiscoverCategoryRecord) error

	CreateConfig(ctx context.Context, config *types.OrganizeConfig) error
	GetConfig(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeConfig, error)
	UpdateConfig(ctx context.Context, config *types.OrganizeConfig) error
	DeleteConfig(ctx context.Context, tenantID uint64, userID, id string) error
	ListConfigs(ctx context.Context, query types.OrganizeConfigQuery) ([]*types.OrganizeConfig, int64, error)
	ListDueConfigs(ctx context.Context, now time.Time, limit int) ([]*types.OrganizeConfig, error)

	CreateJob(ctx context.Context, job *types.OrganizeJob) error
	GetJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	UpdateJob(ctx context.Context, job *types.OrganizeJob) error
	ListJobs(ctx context.Context, query types.OrganizeJobQuery) ([]*types.OrganizeJob, int64, error)
	ListMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) ([]*types.OrganizeMemory, error)

	CreateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	GetTenantMemory(ctx context.Context, tenantID uint64, id string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	CreateMemoryAttachment(ctx context.Context, attachment *types.OrganizeMemoryAttachment) error
	GetMemoryAttachment(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemoryAttachment, error)
	GetTenantMemoryAttachment(ctx context.Context, tenantID uint64, id string) (*types.OrganizeMemoryAttachment, error)
	UpdateMemoryAttachment(ctx context.Context, attachment *types.OrganizeMemoryAttachment) error
	ListMemoryAttachments(ctx context.Context, tenantID uint64, userID, memoryID string) ([]*types.OrganizeMemoryAttachment, error)
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)
	CountMemoriesByKind(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)
	CountMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) (int64, error)
	CountTenantMemoriesByIDs(ctx context.Context, tenantID uint64, ids []string) (int64, error)

	CreateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error
	GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error)
	UpdateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error
	DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error
	ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)
	ListOutputFacets(ctx context.Context, query types.OrganizeListQuery) (*types.OrganizeOutputFacets, error)
	GetOutputByID(ctx context.Context, id string) (*types.OrganizeOutput, error)
	ListPublicContents(ctx context.Context, query types.OrganizePublicContentQuery) ([]*types.OrganizeOutput, int64, error)
	UpdatePublicContent(ctx context.Context, output *types.OrganizeOutput) error
	CountOutputsByStatus(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)

	CreateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error
	GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error)
	UpdateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error
	DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error
	ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error)
	CountSproutReportsByStage(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)

	CreateCourse(ctx context.Context, course *types.OrganizeCourse, lessons []*types.OrganizeCourseLesson) error
	AppendCourseLesson(ctx context.Context, course *types.OrganizeCourse, lesson *types.OrganizeCourseLesson) error
	UpdateCourseLesson(ctx context.Context, course *types.OrganizeCourse, lesson *types.OrganizeCourseLesson) error
	DeleteCourseLesson(ctx context.Context, course *types.OrganizeCourse, lesson *types.OrganizeCourseLesson) error
	GetCourse(ctx context.Context, id string) (*types.OrganizeCourse, error)
	UpdateCourse(ctx context.Context, course *types.OrganizeCourse) error
	UpdateCourseDiscovery(ctx context.Context, id string, input types.OrganizeCourseDiscoveryInput) (*types.OrganizeCourse, error)
	UpdateCoursePublicStatus(ctx context.Context, id, status string) (*types.OrganizeCourse, error)
	UpdateCourseVisibility(ctx context.Context, id, visibilityScope string, organizationIDs []string, createdBy string) (*types.OrganizeCourse, error)
	ReplaceCourseSharedSpaces(ctx context.Context, courseID string, organizationIDs []string, createdBy string) error
	ListCourseSharedSpaceIDs(ctx context.Context, courseID string) ([]string, error)
	DeleteCourse(ctx context.Context, tenantID uint64, id string) error
	ListCourses(ctx context.Context, query types.OrganizeCourseQuery) ([]*types.OrganizeCourse, int64, error)
	GetCourseStats(ctx context.Context, query types.OrganizeCourseQuery) (*types.OrganizeCourseStats, error)
	ListCourseLessons(ctx context.Context, courseID string) ([]*types.OrganizeCourseLesson, error)
	// ListCourseLessonsWithOutputs hydrates each lesson's Output WITHOUT its
	// body columns; use GetCourseLesson when the content itself is needed.
	ListCourseLessonsWithOutputs(ctx context.Context, courseID string) ([]*types.OrganizeCourseLesson, error)
	GetCourseLesson(ctx context.Context, courseID, lessonID string) (*types.OrganizeCourseLesson, error)
}

type OrganizeService interface {
	ListTemplates(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeTemplate, error)
	GetTemplate(ctx context.Context, tenantID uint64, userID, key string) (*types.OrganizeTemplate, error)
	ListExperts(ctx context.Context, tenantID uint64, userID string) ([]types.OrganizeExpert, error)
	ListAdminTemplates(ctx context.Context, query types.OrganizeTemplateAdminQuery) ([]*types.OrganizeTemplate, int64, error)
	GetAdminTemplate(ctx context.Context, key string) (*types.OrganizeTemplate, error)
	CreateAdminTemplate(ctx context.Context, actorID string, input types.OrganizeTemplateAdminInput) (*types.OrganizeTemplate, error)
	UpdateAdminTemplate(ctx context.Context, key, actorID string, input types.OrganizeTemplateAdminInput) (*types.OrganizeTemplate, error)
	PublishAdminTemplate(ctx context.Context, key, actorID, changeNote string) (*types.OrganizeTemplate, error)
	DisableAdminTemplate(ctx context.Context, key, actorID string) (*types.OrganizeTemplate, error)
	ListAdminTemplateVersions(ctx context.Context, query types.OrganizeTemplateVersionQuery) ([]*types.OrganizeTemplateVersion, int64, error)
	RollbackAdminTemplate(ctx context.Context, key, version, actorID, changeNote string) (*types.OrganizeTemplate, error)
	PreviewAdminTemplate(ctx context.Context, key string, input types.OrganizeTemplatePreviewInput) (*types.OrganizeTemplatePreview, error)
	ListDiscoverCategories(ctx context.Context) ([]*types.OrganizeDiscoverCategoryRecord, error)
	ListAdminDiscoverCategories(ctx context.Context, query types.OrganizeDiscoverCategoryQuery) ([]*types.OrganizeDiscoverCategoryRecord, int64, error)
	CreateAdminDiscoverCategory(ctx context.Context, input types.OrganizeDiscoverCategoryInput) (*types.OrganizeDiscoverCategoryRecord, error)
	UpdateAdminDiscoverCategory(ctx context.Context, key string, input types.OrganizeDiscoverCategoryInput) (*types.OrganizeDiscoverCategoryRecord, error)
	DisableAdminDiscoverCategory(ctx context.Context, key string) (*types.OrganizeDiscoverCategoryRecord, error)

	CreateConfig(ctx context.Context, tenantID uint64, userID string, input types.OrganizeConfigInput) (*types.OrganizeConfig, error)
	GetConfig(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeConfig, error)
	UpdateConfig(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeConfigInput) (*types.OrganizeConfig, error)
	DeleteConfig(ctx context.Context, tenantID uint64, userID, id string) error
	ListConfigs(ctx context.Context, query types.OrganizeConfigQuery) ([]*types.OrganizeConfig, int64, error)
	RunConfig(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeJobInput) (*types.OrganizeJob, error)

	CreateJob(ctx context.Context, tenantID uint64, userID string, input types.OrganizeJobInput) (*types.OrganizeJob, error)
	PreviewOrganizeRequirement(ctx context.Context, tenantID uint64, userID string, input types.OrganizeRequirementInput) (*types.OrganizeRequirementPreview, error)
	ConfirmOrganizeRequirement(ctx context.Context, tenantID uint64, userID string, input types.OrganizeRequirementInput) (*types.OrganizeJob, error)
	GetJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	ListJobs(ctx context.Context, query types.OrganizeJobQuery) ([]*types.OrganizeJob, int64, error)
	RetryJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	CancelJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	ProcessOrganizeJob(ctx context.Context, task *asynq.Task) error
	RunDueConfigs(ctx context.Context, now time.Time) error

	CreateMemory(ctx context.Context, tenantID uint64, userID string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	CreateMemoryFromUpload(ctx context.Context, tenantID uint64, userID, fileName, mimeType string, data []byte, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	CreateMemoryFromUploads(ctx context.Context, tenantID uint64, userID string, uploads []types.OrganizeMemoryUpload, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	RetryMemoryAttachment(ctx context.Context, tenantID uint64, userID, memoryID, attachmentID string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)
	ProcessMemoryTranscribe(ctx context.Context, task *asynq.Task) error

	CreateOutput(ctx context.Context, tenantID uint64, userID string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	CreateOutputFromUpload(ctx context.Context, tenantID uint64, userID, fileName, mimeType string, data []byte) (*types.OrganizeOutput, error)
	GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error)
	UpdateOutput(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error
	ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)
	ListOutputFacets(ctx context.Context, query types.OrganizeListQuery) (*types.OrganizeOutputFacets, error)
	GetOutputCitation(ctx context.Context, tenantID uint64, userID, outputID, ref string) (*types.OrganizeMemory, bool, error)
	ListPendingAssignments(ctx context.Context, tenantID uint64, userID string, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)
	AssignOutputToService(ctx context.Context, tenantID uint64, userID, outputID, serviceID string) (*types.OrganizeOutput, error)
	ListAdminPublicContents(ctx context.Context, query types.OrganizePublicContentQuery) ([]*types.OrganizeOutput, int64, error)
	GetAdminPublicContent(ctx context.Context, id string) (*types.OrganizeOutput, error)
	UpdateAdminPublicContent(ctx context.Context, id string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	ModeratePublicContent(ctx context.Context, id, status, reviewNote string) (*types.OrganizeOutput, error)

	CreateSproutReport(ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	CreateSproutReportFromMemory(ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutFromMemoryInput) (*types.OrganizeSproutReport, error)
	GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error)
	UpdateSproutReport(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error
	ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error)

	GetDiscover(ctx context.Context, tenantID uint64, userID string, query types.OrganizeDiscoverQuery) (*types.OrganizeDiscover, error)
	GetOverview(ctx context.Context, tenantID uint64, userID string) (*types.OrganizeOverview, error)

	CreateCourseFromFolder(
		ctx context.Context,
		tenantID uint64,
		userID string,
		input types.OrganizeCourseUploadInput,
		files []types.OrganizeCourseUploadFile,
	) (*types.OrganizeCourseUploadResult, error)
	CreateCourse(
		ctx context.Context,
		tenantID uint64,
		userID string,
		input types.OrganizeCourseUploadInput,
	) (*types.OrganizeCourse, error)
	AppendCourseLessonFromUpload(
		ctx context.Context,
		tenantID uint64,
		userID string,
		courseID string,
		file types.OrganizeCourseUploadFile,
	) (*types.OrganizeCourseUploadResult, error)
	UpdateCourseLesson(
		ctx context.Context,
		courseID string,
		lessonID string,
		input types.OrganizeCourseLessonUpdateInput,
	) (*types.OrganizeCourseLesson, error)
	DeleteCourseLesson(ctx context.Context, courseID string, lessonID string) error
	GetCourse(ctx context.Context, id string) (*types.OrganizeCourse, error)
	GetPublishedCourse(ctx context.Context, id string) (*types.OrganizeCourse, error)
	ListCourses(ctx context.Context, query types.OrganizeCourseQuery) ([]*types.OrganizeCourse, int64, error)
	ListPublishedCourses(ctx context.Context, query types.OrganizeCourseQuery) ([]*types.OrganizeCourse, int64, error)
	UpdateCourseDiscovery(ctx context.Context, id string, input types.OrganizeCourseDiscoveryInput) (*types.OrganizeCourse, error)
	GetCourseStats(ctx context.Context, query types.OrganizeCourseQuery) (*types.OrganizeCourseStats, error)
	OpenPublishedCourseLessonMedia(ctx context.Context, courseID, lessonID string) (io.ReadCloser, string, string, error)
	OpenPublishedCourseCover(ctx context.Context, courseID string) (io.ReadCloser, string, string, error)
	GetPublishedCourseLessonMediaURL(ctx context.Context, courseID, lessonID string) (string, string, string, error)
	GetPublishedCourseLessonContent(ctx context.Context, courseID, lessonID string) (string, error)
	UpdateCoursePublicStatus(ctx context.Context, id, status string) (*types.OrganizeCourse, error)
	UpdateCourseVisibility(ctx context.Context, id string, input types.OrganizeCourseVisibilityInput) (*types.OrganizeCourse, error)
	DeleteCourse(ctx context.Context, id string) error
}
