package pricing

import "testing"

func TestCalculateCombinedChargeSeparatesModelAndServiceMultipliers(t *testing.T) {
	charge, err := CalculateCombinedCharge(
		ModelRates{InputNanoUSDPerMTokens: 1_000_000_000},
		ModelUsage{InputTokens: 1_000},
		2_000_000,
		"call",
		ServiceRates{NanoUSDPerCall: 500_000},
		ServiceUsage{CallCount: 1},
		3_000_000,
		4_000_000,
		1_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if charge.BaseCostNanoUSD != 1_500_000 {
		t.Fatalf("base=%d, want 1500000", charge.BaseCostNanoUSD)
	}
	// Model: 1,000,000 * 2; service: 500,000 * 3; plan: sum * 4.
	if charge.RatedCostNanoUSD != 14_000_000 {
		t.Fatalf("rated=%d, want 14000000", charge.RatedCostNanoUSD)
	}
	if charge.BilledPointMicros != 14_000 {
		t.Fatalf("points=%d, want 14000", charge.BilledPointMicros)
	}
}

func TestResolveTieredModelRates(t *testing.T) {
	raw := []byte(`{"tiers":[
		{"up_to_input_tokens":256000,"input_nanousd_per_m_tokens":2000000000},
		{"up_to_input_tokens":0,"input_nanousd_per_m_tokens":4000000000}
	]}`)
	first, err := ResolveTieredModelRates(raw, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if first.InputNanoUSDPerMTokens != 2_000_000_000 {
		t.Fatalf("first rate=%d", first.InputNanoUSDPerMTokens)
	}
	second, err := ResolveTieredModelRates(raw, 300_000)
	if err != nil {
		t.Fatal(err)
	}
	if second.InputNanoUSDPerMTokens != 4_000_000_000 {
		t.Fatalf("second rate=%d", second.InputNanoUSDPerMTokens)
	}
}
