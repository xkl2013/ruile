package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceContextSourceTypeOrganizeOutput = "organize_output"
)

// ServiceContextSource is an immutable-enough snapshot of a source that was
// explicitly brought into a service space. The original organize output
// remains the source of truth; this snapshot keeps the service context stable
// when the original output is later edited or archived.
type ServiceContextSource struct {
	ID            string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID     string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	SourceType    string         `json:"source_type" gorm:"type:varchar(32);not null"`
	SourceID      string         `json:"source_id" gorm:"type:varchar(36);not null"`
	SourceTitle   string         `json:"source_title" gorm:"type:varchar(512);not null;default:''"`
	SourceVersion string         `json:"source_version" gorm:"type:varchar(64);not null;default:''"`
	SourceSummary string         `json:"source_summary" gorm:"type:text;not null;default:''"`
	SourceContent string         `json:"source_content" gorm:"type:text;not null;default:''"`
	MemoryIDs     StringArray    `json:"memory_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Metadata      JSONMap        `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	ImportedBy    string         `json:"imported_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceContextSource) TableName() string { return "service_context_sources" }

func (source *ServiceContextSource) BeforeCreate(_ *gorm.DB) error {
	if source.ID == "" {
		source.ID = uuid.NewString()
	}
	if source.MemoryIDs == nil {
		source.MemoryIDs = StringArray{}
	}
	if source.Metadata == nil {
		source.Metadata = JSONMap{}
	}
	return nil
}
