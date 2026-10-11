package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type TenantSkillSnapshotRepository interface {
	Create(ctx context.Context, snapshot *types.TenantSkillSnapshot) error
	List(ctx context.Context, tenantID uint64, limit int) ([]*types.TenantSkillSnapshot, error)
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillSnapshot, error)
	GetActive(ctx context.Context, tenantID uint64) (*types.TenantSkillSnapshot, error)
	Activate(ctx context.Context, tenantID uint64, id string, activatedAt, supersededAt time.Time) error
}

type TenantSkillSnapshotService interface {
	List(ctx context.Context, tenantID uint64, limit int) ([]*types.TenantSkillSnapshot, error)
	Get(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillSnapshot, error)
	Create(ctx context.Context, tenantID uint64) (*types.TenantSkillSnapshot, error)
	Activate(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillSnapshot, error)
	Rollback(ctx context.Context, tenantID uint64, id string) (*types.TenantSkillSnapshot, error)
	Verify(ctx context.Context, tenantID uint64, id string) (bool, error)
}
