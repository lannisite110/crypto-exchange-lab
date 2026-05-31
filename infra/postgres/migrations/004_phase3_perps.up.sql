-- Phase 3: perpetual futures (USDT-margined, simulated)

DO $$ BEGIN
    CREATE TYPE position_side AS ENUM ('LONG', 'SHORT');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS perp_markets (
    symbol              VARCHAR(32) PRIMARY KEY,
    spot_symbol         VARCHAR(32) NOT NULL,
    base_asset          VARCHAR(16) NOT NULL,
    quote_asset         VARCHAR(16) NOT NULL DEFAULT 'USDT',
    max_leverage        INT NOT NULL DEFAULT 20,
    maint_margin_rate   NUMERIC(12, 8) NOT NULL DEFAULT 0.005,
    taker_fee_rate      NUMERIC(12, 8) NOT NULL DEFAULT 0.0005
);

CREATE TABLE IF NOT EXISTS mark_prices (
    symbol      VARCHAR(32) PRIMARY KEY REFERENCES perp_markets (symbol),
    price       NUMERIC(36, 18) NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS perp_positions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users (id),
    symbol      VARCHAR(32) NOT NULL REFERENCES perp_markets (symbol),
    side        position_side NOT NULL,
    size        NUMERIC(36, 18) NOT NULL CHECK (size > 0),
    entry_price NUMERIC(36, 18) NOT NULL,
    leverage    INT NOT NULL CHECK (leverage >= 1 AND leverage <= 100),
    margin      NUMERIC(36, 18) NOT NULL CHECK (margin > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_perp_positions_user ON perp_positions (user_id);

CREATE TABLE IF NOT EXISTS perp_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id),
    symbol      VARCHAR(32) NOT NULL,
    event_type  VARCHAR(32) NOT NULL,
    size        NUMERIC(36, 18),
    price       NUMERIC(36, 18),
    leverage    INT,
    pnl         NUMERIC(36, 18),
    ref_id      VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS funding_rates (
    id              BIGSERIAL PRIMARY KEY,
    symbol          VARCHAR(32) NOT NULL REFERENCES perp_markets (symbol),
    rate            NUMERIC(12, 8) NOT NULL,
    interval_start  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS funding_payments (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id),
    symbol      VARCHAR(32) NOT NULL,
    rate        NUMERIC(12, 8) NOT NULL,
    payment     NUMERIC(36, 18) NOT NULL,
    mark_price  NUMERIC(36, 18) NOT NULL,
    position_size NUMERIC(36, 18) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO perp_markets (symbol, spot_symbol, base_asset, quote_asset, max_leverage, maint_margin_rate)
VALUES
    ('BTC-PERP', 'BTC/USDT', 'BTC', 'USDT', 20, 0.005),
    ('ETH-PERP', 'ETH/USDT', 'ETH', 'USDT', 20, 0.005)
ON CONFLICT (symbol) DO NOTHING;

INSERT INTO mark_prices (symbol, price) VALUES
    ('BTC-PERP', 100000),
    ('ETH-PERP', 3500)
ON CONFLICT (symbol) DO NOTHING;

INSERT INTO users (username) VALUES ('perp_house')
ON CONFLICT (username) DO NOTHING;

INSERT INTO balances (user_id, asset_id, available, frozen)
SELECT u.id, a.id, '10000000', 0
FROM users u
CROSS JOIN assets a
WHERE u.username = 'perp_house' AND a.symbol = 'USDT'
ON CONFLICT (user_id, asset_id) DO NOTHING;
