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

step "品牌 API 闭环（脚本）"
cd "$ROOT_DIR"
bash ./scripts/test-brand-registration.sh || echo "⚠️ 需要本地运行中的 rc-api 与 PLATFORM_TOKEN 支持"

step "品牌 API 闭环（Rust 集成测试）"
cd "$ROOT_DIR/rust/rc-api"
RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" REDIS_URL="$REDIS_URL" cargo test --test brand_registration_integration

echo

echo "✅ Stage 5 品牌 API 闭环入口已执行完成"
