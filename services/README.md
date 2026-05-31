# Services

Go microservices (simulated exchange backend).

| Service | Port (local) | Phase |
|---------|----------------|-------|
| `account-service` | 8081 | 0 (health) / 1 (ledger) |
| `order-service` | 8082 | 1 |
| `matching-engine` | 8083 | 1 |
| `orderbook-dex` | 8084 | 2 ✅ |
| `hyperliquid-engine` | 8085 | 3 ✅ |
| `risk-engine` | 8086 | 3 ✅ |
| `liquidation-engine` | 8087 | 3 ✅ |
| `funding-engine` | 8088 | 3 ✅ |
| `rpc-gateway` | 8090 | 5 |
| `indexer` | 8091 | 5 |
| `api-gateway` | 8080 | 6 |

Run account-service health check:

```bash
go run ./services/account-service/cmd/server
curl localhost:8081/health
```
