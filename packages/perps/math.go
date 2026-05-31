package perps

import "github.com/shopspring/decimal"

// Notional returns position notional at mark price.
func Notional(size, markPrice decimal.Decimal) decimal.Decimal {
	return size.Mul(markPrice)
}

// InitialMargin returns required margin for notional at leverage.
func InitialMargin(notional decimal.Decimal, leverage int) decimal.Decimal {
	if leverage < 1 {
		leverage = 1
	}
	return notional.Div(decimal.NewFromInt(int64(leverage)))
}

// MaintenanceMargin returns maintenance requirement.
func MaintenanceMargin(notional, maintRate decimal.Decimal) decimal.Decimal {
	return notional.Mul(maintRate)
}

// UnrealizedPnL computes PnL at mark for a position.
func UnrealizedPnL(side string, size, entryPrice, markPrice decimal.Decimal) decimal.Decimal {
	diff := markPrice.Sub(entryPrice)
	if side == "SHORT" {
		diff = entryPrice.Sub(markPrice)
	}
	return diff.Mul(size)
}

// Equity is isolated margin plus unrealized PnL.
func Equity(margin, unrealizedPnL decimal.Decimal) decimal.Decimal {
	return margin.Add(unrealizedPnL)
}

// MarginRatio is equity / maintenance margin (>1 is safe).
func MarginRatio(equity, maintMargin decimal.Decimal) decimal.Decimal {
	if maintMargin.IsZero() {
		return decimal.NewFromInt(999)
	}
	return equity.Div(maintMargin)
}

// RealizedPnL for a partial close.
func RealizedPnL(side string, closeSize, entryPrice, exitPrice decimal.Decimal) decimal.Decimal {
	return UnrealizedPnL(side, closeSize, entryPrice, exitPrice)
}

// FundingPayment: positive rate => longs pay shorts.
func FundingPayment(side string, size, markPrice, rate decimal.Decimal) decimal.Decimal {
	notional := Notional(size, markPrice)
	payment := notional.Mul(rate)
	if side == "LONG" {
		return payment.Neg()
	}
	return payment
}

// WeightedEntryPrice blends two entries.
func WeightedEntryPrice(oldSize, oldEntry, addSize, addPrice decimal.Decimal) decimal.Decimal {
	total := oldSize.Add(addSize)
	if total.IsZero() {
		return addPrice
	}
	num := oldSize.Mul(oldEntry).Add(addSize.Mul(addPrice))
	return num.Div(total)
}
