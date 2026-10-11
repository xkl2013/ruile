package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TenantSkillSnapshotStateBuilding   = "building"
	TenantSkillSnapshotStateActive     = "active"
	TenantSkillSnapshotStateSuperseded = "superseded"
)

// TenantSkillSnapshot is an immutable manifest of the tenant Skill catalog at
// one point in time. It is a logical runtime snapshot; provider-backed machine
// snapshots are intentionally outside the current local/docker implementation.
type TenantSkillSnapshot struct {
	ID               string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64     `json:"tenant_id" gorm:"not null;index"`
	Generation       int        `json:"generation" gorm:"not null"`
	ParentSnapshotID string     `json:"parent_snapshot_id,omitempty" gorm:"type:varchar(36);index"`
	State            string     `json:"state" gorm:"type:varchar(24);not null;index"`
	Manifest         JSONMap    `json:"manifest" gorm:"type:jsonb;not null"`
	ManifestSHA256   string     `json:"manifest_sha256" gorm:"type:varchar(64);not null"`
	CreatedAt        time.Time  `json:"created_at"`
	ActivatedAt      *time.Time `json:"activated_at,omitempty"`
	SupersededAt     *time.Time `json:"superseded_at,omitempty"`
}

func (TenantSkillSnapshot) TableName() string { return "tenant_skill_snapshots" }

func (s *TenantSkillSnapshot) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.State == "" {
		s.State = TenantSkillSnapshotStateBuilding
	}
	if s.Manifest == nil {
		s.Manifest = JSONMap{}
	}
	return nil
}
