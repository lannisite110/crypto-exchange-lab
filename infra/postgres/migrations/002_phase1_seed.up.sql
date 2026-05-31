-- Demo assets, symbols, and two traders with simulated balances

INSERT INTO assets (symbol) VALUES ('BTC'), ('USDT'), ('ETH')
ON CONFLICT (symbol) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'BTC/USDT', b.id, q.id, '0.01', '0.0001'
FROM assets b, assets q WHERE b.symbol = 'BTC' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO symbols (name, base_asset_id, quote_asset_id, tick_size, lot_size)
SELECT 'ETH/USDT', b.id, q.id, '0.01', '0.001'
FROM assets b, assets q WHERE b.symbol = 'ETH' AND q.symbol = 'USDT'
ON CONFLICT (name) DO NOTHING;

INSERT INTO users (username) VALUES ('alice'), ('bob')
ON CONFLICT (username) DO NOTHING;

-- Alice: 10 BTC, 500k USDT, 100 ETH
INSERT INTO balances (user_id, asset_id, available, frozen)
SELECT u.id, a.id,
    CASE a.symbol
        WHEN 'BTC' THEN '10'::NUMERIC
        WHEN 'USDT' THEN '500000'::NUMERIC
        WHEN 'ETH' THEN '100'::NUMERIC
        ELSE '0'::NUMERIC
    END,
    0
FROM users u
CROSS JOIN assets a
WHERE u.username = 'alice'
ON CONFLICT (user_id, asset_id) DO NOTHING;

-- Bob: 5 BTC, 1M USDT, 50 ETH
INSERT INTO balances (user_id, asset_id, available, frozen)
SELECT u.id, a.id,
    CASE a.symbol
        WHEN 'BTC' THEN '5'::NUMERIC
        WHEN 'USDT' THEN '1000000'::NUMERIC
        WHEN 'ETH' THEN '50'::NUMERIC
        ELSE '0'::NUMERIC
    END,
    0
FROM users u
CROSS JOIN assets a
WHERE u.username = 'bob'
ON CONFLICT (user_id, asset_id) DO NOTHING;
