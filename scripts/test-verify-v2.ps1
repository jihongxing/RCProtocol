param(
    [string]$ApiBase = "http://localhost:8081",
    [string]$Uid = "04A31B2C3D4E5F",
    [string]$Ctr = "010000",
    [string]$Cmac = "0000000000000000"
)

$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Verification V2 联调脚本" -ForegroundColor Cyan
Write-Host "API_BASE=$ApiBase"
Write-Host "UID=$Uid CTR=$Ctr"
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "1) 请求 V1 /verify" -ForegroundColor Yellow
$v1 = Invoke-RestMethod -Uri "$ApiBase/verify?uid=$Uid&ctr=$Ctr&cmac=$Cmac" -Method Get
$v1 | ConvertTo-Json -Depth 10
Write-Host ""

Write-Host "2) 请求 V2 /verify/v2" -ForegroundColor Yellow
$v2 = Invoke-RestMethod -Uri "$ApiBase/verify/v2?uid=$Uid&ctr=$Ctr&cmac=$Cmac" -Method Get
$v2 | ConvertTo-Json -Depth 10
Write-Host ""

Write-Host "✅ 已打印 V1 / V2 完整结构化响应" -ForegroundColor Green
