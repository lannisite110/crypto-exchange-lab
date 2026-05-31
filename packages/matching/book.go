package matching

import (
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Side mirrors exchange order side.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Order is an in-memory resting or incoming order.
type Order struct {
	ID        string
	UserID    string
	Side      Side
	Price     decimal.Decimal // zero for market
	Quantity  decimal.Decimal
	Remaining decimal.Decimal
	CreatedAt time.Time
}

// Trade is a match result.
type Trade struct {
	Symbol       string
	BuyOrderID   string
	SellOrderID  string
	BuyerUserID  string
	SellerUserID string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	Timestamp    time.Time
}

// PriceLevel is aggregated depth for the UI.
type PriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// Book is a price-time priority order book for one symbol.
type Book struct {
	Symbol string
	mu     sync.Mutex
	bids   []*Order // sorted best (highest) first
	asks   []*Order // sorted best (lowest) first
	trades []Trade
}

// NewBook creates an empty order book.
func NewBook(symbol string) *Book {
	return &Book{Symbol: symbol}
}

// Match accepts an order and returns fills plus the resting remainder (if any).
func (b *Book) Match(in *Order) ([]Trade, *Order) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if in.Remaining.IsZero() {
		in.Remaining = in.Quantity
	}
	var trades []Trade

	if in.Side == SideBuy {
		trades, in = b.matchBuy(in)
	} else {
		trades, in = b.matchSell(in)
	}

	if len(trades) > 0 {
		b.trades = append(b.trades, trades...)
		if len(b.trades) > 500 {
			b.trades = b.trades[len(b.trades)-500:]
		}
	}

	if in != nil && in.Remaining.GreaterThan(decimal.Zero) && !isMarket(in) {
		b.insertResting(in)
		return trades, in
	}
	return trades, nil
}

func (b *Book) matchBuy(buy *Order) ([]Trade, *Order) {
	var trades []Trade
	for buy.Remaining.GreaterThan(decimal.Zero) && len(b.asks) > 0 {
		best := b.asks[0]
		if !isMarket(buy) && buy.Price.LessThan(best.Price) {
			break
		}
		fill, fr := b.fillAgainst(buy, best)
		trades = append(trades, fill)
		if fr.buyer == nil {
			buy = nil
			break
		}
		buy = fr.buyer
		if fr.askRemaining == nil {
			b.asks = b.asks[1:]
		} else {
			b.asks[0] = fr.askRemaining
		}
	}
	if buy.Remaining.IsZero() {
		return trades, nil
	}
	return trades, buy
}

func (b *Book) matchSell(sell *Order) ([]Trade, *Order) {
	var trades []Trade
	for sell.Remaining.GreaterThan(decimal.Zero) && len(b.bids) > 0 {
		best := b.bids[0]
		if !isMarket(sell) && sell.Price.GreaterThan(best.Price) {
			break
		}
		fill, fr := b.fillAgainst(best, sell)
		trades = append(trades, fill)
		if fr.seller == nil {
			sell = nil
			break
		}
		sell = fr.seller
		if fr.bidRemaining == nil {
			b.bids = b.bids[1:]
		} else {
			b.bids[0] = fr.bidRemaining
		}
	}
	if sell.Remaining.IsZero() {
		return trades, nil
	}
	return trades, sell
}

type fillResult struct {
	buyer, seller              *Order
	askRemaining, bidRemaining *Order
}

func (b *Book) fillAgainst(buyer, seller *Order) (Trade, fillResult) {
	qty := decimal.Min(buyer.Remaining, seller.Remaining)
	price := seller.Price
	if isMarket(seller) {
		price = buyer.Price
	}
	if price.IsZero() {
		price = seller.Price
	}
	if price.IsZero() {
		price = buyer.Price
	}

	buyer.Remaining = buyer.Remaining.Sub(qty)
	seller.Remaining = seller.Remaining.Sub(qty)

	t := Trade{
		Symbol:       b.Symbol,
		BuyOrderID:   buyer.ID,
		SellOrderID:  seller.ID,
		BuyerUserID:  buyer.UserID,
		SellerUserID: seller.UserID,
		Price:        price,
		Quantity:     qty,
		Timestamp:    time.Now().UTC(),
	}
	var bidRem, askRem *Order
	if buyer.Remaining.IsZero() {
		bidRem = nil
	} else {
		bidRem = buyer
	}
	if seller.Remaining.IsZero() {
		askRem = nil
	} else {
		askRem = seller
	}
	return t, fillResult{buyer: buyer, seller: seller, bidRemaining: bidRem, askRemaining: askRem}
}

func (b *Book) insertResting(o *Order) {
	if o.Side == SideBuy {
		b.bids = append(b.bids, o)
		sort.Slice(b.bids, func(i, j int) bool {
			if b.bids[i].Price.Equal(b.bids[j].Price) {
				return b.bids[i].CreatedAt.Before(b.bids[j].CreatedAt)
			}
			return b.bids[i].Price.GreaterThan(b.bids[j].Price)
		})
		return
	}
	b.asks = append(b.asks, o)
	sort.Slice(b.asks, func(i, j int) bool {
		if b.asks[i].Price.Equal(b.asks[j].Price) {
			return b.asks[i].CreatedAt.Before(b.asks[j].CreatedAt)
		}
		return b.asks[i].Price.LessThan(b.asks[j].Price)
	})
}

// Cancel removes a resting order by id; returns the cancelled order if found.
func (b *Book) Cancel(orderID string) *Order {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, o := range b.bids {
		if o.ID == orderID {
			b.bids = append(b.bids[:i], b.bids[i+1:]...)
			return o
		}
	}
	for i, o := range b.asks {
		if o.ID == orderID {
			b.asks = append(b.asks[:i], b.asks[i+1:]...)
			return o
		}
	}
	return nil
}

// Depth returns aggregated bid/ask levels.
func (b *Book) Depth(levels int) (bids, asks []PriceLevel) {
	b.mu.Lock()
	defer b.mu.Unlock()

	bids = aggregateSide(b.bids, levels, true)
	asks = aggregateSide(b.asks, levels, false)
	return bids, asks
}

func aggregateSide(orders []*Order, levels int, isBid bool) []PriceLevel {
	byPrice := map[string]decimal.Decimal{}
	var prices []decimal.Decimal
	for _, o := range orders {
		if o.Remaining.IsZero() {
			continue
		}
		key := o.Price.String()
		if _, ok := byPrice[key]; !ok {
			prices = append(prices, o.Price)
		}
		byPrice[key] = byPrice[key].Add(o.Remaining)
	}
	sort.Slice(prices, func(i, j int) bool {
		if isBid {
			return prices[i].GreaterThan(prices[j])
		}
		return prices[i].LessThan(prices[j])
	})
	if len(prices) > levels {
		prices = prices[:levels]
	}
	out := make([]PriceLevel, 0, len(prices))
	for _, p := range prices {
		out = append(out, PriceLevel{
			Price:    p.String(),
			Quantity: byPrice[p.String()].String(),
		})
	}
	return out
}

// RecentTrades returns the latest trades for the symbol.
func (b *Book) RecentTrades(limit int) []Trade {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 || len(b.trades) == 0 {
		return nil
	}
	start := len(b.trades) - limit
	if start < 0 {
		start = 0
	}
	out := make([]Trade, len(b.trades)-start)
	copy(out, b.trades[start:])
	return out
}

func isMarket(o *Order) bool {
	return o.Price.IsZero()
}
