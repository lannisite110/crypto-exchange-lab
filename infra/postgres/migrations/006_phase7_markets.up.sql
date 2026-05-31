-- Phase 7 (path A): 10 representative USDT spot markets for CEX + OrderBook DEX demo

INSERT INTO assets (symbol) VALUES
    ('SOL'), ('BNB'), ('XRP'), ('DOGE'), ('LINK'), ('AVAX'), ('ADA'), ('DOT')
ON CONFLICT (symbol) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'SOL/USDT', b.id, q.id, '0.01', '0.01'
FROM assets b, assets q WHERE b.symbol = 'SOL' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'BNB/USDT', b.id, q.id, '0.01', '0.01'
FROM assets b, assets q WHERE b.symbol = 'BNB' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'XRP/USDT', b.id, q.id, '0.0001', '1'
FROM assets b, assets q WHERE b.symbol = 'XRP' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'DOGE/USDT', b.id, q.id, '0.00001', '10'
FROM assets b, assets q WHERE b.symbol = 'DOGE' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'LINK/USDT', b.id, q.id, '0.01', '0.1'
FROM assets b, assets q WHERE b.symbol = 'LINK' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'AVAX/USDT', b.id, q.id, '0.01', '0.1'
FROM assets b, assets q WHERE b.symbol = 'AVAX' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'ADA/USDT', b.id, q.id, '0.0001', '10'
FROM assets b, assets q WHERE b.symbol = 'ADA' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'DOT/USDT', b.id, q.id, '0.01', '0.1'
FROM assets b, assets q WHERE b.symbol = 'DOT' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

-- Extra demo balances for alice / bob (new bases + more USDT)
INSERT INTO balances (user_id, asset_id, available, frozen)
SELECT u.id, a.id,
    CASE a.symbol
        WHEN 'SOL' THEN '500'::NUMERIC
        WHEN 'BNB' THEN '200'::NUMERIC
        WHEN 'XRP' THEN '50000'::NUMERIC
        WHEN 'DOGE' THEN '500000'::NUMERIC
        WHEN 'LINK' THEN '2000'::NUMERIC
        WHEN 'AVAX' THEN '1500'::NUMERIC
        WHEN 'ADA' THEN '100000'::NUMERIC
        WHEN 'DOT' THEN '5000'::NUMERIC
        WHEN 'USDT' THEN '2000000'::NUMERIC
        ELSE '0'::NUMERIC
    END,
    0
FROM users u
CROSS JOIN assets a
WHERE u.username = 'alice'
  AND a.symbol IN ('SOL','BNB','XRP','DOGE','LINK','AVAX','ADA','DOT','USDT')
ON CONFLICT (user_id, asset_id) DO UPDATE
SET available = GREATEST(balances.available, EXCLUDED.available);

INSERT INTO balances (user_id, asset_id, available, frozen)
SELECT u.id, a.id,
    CASE a.symbol
        WHEN 'SOL' THEN '300'::NUMERIC
        WHEN 'BNB' THEN '150'::NUMERIC
        WHEN 'XRP' THEN '30000'::NUMERIC
        WHEN 'DOGE' THEN '300000'::NUMERIC
        WHEN 'LINK' THEN '1200'::NUMERIC
        WHEN 'AVAX' THEN '1000'::NUMERIC
        WHEN 'ADA' THEN '80000'::NUMERIC
        WHEN 'DOT' THEN '4000'::NUMERIC
        WHEN 'USDT' THEN '3000000'::NUMERIC
        ELSE '0'::NUMERIC
    END,
    0
FROM users u
CROSS JOIN assets a
WHERE u.username = 'bob'
  AND a.symbol IN ('SOL','BNB','XRP','DOGE','LINK','AVAX','ADA','DOT','USDT')
ON CONFLICT (user_id, asset_id) DO UPDATE
SET available = GREATEST(balances.available, EXCLUDED.available);
