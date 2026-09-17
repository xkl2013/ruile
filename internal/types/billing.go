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
	Enabled                   bool   `json:"enabled"`
	EnforcementMode           string `json:"enforcement_mode"`
	PointMicrosPerUSD         int64  `json:"point_micros_per_usd"`
	DefaultModelMultiplierPPM int64  `json:"default_model_multiplier_ppm"`
}

// BillingModelPrice is an immutable version of a model's upstream cost.
// model_key is the provider-facing model name; model_id is stored separately
// on usage ledgers so tenant-local model records can be renamed safely.
type BillingModelPrice struct {
	ID                          string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ModelKey                    string     `gorm:"type:varchar(128);not null;uniqueIndex:uq_billing_model_price_version" json:"model_key"`
	Provider                    string     `gorm:"type:varchar(64);not null;default:''" json:"provider"`
	PricingMode                 string     `gorm:"type:varchar(32);not null;default:'token'" json:"pricing_mode"`
	InputNanoUSDPerMTokens      int64      `gorm:"not null;default:0" json:"input_nanousd_per_m_tokens"`
	OutputNanoUSDPerMTokens     int64      `gorm:"not null;default:0" json:"output_nanousd_per_m_tokens"`
	CacheReadNanoUSDPerMTokens  int64      `gorm:"not null;default:0" json:"cache_read_nanousd_per_m_tokens"`
	CacheWriteNanoUSDPerMTokens int64      `gorm:"not null;default:0" json:"cache_write_nanousd_per_m_tokens"`
	CallNanoUSDPerCall          int64      `gorm:"not null;default:0" json:"call_nanousd_per_call"`
	DurationNanoUSDPerSecond    int64      `gorm:"not null;default:0" json:"duration_nanousd_per_second"`
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
	ModelMultiplierPPM          int64      `json:"model_multiplier_ppm"`
	EffectiveAt                 *time.Time `json:"effective_at,omitempty"`
	ExpiresAt                   *time.Time `json:"expires_at,omitempty"`
	Status                      string     `json:"status"`
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
	EstimatedBaseCostNanoUSD   int64      `gorm:"not null;default:0" json:"estimated_base_cost_nanousd"`
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
	InputTokens               int64     `gorm:"not null;default:0" json:"input_tokens"`
	CachedTokens              int64     `gorm:"not null;default:0" json:"cached_tokens"`
	OutputTokens              int64     `gorm:"not null;default:0" json:"output_tokens"`
	ReasoningTokens           int64     `gorm:"not null;default:0" json:"reasoning_tokens"`
	CallCount                 int64     `gorm:"not null;default:1" json:"call_count"`
	DurationMillis            int64     `gorm:"not null;default:0" json:"duration_millis"`
	BaseCostNanoUSD           int64     `gorm:"not null;default:0" json:"base_cost_nanousd"`
	RatedCostNanoUSD          int64     `gorm:"not null;default:0" json:"rated_cost_nanousd"`
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
	Request      BillingUsageStartRequest
	Mode         string
	UsageScope   string
	FailureCode  string
	Price        *BillingModelPrice
	Plan         *BillingPlan
	Subscription *TenantSubscription
	Reservation  *TenantUsageReservation
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
}

type BillingOverviewCredits struct {
	BalancePointMicros int64 `json:"balance_point_micros"`
	PeriodPointMicros  int64 `json:"period_point_micros"`
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
	TenantName string `json:"tenant_name"`
	SpaceType  string `json:"space_type"`
}
