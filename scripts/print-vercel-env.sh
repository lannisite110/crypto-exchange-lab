#!/usr/bin/env bash
# Print NEXT_PUBLIC_* block for Vercel from a public API base (no trailing slash).
# Example: ./scripts/print-vercel-env.sh https://api.yourdomain.com
# With path-based reverse proxy, set per-service URLs manually (see docs/deploy.md).
set -euo pipefail
BASE="${1:-}"
if [[ -z "$BASE" ]]; then
  echo "Usage: $0 <public-api-base>" >&2
  echo "  Per-service (Fly): run once per host, e.g. $0 https://cel-order.fly.dev" >&2
  exit 1
fi
BASE="${BASE%/}"
WS_BASE="${BASE/http:/ws:}"
WS_BASE="${WS_BASE/https:/wss:}"

cat <<EOF
# Paste into Vercel project env (adjust per app)

# cex-web
NEXT_PUBLIC_ACCOUNT_API_URL=${BASE}:8081
NEXT_PUBLIC_ORDER_API_URL=${BASE}:8082
NEXT_PUBLIC_MATCHING_API_URL=${BASE}:8083
NEXT_PUBLIC_MATCHING_WS_URL=${WS_BASE}:8083/ws/v1/market

# dex-web (OrderBook tab — use orderbook host if split on Fly)
NEXT_PUBLIC_ORDERBOOK_DEX_API_URL=${BASE}:8084

# hyperdex-web
NEXT_PUBLIC_HYPERLIQUID_API_URL=${BASE}:8085
NEXT_PUBLIC_RISK_API_URL=${BASE}:8086
NEXT_PUBLIC_FUNDING_API_URL=${BASE}:8088

# explorer-web
NEXT_PUBLIC_RPC_GATEWAY_URL=${BASE}:8089

# dex-web AMM (after Sepolia deploy)
# NEXT_PUBLIC_SEPOLIA_RPC_URL=https://rpc.sepolia.org
# NEXT_PUBLIC_LAB_TOKEN=0x...
EOF
