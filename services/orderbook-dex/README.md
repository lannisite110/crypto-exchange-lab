# orderbook-dex

Off-chain **OrderBook DEX** API (Phase 2). Uses the same matching-engine binary as CEX but with an isolated `DEX` venue book.

```bash
make run-orderbook-dex   # :8084
curl http://localhost:8084/api/v1/venue
```

Shares `account-service` for simulated wallet balances. Orders are tagged `venue=DEX` in Postgres.
