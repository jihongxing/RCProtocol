#!/bin/bash
set -e

API_BASE="http://localhost:8081"

# UUID 生成函数（兼容 Windows Git Bash）
generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen
    elif [ -f /proc/sys/kernel/random/uuid ]; then
        cat /proc/sys/kernel/random/uuid
    else
        # 使用 openssl 生成随机 UUID
        printf '%08x-%04x-%04x-%04x-%012x\n' \
            $((RANDOM * RANDOM)) \
            $((RANDOM % 65536)) \
            $(((RANDOM % 4096) + 16384)) \
            $(((RANDOM % 16384) + 32768)) \
            $((RANDOM * RANDOM * RANDOM))
    fi
}

# 生成 JWT token 的辅助函数
generate_jwt() {
    local role=$1
    local actor_id=$2
    local brand_id=$3

    cd "D:/codeSpace/RCProtocol/rust/rc-api"
    if [ -n "$brand_id" ]; then
        cargo run --example generate_jwt -- "$role" "$actor_id" "$brand_id" 2>/dev/null | tail -1
    else
        cargo run --example generate_jwt -- "$role" "$actor_id" 2>/dev/null | tail -1
    fi
}

echo "=========================================="
echo "批次管理集成测试"
echo "=========================================="
echo ""

# 1. 生成 Platform JWT Token
echo "[1/8] 生成 Platform JWT Token..."
PLATFORM_TOKEN=$(generate_jwt "platform" "platform-system" "")
echo "✓ Platform Token: ${PLATFORM_TOKEN:0:50}..."
echo ""

# 2. 注册测试品牌
echo "[2/8] 注册测试品牌..."
TIMESTAMP=$(date +%s)
BRAND_RESPONSE=$(curl -s -X POST "$API_BASE/brands" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "brand_name": "TestBrand-Batch-'"$TIMESTAMP"'",
    "contact_email": "batch-test-'"$TIMESTAMP"'@example.com",
    "industry": "Watches"
  }')

BRAND_ID=$(echo "$BRAND_RESPONSE" | grep -o '"brand_id":"[^"]*"' | cut -d'"' -f4)
BRAND_API_KEY=$(echo "$BRAND_RESPONSE" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)

if [ -z "$BRAND_ID" ] || [ -z "$BRAND_API_KEY" ]; then
  echo "✗ 品牌注册失败"
  echo "$BRAND_RESPONSE"
  exit 1
fi

echo "✓ 品牌注册成功"
echo "  Brand ID: $BRAND_ID"
echo ""

# 3. 创建批次
echo "[3/8] 创建批次..."
BATCH_RESPONSE=$(curl -s -X POST "$API_BASE/batches" \
  -H "X-Api-Key: $BRAND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "brand_id": "'"$BRAND_ID"'",
    "batch_name": "2024春季系列",
    "factory_id": "factory-001",
    "expected_count": 100
  }')

BATCH_ID=$(echo "$BATCH_RESPONSE" | grep -o '"batch_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$BATCH_ID" ]; then
  echo "✗ 批次创建失败"
  echo "$BATCH_RESPONSE"
  exit 1
fi

echo "✓ 批次创建成功"
echo "  Batch ID: $BATCH_ID"
echo ""

# 4. 查询批次详情
echo "[4/8] 查询批次详情..."
BATCH_DETAIL=$(curl -s -X GET "$API_BASE/batches/$BATCH_ID" \
  -H "X-Api-Key: $BRAND_API_KEY")

if echo "$BATCH_DETAIL" | grep -q '"status":"Open"'; then
  echo "✓ 批次状态为 Open"

  if echo "$BATCH_DETAIL" | grep -q '"expected_count":100'; then
    echo "✓ expected_count 正确"
  else
    echo "✗ expected_count 不正确"
    echo "$BATCH_DETAIL"
    exit 1
  fi
else
  echo "✗ 批次状态不正确"
  echo "$BATCH_DETAIL"
  exit 1
fi
echo ""

# 5. 盲扫资产到批次
echo "[5/8] 盲扫资产到批次..."
ASSET_RESPONSE=$(curl -s -X POST "$API_BASE/assets/blind-scan" \
  -H "X-Api-Key: $BRAND_API_KEY" \
  -H "X-Trace-Id: $(generate_uuid)" \
  -H "X-Idempotency-Key: batch-test-$(date +%s)-$RANDOM" \
  -H "Content-Type: application/json" \
  -d '{
    "uid": "04'$(openssl rand -hex 6 | tr '[:lower:]' '[:upper:]')'",
    "brand_id": "'"$BRAND_ID"'",
    "batch_id": "'"$BATCH_ID"'",
    "metadata": {}
  }')

ASSET_ID=$(echo "$ASSET_RESPONSE" | grep -o '"asset_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ASSET_ID" ]; then
  echo "✗ 盲扫失败"
  echo "$ASSET_RESPONSE"
  exit 1
fi

echo "✓ 盲扫成功，资产已关联到批次"
echo "  Asset ID: $ASSET_ID"
echo ""

# 6. 查询批次列表
echo "[6/8] 查询批次列表..."
BATCH_LIST=$(curl -s -X GET "$API_BASE/batches?page=1&page_size=10" \
  -H "X-Api-Key: $BRAND_API_KEY")

if echo "$BATCH_LIST" | grep -q "\"batch_id\":\"$BATCH_ID\""; then
  echo "✓ 批次列表包含新创建的批次"

  TOTAL=$(echo "$BATCH_LIST" | grep -o '"total":[0-9]*' | cut -d':' -f2)
  echo "  总批次数: $TOTAL"
else
  echo "✗ 批次列表不包含新创建的批次"
  echo "$BATCH_LIST"
  exit 1
fi
echo ""

# 7. 关闭批次
echo "[7/8] 关闭批次..."
CLOSE_RESPONSE=$(curl -s -X POST "$API_BASE/batches/$BATCH_ID/close" \
  -H "X-Api-Key: $BRAND_API_KEY")

if echo "$CLOSE_RESPONSE" | grep -q '"status":"Closed"'; then
  echo "✓ 批次已关闭"

  if echo "$CLOSE_RESPONSE" | grep -q '"closed_at"'; then
    echo "✓ closed_at 时间戳已记录"
  else
    echo "✗ closed_at 时间戳缺失"
    echo "$CLOSE_RESPONSE"
    exit 1
  fi
else
  echo "✗ 批次关闭失败"
  echo "$CLOSE_RESPONSE"
  exit 1
fi
echo ""

# 8. 验证批次不能重复关闭
echo "[8/8] 验证批次不能重复关闭..."
RECLOSE_RESPONSE=$(curl -s -X POST "$API_BASE/batches/$BATCH_ID/close" \
  -H "X-Api-Key: $BRAND_API_KEY")

if echo "$RECLOSE_RESPONSE" | grep -q "already closed\|not found"; then
  echo "✓ 重复关闭被正确拒绝"
else
  echo "✗ 重复关闭未被拒绝"
  echo "$RECLOSE_RESPONSE"
  exit 1
fi
echo ""

echo "=========================================="
echo "✓ 所有测试通过！"
echo "=========================================="
