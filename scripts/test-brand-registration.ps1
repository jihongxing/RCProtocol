# 品牌注册与 API Key 管理测试脚本
# 使用方法: .\scripts\test-brand-registration.ps1

$ErrorActionPreference = "Stop"

$BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:8081" }
$PLATFORM_TOKEN = $env:PLATFORM_TOKEN

if (-not $PLATFORM_TOKEN) {
    Write-Host "❌ 请先设置 PLATFORM_TOKEN 环境变量" -ForegroundColor Red
    exit 1
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "品牌注册与 API Key 管理测试" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "BASE_URL=$BASE_URL"
Write-Host ""

# 1. 注册新品牌
Write-Host "1️⃣  注册新品牌..." -ForegroundColor Yellow
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$contactEmail = "brand-$timestamp@example.com"
$registerBody = @{
    brand_name = "Brand Registration Test"
    contact_email = $contactEmail
    industry = "Watches"
} | ConvertTo-Json

$registerResponse = Invoke-RestMethod -Uri "$BASE_URL/brands" `
    -Method Post `
    -Headers @{
        "Authorization" = "Bearer $PLATFORM_TOKEN"
        "Content-Type" = "application/json"
    } `
    -Body $registerBody

$BRAND_ID = $registerResponse.brand_id
$API_KEY = $registerResponse.api_key.api_key

if (-not $BRAND_ID -or -not $API_KEY) {
    Write-Host "❌ 品牌注册失败" -ForegroundColor Red
    exit 1
}

Write-Host "✅ 品牌注册成功" -ForegroundColor Green
Write-Host "   Brand ID: $BRAND_ID"
Write-Host "   API Key: $($API_KEY.Substring(0,20))****"
Write-Host ""

# 2. 查询品牌详情（不返回 api_key）
Write-Host "2️⃣  查询品牌详情（API Key 认证）..." -ForegroundColor Yellow
$brandDetail = Invoke-RestMethod -Uri "$BASE_URL/brands/$BRAND_ID" `
    -Method Get `
    -Headers @{ "X-Api-Key" = $API_KEY }

if ($null -ne $brandDetail.api_key) {
    Write-Host "❌ 品牌详情不应返回 api_key" -ForegroundColor Red
    exit 1
}

Write-Host "✅ 品牌详情查询成功" -ForegroundColor Green
Write-Host "   品牌名称: $($brandDetail.brand_name)"
Write-Host "   状态: $($brandDetail.status)"
Write-Host "   行业: $($brandDetail.industry)"
Write-Host ""

# 3. 测试邮箱唯一性校验
Write-Host "3️⃣  测试邮箱唯一性校验..." -ForegroundColor Yellow
$duplicateBody = @{
    brand_name = "Duplicate Brand"
    contact_email = $contactEmail
    industry = "Fashion"
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri "$BASE_URL/brands" `
        -Method Post `
        -Headers @{
            "Authorization" = "Bearer $PLATFORM_TOKEN"
            "Content-Type" = "application/json"
        } `
        -Body $duplicateBody
    Write-Host "❌ 邮箱唯一性校验失败" -ForegroundColor Red
    exit 1
} catch {
    if ($_.Exception.Response.StatusCode -eq 409) {
        Write-Host "✅ 邮箱唯一性校验生效（409 Conflict）" -ForegroundColor Green
    } else {
        Write-Host "❌ 意外错误: $($_.Exception.Response.StatusCode)" -ForegroundColor Red
        exit 1
    }
}
Write-Host ""

# 4. 轮换 API Key
Write-Host "4️⃣  轮换 API Key..." -ForegroundColor Yellow
$rotateResponse = Invoke-RestMethod -Uri "$BASE_URL/brands/$BRAND_ID/api-keys/rotate" `
    -Method Post `
    -Headers @{
        "X-Api-Key" = $API_KEY
        "Content-Type" = "application/json"
    } `
    -Body (@{ reason = "manual rotation test" } | ConvertTo-Json)

$NEW_API_KEY = $rotateResponse.api_key
if (-not $NEW_API_KEY) {
    Write-Host "❌ API Key 轮换失败" -ForegroundColor Red
    exit 1
}

Write-Host "✅ API Key 轮换成功" -ForegroundColor Green
Write-Host "   新 API Key: $($NEW_API_KEY.Substring(0,20))****"
Write-Host ""

# 5. 验证旧 API Key 已失效
Write-Host "5️⃣  验证旧 API Key 已失效..." -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$BASE_URL/brands/$BRAND_ID" `
        -Method Get `
        -Headers @{ "X-Api-Key" = $API_KEY }
    Write-Host "❌ 旧 API Key 仍然有效" -ForegroundColor Red
    exit 1
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "✅ 旧 API Key 已正确失效（401）" -ForegroundColor Green
    } else {
        Write-Host "❌ 意外错误: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
}
Write-Host ""

# 6. 验证新 API Key 可用
Write-Host "6️⃣  验证新 API Key 可用..." -ForegroundColor Yellow
$newKeyTest = Invoke-RestMethod -Uri "$BASE_URL/brands/$BRAND_ID" `
    -Method Get `
    -Headers @{ "X-Api-Key" = $NEW_API_KEY }

Write-Host "✅ 新 API Key 验证成功: $($newKeyTest.brand_name)" -ForegroundColor Green
Write-Host ""

# 7. 查询 API Keys 列表
Write-Host "7️⃣  查询 API Keys 列表..." -ForegroundColor Yellow
$apiKeysList = Invoke-RestMethod -Uri "$BASE_URL/brands/$BRAND_ID/api-keys" `
    -Method Get `
    -Headers @{ "X-Api-Key" = $NEW_API_KEY }

$activeCount = @($apiKeysList.keys | Where-Object { $_.status -eq "Active" }).Count
$revokedCount = @($apiKeysList.keys | Where-Object { $_.status -eq "Revoked" }).Count
if ($activeCount -ne 1 -or $revokedCount -ne 1) {
    Write-Host "❌ API Key 列表状态不符合预期" -ForegroundColor Red
    exit 1
}

foreach ($key in $apiKeysList.keys) {
    $statusColor = if ($key.status -eq "Active") { "Green" } else { "Gray" }
    Write-Host "   $($key.key_prefix) - $($key.status)" -ForegroundColor $statusColor
}
Write-Host ""

# 8. 测试权限校验（Brand Key 不允许注册新品牌）
Write-Host "8️⃣  测试权限校验（Brand 角色）..." -ForegroundColor Yellow
$unauthorizedBody = @{
    brand_name = "Unauthorized Brand"
    contact_email = "unauthorized-$timestamp@example.com"
    industry = "Jewelry"
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri "$BASE_URL/brands" `
        -Method Post `
        -Headers @{
            "X-Api-Key" = $NEW_API_KEY
            "Content-Type" = "application/json"
        } `
        -Body $unauthorizedBody
    Write-Host "❌ 权限校验失败" -ForegroundColor Red
    exit 1
} catch {
    if ($_.Exception.Response.StatusCode -eq 403) {
        Write-Host "✅ 权限校验生效（403 Forbidden）" -ForegroundColor Green
    } else {
        Write-Host "❌ 意外错误: $($_.Exception.Response.StatusCode)" -ForegroundColor Red
        exit 1
    }
}
Write-Host ""

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "✅ 品牌注册与 API Key 管理测试通过" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
