package chainstore

import (
	"context"
	"encoding/json"
)

// InsertBlock stores a block (idempotent).
func (s *Store) InsertBlock(ctx context.Context, b Block) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO indexed_blocks (chain_id, number, hash, parent_hash, timestamp, tx_count)
		VALUES ($1, $2, LOWER($3), LOWER($4), $5, $6)
		ON CONFLICT (chain_id, number) DO NOTHING`,
		b.ChainID, b.Number, b.Hash, b.ParentHash, b.Timestamp, b.TxCount)
	return err
}

// InsertTransaction stores a tx (idempotent).
func (s *Store) InsertTransaction(ctx context.Context, tx Transaction) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO indexed_transactions
			(chain_id, hash, block_number, tx_index, from_addr, to_addr, value_wei, gas_used, status)
		VALUES ($1, LOWER($2), $3, $4, LOWER($5), LOWER(NULLIF($6,'')), $7, $8, $9)
		ON CONFLICT (chain_id, hash) DO NOTHING`,
		tx.ChainID, tx.Hash, tx.BlockNumber, tx.TxIndex, tx.FromAddr, tx.ToAddr,
		tx.ValueWei, tx.GasUsed, tx.Status)
	return err
}

// InsertEvent stores a decoded log (idempotent).
func (s *Store) InsertEvent(ctx context.Context, e Event, topic0, rawData string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO indexed_events
			(chain_id, block_number, tx_hash, log_index, contract_address, event_type, topic0, raw_data, payload)
		VALUES ($1, $2, LOWER($3), $4, LOWER($5), $6, $7, $8, $9)
		ON CONFLICT (chain_id, tx_hash, log_index) DO NOTHING`,
		e.ChainID, e.BlockNumber, e.TxHash, e.LogIndex, e.ContractAddress,
		e.EventType, topic0, rawData, e.Payload)
	return err
}

// ListBlocks returns recent blocks.
func (s *Store) ListBlocks(ctx context.Context, chainID string, limit int) ([]Block, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT chain_id, number, hash, parent_hash, timestamp, tx_count
		FROM indexed_blocks WHERE chain_id = $1
		ORDER BY number DESC LIMIT $2`, chainID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlocks(rows)
}

// GetBlock loads one block by number.
func (s *Store) GetBlock(ctx context.Context, chainID string, number int64) (*Block, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT chain_id, number, hash, parent_hash, timestamp, tx_count
		FROM indexed_blocks WHERE chain_id = $1 AND number = $2`, chainID, number)
	var b Block
	if err := row.Scan(&b.ChainID, &b.Number, &b.Hash, &b.ParentHash, &b.Timestamp, &b.TxCount); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetTransaction loads a tx by hash.
func (s *Store) GetTransaction(ctx context.Context, chainID, hash string) (*Transaction, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT chain_id, hash, block_number, tx_index, from_addr, to_addr, value_wei, gas_used, status
		FROM indexed_transactions WHERE chain_id = $1 AND hash = LOWER($2)`, chainID, hash)
	var tx Transaction
	var to *string
	if err := row.Scan(&tx.ChainID, &tx.Hash, &tx.BlockNumber, &tx.TxIndex,
		&tx.FromAddr, &to, &tx.ValueWei, &tx.GasUsed, &tx.Status); err != nil {
		return nil, err
	}
	if to != nil {
		tx.ToAddr = *to
	}
	return &tx, nil
}

// ListTransactionsByAddress returns recent txs involving an address.
func (s *Store) ListTransactionsByAddress(ctx context.Context, chainID, addr string, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT chain_id, hash, block_number, tx_index, from_addr, to_addr, value_wei, gas_used, status
		FROM indexed_transactions
		WHERE chain_id = $1 AND (from_addr = LOWER($2) OR to_addr = LOWER($2))
		ORDER BY block_number DESC, tx_index DESC LIMIT $3`, chainID, addr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var tx Transaction
		var to *string
		if err := rows.Scan(&tx.ChainID, &tx.Hash, &tx.BlockNumber, &tx.TxIndex,
			&tx.FromAddr, &to, &tx.ValueWei, &tx.GasUsed, &tx.Status); err != nil {
			return nil, err
		}
		if to != nil {
			tx.ToAddr = *to
		}
		out = append(out, tx)
	}
	return out, rows.Err()
}

// ListBlockTransactions returns txs in a block.
func (s *Store) ListBlockTransactions(ctx context.Context, chainID string, number int64) ([]Transaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT chain_id, hash, block_number, tx_index, from_addr, to_addr, value_wei, gas_used, status
		FROM indexed_transactions
		WHERE chain_id = $1 AND block_number = $2
		ORDER BY tx_index`, chainID, number)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var tx Transaction
		var to *string
		if err := rows.Scan(&tx.ChainID, &tx.Hash, &tx.BlockNumber, &tx.TxIndex,
			&tx.FromAddr, &to, &tx.ValueWei, &tx.GasUsed, &tx.Status); err != nil {
			return nil, err
		}
		if to != nil {
			tx.ToAddr = *to
		}
		out = append(out, tx)
	}
	return out, rows.Err()
}

// ListEvents returns recent indexed events, optionally filtered by type.
func (s *Store) ListEvents(ctx context.Context, chainID, eventType string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if eventType != "" {
		return s.scanEvents(ctx, `
			SELECT id, chain_id, block_number, tx_hash, log_index, contract_address, event_type, payload, created_at
			FROM indexed_events
			WHERE chain_id = $1 AND event_type = $2
			ORDER BY block_number DESC, log_index DESC LIMIT $3`, chainID, eventType, limit)
	}
	return s.scanEvents(ctx, `
		SELECT id, chain_id, block_number, tx_hash, log_index, contract_address, event_type, payload, created_at
		FROM indexed_events WHERE chain_id = $1
		ORDER BY block_number DESC, log_index DESC LIMIT $2`, chainID, limit)
}

func (s *Store) scanEvents(ctx context.Context, q string, args ...any) ([]Event, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.ChainID, &e.BlockNumber, &e.TxHash, &e.LogIndex,
			&e.ContractAddress, &e.EventType, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(payload) > 0 {
			e.Payload = json.RawMessage(payload)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanBlocks(rows interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Block, error) {
	defer rows.Close()
	var out []Block
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.ChainID, &b.Number, &b.Hash, &b.ParentHash, &b.Timestamp, &b.TxCount); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
