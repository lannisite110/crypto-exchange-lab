package store

import "github.com/shopspring/decimal"

// User is a simulated exchange account holder.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Balance is available + frozen for one asset.
type Balance struct {
	Asset     string          `json:"asset"`
	Available decimal.Decimal `json:"available"`
	Frozen    decimal.Decimal `json:"frozen"`
}

// LedgerLeg is one side of a balanced transaction.
type LedgerLeg struct {
	UserID string
	Asset  string
	Amount decimal.Decimal // signed: + credit, - debit
}
