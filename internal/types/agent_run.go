package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AgentRunStatusQueued    = "queued"
	AgentRunStatusRunning   = "running"
	AgentRunStatusSucceeded = "succeeded"
	AgentRunStatusFailed    = "failed"
	AgentRunStatusCancelled = "cancelled"

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
	ID             string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID       uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID         string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ProfileID      string         `json:"profile_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	RunType        string         `json:"run_type" gorm:"type:varchar(64);not null;index"`
	AgentRef       string         `json:"agent_ref" gorm:"type:varchar(128);not null;default:''"`
	AgentVersion   string         `json:"agent_version" gorm:"type:varchar(64);not null;default:''"`
	TriggerType    string         `json:"trigger_type" gorm:"type:varchar(64);not null;default:''"`
	TriggerID      string         `json:"trigger_id" gorm:"type:varchar(128);not null;default:'';index"`
	Status         string         `json:"status" gorm:"type:varchar(32);not null;default:'queued';index"`
	Input          JSONMap        `json:"input,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Result         JSONMap        `json:"result,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	ErrorCode      string         `json:"error_code,omitempty" gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage   string         `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	IdempotencyKey string         `json:"idempotency_key,omitempty" gorm:"type:varchar(128);not null;default:'';index"`
	TaskID         string         `json:"task_id,omitempty" gorm:"type:varchar(160);not null;default:'';index"`
	Attempt        int            `json:"attempt" gorm:"not null;default:0"`
	QueuedAt       time.Time      `json:"queued_at"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	FinishedAt     *time.Time     `json:"finished_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
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
	case AgentRunStatusQueued, AgentRunStatusRunning, AgentRunStatusSucceeded,
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
	PackageID    string `json:"package_id"`
	DefinitionID string `json:"definition_id"`
	Prompt       string `json:"prompt"`
	ModelID      string `json:"model_id,omitempty"`
	ProfileID    string `json:"profile_id,omitempty"`
}
