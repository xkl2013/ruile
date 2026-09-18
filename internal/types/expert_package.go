package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ExpertPackageSourceRuileNative = "ruile-native"
	ExpertPackageSourceWorkBuddy   = "workbuddy"
	ExpertPackageSourceCustomYAML  = "custom-yaml"

	ExpertPackageVersionTesting    = "testing"
	ExpertPackageVersionPublished  = "published"
	ExpertPackageVersionDeprecated = "deprecated"
	ExpertPackageVersionArchived   = "archived"

	AgentPublicationChannelStable = "stable"
	AgentPublicationChannelCanary = "canary"
)

type ExpertPackage struct {
	ID           string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64                  `json:"tenant_id" gorm:"not null;index"`
	PackageKey   string                  `json:"package_key" gorm:"type:varchar(128);not null"`
	DisplayName  string                  `json:"display_name" gorm:"type:varchar(255);not null"`
	Description  string                  `json:"description" gorm:"type:text;not null;default:''"`
	SourceFormat string                  `json:"source_format" gorm:"type:varchar(64);not null"`
	SourceURI    string                  `json:"source_uri" gorm:"type:text;not null;default:''"`
	License      string                  `json:"license" gorm:"type:varchar(255);not null;default:''"`
	CreatedBy    string                  `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	DeletedAt    gorm.DeletedAt          `json:"-" gorm:"index"`
	Versions     []*ExpertPackageVersion `json:"versions,omitempty" gorm:"-"`
}

func (ExpertPackage) TableName() string { return "expert_packages" }

func (p *ExpertPackage) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

type ExpertPackageVersion struct {
	ID          string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	PackageID   string                    `json:"package_id" gorm:"type:varchar(36);not null;index"`
	Version     string                    `json:"version" gorm:"type:varchar(64);not null"`
	State       string                    `json:"state" gorm:"type:varchar(32);not null;default:'testing';index"`
	Manifest    JSONMap                   `json:"manifest" gorm:"type:jsonb;not null;default:'{}'"`
	PackageHash string                    `json:"package_hash" gorm:"type:varchar(64);not null"`
	Diagnostics JSONMap                   `json:"diagnostics" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy   string                    `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	PublishedBy string                    `json:"published_by" gorm:"type:varchar(36);not null;default:''"`
	PublishedAt *time.Time                `json:"published_at,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	DeletedAt   gorm.DeletedAt            `json:"-" gorm:"index"`
	Definitions []*AgentDefinitionVersion `json:"definitions,omitempty" gorm:"-"`
}

func (ExpertPackageVersion) TableName() string { return "expert_package_versions" }

func (v *ExpertPackageVersion) BeforeCreate(_ *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.State == "" {
		v.State = ExpertPackageVersionTesting
	}
	if v.Manifest == nil {
		v.Manifest = JSONMap{}
	}
	if v.Diagnostics == nil {
		v.Diagnostics = JSONMap{}
	}
	return nil
}

type AgentDefinitionVersion struct {
	ID               string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64         `json:"tenant_id" gorm:"not null;index"`
	PackageID        string         `json:"package_id" gorm:"type:varchar(36);not null;index"`
	PackageVersionID string         `json:"package_version_id" gorm:"type:varchar(36);not null;index"`
	AgentID          string         `json:"agent_id" gorm:"type:varchar(128);not null"`
	Version          string         `json:"version" gorm:"type:varchar(64);not null"`
	DisplayName      string         `json:"display_name" gorm:"type:varchar(255);not null"`
	Description      string         `json:"description" gorm:"type:text;not null;default:''"`
	Domain           string         `json:"domain" gorm:"type:varchar(64);not null;default:''"`
	SystemPrompt     string         `json:"system_prompt" gorm:"type:text;not null"`
	CompiledConfig   JSONMap        `json:"compiled_config" gorm:"type:jsonb;not null;default:'{}'"`
	Skills           StringArray    `json:"skills" gorm:"type:jsonb;not null;default:'[]'"`
	Capabilities     JSONMap        `json:"capabilities" gorm:"type:jsonb;not null;default:'{}'"`
	OutputContract   string         `json:"output_contract" gorm:"type:varchar(64);not null;default:'agent_result_v1'"`
	DefinitionHash   string         `json:"definition_hash" gorm:"type:varchar(64);not null"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentDefinitionVersion) TableName() string { return "agent_definition_versions" }

func (v *AgentDefinitionVersion) BeforeCreate(_ *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.CompiledConfig == nil {
		v.CompiledConfig = JSONMap{}
	}
	if v.Skills == nil {
		v.Skills = StringArray{}
	}
	if v.Capabilities == nil {
		v.Capabilities = JSONMap{}
	}
	if v.OutputContract == "" {
		v.OutputContract = AgentResultSchemaV1
	}
	return nil
}

// PublishedExpert is the user-facing projection of a published package
// definition. It omits prompts and compiled runtime configuration.
type PublishedExpert struct {
	PackageID          string      `json:"package_id"`
	PackageVersionID   string      `json:"package_version_id"`
	PackageKey         string      `json:"package_key"`
	PackageDisplayName string      `json:"package_display_name"`
	PackageDescription string      `json:"package_description"`
	PackageVersion     string      `json:"package_version"`
	DefinitionID       string      `json:"definition_id"`
	AgentID            string      `json:"agent_id"`
	Version            string      `json:"version"`
	DisplayName        string      `json:"display_name"`
	Description        string      `json:"description"`
	Domain             string      `json:"domain"`
	OutputContract     string      `json:"output_contract"`
	Skills             StringArray `json:"skills"`
	Capabilities       JSONMap     `json:"capabilities"`
}

type AgentBinding struct {
	ID                       string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID                 uint64         `json:"tenant_id" gorm:"not null;index"`
	ProfileID                string         `json:"profile_id" gorm:"type:varchar(36);not null;default:'';index"`
	AgentDefinitionVersionID string         `json:"agent_definition_version_id" gorm:"type:varchar(36);not null;index"`
	AgentDomain              string         `json:"agent_domain" gorm:"type:varchar(64);not null;default:'';index"`
	Enabled                  bool           `json:"enabled" gorm:"not null;default:true"`
	CreatedBy                string         `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentBinding) TableName() string { return "agent_bindings" }

func (b *AgentBinding) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

func IsValidExpertPackageSource(source string) bool {
	switch source {
	case ExpertPackageSourceRuileNative, ExpertPackageSourceWorkBuddy, ExpertPackageSourceCustomYAML:
		return true
	default:
		return false
	}
}

type ExpertPackageFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ExpertPackageImportInput struct {
	SourceFormat string                   `json:"source_format"`
	PackageKey   string                   `json:"package_key"`
	Version      string                   `json:"version"`
	DisplayName  string                   `json:"display_name"`
	Description  string                   `json:"description,omitempty"`
	SourceURI    string                   `json:"source_uri,omitempty"`
	License      string                   `json:"license,omitempty"`
	Files        []ExpertPackageFileInput `json:"files"`
}

type AgentBindingInput struct {
	ProfileID                string `json:"profile_id"`
	AgentDefinitionVersionID string `json:"agent_definition_version_id"`
	AgentDomain              string `json:"agent_domain,omitempty"`
	Enabled                  bool   `json:"enabled"`
}
