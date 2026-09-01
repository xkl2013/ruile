package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceWorkProfileStateDraft    = "draft"
	ServiceWorkProfileStateTesting  = "testing"
	ServiceWorkProfileStateEnabled  = "enabled"
	ServiceWorkProfileStateDisabled = "disabled"
	ServiceWorkProfileStateArchived = "archived"

	ServiceAgentDomainMemoryRouter    = "memory_router"
	ServiceAgentDomainLeadIntake      = "lead_intake"
	ServiceAgentDomainSalesConsulting = "sales_consulting"
	ServiceAgentDomainCustomerService = "customer_service"
	ServiceAgentDomainScheduling      = "schedule_coordination"
	ServiceAgentDomainAfterSaleRisk   = "after_sale_risk"
	ServiceAgentDomainDailyReview     = "daily_review"

	ServiceReminderStatusCandidate         = "candidate"
	ServiceReminderStatusPending           = "pending"
	ServiceReminderStatusGenerated         = "generated"
	ServiceReminderStatusConfirmed         = "confirmed"
	ServiceReminderStatusCompleted         = "completed"
	ServiceReminderStatusIgnored           = "ignored"
	ServiceReminderStatusSnoozed           = "snoozed"
	ServiceReminderStatusStale             = "stale"
	ServiceReminderStatusRecomputeRequired = "recompute_required"

	ServicePriorityHigh   = "high"
	ServicePriorityMedium = "medium"
	ServicePriorityLow    = "low"

	ServiceDailyReportRangeDay   = "day"
	ServiceDailyReportRangeWeek  = "week"
	ServiceDailyReportRangeMonth = "month"

	AgentWorkDocTypeCustomerWorkspace = "customer_workspace"
	AgentWorkDocTypeDailyReport       = "daily_report"

	AgentWorkDocStatusCurrent               = "current"
	AgentWorkDocStatusStale                 = "stale"
	AgentWorkDocStatusRecomputeRequired     = "recompute_required"
	AgentWorkDocStatusArchived              = "archived"
	AgentWorkDocStatusHiddenSourcePermitted = "hidden_due_to_source_permission"

	AgentWorkDocLinkTypeTrigger  = "trigger"
	AgentWorkDocLinkTypeEvidence = "evidence"
	AgentWorkDocLinkTypeFollowUp = "follow_up"
	AgentWorkDocLinkTypeRisk     = "risk"
	AgentWorkDocLinkTypeResolved = "resolved"

	AgentActionDraftStatusDraft     = "draft"
	AgentActionDraftStatusConfirmed = "confirmed"
	AgentActionDraftStatusExecuting = "executing"
	AgentActionDraftStatusSucceeded = "succeeded"
	AgentActionDraftStatusIgnored   = "ignored"
	AgentActionDraftStatusSnoozed   = "snoozed"
	AgentActionDraftStatusFailed    = "failed"
	AgentActionDraftStatusRetryable = "retryable"
)

// BuiltinServiceAssistantID is the internal service-module agent. It is
// resolvable by ID but intentionally not listed in the general user agent picker.
const BuiltinServiceAssistantID = "builtin-service-assistant"

func IsValidServiceWorkProfileState(state string) bool {
	switch state {
	case ServiceWorkProfileStateDraft, ServiceWorkProfileStateTesting, ServiceWorkProfileStateEnabled,
		ServiceWorkProfileStateDisabled, ServiceWorkProfileStateArchived:
		return true
	default:
		return false
	}
}

func IsValidServiceAgentDomain(domain string) bool {
	switch domain {
	case ServiceAgentDomainMemoryRouter, ServiceAgentDomainLeadIntake, ServiceAgentDomainSalesConsulting,
		ServiceAgentDomainCustomerService, ServiceAgentDomainScheduling, ServiceAgentDomainAfterSaleRisk,
		ServiceAgentDomainDailyReview:
		return true
	default:
		return false
	}
}

func IsValidServiceReminderStatus(status string) bool {
	switch status {
	case ServiceReminderStatusCandidate, ServiceReminderStatusPending, ServiceReminderStatusGenerated,
		ServiceReminderStatusConfirmed, ServiceReminderStatusCompleted, ServiceReminderStatusIgnored,
		ServiceReminderStatusSnoozed, ServiceReminderStatusStale, ServiceReminderStatusRecomputeRequired:
		return true
	default:
		return false
	}
}

