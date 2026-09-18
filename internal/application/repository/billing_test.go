package repository

import (
	"context"
	"errors"
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
			storage_quota INTEGER NOT NULL DEFAULT 0,
			storage_used INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&types.User{},
		&types.TenantMember{},
		&types.BillingPlan{},
		&types.BillingPrice{},
		&types.TenantSubscription{},
		&types.TenantCreditAccount{},
		&types.TenantCreditTransaction{},
		&types.BillingModelPrice{},
		&types.BillingServicePrice{},
		&types.BillingPurchaseItem{},
		&types.BillingPaymentOrder{},
		&types.TenantStorageAddonGrant{},
		&types.TenantStorageTransaction{},
		&types.TenantBillingPolicy{},
		&types.TenantMemberCreditAllocation{},
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

func TestManualOrdersApplyEntitlementsAndKeepOrderHistory(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	ctx := context.Background()
	spaceType := types.SpaceTypeOrganization
	tenant := &types.Tenant{
		ID:           901,
		Name:         "运营测试企业",
		SpaceType:    &spaceType,
		StorageQuota: 100,
		StorageUsed:  20,
	}
	if err := db.Exec(
		`INSERT INTO tenants(id, name, space_type, storage_quota, storage_used) VALUES (?, ?, ?, ?, ?)`,
		tenant.ID,
		tenant.Name,
		spaceType,
		tenant.StorageQuota,
		tenant.StorageUsed,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.EnsureTenantBilling(ctx, tenant); err != nil {
		t.Fatal(err)
	}

	creditItem, err := repo.CreatePurchaseItem(ctx, types.BillingPurchaseItemInput{
		Code:              "test-credit-100",
		ItemType:          types.BillingPurchaseItemTypeTopup,
		EditionScope:      types.BillingEditionScopeEnterprise,
		Name:              "测试 100 积分",
		Currency:          "CNY",
		AmountCents:       100,
		CreditPointMicros: 100 * types.PointMicrosPerPoint,
		Status:            types.BillingPurchaseItemStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	creditOrder, err := repo.CreateManualPaymentOrder(ctx, types.BillingManualOrderInput{
		TenantID:    tenant.ID,
		OrderType:   types.BillingOrderTypeTopup,
		ItemID:      creditItem.ID,
		Description: "合同赠送积分",
		ActorUserID: "system-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if creditOrder.Status != types.BillingOrderStatusPaid ||
		creditOrder.CreditPointMicros != 100*types.PointMicrosPerPoint ||
		creditOrder.Provider != manualBillingProvider {
		t.Fatalf("credit order=%+v", creditOrder)
	}
	account, err := repo.GetCreditAccount(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if account == nil || account.BalancePointMicros != 100*types.PointMicrosPerPoint {
		t.Fatalf("account=%+v", account)
	}

	storageItem, err := repo.CreatePurchaseItem(ctx, types.BillingPurchaseItemInput{
		Code:              "test-storage-80",
		ItemType:          types.BillingPurchaseItemTypeStorageAddon,
		EditionScope:      types.BillingEditionScopeEnterprise,
		Name:              "测试 80 字节",
		Currency:          "CNY",
		AmountCents:       100,
		StorageQuotaBytes: 80,
		Status:            types.BillingPurchaseItemStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	storageOrder, err := repo.CreateManualPaymentOrder(ctx, types.BillingManualOrderInput{
		TenantID:    tenant.ID,
		OrderType:   types.BillingOrderTypeStorageAddon,
		ItemID:      storageItem.ID,
		Description: "合同赠送存储",
		ActorUserID: "system-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if storageOrder.Status != types.BillingOrderStatusPaid || storageOrder.StorageQuotaBytes != 80 {
		t.Fatalf("storage order=%+v", storageOrder)
	}
	var refreshed struct {
		StorageQuota int64
	}
	if err := db.Table("tenants").Select("storage_quota").Where("id = ?", tenant.ID).Scan(&refreshed).Error; err != nil {
		t.Fatal(err)
	}
	if refreshed.StorageQuota != 180 {
		t.Fatalf("storage quota=%d", refreshed.StorageQuota)
	}
	var grantCount int64
	if err := db.Model(&types.TenantStorageAddonGrant{}).
		Where("tenant_id = ? AND order_no = ?", tenant.ID, storageOrder.OrderNo).
		Count(&grantCount).Error; err != nil {
		t.Fatal(err)
	}
	if grantCount != 1 {
		t.Fatalf("grant count=%d", grantCount)
	}

	contractOrder, err := repo.CreateManualPaymentOrder(ctx, types.BillingManualOrderInput{
		TenantID:        tenant.ID,
		OrderType:       types.BillingOrderTypeManualContract,
		PlanID:          "plan-enterprise-team-v1",
		BillingInterval: "year",
		PeriodDays:      366,
		Description:     "年度合同开通",
		ActorUserID:     "system-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if contractOrder.Status != types.BillingOrderStatusPaid ||
		contractOrder.OrderType != types.BillingOrderTypeManualContract {
		t.Fatalf("contract order=%+v", contractOrder)
	}
	subscription, plan, err := repo.GetCurrentSubscription(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if subscription == nil || plan == nil ||
		subscription.Source != "manual_contract" ||
		subscription.BillingInterval != "year" ||
		subscription.CurrentPeriodEnd == nil {
		t.Fatalf("subscription=%+v plan=%+v", subscription, plan)
	}

	rows, err := repo.ListPaymentOrders(ctx, tenant.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("orders=%d", len(rows))
	}

	order, err := repo.GetPaymentOrder(ctx, tenant.ID, contractOrder.OrderNo)
	if err != nil {
		t.Fatal(err)
	}
	if order == nil || order.OrderNo != contractOrder.OrderNo || order.TenantID != tenant.ID {
		t.Fatalf("order=%+v", order)
	}
	if _, err := repo.GetPaymentOrder(ctx, tenant.ID+1, contractOrder.OrderNo); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant order lookup err=%v", err)
	}
}

func TestBillingPlanCatalogCRUDAndDefaultPriceVersion(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	ctx := context.Background()

	plan, err := repo.CreatePlan(ctx, types.BillingPlanInput{
		Code:                 "enterprise_plus",
		Name:                 "企业增强版",
		Description:          "面向中大型企业",
		Edition:              "enterprise",
		SpaceType:            types.SpaceTypeOrganization,
		Status:               types.BillingStatusActive,
		IsPublic:             true,
		IncludedStorageBytes: 300 * 1024 * 1024 * 1024,
		IncludedPointMicros:  2_000 * types.PointMicrosPerPoint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ID == "" || plan.BillingMultiplierPPM != 1_000_000 {
		t.Fatalf("plan=%+v", plan)
	}

	plan, err = repo.UpdatePlan(ctx, plan.ID, types.BillingPlanInput{
		Code:                 plan.Code,
		Name:                 "企业增强版 2026",
		Description:          "更新后的企业套餐",
		Edition:              "enterprise",
		SpaceType:            types.SpaceTypeOrganization,
		Status:               types.BillingStatusActive,
		IsPublic:             false,
		IncludedStorageBytes: 500 * 1024 * 1024 * 1024,
		IncludedPointMicros:  3_000 * types.PointMicrosPerPoint,
		BillingMultiplierPPM: 1_200_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Name != "企业增强版 2026" || plan.IsPublic ||
		plan.IncludedPointMicros != 3_000*types.PointMicrosPerPoint ||
		plan.BillingMultiplierPPM != 1_200_000 {
		t.Fatalf("updated plan=%+v", plan)
	}

	first, err := repo.CreatePriceVersion(ctx, types.BillingPriceInput{
		PlanID:          plan.ID,
		Code:            "enterprise_plus_cny_v1",
		Currency:        "CNY",
		BillingInterval: "month",
		AmountMinor:     19900,
		Status:          types.BillingStatusActive,
		IsDefault:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreatePriceVersion(ctx, types.BillingPriceInput{
		PlanID:          plan.ID,
		Code:            "enterprise_plus_cny_v2",
		Currency:        "CNY",
		BillingInterval: "month",
		AmountMinor:     29900,
		Status:          types.BillingStatusActive,
		IsDefault:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var prices []types.BillingPrice
	if err := db.Where("plan_id = ?", plan.ID).Order("code ASC").Find(&prices).Error; err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("prices=%d", len(prices))
	}
	var firstDefault, secondDefault bool
	for _, price := range prices {
		switch price.ID {
		case first.ID:
			firstDefault = price.IsDefault
		case second.ID:
			secondDefault = price.IsDefault
		}
	}
	if firstDefault || !secondDefault {
		t.Fatalf("default prices: first=%v second=%v", firstDefault, secondDefault)
	}
}

func TestBillingColumnNamesMatchMigrations(t *testing.T) {
	db, _ := newBillingTestRepository(t)

	expected := map[any][]string{
		&types.BillingModelPrice{}: {
			"input_nanousd_per_m_tokens",
			"output_nanousd_per_m_tokens",
			"cache_read_nanousd_per_m_tokens",
			"cache_write_nanousd_per_m_tokens",
			"call_nanousd_per_call",
			"duration_nanousd_per_second",
		},
		&types.TenantUsageReservation{}: {
			"estimated_base_cost_nanousd",
		},
		&types.TenantUsageLedger{}: {
			"base_cost_nanousd",
			"rated_cost_nanousd",
		},
		&types.TenantMemberCreditAllocation{}: {
			"allocated_period_point_micros",
			"allocated_balance_point_micros",
			"limit_mode",
			"monthly_limit_point_micros",
			"overage_policy",
		},
		&types.TenantBillingPolicy{}: {
			"default_member_monthly_limit_point_micros",
			"member_overage_policy",
		},
	}
	for model, columns := range expected {
		for _, column := range columns {
			if !db.Migrator().HasColumn(model, column) {
				t.Fatalf("%T missing migration-compatible column %q", model, column)
			}
		}
	}

	wrongColumns := map[any][]string{
		&types.BillingModelPrice{}: {
			"input_nano_usd_per_m_tokens",
			"output_nano_usd_per_m_tokens",
			"cache_read_nano_usd_per_m_tokens",
			"cache_write_nano_usd_per_m_tokens",
			"call_nano_usd_per_call",
			"duration_nano_usd_per_second",
		},
		&types.TenantUsageReservation{}: {
			"estimated_base_cost_nano_usd",
		},
		&types.TenantUsageLedger{}: {
			"base_cost_nano_usd",
			"rated_cost_nano_usd",
		},
	}
	for model, columns := range wrongColumns {
		for _, column := range columns {
			if db.Migrator().HasColumn(model, column) {
				t.Fatalf("%T has GORM-derived wrong column %q", model, column)
			}
		}
	}
}

func TestServicePricingOnlySupportsMCPToolCalls(t *testing.T) {
	_, repo := newBillingTestRepository(t)
	ctx := context.Background()

	if _, err := repo.CreateServicePriceVersion(ctx, types.BillingServicePriceInput{
		ServiceCode: "web_search.query",
		ServiceName: "联网搜索",
		PricingMode: "call",
	}); err == nil {
		t.Fatal("web search service pricing should be rejected")
	}
	if _, err := repo.CreateServicePriceVersion(ctx, types.BillingServicePriceInput{
		ServiceCode: types.BillingServiceCodeMCPToolCall,
		ServiceName: "MCP工具调用",
		PricingMode: "unit",
	}); err == nil {
		t.Fatal("MCP service pricing should only support call mode")
	}

	created, err := repo.CreateServicePriceVersion(ctx, types.BillingServicePriceInput{
		ServiceCode:    types.BillingServiceCodeMCPToolCall,
		ServiceName:    "MCP工具调用",
		PricingMode:    "call",
		NanoUSDPerCall: 1_000_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ServiceCode != types.BillingServiceCodeMCPToolCall || created.PricingMode != "call" {
		t.Fatalf("created=%+v", created)
	}

	rows, err := repo.ListServicePrices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ServiceCode != types.BillingServiceCodeMCPToolCall {
		t.Fatalf("rows=%+v", rows)
	}
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

func TestListCreditAccountsAggregatesByUserAndEnterprise(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	ctx := context.Background()
	personal := types.SpaceTypePersonal
	enterprise := types.SpaceTypeOrganization
	tenants := []struct {
		id        uint64
		name      string
		spaceType types.SpaceType
	}{
		{1001, "满乐乐's Workspace", personal},
		{1002, "备用个人空间", personal},
		{1003, "满乐乐的企业", enterprise},
	}
	for _, tenant := range tenants {
		if err := db.Exec(
			"INSERT INTO tenants(id, name, space_type) VALUES (?, ?, ?)",
			tenant.id,
			tenant.name,
			tenant.spaceType,
		).Error; err != nil {
			t.Fatal(err)
		}
	}
	user := &types.User{
		ID:           "user-personal-owner",
		Username:     "满乐乐",
		Email:        "13258978299@example.com",
		PasswordHash: "test-hash",
		TenantID:     1001,
		IsActive:     true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	for _, tenantID := range []uint64{1001, 1002} {
		if err := db.Create(&types.TenantMember{
			UserID:   user.ID,
			TenantID: tenantID,
			Role:     types.TenantRoleOwner,
			Status:   types.TenantMemberStatusActive,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	accounts := []*types.TenantCreditAccount{
		{ID: "account-personal-1", TenantID: 1001, BalancePointMicros: 40 * types.PointMicrosPerPoint},
		{ID: "account-personal-2", TenantID: 1002, BalancePointMicros: 60 * types.PointMicrosPerPoint},
		{ID: "account-enterprise", TenantID: 1003, BalancePointMicros: 500 * types.PointMicrosPerPoint},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := repo.ListCreditAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want two billing subjects", len(rows))
	}
	var personalRow, enterpriseRow *types.BillingCreditAccountSummary
	for _, row := range rows {
		switch row.SubjectType {
		case "user":
			personalRow = row
		case "enterprise":
			enterpriseRow = row
		}
	}
	if personalRow == nil ||
		personalRow.SubjectID != user.ID ||
		personalRow.SubjectName != user.Username ||
		personalRow.BalancePointMicros != 100*types.PointMicrosPerPoint ||
		personalRow.TenantCount != 2 {
		t.Fatalf("personal row=%+v", personalRow)
	}
	if enterpriseRow == nil ||
		enterpriseRow.SubjectID != "1003" ||
		enterpriseRow.SubjectName != "满乐乐的企业" ||
		enterpriseRow.BalancePointMicros != 500*types.PointMicrosPerPoint ||
		enterpriseRow.TenantCount != 1 {
		t.Fatalf("enterprise row=%+v", enterpriseRow)
	}
}

func TestEnterpriseMemberAllocationAndActorUsageSummary(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	spaceType := types.SpaceTypeOrganization
	tenant := &types.Tenant{
		ID:                91,
		Name:              "企业空间",
		SpaceType:         &spaceType,
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
	if err := repo.EnsureTenantBilling(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	if _, err := repo.EnsureTenantBillingPolicy(
		context.Background(),
		tenant.ID,
		100*types.PointMicrosPerPoint,
		"system-admin",
	); err != nil {
		t.Fatal(err)
	}
	allocation, err := repo.EnsureCurrentMemberAllocation(
		context.Background(),
		tenant.ID,
		"user-91",
		periodStart,
		periodEnd,
		100*types.PointMicrosPerPoint,
		"system-admin",
	)
	if err != nil {
		t.Fatal(err)
	}
	allocationAgain, err := repo.EnsureCurrentMemberAllocation(
		context.Background(),
		tenant.ID,
		"user-91",
		periodStart,
		periodEnd,
		100*types.PointMicrosPerPoint,
		"system-admin",
	)
	if err != nil {
		t.Fatal(err)
	}
	if allocation.ID != allocationAgain.ID ||
		allocation.LimitMode != types.MemberLimitModeInherit {
		t.Fatalf("allocation=%+v allocationAgain=%+v", allocation, allocationAgain)
	}

	ledger := &types.TenantUsageLedger{
		ID:                  "ledger-enterprise-91",
		TenantID:            tenant.ID,
		ActorUserID:         "user-91",
		UsageScope:          types.BillingUsageScopeEnterprise,
		AllocationID:        allocation.ID,
		RefNo:               "chat:91:message-1:test-model",
		ModelKey:            "test-model",
		InputTokens:         120,
		OutputTokens:        30,
		BilledPointMicros:   15 * types.PointMicrosPerPoint,
		Status:              "observed",
		BillingAt:           now,
		UsageDate:           now,
		PricingSnapshotJSON: types.JSON([]byte("{}")),
	}
	handle := &types.BillingUsageHandle{
		Request: types.BillingUsageStartRequest{
			TenantID:    tenant.ID,
			ActorUserID: "user-91",
			RefNo:       ledger.RefNo,
			ModelKey:    ledger.ModelKey,
		},
		Mode:         "observe",
		UsageScope:   types.BillingUsageScopeEnterprise,
		AllocationID: allocation.ID,
	}
	if _, err := repo.SettleUsage(context.Background(), handle, ledger); err != nil {
		t.Fatal(err)
	}

	usage, err := repo.ListUsageSummaryByActor(context.Background(), []string{"user-91"})
	if err != nil {
		t.Fatal(err)
	}
	if len(usage) != 1 ||
		usage[0].ActorUserID != "user-91" ||
		usage[0].EnterpriseLedgerCount != 1 ||
		usage[0].InputTokens != 120 ||
		usage[0].OutputTokens != 30 ||
		usage[0].BilledPointMicros != 15*types.PointMicrosPerPoint {
		t.Fatalf("usage=%#v", usage)
	}

	allocations, err := repo.ListMemberAllocations(context.Background(), tenant.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 1 ||
		allocations[0].ID != allocation.ID ||
		allocations[0].EffectiveMonthlyLimitPointMicros != 100*types.PointMicrosPerPoint ||
		allocations[0].UsedPointMicros != 15*types.PointMicrosPerPoint ||
		allocations[0].LedgerCount != 1 {
		t.Fatalf("allocations=%#v", allocations)
	}
}

func TestGetPeriodUsedPointMicrosOnlyCountsSettledPeriodUsage(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	now := time.Now().UTC()
	periodStart := now.Add(-time.Hour)
	periodEnd := now.Add(time.Hour)
	if err := db.Exec(
		"INSERT INTO tenants(id, name, space_type) VALUES (?, ?, ?)",
		901,
		"周期用量企业",
		types.SpaceTypeOrganization,
	).Error; err != nil {
		t.Fatal(err)
	}
	rows := []*types.TenantUsageLedger{
		{
			ID:                       "period-settled",
			TenantID:                 901,
			ActorUserID:              "user-901",
			UsageScope:               types.BillingUsageScopeEnterprise,
			RefNo:                    "period:settled",
			BilledPointMicros:        20 * types.PointMicrosPerPoint,
			PeriodCoveredPointMicros: 12 * types.PointMicrosPerPoint,
			Status:                   "settled",
			BillingAt:                now,
			UsageDate:                now,
			PricingSnapshotJSON:      types.JSON([]byte("{}")),
		},
		{
			ID:                       "period-observed",
			TenantID:                 901,
			ActorUserID:              "user-901",
			UsageScope:               types.BillingUsageScopeEnterprise,
			RefNo:                    "period:observed",
			BilledPointMicros:        30 * types.PointMicrosPerPoint,
			PeriodCoveredPointMicros: 30 * types.PointMicrosPerPoint,
			Status:                   "observed",
			BillingAt:                now,
			UsageDate:                now,
			PricingSnapshotJSON:      types.JSON([]byte("{}")),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	used, err := repo.GetPeriodUsedPointMicros(context.Background(), 901, periodStart, periodEnd)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(12 * types.PointMicrosPerPoint); used != want {
		t.Fatalf("used=%d want=%d", used, want)
	}
}

func TestEnterpriseReservationHonorsMemberLimitAndOveragePolicy(t *testing.T) {
	db, repo := newBillingTestRepository(t)
	spaceType := types.SpaceTypeOrganization
	tenant := &types.Tenant{
		ID:                92,
		Name:              "额度策略企业",
		SpaceType:         &spaceType,
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
	if err := repo.EnsureTenantBilling(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	subscription, plan, err := repo.GetCurrentSubscription(context.Background(), tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan.IncludedPointMicros = 100 * types.PointMicrosPerPoint
	if err := db.Model(&types.BillingPlan{}).
		Where("id = ?", plan.ID).
		Update("included_point_micros", plan.IncludedPointMicros).Error; err != nil {
		t.Fatal(err)
	}
	policy, err := repo.EnsureTenantBillingPolicy(
		context.Background(),
		tenant.ID,
		10*types.PointMicrosPerPoint,
		"owner-92",
	)
	if err != nil {
		t.Fatal(err)
	}
	periodStart, periodEnd := billingPeriod(subscription, time.Now().UTC())
	allocation, err := repo.EnsureCurrentMemberAllocation(
		context.Background(),
		tenant.ID,
		"member-92",
		periodStart,
		periodEnd,
		policy.DefaultMemberMonthlyLimitPointMicros,
		"owner-92",
	)
	if err != nil {
		t.Fatal(err)
	}

	handle := &types.BillingUsageHandle{
		Request: types.BillingUsageStartRequest{
			TenantID:    tenant.ID,
			ActorUserID: "member-92",
			RefNo:       "enterprise-limit-block",
			ModelKey:    "test-model",
		},
		Mode:                          "enforce",
		UsageScope:                    types.BillingUsageScopeEnterprise,
		AllocationID:                  allocation.ID,
		MemberLimitMode:               types.MemberLimitModeInherit,
		MemberMonthlyLimitPointMicros: 10 * types.PointMicrosPerPoint,
		MemberOveragePolicy:           types.MemberOveragePolicyBlock,
		Plan:                          plan,
		Subscription:                  subscription,
		Allocation:                    allocation,
		EnterprisePolicy:              policy,
	}
	if _, err := repo.CreateUsageReservation(
		context.Background(),
		handle,
		0,
		11*types.PointMicrosPerPoint,
	); !errors.Is(err, ErrInsufficientBillingCredits) {
		t.Fatalf("expected overage to be blocked, got %v", err)
	}

	handle.Request.RefNo = "enterprise-limit-use-balance"
	handle.MemberOveragePolicy = types.MemberOveragePolicyUseEnterpriseBalance
	reservation, err := repo.CreateUsageReservation(
		context.Background(),
		handle,
		0,
		11*types.PointMicrosPerPoint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reservation.ReservedPeriodPointMicros != 10*types.PointMicrosPerPoint ||
		reservation.ReservedBalancePointMicros != types.PointMicrosPerPoint {
		t.Fatalf("reservation=%+v", reservation)
	}
	handle.Reservation = reservation
	now := time.Now().UTC()
	ledger, err := repo.SettleUsage(context.Background(), handle, &types.TenantUsageLedger{
		ID:                  "ledger-enterprise-limit-92",
		TenantID:            tenant.ID,
		ActorUserID:         "member-92",
		UsageScope:          types.BillingUsageScopeEnterprise,
		AllocationID:        allocation.ID,
		RefNo:               handle.Request.RefNo,
		ModelKey:            "test-model",
		BilledPointMicros:   11 * types.PointMicrosPerPoint,
		Status:              "settled",
		BillingAt:           now,
		UsageDate:           now,
		PricingSnapshotJSON: types.JSON([]byte("{}")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if ledger.PeriodCoveredPointMicros != 10*types.PointMicrosPerPoint ||
		ledger.BalanceChargedPointMicros != types.PointMicrosPerPoint {
		t.Fatalf("ledger=%+v", ledger)
	}
}
