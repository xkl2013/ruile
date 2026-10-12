package types

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TenantSandboxModeDocker   = "docker"
	TenantSandboxModeLocal    = "local"
	TenantSandboxModeDisabled = "disabled"

	DefaultTenantSandboxTimeoutSeconds = 60
)

// TenantSandboxConfig stores the tenant-scoped sandbox policy used by Agent
// skill execution. It intentionally models only the backends supported by the
// current runtime; remote provider credentials and snapshots are separate work.
type TenantSandboxConfig struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	Name            string         `json:"name" gorm:"type:varchar(128);not null"`
	Mode            string         `json:"mode" gorm:"type:varchar(16);not null"`
	Enabled         bool           `json:"enabled" gorm:"not null;index"`
	Active          bool           `json:"active" gorm:"not null;index"`
	FallbackEnabled bool           `json:"fallback_enabled" gorm:"not null"`
	TimeoutSeconds  int            `json:"timeout_seconds" gorm:"not null;default:60"`
	DockerImage     string         `json:"docker_image" gorm:"type:varchar(255);not null;default:''"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

type TenantSandboxCheck struct {
	ConfigID      string `json:"config_id"`
	Mode          string `json:"mode"`
	Available     bool   `json:"available"`
	EffectiveMode string `json:"effective_mode"`
	Message       string `json:"message,omitempty"`
}

func (TenantSandboxConfig) TableName() string { return "tenant_sandbox_configs" }

func (c *TenantSandboxConfig) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.Name = strings.TrimSpace(c.Name)
	c.Mode = strings.ToLower(strings.TrimSpace(c.Mode))
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = DefaultTenantSandboxTimeoutSeconds
	}
	return nil
}

func (c *TenantSandboxConfig) Normalize() {
	if c == nil {
		return
	}
	c.Name = strings.TrimSpace(c.Name)
	c.Mode = strings.ToLower(strings.TrimSpace(c.Mode))
	c.DockerImage = strings.TrimSpace(c.DockerImage)
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = DefaultTenantSandboxTimeoutSeconds
	}
}

func (c *TenantSandboxConfig) Validate() error {
	if c == nil {
		return gorm.ErrInvalidData
	}
	c.Normalize()
	if c.Name == "" {
		return ErrTenantSandboxConfigNameRequired
	}
	switch c.Mode {
	case TenantSandboxModeDocker, TenantSandboxModeLocal, TenantSandboxModeDisabled:
	default:
		return ErrTenantSandboxConfigModeInvalid
	}
	if c.TimeoutSeconds < 1 || c.TimeoutSeconds > 3600 {
		return ErrTenantSandboxConfigTimeoutInvalid
	}
	return nil
}

var (
	ErrTenantSandboxConfigNameRequired   = errors.New("sandbox config name is required")
	ErrTenantSandboxConfigModeInvalid    = errors.New("sandbox config mode is invalid")
	ErrTenantSandboxConfigTimeoutInvalid = errors.New("sandbox config timeout is invalid")
	ErrTenantSandboxConfigActive         = errors.New("active sandbox config cannot be deleted")
)
