#!/usr/bin/env bash
# Start full backend demo stack (Docker). Frontends: deploy to Vercel or run pnpm dev locally.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/infra/docker/docker-compose.yml -f $ROOT/infra/docker/docker-compose.demo.yml"

echo "Starting Postgres + Redis + Go services..."
$COMPOSE up -d --build

echo "Waiting for Postgres..."
sleep 5
$COMPOSE ps

echo ""
echo "Backend APIs (host ports):"
echo "  account-service   http://localhost:8081"
echo "  order-service     http://localhost:8082  (CEX)"
echo "  matching-engine   http://localhost:8083"
echo "  orderbook-dex     http://localhost:8084"
echo "  hyperliquid       http://localhost:8085"
echo "  risk-engine       http://localhost:8086"
echo "  liquidation       http://localhost:8087"
echo "  funding-engine    http://localhost:8088"
echo "  rpc-gateway       http://localhost:8089"
echo "  indexer           http://localhost:8090"
echo ""
echo "Next: point Vercel env at these URLs (HTTPS required for production — use Fly per deploy.md)."
echo "  ./scripts/print-vercel-env.sh http://YOUR_PUBLIC_HOST"
echo ""
echo "Local frontends: pnpm dev in apps/*-web"
