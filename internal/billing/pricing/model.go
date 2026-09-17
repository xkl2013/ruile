package pricing

import (
	"errors"
	"math/big"
)

type ModelRates struct {
	InputNanoUSDPerMTokens      int64
	OutputNanoUSDPerMTokens     int64
	CacheReadNanoUSDPerMTokens  int64
	CacheWriteNanoUSDPerMTokens int64
	CallNanoUSDPerCall          int64
	DurationNanoUSDPerSecond    int64
}

type ModelUsage struct {
	InputTokens     int64
	CachedTokens    int64
	OutputTokens    int64
	ReasoningTokens int64
	CallCount       int64
	DurationMillis  int64
}

type ModelCharge struct {
	UncachedInputTokens  int64
	InputCostNanoUSD     int64
	CacheReadCostNanoUSD int64
	OutputCostNanoUSD    int64
	CallCostNanoUSD      int64
	DurationCostNanoUSD  int64
	BaseCostNanoUSD      int64
	RatedCostNanoUSD     int64
	BilledPointMicros    int64
}

func roundedUnitCost(units, rate, divisor int64) (int64, error) {
	if units < 0 || rate < 0 {
		return 0, errors.New("pricing: usage and rates must be non-negative")
	}
	result, err := RoundDiv(
		new(big.Int).Mul(big.NewInt(units), big.NewInt(rate)),
		big.NewInt(divisor),
	)
	if err != nil {
		return 0, err
	}
	if !result.IsInt64() {
		return 0, errors.New("pricing: unit cost overflows int64")
	}
	return result.Int64(), nil
}

// CalculateModelCharge computes a model charge without floating point. Input
// tokens include cached tokens, so the cached subset is removed before the
// normal input rate is applied.
func CalculateModelCharge(
	rates ModelRates,
	usage ModelUsage,
	pointMicrosPerUSD int64,
	modelMultiplierPPM int64,
	planMultiplierPPM int64,
) (ModelCharge, error) {
	var result ModelCharge
	if usage.InputTokens < 0 || usage.CachedTokens < 0 || usage.OutputTokens < 0 ||
		usage.ReasoningTokens < 0 || usage.CallCount < 0 || usage.DurationMillis < 0 {
		return result, errors.New("pricing: usage must be non-negative")
	}
	if pointMicrosPerUSD <= 0 || modelMultiplierPPM <= 0 || planMultiplierPPM <= 0 {
		return result, errors.New("pricing: exchange rate and multipliers must be positive")
	}

	cachedTokens := min(usage.CachedTokens, usage.InputTokens)
	result.UncachedInputTokens = usage.InputTokens - cachedTokens
	var err error
	result.InputCostNanoUSD, err = roundedUnitCost(
		result.UncachedInputTokens,
		rates.InputNanoUSDPerMTokens,
		1_000_000,
	)
	if err != nil {
		return ModelCharge{}, err
	}
	cacheRate := rates.CacheReadNanoUSDPerMTokens
	if cacheRate == 0 {
		cacheRate = rates.InputNanoUSDPerMTokens
	}
	result.CacheReadCostNanoUSD, err = roundedUnitCost(cachedTokens, cacheRate, 1_000_000)
	if err != nil {
		return ModelCharge{}, err
	}
	result.OutputCostNanoUSD, err = roundedUnitCost(
		usage.OutputTokens+usage.ReasoningTokens,
		rates.OutputNanoUSDPerMTokens,
		1_000_000,
	)
	if err != nil {
		return ModelCharge{}, err
	}
	result.CallCostNanoUSD, err = roundedUnitCost(usage.CallCount, rates.CallNanoUSDPerCall, 1)
	if err != nil {
		return ModelCharge{}, err
	}
	result.DurationCostNanoUSD, err = roundedUnitCost(
		usage.DurationMillis,
		rates.DurationNanoUSDPerSecond,
		1_000,
	)
	if err != nil {
		return ModelCharge{}, err
	}
	result.BaseCostNanoUSD = result.InputCostNanoUSD +
		result.CacheReadCostNanoUSD +
		result.OutputCostNanoUSD +
		result.CallCostNanoUSD +
		result.DurationCostNanoUSD

	modelRated, err := ApplyMultiplier(result.BaseCostNanoUSD, modelMultiplierPPM)
	if err != nil {
		return ModelCharge{}, err
	}
	result.RatedCostNanoUSD, err = ApplyMultiplier(modelRated, planMultiplierPPM)
	if err != nil {
		return ModelCharge{}, err
	}
	result.BilledPointMicros, err = BaseCostToPointMicros(
		result.RatedCostNanoUSD,
		pointMicrosPerUSD,
		MultiplierScale,
	)
	if err != nil {
		return ModelCharge{}, err
	}
	return result, nil
}
