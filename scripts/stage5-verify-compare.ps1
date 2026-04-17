param(
    [string]$ApiBase = "http://localhost:8081",
    [string]$Uid = "04A31B2C3D4E5F",
    [string]$Ctr = "010000",
    [string]$Cmac = "0000000000000000"
)

$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Stage 5 Verify V1/V2 对照" -ForegroundColor Cyan
Write-Host "API_BASE=$ApiBase"
Write-Host "UID=$Uid CTR=$Ctr"
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "1) 请求 /verify" -ForegroundColor Yellow
$v1 = Invoke-RestMethod -Uri "$ApiBase/verify?uid=$Uid&ctr=$Ctr&cmac=$Cmac" -Method Get
$v1 | ConvertTo-Json -Depth 10
Write-Host ""

Write-Host "2) 请求 /verify/v2" -ForegroundColor Yellow
$v2 = Invoke-RestMethod -Uri "$ApiBase/verify/v2?uid=$Uid&ctr=$Ctr&cmac=$Cmac" -Method Get
$v2 | ConvertTo-Json -Depth 10
Write-Host ""

Write-Host "3) 语义提醒" -ForegroundColor Yellow
Write-Host "- V1 更偏当前 MVP 验真结果"
Write-Host "- V2 额外区分标签真实性、承诺完整性、协议状态"
Write-Host ""

Write-Host "✅ Stage 5 Verify V1/V2 对照完成" -ForegroundColor Green
