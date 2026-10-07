package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	organizeRequirementMemoryLimit = 1000
	organizeBatchInputBudget       = 14000
	organizePreviewSampleLimit     = 5
)

var (
	organizeExplicitDateRangePattern  = regexp.MustCompile(`(?:(\d{4})年)?(\d{1,2})月(\d{1,2})日?(?:至|到|~|～|-)(?:(\d{4})年)?(\d{1,2})月(\d{1,2})日?`)
	organizeExplicitSingleDatePattern = regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
)

type organizeBatchPlan struct {
	MemoryIDs  []string
	InputChars int
}

func parseOrganizeMemoryQueryPlan(
	text string,
	now time.Time,
) (types.OrganizeMemoryQueryPlan, []string, []string) {
	location := organizeScheduleLocation()
	localNow := now.In(location)
	compact := strings.NewReplacer(" ", "", "\t", "", "\n", "").Replace(strings.TrimSpace(text))
	plan := types.OrganizeMemoryQueryPlan{
		TimeField: "occurred_at",
		Timezone:  location.String(),
	}
	ambiguities := make([]string, 0, 2)
	warnings := make([]string, 0, 2)

	var start, end time.Time
	switch {
	case strings.Contains(compact, "今日") || strings.Contains(compact, "今天"):
		start = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
		end = start.AddDate(0, 0, 1)
	case strings.Contains(compact, "昨日") || strings.Contains(compact, "昨天"):
		end = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
		start = end.AddDate(0, 0, -1)
	case strings.Contains(compact, "上周"):
		thisWeek := localWeekStart(localNow, location)
		start = thisWeek.AddDate(0, 0, -7)
		end = thisWeek
	case strings.Contains(compact, "本周") || strings.Contains(compact, "这周"):
		start = localWeekStart(localNow, location)
		end = start.AddDate(0, 0, 7)
	case strings.Contains(compact, "上月"):
		thisMonth := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, location)
		start = thisMonth.AddDate(0, -1, 0)
		end = thisMonth
	case strings.Contains(compact, "本月") || strings.Contains(compact, "这个月"):
		start = time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	default:
		if matches := organizeExplicitDateRangePattern.FindStringSubmatch(compact); len(matches) == 7 {
			startYear := intValue(matches[1], localNow.Year())
			endYear := intValue(matches[4], startYear)
			start = safeLocalDate(startYear, intValue(matches[2], 1), intValue(matches[3], 1), location)
			endDate := safeLocalDate(endYear, intValue(matches[5], 1), intValue(matches[6], 1), location)
			if !start.IsZero() && !endDate.IsZero() {
				end = endDate.AddDate(0, 0, 1)
			}
		} else if matches := organizeExplicitSingleDatePattern.FindStringSubmatch(compact); len(matches) == 4 {
			start = safeLocalDate(
				intValue(matches[1], localNow.Year()),
				intValue(matches[2], 1),
				intValue(matches[3], 1),
				location,
			)
			if !start.IsZero() {
				end = start.AddDate(0, 0, 1)
			}
		}
	}

	if !start.IsZero() && !end.IsZero() && start.Before(end) {
		plan.OccurredFrom = start.Format(time.RFC3339)
		plan.OccurredTo = end.Format(time.RFC3339)
	} else {
		ambiguities = append(ambiguities, "未识别到明确时间范围，将使用当前配置的默认记忆范围")
	}

	kinds := make([]string, 0, 4)
	if strings.Contains(compact, "录音卡") || strings.Contains(compact, "工牌") {
		kinds = append(kinds, types.OrganizeMemoryKindAudioCard)
	}
	if strings.Contains(compact, "录音") || strings.Contains(compact, "音频") {
		kinds = append(kinds, types.OrganizeMemoryKindAudio, types.OrganizeMemoryKindAudioCard)
	}
	if strings.Contains(compact, "笔记") {
		kinds = append(kinds, types.OrganizeMemoryKindNote)
	}
	if strings.Contains(compact, "记录") {
		kinds = append(kinds, types.OrganizeMemoryKindRecord)
	}
	plan.Kinds = types.StringArray(uniqueOrderedStrings(kinds))

	if strings.Contains(compact, "只处理已完成") || strings.Contains(compact, "只整理已完成") {
		plan.ReadyOnly = true
	}
	return plan, ambiguities, warnings
}

func localWeekStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	daysSinceMonday := (int(local.Weekday()) - int(time.Monday) + 7) % 7
	return time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, location)
}

func intValue(raw string, fallback int) int {
	if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		return value
	}
	return fallback
}

func safeLocalDate(year, month, day int, location *time.Location) time.Time {
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}
	}
	value := time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
	if value.Year() != year || int(value.Month()) != month || value.Day() != day {
		return time.Time{}
	}
	return value
}

