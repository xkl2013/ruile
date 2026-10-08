package file

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendScopedLocalURLRetainsBackendID(t *testing.T) {
	t.Setenv("SYSTEM_AES_KEY", "0123456789abcdef0123456789abcdef")
	inner := NewLocalFileService(t.TempDir(), "https://weknora.example.com/base")
	svc := NewBackendScopedFileService("backend-local-a", inner)

	path, err := svc.SaveBytes(context.Background(), []byte("hello"), 7, "exports/image.txt", false)
	require.NoError(t, err)
	assert.Contains(t, path, "storage://backend-local-a/local://")

	signed, err := svc.GetFileURL(context.Background(), path)
	require.NoError(t, err)
	u, err := url.Parse(signed)
	require.NoError(t, err)
	assert.Equal(t, path, u.Query().Get("file_path"))

	reader, err := svc.GetFile(context.Background(), path)
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestBackendScopedDirectURLBypassesResourceGrant(t *testing.T) {
	const (
		ref      = "resource://AbCdEfGhIjKlMnOpQrStUv"
		physical = "oss://private-bucket/42/audio.mp3"
		direct   = "https://private-bucket.oss.example.com/42/audio.mp3?signature=abc"
	)
	inner := NewResourceCatalogFileService(
		&physicalFileStub{fileURL: direct},
		&catalogStub{
			resource: &types.StoredResource{
				TenantID:     42,
				PhysicalPath: physical,
			},
		},
	)
	scoped := NewBackendScopedFileService("backend-oss-a", inner)
	directService, ok := scoped.(interfaces.DirectFileURLService)
	require.True(t, ok)

	got, err := directService.GetDirectFileURL(
		context.Background(),
		types.BuildStorageBackendPath("backend-oss-a", ref),
	)

	require.NoError(t, err)
	assert.Equal(t, direct, got)
}
