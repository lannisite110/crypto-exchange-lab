package perpservice

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/jackc/pgx/v5"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/perps"
	"github.com/crypto-exchange-lab/perpstore"
	"github.com/crypto-exchange-lab/tradeclients"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const refTypePerpMargin = "perp_margin"

// Engine manages perpetual positions.
type Engine struct {
	Store   *perpstore.Store
	Account *tradeclients.AccountClient
}

// OpenRequest opens or increases a position.
type OpenRequest struct {
	UserID   string
	Symbol   string
	Side     exchange.PositionSide
	Size     string
	Leverage int
}

// CloseRequest closes part or all of a position.
type CloseRequest struct {
	UserID string
	Symbol string
	Size   string // empty = full close
}

// RiskSnapshot is margin health for one position.
type RiskSnapshot struct {
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`
	Size            string `json:"size"`
	EntryPrice      string `json:"entry_price"`
	MarkPrice       string `json:"mark_price"`
	Leverage        int    `json:"leverage"`
	Margin          string `json:"margin"`
	UnrealizedPnL   string `json:"unrealized_pnl"`
	Equity          string `json:"equity"`
	Maintenance     string `json:"maintenance_margin"`
	MarginRatio     string `json:"margin_ratio"`
	LiquidationRisk bool   `json:"liquidation_risk"`
}

// Open increases or opens a position at current mark.
func (e *Engine) Open(ctx context.Context, req OpenRequest) (*perpstore.Position, error) {
	mkt, err := e.Store.GetMarket(ctx, req.Symbol)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "unknown market")
	}
	if req.Leverage < 1 || req.Leverage > mkt.MaxLeverage {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, fmt.Sprintf("leverage must be 1-%d", mkt.MaxLeverage))
	}

	size, err := money.Parse(req.Size)
	if err != nil || size.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "invalid size")
	}

	mark, err := e.Store.GetMarkPrice(ctx, req.Symbol)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, "mark price not ready; wait for matching feed")
		}
		return nil, err
	}

	addMargin := perps.InitialMargin(perps.Notional(size, mark), req.Leverage)

	var existing *perpstore.Position
	if p, err := e.Store.GetPosition(ctx, req.UserID, req.Symbol); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	} else {
		if p.Side != req.Side {
			return nil, apperrors.New(apperrors.CodeConflict, "close existing position before flipping side")
		}
		existing = p
	}

	posID := uuid.New().String()
	if existing != nil {
		posID = existing.ID
	}
	// Unique per freeze; ledger (ref_type, ref_id) must not reuse position id on add.
	freezeRefID := uuid.New().String()

	if err := e.Account.Freeze(ctx, refTypePerpMargin, req.UserID, mkt.QuoteAsset, money.Format(addMargin), freezeRefID); err != nil {
		return nil, err
	}

	var pos perpstore.Position
	if existing == nil {
		pos = perpstore.Position{
			ID: posID, UserID: req.UserID, Symbol: req.Symbol, Side: req.Side,
			Size: size, EntryPrice: mark, Leverage: req.Leverage, Margin: addMargin,
		}
	} else {
		newSize := existing.Size.Add(size)
		newEntry := perps.WeightedEntryPrice(existing.Size, existing.EntryPrice, size, mark)
		pos = perpstore.Position{
			ID: existing.ID, UserID: req.UserID, Symbol: req.Symbol, Side: existing.Side,
			Size: newSize, EntryPrice: newEntry, Leverage: req.Leverage,
			Margin: existing.Margin.Add(addMargin),
		}
	}

	if err := e.Store.UpsertPosition(ctx, pos); err != nil {
		_ = e.Account.Unfreeze(ctx, refTypePerpMargin, req.UserID, mkt.QuoteAsset, money.Format(addMargin), freezeRefID)
		return nil, err
	}

	lev := req.Leverage
	_ = e.Store.InsertEvent(ctx, req.UserID, req.Symbol, "OPEN", &size, &mark, nil, &lev, posID)
	return &pos, nil
}

// Close reduces or closes a position at mark.
func (e *Engine) Close(ctx context.Context, req CloseRequest) (*perpstore.Position, decimal.Decimal, error) {
	pos, err := e.Store.GetPosition(ctx, req.UserID, req.Symbol)
	if err != nil {
		return nil, decimal.Zero, apperrors.New(apperrors.CodeNotFound, "no position")
	}

	mark, err := e.Store.GetMarkPrice(ctx, req.Symbol)
	if err != nil {
		return nil, decimal.Zero, err
	}

	closeSize := pos.Size
	if req.Size != "" {
		closeSize, err = money.Parse(req.Size)
		if err != nil || closeSize.LessThanOrEqual(decimal.Zero) {
			return nil, decimal.Zero, apperrors.New(apperrors.CodeInvalidArgument, "invalid size")
		}
		if closeSize.GreaterThan(pos.Size) {
			closeSize = pos.Size
		}
	}

	return e.closeSize(ctx, pos, closeSize, mark, "CLOSE")
}

// Liquidate force-closes a position (liquidation engine).
func (e *Engine) Liquidate(ctx context.Context, userID, symbol string) error {
	pos, err := e.Store.GetPosition(ctx, userID, symbol)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "no position")
	}
	mark, err := e.Store.GetMarkPrice(ctx, symbol)
	if err != nil {
		return err
	}
	_, _, err = e.closeSize(ctx, pos, pos.Size, mark, "LIQUIDATION")
	return err
}

func (e *Engine) closeSize(ctx context.Context, pos *perpstore.Position, closeSize, mark decimal.Decimal, eventType string) (*perpstore.Position, decimal.Decimal, error) {
	mkt, err := e.Store.GetMarket(ctx, pos.Symbol)
	if err != nil {
		return nil, decimal.Zero, err
	}

	pnl := perps.RealizedPnL(string(pos.Side), closeSize, pos.EntryPrice, mark)
	marginRelease := pos.Margin.Mul(closeSize).Div(pos.Size)

	if err := e.Account.Unfreeze(ctx, refTypePerpMargin, pos.UserID, mkt.QuoteAsset, money.Format(marginRelease), pos.ID); err != nil {
		return nil, decimal.Zero, err
	}

	houseID, err := e.Store.GetHouseUserID(ctx)
	if err != nil {
		return nil, decimal.Zero, err
	}
	tradeID := uuid.New().String()
	if !pnl.IsZero() {
		if err := e.Account.SettleBalanced(ctx, tradeID, pos.UserID, houseID, money.Format(pnl)); err != nil {
			return nil, decimal.Zero, err
		}
	}

	remaining := pos.Size.Sub(closeSize)
	var out *perpstore.Position

	if remaining.LessThanOrEqual(decimal.Zero) {
		if err := e.Store.DeletePosition(ctx, pos.UserID, pos.Symbol); err != nil {
			return nil, pnl, err
		}
	} else {
		np := *pos
		np.Size = remaining
		np.Margin = pos.Margin.Sub(marginRelease)
		if err := e.Store.UpsertPosition(ctx, np); err != nil {
			return nil, pnl, err
		}
		out = &np
	}

	_ = e.Store.InsertEvent(ctx, pos.UserID, pos.Symbol, eventType, &closeSize, &mark, &pnl, nil, tradeID)
	return out, pnl, nil
}

// RiskForPosition computes margin health.
func (e *Engine) RiskForPosition(ctx context.Context, userID, symbol string) (*RiskSnapshot, error) {
	pos, err := e.Store.GetPosition(ctx, userID, symbol)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "no position")
	}
	mkt, err := e.Store.GetMarket(ctx, symbol)
	if err != nil {
		return nil, err
	}
	mark, err := e.Store.GetMarkPrice(ctx, symbol)
	if err != nil {
		return nil, err
	}

	notional := perps.Notional(pos.Size, mark)
	upnl := perps.UnrealizedPnL(string(pos.Side), pos.Size, pos.EntryPrice, mark)
	equity := perps.Equity(pos.Margin, upnl)
	maint := perps.MaintenanceMargin(notional, mkt.MaintMarginRate)
	ratio := perps.MarginRatio(equity, maint)

	return &RiskSnapshot{
		Symbol: symbol, Side: string(pos.Side),
		Size: money.Format(pos.Size), EntryPrice: money.Format(pos.EntryPrice),
		MarkPrice: money.Format(mark), Leverage: pos.Leverage,
		Margin: money.Format(pos.Margin), UnrealizedPnL: money.Format(upnl),
		Equity: money.Format(equity), Maintenance: money.Format(maint),
		MarginRatio: money.Format(ratio), LiquidationRisk: ratio.LessThan(decimal.NewFromInt(1)),
	}, nil
}

// ListRisk returns risk for all user positions.
func (e *Engine) ListRisk(ctx context.Context, userID string) ([]RiskSnapshot, error) {
	positions, err := e.Store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	var out []RiskSnapshot
	for _, p := range positions {
		r, err := e.RiskForPosition(ctx, userID, p.Symbol)
		if err != nil {
			continue
		}
		out = append(out, *r)
	}
	return out, nil
}

// ApplyFunding settles funding for one position.
func (e *Engine) ApplyFunding(ctx context.Context, pos *perpstore.Position, rate decimal.Decimal) error {
	mark, err := e.Store.GetMarkPrice(ctx, pos.Symbol)
	if err != nil {
		return err
	}
	payment := perps.FundingPayment(string(pos.Side), pos.Size, mark, rate)
	if payment.IsZero() {
		return nil
	}
	houseID, err := e.Store.GetHouseUserID(ctx)
	if err != nil {
		return err
	}
	tradeID := uuid.New().String()
	if err := e.Account.SettleBalanced(ctx, tradeID, pos.UserID, houseID, money.Format(payment)); err != nil {
		return err
	}
	return e.Store.InsertFundingPayment(ctx, pos.UserID, pos.Symbol, rate, payment, mark, pos.Size)
}
