package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var (
	ErrPublicCreatorNotFound  = errors.New("public creator not found")
	ErrPublicCreatorBadStatus = errors.New("invalid public creator status")
)

type publicCreatorService struct {
	userSvc     interfaces.UserService
	kbRepo      interfaces.KnowledgeBaseRepository
	kbSvc       interfaces.KnowledgeBaseService
	publicKBSvc interfaces.PublicKnowledgeBaseService
	organizeSvc interfaces.OrganizeService
}

func NewPublicCreatorService(
	userSvc interfaces.UserService,
	kbRepo interfaces.KnowledgeBaseRepository,
	kbSvc interfaces.KnowledgeBaseService,
	publicKBSvc interfaces.PublicKnowledgeBaseService,
	organizeSvc interfaces.OrganizeService,
) interfaces.PublicCreatorService {
	return &publicCreatorService{
		userSvc:     userSvc,
		kbRepo:      kbRepo,
		kbSvc:       kbSvc,
		publicKBSvc: publicKBSvc,
		organizeSvc: organizeSvc,
	}
}

type publicCreatorAggregate struct {
	user     *types.User
	kbs      []*types.KnowledgeBase
	contents []*types.OrganizeOutput
}

func (s *publicCreatorService) collect(ctx context.Context) (map[string]*publicCreatorAggregate, error) {
	users, err := s.userSvc.ListUsers(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	aggregates := make(map[string]*publicCreatorAggregate)
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.ID) == "" || !user.IsCreator {
			continue
		}
		aggregates[user.ID] = &publicCreatorAggregate{user: user}
	}

	kbs, err := s.kbRepo.ListKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	for _, kb := range kbs {
		if kb == nil || kb.IsTemporary || strings.TrimSpace(kb.CreatorID) == "" {
			continue
		}
		aggregate := aggregates[kb.CreatorID]
		if aggregate == nil {
			continue
		}
		aggregate.kbs = append(aggregate.kbs, kb)
	}

	contents, err := s.listAllPublicContents(ctx)
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		if content == nil || strings.TrimSpace(content.UserID) == "" {
			continue
		}
		aggregate := aggregates[content.UserID]
		if aggregate == nil {
			continue
		}
		aggregate.contents = append(aggregate.contents, content)
	}
	return aggregates, nil
}

func (s *publicCreatorService) listAllPublicContents(ctx context.Context) ([]*types.OrganizeOutput, error) {
	const pageSize = 100
	all := make([]*types.OrganizeOutput, 0)
	for page := 1; ; page++ {
		rows, total, err := s.organizeSvc.ListAdminPublicContents(ctx, types.OrganizePublicContentQuery{
			Page:     page,
			PageSize: pageSize,
			// Lesson bodies are parts of a course, not works the creator
			// published; counting them would inflate every course author's tally
			// by the length of their course.
			ExcludeCourseLessons: true,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
		if len(rows) == 0 || len(all) >= int(total) {
			return all, nil
		}
	}
}

func (s *publicCreatorService) listPublications(
	ctx context.Context,
) (map[string]*types.PublicKnowledgeBasePublication, error) {
	rows, err := s.publicKBSvc.ListAdminPublications(ctx, nil, "")
	if err != nil {
		return nil, err
	}
	byKBID := make(map[string]*types.PublicKnowledgeBasePublication, len(rows))
	for _, row := range rows {
		if row == nil || row.KnowledgeBase == nil {
			continue
		}
		byKBID[row.KnowledgeBase.ID] = row
	}
	return byKBID, nil
}

func (s *publicCreatorService) ListCreators(
	ctx context.Context,
	query types.PublicCreatorQuery,
) ([]*types.PublicCreatorSummary, int64, error) {
	aggregates, err := s.collect(ctx)
	if err != nil {
		return nil, 0, err
	}
	status := strings.TrimSpace(query.Status)
	if status != "" && status != "all" && status != "pending" && status != "published" {
		return nil, 0, ErrPublicCreatorBadStatus
	}
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	publications, err := s.listPublications(ctx)
	if err != nil {
		return nil, 0, err
	}

	summaries := make([]*types.PublicCreatorSummary, 0, len(aggregates))
	for creatorID, aggregate := range aggregates {
		if aggregate == nil || (len(aggregate.kbs) == 0 && len(aggregate.contents) == 0) {
			continue
		}
		summary := summarizePublicCreatorWithPublications(creatorID, aggregate, publications)
		if !matchesPublicCreatorKeyword(summary, keyword) {
			continue
		}
		if status == "pending" && summary.PendingKnowledgeBaseCount+summary.PendingContentCount == 0 {
			continue
		}
		if status == "published" && summary.PublishedKnowledgeBaseCount+summary.PublishedContentCount == 0 {
			continue
		}
		summaries = append(summaries, summary)
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		left := strings.ToLower(summaries[i].DisplayName)
		right := strings.ToLower(summaries[j].DisplayName)
		if left == right {
			return summaries[i].ID < summaries[j].ID
		}
		return left < right
	})

	total := int64(len(summaries))
	page := query.Page
	pageSize := query.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(summaries) {
		return []*types.PublicCreatorSummary{}, total, nil
	}
	end := start + pageSize
	if end > len(summaries) {
		end = len(summaries)
	}
	return summaries[start:end], total, nil
}

func (s *publicCreatorService) GetCreator(
	ctx context.Context,
	creatorID string,
) (*types.PublicCreatorDetail, error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return nil, ErrPublicCreatorNotFound
	}
	aggregates, err := s.collect(ctx)
	if err != nil {
		return nil, err
	}
	aggregate := aggregates[creatorID]
	if aggregate == nil || (len(aggregate.kbs) == 0 && len(aggregate.contents) == 0) {
		return nil, ErrPublicCreatorNotFound
	}
	publications, err := s.listPublications(ctx)
	if err != nil {
		return nil, err
	}
	return s.detailFromAggregate(ctx, creatorID, aggregate, publications), nil
}

