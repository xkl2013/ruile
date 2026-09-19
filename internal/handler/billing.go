package handler

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type BillingHandler struct {
	subscriptions interfaces.SubscriptionService
	usage         interfaces.UsageBillingService
	operations    interfaces.BillingOperationsService
	members       interfaces.TenantMemberService
	audit         interfaces.AuditLogService
}

func NewBillingHandler(
	subscriptions interfaces.SubscriptionService,
	usage interfaces.UsageBillingService,
	operations interfaces.BillingOperationsService,
	members interfaces.TenantMemberService,
	audit interfaces.AuditLogService,
) *BillingHandler {
	return &BillingHandler{
		subscriptions: subscriptions,
		usage:         usage,
		operations:    operations,
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

type updateMemberPoliciesRequest struct {
	UserIDs            []string `json:"user_ids" binding:"required"`
	LimitMode          string   `json:"limit_mode"`
	MonthlyLimitPoints int64    `json:"monthly_limit_points"`
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
	var (
		rows []*types.BillingUsageLedgerSummary
		err  error
	)
	if types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin) {
		rows, err = h.usage.ListUsageLedgers(ctx, tenantID, billingLimit(c))
	} else {
		actorUserID, actorOK := types.UserIDFromContext(ctx)
		if !actorOK {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user context is required"})
			return
		}
		rows, err = h.usage.ListUsageLedgersByActor(ctx, tenantID, actorUserID, billingLimit(c))
	}
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

func (h *BillingHandler) UpdateCurrentMemberPolicies(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	if h.members == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "workspace membership service unavailable"})
		return
	}
	var input updateMemberPoliciesRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid member usage policies: " + err.Error()})
		return
	}
	if len(input.UserIDs) == 0 || len(input.UserIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_ids must contain between 1 and 100 members"})
		return
	}
	userIDs := make([]string, 0, len(input.UserIDs))
	seen := make(map[string]struct{}, len(input.UserIDs))
	for _, raw := range input.UserIDs {
		userID := strings.TrimSpace(raw)
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id must not be empty"})
			return
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		membership, err := h.members.GetMembership(ctx, userID, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load workspace member"})
			return
		}
		if membership == nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "workspace member not found: " + userID})
			return
		}
		if membership.Status != types.TenantMemberStatusActive {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "only active members can receive an enterprise usage policy"})
			return
		}
		userIDs = append(userIDs, userID)
	}
	monthlyLimitMicros, valid := pointsToMicros(input.MonthlyLimitPoints)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "monthly_limit_points must be a non-negative integer"})
		return
	}
	actorUserID, _ := types.UserIDFromContext(ctx)
	rows, err := h.usage.UpdateMemberPolicies(
		ctx,
		tenantID,
		userIDs,
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
			"user_ids":             userIDs,
			"limit_mode":           strings.TrimSpace(input.LimitMode),
			"monthly_limit_points": input.MonthlyLimitPoints,
		})
		_ = h.audit.Log(ctx, &types.AuditLog{
			TenantID:      tenantID,
			ActorUserID:   actorUserID,
			ActorRole:     string(types.TenantRoleFromContext(ctx)),
			Action:        types.AuditActionMemberUsagePolicyChanged,
			TargetType:    "tenant_members",
			TargetID:      "batch",
			RequestPath:   c.Request.URL.Path,
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			Details:       types.JSON(details),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
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
	if overview.SpaceType == types.SpaceTypeOrganization &&
		!types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin) {
		if err := h.hideEnterprisePoolForMember(ctx, overview); err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{"tenant_id": tenantID})
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load member billing usage"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *BillingHandler) ListPublicCatalog(c *gin.Context) {
	ctx := c.Request.Context()
	overview, err := h.currentBillingOverview(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load billing catalog"})
		return
	}
	plans, err := h.subscriptions.ListPlans(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list billing plans"})
		return
	}
	prices, err := h.subscriptions.ListPrices(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list billing prices"})
		return
	}
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "billing catalog is unavailable"})
		return
	}
	items, err := h.operations.ListPurchaseItems(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list purchase items"})
		return
	}

	allowedPlanIDs := make(map[string]struct{})
	publicPlans := make([]*types.BillingPlan, 0, len(plans))
	for _, plan := range plans {
		if plan == nil || plan.Status != types.BillingStatusActive || !plan.IsPublic {
			continue
		}
		if overview.SpaceType != types.SpaceTypeLegacy && plan.SpaceType != overview.SpaceType {
			continue
		}
		allowedPlanIDs[plan.ID] = struct{}{}
		publicPlans = append(publicPlans, plan)
	}
	publicPrices := make([]*types.BillingPrice, 0, len(prices))
	for _, price := range prices {
		if price == nil || price.Status != types.BillingStatusActive {
			continue
		}
		if _, ok := allowedPlanIDs[price.PlanID]; ok {
			publicPrices = append(publicPrices, price)
		}
	}

	edition := string(overview.SpaceType)
	publicItems := make([]*types.BillingPurchaseItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.Status != types.BillingPurchaseItemStatusActive {
			continue
		}
		if item.EditionScope != types.BillingEditionScopeAll &&
			item.EditionScope != edition {
			continue
		}
		itemType := strings.TrimSpace(c.Query("type"))
		if itemType != "" && item.ItemType != itemType {
			continue
		}
		publicItems = append(publicItems, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": types.BillingPublicCatalog{
			Plans:         publicPlans,
			Prices:        publicPrices,
			PurchaseItems: publicItems,
			Payment: types.BillingPaymentConfig{
				Enabled:   false,
				Currency:  "CNY",
				Providers: []*types.BillingPaymentProvider{},
				Reason:    "online payment is not configured",
			},
		},
	})
}

