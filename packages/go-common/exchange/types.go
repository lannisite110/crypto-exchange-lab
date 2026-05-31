package exchange

// Side is the order direction.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType is limit or market.
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

// OrderStatus tracks order lifecycle.
type OrderStatus string

const (
	StatusNew              OrderStatus = "NEW"
	StatusPartiallyFilled  OrderStatus = "PARTIALLY_FILLED"
	StatusFilled           OrderStatus = "FILLED"
	StatusCancelled        OrderStatus = "CANCELLED"
)

// Supported spot symbols for Phase 1.
const (
	SymbolBTCUSDT = "BTC/USDT"
	SymbolETHUSDT = "ETH/USDT"
)
