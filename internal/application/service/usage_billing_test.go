package service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type staticBillingPolicy struct {
	policy types.BillingRuntimePolicy
}

func (s staticBillingPolicy) RuntimePolicy(context.Context) types.BillingRuntimePolicy {
	return s.policy
}

func TestEnterpriseUsageLinksCurrentMemberAllocation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE tenants (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			space_type TEXT,
			enterprise_credits INTEGER NOT NULL DEFAULT 0,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&types.BillingPlan{},
		&types.BillingPrice{},
		&types.TenantSubscription{},
		&types.TenantCreditAccount{},
		&types.TenantCreditTransaction{},
		&types.BillingModelPrice{},
		&types.BillingServicePrice{},
		&types.TenantBillingPolicy{},
		&types.TenantMemberCreditAllocation{},
		&types.TenantUsageReservation{},
		&types.TenantUsageLedger{},
	); err != nil {
		t.Fatal(err)
	}

	plan := &types.BillingPlan{
		ID:                   "plan-enterprise-team-test",
		Code:                 types.BillingPlanEnterpriseTeam,
		Name:                 "企业团队版",
		Edition:              "enterprise",
		SpaceType:            types.SpaceTypeOrganization,
		Status:               types.BillingStatusActive,
		BillingMultiplierPPM: 1_000_000,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&types.BillingPlan{
		ID:        "plan-legacy-test",
		Code:      types.BillingPlanLegacyCompat,
		Name:      "历史兼容版",
		Edition:   "legacy",
		SpaceType: types.SpaceTypeLegacy,
		Status:    types.BillingStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	tenant := &types.Tenant{
		ID:                301,
		Name:              "测试企业",
		SpaceType:         ptrSpaceType(types.SpaceTypeOrganization),
		EnterpriseCredits: 100,
	}
	if err := db.Exec(
		"INSERT INTO tenants(id, name, space_type, enterprise_credits) VALUES (?, ?, ?, ?)",
		tenant.ID,
		tenant.Name,
		types.SpaceTypeOrganization,
		tenant.EnterpriseCredits,
	).Error; err != nil {
		t.Fatal(err)
	}

	billingRepo := repository.NewBillingRepository(db)
	if err := billingRepo.EnsureTenantBilling(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := billingRepo.CreateModelPriceVersion(context.Background(), types.BillingModelPriceInput{
		ModelKey:                "enterprise-chat-test",
		Provider:                "generic",
		PricingMode:             "token",
		InputNanoUSDPerMTokens:  2_000_000_000,
		OutputNanoUSDPerMTokens: 8_000_000_000,
		ModelMultiplierPPM:      1_000_000,
		Status:                  types.BillingStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	usage := NewUsageBillingService(
		billingRepo,
		staticBillingPolicy{policy: types.BillingRuntimePolicy{
			Enabled:                              true,
			EnforcementMode:                      "observe",
			PointMicrosPerUSD:                    types.PointMicrosPerPoint,
			DefaultModelMultiplierPPM:            1_000_000,
			DefaultMemberMonthlyLimitPointMicros: 100 * types.PointMicrosPerPoint,
			DefaultMemberAllocationPointMicros:   100 * types.PointMicrosPerPoint,
		}},
	)
	handle, err := usage.BeginModelUsage(context.Background(), types.BillingUsageStartRequest{
		TenantID:    tenant.ID,
		ActorUserID: "enterprise-user-301",
		RefNo:       "chat:301:message-1:enterprise-chat-test",
		ModelKey:    "enterprise-chat-test",
		EstimatedUsage: types.BillingModelUsage{
			InputTokens:  100,
			OutputTokens: 20,
			CallCount:    1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if handle.AllocationID == "" || handle.Allocation == nil {
		t.Fatalf("handle missing allocation: %+v", handle)
	}
	if handle.Mode != "observe" || handle.UsageScope != types.BillingUsageScopeEnterprise {
		t.Fatalf("handle mode=%q scope=%q", handle.Mode, handle.UsageScope)
	}

	ledger, err := usage.SettleModelUsage(context.Background(), handle, types.BillingModelUsage{
		InputTokens:  120,
		OutputTokens: 30,
		CallCount:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ledger == nil || ledger.AllocationID != handle.AllocationID {
		t.Fatalf("ledger=%+v handle allocation=%q", ledger, handle.AllocationID)
	}

	allocations, err := usage.ListMemberAllocations(context.Background(), tenant.ID, ledger.BillingAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 1 ||
		allocations[0].ID != handle.AllocationID ||
		allocations[0].LedgerCount != 1 ||
		allocations[0].InputTokens != 120 ||
		allocations[0].OutputTokens != 30 {
		t.Fatalf("allocations=%#v", allocations)
	}
}

func ptrSpaceType(value types.SpaceType) *types.SpaceType {
	return &value
}