func (h *BillingHandler) ListCurrentPaymentOrders(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "billing operations are unavailable"})
		return
	}
	rows, err := h.operations.ListPaymentOrders(ctx, tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list payment orders"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (h *BillingHandler) GetCurrentPaymentOrder(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "billing operations are unavailable"})
		return
	}
	row, err := h.operations.GetPaymentOrder(ctx, tenantID, c.Param("order_no"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "message": "payment order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *BillingHandler) GetPaymentConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": types.BillingPaymentConfig{
			Enabled:   false,
			Currency:  "CNY",
			Providers: []*types.BillingPaymentProvider{},
			Reason:    "online payment is not configured",
		},
	})
}

func (h *BillingHandler) currentBillingOverview(
	ctx context.Context,
) (*types.BillingOverview, error) {
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errors.New("workspace context is required")
	}
	return h.subscriptions.GetOverview(ctx, tenantID)
}

func (h *BillingHandler) hideEnterprisePoolForMember(
	ctx context.Context,
	overview *types.BillingOverview,
) error {
	if overview == nil {
		return nil
	}
	actorUserID, ok := types.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(actorUserID) == "" {
		return errors.New("user context is required")
	}
	policy, err := h.usage.GetTenantBillingPolicy(ctx, overview.TenantID)
	if err != nil {
		return err
	}
	allocations, err := h.usage.ListMemberAllocations(ctx, overview.TenantID, time.Now().UTC())
	if err != nil {
		return err
	}
	memberUsage := &types.BillingOverviewMemberUsage{
		LimitMode:               types.MemberLimitModeInherit,
		MonthlyLimitPointMicros: policy.DefaultMemberMonthlyLimitPointMicros,
		OveragePolicy:           policy.MemberOveragePolicy,
	}
	for _, allocation := range allocations {
		if allocation == nil || allocation.UserID != actorUserID {
			continue
		}
		memberUsage.MemberPolicyID = allocation.ID
		memberUsage.LimitMode = allocation.LimitMode
		memberUsage.MonthlyLimitPointMicros = allocation.EffectiveMonthlyLimitPointMicros
		memberUsage.MonthlyUsedPointMicros = allocation.UsedPointMicros
		memberUsage.OveragePolicy = allocation.EffectiveOveragePolicy
		break
	}
	if memberUsage.LimitMode == types.MemberLimitModeUnlimited {
		memberUsage.MonthlyLimitPointMicros = 0
	} else {
		memberUsage.MonthlyRemainingPointMicros =
			memberUsage.MonthlyLimitPointMicros - memberUsage.MonthlyUsedPointMicros
		if memberUsage.MonthlyRemainingPointMicros < 0 {
			memberUsage.MonthlyRemainingPointMicros = 0
		}
	}
	overview.MemberUsage = memberUsage
	overview.Storage = types.BillingOverviewStorage{Visible: false}
	overview.Credits = types.BillingOverviewCredits{Visible: false}
	return nil
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

func (h *BillingHandler) CreatePlan(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid billing plan: " + err.Error()})
		return
	}
	row, err := h.operations.CreatePlan(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionBillingPlanCreated, "billing_plan", row.ID, 0, map[string]any{
		"code":   row.Code,
		"name":   row.Name,
		"status": row.Status,
	})
	c.JSON(http.StatusCreated, row)
}

func (h *BillingHandler) UpdatePlan(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid billing plan: " + err.Error()})
		return
	}
	row, err := h.operations.UpdatePlan(
		c.Request.Context(),
		strings.TrimSpace(c.Param("id")),
		input,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionBillingPlanUpdated, "billing_plan", row.ID, 0, map[string]any{
		"code":   row.Code,
		"name":   row.Name,
		"status": row.Status,
	})
	c.JSON(http.StatusOK, row)
}

