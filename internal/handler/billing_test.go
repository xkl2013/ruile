package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/types"
)

type stubSubscriptionService struct {
	overview *types.BillingOverview
}

type stubUsageBillingService struct{}

func (s *stubUsageBillingService) BeginModelUsage(
	context.Context,
	types.BillingUsageStartRequest,
) (*types.BillingUsageHandle, error) {
	return nil, nil
}
func (s *stubUsageBillingService) SettleModelUsage(
	context.Context,
	*types.BillingUsageHandle,
	types.BillingModelUsage,
) (*types.TenantUsageLedger, error) {
	return nil, nil
}
func (s *stubUsageBillingService) ReleaseModelUsage(
	context.Context,
	*types.BillingUsageHandle,
	string,
) error {
	return nil
}
func (s *stubUsageBillingService) ListModelPrices(context.Context) ([]*types.BillingModelPrice, error) {
	return nil, nil
}
func (s *stubUsageBillingService) CreateModelPriceVersion(
	context.Context,
	types.BillingModelPriceInput,
) (*types.BillingModelPrice, error) {
	return nil, nil
}
func (s *stubUsageBillingService) ListUsageLedgers(
	context.Context,
	uint64,
	int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return nil, nil
}

func (s *stubSubscriptionService) EnsureTenantBilling(context.Context, *types.Tenant) error {
	return nil
}
func (s *stubSubscriptionService) GetOverview(context.Context, uint64) (*types.BillingOverview, error) {
	return s.overview, nil
}
func (s *stubSubscriptionService) ListPlans(context.Context) ([]*types.BillingPlan, error) {
	return nil, nil
}
func (s *stubSubscriptionService) ListPrices(context.Context) ([]*types.BillingPrice, error) {
	return nil, nil
}
func (s *stubSubscriptionService) ListSubscriptions(context.Context) ([]*types.BillingSubscriptionSummary, error) {
	return nil, nil
}
func (s *stubSubscriptionService) ListCurrentSubscriptionsByTenantIDs(context.Context, []uint64) ([]*types.BillingSubscriptionSummary, error) {
	return nil, nil
}
func (s *stubSubscriptionService) ListCreditAccounts(context.Context) ([]*types.BillingCreditAccountSummary, error) {
	return nil, nil
}

func TestBillingOverviewUsesCurrentTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubSubscriptionService{overview: &types.BillingOverview{
		TenantID:   42,
		TenantName: "当前企业",
		SpaceType:  types.SpaceTypeOrganization,
		Plan: types.BillingOverviewPlan{
			Code: types.BillingPlanEnterpriseTeam,
			Name: "企业团队版",
		},
		Credits: types.BillingOverviewCredits{BalancePointMicros: 100_000_000},
	}}
	h := NewBillingHandler(service, &stubUsageBillingService{})
	router := gin.New()
	router.GET("/billing/overview", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(42))
		c.Request = c.Request.WithContext(ctx)
		h.GetOverview(c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/billing/overview", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Success bool                  `json:"success"`
		Data    types.BillingOverview `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success || response.Data.TenantID != 42 ||
		response.Data.Plan.Code != types.BillingPlanEnterpriseTeam {
		t.Fatalf("response=%+v", response)
	}
}
