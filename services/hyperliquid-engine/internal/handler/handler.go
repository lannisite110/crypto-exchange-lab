package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/perpservice"
	"github.com/crypto-exchange-lab/perpstore"
)

// Handler serves hyperliquid HTTP APIs.
type Handler struct {
	Engine *perpservice.Engine
	Store  *perpstore.Store
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/markets", h.markets)
	mux.HandleFunc("GET /api/v1/mark-prices", h.markPrices)
	mux.HandleFunc("GET /api/v1/positions", h.listPositions)
	mux.HandleFunc("POST /api/v1/positions/open", h.open)
	mux.HandleFunc("POST /api/v1/positions/close", h.close)
	mux.HandleFunc("POST /api/v1/internal/liquidate", h.liquidate)
}

func (h *Handler) markets(w http.ResponseWriter, r *http.Request) {
	m, err := h.Store.ListMarkets(r.Context())
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
		return
	}
	out := make([]map[string]any, len(m))
	for i, mk := range m {
		out[i] = map[string]any{
			"symbol": mk.Symbol, "spot_symbol": mk.SpotSymbol,
			"max_leverage": mk.MaxLeverage, "maint_margin_rate": mk.MaintMarginRate.String(),
		}
	}
	httputil.OK(w, out)
}

func (h *Handler) markPrices(w http.ResponseWriter, r *http.Request) {
	marks, err := h.Store.ListMarkPrices(r.Context())
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
		return
	}
	out := make(map[string]string, len(marks))
	for sym, p := range marks {
		out[sym] = money.Format(p)
	}
	httputil.OK(w, out)
}

func (h *Handler) listPositions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "user_id required"))
		return
	}
	positions, err := h.Store.ListPositions(r.Context(), userID)
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
		return
	}
	risks, _ := h.Engine.ListRisk(r.Context(), userID)
	riskBySym := map[string]perpservice.RiskSnapshot{}
	for _, rsk := range risks {
		riskBySym[rsk.Symbol] = rsk
	}

	out := make([]map[string]any, len(positions))
	for i, p := range positions {
		row := map[string]any{
			"id": p.ID, "user_id": p.UserID, "symbol": p.Symbol, "side": p.Side,
			"size": money.Format(p.Size), "entry_price": money.Format(p.EntryPrice),
			"leverage": p.Leverage, "margin": money.Format(p.Margin),
		}
		if rsk, ok := riskBySym[p.Symbol]; ok {
			row["mark_price"] = rsk.MarkPrice
			row["unrealized_pnl"] = rsk.UnrealizedPnL
			row["margin_ratio"] = rsk.MarginRatio
			row["liquidation_risk"] = rsk.LiquidationRisk
		}
		out[i] = row
	}
	httputil.OK(w, out)
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID   string `json:"user_id"`
		Symbol   string `json:"symbol"`
		Side     string `json:"side"`
		Size     string `json:"size"`
		Leverage int    `json:"leverage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return
	}
	pos, err := h.Engine.Open(r.Context(), perpservice.OpenRequest{
		UserID: body.UserID, Symbol: body.Symbol,
		Side: exchange.PositionSide(body.Side), Size: body.Size, Leverage: body.Leverage,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, formatPos(pos))
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Symbol string `json:"symbol"`
		Size   string `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return
	}
	pos, pnl, err := h.Engine.Close(r.Context(), perpservice.CloseRequest{
		UserID: body.UserID, Symbol: body.Symbol, Size: body.Size,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]any{"position": formatPos(pos), "realized_pnl": money.Format(pnl)})
}

func (h *Handler) liquidate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Symbol string `json:"symbol"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return
	}
	if err := h.Engine.Liquidate(r.Context(), body.UserID, body.Symbol); err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]string{"status": "liquidated"})
}

func formatPos(p *perpstore.Position) map[string]any {
	if p == nil {
		return nil
	}
	return map[string]any{
		"id": p.ID, "symbol": p.Symbol, "side": p.Side,
		"size": money.Format(p.Size), "entry_price": money.Format(p.EntryPrice),
		"leverage": p.Leverage, "margin": money.Format(p.Margin),
	}
}

func writeErr(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apperrors.AppError); ok {
		httputil.Fail(w, ae)
		return
	}
	httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
}
