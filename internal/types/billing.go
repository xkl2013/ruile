package types

import "time"

const (
	BillingPlanPersonalFree       = "personal_free"
	BillingPlanPersonalPro        = "personal_pro"
	BillingPlanEnterpriseTeam     = "enterprise_team"
	BillingPlanEnterpriseBusiness = "enterprise_business"
	BillingPlanLegacyCompat       = "legacy_compat"

	BillingStatusActive   = "active"
	BillingStatusTrialing = "trialing"
	BillingStatusLegacy   = "legacy"
	BillingStatusCanceled = "canceled"

	PointMicrosPerPoint int64 = 1_000_000
)

// BillingPlan is the deploy-wide product definition assigned to a workspace.
type BillingPlan struct {
	ID                   string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code                 string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	Name                 string    `gorm:"type:varchar(128);not null" json:"name"`
	Description          string    `gorm:"type:text;not null;default:''" json:"description"`
	Edition              string    `gorm:"type:varchar(32);not null" json:"edition"`
	SpaceType            SpaceType `gorm:"type:varchar(32);not null" json:"space_type"`
	Status               string    `gorm:"type:varchar(32);not null" json:"status"`
	IsPublic             bool      `gorm:"not null;default:true" json:"is_public"`
	IncludedStorageBytes int64     `gorm:"not null;default:0" json:"included_storage_bytes"`
	IncludedPointMicros  int64     `gorm:"not null;default:0" json:"included_point_micros"`
	BillingMultiplierPPM int64     `gorm:"not null;default:1000000" json:"billing_multiplier_ppm"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (BillingPlan) TableName() string { return "billing_plans" }

// BillingPrice is a versioned commercial price. Draft rows allow the first
// billing milestone to expose the catalog without enabling payments.
type BillingPrice struct {
	ID              string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	PlanID          string     `gorm:"type:varchar(36);not null;index" json:"plan_id"`
	Code            string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	Currency        string     `gorm:"type:varchar(8);not null" json:"currency"`
	BillingInterval string     `gorm:"type:varchar(16);not null" json:"billing_interval"`
	AmountMinor     int64      `gorm:"not null;default:0" json:"amount_minor"`
	Status          string     `gorm:"type:varchar(32);not null" json:"status"`
	IsDefault       bool       `gorm:"not null;default:false" json:"is_default"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (BillingPrice) TableName() string { return "billing_prices" }

type BillingPlanInput struct {
	Code                 string    `json:"code"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Edition              string    `json:"edition"`
	SpaceType            SpaceType `json:"space_type"`
	Status               string    `json:"status"`
	IsPublic             bool      `json:"is_public"`
	IncludedStorageBytes int64     `json:"included_storage_bytes"`
	IncludedPointMicros  int64     `json:"included_point_micros"`
	BillingMultiplierPPM int64     `json:"billing_multiplier_ppm"`
}

type BillingPriceInput struct {
	PlanID          string     `json:"plan_id"`
	Code            string     `json:"code"`
	Currency        string     `json:"currency"`
	BillingInterval string     `json:"billing_interval"`
	AmountMinor     int64      `json:"amount_minor"`
	Status          string     `json:"status"`
	IsDefault       bool       `json:"is_default"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
}

const (
	BillingPurchaseItemTypeTopup        = "topup"
	BillingPurchaseItemTypeStorageAddon = "storage_addon"

	BillingEditionScopeAll        = "all"
	BillingEditionScopePersonal   = "personal"
	BillingEditionScopeEnterprise = "enterprise"

	BillingPurchaseItemStatusActive   = "active"
	BillingPurchaseItemStatusDisabled = "disabled"

	BillingOrderTypeSubscription   = "subscription"
	BillingOrderTypeTopup          = "topup"
	BillingOrderTypeStorageAddon   = "storage_addon"
	BillingOrderTypeManualContract = "manual_contract"

	BillingOrderStatusPending        = "pending"
	BillingOrderStatusPaid           = "paid"
	BillingOrderStatusFailed         = "failed"
	BillingOrderStatusExpired        = "expired"
	BillingOrderStatusClosed         = "closed"
	BillingOrderStatusReconciliation = "reconciliation"
)

// BillingPurchaseItem is a one-time credit or storage package. It is
// deployment-wide and intentionally separate from recurring plan prices.
type BillingPurchaseItem struct {
	ID                string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code              string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	ItemType          string    `gorm:"type:varchar(32);not null" json:"item_type"`
	EditionScope      string    `gorm:"type:varchar(32);not null;default:'all'" json:"edition_scope"`
	Name              string    `gorm:"type:varchar(128);not null" json:"name"`
	Description       string    `gorm:"type:varchar(512);not null;default:''" json:"description"`
	Currency          string    `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`
	AmountCents       int64     `gorm:"not null;default:0" json:"amount_cents"`
	CreditPointMicros int64     `gorm:"not null;default:0" json:"credit_point_micros"`
	StorageQuotaBytes int64     `gorm:"not null;default:0" json:"storage_quota_bytes"`
	DurationDays      int       `gorm:"not null;default:0" json:"duration_days"`
	Status            string    `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	SortOrder         int       `gorm:"not null;default:0" json:"sort_order"`
	MetadataJSON      JSON      `gorm:"type:jsonb;not null" json:"metadata_json"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (BillingPurchaseItem) TableName() string { return "billing_purchase_items" }

type BillingPurchaseItemInput struct {
	Code              string `json:"code"`
	ItemType          string `json:"item_type"`
	EditionScope      string `json:"edition_scope"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Currency          string `json:"currency"`
	AmountCents       int64  `json:"amount_cents"`
	CreditPointMicros int64  `json:"credit_point_micros"`
	StorageQuotaBytes int64  `json:"storage_quota_bytes"`
	DurationDays      int    `json:"duration_days"`
	Status            string `json:"status"`
	SortOrder         int    `json:"sort_order"`
	MetadataJSON      JSON   `json:"metadata_json"`
}

// BillingPaymentOrder is the durable commercial record. The offline
// operations path writes provider=manual and status=paid; online checkout is
// deliberately not enabled until a payment provider is configured.
type BillingPaymentOrder struct {
	ID                  string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	OrderNo             string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"order_no"`
	OrderType           string     `gorm:"type:varchar(32);not null" json:"order_type"`
	TenantID            uint64     `gorm:"not null;index" json:"tenant_id"`
	ActorUserID         string     `gorm:"type:varchar(64);not null;default:''" json:"actor_user_id"`
	PlanID              string     `gorm:"type:varchar(36);not null;default:''" json:"plan_id"`
	PriceID             string     `gorm:"type:varchar(36);not null;default:''" json:"price_id"`
	ItemID              string     `gorm:"type:varchar(36);not null;default:''" json:"item_id"`
	Provider            string     `gorm:"type:varchar(32);not null;default:''" json:"provider"`
	PaymentMethod       string     `gorm:"type:varchar(32);not null;default:''" json:"payment_method"`
	Status              string     `gorm:"type:varchar(32);not null;default:'pending'" json:"status"`
	Currency            string     `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`
	AmountCents         int64      `gorm:"not null;default:0" json:"amount_cents"`
	CreditPointMicros   int64      `gorm:"not null;default:0" json:"credit_point_micros"`
	StorageQuotaBytes   int64      `gorm:"not null;default:0" json:"storage_quota_bytes"`
	BillingInterval     string     `gorm:"type:varchar(16);not null;default:''" json:"billing_interval"`
	Cycles              int        `gorm:"not null;default:1" json:"cycles"`
	ExternalPaymentRef  string     `gorm:"type:varchar(128);not null;default:''" json:"external_payment_ref"`
	ExternalCheckoutRef string     `gorm:"type:varchar(128);not null;default:''" json:"external_checkout_ref"`
	CheckoutURL         string     `gorm:"type:text;not null;default:''" json:"checkout_url"`
	QRCodeURL           string     `gorm:"type:text;not null;default:''" json:"qr_code_url"`
	NotifyURL           string     `gorm:"type:text;not null;default:''" json:"notify_url"`
	ReturnURL           string     `gorm:"type:text;not null;default:''" json:"return_url"`
	ProviderPayloadJSON JSON       `gorm:"type:jsonb;not null" json:"provider_payload_json"`
	NotifySnapshotJSON  JSON       `gorm:"type:jsonb;not null" json:"notify_snapshot_json"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	ExpiredAt           *time.Time `json:"expired_at,omitempty"`
	SnapshotJSON        JSON       `gorm:"type:jsonb;not null" json:"snapshot_json"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (BillingPaymentOrder) TableName() string { return "billing_payment_orders" }

// TenantStorageAddonGrant records a positive storage package issuance. The
// quota column remains the enforcement source, while this table preserves the
// order-level entitlement for audit and future expiry handling.
type TenantStorageAddonGrant struct {
	ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID          uint64     `gorm:"not null;index" json:"tenant_id"`
	OrderNo           string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"order_no"`
	ItemID            string     `gorm:"type:varchar(36);not null;default:''" json:"item_id"`
	StorageQuotaBytes int64      `gorm:"not null" json:"storage_quota_bytes"`
	Status            string     `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	MetadataJSON      JSON       `gorm:"type:jsonb;not null" json:"metadata_json"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (TenantStorageAddonGrant) TableName() string { return "tenant_storage_addon_grants" }

// BillingManualOrderInput is only accepted through the SystemAdmin
// operations route. Topup and storage adjustments accept signed entitlement
// values; contract orders select the target plan and billing period.
type BillingManualOrderInput struct {
	TenantID          uint64 `json:"tenant_id"`
	OrderType         string `json:"order_type"`
	ItemID            string `json:"item_id"`
	PlanID            string `json:"plan_id"`
	BillingInterval   string `json:"billing_interval"`
	PeriodDays        int    `json:"period_days"`
	CreditPointMicros int64  `json:"credit_point_micros"`
	StorageQuotaBytes int64  `json:"storage_quota_bytes"`
	AmountCents       int64  `json:"amount_cents"`
	Description       string `json:"description"`
	ActorUserID       string `json:"-"`
}

// TenantSubscription assigns exactly one current plan to a workspace during
// the compatibility rollout. Historical versions can be added later.
type TenantSubscription struct {
	ID                 string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID           uint64     `gorm:"not null;index" json:"tenant_id"`
	PlanID             string     `gorm:"type:varchar(36);not null;index" json:"plan_id"`
	Status             string     `gorm:"type:varchar(32);not null" json:"status"`
	BillingInterval    string     `gorm:"type:varchar(16);not null" json:"billing_interval"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	PriceSnapshot      JSON       `gorm:"type:jsonb;not null" json:"price_snapshot"`
	Source             string     `gorm:"type:varchar(32);not null" json:"source"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (TenantSubscription) TableName() string { return "tenant_subscriptions" }

// TenantCreditAccount is the authority for the workspace credit balance.
type TenantCreditAccount struct {
	ID                 string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID           uint64    `gorm:"not null;uniqueIndex" json:"tenant_id"`
	BalancePointMicros int64     `gorm:"not null;default:0" json:"balance_point_micros"`
	Version            int64     `gorm:"not null;default:0" json:"version"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (TenantCreditAccount) TableName() string { return "tenant_credit_accounts" }

// TenantCreditTransaction is an append-only balance mutation record.
type TenantCreditTransaction struct {
	ID                 string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID           uint64    `gorm:"not null;index" json:"tenant_id"`
	AccountID          string    `gorm:"type:varchar(36);not null;index" json:"account_id"`
	Type               string    `gorm:"type:varchar(32);not null" json:"type"`
	AmountPointMicros  int64     `gorm:"not null" json:"amount_point_micros"`
	BalancePointMicros int64     `gorm:"not null" json:"balance_point_micros"`
	RefNo              string    `gorm:"type:varchar(128);not null;uniqueIndex" json:"ref_no"`
	Description        string    `gorm:"type:text;not null;default:''" json:"description"`
	CreatedAt          time.Time `json:"created_at"`
}

func (TenantCreditTransaction) TableName() string { return "tenant_credit_transactions" }

type BillingRuntimePolicy struct {
	Enabled                              bool   `json:"enabled"`
	EnforcementMode                      string `json:"enforcement_mode"`
	PointMicrosPerUSD                    int64  `json:"point_micros_per_usd"`
	DefaultModelMultiplierPPM            int64  `json:"default_model_multiplier_ppm"`
	DefaultServiceMultiplierPPM          int64  `json:"default_service_multiplier_ppm"`
	DefaultMemberMonthlyLimitPointMicros int64  `json:"default_member_monthly_limit_point_micros"`
	DefaultMemberAllocationPointMicros   int64  `json:"default_member_allocation_point_micros"`
}

// BillingModelPrice is an immutable version of a model's upstream cost.
// model_key is the provider-facing model name; model_id is stored separately
// on usage ledgers so tenant-local model records can be renamed safely.
type BillingModelPrice struct {
	ID                          string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ModelKey                    string     `gorm:"type:varchar(128);not null;uniqueIndex:uq_billing_model_price_version" json:"model_key"`
	Provider                    string     `gorm:"type:varchar(64);not null;default:''" json:"provider"`
	PricingMode                 string     `gorm:"type:varchar(32);not null;default:'token'" json:"pricing_mode"`
	InputNanoUSDPerMTokens      int64      `gorm:"column:input_nanousd_per_m_tokens;not null;default:0" json:"input_nanousd_per_m_tokens"`
	OutputNanoUSDPerMTokens     int64      `gorm:"column:output_nanousd_per_m_tokens;not null;default:0" json:"output_nanousd_per_m_tokens"`
	CacheReadNanoUSDPerMTokens  int64      `gorm:"column:cache_read_nanousd_per_m_tokens;not null;default:0" json:"cache_read_nanousd_per_m_tokens"`
	CacheWriteNanoUSDPerMTokens int64      `gorm:"column:cache_write_nanousd_per_m_tokens;not null;default:0" json:"cache_write_nanousd_per_m_tokens"`
	CallNanoUSDPerCall          int64      `gorm:"column:call_nanousd_per_call;not null;default:0" json:"call_nanousd_per_call"`
	DurationNanoUSDPerSecond    int64      `gorm:"column:duration_nanousd_per_second;not null;default:0" json:"duration_nanousd_per_second"`
	TieredPricingJSON           JSON       `gorm:"type:jsonb;not null" json:"tiered_pricing_json"`
	ModelMultiplierPPM          int64      `gorm:"not null;default:1000000" json:"model_multiplier_ppm"`
	Version                     int        `gorm:"not null;default:1;uniqueIndex:uq_billing_model_price_version" json:"version"`
	EffectiveAt                 time.Time  `json:"effective_at"`
	ExpiresAt                   *time.Time `json:"expires_at,omitempty"`
	Status                      string     `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	SnapshotJSON                JSON       `gorm:"type:jsonb;not null" json:"snapshot_json"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

func (BillingModelPrice) TableName() string { return "billing_model_prices" }

type BillingModelPriceInput struct {
	ModelKey                    string     `json:"model_key" binding:"required"`
	Provider                    string     `json:"provider"`
	PricingMode                 string     `json:"pricing_mode"`
	InputNanoUSDPerMTokens      int64      `json:"input_nanousd_per_m_tokens"`
	OutputNanoUSDPerMTokens     int64      `json:"output_nanousd_per_m_tokens"`
	CacheReadNanoUSDPerMTokens  int64      `json:"cache_read_nanousd_per_m_tokens"`
	CacheWriteNanoUSDPerMTokens int64      `json:"cache_write_nanousd_per_m_tokens"`
	CallNanoUSDPerCall          int64      `json:"call_nanousd_per_call"`
	DurationNanoUSDPerSecond    int64      `json:"duration_nanousd_per_second"`
	TieredPricingJSON           JSON       `json:"tiered_pricing_json"`
	ModelMultiplierPPM          int64      `json:"model_multiplier_ppm"`
	EffectiveAt                 *time.Time `json:"effective_at,omitempty"`
	ExpiresAt                   *time.Time `json:"expires_at,omitempty"`
	Status                      string     `json:"status"`
}

// BillingServicePrice versions non-model platform costs separately from model
// costs. This prevents a search/OCR/MCP charge from inheriting model markup.
type BillingServicePrice struct {
	ID                   string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ServiceCode          string     `gorm:"type:varchar(64);not null;uniqueIndex:uq_billing_service_price_version" json:"service_code"`
	ServiceName          string     `gorm:"type:varchar(128);not null;default:''" json:"service_name"`
	PricingMode          string     `gorm:"type:varchar(32);not null;default:'call'" json:"pricing_mode"`
	NanoUSDPerCall       int64      `gorm:"column:nanousd_per_call;not null;default:0" json:"nanousd_per_call"`
	NanoUSDPerUnit       int64      `gorm:"column:nanousd_per_unit;not null;default:0" json:"nanousd_per_unit"`
	UnitName             string     `gorm:"type:varchar(32);not null;default:''" json:"unit_name"`
	ServiceMultiplierPPM int64      `gorm:"not null;default:1000000" json:"service_multiplier_ppm"`
	Version              int        `gorm:"not null;default:1;uniqueIndex:uq_billing_service_price_version" json:"version"`
	EffectiveAt          time.Time  `json:"effective_at"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	Status               string     `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	SnapshotJSON         JSON       `gorm:"type:jsonb;not null" json:"snapshot_json"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (BillingServicePrice) TableName() string { return "billing_service_prices" }

type BillingServicePriceInput struct {
	ServiceCode          string     `json:"service_code" binding:"required"`
	ServiceName          string     `json:"service_name"`
	PricingMode          string     `json:"pricing_mode"`
	NanoUSDPerCall       int64      `json:"nanousd_per_call"`
	NanoUSDPerUnit       int64      `json:"nanousd_per_unit"`
	UnitName             string     `json:"unit_name"`
	ServiceMultiplierPPM int64      `json:"service_multiplier_ppm"`
	EffectiveAt          *time.Time `json:"effective_at,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	Status               string     `json:"status"`
}

type TenantUsageReservation struct {
	ID                         string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID                   uint64     `gorm:"not null;uniqueIndex:uq_tenant_usage_reservation_ref" json:"tenant_id"`
	ActorUserID                string     `gorm:"type:varchar(64);not null;default:''" json:"actor_user_id"`
	UsageScope                 string     `gorm:"type:varchar(32);not null" json:"usage_scope"`
	AllocationID               string     `gorm:"type:varchar(36);not null;default:''" json:"allocation_id"`
	RefNo                      string     `gorm:"type:varchar(160);not null;uniqueIndex:uq_tenant_usage_reservation_ref" json:"ref_no"`
	ModelKey                   string     `gorm:"type:varchar(128);not null;default:''" json:"model_key"`
	PricingID                  string     `gorm:"type:varchar(36);not null;default:''" json:"pricing_id"`
	ServicePricingID           string     `gorm:"type:varchar(36);not null;default:''" json:"service_pricing_id"`
	EstimatedBaseCostNanoUSD   int64      `gorm:"column:estimated_base_cost_nanousd;not null;default:0" json:"estimated_base_cost_nanousd"`
	EstimatedBilledPointMicros int64      `gorm:"not null;default:0" json:"estimated_billed_point_micros"`
	ReservedPeriodPointMicros  int64      `gorm:"not null;default:0" json:"reserved_period_point_micros"`
	ReservedBalancePointMicros int64      `gorm:"not null;default:0" json:"reserved_balance_point_micros"`
	PeriodLimitPointMicros     int64      `gorm:"not null;default:0" json:"period_limit_point_micros"`
	PeriodStartAt              *time.Time `json:"period_start_at,omitempty"`
	PeriodEndAt                *time.Time `json:"period_end_at,omitempty"`
	Status                     string     `gorm:"type:varchar(32);not null" json:"status"`
	UsageLedgerID              string     `gorm:"type:varchar(36);not null;default:''" json:"usage_ledger_id"`
	ExpiresAt                  time.Time  `json:"expires_at"`
	SettledAt                  *time.Time `json:"settled_at,omitempty"`
	ReleasedAt                 *time.Time `json:"released_at,omitempty"`
	ReconciliationAt           *time.Time `json:"reconciliation_at,omitempty"`
	FailureCode                string     `gorm:"type:varchar(64);not null;default:''" json:"failure_code"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

func (TenantUsageReservation) TableName() string { return "tenant_usage_reservations" }

type TenantUsageLedger struct {
	ID                        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID                  uint64    `gorm:"not null;uniqueIndex:uq_tenant_usage_ledger_ref" json:"tenant_id"`
	ActorUserID               string    `gorm:"type:varchar(64);not null;default:''" json:"actor_user_id"`
	UsageScope                string    `gorm:"type:varchar(32);not null" json:"usage_scope"`
	AllocationID              string    `gorm:"type:varchar(36);not null;default:''" json:"allocation_id"`
	RefNo                     string    `gorm:"type:varchar(160);not null;uniqueIndex:uq_tenant_usage_ledger_ref" json:"ref_no"`
	Source                    string    `gorm:"type:varchar(32);not null;default:'web'" json:"source"`
	SessionID                 string    `gorm:"type:varchar(64);not null;default:''" json:"session_id"`
	ServiceCode               string    `gorm:"type:varchar(64);not null;default:'chat.completion'" json:"service_code"`
	ModelID                   string    `gorm:"type:varchar(64);not null;default:''" json:"model_id"`
	ModelKey                  string    `gorm:"type:varchar(128);not null;default:''" json:"model_key"`
	Provider                  string    `gorm:"type:varchar(64);not null;default:''" json:"provider"`
	PricingID                 string    `gorm:"type:varchar(36);not null;default:''" json:"pricing_id"`
	PricingVersion            int       `gorm:"not null;default:0" json:"pricing_version"`
	ServicePricingID          string    `gorm:"type:varchar(36);not null;default:''" json:"service_pricing_id"`
	ServicePricingVersion     int       `gorm:"not null;default:0" json:"service_pricing_version"`
	ServiceUnits              int64     `gorm:"not null;default:0" json:"service_units"`
	InputTokens               int64     `gorm:"not null;default:0" json:"input_tokens"`
	CachedTokens              int64     `gorm:"not null;default:0" json:"cached_tokens"`
	OutputTokens              int64     `gorm:"not null;default:0" json:"output_tokens"`
	ReasoningTokens           int64     `gorm:"not null;default:0" json:"reasoning_tokens"`
	CallCount                 int64     `gorm:"not null;default:1" json:"call_count"`
	DurationMillis            int64     `gorm:"not null;default:0" json:"duration_millis"`
	BaseCostNanoUSD           int64     `gorm:"column:base_cost_nanousd;not null;default:0" json:"base_cost_nanousd"`
	RatedCostNanoUSD          int64     `gorm:"column:rated_cost_nanousd;not null;default:0" json:"rated_cost_nanousd"`
	BilledPointMicros         int64     `gorm:"not null;default:0" json:"billed_point_micros"`
	PeriodCoveredPointMicros  int64     `gorm:"not null;default:0" json:"period_covered_point_micros"`
	BalanceChargedPointMicros int64     `gorm:"not null;default:0" json:"balance_charged_point_micros"`
	BalanceAfterPointMicros   *int64    `json:"balance_after_point_micros,omitempty"`
	Status                    string    `gorm:"type:varchar(32);not null;default:'observed'" json:"status"`
	FailureCode               string    `gorm:"type:varchar(64);not null;default:''" json:"failure_code"`
	BillingAt                 time.Time `json:"billing_at"`
	UsageDate                 time.Time `gorm:"type:date" json:"usage_date"`
	PricingSnapshotJSON       JSON      `gorm:"type:jsonb;not null" json:"pricing_snapshot_json"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

func (TenantUsageLedger) TableName() string { return "tenant_usage_ledgers" }

type BillingUsageLedgerSummary struct {
	TenantUsageLedger
	TenantName string `json:"tenant_name"`
}

type BillingModelUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CallCount       int64 `json:"call_count"`
	DurationMillis  int64 `json:"duration_millis"`
	ServiceUnits    int64 `json:"service_units"`
}

type BillingUsageStartRequest struct {
	TenantID       uint64            `json:"tenant_id"`
	ActorUserID    string            `json:"actor_user_id"`
	RefNo          string            `json:"ref_no"`
	Source         string            `json:"source"`
	SessionID      string            `json:"session_id"`
	ServiceCode    string            `json:"service_code"`
	ModelID        string            `json:"model_id"`
	ModelKey       string            `json:"model_key"`
	EstimatedUsage BillingModelUsage `json:"estimated_usage"`
}

type BillingUsageHandle struct {
	Request                       BillingUsageStartRequest
	Mode                          string
	UsageScope                    string
	AllocationID                  string
	FailureCode                   string
	MemberLimitMode               string
	MemberMonthlyLimitPointMicros int64
	MemberOveragePolicy           string
	Price                         *BillingModelPrice
	ServicePrice                  *BillingServicePrice
	Plan                          *BillingPlan
	Subscription                  *TenantSubscription
	Allocation                    *TenantMemberCreditAllocation
	EnterprisePolicy              *TenantBillingPolicy
	Reservation                   *TenantUsageReservation
}

const (
	MemberLimitModeInherit   = "inherit"
	MemberLimitModeCustom    = "custom"
	MemberLimitModeUnlimited = "unlimited"

	MemberOveragePolicyInherit              = "inherit"
	MemberOveragePolicyBlock                = "block"
	MemberOveragePolicyUseEnterpriseBalance = "use_enterprise_balance"

	BillingUsageScopePersonal         = "personal_usage"
	BillingUsageScopeEnterprise       = "enterprise_usage"
	BillingUsageScopeEnterpriseLegacy = "enterprise_allocated_usage"
)

type TenantBillingPolicy struct {
	TenantID                             uint64    `gorm:"primaryKey" json:"tenant_id"`
	DefaultMemberMonthlyLimitPointMicros int64     `gorm:"not null;default:100000000" json:"default_member_monthly_limit_point_micros"`
	MemberOveragePolicy                  string    `gorm:"type:varchar(32);not null;default:'block'" json:"member_overage_policy"`
	UpdatedByUserID                      string    `gorm:"type:varchar(64);not null;default:''" json:"updated_by_user_id"`
	CreatedAt                            time.Time `json:"created_at"`
	UpdatedAt                            time.Time `json:"updated_at"`
}

func (TenantBillingPolicy) TableName() string { return "tenant_billing_policies" }

// TenantMemberCreditAllocation scopes an enterprise member's usable credits
// for one billing period. Usage is derived from tenant_usage_ledgers by
// allocation_id; this row is the authorization envelope, not a mutable balance.
type TenantMemberCreditAllocation struct {
	ID                          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID                    uint64    `gorm:"not null;uniqueIndex:uq_tenant_member_credit_allocation_period;index" json:"tenant_id"`
	UserID                      string    `gorm:"type:varchar(64);not null;uniqueIndex:uq_tenant_member_credit_allocation_period;index" json:"user_id"`
	PeriodStartAt               time.Time `gorm:"not null;uniqueIndex:uq_tenant_member_credit_allocation_period" json:"period_start_at"`
	PeriodEndAt                 time.Time `gorm:"not null;uniqueIndex:uq_tenant_member_credit_allocation_period" json:"period_end_at"`
	AllocatedPeriodPointMicros  int64     `gorm:"not null;default:0" json:"allocated_period_point_micros"`
	AllocatedBalancePointMicros int64     `gorm:"not null;default:0" json:"allocated_balance_point_micros"`
	LimitMode                   string    `gorm:"type:varchar(32);not null;default:'inherit'" json:"limit_mode"`
	MonthlyLimitPointMicros     int64     `gorm:"not null;default:0" json:"monthly_limit_point_micros"`
	OveragePolicy               string    `gorm:"type:varchar(32);not null;default:'inherit'" json:"overage_policy"`
	Status                      string    `gorm:"type:varchar(32);not null;default:'active';index" json:"status"`
	CreatedByUserID             string    `gorm:"type:varchar(64);not null;default:''" json:"created_by_user_id"`
	UpdatedByUserID             string    `gorm:"type:varchar(64);not null;default:''" json:"updated_by_user_id"`
	SnapshotJSON                JSON      `gorm:"type:jsonb;not null" json:"snapshot_json"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

func (TenantMemberCreditAllocation) TableName() string {
	return "tenant_member_credit_allocations"
}

type TenantMemberCreditAllocationSummary struct {
	TenantMemberCreditAllocation
	EffectiveMonthlyLimitPointMicros int64      `json:"effective_monthly_limit_point_micros"`
	EffectiveOveragePolicy           string     `json:"effective_overage_policy"`
	UsedPointMicros                  int64      `json:"used_point_micros"`
	InputTokens                      int64      `json:"input_tokens"`
	OutputTokens                     int64      `json:"output_tokens"`
	ReasoningTokens                  int64      `json:"reasoning_tokens"`
	LedgerCount                      int64      `json:"ledger_count"`
	LastBillingAt                    *time.Time `json:"last_billing_at,omitempty"`
}

type BillingActorUsageSummary struct {
	ActorUserID                 string     `json:"actor_user_id"`
	LedgerCount                 int64      `json:"ledger_count"`
	PersonalLedgerCount         int64      `json:"personal_ledger_count"`
	EnterpriseLedgerCount       int64      `json:"enterprise_ledger_count"`
	InputTokens                 int64      `json:"input_tokens"`
	CachedTokens                int64      `json:"cached_tokens"`
	OutputTokens                int64      `json:"output_tokens"`
	ReasoningTokens             int64      `json:"reasoning_tokens"`
	BilledPointMicros           int64      `json:"billed_point_micros"`
	PersonalBilledPointMicros   int64      `json:"personal_billed_point_micros"`
	EnterpriseBilledPointMicros int64      `json:"enterprise_billed_point_micros"`
	LastBillingAt               *time.Time `json:"last_billing_at,omitempty"`
}

type BillingOverviewPlan struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Edition              string `json:"edition"`
	Status               string `json:"status"`
	IncludedStorageBytes int64  `json:"included_storage_bytes"`
	IncludedPointMicros  int64  `json:"included_point_micros"`
}

type BillingOverviewSubscription struct {
	Status             string     `json:"status"`
	BillingInterval    string     `json:"billing_interval"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	Source             string     `json:"source"`
}

type BillingOverviewStorage struct {
	UsedBytes      int64   `json:"used_bytes"`
	QuotaBytes     int64   `json:"quota_bytes"`
	RemainingBytes int64   `json:"remaining_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	Unlimited      bool    `json:"unlimited"`
	Status         string  `json:"status"`
	Visible        bool    `json:"visible"`
}

type BillingOverviewCredits struct {
	BalancePointMicros         int64 `json:"balance_point_micros"`
	PeriodPointMicros          int64 `json:"period_point_micros"`
	PeriodUsedPointMicros      int64 `json:"period_used_point_micros"`
	PeriodRemainingPointMicros int64 `json:"period_remaining_point_micros"`
	Visible                    bool  `json:"visible"`
}

type BillingOverviewMemberUsage struct {
	MemberPolicyID              string `json:"member_policy_id"`
	LimitMode                   string `json:"limit_mode"`
	MonthlyLimitPointMicros     int64  `json:"monthly_limit_point_micros"`
	MonthlyUsedPointMicros      int64  `json:"monthly_used_point_micros"`
	MonthlyRemainingPointMicros int64  `json:"monthly_remaining_point_micros"`
	OveragePolicy               string `json:"overage_policy"`
}

type BillingOverview struct {
	TenantID          uint64                      `json:"tenant_id"`
	TenantName        string                      `json:"tenant_name"`
	SpaceType         SpaceType                   `json:"space_type"`
	Policy            BillingRuntimePolicy        `json:"policy"`
	Plan              BillingOverviewPlan         `json:"plan"`
	Subscription      BillingOverviewSubscription `json:"subscription"`
	Storage           BillingOverviewStorage      `json:"storage"`
	Credits           BillingOverviewCredits      `json:"credits"`
	MemberUsage       *BillingOverviewMemberUsage `json:"member_usage,omitempty"`
	CompatibilityMode bool                        `json:"compatibility_mode"`
}

type BillingSubscriptionSummary struct {
	TenantSubscription
	TenantName string `json:"tenant_name"`
	PlanCode   string `json:"plan_code"`
	PlanName   string `json:"plan_name"`
	SpaceType  string `json:"space_type"`
}

type BillingCreditAccountSummary struct {
	TenantCreditAccount
	TenantName     string `json:"tenant_name"`
	SpaceType      string `json:"space_type"`
	SubjectType    string `json:"subject_type"`
	SubjectID      string `json:"subject_id"`
	SubjectName    string `json:"subject_name"`
	SubjectContact string `json:"subject_contact,omitempty"`
	TenantCount    int    `json:"tenant_count"`
}

type BillingPaymentOrderSummary struct {
	BillingPaymentOrder
	TenantName string `json:"tenant_name"`
	PlanName   string `json:"plan_name"`
	ItemName   string `json:"item_name"`
}

type BillingStorageTransactionSummary struct {
	TenantStorageTransaction
	TenantName string `json:"tenant_name"`
}

const (
	StorageReservationStatusReserved  = "reserved"
	StorageReservationStatusCommitted = "committed"
	StorageReservationStatusReleased  = "released"
	StorageReservationStatusExpired   = "expired"
)

// TenantStorageReservation protects the shared workspace quota while an
// asynchronous operation is still producing its final storage footprint.
type TenantStorageReservation struct {
	ID             string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID       uint64     `gorm:"not null;uniqueIndex:uq_tenant_storage_reservation_ref;index" json:"tenant_id"`
	ActorUserID    string     `gorm:"type:varchar(64);not null;default:''" json:"actor_user_id"`
	RefNo          string     `gorm:"type:varchar(160);not null;uniqueIndex:uq_tenant_storage_reservation_ref" json:"ref_no"`
	Operation      string     `gorm:"type:varchar(32);not null" json:"operation"`
	RequestedBytes int64      `gorm:"not null;default:0" json:"requested_bytes"`
	ActualBytes    int64      `gorm:"not null;default:0" json:"actual_bytes"`
	Status         string     `gorm:"type:varchar(32);not null;default:'reserved';index" json:"status"`
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expires_at"`
	CommittedAt    *time.Time `json:"committed_at,omitempty"`
	ReleasedAt     *time.Time `json:"released_at,omitempty"`
	FailureCode    string     `gorm:"type:varchar(64);not null;default:''" json:"failure_code"`
	MetadataJSON   JSON       `gorm:"type:jsonb;not null" json:"metadata_json"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (TenantStorageReservation) TableName() string {
	return "tenant_storage_reservations"
}

// TenantStorageTransaction is the append-only authority for storage usage
// changes. Positive amounts consume quota and negative amounts release it.
type TenantStorageTransaction struct {
	ID                    string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID              uint64    `gorm:"not null;uniqueIndex:uq_tenant_storage_transaction_ref;index" json:"tenant_id"`
	ActorUserID           string    `gorm:"type:varchar(64);not null;default:''" json:"actor_user_id"`
	ReservationID         string    `gorm:"type:varchar(36);not null;default:''" json:"reservation_id"`
	RefNo                 string    `gorm:"type:varchar(160);not null;uniqueIndex:uq_tenant_storage_transaction_ref" json:"ref_no"`
	Operation             string    `gorm:"type:varchar(32);not null" json:"operation"`
	AmountBytes           int64     `gorm:"not null" json:"amount_bytes"`
	StorageUsedAfterBytes int64     `gorm:"not null" json:"storage_used_after_bytes"`
	MetadataJSON          JSON      `gorm:"type:jsonb;not null" json:"metadata_json"`
	CreatedAt             time.Time `json:"created_at"`
}

func (TenantStorageTransaction) TableName() string {
	return "tenant_storage_transactions"
}
