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

	OrganizeJobModeSingle = "single"
	OrganizeJobModeBatch  = "batch"

	OrganizeJobBatchStatusQueued    = "queued"
	OrganizeJobBatchStatusRunning   = "running"
	OrganizeJobBatchStatusCompleted = "completed"
	OrganizeJobBatchStatusFallback  = "fallback"
	OrganizeJobBatchStatusFailed    = "failed"
	OrganizeJobBatchStatusCanceled  = "canceled"
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
	MarkdownTemplate   string         `json:"markdown_template" gorm:"type:text;not null;default:''"`
	ExpertIDs          StringArray    `json:"expert_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Spec               JSONMap        `json:"spec" gorm:"type:jsonb;not null;default:'{}'"`
	Status             string         `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	PublishedVersion   string         `json:"published_version" gorm:"type:varchar(32);not null;default:''"`
	PublishedAt        *time.Time     `json:"published_at,omitempty"`
	PublishedBy        string         `json:"published_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	ValidationResult   JSONMap        `json:"validation_result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
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
	if t.ValidationResult == nil {
		t.ValidationResult = JSONMap{}
	}
	return nil
}

// IsOrganizeInternalTemplateKey identifies recipes used by ingestion and
// upload pipelines rather than user-created organize configurations.
func IsOrganizeInternalTemplateKey(key string) bool {
	switch key {
	case "note_import_meta", "note_audio_transcribe", "output_card_meta":
		return true
	default:
		return false
	}
}

