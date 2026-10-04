package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceFactProposalStatusPending   = "pending"
	ServiceFactProposalStatusConfirmed = "confirmed"
	ServiceFactProposalStatusRejected  = "rejected"
)

type ServiceFactProposalItem struct {
	FieldKey   string  `json:"field_key"`
	FieldLabel string  `json:"field_label"`
	Value      any     `json:"value"`
	Confidence float64 `json:"confidence,omitempty"`
	Evidence   string  `json:"evidence,omitempty"`
	SubjectID  string  `json:"subject_id,omitempty"`
}

type ServiceFactProposal struct {
	ID            string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64                    `json:"tenant_id" gorm:"not null;index"`
	ServiceID     string                    `json:"service_id" gorm:"type:varchar(36);not null;index"`
	SessionID     string                    `json:"session_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	SourceType    string                    `json:"source_type" gorm:"type:varchar(64);not null"`
	SourceID      string                    `json:"source_id" gorm:"type:varchar(128);not null"`
	SourceVersion string                    `json:"source_version,omitempty" gorm:"type:varchar(128);not null;default:''"`
	SubjectID     string                    `json:"subject_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	Status        string                    `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	NeedsSubject  bool                      `json:"needs_subject"`
	Question      string                    `json:"question,omitempty"`
	Items         []ServiceFactProposalItem `json:"items" gorm:"-"`
	ItemsJSON     JSON                      `json:"-" gorm:"column:proposal_items;type:jsonb;not null;default:'[]'"`
	CreatedBy     string                    `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	ResolvedBy    string                    `json:"resolved_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	ResolvedAt    *time.Time                `json:"resolved_at,omitempty"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

func (ServiceFactProposal) TableName() string { return "service_fact_proposals" }

func (p *ServiceFactProposal) BeforeSave(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.ServiceID = strings.TrimSpace(p.ServiceID)
	p.SessionID = strings.TrimSpace(p.SessionID)
	p.SourceType = strings.TrimSpace(p.SourceType)
	p.SourceID = strings.TrimSpace(p.SourceID)
	p.SourceVersion = strings.TrimSpace(p.SourceVersion)
	p.SubjectID = strings.TrimSpace(p.SubjectID)
	p.Status = strings.TrimSpace(p.Status)
	if p.Status == "" {
		p.Status = ServiceFactProposalStatusPending
	}
	if len(p.ItemsJSON) == 0 {
		encoded, err := json.Marshal(p.Items)
		if err != nil {
			return err
		}
		p.ItemsJSON = encoded
	}
	return p.Validate()
}

func (p *ServiceFactProposal) AfterFind(_ *gorm.DB) error {
	if len(p.ItemsJSON) == 0 {
		p.Items = []ServiceFactProposalItem{}
		return nil
	}
	return json.Unmarshal(p.ItemsJSON, &p.Items)
}

func (p *ServiceFactProposal) Validate() error {
	if p.TenantID == 0 || p.ServiceID == "" {
		return fmt.Errorf("service fact proposal scope is required")
	}
	if p.SourceType == "" || p.SourceID == "" {
		return fmt.Errorf("service fact proposal source is required")
	}
	switch p.Status {
	case ServiceFactProposalStatusPending,
		ServiceFactProposalStatusConfirmed,
		ServiceFactProposalStatusRejected:
	default:
		return fmt.Errorf("invalid service fact proposal status")
	}
	if len(p.Items) > 50 {
		return fmt.Errorf("service fact proposal contains too many items")
	}
	return nil
}

type ServiceFactProposalPreviewInput struct {
	SessionID     string `json:"session_id,omitempty"`
	Text          string `json:"text"`
	SubjectID     string `json:"subject_id,omitempty"`
	SourceType    string `json:"source_type"`
	SourceID      string `json:"source_id"`
	SourceVersion string `json:"source_version,omitempty"`
}

func (i ServiceFactProposalPreviewInput) Validate() error {
	if strings.TrimSpace(i.Text) == "" {
		return fmt.Errorf("proposal text is required")
	}
	if strings.TrimSpace(i.SourceType) == "" {
		return fmt.Errorf("proposal source_type is required")
	}
	if strings.TrimSpace(i.SourceID) == "" {
		return fmt.Errorf("proposal source_id is required")
	}
	return nil
}

type ServiceFactProposalResolveInput struct {
	Decision  string                     `json:"decision"`
	SubjectID string                     `json:"subject_id,omitempty"`
	Items     *[]ServiceFactProposalItem `json:"items,omitempty"`
}

func (i ServiceFactProposalResolveInput) Validate() error {
	switch strings.ToLower(strings.TrimSpace(i.Decision)) {
	case ServiceFactProposalStatusConfirmed, ServiceFactProposalStatusRejected:
		return nil
	default:
		return fmt.Errorf("proposal decision must be confirmed or rejected")
	}
}