func (s *publicCreatorService) detailFromAggregate(
	ctx context.Context,
	creatorID string,
	aggregate *publicCreatorAggregate,
	publications map[string]*types.PublicKnowledgeBasePublication,
) *types.PublicCreatorDetail {
	summary := summarizePublicCreatorWithPublications(creatorID, aggregate, publications)
	detail := &types.PublicCreatorDetail{
		PublicCreatorSummary: *summary,
		KnowledgeBases:       make([]*types.PublicCreatorKnowledgeBase, 0, len(aggregate.kbs)),
		Contents:             make([]*types.PublicCreatorContent, 0, len(aggregate.contents)),
	}
	for _, kb := range aggregate.kbs {
		if kb == nil {
			continue
		}
		if s.kbSvc != nil {
			_ = s.kbSvc.FillKnowledgeBaseCounts(ctx, kb)
		}
		item := &types.PublicCreatorKnowledgeBase{
			ID:                kb.ID,
			Name:              kb.Name,
			Description:       kb.Description,
			Type:              kb.Type,
			Icon:              kb.Icon,
			CreatorID:         kb.CreatorID,
			TenantID:          kb.TenantID,
			PublicationStatus: "unpublished",
			KnowledgeCount:    kb.KnowledgeCount,
			ChunkCount:        kb.ChunkCount,
			IsProcessing:      kb.IsProcessing,
			ProcessingCount:   kb.ProcessingCount,
			CreatedAt:         kb.CreatedAt,
			UpdatedAt:         kb.UpdatedAt,
		}
		var publication *types.PublicKnowledgeBasePublication
		if current := publications[kb.ID]; current != nil {
			publication = current
			item.PublicationID = publication.ID
			item.PublicationStatus = string(publication.Status)
		}
		item.CanPublish, item.PublishBlockReason = publicKnowledgeBasePublishReadiness(item, publication)
		detail.KnowledgeBases = append(detail.KnowledgeBases, item)
	}
	for _, content := range aggregate.contents {
		if content == nil {
			continue
		}
		detail.Contents = append(detail.Contents, &types.PublicCreatorContent{
			ID:                content.ID,
			Title:             content.Title,
			Content:           content.Content,
			OutputType:        content.OutputType,
			SourceSummary:     content.SourceSummary,
			PublicContentType: content.PublicContentType,
			PublicStatus:      content.PublicStatus,
			SeriesID:          content.SeriesID,
			SeriesTitle:       content.SeriesTitle,
			SeriesOrder:       content.SeriesOrder,
			ReviewNote:        content.ReviewNote,
			PublishedAt:       content.PublishedAt,
			Metadata:          content.Metadata,
			CreatedAt:         content.CreatedAt,
			UpdatedAt:         content.UpdatedAt,
		})
	}
	sort.SliceStable(detail.KnowledgeBases, func(i, j int) bool {
		return detail.KnowledgeBases[i].UpdatedAt.After(detail.KnowledgeBases[j].UpdatedAt)
	})
	sort.SliceStable(detail.Contents, func(i, j int) bool {
		return detail.Contents[i].UpdatedAt.After(detail.Contents[j].UpdatedAt)
	})
	return detail
}

