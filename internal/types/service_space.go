package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ServiceSpaceStateDraft    = "draft"
	ServiceSpaceStateActive   = "active"
	ServiceSpaceStatePaused   = "paused"
	ServiceSpaceStateArchived = "archived"

	ServiceSpaceVisibilityPrivate = "private"
	ServiceSpaceVisibilityTenant  = "tenant"

	ServiceMemberRoleOwner  = "owner"
	ServiceMemberRoleAdmin  = "admin"
	ServiceMemberRoleEditor = "editor"
	ServiceMemberRoleViewer = "viewer"

	ServiceMemberStatusActive = "active"
	ServiceMemberStatusLeft   = "left"

	ServiceArtifactLifecycleTemporary = "temporary"
	ServiceArtifactLifecycleSaved     = "saved"
	ServiceArtifactLifecycleShared    = "shared"
	ServiceArtifactLifecycleArchived  = "archived"
)

// ServiceSpace is the durable service workspace boundary. The table name stays
// "services" while the Go name avoids colliding with application services.
type ServiceSpace struct {
	ID                    string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID              uint64         `json:"tenant_id" gorm:"not null;index"`
	OwnerUserID           string         `json:"owner_user_id" gorm:"type:varchar(36);not null;index"`
	Name                  string         `json:"name" gorm:"type:varchar(255);not null"`
	Description           string         `json:"description" gorm:"type:text;not null;default:''"`
	Instruction           string         `json:"instruction" gorm:"type:text;not null;default:''"`
	KnowledgeBaseIDs      StringArray    `json:"knowledge_base_ids" gorm:"type:jsonb;not null;default:'[]'"`
	SelectedSkills        StringArray    `json:"selected_skills" gorm:"type:jsonb;not null;default:'[]'"`
	CampusScope           StringArray    `json:"campus_scope" gorm:"type:jsonb;not null;default:'[]'"`
	CourseScope           StringArray    `json:"course_scope" gorm:"type:jsonb;not null;default:'[]'"`
	TemplateKey           string         `json:"template_key" gorm:"type:varchar(64);not null;default:'';index"`
	State                 string         `json:"state" gorm:"type:varchar(32);not null;default:'draft';index"`
	IsDefault             bool           `json:"is_default" gorm:"not null;default:false;index"`
	Visibility            string         `json:"visibility" gorm:"type:varchar(32);not null;default:'private'"`
	MemberLimit           int            `json:"member_limit" gorm:"not null;default:20"`
	Settings              JSONMap        `json:"settings" gorm:"type:jsonb;not null;default:'{}'"`
	Metadata              JSONMap        `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	MigratedFromProfileID string         `json:"migrated_from_profile_id,omitempty" gorm:"type:varchar(36);index"`
	CreatedBy             string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	UpdatedBy             string         `json:"updated_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceSpace) TableName() string { return "services" }

func (s *ServiceSpace) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.State == "" {
		s.State = ServiceSpaceStateDraft
	}
	if s.Visibility == "" {
		s.Visibility = ServiceSpaceVisibilityPrivate
	}
	if s.MemberLimit == 0 {
		s.MemberLimit = 20
	}
	if s.KnowledgeBaseIDs == nil {
		s.KnowledgeBaseIDs = StringArray{}
	}
	if s.SelectedSkills == nil {
		s.SelectedSkills = StringArray{}
	}
	if s.CampusScope == nil {
		s.CampusScope = StringArray{}
	}
	if s.CourseScope == nil {
		s.CourseScope = StringArray{}
	}
	if s.Settings == nil {
		s.Settings = JSONMap{}
	}
	if s.Metadata == nil {
		s.Metadata = JSONMap{}
	}
	return nil
}

