package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OrganizeTabMemory = "memory"
	OrganizeTabOutput = "output"
	OrganizeTabSprout = "sprout"

	OrganizeMemoryKindNote      = "note"
	OrganizeMemoryKindRecord    = "record"
	OrganizeMemoryKindAudio     = "audio"
	OrganizeMemoryKindAudioCard = "audio_card"

	OrganizeOutputStatusDraft    = "draft"
	OrganizeOutputStatusReview   = "review"
	OrganizeOutputStatusReady    = "ready"
	OrganizeOutputStatusArchived = "archived"

	OrganizeSproutStageOrganizing = "organizing"
	OrganizeSproutStageExpandable = "expandable"
	OrganizeSproutStageFormed     = "formed"
)

func IsValidOrganizeMemoryKind(kind string) bool {
	switch kind {
	case OrganizeMemoryKindNote, OrganizeMemoryKindRecord, OrganizeMemoryKindAudio, OrganizeMemoryKindAudioCard:
		return true
	default:
		return false
	}
}

func IsValidOrganizeOutputStatus(status string) bool {
	switch status {
	case OrganizeOutputStatusDraft, OrganizeOutputStatusReview, OrganizeOutputStatusReady, OrganizeOutputStatusArchived:
		return true
	default:
		return false
	}
}

func IsValidOrganizeSproutStage(stage string) bool {
	switch stage {
	case OrganizeSproutStageOrganizing, OrganizeSproutStageExpandable, OrganizeSproutStageFormed:
		return true
	default:
		return false
	}
}

// OrganizeMemory is a user-scoped memory item in the Organize section.
type OrganizeMemory struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID          string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Kind            string         `json:"kind" gorm:"type:varchar(32);not null;index"`
	Title           string         `json:"title" gorm:"type:varchar(512);not null"`
	Content         string         `json:"content,omitempty" gorm:"type:text;not null;default:''"`
	Source          string         `json:"source,omitempty" gorm:"type:varchar(255);not null;default:''"`
	OccurredAt      time.Time      `json:"occurred_at" gorm:"not null;index"`
	DurationSeconds int            `json:"duration_seconds,omitempty" gorm:"not null;default:0"`
	Metadata        JSONMap        `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (OrganizeMemory) TableName() string { return "organize_memories" }

func (m *OrganizeMemory) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.OccurredAt.IsZero() {
		m.OccurredAt = time.Now().UTC()
	}
	if m.Metadata == nil {
		m.Metadata = JSONMap{}
	}
	return nil
}

// OrganizeMemoryReference is the compact memory citation returned on generated reports.
type OrganizeMemoryReference struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Source string `json:"source,omitempty"`
}

// OrganizeOutput is a user-scoped deliverable produced from memories.
type OrganizeOutput struct {
	ID            string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID        string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Title         string         `json:"title" gorm:"type:varchar(512);not null"`
	OutputType    string         `json:"output_type" gorm:"type:varchar(64);not null;default:''"`
	Content       string         `json:"content,omitempty" gorm:"type:text;not null;default:''"`
	SourceSummary string         `json:"source_summary,omitempty" gorm:"type:varchar(255);not null;default:''"`
	Status        string         `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	Icon          string         `json:"icon,omitempty" gorm:"type:varchar(64);not null;default:''"`
	Metadata      JSONMap        `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryCount   int64          `json:"memory_count" gorm:"-"`
	MemoryIDs     []string       `json:"memory_ids,omitempty" gorm:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (OrganizeOutput) TableName() string { return "organize_outputs" }

func (o *OrganizeOutput) BeforeCreate(_ *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.Status == "" {
		o.Status = OrganizeOutputStatusDraft
	}
	if o.Metadata == nil {
		o.Metadata = JSONMap{}
	}
	return nil
}

// OrganizeOutputMemory links outputs to the memories they were produced from.
type OrganizeOutputMemory struct {
	OutputID  string    `json:"output_id" gorm:"type:varchar(36);primaryKey"`
	MemoryID  string    `json:"memory_id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64    `json:"tenant_id" gorm:"not null;index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt time.Time `json:"created_at"`
}

func (OrganizeOutputMemory) TableName() string { return "organize_output_memories" }

