package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const organizeDiscoverDefaultPageSize = 30

func (s *organizeService) GetDiscover(
	ctx context.Context,
	tenantID uint64,
	userID string,
	query types.OrganizeDiscoverQuery,
) (*types.OrganizeDiscover, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}

	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Tab = normalizeDiscoverTab(strings.TrimSpace(query.Tab))
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = organizeDiscoverDefaultPageSize
	}
	query.FeaturedOffset = max(0, query.FeaturedOffset)

	outputs, err := s.listAllDiscoverOutputs(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	tabs := buildDiscoverTabs(outputs)
	filtered := filterDiscoverOutputs(outputs, query.Tab, query.Keyword)
	sortDiscoverOutputs(filtered)

	totalPages := max(1, (len(filtered)+query.PageSize-1)/query.PageSize)
	if query.Page > totalPages {
		query.Page = totalPages
	}

	featured := pickDiscoverFeaturedOutputs(filtered, query.FeaturedOffset)
	return &types.OrganizeDiscover{
		Tabs:            tabs,
		FeaturedOutputs: featured,
		Items:           paginateDiscoverOutputs(filtered, query.Page, query.PageSize),
		Total:           int64(len(filtered)),
		Page:            query.Page,
		PageSize:        query.PageSize,
		FeaturedOffset:  query.FeaturedOffset,
	}, nil
}

func (s *organizeService) listAllDiscoverOutputs(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeOutput, error) {
	pageSize := organizeMaxPageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	query := types.OrganizeListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PageSize: pageSize,
	}

	outputs := make([]*types.OrganizeOutput, 0, pageSize)
	for {
		items, total, err := s.repo.ListOutputs(ctx, query)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, items...)
		if len(items) == 0 || len(outputs) >= int(total) {
			break
		}
		query.Page++
	}
	return outputs, nil
}

func buildDiscoverTabs(outputs []*types.OrganizeOutput) []types.OrganizeDiscoverTab {
	categoryCounts := make(map[string]int64)

	for _, output := range outputs {
		if output == nil {
			continue
		}
		if category := discoverOutputCategory(output); category != "" {
			categoryCounts[category]++
		}
	}

	tabs := []types.OrganizeDiscoverTab{{Label: "推荐", Value: "recommended", Count: int64(len(outputs))}}
	for _, category := range types.OrganizeDiscoverCategories() {
		tabs = append(tabs, types.OrganizeDiscoverTab{
			Label: category.Label,
			Value: category.Key,
			Count: categoryCounts[category.Key],
		})
	}
	return tabs
}

func filterDiscoverOutputs(outputs []*types.OrganizeOutput, tab, keyword string) []*types.OrganizeOutput {
	filtered := make([]*types.OrganizeOutput, 0, len(outputs))
	for _, output := range outputs {
		if output == nil {
			continue
		}
		if !matchesDiscoverTab(output, tab) {
			continue
		}
		if keyword != "" && !matchesDiscoverKeyword(output, keyword) {
			continue
		}
		filtered = append(filtered, output)
	}
	return filtered
}

func sortDiscoverOutputs(outputs []*types.OrganizeOutput) {
	sort.SliceStable(outputs, func(i, j int) bool {
		left := outputs[i]
		right := outputs[j]
		if left == nil || right == nil {
			return left != nil
		}

		leftStatus := discoverStatusWeight(left.Status)
		rightStatus := discoverStatusWeight(right.Status)
		if leftStatus != rightStatus {
			return leftStatus > rightStatus
		}

		if left.MemoryCount != right.MemoryCount {
			return left.MemoryCount > right.MemoryCount
		}

		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}

		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}

		return left.ID < right.ID
	})
}

func pickDiscoverFeaturedOutputs(outputs []*types.OrganizeOutput, offset int) []*types.OrganizeOutput {
	if len(outputs) == 0 {
		return []*types.OrganizeOutput{}
	}
	count := min(4, len(outputs))
	if count == 0 {
		return []*types.OrganizeOutput{}
	}
	start := offset % len(outputs)
	if start < 0 {
		start = 0
	}
	featured := make([]*types.OrganizeOutput, 0, count)
	for i := 0; i < count; i++ {
		featured = append(featured, outputs[(start+i)%len(outputs)])
	}
	return featured
}

