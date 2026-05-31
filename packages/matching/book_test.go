package matching

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMatchLimitBuyHitsAsk(t *testing.T) {
	b := NewBook("BTC/USDT")

	sell := &Order{
		ID: "s1", UserID: "bob", Side: SideSell,
		Price: decimal.NewFromInt(100000), Quantity: decimal.NewFromFloat(1),
		Remaining: decimal.NewFromFloat(1),
	}
	b.Match(sell)

	buy := &Order{
		ID: "b1", UserID: "alice", Side: SideBuy,
		Price: decimal.NewFromInt(100000), Quantity: decimal.NewFromFloat(0.5),
		Remaining: decimal.NewFromFloat(0.5),
	}
	trades, rest := b.Match(buy)
	if rest != nil {
		t.Fatalf("expected full fill, rest=%v", rest)
	}
	if len(trades) != 1 {
		t.Fatalf("trades=%d", len(trades))
	}
	if !trades[0].Quantity.Equal(decimal.NewFromFloat(0.5)) {
		t.Fatalf("qty %s", trades[0].Quantity)
	}
}

func TestPriceTimePriority(t *testing.T) {
	b := NewBook("BTC/USDT")

	b.Match(&Order{
		ID: "s2", UserID: "bob", Side: SideSell,
		Price: decimal.NewFromInt(101000), Quantity: decimal.NewFromInt(1),
		Remaining: decimal.NewFromInt(1),
	})
	b.Match(&Order{
		ID: "s1", UserID: "bob", Side: SideSell,
		Price: decimal.NewFromInt(100000), Quantity: decimal.NewFromInt(1),
		Remaining: decimal.NewFromInt(1),
	})

	trades, _ := b.Match(&Order{
		ID: "b1", UserID: "alice", Side: SideBuy,
		Price: decimal.NewFromInt(101000), Quantity: decimal.NewFromInt(1),
		Remaining: decimal.NewFromInt(1),
	})
	if trades[0].Price.IntPart() != 100000 {
		t.Fatalf("expected best ask 100000, got %s", trades[0].Price)
	}
}
