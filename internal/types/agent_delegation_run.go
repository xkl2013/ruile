package types

import "time"

const (
	AgentDelegationRunStatusRunning   = "running"
	AgentDelegationRunStatusSucceeded = "succeeded"
	AgentDelegationRunStatusFailed    = "failed"
)

// AgentDelegationRun is the durable execution record for one member Agent
// invocation created by delegate_agent.
type AgentDelegationRun struct {
	ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID          uint64     `gorm:"not null;index" json:"tenant_id"`
	ParentSessionID   string     `gorm:"type:varchar(128);not null;index" json:"parent_session_id"`
	ParentMessageID   string     `gorm:"type:varchar(128);not null;default:'';index" json:"parent_message_id"`
	MemberAgentID     string     `gorm:"type:varchar(128);not null;index" json:"member_agent_id"`
	Task              string     `gorm:"type:text;not null" json:"task"`
	Status            string     `gorm:"type:varchar(32);not null;index" json:"status"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	DurationMillis    int64      `gorm:"not null;default:0" json:"duration_millis"`
	InputTokens       int64      `gorm:"not null;default:0" json:"input_tokens"`
	CachedTokens      int64      `gorm:"not null;default:0" json:"cached_tokens"`
	OutputTokens      int64      `gorm:"not null;default:0" json:"output_tokens"`
	ReasoningTokens   int64      `gorm:"not null;default:0" json:"reasoning_tokens"`
	CallCount         int64      `gorm:"not null;default:0" json:"call_count"`
	BaseCostNanoUSD   int64      `gorm:"column:base_cost_nanousd;not null;default:0" json:"base_cost_nanousd"`
	RatedCostNanoUSD  int64      `gorm:"column:rated_cost_nanousd;not null;default:0" json:"rated_cost_nanousd"`
	BilledPointMicros int64      `gorm:"not null;default:0" json:"billed_point_micros"`
	UsageLedgerCount  int        `gorm:"not null;default:0" json:"usage_ledger_count"`
	OutputPreview     string     `gorm:"type:text;not null;default:''" json:"output_preview"`
	ErrorCode         string     `gorm:"type:varchar(64);not null;default:''" json:"error_code"`
	ErrorMessage      string     `gorm:"type:text;not null;default:''" json:"error_message"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (AgentDelegationRun) TableName() string { return "agent_delegation_runs" }

type AgentDelegationRunStartInput struct {
	TenantID        uint64
	ParentSessionID string
	ParentMessageID string
	MemberAgentID   string
	Task            string
}
