package orderflow

import (
	"context"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/money"
	"github.com/crypto-exchange-lab/orderstore"
	"github.com/crypto-exchange-lab/tradeclients"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Engine orchestrates freeze, match, and settlement for one venue.
type Engine struct {
	Venue     exchange.Venue
	RefType   string // ledger ref_type for freeze/unfreeze
	Store     *orderstore.Store
	Account   *tradeclients.AccountClient
	Matching  *tradeclients.MatchingClient
}

// PlaceOrderRequest is the public place-order payload.
type PlaceOrderRequest struct {
	UserID   string
	Symbol   string
	Side     exchange.Side
	Type     exchange.OrderType
	Price    string
	Quantity string
}

// PlaceOrder runs the full order flow.
func (e *Engine) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*orderstore.Order, []orderstore.Trade, error) {
	sym, err := e.Store.GetSymbol(ctx, req.Symbol)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.CodeInvalidArgument, "unknown symbol")
	}

	qty, err := money.Parse(req.Quantity)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.CodeInvalidArgument, err.Error())
	}

	var price decimal.Decimal
	if req.Type == exchange.OrderTypeMarket {
		price = decimal.Zero
	} else {
		price, err = money.Parse(req.Price)
		if err != nil || price.LessThanOrEqual(decimal.Zero) {
			return nil, nil, apperrors.New(apperrors.CodeInvalidArgument, "invalid price")
		}
	}

	freezeAsset, freezeAmt, err := collateralForOrder(req.Side, sym, qty, price)
	if err != nil {
		return nil, nil, err
	}

	orderID := uuid.New().String()
	if err := e.Account.Freeze(ctx, e.RefType, req.UserID, freezeAsset, money.Format(freezeAmt), orderID); err != nil {
		return nil, nil, err
	}

	var pricePtr *decimal.Decimal
	if req.Type != exchange.OrderTypeMarket {
		pricePtr = &price
	}

	o := orderstore.Order{
		ID: orderID, UserID: req.UserID, Venue: e.Venue, Symbol: sym.Name,
		Side: req.Side, Type: req.Type, Status: exchange.StatusNew,
		Price: pricePtr, Quantity: qty, FilledQty: decimal.Zero,
	}
	if err := e.Store.CreateOrder(ctx, o, sym.ID); err != nil {
		_ = e.Account.Unfreeze(ctx, e.RefType, req.UserID, freezeAsset, money.Format(freezeAmt), orderID)
		return nil, nil, err
	}

	priceStr := "0"
	if pricePtr != nil {
		priceStr = money.Format(*pricePtr)
	}

	matchRes, err := e.Matching.SubmitOrder(ctx, tradeclients.SubmitOrderRequest{
		Venue: e.Venue, OrderID: orderID, UserID: req.UserID, Symbol: sym.Name,
		Side: string(req.Side), Type: string(req.Type), Price: priceStr, Quantity: money.Format(qty),
	})
	if err != nil {
		_ = e.Account.Unfreeze(ctx, e.RefType, req.UserID, freezeAsset, money.Format(freezeAmt), orderID)
		return nil, nil, err
	}

	var recorded []orderstore.Trade
	filled := decimal.Zero

	for _, mt := range matchRes.Trades {
		p, _ := money.Parse(mt.Price)
		q, _ := money.Parse(mt.Quantity)
		tradeID := uuid.New().String()

		if err := e.settleMatch(ctx, tradeID, sym, mt); err != nil {
			return nil, nil, err
		}

		dbTrade := orderstore.Trade{
			Venue: e.Venue, Symbol: sym.Name, BuyOrderID: mt.BuyOrderID, SellOrderID: mt.SellOrderID,
			BuyerUserID: mt.BuyerUserID, SellerUserID: mt.SellerUserID, Price: p, Quantity: q,
		}
		id, err := e.Store.InsertTrade(ctx, e.Venue, sym.ID, dbTrade)
		if err != nil {
			return nil, nil, err
		}
		dbTrade.ID = id
		recorded = append(recorded, dbTrade)

		if mt.BuyOrderID == orderID || mt.SellOrderID == orderID {
			filled = filled.Add(q)
		}

		for _, oid := range []string{mt.BuyOrderID, mt.SellOrderID} {
			if oid == orderID {
				continue
			}
			if co, err := e.Store.GetOrder(ctx, oid); err == nil {
				newFilled := co.FilledQty.Add(q)
				st := exchange.StatusPartiallyFilled
				if newFilled.GreaterThanOrEqual(co.Quantity) {
					st = exchange.StatusFilled
				}
				_ = e.Store.UpdateOrderFill(ctx, oid, newFilled, st)
			}
		}
	}

	status := exchange.StatusNew
	if filled.GreaterThan(decimal.Zero) {
		status = exchange.StatusPartiallyFilled
	}
	if filled.GreaterThanOrEqual(qty) {
		status = exchange.StatusFilled
	}
	if !matchRes.Resting && filled.LessThan(qty) && filled.IsZero() {
		status = exchange.StatusCancelled
	}
	_ = e.Store.UpdateOrderFill(ctx, orderID, filled, status)

	if !matchRes.Resting {
		if err := e.releaseRemainingFreeze(ctx, req.Side, sym, orderID, qty, price, filled); err != nil {
			return nil, nil, err
		}
	}

	updated, err := e.Store.GetOrder(ctx, orderID)
	return updated, recorded, err
}

