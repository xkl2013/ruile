package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type BillingHandler struct {
	subscriptions interfaces.SubscriptionService
	usage         interfaces.UsageBillingService
}

func NewBillingHandler(
	subscriptions interfaces.SubscriptionService,
	usage interfaces.UsageBillingService,
) *BillingHandler {
	return &BillingHandler{subscriptions: subscriptions, usage: usage}
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
