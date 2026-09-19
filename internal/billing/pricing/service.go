package pricing

import "errors"

type ServiceRates struct {
	NanoUSDPerCall int64
	NanoUSDPerUnit int64
}

type ServiceUsage struct {
	CallCount int64
	Units     int64
}

type ServiceCharge struct {
	CallCostNanoUSD   int64
	UnitCostNanoUSD   int64
	BaseCostNanoUSD   int64
	RatedCostNanoUSD  int64
	BilledPointMicros int64
}

func CalculateServiceCharge(
	mode string,
	rates ServiceRates,
	usage ServiceUsage,
	pointMicrosPerUSD int64,
	serviceMultiplierPPM int64,
	planMultiplierPPM int64,
) (ServiceCharge, error) {
	var result ServiceCharge
	if usage.CallCount < 0 || usage.Units < 0 {
		return result, errors.New("pricing: service usage must be non-negative")
	}
	if pointMicrosPerUSD <= 0 || serviceMultiplierPPM <= 0 || planMultiplierPPM <= 0 {
		return result, errors.New("pricing: exchange rate and multipliers must be positive")
	}
	var err error
	switch mode {
	case "", "call":
		result.CallCostNanoUSD, err = roundedUnitCost(
			usage.CallCount,
			rates.NanoUSDPerCall,
			1,
		)
	case "unit":
		result.UnitCostNanoUSD, err = roundedUnitCost(
			usage.Units,
			rates.NanoUSDPerUnit,
			1,
		)
	default:
		return result, errors.New("pricing: unsupported service pricing mode")
	}
	if err != nil {
		return ServiceCharge{}, err
	}
	result.BaseCostNanoUSD = result.CallCostNanoUSD + result.UnitCostNanoUSD
	serviceRated, err := ApplyMultiplier(result.BaseCostNanoUSD, serviceMultiplierPPM)
	if err != nil {
		return ServiceCharge{}, err
	}
	result.RatedCostNanoUSD, err = ApplyMultiplier(serviceRated, planMultiplierPPM)
	if err != nil {
		return ServiceCharge{}, err
	}
	result.BilledPointMicros, err = BaseCostToPointMicros(
		result.RatedCostNanoUSD,
		pointMicrosPerUSD,
		MultiplierScale,
	)
	if err != nil {
		return ServiceCharge{}, err
	}
	return result, nil
}

type CombinedCharge struct {
	Model             ModelCharge
	Service           ServiceCharge
	BaseCostNanoUSD   int64
	RatedCostNanoUSD  int64
	BilledPointMicros int64
}

// CalculateCombinedCharge applies model and service multipliers independently,
// adds their rated costs, and applies the plan multiplier exactly once.
func CalculateCombinedCharge(
	modelRates ModelRates,
	modelUsage ModelUsage,
	modelMultiplierPPM int64,
	serviceMode string,
	serviceRates ServiceRates,
	serviceUsage ServiceUsage,
	serviceMultiplierPPM int64,
	planMultiplierPPM int64,
	pointMicrosPerUSD int64,
) (CombinedCharge, error) {
	var result CombinedCharge
	modelCharge, err := CalculateModelCharge(
		modelRates,
		modelUsage,
		pointMicrosPerUSD,
		modelMultiplierPPM,
		MultiplierScale,
	)
	if err != nil {
		return result, err
	}
	serviceCharge, err := CalculateServiceCharge(
		serviceMode,
		serviceRates,
		serviceUsage,
		pointMicrosPerUSD,
		serviceMultiplierPPM,
		MultiplierScale,
	)
	if err != nil {
		return result, err
	}
	result.Model = modelCharge
	result.Service = serviceCharge
	result.BaseCostNanoUSD = modelCharge.BaseCostNanoUSD + serviceCharge.BaseCostNanoUSD
	result.RatedCostNanoUSD, err = ApplyMultiplier(
		modelCharge.RatedCostNanoUSD+serviceCharge.RatedCostNanoUSD,
		planMultiplierPPM,
	)
	if err != nil {
		return CombinedCharge{}, err
	}
	result.BilledPointMicros, err = BaseCostToPointMicros(
		result.RatedCostNanoUSD,
		pointMicrosPerUSD,
		MultiplierScale,
	)
	if err != nil {
		return CombinedCharge{}, err
	}
	return result, nil
}
