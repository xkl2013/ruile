package types

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceReminderStatusCategoryOpen       = "open"
	ServiceReminderStatusCategoryInProgress = "in_progress"
	ServiceReminderStatusCategoryDone       = "done"
	ServiceReminderStatusCategoryDismissed  = "dismissed"
	ServiceReminderStatusKeyMaxLen          = 64
	ServiceReminderStatusLabelMaxLen        = 128
	ServiceReminderStatusColorMaxLen        = 32
	ServiceReminderStatusDescriptionMaxLen  = 512
)

// ServiceReminderStatus is the configurable lifecycle state of a service
// reminder. The category is fixed for reporting, while status_key and label
// are owned by the service.
type ServiceReminderStatus struct {
	ID           string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID    string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	StatusKey    string         `json:"status_key" gorm:"type:varchar(64);not null"`
	Label        string         `json:"label" gorm:"type:varchar(128);not null"`
	Category     string         `json:"category" gorm:"type:varchar(32);not null"`
	IsInitial    bool           `json:"is_initial" gorm:"not null;default:false"`
	IsTerminal   bool           `json:"is_terminal" gorm:"not null;default:false"`
	DisplayOrder int            `json:"display_order" gorm:"not null;default:0"`
	Color        string         `json:"color" gorm:"type:varchar(32);not null;default:''"`
	Description  string         `json:"description" gorm:"type:varchar(512);not null;default:''"`
	IsSystem     bool           `json:"is_system" gorm:"not null;default:false"`
	Enabled      bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedBy    string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceReminderStatus) TableName() string { return "service_reminder_statuses" }

func (s *ServiceReminderStatus) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.StatusKey = strings.ToLower(strings.TrimSpace(s.StatusKey))
	s.Label = strings.TrimSpace(s.Label)
	s.Category = strings.TrimSpace(s.Category)
	s.Color = strings.TrimSpace(s.Color)
	s.Description = strings.TrimSpace(s.Description)
	s.CreatedBy = strings.TrimSpace(s.CreatedBy)
	return s.Validate()
}

func (s ServiceReminderStatus) Validate() error {
	if strings.TrimSpace(s.ServiceID) == "" {
		return fmt.Errorf("service reminder status service_id is required")
	}
	if _, err := NormalizeServiceReminderStatusKey(s.StatusKey); err != nil {
		return err
	}
	if s.Label == "" || utf8.RuneCountInString(s.Label) > ServiceReminderStatusLabelMaxLen {
		return fmt.Errorf("service reminder status label is invalid")
	}
	if !IsValidServiceReminderStatusCategory(s.Category) {
		return fmt.Errorf("service reminder status category is invalid")
	}
	if utf8.RuneCountInString(s.Color) > ServiceReminderStatusColorMaxLen {
		return fmt.Errorf("service reminder status color is too long")
	}
	if utf8.RuneCountInString(s.Description) > ServiceReminderStatusDescriptionMaxLen {
		return fmt.Errorf("service reminder status description is too long")
	}
	return nil
}

// ServiceReminderStatusTransition is an allow-list edge in the service's
// reminder status graph. Self-transitions are implicit and are not persisted.
type ServiceReminderStatusTransition struct {
	ID           string      `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64      `json:"tenant_id" gorm:"not null;index"`
	ServiceID    string      `json:"service_id" gorm:"type:varchar(36);not null;index"`
	FromStatusID string      `json:"from_status_id" gorm:"type:varchar(36);not null"`
	ToStatusID   string      `json:"to_status_id" gorm:"type:varchar(36);not null"`
	AllowedRoles StringArray `json:"allowed_roles" gorm:"type:jsonb;not null;default:'[]'"`
	Enabled      bool        `json:"enabled" gorm:"not null;default:true"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (ServiceReminderStatusTransition) TableName() string {
	return "service_reminder_status_transitions"
}

func (t *ServiceReminderStatusTransition) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	t.ServiceID = strings.TrimSpace(t.ServiceID)
	t.FromStatusID = strings.TrimSpace(t.FromStatusID)
	t.ToStatusID = strings.TrimSpace(t.ToStatusID)
	if t.AllowedRoles == nil {
		t.AllowedRoles = StringArray{}
	}
	if t.ServiceID == "" || t.FromStatusID == "" || t.ToStatusID == "" {
		return fmt.Errorf("service reminder status transition endpoints are required")
	}
	if t.FromStatusID == t.ToStatusID {
		return fmt.Errorf("service reminder status self-transition is implicit")
	}
	return nil
}

type ServiceReminderStatusCreateInput struct {
	StatusKey    string `json:"status_key"`
	Label        string `json:"label"`
	Category     string `json:"category"`
	IsInitial    bool   `json:"is_initial"`
	IsTerminal   bool   `json:"is_terminal"`
	DisplayOrder int    `json:"display_order"`
	Color        string `json:"color,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type ServiceReminderStatusUpdateInput struct {
	Label        *string `json:"label,omitempty"`
	IsInitial    *bool   `json:"is_initial,omitempty"`
	IsTerminal   *bool   `json:"is_terminal,omitempty"`
	DisplayOrder *int    `json:"display_order,omitempty"`
	Color        *string `json:"color,omitempty"`
	Description  *string `json:"description,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

type ServiceReminderStatusTransitionInput struct {
	FromStatusID string      `json:"from_status_id"`
	ToStatusID   string      `json:"to_status_id"`
	AllowedRoles StringArray `json:"allowed_roles,omitempty"`
	Enabled      bool        `json:"enabled"`
}

type ServiceReminderStatusTransitionReplaceInput struct {
	Transitions []ServiceReminderStatusTransitionInput `json:"transitions"`
}

func IsValidServiceReminderStatusCategory(category string) bool {
	switch strings.TrimSpace(category) {
	case ServiceReminderStatusCategoryOpen,
		ServiceReminderStatusCategoryInProgress,
		ServiceReminderStatusCategoryDone,
		ServiceReminderStatusCategoryDismissed:
		return true
	default:
		return false
	}
}

func NormalizeServiceReminderStatusKey(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("service reminder status key is required")
	}
	if utf8.RuneCountInString(value) > ServiceReminderStatusKeyMaxLen {
		return "", fmt.Errorf("service reminder status key exceeds %d characters", ServiceReminderStatusKeyMaxLen)
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			if i == 0 && (r < 'a' || r > 'z') {
				return "", fmt.Errorf("service reminder status key must start with a lowercase letter")
			}
			continue
		}
		return "", fmt.Errorf("service reminder status key contains invalid character %q", r)
	}
	return value, nil
}
