package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceReminderAssigneeRolePrimary = "primary"
	ServiceReminderAssigneeRoleMember  = "member"
)

type ServiceReminderAssignee struct {
	ID         string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID  string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ReminderID string         `json:"reminder_id" gorm:"type:varchar(36);not null;index"`
	UserID     string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Role       string         `json:"role" gorm:"type:varchar(16);not null;default:'member'"`
	AssignedBy string         `json:"assigned_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceReminderAssignee) TableName() string { return "service_reminder_assignees" }

func (a *ServiceReminderAssignee) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	a.UserID = strings.TrimSpace(a.UserID)
	a.Role = strings.TrimSpace(a.Role)
	if a.Role == "" {
		a.Role = ServiceReminderAssigneeRoleMember
	}
	return nil
}

type ServiceReminderComment struct {
	ID         string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID  string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ReminderID string         `json:"reminder_id" gorm:"type:varchar(36);not null;index"`
	UserID     string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Content    string         `json:"content" gorm:"type:text;not null"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceReminderComment) TableName() string { return "service_reminder_comments" }

func (c *ServiceReminderComment) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.Content = strings.TrimSpace(c.Content)
	return nil
}

type ServiceReminderHistory struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64    `json:"tenant_id" gorm:"not null;index"`
	ServiceID    string    `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ReminderID   string    `json:"reminder_id" gorm:"type:varchar(36);not null;index"`
	UserID       string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Action       string    `json:"action" gorm:"type:varchar(64);not null"`
	FromStatus   string    `json:"from_status,omitempty" gorm:"type:varchar(32);not null;default:''"`
	ToStatus     string    `json:"to_status,omitempty" gorm:"type:varchar(32);not null;default:''"`
	ChangeDetail JSONMap   `json:"change_detail,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ServiceReminderHistory) TableName() string { return "service_reminder_history" }

func (h *ServiceReminderHistory) BeforeCreate(_ *gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	h.Action = strings.TrimSpace(h.Action)
	if h.ChangeDetail == nil {
		h.ChangeDetail = JSONMap{}
	}
	return nil
}

type ServiceReminderCommentCreateInput struct {
	Content string `json:"content"`
}

type ServiceReminderAssigneeReplaceInput struct {
	UserIDs []string `json:"user_ids"`
}
