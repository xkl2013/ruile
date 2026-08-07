package repository

import (
	"context"
	"errors"
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
	return &memory, err
}

func (r *organizeRepository) UpdateMemory(ctx context.Context, memory *types.OrganizeMemory) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeMemory{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", memory.TenantID, memory.UserID, memory.ID).
		Select("kind", "title", "content", "source", "occurred_at", "duration_seconds", "metadata", "updated_at").
		Updates(memory).Error
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
	return memories, total, err
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

func (r *organizeRepository) UpdateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.OrganizeOutput{}).
			Where("tenant_id = ? AND user_id = ? AND id = ?", output.TenantID, output.UserID, output.ID).
			Select("title", "output_type", "content", "source_summary", "status", "icon", "metadata", "updated_at").
			Updates(output).Error; err != nil {
			return err
		}
		return replaceOutputMemoryLinks(tx, output.TenantID, output.UserID, output.ID, memoryIDs)
	})
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
			Select("title", "summary", "stage", "output_hint", "chips", "metadata", "updated_at").
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
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.Stage != "" {
		dbq = dbq.Where("stage = ?", query.Stage)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "title", "summary", "output_hint")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reports []*types.OrganizeSproutReport
	err := dbq.
		Order("updated_at DESC").
		Order("created_at DESC").
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
	for _, link := range links {
		if report := byID[link.ReportID]; report != nil {
			report.MemoryIDs = append(report.MemoryIDs, link.MemoryID)
			report.MemoryCount++
		}
	}
	return nil
}