func organizeQueryPlanListQuery(
	tenantID uint64,
	userID string,
	plan types.OrganizeMemoryQueryPlan,
) (types.OrganizeListQuery, error) {
	query := types.OrganizeListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		Keyword:   strings.TrimSpace(plan.Keyword),
		Kinds:     append([]string(nil), plan.Kinds...),
		Sources:   append([]string(nil), plan.Sources...),
		ReadyOnly: plan.ReadyOnly,
		Page:      1,
		PageSize:  organizeRequirementMemoryLimit,
	}
	if raw := strings.TrimSpace(plan.OccurredFrom); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return query, fmt.Errorf("invalid occurred_from: %w", err)
		}
		query.OccurredFrom = &value
	}
	if raw := strings.TrimSpace(plan.OccurredTo); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return query, fmt.Errorf("invalid occurred_to: %w", err)
		}
		query.OccurredTo = &value
	}
	return query, nil
}

func organizeMemoryReady(memory *types.OrganizeMemory, allowPartial bool) bool {
	if memory == nil {
		return false
	}
	attachmentStatus := strings.TrimSpace(stringValue(memory.Metadata, "attachment_status"))
	transcriptionStatus := strings.ToLower(strings.TrimSpace(stringValue(memory.Metadata, "transcription_status")))
	switch attachmentStatus {
	case types.OrganizeMemoryAttachmentAggregatePending,
		types.OrganizeMemoryAttachmentAggregateProcessing,
		types.OrganizeMemoryAttachmentAggregateFailed:
		return false
	case types.OrganizeMemoryAttachmentAggregatePartial:
		if !allowPartial {
			return false
		}
	}
	switch transcriptionStatus {
	case "pending", "transcribing", "failed":
		return false
	case "partial":
		return allowPartial
	}
	return true
}

func splitOrganizeMemoriesByReadiness(
	memories []*types.OrganizeMemory,
	allowPartial bool,
) ([]*types.OrganizeMemory, []*types.OrganizeMemory) {
	ready := make([]*types.OrganizeMemory, 0, len(memories))
	unready := make([]*types.OrganizeMemory, 0)
	for _, memory := range memories {
		if organizeMemoryReady(memory, allowPartial) {
			ready = append(ready, memory)
		} else {
			unready = append(unready, memory)
		}
	}
	return ready, unready
}

func planOrganizeMemoryBatches(memories []*types.OrganizeMemory) []organizeBatchPlan {
	if len(memories) == 0 {
		return nil
	}
	plans := make([]organizeBatchPlan, 0, 1)
	current := organizeBatchPlan{}
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		inputChars := organizeMemoryPromptChars(memory)
		if len(current.MemoryIDs) > 0 && current.InputChars+inputChars > organizeBatchInputBudget {
			plans = append(plans, current)
			current = organizeBatchPlan{}
		}
		current.MemoryIDs = append(current.MemoryIDs, memory.ID)
		current.InputChars += inputChars
	}
	if len(current.MemoryIDs) > 0 {
		plans = append(plans, current)
	}
	return plans
}

func organizeMemoryPromptChars(memory *types.OrganizeMemory) int {
	content := strings.TrimSpace(memory.Content)
	if content == "" {
		content = memory.Title
	}
	runes := utf8.RuneCountInString(content)
	if runes > 2200 {
		runes = 2200
	}
	return runes + utf8.RuneCountInString(memory.Title) + 160
}

func organizeSelectionSnapshot(
	plan types.OrganizeMemoryQueryPlan,
	text string,
	memories []*types.OrganizeMemory,
	now time.Time,
) types.JSONMap {
	versions := make(types.JSONMap, len(memories))
	ids := make([]string, 0, len(memories))
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		ids = append(ids, memory.ID)
		hash := sha256.Sum256([]byte(memory.Content))
		versions[memory.ID] = types.JSONMap{
			"updated_at":   memory.UpdatedAt.UTC().Format(time.RFC3339Nano),
			"content_hash": hex.EncodeToString(hash[:]),
		}
	}
	return types.JSONMap{
		"query_plan":             plan,
		"normalized_instruction": strings.TrimSpace(text),
		"memory_ids":             ids,
		"memory_versions":        versions,
		"matched_at":             now.UTC().Format(time.RFC3339Nano),
		"timezone":               emptyFallback(plan.Timezone, organizeScheduleLocation().String()),
	}
}

func organizeInputFingerprint(
	tenantID uint64,
	userID string,
	configID string,
	templateKey string,
	templateVersion string,
	requirement string,
	memories []*types.OrganizeMemory,
) string {
	type memoryVersion struct {
		ID        string
		UpdatedAt string
	}
	versions := make([]memoryVersion, 0, len(memories))
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		versions = append(versions, memoryVersion{
			ID:        memory.ID,
			UpdatedAt: memory.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].ID < versions[j].ID })
	var builder strings.Builder
	fmt.Fprintf(
		&builder,
		"%d|%s|%s|%s|%s|%s",
		tenantID,
		userID,
		configID,
		templateKey,
		templateVersion,
		strings.TrimSpace(requirement),
	)
	for _, version := range versions {
		fmt.Fprintf(&builder, "|%s:%s", version.ID, version.UpdatedAt)
	}
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

func uniqueOrderedStrings(values []string) []string {
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
