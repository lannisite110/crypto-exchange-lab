# Deployment guide

Educational demo only — **no real funds**. Two recommended paths:

| Path | Best for | HTTPS |
|------|----------|-------|
| **A — Docker demo** | VPS / home lab / Railway Docker | Add reverse proxy (Caddy) or tunnel |
| **B — Vercel + Fly** | Portfolio public URLs | Yes (Fly + Vercel default) |

---

## Path A — One-command backend (Docker)

Runs Postgres, Redis, migrations, and **all 10 Go services** on the host ports `8081–8090`.

```bash
chmod +x scripts/*.sh
./scripts/deploy-demo.sh
```

Or manually:

```bash
docker compose -f infra/docker/docker-compose.yml \
  -f infra/docker/docker-compose.demo.yml up -d --build
```

Verify:

```bash
curl -s http://localhost:8081/health
curl -s http://localhost:8082/api/v1/symbols
curl -s http://localhost:8083/api/v1/venues
```

**Frontends** (pick one):

1. **Local:** `cd apps/cex-web && pnpm dev` (and other apps).
2. **Vercel:** import repo, set root to `apps/cex-web`, use `vercel.json`.  
   For Docker-only APIs you need **HTTPS** on the same host (mixed content blocks `http://` from Vercel). Use Path B for public demo, or put **Caddy** in front of Docker with TLS.

Stop stack:

```bash
docker compose -f infra/docker/docker-compose.yml \
  -f infra/docker/docker-compose.demo.yml down
```

Migrations on a remote DB:

```bash
DATABASE_URL='postgres://...' ./scripts/migrate.sh
```

---

## Path B — Vercel (UI) + Fly.io (API)

**Recommended (Neon + Fly + Vercel):** step-by-step guide in Chinese — **[deploy-neon-fly-vercel.md](./deploy-neon-fly-vercel.md)**.  
Quick scripts: copy `.env.deploy.example` → `.env.deploy`, migrate, then `./scripts/fly-deploy-minimal.sh`.

### 1. Database

**Option A — Neon (recommended for portfolio):**

```bash
# Neon console → copy URI with sslmode=require
cp .env.deploy.example .env.deploy   # edit DATABASE_URL
./scripts/migrate.sh
```

**Option B — Fly Postgres:**

```bash
fly postgres create --name cel-db --region sin
fly postgres attach cel-db -a cel-account
DATABASE_URL='postgres://...' ./scripts/migrate.sh
```

### 2. Deploy Go services

From repo root (requires [flyctl](https://fly.io/docs/hands-on/install-flyctl/) and `fly auth login`):

```bash
./scripts/fly-setup.sh   # prints app names and commands

fly secrets set DATABASE_URL='postgres://...' -a cel-account
fly deploy . -c infra/deploy/fly/account-service.toml -a cel-account

fly secrets set DATABASE_URL='...' MATCHING_SEED_BOOKS=true -a cel-matching
fly deploy . -c infra/deploy/fly/matching-engine.toml -a cel-matching

fly secrets set DATABASE_URL='...' \
  ACCOUNT_SERVICE_URL=https://cel-account.fly.dev \
  MATCHING_ENGINE_URL=https://cel-matching.fly.dev \
  -a cel-order
fly deploy . -c infra/deploy/fly/order-service.toml -a cel-order
```

Repeat for `orderbook-dex`, `hyperliquid-engine`, `risk-engine`, `liquidation-engine`, `funding-engine`, `rpc-gateway`, `indexer` using configs under `infra/deploy/fly/`.  
Generate more configs by copying `order-service.toml` and changing `SERVICE`, port, and app name.

**Deploy order:** Postgres+migrate → account + matching → order + orderbook-dex → hyperliquid → risk → liquidation → funding → rpc-gateway → indexer.

### 3. Vercel — four projects

| Project | Root directory | Config |
|---------|----------------|--------|
| CEX | `apps/cex-web` | `vercel.json` |
| DEX | `apps/dex-web` | `vercel.json` |
| HyperDEX | `apps/hyperdex-web` | `vercel.json` |
| Explorer | `apps/explorer-web` | `vercel.json` |

Environment variables (example — use your Fly hostnames):

```bash
# cex-web
NEXT_PUBLIC_ACCOUNT_API_URL=https://cel-account.fly.dev
NEXT_PUBLIC_ORDER_API_URL=https://cel-order.fly.dev
NEXT_PUBLIC_MATCHING_API_URL=https://cel-matching.fly.dev
NEXT_PUBLIC_MATCHING_WS_URL=wss://cel-matching.fly.dev/ws/v1/market

# dex-web
NEXT_PUBLIC_ORDERBOOK_DEX_API_URL=https://cel-orderbook-dex.fly.dev
# + NEXT_PUBLIC_LAB_* after Sepolia AMM deploy

# hyperdex-web
NEXT_PUBLIC_HYPERLIQUID_API_URL=https://cel-hyperliquid.fly.dev
NEXT_PUBLIC_RISK_API_URL=https://cel-risk.fly.dev
NEXT_PUBLIC_FUNDING_API_URL=https://cel-funding.fly.dev

# explorer-web
NEXT_PUBLIC_RPC_GATEWAY_URL=https://cel-rpc-gateway.fly.dev
```

Template file: [`.env.production.example`](../.env.production.example).

### 4. Sepolia AMM (optional)

```bash
export SEPOLIA_RPC_URL=https://ethereum-sepolia-rpc.publicnode.com
export PRIVATE_KEY=0x...   # test wallet only
pnpm contracts:deploy:sepolia
```

Copy addresses to Vercel (`NEXT_PUBLIC_*`) and Fly indexer secrets (`INDEXER_AMM_PAIR`, etc.).

---

## Railway (alternative)

1. New project from GitHub repo.
2. Add **PostgreSQL** plugin; set `DATABASE_URL` on every service.
3. Create a service per Go binary; **Dockerfile path:** `infra/deploy/Dockerfile.go-service` with build arg `SERVICE=account-service`.
4. Or run `./scripts/deploy-demo.sh` on a Railway **Docker** service with public networking.

---

## Docker image (single service)

```bash
docker build -f infra/deploy/Dockerfile.go-service \
  --build-arg SERVICE=matching-engine \
  -t cel-matching .
docker run --rm -p 8083:8083 \
  -e MATCHING_ENGINE_HTTP_PORT=8083 \
  -e DATABASE_URL=... \
  cel-matching
```

---

## Demo checklist

- [ ] `GET /health` on each Fly app
- [ ] CEX: 10 symbols in `/api/v1/symbols`, place order
- [ ] DEX order book: venue isolated trades
- [ ] HyperDEX: perp open/close
- [ ] Explorer: blocks after indexer runs
- [ ] README **Live demo** links updated

---

## Portfolio README

After URLs exist:

```markdown
**Live demo:** [CEX](https://your-cex.vercel.app) · [DEX](https://your-dex.vercel.app) · [HyperDEX](https://your-hyperdex.vercel.app) · [Explorer](https://your-explorer.vercel.app)
```

Keep [DISCLAIMER.md](./DISCLAIMER.md) on every public page.
