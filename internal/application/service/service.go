package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrServiceInvalidScope         = errors.New("invalid service scope")
	ErrServiceNotFound             = errors.New("service item not found")
	ErrServiceProfileNameRequired  = errors.New("work profile name is required")
	ErrServiceInvalidProfileState  = errors.New("invalid work profile state")
	ErrServiceInvalidAgentDomain   = errors.New("invalid service agent domain")
	ErrServiceInvalidStatus        = errors.New("invalid service status")
	ErrServiceInvalidReportRange   = errors.New("invalid service daily report range")
	ErrServiceInvalidReportDate    = errors.New("invalid service daily report date")
	ErrServiceProfileNotConfigured = errors.New("service profile is not configured")
)

const (
	serviceDefaultPage     = 1
	serviceDefaultPageSize = 20
	serviceMaxPageSize     = 100
	serviceMaxTitleLength  = 512
	serviceMaxShortText    = 255
)

const (
	serviceAgentSettingsSourceKey      = "auto_profile_source"
	serviceAgentSettingsHashKey        = "auto_profile_hash"
	serviceAgentSettingsModelKey       = "auto_profile_model_id"
	serviceAgentSettingsReasonKey      = "auto_profile_reason"
	serviceAgentSettingsSourceLLM      = "work_profile_description_llm"
	serviceAgentSettingsSourceFallback = "work_profile_description_rules"

	serviceMemoryExtractionSourceLLM      = "memory_service_llm"
	serviceMemoryExtractionSourceFallback = "memory_service_rules"
)

var serviceAgentCapabilityPlanSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "enabled_domains": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["memory_router", "lead_intake", "sales_consulting", "customer_service", "schedule_coordination", "after_sale_risk", "daily_review"]
      }
    },
    "reason": { "type": "string" }
  },
  "required": ["enabled_domains"]
}`)

type serviceAgentCapabilityPlan struct {
	EnabledDomains []string `json:"enabled_domains"`
	Reason         string   `json:"reason,omitempty"`
}

var serviceMemoryExtractionPlanSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "should_generate": { "type": "boolean" },
    "agent_domain": {
      "type": "string",
      "enum": ["memory_router", "lead_intake", "sales_consulting", "customer_service", "schedule_coordination", "after_sale_risk", "daily_review"]
    },
    "service_mode": { "type": "string" },
    "subject_name": { "type": "string" },
    "customer_name": { "type": "string" },
    "student_name": { "type": "string" },
    "title": { "type": "string" },
    "summary": { "type": "string" },
    "stage": { "type": "string" },
    "priority": { "type": "string", "enum": ["high", "medium", "low"] },
    "due_text": { "type": "string" },
    "risk_label": { "type": "string" },
    "assist_reason": { "type": "string" },
    "primary_action": { "type": "string" },
    "next_action": { "type": "string" },
    "avoid_action": { "type": "string" },
    "reply_draft": { "type": "string" },
    "memory_signals": { "type": "array", "items": { "type": "string" } },
    "sales_highlights": { "type": "array", "items": { "type": "string" } },
    "work_doc_directory": { "type": "string" },
    "reason": { "type": "string" }
  },
  "required": ["should_generate"]
}`)

type serviceMemoryExtractionPlan struct {
	ShouldGenerate   bool     `json:"should_generate"`
	AgentDomain      string   `json:"agent_domain,omitempty"`
	ServiceMode      string   `json:"service_mode,omitempty"`
	SubjectName      string   `json:"subject_name,omitempty"`
	CustomerName     string   `json:"customer_name,omitempty"`
	StudentName      string   `json:"student_name,omitempty"`
	Title            string   `json:"title,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Stage            string   `json:"stage,omitempty"`
	Priority         string   `json:"priority,omitempty"`
	DueText          string   `json:"due_text,omitempty"`
	RiskLabel        string   `json:"risk_label,omitempty"`
	AssistReason     string   `json:"assist_reason,omitempty"`
	PrimaryAction    string   `json:"primary_action,omitempty"`
	NextAction       string   `json:"next_action,omitempty"`
	AvoidAction      string   `json:"avoid_action,omitempty"`
	ReplyDraft       string   `json:"reply_draft,omitempty"`
	MemorySignals    []string `json:"memory_signals,omitempty"`
	SalesHighlights  []string `json:"sales_highlights,omitempty"`
	WorkDocDirectory string   `json:"work_doc_directory,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

type serviceService struct {
	repo         interfaces.ServiceRepository
	members      interfaces.TenantMemberRepository
	modelService interfaces.ModelService

	organize interfaces.OrganizeRepository
}

func NewServiceService(
	repo interfaces.ServiceRepository,
	organize interfaces.OrganizeRepository,
) interfaces.ServiceService {
	return newServiceService(repo, organize, nil, nil)
}

func NewServiceServiceWithMembers(
	repo interfaces.ServiceRepository,
	organize interfaces.OrganizeRepository,
	members interfaces.TenantMemberRepository,
) interfaces.ServiceService {
	return newServiceService(repo, organize, members, nil)
}

func NewServiceServiceWithMembersAndModel(
	repo interfaces.ServiceRepository,
	organize interfaces.OrganizeRepository,
	members interfaces.TenantMemberRepository,
	modelService interfaces.ModelService,
) interfaces.ServiceService {
	return newServiceService(repo, organize, members, modelService)
}

func newServiceService(
	repo interfaces.ServiceRepository,
	organize interfaces.OrganizeRepository,
	members interfaces.TenantMemberRepository,
	modelService interfaces.ModelService,
) interfaces.ServiceService {
	return &serviceService{repo: repo, organize: organize, members: members, modelService: modelService}
}

func (s *serviceService) GetBootstrap(ctx context.Context, tenantID uint64, userID string) (*types.ServiceBootstrap, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	profile, settings, err := s.defaultProfileAndSettings(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateWorkProfileDescription(ctx, tenantID, userID, profile); err != nil {
		return nil, err
	}
	if profile == nil || len(settings) == 0 {
		return &types.ServiceBootstrap{
			Profile:       profile,
			AgentSettings: settings,
			Reminders:     []*types.ServiceReminder{},
			Total:         0,
			Stats:         map[string]int64{},
			Templates:     s.ListAgentTemplates(ctx),
		}, nil
	}
	if err := s.refreshFromMemories(ctx, tenantID, userID, profile, settings); err != nil {
		return nil, err
	}
	query := normalizeServiceListQuery(types.ServiceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: profileID(profile),
		Page:      1,
		PageSize:  serviceDefaultPageSize,
	})
	reminders, total, err := s.repo.ListReminders(ctx, query)
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.CountRemindersByStatus(ctx, tenantID, userID, profileID(profile))
	if err != nil {
		return nil, err
	}
	return &types.ServiceBootstrap{
		Profile:       profile,
		AgentSettings: settings,
		Reminders:     reminders,
		Total:         total,
		Stats:         stats,
		Templates:     s.ListAgentTemplates(ctx),
	}, nil
}

func (s *serviceService) ListAgentTemplates(ctx context.Context) []types.ServiceAgentTemplate {
	_ = ctx
	return []types.ServiceAgentTemplate{
		{
			AgentDomain:      types.ServiceAgentDomainMemoryRouter,
			DisplayName:      "记忆路由器",
			Description:      "识别记忆属于哪个业务域，仅写路由 metadata，不直接服务用户。",
			DefaultEnabled:   true,
			UserVisible:      false,
			WorkDocDirectory: "路由/",
			MemoryFilter: types.JSONMap{
				"source": "organize_memories",
			},
			OutputPolicy: types.JSONMap{
				"generate_reminder": false,
			},
		},
		{
			AgentDomain:      types.ServiceAgentDomainLeadIntake,
			DisplayName:      "线索录入",
			Description:      "从咨询、试听、报名意向记忆中整理线索草稿和缺失信息。",
			DefaultEnabled:   false,
			UserVisible:      false,
			WorkDocDirectory: "线索/",
			MemoryFilter: types.JSONMap{
				"keywords": []string{"咨询", "试听", "体验课", "报名", "邀约"},
			},
			OutputPolicy: types.JSONMap{
				"requires_user_confirmation": true,
				"external_write":             "draft_only",
			},
		},
		{
			AgentDomain:      types.ServiceAgentDomainSalesConsulting,
			DisplayName:      "招生咨询",
			Description:      "生成异议处理、邀约话术、试听后跟进和下一步建议。",
			DefaultEnabled:   false,
			UserVisible:      false,
			WorkDocDirectory: "线索/",
			MemoryFilter: types.JSONMap{
				"keywords": []string{"价格", "顾虑", "异议", "试听", "成交"},
			},
			OutputPolicy: types.JSONMap{
				"requires_user_confirmation": true,
				"text_first":                 true,
			},
		},
		{
			AgentDomain:      types.ServiceAgentDomainCustomerService,
			DisplayName:      "客户服务",
			Description:      "整理客户摘要、跟进记录、续费窗口和服务闭环事项。",
			DefaultEnabled:   false,
			UserVisible:      false,
			WorkDocDirectory: "客户/",
			MemoryFilter: types.JSONMap{
				"keywords": []string{"客户", "家长", "续费", "回访", "反馈"},
			},
			OutputPolicy: types.JSONMap{
				"requires_user_confirmation": true,
				"store_markdown":             true,
			},
		},
		{
			AgentDomain:      types.ServiceAgentDomainScheduling,
			DisplayName:      "排课调课",
			Description:      "识别请假、补课、排课和调课信号，生成待确认安排。",
			DefaultEnabled:   false,
			UserVisible:      false,
			WorkDocDirectory: "排课/",
			MemoryFilter: types.JSONMap{
				"keywords": []string{"排课", "调课", "请假", "补课", "老师", "教室"},
			},
			OutputPolicy: types.JSONMap{
				"requires_user_confirmation": true,
				"external_write":             "draft_only",
			},
		},
		{
			AgentDomain:      types.ServiceAgentDomainDailyReview,
			DisplayName:      "日报复盘",
			Description:      "按用户触发汇总服务提醒、风险、动作闭环和知识补齐建议。",
			DefaultEnabled:   false,
			UserVisible:      false,
			WorkDocDirectory: "日报/",
			MemoryFilter: types.JSONMap{
				"source": "service_reminders",
			},
			OutputPolicy: types.JSONMap{
				"store_markdown": true,
				"trigger":        "user_requested",
			},
		},
	}
}

func (s *serviceService) RefreshUserService(ctx context.Context, tenantID uint64, userID string) (*types.ServiceBootstrap, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	return s.GetBootstrap(ctx, tenantID, userID)
}

func (s *serviceService) ExtractMemory(ctx context.Context, tenantID uint64, userID, memoryID string) (*types.ServiceMemoryExtraction, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	ctx = withServiceTenantContext(ctx, tenantID)
	memoryID = strings.TrimSpace(memoryID)
	if memoryID == "" {
		return nil, ErrServiceNotFound
	}
	profile, settings, err := s.defaultProfileAndSettings(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return &types.ServiceMemoryExtraction{
			MemoryID:  memoryID,
			Generated: false,
			Reason:    "profile_not_configured",
		}, nil
	}
	memory, err := s.organize.GetMemory(ctx, tenantID, userID, memoryID)
	if err != nil {
		return nil, err
	}
	if memory == nil {
		return nil, ErrServiceNotFound
	}
	enabledDomains := enabledServiceAgentSettings(settings)
	if len(settings) == 0 || len(enabledDomains) == 0 {
		return s.extractMemoryWithGeneratedService(ctx, tenantID, userID, profile, enabledDomains, memory, "agent_not_enabled")
	}
	if !isServiceMemory(memory) {
		return s.extractMemoryWithGeneratedService(ctx, tenantID, userID, profile, enabledDomains, memory, "memory_not_relevant")
	}
	s.annotateMemoryRoute(ctx, memory)
	reminder, reason, err := s.upsertServiceMemoryGroup(ctx, tenantID, userID, profile, enabledDomains, extractCustomerName(memory), []*types.OrganizeMemory{memory})
	if err != nil {
		return nil, err
	}
	if reminder == nil && reason == "agent_not_enabled" {
		return s.extractMemoryWithGeneratedService(ctx, tenantID, userID, profile, enabledDomains, memory, reason)
	}
	return &types.ServiceMemoryExtraction{
		MemoryID:  memory.ID,
		Generated: reminder != nil,
		Reason:    reason,
		Reminder:  reminder,
	}, nil
}

func (s *serviceService) GenerateDailyReport(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceDailyReportInput,
) (*types.ServiceDailyReport, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	period, err := resolveServiceDailyReportPeriod(input)
	if err != nil {
		return nil, err
	}
	profile, settings, err := s.defaultProfileAndSettings(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrServiceProfileNotConfigured
	}
	if len(settings) > 0 {
		if err := s.refreshFromMemories(ctx, tenantID, userID, profile, settings); err != nil {
			return nil, err
		}
	}
	reminders, _, err := s.repo.ListReminders(ctx, normalizeServiceListQuery(types.ServiceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: profile.ID,
		Page:      1,
		PageSize:  serviceMaxPageSize,
	}))
	if err != nil {
		return nil, err
	}
	reportReminders := filterRemindersForDailyReport(reminders, period)
	var dailySources []*types.AgentWorkDoc
	if period.Range != types.ServiceDailyReportRangeDay {
		dailySources, err = s.dailySourceReportDocs(ctx, tenantID, userID, profile.ID, period)
		if err != nil {
			return nil, err
		}
	}
	stats := buildDailyReportStats(reportReminders, dailySources)
	reportSubject := buildDailyReportSubject(tenantID, userID, profile.ID)
	if err := s.repo.UpsertSubject(ctx, reportSubject); err != nil {
		return nil, err
	}
	content := buildDailyReportMarkdown(period, profile, reportReminders, dailySources, stats)
	sourceMemoryIDs := sourceMemoryIDsFromReminders(reportReminders)
	now := time.Now().UTC()
	doc := &types.AgentWorkDoc{
		TenantID:        tenantID,
		ProfileID:       profile.ID,
		SubjectID:       reportSubject.ID,
		OwnerUserID:     userID,
		AgentDomain:     types.ServiceAgentDomainDailyReview,
		DocType:         types.AgentWorkDocTypeDailyReport,
		DocPath:         dailyReportDocPath(period),
		Title:           dailyReportTitle(period),
		Content:         content,
		Status:          types.AgentWorkDocStatusCurrent,
		SourceMemoryIDs: sourceMemoryIDs,
		Metadata: types.JSONMap{
			"source":                 "service_module",
			"trigger":                normalizeServiceDailyReportTrigger(input.Trigger),
			"report_range":           period.Range,
			"report_date":            period.ReportDate,
			"period_start":           period.Start.UTC().Format(time.RFC3339),
			"period_end":             period.End.UTC().Format(time.RFC3339),
			"period_label":           dailyReportPeriodLabel(period),
			"timezone":               period.Location.String(),
			"stage":                  "已生成",
			"stage_key":              "formed",
			"updated":                now.In(period.Location).Format("01月02日 15:04") + " 生成",
			"action_count":           stats.ActionCount,
			"customer_count":         stats.SubjectCount,
			"subject_count":          stats.SubjectCount,
			"open_action_count":      stats.OpenActionCount,
			"closed_action_count":    stats.ClosedActionCount,
			"high_risk_count":        stats.HighRiskCount,
			"knowledge_gap_count":    stats.KnowledgeGapCount,
			"source_memory_count":    stats.SourceMemoryCount,
			"daily_source_count":     stats.DailySourceCount,
			"evidence_complete_rate": stats.EvidenceCompleteRate,
			"profile_id":             profile.ID,
			"profile_name":           profile.Name,
			"profile_description":    profile.WorkProfileDescription,
			"memory_scope":           profile.MemoryScope,
			"can_supplement":         period.Range == types.ServiceDailyReportRangeDay,
			"collection_source":      "avatar_memory_reminders",
			"daily_report_logic":     "memory_to_work_loop",
			"write_back_policy":      "manual_confirmation",
			"chips":                  dailyReportChips(reportReminders, dailySources),
			"generated_by":           userID,
			"generated_at":           now.Format(time.RFC3339),
			"source_profile":         profile.ID,
			"source_doc_type":        types.AgentWorkDocTypeDailyReport,
		},
		UpdatedAt: now,
	}
	if err := s.repo.UpsertWorkDocWithLinks(ctx, doc, buildDailyReportMemoryLinks(doc, reportReminders)); err != nil {
		return nil, err
	}
	persisted, err := s.repo.GetDailyReportDoc(ctx, tenantID, userID, doc.ID)
	if err != nil {
		return nil, err
	}
	if persisted == nil {
		return nil, ErrServiceNotFound
	}
	return serviceDailyReportFromDoc(persisted), nil
}

func (s *serviceService) GetDailyReport(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceDailyReport, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	doc, err := s.repo.GetDailyReportDoc(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrServiceNotFound
	}
	return serviceDailyReportFromDoc(doc), nil
}

func (s *serviceService) ListDailyReports(
	ctx context.Context,
	query types.ServiceDailyReportListQuery,
) ([]*types.ServiceDailyReport, int64, error) {
	if err := validateServiceScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query = normalizeServiceDailyReportListQuery(query)
	if query.Range != "" && !types.IsValidServiceDailyReportRange(query.Range) {
		return nil, 0, ErrServiceInvalidReportRange
	}
	docs, total, err := s.repo.ListDailyReportDocs(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	reports := make([]*types.ServiceDailyReport, 0, len(docs))
	for _, doc := range docs {
		reports = append(reports, serviceDailyReportFromDoc(doc))
	}
	return reports, total, nil
}

func (s *serviceService) dailySourceReportDocs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profileID string,
	period serviceDailyReportPeriod,
) ([]*types.AgentWorkDoc, error) {
	docs, _, err := s.repo.ListDailyReportDocs(ctx, types.ServiceDailyReportListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: profileID,
		Range:     types.ServiceDailyReportRangeDay,
		Page:      1,
		PageSize:  serviceMaxPageSize,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*types.AgentWorkDoc, 0, len(docs))
	for _, doc := range docs {
		if dailyReportDocInPeriod(doc, period) {
			out = append(out, doc)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := dailyReportDocStartTime(out[i], period.Location)
		right := dailyReportDocStartTime(out[j], period.Location)
		if left.Equal(right) {
			return out[i].Title < out[j].Title
		}
		return left.Before(right)
	})
	return out, nil
}

func (s *serviceService) ListCustomerSpaces(
	ctx context.Context,
	query types.ServiceCustomerSpaceListQuery,
) ([]*types.ServiceCustomerSpace, int64, error) {
	if err := validateServiceScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query = normalizeServiceCustomerSpaceListQuery(query)
	profile, settings, err := s.resolveCustomerSpaceProfile(ctx, query.TenantID, query.UserID, query.ProfileID)
	if err != nil {
		return nil, 0, err
	}
	if profile == nil {
		return []*types.ServiceCustomerSpace{}, 0, nil
	}
	query.ProfileID = profile.ID
	if len(settings) > 0 {
		if err := s.refreshFromMemories(ctx, query.TenantID, query.UserID, profile, settings); err != nil {
			return nil, 0, err
		}
	}
	subjects, total, err := s.repo.ListSubjects(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	spaces := make([]*types.ServiceCustomerSpace, 0, len(subjects))
	for _, subject := range subjects {
		docs, err := s.repo.ListWorkDocsBySubject(ctx, query.TenantID, profile.ID, subject.ID)
		if err != nil {
			return nil, 0, err
		}
		reminders, _, err := s.repo.ListReminders(ctx, types.ServiceListQuery{
			TenantID:  query.TenantID,
			UserID:    query.UserID,
			ProfileID: profile.ID,
			SubjectID: subject.ID,
			Page:      1,
			PageSize:  serviceMaxPageSize,
		})
		if err != nil {
			return nil, 0, err
		}
		spaces = append(spaces, buildCustomerSpaceSummary(profile.ID, subject, docs, reminders))
	}
	return spaces, total, nil
}

func (s *serviceService) GetCustomerSpace(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
	profileID string,
) (*types.ServiceCustomerSpaceDetail, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrServiceNotFound
	}
	profile, settings, err := s.resolveCustomerSpaceProfile(ctx, tenantID, userID, profileID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrServiceProfileNotConfigured
	}
	if len(settings) > 0 {
		if err := s.refreshFromMemories(ctx, tenantID, userID, profile, settings); err != nil {
			return nil, err
		}
	}
	subject, err := s.repo.GetSubject(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrServiceNotFound
	}
	docs, err := s.repo.ListWorkDocsBySubject(ctx, tenantID, profile.ID, subject.ID)
	if err != nil {
		return nil, err
	}
	reminders, _, err := s.repo.ListReminders(ctx, types.ServiceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: profile.ID,
		SubjectID: subject.ID,
		Page:      1,
		PageSize:  serviceMaxPageSize,
	})
	if err != nil {
		return nil, err
	}
	summary := buildCustomerSpaceSummary(profile.ID, subject, docs, reminders)
	memoryEvidence, err := s.customerSpaceMemoryEvidence(ctx, tenantID, userID, collectCustomerSpaceMemoryIDs(docs, reminders))
	if err != nil {
		return nil, err
	}
	return &types.ServiceCustomerSpaceDetail{
		Summary:        summary,
		Subject:        subject,
		WorkDocs:       filterCustomerWorkDocs(docs),
		Reminders:      reminders,
		MemoryEvidence: memoryEvidence,
		Directories:    summary.Directories,
		Stats: map[string]int64{
			"work_doc_count":      int64(summary.WorkDocCount),
			"reminder_count":      int64(summary.ReminderCount),
			"open_reminder_count": int64(summary.OpenReminderCount),
			"source_memory_count": int64(summary.SourceMemoryCount),
		},
	}, nil
}

