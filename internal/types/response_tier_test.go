package types

import (
	"testing"
)

func TestResponseTierConfigNormalizeAndProfile(t *testing.T) {
	thinking := true
	cfg := ResponseTierConfig{
		DefaultTier: "unknown",
		Fast:        ResponseTierProfile{ModelID: "  fast-model  "},
		Balanced:    ResponseTierProfile{ModelID: "balanced-model"},
		Ultimate:    ResponseTierProfile{ModelID: "ultimate-model", Thinking: &thinking},
	}

	cfg.Normalize()

	if cfg.DefaultTier != ResponseTierBalanced {
		t.Fatalf("expected balanced default, got %q", cfg.DefaultTier)
	}
	if cfg.Fast.ModelID != "fast-model" {
		t.Fatalf("expected trimmed fast model, got %q", cfg.Fast.ModelID)
	}
	if got := cfg.Profile(ResponseTierUltimate); got.ModelID != "ultimate-model" || got.Thinking != &thinking {
		t.Fatalf("unexpected ultimate profile: %+v", got)
	}
}

func TestResponseTierConfigJSONValueAndScan(t *testing.T) {
	cfg := ResponseTierConfig{
		Enabled:     true,
		DefaultTier: ResponseTierFast,
		Fast:        ResponseTierProfile{ModelID: "fast-model"},
	}

	value, err := cfg.Value()
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	var scanned ResponseTierConfig
	if err := scanned.Scan(value); err != nil {
		t.Fatalf("scan config: %v", err)
	}
	if scanned.Enabled != cfg.Enabled || scanned.DefaultTier != cfg.DefaultTier || scanned.Fast.ModelID != cfg.Fast.ModelID {
		t.Fatalf("scanned config differs: got %+v want %+v", scanned, cfg)
	}
}
