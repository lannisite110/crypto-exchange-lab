# 上线指南：Neon + Fly + Vercel（分步）

按顺序做。每步都有 **验收命令**；通过后再做下一步。

> 全程使用测试网/模拟资金。`PRIVATE_KEY` 不要提交 Git。

---

## 第 0 步：准备工具与账号

| 工具 | 安装 | 验收 |
|------|------|------|
| [Neon](https://neon.tech) | 注册，创建 Project | 控制台能看到 Connection string |
| [Fly.io](https://fly.io) | `curl -L https://fly.io/install.sh \| sh` | `fly version` |
| [Vercel](https://vercel.com) | 注册，连接 GitHub | 能 Import 仓库 |
| 本机 | Go 20+、pnpm 9+、psql 或 Docker | 仓库根目录 `pnpm install` |

```bash
fly auth login
```

在 Fly 添加支付方式（免费额度通常也要绑卡）。10 个微服务 = 10 个 Fly App，可先走 **最小集（6 个）** 再补全。

---

## 第 1 步：Neon 数据库

1. Neon 控制台 → **New Project** → 选区域（离 Fly 近：`aws-ap-southeast-1` 对应 Fly `sin`）。
2. 打开 **Connection details** → 选 **URI**。
3. 复制连接串，形如：

```text
postgresql://user:pass@ep-xxxx.ap-southeast-1.aws.neon.tech/neondb?sslmode=require
```

4. 在本机仓库根目录创建 **不要提交 Git** 的文件 `.env.deploy`：

```bash
# .env.deploy — 仅本地使用，已加入 .gitignore
DATABASE_URL='postgresql://USER:PASS@ep-xxx.region.aws.neon.tech/neondb?sslmode=require'
```

5. 验收（能连上即可）：

```bash
psql "$DATABASE_URL" -c 'SELECT 1'
```

没有本机 `psql` 也没关系：`./scripts/migrate.sh` 会自动用 **Docker** 里的 `postgres:16-alpine` 连接 Neon（需已安装 Docker Desktop）。

---

## 第 2 步：跑数据库迁移

在仓库根目录：

```bash
chmod +x scripts/migrate.sh
set -a && source .env.deploy && set +a
./scripts/migrate.sh
```

验收：

```bash
psql "$DATABASE_URL" -c "SELECT name FROM symbols ORDER BY name;"
```

应看到 10 个交易对（`BTC/USDT` … `DOT/USDT`）。

---

## 第 3 步：在 Fly 创建 App（只需一次）

在仓库根目录执行（app 名与 `infra/deploy/fly/*.toml` 一致）：

```bash
# 最小集（CEX + 永续，约 6 个）— 建议第一天先这些
for app in cel-account cel-matching cel-order cel-hyperliquid cel-risk cel-liquidation cel-funding; do
  fly apps create "$app" || true
done

# 完整集（再加 DEX + Explorer 链）
for app in cel-orderbook-dex cel-rpc-gateway cel-indexer; do
  fly apps create "$app" || true
done
```

`|| true` 表示已存在则跳过。

---

## 第 4 步：部署 Fly 后端（严格按顺序）

先把下面变量换成 **你真实的 Fly 地址**（部署完前两步后就有）：

```bash
# 写入 .env.deploy（示例域名，部署后以 fly apps list 为准）
export ACCOUNT_URL=https://cel-account.fly.dev
export MATCHING_URL=https://cel-matching.fly.dev
export HYPER_URL=https://cel-hyperliquid.fly.dev
```

### 4.1 account-service

```bash
source .env.deploy

fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  APP_ENV=production \
  LOG_LEVEL=info \
  -a cel-account

fly deploy . -c infra/deploy/fly/account-service.toml -a cel-account
```

验收：

```bash
curl -s https://cel-account.fly.dev/health
```

### 4.2 matching-engine

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  MATCHING_SEED_BOOKS=true \
  APP_ENV=production \
  -a cel-matching

fly deploy . -c infra/deploy/fly/matching-engine.toml -a cel-matching
```

验收：

```bash
curl -s https://cel-matching.fly.dev/health
curl -s https://cel-matching.fly.dev/api/v1/venues | head -c 200
```

把 `MATCHING_URL` 更新进 `.env.deploy`。

### 4.3 order-service（CEX）

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" \
  MATCHING_ENGINE_URL="$MATCHING_URL" \
  APP_ENV=production \
  -a cel-order

fly deploy . -c infra/deploy/fly/order-service.toml -a cel-order
```

验收：

```bash
curl -s https://cel-order.fly.dev/health
curl -s https://cel-order.fly.dev/api/v1/symbols
```

### 4.4 orderbook-dex（可选，DEX 订单簿）

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" \
  MATCHING_ENGINE_URL="$MATCHING_URL" \
  -a cel-orderbook-dex

fly deploy . -c infra/deploy/fly/orderbook-dex.toml -a cel-orderbook-dex
```

### 4.5 hyperliquid-engine

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" \
  MATCHING_ENGINE_URL="$MATCHING_URL" \
  -a cel-hyperliquid

fly deploy . -c infra/deploy/fly/hyperliquid-engine.toml -a cel-hyperliquid
```

### 4.6 risk-engine

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" \
  -a cel-risk

fly deploy . -c infra/deploy/fly/risk-engine.toml -a cel-risk
```

### 4.7 liquidation-engine

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  HYPERLIQUID_ENGINE_URL="$HYPER_URL" \
  -a cel-liquidation

fly deploy . -c infra/deploy/fly/liquidation-engine.toml -a cel-liquidation
```

### 4.8 funding-engine

```bash
fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" \
  FUNDING_INTERVAL=5m \
  FUNDING_RATE=0.0001 \
  -a cel-funding

fly deploy . -c infra/deploy/fly/funding-engine.toml -a cel-funding
```

### 4.9 rpc-gateway + indexer（Explorer 需要）

```bash
fly secrets set DATABASE_URL="$DATABASE_URL" -a cel-rpc-gateway
fly deploy . -c infra/deploy/fly/rpc-gateway.toml -a cel-rpc-gateway

fly secrets set \
  DATABASE_URL="$DATABASE_URL" \
  SEPOLIA_RPC_URL=https://rpc.sepolia.org \
  INDEXER_CHAIN=sepolia \
  -a cel-indexer
fly deploy . -c infra/deploy/fly/indexer.toml -a cel-indexer
```

**一键脚本（填好 `.env.deploy` 后）：**

```bash
./scripts/fly-deploy-minimal.sh   # 最小 7 服务
# 或
./scripts/fly-deploy-all.sh       # 全部 10 服务
```

---

## 第 5 步：Vercel 部署 4 个前端

每个 **独立 Vercel Project**，Root Directory 指向对应 `apps/*`（仓库已含 `vercel.json`）。

### 5.1 导入仓库

Vercel → **Add New Project** → 选 `crypto-exchange-lab` 仓库。

### 5.2 环境变量（Production）

用 Fly 的 **https** 地址。在 **每个** 项目里设置需要的变量：

**cex-web**（Root: `apps/cex-web`）

| 变量 | 示例值 |
|------|--------|
| `NEXT_PUBLIC_ACCOUNT_API_URL` | `https://cel-account.fly.dev` |
| `NEXT_PUBLIC_ORDER_API_URL` | `https://cel-order.fly.dev` |
| `NEXT_PUBLIC_MATCHING_API_URL` | `https://cel-matching.fly.dev` |
| `NEXT_PUBLIC_MATCHING_WS_URL` | `wss://cel-matching.fly.dev/ws/v1/market` |

**dex-web**（Root: `apps/dex-web`）

| 变量 | 示例值 |
|------|--------|
| `NEXT_PUBLIC_ORDERBOOK_DEX_API_URL` | `https://cel-orderbook-dex.fly.dev` |
| `NEXT_PUBLIC_SEPOLIA_RPC_URL` | `https://rpc.sepolia.org` |
| `NEXT_PUBLIC_LAB_TOKEN` 等 | Sepolia 部署后填写（见 [amm-dex.md](./amm-dex.md)） |

**hyperdex-web**（Root: `apps/hyperdex-web`）

| 变量 | 示例值 |
|------|--------|
| `NEXT_PUBLIC_HYPERLIQUID_API_URL` | `https://cel-hyperliquid.fly.dev` |
| `NEXT_PUBLIC_RISK_API_URL` | `https://cel-risk.fly.dev` |
| `NEXT_PUBLIC_FUNDING_API_URL` | `https://cel-funding.fly.dev` |

**explorer-web**（Root: `apps/explorer-web`）

| 变量 | 示例值 |
|------|--------|
| `NEXT_PUBLIC_RPC_GATEWAY_URL` | `https://cel-rpc-gateway.fly.dev` |

生成本地模板：

```bash
./scripts/print-vercel-env-fly.sh
```

会输出 4 个前端各自需要的 `NEXT_PUBLIC_*`（默认 `cel-*.fly.dev` 主机名）。

### 5.3 部署

每个项目点 **Deploy**。记下 4 个 URL，例如：

- `https://crypto-exchange-lab-cex.vercel.app`
- …

---

## 第 6 步：端到端验收

| 检查 | 操作 |
|------|------|
| CEX | 打开 cex-web → 选 alice → 看 10 个 symbol → 下单 |
| 深度 | 订单簿有 bid/ask（matching seed） |
| HyperDEX | 开仓/平仓一条 perp |
| Explorer | 有链数据需 indexer 跑一段时间；至少 `/health` 正常 |

---

## 第 7 步：更新 README Live demo

把真实链接写进根目录 `README.md`：

```markdown
**Live demo:** [CEX](https://xxx.vercel.app) · [DEX](https://yyy.vercel.app) · [HyperDEX](https://zzz.vercel.app) · [Explorer](https://www.vercel.app)
```

---

## 常见问题

**Fly build 很慢**  
第一次会拉 Go 依赖，正常。失败时本地先跑：  
`docker build -f infra/deploy/Dockerfile.go-service --build-arg SERVICE=account-service .`

**Vercel 调 API 失败**  
检查是否用了 `https://` 的 Fly 地址；浏览器控制台看 CORS（后端已 `Access-Control-Allow-Origin: *`）。

**Neon 连接失败**  
连接串必须带 `sslmode=require`；密码含特殊字符需 URL 编码。

**WebSocket 连不上**  
确认 `NEXT_PUBLIC_MATCHING_WS_URL` 是 `wss://cel-matching.fly.dev/...`，不是 `https://`。

**控制成本**  
Fly 可设 `min_machines_running = 0`（已在 toml），冷启动第一次请求会慢几秒。

---

## 推荐第一天范围

| 必做 | 服务 |
|------|------|
| ✅ | account, matching, order, hyperliquid, risk, liquidation, funding |
| 可选 | orderbook-dex, rpc-gateway, indexer |
| 可选 | Sepolia AMM 合约 + dex-web AMM 环境变量 |

完成最小集即可在简历写：**模拟 CEX + 永续风控链 + Vercel 公网 demo**。
