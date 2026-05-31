#!/usr/bin/env bash
# Deploy minimal Fly stack (CEX + perps). Requires .env.deploy with DATABASE_URL.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env.deploy ]]; then
  echo "Create .env.deploy with DATABASE_URL=... (see docs/deploy-neon-fly-vercel.md)" >&2
  exit 1
fi
set -a
# shellcheck source=/dev/null
source .env.deploy
set +a

: "${DATABASE_URL:?Set DATABASE_URL in .env.deploy}"

ACCOUNT_URL="${ACCOUNT_URL:-https://cel-account.fly.dev}"
MATCHING_URL="${MATCHING_URL:-https://cel-matching.fly.dev}"
HYPER_URL="${HYPER_URL:-https://cel-hyperliquid.fly.dev}"

deploy_one() {
  local app="$1"
  local cfg="$2"
  shift 2
  echo "========== $app =========="
  fly secrets set DATABASE_URL="$DATABASE_URL" "$@" -a "$app"
  # Repo root = build context; dockerfile relative to fly.toml dir (see infra/deploy/fly/*.toml)
  fly deploy . \
    --dockerfile infra/deploy/Dockerfile.go-service \
    -c "$cfg" \
    -a "$app"
  echo ""
}

deploy_one cel-account infra/deploy/fly/account-service.toml APP_ENV=production LOG_LEVEL=info
deploy_one cel-matching infra/deploy/fly/matching-engine.toml \
  MATCHING_SEED_BOOKS=true APP_ENV=production
deploy_one cel-order infra/deploy/fly/order-service.toml \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" MATCHING_ENGINE_URL="$MATCHING_URL" APP_ENV=production
deploy_one cel-hyperliquid infra/deploy/fly/hyperliquid-engine.toml \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" MATCHING_ENGINE_URL="$MATCHING_URL" APP_ENV=production
deploy_one cel-risk infra/deploy/fly/risk-engine.toml \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" APP_ENV=production
deploy_one cel-liquidation infra/deploy/fly/liquidation-engine.toml \
  HYPERLIQUID_ENGINE_URL="$HYPER_URL" APP_ENV=production
deploy_one cel-funding infra/deploy/fly/funding-engine.toml \
  ACCOUNT_SERVICE_URL="$ACCOUNT_URL" FUNDING_INTERVAL=5m FUNDING_RATE=0.0001 APP_ENV=production

echo "Done. Verify:"
echo "  curl -s $ACCOUNT_URL/health"
echo "  curl -s $MATCHING_URL/api/v1/venues"
echo "  curl -s https://cel-order.fly.dev/api/v1/symbols"
echo ""
echo "Next: Vercel env — docs/deploy-neon-fly-vercel.md step 5"