// OrganizeSproutReport is a user-scoped synthesis prompt/report over memories.
type OrganizeSproutReport struct {
	ID          string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64                    `json:"tenant_id" gorm:"not null;index"`
	UserID      string                    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Title       string                    `json:"title" gorm:"type:varchar(512);not null"`
	Summary     string                    `json:"summary,omitempty" gorm:"type:text;not null;default:''"`
	Stage       string                    `json:"stage" gorm:"type:varchar(64);not null;default:'organizing';index"`
	OutputHint  string                    `json:"output_hint,omitempty" gorm:"type:varchar(255);not null;default:''"`
	Chips       StringArray               `json:"chips,omitempty" gorm:"type:jsonb;not null;default:'[]'"`
	Metadata    JSONMap                   `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryCount int64                     `json:"memory_count" gorm:"-"`
	MemoryIDs   []string                  `json:"memory_ids,omitempty" gorm:"-"`
	MemoryRefs  []OrganizeMemoryReference `json:"memory_refs,omitempty" gorm:"-"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	DeletedAt   gorm.DeletedAt            `json:"deleted_at" gorm:"index"`
}

func (OrganizeSproutReport) TableName() string { return "organize_sprout_reports" }

func (r *OrganizeSproutReport) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Stage == "" {
		r.Stage = OrganizeSproutStageOrganizing
	}
	if r.Chips == nil {
		r.Chips = StringArray{}
	}
	if r.Metadata == nil {
		r.Metadata = JSONMap{}
	}
	return nil
}

// OrganizeSproutMemory links sprout reports to the memories they summarize.
type OrganizeSproutMemory struct {
	ReportID  string    `json:"report_id" gorm:"type:varchar(36);primaryKey"`
	MemoryID  string    `json:"memory_id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64    `json:"tenant_id" gorm:"not null;index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt time.Time `json:"created_at"`
}

func (OrganizeSproutMemory) TableName() string { return "organize_sprout_memories" }

type OrganizeListQuery struct {
	TenantID uint64
	UserID   string
	Keyword  string
	Kind     string
	Status   string
	Stage    string
	MemoryID string
	Page     int
	PageSize int
}

type OrganizeDiscoverQuery struct {
	TenantID       uint64
	UserID         string
	Keyword        string
	Tab            string
	Page           int
	PageSize       int
	FeaturedOffset int
}

type OrganizeDiscoverTab struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type OrganizeDiscover struct {
	Tabs            []OrganizeDiscoverTab `json:"tabs"`
	FeaturedOutputs []*OrganizeOutput     `json:"featured_outputs"`
	Items           []*OrganizeOutput     `json:"items"`
	Total           int64                 `json:"total"`
	Page            int                   `json:"page"`
	PageSize        int                   `json:"page_size"`
	FeaturedOffset  int                   `json:"featured_offset"`
}

type OrganizeMemoryInput struct {
	Kind            string     `json:"kind"`
	Title           string     `json:"title"`
	Content         string     `json:"content,omitempty"`
	Source          string     `json:"source,omitempty"`
	OccurredAt      *time.Time `json:"occurred_at,omitempty"`
	DurationSeconds int        `json:"duration_seconds,omitempty"`
	Metadata        JSONMap    `json:"metadata,omitempty"`
}

type OrganizeOutputInput struct {
	Title         string   `json:"title"`
	OutputType    string   `json:"output_type,omitempty"`
	Content       string   `json:"content,omitempty"`
	SourceSummary string   `json:"source_summary,omitempty"`
	Status        string   `json:"status,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	MemoryIDs     []string `json:"memory_ids,omitempty"`
	Metadata      JSONMap  `json:"metadata,omitempty"`
}

type OrganizeSproutReportInput struct {
	Title      string      `json:"title"`
	Summary    string      `json:"summary,omitempty"`
	Stage      string      `json:"stage,omitempty"`
	OutputHint string      `json:"output_hint,omitempty"`
	Chips      StringArray `json:"chips,omitempty"`
	MemoryIDs  []string    `json:"memory_ids,omitempty"`
	Metadata   JSONMap     `json:"metadata,omitempty"`
}

type OrganizeSproutFromMemoryInput struct {
	MemoryID   string  `json:"memory_id"`
	ModelID    string  `json:"model_id,omitempty"`
	RoleConfig JSONMap `json:"role_config,omitempty"`
}

type OrganizeTabSummary struct {
	Key          string     `json:"key"`
	Count        int64      `json:"count"`
	UpdatedToday int64      `json:"updated_today,omitempty"`
	LatestAt     *time.Time `json:"latest_at,omitempty"`
}

type OrganizeMemoryAssetSummary struct {
	Kind  string `json:"kind"`
	Count int64  `json:"count"`
}

type OrganizeOverview struct {
	Tabs        []OrganizeTabSummary         `json:"tabs"`
	MemoryKinds []OrganizeMemoryAssetSummary `json:"memory_kinds"`
	OutputStats map[string]int64             `json:"output_stats"`
	SproutStats map[string]int64             `json:"sprout_stats"`
}
