package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceSpaceTemplateRecord is the persisted form of a published service
// template. JSON fields keep the template contract extensible without
// hard-coding domain-specific subject fields into the platform schema.
type ServiceSpaceTemplateRecord struct {
	ID           string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64         `json:"tenant_id" gorm:"not null;index"`
	Key          string         `json:"key" gorm:"type:varchar(128);not null;index"`
	Name         string         `json:"name" gorm:"type:varchar(255);not null"`
	Version      int            `json:"version" gorm:"not null;default:1"`
	Status       string         `json:"status" gorm:"type:varchar(32);not null;default:'published';index"`
	MatchRules   JSON           `json:"match_rules" gorm:"type:jsonb;not null;default:'{}'"`
	Blueprint    JSON           `json:"blueprint" gorm:"type:jsonb;not null;default:'{}'"`
	AutoApply    bool           `json:"auto_apply" gorm:"not null;default:false"`
	AutoActivate bool           `json:"auto_activate" gorm:"not null;default:false"`
	RiskLevel    string         `json:"risk_level" gorm:"type:varchar(16);not null;default:'low'"`
	PublishedBy  string         `json:"published_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	PublishedAt  *time.Time     `json:"published_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceSpaceTemplateRecord) TableName() string { return "service_space_templates" }

func (r *ServiceSpaceTemplateRecord) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Version < 1 {
		r.Version = 1
	}
	if len(r.MatchRules) == 0 {
		r.MatchRules = JSON([]byte(`{}`))
	}
	if len(r.Blueprint) == 0 {
		r.Blueprint = JSON([]byte(`{}`))
	}
	return nil
}

