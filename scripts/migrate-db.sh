#!/usr/bin/env bash
set -euo pipefail

echo "migrating db..."

PROJ_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
goose -dir "$PROJ_ROOT/dbMigration" postgres "host=127.0.0.1 port=5432 user=user password=password dbname=video_db sslmode=disable" up
