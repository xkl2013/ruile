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
	categories, err := s.ListDiscoverCategories(ctx)
	if err != nil {
		return nil, err
	}
	categoryDTOs := discoverCategoryDTOs(categories)
	if len(categoryDTOs) == 0 {
		categoryDTOs = types.OrganizeDiscoverCategories()
	}
	query.Tab = normalizeDiscoverTab(strings.TrimSpace(query.Tab), categoryDTOs)
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

	// Courses have their own tab. Their count is still calculated from the
	// published course pool so the tab remains useful even when no curation
	// flags have been configured yet.
	courseCount, err := s.countPublishedCourses(ctx)
	if err != nil {
		return nil, err
	}

	tabs := buildDiscoverTabs(outputs, courseCount, categoryDTOs)
	filtered := filterDiscoverOutputs(outputs, query.Tab, query.Keyword, categoryDTOs)
	sortDiscoverOutputs(filtered)

	totalPages := max(1, (len(filtered)+query.PageSize-1)/query.PageSize)
	if query.Page > totalPages {
		query.Page = totalPages
	}

	featuredCandidates := filtered
	if query.Tab == "" || query.Tab == "recommended" {
		featuredCandidates = outputs
	}
	featured := pickDiscoverFeaturedOutputs(featuredCandidates, query.Keyword, query.FeaturedOffset)
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

