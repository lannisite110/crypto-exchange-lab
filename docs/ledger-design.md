# Double-entry ledger (Phase 1)

## Model

- `balances`: per-user `available` and `frozen`
- `ledger_transactions`: idempotent by `(ref_type, ref_id)`
- `ledger_entries`: signed `amount` per user/asset (`+` credit, `-` debit)

Each transaction must sum to **zero per asset** across all legs.

## Order collateral

| Side | Frozen asset | Amount |
|------|----------------|--------|
| BUY  | Quote (USDT)   | `quantity × price` |
| SELL | Base (BTC/ETH) | `quantity` |

## Trade settlement (example)

BTC/USDT fill: `0.5 @ 100000`

| User | Asset | Amount |
|------|-------|--------|
| Buyer  | USDT | -50000 |
| Buyer  | BTC  | +0.5   |
| Seller | BTC  | -0.5   |
| Seller | USDT | +50000 |

Frozen collateral is released on the `freeze_release` legs without returning to available.

## APIs

- `POST /api/v1/internal/freeze`
- `POST /api/v1/internal/unfreeze`
- `POST /api/v1/internal/settle-trade`

Public: `GET /api/v1/users/{id}/balances`