func (s *serviceService) ListReminders(ctx context.Context, query types.ServiceListQuery) ([]*types.ServiceReminder, int64, error) {
	if err := validateServiceScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query = normalizeServiceListQuery(query)
	if query.Status != "" && !types.IsValidServiceReminderStatus(query.Status) {
		return nil, 0, ErrServiceInvalidStatus
	}
	if query.AgentDomain != "" && !types.IsValidServiceAgentDomain(query.AgentDomain) {
		return nil, 0, ErrServiceInvalidAgentDomain
	}
	return s.repo.ListReminders(ctx, query)
}

func (s *serviceService) GetReminder(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceReminder, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	reminder, err := s.repo.GetReminder(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if reminder == nil {
		return nil, ErrServiceNotFound
	}
	return reminder, nil
}

func (s *serviceService) UpdateReminderStatus(ctx context.Context, tenantID uint64, userID, id, status string) (*types.ServiceReminder, error) {
	status = strings.TrimSpace(status)
	if !types.IsValidServiceReminderStatus(status) {
		return nil, ErrServiceInvalidStatus
	}
	if _, err := s.GetReminder(ctx, tenantID, userID, id); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateReminderStatus(ctx, tenantID, userID, id, status); err != nil {
		return nil, err
	}
	return s.GetReminder(ctx, tenantID, userID, id)
}

func (s *serviceService) ListWorkProfiles(ctx context.Context, tenantID uint64, userID string) ([]*types.UserWorkProfile, error) {
	if tenantID == 0 {
		return nil, ErrServiceInvalidScope
	}
	if err := s.materializeWorkProfilesFromMembers(ctx, tenantID, strings.TrimSpace(userID)); err != nil {
		return nil, err
	}
	profiles, err := s.repo.ListWorkProfiles(ctx, tenantID, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	if err := s.hydrateWorkProfileDescriptions(ctx, tenantID, strings.TrimSpace(userID), profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *serviceService) CreateWorkProfile(ctx context.Context, tenantID uint64, operatorUserID string, input types.ServiceWorkProfileInput) (*types.UserWorkProfile, error) {
	if tenantID == 0 || strings.TrimSpace(operatorUserID) == "" {
		return nil, ErrServiceInvalidScope
	}
	profile, err := buildWorkProfile(tenantID, operatorUserID, "", input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateWorkProfile(ctx, profile); err != nil {
		return nil, err
	}
	return s.repo.GetWorkProfile(ctx, tenantID, profile.ID)
}

func (s *serviceService) UpdateWorkProfile(ctx context.Context, tenantID uint64, operatorUserID, id string, input types.ServiceWorkProfileInput) (*types.UserWorkProfile, error) {
	if tenantID == 0 || strings.TrimSpace(operatorUserID) == "" {
		return nil, ErrServiceInvalidScope
	}
	existing, err := s.repo.GetWorkProfile(ctx, tenantID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrServiceNotFound
	}
	if strings.TrimSpace(input.UserID) == "" {
		input.UserID = existing.UserID
	}
	profile, err := buildWorkProfile(tenantID, operatorUserID, existing.ID, input)
	if err != nil {
		return nil, err
	}
	profile.CreatedBy = existing.CreatedBy
	if err := s.repo.UpdateWorkProfile(ctx, profile); err != nil {
		return nil, err
	}
	return s.repo.GetWorkProfile(ctx, tenantID, profile.ID)
}

func (s *serviceService) ReplaceAgentSettings(
	ctx context.Context, tenantID uint64, operatorUserID, profileID string, input types.WorkProfileAgentSettingsInput,
) ([]*types.WorkProfileAgentSetting, error) {
	if tenantID == 0 || strings.TrimSpace(operatorUserID) == "" {
		return nil, ErrServiceInvalidScope
	}
	profile, err := s.repo.GetWorkProfile(ctx, tenantID, strings.TrimSpace(profileID))
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrServiceNotFound
	}
	settings := make([]*types.WorkProfileAgentSetting, 0, len(input.Settings))
	for index, item := range input.Settings {
		setting, err := buildAgentSetting(tenantID, operatorUserID, profile.ID, item, index)
		if err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	if err := s.repo.ReplaceAgentSettings(ctx, tenantID, profile.ID, settings); err != nil {
		return nil, err
	}
	return s.repo.ListAgentSettings(ctx, tenantID, profile.ID, false)
}

func (s *serviceService) ListAgentSettings(ctx context.Context, tenantID uint64, profileID string, onlyEnabled bool) ([]*types.WorkProfileAgentSetting, error) {
	if tenantID == 0 || strings.TrimSpace(profileID) == "" {
		return nil, ErrServiceInvalidScope
	}
	ctx = withServiceTenantContext(ctx, tenantID)
	profile, err := s.repo.GetWorkProfile(ctx, tenantID, strings.TrimSpace(profileID))
	if err != nil || profile == nil {
		return nil, err
	}
	if err := s.hydrateWorkProfileDescription(ctx, tenantID, profile.UserID, profile); err != nil {
		return nil, err
	}
	if _, err := s.ensureInferredAgentSettings(ctx, tenantID, profile.UserID, profile); err != nil {
		return nil, err
	}
	return s.repo.ListAgentSettings(ctx, tenantID, profile.ID, onlyEnabled)
}

func (s *serviceService) CreateActionDraft(
	ctx context.Context, tenantID uint64, userID, reminderID string, input types.AgentActionDraftInput,
) (*types.AgentActionDraft, error) {
	reminder, err := s.GetReminder(ctx, tenantID, userID, reminderID)
	if err != nil {
		return nil, err
	}
	draft := buildActionDraftFromReminder(tenantID, userID, reminder, input)
	if err := s.repo.CreateActionDraft(ctx, draft); err != nil {
		return nil, err
	}
	if _, err := s.UpdateReminderStatus(ctx, tenantID, userID, reminderID, types.ServiceReminderStatusConfirmed); err != nil {
		return nil, err
	}
	return s.repo.GetActionDraft(ctx, tenantID, userID, draft.ID)
}

func (s *serviceService) ListActionDrafts(ctx context.Context, tenantID uint64, userID, reminderID string) ([]*types.AgentActionDraft, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reminderID) != "" {
		if _, err := s.GetReminder(ctx, tenantID, userID, reminderID); err != nil {
			return nil, err
		}
	}
	return s.repo.ListActionDrafts(ctx, tenantID, userID, reminderID)
}

func (s *serviceService) UpdateActionDraftStatus(ctx context.Context, tenantID uint64, userID, id, status string) (*types.AgentActionDraft, error) {
	status = strings.TrimSpace(status)
	if !types.IsValidAgentActionDraftStatus(status) {
		return nil, ErrServiceInvalidStatus
	}
	draft, err := s.repo.GetActionDraft(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, ErrServiceNotFound
	}
	if err := s.repo.UpdateActionDraftStatus(ctx, tenantID, userID, id, status); err != nil {
		return nil, err
	}
	_ = s.repo.CreateActionLog(ctx, &types.AgentActionLog{
		TenantID:      tenantID,
		ActionDraftID: id,
		Status:        status,
		Message:       "action draft status updated",
		Payload: types.JSONMap{
			"source": "service_module",
		},
	})
	return s.repo.GetActionDraft(ctx, tenantID, userID, id)
}

func (s *serviceService) defaultProfileAndSettings(
	ctx context.Context, tenantID uint64, userID string,
) (*types.UserWorkProfile, []*types.WorkProfileAgentSetting, error) {
	ctx = withServiceTenantContext(ctx, tenantID)
	profile, err := s.repo.GetDefaultWorkProfile(ctx, tenantID, userID)
	if err != nil {
		return nil, nil, err
	}
	if profile == nil {
		profile, err = s.ensureDefaultWorkProfileFromMember(ctx, tenantID, userID)
		if err != nil || profile == nil {
			return profile, nil, err
		}
	}
	if err := s.hydrateWorkProfileDescription(ctx, tenantID, userID, profile); err != nil {
		return nil, nil, err
	}
	settings, err := s.repo.ListAgentSettings(ctx, tenantID, profile.ID, true)
	if err != nil {
		return nil, nil, err
	}
	if s.shouldInferAgentSettings(profile, settings) {
		inferred, err := s.ensureInferredAgentSettings(ctx, tenantID, userID, profile)
		if err != nil {
			return nil, nil, err
		}
		if len(inferred) > 0 {
			settings = inferred
		}
	}
	return profile, settings, nil
}

func (s *serviceService) shouldInferAgentSettings(
	profile *types.UserWorkProfile,
	enabledSettings []*types.WorkProfileAgentSetting,
) bool {
	if profile == nil || strings.TrimSpace(profile.ID) == "" {
		return false
	}
	if strings.TrimSpace(workProfileAgentInferenceText(profile)) == "" {
		return false
	}
	if len(enabledSettings) == 0 {
		return true
	}
	hash := workProfileAgentSettingsHash(profile)
	for _, setting := range enabledSettings {
		if setting == nil {
			continue
		}
		if asString(setting.OutputPolicy[serviceAgentSettingsSourceKey]) == "" {
			return false
		}
		if asString(setting.OutputPolicy[serviceAgentSettingsHashKey]) != hash {
			return true
		}
	}
	return false
}

func (s *serviceService) ensureInferredAgentSettings(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
) ([]*types.WorkProfileAgentSetting, error) {
	if profile == nil || tenantID == 0 || strings.TrimSpace(profile.ID) == "" {
		return nil, nil
	}
	if err := s.hydrateWorkProfileDescription(ctx, tenantID, userID, profile); err != nil {
		return nil, err
	}
	if strings.TrimSpace(profile.WorkProfileDescription) == "" {
		return s.repo.ListAgentSettings(ctx, tenantID, profile.ID, true)
	}
	existing, err := s.repo.ListAgentSettings(ctx, tenantID, profile.ID, false)
	if err != nil {
		return nil, err
	}
	hash := workProfileAgentSettingsHash(profile)
	if !shouldReplaceAgentSettingsFromProfile(existing, hash) {
		return s.repo.ListAgentSettings(ctx, tenantID, profile.ID, true)
	}
	operatorUserID := firstNonEmpty(profile.UpdatedBy, profile.CreatedBy, profile.UserID, userID)
	plan, source, modelID := s.inferAgentCapabilityPlan(ctx, profile)
	settings, err := buildInferredAgentSettings(
		tenantID,
		operatorUserID,
		profile.ID,
		s.ListAgentTemplates(ctx),
		plan,
		source,
		modelID,
		hash,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceAgentSettings(ctx, tenantID, profile.ID, settings); err != nil {
		return nil, err
	}
	return s.repo.ListAgentSettings(ctx, tenantID, profile.ID, true)
}

func shouldReplaceAgentSettingsFromProfile(settings []*types.WorkProfileAgentSetting, hash string) bool {
	if len(settings) == 0 {
		return true
	}
	enabledCount := 0
	for _, setting := range settings {
		if setting == nil {
			continue
		}
		if setting.Enabled {
			enabledCount++
		}
		if asString(setting.OutputPolicy[serviceAgentSettingsSourceKey]) != "" {
			if asString(setting.OutputPolicy[serviceAgentSettingsHashKey]) != hash {
				return true
			}
		}
	}
	if enabledCount == 0 {
		return true
	}
	return false
}

func buildInferredAgentSettings(
	tenantID uint64,
	operatorUserID string,
	profileID string,
	templates []types.ServiceAgentTemplate,
	plan serviceAgentCapabilityPlan,
	source string,
	modelID string,
	hash string,
) ([]*types.WorkProfileAgentSetting, error) {
	enabledDomains := normalizedServiceAgentDomainSet(plan.EnabledDomains)
	enabledDomains[types.ServiceAgentDomainMemoryRouter] = true
	if !hasEnabledExtractionAgentDomain(enabledDomains) {
		enabledDomains[types.ServiceAgentDomainCustomerService] = true
	}
	if source == "" {
		source = serviceAgentSettingsSourceFallback
	}
	settings := make([]*types.WorkProfileAgentSetting, 0, len(templates))
	for index, tmpl := range templates {
		setting, err := buildAgentSetting(tenantID, operatorUserID, profileID, types.WorkProfileAgentSettingInput{
			AgentDomain:      tmpl.AgentDomain,
			Enabled:          enabledDomains[tmpl.AgentDomain],
			DisplayName:      tmpl.DisplayName,
			DisplayOrder:     index + 1,
			MemoryFilter:     tmpl.MemoryFilter,
			WorkDocDirectory: tmpl.WorkDocDirectory,
			SelectedSkills:   tmpl.SelectedSkills,
			OutputPolicy:     tmpl.OutputPolicy,
		}, index)
		if err != nil {
			return nil, err
		}
		setting.OutputPolicy = mergeJSONMap(setting.OutputPolicy, types.JSONMap{
			serviceAgentSettingsSourceKey: source,
			serviceAgentSettingsHashKey:   hash,
			serviceAgentSettingsReasonKey: trimMax(plan.Reason, serviceMaxTitleLength),
		})
		if modelID != "" {
			setting.OutputPolicy[serviceAgentSettingsModelKey] = modelID
		}
		settings = append(settings, setting)
	}
	return settings, nil
}

func (s *serviceService) inferAgentCapabilityPlan(
	ctx context.Context,
	profile *types.UserWorkProfile,
) (serviceAgentCapabilityPlan, string, string) {
	description := workProfileAgentInferenceText(profile)
	if strings.TrimSpace(description) == "" {
		return fallbackServiceAgentCapabilityPlan(profile), serviceAgentSettingsSourceFallback, ""
	}
	if modelID := s.defaultServiceChatModelID(ctx); modelID != "" {
		if plan, err := s.generateAgentCapabilityPlanWithModel(ctx, modelID, profile, description); err == nil {
			if normalized := normalizeServiceAgentCapabilityPlan(plan); len(normalized.EnabledDomains) > 0 {
				return normalized, serviceAgentSettingsSourceLLM, modelID
			}
		} else {
			logger.Warnf(ctx, "service agent capability inference failed: model=%s err=%v", modelID, err)
		}
	}
	return fallbackServiceAgentCapabilityPlan(profile), serviceAgentSettingsSourceFallback, ""
}

func (s *serviceService) generateAgentCapabilityPlanWithModel(
	ctx context.Context,
	modelID string,
	profile *types.UserWorkProfile,
	description string,
) (serviceAgentCapabilityPlan, error) {
	if s.modelService == nil {
		return serviceAgentCapabilityPlan{}, errors.New("model service is not configured")
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return serviceAgentCapabilityPlan{}, fmt.Errorf("get chat model: %w", err)
	}
	tmpls := s.ListAgentTemplates(ctx)
	prompt := buildServiceAgentCapabilityPrompt(profile, description, tmpls)
	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是服务能力配置助手。你只根据成员分身描述判断应启用哪些服务能力，不输出 Markdown，不输出多余解释。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}, &chat.ChatOptions{
		Temperature: 0.15,
		MaxTokens:   800,
		Thinking:    &thinking,
		Format:      serviceAgentCapabilityPlanSchema,
	})
	if err != nil {
		return serviceAgentCapabilityPlan{}, err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return serviceAgentCapabilityPlan{}, errors.New("empty plan response")
	}
	return parseServiceAgentCapabilityPlan(resp.Content)
}

func parseServiceAgentCapabilityPlan(raw string) (serviceAgentCapabilityPlan, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	var plan serviceAgentCapabilityPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return serviceAgentCapabilityPlan{}, err
	}
	return normalizeServiceAgentCapabilityPlan(plan), nil
}

func normalizeServiceAgentCapabilityPlan(plan serviceAgentCapabilityPlan) serviceAgentCapabilityPlan {
	seen := map[string]bool{}
	enabled := make([]string, 0, len(plan.EnabledDomains))
	for _, domain := range plan.EnabledDomains {
		domain = strings.TrimSpace(domain)
		if !types.IsValidServiceAgentDomain(domain) || seen[domain] {
			continue
		}
		seen[domain] = true
		enabled = append(enabled, domain)
	}
	if !seen[types.ServiceAgentDomainMemoryRouter] {
		enabled = append([]string{types.ServiceAgentDomainMemoryRouter}, enabled...)
	}
	if len(enabled) > 4 {
		enabled = enabled[:4]
	}
	plan.EnabledDomains = enabled
	plan.Reason = trimMax(strings.TrimSpace(plan.Reason), serviceMaxTitleLength)
	return plan
}

func fallbackServiceAgentCapabilityPlan(profile *types.UserWorkProfile) serviceAgentCapabilityPlan {
	description := workProfileAgentInferenceText(profile)
	enabled := []string{types.ServiceAgentDomainMemoryRouter}
	switch {
	case containsAny(description, []string{"排课", "调课", "请假", "补课", "老师", "教室", "教务"}):
		enabled = append(enabled, types.ServiceAgentDomainScheduling, types.ServiceAgentDomainCustomerService)
	case containsAny(description, []string{"投诉", "退费", "退款", "不满", "售后", "闭环", "反馈"}):
		enabled = append(enabled, types.ServiceAgentDomainCustomerService, types.ServiceAgentDomainAfterSaleRisk)
	case containsAny(description, []string{"试听", "体验课", "咨询", "报名", "邀约", "招生", "顾问"}):
		enabled = append(enabled, types.ServiceAgentDomainLeadIntake, types.ServiceAgentDomainSalesConsulting, types.ServiceAgentDomainCustomerService)
	case containsAny(description, []string{"续费", "续课", "到期", "课次"}):
		enabled = append(enabled, types.ServiceAgentDomainCustomerService)
	case containsAny(description, []string{"日报", "复盘", "经营", "总结"}):
		enabled = append(enabled, types.ServiceAgentDomainDailyReview, types.ServiceAgentDomainCustomerService)
	default:
		enabled = append(enabled, types.ServiceAgentDomainCustomerService)
	}
	return normalizeServiceAgentCapabilityPlan(serviceAgentCapabilityPlan{
		EnabledDomains: enabled,
		Reason:         "基于分身描述规则兜底自动开启服务能力",
	})
}

func buildServiceAgentCapabilityPrompt(
	profile *types.UserWorkProfile,
	description string,
	templates []types.ServiceAgentTemplate,
) string {
	var b strings.Builder
	b.WriteString("分身信息：\n")
	if profile != nil {
		b.WriteString("姓名/名称：")
		b.WriteString(firstNonEmpty(profile.Name, "未命名分身"))
		b.WriteString("\n")
		if profile.RoleType != "" {
			b.WriteString("岗位类型：")
			b.WriteString(profile.RoleType)
			b.WriteString("\n")
		}
		if profile.MemoryScope != "" {
			b.WriteString("记忆范围：")
			b.WriteString(profile.MemoryScope)
			b.WriteString("\n")
		}
		if profile.TonePreference != "" {
			b.WriteString("沟通风格：")
			b.WriteString(profile.TonePreference)
			b.WriteString("\n")
		}
	}
	b.WriteString("分身描述：")
	b.WriteString(description)
	b.WriteString("\n\n可选服务能力：\n")
	for _, tmpl := range templates {
		b.WriteString("- ")
		b.WriteString(tmpl.AgentDomain)
		b.WriteString("：")
		b.WriteString(tmpl.Description)
		b.WriteString("\n")
	}
	b.WriteString("\n要求：\n")
	b.WriteString("1. 只能从可选服务能力里选择，不要新增。\n")
	b.WriteString("2. 只开启与岗位职责直接相关的能力。\n")
	b.WriteString("3. 至少保留 memory_router；如果描述明显服务客户/家长/学员，还要开启 customer_service。\n")
	b.WriteString("4. 输出 enabled_domains 数组，按优先级从高到低排列。\n")
	return b.String()
}

func (s *serviceService) defaultServiceChatModelID(ctx context.Context) string {
	if s.modelService == nil {
		return ""
	}
	if _, ok := types.TenantIDFromContext(ctx); !ok {
		return ""
	}
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return ""
	}
	var fallback string
	for _, model := range models {
		if model == nil || model.Status != types.ModelStatusActive || model.Type != types.ModelTypeKnowledgeQA {
			continue
		}
		if model.IsDefault {
			return model.ID
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	return fallback
}

func workProfileAgentInferenceText(profile *types.UserWorkProfile) string {
	if profile == nil {
		return ""
	}
	description := strings.TrimSpace(profile.WorkProfileDescription)
	if description == "" {
		return ""
	}
	parts := []string{
		trimMax(strings.TrimSpace(profile.Name), 120),
		trimMax(strings.TrimSpace(profile.RoleType), 80),
		trimMax(description, 1500),
		trimMax(strings.TrimSpace(profile.MemoryScope), 160),
		trimMax(strings.TrimSpace(profile.TonePreference), 160),
		strings.Join(cleanStringArray([]string(profile.CampusScope), 10, 80), "、"),
		strings.Join(cleanStringArray([]string(profile.CourseScope), 10, 80), "、"),
	}
	return strings.Join(cleanStringArray(parts, 10, 256), "\n")
}

func workProfileAgentSettingsHash(profile *types.UserWorkProfile) string {
	if profile == nil {
		return ""
	}
	return deterministicServiceID(
		"service-agent-settings",
		strconv.FormatUint(profile.TenantID, 10),
		profile.UserID,
		strings.TrimSpace(profile.Name),
		strings.TrimSpace(profile.RoleType),
		strings.Join([]string(profile.CampusScope), "|"),
		strings.Join([]string(profile.CourseScope), "|"),
		strings.TrimSpace(profile.MemoryScope),
		strings.TrimSpace(profile.TonePreference),
		strings.TrimSpace(profile.WorkProfileDescription),
	)
}

func normalizedServiceAgentDomainSet(domains []string) map[string]bool {
	out := make(map[string]bool, len(domains))
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if types.IsValidServiceAgentDomain(domain) {
			out[domain] = true
		}
	}
	return out
}

func hasEnabledExtractionAgentDomain(domains map[string]bool) bool {
	if domains[types.ServiceAgentDomainSalesConsulting] || domains[types.ServiceAgentDomainLeadIntake] {
		return true
	}
	return domains[types.ServiceAgentDomainCustomerService] || domains[types.ServiceAgentDomainScheduling] || domains[types.ServiceAgentDomainAfterSaleRisk]
}

func memoryHasCustomerIdentity(memory *types.OrganizeMemory) bool {
	if memory == nil {
		return false
	}
	if extractCustomerName(memory) != "" {
		return true
	}
	if firstMetadataString(memory.Metadata, "student_name", "studentName", "learner_name", "learnerName", "child_name", "childName") != "" {
		return true
	}
	text := memorySearchText(memory)
	return regexp.MustCompile(`(?:家长|妈妈|爸爸|学员|学生|孩子)[:：]`).MatchString(text)
}

func isCustomerFacingStage(stage string) bool {
	switch strings.TrimSpace(stage) {
	case "客户摘要", "售前试听", "报名确认", "续费服务", "在园服务", "转介绍", "客户跟进", "排课调课":
		return true
	default:
		return false
	}
}

func genericServiceModeFromText(text string) string {
	switch {
	case containsAny(text, []string{"投资", "财报", "估值", "研究", "分析", "IDM", "GaN", "半导体", "龙头"}):
		return "投资分析"
	case containsAny(text, []string{"培训", "教研", "讲师", "签到", "资料", "大纲"}):
		return "培训准备"
	case containsAny(text, []string{"会议", "纪要", "议程", "参会", "汇报"}):
		return "会议纪要"
	case containsAny(text, []string{"项目", "方案", "计划", "事项", "推进"}):
		return "事项整理"
	default:
		return "通用服务"
	}
}

func genericNextActionForServiceMode(mode string) string {
	switch mode {
	case "投资分析":
		return "先提炼核心结论、风险点和下一步跟踪项。"
	case "培训准备":
		return "先整理培训清单，再确认负责人、讲师和交付时间。"
	case "会议纪要":
		return "先补齐议题、结论和待办。"
	case "事项整理":
		return "先把记忆中的事实转成可确认事项。"
	default:
		return "先把记忆整理成可执行事项，再确认下一步。"
	}
}

func genericPrimaryActionForServiceMode(mode string) string {
	switch mode {
	case "投资分析":
		return "先把文档里的投资判断、依据和后续动作分开整理。"
	case "培训准备":
		return "先把培训流程、资料和资源需求拆成清单。"
	case "会议纪要":
		return "先沉淀会议结论，再落到待办和责任人。"
	case "事项整理":
		return "先把记忆中的关键事实归并为可执行条目。"
	default:
		return "先确认记忆里的真实事实，再生成可执行内容。"
	}
}

func genericReplyDraftForServiceMode(mode, subject string) string {
	switch mode {
	case "投资分析":
		return fmt.Sprintf("我先把%s整理成投资分析要点和后续跟踪项。", firstNonEmpty(subject, "这条记忆"))
	case "培训准备":
		return fmt.Sprintf("我先把%s整理成培训准备清单，并补齐下一步安排。", firstNonEmpty(subject, "这条记忆"))
	case "会议纪要":
		return fmt.Sprintf("我先把%s整理成会议纪要和待办事项。", firstNonEmpty(subject, "这条记忆"))
	case "事项整理":
		return fmt.Sprintf("我先把%s整理成可确认的事项清单。", firstNonEmpty(subject, "这条记忆"))
	default:
		return fmt.Sprintf("我先把%s整理成待确认事项。", firstNonEmpty(subject, "这条记忆"))
	}
}

func genericRiskLabelForServiceMode(mode string) string {
	if mode == "投资分析" {
		return "投资风险待确认"
	}
	return "待判断"
}

func nonCustomerSubjectName(plan serviceMemoryExtractionPlan, memory *types.OrganizeMemory, text string) string {
	memoryTitle := ""
	if memory != nil {
		memoryTitle = trimMax(memory.Title, serviceMaxShortText)
	}
	modelCustomerName := trimMax(plan.CustomerName, serviceMaxShortText)
	for _, raw := range []string{plan.SubjectName, plan.Title} {
		candidate := trimMax(raw, serviceMaxShortText)
		if candidate == "" || isCustomerFacingGeneratedText(candidate, modelCustomerName, text) {
			continue
		}
		if memoryTitle == "" || strings.Contains(text, candidate) || strings.Contains(memoryTitle, candidate) || strings.Contains(candidate, memoryTitle) {
			return candidate
		}
	}
	return firstNonEmpty(memoryTitle, genericServiceModeFromText(text), "服务事项")
}

func isCustomerFacingGeneratedText(value, modelCustomerName, memoryText string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	modelCustomerName = strings.TrimSpace(modelCustomerName)
	if modelCustomerName != "" && !strings.Contains(memoryText, modelCustomerName) && strings.Contains(value, modelCustomerName) {
		return true
	}
	return regexp.MustCompile(`客户摘要|客户跟进|客户状态|客户服务|客户事实|售前试听|试听|体验课|报名确认|报名|招生|续费服务|在园服务|转介绍|价格顾虑|适应焦虑|续费窗口|售后风险|家长|妈妈|爸爸|孩子|学员|续课|退费`).MatchString(value)
}

func hasCustomerFacingGeneratedText(values []string, modelCustomerName, memoryText string) bool {
	for _, value := range values {
		if isCustomerFacingGeneratedText(value, modelCustomerName, memoryText) {
			return true
		}
	}
	return false
}

func withServiceTenantContext(ctx context.Context, tenantID uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tenantID == 0 {
		return ctx
	}
	if _, ok := types.TenantIDFromContext(ctx); ok {
		return ctx
	}
	return context.WithValue(ctx, types.TenantIDContextKey, tenantID)
}

func (s *serviceService) extractMemoryWithGeneratedService(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
	fallbackReason string,
) (*types.ServiceMemoryExtraction, error) {
	reminder, reason, err := s.upsertModelGeneratedServiceMemory(
		ctx,
		tenantID,
		userID,
		profile,
		enabledDomains,
		memory,
		fallbackReason,
	)
	if err != nil {
		return nil, err
	}
	return &types.ServiceMemoryExtraction{
		MemoryID:  memory.ID,
		Generated: reminder != nil,
		Reason:    reason,
		Reminder:  reminder,
	}, nil
}

func (s *serviceService) upsertModelGeneratedServiceMemory(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
	fallbackReason string,
) (*types.ServiceReminder, string, error) {
	plan, source, modelID := s.inferServiceMemoryExtractionPlan(ctx, profile, enabledDomains, memory)
	if !plan.ShouldGenerate {
		return nil, firstNonEmpty(plan.Reason, fallbackReason, "memory_not_relevant"), nil
	}
	setting := s.generatedServiceAgentSetting(ctx, tenantID, userID, profile, enabledDomains, plan, source, modelID)
	if setting == nil {
		return nil, firstNonEmpty(fallbackReason, "agent_not_enabled"), nil
	}
	reminder := buildReminderFromServiceMemoryPlan(tenantID, userID, profile, memory, plan, setting, source, modelID)
	subjectName := firstNonEmpty(
		asString(reminder.Metadata["customer_name"]),
		asString(reminder.Metadata["subject_name"]),
		reminder.Title,
		asString(reminder.Metadata["service_mode"]),
	)
	subject := buildServiceSubject(
		tenantID,
		userID,
		subjectName,
		asString(reminder.Metadata["student_name"]),
		reminder.Confidence,
	)
	if err := s.repo.UpsertSubject(ctx, subject); err != nil {
		return nil, "", err
	}
	reminder.SubjectID = subject.ID
	reminder.ProfileID = profile.ID
	reminder.Metadata = mergeJSONMap(reminder.Metadata, types.JSONMap{
		"subject_id":         subject.ID,
		"work_doc_directory": setting.WorkDocDirectory,
		"agent_display_name": setting.DisplayName,
	})
	if err := s.repo.UpsertReminder(ctx, reminder); err != nil {
		return nil, "", err
	}
	for _, doc := range buildWorkDocs(tenantID, userID, profile.ID, subject.ID, reminder, []*types.OrganizeMemory{memory}) {
		links := buildWorkDocLinks(doc, reminder, []*types.OrganizeMemory{memory})
		if err := s.repo.UpsertWorkDocWithLinks(ctx, doc, links); err != nil {
			return nil, "", err
		}
	}
	persisted, err := s.repo.GetReminder(ctx, tenantID, userID, reminder.ID)
	if err != nil {
		return nil, "", err
	}
	return persisted, "generated", nil
}

func (s *serviceService) inferServiceMemoryExtractionPlan(
	ctx context.Context,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
) (serviceMemoryExtractionPlan, string, string) {
	if strings.TrimSpace(memorySearchText(memory)) == "" {
		return serviceMemoryExtractionPlan{
			ShouldGenerate: false,
			Reason:         "记忆内容为空，无法生成服务事项",
		}, serviceMemoryExtractionSourceFallback, ""
	}
	if modelID := s.defaultServiceChatModelID(ctx); modelID != "" {
		if plan, err := s.generateServiceMemoryExtractionPlanWithModel(ctx, modelID, profile, enabledDomains, memory); err == nil {
			normalized := normalizeServiceMemoryExtractionPlan(plan, profile, enabledDomains, memory)
			if normalized.ShouldGenerate {
				return normalized, serviceMemoryExtractionSourceLLM, modelID
			}
			return normalized, serviceMemoryExtractionSourceLLM, modelID
		} else {
			logger.Warnf(ctx, "service memory extraction inference failed: model=%s memory=%s err=%v", modelID, memory.ID, err)
		}
	}
	return fallbackServiceMemoryExtractionPlan(profile, enabledDomains, memory), serviceMemoryExtractionSourceFallback, ""
}

func (s *serviceService) generateServiceMemoryExtractionPlanWithModel(
	ctx context.Context,
	modelID string,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
) (serviceMemoryExtractionPlan, error) {
	if s.modelService == nil {
		return serviceMemoryExtractionPlan{}, errors.New("model service is not configured")
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return serviceMemoryExtractionPlan{}, fmt.Errorf("get chat model: %w", err)
	}
	prompt := buildServiceMemoryExtractionPrompt(profile, memory, enabledDomains, s.ListAgentTemplates(ctx))
	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是睿乐服务助理。你根据分身描述和记忆内容生成精准服务事项；只能输出 JSON，不输出 Markdown 或解释。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}, &chat.ChatOptions{
		Temperature: 0.2,
		MaxTokens:   1200,
		Thinking:    &thinking,
		Format:      serviceMemoryExtractionPlanSchema,
	})
	if err != nil {
		return serviceMemoryExtractionPlan{}, err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return serviceMemoryExtractionPlan{}, errors.New("empty service extraction response")
	}
	return parseServiceMemoryExtractionPlan(resp.Content)
}

