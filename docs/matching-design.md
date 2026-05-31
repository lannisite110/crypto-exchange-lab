# Matching engine (Phase 1)

## Algorithm

Price-time priority per symbol:

1. Incoming order matches against the opposite side while price crosses.
2. Limit buy matches asks at `ask.price <= buy.price`.
3. Limit sell matches bids at `bid.price >= sell.price`.
4. Same price level: earlier `CreatedAt` wins.

## Package

Core logic lives in `packages/matching` (`Book.Match`, `Book.Cancel`, `Book.Depth`).

`services/matching-engine` holds in-memory books for `BTC/USDT` and `ETH/USDT`.

## APIs

- `POST /api/v1/orders` — submit order to the book
- `DELETE /api/v1/orders/{id}?symbol=` — cancel resting order
- `GET /api/v1/markets/{symbol}/depth`
- `GET /ws/v1/market` — depth/trade snapshots
