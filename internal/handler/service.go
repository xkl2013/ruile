package handler

import (
	stderrors "errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	appsvc "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	service interfaces.ServiceService
}

func NewServiceHandler(svc interfaces.ServiceService) *ServiceHandler {
	return &ServiceHandler{service: svc}
}

func serviceScope(c *gin.Context) (uint64, string, bool) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("workspace ID not found"))
		return 0, "", false
	}
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user ID not found"))
		return 0, "", false
	}
	return tenantID, userID, true
}

func (h *ServiceHandler) GetBootstrap(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.service.GetBootstrap(ctx, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.service.RefreshUserService(ctx, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) ExtractMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.service.ExtractMemory(ctx, tenantID, userID, c.Param("memory_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) ListDailyReports(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.dailyReportListQuery(c)
	if !ok {
		return
	}
	reports, total, err := h.service.ListDailyReports(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(reports, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetDailyReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	report, err := h.service.GetDailyReport(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ServiceHandler) GenerateDailyReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceDailyReportInput
	if err := c.ShouldBindJSON(&req); err != nil && !stderrors.Is(err, io.EOF) {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	if req.Range == "" {
		req.Range = strings.TrimSpace(c.Query("range"))
	}
	if req.Date == "" {
		req.Date = strings.TrimSpace(c.Query("date"))
	}
	if req.Timezone == "" {
		req.Timezone = strings.TrimSpace(c.Query("timezone"))
	}
	report, err := h.service.GenerateDailyReport(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ServiceHandler) ListCustomerSpaces(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.customerSpaceListQuery(c)
	if !ok {
		return
	}
	spaces, total, err := h.service.ListCustomerSpaces(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(spaces, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetCustomerSpace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	space, err := h.service.GetCustomerSpace(ctx, tenantID, userID, c.Param("id"), strings.TrimSpace(c.Query("profile_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": space})
}

func (h *ServiceHandler) ListAgentTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.service.ListAgentTemplates(c.Request.Context())})
}

func (h *ServiceHandler) ListReminders(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.listQuery(c)
	if !ok {
		return
	}
	query.Status = strings.TrimSpace(c.Query("status"))
	query.AgentDomain = strings.TrimSpace(c.Query("agent_domain"))
	query.ProfileID = strings.TrimSpace(c.Query("profile_id"))
	reminders, total, err := h.service.ListReminders(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(reminders, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetReminder(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	reminder, err := h.service.GetReminder(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceHandler) UpdateReminderStatus(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceReminderStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	reminder, err := h.service.UpdateReminderStatus(ctx, tenantID, userID, c.Param("id"), req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceHandler) ListActionDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	drafts, err := h.service.ListActionDrafts(ctx, tenantID, userID, strings.TrimSpace(c.Query("reminder_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": drafts})
}

func (h *ServiceHandler) CreateActionDraft(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.AgentActionDraftInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.CreateActionDraft(ctx, tenantID, userID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ServiceHandler) UpdateActionDraftStatus(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.AgentActionDraftStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.UpdateActionDraftStatus(ctx, tenantID, userID, c.Param("id"), req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ServiceHandler) ListWorkProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	profiles, err := h.service.ListWorkProfiles(ctx, tenantID, strings.TrimSpace(c.Query("user_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profiles})
}

func (h *ServiceHandler) CreateWorkProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceWorkProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	profile, err := h.service.CreateWorkProfile(ctx, tenantID, operatorUserID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceHandler) UpdateWorkProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceWorkProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	profile, err := h.service.UpdateWorkProfile(ctx, tenantID, operatorUserID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceHandler) ListAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	settings, err := h.service.ListAgentSettings(ctx, tenantID, c.Param("id"), strings.TrimSpace(c.Query("enabled")) == "true")
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

func (h *ServiceHandler) ReplaceAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.WorkProfileAgentSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	settings, err := h.service.ReplaceAgentSettings(ctx, tenantID, operatorUserID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

func (h *ServiceHandler) listQuery(c *gin.Context) (types.ServiceListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}, true
}

func (h *ServiceHandler) dailyReportListQuery(c *gin.Context) (types.ServiceDailyReportListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceDailyReportListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceDailyReportListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceDailyReportListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: strings.TrimSpace(c.Query("profile_id")),
		Range:     strings.TrimSpace(c.Query("range")),
		Keyword:   keyword,
		Page:      page,
		PageSize:  pageSize,
	}, true
}

func (h *ServiceHandler) customerSpaceListQuery(c *gin.Context) (types.ServiceCustomerSpaceListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceCustomerSpaceListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceCustomerSpaceListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceCustomerSpaceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: strings.TrimSpace(c.Query("profile_id")),
		Keyword:   keyword,
		Page:      page,
		PageSize:  pageSize,
	}, true
}

func parseServicePagination(c *gin.Context) (int, int, bool) {
	page := 1
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			c.Error(apperrors.NewBadRequestError("page must be a positive integer"))
			return 0, 0, false
		}
		page = parsed
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			c.Error(apperrors.NewBadRequestError("page_size must be a positive integer"))
			return 0, 0, false
		}
		if parsed > 100 {
			parsed = 100
		}
		pageSize = parsed
	}
	return page, pageSize, true
}

func (h *ServiceHandler) handleError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, appsvc.ErrServiceInvalidScope):
		c.Error(apperrors.NewUnauthorizedError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceProfileNameRequired),
		stderrors.Is(err, appsvc.ErrServiceInvalidProfileState),
		stderrors.Is(err, appsvc.ErrServiceInvalidAgentDomain),
		stderrors.Is(err, appsvc.ErrServiceInvalidReportRange),
		stderrors.Is(err, appsvc.ErrServiceInvalidReportDate),
		stderrors.Is(err, appsvc.ErrServiceInvalidStatus),
		stderrors.Is(err, appsvc.ErrServiceProfileNotConfigured):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