func IsValidServicePriority(priority string) bool {
	switch priority {
	case ServicePriorityHigh, ServicePriorityMedium, ServicePriorityLow:
		return true
	default:
		return false
	}
}

func IsValidServiceDailyReportRange(reportRange string) bool {
	switch reportRange {
	case ServiceDailyReportRangeDay, ServiceDailyReportRangeWeek, ServiceDailyReportRangeMonth:
		return true
	default:
		return false
	}
}

func IsValidAgentWorkDocLinkType(linkType string) bool {
	switch linkType {
	case AgentWorkDocLinkTypeTrigger, AgentWorkDocLinkTypeEvidence, AgentWorkDocLinkTypeFollowUp,
		AgentWorkDocLinkTypeRisk, AgentWorkDocLinkTypeResolved:
		return true
	default:
		return false
	}
}

func IsValidAgentActionDraftStatus(status string) bool {
	switch status {
	case AgentActionDraftStatusDraft, AgentActionDraftStatusConfirmed, AgentActionDraftStatusExecuting,
		AgentActionDraftStatusSucceeded, AgentActionDraftStatusIgnored, AgentActionDraftStatusSnoozed,
		AgentActionDraftStatusFailed, AgentActionDraftStatusRetryable:
		return true
	default:
		return false
	}
}

type UserWorkProfile struct {
	ID             string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID       uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID         string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Name           string         `json:"name" gorm:"type:varchar(255);not null"`
	RoleType       string         `json:"role_type" gorm:"type:varchar(64);not null;default:''"`
	CampusScope    StringArray    `json:"campus_scope,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	CourseScope    StringArray    `json:"course_scope,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	MemoryScope    string         `json:"memory_scope" gorm:"type:text;not null;default:''"`
	TonePreference string         `json:"tone_preference" gorm:"type:varchar(255);not null;default:''"`
	DefaultProfile bool           `json:"default_profile" gorm:"not null;default:false;index"`
	Enabled        bool           `json:"enabled" gorm:"not null;default:false;index"`
	State          string         `json:"state" gorm:"type:varchar(32);not null;default:'draft';index"`
	CreatedBy      string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	UpdatedBy      string         `json:"updated_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (UserWorkProfile) TableName() string { return "user_work_profiles" }

func (p *UserWorkProfile) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.CampusScope == nil {
		p.CampusScope = StringArray{}
	}
	if p.CourseScope == nil {
		p.CourseScope = StringArray{}
	}
	if p.State == "" {
		p.State = ServiceWorkProfileStateDraft
	}
	return nil
}

type WorkProfileAgentSetting struct {
	ID               string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64         `json:"tenant_id" gorm:"not null;index"`
	ProfileID        string         `json:"profile_id" gorm:"type:varchar(36);not null;index"`
	AgentID          string         `json:"agent_id" gorm:"type:varchar(64);not null;default:''"`
	AgentDomain      string         `json:"agent_domain" gorm:"type:varchar(64);not null;index"`
	Enabled          bool           `json:"enabled" gorm:"not null;default:false;index"`
	DisplayName      string         `json:"display_name" gorm:"type:varchar(255);not null;default:''"`
	DisplayOrder     int            `json:"display_order" gorm:"not null;default:0"`
	MemoryFilter     JSONMap        `json:"memory_filter,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	KnowledgeBaseIDs StringArray    `json:"knowledge_base_ids,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	WorkDocDirectory string         `json:"work_doc_directory" gorm:"type:varchar(255);not null;default:''"`
	SelectedSkills   StringArray    `json:"selected_skills,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	OutputPolicy     JSONMap        `json:"output_policy,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy        string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	UpdatedBy        string         `json:"updated_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (WorkProfileAgentSetting) TableName() string { return "work_profile_agent_settings" }

func (s *WorkProfileAgentSetting) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.MemoryFilter == nil {
		s.MemoryFilter = JSONMap{}
	}
	if s.KnowledgeBaseIDs == nil {
		s.KnowledgeBaseIDs = StringArray{}
	}
	if s.SelectedSkills == nil {
		s.SelectedSkills = StringArray{}
	}
	if s.OutputPolicy == nil {
		s.OutputPolicy = JSONMap{}
	}
	return nil
}

