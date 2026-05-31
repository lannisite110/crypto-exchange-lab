# AMM DEX (Phase 4)

Constant-product AMM (Uniswap V2–style) deployed to **Sepolia** for learning. No real economic value — `LabToken` (LAB) and `LabUSD` (LUSD) are test ERC20s.

## Contracts

| Contract | Role |
|----------|------|
| `LabToken` / `LabUSD` | Test tokens |
| `UniswapV2Factory` | Creates LAB/LUSD pairs |
| `UniswapV2Pair` | Pool + LP token (`CEL-LP`) |
| `UniswapV2Router` | Swap, add/remove liquidity (0.3% fee) |

## Deploy to Sepolia

```bash
cd contracts
# .env or shell: SEPOLIA_RPC_URL, PRIVATE_KEY (funded test wallet)
pnpm exec hardhat run scripts/deploy-amm.ts --network sepolia
```

Output: `contracts/deployments/sepolia.json`. Copy addresses into `apps/dex-web` env (see `.env.example`).

## dex-web

- **AMM (Sepolia)** — wagmi wallet, swap LAB↔LUSD, liquidity
- **OrderBook (simulated)** — Phase 2 off-chain matcher (`orderbook-dex` :8084)

## vs OrderBook DEX

| | OrderBook DEX | AMM |
|--|---------------|-----|
| Matching | Central limit order book | `x * y = k` pool |
| Settlement | Postgres ledger | On-chain ERC20 transfers |
| Liquidity | Limit orders | LP deposits |

See also [cex-vs-dex.md](./cex-vs-dex.md).
