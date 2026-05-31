package perpstore

import (
	"context"
	"fmt"
	"time"

	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Store provides Postgres access for perpetual futures.
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

// Market metadata.
type Market struct {
	Symbol           string
	SpotSymbol       string
	BaseAsset        string
	QuoteAsset       string
	MaxLeverage      int
	MaintMarginRate  decimal.Decimal
	TakerFeeRate     decimal.Decimal
}

// Position row.
type Position struct {
	ID         string
	UserID     string
	Symbol     string
	Side       exchange.PositionSide
	Size       decimal.Decimal
	EntryPrice decimal.Decimal
	Leverage   int
	Margin     decimal.Decimal
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ListMarkets returns configured perp markets.
func (s *Store) ListMarkets(ctx context.Context) ([]Market, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT symbol, spot_symbol, base_asset, quote_asset, max_leverage, maint_margin_rate, taker_fee_rate
		FROM perp_markets ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Market
	for rows.Next() {
		var m Market
		if err := rows.Scan(&m.Symbol, &m.SpotSymbol, &m.BaseAsset, &m.QuoteAsset,
			&m.MaxLeverage, &m.MaintMarginRate, &m.TakerFeeRate); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMarket loads one market.
func (s *Store) GetMarket(ctx context.Context, symbol string) (*Market, error) {
	var m Market
	err := s.pool.QueryRow(ctx, `
		SELECT symbol, spot_symbol, base_asset, quote_asset, max_leverage, maint_margin_rate, taker_fee_rate
		FROM perp_markets WHERE symbol = $1`, symbol).Scan(
		&m.Symbol, &m.SpotSymbol, &m.BaseAsset, &m.QuoteAsset,
		&m.MaxLeverage, &m.MaintMarginRate, &m.TakerFeeRate)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMarkPrice returns latest mark for symbol.
func (s *Store) GetMarkPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	var p decimal.Decimal
	err := s.pool.QueryRow(ctx, `SELECT price FROM mark_prices WHERE symbol = $1`, symbol).Scan(&p)
	return p, err
}

// SetMarkPrice upserts mark price.
func (s *Store) SetMarkPrice(ctx context.Context, symbol string, price decimal.Decimal) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO mark_prices (symbol, price, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (symbol) DO UPDATE SET price = $2, updated_at = NOW()`, symbol, price)
	return err
}

// ListMarkPrices returns all marks.
func (s *Store) ListMarkPrices(ctx context.Context) (map[string]decimal.Decimal, error) {
	rows, err := s.pool.Query(ctx, `SELECT symbol, price FROM mark_prices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]decimal.Decimal)
	for rows.Next() {
		var sym string
		var p decimal.Decimal
		if err := rows.Scan(&sym, &p); err != nil {
			return nil, err
		}
		out[sym] = p
	}
	return out, rows.Err()
}

// GetPosition loads user position for symbol.
func (s *Store) GetPosition(ctx context.Context, userID, symbol string) (*Position, error) {
	var p Position
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, symbol, side, size, entry_price, leverage, margin, created_at, updated_at
		FROM perp_positions WHERE user_id = $1 AND symbol = $2`, userID, symbol).Scan(
		&p.ID, &p.UserID, &p.Symbol, &p.Side, &p.Size, &p.EntryPrice, &p.Leverage, &p.Margin, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPositions lists all open positions.
func (s *Store) ListPositions(ctx context.Context, userID string) ([]Position, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, user_id::text, symbol, side, size, entry_price, leverage, margin, created_at, updated_at
		FROM perp_positions WHERE user_id = $1 ORDER BY symbol`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Position
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.ID, &p.UserID, &p.Symbol, &p.Side, &p.Size, &p.EntryPrice,
			&p.Leverage, &p.Margin, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListAllPositions returns every open position (for liquidation scan).
func (s *Store) ListAllPositions(ctx context.Context) ([]Position, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, user_id::text, symbol, side, size, entry_price, leverage, margin, created_at, updated_at
		FROM perp_positions ORDER BY symbol, user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Position
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.ID, &p.UserID, &p.Symbol, &p.Side, &p.Size, &p.EntryPrice,
			&p.Leverage, &p.Margin, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPosition saves or updates a position.
func (s *Store) UpsertPosition(ctx context.Context, p Position) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO perp_positions (id, user_id, symbol, side, size, entry_price, leverage, margin)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, symbol) DO UPDATE SET
			side = $4, size = $5, entry_price = $6, leverage = $7, margin = $8, updated_at = NOW()`,
		p.ID, p.UserID, p.Symbol, p.Side, p.Size, p.EntryPrice, p.Leverage, p.Margin)
	return err
}

// DeletePosition removes a position row.
func (s *Store) DeletePosition(ctx context.Context, userID, symbol string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM perp_positions WHERE user_id = $1 AND symbol = $2`, userID, symbol)
	return err
}

// InsertEvent logs a perp lifecycle event.
func (s *Store) InsertEvent(ctx context.Context, userID, symbol, eventType string, size, price, pnl *decimal.Decimal, leverage *int, refID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO perp_events (user_id, symbol, event_type, size, price, leverage, pnl, ref_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, symbol, eventType, size, price, leverage, pnl, refID)
	return err
}

// GetHouseUserID returns perp_house user id for PnL balancing.
func (s *Store) GetHouseUserID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM users WHERE username = 'perp_house'`).Scan(&id)
	return id, err
}

// InsertFundingRate records a funding interval rate.
func (s *Store) InsertFundingRate(ctx context.Context, symbol string, rate decimal.Decimal) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO funding_rates (symbol, rate) VALUES ($1, $2)`, symbol, rate)
	return err
}

// LatestFundingRate returns most recent rate for symbol.
func (s *Store) LatestFundingRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	var r decimal.Decimal
	err := s.pool.QueryRow(ctx, `
		SELECT rate FROM funding_rates WHERE symbol = $1 ORDER BY interval_start DESC LIMIT 1`, symbol).Scan(&r)
	return r, err
}

// InsertFundingPayment logs a user funding payment.
func (s *Store) InsertFundingPayment(ctx context.Context, userID, symbol string, rate, payment, mark, size decimal.Decimal) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO funding_payments (user_id, symbol, rate, payment, mark_price, position_size)
		VALUES ($1, $2, $3, $4, $5, $6)`, userID, symbol, rate, payment, mark, size)
	return err
}
