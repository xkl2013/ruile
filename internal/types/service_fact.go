package types

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceFactTypeOrganizeOutput = "organize_output"
	ServiceFactTypeMaxLen         = 64
	ServiceFactKeyMaxLen          = 128
	ServiceFactSourceTypeMaxLen   = 64
	ServiceFactSourceIDMaxLen     = 128
)

// ServiceFact is an append-only, structured fact belonging to one service
// space. The source fields are mandatory so every fact can be traced back to
// an imported output, memory, or another first-party source.
type ServiceFact struct {
	ID            string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64    `json:"tenant_id" gorm:"not null;index"`
	ServiceID     string    `json:"service_id" gorm:"type:varchar(36);not null;index"`
	SubjectID     string    `json:"subject_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	FactType      string    `json:"fact_type" gorm:"type:varchar(64);not null;index"`
	FactKey       string    `json:"fact_key,omitempty" gorm:"type:varchar(128);not null;default:''"`
	Value         JSONMap   `json:"value" gorm:"column:fact_value;type:jsonb;not null;default:'{}'"`
	SourceType    string    `json:"source_type" gorm:"type:varchar(64);not null"`
	SourceID      string    `json:"source_id" gorm:"type:varchar(128);not null"`
	SourceVersion string    `json:"source_version,omitempty" gorm:"type:varchar(128);not null;default:''"`
	CreatedBy     string    `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ServiceFact) TableName() string { return "service_facts" }

func (f *ServiceFact) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	if f.Value == nil {
		f.Value = JSONMap{}
	}
	f.ServiceID = strings.TrimSpace(f.ServiceID)
	f.SubjectID = strings.TrimSpace(f.SubjectID)
	f.FactType = strings.TrimSpace(f.FactType)
	f.FactKey = strings.TrimSpace(f.FactKey)
	f.SourceType = strings.TrimSpace(f.SourceType)
	f.SourceID = strings.TrimSpace(f.SourceID)
	f.SourceVersion = strings.TrimSpace(f.SourceVersion)
	f.CreatedBy = strings.TrimSpace(f.CreatedBy)
	return f.ValidateForAppend()
}

func (f *ServiceFact) ValidateForAppend() error {
	if strings.TrimSpace(f.ServiceID) == "" {
		return errServiceFactFieldRequired("service_id")
	}
	if strings.TrimSpace(f.FactType) == "" {
		return errServiceFactFieldRequired("fact_type")
	}
	if utf8.RuneCountInString(f.FactType) > ServiceFactTypeMaxLen {
		return errServiceFactFieldTooLong("fact_type", ServiceFactTypeMaxLen)
	}
	if utf8.RuneCountInString(f.FactKey) > ServiceFactKeyMaxLen {
		return errServiceFactFieldTooLong("fact_key", ServiceFactKeyMaxLen)
	}
	if strings.TrimSpace(f.SourceType) == "" {
		return errServiceFactFieldRequired("source_type")
	}
	if utf8.RuneCountInString(f.SourceType) > ServiceFactSourceTypeMaxLen {
		return errServiceFactFieldTooLong("source_type", ServiceFactSourceTypeMaxLen)
	}
	if strings.TrimSpace(f.SourceID) == "" {
		return errServiceFactFieldRequired("source_id")
	}
	if utf8.RuneCountInString(f.SourceID) > ServiceFactSourceIDMaxLen {
		return errServiceFactFieldTooLong("source_id", ServiceFactSourceIDMaxLen)
	}
	return nil
}

type ServiceFactAppendInput struct {
	SubjectID     string  `json:"subject_id,omitempty"`
	FactType      string  `json:"fact_type"`
	FactKey       string  `json:"fact_key,omitempty"`
	Value         JSONMap `json:"value"`
	SourceType    string  `json:"source_type"`
	SourceID      string  `json:"source_id"`
	SourceVersion string  `json:"source_version,omitempty"`
}

// The service layer maps these validation errors to its public domain errors.
// Keeping the model validation independent avoids importing application code
// into the types package.
type serviceFactFieldError struct {
	field string
	msg   string
}

func (e *serviceFactFieldError) Error() string { return e.msg }

func errServiceFactFieldRequired(field string) error {
	return &serviceFactFieldError{field: field, msg: field + " is required"}
}

func errServiceFactFieldTooLong(field string, max int) error {
	return &serviceFactFieldError{field: field, msg: field + " exceeds character limit"}
}