func (r ServiceSpaceTemplateRecord) ToTemplate() (ServiceSpaceTemplate, error) {
	var rules ServiceSpaceTemplateMatchRules
	var blueprint ServiceSpaceBlueprint
	if err := json.Unmarshal(r.MatchRules, &rules); err != nil {
		return ServiceSpaceTemplate{}, err
	}
	if err := json.Unmarshal(r.Blueprint, &blueprint); err != nil {
		return ServiceSpaceTemplate{}, err
	}
	publishedAt := r.PublishedAt
	return ServiceSpaceTemplate{
		ID: r.ID, TenantID: r.TenantID, Key: r.Key, Name: r.Name,
		Version: r.Version, Status: ServiceSpaceTemplateStatus(r.Status),
		MatchRules: rules, Blueprint: blueprint, AutoApply: r.AutoApply,
		AutoActivate: r.AutoActivate, RiskLevel: ServiceSpaceTemplateRiskLevel(r.RiskLevel),
		PublishedBy: r.PublishedBy, PublishedAt: publishedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

type ServiceSpaceBlueprintRecord struct {
	ID                string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID          uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID         string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	Version           int            `json:"version" gorm:"not null;default:1"`
	SourceType        string         `json:"source_type" gorm:"type:varchar(32);not null"`
	SourceInstruction string         `json:"source_instruction" gorm:"type:text;not null"`
	Blueprint         JSON           `json:"blueprint" gorm:"type:jsonb;not null;default:'{}'"`
	Status            string         `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	ConfirmationMode  string         `json:"confirmation_mode" gorm:"type:varchar(32);not null;default:'pending'"`
	ProfileVersion    int            `json:"profile_version" gorm:"not null;default:1"`
	ProfileHash       string         `json:"profile_hash" gorm:"type:varchar(64);not null;default:''"`
	ConfirmedBy       string         `json:"confirmed_by" gorm:"type:varchar(36);not null;default:''"`
	ConfirmedAt       *time.Time     `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceSpaceBlueprintRecord) TableName() string { return "service_space_blueprints" }

func (r *ServiceSpaceBlueprintRecord) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Version < 1 {
		r.Version = 1
	}
	if r.ProfileVersion < 1 {
		r.ProfileVersion = 1
	}
	if len(r.Blueprint) == 0 {
		r.Blueprint = JSON([]byte(`{}`))
	}
	return nil
}

func (r ServiceSpaceBlueprintRecord) ToBlueprint() (ServiceSpaceBlueprint, error) {
	var blueprint ServiceSpaceBlueprint
	if err := json.Unmarshal(r.Blueprint, &blueprint); err != nil {
		return ServiceSpaceBlueprint{}, err
	}
	blueprint.ID = r.ID
	blueprint.TenantID = r.TenantID
	blueprint.ServiceID = r.ServiceID
	blueprint.SourceType = ServiceSpaceBlueprintSourceType(r.SourceType)
	blueprint.SourceInstruction = r.SourceInstruction
	blueprint.Status = ServiceSpaceBlueprintStatus(r.Status)
	blueprint.ConfirmationMode = ServiceSpaceBlueprintConfirmationMode(r.ConfirmationMode)
	blueprint.ProfileVersion = r.ProfileVersion
	blueprint.ProfileHash = r.ProfileHash
	blueprint.Version = r.Version
	blueprint.ConfirmedBy = r.ConfirmedBy
	blueprint.ConfirmedAt = r.ConfirmedAt
	blueprint.CreatedAt = r.CreatedAt
	blueprint.UpdatedAt = r.UpdatedAt
	return blueprint, nil
}

type ServiceSpaceTemplateApplicationRecord struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id" gorm:"not null;index"`
	ServiceID       string    `json:"service_id" gorm:"type:varchar(36);not null;index"`
	TemplateKey     string    `json:"template_key" gorm:"type:varchar(128);not null"`
	TemplateVersion int       `json:"template_version" gorm:"not null"`
	ProfileVersion  int       `json:"profile_version" gorm:"not null;default:1"`
	ProfileHash     string    `json:"profile_hash" gorm:"type:varchar(64);not null;default:''"`
	MatchReason     JSONMap   `json:"match_reason" gorm:"type:jsonb;not null;default:'{}'"`
	ApplyMode       string    `json:"apply_mode" gorm:"type:varchar(32);not null"`
	IdempotencyKey  string    `json:"idempotency_key" gorm:"type:varchar(128);not null"`
	Result          string    `json:"result" gorm:"type:varchar(32);not null"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ServiceSpaceTemplateApplicationRecord) TableName() string {
	return "service_space_template_applications"
}

func (r *ServiceSpaceTemplateApplicationRecord) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.MatchReason == nil {
		r.MatchReason = JSONMap{}
	}
	return nil
}

func (r ServiceSpaceTemplateApplicationRecord) ToApplication() ServiceSpaceTemplateApplication {
	return ServiceSpaceTemplateApplication{
		ID: r.ID, TenantID: r.TenantID, ServiceID: r.ServiceID,
		TemplateKey: r.TemplateKey, TemplateVersion: r.TemplateVersion,
		ProfileVersion: r.ProfileVersion, ProfileHash: r.ProfileHash,
		MatchReason:    r.MatchReason,
		ApplyMode:      ServiceSpaceBlueprintConfirmationMode(r.ApplyMode),
		IdempotencyKey: r.IdempotencyKey,
		Result:         ServiceSpaceTemplateApplicationResult(r.Result),
		CreatedAt:      r.CreatedAt,
	}
}

type ServiceSpaceProfile struct {
	ID               string                     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64                     `json:"tenant_id" gorm:"not null;index"`
	ServiceID        string                     `json:"service_id" gorm:"type:varchar(36);not null;index"`
	BlueprintVersion int                        `json:"blueprint_version" gorm:"not null"`
	Version          int                        `json:"version" gorm:"not null;default:1"`
	Schema           []ServiceSpaceProfileField `json:"schema" gorm:"-"`
	SchemaJSON       JSON                       `json:"-" gorm:"column:schema_json;type:jsonb;not null;default:'[]'"`
	Values           JSONMap                    `json:"values" gorm:"column:profile_values;type:jsonb;not null;default:'{}'"`
	SourceWatermark  string                     `json:"source_watermark" gorm:"type:varchar(128);not null;default:''"`
	Frozen           bool                       `json:"frozen" gorm:"not null;default:false"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

func (ServiceSpaceProfile) TableName() string { return "service_space_profiles" }

func (p *ServiceSpaceProfile) BeforeSave(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Version < 1 {
		p.Version = 1
	}
	if p.Values == nil {
		p.Values = JSONMap{}
	}
	if len(p.SchemaJSON) == 0 {
		data, err := json.Marshal(p.Schema)
		if err != nil {
			return err
		}
		p.SchemaJSON = JSON(data)
	}
	return nil
}

func (p *ServiceSpaceProfile) AfterFind(_ *gorm.DB) error {
	if len(p.SchemaJSON) == 0 {
		p.Schema = []ServiceSpaceProfileField{}
		return nil
	}
	return json.Unmarshal(p.SchemaJSON, &p.Schema)
}

// ServiceSubjectProfile is the materialized profile for one business subject
// inside a service space. Keeping SubjectID in the storage key prevents facts
// from multiple customers being mixed into one service-level profile.
type ServiceSubjectProfile struct {
	ID               string                     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64                     `json:"tenant_id" gorm:"not null;index"`
	ServiceID        string                     `json:"service_id" gorm:"type:varchar(36);not null;index"`
	SubjectID        string                     `json:"subject_id" gorm:"type:varchar(36);not null;index"`
	BlueprintVersion int                        `json:"blueprint_version" gorm:"not null"`
	Version          int                        `json:"version" gorm:"not null;default:1"`
	Schema           []ServiceSpaceProfileField `json:"schema" gorm:"-"`
	SchemaJSON       JSON                       `json:"-" gorm:"column:schema_json;type:jsonb;not null;default:'[]'"`
	Values           JSONMap                    `json:"values" gorm:"column:profile_values;type:jsonb;not null;default:'{}'"`
	SourceWatermark  string                     `json:"source_watermark" gorm:"type:varchar(128);not null;default:''"`
	Frozen           bool                       `json:"frozen" gorm:"not null;default:false"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

func (ServiceSubjectProfile) TableName() string { return "service_subject_profiles" }

func (p *ServiceSubjectProfile) BeforeSave(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Version < 1 {
		p.Version = 1
	}
	if p.Values == nil {
		p.Values = JSONMap{}
	}
	if len(p.SchemaJSON) == 0 {
		data, err := json.Marshal(p.Schema)
		if err != nil {
			return err
		}
		p.SchemaJSON = JSON(data)
	}
	return nil
}

func (p *ServiceSubjectProfile) AfterFind(_ *gorm.DB) error {
	if len(p.SchemaJSON) == 0 {
		p.Schema = []ServiceSpaceProfileField{}
		return nil
	}
	return json.Unmarshal(p.SchemaJSON, &p.Schema)
}

type ServiceSpaceSummary struct {
	ID               string                       `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64                       `json:"tenant_id" gorm:"not null;index"`
	ServiceID        string                       `json:"service_id" gorm:"type:varchar(36);not null;index"`
	BlueprintVersion int                          `json:"blueprint_version" gorm:"not null"`
	Version          int                          `json:"version" gorm:"not null;default:1"`
	Schema           []ServiceSpaceSummarySection `json:"schema" gorm:"-"`
	SchemaJSON       JSON                         `json:"-" gorm:"column:schema_json;type:jsonb;not null;default:'[]'"`
	Sections         JSONMap                      `json:"sections" gorm:"type:jsonb;not null;default:'{}'"`
	SourceWatermark  string                       `json:"source_watermark" gorm:"type:varchar(128);not null;default:''"`
	Frozen           bool                         `json:"frozen" gorm:"not null;default:false"`
	RefreshStatus    string                       `json:"refresh_status" gorm:"type:varchar(32);not null;default:'ready'"`
	ErrorMessage     string                       `json:"error_message" gorm:"type:text;not null;default:''"`
	GeneratedAt      time.Time                    `json:"generated_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

func (ServiceSpaceSummary) TableName() string { return "service_space_summaries" }

func (s *ServiceSpaceSummary) BeforeSave(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Version < 1 {
		s.Version = 1
	}
	if s.Sections == nil {
		s.Sections = JSONMap{}
	}
	if len(s.SchemaJSON) == 0 {
		data, err := json.Marshal(s.Schema)
		if err != nil {
			return err
		}
		s.SchemaJSON = JSON(data)
	}
	if s.GeneratedAt.IsZero() {
		s.GeneratedAt = time.Now().UTC()
	}
	return nil
}

func (s *ServiceSpaceSummary) AfterFind(_ *gorm.DB) error {
	if len(s.SchemaJSON) == 0 {
		s.Schema = []ServiceSpaceSummarySection{}
		return nil
	}
	return json.Unmarshal(s.SchemaJSON, &s.Schema)
}

type ServiceRuntimeContext struct {
	ServiceID        string                  `json:"service_id"`
	SpaceType        ServiceSpaceType        `json:"space_type"`
	Instruction      string                  `json:"instruction"`
	KnowledgeBaseIDs []string                `json:"knowledge_base_ids"`
	TemplateKey      string                  `json:"template_key,omitempty"`
	TemplateVersion  int                     `json:"template_version,omitempty"`
	BlueprintVersion int                     `json:"blueprint_version,omitempty"`
	Experts          []*ServiceExpertBinding `json:"experts,omitempty"`
	Profile          *ServiceSpaceProfile    `json:"profile,omitempty"`
	Summary          *ServiceSpaceSummary    `json:"summary,omitempty"`
	Artifacts        []*ServiceArtifact      `json:"artifacts,omitempty"`
	ContextSources   []*ServiceContextSource `json:"context_sources,omitempty"`
	Facts            []*ServiceFact          `json:"facts,omitempty"`
	ContextHash      string                  `json:"context_hash"`
}

type ServiceContextSourceImportInput struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

type ServiceSpaceProfileUpdateInput struct {
	Values JSONMap                     `json:"values"`
	Schema *[]ServiceSpaceProfileField `json:"schema,omitempty"`
}

type ServiceSpaceArtifactLifecycleInput struct {
	Lifecycle      string `json:"lifecycle"`
	IdempotencyKey string `json:"idempotency_key"`
}