func parseServiceMemoryExtractionPlan(raw string) (serviceMemoryExtractionPlan, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	var plan serviceMemoryExtractionPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return serviceMemoryExtractionPlan{}, err
	}
	return plan, nil
}

func normalizeServiceMemoryExtractionPlan(
	plan serviceMemoryExtractionPlan,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
) serviceMemoryExtractionPlan {
	text := memorySearchText(memory)
	memoryCustomerName := extractCustomerName(memory)
	modelCustomerName := trimMax(plan.CustomerName, serviceMaxShortText)
	hasCustomerIdentity := memoryHasCustomerIdentity(memory)
	stage := firstNonEmpty(trimMax(plan.Stage, 128), trimMax(plan.ServiceMode, 128), inferStage(text))
	riskLabel := firstNonEmpty(trimMax(plan.RiskLabel, 128), inferRiskLabel(text))
	customerName := firstNonEmpty(memoryCustomerName, modelCustomerName)
	subjectName := firstNonEmpty(
		trimMax(plan.SubjectName, serviceMaxShortText),
		trimMax(plan.Title, serviceMaxShortText),
		trimMax(memory.Title, serviceMaxShortText),
		"服务事项",
	)
	studentName := firstNonEmpty(trimMax(plan.StudentName, serviceMaxShortText), extractStudentName(memory, customerName), "待补充")
	priority := strings.ToLower(strings.TrimSpace(plan.Priority))
	if !types.IsValidServicePriority(priority) {
		priority = inferPriority(text, riskLabel)
	}
	domain := normalizeGeneratedServiceAgentDomain(plan.AgentDomain, stage, riskLabel, text, enabledDomains)
	serviceMode := firstNonEmpty(trimMax(plan.ServiceMode, 128), stage, serviceAgentDomainLabel(domain))
	summary := firstNonEmpty(trimMax(plan.Summary, 512), serviceMemoryExcerpt(memory))
	nextAction := firstNonEmpty(trimMax(plan.NextAction, 512), inferNextAction(stage, riskLabel))
	primaryAction := firstNonEmpty(trimMax(plan.PrimaryAction, 512), inferPrimaryAction(stage, riskLabel))
	avoidAction := firstNonEmpty(trimMax(plan.AvoidAction, 512), inferAvoidAction(riskLabel))
	replyDraft := firstNonEmpty(trimMax(plan.ReplyDraft, 800), inferReplyDraft(customerName, stage, riskLabel))
	_, dueText := inferDue(plan.DueText, priority)
	if !hasCustomerIdentity {
		subjectName = nonCustomerSubjectName(plan, memory, text)
		if strings.TrimSpace(serviceMode) == "" || isCustomerFacingStage(serviceMode) || isCustomerFacingGeneratedText(serviceMode, modelCustomerName, text) {
			serviceMode = firstNonEmpty(genericServiceModeFromText(text), subjectName, "通用服务")
		}
		if strings.TrimSpace(stage) == "" || isCustomerFacingStage(stage) || isCustomerFacingGeneratedText(stage, modelCustomerName, text) {
			stage = serviceMode
		}
		if isCustomerFacingGeneratedText(riskLabel, modelCustomerName, text) {
			riskLabel = genericRiskLabelForServiceMode(serviceMode)
		}
		customerName = ""
		studentName = ""
		subjectName = firstNonEmpty(subjectName, serviceMode, trimMax(memory.Title, serviceMaxShortText))
		if strings.TrimSpace(summary) == "" || isCustomerFacingGeneratedText(summary, modelCustomerName, text) {
			summary = firstNonEmpty(contentExcerpt(memorySearchText(memory), ""), subjectName)
		}
		if strings.TrimSpace(nextAction) == "" || isCustomerFacingGeneratedText(nextAction, modelCustomerName, text) || nextAction == inferNextAction(stage, riskLabel) {
			nextAction = genericNextActionForServiceMode(serviceMode)
		}
		if strings.TrimSpace(primaryAction) == "" || isCustomerFacingGeneratedText(primaryAction, modelCustomerName, text) || primaryAction == inferPrimaryAction(stage, riskLabel) {
			primaryAction = genericPrimaryActionForServiceMode(serviceMode)
		}
		if strings.TrimSpace(avoidAction) == "" || isCustomerFacingGeneratedText(avoidAction, modelCustomerName, text) || avoidAction == inferAvoidAction(riskLabel) {
			avoidAction = "不要把未确认的内容直接当作已完成事项，先整理成可执行清单。"
		}
		if strings.TrimSpace(replyDraft) == "" || isCustomerFacingGeneratedText(replyDraft, modelCustomerName, text) || replyDraft == inferReplyDraft(customerName, stage, riskLabel) {
			replyDraft = genericReplyDraftForServiceMode(serviceMode, subjectName)
		}
		if isCustomerFacingGeneratedText(plan.Title, modelCustomerName, text) {
			plan.Title = ""
		}
		if isCustomerFacingGeneratedText(plan.AssistReason, modelCustomerName, text) {
			plan.AssistReason = ""
		}
		if strings.TrimSpace(plan.AssistReason) == "" {
			plan.AssistReason = fmt.Sprintf("基于分身%s和当前记忆自动生成%s。", firstNonEmpty(profileName(profile), "服务助理"), serviceMode)
		}
		if len(plan.SalesHighlights) == 0 || hasCustomerFacingGeneratedText(plan.SalesHighlights, modelCustomerName, text) {
			plan.SalesHighlights = []string{fmt.Sprintf("该记忆可沉淀为%s，适合整理成待确认事项。", serviceMode)}
		}
		if domain == types.ServiceAgentDomainSalesConsulting || domain == types.ServiceAgentDomainLeadIntake {
			domain = types.ServiceAgentDomainCustomerService
		}
	}
	plan.ShouldGenerate = plan.ShouldGenerate || summary != "" || nextAction != ""
	plan.AgentDomain = domain
	plan.ServiceMode = serviceMode
	plan.SubjectName = subjectName
	plan.CustomerName = customerName
	plan.StudentName = studentName
	plan.Title = firstNonEmpty(trimMax(plan.Title, serviceMaxTitleLength), subjectName, trimMax(memory.Title, serviceMaxTitleLength), serviceMode)
	plan.Summary = summary
	plan.Stage = stage
	plan.Priority = priority
	plan.DueText = dueText
	plan.RiskLabel = riskLabel
	plan.AssistReason = firstNonEmpty(trimMax(plan.AssistReason, 800), fmt.Sprintf("由%s生成服务事项：%s", firstNonEmpty(profileName(profile), "服务助理"), summary))
	plan.PrimaryAction = primaryAction
	plan.NextAction = nextAction
	plan.AvoidAction = avoidAction
	plan.ReplyDraft = replyDraft
	plan.MemorySignals = cleanStringArray(plan.MemorySignals, 8, 64)
	if len(plan.MemorySignals) == 0 {
		plan.MemorySignals = cleanStringArray(deriveMemorySignals(text, nil), 8, 64)
	}
	if len(plan.MemorySignals) == 0 {
		plan.MemorySignals = types.StringArray{serviceMode}
	}
	plan.SalesHighlights = cleanStringArray(plan.SalesHighlights, 6, 160)
	if len(plan.SalesHighlights) == 0 {
		plan.SalesHighlights = cleanStringArray(salesHighlightsFrom(stage, riskLabel, text), 6, 160)
	}
	plan.WorkDocDirectory = trimMax(plan.WorkDocDirectory, serviceMaxShortText)
	plan.Reason = trimMax(plan.Reason, serviceMaxTitleLength)
	return plan
}

