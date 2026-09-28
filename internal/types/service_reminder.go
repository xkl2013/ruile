package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceReminderStatusCandidate         = "candidate"
	ServiceReminderStatusPending           = "pending"
	ServiceReminderStatusGenerated         = "generated"
	ServiceReminderStatusConfirmed         = "confirmed"
	ServiceReminderStatusCompleted         = "completed"
	ServiceReminderStatusIgnored           = "ignored"
	ServiceReminderStatusSnoozed           = "snoozed"
	ServiceReminderStatusStale             = "stale"
	ServiceReminderStatusRecomputeRequired = "recompute_required"

	ServiceReminderPriorityHigh   = "high"
	ServiceReminderPriorityMedium = "medium"
	ServiceReminderPriorityLow    = "low"
)

// ServiceReminder is a service-scoped work item. The legacy profile_id and
// user_id columns remain populated for compatibility with the old service
// module, while service_id is the new ownership boundary.
type ServiceReminder struct {
	ID                string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID          uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID            string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ServiceID         string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ProfileID         string         `json:"profile_id" gorm:"type:varchar(36);not null;index"`
	ParentReminderID  string         `json:"parent_reminder_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Depth             int            `json:"depth" gorm:"not null;default:0"`
	SubjectID         string         `json:"subject_id" gorm:"type:varchar(36);not null;default:'';index"`
	AgentDomain       string         `json:"agent_domain" gorm:"type:varchar(64);not null;default:''"`
	Title             string         `json:"title" gorm:"type:varchar(512);not null"`
	Summary           string         `json:"summary" gorm:"type:text;not null;default:''"`
	Status            string         `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	Priority          string         `json:"priority" gorm:"type:varchar(16);not null;default:'medium';index"`
	DueAt             *time.Time     `json:"due_at,omitempty"`
	DueText           string         `json:"due_text" gorm:"type:varchar(64);not null;default:''"`
	Stage             string         `json:"stage" gorm:"type:varchar(128);not null;default:''"`
	Channel           string         `json:"channel" gorm:"type:varchar(128);not null;default:''"`
	DecisionRole      string         `json:"decision_role" gorm:"type:varchar(128);not null;default:''"`
	RiskLabel         string         `json:"risk_label" gorm:"type:varchar(128);not null;default:''"`
	AssistReason      string         `json:"assist_reason" gorm:"type:text;not null;default:''"`
	PrimaryAction     string         `json:"primary_action" gorm:"type:text;not null;default:''"`
	NextAction        string         `json:"next_action" gorm:"type:text;not null;default:''"`
	AvoidAction       string         `json:"avoid_action" gorm:"type:text;not null;default:''"`
	ContextItems      JSON           `json:"context_items" gorm:"type:jsonb;not null;default:'[]'"`
	MemorySignals     JSON           `json:"memory_signals" gorm:"type:jsonb;not null;default:'[]'"`
	SourceMemoryIDs   StringArray    `json:"source_memory_ids" gorm:"type:jsonb;not null;default:'[]'"`
	SourceMemoryCount int            `json:"source_memory_count" gorm:"not null;default:0"`
	LastMemoryAt      *time.Time     `json:"last_memory_at,omitempty"`
	Confidence        float64        `json:"confidence" gorm:"not null;default:0"`
	SalesHighlights   JSON           `json:"sales_highlights" gorm:"type:jsonb;not null;default:'[]'"`
	WriteBackStatus   string         `json:"write_back_status" gorm:"type:varchar(64);not null;default:''"`
	WriteBackDraft    string         `json:"write_back_draft" gorm:"type:text;not null;default:''"`
	ReplyDraft        string         `json:"reply_draft" gorm:"type:text;not null;default:''"`
	Metadata          JSONMap        `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceReminder) TableName() string { return "service_reminders" }

func (r *ServiceReminder) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	r.ServiceID = strings.TrimSpace(r.ServiceID)
	if r.ProfileID == "" {
		r.ProfileID = r.ServiceID
	}
	if r.Status == "" {
		r.Status = ServiceReminderStatusPending
	}
	if r.Priority == "" {
		r.Priority = ServiceReminderPriorityMedium
	}
	if r.ContextItems == nil {
		r.ContextItems = JSON([]byte(`[]`))
	}
	if r.MemorySignals == nil {
		r.MemorySignals = JSON([]byte(`[]`))
	}
	if r.SourceMemoryIDs == nil {
		r.SourceMemoryIDs = StringArray{}
	}
	if r.SalesHighlights == nil {
		r.SalesHighlights = JSON([]byte(`[]`))
	}
	if r.Metadata == nil {
		r.Metadata = JSONMap{}
	}
	return nil
}

type ServiceReminderCreateInput struct {
	SubjectID        string     `json:"subject_id,omitempty"`
	ParentReminderID string     `json:"parent_reminder_id,omitempty"`
	Title            string     `json:"title"`
	Summary          string     `json:"summary,omitempty"`
	Status           string     `json:"status,omitempty"`
	Priority         string     `json:"priority,omitempty"`
	DueAt            *time.Time `json:"due_at,omitempty"`
	DueText          string     `json:"due_text,omitempty"`
	NextAction       string     `json:"next_action,omitempty"`
	AgentDomain      string     `json:"agent_domain,omitempty"`
	AssigneeUserIDs  []string   `json:"assignee_user_ids,omitempty"`
	Metadata         JSONMap    `json:"metadata,omitempty"`
}

type ServiceReminderUpdateInput struct {
	Title            *string     `json:"title,omitempty"`
	ParentReminderID *string     `json:"parent_reminder_id,omitempty"`
	Summary          *string     `json:"summary,omitempty"`
	Status           *string     `json:"status,omitempty"`
	Priority         *string     `json:"priority,omitempty"`
	DueAt            **time.Time `json:"due_at,omitempty"`
	DueText          *string     `json:"due_text,omitempty"`
	NextAction       *string     `json:"next_action,omitempty"`
	AssigneeUserIDs  *[]string   `json:"assignee_user_ids,omitempty"`
	Metadata         *JSONMap    `json:"metadata,omitempty"`
}
