package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type recordingOrganizeStorageResolver struct {
	fileService interfaces.FileService
	tenantID    uint64
	backendID   string
	provider    string
}

func (r *recordingOrganizeStorageResolver) ResolveFileService(
	_ context.Context,
	tenant *types.Tenant,
	backendID, provider, _ string,
) (interfaces.FileService, string, error) {
	if tenant != nil {
		r.tenantID = tenant.ID
	}
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
