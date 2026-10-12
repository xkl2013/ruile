package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type TenantSandboxConfigRepository interface {
	Create(ctx context.Context, config *types.TenantSandboxConfig) error
	Update(ctx context.Context, config *types.TenantSandboxConfig) error
	List(ctx context.Context, tenantID uint64) ([]*types.TenantSandboxConfig, error)
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.TenantSandboxConfig, error)
	GetActive(ctx context.Context, tenantID uint64) (*types.TenantSandboxConfig, error)
	SetActive(ctx context.Context, tenantID uint64, id string) error
	SetInactive(ctx context.Context, tenantID uint64, id string) error
	Delete(ctx context.Context, tenantID uint64, id string) error
}

type TenantSandboxConfigService interface {
	List(ctx context.Context, tenantID uint64) ([]*types.TenantSandboxConfig, error)
	Get(ctx context.Context, tenantID uint64, id string) (*types.TenantSandboxConfig, error)
	Create(ctx context.Context, tenantID uint64, config *types.TenantSandboxConfig) (*types.TenantSandboxConfig, error)
	Update(ctx context.Context, tenantID uint64, id string, config *types.TenantSandboxConfig) (*types.TenantSandboxConfig, error)
	Activate(ctx context.Context, tenantID uint64, id string) (*types.TenantSandboxConfig, error)
	Deactivate(ctx context.Context, tenantID uint64, id string) (*types.TenantSandboxConfig, error)
	Delete(ctx context.Context, tenantID uint64, id string) error
	Check(ctx context.Context, tenantID uint64, id string) (*types.TenantSandboxCheck, error)
	Resolve(ctx context.Context, tenantID uint64) (*types.TenantSandboxConfig, error)
}
