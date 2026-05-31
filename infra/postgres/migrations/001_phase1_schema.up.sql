-- Phase 1: CEX ledger, orders, and trades (simulated)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS assets (
    id          SMALLSERIAL PRIMARY KEY,
    symbol      VARCHAR(16) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(64) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asset_id    SMALLINT NOT NULL REFERENCES assets (id),
    available   NUMERIC(36, 18) NOT NULL DEFAULT 0 CHECK (available >= 0),
    frozen      NUMERIC(36, 18) NOT NULL DEFAULT 0 CHECK (frozen >= 0),
    PRIMARY KEY (user_id, asset_id)
);

CREATE TABLE IF NOT EXISTS ledger_transactions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ref_type    VARCHAR(32) NOT NULL,
    ref_id      VARCHAR(64) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ref_type, ref_id)
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id              BIGSERIAL PRIMARY KEY,
    transaction_id  UUID NOT NULL REFERENCES ledger_transactions (id),
    user_id         UUID NOT NULL REFERENCES users (id),
    asset_id        SMALLINT NOT NULL REFERENCES assets (id),
    amount          NUMERIC(36, 18) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_txn ON ledger_entries (transaction_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user ON ledger_entries (user_id, asset_id);

CREATE TABLE IF NOT EXISTS symbols (
    id              SMALLSERIAL PRIMARY KEY,
    name            VARCHAR(32) NOT NULL UNIQUE,
    base_asset_id   SMALLINT NOT NULL REFERENCES assets (id),
    quote_asset_id  SMALLINT NOT NULL REFERENCES assets (id),
    tick_size       NUMERIC(36, 18) NOT NULL DEFAULT '0.01',
    lot_size        NUMERIC(36, 18) NOT NULL DEFAULT '0.0001'
);

DO $$ BEGIN
    CREATE TYPE order_side AS ENUM ('BUY', 'SELL');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE order_type AS ENUM ('LIMIT', 'MARKET');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE order_status AS ENUM ('NEW', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users (id),
    symbol_id   SMALLINT NOT NULL REFERENCES symbols (id),
    side        order_side NOT NULL,
    type        order_type NOT NULL,
    status      order_status NOT NULL DEFAULT 'NEW',
    price       NUMERIC(36, 18),
    quantity    NUMERIC(36, 18) NOT NULL CHECK (quantity > 0),
    filled_qty  NUMERIC(36, 18) NOT NULL DEFAULT 0 CHECK (filled_qty >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user ON orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_symbol_status ON orders (symbol_id, status);

CREATE TABLE IF NOT EXISTS trades (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol_id       SMALLINT NOT NULL REFERENCES symbols (id),
    buy_order_id    UUID NOT NULL REFERENCES orders (id),
    sell_order_id   UUID NOT NULL REFERENCES orders (id),
    buyer_user_id   UUID NOT NULL REFERENCES users (id),
    seller_user_id  UUID NOT NULL REFERENCES users (id),
    price           NUMERIC(36, 18) NOT NULL,
    quantity        NUMERIC(36, 18) NOT NULL CHECK (quantity > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_symbol_time ON trades (symbol_id, created_at DESC);
