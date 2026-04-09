#!/bin/bash

# 激活链路集成测试脚本

set -e

API_BASE="${API_BASE:-http://localhost:8081}"
# 必须与 scripts/start-api.ps1 中的 RC_JWT_SECRET 保持一致；也可在运行前自行 export 覆盖
RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
PLATFORM_TOKEN=""
BRAND_API_KEY=""
BRAND_ID=""
ASSET_ID=""

echo "=========================================="
echo "激活链路集成测试"
echo "=========================================="
echo ""

# 生成 Platform JWT Token
echo "[1/7] 生成 Platform JWT Token..."
cd rust/rc-api
PLATFORM_TOKEN=$(RC_JWT_SECRET="$RC_JWT_SECRET" cargo run --release --example generate_jwt 2>/dev/null | tail -1)
cd ../..
if [ -z "$PLATFORM_TOKEN" ]; then
  echo "✗ Token 生成失败"
  exit 1
fi
echo "✓ Platform Token: ${PLATFORM_TOKEN:0:50}..."
echo ""

# 注册测试品牌
echo "[2/7] 注册测试品牌..."
TIMESTAMP=$(date +%s)
BRAND_RESPONSE=$(curl -s -X POST "$API_BASE/brands" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"brand_name\": \"测试品牌-激活链路-$TIMESTAMP\",
    \"contact_email\": \"activation-test-$TIMESTAMP@example.com\",
    \"industry\": \"Watches\"
  }")

BRAND_ID=$(echo "$BRAND_RESPONSE" | grep -o '"brand_id":"[^"]*"' | cut -d'"' -f4)
BRAND_API_KEY=$(echo "$BRAND_RESPONSE" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)

if [ -z "$BRAND_ID" ]; then
  echo "✗ 品牌注册失败"
  echo "$BRAND_RESPONSE"
  exit 1
fi

echo "✓ 品牌注册成功"
echo "  Brand ID: $BRAND_ID"
echo "  API Key: $BRAND_API_KEY"
echo ""

# 盲扫资产
echo "[3/7] 盲扫资产..."
TRACE_ID=$(uuidgen 2>/dev/null || powershell.exe -Command "[guid]::NewGuid().ToString()" | tr -d '\r')
IDEMPOTENCY_KEY="blind-scan-test-$(date +%s)"
BLIND_SCAN_RESPONSE=$(curl -s -X POST "$API_BASE/assets/blind-scan" \
  -H "X-Api-Key: $BRAND_API_KEY" \
  -H "X-Trace-Id: $TRACE_ID" \
  -H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"uid\": \"04$(openssl rand -hex 6 | tr '[:lower:]' '[:upper:]')\",
    \"brand_id\": \"$BRAND_ID\"
  }")

ASSET_ID=$(echo "$BLIND_SCAN_RESPONSE" | grep -o '"asset_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ASSET_ID" ]; then
  echo "✗ 盲扫失败"
  echo "$BLIND_SCAN_RESPONSE"
  exit 1
fi

echo "✓ 盲扫成功"
echo "  Asset ID: $ASSET_ID"
echo ""

# 入库资产（使用 Platform token，因为 StockIn 需要 Factory 或 Platform 角色）
# Platform 执行业务操作需要提供 approval_id
echo "[4/7] 入库资产..."
TRACE_ID=$(uuidgen 2>/dev/null || powershell.exe -Command "[guid]::NewGuid().ToString()" | tr -d '\r')
IDEMPOTENCY_KEY="stock-in-test-$(date +%s)"
APPROVAL_ID=$(uuidgen 2>/dev/null || powershell.exe -Command "[guid]::NewGuid().ToString()" | tr -d '\r')

STOCK_IN_RESPONSE=$(curl -s -X POST "$API_BASE/assets/$ASSET_ID/stock-in" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  -H "X-Trace-Id: $TRACE_ID" \
  -H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "X-Approval-Id: $APPROVAL_ID" \
  -H "Content-Type: application/json" \
  -d '{}')

if echo "$STOCK_IN_RESPONSE" | grep -q '"to_state":"Unassigned"'; then
  echo "✓ 入库成功"
else
  echo "✗ 入库失败"
  echo "$STOCK_IN_RESPONSE"
  exit 1
fi
echo ""

# 激活资产
echo "[5/7] 激活资产..."
TRACE_ID=$(uuidgen 2>/dev/null || powershell.exe -Command "[guid]::NewGuid().ToString()" | tr -d '\r')
IDEMPOTENCY_KEY="activate-test-$(date +%s)"

ACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/assets/$ASSET_ID/activate" \
  -H "X-Api-Key: $BRAND_API_KEY" \
  -H "X-Trace-Id: $TRACE_ID" \
  -H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "external_product_id": "SKU-LUXURY-2024-001",
    "external_product_name": "经典款手提包",
    "external_product_url": "https://brand.com/products/SKU-LUXURY-2024-001"
  }')

echo "$ACTIVATE_RESPONSE"
echo ""

# 检查响应中是否包含虚拟母卡信息
if echo "$ACTIVATE_RESPONSE" | grep -q "virtual_mother_card"; then
  echo "✓ 激活成功，虚拟母卡已生成"

  AUTHORITY_UID=$(echo "$ACTIVATE_RESPONSE" | grep -o '"authority_uid":"[^"]*"' | cut -d'"' -f4)
  CREDENTIAL_HASH=$(echo "$ACTIVATE_RESPONSE" | grep -o '"credential_hash":"[^"]*"' | cut -d'"' -f4)

  echo "  Authority UID: $AUTHORITY_UID"
  echo "  Credential Hash: ${CREDENTIAL_HASH:0:40}..."
else
  echo "✗ 激活失败或虚拟母卡未生成"
  exit 1
fi
echo ""

# 查询资产详情验证状态
echo "[6/7] 查询资产详情..."
ASSET_DETAIL=$(curl -s -X GET "$API_BASE/assets/$ASSET_ID" \
  -H "X-Api-Key: $BRAND_API_KEY")

echo "$ASSET_DETAIL"
echo ""

if echo "$ASSET_DETAIL" | grep -q '"current_state":"RotatingKeys"'; then
  echo "✓ 资产状态已更新为 RotatingKeys（激活第一步完成）"
else
  echo "✗ 资产状态未正确更新"
  exit 1
fi

if echo "$ASSET_DETAIL" | grep -q '"external_product_id":"SKU-LUXURY-2024-001"'; then
  echo "✓ 外部产品映射已保存"
else
  echo "✗ 外部产品映射未保存"
  exit 1
fi
echo ""

# 测试幂等性
echo "[7/7] 测试幂等性..."
IDEMPOTENT_RESPONSE=$(curl -s -X POST "$API_BASE/assets/$ASSET_ID/activate" \
  -H "X-Api-Key: $BRAND_API_KEY" \
  -H "X-Trace-Id: $TRACE_ID" \
  -H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "external_product_id": "SKU-LUXURY-2024-001",
    "external_product_name": "经典款手提包",
    "external_product_url": "https://brand.com/products/SKU-LUXURY-2024-001"
  }')

if echo "$IDEMPOTENT_RESPONSE" | grep -q "virtual_mother_card"; then
  echo "✓ 幂等性测试通过（返回缓存响应）"
else
  echo "✗ 幂等性测试失败"
  echo "$IDEMPOTENT_RESPONSE"
  exit 1
fi
echo ""

echo "=========================================="
echo "✓ 所有测试通过！"
echo "=========================================="
