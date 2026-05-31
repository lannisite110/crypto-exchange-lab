package tradeclients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/httputil"
)

// MatchingClient calls matching-engine APIs.
type MatchingClient struct {
	base   string
	client *http.Client
}

// NewMatchingClient creates a matching engine HTTP client.
func NewMatchingClient(base string) *MatchingClient {
	return &MatchingClient{
		base:   base,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// MatchTrade is a fill returned by the matcher.
type MatchTrade struct {
	ID           string
	Symbol       string
	BuyOrderID   string
	SellOrderID  string
	BuyerUserID  string
	SellerUserID string
	Price        string
	Quantity     string
}

type matchTradeJSON struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	BuyOrderID   string `json:"buy_order_id"`
	SellOrderID  string `json:"sell_order_id"`
	BuyerUserID  string `json:"buyer_user_id"`
	SellerUserID string `json:"seller_user_id"`
	Price        string `json:"price"`
	Quantity     string `json:"quantity"`
}

// SubmitOrderRequest is sent to the matcher.
type SubmitOrderRequest struct {
	Venue    exchange.Venue
	OrderID  string
	UserID   string
	Symbol   string
	Side     string
	Type     string
	Price    string
	Quantity string
}

// MatchResult contains fills and optional resting order.
type MatchResult struct {
	Trades  []MatchTrade
	Resting bool
}

// SubmitOrder sends an order to the matcher.
func (c *MatchingClient) SubmitOrder(ctx context.Context, req SubmitOrderRequest) (*MatchResult, error) {
	body := map[string]string{
		"venue": string(req.Venue), "order_id": req.OrderID, "user_id": req.UserID,
		"symbol": req.Symbol, "side": req.Side, "type": req.Type,
		"price": req.Price, "quantity": req.Quantity,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/orders", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK    bool `json:"ok"`
		Data  struct {
			Trades  []matchTradeJSON `json:"trades"`
			Resting *struct{}        `json:"resting"`
		} `json:"data"`
		Error *httputil.ErrorBody `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		if parsed.Error != nil {
			return nil, apperrors.New(parsed.Error.Code, parsed.Error.Message)
		}
		return nil, fmt.Errorf("matching error")
	}

	out := &MatchResult{Resting: parsed.Data.Resting != nil}
	for _, t := range parsed.Data.Trades {
		out.Trades = append(out.Trades, MatchTrade{
			ID: t.ID, Symbol: t.Symbol, BuyOrderID: t.BuyOrderID, SellOrderID: t.SellOrderID,
			BuyerUserID: t.BuyerUserID, SellerUserID: t.SellerUserID, Price: t.Price, Quantity: t.Quantity,
		})
	}
	return out, nil
}

// CancelOrder removes a resting order from the book.
func (c *MatchingClient) CancelOrder(ctx context.Context, venue exchange.Venue, orderID, symbol string) error {
	q := url.Values{"symbol": {symbol}, "venue": {string(venue)}}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.base+"/api/v1/orders/"+orderID+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env httputil.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if !env.OK && env.Error != nil && env.Error.Code != apperrors.CodeNotFound {
		return apperrors.New(env.Error.Code, env.Error.Message)
	}
	return nil
}
