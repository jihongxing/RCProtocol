#!/bin/bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080/api}"
FRONTEND_BASE="${FRONTEND_BASE:-http://localhost:5173}"
RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
DATABASE_URL="${DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5433/rcprotocol}"
TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5433/postgres}"

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

print_step() {
  echo
  echo "==> $1"
}

print_step "环境预检查"
echo "API_BASE=$API_BASE"
echo "FRONTEND_BASE=$FRONTEND_BASE"
echo "DATABASE_URL=$DATABASE_URL"
echo "TEST_DATABASE_URL=$TEST_DATABASE_URL"

print_step "运行前端关键页面测试"
cd "$ROOT_DIR/frontend"
pnpm vitest run apps/c-app/src/pages/vault.transfer.test.ts apps/c-app/src/pages/vault.transfer-confirm.test.ts

print_step "运行 Go BFF 联调测试"
cd "$ROOT_DIR/services/go-bff"
go test ./internal/handler ./internal/router

print_step "运行 Rust transfer 集成测试"
cd "$ROOT_DIR/rust/rc-api"
RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" cargo test --test transfer_integration --test transfer_actions_integration

print_step "可选本地演示提示"
echo "1. 启动依赖：docker compose -f deploy/compose/docker-compose.yml up -d postgres redis"
echo "2. 启动服务：pwsh ./scripts/start-api.ps1"
echo "3. 启动前端：cd frontend && pnpm dev:c-app"
echo "4. 打开页面：$FRONTEND_BASE"

echo
echo "✅ transfer 联调、页面、异常流测试已完成"
