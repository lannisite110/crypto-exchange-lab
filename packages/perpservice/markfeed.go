package perpservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/crypto-exchange-lab/perpstore"
	"github.com/shopspring/decimal"
)

// MarkFeed updates mark prices from CEX mid quotes.
type MarkFeed struct {
	Store        *perpstore.Store
	MatchingURL  string
	HTTPClient   *http.Client
}

// Run starts the mark price updater loop.
func (f *MarkFeed) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		f.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (f *MarkFeed) tick(ctx context.Context) {
	markets, err := f.Store.ListMarkets(ctx)
	if err != nil {
		return
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	for _, m := range markets {
		mid, err := fetchCEXMid(ctx, client, f.MatchingURL, m.SpotSymbol)
		if err != nil || mid.LessThanOrEqual(decimal.Zero) {
			continue
		}
		_ = f.Store.SetMarkPrice(ctx, m.Symbol, mid)
	}
}

func fetchCEXMid(ctx context.Context, client *http.Client, base, spotSymbol string) (decimal.Decimal, error) {
	path := fmt.Sprintf("%s/api/v1/markets/%s/depth?venue=CEX", base, url.PathEscape(spotSymbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return decimal.Zero, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK   bool `json:"ok"`
		Data struct {
			Bids []struct {
				Price string `json:"price"`
			} `json:"bids"`
			Asks []struct {
				Price string `json:"price"`
			} `json:"asks"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || !parsed.OK {
		return decimal.Zero, fmt.Errorf("depth fetch failed")
	}
	if len(parsed.Data.Bids) == 0 || len(parsed.Data.Asks) == 0 {
		return decimal.Zero, fmt.Errorf("empty book")
	}
	bid, _ := decimal.NewFromString(parsed.Data.Bids[0].Price)
	ask, _ := decimal.NewFromString(parsed.Data.Asks[0].Price)
	return bid.Add(ask).Div(decimal.NewFromInt(2)), nil
}