type ServiceSubject struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	OwnerUserID     string         `json:"owner_user_id" gorm:"type:varchar(36);not null;index"`
	SubjectKey      string         `json:"subject_key" gorm:"type:varchar(255);not null;index"`
	DisplayName     string         `json:"display_name" gorm:"type:varchar(255);not null"`
	StudentName     string         `json:"student_name" gorm:"type:varchar(255);not null;default:''"`
	Relation        string         `json:"relation" gorm:"type:varchar(64);not null;default:''"`
	Aliases         StringArray    `json:"aliases,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	ExternalRefs    JSONMap        `json:"external_refs,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	VisibilityScope string         `json:"visibility_scope" gorm:"type:varchar(64);not null;default:'private'"`
	Confidence      float64        `json:"confidence" gorm:"not null;default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (ServiceSubject) TableName() string { return "service_subjects" }

func (s *ServiceSubject) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Aliases == nil {
		s.Aliases = StringArray{}
	}
	if s.ExternalRefs == nil {
		s.ExternalRefs = JSONMap{}
	}
	if s.VisibilityScope == "" {
		s.VisibilityScope = "private"
	}
	return nil
}

type AgentWorkDoc struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	ProfileID       string         `json:"profile_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       string         `json:"subject_id" gorm:"type:varchar(36);not null;index"`
	OwnerUserID     string         `json:"owner_user_id" gorm:"type:varchar(36);not null;index"`
	AgentDomain     string         `json:"agent_domain" gorm:"type:varchar(64);not null;index"`
	DocType         string         `json:"doc_type" gorm:"type:varchar(64);not null;default:'customer_workspace'"`
	DocPath         string         `json:"doc_path" gorm:"type:varchar(512);not null;index"`
	Title           string         `json:"title" gorm:"type:varchar(512);not null"`
	Content         string         `json:"content,omitempty" gorm:"type:text;not null;default:''"`
	Status          string         `json:"status" gorm:"type:varchar(64);not null;default:'current';index"`
	SourceMemoryIDs StringArray    `json:"source_memory_ids,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	Metadata        JSONMap        `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (AgentWorkDoc) TableName() string { return "agent_work_docs" }

func (d *AgentWorkDoc) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.DocType == "" {
		d.DocType = AgentWorkDocTypeCustomerWorkspace
	}
	if d.Status == "" {
		d.Status = AgentWorkDocStatusCurrent
	}
	if d.SourceMemoryIDs == nil {
		d.SourceMemoryIDs = StringArray{}
	}
	if d.Metadata == nil {
		d.Metadata = JSONMap{}
	}
	return nil
}

