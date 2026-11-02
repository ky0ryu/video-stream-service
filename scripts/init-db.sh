#!/usr/bin/env bash
set -euo pipefail

echo "initializing db schema..."

PROJ_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
goose -dir "$PROJ_ROOT/dbMigration" create init_schema sql
