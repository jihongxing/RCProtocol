# 启动 RC API 服务
$env:RC_ROOT_KEY_HEX = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
$env:RC_SYSTEM_ID = "rcprotocol-dev"
$env:RC_JWT_SECRET = "my-super-secret-jwt-key-for-testing-only"
$env:RC_API_KEY_SECRET = "rc-dev-api-key-secret-do-not-use-in-prod"
$env:DATABASE_URL = "postgresql://rcprotocol:rcprotocol_dev@localhost:5433/rcprotocol"
$env:TEST_DATABASE_URL = "postgresql://rcprotocol:rcprotocol_dev@localhost:5433/postgres"
$env:REDIS_URL = "redis://localhost:6380"

Write-Host "环境变量已设置:" -ForegroundColor Green
Write-Host "  RC_JWT_SECRET: $env:RC_JWT_SECRET"
Write-Host "  DATABASE_URL: $env:DATABASE_URL"
Write-Host "  TEST_DATABASE_URL: $env:TEST_DATABASE_URL"
Write-Host ""

Set-Location D:\codeSpace\RCProtocol\rust\rc-api
Write-Host "启动服务..." -ForegroundColor Yellow
cargo run --release
