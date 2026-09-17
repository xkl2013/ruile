// Package pricing contains integer-only billing arithmetic.
//
// The separation of base cost, composable fixed-point multipliers, and final
// billed units is adapted for Ruile from the DEEIX-Chat billing architecture.
// This file is a modified implementation, not a verbatim copy.
package pricing

import (
	"errors"
	"math/big"
)

const MultiplierScale int64 = 1_000_000

var ErrInvalidScale = errors.New("pricing: scale must be positive")

// RoundDiv divides numerator by denominator using half-away-from-zero
// rounding. big.Int keeps intermediate products deterministic and overflow-free.
func RoundDiv(numerator, denominator *big.Int) (*big.Int, error) {
	if denominator == nil || denominator.Sign() <= 0 {
		return nil, ErrInvalidScale
	}
	if numerator == nil || numerator.Sign() == 0 {
		return big.NewInt(0), nil
	}

	sign := numerator.Sign()
	absolute := new(big.Int).Abs(new(big.Int).Set(numerator))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(absolute, denominator, remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if sign < 0 {
		quotient.Neg(quotient)
	}
	return quotient, nil
}

// CeilDiv divides non-negative integer values and rounds toward positive
// infinity. Billing inputs are rejected when negative because a usage charge
// must never become an implicit refund.
func CeilDiv(numerator, denominator *big.Int) (*big.Int, error) {
	if denominator == nil || denominator.Sign() <= 0 {
		return nil, ErrInvalidScale
	}
	if numerator == nil || numerator.Sign() == 0 {
		return big.NewInt(0), nil
	}
	if numerator.Sign() < 0 {
		return nil, errors.New("pricing: ceil division requires a non-negative numerator")
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if remainder.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient, nil
}

// ApplyMultiplier applies a fixed-point multiplier where 1_000_000 means 1x.
func ApplyMultiplier(value, multiplierMicros int64) (int64, error) {
	result, err := RoundDiv(
		new(big.Int).Mul(big.NewInt(value), big.NewInt(multiplierMicros)),
		big.NewInt(MultiplierScale),
	)
	if err != nil {
		return 0, err
	}
	if !result.IsInt64() {
		return 0, errors.New("pricing: result overflows int64")
	}
	return result.Int64(), nil
}

// ComposeMultipliers combines any number of fixed-point multipliers.
func ComposeMultipliers(multipliers ...int64) (int64, error) {
	result := MultiplierScale
	var err error
	for _, multiplier := range multipliers {
		result, err = ApplyMultiplier(result, multiplier)
		if err != nil {
			return 0, err
		}
	}
	return result, nil
}

// BaseCostToPointMicros converts nano-USD cost into billed point micros, then
// applies the composed multiplier. The exchange rate is point micros per USD.
func BaseCostToPointMicros(baseCostNanoUSD, pointMicrosPerUSD, multiplierMicros int64) (int64, error) {
	ratedCost, err := ApplyMultiplier(baseCostNanoUSD, multiplierMicros)
	if err != nil {
		return 0, err
	}
	converted, err := CeilDiv(
		new(big.Int).Mul(big.NewInt(ratedCost), big.NewInt(pointMicrosPerUSD)),
		big.NewInt(1_000_000_000),
	)
	if err != nil {
		return 0, err
	}
	if !converted.IsInt64() {
		return 0, errors.New("pricing: converted value overflows int64")
	}
	return converted.Int64(), nil
}
