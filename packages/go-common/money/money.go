package money

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Zero is a reusable zero decimal for ledger and balance math.
var Zero = decimal.Zero

// Parse parses a decimal string used in API payloads (never use float64 for amounts).
func Parse(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Zero, fmt.Errorf("invalid decimal %q: %w", s, err)
	}
	return d, nil
}

// MustParse parses s or panics — for tests only.
func MustParse(s string) decimal.Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Format returns a plain string suitable for JSON API responses.
func Format(d decimal.Decimal) string {
	return d.String()
}
