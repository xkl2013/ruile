package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OrganizeTemplateScopePlatform = "platform"
	OrganizeTemplateScopeTenant   = "tenant"
	OrganizeTemplateScopePersonal = "personal"

	OrganizeTemplateStatusDraft    = "draft"
	OrganizeTemplateStatusEnabled  = "enabled"
	OrganizeTemplateStatusDisabled = "disabled"

	OrganizeConfigStatusActive   = "active"
	OrganizeConfigStatusDisabled = "disabled"

	OrganizeScheduleManual  = "manual"
	OrganizeScheduleDaily   = "daily"
	OrganizeScheduleWeekly  = "weekly"
	OrganizeScheduleMonthly = "monthly"

	OrganizeJobStatusQueued    = "queued"
	OrganizeJobStatusRunning   = "running"
	OrganizeJobStatusRepairing = "repairing"
	OrganizeJobStatusCompleted = "completed"
	OrganizeJobStatusFallback  = "fallback"
	OrganizeJobStatusFailed    = "failed"
	OrganizeJobStatusCanceled  = "canceled"
)

func IsValidOrganizeSchedule(schedule string) bool {
	switch schedule {
	case OrganizeScheduleManual, OrganizeScheduleDaily, OrganizeScheduleWeekly, OrganizeScheduleMonthly:
		return true
	default:
		return false
	}
}

func IsTerminalOrganizeJobStatus(status string) bool {
	switch status {
	case OrganizeJobStatusCompleted, OrganizeJobStatusFallback, OrganizeJobStatusFailed, OrganizeJobStatusCanceled:
		return true
	default:
		return false
	}
}

// OrganizeTemplate is a published server-side organize recipe.
type OrganizeTemplate struct {
	ID                 string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64         `json:"tenant_id" gorm:"not null;default:0;index"`
	OwnerUserID        string         `json:"owner_user_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Scope              string         `json:"scope" gorm:"type:varchar(32);not null;default:'platform';index"`
	Key                string         `json:"key" gorm:"type:varchar(64);not null;index"`
	Name               string         `json:"name" gorm:"type:varchar(255);not null"`
	Scene              string         `json:"scene" gorm:"type:varchar(128);not null;default:'';index"`
	Description        string         `json:"description" gorm:"type:text;not null;default:''"`
	OutputLabel        string         `json:"output_label" gorm:"type:varchar(128);not null;default:''"`
	Icon               string         `json:"icon" gorm:"type:varchar(64);not null;default:''"`
	DefaultInstruction string         `json:"default_instruction" gorm:"type:text;not null;default:''"`
	ExpertIDs          StringArray    `json:"expert_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Spec               JSONMap        `json:"spec" gorm:"type:jsonb;not null;default:'{}'"`
	Status             string         `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	PublishedVersion   string         `json:"published_version" gorm:"type:varchar(32);not null;default:''"`
	SortOrder          int            `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (OrganizeTemplate) TableName() string { return "organize_templates" }

func (t *OrganizeTemplate) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Scope == "" {
		t.Scope = OrganizeTemplateScopePlatform
	}
	if t.Status == "" {
		t.Status = OrganizeTemplateStatusDraft
	}
	if t.ExpertIDs == nil {
		t.ExpertIDs = StringArray{}
	}
	if t.Spec == nil {
		t.Spec = JSONMap{}
	}
	return nil
}

// OrganizeTemplateVersion stores the immutable snapshot used by a job.
type OrganizeTemplateVersion struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TemplateID  string    `json:"template_id" gorm:"type:varchar(36);not null;index"`
	TemplateKey string    `json:"template_key" gorm:"type:varchar(64);not null;index"`
	Version     string    `json:"version" gorm:"type:varchar(32);not null"`
	Snapshot    JSONMap   `json:"snapshot" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy   string    `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OrganizeTemplateVersion) TableName() string { return "organize_template_versions" }

func (v *OrganizeTemplateVersion) BeforeCreate(_ *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.Snapshot == nil {
		v.Snapshot = JSONMap{}
	}
	return nil
}

