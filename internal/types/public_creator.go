package types

import "time"

// PublicCreatorQuery controls the SystemAdmin creator-management list.
type PublicCreatorQuery struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

// PublicCreatorSummary is the compact creator row shown in Admin.
type PublicCreatorSummary struct {
	ID                          string `json:"id"`
	DisplayName                 string `json:"display_name"`
	Username                    string `json:"username"`
	Email                       string `json:"email"`
	Avatar                      string `json:"avatar,omitempty"`
	KnowledgeBaseCount          int    `json:"knowledge_base_count"`
	ContentCount                int    `json:"content_count"`
	PublishedKnowledgeBaseCount int    `json:"published_knowledge_base_count"`
	PublishedContentCount       int    `json:"published_content_count"`
	PendingKnowledgeBaseCount   int    `json:"pending_knowledge_base_count"`
	PendingContentCount         int    `json:"pending_content_count"`
}

// PublicCreatorKnowledgeBase is a creator-owned KB plus its platform
// publication projection. A missing publication means the KB has not yet
// entered the public catalogue.
type PublicCreatorKnowledgeBase struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Type               string    `json:"type"`
	Icon               string    `json:"icon,omitempty"`
	CreatorID          string    `json:"creator_id"`
	TenantID           uint64    `json:"tenant_id"`
	PublicationID      string    `json:"publication_id,omitempty"`
	PublicationStatus  string    `json:"publication_status"`
	KnowledgeCount     int64     `json:"knowledge_count"`
	ChunkCount         int64     `json:"chunk_count"`
	IsProcessing       bool      `json:"is_processing"`
	ProcessingCount    int64     `json:"processing_count"`
	CanPublish         bool      `json:"can_publish"`
	PublishBlockReason string    `json:"publish_block_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// PublicCreatorContent is the public-facing moderation projection of an
// organize output. It deliberately omits memory links and internal task data.
type PublicCreatorContent struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Content           string     `json:"content,omitempty"`
	OutputType        string     `json:"output_type"`
	SourceSummary     string     `json:"source_summary,omitempty"`
	PublicContentType string     `json:"public_content_type"`
	PublicStatus      string     `json:"public_status"`
	SeriesID          string     `json:"series_id,omitempty"`
	SeriesTitle       string     `json:"series_title,omitempty"`
	SeriesOrder       int        `json:"series_order,omitempty"`
	ReviewNote        string     `json:"review_note,omitempty"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	Metadata          JSONMap    `json:"metadata,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// PublicCreatorDetail is the creator drawer payload in Admin.
type PublicCreatorDetail struct {
	PublicCreatorSummary
	KnowledgeBases []*PublicCreatorKnowledgeBase `json:"knowledge_bases"`
	Contents       []*PublicCreatorContent       `json:"contents"`
}

type PublicCreatorPublishFailure struct {
	AssetType string `json:"asset_type"`
	AssetID   string `json:"asset_id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
}

// PublicCreatorPublishResult reports partial success so one invalid KB does
// not prevent the rest of a creator's catalogue from being published.
type PublicCreatorPublishResult struct {
	CreatorID               string                         `json:"creator_id"`
	KnowledgeBasesPublished int                            `json:"knowledge_bases_published"`
	ContentsPublished       int                            `json:"contents_published"`
	KnowledgeBasesOfflined  int                            `json:"knowledge_bases_offlined"`
	ContentsOfflined        int                            `json:"contents_offlined"`
	KnowledgeBasesSkipped   int                            `json:"knowledge_bases_skipped"`
	ContentsSkipped         int                            `json:"contents_skipped"`
	Failures                []*PublicCreatorPublishFailure `json:"failures"`
}
