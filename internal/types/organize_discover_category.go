package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OrganizeDiscoverCategoryStatusEnabled  = "enabled"
	OrganizeDiscoverCategoryStatusDisabled = "disabled"
)

type OrganizeDiscoverCategoryRecord struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	Key         string         `json:"key" gorm:"type:varchar(64);not null;uniqueIndex"`
	Label       string         `json:"label" gorm:"type:varchar(128);not null"`
	Description string         `json:"description" gorm:"type:text;not null;default:''"`
	SortOrder   int            `json:"sort_order" gorm:"not null;default:0"`
	Status      string         `json:"status" gorm:"type:varchar(32);not null;default:'enabled';index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (OrganizeDiscoverCategoryRecord) TableName() string {
	return "organize_discover_categories"
}

func (c *OrganizeDiscoverCategoryRecord) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.Status == "" {
		c.Status = OrganizeDiscoverCategoryStatusEnabled
	}
	return nil
}

type OrganizeDiscoverCategoryInput struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type OrganizeDiscoverCategoryQuery struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}
