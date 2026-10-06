package types

import (
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Course sources. Only "official" is produced today: platform administrators
// create the course metadata first and then add lessons in the admin console.
// "creator" is kept in the vocabulary so a creator-side channel can land later
// without a migration or a second taxonomy.
const (
	OrganizeCourseSourceOfficial = "official"
	OrganizeCourseSourceCreator  = "creator"
)

// Course visibility is independent from publication status. A published
// course can be visible platform-wide, to selected shared spaces, or only to
// its owning workspace/creator.
const (
	OrganizeCourseVisibilitySystem      = "system"
	OrganizeCourseVisibilitySharedSpace = "shared_space"
	OrganizeCourseVisibilityPrivate     = "private"
)

// Course lesson kinds. Same three values as the organize upload pipeline's
// content kinds (see organizeOutputKind* in the service package), so a lesson
// type can be derived from the uploaded file without a second mapping table.
const (
	OrganizeCourseLessonTypeVideo   = "video"
	OrganizeCourseLessonTypeAudio   = "audio"
	OrganizeCourseLessonTypeArticle = "article"
)

const (
	organizeCourseIDPrefix       = "crs_"
	organizeCourseLessonIDPrefix = "lsn_"
)

// OrganizeCourse is the course entity: one row per course, owning an ordered
// list of lessons.
//
// The lesson bodies are deliberately NOT stored here. Each lesson points at an
// existing organize_outputs row through output_id, so the upload, parsing,
// editing and preview pipelines keep exactly one source of truth for content.
type OrganizeCourse struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	UserID          string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Source          string         `json:"source" gorm:"type:varchar(16);not null;default:'official';index"`
	Title           string         `json:"title" gorm:"type:varchar(255);not null"`
	Summary         string         `json:"summary" gorm:"type:text;not null;default:''"`
	Category        string         `json:"category" gorm:"type:varchar(64);not null;default:'';index"`
	CoverURL        string         `json:"cover_url" gorm:"type:varchar(512);not null;default:''"`
	TeacherName     string         `json:"teacher_name" gorm:"type:varchar(64);not null;default:''"`
	TeacherTitle    string         `json:"teacher_title" gorm:"type:varchar(128);not null;default:''"`
	PublicStatus    string         `json:"public_status" gorm:"type:varchar(32);not null;default:'published';index"`
	VisibilityScope string         `json:"visibility_scope" gorm:"type:varchar(32);not null;default:'system';index"`
	Featured        bool           `json:"featured" gorm:"not null;default:false;index"`
	Recommendable   bool           `json:"recommendable" gorm:"not null;default:true;index"`
	SortOrder       int            `json:"sort_order" gorm:"not null;default:0"`
	LessonCount     int            `json:"lesson_count" gorm:"not null;default:0"`
	LearnerCount    int            `json:"learner_count" gorm:"not null;default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Lessons is populated by the read paths that need the outline. It is never
	// persisted here: rows live in organize_course_lessons.
	Lessons []*OrganizeCourseLesson `json:"lessons,omitempty" gorm:"-"`

	// SharedSpaceIDs is hydrated by admin detail/list reads. The IDs live in
	// organize_course_shared_spaces and are not persisted on this row.
	SharedSpaceIDs []string `json:"shared_space_ids,omitempty" gorm:"-"`
}

func (OrganizeCourse) TableName() string { return "organize_courses" }

func (c *OrganizeCourse) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = newOrganizeCourseID(organizeCourseIDPrefix)
	}
	if c.Source == "" {
		c.Source = OrganizeCourseSourceOfficial
	}
	if c.PublicStatus == "" {
		c.PublicStatus = OrganizePublicContentStatusPublished
	}
	if c.VisibilityScope == "" {
		c.VisibilityScope = OrganizeCourseVisibilitySystem
	}
	c.LessonCount = nonNegativeCount(c.LessonCount)
	c.LearnerCount = nonNegativeCount(c.LearnerCount)
	return nil
}

// OrganizeCourseSharedSpace links a course to an existing organization
// (shared space). Membership is resolved through the organization member
// table, so courses do not introduce a second membership model.
type OrganizeCourseSharedSpace struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	CourseID       string    `json:"course_id" gorm:"type:varchar(36);not null;index"`
	OrganizationID string    `json:"organization_id" gorm:"type:varchar(36);not null;index"`
	CreatedBy      string    `json:"created_by" gorm:"type:varchar(36);not null"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (OrganizeCourseSharedSpace) TableName() string {
	return "organize_course_shared_spaces"
}

// OrganizeCourseLesson is one position inside a course. It owns the ordering
// and the display title only; the body is read from OrganizeOutput by OutputID.
type OrganizeCourseLesson struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	CourseID        string    `json:"course_id" gorm:"type:varchar(36);not null;index"`
	TenantID        uint64    `json:"tenant_id" gorm:"not null;index"`
	OutputID        string    `json:"output_id" gorm:"type:varchar(36);not null;index"`
	Title           string    `json:"title" gorm:"type:varchar(512);not null"`
	LessonType      string    `json:"lesson_type" gorm:"type:varchar(16);not null;default:'article'"`
	DurationSeconds int       `json:"duration_seconds" gorm:"not null;default:0"`
	SortOrder       int       `json:"sort_order" gorm:"not null;default:0;index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Output is filled by detail reads so the learning page can render the body
	// without an extra round trip per lesson.
	Output *OrganizeOutput `json:"output,omitempty" gorm:"-:all"`
}

func (OrganizeCourseLesson) TableName() string { return "organize_course_lessons" }

// OrganizeCourseLessonUpdateInput contains the administrator-editable fields
// for one lesson. The uploaded file itself remains immutable; replacing a
// lesson is done by deleting it and uploading a new lesson.
type OrganizeCourseLessonUpdateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// OrganizePublicCourse is the redacted course contract used by Discover.
// Tenant, user, output and storage fields deliberately do not cross this
// boundary.
type OrganizePublicCourse struct {
	ID            string                        `json:"id"`
	Source        string                        `json:"source"`
	Title         string                        `json:"title"`
	Summary       string                        `json:"summary"`
	Category      string                        `json:"category"`
	CoverURL      string                        `json:"cover_url"`
	TeacherName   string                        `json:"teacher_name"`
	TeacherTitle  string                        `json:"teacher_title"`
	LessonCount   int                           `json:"lesson_count"`
	LearnerCount  int                           `json:"learner_count"`
	Featured      bool                          `json:"featured"`
	Recommendable bool                          `json:"recommendable"`
	SortOrder     int                           `json:"sort_order"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
	Lessons       []*OrganizePublicCourseLesson `json:"lessons,omitempty"`
}

