param(
    [string]$BaseUrl = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"

function Invoke-Verify {
    param([Parameter(Mandatory = $true)][string]$AssetId)
    return Invoke-RestMethod -Method Get -Uri "$BaseUrl/verify/$AssetId"
}

function Assert-Equal {
    param(
        [Parameter(Mandatory = $true)]$Actual,
        [Parameter(Mandatory = $true)]$Expected,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if ($Actual -ne $Expected) {
        throw "$Message. Expected=$Expected Actual=$Actual"
    }
}

Write-Host "== Reset note =="
Write-Host "Please recreate the local PostgreSQL volume or database before running this script."
Write-Host "This script assumes 001_init.sql and 002_seed.sql were freshly applied."

& "$PSScriptRoot\local-main-chain.ps1" -BaseUrl $BaseUrl

Write-Host "== Assertions =="
$main = Invoke-Verify -AssetId "asset-main-001"
Assert-Equal -Actual $main.current_state -Expected "Transferred" -Message "asset-main-001 final state mismatch"
Assert-Equal -Actual $main.event_count -Expected 7 -Message "asset-main-001 event count mismatch"
Assert-Equal -Actual $main.verification_result -Expected "verified" -Message "asset-main-001 verification mismatch"

$freeze = Invoke-Verify -AssetId "asset-freeze-001"
Assert-Equal -Actual $freeze.current_state -Expected "Activated" -Message "asset-freeze-001 final state mismatch"
Assert-Equal -Actual $freeze.event_count -Expected 2 -Message "asset-freeze-001 event count mismatch"
Assert-Equal -Actual $freeze.verification_result -Expected "verified" -Message "asset-freeze-001 verification mismatch"

$consume = Invoke-Verify -AssetId "asset-transfer-001"
Assert-Equal -Actual $consume.current_state -Expected "Consumed" -Message "asset-transfer-001 final state mismatch"
Assert-Equal -Actual $consume.event_count -Expected 1 -Message "asset-transfer-001 event count mismatch"
Assert-Equal -Actual $consume.verification_result -Expected "pending" -Message "asset-transfer-001 verification mismatch"

$legacy = Invoke-Verify -AssetId "asset-terminal-001"
Assert-Equal -Actual $legacy.current_state -Expected "Legacy" -Message "asset-terminal-001 final state mismatch"
Assert-Equal -Actual $legacy.event_count -Expected 1 -Message "asset-terminal-001 event count mismatch"
Assert-Equal -Actual $legacy.verification_result -Expected "pending" -Message "asset-terminal-001 verification mismatch"

Write-Host "All local reset + assert checks passed."
