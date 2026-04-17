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

step "Stage 5 主链路环境信息"
echo "DATABASE_URL=$DATABASE_URL"
echo "TEST_DATABASE_URL=$TEST_DATABASE_URL"
echo "REDIS_URL=$REDIS_URL"

action_cargo_test() {
  local test_name="$1"
  step "cargo test --test $test_name"
  cd "$ROOT_DIR/rust/rc-api"
  RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" REDIS_URL="$REDIS_URL" cargo test --test "$test_name"
}

step "运行品牌接入闭环测试"
cd "$ROOT_DIR"
bash ./scripts/test-brand-registration.sh || echo "⚠️ 品牌脚本依赖本地运行中的 rc-api，若当前仅做测试基线可稍后单独执行"

action_cargo_test activation_integration
action_cargo_test transfer_integration
action_cargo_test transfer_actions_integration
action_cargo_test verify_integration

step "可选前端关键页面测试"
cd "$ROOT_DIR/frontend"
pnpm vitest run apps/c-app/src/pages/vault.transfer.test.ts apps/c-app/src/pages/vault.transfer-confirm.test.ts || echo "⚠️ 前端测试未通过，请单独排查"

echo
echo "✅ Stage 5 主链路测试入口已执行完成"
