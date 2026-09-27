package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TenantSkillStatusReady = "ready"
	// SystemSkillTenantID stores platform-wide Skills in the existing catalog
	// table without tying them to a workspace.
	SystemSkillTenantID uint64 = 0
)

// TenantSkill stores a tenant-owned skill bundle.
// The archive itself lives in the configured file service; the row keeps the
// validated metadata and the provider path needed to materialize it at runtime.
type TenantSkill struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id" gorm:"not null;index"`
	CreatedBy   string         `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	Name        string         `json:"name" gorm:"type:varchar(64);not null"`
	Description string         `json:"description" gorm:"type:text;not null;default:''"`
	Version     string         `json:"version" gorm:"type:varchar(64);not null;default:''"`
	BundlePath  string         `json:"-" gorm:"type:text;not null"`
	BundleSHA   string         `json:"bundle_sha256" gorm:"column:bundle_sha256;type:varchar(64);not null;default:''"`
	Status      string         `json:"status" gorm:"type:varchar(32);not null;default:'ready';index"`
	Enabled     bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (TenantSkill) TableName() string { return "tenant_skills" }

func (s *TenantSkill) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Status == "" {
		s.Status = TenantSkillStatusReady
	}
	return nil
}
