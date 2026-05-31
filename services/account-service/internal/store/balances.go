package store

import (
	"context"
	"fmt"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// FreezeRelease describes frozen collateral consumed on trade settlement.
type FreezeRelease struct {
	UserID string
	Asset  string
	Amount decimal.Decimal
}

// GetBalances returns all asset balances for a user.
func (s *Store) GetBalances(ctx context.Context, userID string) ([]Balance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.symbol, b.available, b.frozen
		FROM balances b
		JOIN assets a ON a.id = b.asset_id
		WHERE b.user_id = $1
		ORDER BY a.symbol`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Balance
	for rows.Next() {
		var bal Balance
		if err := rows.Scan(&bal.Asset, &bal.Available, &bal.Frozen); err != nil {
			return nil, err
		}
		out = append(out, bal)
	}
	return out, rows.Err()
}

// Freeze moves amount from available to frozen.
func (s *Store) Freeze(ctx context.Context, userID, asset string, amount decimal.Decimal, refType, refID string) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return apperrors.New(apperrors.CodeInvalidArgument, "freeze amount must be positive")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE balances b
		SET available = available - $3, frozen = frozen + $3
		FROM assets a
		WHERE b.user_id = $1 AND b.asset_id = a.id AND a.symbol = $2
		  AND b.available >= $3`,
		userID, asset, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.CodeInsufficient, "insufficient available balance")
	}

	if err := insertFreezeLedger(ctx, tx, userID, asset, amount, refType, refID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Unfreeze moves amount from frozen back to available.
func (s *Store) Unfreeze(ctx context.Context, userID, asset string, amount decimal.Decimal, refType, refID string) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return apperrors.New(apperrors.CodeInvalidArgument, "unfreeze amount must be positive")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE balances b
		SET available = available + $3, frozen = frozen - $3
		FROM assets a
		WHERE b.user_id = $1 AND b.asset_id = a.id AND a.symbol = $2
		  AND b.frozen >= $3`,
		userID, asset, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.CodeInsufficient, "insufficient frozen balance")
	}

	if err := insertUnfreezeLedger(ctx, tx, userID, asset, amount, refType, refID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SettleTrade posts balanced ledger legs, updates available balances, and releases frozen collateral.
func (s *Store) SettleTrade(ctx context.Context, refID string, legs []LedgerLeg, release []FreezeRelease) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM ledger_transactions WHERE ref_type = 'trade' AND ref_id = $1)`, refID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return tx.Commit(ctx)
	}

	var txnID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_transactions (ref_type, ref_id)
		VALUES ('trade', $1) RETURNING id::text`, refID).Scan(&txnID)
	if err != nil {
		return err
	}

	for _, leg := range legs {
		if err := applyTradeLeg(ctx, tx, txnID, leg); err != nil {
			return err
		}
	}

	for _, fr := range release {
		tag, err := tx.Exec(ctx, `
			UPDATE balances b
			SET frozen = frozen - $3
			FROM assets a
			WHERE b.user_id = $1 AND b.asset_id = a.id AND a.symbol = $2
			  AND b.frozen >= $3`,
			fr.UserID, fr.Asset, fr.Amount)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.New(apperrors.CodeInsufficient, "insufficient frozen for settlement")
		}
	}

	if err := verifyLedgerBalance(ctx, tx, txnID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func applyTradeLeg(ctx context.Context, tx pgx.Tx, txnID string, leg LedgerLeg) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO ledger_entries (transaction_id, user_id, asset_id, amount)
		SELECT $1, $2, a.id, $3 FROM assets a WHERE a.symbol = $4`,
		txnID, leg.UserID, leg.Amount, leg.Asset)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE balances b
		SET available = available + $3
		FROM assets a
		WHERE b.user_id = $1 AND b.asset_id = a.id AND a.symbol = $2`,
		leg.UserID, leg.Asset, leg.Amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("balance row missing for user %s asset %s", leg.UserID, leg.Asset)
	}
	return nil
}

func verifyLedgerBalance(ctx context.Context, tx pgx.Tx, txnID string) error {
	rows, err := tx.Query(ctx, `
		SELECT a.symbol, COALESCE(SUM(e.amount), 0)
		FROM ledger_entries e
		JOIN assets a ON a.id = e.asset_id
		WHERE e.transaction_id = $1
		GROUP BY a.symbol`, txnID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sym string
		var sum decimal.Decimal
		if err := rows.Scan(&sym, &sum); err != nil {
			return err
		}
		if !sum.IsZero() {
			return fmt.Errorf("ledger imbalance for %s: sum=%s", sym, sum.String())
		}
	}
	return rows.Err()
}
