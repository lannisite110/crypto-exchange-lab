-- Phase 2: isolate CEX vs OrderBook DEX orders/trades

DO $$ BEGIN
    CREATE TYPE venue_type AS ENUM ('CEX', 'DEX');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS venue venue_type NOT NULL DEFAULT 'CEX';
ALTER TABLE trades ADD COLUMN IF NOT EXISTS venue venue_type NOT NULL DEFAULT 'CEX';

CREATE INDEX IF NOT EXISTS idx_orders_venue_user ON orders (venue, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_venue_symbol ON trades (venue, symbol_id, created_at DESC);