func normalizeGeneratedServiceAgentDomain(
	raw string,
	stage string,
	riskLabel string,
	text string,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
) string {
	raw = strings.TrimSpace(raw)
	if types.IsValidServiceAgentDomain(raw) && raw != types.ServiceAgentDomainMemoryRouter {
		return raw
	}
	inferred := domainForText(text, stage, riskLabel)
	if types.IsValidServiceAgentDomain(inferred) && inferred != types.ServiceAgentDomainMemoryRouter {
		return inferred
	}
	for _, domain := range []string{
		types.ServiceAgentDomainSalesConsulting,
		types.ServiceAgentDomainCustomerService,
		types.ServiceAgentDomainScheduling,
		types.ServiceAgentDomainAfterSaleRisk,
		types.ServiceAgentDomainLeadIntake,
	} {
		if enabledDomains[domain] != nil {
			return domain
		}
	}
	return types.ServiceAgentDomainCustomerService
}

func fallbackServiceMemoryExtractionPlan(
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	memory *types.OrganizeMemory,
) serviceMemoryExtractionPlan {
	text := memorySearchText(memory)
	stage := inferStage(text)
	hasCustomerIdentity := memoryHasCustomerIdentity(memory)
	if !hasCustomerIdentity {
		stage = genericServiceModeFromText(text)
	} else if containsAny(text, []string{"培训", "准备", "清单", "会议", "教研", "资料", "事项"}) {
		stage = "服务事项"
	}
	riskLabel := inferRiskLabel(text)
	domain := normalizeGeneratedServiceAgentDomain("", stage, riskLabel, text, enabledDomains)
	customerName := extractCustomerName(memory)
	subjectName := firstNonEmpty(trimMax(memory.Title, serviceMaxShortText), stage, "服务事项")
	summary := serviceMemoryExcerpt(memory)
	nextAction := inferNextAction(stage, riskLabel)
	primaryAction := "先把记忆中的事实转成可确认事项，再补齐负责人、时间和交付物。"
	replyDraft := fmt.Sprintf("我先把「%s」整理成待确认事项，并补齐负责人、时间和下一步。", firstNonEmpty(memory.Title, "这条记忆"))
	if !hasCustomerIdentity {
		riskLabel = genericRiskLabelForServiceMode(stage)
		nextAction = genericNextActionForServiceMode(stage)
		primaryAction = genericPrimaryActionForServiceMode(stage)
		replyDraft = genericReplyDraftForServiceMode(stage, subjectName)
	} else if stage == "服务事项" {
		nextAction = "整理服务事项清单并确认下一步负责人和时间"
	}
	return normalizeServiceMemoryExtractionPlan(serviceMemoryExtractionPlan{
		ShouldGenerate:  true,
		AgentDomain:     domain,
		ServiceMode:     stage,
		SubjectName:     subjectName,
		CustomerName:    customerName,
		Title:           firstNonEmpty(memory.Title, stage),
		Summary:         summary,
		Stage:           stage,
		Priority:        inferPriority(text, riskLabel),
		RiskLabel:       riskLabel,
		AssistReason:    fmt.Sprintf("基于分身%s和当前记忆自动生成服务事项。", firstNonEmpty(profileName(profile), "服务助理")),
		PrimaryAction:   primaryAction,
		NextAction:      nextAction,
		AvoidAction:     "不要把模型生成内容直接当作已完成事实，先人工确认后再执行。",
		ReplyDraft:      replyDraft,
		MemorySignals:   deriveMemorySignals(text, nil),
		SalesHighlights: []string{"这条记忆可转成服务事项，适合沉淀为下一步动作。"},
		Reason:          "规则兜底生成服务模式和服务内容",
	}, profile, enabledDomains, memory)
}

func (s *serviceService) generatedServiceAgentSetting(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	plan serviceMemoryExtractionPlan,
	source string,
	modelID string,
) *types.WorkProfileAgentSetting {
	if setting, ok := serviceAgentSettingForDomain(enabledDomains, plan.AgentDomain); ok && setting != nil {
		return setting
	}
	operatorUserID := userID
	if profile != nil {
		operatorUserID = firstNonEmpty(profile.UpdatedBy, profile.CreatedBy, profile.UserID, userID)
	}
	tmpl := serviceAgentTemplateForDomain(s.ListAgentTemplates(ctx), plan.AgentDomain)
	displayName := firstNonEmpty(plan.ServiceMode, serviceAgentDomainLabel(plan.AgentDomain))
	workDocDirectory := firstNonEmpty(plan.WorkDocDirectory, generatedServiceWorkDocDirectory(plan), defaultWorkDocDirectory(plan.AgentDomain))
	setting, err := buildAgentSetting(tenantID, operatorUserID, profileID(profile), types.WorkProfileAgentSettingInput{
		AgentDomain:      plan.AgentDomain,
		Enabled:          true,
		DisplayName:      displayName,
		WorkDocDirectory: workDocDirectory,
		MemoryFilter:     tmpl.MemoryFilter,
		SelectedSkills:   tmpl.SelectedSkills,
		OutputPolicy: mergeJSONMap(tmpl.OutputPolicy, types.JSONMap{
			"service_mode":               plan.ServiceMode,
			"generated_service_source":   source,
			"generated_service_model_id": modelID,
		}),
	}, 0)
	if err != nil {
		return nil
	}
	return setting
}

func serviceAgentTemplateForDomain(templates []types.ServiceAgentTemplate, domain string) types.ServiceAgentTemplate {
	for _, tmpl := range templates {
		if tmpl.AgentDomain == domain {
			return tmpl
		}
	}
	return types.ServiceAgentTemplate{}
}

func generatedServiceWorkDocDirectory(plan serviceMemoryExtractionPlan) string {
	mode := strings.TrimSpace(plan.ServiceMode)
	if mode == "" {
		return ""
	}
	return "服务/" + sanitizePathSegment(mode) + "/"
}

func buildReminderFromServiceMemoryPlan(
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	memory *types.OrganizeMemory,
	plan serviceMemoryExtractionPlan,
	setting *types.WorkProfileAgentSetting,
	source string,
	modelID string,
) *types.ServiceReminder {
	dueAt, dueText := inferDue(plan.DueText, plan.Priority)
	lastMemoryAt := memory.UpdatedAt
	if !memory.OccurredAt.IsZero() {
		lastMemoryAt = memory.OccurredAt
	}
	sourceMemoryIDs := cleanStringArray([]string{memory.ID}, 100, 64)
	confidence := 0.78
	if source == serviceMemoryExtractionSourceLLM {
		confidence = 0.84
	}
	displayName := firstNonEmpty(plan.SubjectName, plan.CustomerName, plan.Title, plan.ServiceMode, "服务事项")
	subjectKey := serviceSubjectKey(plan.CustomerName, plan.StudentName)
	if subjectKey == "" {
		subjectKey = serviceSubjectKey(plan.SubjectName, memory.ID)
	}
	reminderID := deterministicServiceID("service-reminder", profileID(profile), subjectKey, plan.AgentDomain, memory.ID)
	metadata := types.JSONMap{
		"customer_name":              plan.CustomerName,
		"student_name":               plan.StudentName,
		"subject_name":               plan.SubjectName,
		"service_mode":               plan.ServiceMode,
		"memory_scope":               profileMemoryScope(profile),
		"source_type":                "memory",
		"generated_service_source":   source,
		"generated_service_model_id": modelID,
		"generated_service_reason":   plan.Reason,
		"work_doc_directory":         setting.WorkDocDirectory,
		"agent_display_name":         setting.DisplayName,
	}
	return &types.ServiceReminder{
		ID:                reminderID,
		TenantID:          tenantID,
		UserID:            userID,
		ProfileID:         profileID(profile),
		AgentDomain:       plan.AgentDomain,
		Title:             plan.Title,
		Summary:           plan.Summary,
		Status:            types.ServiceReminderStatusPending,
		Priority:          plan.Priority,
		DueAt:             dueAt,
		DueText:           dueText,
		Stage:             plan.Stage,
		Channel:           inferChannel(memorySearchText(memory)),
		DecisionRole:      inferDecisionRole(memorySearchText(memory)),
		RiskLabel:         plan.RiskLabel,
		AssistReason:      plan.AssistReason,
		PrimaryAction:     plan.PrimaryAction,
		NextAction:        plan.NextAction,
		AvoidAction:       plan.AvoidAction,
		ContextItems:      cleanStringArray(append([]string{"个人记忆", plan.ServiceMode}, plan.MemorySignals...), 10, 64),
		MemorySignals:     cleanStringArray(plan.MemorySignals, 10, 64),
		SourceMemoryIDs:   sourceMemoryIDs,
		SourceMemoryCount: len(sourceMemoryIDs),
		LastMemoryAt:      &lastMemoryAt,
		Confidence:        confidence,
		SalesHighlights:   cleanStringArray(plan.SalesHighlights, 6, 160),
		WriteBackStatus:   "待确认",
		WriteBackDraft:    fmt.Sprintf("%s｜%s。依据：%s", displayName, plan.NextAction, firstNonEmpty(memory.Title, "当前记忆")),
		ReplyDraft:        plan.ReplyDraft,
		Metadata:          metadata,
		UpdatedAt:         time.Now().UTC(),
	}
}

func buildServiceMemoryExtractionPrompt(
	profile *types.UserWorkProfile,
	memory *types.OrganizeMemory,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	templates []types.ServiceAgentTemplate,
) string {
	var b strings.Builder
	hasCustomerIdentity := memoryHasCustomerIdentity(memory)
	customerName := extractCustomerName(memory)
	b.WriteString("分身描述：\n")
	b.WriteString(firstNonEmpty(workProfileAgentInferenceText(profile), profileName(profile), "未配置分身描述"))
	b.WriteString("\n\n已启用服务能力：\n")
	enabledLines := enabledServiceSettingPromptLines(enabledDomains)
	if len(enabledLines) == 0 {
		b.WriteString("- 无业务服务能力，需要根据记忆自动生成服务模式和内容。\n")
	} else {
		b.WriteString(strings.Join(enabledLines, "\n"))
		b.WriteString("\n")
	}
	b.WriteString("\n可用内置存储域：\n")
	for _, tmpl := range templates {
		if tmpl.AgentDomain == types.ServiceAgentDomainMemoryRouter {
			continue
		}
		b.WriteString("- ")
		b.WriteString(tmpl.AgentDomain)
		b.WriteString("：")
		b.WriteString(tmpl.Description)
		b.WriteString("\n")
	}
	b.WriteString("\n当前记忆：\n")
	b.WriteString("标题：")
	b.WriteString(firstNonEmpty(memory.Title, "未命名记忆"))
	b.WriteString("\n来源：")
	b.WriteString(firstNonEmpty(memory.Source, "个人记忆"))
	b.WriteString("\n内容：")
	b.WriteString(trimMax(memorySearchText(memory), 2000))
	if len(memory.Metadata) > 0 {
		if raw, err := json.Marshal(memory.Metadata); err == nil {
			b.WriteString("\n元数据：")
			b.WriteString(trimMax(string(raw), 1000))
		}
	}
	b.WriteString("\n客户身份判断：")
	if hasCustomerIdentity {
		b.WriteString("当前记忆检测到明确客户/家长/学员身份")
		if customerName != "" {
			b.WriteString("：")
			b.WriteString(customerName)
		}
		b.WriteString("。")
	} else {
		b.WriteString("当前记忆没有明确客户/家长/学员身份。")
	}
	b.WriteString("\n\n要求：\n")
	b.WriteString("1. 如果已启用能力能精准覆盖当前记忆，agent_domain 选择对应能力。\n")
	b.WriteString("2. 如果没有配置对应能力，不要拒绝；生成 service_mode 表示新的服务模式，并把 agent_domain 映射到最接近的内置存储域。\n")
	b.WriteString("3. customer_name/student_name 只能填写当前记忆中明确出现的客户、家长、学员或孩子身份；没有明确身份时必须留空，禁止编造“小明妈妈”等示例人物。\n")
	b.WriteString("4. subject_name 用当前记忆的真实主题、项目、公司、文档名或事项对象；非客户类记忆必须用 subject_name 承载对象，不要把对象写进 customer_name。\n")
	b.WriteString("5. 如果记忆是投资、研报、财报、估值、公司分析、股票跟踪类内容，service_mode 应围绕投研分析，summary/primary_action/next_action/reply_draft 必须聚焦投资结论、依据、风险和后续跟踪项，不能输出客户摘要、售前试听或招生销售话术。\n")
	b.WriteString("6. next_action、primary_action、reply_draft 必须是可执行服务内容，且只能基于当前记忆事实。\n")
	b.WriteString("7. 只有记忆完全没有可行动信息时，should_generate 才能为 false。\n")
	return b.String()
}

func enabledServiceSettingPromptLines(enabledDomains map[string]*types.WorkProfileAgentSetting) []string {
	keys := make([]string, 0, len(enabledDomains))
	for domain := range enabledDomains {
		if domain != types.ServiceAgentDomainMemoryRouter {
			keys = append(keys, domain)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, domain := range keys {
		setting := enabledDomains[domain]
		if setting == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s：%s，工作目录：%s", domain, setting.DisplayName, setting.WorkDocDirectory))
	}
	return lines
}

func profileName(profile *types.UserWorkProfile) string {
	if profile == nil {
		return ""
	}
	return strings.TrimSpace(profile.Name)
}

func profileMemoryScope(profile *types.UserWorkProfile) string {
	if profile == nil {
		return ""
	}
	return strings.TrimSpace(profile.MemoryScope)
}

func (s *serviceService) hydrateWorkProfileDescription(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
) error {
	if profile == nil {
		return nil
	}
	return s.hydrateWorkProfileDescriptions(ctx, tenantID, firstNonEmpty(profile.UserID, userID), []*types.UserWorkProfile{profile})
}

func (s *serviceService) hydrateWorkProfileDescriptions(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profiles []*types.UserWorkProfile,
) error {
	if s.members == nil || len(profiles) == 0 {
		return nil
	}
	userID = strings.TrimSpace(userID)
	descriptions := map[string]string{}
	if userID != "" {
		member, err := s.members.Get(ctx, userID, tenantID)
		if err != nil {
			return err
		}
		if member != nil {
			descriptions[member.UserID] = strings.TrimSpace(member.WorkProfileDescription)
		}
	} else {
		members, err := s.members.ListByTenant(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member == nil {
				continue
			}
			descriptions[member.UserID] = strings.TrimSpace(member.WorkProfileDescription)
		}
	}
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		profile.WorkProfileDescription = descriptions[profile.UserID]
	}
	return nil
}

func (s *serviceService) materializeWorkProfilesFromMembers(ctx context.Context, tenantID uint64, userID string) error {
	if s.members == nil {
		return nil
	}
	if userID != "" {
		_, err := s.ensureDefaultWorkProfileFromMember(ctx, tenantID, userID)
		return err
	}
	members, err := s.members.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member == nil || member.Status != types.TenantMemberStatusActive {
			continue
		}
		if strings.TrimSpace(member.WorkProfileDescription) == "" {
			continue
		}
		if _, err := s.ensureDefaultWorkProfileFromMember(ctx, tenantID, member.UserID); err != nil {
			return err
		}
	}
	return nil
}

func (s *serviceService) ensureDefaultWorkProfileFromMember(
	ctx context.Context,
	tenantID uint64,
	userID string,
) (*types.UserWorkProfile, error) {
	if s.members == nil {
		return nil, nil
	}
	userID = strings.TrimSpace(userID)
	if tenantID == 0 || userID == "" {
		return nil, nil
	}
	member, err := s.members.Get(ctx, userID, tenantID)
	if err != nil || member == nil {
		return nil, err
	}
	if member.Status != types.TenantMemberStatusActive || strings.TrimSpace(member.WorkProfileDescription) == "" {
		return nil, nil
	}
	profiles, err := s.repo.ListWorkProfiles(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		if profile != nil && profile.DefaultProfile {
			if profile.Enabled && profile.State == types.ServiceWorkProfileStateEnabled {
				return profile, nil
			}
			return nil, nil
		}
	}
	profile := buildWorkProfileFromMemberDescription(ctx, member)
	if err := s.repo.CreateWorkProfile(ctx, profile); err != nil {
		if isDuplicateServiceConfig(err) {
			return s.repo.GetDefaultWorkProfile(ctx, tenantID, userID)
		}
		return nil, err
	}
	return s.repo.GetDefaultWorkProfile(ctx, tenantID, userID)
}

func (s *serviceService) resolveCustomerSpaceProfile(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profileID string,
) (*types.UserWorkProfile, []*types.WorkProfileAgentSetting, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return s.defaultProfileAndSettings(ctx, tenantID, userID)
	}
	profile, err := s.repo.GetWorkProfile(ctx, tenantID, profileID)
	if err != nil || profile == nil {
		return profile, nil, err
	}
	if profile.UserID != strings.TrimSpace(userID) {
		return nil, nil, ErrServiceNotFound
	}
	if !profile.Enabled || profile.State != types.ServiceWorkProfileStateEnabled {
		return nil, nil, nil
	}
	settings, err := s.repo.ListAgentSettings(ctx, tenantID, profile.ID, true)
	if err != nil {
		return nil, nil, err
	}
	return profile, settings, nil
}

func (s *serviceService) customerSpaceMemoryEvidence(
	ctx context.Context,
	tenantID uint64,
	userID string,
	ids []string,
) ([]types.ServiceMemoryEvidence, error) {
	ids = []string(cleanStringArray(ids, 100, 64))
	if len(ids) > 12 {
		ids = ids[:12]
	}
	evidence := make([]types.ServiceMemoryEvidence, 0, len(ids))
	for _, id := range ids {
		memory, err := s.organize.GetMemory(ctx, tenantID, userID, id)
		if err != nil {
			return nil, err
		}
		if memory == nil {
			continue
		}
		evidence = append(evidence, types.ServiceMemoryEvidence{
			ID:              memory.ID,
			Title:           memory.Title,
			Summary:         serviceMemoryExcerpt(memory),
			SourceLabel:     firstNonEmpty(memory.Source, "个人记忆"),
			OccurredAtLabel: formatMonthDay(memory.OccurredAt),
		})
	}
	return evidence, nil
}

func profileID(profile *types.UserWorkProfile) string {
	if profile == nil {
		return ""
	}
	return profile.ID
}

func (s *serviceService) refreshFromMemories(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	settings []*types.WorkProfileAgentSetting,
) error {
	enabledDomains := enabledServiceAgentSettings(settings)
	if len(enabledDomains) == 0 {
		return nil
	}
	memories, _, err := s.organize.ListMemories(ctx, types.OrganizeListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PageSize: serviceMaxPageSize,
	})
	if err != nil {
		return err
	}
	serviceMemories := make([]*types.OrganizeMemory, 0, len(memories))
	for _, memory := range memories {
		if isServiceMemory(memory) {
			serviceMemories = append(serviceMemories, memory)
			s.annotateMemoryRoute(ctx, memory)
		}
	}
	buckets := bucketServiceMemories(serviceMemories)
	for subjectName, groupedMemories := range buckets {
		if _, _, err := s.upsertServiceMemoryGroup(ctx, tenantID, userID, profile, enabledDomains, subjectName, groupedMemories); err != nil {
			return err
		}
	}
	return nil
}

