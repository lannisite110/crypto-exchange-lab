# Roadmap

## Phase 0 — Scaffold ✅

- [x] Monorepo (pnpm + Turborepo + Go workspace)
- [x] Docker Compose (Postgres, Redis)
- [x] `go-common` package
- [x] Stub apps and Hardhat `LabToken`
- [x] CI workflow

## Phase 1 — CEX core ✅

- [x] Account service: balances, freeze/unfreeze
- [x] Double-entry ledger
- [x] Matching engine (BTC/USDT, ETH/USDT)
- [x] Order service
- [x] CEX web UI

## Phase 2 — OrderBook DEX ✅

- [x] Orderbook-dex API (shared matcher, isolated DEX book)
- [x] Venue column on orders/trades
- [x] dex-web UI

## Phase 3 — Hyperliquid-style perps ✅

- [x] Positions, leverage, margin (hyperliquid-engine)
- [x] Risk, liquidation, funding engines
- [x] HyperDEX web

## Phase 4 — AMM DEX ✅

- [x] Uniswap V2–style contracts (Factory, Pair, Router)
- [x] LAB / LUSD test tokens + Hardhat tests
- [x] Sepolia deploy script (`pnpm contracts:deploy:sepolia`)
- [x] dex-web AMM tab (wagmi) + OrderBook tab

## Phase 5 — RPC gateway + Explorer ✅

- [x] `rpc-gateway` multi-chain read API (Sepolia + live RPC fallback)
- [x] `indexer` — blocks, txs, AMM Swap/Mint/Burn events
- [x] `explorer-web` UI (:3004)

## Phase 6 — Production demo polish ✅

- [x] Prometheus + Grafana (`make monitoring-up`)
- [x] `/metrics` on all Go services (`go-common/metrics`)
- [x] Deploy runbooks + Docker full stack + Fly/Vercel configs — [deploy.md](./deploy.md)

## Phase 7 — Multi-asset spot demo (path A) ✅

- [x] 10 USDT markets (BTC, ETH, SOL, BNB, XRP, DOGE, LINK, AVAX, ADA, DOT)
- [x] DB migration + dynamic `/api/v1/symbols`
- [x] Matching engine seed liquidity — [markets.md](./markets.md)
