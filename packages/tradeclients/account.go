package tradeclients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/httputil"
)

// AccountClient calls account-service internal APIs.
type AccountClient struct {
	base   string
	client *http.Client
}

// NewAccountClient creates a client with the given base URL.
func NewAccountClient(base string) *AccountClient {
	return &AccountClient{
		base:   base,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// LedgerLeg is one entry in a trade settlement.
type LedgerLeg struct {
	UserID string `json:"user_id"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}

// FreezeRelease consumes frozen collateral on fill.
type FreezeRelease struct {
	UserID string `json:"user_id"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}

// SettleTradeRequest posts balanced ledger legs.
type SettleTradeRequest struct {
	TradeID       string          `json:"trade_id"`
	Legs          []LedgerLeg     `json:"legs"`
	FreezeRelease []FreezeRelease `json:"freeze_release"`
}

// Freeze locks collateral for an order.
func (c *AccountClient) Freeze(ctx context.Context, refType, userID, asset, amount, refID string) error {
	return c.post(ctx, "/api/v1/internal/freeze", map[string]string{
		"user_id": userID, "asset": asset, "amount": amount, "ref_type": refType, "ref_id": refID,
	})
}

// Unfreeze releases remaining collateral.
func (c *AccountClient) Unfreeze(ctx context.Context, refType, userID, asset, amount, refID string) error {
	return c.post(ctx, "/api/v1/internal/unfreeze", map[string]string{
		"user_id": userID, "asset": asset, "amount": amount, "ref_type": refType, "ref_id": refID,
	})
}

// SettleTrade posts trade ledger entries.
func (c *AccountClient) SettleTrade(ctx context.Context, req SettleTradeRequest) error {
	return c.post(ctx, "/api/v1/internal/settle-trade", req)
}

// SettleBalanced posts a zero-sum USDT transfer between two users (perp PnL / funding).
func (c *AccountClient) SettleBalanced(ctx context.Context, tradeID, userID, houseID, amount string) error {
	return c.SettleTrade(ctx, SettleTradeRequest{
		TradeID: tradeID,
		Legs: []LedgerLeg{
			{UserID: userID, Asset: "USDT", Amount: amount},
			{UserID: houseID, Asset: "USDT", Amount: negateAmount(amount)},
		},
	})
}

func negateAmount(s string) string {
	if len(s) > 0 && s[0] == '-' {
		return s[1:]
	}
	return "-" + s
}

func (c *AccountClient) post(ctx context.Context, path string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var env httputil.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		if env.Error != nil {
			return apperrors.New(env.Error.Code, env.Error.Message)
		}
		return fmt.Errorf("account error: status %d", resp.StatusCode)
	}
	return nil
}
