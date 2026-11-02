#!/usr/bin/env bash
set -euo pipefail

echo "initializing sqlc..."

PROJ_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sqlc generate -f "$PROJ_ROOT/sqlc/sqlc.yaml"
