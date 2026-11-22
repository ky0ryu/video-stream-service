#!/usr/bin/env bash
set -euo pipefail

echo "migrating db..."

PROJ_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
set -a
. "$(dirname "$0")/../.env"
set +a
goose -dir "$PROJ_ROOT/dbMigration" postgres "host=127.0.0.1 port=5432 user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=video_db sslmode=disable" up
