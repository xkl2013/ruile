package interfaces

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/Tencent/WeKnora/internal/types"
)

type TenantSkillRepository interface {
	Create(ctx context.Context, skill *types.TenantSkill) error
	List(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error)
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.TenantSkill, error)
	GetByName(ctx context.Context, tenantID uint64, name string) (*types.TenantSkill, error)
	SetEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error
	Delete(ctx context.Context, tenantID uint64, id string) error
}

// TenantSkillService manages tenant-owned skill bundles and exposes the
// materialized directories used by the existing local/docker skill runner.
type TenantSkillService interface {
	ListTenantSkills(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error)
	UploadTenantSkill(ctx context.Context, tenantID uint64, userID string, file *multipart.FileHeader) (*types.TenantSkill, error)
	SetTenantSkillEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error
	DeleteTenantSkill(ctx context.Context, tenantID uint64, id string) error
	ListTenantSkillFiles(ctx context.Context, tenantID uint64, id string) ([]string, error)
	ReadTenantSkillFile(ctx context.Context, tenantID uint64, id, path string) (io.ReadCloser, error)
	ListGlobalSkills(ctx context.Context) ([]*types.TenantSkill, error)
	UploadGlobalSkill(ctx context.Context, userID string, file *multipart.FileHeader) (*types.TenantSkill, error)
	SetGlobalSkillEnabled(ctx context.Context, id string, enabled bool) error
	DeleteGlobalSkill(ctx context.Context, id string) error
	ListGlobalSkillFiles(ctx context.Context, id string) ([]string, error)
	ReadGlobalSkillFile(ctx context.Context, id, path string) (io.ReadCloser, error)
	MaterializeTenantSkillDirs(ctx context.Context, tenantID uint64) ([]string, error)
}