type AgentWorkDocMemoryLink struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id" gorm:"not null;index"`
	DocID           string    `json:"doc_id" gorm:"type:varchar(36);not null;index"`
	DocPath         string    `json:"doc_path" gorm:"type:varchar(512);not null;index"`
	MemoryID        string    `json:"memory_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       string    `json:"subject_id" gorm:"type:varchar(36);not null;index"`
	AgentDomain     string    `json:"agent_domain" gorm:"type:varchar(64);not null;index"`
	LinkType        string    `json:"link_type" gorm:"type:varchar(32);not null;default:'evidence';index"`
	Confidence      float64   `json:"confidence" gorm:"not null;default:0"`
	EvidenceExcerpt string    `json:"evidence_excerpt,omitempty" gorm:"type:text;not null;default:''"`
	CreatedAt       time.Time `json:"created_at"`
}

func (AgentWorkDocMemoryLink) TableName() string { return "agent_work_doc_memory_links" }

func (l *AgentWorkDocMemoryLink) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	if l.LinkType == "" {
		l.LinkType = AgentWorkDocLinkTypeEvidence
	}
	return nil
}

type ServiceMemoryEvidence struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	SourceLabel     string `json:"sourceLabel"`
	OccurredAtLabel string `json:"occurredAtLabel"`
}

type ServiceReminder struct {
	ID                string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID          uint64                  `json:"tenant_id" gorm:"not null;index"`
	UserID            string                  `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ProfileID         string                  `json:"profile_id" gorm:"type:varchar(36);not null;index"`
	SubjectID         string                  `json:"subject_id" gorm:"type:varchar(36);not null;index"`
	AgentDomain       string                  `json:"agent_domain" gorm:"type:varchar(64);not null;index"`
	Title             string                  `json:"title" gorm:"type:varchar(512);not null"`
	Summary           string                  `json:"summary" gorm:"type:text;not null;default:''"`
	Status            string                  `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	Priority          string                  `json:"priority" gorm:"type:varchar(16);not null;default:'low';index"`
	DueAt             *time.Time              `json:"due_at,omitempty" gorm:"index"`
	DueText           string                  `json:"due_text" gorm:"type:varchar(64);not null;default:''"`
	Stage             string                  `json:"stage" gorm:"type:varchar(128);not null;default:''"`
	Channel           string                  `json:"channel" gorm:"type:varchar(128);not null;default:''"`
	DecisionRole      string                  `json:"decision_role" gorm:"type:varchar(128);not null;default:''"`
	RiskLabel         string                  `json:"risk_label" gorm:"type:varchar(128);not null;default:''"`
	AssistReason      string                  `json:"assist_reason" gorm:"type:text;not null;default:''"`
	PrimaryAction     string                  `json:"primary_action" gorm:"type:text;not null;default:''"`
	NextAction        string                  `json:"next_action" gorm:"type:text;not null;default:''"`
	AvoidAction       string                  `json:"avoid_action" gorm:"type:text;not null;default:''"`
	ContextItems      StringArray             `json:"context_items,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	MemorySignals     StringArray             `json:"memory_signals,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	SourceMemoryIDs   StringArray             `json:"source_memory_ids,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	SourceMemoryCount int                     `json:"source_memory_count" gorm:"not null;default:0"`
	LastMemoryAt      *time.Time              `json:"last_memory_at,omitempty" gorm:"index"`
	Confidence        float64                 `json:"confidence" gorm:"not null;default:0"`
	SalesHighlights   StringArray             `json:"sales_highlights,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	WriteBackStatus   string                  `json:"write_back_status" gorm:"type:varchar(64);not null;default:''"`
	WriteBackDraft    string                  `json:"write_back_draft" gorm:"type:text;not null;default:''"`
	ReplyDraft        string                  `json:"reply_draft" gorm:"type:text;not null;default:''"`
	Metadata          JSONMap                 `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryEvidence    []ServiceMemoryEvidence `json:"memory_evidence,omitempty" gorm:"-"`
	WorkDocs          []*AgentWorkDoc         `json:"work_docs,omitempty" gorm:"-"`
	ActionDrafts      []*AgentActionDraft     `json:"action_drafts,omitempty" gorm:"-"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	DeletedAt         gorm.DeletedAt          `json:"deleted_at" gorm:"index"`
}

func (ServiceReminder) TableName() string { return "service_reminders" }

func (r *ServiceReminder) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Status == "" {
		r.Status = ServiceReminderStatusPending
	}
	if r.Priority == "" {
		r.Priority = ServicePriorityLow
	}
	if r.ContextItems == nil {
		r.ContextItems = StringArray{}
	}
	if r.MemorySignals == nil {
		r.MemorySignals = StringArray{}
	}
	if r.SourceMemoryIDs == nil {
		r.SourceMemoryIDs = StringArray{}
	}
	if r.SalesHighlights == nil {
		r.SalesHighlights = StringArray{}
	}
	if r.Metadata == nil {
		r.Metadata = JSONMap{}
	}
	return nil
}

