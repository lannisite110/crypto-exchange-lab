package seed

import (
	"fmt"
	"os"
	"time"

	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/matching"
	"github.com/crypto-exchange-lab/matching-engine/internal/engine"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Enabled returns true unless MATCHING_SEED_BOOKS=false.
func Enabled() bool {
	v := os.Getenv("MATCHING_SEED_BOOKS")
	return v == "" || v == "1" || v == "true"
}

// Liquidity places demo bid/ask quotes on each venue book (in-memory only).
func Liquidity(hub *engine.Hub) {
	spreadBps := decimal.NewFromInt(20) // 0.20% each side
	for _, sym := range hub.Symbols() {
		midStr := exchange.ReferenceMid(sym)
		if midStr == "" {
			continue
		}
		mid, err := money.Parse(midStr)
		if err != nil || mid.LessThanOrEqual(decimal.Zero) {
			continue
		}
		bidPx := mid.Mul(decimal.NewFromInt(1).Sub(spreadBps.Div(decimal.NewFromInt(10000))))
		askPx := mid.Mul(decimal.NewFromInt(1).Add(spreadBps.Div(decimal.NewFromInt(10000))))
		qty := defaultSeedQty(sym)

		for _, venue := range hub.Venues() {
			book, ok := hub.Book(venue, sym)
			if !ok {
				continue
			}
			placeResting(book, matching.SideBuy, bidPx, qty)
			placeResting(book, matching.SideSell, askPx, qty)
		}
	}
}

func placeResting(book *matching.Book, side matching.Side, price, qty decimal.Decimal) {
	o := &matching.Order{
		ID:        uuid.New().String(),
		UserID:    "seed-mm",
		Side:      side,
		Price:     price,
		Quantity:  qty,
		Remaining: qty,
		CreatedAt: time.Now().UTC(),
	}
	book.Match(o)
}

func defaultSeedQty(symbol string) decimal.Decimal {
	switch symbol {
	case exchange.SymbolBTCUSDT:
		return decimal.NewFromFloat(0.5)
	case exchange.SymbolETHUSDT:
		return decimal.NewFromInt(5)
	case "SOL/USDT", "BNB/USDT":
		return decimal.NewFromInt(50)
	case "XRP/USDT", "ADA/USDT":
		return decimal.NewFromInt(5000)
	case "DOGE/USDT":
		return decimal.NewFromInt(50000)
	case "LINK/USDT", "AVAX/USDT", "DOT/USDT":
		return decimal.NewFromInt(200)
	default:
		return decimal.NewFromInt(1)
	}
}

// LogSummary prints seeded markets count (for startup logs).
func LogSummary(hub *engine.Hub) string {
	return fmt.Sprintf("%d symbols × %d venues", len(hub.Symbols()), len(hub.Venues()))
}
