#!/usr/bin/env bash
set -euo pipefail

echo "migrating db..."

PROJ_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# credentials live in the gitignored .env (see .env.example)
set -a
. "$PROJ_ROOT/.env"
set +a

: "${DB_HOST:?set in .env}" "${DB_USER:?set in .env}" "${DB_NAME:?set in .env}" "${DB_PASSWORD:?set in .env}"

# assign the pgx engine password variable
export PGPASSWORD="$DB_PASSWORD"
# The PASSWORD stays in the environment for the driver to pick up, so it is not
# copied into this script's arguments (where it would show up in `ps`).
goose -dir "$PROJ_ROOT/dbMigration" postgres \
  "host=${DB_HOST} port=${DB_PORT:-5432} user=${DB_USER} dbname=${DB_NAME} sslmode=${DB_SSLMODE:-disable}" up
