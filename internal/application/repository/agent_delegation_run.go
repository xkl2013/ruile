package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type agentDelegationRunRepository struct {
	db *gorm.DB
}

func NewAgentDelegationRunRepository(db *gorm.DB) interfaces.AgentDelegationRunRepository {
	return &agentDelegationRunRepository{db: db}
}

func (r *agentDelegationRunRepository) Create(ctx context.Context, run *types.AgentDelegationRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *agentDelegationRunRepository) GetByID(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.AgentDelegationRun, error) {
	var run types.AgentDelegationRun
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}

func (r *agentDelegationRunRepository) Update(ctx context.Context, run *types.AgentDelegationRun) error {
	if run == nil {
		return errors.New("agent delegation run is required")
	}
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *agentDelegationRunRepository) List(
	ctx context.Context,
	tenantID uint64,
	parentSessionID string,
	memberAgentID string,
	limit int,
) ([]*types.AgentDelegationRun, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit)
	if value := strings.TrimSpace(parentSessionID); value != "" {
		query = query.Where("parent_session_id = ?", value)
	}
	if value := strings.TrimSpace(memberAgentID); value != "" {
		query = query.Where("member_agent_id = ?", value)
	}
	var runs []*types.AgentDelegationRun
	return runs, query.Find(&runs).Error
}
