package orderapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/orderflow"
	"github.com/crypto-exchange-lab/orderstore"
)

// Handler serves order HTTP APIs for one venue.
type Handler struct {
	Engine  *orderflow.Engine
	Store   *orderstore.Store
	Service string
	Phase   string
}

// Register mounts routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orders", h.placeOrder)
	mux.HandleFunc("DELETE /api/v1/orders/{id}", h.cancelOrder)
	mux.HandleFunc("GET /api/v1/orders", h.listOrders)
	mux.HandleFunc("GET /api/v1/markets/{symbol}/trades", h.listTrades)
	mux.HandleFunc("GET /api/v1/symbols", h.listSymbols)
	mux.HandleFunc("GET /api/v1/venue", h.venueInfo)
}

func (h *Handler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID   string `json:"user_id"`
		Symbol   string `json:"symbol"`
		Side     string `json:"side"`
		Type     string `json:"type"`
		Price    string `json:"price"`
		Quantity string `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return
	}

	o, trades, err := h.Engine.PlaceOrder(r.Context(), orderflow.PlaceOrderRequest{
		UserID: body.UserID, Symbol: body.Symbol,
		Side: exchange.Side(body.Side), Type: exchange.OrderType(body.Type),
		Price: body.Price, Quantity: body.Quantity,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]any{"order": formatOrder(o), "trades": formatTrades(trades)})
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	o, err := h.Engine.CancelOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, formatOrder(o))
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "user_id required"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	orders, err := h.Store.ListOrders(r.Context(), h.Engine.Venue, userID, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]any, len(orders))
	for i := range orders {
		out[i] = formatOrder(&orders[i])
	}
	httputil.OK(w, out)
}

func (h *Handler) listTrades(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	trades, err := h.Store.ListTrades(r.Context(), h.Engine.Venue, symbol, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, formatTrades(trades))
}

func (h *Handler) listSymbols(w http.ResponseWriter, r *http.Request) {
	syms, err := h.Store.ListSymbols(r.Context())
	if err != nil || len(syms) == 0 {
		httputil.OK(w, exchange.DefaultSpotSymbols)
		return
	}
	httputil.OK(w, syms)
}

func (h *Handler) venueInfo(w http.ResponseWriter, _ *http.Request) {
	httputil.OK(w, map[string]any{
		"venue":   h.Engine.Venue,
		"service": h.Service,
		"matcher": "shared price-time priority engine",
		"phase":   h.Phase,
	})
}

func formatOrder(o *orderstore.Order) map[string]any {
	if o == nil {
		return nil
	}
	m := map[string]any{
		"id": o.ID, "user_id": o.UserID, "venue": o.Venue, "symbol": o.Symbol,
		"side": string(o.Side), "type": string(o.Type), "status": string(o.Status),
		"quantity": money.Format(o.Quantity), "filled_qty": money.Format(o.FilledQty),
		"created_at": o.CreatedAt,
	}
	if o.Price != nil {
		m["price"] = money.Format(*o.Price)
	}
	return m
}

func formatTrades(trades []orderstore.Trade) []map[string]any {
	out := make([]map[string]any, len(trades))
	for i, t := range trades {
		out[i] = map[string]any{
			"id": t.ID, "venue": t.Venue, "symbol": t.Symbol,
			"buy_order_id": t.BuyOrderID, "sell_order_id": t.SellOrderID,
			"price": money.Format(t.Price), "quantity": money.Format(t.Quantity),
			"created_at": t.CreatedAt,
		}
	}
	return out
}

func writeErr(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apperrors.AppError); ok {
		httputil.Fail(w, ae)
		return
	}
	httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
}
