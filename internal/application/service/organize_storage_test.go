package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type recordingOrganizeStorageResolver struct {
	fileService     interfaces.FileService
	tenantID        uint64
	contextTenantID uint64
	backendID       string
	provider        string
}

func (r *recordingOrganizeStorageResolver) ResolveFileService(
	ctx context.Context,
	tenant *types.Tenant,
	backendID, provider, _ string,
) (interfaces.FileService, string, error) {
	if tenant != nil {
		r.tenantID = tenant.ID
	}
	r.contextTenantID, _ = types.TenantIDFromContext(ctx)
	r.backendID = backendID
	r.provider = provider
	return r.fileService, "oss", nil
}

func (r *recordingOrganizeStorageResolver) ResolveBackend(
	context.Context,
	*types.Tenant,
	string,
	string,
) (*types.StorageBackend, error) {
	return nil, nil
}

func TestOrganizeServiceUploadUsesTenantStorageResolver(t *testing.T) {
	backendID := "tenant-oss"
	tenant := &types.Tenant{
		ID:                      9,
		DefaultStorageBackendID: &backendID,
	}
	ctx := context.WithValue(context.Background(), types.TenantInfoContextKey, tenant)
	globalFileService := &stubOrganizeFileService{}
	tenantFileService := &stubOrganizeFileService{}
	resolver := &recordingOrganizeStorageResolver{fileService: tenantFileService}
	svc := newOrganizeUploadServiceForTest(
		t,
		&stubOrganizeModelService{},
		globalFileService,
		&stubOrganizeDocumentReader{},
	)
	svc.storageResolver = resolver

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"mobile-recording.m4a",
		"audio/mp4",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:  types.OrganizeMemoryKindAudio,
			Title: "移动端录音",
			Metadata: types.JSONMap{
				"audio_local_path": "/var/mobile/Containers/Data/mobile-recording.m4a",
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, 9, int(resolver.tenantID))
	require.Empty(t, resolver.backendID)
	require.Empty(t, resolver.provider)
	require.Equal(t, 0, globalFileService.saveCalls)
	require.Equal(t, 1, tenantFileService.saveCalls)
}

func TestOrganizeServiceDeleteUsesProviderFromHistoricalMobileFilePath(t *testing.T) {
	fileService := &stubOrganizeFileService{}
	resolver := &recordingOrganizeStorageResolver{fileService: fileService}
	svc := newOrganizeServiceForTest(t)
	svc.fileService = &stubOrganizeFileService{}
	svc.storageResolver = resolver
	ctx := context.WithValue(
		context.Background(),
		types.TenantInfoContextKey,
		&types.Tenant{ID: 7},
	)

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:  types.OrganizeMemoryKindAudio,
		Title: "历史移动端录音",
		Metadata: types.JSONMap{
			"file_path": "local://7/legacy/mobile-recording.mp3",
		},
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteMemory(ctx, 7, "user-a", memory.ID))
	require.Equal(t, "local", resolver.provider)
	require.Equal(t, []string{"local://7/legacy/mobile-recording.mp3"}, fileService.deletedPaths)
}

func TestOrganizeServiceStorageResolverUsesExplicitOwnerTenant(t *testing.T) {
	ownerTenantID := uint64(9)
	viewerTenantID := uint64(42)
	ctx := context.WithValue(
		context.Background(),
		types.TenantInfoContextKey,
		&types.Tenant{ID: ownerTenantID},
	)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, viewerTenantID)

	resolver := &recordingOrganizeStorageResolver{fileService: &stubOrganizeFileService{}}
	svc := newOrganizeServiceForTest(t)
	svc.storageResolver = resolver

	_, err := svc.resolveOrganizeFileService(ctx, ownerTenantID, "oss://bucket/course.mp4")
	require.NoError(t, err)
	require.Equal(t, ownerTenantID, resolver.tenantID)
	require.Equal(t, ownerTenantID, resolver.contextTenantID)
	require.Equal(t, "oss", resolver.provider)
}

type recordingOrganizeMediaFileService struct {
	stubOrganizeFileService
	contextTenantID       uint64
	filePath              string
	directContextTenantID uint64
	directFilePath        string
	directURL             string
}

