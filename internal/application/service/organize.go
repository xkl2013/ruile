package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var (
	ErrOrganizeInvalidScope      = errors.New("invalid organize scope")
	ErrOrganizeNotFound          = errors.New("organize item not found")
	ErrOrganizeTitleRequired     = errors.New("title is required")
	ErrOrganizeInvalidMemoryKind = errors.New("invalid memory kind")
	ErrOrganizeInvalidStatus     = errors.New("invalid output status")
	ErrOrganizeInvalidStage      = errors.New("invalid sprout stage")
	ErrOrganizeInvalidMemoryRefs = errors.New("memory_ids contains unknown memories")
)

const (
	organizeMaxTitleLength  = 512
	organizeMaxShortText    = 255
	organizeDefaultPage     = 1
	organizeDefaultPageSize = 20
	organizeMaxPageSize     = 100
)

type organizeService struct {
	repo interfaces.OrganizeRepository
}

func NewOrganizeService(repo interfaces.OrganizeRepository) interfaces.OrganizeService {
	return &organizeService{repo: repo}
}

func (s *organizeService) CreateMemory(
	ctx context.Context, tenantID uint64, userID string, input types.OrganizeMemoryInput,
) (*types.OrganizeMemory, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	kind, title, err := normalizeMemoryBasics(input.Kind, types.OrganizeMemoryKindNote, input.Title)
	if err != nil {
		return nil, err
	}
	occurredAt := time.Now().UTC()
	if input.OccurredAt != nil && !input.OccurredAt.IsZero() {
		occurredAt = input.OccurredAt.UTC()
	}
	memory := &types.OrganizeMemory{
		TenantID:        tenantID,
		UserID:          userID,
		Kind:            kind,
		Title:           title,
		Content:         strings.TrimSpace(input.Content),
		Source:          trimMax(input.Source, organizeMaxShortText),
		OccurredAt:      occurredAt,
		DurationSeconds: nonNegative(input.DurationSeconds),
		Metadata:        normalizeJSONMap(input.Metadata),
	}
	if err := s.repo.CreateMemory(ctx, memory); err != nil {
		return nil, err
	}
	return memory, nil
}

func (s *organizeService) GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	memory, err := s.repo.GetMemory(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if memory == nil {
		return nil, ErrOrganizeNotFound
	}
	return memory, nil
}

