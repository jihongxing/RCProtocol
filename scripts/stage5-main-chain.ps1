param(
    [string]$RcJwtSecret = "my-super-secret-jwt-key-for-testing-only",
    [string]$DatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol",
    [string]$TestDatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres",
    [string]$RedisUrl = "redis://localhost:6379"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Step($message) {
    Write-Host "`n==> $message" -ForegroundColor Yellow
}

Step "Stage 5 主链路环境信息"
Write-Host "DATABASE_URL=$DatabaseUrl"
Write-Host "TEST_DATABASE_URL=$TestDatabaseUrl"
Write-Host "REDIS_URL=$RedisUrl"

Step "运行品牌接入闭环脚本"
try {
    pwsh "$root\scripts\test-brand-registration.ps1"
} catch {
    Write-Host "⚠️ 品牌脚本依赖本地运行中的 rc-api，若当前仅做测试基线可稍后单独执行" -ForegroundColor DarkYellow
}

Push-Location "$root\rust\rc-api"
$env:RC_JWT_SECRET = $RcJwtSecret
$env:DATABASE_URL = $DatabaseUrl
$env:TEST_DATABASE_URL = $TestDatabaseUrl
$env:REDIS_URL = $RedisUrl

Step "cargo test --test activation_integration"
cargo test --test activation_integration

Step "cargo test --test transfer_integration"
cargo test --test transfer_integration

Step "cargo test --test transfer_actions_integration"
cargo test --test transfer_actions_integration

Step "cargo test --test verify_integration"
cargo test --test verify_integration
Pop-Location

Step "可选前端关键页面测试"
Push-Location "$root\frontend"
try {
    pnpm vitest run apps/c-app/src/pages/vault.transfer.test.ts apps/c-app/src/pages/vault.transfer-confirm.test.ts
} catch {
    Write-Host "⚠️ 前端测试未通过，请单独排查" -ForegroundColor DarkYellow
}
Pop-Location

Write-Host "`n✅ Stage 5 主链路测试入口已执行完成" -ForegroundColor Green