type ServiceSpaceMember struct {
	ID        string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	UserID    string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Role      string         `json:"role" gorm:"type:varchar(32);not null;default:'viewer'"`
	Status    string         `json:"status" gorm:"type:varchar(16);not null;default:'active';index"`
	InvitedBy string         `json:"invited_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	JoinedAt  *time.Time     `json:"joined_at,omitempty"`
	LeftAt    *time.Time     `json:"left_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceSpaceMember) TableName() string { return "service_members" }

func (m *ServiceSpaceMember) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Role == "" {
		m.Role = ServiceMemberRoleViewer
	}
	if m.Status == "" {
		m.Status = ServiceMemberStatusActive
	}
	if m.JoinedAt == nil {
		now := time.Now().UTC()
		m.JoinedAt = &now
	}
	return nil
}

type ServiceExpertBinding struct {
	ID                  string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID            uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID           string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	ExpertRef           string         `json:"expert_ref" gorm:"type:varchar(128);not null"`
	ExpertName          string         `json:"expert_name" gorm:"type:varchar(255);not null;default:''"`
	ExpertDomain        string         `json:"expert_domain" gorm:"type:varchar(64);not null;default:''"`
	Source              string         `json:"source" gorm:"type:varchar(32);not null;default:'builtin'"`
	Enabled             bool           `json:"enabled" gorm:"not null;default:true"`
	DisplayOrder        int            `json:"display_order" gorm:"not null;default:0"`
	InstructionOverride string         `json:"instruction_override" gorm:"type:text;not null;default:''"`
	WorkDocDirectory    string         `json:"work_doc_directory" gorm:"type:varchar(255);not null;default:''"`
	OutputPolicy        JSONMap        `json:"output_policy" gorm:"type:jsonb;not null;default:'{}'"`
	MemoryFilter        JSONMap        `json:"memory_filter" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy           string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	UpdatedBy           string         `json:"updated_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceExpertBinding) TableName() string { return "service_expert_bindings" }

func (b *ServiceExpertBinding) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if b.Source == "" {
		b.Source = "builtin"
	}
	if b.OutputPolicy == nil {
		b.OutputPolicy = JSONMap{}
	}
	if b.MemoryFilter == nil {
		b.MemoryFilter = JSONMap{}
	}
	return nil
}