func (s *organizeService) UpdateMemory(
	ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeMemoryInput,
) (*types.OrganizeMemory, error) {
	memory, err := s.GetMemory(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	kind, title, err := normalizeMemoryBasics(input.Kind, memory.Kind, input.Title)
	if err != nil {
		return nil, err
	}
	memory.Kind = kind
	memory.Title = title
	memory.Content = strings.TrimSpace(input.Content)
	memory.Source = trimMax(input.Source, organizeMaxShortText)
	if input.OccurredAt != nil && !input.OccurredAt.IsZero() {
		memory.OccurredAt = input.OccurredAt.UTC()
	}
	memory.DurationSeconds = nonNegative(input.DurationSeconds)
	memory.Metadata = normalizeJSONMap(input.Metadata)
	memory.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMemory(ctx, memory); err != nil {
		return nil, err
	}
	return s.GetMemory(ctx, tenantID, userID, id)
}

func (s *organizeService) DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error {
	if _, err := s.GetMemory(ctx, tenantID, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteMemory(ctx, tenantID, userID, strings.TrimSpace(id))
}

func (s *organizeService) ListMemories(
	ctx context.Context, query types.OrganizeListQuery,
) ([]*types.OrganizeMemory, int64, error) {
	if err := validateOrganizeScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(query.Kind) != "" {
		kind := normalizeMemoryKind(query.Kind)
		if !types.IsValidOrganizeMemoryKind(kind) {
			return nil, 0, ErrOrganizeInvalidMemoryKind
		}
		query.Kind = kind
	}
	query = normalizeOrganizePagination(query)
	return s.repo.ListMemories(ctx, query)
}

func (s *organizeService) CreateOutput(
	ctx context.Context, tenantID uint64, userID string, input types.OrganizeOutputInput,
) (*types.OrganizeOutput, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	output, memoryIDs, err := s.buildOutput(ctx, tenantID, userID, "", input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateOutput(ctx, output, memoryIDs); err != nil {
		return nil, err
	}
	return s.repo.GetOutput(ctx, tenantID, userID, output.ID)
}

func (s *organizeService) GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	output, err := s.repo.GetOutput(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, ErrOrganizeNotFound
	}
	return output, nil
}

func (s *organizeService) UpdateOutput(
	ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeOutputInput,
) (*types.OrganizeOutput, error) {
	if _, err := s.GetOutput(ctx, tenantID, userID, id); err != nil {
		return nil, err
	}
	output, memoryIDs, err := s.buildOutput(ctx, tenantID, userID, strings.TrimSpace(id), input)
	if err != nil {
		return nil, err
	}
	output.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateOutput(ctx, output, memoryIDs); err != nil {
		return nil, err
	}
	return s.GetOutput(ctx, tenantID, userID, id)
}

func (s *organizeService) DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error {
	if _, err := s.GetOutput(ctx, tenantID, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteOutput(ctx, tenantID, userID, strings.TrimSpace(id))
}

func (s *organizeService) ListOutputs(
	ctx context.Context, query types.OrganizeListQuery,
) ([]*types.OrganizeOutput, int64, error) {
	if err := validateOrganizeScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(query.Status) != "" && !types.IsValidOrganizeOutputStatus(strings.TrimSpace(query.Status)) {
		return nil, 0, ErrOrganizeInvalidStatus
	}
	query.Status = strings.TrimSpace(query.Status)
	query = normalizeOrganizePagination(query)
	return s.repo.ListOutputs(ctx, query)
}

func (s *organizeService) CreateSproutReport(
	ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutReportInput,
) (*types.OrganizeSproutReport, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	report, memoryIDs, err := s.buildSproutReport(ctx, tenantID, userID, "", input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSproutReport(ctx, report, memoryIDs); err != nil {
		return nil, err
	}
	return s.repo.GetSproutReport(ctx, tenantID, userID, report.ID)
}

func (s *organizeService) GetSproutReport(
	ctx context.Context, tenantID uint64, userID, id string,
) (*types.OrganizeSproutReport, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	report, err := s.repo.GetSproutReport(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, ErrOrganizeNotFound
	}
	return report, nil
}

func (s *organizeService) UpdateSproutReport(
	ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeSproutReportInput,
) (*types.OrganizeSproutReport, error) {
	if _, err := s.GetSproutReport(ctx, tenantID, userID, id); err != nil {
		return nil, err
	}
	report, memoryIDs, err := s.buildSproutReport(ctx, tenantID, userID, strings.TrimSpace(id), input)
	if err != nil {
		return nil, err
	}
	report.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateSproutReport(ctx, report, memoryIDs); err != nil {
		return nil, err
	}
	return s.GetSproutReport(ctx, tenantID, userID, id)
}

func (s *organizeService) DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error {
	if _, err := s.GetSproutReport(ctx, tenantID, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteSproutReport(ctx, tenantID, userID, strings.TrimSpace(id))
}

func (s *organizeService) ListSproutReports(
	ctx context.Context, query types.OrganizeListQuery,
) ([]*types.OrganizeSproutReport, int64, error) {
	if err := validateOrganizeScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query.Stage = strings.TrimSpace(query.Stage)
	if query.Stage != "" && !types.IsValidOrganizeSproutStage(query.Stage) {
		return nil, 0, ErrOrganizeInvalidStage
	}
	query = normalizeOrganizePagination(query)
	return s.repo.ListSproutReports(ctx, query)
}

func (s *organizeService) GetOverview(ctx context.Context, tenantID uint64, userID string) (*types.OrganizeOverview, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	memoryCounts, err := s.repo.CountMemoriesByKind(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	outputStats, err := s.repo.CountOutputsByStatus(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	sproutStats, err := s.repo.CountSproutReportsByStage(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	memoryTotal := sumCounts(memoryCounts)
	outputTotal := sumCounts(outputStats)
	sproutTotal := sumCounts(sproutStats)
	return &types.OrganizeOverview{
		Tabs: []types.OrganizeTabSummary{
			{Key: types.OrganizeTabMemory, Count: memoryTotal},
			{Key: types.OrganizeTabOutput, Count: outputTotal},
			{Key: types.OrganizeTabSprout, Count: sproutTotal},
		},
		MemoryKinds: []types.OrganizeMemoryAssetSummary{
			{Kind: types.OrganizeMemoryKindNote, Count: memoryCounts[types.OrganizeMemoryKindNote]},
			{Kind: types.OrganizeMemoryKindRecord, Count: memoryCounts[types.OrganizeMemoryKindRecord]},
			{Kind: types.OrganizeMemoryKindAudio, Count: memoryCounts[types.OrganizeMemoryKindAudio]},
			{Kind: types.OrganizeMemoryKindAudioCard, Count: memoryCounts[types.OrganizeMemoryKindAudioCard]},
		},
		OutputStats: outputStats,
		SproutStats: sproutStats,
	}, nil
}

func (s *organizeService) buildOutput(
	ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeOutputInput,
) (*types.OrganizeOutput, []string, error) {
	title, err := normalizeTitle(input.Title)
	if err != nil {
		return nil, nil, err
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = types.OrganizeOutputStatusDraft
	}
	if !types.IsValidOrganizeOutputStatus(status) {
		return nil, nil, ErrOrganizeInvalidStatus
	}
	memoryIDs, err := s.validateMemoryIDs(ctx, tenantID, userID, input.MemoryIDs)
	if err != nil {
		return nil, nil, err
	}
	return &types.OrganizeOutput{
		ID:            id,
		TenantID:      tenantID,
		UserID:        userID,
		Title:         title,
		OutputType:    trimMax(input.OutputType, 64),
		Content:       strings.TrimSpace(input.Content),
		SourceSummary: trimMax(input.SourceSummary, organizeMaxShortText),
		Status:        status,
		Icon:          trimMax(input.Icon, 64),
		Metadata:      normalizeJSONMap(input.Metadata),
	}, memoryIDs, nil
}

func (s *organizeService) buildSproutReport(
	ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeSproutReportInput,
) (*types.OrganizeSproutReport, []string, error) {
	title, err := normalizeTitle(input.Title)
	if err != nil {
		return nil, nil, err
	}
	stage := trimMax(input.Stage, 64)
	if stage == "" {
		stage = types.OrganizeSproutStageOrganizing
	}
	if !types.IsValidOrganizeSproutStage(stage) {
		return nil, nil, ErrOrganizeInvalidStage
	}
	memoryIDs, err := s.validateMemoryIDs(ctx, tenantID, userID, input.MemoryIDs)
	if err != nil {
		return nil, nil, err
	}
	return &types.OrganizeSproutReport{
		ID:         id,
		TenantID:   tenantID,
		UserID:     userID,
		Title:      title,
		Summary:    strings.TrimSpace(input.Summary),
		Stage:      stage,
		OutputHint: trimMax(input.OutputHint, organizeMaxShortText),
		Chips:      cleanStringArray(input.Chips, 20, 64),
		Metadata:   normalizeJSONMap(input.Metadata),
	}, memoryIDs, nil
}

func (s *organizeService) validateMemoryIDs(ctx context.Context, tenantID uint64, userID string, ids []string) ([]string, error) {
	cleaned := organizeUniqueNonEmptyStrings(ids)
	if len(cleaned) == 0 {
		return nil, nil
	}
	count, err := s.repo.CountMemoriesByIDs(ctx, tenantID, userID, cleaned)
	if err != nil {
		return nil, err
	}
	if count != int64(len(cleaned)) {
		return nil, ErrOrganizeInvalidMemoryRefs
	}
	return cleaned, nil
}

func validateOrganizeScope(tenantID uint64, userID string) error {
	if tenantID == 0 || strings.TrimSpace(userID) == "" {
		return ErrOrganizeInvalidScope
	}
	return nil
}

func normalizeMemoryBasics(rawKind, fallbackKind, rawTitle string) (string, string, error) {
	kind := normalizeMemoryKind(rawKind)
	if kind == "" {
		kind = fallbackKind
	}
	if !types.IsValidOrganizeMemoryKind(kind) {
		return "", "", ErrOrganizeInvalidMemoryKind
	}
	title, err := normalizeTitle(rawTitle)
	if err != nil {
		return "", "", err
	}
	return kind, title, nil
}

func normalizeMemoryKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "audio-card" {
		return types.OrganizeMemoryKindAudioCard
	}
	return kind
}

func normalizeTitle(title string) (string, error) {
	title = trimMax(title, organizeMaxTitleLength)
	if title == "" {
		return "", ErrOrganizeTitleRequired
	}
	return title, nil
}

func trimMax(s string, max int) string {
	s = strings.TrimSpace(s)
	if max > 0 && len([]rune(s)) > max {
		runes := []rune(s)
		return string(runes[:max])
	}
	return s
}

func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func normalizeJSONMap(m types.JSONMap) types.JSONMap {
	if m == nil {
		return types.JSONMap{}
	}
	return m
}

func normalizeOrganizePagination(query types.OrganizeListQuery) types.OrganizeListQuery {
	if query.Page <= 0 {
		query.Page = organizeDefaultPage
	}
	if query.PageSize <= 0 {
		query.PageSize = organizeDefaultPageSize
	}
	if query.PageSize > organizeMaxPageSize {
		query.PageSize = organizeMaxPageSize
	}
	return query
}

func organizeUniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cleanStringArray(values []string, maxItems, maxLen int) types.StringArray {
	values = organizeUniqueNonEmptyStrings(values)
	if maxItems > 0 && len(values) > maxItems {
		values = values[:maxItems]
	}
	out := make(types.StringArray, 0, len(values))
	for _, value := range values {
		out = append(out, trimMax(value, maxLen))
	}
	return out
}

func sumCounts(values map[string]int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}
