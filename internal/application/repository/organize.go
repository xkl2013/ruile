package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type organizeRepository struct {
	db *gorm.DB
}

func NewOrganizeRepository(db *gorm.DB) interfaces.OrganizeRepository {
	return &organizeRepository{db: db}
}

func (r *organizeRepository) CreateMemory(ctx context.Context, memory *types.OrganizeMemory) error {
	return r.db.WithContext(ctx).Create(memory).Error
}

func (r *organizeRepository) GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error) {
	var memory types.OrganizeMemory
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		First(&memory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return &memory, err
	}
	if err := r.fillMemoryAttachments(ctx, tenantID, userID, []*types.OrganizeMemory{&memory}); err != nil {
		return nil, err
	}
	return &memory, err
}

func (r *organizeRepository) GetTenantMemory(ctx context.Context, tenantID uint64, id string) (*types.OrganizeMemory, error) {
	var memory types.OrganizeMemory
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&memory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return &memory, err
	}
	if err := r.fillMemoryAttachments(ctx, tenantID, "", []*types.OrganizeMemory{&memory}); err != nil {
		return nil, err
	}
	return &memory, err
}

func (r *organizeRepository) UpdateMemory(ctx context.Context, memory *types.OrganizeMemory) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeMemory{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", memory.TenantID, memory.UserID, memory.ID).
		Select("kind", "title", "content", "source", "occurred_at", "duration_seconds", "metadata", "updated_at").
		Updates(memory).Error
}

func (r *organizeRepository) CreateMemoryAttachment(ctx context.Context, attachment *types.OrganizeMemoryAttachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

func (r *organizeRepository) GetMemoryAttachment(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) (*types.OrganizeMemoryAttachment, error) {
	var attachment types.OrganizeMemoryAttachment
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id)
	if strings.TrimSpace(userID) != "" {
		query = query.Where("user_id = ?", userID)
	}
	err := query.First(&attachment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attachment, err
}

func (r *organizeRepository) GetTenantMemoryAttachment(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.OrganizeMemoryAttachment, error) {
	return r.GetMemoryAttachment(ctx, tenantID, "", id)
}

func (r *organizeRepository) UpdateMemoryAttachment(
	ctx context.Context,
	attachment *types.OrganizeMemoryAttachment,
) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeMemoryAttachment{}).
		Where("tenant_id = ? AND id = ?", attachment.TenantID, attachment.ID).
		Select(
			"file_name",
			"mime_type",
			"storage_path",
			"storage_url",
			"size_bytes",
			"sort_order",
			"status",
			"error_stage",
			"error_message",
			"content",
			"transcript",
			"metadata",
			"updated_at",
		).
		Updates(attachment).Error
}

func (r *organizeRepository) ListMemoryAttachments(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID string,
) ([]*types.OrganizeMemoryAttachment, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND memory_id = ?", tenantID, memoryID)
	if strings.TrimSpace(userID) != "" {
		query = query.Where("user_id = ?", userID)
	}
	var attachments []*types.OrganizeMemoryAttachment
	err := query.Order("sort_order ASC").Order("created_at ASC").Find(&attachments).Error
	return attachments, err
}

func (r *organizeRepository) DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND user_id = ? AND memory_id = ?", tenantID, userID, id).
			Delete(&types.OrganizeOutputMemory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND user_id = ? AND memory_id = ?", tenantID, userID, id).
			Delete(&types.OrganizeSproutMemory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND user_id = ? AND memory_id = ?", tenantID, userID, id).
			Delete(&types.OrganizeMemoryAttachment{}).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
			Delete(&types.OrganizeMemory{}).Error
	})
}

