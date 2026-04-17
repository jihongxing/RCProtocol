param(
    [string]$ApiBase = "http://localhost:8081",
    [string]$VerifyUid = "04A31B2C3D4E5F",
    [string]$VerifyCtr = "010000",
    [string]$VerifyCmac = "0000000000000000",
    [string]$RcJwtSecret = "my-super-secret-jwt-key-for-testing-only",
    [string]$DatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol",
    [string]$TestDatabaseUrl = "postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres",
    [string]$RedisUrl = "redis://localhost:6379",
    [int]$Count = 5
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Step($message) {
    Write-Host "`n==> $message" -ForegroundColor Yellow
}

function Measure-Url {
    param(
        [string]$Url,
        [int]$SampleCount
    )

    $values = @()
    for ($i = 0; $i -lt $SampleCount; $i++) {
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        $null = Invoke-WebRequest -Uri $Url -UseBasicParsing
        $sw.Stop()
        $values += $sw.Elapsed.TotalMilliseconds
    }

    $sorted = $values | Sort-Object
    $p95Index = [Math]::Max(0, [Math]::Min($sorted.Count - 1, [Math]::Floor($sorted.Count * 0.95) - 1))
    $avg = ($values | Measure-Object -Average).Average
    Write-Host ("count={0} avg_ms={1:N2} p95_ms={2:N2}" -f $SampleCount, $avg, $sorted[$p95Index])
}

Step "健康检查性能样本"
Measure-Url -Url "$ApiBase/healthz" -SampleCount $Count

Step "verify 性能样本"
Measure-Url -Url "$ApiBase/verify?uid=$VerifyUid&ctr=$VerifyCtr&cmac=$VerifyCmac" -SampleCount $Count

Step "verify v2 性能样本"
Measure-Url -Url "$ApiBase/verify/v2?uid=$VerifyUid&ctr=$VerifyCtr&cmac=$VerifyCmac" -SampleCount $Count

Push-Location "$root\rust\rc-api"
$env:RC_JWT_SECRET = $RcJwtSecret
$env:DATABASE_URL = $DatabaseUrl
$env:TEST_DATABASE_URL = $TestDatabaseUrl
$env:REDIS_URL = $RedisUrl

Step "activation_integration 耗时观测"
Measure-Command { cargo test --test activation_integration } | ForEach-Object {
    Write-Host ("elapsed={0:N2}s" -f $_.TotalSeconds)
}

Step "transfer_integration 耗时观测"
Measure-Command { cargo test --test transfer_integration } | ForEach-Object {
    Write-Host ("elapsed={0:N2}s" -f $_.TotalSeconds)
}
Pop-Location

Write-Host "`n✅ Stage 5 性能基线采集完成，请将结果回填 docs/ops/stage-5-performance-baseline.md" -ForegroundColor Green
