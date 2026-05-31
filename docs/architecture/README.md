# Architecture

High-level modules (see root README diagram):

| Module | Role |
|--------|------|
| `account-service` | Balances, freeze, double-entry ledger |
| `matching-engine` | Price-time priority matching |
| `order-service` | Place/cancel orders |
| `hyperliquid-engine` | Perpetual positions and margin |
| `risk-engine` | Margin ratio checks |
| `liquidation-engine` | Forced close |
| `funding-engine` | Funding rate settlement |
| `rpc-gateway` | Unified chain read API ✅ |
| `indexer` | Chain events → Postgres ✅ |
| `contracts/amm` | AMM (Phase 4) ✅ |

Detailed design docs (`matching-design.md`, `ledger-design.md`) land in Phase 1+.

Observability: Prometheus scrapes `GET /metrics` on each Go service — see [observability.md](../observability.md).