func (r *organizeRepository) ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeMemory{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.Kind != "" {
		dbq = dbq.Where("kind = ?", query.Kind)
	}
	if len(query.Kinds) > 0 {
		dbq = dbq.Where("kind IN ?", query.Kinds)
	}
	if len(query.Sources) > 0 {
		dbq = dbq.Where("source IN ?", query.Sources)
	}
	if query.OccurredFrom != nil {
		dbq = dbq.Where("occurred_at >= ?", query.OccurredFrom.UTC())
	}
	if query.OccurredTo != nil {
		dbq = dbq.Where("occurred_at < ?", query.OccurredTo.UTC())
	}
	if query.ReadyOnly {
		pendingAttachmentStatuses := []string{
			types.OrganizeMemoryAttachmentAggregatePending,
			types.OrganizeMemoryAttachmentAggregateProcessing,
			types.OrganizeMemoryAttachmentAggregateFailed,
		}
		pendingTranscriptionStatuses := []string{"pending", "transcribing", "failed"}
		if r.db.Dialector.Name() == "postgres" {
			dbq = dbq.Where(
				"COALESCE(metadata->>'attachment_status', '') NOT IN ? AND "+
					"COALESCE(metadata->>'transcription_status', '') NOT IN ?",
				pendingAttachmentStatuses,
				pendingTranscriptionStatuses,
			)
		} else {
			dbq = dbq.Where(
				"COALESCE(json_extract(metadata, '$.attachment_status'), '') NOT IN ? AND "+
					"COALESCE(json_extract(metadata, '$.transcription_status'), '') NOT IN ?",
				pendingAttachmentStatuses,
				pendingTranscriptionStatuses,
			)
		}
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "content", "source")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var memories []*types.OrganizeMemory
	err := dbq.
		Order("occurred_at DESC").
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&memories).Error
	if err != nil {
		return memories, total, err
	}
	if err := r.fillMemoryAttachments(ctx, query.TenantID, query.UserID, memories); err != nil {
		return nil, 0, err
	}
	return memories, total, err
}

func (r *organizeRepository) CountOutputMemoryOverlap(
	ctx context.Context,
	tenantID uint64,
	userID string,
	ids []string,
) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Model(&types.OrganizeOutputMemory{}).
		Distinct("memory_id").
		Where("tenant_id = ? AND user_id = ? AND memory_id IN ?", tenantID, userID, ids).
		Count(&count).Error
	return count, err
}

func (r *organizeRepository) fillMemoryAttachments(
	ctx context.Context,
	tenantID uint64,
	userID string,
	memories []*types.OrganizeMemory,
) error {
	if len(memories) == 0 {
		return nil
	}
	ids := make([]string, 0, len(memories))
	byID := make(map[string]*types.OrganizeMemory, len(memories))
	for _, memory := range memories {
		if memory == nil || strings.TrimSpace(memory.ID) == "" {
			continue
		}
		ids = append(ids, memory.ID)
		byID[memory.ID] = memory
		memory.Attachments = nil
	}
	if len(ids) == 0 {
		return nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND memory_id IN ?", tenantID, ids)
	if strings.TrimSpace(userID) != "" {
		query = query.Where("user_id = ?", userID)
	}
	var attachments []*types.OrganizeMemoryAttachment
	if err := query.Order("sort_order ASC").Order("created_at ASC").Find(&attachments).Error; err != nil {
		return err
	}
	for _, attachment := range attachments {
		if memory := byID[attachment.MemoryID]; memory != nil {
			memory.Attachments = append(memory.Attachments, attachment)
		}
	}
	return nil
}

func (r *organizeRepository) CountMemoriesByKind(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error) {
	var rows []struct {
		Kind  string
		Count int64
	}
	err := r.db.WithContext(ctx).Model(&types.OrganizeMemory{}).
		Select("kind, COUNT(*) AS count").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Group("kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Kind] = row.Count
	}
	return out, nil
}

func (r *organizeRepository) CountMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&types.OrganizeMemory{}).
		Where("tenant_id = ? AND user_id = ? AND id IN ?", tenantID, userID, ids).
		Count(&count).Error
	return count, err
}

func (r *organizeRepository) CountTenantMemoriesByIDs(ctx context.Context, tenantID uint64, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&types.OrganizeMemory{}).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Count(&count).Error
	return count, err
}

