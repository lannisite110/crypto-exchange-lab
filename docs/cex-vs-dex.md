# CEX vs OrderBook DEX (Phase 2+)

Spot markets: **10 USDT pairs** (BTC, ETH, SOL, BNB, XRP, DOGE, LINK, AVAX, ADA, DOT) — see [markets.md](./markets.md).

| | CEX (`order-service`) | OrderBook DEX (`orderbook-dex`) |
|--|------------------------|----------------------------------|
| **Port** | 8082 | 8084 |
| **UI** | cex-web :3003 | dex-web :3001 |
| **Venue** | `CEX` | `DEX` |
| **Order book** | Isolated in matching-engine | Isolated in matching-engine |
| **Matcher** | Same binary, same algorithm | Same binary, same algorithm |
| **Balances** | Shared `account-service` ledger | Shared `account-service` ledger |
| **Settlement** | Off-chain simulated | Off-chain simulated (on-chain later) |
| **Freeze ref** | `cex_order` | `dex_order` |

## Why two books?

CEX and DEX orders **do not cross** each other. A sell on the CEX book will not fill a buy on the DEX book. This mirrors production where central limit order books and DEX L2 books are separate liquidity pools.

## AMM (Phase 4) ✅

| | OrderBook DEX | AMM (`contracts/src/amm`) |
|--|---------------|---------------------------|
| **UI tab** | dex-web → OrderBook | dex-web → AMM (Sepolia) |
| **Matching** | Limit order book | `x * y = k` pool |
| **Settlement** | Postgres ledger | On-chain LAB / LUSD |

See [amm-dex.md](./amm-dex.md).
