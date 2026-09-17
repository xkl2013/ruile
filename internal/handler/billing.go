package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type BillingHandler struct {
	subscriptions interfaces.SubscriptionService
	usage         interfaces.UsageBillingService
	members       interfaces.TenantMemberService
	audit         interfaces.AuditLogService
}

func NewBillingHandler(
	subscriptions interfaces.SubscriptionService,
	usage interfaces.UsageBillingService,
	members interfaces.TenantMemberService,
	audit interfaces.AuditLogService,
) *BillingHandler {
	return &BillingHandler{
		subscriptions: subscriptions,
		usage:         usage,
		members:       members,
		audit:         audit,
	}
}

type updateMemberAllocationRequest struct {
	PeriodPoints  int64 `json:"period_points"`
	BalancePoints int64 `json:"balance_points"`
}

type updateTenantBillingPolicyRequest struct {
	DefaultMemberMonthlyLimitPoints int64  `json:"default_member_monthly_limit_points"`
	MemberOveragePolicy             string `json:"member_overage_policy"`
}

type updateMemberPolicyRequest struct {
	LimitMode          string `json:"limit_mode"`
	MonthlyLimitPoints int64  `json:"monthly_limit_points"`
}

func billingLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		return 50
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func (h *BillingHandler) ListCurrentUsage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	rows, err := h.usage.ListUsageLedgers(ctx, tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list billing usage"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (h *BillingHandler) ListCurrentMemberAllocations(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	rows, err := h.usage.ListMemberAllocations(ctx, tenantID, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list member allocations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (h *BillingHandler) GetCurrentTenantBillingPolicy(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	row, err := h.usage.GetTenantBillingPolicy(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *BillingHandler) UpdateCurrentTenantBillingPolicy(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	var input updateTenantBillingPolicyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid enterprise policy: " + err.Error()})
		return
	}
	limitMicros, ok := pointsToMicros(input.DefaultMemberMonthlyLimitPoints)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "default_member_monthly_limit_points must be a non-negative integer"})
		return
	}
	actorUserID, _ := types.UserIDFromContext(ctx)
	row, err := h.usage.UpdateTenantBillingPolicy(
		ctx,
		tenantID,
		limitMicros,
		strings.TrimSpace(input.MemberOveragePolicy),
		actorUserID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if h.audit != nil {
		details, _ := json.Marshal(map[string]any{
			"default_member_monthly_limit_points": input.DefaultMemberMonthlyLimitPoints,
			"member_overage_policy":               row.MemberOveragePolicy,
		})
		_ = h.audit.Log(ctx, &types.AuditLog{
			TenantID:      tenantID,
			ActorUserID:   actorUserID,
			ActorRole:     string(types.TenantRoleFromContext(ctx)),
			Action:        types.AuditActionEnterpriseBillingPolicyChanged,
			TargetType:    "tenant_billing_policy",
			TargetID:      strconv.FormatUint(tenantID, 10),
			RequestPath:   c.Request.URL.Path,
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			Details:       types.JSON(details),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func pointsToMicros(points int64) (int64, bool) {
	if points < 0 || points > math.MaxInt64/types.PointMicrosPerPoint {
		return 0, false
	}
	return points * types.PointMicrosPerPoint, true
}

func (h *BillingHandler) UpdateCurrentMemberAllocation(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id is required"})
		return
	}
	if h.members == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "workspace membership service unavailable"})
		return
	}
	membership, err := h.members.GetMembership(ctx, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load workspace member"})
		return
	}
	if membership == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "workspace member not found"})
		return
	}
	if membership.Status != types.TenantMemberStatusActive {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "only active members can receive enterprise credits"})
		return
	}

	var input updateMemberAllocationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid member allocation: " + err.Error()})
		return
	}
	periodMicros, ok := pointsToMicros(input.PeriodPoints)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "period_points must be a non-negative integer"})
		return
	}
	balanceMicros, ok := pointsToMicros(input.BalancePoints)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "balance_points must be a non-negative integer"})
		return
	}
	actorUserID, _ := types.UserIDFromContext(ctx)
	row, err := h.usage.UpdateMemberAllocation(
		ctx,
		tenantID,
		userID,
		periodMicros,
		balanceMicros,
		actorUserID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if h.audit != nil {
		details, _ := json.Marshal(map[string]any{
			"period_points":  input.PeriodPoints,
			"balance_points": input.BalancePoints,
			"period_start":   row.PeriodStartAt,
			"period_end":     row.PeriodEndAt,
		})
		_ = h.audit.Log(ctx, &types.AuditLog{
			TenantID:      tenantID,
			ActorUserID:   actorUserID,
			ActorRole:     string(types.TenantRoleFromContext(ctx)),
			Action:        types.AuditActionMemberAllocationChanged,
			TargetType:    "tenant_member",
			TargetID:      row.ID,
			TargetUserID:  userID,
			RequestPath:   c.Request.URL.Path,
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			Details:       types.JSON(details),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *BillingHandler) UpdateCurrentMemberPolicy(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id is required"})
		return
	}
	if h.members == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "workspace membership service unavailable"})
		return
	}
	membership, err := h.members.GetMembership(ctx, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load workspace member"})
		return
	}
	if membership == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "workspace member not found"})
		return
	}
	if membership.Status != types.TenantMemberStatusActive {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "only active members can receive an enterprise usage policy"})
		return
	}

	var input updateMemberPolicyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid member usage policy: " + err.Error()})
		return
	}
	monthlyLimitMicros, ok := pointsToMicros(input.MonthlyLimitPoints)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "monthly_limit_points must be a non-negative integer"})
		return
	}
	actorUserID, _ := types.UserIDFromContext(ctx)
	row, err := h.usage.UpdateMemberPolicy(
		ctx,
		tenantID,
		userID,
		strings.TrimSpace(input.LimitMode),
		monthlyLimitMicros,
		actorUserID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if h.audit != nil {
		details, _ := json.Marshal(map[string]any{
			"limit_mode":           row.LimitMode,
			"monthly_limit_points": input.MonthlyLimitPoints,
			"period_start":         row.PeriodStartAt,
			"period_end":           row.PeriodEndAt,
		})
		_ = h.audit.Log(ctx, &types.AuditLog{
			TenantID:      tenantID,
			ActorUserID:   actorUserID,
			ActorRole:     string(types.TenantRoleFromContext(ctx)),
			Action:        types.AuditActionMemberUsagePolicyChanged,
			TargetType:    "tenant_member",
			TargetID:      row.ID,
			TargetUserID:  userID,
			RequestPath:   c.Request.URL.Path,
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			Details:       types.JSON(details),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *BillingHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	overview, err := h.subscriptions.GetOverview(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"tenant_id": tenantID})
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load billing overview"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *BillingHandler) ListPlans(c *gin.Context) {
	rows, err := h.subscriptions.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list billing plans"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListPrices(c *gin.Context) {
	rows, err := h.subscriptions.ListPrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list billing prices"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListSubscriptions(c *gin.Context) {
	rows, err := h.subscriptions.ListSubscriptions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list tenant subscriptions"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListCreditAccounts(c *gin.Context) {
	rows, err := h.subscriptions.ListCreditAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list tenant credit accounts"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListModelPrices(c *gin.Context) {
	rows, err := h.usage.ListModelPrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list model prices"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) CreateModelPriceVersion(c *gin.Context) {
	var input types.BillingModelPriceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid model price: " + err.Error()})
		return
	}
	row, err := h.usage.CreateModelPriceVersion(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (h *BillingHandler) ListUsageLedgers(c *gin.Context) {
	var tenantID uint64
	if raw := c.Query("tenant_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
			return
		}
		tenantID = parsed
	}
	rows, err := h.usage.ListUsageLedgers(c.Request.Context(), tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list usage ledgers"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListMemberAllocations(c *gin.Context) {
	var tenantID uint64
	if raw := c.Query("tenant_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
			return
		}
		tenantID = parsed
	}
	rows, err := h.usage.ListMemberAllocations(c.Request.Context(), tenantID, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list member allocations"})
		return
	}
	c.JSON(http.StatusOK, rows)
}
