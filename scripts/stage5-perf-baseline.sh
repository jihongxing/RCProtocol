#!/bin/bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8081}"
VERIFY_UID="${VERIFY_UID:-04A31B2C3D4E5F}"
VERIFY_CTR="${VERIFY_CTR:-010000}"
VERIFY_CMAC="${VERIFY_CMAC:-0000000000000000}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
DATABASE_URL="${DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol}"
TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres}"
REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
COUNT="${COUNT:-5}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ 缺少命令: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd python

measure_get() {
  local url="$1"
  local count="$2"
  python - "$url" "$count" <<'PY'
import statistics
import sys
import time
import urllib.request

url = sys.argv[1]
count = int(sys.argv[2])
values = []
for _ in range(count):
    start = time.perf_counter()
    with urllib.request.urlopen(url) as resp:
        resp.read()
    values.append((time.perf_counter() - start) * 1000)
values_sorted = sorted(values)
p95_idx = max(0, min(len(values_sorted) - 1, int(len(values_sorted) * 0.95) - 1))
print(f"count={count} avg_ms={statistics.mean(values):.2f} p95_ms={values_sorted[p95_idx]:.2f}")
PY
}

step() {
  echo
  echo "==> $1"
}

step "健康检查性能样本"
measure_get "$API_BASE/healthz" "$COUNT"

step "verify 性能样本"
measure_get "$API_BASE/verify?uid=$VERIFY_UID&ctr=$VERIFY_CTR&cmac=$VERIFY_CMAC" "$COUNT"

step "verify v2 性能样本"
measure_get "$API_BASE/verify/v2?uid=$VERIFY_UID&ctr=$VERIFY_CTR&cmac=$VERIFY_CMAC" "$COUNT"

step "activation_integration 耗时观测"
cd "$ROOT_DIR/rust/rc-api"
time RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" REDIS_URL="$REDIS_URL" cargo test --test activation_integration

step "transfer_integration 耗时观测"
time RC_JWT_SECRET="$RC_JWT_SECRET" DATABASE_URL="$DATABASE_URL" TEST_DATABASE_URL="$TEST_DATABASE_URL" REDIS_URL="$REDIS_URL" cargo test --test transfer_integration

echo

echo "✅ Stage 5 性能基线采集完成，请将结果回填 docs/ops/stage-5-performance-baseline.md"
