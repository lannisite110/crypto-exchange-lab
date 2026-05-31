# RPC Gateway + Explorer (Phase 5)

Read-only multi-chain layer for the lab Sepolia AMM.

## Services

| Service | Port | Role |
|---------|------|------|
| `rpc-gateway` | 8089 | REST API: blocks, txs, events, live RPC passthrough |
| `indexer` | 8090 | Polls Sepolia, writes Postgres, watches AMM contracts |

## API (rpc-gateway)

- `GET /api/v1/chains`
- `GET /api/v1/chains/{chainId}/status` — sync lag vs live head
- `GET /api/v1/chains/{chainId}/blocks`
- `GET /api/v1/chains/{chainId}/blocks/{number}`
- `GET /api/v1/chains/{chainId}/transactions/{hash}` — DB first, then live RPC
- `GET /api/v1/chains/{chainId}/addresses/{addr}/balance` — live `eth_getBalance`
- `GET /api/v1/chains/{chainId}/events?type=Swap`
- `GET /api/v1/chains/{chainId}/live/block/latest`

## Indexer env

After `pnpm contracts:deploy:sepolia`, set:

```bash
SEPOLIA_RPC_URL=https://rpc.sepolia.org
INDEXER_AMM_PAIR=0x...
INDEXER_LAB_TOKEN=0x...
INDEXER_LAB_USD=0x...
```

## Local start

```bash
make migrate   # includes 005
make run-rpc-gateway
make run-indexer
cd apps/explorer-web && pnpm dev   # :3004
```

Explorer UI: [http://localhost:3004](http://localhost:3004)
