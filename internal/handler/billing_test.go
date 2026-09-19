package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/types"
)

type stubSubscriptionService struct {
	overview *types.BillingOverview
}

type stubUsageBillingService struct {
	rows         []*types.BillingUsageLedgerSummary
	rowsByActor  []*types.BillingUsageLedgerSummary
	allocations  []*types.TenantMemberCreditAllocationSummary
	policy       *types.TenantBillingPolicy
	lastActorID  string
	listAllCalls int
}

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
func (s *stubUsageBillingService) ListServicePrices(context.Context) ([]*types.BillingServicePrice, error) {
	return nil, nil
}
func (s *stubUsageBillingService) CreateServicePriceVersion(
	context.Context,
	types.BillingServicePriceInput,
) (*types.BillingServicePrice, error) {
	return nil, nil
}
func (s *stubUsageBillingService) ListUsageLedgers(
	context.Context,
	uint64,
	int,
) ([]*types.BillingUsageLedgerSummary, error) {
	s.listAllCalls++
	return s.rows, nil
}
func (s *stubUsageBillingService) ListUsageLedgersByActor(
	_ context.Context,
	_ uint64,
	actorUserID string,
	_ int,
) ([]*types.BillingUsageLedgerSummary, error) {
	s.lastActorID = actorUserID
	return s.rowsByActor, nil
}
func (s *stubUsageBillingService) ListUsageSummaryByActor(
	context.Context,
	[]string,
) ([]*types.BillingActorUsageSummary, error) {
	return nil, nil
}
func (s *stubUsageBillingService) ListMemberAllocations(
	context.Context,
	uint64,
	time.Time,
) ([]*types.TenantMemberCreditAllocationSummary, error) {
	return s.allocations, nil
}
func (s *stubUsageBillingService) GetTenantBillingPolicy(
	context.Context,
	uint64,
) (*types.TenantBillingPolicy, error) {
	return s.policy, nil
}
func (s *stubUsageBillingService) UpdateTenantBillingPolicy(
	context.Context,
	uint64,
	int64,
	string,
	string,
) (*types.TenantBillingPolicy, error) {
	return nil, nil
}
func (s *stubUsageBillingService) UpdateMemberPolicy(
	context.Context,
	uint64,
	string,
	string,
	int64,
	string,
) (*types.TenantMemberCreditAllocation, error) {
	return nil, nil
}
func (s *stubUsageBillingService) UpdateMemberPolicies(
	context.Context,
	uint64,
	[]string,
	string,
	int64,
	string,
) ([]*types.TenantMemberCreditAllocation, error) {
	return nil, nil
}
func (s *stubUsageBillingService) UpdateMemberAllocation(
	context.Context,
	uint64,
	string,
	int64,
	int64,
	string,
) (*types.TenantMemberCreditAllocation, error) {
	return nil, nil
}
func (s *stubUsageBillingService) ListStorageTransactions(
	context.Context,
	uint64,
	int,
) ([]*types.BillingStorageTransactionSummary, error) {
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
	h := NewBillingHandler(service, &stubUsageBillingService{}, nil, nil, nil)
	router := gin.New()
	router.GET("/billing/overview", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(42))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleAdmin)
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

func TestBillingOverviewHidesEnterprisePoolForOrdinaryMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubSubscriptionService{overview: &types.BillingOverview{
		TenantID:  43,
		SpaceType: types.SpaceTypeOrganization,
		Storage: types.BillingOverviewStorage{
			UsedBytes:  100,
			QuotaBytes: 1000,
			Visible:    true,
		},
		Credits: types.BillingOverviewCredits{
			BalancePointMicros: 100 * types.PointMicrosPerPoint,
			PeriodPointMicros:  200 * types.PointMicrosPerPoint,
			Visible:            true,
		},
	}}
	usage := &stubUsageBillingService{
		policy: &types.TenantBillingPolicy{
			TenantID:                             43,
			DefaultMemberMonthlyLimitPointMicros: 50 * types.PointMicrosPerPoint,
			MemberOveragePolicy:                  types.MemberOveragePolicyBlock,
		},
		allocations: []*types.TenantMemberCreditAllocationSummary{{
			TenantMemberCreditAllocation: types.TenantMemberCreditAllocation{
				ID:        "member-policy-43",
				TenantID:  43,
				UserID:    "member-43",
				LimitMode: types.MemberLimitModeCustom,
			},
			EffectiveMonthlyLimitPointMicros: 30 * types.PointMicrosPerPoint,
			EffectiveOveragePolicy:           types.MemberOveragePolicyBlock,
			UsedPointMicros:                  12 * types.PointMicrosPerPoint,
		}},
	}
	h := NewBillingHandler(service, usage, nil, nil, nil)
	router := gin.New()
	router.GET("/billing/overview", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(43))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleContributor)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "member-43")
		c.Request = c.Request.WithContext(ctx)
		h.GetOverview(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/billing/overview", nil))
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
	if !response.Success ||
		response.Data.Storage.Visible ||
		response.Data.Credits.Visible ||
		response.Data.Credits.BalancePointMicros != 0 ||
		response.Data.MemberUsage == nil ||
		response.Data.MemberUsage.MonthlyUsedPointMicros != 12*types.PointMicrosPerPoint ||
		response.Data.MemberUsage.MonthlyRemainingPointMicros != 18*types.PointMicrosPerPoint {
		t.Fatalf("response=%+v", response)
	}
}

func TestBillingUsageScopesOrdinaryMemberToOwnActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usage := &stubUsageBillingService{
		rowsByActor: []*types.BillingUsageLedgerSummary{{TenantUsageLedger: types.TenantUsageLedger{
			ID:          "own-ledger",
			ActorUserID: "member-44",
		}}},
	}
	h := NewBillingHandler(&stubSubscriptionService{}, usage, nil, nil, nil)
	router := gin.New()
	router.GET("/billing/usage", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(44))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleViewer)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "member-44")
		c.Request = c.Request.WithContext(ctx)
		h.ListCurrentUsage(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/billing/usage", nil))
	if recorder.Code != http.StatusOK || usage.lastActorID != "member-44" || usage.listAllCalls != 0 {
		t.Fatalf("status=%d actor=%q all_calls=%d body=%s",
			recorder.Code, usage.lastActorID, usage.listAllCalls, recorder.Body.String())
	}
}
