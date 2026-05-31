package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/matching"
	"github.com/crypto-exchange-lab/matching-engine/internal/engine"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsClient struct {
	conn   *websocket.Conn
	venue  exchange.Venue
	symbol string
}

// Handler exposes matching HTTP and WebSocket APIs.
type Handler struct {
	hub     *engine.Hub
	mu      sync.RWMutex
	clients []wsClient
}

// New creates a matching handler.
func New(hub *engine.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/venues", h.listVenues)
	mux.HandleFunc("POST /api/v1/orders", h.submitOrder)
	mux.HandleFunc("DELETE /api/v1/orders/{id}", h.cancelOrder)
	mux.HandleFunc("GET /api/v1/markets/{symbol}/depth", h.depth)
	mux.HandleFunc("GET /api/v1/markets/{symbol}/trades", h.trades)
	mux.HandleFunc("GET /ws/v1/market", h.marketWS)
}

func (h *Handler) listVenues(w http.ResponseWriter, _ *http.Request) {
	httputil.OK(w, map[string]any{
		"venues": h.hub.Venues(),
		"symbols": h.hub.Symbols(),
	})
}

type submitReq struct {
	Venue    string `json:"venue"`
	OrderID  string `json:"order_id"`
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

func parseVenue(r *http.Request, bodyVenue string) (exchange.Venue, error) {
	v := bodyVenue
	if v == "" {
		v = r.URL.Query().Get("venue")
	}
	if v == "" {
		v = string(exchange.VenueCEX)
	}
	venue := exchange.Venue(v)
	if !exchange.ValidVenue(venue) {
		return "", apperrors.New(apperrors.CodeInvalidArgument, "invalid venue")
	}
	return venue, nil
}

func (h *Handler) submitOrder(w http.ResponseWriter, r *http.Request) {
	var req submitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return
	}
	venue, err := parseVenue(r, req.Venue)
	if err != nil {
		httputil.Fail(w, err.(*apperrors.AppError))
		return
	}
	book, ok := h.hub.Book(venue, req.Symbol)
	if !ok {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "unknown symbol"))
		return
	}

	qty, err := money.Parse(req.Quantity)
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
		return
	}

	var price decimal.Decimal
	if req.Type == "MARKET" {
		price = decimal.Zero
	} else {
		price, err = money.Parse(req.Price)
		if err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
			return
		}
	}

	o := &matching.Order{
		ID: req.OrderID, UserID: req.UserID, Side: matching.Side(req.Side),
		Price: price, Quantity: qty, Remaining: qty, CreatedAt: time.Now().UTC(),
	}

	trades, resting := book.Match(o)
	h.broadcast(venue, req.Symbol)

	type tradeResp struct {
		ID           string `json:"id"`
		Symbol       string `json:"symbol"`
		BuyOrderID   string `json:"buy_order_id"`
		SellOrderID  string `json:"sell_order_id"`
		BuyerUserID  string `json:"buyer_user_id"`
		SellerUserID string `json:"seller_user_id"`
		Price        string `json:"price"`
		Quantity     string `json:"quantity"`
	}
	out := make([]tradeResp, len(trades))
	for i, t := range trades {
		out[i] = tradeResp{
			ID: t.BuyOrderID + "-" + t.SellOrderID + "-" + t.Timestamp.Format(time.RFC3339Nano),
			Symbol: t.Symbol, BuyOrderID: t.BuyOrderID, SellOrderID: t.SellOrderID,
			BuyerUserID: t.BuyerUserID, SellerUserID: t.SellerUserID,
			Price: money.Format(t.Price), Quantity: money.Format(t.Quantity),
		}
	}

	var restingResp *map[string]string
	if resting != nil {
		restingResp = &map[string]string{
			"order_id": resting.ID, "remaining": money.Format(resting.Remaining),
		}
	}

	httputil.OK(w, map[string]any{"venue": venue, "trades": out, "resting": restingResp})
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "symbol query required"))
		return
	}
	venue, err := parseVenue(r, "")
	if err != nil {
		httputil.Fail(w, err.(*apperrors.AppError))
		return
	}
	book, ok := h.hub.Book(venue, symbol)
	if !ok {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "unknown symbol"))
		return
	}
	cancelled := book.Cancel(r.PathValue("id"))
	if cancelled == nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeNotFound, "order not on book"))
		return
	}
	h.broadcast(venue, symbol)
	httputil.OK(w, map[string]string{"order_id": cancelled.ID, "remaining": money.Format(cancelled.Remaining)})
}

func (h *Handler) depth(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	venue, err := parseVenue(r, "")
	if err != nil {
		httputil.Fail(w, err.(*apperrors.AppError))
		return
	}
	book, ok := h.hub.Book(venue, symbol)
	if !ok {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "unknown symbol"))
		return
	}
	bids, asks := book.Depth(15)
	httputil.OK(w, map[string]any{"venue": venue, "symbol": symbol, "bids": bids, "asks": asks})
}

func (h *Handler) trades(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	venue, err := parseVenue(r, "")
	if err != nil {
		httputil.Fail(w, err.(*apperrors.AppError))
		return
	}
	book, ok := h.hub.Book(venue, symbol)
	if !ok {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "unknown symbol"))
		return
	}
	recent := book.RecentTrades(50)
	type row struct {
		Price     string `json:"price"`
		Quantity  string `json:"quantity"`
		Timestamp string `json:"timestamp"`
	}
	out := make([]row, len(recent))
	for i, t := range recent {
		out[i] = row{
			Price: money.Format(t.Price), Quantity: money.Format(t.Quantity),
			Timestamp: t.Timestamp.Format(time.RFC3339),
		}
	}
	httputil.OK(w, map[string]any{"venue": venue, "symbol": symbol, "trades": out})
}

func (h *Handler) marketWS(w http.ResponseWriter, r *http.Request) {
	venue, err := parseVenue(r, "")
	if err != nil {
		httputil.Fail(w, err.(*apperrors.AppError))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		symbol = exchange.SymbolBTCUSDT
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := wsClient{conn: conn, venue: venue, symbol: symbol}
	h.mu.Lock()
	h.clients = append(h.clients, client)
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		for i, c := range h.clients {
			if c.conn == conn {
				h.clients = append(h.clients[:i], h.clients[i+1:]...)
				break
			}
		}
		h.mu.Unlock()
		conn.Close()
	}()

	h.sendSnapshot(client)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Handler) sendSnapshot(c wsClient) {
	book, ok := h.hub.Book(c.venue, c.symbol)
	if !ok {
		return
	}
	bids, asks := book.Depth(15)
	payload, _ := json.Marshal(map[string]any{
		"type": "snapshot", "venue": c.venue, "symbol": c.symbol,
		"bids": bids, "asks": asks, "trades": book.RecentTrades(20),
	})
	_ = c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (h *Handler) broadcast(venue exchange.Venue, symbol string) {
	book, ok := h.hub.Book(venue, symbol)
	if !ok {
		return
	}
	bids, asks := book.Depth(15)
	payload, _ := json.Marshal(map[string]any{
		"type": "snapshot", "venue": venue, "symbol": symbol,
		"bids": bids, "asks": asks, "trades": book.RecentTrades(20),
	})

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.venue == venue && c.symbol == symbol {
			_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		}
	}
}
