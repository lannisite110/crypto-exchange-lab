#!/usr/bin/env bash
# One-time Fly.io setup helper (run from repo root). Requires flyctl and user login.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v fly >/dev/null 2>&1; then
  echo "Install flyctl: https://fly.io/docs/hands-on/install-flyctl/" >&2
  exit 1
fi

SERVICES=(
  account-service:8081:cel-account
  matching-engine:8083:cel-matching
  order-service:8082:cel-order
  orderbook-dex:8084:cel-orderbook-dex
  hyperliquid-engine:8085:cel-hyperliquid
  risk-engine:8086:cel-risk
  liquidation-engine:8087:cel-liquidation
  funding-engine:8088:cel-funding
  rpc-gateway:8089:cel-rpc-gateway
  indexer:8090:cel-indexer
)

echo "Create Fly Postgres (once):"
echo "  fly postgres create --name cel-db --region sin"
echo "  fly postgres attach cel-db -a cel-account"
echo ""
echo "Per service (example account-service):"
echo "  fly apps create cel-account --org personal  # if not exists"
echo "  fly secrets set DATABASE_URL='...' -a cel-account"
echo "  fly deploy . -c infra/deploy/fly/account-service.toml -a cel-account"
echo ""
echo "Service map:"
for entry in "${SERVICES[@]}"; do
  IFS=: read -r svc port app <<< "$entry"
  echo "  $app  :$port  ($svc)  -> infra/deploy/fly/${svc}.toml"
done
echo ""
echo "After deploy, set Vercel env (HTTPS URLs):"
echo "  NEXT_PUBLIC_ACCOUNT_API_URL=https://cel-account.fly.dev"
echo "  NEXT_PUBLIC_ORDER_API_URL=https://cel-order.fly.dev"
echo "  ..."
