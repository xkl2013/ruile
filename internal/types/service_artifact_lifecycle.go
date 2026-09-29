package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceArtifactLifecycleOperation records one idempotent lifecycle request.
// The row is immutable and keyed by the caller-provided idempotency key.
type ServiceArtifactLifecycleOperation struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id" gorm:"not null;index"`
	ServiceID       string    `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ArtifactID      string    `json:"artifact_id" gorm:"type:varchar(36);not null;index"`
	IdempotencyKey  string    `json:"idempotency_key" gorm:"type:varchar(128);not null"`
	FromLifecycle   string    `json:"from_lifecycle" gorm:"type:varchar(32);not null"`
	ToLifecycle     string    `json:"to_lifecycle" gorm:"type:varchar(32);not null"`
	ResultVersionID string    `json:"result_version_id" gorm:"type:varchar(36);not null;default:''"`
	CreatedBy       string    `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ServiceArtifactLifecycleOperation) TableName() string {
	return "service_artifact_lifecycle_operations"
}

func (o *ServiceArtifactLifecycleOperation) BeforeCreate(_ *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	o.ServiceID = strings.TrimSpace(o.ServiceID)
	o.ArtifactID = strings.TrimSpace(o.ArtifactID)
	o.IdempotencyKey = strings.TrimSpace(o.IdempotencyKey)
	o.FromLifecycle = strings.TrimSpace(o.FromLifecycle)
	o.ToLifecycle = strings.TrimSpace(o.ToLifecycle)
	o.ResultVersionID = strings.TrimSpace(o.ResultVersionID)
	o.CreatedBy = strings.TrimSpace(o.CreatedBy)
	return nil
}