func (r *organizeRepository) CreateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(output).Error; err != nil {
			return err
		}
		return replaceOutputMemoryLinks(tx, output.TenantID, output.UserID, output.ID, memoryIDs)
	})
}

func (r *organizeRepository) GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error) {
	var output types.OrganizeOutput
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		First(&output).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.fillOutputLinks(ctx, tenantID, userID, []*types.OrganizeOutput{&output}); err != nil {
		return nil, err
	}
	return &output, nil
}

func (r *organizeRepository) GetOutputByID(ctx context.Context, id string) (*types.OrganizeOutput, error) {
	var output types.OrganizeOutput
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&output).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.fillOutputLinks(ctx, output.TenantID, output.UserID, []*types.OrganizeOutput{&output}); err != nil {
		return nil, err
	}
	return &output, nil
}

func (r *organizeRepository) UpdateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.OrganizeOutput{}).
			Where("tenant_id = ? AND user_id = ? AND id = ?", output.TenantID, output.UserID, output.ID).
			Select(
				"config_id",
				"job_id",
				"assigned_service_id",
				"assignment_status",
				"assignment_reason",
				"template_key",
				"template_version",
				"title",
				"output_type",
				"content",
				"source_summary",
				"status",
				"public_content_type",
				"public_status",
				"series_id",
				"series_title",
				"series_order",
				"review_note",
				"published_at",
				"published_by",
				"icon",
				"fields",
				"citations",
				"metadata",
				"updated_at",
			).
			Updates(output).Error; err != nil {
			return err
		}
		return replaceOutputMemoryLinks(tx, output.TenantID, output.UserID, output.ID, memoryIDs)
	})
}

func (r *organizeRepository) UpdatePublicContent(ctx context.Context, output *types.OrganizeOutput) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeOutput{}).
		Where("id = ?", output.ID).
		Select(
			"title",
			"content",
			"source_summary",
			"status",
			"public_content_type",
			"public_status",
			"featured",
			"recommendable",
			"sort_order",
			"series_id",
			"series_title",
			"series_order",
			"review_note",
			"published_at",
			"published_by",
			"icon",
			"metadata",
			"updated_at",
		).
		Updates(output).Error
}

func (r *organizeRepository) DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND user_id = ? AND output_id = ?", tenantID, userID, id).
			Delete(&types.OrganizeOutputMemory{}).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
			Delete(&types.OrganizeOutput{}).Error
	})
}

func (r *organizeRepository) ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeOutput{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.Status != "" {
		dbq = dbq.Where("status = ?", query.Status)
	}
	if query.AssignmentStatus != "" {
		dbq = dbq.Where("assignment_status = ?", query.AssignmentStatus)
	}
	if query.ConfigID != "" {
		dbq = dbq.Where("config_id = ?", query.ConfigID)
	}
	if query.TemplateKey != "" {
		dbq = dbq.Where("template_key = ?", query.TemplateKey)
	}
	if query.Scene != "" {
		dbq = applyOrganizeJSONTextFilter(dbq, r.db.Dialector.Name(), "metadata", "scene", query.Scene)
	}
	for key, value := range query.FieldFilters {
		dbq = applyOrganizeJSONTextFilter(dbq, r.db.Dialector.Name(), "fields", key, value)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "content", "output_type", "source_summary")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var outputs []*types.OrganizeOutput
	err := dbq.
		Order("updated_at DESC").
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&outputs).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.fillOutputLinks(ctx, query.TenantID, query.UserID, outputs); err != nil {
		return nil, 0, err
	}
	return outputs, total, nil
}