// OrganizeConfig is a user-owned reusable organize configuration.
type OrganizeConfig struct {
	ID          string            `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64            `json:"tenant_id" gorm:"not null;index"`
	UserID      string            `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Name        string            `json:"name" gorm:"type:varchar(255);not null"`
	TemplateKey string            `json:"template_key" gorm:"type:varchar(64);not null;index"`
	Instruction string            `json:"instruction" gorm:"type:text;not null;default:''"`
	ExpertIDs   StringArray       `json:"expert_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Schedule    string            `json:"schedule" gorm:"type:varchar(32);not null;default:'manual';index"`
	Status      string            `json:"status" gorm:"type:varchar(32);not null;default:'active';index"`
	NextRunAt   *time.Time        `json:"next_run_at,omitempty" gorm:"index"`
	LastRunAt   *time.Time        `json:"last_run_at,omitempty"`
	Metadata    JSONMap           `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Template    *OrganizeTemplate `json:"template,omitempty" gorm:"-"`
	LatestJob   *OrganizeJob      `json:"latest_job,omitempty" gorm:"-"`
	JobCount    int64             `json:"job_count" gorm:"-"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeletedAt   gorm.DeletedAt    `json:"deleted_at,omitempty" gorm:"index"`
}

func (OrganizeConfig) TableName() string { return "organize_configs" }

func (c *OrganizeConfig) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.Schedule == "" {
		c.Schedule = OrganizeScheduleManual
	}
	if c.Status == "" {
		c.Status = OrganizeConfigStatusActive
	}
	if c.ExpertIDs == nil {
		c.ExpertIDs = StringArray{}
	}
	if c.Metadata == nil {
		c.Metadata = JSONMap{}
	}
	return nil
}

// OrganizeJob is the durable execution record for one organize run.
type OrganizeJob struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID          string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ConfigID        string         `json:"config_id" gorm:"type:varchar(36);not null;index"`
	TemplateKey     string         `json:"template_key" gorm:"type:varchar(64);not null;index"`
	TemplateVersion string         `json:"template_version" gorm:"type:varchar(32);not null;default:''"`
	Status          string         `json:"status" gorm:"type:varchar(32);not null;default:'queued';index"`
	Stage           string         `json:"stage" gorm:"type:varchar(64);not null;default:'queued'"`
	Progress        int            `json:"progress" gorm:"not null;default:0"`
	Requirement     JSONMap        `json:"requirement" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryIDs       StringArray    `json:"memory_ids" gorm:"type:jsonb;not null;default:'[]'"`
	ModelID         string         `json:"model_id,omitempty" gorm:"type:varchar(64);not null;default:''"`
	PromptHash      string         `json:"prompt_hash,omitempty" gorm:"type:varchar(64);not null;default:''"`
	OutputID        string         `json:"output_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Summary         string         `json:"summary,omitempty" gorm:"type:text;not null;default:''"`
	Result          JSONMap        `json:"result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	ErrorMessage    string         `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	DedupeKey       string         `json:"-" gorm:"type:varchar(255);not null;default:'';index"`
	ScheduledFor    *time.Time     `json:"scheduled_for,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (OrganizeJob) TableName() string { return "organize_jobs" }

func (j *OrganizeJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	if j.Status == "" {
		j.Status = OrganizeJobStatusQueued
	}
	if j.Stage == "" {
		j.Stage = OrganizeJobStatusQueued
	}
	if j.Requirement == nil {
		j.Requirement = JSONMap{}
	}
	if j.MemoryIDs == nil {
		j.MemoryIDs = StringArray{}
	}
	if j.Result == nil {
		j.Result = JSONMap{}
	}
	return nil
}

type OrganizeExpert struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OrganizeConfigQuery struct {
	TenantID uint64
	UserID   string
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

type OrganizeJobQuery struct {
	TenantID uint64
	UserID   string
	ConfigID string
	Status   string
	Page     int
	PageSize int
}

type OrganizeConfigInput struct {
	Name        string      `json:"name"`
	TemplateKey string      `json:"template_key"`
	Instruction string      `json:"instruction,omitempty"`
	ExpertIDs   StringArray `json:"expert_ids,omitempty"`
	Schedule    string      `json:"schedule,omitempty"`
	Metadata    JSONMap     `json:"metadata,omitempty"`
}

type OrganizeJobInput struct {
	ConfigID    string      `json:"config_id,omitempty"`
	MemoryIDs   StringArray `json:"memory_ids,omitempty"`
	ModelID     string      `json:"model_id,omitempty"`
	Requirement string      `json:"requirement,omitempty"`
}

type OrganizeJobTaskPayload struct {
	TenantID uint64 `json:"tenant_id"`
	UserID   string `json:"user_id"`
	JobID    string `json:"job_id"`
}
