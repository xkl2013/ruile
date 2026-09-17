package pricing

import (
	"math/big"
	"testing"
)

func TestRoundDivHalfAwayFromZero(t *testing.T) {
	tests := []struct {
		numerator int64
		denom     int64
		want      int64
	}{
		{1, 2, 1},
		{4, 3, 1},
		{5, 3, 2},
		{-1, 2, -1},
		{-4, 3, -1},
		{-5, 3, -2},
	}
	for _, tt := range tests {
		got, err := RoundDiv(big.NewInt(tt.numerator), big.NewInt(tt.denom))
		if err != nil {
			t.Fatalf("RoundDiv(%d, %d): %v", tt.numerator, tt.denom, err)
		}
		if got.Int64() != tt.want {
			t.Fatalf("RoundDiv(%d, %d) = %d, want %d", tt.numerator, tt.denom, got.Int64(), tt.want)
		}
	}
}

func TestComposeMultipliers(t *testing.T) {
	got, err := ComposeMultipliers(1_500_000, 800_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1_200_000 {
		t.Fatalf("got %d, want 1200000", got)
	}
}

func TestBaseCostToPointMicros(t *testing.T) {
	got, err := BaseCostToPointMicros(2_000_000_000, 1_000_000, 1_250_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2_500_000 {
		t.Fatalf("got %d, want 2500000", got)
	}
}

func TestBaseCostToPointMicrosRoundsUpFinalPoints(t *testing.T) {
	got, err := BaseCostToPointMicros(1, 1_000_000, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("got %d, want the minimum non-zero micro-point charge", got)
	}
}