func (r *organizeRepository) ListOutputFacets(
	ctx context.Context,
	query types.OrganizeListQuery,
) (*types.OrganizeOutputFacets, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeOutput{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.Status != "" {
		dbq = dbq.Where("status = ?", query.Status)
	}
	if query.AssignmentStatus != "" {
		dbq = dbq.Where("assignment_status = ?", query.AssignmentStatus)
	}
	if query.ConfigID != "" {
		dbq = dbq.Where("config_id = ?", query.ConfigID)
	}
	if query.TemplateKey != "" {
		dbq = dbq.Where("template_key = ?", query.TemplateKey)
	}
	if query.Scene != "" {
		dbq = applyOrganizeJSONTextFilter(dbq, r.db.Dialector.Name(), "metadata", "scene", query.Scene)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "content", "output_type", "source_summary")

	var outputs []*types.OrganizeOutput
	if err := dbq.Select("fields").Find(&outputs).Error; err != nil {
		return nil, err
	}
	facetCounts := make(map[string]map[string]int64)
	for _, output := range outputs {
		if output == nil {
			continue
		}
		for key, raw := range output.Fields {
			value := strings.TrimSpace(fmt.Sprint(raw))
			if value == "" || value == "<nil>" {
				continue
			}
			if facetCounts[key] == nil {
				facetCounts[key] = make(map[string]int64)
			}
			facetCounts[key][value]++
		}
	}
	keys := make([]string, 0, len(facetCounts))
	for key := range facetCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	facets := make([]types.OrganizeOutputFacet, 0, len(keys))
	for _, key := range keys {
		values := make([]types.OrganizeFacetValue, 0, len(facetCounts[key]))
		for value, count := range facetCounts[key] {
			values = append(values, types.OrganizeFacetValue{Value: value, Count: count})
		}
		sort.Slice(values, func(i, j int) bool {
			if values[i].Count == values[j].Count {
				return values[i].Value < values[j].Value
			}
			return values[i].Count > values[j].Count
		})
		facets = append(facets, types.OrganizeOutputFacet{
			Key:    key,
			Label:  key,
			Values: values,
		})
	}
	return &types.OrganizeOutputFacets{Fields: facets}, nil
}

func (r *organizeRepository) ListPublicContents(
	ctx context.Context,
	query types.OrganizePublicContentQuery,
) ([]*types.OrganizeOutput, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeOutput{})
	dbq = dbq.Where("public_status <> ?", "")
	if query.UserID != "" {
		dbq = dbq.Where("user_id = ?", query.UserID)
	}
	if query.PublicStatus != "" {
		dbq = dbq.Where("public_status = ?", query.PublicStatus)
	}
	if query.PublicContentType != "" {
		dbq = dbq.Where("public_content_type = ?", query.PublicContentType)
	}
	if query.Featured != nil {
		dbq = dbq.Where("featured = ?", *query.Featured)
	}
	if query.Recommendable != nil {
		dbq = dbq.Where("recommendable = ?", *query.Recommendable)
	}
	if query.ExcludeFeatured {
		dbq = dbq.Where("featured = ?", false)
	}
	// A row whose series_id points at a real course is a lesson body, not a post.
	// Matching on series_id rather than public_content_type keeps the legacy
	// single "学习课程" posts (which carry the type but belong to no course) in
	// the listing.
	if query.ExcludeCourseLessons {
		dbq = dbq.Where("COALESCE(series_id, '') NOT IN (SELECT id FROM organize_courses)")
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "content", "output_type", "source_summary", "series_title")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var outputs []*types.OrganizeOutput
	if err := dbq.
		Order("CASE WHEN sort_order = 0 THEN 1 ELSE 0 END ASC").
		Order("sort_order ASC").
		Order("updated_at DESC").
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&outputs).Error; err != nil {
		return nil, 0, err
	}
	for _, output := range outputs {
		if output == nil {
			continue
		}
		if err := r.fillOutputLinks(ctx, output.TenantID, output.UserID, []*types.OrganizeOutput{output}); err != nil {
			return nil, 0, err
		}
	}
	return outputs, total, nil
}

func (r *organizeRepository) CountOutputsByStatus(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	err := r.db.WithContext(ctx).Model(&types.OrganizeOutput{}).
		Select("status, COUNT(*) AS count").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Count
	}
	return out, nil
}

