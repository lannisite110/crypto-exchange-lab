package exchange

// DefaultSpotSymbols are USDT-margined demo markets (Phase 7 — path A).
var DefaultSpotSymbols = []string{
	SymbolBTCUSDT,
	SymbolETHUSDT,
	"SOL/USDT",
	"BNB/USDT",
	"XRP/USDT",
	"DOGE/USDT",
	"LINK/USDT",
	"AVAX/USDT",
	"ADA/USDT",
	"DOT/USDT",
}

// SymbolSpec holds display and matching precision hints.
type SymbolSpec struct {
	TickSize string
	LotSize  string
	RefMid   string // demo reference mid price in USDT
}

// SpotSymbolSpecs maps symbol name to metadata.
var SpotSymbolSpecs = map[string]SymbolSpec{
	SymbolBTCUSDT:  {TickSize: "0.01", LotSize: "0.0001", RefMid: "100000"},
	SymbolETHUSDT:  {TickSize: "0.01", LotSize: "0.001", RefMid: "3500"},
	"SOL/USDT":     {TickSize: "0.01", LotSize: "0.01", RefMid: "180"},
	"BNB/USDT":     {TickSize: "0.01", LotSize: "0.01", RefMid: "600"},
	"XRP/USDT":     {TickSize: "0.0001", LotSize: "1", RefMid: "2.2"},
	"DOGE/USDT":    {TickSize: "0.00001", LotSize: "10", RefMid: "0.15"},
	"LINK/USDT":    {TickSize: "0.01", LotSize: "0.1", RefMid: "15"},
	"AVAX/USDT":    {TickSize: "0.01", LotSize: "0.1", RefMid: "35"},
	"ADA/USDT":     {TickSize: "0.0001", LotSize: "10", RefMid: "0.55"},
	"DOT/USDT":     {TickSize: "0.01", LotSize: "0.1", RefMid: "7"},
}

// ReferenceMid returns the demo reference price for a symbol, or empty if unknown.
func ReferenceMid(symbol string) string {
	if s, ok := SpotSymbolSpecs[symbol]; ok {
		return s.RefMid
	}
	return ""
}