func paginateDiscoverOutputs(outputs []*types.OrganizeOutput, page, pageSize int) []*types.OrganizeOutput {
	if len(outputs) == 0 || pageSize < 1 {
		return []*types.OrganizeOutput{}
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= len(outputs) {
		return []*types.OrganizeOutput{}
	}
	end := start + pageSize
	if end > len(outputs) {
		end = len(outputs)
	}
	return outputs[start:end]
}

func matchesDiscoverTab(output *types.OrganizeOutput, tab string) bool {
	normalizedTab := normalizeDiscoverTab(tab)
	switch normalizedTab {
	case "", "recommended":
		return true
	default:
		return discoverOutputCategory(output) == normalizedTab
	}
}

func matchesDiscoverKeyword(output *types.OrganizeOutput, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		output.Title,
		output.OutputType,
		output.Content,
		output.SourceSummary,
		output.UserID,
		output.Icon,
		strings.Join(discoverOutputTags(output.Metadata), " "),
		discoverOutputMetadataText(output.Metadata, "creator_name", "creator_username", "author_name", "user_name"),
		discoverOutputMetadataText(output.Metadata, "creator_id", "created_by", "user_id"),
	}, " "))
	return strings.Contains(haystack, needle)
}

func discoverOutputKind(output *types.OrganizeOutput) string {
	metadata := output.Metadata
	if metadata != nil {
		if kind := normalizeDiscoverOutputKind(discoverOutputMetadataText(metadata, "content_kind", "kind")); kind != "" {
			return kind
		}
		if kind := normalizeDiscoverOutputKind(discoverOutputMetadataText(metadata, "content_kind_label", "output_type")); kind != "" {
			return kind
		}
	}
	if kind := normalizeDiscoverOutputKind(output.OutputType); kind != "" {
		return kind
	}
	switch strings.ToLower(strings.TrimSpace(output.Icon)) {
	case "play-circle":
		return "video"
	case "sound":
		return "audio"
	default:
		return "article"
	}
}

func discoverOutputCategory(output *types.OrganizeOutput) string {
	if output == nil {
		return ""
	}
	metadata := output.Metadata
	category := discoverOutputMetadataText(metadata, "discover_category", "category", "discover_category_label")
	for _, candidate := range types.OrganizeDiscoverCategories() {
		if category == candidate.Key || category == candidate.Label {
			return candidate.Key
		}
	}
	if category == "团队运营与园长领导力" {
		return types.OrganizeDiscoverCategoryTeamLeadership
	}
	if category == "空间设计" {
		return types.OrganizeDiscoverCategorySpaceEnvironment
	}
	if category == "教师成长" {
		return types.OrganizeDiscoverCategoryTeacherResearch
	}
	return ""
}

func normalizeDiscoverOutputKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "video", "视频类":
		return "video"
	case "audio", "音频类":
		return "audio"
	case "article", "图文类":
		return "article"
	default:
		return ""
	}
}

func discoverStatusWeight(status string) int {
	switch strings.TrimSpace(status) {
	case types.OrganizeOutputStatusReady:
		return 3
	case types.OrganizeOutputStatusReview:
		return 2
	case types.OrganizeOutputStatusDraft:
		return 1
	case types.OrganizeOutputStatusArchived:
		return 0
	default:
		return 0
	}
}

func discoverOutputTags(metadata types.JSONMap) []string {
	raw, ok := metadata["tags"]
	if !ok || raw == nil {
		return nil
	}

	var candidates []string
	switch value := raw.(type) {
	case types.StringArray:
		candidates = append(candidates, []string(value)...)
	case []string:
		candidates = append(candidates, value...)
	case []any:
		for _, item := range value {
			if s, ok := item.(string); ok {
				candidates = append(candidates, s)
			}
		}
	case string:
		candidates = append(candidates, strings.FieldsFunc(value, func(r rune) bool {
			switch r {
			case ',', '，', ';', '；', '\n':
				return true
			default:
				return false
			}
		})...)
	default:
		return nil
	}

	cleaned := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		tag := strings.TrimSpace(candidate)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, tag)
	}
	return cleaned
}

func discoverOutputMetadataText(metadata types.JSONMap, keys ...string) string {
	if metadata == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := metadata[key]; ok {
			if s, ok := raw.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func normalizeDiscoverTab(tab string) string {
	tab = strings.TrimSpace(tab)
	switch tab {
	case "推荐":
		return "recommended"
	default:
		for _, category := range types.OrganizeDiscoverCategories() {
			if tab == category.Label {
				return category.Key
			}
		}
		if tab == "团队运营与园长领导力" {
			return types.OrganizeDiscoverCategoryTeamLeadership
		}
		if tab == "空间设计" {
			return types.OrganizeDiscoverCategorySpaceEnvironment
		}
		if tab == "教师成长" {
			return types.OrganizeDiscoverCategoryTeacherResearch
		}
		return tab
	}
}