// CancelOrder cancels a resting order and unfreezes collateral.
func (e *Engine) CancelOrder(ctx context.Context, orderID string) (*orderstore.Order, error) {
	o, err := e.Store.GetOrder(ctx, orderID)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "order not found")
	}
	if o.Venue != e.Venue {
		return nil, apperrors.New(apperrors.CodeNotFound, "order not found")
	}
	if o.Status == exchange.StatusFilled || o.Status == exchange.StatusCancelled {
		return nil, apperrors.New(apperrors.CodeConflict, "order not cancellable")
	}

	_ = e.Matching.CancelOrder(ctx, e.Venue, orderID, o.Symbol)

	sym, _ := e.Store.GetSymbol(ctx, o.Symbol)
	remaining := o.Quantity.Sub(o.FilledQty)
	price := decimal.Zero
	if o.Price != nil {
		price = *o.Price
	}
	asset, amt, _ := collateralForOrder(o.Side, sym, remaining, price)
	if amt.GreaterThan(decimal.Zero) {
		_ = e.Account.Unfreeze(ctx, e.RefType, o.UserID, asset, money.Format(amt), orderID)
	}

	return e.Store.CancelOrder(ctx, orderID)
}

func (e *Engine) settleMatch(ctx context.Context, tradeID string, sym *orderstore.Symbol, mt tradeclients.MatchTrade) error {
	p, err := money.Parse(mt.Price)
	if err != nil {
		return err
	}
	q, err := money.Parse(mt.Quantity)
	if err != nil {
		return err
	}
	notional := p.Mul(q)

	return e.Account.SettleTrade(ctx, tradeclients.SettleTradeRequest{
		TradeID: tradeID,
		Legs: []tradeclients.LedgerLeg{
			{UserID: mt.BuyerUserID, Asset: sym.QuoteAsset, Amount: money.Format(notional.Neg())},
			{UserID: mt.BuyerUserID, Asset: sym.BaseAsset, Amount: money.Format(q)},
			{UserID: mt.SellerUserID, Asset: sym.BaseAsset, Amount: money.Format(q.Neg())},
			{UserID: mt.SellerUserID, Asset: sym.QuoteAsset, Amount: money.Format(notional)},
		},
		FreezeRelease: []tradeclients.FreezeRelease{
			{UserID: mt.BuyerUserID, Asset: sym.QuoteAsset, Amount: money.Format(notional)},
			{UserID: mt.SellerUserID, Asset: sym.BaseAsset, Amount: money.Format(q)},
		},
	})
}

func (e *Engine) releaseRemainingFreeze(
	ctx context.Context,
	side exchange.Side,
	sym *orderstore.Symbol,
	orderID string,
	qty, price, filled decimal.Decimal,
) error {
	remaining := qty.Sub(filled)
	if remaining.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	asset, amt, err := collateralForOrder(side, sym, remaining, price)
	if err != nil || amt.IsZero() {
		return err
	}
	o, err := e.Store.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	return e.Account.Unfreeze(ctx, e.RefType, o.UserID, asset, money.Format(amt), orderID)
}

func collateralForOrder(side exchange.Side, sym *orderstore.Symbol, qty, price decimal.Decimal) (asset string, amount decimal.Decimal, err error) {
	if side == exchange.SideBuy {
		if price.IsZero() {
			return "", decimal.Zero, apperrors.New(apperrors.CodeInvalidArgument, "market buy not supported in Phase 2 — use limit")
		}
		return sym.QuoteAsset, qty.Mul(price), nil
	}
	return sym.BaseAsset, qty, nil
}