func enabledServiceAgentSettings(settings []*types.WorkProfileAgentSetting) map[string]*types.WorkProfileAgentSetting {
	enabledDomains := map[string]*types.WorkProfileAgentSetting{}
	for _, setting := range settings {
		if setting.Enabled && types.IsValidServiceAgentDomain(setting.AgentDomain) {
			enabledDomains[setting.AgentDomain] = setting
		}
	}
	return enabledDomains
}

func serviceAgentSettingForDomain(
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	domain string,
) (*types.WorkProfileAgentSetting, bool) {
	setting, ok := enabledDomains[domain]
	if ok {
		return setting, true
	}
	// Sales-consulting and lead-intake are close enough that either setting
	// may route a first-contact sales reminder in early pilots.
	if domain == types.ServiceAgentDomainSalesConsulting {
		setting, ok = enabledDomains[types.ServiceAgentDomainLeadIntake]
		return setting, ok
	}
	if domain == types.ServiceAgentDomainLeadIntake {
		setting, ok = enabledDomains[types.ServiceAgentDomainSalesConsulting]
		return setting, ok
	}
	return nil, false
}

func (s *serviceService) upsertServiceMemoryGroup(
	ctx context.Context,
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	enabledDomains map[string]*types.WorkProfileAgentSetting,
	subjectName string,
	groupedMemories []*types.OrganizeMemory,
) (*types.ServiceReminder, string, error) {
	if profile == nil || len(groupedMemories) == 0 {
		return nil, "memory_not_relevant", nil
	}
	sort.SliceStable(groupedMemories, func(i, j int) bool {
		return groupedMemories[i].OccurredAt.After(groupedMemories[j].OccurredAt)
	})
	reminder := buildReminderFromMemories(tenantID, userID, profile, subjectName, groupedMemories)
	setting, ok := serviceAgentSettingForDomain(enabledDomains, reminder.AgentDomain)
	if !ok {
		return nil, "agent_not_enabled", nil
	}
	subject := buildServiceSubject(tenantID, userID, subjectName, asString(reminder.Metadata["student_name"]), reminder.Confidence)
	if err := s.repo.UpsertSubject(ctx, subject); err != nil {
		return nil, "", err
	}
	reminder.SubjectID = subject.ID
	reminder.ProfileID = profile.ID
	reminder.Metadata = mergeJSONMap(reminder.Metadata, types.JSONMap{
		"work_doc_directory": setting.WorkDocDirectory,
		"agent_display_name": setting.DisplayName,
	})
	if err := s.repo.UpsertReminder(ctx, reminder); err != nil {
		return nil, "", err
	}
	for _, doc := range buildWorkDocs(tenantID, userID, profile.ID, subject.ID, reminder, groupedMemories) {
		links := buildWorkDocLinks(doc, reminder, groupedMemories)
		if err := s.repo.UpsertWorkDocWithLinks(ctx, doc, links); err != nil {
			return nil, "", err
		}
	}
	persisted, err := s.repo.GetReminder(ctx, tenantID, userID, reminder.ID)
	if err != nil {
		return nil, "", err
	}
	return persisted, "generated", nil
}

func (s *serviceService) annotateMemoryRoute(ctx context.Context, memory *types.OrganizeMemory) {
	if memory == nil {
		return
	}
	metadata := mergeJSONMap(memory.Metadata, types.JSONMap{
		"agent_domains": []string{domainForText(memorySearchText(memory), inferStage(memorySearchText(memory)), inferRiskLabel(memorySearchText(memory)))},
		"service_subject": types.JSONMap{
			"display_name": extractCustomerName(memory),
			"student_name": extractStudentName(memory, extractCustomerName(memory)),
		},
		"source_evidence": types.JSONMap{
			"memory_id": memory.ID,
			"source":    firstNonEmpty(memory.Source, "个人记忆"),
		},
	})
	memory.Metadata = metadata
	memory.UpdatedAt = time.Now().UTC()
	_ = s.organize.UpdateMemory(ctx, memory)
}

func buildWorkProfile(tenantID uint64, operatorUserID, id string, input types.ServiceWorkProfileInput) (*types.UserWorkProfile, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		userID = strings.TrimSpace(operatorUserID)
	}
	name := trimMax(input.Name, serviceMaxShortText)
	if name == "" {
		return nil, ErrServiceProfileNameRequired
	}
	state := strings.TrimSpace(input.State)
	if state == "" {
		if input.Enabled {
			state = types.ServiceWorkProfileStateEnabled
		} else {
			state = types.ServiceWorkProfileStateDraft
		}
	}
	if !types.IsValidServiceWorkProfileState(state) {
		return nil, ErrServiceInvalidProfileState
	}
	return &types.UserWorkProfile{
		ID:             id,
		TenantID:       tenantID,
		UserID:         userID,
		Name:           name,
		RoleType:       trimMax(input.RoleType, 64),
		CampusScope:    cleanStringArray(input.CampusScope, 50, 128),
		CourseScope:    cleanStringArray(input.CourseScope, 50, 128),
		MemoryScope:    firstNonEmpty(trimMax(input.MemoryScope, 0), "本人记忆 · 服务相关"),
		TonePreference: trimMax(input.TonePreference, serviceMaxShortText),
		DefaultProfile: input.DefaultProfile,
		Enabled:        input.Enabled && state == types.ServiceWorkProfileStateEnabled,
		State:          state,
		CreatedBy:      operatorUserID,
		UpdatedBy:      operatorUserID,
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func buildWorkProfileFromMemberDescription(ctx context.Context, member *types.TenantMember) *types.UserWorkProfile {
	operatorUserID := firstNonEmpty(auditActor(ctx), member.UserID)
	description := strings.TrimSpace(member.WorkProfileDescription)
	return &types.UserWorkProfile{
		ID:             deterministicServiceID("work-profile", strconv.FormatUint(member.TenantID, 10), member.UserID, "default"),
		TenantID:       member.TenantID,
		UserID:         member.UserID,
		Name:           "默认服务助理",
		RoleType:       inferWorkProfileRoleType(description),
		CampusScope:    types.StringArray{},
		CourseScope:    types.StringArray{},
		MemoryScope:    "本人记忆 · 服务相关",
		TonePreference: inferWorkProfileTone(description),
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
		CreatedBy:      operatorUserID,
		UpdatedBy:      operatorUserID,
		UpdatedAt:      time.Now().UTC(),
	}
}

func inferWorkProfileRoleType(description string) string {
	switch {
	case containsAny(description, []string{"园长", "校长", "负责人"}):
		return "principal"
	case containsAny(description, []string{"招生", "销售", "顾问", "咨询", "试听", "邀约", "报名"}):
		return "consultant"
	case containsAny(description, []string{"班主任", "老师", "教师", "教务"}):
		return "teacher"
	default:
		return "service"
	}
}

func inferWorkProfileTone(description string) string {
	if containsAny(description, []string{"专业温和", "耐心", "亲和", "共情"}) {
		return "专业温和、耐心细致"
	}
	return "先确认事实，再生成可发送话术和下一步"
}

func isDuplicateServiceConfig(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}

func buildAgentSetting(
	tenantID uint64,
	operatorUserID string,
	profileID string,
	input types.WorkProfileAgentSettingInput,
	index int,
) (*types.WorkProfileAgentSetting, error) {
	domain := strings.TrimSpace(input.AgentDomain)
	if !types.IsValidServiceAgentDomain(domain) {
		return nil, ErrServiceInvalidAgentDomain
	}
	displayName := trimMax(input.DisplayName, serviceMaxShortText)
	if displayName == "" {
		displayName = serviceAgentDomainLabel(domain)
	}
	workDocDirectory := trimMax(input.WorkDocDirectory, serviceMaxShortText)
	if workDocDirectory == "" {
		workDocDirectory = defaultWorkDocDirectory(domain)
	}
	order := input.DisplayOrder
	if order == 0 {
		order = index + 1
	}
	return &types.WorkProfileAgentSetting{
		ID:               strings.TrimSpace(input.ID),
		TenantID:         tenantID,
		ProfileID:        profileID,
		AgentID:          firstNonEmpty(trimMax(input.AgentID, 64), types.BuiltinServiceAssistantID),
		AgentDomain:      domain,
		Enabled:          input.Enabled,
		DisplayName:      displayName,
		DisplayOrder:     order,
		MemoryFilter:     normalizeJSONMap(input.MemoryFilter),
		KnowledgeBaseIDs: cleanStringArray(input.KnowledgeBaseIDs, 100, 64),
		WorkDocDirectory: workDocDirectory,
		SelectedSkills:   cleanStringArray(input.SelectedSkills, 100, 128),
		OutputPolicy:     normalizeJSONMap(input.OutputPolicy),
		CreatedBy:        operatorUserID,
		UpdatedBy:        operatorUserID,
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

func buildServiceSubject(tenantID uint64, userID, name, studentName string, confidence float64) *types.ServiceSubject {
	return &types.ServiceSubject{
		TenantID:        tenantID,
		OwnerUserID:     userID,
		SubjectKey:      serviceSubjectKey(name, studentName),
		DisplayName:     firstNonEmpty(name, "服务事项"),
		StudentName:     studentName,
		Relation:        inferRelation(name),
		Aliases:         types.StringArray{firstNonEmpty(name, studentName)},
		ExternalRefs:    types.JSONMap{},
		VisibilityScope: "private",
		Confidence:      confidence,
		UpdatedAt:       time.Now().UTC(),
	}
}

func buildActionDraftFromReminder(
	tenantID uint64,
	userID string,
	reminder *types.ServiceReminder,
	input types.AgentActionDraftInput,
) *types.AgentActionDraft {
	actionType := firstNonEmpty(trimMax(input.ActionType, 64), actionTypeForDomain(reminder.AgentDomain))
	title := firstNonEmpty(trimMax(input.Title, serviceMaxTitleLength), reminder.NextAction)
	summary := firstNonEmpty(strings.TrimSpace(input.Summary), reminder.WriteBackDraft)
	sourceMemoryIDs := cleanStringArray(input.SourceMemoryIDs, 100, 64)
	if len(sourceMemoryIDs) == 0 {
		sourceMemoryIDs = cleanStringArray(reminder.SourceMemoryIDs, 100, 64)
	}
	payload := normalizeJSONMap(input.Payload)
	if len(payload) == 0 {
		payload = types.JSONMap{
			"reply_draft":    reminder.ReplyDraft,
			"next_action":    reminder.NextAction,
			"service_stage":  reminder.Stage,
			"service_mode":   reminder.Metadata["service_mode"],
			"subject_name":   reminder.Metadata["subject_name"],
			"customer_name":  reminder.Metadata["customer_name"],
			"student_name":   reminder.Metadata["student_name"],
			"external_write": "draft_only",
		}
	}
	return &types.AgentActionDraft{
		TenantID:         tenantID,
		UserID:           userID,
		ReminderID:       reminder.ID,
		AgentID:          firstNonEmpty(trimMax(input.AgentID, 64), types.BuiltinServiceAssistantID),
		AgentDomain:      firstNonEmpty(trimMax(input.AgentDomain, 64), reminder.AgentDomain),
		ActionType:       actionType,
		Status:           types.AgentActionDraftStatusDraft,
		Title:            title,
		Summary:          summary,
		Payload:          payload,
		SourceMemoryIDs:  sourceMemoryIDs,
		ExternalSystem:   trimMax(input.ExternalSystem, 128),
		ExternalObjectID: trimMax(input.ExternalObjectID, serviceMaxShortText),
	}
}

func bucketServiceMemories(memories []*types.OrganizeMemory) map[string][]*types.OrganizeMemory {
	buckets := map[string][]*types.OrganizeMemory{}
	for _, memory := range memories {
		customerName := extractCustomerName(memory)
		if customerName == "" {
			continue
		}
		buckets[customerName] = append(buckets[customerName], memory)
	}
	return buckets
}

func buildReminderFromMemories(
	tenantID uint64,
	userID string,
	profile *types.UserWorkProfile,
	customerName string,
	memories []*types.OrganizeMemory,
) *types.ServiceReminder {
	latest := memories[0]
	metadata := latest.Metadata
	textParts := make([]string, 0, len(memories))
	tags := make([]string, 0)
	sourceMemoryIDs := make([]string, 0, len(memories))
	for _, memory := range memories {
		textParts = append(textParts, memorySearchText(memory))
		tags = append(tags, asStringList(memory.Metadata["tags"])...)
		sourceMemoryIDs = append(sourceMemoryIDs, memory.ID)
	}
	text := strings.Join(textParts, "\n")
	stage := firstMetadataString(metadata, "stage", "sales_stage", "salesStage", "service_stage", "serviceStage")
	if stage == "" {
		stage = inferStage(text)
	}
	riskLabel := firstMetadataString(metadata, "risk_label", "riskLabel", "risk", "concern")
	if riskLabel == "" {
		riskLabel = inferRiskLabel(text)
	}
	priority := inferPriority(text, riskLabel)
	nextAction := firstMetadataString(metadata, "next_action", "nextAction", "follow_up_action", "followUpAction")
	if nextAction == "" {
		nextAction = inferNextAction(stage, riskLabel)
	}
	studentName := extractStudentName(latest, customerName)
	channel := firstMetadataString(metadata, "channel", "source_channel", "sourceChannel")
	if channel == "" {
		channel = inferChannel(text)
	}
	decisionRole := firstMetadataString(metadata, "decision_role", "decisionRole", "decision_maker", "decisionMaker")
	if decisionRole == "" {
		decisionRole = inferDecisionRole(text)
	}
	summary := firstMetadataString(metadata, "summary", "source_summary", "description", "abstract")
	if summary == "" {
		summary = serviceMemoryExcerpt(latest)
	}
	primaryAction := firstMetadataString(metadata, "primary_action", "primaryAction")
	if primaryAction == "" {
		primaryAction = inferPrimaryAction(stage, riskLabel)
	}
	avoidAction := firstMetadataString(metadata, "avoid_action", "avoidAction")
	if avoidAction == "" {
		avoidAction = inferAvoidAction(riskLabel)
	}
	replyDraft := firstMetadataString(metadata, "reply_draft", "replyDraft", "draft")
	if replyDraft == "" {
		replyDraft = inferReplyDraft(customerName, stage, riskLabel)
	}
	dueAt, dueText := inferDue(
		firstMetadataString(metadata, "next_follow_up_at", "nextFollowUpAt", "follow_up_at", "followUpAt", "due_at", "dueAt", "due_text", "dueText"),
		priority,
	)
	signals := deriveMemorySignals(text, tags)
	confidence := 0.68
	if len(sourceMemoryIDs) > 1 {
		confidence = 0.86
	}
	domain := domainForText(text, stage, riskLabel)
	reminderID := deterministicServiceID("service-reminder", profile.ID, serviceSubjectKey(customerName, studentName), domain)
	return &types.ServiceReminder{
		ID:                reminderID,
		TenantID:          tenantID,
		UserID:            userID,
		ProfileID:         profile.ID,
		AgentDomain:       domain,
		Title:             firstNonEmpty(latest.Title, stage+"提醒"),
		Summary:           summary,
		Status:            types.ServiceReminderStatusPending,
		Priority:          priority,
		DueAt:             dueAt,
		DueText:           dueText,
		Stage:             stage,
		Channel:           channel,
		DecisionRole:      decisionRole,
		RiskLabel:         riskLabel,
		AssistReason:      fmt.Sprintf("从 %d 条客户记忆抽到：%s", len(sourceMemoryIDs), summary),
		PrimaryAction:     primaryAction,
		NextAction:        nextAction,
		AvoidAction:       avoidAction,
		ContextItems:      cleanStringArray(append([]string{"个人记忆"}, signals...), 10, 64),
		MemorySignals:     cleanStringArray(signals, 10, 64),
		SourceMemoryIDs:   cleanStringArray(sourceMemoryIDs, 100, 64),
		SourceMemoryCount: len(sourceMemoryIDs),
		LastMemoryAt:      &latest.OccurredAt,
		Confidence:        confidence,
		SalesHighlights:   cleanStringArray(salesHighlightsFrom(stage, riskLabel, text), 6, 160),
		WriteBackStatus:   "待确认",
		WriteBackDraft:    fmt.Sprintf("%s｜%s。依据：%s", customerName, nextAction, firstNonEmpty(latest.Title, "最近记忆")),
		ReplyDraft:        replyDraft,
		Metadata: types.JSONMap{
			"customer_name": customerName,
			"student_name":  studentName,
			"memory_scope":  profile.MemoryScope,
			"source_type":   "memory",
		},
		UpdatedAt: time.Now().UTC(),
	}
}

func buildWorkDocs(
	tenantID uint64,
	userID string,
	profileID string,
	subjectID string,
	reminder *types.ServiceReminder,
	memories []*types.OrganizeMemory,
) []*types.AgentWorkDoc {
	customerName := asString(reminder.Metadata["customer_name"])
	studentName := asString(reminder.Metadata["student_name"])
	if studentName == "待补充" && customerName == "" {
		studentName = ""
	}
	subjectName := asString(reminder.Metadata["subject_name"])
	subjectLabel := firstNonEmpty(studentName, customerName, subjectName, reminder.Title, "服务事项")
	workDocDirectory := strings.Trim(asString(reminder.Metadata["work_doc_directory"]), "/")
	if workDocDirectory == "" {
		workDocDirectory = strings.Trim(defaultWorkDocDirectory(reminder.AgentDomain), "/")
	}
	basePath := workDocDirectory + "/" + sanitizePathSegment(subjectLabel)
	sourceIDs := cleanStringArray(reminder.SourceMemoryIDs, 100, 64)
	now := time.Now().UTC()
	summaryDocName := "服务摘要.md"
	summaryDocTitle := "服务摘要"
	if customerName != "" {
		summaryDocName = "客户摘要.md"
		summaryDocTitle = "客户摘要"
	}
	docs := []struct {
		name    string
		title   string
		content string
	}{
		{summaryDocName, summaryDocTitle, buildServiceSummaryMarkdown(reminder, memories)},
		{"跟进记录.md", "跟进记录", buildFollowUpMarkdown(reminder, memories)},
		{"未闭环事项.md", "未闭环事项", buildOpenItemsMarkdown(reminder)},
		{"证据索引.md", "证据索引", buildEvidenceMarkdown(reminder, memories)},
	}
	out := make([]*types.AgentWorkDoc, 0, len(docs))
	for _, doc := range docs {
		out = append(out, &types.AgentWorkDoc{
			TenantID:        tenantID,
			ProfileID:       profileID,
			SubjectID:       subjectID,
			OwnerUserID:     userID,
			AgentDomain:     reminder.AgentDomain,
			DocType:         "customer_workspace",
			DocPath:         basePath + "/" + doc.name,
			Title:           doc.title,
			Content:         doc.content,
			Status:          types.AgentWorkDocStatusCurrent,
			SourceMemoryIDs: sourceIDs,
			Metadata: types.JSONMap{
				"subject_id":    subjectID,
				"customer_name": customerName,
				"student_name":  studentName,
				"subject_name":  subjectName,
				"source":        "service_module",
			},
			UpdatedAt: now,
		})
	}
	return out
}

func buildWorkDocLinks(
	doc *types.AgentWorkDoc,
	reminder *types.ServiceReminder,
	memories []*types.OrganizeMemory,
) []*types.AgentWorkDocMemoryLink {
	links := make([]*types.AgentWorkDocMemoryLink, 0, len(memories))
	linkType := types.AgentWorkDocLinkTypeEvidence
	if strings.Contains(doc.DocPath, "未闭环") {
		linkType = types.AgentWorkDocLinkTypeTrigger
	} else if strings.Contains(doc.DocPath, "跟进") {
		linkType = types.AgentWorkDocLinkTypeFollowUp
	} else if strings.Contains(doc.DocPath, "证据") {
		linkType = types.AgentWorkDocLinkTypeEvidence
	}
	if reminder.RiskLabel == "售后风险" && strings.Contains(doc.DocPath, "未闭环") {
		linkType = types.AgentWorkDocLinkTypeRisk
	}
	for _, memory := range memories {
		links = append(links, &types.AgentWorkDocMemoryLink{
			TenantID:        doc.TenantID,
			DocID:           doc.ID,
			DocPath:         doc.DocPath,
			MemoryID:        memory.ID,
			SubjectID:       doc.SubjectID,
			AgentDomain:     doc.AgentDomain,
			LinkType:        linkType,
			Confidence:      reminder.Confidence,
			EvidenceExcerpt: contentExcerpt(memory.Content, memory.Title),
		})
	}
	return links
}

func buildServiceSummaryMarkdown(reminder *types.ServiceReminder, memories []*types.OrganizeMemory) string {
	customerName := asString(reminder.Metadata["customer_name"])
	studentName := asString(reminder.Metadata["student_name"])
	subjectName := firstNonEmpty(asString(reminder.Metadata["subject_name"]), reminder.Title, asString(reminder.Metadata["service_mode"]), "服务事项")
	if customerName == "" {
		return serviceDocFrontMatter("service_workspace", reminder) + fmt.Sprintf(`# %s

## 当前摘要

%s

## 当前阶段

- 服务模式：%s
- 风险/关注：%s
- 渠道：%s

## 下一步建议

%s
`, subjectName, reminder.Summary, firstNonEmpty(asString(reminder.Metadata["service_mode"]), reminder.Stage), reminder.RiskLabel, reminder.Channel, reminder.NextAction) + sourceMemoryBullets(memories)
	}
	return serviceDocFrontMatter("customer_workspace", reminder) + fmt.Sprintf(`# %s / %s

## 当前摘要

%s

## 当前阶段

- 阶段：%s
- 风险：%s
- 决策人：%s
- 渠道：%s

## 下一步建议

%s
`, studentName, customerName,
		reminder.Summary, reminder.Stage, reminder.RiskLabel, reminder.DecisionRole, reminder.Channel, reminder.NextAction) + sourceMemoryBullets(memories)
}

func buildFollowUpMarkdown(reminder *types.ServiceReminder, memories []*types.OrganizeMemory) string {
	var builder strings.Builder
	builder.WriteString(serviceDocFrontMatter("follow_up_records", reminder))
	builder.WriteString("# 跟进记录\n\n")
	for _, memory := range memories {
		builder.WriteString(fmt.Sprintf("- %s：%s。来源：%s\n", formatMonthDay(memory.OccurredAt), contentExcerpt(memory.Content, memory.Title), memory.ID))
	}
	builder.WriteString("\n## 建议话术\n\n")
	builder.WriteString(reminder.ReplyDraft)
	builder.WriteString("\n")
	return builder.String()
}

func buildOpenItemsMarkdown(reminder *types.ServiceReminder) string {
	return serviceDocFrontMatter("open_items", reminder) + fmt.Sprintf(`# 未闭环事项

- %s
- 待确认：%s
- 避免动作：%s
`, reminder.NextAction, reminder.WriteBackDraft, reminder.AvoidAction)
}

func buildEvidenceMarkdown(reminder *types.ServiceReminder, memories []*types.OrganizeMemory) string {
	var builder strings.Builder
	builder.WriteString(serviceDocFrontMatter("evidence_index", reminder))
	builder.WriteString("# 证据索引\n\n")
	builder.WriteString("| 记忆ID | 类型 | 作用 |\n|--------|------|------|\n")
	for _, memory := range memories {
		builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", memory.ID, memory.Kind, contentExcerpt(memory.Content, memory.Title)))
	}
	return builder.String()
}

func serviceDocFrontMatter(docType string, reminder *types.ServiceReminder) string {
	return fmt.Sprintf(`---
doc_type: %s
subject_id: %s
agent_domain: %s
source_memory_ids:
%s
updated_at: %s
---

`, docType, reminder.SubjectID, reminder.AgentDomain, yamlMemoryIDs(reminder.SourceMemoryIDs), time.Now().UTC().Format(time.RFC3339))
}

func yamlMemoryIDs(ids []string) string {
	if len(ids) == 0 {
		return "  []"
	}
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, "  - "+id)
	}
	return strings.Join(lines, "\n")
}

func sourceMemoryBullets(memories []*types.OrganizeMemory) string {
	if len(memories) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\n## 证据来源\n\n")
	for _, memory := range memories {
		builder.WriteString(fmt.Sprintf("- %s：%s\n", memory.ID, firstNonEmpty(memory.Title, "未命名记忆")))
	}
	return builder.String()
}

func buildCustomerSpaceSummary(
	profileID string,
	subject *types.ServiceSubject,
	docs []*types.AgentWorkDoc,
	reminders []*types.ServiceReminder,
) *types.ServiceCustomerSpace {
	customerDocs := filterCustomerWorkDocs(docs)
	latestReminder := latestServiceReminder(reminders)
	openReminderCount := 0
	for _, reminder := range reminders {
		if isOpenServiceReminderStatus(reminder.Status) {
			openReminderCount++
		}
	}
	sourceMemoryIDs := collectCustomerSpaceMemoryIDs(customerDocs, reminders)
	latestMemoryAt := latestMemoryTime(reminders)
	latestReminderAt := latestReminderTime(reminders)
	updatedAt := latestCustomerSpaceUpdate(subject, customerDocs, reminders)
	summary := firstNonEmpty(
		customerSpaceSummaryFromDocs(customerDocs),
		customerSpaceSummaryFromReminder(latestReminder),
		"暂无服务摘要",
	)
	stage := ""
	riskLabel := ""
	priority := ""
	latestAction := ""
	status := "current"
	if latestReminder != nil {
		stage = latestReminder.Stage
		riskLabel = latestReminder.RiskLabel
		priority = latestReminder.Priority
		latestAction = latestReminder.NextAction
		status = latestReminder.Status
	}
	if openReminderCount > 0 {
		status = types.ServiceReminderStatusPending
	} else if len(reminders) > 0 {
		status = types.ServiceReminderStatusCompleted
	}
	chips := cleanStringArray([]string{
		firstNonEmpty(stage, subject.Relation),
		riskLabel,
		latestAction,
	}, 4, 80)
	name := firstNonEmpty(subject.DisplayName, subject.StudentName, "服务事项")
	description := summary
	if subject.StudentName != "" && subject.DisplayName != subject.StudentName {
		description = fmt.Sprintf("%s · 学员：%s", summary, subject.StudentName)
	}
	return &types.ServiceCustomerSpace{
		ID:                subject.ID,
		TenantID:          subject.TenantID,
		OwnerUserID:       subject.OwnerUserID,
		ProfileID:         profileID,
		SubjectKey:        subject.SubjectKey,
		DisplayName:       subject.DisplayName,
		Name:              name,
		StudentName:       subject.StudentName,
		Relation:          subject.Relation,
		Description:       description,
		Summary:           summary,
		Status:            status,
		Priority:          priority,
		Stage:             stage,
		RiskLabel:         riskLabel,
		LatestAction:      latestAction,
		VisibilityScope:   subject.VisibilityScope,
		Confidence:        subject.Confidence,
		WorkDocCount:      len(customerDocs),
		ReminderCount:     len(reminders),
		OpenReminderCount: openReminderCount,
		SourceMemoryCount: len(sourceMemoryIDs),
		Directories:       customerSpaceDirectories(customerDocs),
		Chips:             chips,
		LatestMemoryAt:    latestMemoryAt,
		LatestReminderAt:  latestReminderAt,
		CreatedAt:         subject.CreatedAt,
		UpdatedAt:         updatedAt,
	}
}

func filterCustomerWorkDocs(docs []*types.AgentWorkDoc) []*types.AgentWorkDoc {
	out := make([]*types.AgentWorkDoc, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		if doc.DocType == "" || doc.DocType == types.AgentWorkDocTypeCustomerWorkspace {
			out = append(out, doc)
		}
	}
	return out
}

func latestServiceReminder(reminders []*types.ServiceReminder) *types.ServiceReminder {
	if len(reminders) == 0 {
		return nil
	}
	latest := reminders[0]
	for _, reminder := range reminders[1:] {
		if reminder.UpdatedAt.After(latest.UpdatedAt) {
			latest = reminder
		}
	}
	return latest
}

func customerSpaceSummaryFromReminder(reminder *types.ServiceReminder) string {
	if reminder == nil {
		return ""
	}
	return firstNonEmpty(reminder.Summary, reminder.AssistReason, reminder.NextAction)
}

func serviceMemoryExcerpt(memory *types.OrganizeMemory) string {
	if memory == nil {
		return "暂无记忆内容"
	}
	return contentExcerpt(
		firstNonEmpty(
			firstMetadataString(memory.Metadata, serviceSummaryKeys...),
			firstMetadataString(memory.Metadata, serviceTranscriptKeys...),
			memory.Content,
		),
		"暂无记忆内容",
	)
}

func customerSpaceSummaryFromDocs(docs []*types.AgentWorkDoc) string {
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		if doc.Title == "客户摘要" || doc.Title == "服务摘要" || strings.HasSuffix(doc.DocPath, "/客户摘要.md") || strings.HasSuffix(doc.DocPath, "/服务摘要.md") {
			return markdownSectionExcerpt(doc.Content, "当前摘要", "")
		}
	}
	for _, doc := range docs {
		if doc != nil {
			return contentExcerpt(firstNonEmpty(asString(doc.Metadata["summary"]), doc.Content), "")
		}
	}
	return ""
}