// ServiceArtifact is a service-domain index over Agent artifact versions.
// ArtifactID is the logical artifact identity; VersionID is immutable.
type ServiceArtifact struct {
	ID           string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64         `json:"tenant_id" gorm:"not null;index"`
	ServiceID    string         `json:"service_id" gorm:"type:varchar(36);not null;index"`
	SubjectID    string         `json:"subject_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	RunID        string         `json:"run_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	WorkDocID    string         `json:"work_doc_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	ArtifactID   string         `json:"artifact_id" gorm:"type:varchar(36);not null;index"`
	VersionID    string         `json:"version_id" gorm:"type:varchar(36);not null;index"`
	Kind         string         `json:"kind" gorm:"type:varchar(64);not null;default:'document'"`
	Format       string         `json:"format,omitempty" gorm:"type:varchar(64);not null;default:''"`
	Title        string         `json:"title" gorm:"type:varchar(512);not null;default:''"`
	Summary      string         `json:"summary" gorm:"type:text;not null;default:''"`
	ResourceID   string         `json:"resource_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	ResourceRef  string         `json:"resource_ref,omitempty" gorm:"type:text;not null;default:''"`
	MimeType     string         `json:"mime_type,omitempty" gorm:"type:varchar(255);not null;default:''"`
	OriginalName string         `json:"original_name,omitempty" gorm:"type:varchar(1024);not null;default:''"`
	Lifecycle    string         `json:"lifecycle" gorm:"type:varchar(32);not null;default:'saved';index"`
	Version      int            `json:"version" gorm:"not null;default:1"`
	IsCurrent    bool           `json:"is_current" gorm:"not null;default:true;index"`
	Previewable  bool           `json:"previewable" gorm:"not null;default:false"`
	Downloadable bool           `json:"downloadable" gorm:"not null;default:false"`
	Metadata     JSONMap        `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy    string         `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceArtifact) TableName() string { return "service_artifacts" }

func (a *ServiceArtifact) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.ArtifactID == "" {
		a.ArtifactID = uuid.NewString()
	}
	if a.VersionID == "" {
		a.VersionID = uuid.NewString()
	}
	if a.Lifecycle == "" {
		a.Lifecycle = ServiceArtifactLifecycleSaved
	}
	if a.Version < 1 {
		a.Version = 1
	}
	if a.Metadata == nil {
		a.Metadata = JSONMap{}
	}
	return nil
}

type ServiceSpaceView struct {
	ServiceSpace
	Role        string `json:"role" gorm:"column:role"`
	MemberCount int64  `json:"member_count" gorm:"column:member_count"`
}

type ServiceExpertBindingInput struct {
	ExpertRef           string  `json:"expert_ref"`
	ExpertName          string  `json:"expert_name"`
	ExpertDomain        string  `json:"expert_domain,omitempty"`
	Source              string  `json:"source,omitempty"`
	Enabled             *bool   `json:"enabled,omitempty"`
	DisplayOrder        int     `json:"display_order,omitempty"`
	InstructionOverride string  `json:"instruction_override,omitempty"`
	WorkDocDirectory    string  `json:"work_doc_directory,omitempty"`
	OutputPolicy        JSONMap `json:"output_policy,omitempty"`
	MemoryFilter        JSONMap `json:"memory_filter,omitempty"`
}

type ServiceSpaceCreateInput struct {
	Name             string                      `json:"name"`
	Description      string                      `json:"description,omitempty"`
	Instruction      string                      `json:"instruction,omitempty"`
	KnowledgeBaseIDs StringArray                 `json:"knowledge_base_ids,omitempty"`
	SelectedSkills   StringArray                 `json:"selected_skills,omitempty"`
	TemplateKey      string                      `json:"template_key,omitempty"`
	Experts          []ServiceExpertBindingInput `json:"experts,omitempty"`
	Activate         bool                        `json:"activate,omitempty"`
}

type ServiceSpaceUpdateInput struct {
	Name             *string      `json:"name,omitempty"`
	Description      *string      `json:"description,omitempty"`
	Instruction      *string      `json:"instruction,omitempty"`
	KnowledgeBaseIDs *StringArray `json:"knowledge_base_ids,omitempty"`
	SelectedSkills   *StringArray `json:"selected_skills,omitempty"`
	CampusScope      *StringArray `json:"campus_scope,omitempty"`
	CourseScope      *StringArray `json:"course_scope,omitempty"`
	Visibility       *string      `json:"visibility,omitempty"`
	MemberLimit      *int         `json:"member_limit,omitempty"`
}

type ServiceSessionCreateInput struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ExpertRef   string `json:"expert_ref,omitempty"`
	ExpertName  string `json:"expert_name,omitempty"`
}

type ServiceSessionUpdateInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	ExpertRef   *string `json:"expert_ref,omitempty"`
	ExpertName  *string `json:"expert_name,omitempty"`
}

type ServiceMemberInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type ServiceSpaceOverview struct {
	Service         *ServiceSpaceView  `json:"service"`
	SessionCount    int64              `json:"session_count"`
	ArtifactCount   int64              `json:"artifact_count"`
	MemberCount     int64              `json:"member_count"`
	RecentSessions  []*Session         `json:"recent_sessions"`
	RecentArtifacts []*ServiceArtifact `json:"recent_artifacts"`
}

func IsValidServiceSpaceState(state string) bool {
	switch state {
	case ServiceSpaceStateDraft, ServiceSpaceStateActive, ServiceSpaceStatePaused, ServiceSpaceStateArchived:
		return true
	default:
		return false
	}
}

func IsValidServiceMemberRole(role string) bool {
	switch role {
	case ServiceMemberRoleOwner, ServiceMemberRoleAdmin, ServiceMemberRoleEditor, ServiceMemberRoleViewer:
		return true
	default:
		return false
	}
}

func ServiceMemberRoleRank(role string) int {
	switch role {
	case ServiceMemberRoleOwner:
		return 4
	case ServiceMemberRoleAdmin:
		return 3
	case ServiceMemberRoleEditor:
		return 2
	case ServiceMemberRoleViewer:
		return 1
	default:
		return 0
	}
}
