package pricing

import "testing"

func TestCalculateModelChargeSeparatesCachedInput(t *testing.T) {
	charge, err := CalculateModelCharge(
		ModelRates{
			InputNanoUSDPerMTokens:     2_000_000_000,
			OutputNanoUSDPerMTokens:    8_000_000_000,
			CacheReadNanoUSDPerMTokens: 500_000_000,
			CallNanoUSDPerCall:         10_000,
		},
		ModelUsage{
			InputTokens:    1_000,
			CachedTokens:   400,
			OutputTokens:   200,
			CallCount:      1,
			DurationMillis: 250,
		},
		1_000_000,
		1_250_000,
		800_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if charge.UncachedInputTokens != 600 {
		t.Fatalf("uncached tokens=%d, want 600", charge.UncachedInputTokens)
	}
	if charge.InputCostNanoUSD != 1_200_000 || charge.CacheReadCostNanoUSD != 200_000 {
		t.Fatalf("input=%d cache=%d", charge.InputCostNanoUSD, charge.CacheReadCostNanoUSD)
	}
	if charge.BaseCostNanoUSD != 3_010_000 {
		t.Fatalf("base=%d, want 3010000", charge.BaseCostNanoUSD)
	}
	if charge.RatedCostNanoUSD != 3_010_000 {
		t.Fatalf("rated=%d, want 3010000", charge.RatedCostNanoUSD)
	}
	if charge.BilledPointMicros != 3_010 {
		t.Fatalf("points=%d, want 3010", charge.BilledPointMicros)
	}
}

func TestCalculateModelChargeUsesInputPriceWhenCachePriceUnset(t *testing.T) {
	charge, err := CalculateModelCharge(
		ModelRates{InputNanoUSDPerMTokens: 1_000_000_000},
		ModelUsage{InputTokens: 100, CachedTokens: 100},
		1_000_000,
		1_000_000,
		1_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if charge.CacheReadCostNanoUSD != 100_000 || charge.InputCostNanoUSD != 0 {
		t.Fatalf("charge=%+v", charge)
	}
}