func markdownSectionExcerpt(content string, heading string, fallback string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return fallback
	}
	marker := "## " + heading
	if idx := strings.Index(content, marker); idx >= 0 {
		section := strings.TrimSpace(content[idx+len(marker):])
		if next := strings.Index(section, "\n## "); next >= 0 {
			section = section[:next]
		}
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(strings.TrimLeft(line, "-* "))
			if line != "" {
				return trimMax(line, 120)
			}
		}
	}
	return contentExcerpt(content, fallback)
}

func collectCustomerSpaceMemoryIDs(docs []*types.AgentWorkDoc, reminders []*types.ServiceReminder) []string {
	ids := make([]string, 0)
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		ids = append(ids, reminder.SourceMemoryIDs...)
	}
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		ids = append(ids, doc.SourceMemoryIDs...)
	}
	return []string(cleanStringArray(ids, 200, 64))
}

func customerSpaceDirectories(docs []*types.AgentWorkDoc) types.StringArray {
	out := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		out = append(out, doc.DocPath)
	}
	return cleanStringArray(out, 40, 512)
}

func latestMemoryTime(reminders []*types.ServiceReminder) *time.Time {
	var latest *time.Time
	for _, reminder := range reminders {
		if reminder == nil || reminder.LastMemoryAt == nil || reminder.LastMemoryAt.IsZero() {
			continue
		}
		if latest == nil || reminder.LastMemoryAt.After(*latest) {
			t := *reminder.LastMemoryAt
			latest = &t
		}
	}
	return latest
}

func latestReminderTime(reminders []*types.ServiceReminder) *time.Time {
	var latest *time.Time
	for _, reminder := range reminders {
		if reminder == nil || reminder.UpdatedAt.IsZero() {
			continue
		}
		if latest == nil || reminder.UpdatedAt.After(*latest) {
			t := reminder.UpdatedAt
			latest = &t
		}
	}
	return latest
}

func latestCustomerSpaceUpdate(
	subject *types.ServiceSubject,
	docs []*types.AgentWorkDoc,
	reminders []*types.ServiceReminder,
) time.Time {
	updatedAt := subject.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = subject.CreatedAt
	}
	for _, doc := range docs {
		if doc != nil && doc.UpdatedAt.After(updatedAt) {
			updatedAt = doc.UpdatedAt
		}
	}
	for _, reminder := range reminders {
		if reminder != nil && reminder.UpdatedAt.After(updatedAt) {
			updatedAt = reminder.UpdatedAt
		}
	}
	return updatedAt
}

func validateServiceScope(tenantID uint64, userID string) error {
	if tenantID == 0 || strings.TrimSpace(userID) == "" {
		return ErrServiceInvalidScope
	}
	return nil
}

func normalizeServiceListQuery(query types.ServiceListQuery) types.ServiceListQuery {
	query.UserID = strings.TrimSpace(query.UserID)
	query.ProfileID = strings.TrimSpace(query.ProfileID)
	query.SubjectID = strings.TrimSpace(query.SubjectID)
	query.MemoryID = strings.TrimSpace(query.MemoryID)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Status = strings.TrimSpace(query.Status)
	query.AgentDomain = strings.TrimSpace(query.AgentDomain)
	if query.Page <= 0 {
		query.Page = serviceDefaultPage
	}
	if query.PageSize <= 0 {
		query.PageSize = serviceDefaultPageSize
	}
	if query.PageSize > serviceMaxPageSize {
		query.PageSize = serviceMaxPageSize
	}
	return query
}

func normalizeServiceCustomerSpaceListQuery(query types.ServiceCustomerSpaceListQuery) types.ServiceCustomerSpaceListQuery {
	query.UserID = strings.TrimSpace(query.UserID)
	query.ProfileID = strings.TrimSpace(query.ProfileID)
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Page <= 0 {
		query.Page = serviceDefaultPage
	}
	if query.PageSize <= 0 {
		query.PageSize = serviceDefaultPageSize
	}
	if query.PageSize > serviceMaxPageSize {
		query.PageSize = serviceMaxPageSize
	}
	return query
}

type serviceDailyReportPeriod struct {
	Range      string
	ReportDate string
	Start      time.Time
	End        time.Time
	Location   *time.Location
}

func normalizeServiceDailyReportListQuery(query types.ServiceDailyReportListQuery) types.ServiceDailyReportListQuery {
	query.UserID = strings.TrimSpace(query.UserID)
	query.ProfileID = strings.TrimSpace(query.ProfileID)
	query.Range = normalizeServiceDailyReportRange(query.Range)
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Page <= 0 {
		query.Page = serviceDefaultPage
	}
	if query.PageSize <= 0 {
		query.PageSize = serviceDefaultPageSize
	}
	if query.PageSize > serviceMaxPageSize {
		query.PageSize = serviceMaxPageSize
	}
	return query
}

func resolveServiceDailyReportPeriod(input types.ServiceDailyReportInput) (serviceDailyReportPeriod, error) {
	reportRange := normalizeServiceDailyReportRange(input.Range)
	if reportRange == "" {
		reportRange = types.ServiceDailyReportRangeDay
	}
	if !types.IsValidServiceDailyReportRange(reportRange) {
		return serviceDailyReportPeriod{}, ErrServiceInvalidReportRange
	}
	loc := serviceDailyReportLocation(input.Timezone)
	baseDate, err := parseServiceDailyReportDate(input.Date, loc)
	if err != nil {
		return serviceDailyReportPeriod{}, err
	}
	start := periodStart(baseDate, reportRange)
	end := periodEnd(start, reportRange)
	return serviceDailyReportPeriod{
		Range:      reportRange,
		ReportDate: baseDate.Format("2006-01-02"),
		Start:      start,
		End:        end,
		Location:   loc,
	}, nil
}

func normalizeServiceDailyReportRange(reportRange string) string {
	switch strings.ToLower(strings.TrimSpace(reportRange)) {
	case "":
		return ""
	case "day", "daily", "today", "date":
		return types.ServiceDailyReportRangeDay
	case "week", "weekly":
		return types.ServiceDailyReportRangeWeek
	case "month", "monthly":
		return types.ServiceDailyReportRangeMonth
	default:
		return strings.ToLower(strings.TrimSpace(reportRange))
	}
}

func normalizeServiceDailyReportTrigger(trigger string) string {
	switch strings.ToLower(strings.TrimSpace(trigger)) {
	case "supplement", "backfill", "补记", "补入":
		return "supplement"
	case "scheduled", "schedule", "auto", "自动":
		return "scheduled"
	default:
		return "user_requested"
	}
}

func serviceDailyReportLocation(timezone string) *time.Location {
	name := strings.TrimSpace(timezone)
	if name == "" {
		name = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(name)
	if err == nil {
		return loc
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func parseServiceDailyReportDate(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	}
	if parsed, err := time.ParseInLocation("2006-01-02", raw, loc); err == nil {
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc), nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		local := parsed.In(loc)
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc), nil
	}
	return time.Time{}, ErrServiceInvalidReportDate
}

func periodStart(baseDate time.Time, reportRange string) time.Time {
	switch reportRange {
	case types.ServiceDailyReportRangeWeek:
		offset := (int(baseDate.Weekday()) + 6) % 7
		weekStart := baseDate.AddDate(0, 0, -offset)
		return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, baseDate.Location())
	case types.ServiceDailyReportRangeMonth:
		return time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, baseDate.Location())
	default:
		return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, baseDate.Location())
	}
}

func periodEnd(start time.Time, reportRange string) time.Time {
	switch reportRange {
	case types.ServiceDailyReportRangeWeek:
		return start.AddDate(0, 0, 7)
	case types.ServiceDailyReportRangeMonth:
		return start.AddDate(0, 1, 0)
	default:
		return start.AddDate(0, 0, 1)
	}
}

func filterRemindersForDailyReport(reminders []*types.ServiceReminder, period serviceDailyReportPeriod) []*types.ServiceReminder {
	out := make([]*types.ServiceReminder, 0, len(reminders))
	for _, reminder := range reminders {
		if reminder == nil || reminder.Status == types.ServiceReminderStatusIgnored || reminder.Status == types.ServiceReminderStatusStale {
			continue
		}
		if isOpenServiceReminderStatus(reminder.Status) || timeInServiceReportPeriod(serviceReminderReferenceTime(reminder), period) {
			out = append(out, reminder)
		}
	}
	return out
}

func isOpenServiceReminderStatus(status string) bool {
	switch status {
	case types.ServiceReminderStatusCandidate, types.ServiceReminderStatusPending, types.ServiceReminderStatusGenerated,
		types.ServiceReminderStatusSnoozed, types.ServiceReminderStatusRecomputeRequired:
		return true
	default:
		return false
	}
}

func serviceReminderReferenceTime(reminder *types.ServiceReminder) time.Time {
	if reminder == nil {
		return time.Time{}
	}
	if reminder.DueAt != nil && !reminder.DueAt.IsZero() {
		return *reminder.DueAt
	}
	if reminder.LastMemoryAt != nil && !reminder.LastMemoryAt.IsZero() {
		return *reminder.LastMemoryAt
	}
	if !reminder.UpdatedAt.IsZero() {
		return reminder.UpdatedAt
	}
	return reminder.CreatedAt
}

func timeInServiceReportPeriod(value time.Time, period serviceDailyReportPeriod) bool {
	if value.IsZero() {
		return false
	}
	local := value.In(period.Location)
	return !local.Before(period.Start) && local.Before(period.End)
}

func buildDailyReportSubject(tenantID uint64, userID, profileID string) *types.ServiceSubject {
	return &types.ServiceSubject{
		TenantID:        tenantID,
		OwnerUserID:     userID,
		SubjectKey:      serviceSubjectKey("日报复盘", profileID),
		DisplayName:     "日报复盘",
		StudentName:     "",
		Relation:        "",
		Aliases:         types.StringArray{"日报复盘"},
		ExternalRefs:    types.JSONMap{"profile_id": profileID},
		VisibilityScope: "private",
		Confidence:      1,
		UpdatedAt:       time.Now().UTC(),
	}
}

type serviceDailyReportStats struct {
	ActionCount          int
	SubjectCount         int
	OpenActionCount      int
	ClosedActionCount    int
	HighRiskCount        int
	KnowledgeGapCount    int
	SourceMemoryCount    int
	DailySourceCount     int
	EvidenceCompleteRate int
}

func buildDailyReportStats(reminders []*types.ServiceReminder, dailySources []*types.AgentWorkDoc) serviceDailyReportStats {
	return serviceDailyReportStats{
		ActionCount:          len(reminders),
		SubjectCount:         dailyReportSubjectCount(reminders),
		OpenActionCount:      dailyReportOpenActionCount(reminders),
		ClosedActionCount:    dailyReportClosedActionCount(reminders),
		HighRiskCount:        dailyReportHighRiskCount(reminders),
		KnowledgeGapCount:    dailyReportKnowledgeGapCount(reminders),
		SourceMemoryCount:    len(sourceMemoryIDsFromReminders(reminders)),
		DailySourceCount:     len(dailySources),
		EvidenceCompleteRate: dailyReportEvidenceCompleteRate(reminders),
	}
}