// OrganizePublicCourseLesson is the redacted lesson contract used by the
// learning page. Media is served through MediaURL and the body through
// GET /courses/:id/lessons/:lesson_id/content, both after course/lesson access
// is checked by the backend. Neither is inlined here.
type OrganizePublicCourseLesson struct {
	ID              string    `json:"id"`
	CourseID        string    `json:"course_id"`
	Title           string    `json:"title"`
	LessonType      string    `json:"lesson_type"`
	DurationSeconds int       `json:"duration_seconds"`
	SortOrder       int       `json:"sort_order"`
	Available       bool      `json:"available"`
	MediaURL        string    `json:"media_url,omitempty"`
	SourceFileName  string    `json:"source_file_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (l *OrganizeCourseLesson) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		l.ID = newOrganizeCourseID(organizeCourseLessonIDPrefix)
	}
	if l.LessonType == "" {
		l.LessonType = OrganizeCourseLessonTypeArticle
	}
	l.DurationSeconds = nonNegativeCount(l.DurationSeconds)
	l.SortOrder = nonNegativeCount(l.SortOrder)
	return nil
}

// OrganizeCourseQuery filters course list reads. TenantID/UserID scope the list
// to the caller's workspace; empty filters mean "no constraint".
type OrganizeCourseQuery struct {
	TenantID      uint64
	UserID        string
	Keyword       string
	Category      string
	Source        string
	PublicStatus  string
	Featured      *bool
	Recommendable *bool
	Page          int
	PageSize      int

	// VisibilityFilter is set only for public published-course reads.
	// Admin list reads intentionally remain platform-wide.
	VisibilityFilter      bool
	ViewerTenantID        uint64
	ViewerUserID          string
	ViewerOrganizationIDs []string
}

type OrganizeCourseStats struct {
	Total       int64 `json:"total"`
	Published   int64 `json:"published"`
	Offline     int64 `json:"offline"`
	LessonTotal int64 `json:"lesson_total"`
}

// OrganizeCourseUploadInput carries the course-level metadata an administrator
// fills in the upload wizard. Lesson-level titles come from the file names.
type OrganizeCourseUploadInput struct {
	Title              string
	Summary            string
	Category           string
	CoverURL           string
	CoverImageFileName string
	CoverImageMimeType string
	CoverImageData     []byte
	TeacherName        string
	TeacherTitle       string
	Source             string
	PublicStatus       string
	Featured           bool
	Recommendable      bool
	SortOrder          int
	VisibilityScope    string
	SharedSpaceIDs     []string
	DirectoryName      string
}

type OrganizeCourseVisibilityInput struct {
	VisibilityScope string   `json:"visibility_scope"`
	SharedSpaceIDs  []string `json:"shared_space_ids"`
}

type OrganizeCourseDiscoveryInput struct {
	Featured      bool `json:"featured"`
	Recommendable bool `json:"recommendable"`
	SortOrder     int  `json:"sort_order"`
}

// OrganizeCourseUploadFile is one file picked from the uploaded folder.
// RelativePath keeps the folder-relative location the browser reported, which
// is what filters out __MACOSX / dotfile noise when the folder was zipped on a
// Mac. It is optional; FileName is always the base name.
type OrganizeCourseUploadFile struct {
	FileName     string
	RelativePath string
	MimeType     string
	Size         int64
	Data         []byte
	Open         func() (io.ReadCloser, error) `json:"-"`
}

// OrganizeCourseUploadResult reports what the folder upload produced, including
// per-file failures. A folder often contains a stray .DS_Store or an oversized
// video; those must not abort the whole course, so they are collected here and
// surfaced to the administrator.
type OrganizeCourseUploadResult struct {
	Course   *OrganizeCourse         `json:"course"`
	Skipped  []OrganizeCourseSkipped `json:"skipped,omitempty"`
	Warnings []string                `json:"warnings,omitempty"`
}

// OrganizeCourseSkipped records a file that was not turned into a lesson.
type OrganizeCourseSkipped struct {
	FileName string `json:"file_name"`
	Reason   string `json:"reason"`
}

// NewOrganizeCourseID mints a course ID. It is exported because the folder
// upload path has to know the course ID *before* the course row is written: the
// lessons' underlying organize_outputs rows reference it through series_id.
func NewOrganizeCourseID() string {
	return newOrganizeCourseID(organizeCourseIDPrefix)
}

func newOrganizeCourseID(prefix string) string {
	// 4 + 32 = 36 characters, which is exactly the column width.
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func nonNegativeCount(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func IsValidOrganizeCourseSource(source string) bool {
	switch strings.TrimSpace(source) {
	case OrganizeCourseSourceOfficial, OrganizeCourseSourceCreator:
		return true
	default:
		return false
	}
}

func IsValidOrganizeCourseVisibilityScope(scope string) bool {
	switch strings.TrimSpace(scope) {
	case OrganizeCourseVisibilitySystem,
		OrganizeCourseVisibilitySharedSpace,
		OrganizeCourseVisibilityPrivate:
		return true
	default:
		return false
	}
}

func IsValidOrganizeCourseLessonType(lessonType string) bool {
	switch strings.TrimSpace(lessonType) {
	case OrganizeCourseLessonTypeVideo, OrganizeCourseLessonTypeAudio, OrganizeCourseLessonTypeArticle:
		return true
	default:
		return false
	}
}
