#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/../rust/rc-api"

export RC_ROOT_KEY_HEX="${RC_ROOT_KEY_HEX:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
export RC_SYSTEM_ID="${RC_SYSTEM_ID:-rcprotocol-dev}"
export RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
export RC_API_KEY_SECRET="${RC_API_KEY_SECRET:-rc-dev-api-key-secret-do-not-use-in-prod}"
export DATABASE_URL="${DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol}"
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

echo "环境变量已设置:"
echo "  DATABASE_URL=$DATABASE_URL"
echo "  TEST_DATABASE_URL=$TEST_DATABASE_URL"
echo "  REDIS_URL=$REDIS_URL"
echo

echo "启动 rc-api ..."
cargo run --release
