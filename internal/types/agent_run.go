package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AgentRunStatusQueued       = "queued"
	AgentRunStatusRunning      = "running"
	AgentRunStatusWaitingInput = "waiting_input"
	AgentRunStatusSucceeded    = "succeeded"
	AgentRunStatusFailed       = "failed"
	AgentRunStatusCancelled    = "cancelled"

	AgentRunPhaseIntake    = "intake"
	AgentRunPhasePlanning  = "planning"
	AgentRunPhaseDrafting  = "drafting"
	AgentRunPhaseReviewing = "reviewing"
	AgentRunPhaseRevising  = "revising"
	AgentRunPhasePackaging = "packaging"
	AgentRunPhaseCompleted = "completed"

	AgentRunStepStatusRunning   = "running"
	AgentRunStepStatusSucceeded = "succeeded"
	AgentRunStepStatusFailed    = "failed"

	AgentRunStepTypeIntake    = "intake"
	AgentRunStepTypePlanning  = "planning"
	AgentRunStepTypeDrafting  = "drafting"
	AgentRunStepTypeReviewing = "reviewing"
	AgentRunStepTypeRevising  = "revising"
	AgentRunStepTypePackaging = "packaging"

	AgentRunTypeServiceDailyReport   = "service_daily_report"
	AgentRunTypeServiceMemoryExtract = "service_memory_extract"
	AgentRunTypeExpertAgentTest      = "expert_agent_test"

	AgentRunErrorEnqueueFailed   = "agent_run_enqueue_failed"
	AgentRunErrorExecutionFailed = "agent_run_execution_failed"
	AgentRunErrorInvalidInput    = "agent_run_invalid_input"
	AgentRunErrorInvalidOutput   = "agent_run_invalid_output"
)

// AgentRun is the durable execution record for an asynchronous agent task.
// The Result field stores references to domain artifacts instead of the full
// artifact payload, which keeps run history small and lets each artifact keep
// its own lifecycle and access rules.
type AgentRun struct {
	ID                    string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID              uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID                string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ProfileID             string         `json:"profile_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	ParentRunID           string         `json:"parent_run_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	RequirementSnapshotID string         `json:"requirement_snapshot_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	RunType               string         `json:"run_type" gorm:"type:varchar(64);not null;index"`
	AgentRef              string         `json:"agent_ref" gorm:"type:varchar(128);not null;default:''"`
	AgentVersion          string         `json:"agent_version" gorm:"type:varchar(64);not null;default:''"`
	TriggerType           string         `json:"trigger_type" gorm:"type:varchar(64);not null;default:''"`
	TriggerID             string         `json:"trigger_id" gorm:"type:varchar(128);not null;default:'';index"`
	Status                string         `json:"status" gorm:"type:varchar(32);not null;default:'queued';index"`
	Phase                 string         `json:"phase,omitempty" gorm:"type:varchar(32);not null;default:'';index"`
	Input                 JSONMap        `json:"input,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Interaction           JSONMap        `json:"interaction,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Quality               JSONMap        `json:"quality,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Result                JSONMap        `json:"result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	ErrorCode             string         `json:"error_code,omitempty" gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage          string         `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	IdempotencyKey        string         `json:"idempotency_key,omitempty" gorm:"type:varchar(128);not null;default:'';index"`
	TaskID                string         `json:"task_id,omitempty" gorm:"type:varchar(160);not null;default:'';index"`
	Attempt               int            `json:"attempt" gorm:"not null;default:0"`
	QueuedAt              time.Time      `json:"queued_at"`
	ResumedAt             *time.Time     `json:"resumed_at,omitempty"`
	StartedAt             *time.Time     `json:"started_at,omitempty"`
	FinishedAt            *time.Time     `json:"finished_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentRun) TableName() string { return "agent_runs" }

func (r *AgentRun) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Status == "" {
		r.Status = AgentRunStatusQueued
	}
	if r.Input == nil {
		r.Input = JSONMap{}
	}
	if r.Interaction == nil {
		r.Interaction = JSONMap{}
	}
	if r.Quality == nil {
		r.Quality = JSONMap{}
	}
	if r.Result == nil {
		r.Result = JSONMap{}
	}
	if r.QueuedAt.IsZero() {
		r.QueuedAt = time.Now().UTC()
	}
	return nil
}

