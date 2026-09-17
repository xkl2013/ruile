package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

func newBillingTestRepository(t *testing.T) (*gorm.DB, *billingRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE tenants (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			space_type TEXT,
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
		&types.TenantUsageReservation{},
		&types.TenantUsageLedger{},
	); err != nil {
		t.Fatal(err)
	}
	plans := []*types.BillingPlan{
		{ID: "plan-personal-free-v1", Code: types.BillingPlanPersonalFree, Name: "个人免费版", Edition: "personal", SpaceType: types.SpaceTypePersonal, Status: "active"},
		{ID: "plan-enterprise-team-v1", Code: types.BillingPlanEnterpriseTeam, Name: "企业团队版", Edition: "enterprise", SpaceType: types.SpaceTypeOrganization, Status: "active"},
		{ID: "plan-legacy-compat-v1", Code: types.BillingPlanLegacyCompat, Name: "历史兼容版", Edition: "legacy", SpaceType: types.SpaceTypeLegacy, Status: "active"},
	}
	if err := db.Create(&plans).Error; err != nil {
		t.Fatal(err)
	}
	return db, &billingRepository{db: db}
}

func TestModelPriceVersionsAndUsageSettlementAreIdempotent(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	spaceType := types.SpaceTypePersonal
	tenant := &types.Tenant{
		ID:        88,
		Name:      "个人空间",
		SpaceType: &spaceType,
	}
	if err := db.Exec(
		"INSERT INTO tenants(id, name, space_type) VALUES (?, ?, ?)",
		tenant.ID,
		tenant.Name,
		spaceType,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.EnsureTenantBilling(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&types.TenantCreditAccount{}).
		Where("tenant_id = ?", tenant.ID).
		Update("balance_point_micros", 100*types.PointMicrosPerPoint).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&types.BillingPlan{}).
		Where("code = ?", types.BillingPlanPersonalFree).
		Updates(map[string]any{
			"included_point_micros":  10 * types.PointMicrosPerPoint,
			"billing_multiplier_ppm": 1_000_000,
		}).Error; err != nil {
		t.Fatal(err)
	}

	for _, inputRate := range []int64{1_000_000_000, 2_000_000_000} {
		if _, err := repo.CreateModelPriceVersion(context.Background(), types.BillingModelPriceInput{
			ModelKey:                "test-model",
			PricingMode:             "token",
			InputNanoUSDPerMTokens:  inputRate,
			OutputNanoUSDPerMTokens: 4_000_000_000,
			ModelMultiplierPPM:      1_000_000,
			Status:                  "active",
		}); err != nil {
			t.Fatal(err)
		}
	}
	activePrice, err := repo.GetActiveModelPrice(context.Background(), "test-model")
	if err != nil {
		t.Fatal(err)
	}
	if activePrice == nil || activePrice.Version != 2 ||
		activePrice.InputNanoUSDPerMTokens != 2_000_000_000 {
		t.Fatalf("activePrice=%#v", activePrice)
	}

	subscription, plan, err := repo.GetCurrentSubscription(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	handle := &types.BillingUsageHandle{
		Request: types.BillingUsageStartRequest{
			TenantID:    tenant.ID,
			ActorUserID: "user-88",
			RefNo:       "chat:88:message-1:test-model",
			ModelID:     "model-88",
			ModelKey:    "test-model",
		},
		Mode:         "enforce",
		UsageScope:   "personal_usage",
		Price:        activePrice,
		Plan:         plan,
		Subscription: subscription,
	}
	reservation, err := repo.CreateUsageReservation(
		context.Background(),
		handle,
		20_000_000,
		20*types.PointMicrosPerPoint,
	)
	if err != nil {
		t.Fatal(err)
	}
	handle.Reservation = reservation
	if reservation.ReservedPeriodPointMicros != 10*types.PointMicrosPerPoint ||
		reservation.ReservedBalancePointMicros != 10*types.PointMicrosPerPoint {
		t.Fatalf("reservation=%+v", reservation)
	}

	now := time.Now().UTC()
	ledger := &types.TenantUsageLedger{
		ID:                  "ledger-88",
		TenantID:            tenant.ID,
		ActorUserID:         "user-88",
		UsageScope:          "personal_usage",
		RefNo:               handle.Request.RefNo,
		ModelID:             handle.Request.ModelID,
		ModelKey:            handle.Request.ModelKey,
		PricingID:           activePrice.ID,
		PricingVersion:      activePrice.Version,
		BilledPointMicros:   15 * types.PointMicrosPerPoint,
		Status:              "settled",
		BillingAt:           now,
		UsageDate:           now,
		PricingSnapshotJSON: types.JSON([]byte("{}")),
	}
	first, err := repo.SettleUsage(context.Background(), handle, ledger)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.SettleUsage(context.Background(), handle, ledger)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID ||
		first.PeriodCoveredPointMicros != 10*types.PointMicrosPerPoint ||
		first.BalanceChargedPointMicros != 5*types.PointMicrosPerPoint {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	account, err := repo.GetCreditAccount(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if account.BalancePointMicros != 95*types.PointMicrosPerPoint {
		t.Fatalf("balance=%d, want %d", account.BalancePointMicros, 95*types.PointMicrosPerPoint)
	}
	var transactionCount int64
	if err := db.Model(&types.TenantCreditTransaction{}).
		Where("ref_no = ?", "usage-debit:"+ledger.ID).
		Count(&transactionCount).Error; err != nil {
		t.Fatal(err)
	}
	if transactionCount != 1 {
		t.Fatalf("usage transaction count=%d, want 1", transactionCount)
	}
}

func TestEnsureTenantBillingIsIdempotent(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	spaceType := types.SpaceTypeOrganization
	tenant := &types.Tenant{
		ID:                42,
		Name:              "测试企业",
		SpaceType:         &spaceType,
		StorageQuota:      100 * 1024 * 1024 * 1024,
		EnterpriseCredits: 100,
	}
	if err := db.Exec(
		"INSERT INTO tenants(id, name, space_type) VALUES (?, ?, ?)",
		tenant.ID,
		tenant.Name,
		spaceType,
	).Error; err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := repo.EnsureTenantBilling(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	var accountCount, subscriptionCount, transactionCount int64
	db.Model(&types.TenantCreditAccount{}).Where("tenant_id = ?", tenant.ID).Count(&accountCount)
	db.Model(&types.TenantSubscription{}).Where("tenant_id = ?", tenant.ID).Count(&subscriptionCount)
	db.Model(&types.TenantCreditTransaction{}).Where("tenant_id = ?", tenant.ID).Count(&transactionCount)
	if accountCount != 1 || subscriptionCount != 1 || transactionCount != 1 {
		t.Fatalf(
			"counts account=%d subscription=%d transaction=%d, want 1/1/1",
			accountCount,
			subscriptionCount,
			transactionCount,
		)
	}

	account, err := repo.GetCreditAccount(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if account == nil || account.BalancePointMicros != 100*types.PointMicrosPerPoint {
		t.Fatalf("account = %#v", account)
	}
	subscription, plan, err := repo.GetCurrentSubscription(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if subscription == nil || plan == nil || plan.Code != types.BillingPlanEnterpriseTeam {
		t.Fatalf("subscription=%#v plan=%#v", subscription, plan)
	}
	subscriptions, err := repo.ListSubscriptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := repo.ListCreditAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 || subscriptions[0].TenantName != tenant.Name ||
		subscriptions[0].PlanCode != types.BillingPlanEnterpriseTeam {
		t.Fatalf("subscriptions=%#v", subscriptions)
	}
	if len(accounts) != 1 || accounts[0].TenantName != tenant.Name ||
		accounts[0].BalancePointMicros != 100*types.PointMicrosPerPoint {
		t.Fatalf("accounts=%#v", accounts)
	}
}

func TestEnsureTenantBillingClassifiesLegacyWorkspace(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	tenant := &types.Tenant{ID: 77, Name: "历史空间"}
	if err := db.Exec(
		"INSERT INTO tenants(id, name, space_type) VALUES (?, ?, NULL)",
		tenant.ID,
		tenant.Name,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.EnsureTenantBilling(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	subscription, plan, err := repo.GetCurrentSubscription(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if subscription.Status != types.BillingStatusLegacy || plan.Code != types.BillingPlanLegacyCompat {
		t.Fatalf("subscription=%#v plan=%#v", subscription, plan)
	}
}
