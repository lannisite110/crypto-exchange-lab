package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func insertFreezeLedger(ctx context.Context, tx pgx.Tx, userID, asset string, amount decimal.Decimal, refType, refID string) error {
	return insertCollateralLedger(ctx, tx, userID, asset, amount.Neg(), amount, refType, refID, "freeze")
}

func insertUnfreezeLedger(ctx context.Context, tx pgx.Tx, userID, asset string, amount decimal.Decimal, refType, refID string) error {
	return insertCollateralLedger(ctx, tx, userID, asset, amount, amount.Neg(), refType, refID, "unfreeze")
}

func insertCollateralLedger(ctx context.Context, tx pgx.Tx, userID, asset string, availDelta, frozenDelta decimal.Decimal, refType, refID, suffix string) error {
	var txnID string
	fullRef := refType + ":" + refID + ":" + suffix
	err := tx.QueryRow(ctx, `
		INSERT INTO ledger_transactions (ref_type, ref_id)
		VALUES ($1, $2)
		RETURNING id::text`, refType, fullRef).Scan(&txnID)
	if err != nil {
		return err
	}

	for _, leg := range []struct {
		amount decimal.Decimal
		label  string
	}{
		{availDelta, asset + "_available"},
		{frozenDelta, asset + "_frozen"},
	} {
		if leg.amount.IsZero() {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO ledger_entries (transaction_id, user_id, asset_id, amount)
			SELECT $1, $2, id, $3 FROM assets WHERE symbol = $4`,
			txnID, userID, leg.amount, asset)
		if err != nil {
			return err
		}
	}
	return nil
}
