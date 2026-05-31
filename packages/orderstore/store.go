package orderstore

import (
	"context"
	"fmt"
	"time"

	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store persists orders and trades with venue isolation.
type Store struct {
	pool *pgxpool.Pool
}

// New connects to Postgres.
func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Symbol metadata for a trading pair.
type Symbol struct {
	ID         int16
	Name       string
	BaseAsset  string
	QuoteAsset string
}

// Order row.
type Order struct {
	ID        string
	UserID    string
	Venue     exchange.Venue
	Symbol    string
	Side      exchange.Side
	Type      exchange.OrderType
	Status    exchange.OrderStatus
	Price     *decimal.Decimal
	Quantity  decimal.Decimal
	FilledQty decimal.Decimal
	CreatedAt time.Time
}

// Trade row.
type Trade struct {
	ID           string
	Venue        exchange.Venue
	Symbol       string
	BuyOrderID   string
	SellOrderID  string
	BuyerUserID  string
	SellerUserID string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	CreatedAt    time.Time
}

// ListSymbols returns all configured spot symbols.
func (s *Store) ListSymbols(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT name FROM symbols ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// GetSymbol loads symbol metadata by name.
func (s *Store) GetSymbol(ctx context.Context, name string) (*Symbol, error) {
	var sym Symbol
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.name, ba.symbol, qa.symbol
		FROM symbols s
		JOIN assets ba ON ba.id = s.base_asset_id
		JOIN assets qa ON qa.id = s.quote_asset_id
		WHERE s.name = $1`, name).Scan(&sym.ID, &sym.Name, &sym.BaseAsset, &sym.QuoteAsset)
	if err != nil {
		return nil, fmt.Errorf("symbol %s: %w", name, err)
	}
	return &sym, nil
}

// CreateOrder inserts a new order for the given venue.
func (s *Store) CreateOrder(ctx context.Context, o Order, symbolID int16) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO orders (id, user_id, symbol_id, venue, side, type, status, price, quantity, filled_qty)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		o.ID, o.UserID, symbolID, o.Venue, o.Side, o.Type, o.Status, o.Price, o.Quantity, o.FilledQty)
	return err
}

// UpdateOrderFill updates filled quantity and status.
func (s *Store) UpdateOrderFill(ctx context.Context, orderID string, filledQty decimal.Decimal, status exchange.OrderStatus) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET filled_qty = $2, status = $3, updated_at = NOW() WHERE id = $1`,
		orderID, filledQty, status)
	return err
}

// CancelOrder marks order cancelled.
func (s *Store) CancelOrder(ctx context.Context, orderID string) (*Order, error) {
	var o Order
	var price *decimal.Decimal
	var symName string
	err := s.pool.QueryRow(ctx, `
		UPDATE orders o
		SET status = 'CANCELLED', updated_at = NOW()
		FROM symbols s
		WHERE o.id = $1 AND s.id = o.symbol_id AND o.status IN ('NEW', 'PARTIALLY_FILLED')
		RETURNING o.id::text, o.user_id::text, o.venue, s.name, o.side, o.type, o.status,
		          o.price, o.quantity, o.filled_qty, o.created_at`,
		orderID).Scan(&o.ID, &o.UserID, &o.Venue, &symName, &o.Side, &o.Type, &o.Status,
		&price, &o.Quantity, &o.FilledQty, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	o.Symbol = symName
	o.Price = price
	return &o, nil
}

// GetOrder returns an order by id.
func (s *Store) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	var o Order
	var price *decimal.Decimal
	var symName string
	err := s.pool.QueryRow(ctx, `
		SELECT o.id::text, o.user_id::text, o.venue, s.name, o.side, o.type, o.status,
		       o.price, o.quantity, o.filled_qty, o.created_at
		FROM orders o
		JOIN symbols s ON s.id = o.symbol_id
		WHERE o.id = $1`, orderID).Scan(
		&o.ID, &o.UserID, &o.Venue, &symName, &o.Side, &o.Type, &o.Status,
		&price, &o.Quantity, &o.FilledQty, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	o.Symbol = symName
	o.Price = price
	return &o, nil
}

// ListOrders returns recent orders for a user within a venue.
func (s *Store) ListOrders(ctx context.Context, venue exchange.Venue, userID string, limit int) ([]Order, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT o.id::text, o.user_id::text, o.venue, s.name, o.side, o.type, o.status,
		       o.price, o.quantity, o.filled_qty, o.created_at
		FROM orders o
		JOIN symbols s ON s.id = o.symbol_id
		WHERE o.user_id = $1 AND o.venue = $2
		ORDER BY o.created_at DESC
		LIMIT $3`, userID, venue, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Order
	for rows.Next() {
		var o Order
		var price *decimal.Decimal
		if err := rows.Scan(&o.ID, &o.UserID, &o.Venue, &o.Symbol, &o.Side, &o.Type, &o.Status,
			&price, &o.Quantity, &o.FilledQty, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.Price = price
		out = append(out, o)
	}
	return out, rows.Err()
}

// InsertTrade records a trade for a venue.
func (s *Store) InsertTrade(ctx context.Context, venue exchange.Venue, symbolID int16, t Trade) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO trades (symbol_id, venue, buy_order_id, sell_order_id, buyer_user_id, seller_user_id, price, quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text`,
		symbolID, venue, t.BuyOrderID, t.SellOrderID, t.BuyerUserID, t.SellerUserID, t.Price, t.Quantity).Scan(&id)
	return id, err
}

// ListTrades returns recent trades for a symbol within a venue.
func (s *Store) ListTrades(ctx context.Context, venue exchange.Venue, symbol string, limit int) ([]Trade, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT t.id::text, t.venue, s.name, t.buy_order_id::text, t.sell_order_id::text,
		       t.buyer_user_id::text, t.seller_user_id::text, t.price, t.quantity, t.created_at
		FROM trades t
		JOIN symbols s ON s.id = t.symbol_id
		WHERE s.name = $1 AND t.venue = $2
		ORDER BY t.created_at DESC
		LIMIT $3`, symbol, venue, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.ID, &t.Venue, &t.Symbol, &t.BuyOrderID, &t.SellOrderID,
			&t.BuyerUserID, &t.SellerUserID, &t.Price, &t.Quantity, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
