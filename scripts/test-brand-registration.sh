#!/bin/bash
# 品牌注册与 API Key 管理联调脚本
# 覆盖：品牌注册、邮箱唯一性、Key 轮换、旧 Key 失效、新 Key 生效、Key 列表查询

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PLATFORM_TOKEN="${PLATFORM_TOKEN:-}"

if [ -z "$PLATFORM_TOKEN" ]; then
  if [ -x "$SCRIPT_DIR/generate-platform-token.sh" ]; then
    PLATFORM_TOKEN="$($SCRIPT_DIR/generate-platform-token.sh | tail -1)"
  else
    echo "❌ PLATFORM_TOKEN 未设置，且 generate-platform-token.sh 不可执行"
    exit 1
  fi
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ 缺少命令: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd jq

EMAIL_SUFFIX="$(date +%s)"
CONTACT_EMAIL="brand-$EMAIL_SUFFIX@example.com"

json_post() {
  local url="$1"
  local auth_header="$2"
  local payload="$3"
  curl -sS -X POST "$url" \
    -H "$auth_header" \
    -H "Content-Type: application/json" \
    -d "$payload"
}

echo "=========================================="
echo "品牌注册与 API Key 管理测试"
echo "BASE_URL=$BASE_URL"
echo "=========================================="
echo

echo "1) 注册新品牌"
REGISTER_RESPONSE=$(json_post "$BASE_URL/brands" "Authorization: Bearer $PLATFORM_TOKEN" "{
  \"brand_name\": \"Brand Registration Test\",
  \"contact_email\": \"$CONTACT_EMAIL\",
  \"industry\": \"Watches\"
}")

echo "$REGISTER_RESPONSE" | jq '.'
BRAND_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.brand_id')
API_KEY=$(echo "$REGISTER_RESPONSE" | jq -r '.api_key.api_key')

if [ -z "$BRAND_ID" ] || [ "$BRAND_ID" = "null" ] || [ -z "$API_KEY" ] || [ "$API_KEY" = "null" ]; then
  echo "❌ 品牌注册失败"
  exit 1
fi

echo
echo "2) 校验品牌详情查询（不返回 api_key）"
DETAIL_RESPONSE=$(curl -sS -X GET "$BASE_URL/brands/$BRAND_ID" -H "X-Api-Key: $API_KEY")
echo "$DETAIL_RESPONSE" | jq '.'
if echo "$DETAIL_RESPONSE" | jq -e '.api_key' >/dev/null; then
  echo "❌ 品牌详情不应返回 api_key"
  exit 1
fi

echo
echo "3) 校验邮箱唯一性"
DUPLICATE_HTTP=$(curl -sS -o /tmp/brand-duplicate.out -w "%{http_code}" \
  -X POST "$BASE_URL/brands" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"brand_name\": \"Duplicate Brand\",
    \"contact_email\": \"$CONTACT_EMAIL\",
    \"industry\": \"Fashion\"
  }")
cat /tmp/brand-duplicate.out | jq '.' || true
if [ "$DUPLICATE_HTTP" != "409" ]; then
  echo "❌ 邮箱唯一性校验失败，HTTP=$DUPLICATE_HTTP"
  exit 1
fi

echo
echo "4) 轮换 API Key"
ROTATE_RESPONSE=$(json_post "$BASE_URL/brands/$BRAND_ID/api-keys/rotate" "X-Api-Key: $API_KEY" '{"reason":"manual rotation test"}')
echo "$ROTATE_RESPONSE" | jq '.'
NEW_API_KEY=$(echo "$ROTATE_RESPONSE" | jq -r '.api_key')
REVOKED_KEY_ID=$(echo "$ROTATE_RESPONSE" | jq -r '.revoked_key_id')

if [ -z "$NEW_API_KEY" ] || [ "$NEW_API_KEY" = "null" ] || [ -z "$REVOKED_KEY_ID" ] || [ "$REVOKED_KEY_ID" = "null" ]; then
  echo "❌ API Key 轮换失败"
  exit 1
fi

echo
echo "5) 验证旧 Key 已失效"
OLD_KEY_HTTP=$(curl -sS -o /tmp/brand-old-key.out -w "%{http_code}" -X GET "$BASE_URL/brands/$BRAND_ID" -H "X-Api-Key: $API_KEY")
cat /tmp/brand-old-key.out | jq '.' || true
if [ "$OLD_KEY_HTTP" != "401" ]; then
  echo "❌ 旧 Key 未失效，HTTP=$OLD_KEY_HTTP"
  exit 1
fi

echo
echo "6) 验证新 Key 可用"
NEW_DETAIL_HTTP=$(curl -sS -o /tmp/brand-new-key.out -w "%{http_code}" -X GET "$BASE_URL/brands/$BRAND_ID" -H "X-Api-Key: $NEW_API_KEY")
cat /tmp/brand-new-key.out | jq '.'
if [ "$NEW_DETAIL_HTTP" != "200" ]; then
  echo "❌ 新 Key 不可用，HTTP=$NEW_DETAIL_HTTP"
  exit 1
fi

echo
echo "7) 查询 API Key 列表"
KEY_LIST_RESPONSE=$(curl -sS -X GET "$BASE_URL/brands/$BRAND_ID/api-keys" -H "X-Api-Key: $NEW_API_KEY")
echo "$KEY_LIST_RESPONSE" | jq '.'
ACTIVE_COUNT=$(echo "$KEY_LIST_RESPONSE" | jq '[.keys[] | select(.status == "Active")] | length')
REVOKED_COUNT=$(echo "$KEY_LIST_RESPONSE" | jq '[.keys[] | select(.status == "Revoked")] | length')
if [ "$ACTIVE_COUNT" != "1" ] || [ "$REVOKED_COUNT" != "1" ]; then
  echo "❌ API Key 列表状态不符合预期"
  exit 1
fi

echo
echo "8) 测试权限校验（Brand Key 不允许注册新品牌）"
FORBIDDEN_HTTP=$(curl -sS -o /tmp/brand-forbidden.out -w "%{http_code}" \
  -X POST "$BASE_URL/brands" \
  -H "X-Api-Key: $NEW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"brand_name":"Unauthorized","contact_email":"unauthorized@example.com","industry":"Jewelry"}')
cat /tmp/brand-forbidden.out | jq '.' || true
if [ "$FORBIDDEN_HTTP" != "403" ]; then
  echo "❌ 权限校验失败，HTTP=$FORBIDDEN_HTTP"
  exit 1
fi

echo
echo "✅ 品牌注册与 API Key 管理测试通过"