func publicKnowledgeBasePublishReadiness(
	kb *types.PublicCreatorKnowledgeBase,
	publication *types.PublicKnowledgeBasePublication,
) (bool, string) {
	if kb == nil {
		return false, "knowledge_base_missing"
	}
	title := strings.TrimSpace(kb.Name)
	if publication != nil {
		title = strings.TrimSpace(publication.Title)
	}
	if title == "" {
		return false, "title_required"
	}
	contentCount := kb.KnowledgeCount
	if kb.Type == types.KnowledgeBaseTypeFAQ {
		contentCount = kb.ChunkCount
	}
	if contentCount == 0 {
		return false, "content_required"
	}
	if kb.IsProcessing || kb.ProcessingCount > 0 {
		return false, "processing"
	}
	return true, ""
}

func (s *publicCreatorService) PublishCreator(
	ctx context.Context,
	creatorID string,
) (*types.PublicCreatorPublishResult, error) {
	return s.moderateCreator(ctx, creatorID, true)
}

func (s *publicCreatorService) PublishKnowledgeBase(
	ctx context.Context,
	creatorID, knowledgeBaseID string,
) (*types.PublicCreatorPublishResult, error) {
	detail, err := s.GetCreator(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	for _, kb := range detail.KnowledgeBases {
		if kb == nil || kb.ID != strings.TrimSpace(knowledgeBaseID) {
			continue
		}
		result := &types.PublicCreatorPublishResult{
			CreatorID: creatorID,
			Failures:  make([]*types.PublicCreatorPublishFailure, 0),
		}
		s.publishKnowledgeBase(ctx, kb, result)
		return result, nil
	}
	return nil, ErrPublicCreatorNotFound
}

func (s *publicCreatorService) OfflineCreator(
	ctx context.Context,
	creatorID string,
) (*types.PublicCreatorPublishResult, error) {
	return s.moderateCreator(ctx, creatorID, false)
}

func (s *publicCreatorService) moderateCreator(
	ctx context.Context,
	creatorID string,
	publish bool,
) (*types.PublicCreatorPublishResult, error) {
	detail, err := s.GetCreator(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	result := &types.PublicCreatorPublishResult{
		CreatorID: creatorID,
		Failures:  make([]*types.PublicCreatorPublishFailure, 0),
	}
	for _, kb := range detail.KnowledgeBases {
		if kb == nil {
			continue
		}
		if publish {
			s.publishKnowledgeBase(ctx, kb, result)
			continue
		}
		if kb.PublicationStatus != string(types.PublicKnowledgeBasePublicationPublished) {
			result.KnowledgeBasesSkipped++
			continue
		}
		if _, offlineErr := s.publicKBSvc.OfflinePublication(ctx, kb.PublicationID); offlineErr != nil {
			addCreatorFailure(result, "knowledge_base", kb.ID, kb.Name, offlineErr)
			continue
		}
		result.KnowledgeBasesOfflined++
	}

	for _, content := range detail.Contents {
		if content == nil {
			continue
		}
		if publish {
			if content.PublicStatus == types.OrganizePublicContentStatusPublished {
				result.ContentsSkipped++
				continue
			}
			if _, publishErr := s.organizeSvc.ModeratePublicContent(
				ctx,
				content.ID,
				types.OrganizePublicContentStatusPublished,
				"",
			); publishErr != nil {
				addCreatorFailure(result, "content", content.ID, content.Title, publishErr)
				continue
			}
			result.ContentsPublished++
			continue
		}
		if content.PublicStatus != types.OrganizePublicContentStatusPublished {
			result.ContentsSkipped++
			continue
		}
		if _, offlineErr := s.organizeSvc.ModeratePublicContent(
			ctx,
			content.ID,
			types.OrganizePublicContentStatusOffline,
			"",
		); offlineErr != nil {
			addCreatorFailure(result, "content", content.ID, content.Title, offlineErr)
			continue
		}
		result.ContentsOfflined++
	}
	return result, nil
}

func (s *publicCreatorService) publishKnowledgeBase(
	ctx context.Context,
	kb *types.PublicCreatorKnowledgeBase,
	result *types.PublicCreatorPublishResult,
) {
	if kb.PublicationStatus == string(types.PublicKnowledgeBasePublicationPublished) {
		result.KnowledgeBasesSkipped++
		return
	}
	publicationID := kb.PublicationID
	if publicationID == "" {
		publication, createErr := s.publicKBSvc.CreatePublication(
			ctx,
			kb.ID,
			kb.Name,
			kb.Description,
			"",
		)
		if createErr != nil && publication == nil {
			addCreatorFailure(result, "knowledge_base", kb.ID, kb.Name, createErr)
			return
		}
		if publication != nil {
			publicationID = publication.ID
		}
	}
	if publicationID == "" {
		addCreatorFailure(result, "knowledge_base", kb.ID, kb.Name, errors.New("publication record is missing"))
		return
	}
	if _, publishErr := s.publicKBSvc.PublishPublication(ctx, publicationID); publishErr != nil {
		addCreatorFailure(result, "knowledge_base", kb.ID, kb.Name, publishErr)
		return
	}
	result.KnowledgeBasesPublished++
}

func addCreatorFailure(
	result *types.PublicCreatorPublishResult,
	assetType, assetID, title string,
	err error,
) {
	result.Failures = append(result.Failures, &types.PublicCreatorPublishFailure{
		AssetType: assetType,
		AssetID:   assetID,
		Title:     title,
		Message:   err.Error(),
	})
}

func summarizePublicCreatorWithPublications(
	creatorID string,
	aggregate *publicCreatorAggregate,
	publications map[string]*types.PublicKnowledgeBasePublication,
) *types.PublicCreatorSummary {
	summary := &types.PublicCreatorSummary{ID: creatorID}
	if aggregate != nil && aggregate.user != nil {
		summary.Username = aggregate.user.Username
		summary.Email = aggregate.user.Email
		summary.Avatar = aggregate.user.Avatar
	}
	summary.DisplayName = strings.TrimSpace(summary.Username)
	if summary.DisplayName == "" {
		summary.DisplayName = strings.TrimSpace(summary.Email)
	}
	if summary.DisplayName == "" {
		summary.DisplayName = creatorID
	}
	if aggregate == nil {
		return summary
	}
	summary.KnowledgeBaseCount = len(aggregate.kbs)
	summary.ContentCount = len(aggregate.contents)
	for _, kb := range aggregate.kbs {
		if kb == nil {
			continue
		}
		status := "unpublished"
		if publications != nil {
			if publication := publications[kb.ID]; publication != nil {
				status = string(publication.Status)
			}
		}
		if status == string(types.PublicKnowledgeBasePublicationPublished) {
			summary.PublishedKnowledgeBaseCount++
		} else {
			summary.PendingKnowledgeBaseCount++
		}
	}
	for _, content := range aggregate.contents {
		if content == nil {
			continue
		}
		if content.PublicStatus == types.OrganizePublicContentStatusPublished {
			summary.PublishedContentCount++
		} else {
			summary.PendingContentCount++
		}
	}
	return summary
}

func matchesPublicCreatorKeyword(summary *types.PublicCreatorSummary, keyword string) bool {
	if keyword == "" {
		return true
	}
	for _, value := range []string{
		summary.ID,
		summary.DisplayName,
		summary.Username,
		summary.Email,
	} {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}
