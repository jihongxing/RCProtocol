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

Push-Location "$root\rust\rc-api"
$env:RC_JWT_SECRET = $RcJwtSecret
$env:DATABASE_URL = $DatabaseUrl
$env:TEST_DATABASE_URL = $TestDatabaseUrl
$env:REDIS_URL = $RedisUrl

Step "cargo test --test verify_integration"
cargo test --test verify_integration

Step "cargo test --test brand_registration_integration"
cargo test --test brand_registration_integration

Step "cargo test --test transfer_actions_integration"
cargo test --test transfer_actions_integration

Step "cargo test --test protocol_write_flow_integration"
cargo test --test protocol_write_flow_integration

Pop-Location
Write-Host "`n✅ Stage 5 异常流矩阵回归入口已执行完成" -ForegroundColor Green