type AgentActionDraft struct {
	ID               string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID           string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ReminderID       string         `json:"reminder_id" gorm:"type:varchar(36);not null;index"`
	AgentID          string         `json:"agent_id" gorm:"type:varchar(64);not null;default:''"`
	AgentDomain      string         `json:"agent_domain" gorm:"type:varchar(64);not null;index"`
	ActionType       string         `json:"action_type" gorm:"type:varchar(64);not null;default:'follow_up'"`
	Status           string         `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	Title            string         `json:"title" gorm:"type:varchar(512);not null"`
	Summary          string         `json:"summary" gorm:"type:text;not null;default:''"`
	Payload          JSONMap        `json:"payload,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	SourceMemoryIDs  StringArray    `json:"source_memory_ids,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	ExternalSystem   string         `json:"external_system,omitempty" gorm:"type:varchar(128);not null;default:''"`
	ExternalObjectID string         `json:"external_object_id,omitempty" gorm:"type:varchar(255);not null;default:''"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (AgentActionDraft) TableName() string { return "agent_action_drafts" }

func (d *AgentActionDraft) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.Status == "" {
		d.Status = AgentActionDraftStatusDraft
	}
	if d.ActionType == "" {
		d.ActionType = "follow_up"
	}
	if d.Payload == nil {
		d.Payload = JSONMap{}
	}
	if d.SourceMemoryIDs == nil {
		d.SourceMemoryIDs = StringArray{}
	}
	return nil
}

type AgentActionLog struct {
	ID            string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64    `json:"tenant_id" gorm:"not null;index"`
	ActionDraftID string    `json:"action_draft_id" gorm:"type:varchar(36);not null;index"`
	Status        string    `json:"status" gorm:"type:varchar(32);not null;default:''"`
	Message       string    `json:"message" gorm:"type:text;not null;default:''"`
	Payload       JSONMap   `json:"payload,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AgentActionLog) TableName() string { return "agent_action_logs" }

func (l *AgentActionLog) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	if l.Payload == nil {
		l.Payload = JSONMap{}
	}
	return nil
}

type ServiceListQuery struct {
	TenantID    uint64
	UserID      string
	ProfileID   string
	SubjectID   string
	Keyword     string
	Status      string
	AgentDomain string
	Page        int
	PageSize    int
}

type ServiceDailyReportListQuery struct {
	TenantID  uint64
	UserID    string
	ProfileID string
	Range     string
	Keyword   string
	Page      int
	PageSize  int
}

type ServiceCustomerSpaceListQuery struct {
	TenantID  uint64
	UserID    string
	ProfileID string
	Keyword   string
	Page      int
	PageSize  int
}

type ServiceWorkProfileInput struct {
	UserID         string      `json:"user_id,omitempty"`
	Name           string      `json:"name"`
	RoleType       string      `json:"role_type,omitempty"`
	CampusScope    StringArray `json:"campus_scope,omitempty"`
	CourseScope    StringArray `json:"course_scope,omitempty"`
	MemoryScope    string      `json:"memory_scope,omitempty"`
	TonePreference string      `json:"tone_preference,omitempty"`
	DefaultProfile bool        `json:"default_profile,omitempty"`
	Enabled        bool        `json:"enabled,omitempty"`
	State          string      `json:"state,omitempty"`
}

type WorkProfileAgentSettingInput struct {
	ID               string      `json:"id,omitempty"`
	AgentID          string      `json:"agent_id,omitempty"`
	AgentDomain      string      `json:"agent_domain"`
	Enabled          bool        `json:"enabled"`
	DisplayName      string      `json:"display_name,omitempty"`
	DisplayOrder     int         `json:"display_order,omitempty"`
	MemoryFilter     JSONMap     `json:"memory_filter,omitempty"`
	KnowledgeBaseIDs StringArray `json:"knowledge_base_ids,omitempty"`
	WorkDocDirectory string      `json:"work_doc_directory,omitempty"`
	SelectedSkills   StringArray `json:"selected_skills,omitempty"`
	OutputPolicy     JSONMap     `json:"output_policy,omitempty"`
}

type WorkProfileAgentSettingsInput struct {
	Settings []WorkProfileAgentSettingInput `json:"settings"`
}

type ServiceReminderStatusInput struct {
	Status string `json:"status"`
}

type AgentActionDraftInput struct {
	AgentID          string      `json:"agent_id,omitempty"`
	AgentDomain      string      `json:"agent_domain,omitempty"`
	ActionType       string      `json:"action_type,omitempty"`
	Title            string      `json:"title,omitempty"`
	Summary          string      `json:"summary,omitempty"`
	Payload          JSONMap     `json:"payload,omitempty"`
	SourceMemoryIDs  StringArray `json:"source_memory_ids,omitempty"`
	ExternalSystem   string      `json:"external_system,omitempty"`
	ExternalObjectID string      `json:"external_object_id,omitempty"`
}

