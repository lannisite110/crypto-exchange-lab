#!/usr/bin/env bash
# Print NEXT_PUBLIC_* for Vercel when each API is a separate Fly app (default cel-* names).
set -euo pipefail
A="${ACCOUNT_HOST:-https://cel-account.fly.dev}"
O="${ORDER_HOST:-https://cel-order.fly.dev}"
M="${MATCHING_HOST:-https://cel-matching.fly.dev}"
D="${DEX_HOST:-https://cel-orderbook-dex.fly.dev}"
H="${HYPER_HOST:-https://cel-hyperliquid.fly.dev}"
R="${RISK_HOST:-https://cel-risk.fly.dev}"
F="${FUNDING_HOST:-https://cel-funding.fly.dev}"
G="${RPC_HOST:-https://cel-rpc-gateway.fly.dev}"

for u in "$A" "$O" "$M" "$D" "$H" "$R" "$F" "$G"; do
  [[ "$u" == */ ]] && continue
done

WS="${M/https:/wss:}/ws/v1/market"

cat <<EOF
# cex-web
NEXT_PUBLIC_ACCOUNT_API_URL=$A
NEXT_PUBLIC_ORDER_API_URL=$O
NEXT_PUBLIC_MATCHING_API_URL=$M
NEXT_PUBLIC_MATCHING_WS_URL=$WS

# dex-web
NEXT_PUBLIC_ORDERBOOK_DEX_API_URL=$D

# hyperdex-web
NEXT_PUBLIC_HYPERLIQUID_API_URL=$H
NEXT_PUBLIC_RISK_API_URL=$R
NEXT_PUBLIC_FUNDING_API_URL=$F

# explorer-web
NEXT_PUBLIC_RPC_GATEWAY_URL=$G
EOF