func (s *recordingOrganizeMediaFileService) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	s.contextTenantID, _ = types.TenantIDFromContext(ctx)
	s.filePath = filePath
	return io.NopCloser(strings.NewReader("video-bytes")), nil
}

func (s *recordingOrganizeMediaFileService) GetDirectFileURL(
	ctx context.Context,
	filePath string,
) (string, error) {
	s.directContextTenantID, _ = types.TenantIDFromContext(ctx)
	s.directFilePath = filePath
	return s.directURL, nil
}

type directOrganizePreviewFileService struct {
	stubOrganizeFileService
	filePaths []string
}

func (s *directOrganizePreviewFileService) GetDirectFileURL(_ context.Context, filePath string) (string, error) {
	s.filePaths = append(s.filePaths, filePath)
	return "https://bucket.oss-cn-hangzhou.aliyuncs.com/memory.mp3?signature=test", nil
}

func TestOrganizeServiceMemoryPreviewUsesDirectStorageURL(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	fileService := &directOrganizePreviewFileService{}
	svc.fileService = fileService

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:  types.OrganizeMemoryKindAudio,
		Title: "家长沟通录音",
	})
	require.NoError(t, err)
	attachment := &types.OrganizeMemoryAttachment{
		TenantID:    7,
		UserID:      "user-a",
		MemoryID:    memory.ID,
		FileName:    "memory.mp3",
		MimeType:    "audio/mpeg",
		StoragePath: "oss://bucket/memory.mp3",
		Status:      types.OrganizeMemoryAttachmentStatusCompleted,
	}
	require.NoError(t, svc.repo.CreateMemoryAttachment(ctx, attachment))

	fileURL, fileName, mimeType, err := svc.GetMemoryAudioURL(ctx, 7, "user-a", memory.ID)
	require.NoError(t, err)
	require.Contains(t, fileURL, "oss-cn-hangzhou.aliyuncs.com")
	require.Equal(t, "memory.mp3", fileName)
	require.Equal(t, "audio/mpeg", mimeType)
	require.Equal(t, []string{attachment.StoragePath}, fileService.filePaths)

	attachmentURL, _, _, err := svc.GetMemoryAttachmentPreviewURL(
		ctx,
		7,
		"user-a",
		memory.ID,
		attachment.ID,
	)
	require.NoError(t, err)
	require.Equal(t, fileURL, attachmentURL)
	require.Equal(t, []string{attachment.StoragePath, attachment.StoragePath}, fileService.filePaths)
}

func TestOrganizeServiceMemoryPreviewFallsBackToStableMetadataPath(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	fileService := &directOrganizePreviewFileService{}
	svc.fileService = fileService

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:  types.OrganizeMemoryKindAudio,
		Title: "历史录音",
		Metadata: types.JSONMap{
			"audio_file_path": "oss://bucket/stable-memory.mp3",
			"audio_file_name": "stable-memory.mp3",
			"audio_mime_type": "audio/mpeg",
			"audio_url":       "https://bucket.oss-cn-hangzhou.aliyuncs.com/memory.mp3?Expires=1&Signature=expired",
		},
	})
	require.NoError(t, err)
	require.NoError(t, svc.repo.CreateMemoryAttachment(ctx, &types.OrganizeMemoryAttachment{
		TenantID:   7,
		UserID:     "user-a",
		MemoryID:   memory.ID,
		FileName:   "memory.mp3",
		MimeType:   "audio/mpeg",
		StorageURL: "https://bucket.oss-cn-hangzhou.aliyuncs.com/memory.mp3?Expires=1&Signature=expired",
		Status:     types.OrganizeMemoryAttachmentStatusCompleted,
	}))

	fileURL, fileName, mimeType, err := svc.GetMemoryAudioURL(ctx, 7, "user-a", memory.ID)
	require.NoError(t, err)
	require.Contains(t, fileURL, "oss-cn-hangzhou.aliyuncs.com")
	require.Equal(t, "stable-memory.mp3", fileName)
	require.Equal(t, "audio/mpeg", mimeType)
	require.Equal(t, []string{"oss://bucket/stable-memory.mp3"}, fileService.filePaths)
}

