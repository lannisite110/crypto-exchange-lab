package engine

import (
	"sync"

	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/matching"
)

// Hub holds per-venue, per-symbol order books.
type Hub struct {
	mu      sync.RWMutex
	symbols []string
	books   map[string]*matching.Book
}

func bookKey(venue exchange.Venue, symbol string) string {
	return string(venue) + ":" + symbol
}

// NewHub creates isolated CEX and DEX books for each symbol.
func NewHub(symbols []string) *Hub {
	if len(symbols) == 0 {
		symbols = exchange.DefaultSpotSymbols
	}
	h := &Hub{
		symbols: append([]string(nil), symbols...),
		books:   make(map[string]*matching.Book),
	}
	venues := []exchange.Venue{exchange.VenueCEX, exchange.VenueDEX}
	for _, v := range venues {
		for _, sym := range symbols {
			h.books[bookKey(v, sym)] = matching.NewBook(sym)
		}
	}
	return h
}

// Symbols returns configured market names.
func (h *Hub) Symbols() []string {
	out := make([]string, len(h.symbols))
	copy(out, h.symbols)
	return out
}

// Book returns the book for a venue and symbol.
func (h *Hub) Book(venue exchange.Venue, symbol string) (*matching.Book, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	b, ok := h.books[bookKey(venue, symbol)]
	return b, ok
}

// Venues lists configured venues.
func (h *Hub) Venues() []exchange.Venue {
	return []exchange.Venue{exchange.VenueCEX, exchange.VenueDEX}
}
