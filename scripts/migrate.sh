#!/usr/bin/env bash
# Apply SQL migrations in order. Uses local psql, or Docker postgres image if psql is missing.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-postgres://lab:lab@localhost:5432/crypto_exchange_lab?sslmode=disable}"

run_psql_file() {
  local f="$1"
  if command -v psql >/dev/null 2>&1; then
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
    return
  fi
  if command -v docker >/dev/null 2>&1; then
    docker run --rm -i postgres:16-alpine \
      psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <"$f"
    return
  fi
  echo "psql not found and docker unavailable." >&2
  echo "  Install: sudo apt install -y postgresql-client" >&2
  echo "  Or local stack: make up && make migrate" >&2
  exit 1
}

for f in "$ROOT"/infra/postgres/migrations/*.up.sql; do
  echo "==> $(basename "$f")"
  run_psql_file "$f"
done
echo "Migrations complete."
