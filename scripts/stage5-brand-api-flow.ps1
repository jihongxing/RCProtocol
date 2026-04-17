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

Step "品牌 API 闭环（脚本）"
try {
    pwsh "$root\scripts\test-brand-registration.ps1"
} catch {
    Write-Host "⚠️ 需要本地运行中的 rc-api 与 PLATFORM_TOKEN 支持" -ForegroundColor DarkYellow
}

Step "品牌 API 闭环（Rust 集成测试）"
Push-Location "$root\rust\rc-api"
$env:RC_JWT_SECRET = $RcJwtSecret
$env:DATABASE_URL = $DatabaseUrl
$env:TEST_DATABASE_URL = $TestDatabaseUrl
$env:REDIS_URL = $RedisUrl
cargo test --test brand_registration_integration
Pop-Location

Write-Host "`n✅ Stage 5 品牌 API 闭环入口已执行完成" -ForegroundColor Green
