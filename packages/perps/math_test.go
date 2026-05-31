package perps

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestUnrealizedPnLLong(t *testing.T) {
	pnl := UnrealizedPnL("LONG", decimal.NewFromFloat(1), decimal.NewFromInt(100000), decimal.NewFromInt(101000))
	if !pnl.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("got %s", pnl)
	}
}

func TestMarginRatio(t *testing.T) {
	r := MarginRatio(decimal.NewFromInt(1100), decimal.NewFromInt(1000))
	if r.LessThanOrEqual(decimal.NewFromInt(1)) {
		t.Fatalf("expected > 1, got %s", r)
	}
}