type AgentActionDraftStatusInput struct {
	Status string `json:"status"`
}

type ServiceDailyReportInput struct {
	Range    string `json:"range,omitempty"`
	Date     string `json:"date,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type ServiceDailyReport struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Summary         string      `json:"summary,omitempty"`
	Content         string      `json:"content"`
	Range           string      `json:"range"`
	Stage           string      `json:"stage"`
	StageKey        string      `json:"stage_key"`
	Updated         string      `json:"updated"`
	ActionCount     int         `json:"action_count"`
	CustomerCount   int         `json:"customer_count"`
	Chips           StringArray `json:"chips,omitempty"`
	SourceMemoryIDs StringArray `json:"source_memory_ids,omitempty"`
	Metadata        JSONMap     `json:"metadata,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type ServiceCustomerSpace struct {
	ID                string      `json:"id"`
	TenantID          uint64      `json:"tenant_id"`
	OwnerUserID       string      `json:"owner_user_id"`
	ProfileID         string      `json:"profile_id,omitempty"`
	SubjectKey        string      `json:"subject_key"`
	DisplayName       string      `json:"display_name"`
	Name              string      `json:"name"`
	StudentName       string      `json:"student_name,omitempty"`
	Relation          string      `json:"relation,omitempty"`
	Description       string      `json:"description,omitempty"`
	Summary           string      `json:"summary,omitempty"`
	Status            string      `json:"status"`
	Priority          string      `json:"priority,omitempty"`
	Stage             string      `json:"stage,omitempty"`
	RiskLabel         string      `json:"risk_label,omitempty"`
	LatestAction      string      `json:"latest_action,omitempty"`
	VisibilityScope   string      `json:"visibility_scope"`
	Confidence        float64     `json:"confidence"`
	WorkDocCount      int         `json:"work_doc_count"`
	ReminderCount     int         `json:"reminder_count"`
	OpenReminderCount int         `json:"open_reminder_count"`
	SourceMemoryCount int         `json:"source_memory_count"`
	Directories       StringArray `json:"directories,omitempty"`
	Chips             StringArray `json:"chips,omitempty"`
	LatestMemoryAt    *time.Time  `json:"latest_memory_at,omitempty"`
	LatestReminderAt  *time.Time  `json:"latest_reminder_at,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type ServiceCustomerSpaceDetail struct {
	Summary        *ServiceCustomerSpace   `json:"summary"`
	Subject        *ServiceSubject         `json:"subject"`
	WorkDocs       []*AgentWorkDoc         `json:"work_docs"`
	Reminders      []*ServiceReminder      `json:"reminders"`
	MemoryEvidence []ServiceMemoryEvidence `json:"memory_evidence"`
	Directories    StringArray             `json:"directories,omitempty"`
	Stats          map[string]int64        `json:"stats"`
}

type ServiceAgentTemplate struct {
	AgentDomain      string      `json:"agent_domain"`
	DisplayName      string      `json:"display_name"`
	Description      string      `json:"description"`
	DefaultEnabled   bool        `json:"default_enabled"`
	UserVisible      bool        `json:"user_visible"`
	WorkDocDirectory string      `json:"work_doc_directory"`
	MemoryFilter     JSONMap     `json:"memory_filter,omitempty"`
	OutputPolicy     JSONMap     `json:"output_policy,omitempty"`
	SelectedSkills   StringArray `json:"selected_skills,omitempty"`
}

type ServiceBootstrap struct {
	Profile       *UserWorkProfile           `json:"profile,omitempty"`
	AgentSettings []*WorkProfileAgentSetting `json:"agent_settings,omitempty"`
	Reminders     []*ServiceReminder         `json:"reminders"`
	Total         int64                      `json:"total"`
	Stats         map[string]int64           `json:"stats"`
	Templates     []ServiceAgentTemplate     `json:"templates,omitempty"`
}

type ServiceMemoryExtraction struct {
	MemoryID  string           `json:"memory_id"`
	Generated bool             `json:"generated"`
	Reason    string           `json:"reason,omitempty"`
	Reminder  *ServiceReminder `json:"reminder,omitempty"`
}
