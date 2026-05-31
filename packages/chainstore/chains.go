package chainstore

import "context"

// ListChains returns enabled chains.
func (s *Store) ListChains(ctx context.Context) ([]Chain, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, chain_id, name, rpc_url
		FROM chains WHERE enabled = TRUE ORDER BY chain_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chain
	for rows.Next() {
		var c Chain
		if err := rows.Scan(&c.ID, &c.ChainID, &c.Name, &c.RPCURL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChain loads one chain by id.
func (s *Store) GetChain(ctx context.Context, id string) (*Chain, error) {
	var c Chain
	err := s.pool.QueryRow(ctx, `
		SELECT id, chain_id, name, rpc_url
		FROM chains WHERE id = $1 AND enabled = TRUE`, id).
		Scan(&c.ID, &c.ChainID, &c.Name, &c.RPCURL)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetSyncState returns indexer cursor.
func (s *Store) GetSyncState(ctx context.Context, chainID string) (*SyncState, error) {
	var st SyncState
	err := s.pool.QueryRow(ctx, `
		SELECT chain_id, last_indexed_block, updated_at
		FROM chain_sync_state WHERE chain_id = $1`, chainID).
		Scan(&st.ChainID, &st.LastIndexedBlock, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SetSyncState updates indexer cursor.
func (s *Store) SetSyncState(ctx context.Context, chainID string, block int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO chain_sync_state (chain_id, last_indexed_block, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (chain_id) DO UPDATE
		SET last_indexed_block = EXCLUDED.last_indexed_block, updated_at = NOW()`,
		chainID, block)
	return err
}

// UpsertWatchedContract registers a contract to watch.
func (s *Store) UpsertWatchedContract(ctx context.Context, chainID, address, label string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO watched_contracts (chain_id, address, label)
		VALUES ($1, LOWER($2), $3)
		ON CONFLICT (chain_id, address) DO UPDATE SET label = EXCLUDED.label`,
		chainID, address, label)
	return err
}

// ListWatchedContracts returns watched addresses for a chain.
func (s *Store) ListWatchedContracts(ctx context.Context, chainID string) ([]WatchedContract, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT chain_id, address, label FROM watched_contracts
		WHERE chain_id = $1 ORDER BY label`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WatchedContract
	for rows.Next() {
		var w WatchedContract
		if err := rows.Scan(&w.ChainID, &w.Address, &w.Label); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
