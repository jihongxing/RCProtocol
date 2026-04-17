#!/bin/bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8081}"
UID="${UID:-04A31B2C3D4E5F}"
CTR="${CTR:-010000}"
CMAC="${CMAC:-0000000000000000}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ 缺少命令: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd jq

echo "=========================================="
echo "Verification V2 联调脚本"
echo "API_BASE=$API_BASE"
echo "UID=$UID CTR=$CTR"
echo "=========================================="
echo

echo "1) 请求 V1 /verify"
V1=$(curl -sS "$API_BASE/verify?uid=$UID&ctr=$CTR&cmac=$CMAC")
echo "$V1" | jq '.'
echo

echo "2) 请求 V2 /verify/v2"
V2=$(curl -sS "$API_BASE/verify/v2?uid=$UID&ctr=$CTR&cmac=$CMAC")
echo "$V2" | jq '.'
echo

echo "✅ 已打印 V1 / V2 完整结构化响应"