func TestOrganizePreviewRejectsApplicationFileProxy(t *testing.T) {
	require.False(t, isDirectOrganizePreviewURL(
		"https://api.example.test/files?file_path=oss%3A%2F%2Fbucket%2Fmemory.mp3",
	))
	require.False(t, isDirectOrganizePreviewURL(
		"https://api.example.test/api/v1/files/presigned?file_path=oss%3A%2F%2Fbucket%2Fmemory.mp3",
	))
	require.False(t, isDirectOrganizePreviewURL(
		"https://api.example.test/r/temporary-grant",
	))
	require.True(t, isDirectOrganizePreviewURL(
		"https://bucket.oss-cn-hangzhou.aliyuncs.com/memory.mp3?signature=test",
	))
}

func TestOrganizeServiceMediaReadUsesCourseOwnerTenant(t *testing.T) {
	ownerTenantID := uint64(9)
	viewerTenantID := uint64(42)
	ctx := context.WithValue(
		context.Background(),
		types.TenantInfoContextKey,
		&types.Tenant{ID: ownerTenantID},
	)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, viewerTenantID)

	svc := newOrganizeServiceForTest(t)
	fileService := &recordingOrganizeMediaFileService{
		stubOrganizeFileService: stubOrganizeFileService{
			fileURL: "https://api.example.test/r/application-proxy",
		},
		directURL: "https://course-bucket.oss-cn-beijing.aliyuncs.com/course.mp4?signature=test",
	}
	resolver := &recordingOrganizeStorageResolver{fileService: fileService}
	svc.storageResolver = resolver
	resourceCatalog, _ := newResourceCatalogForTest(t)
	svc.resourceCatalog = resourceCatalog
	resourceRef, err := resourceCatalog.Register(
		ctx,
		ownerTenantID,
		"storage://course-oss/oss://course-bucket/course.mp4",
		interfaces.ResourceRegistration{
			Kind:         "video",
			OriginalName: "course.mp4",
			MimeType:     "video/mp4",
		},
	)
	require.NoError(t, err)

	courseID := types.NewOrganizeCourseID()
	output := &types.OrganizeOutput{
		TenantID:          ownerTenantID,
		UserID:            "owner",
		Title:             "公开课程第一讲",
		Status:            types.OrganizeOutputStatusReady,
		PublicContentType: types.OrganizePublicContentTypeCourse,
		Metadata: types.JSONMap{
			"file_path": resourceRef,
			"file_name": "course.mp4",
			"mime_type": "video/mp4",
		},
	}
	require.NoError(t, svc.repo.CreateOutput(ctx, output, nil))
	course := &types.OrganizeCourse{
		ID:           courseID,
		TenantID:     ownerTenantID,
		UserID:       "owner",
		Source:       types.OrganizeCourseSourceOfficial,
		Title:        "公开课程",
		PublicStatus: types.OrganizePublicContentStatusPublished,
	}
	lesson := &types.OrganizeCourseLesson{
		CourseID:   courseID,
		TenantID:   ownerTenantID,
		OutputID:   output.ID,
		Title:      output.Title,
		LessonType: types.OrganizeCourseLessonTypeVideo,
		SortOrder:  0,
	}
	require.NoError(t, svc.repo.CreateCourse(ctx, course, []*types.OrganizeCourseLesson{lesson}))

	reader, fileName, mimeType, err := svc.OpenPublishedCourseLessonMedia(ctx, courseID, lesson.ID)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.Equal(t, "course.mp4", fileName)
	require.Equal(t, "video/mp4", mimeType)
	require.Equal(t, ownerTenantID, resolver.contextTenantID)
	require.Equal(t, ownerTenantID, fileService.contextTenantID)
	require.Equal(t, "course-oss", resolver.backendID)
	require.Equal(t, "oss", resolver.provider)
	require.Equal(t, resourceRef, fileService.filePath)
	require.NoError(t, reader.Close())

	mediaURL, mediaName, mediaType, err := svc.GetPublishedCourseLessonMediaURL(ctx, courseID, lesson.ID)
	require.NoError(t, err)
	require.Equal(t, fileService.directURL, mediaURL)
	require.Equal(t, "course.mp4", mediaName)
	require.Equal(t, "video/mp4", mediaType)
	require.Equal(t, ownerTenantID, resolver.contextTenantID)
	require.Equal(t, ownerTenantID, fileService.directContextTenantID)
	require.Equal(t, resourceRef, fileService.directFilePath)
}