func buildDailyReportMarkdown(
	period serviceDailyReportPeriod,
	profile *types.UserWorkProfile,
	reminders []*types.ServiceReminder,
	dailySources []*types.AgentWorkDoc,
	stats serviceDailyReportStats,
) string {
	stageLines := dailyReportStageLines(reminders)
	riskLines := dailyReportRiskLines(reminders)
	actionLines := dailyReportActionLines(reminders)
	gapLines := dailyReportKnowledgeGapLines(reminders)
	evidenceLines := dailyReportEvidenceLines(reminders)
	sourceSection := ""
	writeBackSectionIndex := 6
	if period.Range != types.ServiceDailyReportRangeDay || len(dailySources) > 0 {
		sourceSection = fmt.Sprintf(`

## 6、日粒度来源

%s
`, dailyReportDailySourceLines(period, dailySources))
		writeBackSectionIndex = 7
	}

	return fmt.Sprintf(`# %s

%s

分身：%s
收录范围：%s
生成逻辑：从分身授权记忆提取工作事项，形成提醒、日报和回写建议；动作确认后回写记忆，知识缺口进入知识库审核。
统计：%d 个动作，覆盖 %d 个服务对象，%d 条来源记忆，高风险 %d 条，证据完整率 %d%%。

## 1、服务回顾

%s

## 2、风险归因

%s

## 3、行动闭环

%s

## 4、知识补齐

%s

## 5、证据来源

%s
%s
## %d、回写建议

%s
`, dailyReportTitle(period), dailyReportDiagnosis(reminders, stats), dailyReportProfileName(profile), dailyReportMemoryScope(profile),
		stats.ActionCount, stats.SubjectCount, stats.SourceMemoryCount, stats.HighRiskCount, stats.EvidenceCompleteRate,
		stageLines, riskLines, actionLines, gapLines, evidenceLines, sourceSection, writeBackSectionIndex, dailyReportWriteBackLines(period, stats))
}

func dailyReportTitle(period serviceDailyReportPeriod) string {
	switch period.Range {
	case types.ServiceDailyReportRangeWeek:
		return fmt.Sprintf("%s-%s服务周报", formatReportShortDate(period.Start), formatReportShortDate(period.End.Add(-time.Nanosecond)))
	case types.ServiceDailyReportRangeMonth:
		return period.Start.Format("2006年1月服务月报")
	default:
		return period.Start.Format("2006年1月2日服务日报")
	}
}

func dailyReportDocPath(period serviceDailyReportPeriod) string {
	switch period.Range {
	case types.ServiceDailyReportRangeWeek:
		return "日报/week/" + period.Start.Format("2006-01-02") + ".md"
	case types.ServiceDailyReportRangeMonth:
		return "日报/month/" + period.Start.Format("2006-01") + ".md"
	default:
		return "日报/" + period.Start.Format("2006-01-02") + ".md"
	}
}

func formatReportShortDate(value time.Time) string {
	return value.Format("1月2日")
}

func dailyReportPeriodLabel(period serviceDailyReportPeriod) string {
	switch period.Range {
	case types.ServiceDailyReportRangeWeek:
		return fmt.Sprintf("%s-%s", formatReportShortDate(period.Start), formatReportShortDate(period.End.Add(-time.Nanosecond)))
	case types.ServiceDailyReportRangeMonth:
		return period.Start.Format("2006年1月")
	default:
		return period.Start.Format("1月2日")
	}
}

func dailyReportProfileName(profile *types.UserWorkProfile) string {
	if profile == nil {
		return "服务分身"
	}
	return firstNonEmpty(profile.Name, "服务分身")
}

func dailyReportMemoryScope(profile *types.UserWorkProfile) string {
	if profile == nil {
		return "本人服务相关记忆"
	}
	return firstNonEmpty(profile.MemoryScope, "本人服务相关记忆")
}

func dailyReportDiagnosis(reminders []*types.ServiceReminder, stats serviceDailyReportStats) string {
	if stats.ActionCount == 0 {
		return "当前没有可汇总的服务提醒。建议先补充工作记忆或刷新分身服务提醒，再生成日报。"
	}
	return fmt.Sprintf("本次共汇总 %d 个工作动作，覆盖 %d 个服务对象；其中高风险 %d 条，待处理 %d 条，已确认或完成 %d 条。",
		stats.ActionCount, stats.SubjectCount, stats.HighRiskCount, stats.OpenActionCount, stats.ClosedActionCount)
}

func dailyReportStageLines(reminders []*types.ServiceReminder) string {
	if len(reminders) == 0 {
		return "- 暂无可回顾的客户阶段。"
	}
	return countedReminderLines(reminders, func(reminder *types.ServiceReminder) string {
		return firstNonEmpty(reminder.Stage, "客户跟进")
	}, func(stage string, items []*types.ServiceReminder) string {
		return fmt.Sprintf("- %s：%d 条，建议动作：%s", stage, len(items), firstNonEmpty(items[0].NextAction, "确认下一步服务动作"))
	})
}

func dailyReportRiskLines(reminders []*types.ServiceReminder) string {
	if len(reminders) == 0 {
		return "- 暂无明显风险。"
	}
	return countedReminderLines(reminders, func(reminder *types.ServiceReminder) string {
		return firstNonEmpty(reminder.RiskLabel, "待判断")
	}, func(riskLabel string, items []*types.ServiceReminder) string {
		return fmt.Sprintf("- %s：%d 条，%s", riskLabel, len(items), dailyReportRiskDescription(riskLabel))
	})
}

func dailyReportRiskDescription(riskLabel string) string {
	switch riskLabel {
	case "售后风险":
		return "先确认处理结果，再决定是否补充解释或升级沟通。"
	case "价格顾虑":
		return "先补价值证明和孩子变化，再谈价格或优惠边界。"
	case "适应焦虑":
		return "需要具体观察支撑，避免用泛泛安慰替代事实。"
	case "续费窗口":
		return "先做阶段成长回顾，再进入续费判断。"
	case "未闭环":
		return "需要补齐处理结果、家长反馈和下一步记录。"
	default:
		return "记忆证据不足，先补关键事实再生成判断。"
	}
}

func dailyReportActionLines(reminders []*types.ServiceReminder) string {
	if len(reminders) == 0 {
		return "- 暂无待处理动作。"
	}
	lines := make([]string, 0, len(reminders))
	for _, reminder := range reminders {
		subjectName := firstNonEmpty(
			asString(reminder.Metadata["customer_name"]),
			asString(reminder.Metadata["subject_name"]),
			reminder.Title,
			asString(reminder.Metadata["service_mode"]),
			"服务事项",
		)
		confidence := "待确认"
		if reminder.Confidence >= 0.8 {
			confidence = "较高"
		} else if reminder.Confidence < 0.6 {
			confidence = "低"
		}
		lines = append(lines, fmt.Sprintf("- %s：%s（%s，%s，%s）",
			subjectName,
			firstNonEmpty(reminder.NextAction, "确认服务事项并补一条下一步记忆"),
			firstNonEmpty(reminder.DueText, formatMonthDay(serviceReminderReferenceTime(reminder))),
			serviceReminderStatusLabel(reminder.Status),
			confidence,
		))
		if len(lines) >= 12 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func dailyReportKnowledgeGapLines(reminders []*types.ServiceReminder) string {
	if len(reminders) == 0 {
		return "- 暂无知识补齐建议。"
	}
	gaps := []string{}
	if hasRiskLabel(reminders, "价格顾虑") {
		gaps = append(gaps, "- 价格异议材料：补课程价值、孩子变化和费用边界说明。")
	}
	if hasRiskLabel(reminders, "适应焦虑") {
		gaps = append(gaps, "- 适应期回应模板：补具体观察、适应节奏和老师沟通口径。")
	}
	if hasRiskLabel(reminders, "售后风险") || hasRiskLabel(reminders, "未闭环") {
		gaps = append(gaps, "- 售后闭环 SOP：补首次回应、处理进展、结果确认和复盘记录。")
	}
	if hasStage(reminders, "续费服务") || hasRiskLabel(reminders, "续费窗口") {
		gaps = append(gaps, "- 成长回顾模板：补阶段变化、当前目标和下一阶段安排。")
	}
	if hasLowConfidenceReminder(reminders) {
		gaps = append(gaps, "- 关键事实补录：补客户、学员、决策人和下一步时间。")
	}
	if len(gaps) == 0 {
		gaps = append(gaps, "- 有效服务动作样本：选择完整服务记忆，沉淀成可复用话术。")
	}
	if len(gaps) > 4 {
		gaps = gaps[:4]
	}
	return strings.Join(gaps, "\n")
}

func dailyReportEvidenceLines(reminders []*types.ServiceReminder) string {
	sourceIDs := sourceMemoryIDsFromReminders(reminders)
	if len(sourceIDs) == 0 {
		return "- 暂无记忆证据。"
	}
	lines := make([]string, 0, len(sourceIDs))
	seen := map[string]bool{}
	for _, reminder := range reminders {
		for _, evidence := range reminder.MemoryEvidence {
			if seen[evidence.ID] {
				continue
			}
			seen[evidence.ID] = true
			lines = append(lines, fmt.Sprintf("- %s：%s（%s）", evidence.ID, firstNonEmpty(evidence.Title, "未命名记忆"), firstNonEmpty(evidence.OccurredAtLabel, "最近")))
			if len(lines) >= 12 {
				return strings.Join(lines, "\n")
			}
		}
	}
	for _, id := range sourceIDs {
		if seen[id] {
			continue
		}
		lines = append(lines, "- "+id)
		if len(lines) >= 12 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func countedReminderLines(
	reminders []*types.ServiceReminder,
	keyFn func(*types.ServiceReminder) string,
	lineFn func(string, []*types.ServiceReminder) string,
) string {
	groups := map[string][]*types.ServiceReminder{}
	for _, reminder := range reminders {
		key := firstNonEmpty(keyFn(reminder), "待判断")
		groups[key] = append(groups[key], reminder)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if len(groups[keys[i]]) == len(groups[keys[j]]) {
			return keys[i] < keys[j]
		}
		return len(groups[keys[i]]) > len(groups[keys[j]])
	})
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, lineFn(key, groups[key]))
		if len(lines) >= 6 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func hasRiskLabel(reminders []*types.ServiceReminder, riskLabel string) bool {
	for _, reminder := range reminders {
		if reminder.RiskLabel == riskLabel {
			return true
		}
	}
	return false
}

func hasStage(reminders []*types.ServiceReminder, stage string) bool {
	for _, reminder := range reminders {
		if reminder.Stage == stage {
			return true
		}
	}
	return false
}

func hasLowConfidenceReminder(reminders []*types.ServiceReminder) bool {
	for _, reminder := range reminders {
		if reminder.Confidence < 0.8 {
			return true
		}
	}
	return false
}

func serviceReminderStatusLabel(status string) string {
	switch status {
	case types.ServiceReminderStatusConfirmed:
		return "已确认"
	case types.ServiceReminderStatusCompleted:
		return "已完成"
	case types.ServiceReminderStatusGenerated:
		return "已生成话术"
	case types.ServiceReminderStatusSnoozed:
		return "稍后处理"
	case types.ServiceReminderStatusIgnored:
		return "已忽略"
	default:
		return "待处理"
	}
}

func sourceMemoryIDsFromReminders(reminders []*types.ServiceReminder) types.StringArray {
	ids := make([]string, 0)
	seen := map[string]bool{}
	for _, reminder := range reminders {
		for _, id := range reminder.SourceMemoryIDs {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return cleanStringArray(ids, 100, 64)
}

func dailyReportHighRiskCount(reminders []*types.ServiceReminder) int {
	count := 0
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		if reminder.Priority == types.ServicePriorityHigh || reminder.RiskLabel == "售后风险" || reminder.RiskLabel == "未闭环" || reminder.RiskLabel == "价格顾虑" {
			count++
		}
	}
	return count
}

func dailyReportOpenActionCount(reminders []*types.ServiceReminder) int {
	count := 0
	for _, reminder := range reminders {
		if reminder != nil && isOpenServiceReminderStatus(reminder.Status) {
			count++
		}
	}
	return count
}

func dailyReportClosedActionCount(reminders []*types.ServiceReminder) int {
	count := 0
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		if reminder.Status == types.ServiceReminderStatusConfirmed || reminder.Status == types.ServiceReminderStatusCompleted {
			count++
		}
	}
	return count
}

func dailyReportKnowledgeGapCount(reminders []*types.ServiceReminder) int {
	if len(reminders) == 0 {
		return 0
	}
	count := 0
	if hasRiskLabel(reminders, "价格顾虑") {
		count++
	}
	if hasRiskLabel(reminders, "适应焦虑") {
		count++
	}
	if hasRiskLabel(reminders, "售后风险") || hasRiskLabel(reminders, "未闭环") {
		count++
	}
	if hasStage(reminders, "续费服务") || hasRiskLabel(reminders, "续费窗口") {
		count++
	}
	if hasLowConfidenceReminder(reminders) {
		count++
	}
	if count == 0 {
		return 1
	}
	if count > 4 {
		return 4
	}
	return count
}

func dailyReportEvidenceCompleteRate(reminders []*types.ServiceReminder) int {
	if len(reminders) == 0 {
		return 0
	}
	covered := 0
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		if len(reminder.SourceMemoryIDs) > 0 || len(reminder.MemoryEvidence) > 0 {
			covered++
		}
	}
	return int(float64(covered) / float64(len(reminders)) * 100)
}

func dailyReportSubjectCount(reminders []*types.ServiceReminder) int {
	seen := map[string]bool{}
	for _, reminder := range reminders {
		if reminder == nil {
			continue
		}
		key := firstNonEmpty(
			reminder.SubjectID,
			asString(reminder.Metadata["subject_name"]),
			asString(reminder.Metadata["customer_name"]),
			asString(reminder.Metadata["service_mode"]),
			reminder.Title,
		)
		if key != "" {
			seen[key] = true
		}
	}
	return len(seen)
}

func dailyReportCustomerCount(reminders []*types.ServiceReminder) int {
	return dailyReportSubjectCount(reminders)
}

func dailyReportChips(reminders []*types.ServiceReminder, dailySources []*types.AgentWorkDoc) types.StringArray {
	chips := []string{"分身日报", "记忆提取", "行动闭环"}
	if len(reminders) == 0 {
		chips = []string{"分身日报", "暂无提醒", "待补记忆"}
	}
	if len(dailySources) > 0 {
		chips = append(chips, "日粒度汇总")
	}
	if hasRiskLabel(reminders, "售后风险") || hasRiskLabel(reminders, "未闭环") {
		chips = append(chips, "风险闭环")
	}
	if hasRiskLabel(reminders, "价格顾虑") {
		chips = append(chips, "价格异议")
	}
	if hasStage(reminders, "续费服务") {
		chips = append(chips, "续费服务")
	}
	return cleanStringArray(chips, 6, 16)
}

func dailyReportDailySourceLines(period serviceDailyReportPeriod, docs []*types.AgentWorkDoc) string {
	if len(docs) == 0 {
		if period.Range == types.ServiceDailyReportRangeDay {
			return "- 当前为日粒度日报，无需引用其他日报。"
		}
		return "- 暂无已生成的日粒度日报，本次直接从分身服务提醒汇总。"
	}
	lines := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		metadata := normalizeJSONMap(doc.Metadata)
		subjectCount := firstPositiveInt(asInt(metadata["subject_count"]), asInt(metadata["customer_count"]))
		line := fmt.Sprintf("- %s：%s（%d 个动作，%d 个服务对象）",
			firstNonEmpty(asString(metadata["period_label"]), formatReportShortDate(dailyReportDocStartTime(doc, period.Location))),
			firstNonEmpty(doc.Title, "服务日报"),
			asInt(metadata["action_count"]),
			subjectCount,
		)
		lines = append(lines, line)
		if len(lines) >= 12 {
			break
		}
	}
	if len(lines) == 0 {
		return "- 暂无已生成的日粒度日报，本次直接从分身服务提醒汇总。"
	}
	return strings.Join(lines, "\n")
}

func dailyReportWriteBackLines(period serviceDailyReportPeriod, stats serviceDailyReportStats) string {
	lines := []string{}
	if period.Range == types.ServiceDailyReportRangeDay {
		lines = append(lines, "- 今天遗漏的事实可继续补记，重新生成会覆盖同一日期日报并保留来源索引。")
	} else {
		lines = append(lines, "- 周/月结论先回看对应日粒度日报，必要时回到具体日期补齐记忆。")
	}
	if stats.OpenActionCount > 0 {
		lines = append(lines, "- 未闭环动作继续留在服务提醒列表，完成后再回写执行结果。")
	}
	if stats.KnowledgeGapCount > 0 {
		lines = append(lines, "- 知识补齐建议进入知识库审核，不直接写入公共知识库。")
	}
	if len(lines) == 0 {
		lines = append(lines, "- 当前没有新的回写动作，保留日报作为可检索工作产物。")
	}
	return strings.Join(lines, "\n")
}

func dailyReportDocInPeriod(doc *types.AgentWorkDoc, period serviceDailyReportPeriod) bool {
	if doc == nil {
		return false
	}
	start := dailyReportDocStartTime(doc, period.Location)
	if start.IsZero() {
		return false
	}
	return !start.Before(period.Start) && start.Before(period.End)
}

func dailyReportDocStartTime(doc *types.AgentWorkDoc, loc *time.Location) time.Time {
	if doc == nil {
		return time.Time{}
	}
	if loc == nil {
		loc = serviceDailyReportLocation("")
	}
	metadata := normalizeJSONMap(doc.Metadata)
	if raw := asString(metadata["period_start"]); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			local := parsed.In(loc)
			return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		}
	}
	date := strings.TrimPrefix(strings.TrimSuffix(doc.DocPath, ".md"), "日报/")
	if strings.Contains(date, "/") {
		return time.Time{}
	}
	parsed, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
}

func buildDailyReportMemoryLinks(
	doc *types.AgentWorkDoc,
	reminders []*types.ServiceReminder,
) []*types.AgentWorkDocMemoryLink {
	links := make([]*types.AgentWorkDocMemoryLink, 0)
	seen := map[string]bool{}
	excerpts := map[string]string{}
	for _, reminder := range reminders {
		for _, evidence := range reminder.MemoryEvidence {
			if evidence.ID != "" && excerpts[evidence.ID] == "" {
				excerpts[evidence.ID] = firstNonEmpty(evidence.Summary, evidence.Title)
			}
		}
		for _, id := range reminder.SourceMemoryIDs {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			links = append(links, &types.AgentWorkDocMemoryLink{
				TenantID:        doc.TenantID,
				DocID:           doc.ID,
				DocPath:         doc.DocPath,
				MemoryID:        id,
				SubjectID:       doc.SubjectID,
				AgentDomain:     types.ServiceAgentDomainDailyReview,
				LinkType:        types.AgentWorkDocLinkTypeEvidence,
				Confidence:      0.86,
				EvidenceExcerpt: trimMax(excerpts[id], 160),
			})
		}
	}
	return links
}

func serviceDailyReportFromDoc(doc *types.AgentWorkDoc) *types.ServiceDailyReport {
	if doc == nil {
		return nil
	}
	metadata := normalizeJSONMap(doc.Metadata)
	subjectCount := firstPositiveInt(asInt(metadata["subject_count"]), asInt(metadata["customer_count"]))
	return &types.ServiceDailyReport{
		ID:                   doc.ID,
		Title:                doc.Title,
		Summary:              firstNonEmpty(asString(metadata["summary"]), contentExcerpt(doc.Content, doc.Title)),
		Content:              doc.Content,
		Range:                firstNonEmpty(asString(metadata["report_range"]), inferDailyReportRangeFromPath(doc.DocPath)),
		Stage:                firstNonEmpty(asString(metadata["stage"]), "已生成"),
		StageKey:             firstNonEmpty(asString(metadata["stage_key"]), "formed"),
		Updated:              firstNonEmpty(asString(metadata["updated"]), formatMonthDay(doc.UpdatedAt)),
		ActionCount:          asInt(metadata["action_count"]),
		CustomerCount:        subjectCount,
		SubjectCount:         subjectCount,
		OpenActionCount:      asInt(metadata["open_action_count"]),
		ClosedActionCount:    asInt(metadata["closed_action_count"]),
		HighRiskCount:        asInt(metadata["high_risk_count"]),
		KnowledgeGapCount:    asInt(metadata["knowledge_gap_count"]),
		SourceMemoryCount:    asInt(metadata["source_memory_count"]),
		DailySourceCount:     asInt(metadata["daily_source_count"]),
		EvidenceCompleteRate: asInt(metadata["evidence_complete_rate"]),
		ProfileID:            firstNonEmpty(asString(metadata["profile_id"]), doc.ProfileID),
		ProfileName:          asString(metadata["profile_name"]),
		MemoryScope:          asString(metadata["memory_scope"]),
		CanSupplement:        asBool(metadata["can_supplement"]),
		Chips:                cleanStringArray(asStringList(metadata["chips"]), 6, 16),
		SourceMemoryIDs:      cleanStringArray(doc.SourceMemoryIDs, 100, 64),
		Metadata:             metadata,
		CreatedAt:            doc.CreatedAt,
		UpdatedAt:            doc.UpdatedAt,
	}
}

func inferDailyReportRangeFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "日报/week/"):
		return types.ServiceDailyReportRangeWeek
	case strings.HasPrefix(path, "日报/month/"):
		return types.ServiceDailyReportRangeMonth
	default:
		return types.ServiceDailyReportRangeDay
	}
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case jsonNumber:
		parsed, _ := strconv.Atoi(v.String())
		return parsed
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(v))
		return parsed
	default:
		return 0
	}
}

func asBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(v))
		return parsed
	default:
		return false
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

type jsonNumber interface {
	String() string
}

func serviceAgentDomainLabel(domain string) string {
	switch domain {
	case types.ServiceAgentDomainMemoryRouter:
		return "记忆路由器"
	case types.ServiceAgentDomainLeadIntake:
		return "线索录入"
	case types.ServiceAgentDomainSalesConsulting:
		return "招生咨询"
	case types.ServiceAgentDomainCustomerService:
		return "客户服务"
	case types.ServiceAgentDomainScheduling:
		return "排课调课"
	case types.ServiceAgentDomainAfterSaleRisk:
		return "售后风险"
	case types.ServiceAgentDomainDailyReview:
		return "日报复盘"
	default:
		return "服务助理"
	}
}

func defaultWorkDocDirectory(domain string) string {
	switch domain {
	case types.ServiceAgentDomainLeadIntake, types.ServiceAgentDomainSalesConsulting:
		return "线索/"
	case types.ServiceAgentDomainScheduling:
		return "排课/"
	case types.ServiceAgentDomainAfterSaleRisk:
		return "售后风险/"
	case types.ServiceAgentDomainDailyReview:
		return "日报/"
	default:
		return "客户/"
	}
}

func actionTypeForDomain(domain string) string {
	switch domain {
	case types.ServiceAgentDomainScheduling:
		return "schedule_confirm"
	case types.ServiceAgentDomainLeadIntake:
		return "lead_follow_up"
	case types.ServiceAgentDomainAfterSaleRisk:
		return "risk_close_loop"
	default:
		return "follow_up"
	}
}

var (
	servicePrimaryKeywords    = []string{"试听", "体验课", "公开课", "咨询", "到访", "邀约", "意向", "报名", "定金", "合同", "付款", "成交", "续费", "续课", "剩余课次", "到期", "跟进", "回访", "转介绍", "老带新", "活动邀请", "售后", "投诉", "反馈", "退费", "退款", "餐食", "午睡", "请假", "适应", "排课", "调课", "补课"}
	serviceSupportingKeywords = []string{"价格", "费用", "学费", "优惠", "预算", "顾虑", "异议"}
	serviceContextKeywords    = []string{"客户", "家长", "学员", "学生", "孩子", "联系人", "试听", "咨询", "报名", "回访"}
	serviceKeywords           = []string{"客户", "家长", "学员", "试听", "续费", "报名", "咨询", "价格", "顾虑", "异议", "跟进", "回访", "转介绍", "商机", "招生", "成交", "微信", "企微", "排课", "调课", "请假", "补课"}
	serviceMetadataKeys       = []string{"stage", "sales_stage", "salesStage", "service_stage", "serviceStage", "risk_label", "riskLabel", "risk", "concern", "next_action", "nextAction", "follow_up_action", "followUpAction", "next_follow_up_at", "nextFollowUpAt", "follow_up_at", "followUpAt", "due_at", "dueAt", "lead_status", "leadStatus", "deal_status", "dealStatus"}
	serviceSummaryKeys        = []string{"summary", "source_summary", "sourceSummary", "description", "abstract"}
	serviceTranscriptKeys     = []string{"transcript", "transcription_text", "transcriptionText", "transcription_result", "transcriptionResult", "asr_text", "asrText", "speech_text", "speechText", "raw_transcript", "rawTranscript", "original_text", "originalText"}
	serviceTextMetadataKeys   = []string{"note_markdown", "noteMarkdown", "customer_name", "customerName", "parent_name", "parentName", "contact_name", "contactName", "lead_name", "leadName", "student_name", "studentName", "learner_name", "learnerName", "child_name", "childName", "channel", "source_channel", "sourceChannel", "decision_role", "decisionRole", "decision_maker", "decisionMaker"}
	serviceListMetadataKeys   = []string{"tags", "agent_domains", "memory_signals"}
	serviceSearchMetadataKeys = append(append(append(append([]string{}, serviceMetadataKeys...), serviceSummaryKeys...), serviceTranscriptKeys...), serviceTextMetadataKeys...)

	customerDirectRe = regexp.MustCompile(`(?:客户|家长|联系人)[:：]\s*([^\s，,。；;\n]{2,18})`)
	leadDirectRe     = regexp.MustCompile(`线索[:：]\s*([^\s，,。；;\n]{2,18})`)
	customerSuffixRe = regexp.MustCompile(`([\p{Han}A-Za-z0-9]{1,12}(?:家长|妈妈|爸爸))`)
	studentDirectRe  = regexp.MustCompile(`(?:学员|学生|孩子)[:：]\s*([^\s，,。；;\n]{2,12})`)
	cutPunctRe       = regexp.MustCompile(`[，,。；;：:].*$`)
)

func isServiceMemory(memory *types.OrganizeMemory) bool {
	if memory == nil || extractCustomerName(memory) == "" {
		return false
	}
	text := memorySearchText(memory)
	return hasServiceBusinessSignal(memory.Metadata, text)
}

func hasServiceBusinessSignal(metadata types.JSONMap, text string) bool {
	if firstMetadataString(metadata, serviceMetadataKeys...) != "" {
		return true
	}
	hasPrimary := containsAny(text, servicePrimaryKeywords)
	hasSupporting := containsAny(text, serviceSupportingKeywords)
	hasContext := containsAny(text, serviceContextKeywords)
	return hasPrimary || (hasSupporting && hasContext)
}

func memorySearchText(memory *types.OrganizeMemory) string {
	if memory == nil {
		return ""
	}
	parts := []string{
		memory.Title,
		stripServiceMarkup(memory.Content),
		memory.Source,
	}
	parts = append(parts, serviceMetadataTextParts(memory.Metadata)...)
	return strings.Join(parts, "\n")
}

func serviceMetadataTextParts(metadata types.JSONMap) []string {
	if len(metadata) == 0 {
		return nil
	}
	parts := make([]string, 0, len(serviceSearchMetadataKeys)+len(serviceListMetadataKeys))
	for _, key := range serviceSearchMetadataKeys {
		if value := asString(metadata[key]); value != "" {
			parts = append(parts, stripServiceMarkup(value))
		}
	}
	for _, key := range serviceListMetadataKeys {
		values := asStringList(metadata[key])
		if len(values) > 0 {
			parts = append(parts, strings.Join(values, " "))
		}
	}
	return parts
}

func extractCustomerName(memory *types.OrganizeMemory) string {
	metadata := memory.Metadata
	for _, key := range []string{"customer_name", "customerName", "parent_name", "parentName", "contact_name", "contactName", "lead_name", "leadName"} {
		if value := asString(metadata[key]); value != "" {
			return normalizeCustomerName(value)
		}
	}
	text := memorySearchText(memory)
	if match := customerDirectRe.FindStringSubmatch(text); len(match) > 1 {
		return normalizeCustomerName(match[1])
	}
	if match := leadDirectRe.FindStringSubmatch(text); len(match) > 1 && containsAny(match[1], []string{"家长", "妈妈", "爸爸", "客户", "联系人"}) {
		return normalizeCustomerName(match[1])
	}
	if match := customerSuffixRe.FindStringSubmatch(text); len(match) > 1 {
		return normalizeCustomerName(match[1])
	}
	return ""
}

func extractStudentName(memory *types.OrganizeMemory, customerName string) string {
	metadata := memory.Metadata
	for _, key := range []string{"student_name", "studentName", "learner_name", "learnerName", "child_name", "childName"} {
		if value := asString(metadata[key]); value != "" {
			return value
		}
	}
	text := memorySearchText(memory)
	if match := studentDirectRe.FindStringSubmatch(text); len(match) > 1 {
		return match[1]
	}
	if strings.HasSuffix(customerName, "家长") {
		return strings.TrimSuffix(customerName, "家长")
	}
	return "待补充"
}

func normalizeCustomerName(value string) string {
	return strings.TrimSpace(cutPunctRe.ReplaceAllString(value, ""))
}

func inferStage(text string) string {
	switch {
	case regexp.MustCompile(`续费|续课|剩余课次|到期`).MatchString(text):
		return "续费服务"
	case regexp.MustCompile(`排课|调课|请假|补课|老师|教室`).MatchString(text):
		return "排课调课"
	case regexp.MustCompile(`试听|到访|体验课|公开课|咨询`).MatchString(text):
		return "售前试听"
	case regexp.MustCompile(`报名|定金|合同|成单|成交|付款`).MatchString(text):
		return "报名确认"
	case regexp.MustCompile(`投诉|售后|餐食|午睡|反馈|不满`).MatchString(text):
		return "在园服务"
	case regexp.MustCompile(`转介绍|推荐|老带新|活动邀请`).MatchString(text):
		return "转介绍"
	default:
		return "客户跟进"
	}
}

func inferChannel(text string) string {
	switch {
	case strings.Contains(text, "企微"):
		return "企微"
	case strings.Contains(text, "微信"):
		return "微信"
	case strings.Contains(text, "电话"):
		return "电话"
	case strings.Contains(text, "面谈") || strings.Contains(text, "到访"):
		return "线下"
	default:
		return "记忆"
	}
}

func inferDecisionRole(text string) string {
	switch {
	case regexp.MustCompile(`妈妈|母亲`).MatchString(text):
		return "妈妈主沟通"
	case regexp.MustCompile(`爸爸|父亲`).MatchString(text):
		return "爸爸主沟通"
	case regexp.MustCompile(`父母|夫妻|双方`).MatchString(text):
		return "父母共同决策"
	default:
		return "决策人待补充"
	}
}

func inferRiskLabel(text string) string {
	switch {
	case regexp.MustCompile(`投诉|不满|退款|退费|差评`).MatchString(text):
		return "售后风险"
	case regexp.MustCompile(`价格|费用|优惠|太贵|预算`).MatchString(text):
		return "价格顾虑"
	case regexp.MustCompile(`适应|焦虑|哭|不习惯`).MatchString(text):
		return "适应焦虑"
	case regexp.MustCompile(`续费|续课|剩余课次|到期`).MatchString(text):
		return "续费窗口"
	case regexp.MustCompile(`未回复|没回复|超过|逾期|拖延`).MatchString(text):
		return "未闭环"
	default:
		return "待判断"
	}
}

func inferPriority(text, riskLabel string) string {
	if regexp.MustCompile(`今天|上午|下午|今晚|投诉|不满|退款|退费|差评|未回复|逾期|48\s*小时`).MatchString(text) {
		return types.ServicePriorityHigh
	}
	if riskLabel != "待判断" || regexp.MustCompile(`本周|周五|续费|试听|报名|请假|补课`).MatchString(text) {
		return types.ServicePriorityMedium
	}
	return types.ServicePriorityLow
}

func inferNextAction(stage, riskLabel string) string {
	switch {
	case riskLabel == "售后风险":
		return "先确认处理结果并补齐服务闭环"
	case riskLabel == "价格顾虑":
		return "准备价值证明后再回应价格问题"
	case riskLabel == "适应焦虑":
		return "补充孩子观察记录并安排低压力回访"
	case stage == "续费服务":
		return "生成阶段成长回顾后再进入续费沟通"
	case stage == "售前试听":
		return "完成试听后回访并确认下一步安排"
	case stage == "排课调课":
		return "确认可选排课资源后回复家长"
	case stage == "转介绍":
		return "用活动资料做一次轻触达"
	default:
		return "确认客户状态并补一条下一步记忆"
	}
}

func inferPrimaryAction(stage, riskLabel string) string {
	switch {
	case riskLabel == "售后风险":
		return "先回应家长感受和处理进展，再补老师观察，不展开长解释。"
	case riskLabel == "价格顾虑":
		return "先把课程价值和孩子变化讲清楚，再讨论价格或优惠边界。"
	case riskLabel == "适应焦虑":
		return "先给到具体观察，再说明适应节奏，最后确认是否需要老师补充沟通。"
	case stage == "续费服务":
		return "先整理阶段变化和下一阶段目标，再约一次低压力沟通。"
	case stage == "排课调课":
		return "先核对老师、教室和家长可选时间，再输出待确认安排。"
	default:
		return "先确认记忆里的真实事实，再生成可发送话术和下一步。"
	}
}

func inferAvoidAction(riskLabel string) string {
	switch riskLabel {
	case "价格顾虑":
		return "不要先抛优惠或承诺结果，避免把沟通变成纯价格谈判。"
	case "售后风险":
		return "不要先解释原因或转移责任，先确认问题是否真正闭环。"
	case "适应焦虑":
		return "不要承诺马上适应，也不要用泛泛安慰替代具体观察。"
	default:
		return "不要把未确认的梳理结果直接当作客户事实落地。"
	}
}

func inferReplyDraft(customerName, stage, riskLabel string) string {
	switch {
	case riskLabel == "价格顾虑":
		return fmt.Sprintf("您好，%s这边我先把孩子目前的学习变化和后续安排梳理一下，再跟您说明费用和可选方案，方便您一起判断是否合适。", customerName)
	case riskLabel == "售后风险":
		return "您好，之前反馈的问题我们已经重点跟进。我想先跟您确认这两天的改善感受，再把老师观察到的情况同步给您。"
	case stage == "续费服务":
		return "这段时间孩子有几处比较明确的变化，我先整理成阶段回顾发您，也想听听您对下一阶段最关注的目标。"
	case stage == "排课调课":
		return "您好，我先帮您确认老师和可选时段，确认资源后再给您两个选择，避免时间来回调整。"
	default:
		return "您好，我根据最近记录把当前情况整理了一下，想跟您确认一个下一步安排，避免遗漏您的重点关注。"
	}
}

func inferDue(raw, priority string) (*time.Time, string) {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			utc := parsed.UTC()
			return &utc, parsed.Local().Format("1月2日 15:04")
		}
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			utc := parsed.UTC()
			return &utc, parsed.Local().Format("1月2日 15:04")
		}
		return nil, raw
	}
	switch priority {
	case types.ServicePriorityHigh:
		return nil, "今天"
	case types.ServicePriorityMedium:
		return nil, "本周"
	default:
		return nil, "待定"
	}
}

func domainForText(text, stage, riskLabel string) string {
	if regexp.MustCompile(`排课|调课|请假|补课|老师|教室|时间`).MatchString(text) || stage == "排课调课" {
		return types.ServiceAgentDomainScheduling
	}
	if riskLabel == "售后风险" {
		return types.ServiceAgentDomainCustomerService
	}
	if regexp.MustCompile(`试听|体验课|咨询|报名|邀约|价格|异议`).MatchString(text) || stage == "售前试听" || stage == "报名确认" {
		return types.ServiceAgentDomainSalesConsulting
	}
	return types.ServiceAgentDomainCustomerService
}

func deriveMemorySignals(text string, tags []string) []string {
	matched := make([]string, 0, len(tags)+len(serviceKeywords))
	matched = append(matched, tags...)
	for _, keyword := range serviceKeywords {
		if strings.Contains(text, keyword) {
			matched = append(matched, keyword)
		}
	}
	return cleanStringArray(matched, 6, 64)
}

func salesHighlightsFrom(stage, riskLabel, text string) []string {
	highlights := []string{}
	if regexp.MustCompile(`试听|到访|体验课|公开课|咨询`).MatchString(text) || stage == "售前试听" {
		highlights = append(highlights, "出现售前接触信号，适合尽快回访并确认下一步安排。")
	}
	if riskLabel == "价格顾虑" {
		highlights = append(highlights, "客户对价格敏感，先补价值证明和孩子变化，再谈费用。")
	}
	if riskLabel == "适应焦虑" {
		highlights = append(highlights, "客户关注适应情况，用具体观察降低不确定感。")
	}
	if riskLabel == "续费窗口" || stage == "续费服务" {
		highlights = append(highlights, "已经进入续费窗口，先做阶段成长回顾再推进判断。")
	}
	if riskLabel == "售后风险" || stage == "在园服务" {
		highlights = append(highlights, "存在售后或服务反馈，先闭环处理结果再继续后续沟通。")
	}
	if regexp.MustCompile(`转介绍|推荐|老带新|活动邀请`).MatchString(text) || stage == "转介绍" {
		highlights = append(highlights, "适合用活动资料轻触达，不直接索要转介绍。")
	}
	if regexp.MustCompile(`报名|定金|合同|付款|成交`).MatchString(text) || stage == "报名确认" {
		highlights = append(highlights, "出现报名确认信号，优先补齐决策人、时间和付款/合同状态。")
	}
	if len(highlights) == 0 {
		highlights = append(highlights, "已有客户服务信息，先补齐客户状态、决策人和下一步。")
	}
	return cleanStringArray(highlights, 4, 160)
}

func contentExcerpt(value, fallback string) string {
	text := stripServiceMarkup(value)
	if text == "" {
		return fallback
	}
	return trimMax(text, 86)
}

func stripServiceMarkup(value string) string {
	replacer := strings.NewReplacer(
		"<br>", " ", "<br/>", " ", "<br />", " ",
	)
	value = replacer.Replace(value)
	var builder strings.Builder
	inTag := false
	for _, r := range value {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
			builder.WriteRune(' ')
		default:
			if !inTag {
				builder.WriteRune(r)
			}
		}
	}
	text := strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'",
		"#", " ", ">", " ", "*", " ", "_", " ", "`", " ", "~", " ", "-", " ",
	).Replace(builder.String())
	return strings.Join(strings.Fields(text), " ")
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func firstMetadataString(metadata types.JSONMap, keys ...string) string {
	for _, key := range keys {
		if value := asString(metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func asStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return cleanStringArray(v, 100, 128)
	case types.StringArray:
		return cleanStringArray([]string(v), 100, 128)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, asString(item))
		}
		return cleanStringArray(out, 100, 128)
	case string:
		return cleanStringArray(strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == '，' || r == '、' || r == ' ' || r == '\n' || r == '\t'
		}), 100, 128)
	default:
		return nil
	}
}

func mergeJSONMap(base types.JSONMap, pairs types.JSONMap) types.JSONMap {
	out := normalizeJSONMap(base)
	for key, value := range pairs {
		out[key] = value
	}
	return out
}

func deterministicServiceID(parts ...string) string {
	joined := strings.Join(parts, ":")
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(joined)).String()
}

func serviceSubjectKey(customerName, studentName string) string {
	key := strings.ToLower(strings.Join([]string{customerName, studentName}, "|"))
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "，", ",", "：", ":")
	return replacer.Replace(key)
}

func inferRelation(customerName string) string {
	switch {
	case strings.Contains(customerName, "妈妈"):
		return "妈妈"
	case strings.Contains(customerName, "爸爸"):
		return "爸爸"
	case strings.Contains(customerName, "家长"):
		return "家长"
	default:
		return ""
	}
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "待补充"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", `"`, "", "<", "", ">", "", "|", "-")
	return replacer.Replace(value)
}

func formatMonthDay(t time.Time) string {
	if t.IsZero() {
		return "最近"
	}
	return t.Local().Format("1月2日")
}
