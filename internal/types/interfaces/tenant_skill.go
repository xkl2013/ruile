package interfaces

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type TenantSkillRepository interface {
	Create(ctx context.Context, skill *types.TenantSkill) error
	Update(ctx context.Context, skill *types.TenantSkill) error
	List(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error)
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.TenantSkill, error)
	GetByName(ctx context.Context, tenantID uint64, name string) (*types.TenantSkill, error)
	SetEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error
	SetEnvVars(ctx context.Context, tenantID uint64, id string, envVars types.TenantSkillEnvVars) error
	Delete(ctx context.Context, tenantID uint64, id string) error
	CreateInstallJob(ctx context.Context, job *types.TenantSkillInstallJob) error
	UpdateInstallJob(ctx context.Context, job *types.TenantSkillInstallJob) error
	GetInstallJob(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillInstallJob, error)
	ListInstallJobs(ctx context.Context, tenantID uint64, limit int) ([]*types.TenantSkillInstallJob, error)
	ListInstallJobsByStatus(ctx context.Context, statuses []string, limit int) ([]*types.TenantSkillInstallJob, error)
}

// TenantSkillService manages tenant-owned skill bundles and exposes the
// materialized directories used by the existing local/docker skill runner.
type TenantSkillService interface {
	ListTenantSkills(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error)
	UploadTenantSkill(ctx context.Context, tenantID uint64, userID string, file *multipart.FileHeader) (*types.TenantSkill, error)
	UploadTenantSkillFromSource(ctx context.Context, tenantID uint64, userID, source string) (*types.TenantSkill, error)
	StartTenantSkillSourceInstall(ctx context.Context, tenantID uint64, userID, source string) (*types.TenantSkillInstallJob, error)
	ListTenantSkillInstallJobs(ctx context.Context, tenantID uint64, limit int) ([]*types.TenantSkillInstallJob, error)
	GetTenantSkillInstallJob(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillInstallJob, error)
	CancelTenantSkillInstall(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillInstallJob, error)
	RetryTenantSkillInstall(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillInstallJob, error)
	SetTenantSkillEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error
	SetTenantSkillEnvVars(ctx context.Context, tenantID uint64, id string, envVars map[string]string) error
	GetTenantSkillEnvVars(ctx context.Context, tenantID uint64, id string) (map[string]string, error)
	DeleteTenantSkill(ctx context.Context, tenantID uint64, id string) error
	ListTenantSkillFiles(ctx context.Context, tenantID uint64, id string) ([]string, error)
	ReadTenantSkillFile(ctx context.Context, tenantID uint64, id, path string) (io.ReadCloser, error)
	ListGlobalSkills(ctx context.Context) ([]*types.TenantSkill, error)
	UploadGlobalSkill(ctx context.Context, userID string, file *multipart.FileHeader) (*types.TenantSkill, error)
	UploadGlobalSkillFromSource(ctx context.Context, userID, source string) (*types.TenantSkill, error)
	StartGlobalSkillSourceInstall(ctx context.Context, userID, source string) (*types.TenantSkillInstallJob, error)
	ListGlobalSkillInstallJobs(ctx context.Context, limit int) ([]*types.TenantSkillInstallJob, error)
	GetGlobalSkillInstallJob(ctx context.Context, id string) (*types.TenantSkillInstallJob, error)
	CancelGlobalSkillInstall(ctx context.Context, id string) (*types.TenantSkillInstallJob, error)
	RetryGlobalSkillInstall(ctx context.Context, id string) (*types.TenantSkillInstallJob, error)
	SetGlobalSkillEnabled(ctx context.Context, id string, enabled bool) error
	SetGlobalSkillEnvVars(ctx context.Context, id string, envVars map[string]string) error
	GetGlobalSkillEnvVars(ctx context.Context, id string) (map[string]string, error)
	DeleteGlobalSkill(ctx context.Context, id string) error
	ListGlobalSkillFiles(ctx context.Context, id string) ([]string, error)
	ReadGlobalSkillFile(ctx context.Context, id, path string) (io.ReadCloser, error)
	MaterializeTenantSkillDirs(ctx context.Context, tenantID uint64) ([]string, error)
	ListRuntimeTenantSkills(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error)
	ValidateSkillSnapshot(ctx context.Context, tenantID uint64, manifest types.JSONMap) error
	RecoverPendingTenantSkillInstalls(ctx context.Context) error
	Handle(ctx context.Context, task *asynq.Task) error
}