func (r *organizeRepository) CreateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(report).Error; err != nil {
			return err
		}
		return replaceSproutMemoryLinks(tx, report.TenantID, report.UserID, report.ID, memoryIDs)
	})
}

func (r *organizeRepository) GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error) {
	var report types.OrganizeSproutReport
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		First(&report).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.fillSproutLinks(ctx, tenantID, userID, []*types.OrganizeSproutReport{&report}); err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *organizeRepository) UpdateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.OrganizeSproutReport{}).
			Where("tenant_id = ? AND user_id = ? AND id = ?", report.TenantID, report.UserID, report.ID).
			Select(
				"template_key",
				"template_version",
				"title",
				"summary",
				"stage",
				"output_hint",
				"chips",
				"fields",
				"metadata",
				"updated_at",
			).
			Updates(report).Error; err != nil {
			return err
		}
		return replaceSproutMemoryLinks(tx, report.TenantID, report.UserID, report.ID, memoryIDs)
	})
}

func (r *organizeRepository) DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND user_id = ? AND report_id = ?", tenantID, userID, id).
			Delete(&types.OrganizeSproutMemory{}).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
			Delete(&types.OrganizeSproutReport{}).Error
	})
}

func (r *organizeRepository) ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeSproutReport{}).
		Where("organize_sprout_reports.tenant_id = ? AND organize_sprout_reports.user_id = ?", query.TenantID, query.UserID)
	if query.Stage != "" {
		dbq = dbq.Where("organize_sprout_reports.stage = ?", query.Stage)
	}
	if memoryID := strings.TrimSpace(query.MemoryID); memoryID != "" {
		dbq = dbq.Joins(
			"JOIN organize_sprout_memories ON organize_sprout_memories.report_id = organize_sprout_reports.id AND organize_sprout_memories.tenant_id = organize_sprout_reports.tenant_id AND organize_sprout_memories.user_id = organize_sprout_reports.user_id",
		).Where("organize_sprout_memories.memory_id = ?", memoryID)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "organize_sprout_reports.title", "organize_sprout_reports.summary", "organize_sprout_reports.output_hint")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reports []*types.OrganizeSproutReport
	err := dbq.
		Order("organize_sprout_reports.updated_at DESC").
		Order("organize_sprout_reports.created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&reports).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.fillSproutLinks(ctx, query.TenantID, query.UserID, reports); err != nil {
		return nil, 0, err
	}
	return reports, total, nil
}

func (r *organizeRepository) CountSproutReportsByStage(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error) {
	var rows []struct {
		Stage string
		Count int64
	}
	err := r.db.WithContext(ctx).Model(&types.OrganizeSproutReport{}).
		Select("stage, COUNT(*) AS count").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Group("stage").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Stage] = row.Count
	}
	return out, nil
}

func applyOrganizeKeyword(dbq *gorm.DB, keyword string, columns ...string) *gorm.DB {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" || len(columns) == 0 {
		return dbq
	}
	parts := make([]string, 0, len(columns))
	args := make([]interface{}, 0, len(columns))
	pattern := "%" + keyword + "%"
	for _, col := range columns {
		parts = append(parts, "LOWER("+col+") LIKE ?")
		args = append(args, pattern)
	}
	return dbq.Where(strings.Join(parts, " OR "), args...)
}

func applyOrganizeJSONTextFilter(dbq *gorm.DB, dialect, column, key, value string) *gorm.DB {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return dbq
	}
	if dialect == "postgres" {
		return dbq.Where(column+"->>? = ?", key, value)
	}
	return dbq.Where("json_extract("+column+", ?) = ?", "$."+key, value)
}

