param(
    [string]$ApiBase = "http://localhost:8080/api",
    [string]$FrontendBase = "http://localhost:5173",
    [string]$RcJwtSecret = "my-super-secret-jwt-key-for-testing-only",
    [string]$DatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5433/rcprotocol",
    [string]$TestDatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5433/postgres"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Step($message) {
    Write-Host "`n==> $message" -ForegroundColor Yellow
}

Step "环境预检查"
Write-Host "API_BASE=$ApiBase"
Write-Host "FRONTEND_BASE=$FrontendBase"
Write-Host "DATABASE_URL=$DatabaseUrl"
Write-Host "TEST_DATABASE_URL=$TestDatabaseUrl"

Step "运行前端关键页面测试"
Push-Location "$root\frontend"
pnpm vitest run apps/c-app/src/pages/vault.transfer.test.ts apps/c-app/src/pages/vault.transfer-confirm.test.ts
Pop-Location

Step "运行 Go BFF 联调测试"
Push-Location "$root\services\go-bff"
go test ./internal/handler ./internal/router
Pop-Location

Step "运行 Rust transfer 集成测试"
Push-Location "$root\rust\rc-api"
$env:RC_JWT_SECRET = $RcJwtSecret
$env:DATABASE_URL = $DatabaseUrl
$env:TEST_DATABASE_URL = $TestDatabaseUrl
cargo test --test transfer_integration --test transfer_actions_integration
Pop-Location

Step "本地演示提示"
Write-Host "1. 启动依赖：docker compose -f deploy/compose/docker-compose.yml up -d postgres redis"
Write-Host "2. 启动服务：pwsh ./scripts/start-api.ps1"
Write-Host "3. 启动前端：cd frontend; pnpm dev:c-app"
Write-Host "4. 打开页面：$FrontendBase"

Write-Host "`n✅ transfer 联调、页面、异常流测试已完成" -ForegroundColor Green
