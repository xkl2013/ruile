package pricing

import (
	"encoding/json"
	"errors"
)

type ModelRateTier struct {
	UpToInputTokens             int64 `json:"up_to_input_tokens"`
	InputNanoUSDPerMTokens      int64 `json:"input_nanousd_per_m_tokens"`
	OutputNanoUSDPerMTokens     int64 `json:"output_nanousd_per_m_tokens"`
	CacheReadNanoUSDPerMTokens  int64 `json:"cache_read_nanousd_per_m_tokens"`
	CacheWriteNanoUSDPerMTokens int64 `json:"cache_write_nanousd_per_m_tokens"`
	CallNanoUSDPerCall          int64 `json:"call_nanousd_per_call"`
	DurationNanoUSDPerSecond    int64 `json:"duration_nanousd_per_second"`
}

type TieredModelRates struct {
	Tiers []ModelRateTier `json:"tiers"`
}

func ResolveTieredModelRates(raw []byte, inputTokens int64) (ModelRates, error) {
	if inputTokens < 0 {
		return ModelRates{}, errors.New("pricing: input tokens must be non-negative")
	}
	var config TieredModelRates
	if len(raw) == 0 || json.Unmarshal(raw, &config) != nil || len(config.Tiers) == 0 {
		return ModelRates{}, errors.New("pricing: invalid tiered pricing configuration")
	}
	for _, tier := range config.Tiers {
		if tier.UpToInputTokens < 0 {
			return ModelRates{}, errors.New("pricing: tier limit must be non-negative")
		}
		if tier.UpToInputTokens == 0 || inputTokens <= tier.UpToInputTokens {
			return ModelRates{
				InputNanoUSDPerMTokens:      tier.InputNanoUSDPerMTokens,
				OutputNanoUSDPerMTokens:     tier.OutputNanoUSDPerMTokens,
				CacheReadNanoUSDPerMTokens:  tier.CacheReadNanoUSDPerMTokens,
				CacheWriteNanoUSDPerMTokens: tier.CacheWriteNanoUSDPerMTokens,
				CallNanoUSDPerCall:          tier.CallNanoUSDPerCall,
				DurationNanoUSDPerSecond:    tier.DurationNanoUSDPerSecond,
			}, nil
		}
	}
	return ModelRates{}, errors.New("pricing: tiered pricing has no matching tier")
}
