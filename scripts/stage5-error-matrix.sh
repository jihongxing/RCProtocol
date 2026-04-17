#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
DATABASE_URL="${DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol}"
TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres}"
REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

step() {
  echo
  echo "==> $1"
}

run_test() {
  local test_name="$1"
  step "cargo test --test $test_name"
  cd "$ROOT_DIR/rust/rc-api"
  RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" REDIS_URL="$REDIS_URL" cargo test --test "$test_name"
}

step "Stage 5 异常流矩阵回归"
run_test verify_integration
run_test brand_registration_integration
run_test transfer_actions_integration
run_test protocol_write_flow_integration

echo
echo "✅ Stage 5 异常流矩阵回归入口已执行完成"
