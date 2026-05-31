# Spot markets (Phase 7 — path A)

Ten **USDT-quoted** demo markets for simulated CEX and OrderBook DEX. Balances are not real assets.

| Symbol | Base | Reference mid (demo) |
|--------|------|----------------------|
| BTC/USDT | BTC | 100,000 |
| ETH/USDT | ETH | 3,500 |
| SOL/USDT | SOL | 180 |
| BNB/USDT | BNB | 600 |
| XRP/USDT | XRP | 2.2 |
| DOGE/USDT | DOGE | 0.15 |
| LINK/USDT | LINK | 15 |
| AVAX/USDT | AVAX | 35 |
| ADA/USDT | ADA | 0.55 |
| DOT/USDT | DOT | 7 |

## Setup

```bash
make migrate   # includes 006_phase7_markets.up.sql
make run-matching   # seeds ±0.2% bid/ask on CEX + DEX books (default on)
make run-order      # CEX
make run-orderbook-dex
```

Disable in-memory seed quotes: `MATCHING_SEED_BOOKS=false`.

## API

- `GET /api/v1/symbols` on **order-service** (:8082) and **orderbook-dex** (:8084) — reads Postgres `symbols` table.
- **matching-engine** builds one order book per `(venue, symbol)`; venues API lists all symbols.

Perpetuals remain **BTC-PERP** and **ETH-PERP** only (mark price from CEX mid on the spot pair).
