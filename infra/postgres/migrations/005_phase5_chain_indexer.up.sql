-- Phase 5: chain indexer + explorer

CREATE TABLE IF NOT EXISTS chains (
    id          VARCHAR(32) PRIMARY KEY,
    chain_id    BIGINT NOT NULL UNIQUE,
    name        VARCHAR(64) NOT NULL,
    rpc_url     TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE
);

INSERT INTO chains (id, chain_id, name, rpc_url)
VALUES ('sepolia', 11155111, 'Sepolia', 'https://rpc.sepolia.org')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS chain_sync_state (
    chain_id            VARCHAR(32) PRIMARY KEY REFERENCES chains (id),
    last_indexed_block  BIGINT NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO chain_sync_state (chain_id, last_indexed_block)
VALUES ('sepolia', 0)
ON CONFLICT (chain_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS watched_contracts (
    chain_id    VARCHAR(32) NOT NULL REFERENCES chains (id),
    address     VARCHAR(42) NOT NULL,
    label       VARCHAR(64) NOT NULL,
    PRIMARY KEY (chain_id, address)
);

CREATE TABLE IF NOT EXISTS indexed_blocks (
    chain_id      VARCHAR(32) NOT NULL REFERENCES chains (id),
    number        BIGINT NOT NULL,
    hash          VARCHAR(66) NOT NULL,
    parent_hash   VARCHAR(66) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    tx_count      INT NOT NULL DEFAULT 0,
    PRIMARY KEY (chain_id, number)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_indexed_blocks_hash
    ON indexed_blocks (chain_id, hash);

CREATE TABLE IF NOT EXISTS indexed_transactions (
    chain_id      VARCHAR(32) NOT NULL,
    hash          VARCHAR(66) NOT NULL,
    block_number  BIGINT NOT NULL,
    tx_index      INT NOT NULL,
    from_addr     VARCHAR(42) NOT NULL,
    to_addr       VARCHAR(42),
    value_wei     NUMERIC(78, 0) NOT NULL DEFAULT 0,
    gas_used      BIGINT,
    status        SMALLINT,
    PRIMARY KEY (chain_id, hash),
    FOREIGN KEY (chain_id, block_number)
        REFERENCES indexed_blocks (chain_id, number) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_indexed_tx_block
    ON indexed_transactions (chain_id, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_indexed_tx_from
    ON indexed_transactions (chain_id, from_addr);

CREATE TABLE IF NOT EXISTS indexed_events (
    id                BIGSERIAL PRIMARY KEY,
    chain_id          VARCHAR(32) NOT NULL,
    block_number      BIGINT NOT NULL,
    tx_hash           VARCHAR(66) NOT NULL,
    log_index         INT NOT NULL,
    contract_address  VARCHAR(42) NOT NULL,
    event_type        VARCHAR(32) NOT NULL,
    topic0            VARCHAR(66),
    raw_data          TEXT,
    payload           JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_indexed_events_chain_block
    ON indexed_events (chain_id, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_indexed_events_type
    ON indexed_events (chain_id, event_type);
