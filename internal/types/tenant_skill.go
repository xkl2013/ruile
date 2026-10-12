package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TenantSkillStatusReady    = "ready"
	TenantSkillSourceUpload   = "upload"
	TenantSkillSourceDirect   = "direct"
	TenantSkillSourceGitHub   = "github"
	TenantSkillSourceGitLab   = "gitlab"
	TenantSkillSourceRegistry = "registry"
	TenantSkillSourceSkillsSh = "skills-sh"
	// SystemSkillTenantID stores platform-wide Skills in the existing catalog
	// table without tying them to a workspace.
	SystemSkillTenantID uint64 = 0
)

type TenantSkillEnvVars map[string]string

func (e TenantSkillEnvVars) Value() (driver.Value, error) {
	if e == nil {
		return []byte("{}"), nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (e *TenantSkillEnvVars) Scan(value any) error {
	if value == nil {
		*e = nil
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported tenant skill env vars type %T", value)
	}
	if len(data) == 0 {
		*e = nil
		return nil
	}
	return json.Unmarshal(data, e)
}

const (
	TenantSkillInstallStatusPending    = "pending"
	TenantSkillInstallStatusRunning    = "running"
	TenantSkillInstallStatusSucceeded  = "succeeded"
	TenantSkillInstallStatusFailed     = "failed"
	TenantSkillInstallStatusCancelling = "cancelling"
	TenantSkillInstallStatusCancelled  = "cancelled"
)

// TenantSkill stores a tenant-owned skill bundle.
// The archive itself lives in the configured file service; the row keeps the
// validated metadata and the provider path needed to materialize it at runtime.
type TenantSkill struct {
	ID          string             `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64             `json:"tenant_id" gorm:"not null;index"`
	CreatedBy   string             `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	Name        string             `json:"name" gorm:"type:varchar(64);not null"`
	Description string             `json:"description" gorm:"type:text;not null;default:''"`
	Version     string             `json:"version" gorm:"type:varchar(64);not null;default:''"`
	BundlePath  string             `json:"-" gorm:"type:text;not null"`
	BundleSHA   string             `json:"bundle_sha256" gorm:"column:bundle_sha256;type:varchar(64);not null;default:''"`
	SourceType  string             `json:"source_type" gorm:"type:varchar(32);not null;default:'upload'"`
	SourceRef   string             `json:"source_ref" gorm:"type:text;not null;default:''"`
	EnvVars     TenantSkillEnvVars `json:"-" gorm:"type:jsonb;not null;default:'{}'"`
	Status      string             `json:"status" gorm:"type:varchar(32);not null;default:'ready';index"`
	Enabled     bool               `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `json:"-" gorm:"index"`
}

func (TenantSkill) TableName() string { return "tenant_skills" }

func (s *TenantSkill) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Status == "" {
		s.Status = TenantSkillStatusReady
	}
	if s.SourceType == "" {
		s.SourceType = TenantSkillSourceUpload
	}
	return nil
}

// TenantSkillInstallJob tracks an asynchronous source installation. It is
// deliberately separate from TenantSkill so a failed or cancelled attempt
// cannot make a previously ready catalog entry unavailable.
type TenantSkillInstallJob struct {
	ID              string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64     `json:"tenant_id" gorm:"not null;index"`
	CreatedBy       string     `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	SourceType      string     `json:"source_type" gorm:"type:varchar(32);not null"`
	SourceRef       string     `json:"source_ref" gorm:"type:text;not null"`
	SkillID         string     `json:"skill_id" gorm:"type:varchar(36);not null;default:'';index"`
	TaskID          string     `json:"task_id,omitempty" gorm:"type:varchar(160);not null;default:'';index"`
	Status          string     `json:"status" gorm:"type:varchar(32);not null;index"`
	Error           string     `json:"error" gorm:"type:text;not null;default:''"`
	Attempt         int        `json:"attempt" gorm:"not null;default:1"`
	CancelRequested bool       `json:"cancel_requested" gorm:"not null;default:false"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (TenantSkillInstallJob) TableName() string { return "tenant_skill_install_jobs" }

func (j *TenantSkillInstallJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	if j.Status == "" {
		j.Status = TenantSkillInstallStatusPending
	}
	if j.Attempt <= 0 {
		j.Attempt = 1
	}
	return nil
}