func IsValidAgentRunStatus(status string) bool {
	switch status {
	case AgentRunStatusQueued, AgentRunStatusRunning, AgentRunStatusWaitingInput, AgentRunStatusSucceeded,
		AgentRunStatusFailed, AgentRunStatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidAgentRunType(runType string) bool {
	switch runType {
	case AgentRunTypeServiceDailyReport, AgentRunTypeServiceMemoryExtract, AgentRunTypeExpertAgentTest:
		return true
	default:
		return false
	}
}

// ExpertAgentTestInput is persisted with an administrator-triggered expert
// test run. Test runs never create service cards or artifacts in business
// tables; their validated output remains on AgentRun for inspection.
type ExpertAgentTestInput struct {
	PackageID    string  `json:"package_id"`
	DefinitionID string  `json:"definition_id"`
	Prompt       string  `json:"prompt"`
	ModelID      string  `json:"model_id,omitempty"`
	ProfileID    string  `json:"profile_id,omitempty"`
	Answers      JSONMap `json:"answers,omitempty"`
	Feedback     string  `json:"feedback,omitempty"`
}

type AgentRunAnswersInput struct {
	Answers JSONMap `json:"answers"`
}

type AgentRunRegenerateInput struct {
	Feedback string `json:"feedback,omitempty"`
}

type ExpertIntakeQuestion struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"`
	Description string   `json:"description,omitempty"`
}

type ExpertIntakeInteraction struct {
	SchemaVersion string                 `json:"schema_version"`
	Questions     []ExpertIntakeQuestion `json:"questions"`
}

type AgentRunStep struct {
	ID         string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	RunID      string         `json:"run_id" gorm:"type:varchar(36);not null;index"`
	Sequence   int            `json:"sequence" gorm:"not null"`
	StepType   string         `json:"step_type" gorm:"type:varchar(32);not null;index"`
	Status     string         `json:"status" gorm:"type:varchar(32);not null;default:'running';index"`
	ModelID    string         `json:"model_id,omitempty" gorm:"type:varchar(128);not null;default:''"`
	Input      JSONMap        `json:"input,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Output     JSONMap        `json:"output,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Error      string         `json:"error,omitempty" gorm:"type:text;not null;default:''"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentRunStep) TableName() string { return "agent_run_steps" }

func (s *AgentRunStep) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Status == "" {
		s.Status = AgentRunStepStatusRunning
	}
	if s.Input == nil {
		s.Input = JSONMap{}
	}
	if s.Output == nil {
		s.Output = JSONMap{}
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now().UTC()
	}
	return nil
}

type AgentRunInputRevision struct {
	ID        string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	RunID     string         `json:"run_id" gorm:"type:varchar(36);not null;index"`
	Revision  int            `json:"revision" gorm:"not null"`
	Source    string         `json:"source" gorm:"type:varchar(32);not null;default:'request'"`
	Input     JSONMap        `json:"input" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentRunInputRevision) TableName() string { return "agent_run_input_revisions" }

func (r *AgentRunInputRevision) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Input == nil {
		r.Input = JSONMap{}
	}
	return nil
}

type AgentRequirementSnapshot struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	RunID       string         `json:"run_id" gorm:"type:varchar(36);not null;index"`
	Revision    int            `json:"revision" gorm:"not null"`
	Values      JSONMap        `json:"values" gorm:"column:snapshot_values;type:jsonb;not null;default:'{}'"`
	Assumptions StringArray    `json:"assumptions" gorm:"type:jsonb;not null;default:'[]'"`
	Missing     StringArray    `json:"missing" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentRequirementSnapshot) TableName() string { return "agent_requirement_snapshots" }

func (s *AgentRequirementSnapshot) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Values == nil {
		s.Values = JSONMap{}
	}
	if s.Assumptions == nil {
		s.Assumptions = StringArray{}
	}
	if s.Missing == nil {
		s.Missing = StringArray{}
	}
	return nil
}
