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
	ServiceSubjectTypeCustom = "custom"
	ServiceSubjectTypeMaxLen = 64
	ServiceSubjectKeyMaxLen  = 255
)

// ServiceSubject is the generic business object served by a service space.
// The platform does not hard-code domain-specific subject types.
type ServiceSubject struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID       string         `json:"service_id" gorm:"type:varchar(36);index"`
	OwnerUserID     string         `json:"owner_user_id,omitempty" gorm:"type:varchar(36);index"`
	SubjectType     string         `json:"subject_type" gorm:"type:varchar(64);index"`
	SubjectKey      string         `json:"subject_key" gorm:"type:varchar(255);not null"`
	DisplayName     string         `json:"display_name" gorm:"type:varchar(255);not null;default:''"`
	ParentSubjectID *string        `json:"parent_subject_id,omitempty" gorm:"type:varchar(36);index"`
	StudentName     string         `json:"student_name,omitempty" gorm:"type:varchar(255);not null;default:''"`
	Relation        string         `json:"relation,omitempty" gorm:"type:varchar(64);not null;default:''"`
	Aliases         StringArray    `json:"aliases,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	ExternalRefs    JSONMap        `json:"external_refs,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	Metadata        JSONMap        `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	VisibilityScope string         `json:"visibility_scope" gorm:"type:varchar(64);not null;default:'private'"`
	Confidence      float64        `json:"confidence" gorm:"not null;default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceSubject) TableName() string { return "service_subjects" }

// NormalizeServiceSubjectType validates a configurable, machine-readable type
// key without restricting the business domain to a platform enum.
func NormalizeServiceSubjectType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("service subject type is required")
	}
	if utf8.RuneCountInString(value) > ServiceSubjectTypeMaxLen {
		return "", fmt.Errorf("service subject type exceeds %d characters", ServiceSubjectTypeMaxLen)
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			if i == 0 && (r < 'a' || r > 'z') {
				return "", fmt.Errorf("service subject type must start with a lowercase letter")
			}
			continue
		}
		return "", fmt.Errorf("service subject type contains invalid character %q", r)
	}
	return value, nil
}

func (s *ServiceSubject) ValidateForServiceWrite() error {
	if strings.TrimSpace(s.ServiceID) == "" {
		return fmt.Errorf("service subject service_id is required")
	}
	if strings.TrimSpace(s.SubjectKey) == "" {
		return fmt.Errorf("service subject subject_key is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(s.SubjectKey)) > ServiceSubjectKeyMaxLen {
		return fmt.Errorf("service subject subject_key exceeds %d characters", ServiceSubjectKeyMaxLen)
	}
	normalizedType, err := NormalizeServiceSubjectType(s.SubjectType)
	if err != nil {
		return err
	}
	s.SubjectType = normalizedType
	s.ServiceID = strings.TrimSpace(s.ServiceID)
	s.SubjectKey = strings.TrimSpace(s.SubjectKey)
	return nil
}

func (s *ServiceSubject) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if strings.TrimSpace(s.SubjectType) == "" {
		s.SubjectType = ServiceSubjectTypeCustom
	}
	if err := s.ValidateForServiceWrite(); err != nil {
		return err
	}
	if s.VisibilityScope == "" {
		s.VisibilityScope = ServiceSpaceVisibilityPrivate
	}
	if s.Aliases == nil {
		s.Aliases = StringArray{}
	}
	if s.ExternalRefs == nil {
		s.ExternalRefs = JSONMap{}
	}
	if s.Metadata == nil {
		s.Metadata = JSONMap{}
	}
	return nil
}
