package file

import (
	"context"
	"io"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type catalogStub struct {
	resource *types.StoredResource
	ref      string
}

func (c *catalogStub) Register(
	_ context.Context,
	tenantID uint64,
	physicalPath string,
	meta interfaces.ResourceRegistration,
) (string, error) {
	c.resource = &types.StoredResource{
		ID:           "resource-1",
		Handle:       "AbCdEfGhIjKlMnOpQrStUv",
		TenantID:     tenantID,
		PhysicalPath: physicalPath,
		OriginalName: meta.OriginalName,
	}
	c.ref = types.BuildResourcePath(c.resource.Handle)
	return c.ref, nil
}

func (c *catalogStub) Resolve(_ context.Context, _ string) (*types.StoredResource, error) {
	return c.resource, nil
}

func (c *catalogStub) ResolvePath(_ context.Context, value string) (string, *types.StoredResource, error) {
	if value == c.ref {
		return c.resource.PhysicalPath, c.resource, nil
	}
	return value, nil, nil
}
func (c *catalogStub) Bind(context.Context, string, string, string, string) error { return nil }
func (c *catalogStub) MarkDeleted(context.Context, string) error                  { return nil }
func (c *catalogStub) CreateAccessGrant(context.Context, string, time.Duration) (string, error) {
	return "GrantTokenAbCdEfGhIjKl", nil
}

func (c *catalogStub) ResolveAccessGrant(context.Context, string) (*types.StoredResource, error) {
	return c.resource, nil
}

type physicalFileStub struct {
	savedPath string
	readPath  string
	urlPath   string
	fileURL   string
}

func (s *physicalFileStub) CheckConnectivity(context.Context) error { return nil }
func (s *physicalFileStub) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	return "", nil
}

func (s *physicalFileStub) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	return s.savedPath, nil
}

func (s *physicalFileStub) GetFile(_ context.Context, path string) (io.ReadCloser, error) {
	s.readPath = path
	return io.NopCloser(strings.NewReader("body")), nil
}
func (s *physicalFileStub) GetFileURL(_ context.Context, path string) (string, error) {
	s.urlPath = path
	return s.fileURL, nil
}
func (s *physicalFileStub) DeleteFile(context.Context, string) error { return nil }
func (s *physicalFileStub) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", nil
}

func TestResourceCatalogFileServiceReturnsReferenceAndResolvesReads(t *testing.T) {
	inner := &physicalFileStub{savedPath: "local://7/exports/a.png"}
	catalog := &catalogStub{}
	svc := NewResourceCatalogFileService(inner, catalog)

	ref, err := svc.SaveBytes(context.Background(), []byte("image"), 7, "a.png", false)
	require.NoError(t, err)
	require.Equal(t, "resource://AbCdEfGhIjKlMnOpQrStUv", ref)
	reader, err := svc.GetFile(context.Background(), ref)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, inner.savedPath, inner.readPath)
}

func TestResourceCatalogFileServiceReturnsShortExternalGrantURL(t *testing.T) {
	t.Setenv("APP_EXTERNAL_URL", "https://weknora.example.com/")
	inner := &physicalFileStub{savedPath: "local://7/exports/a.png"}
	catalog := &catalogStub{}
	svc := NewResourceCatalogFileService(inner, catalog)

	ref, err := svc.SaveBytes(context.Background(), []byte("image"), 7, "a.png", false)
	require.NoError(t, err)
	externalURL, err := svc.GetFileURL(context.Background(), ref)
	require.NoError(t, err)
	require.Equal(t, "https://weknora.example.com/r/GrantTokenAbCdEfGhIjKl", externalURL)
}

func TestResourceCatalogFileServiceReturnsProviderDirectURL(t *testing.T) {
	t.Setenv("APP_EXTERNAL_URL", "https://weknora.example.com/")
	inner := &physicalFileStub{
		savedPath: "oss://bucket/tenant/doc.pdf",
		fileURL:   "https://bucket.oss.example.com/tenant/doc.pdf?signature=abc",
	}
	catalog := &catalogStub{}
	svc := NewResourceCatalogFileService(inner, catalog)

	ref, err := svc.SaveBytes(context.Background(), []byte("pdf"), 7, "doc.pdf", false)
	require.NoError(t, err)
	directSvc, ok := svc.(interfaces.DirectFileURLService)
	require.True(t, ok)
	directURL, err := directSvc.GetDirectFileURL(context.Background(), ref)
	require.NoError(t, err)
	require.Equal(t, inner.fileURL, directURL)
	require.Equal(t, inner.savedPath, inner.urlPath)
}
