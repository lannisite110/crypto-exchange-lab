package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/account-service/internal/store"
)

// Handler serves account HTTP APIs.
type Handler struct {
	store *store.Store
}

// New creates an account HTTP handler.
func New(st *store.Store) *Handler {
	return &Handler{store: st}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/users", h.listUsers)
	mux.HandleFunc("POST /api/v1/users", h.createUser)
	mux.HandleFunc("GET /api/v1/users/{id}/balances", h.getBalances)
	mux.HandleFunc("POST /api/v1/internal/freeze", h.freeze)
	mux.HandleFunc("POST /api/v1/internal/unfreeze", h.unfreeze)
	mux.HandleFunc("POST /api/v1/internal/settle-trade", h.settleTrade)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, users)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "username required"))
		return
	}
	u, err := h.store.CreateUser(r.Context(), body.Username)
	if err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, u)
}

func (h *Handler) getBalances(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	bals, err := h.store.GetBalances(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	type row struct {
		Asset     string `json:"asset"`
		Available string `json:"available"`
		Frozen    string `json:"frozen"`
	}
	out := make([]row, len(bals))
	for i, b := range bals {
		out[i] = row{
			Asset:     b.Asset,
			Available: money.Format(b.Available),
			Frozen:    money.Format(b.Frozen),
		}
	}
	httputil.OK(w, map[string]any{"user_id": id, "balances": out})
}

func (h *Handler) freeze(w http.ResponseWriter, r *http.Request) {
	var body collateralReq
	if !decode(w, r, &body) {
		return
	}
	amt, err := money.Parse(body.Amount)
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
		return
	}
	if err := h.store.Freeze(r.Context(), body.UserID, body.Asset, amt, body.RefType, body.RefID); err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]string{"status": "frozen"})
}

func (h *Handler) unfreeze(w http.ResponseWriter, r *http.Request) {
	var body collateralReq
	if !decode(w, r, &body) {
		return
	}
	amt, err := money.Parse(body.Amount)
	if err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
		return
	}
	if err := h.store.Unfreeze(r.Context(), body.UserID, body.Asset, amt, body.RefType, body.RefID); err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]string{"status": "unfrozen"})
}

func (h *Handler) settleTrade(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TradeID       string `json:"trade_id"`
		Legs          []legReq `json:"legs"`
		FreezeRelease []collateralReq `json:"freeze_release"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.TradeID == "" || len(body.Legs) == 0 {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "trade_id and legs required"))
		return
	}

	legs := make([]store.LedgerLeg, len(body.Legs))
	for i, l := range body.Legs {
		amt, err := money.Parse(l.Amount)
		if err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
			return
		}
		legs[i] = store.LedgerLeg{UserID: l.UserID, Asset: l.Asset, Amount: amt}
	}

	release := make([]store.FreezeRelease, len(body.FreezeRelease))
	for i, fr := range body.FreezeRelease {
		amt, err := money.Parse(fr.Amount)
		if err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, err.Error()))
			return
		}
		release[i] = store.FreezeRelease{UserID: fr.UserID, Asset: fr.Asset, Amount: amt}
	}

	if err := h.store.SettleTrade(r.Context(), body.TradeID, legs, release); err != nil {
		writeErr(w, err)
		return
	}
	httputil.OK(w, map[string]string{"status": "settled"})
}

type collateralReq struct {
	UserID  string `json:"user_id"`
	Asset   string `json:"asset"`
	Amount  string `json:"amount"`
	RefType string `json:"ref_type"`
	RefID   string `json:"ref_id"`
}

type legReq struct {
	UserID string `json:"user_id"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		httputil.Fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid json"))
		return false
	}
	return true
}

func writeErr(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apperrors.AppError); ok {
		httputil.Fail(w, ae)
		return
	}
	httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
}
