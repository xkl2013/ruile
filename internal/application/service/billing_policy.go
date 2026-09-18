package service

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type billingPolicyService struct {
	settings interfaces.SystemSettingService
}

func NewBillingPolicyService(settings interfaces.SystemSettingService) interfaces.BillingPolicyService {
	return &billingPolicyService{settings: settings}
}

func (s *billingPolicyService) RuntimePolicy(ctx context.Context) types.BillingRuntimePolicy {
	policy := types.BillingRuntimePolicy{
		Enabled:                              false,
		EnforcementMode:                      "off",
		PointMicrosPerUSD:                    types.PointMicrosPerPoint,
		DefaultModelMultiplierPPM:            1_000_000,
		DefaultServiceMultiplierPPM:          1_000_000,
		DefaultMemberMonthlyLimitPointMicros: 100 * types.PointMicrosPerPoint,
		DefaultMemberAllocationPointMicros:   100 * types.PointMicrosPerPoint,
	}
	if s.settings == nil {
		return policy
	}
	policy.Enabled = s.settings.GetBool(ctx, "billing.enabled", "", true)
	policy.EnforcementMode = strings.ToLower(strings.TrimSpace(
		s.settings.GetString(ctx, "billing.enforcement_mode", "", "observe"),
	))
	switch policy.EnforcementMode {
	case "off", "observe", "enforce":
	default:
		policy.EnforcementMode = "off"
	}
	policy.PointMicrosPerUSD = s.settings.GetInt(
		ctx,
		"billing.point_micros_per_usd",
		"",
		types.PointMicrosPerPoint,
	)
	if policy.PointMicrosPerUSD <= 0 {
		policy.PointMicrosPerUSD = types.PointMicrosPerPoint
	}
	policy.DefaultModelMultiplierPPM = s.settings.GetInt(
		ctx,
		"billing.default_model_multiplier_ppm",
		"",
		1_000_000,
	)
	if policy.DefaultModelMultiplierPPM <= 0 {
		policy.DefaultModelMultiplierPPM = 1_000_000
	}
	policy.DefaultServiceMultiplierPPM = s.settings.GetInt(
		ctx,
		"billing.default_service_multiplier_ppm",
		"",
		1_000_000,
	)
	if policy.DefaultServiceMultiplierPPM <= 0 {
		policy.DefaultServiceMultiplierPPM = 1_000_000
	}
	defaultMemberMonthlyLimitPoints := s.settings.GetInt(
		ctx,
		"billing.default_member_monthly_limit_points",
		"",
		-1,
	)
	if defaultMemberMonthlyLimitPoints < 0 {
		defaultMemberMonthlyLimitPoints = s.settings.GetInt(
			ctx,
			"billing.default_member_allocation_points",
			"",
			100,
		)
	}
	if defaultMemberMonthlyLimitPoints < 0 {
		defaultMemberMonthlyLimitPoints = 0
	}
	policy.DefaultMemberMonthlyLimitPointMicros =
		defaultMemberMonthlyLimitPoints * types.PointMicrosPerPoint
	policy.DefaultMemberAllocationPointMicros =
		policy.DefaultMemberMonthlyLimitPointMicros
	return policy
}
