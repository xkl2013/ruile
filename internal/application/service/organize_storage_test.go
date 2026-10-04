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
	contextTenantID uint64
	filePath        string
}

func (s *recordingOrganizeMediaFileService) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	s.contextTenantID, _ = types.TenantIDFromContext(ctx)
	s.filePath = filePath
	return io.NopCloser(strings.NewReader("video-bytes")), nil
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
			fileURL: "https://cdn.example.test/course.mp4",
		},
	}
	resolver := &recordingOrganizeStorageResolver{fileService: fileService}
	svc.storageResolver = resolver

	courseID := types.NewOrganizeCourseID()
	output := &types.OrganizeOutput{
		TenantID:          ownerTenantID,
		UserID:            "owner",
		Title:             "公开课程第一讲",
		Status:            types.OrganizeOutputStatusReady,
		PublicContentType: types.OrganizePublicContentTypeCourse,
		Metadata: types.JSONMap{
			"file_path": "oss://course-bucket/course.mp4",
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
	require.Equal(t, "oss://course-bucket/course.mp4", fileService.filePath)
	require.NoError(t, reader.Close())

	mediaURL, mediaName, mediaType, err := svc.GetPublishedCourseLessonMediaURL(ctx, courseID, lesson.ID)
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.test/course.mp4", mediaURL)
	require.Equal(t, "course.mp4", mediaName)
	require.Equal(t, "video/mp4", mediaType)
	require.Equal(t, ownerTenantID, resolver.contextTenantID)
}
