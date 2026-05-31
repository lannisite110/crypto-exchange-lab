package exchange

// Venue separates CEX and OrderBook DEX liquidity pools (shared matcher, isolated books).
type Venue string

const (
	VenueCEX Venue = "CEX"
	VenueDEX Venue = "DEX"
)

// ValidVenue reports whether v is a known venue.
func ValidVenue(v Venue) bool {
	return v == VenueCEX || v == VenueDEX
}
