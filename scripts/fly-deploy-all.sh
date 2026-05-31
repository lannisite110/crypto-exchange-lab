#!/usr/bin/env bash
# Deploy all 10 Fly apps. Run fly-deploy-minimal.sh first or ensure apps exist.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
"$ROOT/scripts/fly-deploy-minimal.sh"

if [[ ! -f .env.deploy ]]; then
  exit 1
fi
set -a
# shellcheck source=/dev/null
source .env.deploy
set +a

ACCOUNT_URL="${ACCOUNT_URL:-https://cel-account.fly.dev}"
MATCHING_URL="${MATCHING_URL:-https://cel-matching.fly.dev}"
HYPER_URL="${HYPER_URL:-https://cel-hyperliquid.fly.dev}"

deploy_one() {
  local app="$1"
  local cfg="$2"
  shift 2
  echo "========== $app =========="
  fly secrets set DATABASE_URL="$DATABASE_URL" "$@" -a "$app"
  fly deploy . \
    --dockerfile infra/deploy/Dockerfile.go-service \
    -c "$cfg" \
    -a "$app"
}

deploy_one cel-orderbook-dex infra/deploy/fly/orderbook-dex.toml \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" MATCHING_ENGINE_URL="$MATCHING_URL" APP_ENV=production
deploy_one cel-rpc-gateway infra/deploy/fly/rpc-gateway.toml APP_ENV=production
deploy_one cel-indexer infra/deploy/fly/indexer.toml \
  SEPOLIA_RPC_URL="${SEPOLIA_RPC_URL:-https://rpc.sepolia.org}" \
  INDEXER_CHAIN=sepolia APP_ENV=production

echo "All Fly apps deployed."