func (h *BillingHandler) CreatePriceVersion(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingPriceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid billing price: " + err.Error()})
		return
	}
	row, err := h.operations.CreatePriceVersion(c.Request.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionBillingPriceCreated, "billing_price", row.ID, 0, map[string]any{
		"plan_id":          row.PlanID,
		"code":             row.Code,
		"billing_interval": row.BillingInterval,
		"amount_minor":     row.AmountMinor,
		"status":           row.Status,
		"is_default":       row.IsDefault,
	})
	c.JSON(http.StatusCreated, row)
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

func (h *BillingHandler) ListServicePrices(c *gin.Context) {
	rows, err := h.usage.ListServicePrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list service prices"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) CreateServicePriceVersion(c *gin.Context) {
	var input types.BillingServicePriceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid service price: " + err.Error()})
		return
	}
	row, err := h.usage.CreateServicePriceVersion(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (h *BillingHandler) ListPurchaseItems(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	rows, err := h.operations.ListPurchaseItems(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list purchase items"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) CreatePurchaseItem(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingPurchaseItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid purchase item: " + err.Error()})
		return
	}
	row, err := h.operations.CreatePurchaseItem(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionPurchaseItemCreated, "billing_purchase_item", row.ID, 0, map[string]any{
		"code":      row.Code,
		"item_type": row.ItemType,
		"status":    row.Status,
	})
	c.JSON(http.StatusCreated, row)
}

func (h *BillingHandler) UpdatePurchaseItem(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingPurchaseItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid purchase item: " + err.Error()})
		return
	}
	row, err := h.operations.UpdatePurchaseItem(
		c.Request.Context(),
		strings.TrimSpace(c.Param("id")),
		input,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionPurchaseItemUpdated, "billing_purchase_item", row.ID, 0, map[string]any{
		"code":      row.Code,
		"item_type": row.ItemType,
		"status":    row.Status,
	})
	c.JSON(http.StatusOK, row)
}

func (h *BillingHandler) ListPaymentOrders(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	tenantID, err := billingTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
		return
	}
	rows, err := h.operations.ListPaymentOrders(c.Request.Context(), tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list payment orders"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) CreateManualPaymentOrder(c *gin.Context) {
	if h.operations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "billing operations are unavailable"})
		return
	}
	var input types.BillingManualOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid manual billing operation: " + err.Error()})
		return
	}
	input.ActorUserID, _ = types.UserIDFromContext(c.Request.Context())
	row, err := h.operations.CreateManualPaymentOrder(c.Request.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}
	h.emitSystemBillingAudit(c, types.AuditActionManualBillingOrderCreated, "billing_payment_order", row.OrderNo, row.TenantID, map[string]any{
		"order_type":          row.OrderType,
		"provider":            row.Provider,
		"credit_point_micros": row.CreditPointMicros,
		"storage_quota_bytes": row.StorageQuotaBytes,
		"billing_interval":    row.BillingInterval,
		"plan_id":             row.PlanID,
		"item_id":             row.ItemID,
	})
	c.JSON(http.StatusCreated, row)
}

func billingTenantID(c *gin.Context) (uint64, error) {
	raw := strings.TrimSpace(c.Query("tenant_id"))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseUint(raw, 10, 64)
}

func (h *BillingHandler) emitSystemBillingAudit(
	c *gin.Context,
	action types.AuditAction,
	targetType, targetID string,
	targetTenantID uint64,
	detailsMap map[string]any,
) {
	if h.audit == nil {
		return
	}
	ctx := c.Request.Context()
	actorUserID, _ := types.UserIDFromContext(ctx)
	details, _ := json.Marshal(detailsMap)
	_ = h.audit.Log(ctx, &types.AuditLog{
		TenantID:      0,
		ActorUserID:   actorUserID,
		ActorRole:     "system_admin",
		Action:        action,
		TargetType:    targetType,
		TargetID:      targetID,
		RequestPath:   c.Request.URL.Path,
		RequestMethod: c.Request.Method,
		Outcome:       types.AuditOutcomeSuccess,
		Details:       types.JSON(details),
	})
}

func (h *BillingHandler) ListUsageLedgers(c *gin.Context) {
	tenantID, err := billingTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
		return
	}
	rows, err := h.usage.ListUsageLedgers(c.Request.Context(), tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list usage ledgers"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListMemberAllocations(c *gin.Context) {
	tenantID, err := billingTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
		return
	}
	rows, err := h.usage.ListMemberAllocations(c.Request.Context(), tenantID, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list member allocations"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *BillingHandler) ListStorageTransactions(c *gin.Context) {
	tenantID, err := billingTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tenant_id"})
		return
	}
	rows, err := h.usage.ListStorageTransactions(c.Request.Context(), tenantID, billingLimit(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list storage transactions"})
		return
	}
	c.JSON(http.StatusOK, rows)
}