// countPublishedCourses returns how many course cards belong in 推荐.
// PageSize 1 keeps it a COUNT without dragging rows across.
func (s *organizeService) countPublishedCourses(ctx context.Context) (int64, error) {
	_, total, err := s.repo.ListCourses(ctx, types.OrganizeCourseQuery{
		PublicStatus: types.OrganizePublicContentStatusPublished,
		Page:         1,
		PageSize:     1,
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *organizeService) listAllDiscoverOutputs(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeOutput, error) {
	pageSize := organizeMaxPageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	outputs := make([]*types.OrganizeOutput, 0, pageSize)
	query := types.OrganizePublicContentQuery{
		PublicStatus:         types.OrganizePublicContentStatusPublished,
		ExcludeCourseLessons: true,
		Page:                 1,
		PageSize:             pageSize,
	}
	for {
		items, total, err := s.repo.ListPublicContents(ctx, query)
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

func buildDiscoverTabs(
	outputs []*types.OrganizeOutput,
	courseCount int64,
	categories []types.OrganizeDiscoverCategory,
) []types.OrganizeDiscoverTab {
	categoryCounts := make(map[string]int64)
	var recommendedCount int64

	for _, output := range outputs {
		if output == nil {
			continue
		}
		if isDiscoverCourseOutput(output) {
			continue
		}
		// Recommendation is the default public feed. Curation flags affect
		// featured/category placement, but must not make the first tab empty.
		recommendedCount++
		if output.Recommendable {
			if category := discoverOutputCategory(output, categories); category != "" {
				categoryCounts[category]++
			}
		}
	}

	tabs := []types.OrganizeDiscoverTab{{
		Label: "推荐",
		Value: "recommended",
		Count: recommendedCount,
	}, {
		Label: "系列课程",
		Value: types.OrganizeDiscoverTabCourse,
		Count: courseCount,
	}}
	for _, category := range categories {
		tabs = append(tabs, types.OrganizeDiscoverTab{
			Label: category.Label,
			Value: category.Key,
			Count: categoryCounts[category.Key],
		})
	}
	return tabs
}

func filterDiscoverOutputs(
	outputs []*types.OrganizeOutput,
	tab,
	keyword string,
	categories []types.OrganizeDiscoverCategory,
) []*types.OrganizeOutput {
	filtered := make([]*types.OrganizeOutput, 0, len(outputs))
	for _, output := range outputs {
		if output == nil {
			continue
		}
		if !matchesDiscoverTab(output, tab, categories) {
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

		if left.SortOrder != right.SortOrder {
			if left.SortOrder == 0 {
				return false
			}
			if right.SortOrder == 0 {
				return true
			}
			return left.SortOrder < right.SortOrder
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

func pickDiscoverFeaturedOutputs(outputs []*types.OrganizeOutput, keyword string, offset int) []*types.OrganizeOutput {
	if len(outputs) == 0 {
		return []*types.OrganizeOutput{}
	}
	candidates := make([]*types.OrganizeOutput, 0, len(outputs))
	explicit := hasExplicitFeatured(outputs)
	for _, output := range outputs {
		if output == nil || isDiscoverCourseOutput(output) {
			continue
		}
		if explicit {
			if !output.Featured {
				continue
			}
		} else if !output.Recommendable {
			continue
		}
		if keyword != "" && !matchesDiscoverKeyword(output, keyword) {
			continue
		}
		candidates = append(candidates, output)
	}
	sortDiscoverOutputs(candidates)
	count := min(4, len(candidates))
	if count == 0 {
		return []*types.OrganizeOutput{}
	}
	start := offset % len(candidates)
	if start < 0 {
		start = 0
	}
	featured := make([]*types.OrganizeOutput, 0, count)
	for i := 0; i < count; i++ {
		featured = append(featured, candidates[(start+i)%len(candidates)])
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

func matchesDiscoverTab(output *types.OrganizeOutput, tab string, categories []types.OrganizeDiscoverCategory) bool {
	normalizedTab := normalizeDiscoverTab(tab, categories)
	isCourseOutput := isDiscoverCourseOutput(output)
	if output == nil {
		return false
	}

	switch normalizedTab {
	case "", "recommended":
		return !isCourseOutput
	case types.OrganizeDiscoverTabCourse:
		// Course cards come from /organize/courses, not organize_outputs.
		return false
	default:
		if !output.Recommendable {
			return false
		}
		return !isCourseOutput && discoverOutputCategory(output, categories) == normalizedTab
	}
}

// isDiscoverCourseOutput reports whether an output is a course lesson rather
// than a post of its own.
//
// This is a second line of defence, not the mechanism. Lessons are written with
// public_status = 'draft' and therefore never reach ListPublicContents in the
// first place; this guard catches rows that predate that rule (or that someone
// published by hand) so a stray lesson can never surface as a standalone card.
func isDiscoverCourseOutput(output *types.OrganizeOutput) bool {
	if output == nil {
		return false
	}
	// Legacy posts may use series_id for ordinary grouping. A course lesson is
	// identified by both the explicit course content type and a real course
	// linkage; the repository query above removes linked lessons already, while
	// this guard protects against manually published or stale rows.
	return output.PublicContentType == types.OrganizePublicContentTypeCourse &&
		strings.TrimSpace(output.SeriesID) != ""
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

func discoverOutputCategory(output *types.OrganizeOutput, categories []types.OrganizeDiscoverCategory) string {
	if output == nil {
		return ""
	}
	metadata := output.Metadata
	category := discoverOutputMetadataText(metadata, "discover_category", "category", "discover_category_label")
	for _, candidate := range categories {
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

func hasExplicitFeatured(outputs []*types.OrganizeOutput) bool {
	for _, output := range outputs {
		if output != nil && output.Featured {
			return true
		}
	}
	return false
}

func boolPtr(value bool) *bool {
	return &value
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

func normalizeDiscoverTab(tab string, categories []types.OrganizeDiscoverCategory) string {
	tab = strings.TrimSpace(tab)
	switch tab {
	case "推荐":
		return "recommended"
	case "系列课程", types.OrganizeDiscoverTabCourse:
		return types.OrganizeDiscoverTabCourse
	default:
		for _, category := range categories {
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

func discoverCategoryDTOs(
	records []*types.OrganizeDiscoverCategoryRecord,
) []types.OrganizeDiscoverCategory {
	items := make([]types.OrganizeDiscoverCategory, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		items = append(items, types.OrganizeDiscoverCategory{
			Key:   record.Key,
			Label: record.Label,
		})
	}
	return items
}