func replaceOutputMemoryLinks(tx *gorm.DB, tenantID uint64, userID, outputID string, memoryIDs []string) error {
	if err := tx.Where("tenant_id = ? AND user_id = ? AND output_id = ?", tenantID, userID, outputID).
		Delete(&types.OrganizeOutputMemory{}).Error; err != nil {
		return err
	}
	if len(memoryIDs) == 0 {
		return nil
	}
	links := make([]types.OrganizeOutputMemory, 0, len(memoryIDs))
	for _, memoryID := range memoryIDs {
		links = append(links, types.OrganizeOutputMemory{
			OutputID: outputID,
			MemoryID: memoryID,
			TenantID: tenantID,
			UserID:   userID,
		})
	}
	return tx.Create(&links).Error
}

func replaceSproutMemoryLinks(tx *gorm.DB, tenantID uint64, userID, reportID string, memoryIDs []string) error {
	if err := tx.Where("tenant_id = ? AND user_id = ? AND report_id = ?", tenantID, userID, reportID).
		Delete(&types.OrganizeSproutMemory{}).Error; err != nil {
		return err
	}
	if len(memoryIDs) == 0 {
		return nil
	}
	links := make([]types.OrganizeSproutMemory, 0, len(memoryIDs))
	for _, memoryID := range memoryIDs {
		links = append(links, types.OrganizeSproutMemory{
			ReportID: reportID,
			MemoryID: memoryID,
			TenantID: tenantID,
			UserID:   userID,
		})
	}
	return tx.Create(&links).Error
}

func (r *organizeRepository) fillOutputLinks(ctx context.Context, tenantID uint64, userID string, outputs []*types.OrganizeOutput) error {
	if len(outputs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(outputs))
	byID := make(map[string]*types.OrganizeOutput, len(outputs))
	for _, output := range outputs {
		ids = append(ids, output.ID)
		byID[output.ID] = output
	}
	var links []types.OrganizeOutputMemory
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND output_id IN ?", tenantID, userID, ids).
		Order("created_at ASC").
		Find(&links).Error; err != nil {
		return err
	}
	for _, link := range links {
		if output := byID[link.OutputID]; output != nil {
			output.MemoryIDs = append(output.MemoryIDs, link.MemoryID)
			output.MemoryCount++
		}
	}
	return nil
}

func (r *organizeRepository) fillSproutLinks(ctx context.Context, tenantID uint64, userID string, reports []*types.OrganizeSproutReport) error {
	if len(reports) == 0 {
		return nil
	}
	ids := make([]string, 0, len(reports))
	byID := make(map[string]*types.OrganizeSproutReport, len(reports))
	for _, report := range reports {
		ids = append(ids, report.ID)
		byID[report.ID] = report
	}
	var links []types.OrganizeSproutMemory
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND report_id IN ?", tenantID, userID, ids).
		Order("created_at ASC").
		Find(&links).Error; err != nil {
		return err
	}
	memoryIDs := make([]string, 0, len(links))
	memoryIDSet := make(map[string]struct{}, len(links))
	for _, link := range links {
		if _, ok := memoryIDSet[link.MemoryID]; ok {
			continue
		}
		memoryIDSet[link.MemoryID] = struct{}{}
		memoryIDs = append(memoryIDs, link.MemoryID)
	}
	memoriesByID := make(map[string]types.OrganizeMemory, len(memoryIDs))
	if len(memoryIDs) > 0 {
		var memories []types.OrganizeMemory
		if err := r.db.WithContext(ctx).
			Select("id", "kind", "title", "source").
			Where("tenant_id = ? AND id IN ?", tenantID, memoryIDs).
			Find(&memories).Error; err != nil {
			return err
		}
		for _, memory := range memories {
			memoriesByID[memory.ID] = memory
		}
	}
	for _, link := range links {
		if report := byID[link.ReportID]; report != nil {
			report.MemoryIDs = append(report.MemoryIDs, link.MemoryID)
			report.MemoryCount++
			if memory, ok := memoriesByID[link.MemoryID]; ok {
				report.MemoryRefs = append(report.MemoryRefs, types.OrganizeMemoryReference{
					ID:     memory.ID,
					Kind:   memory.Kind,
					Title:  memory.Title,
					Source: memory.Source,
				})
			}
		}
	}
	return nil
}