// OrganizeTemplateVersion stores the immutable snapshot used by a job.
type OrganizeTemplateVersion struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TemplateID  string    `json:"template_id" gorm:"type:varchar(36);not null;index"`
	TemplateKey string    `json:"template_key" gorm:"type:varchar(64);not null;index"`
	Version     string    `json:"version" gorm:"type:varchar(32);not null"`
	Snapshot    JSONMap   `json:"snapshot" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy   string    `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	ChangeNote  string    `json:"change_note,omitempty" gorm:"type:text;not null;default:''"`
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
	ID              string            `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64            `json:"tenant_id" gorm:"not null;index"`
	UserID          string            `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Name            string            `json:"name" gorm:"type:varchar(255);not null"`
	TemplateKey     string            `json:"template_key" gorm:"type:varchar(64);not null;index"`
	TargetServiceID string            `json:"target_service_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Instruction     string            `json:"instruction" gorm:"type:text;not null;default:''"`
	ExpertIDs       StringArray       `json:"expert_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Schedule        string            `json:"schedule" gorm:"type:varchar(32);not null;default:'manual';index"`
	Status          string            `json:"status" gorm:"type:varchar(32);not null;default:'active';index"`
	NextRunAt       *time.Time        `json:"next_run_at,omitempty" gorm:"index"`
	LastRunAt       *time.Time        `json:"last_run_at,omitempty"`
	Metadata        JSONMap           `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Template        *OrganizeTemplate `json:"template,omitempty" gorm:"-"`
	LatestJob       *OrganizeJob      `json:"latest_job,omitempty" gorm:"-"`
	JobCount        int64             `json:"job_count" gorm:"-"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `json:"deleted_at,omitempty" gorm:"index"`
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
	ID                string              `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID          uint64              `json:"tenant_id" gorm:"not null;index"`
	UserID            string              `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ConfigID          string              `json:"config_id" gorm:"type:varchar(36);not null;index"`
	TemplateKey       string              `json:"template_key" gorm:"type:varchar(64);not null;index"`
	TemplateVersion   string              `json:"template_version" gorm:"type:varchar(32);not null;default:''"`
	TargetServiceID   string              `json:"target_service_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Status            string              `json:"status" gorm:"type:varchar(32);not null;default:'queued';index"`
	Stage             string              `json:"stage" gorm:"type:varchar(64);not null;default:'queued'"`
	Progress          int                 `json:"progress" gorm:"not null;default:0"`
	Requirement       JSONMap             `json:"requirement" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryIDs         StringArray         `json:"memory_ids" gorm:"type:jsonb;not null;default:'[]'"`
	ModelID           string              `json:"model_id,omitempty" gorm:"type:varchar(64);not null;default:''"`
	PromptHash        string              `json:"prompt_hash,omitempty" gorm:"type:varchar(64);not null;default:''"`
	OutputID          string              `json:"output_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Summary           string              `json:"summary,omitempty" gorm:"type:text;not null;default:''"`
	Result            JSONMap             `json:"result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	ErrorMessage      string              `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	DedupeKey         string              `json:"-" gorm:"type:varchar(255);not null;default:'';index"`
	JobMode           string              `json:"job_mode" gorm:"type:varchar(16);not null;default:'single'"`
	SelectionSnapshot JSONMap             `json:"selection_snapshot,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	InputFingerprint  string              `json:"input_fingerprint,omitempty" gorm:"type:varchar(64);not null;default:'';index"`
	SelectedCount     int                 `json:"selected_count" gorm:"not null;default:0"`
	ReadyCount        int                 `json:"ready_count" gorm:"not null;default:0"`
	ProcessedCount    int                 `json:"processed_count" gorm:"not null;default:0"`
	FailedCount       int                 `json:"failed_count" gorm:"not null;default:0"`
	OverlapCount      int                 `json:"overlap_count" gorm:"not null;default:0"`
	BatchCount        int                 `json:"batch_count" gorm:"not null;default:0"`
	Coverage          JSONMap             `json:"coverage,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Batches           []*OrganizeJobBatch `json:"batches,omitempty" gorm:"-"`
	ScheduledFor      *time.Time          `json:"scheduled_for,omitempty"`
	StartedAt         *time.Time          `json:"started_at,omitempty"`
	FinishedAt        *time.Time          `json:"finished_at,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	DeletedAt         gorm.DeletedAt      `json:"deleted_at,omitempty" gorm:"index"`
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
	if j.JobMode == "" {
		j.JobMode = OrganizeJobModeSingle
	}
	if j.SelectionSnapshot == nil {
		j.SelectionSnapshot = JSONMap{}
	}
	if j.Coverage == nil {
		j.Coverage = JSONMap{}
	}
	return nil
}

// OrganizeJobBatch is an internal execution unit of a batch organize job.
type OrganizeJobBatch struct {
	ID               string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID           string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ParentJobID      string         `json:"parent_job_id" gorm:"type:varchar(36);not null;index"`
	BatchIndex       int            `json:"batch_index" gorm:"not null"`
	BatchCount       int            `json:"batch_count" gorm:"not null"`
	Status           string         `json:"status" gorm:"type:varchar(32);not null;default:'queued';index"`
	Stage            string         `json:"stage" gorm:"type:varchar(64);not null;default:'queued'"`
	Progress         int            `json:"progress" gorm:"not null;default:0"`
	MemoryIDs        StringArray    `json:"memory_ids" gorm:"type:jsonb;not null;default:'[]'"`
	InputChars       int            `json:"input_chars" gorm:"not null;default:0"`
	PromptHash       string         `json:"prompt_hash,omitempty" gorm:"type:varchar(64);not null;default:''"`
	Summary          string         `json:"summary,omitempty" gorm:"type:text;not null;default:''"`
	StructuredResult JSONMap        `json:"structured_result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Citations        JSONMap        `json:"citations,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	ErrorMessage     string         `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	RetryCount       int            `json:"retry_count" gorm:"not null;default:0"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	FinishedAt       *time.Time     `json:"finished_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (OrganizeJobBatch) TableName() string { return "organize_job_batches" }

func (b *OrganizeJobBatch) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if b.Status == "" {
		b.Status = OrganizeJobBatchStatusQueued
	}
	if b.Stage == "" {
		b.Stage = OrganizeJobBatchStatusQueued
	}
	if b.MemoryIDs == nil {
		b.MemoryIDs = StringArray{}
	}
	if b.StructuredResult == nil {
		b.StructuredResult = JSONMap{}
	}
	if b.Citations == nil {
		b.Citations = JSONMap{}
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

type OrganizeTemplateAdminQuery struct {
	Keyword  string
	Scene    string
	Status   string
	Page     int
	PageSize int
}

type OrganizeTemplateAdminInput struct {
	Key                string      `json:"key"`
	Name               string      `json:"name"`
	Scene              string      `json:"scene"`
	Description        string      `json:"description"`
	OutputLabel        string      `json:"output_label"`
	Icon               string      `json:"icon"`
	DefaultInstruction string      `json:"default_instruction"`
	MarkdownTemplate   string      `json:"markdown_template"`
	ExpertIDs          StringArray `json:"expert_ids"`
	Spec               JSONMap     `json:"spec"`
	SortOrder          int         `json:"sort_order"`
	ChangeNote         string      `json:"change_note"`
}

type OrganizeTemplatePreviewInput struct {
	MemoryIDs []string `json:"memory_ids"`
	Variables JSONMap  `json:"variables"`
}

type OrganizeTemplatePreview struct {
	TemplateKey      string   `json:"template_key"`
	Version          string   `json:"version"`
	Prompt           string   `json:"prompt"`
	MarkdownTemplate string   `json:"markdown_template"`
	PreviewMarkdown  string   `json:"preview_markdown"`
	Spec             JSONMap  `json:"spec"`
	Errors           []string `json:"errors"`
}

type OrganizeTemplateCompileInput struct {
	SourceMarkdown string `json:"source_markdown"`
}

type OrganizeTemplateCompileResult struct {
	Key                string   `json:"key"`
	Name               string   `json:"name"`
	Scene              string   `json:"scene"`
	Description        string   `json:"description"`
	OutputLabel        string   `json:"output_label"`
	DefaultInstruction string   `json:"default_instruction"`
	MarkdownTemplate   string   `json:"markdown_template"`
	Spec               JSONMap  `json:"spec"`
	Sections           []string `json:"sections"`
	Warnings           []string `json:"warnings"`
}

type OrganizeTemplateVersionQuery struct {
	TemplateKey string
	Page        int
	PageSize    int
}

type OrganizeConfigInput struct {
	Name            string      `json:"name"`
	TemplateKey     string      `json:"template_key"`
	TargetServiceID string      `json:"target_service_id,omitempty"`
	Instruction     string      `json:"instruction,omitempty"`
	ExpertIDs       StringArray `json:"expert_ids,omitempty"`
	Schedule        string      `json:"schedule,omitempty"`
	Metadata        JSONMap     `json:"metadata,omitempty"`
}

type OrganizeJobInput struct {
	ConfigID          string      `json:"config_id,omitempty"`
	MemoryIDs         StringArray `json:"memory_ids,omitempty"`
	ModelID           string      `json:"model_id,omitempty"`
	Requirement       string      `json:"requirement,omitempty"`
	AllowPartial      bool        `json:"allow_partial,omitempty"`
	BatchPolicy       string      `json:"batch_policy,omitempty"`
	ForceRerun        bool        `json:"force_rerun,omitempty"`
	SelectionSnapshot JSONMap     `json:"selection_snapshot,omitempty"`
}

type OrganizeRequirementInput struct {
	ConfigID     string      `json:"config_id"`
	TemplateKey  string      `json:"template_key,omitempty"`
	Text         string      `json:"text"`
	MemoryIDs    StringArray `json:"memory_ids,omitempty"`
	ModelID      string      `json:"model_id,omitempty"`
	AllowPartial bool        `json:"allow_partial,omitempty"`
	Confirmed    bool        `json:"confirmed,omitempty"`
}

type OrganizeRequirementPreview struct {
	ConfigID            string                    `json:"config_id"`
	TemplateKey         string                    `json:"template_key"`
	TemplateName        string                    `json:"template_name"`
	Scene               string                    `json:"scene"`
	NormalizedText      string                    `json:"normalized_text"`
	QueryPlan           OrganizeMemoryQueryPlan   `json:"query_plan"`
	MemoryIDs           StringArray               `json:"memory_ids,omitempty"`
	SelectedCount       int                       `json:"selected_count"`
	ReadyCount          int                       `json:"ready_count"`
	UnreadyCount        int                       `json:"unready_count"`
	OverlapCount        int                       `json:"overlap_count"`
	EstimatedBatchCount int                       `json:"estimated_batch_count"`
	SampleMemories      []OrganizeMemoryReference `json:"sample_memories,omitempty"`
	Ambiguities         []string                  `json:"ambiguities,omitempty"`
	Suggestions         []string                  `json:"suggestions,omitempty"`
	Warnings            []string                  `json:"warnings,omitempty"`
	NeedConfirmation    bool                      `json:"need_confirmation"`
}

type OrganizeMemoryQueryPlan struct {
	TimeField    string      `json:"time_field,omitempty"`
	OccurredFrom string      `json:"occurred_from,omitempty"`
	OccurredTo   string      `json:"occurred_to,omitempty"`
	Kinds        StringArray `json:"kinds,omitempty"`
	Keyword      string      `json:"keyword,omitempty"`
	Sources      StringArray `json:"sources,omitempty"`
	ReadyOnly    bool        `json:"ready_only,omitempty"`
	Timezone     string      `json:"timezone,omitempty"`
}

type OrganizeJobTaskPayload struct {
	TenantID uint64 `json:"tenant_id"`
	UserID   string `json:"user_id"`
	JobID    string `json:"job_id"`
}
