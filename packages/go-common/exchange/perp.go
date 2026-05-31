package exchange

// Position side for perpetual futures.
type PositionSide string

const (
	PositionLong  PositionSide = "LONG"
	PositionShort PositionSide = "SHORT"
)

// Perp symbols (USDT-margined).
const (
	SymbolBTCPerp = "BTC-PERP"
	SymbolETHPerp = "ETH-PERP"
)

// PerpMarkets lists Phase 3 perpetual markets.
var PerpMarkets = []string{SymbolBTCPerp, SymbolETHPerp}
