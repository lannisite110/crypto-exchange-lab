package chainstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides Postgres access for chain indexer data.
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

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Chain metadata.
type Chain struct {
	ID      string `json:"id"`
	ChainID int64  `json:"chain_id"`
	Name    string `json:"name"`
	RPCURL  string `json:"rpc_url"`
}

// SyncState for a chain.
type SyncState struct {
	ChainID          string    `json:"chain_id"`
	LastIndexedBlock int64     `json:"last_indexed_block"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Block row.
type Block struct {
	ChainID    string    `json:"chain_id"`
	Number     int64     `json:"number"`
	Hash       string    `json:"hash"`
	ParentHash string    `json:"parent_hash"`
	Timestamp  time.Time `json:"timestamp"`
	TxCount    int       `json:"tx_count"`
}

// Transaction row.
type Transaction struct {
	ChainID     string  `json:"chain_id"`
	Hash        string  `json:"hash"`
	BlockNumber int64   `json:"block_number"`
	TxIndex     int     `json:"tx_index"`
	FromAddr    string  `json:"from_addr"`
	ToAddr      string  `json:"to_addr,omitempty"`
	ValueWei    string  `json:"value_wei"`
	GasUsed     *int64  `json:"gas_used,omitempty"`
	Status      *int    `json:"status,omitempty"`
}

// Event row.
type Event struct {
	ID              int64           `json:"id"`
	ChainID         string          `json:"chain_id"`
	BlockNumber     int64           `json:"block_number"`
	TxHash          string          `json:"tx_hash"`
	LogIndex        int             `json:"log_index"`
	ContractAddress string          `json:"contract_address"`
	EventType       string          `json:"event_type"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// WatchedContract label for explorer.
type WatchedContract struct {
	ChainID string `json:"chain_id"`
	Address string `json:"address"`
	Label   string `json:"label"`
}
