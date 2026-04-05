param(
    [string]$BaseUrl = "http://localhost:8081",
    [string]$MainAssetId = "asset-main-001",
    [string]$FreezeAssetId = "asset-freeze-001",
    [string]$TransferAssetId = "asset-transfer-001"
)

$ErrorActionPreference = "Stop"

function New-TraceId {
    return [guid]::NewGuid().ToString()
}

function Invoke-ProtocolAction {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$ActorRole,
        [string]$ApprovalId,
        [string]$PolicyVersion,
        [object]$Body = @{ previous_state = $null }
    )

    $headers = @{
        Authorization        = "local-dev-actor"
        "X-Trace-Id"        = New-TraceId
        "X-Idempotency-Key" = [guid]::NewGuid().ToString()
        "X-Actor-Role"      = $ActorRole
        "X-Actor-Org"       = "org-demo"
    }

    if ($ApprovalId) {
        $headers["X-Approval-Id"] = $ApprovalId
    }

    if ($PolicyVersion) {
        $headers["X-Policy-Version"] = $PolicyVersion
    }

    $jsonBody = $Body | ConvertTo-Json -Depth 4
    Write-Host "--> POST $Path [$ActorRole]"
    Invoke-RestMethod -Method Post -Uri "$BaseUrl$Path" -Headers $headers -ContentType "application/json" -Body $jsonBody
}

function Invoke-Verify {
    param([Parameter(Mandatory = $true)][string]$AssetId)
    Invoke-RestMethod -Method Get -Uri "$BaseUrl/verify/$AssetId"
}

Write-Host "=== RCProtocol main chain flow ==="
Invoke-ProtocolAction -Path "/assets/$MainAssetId/blind-log" -ActorRole "Factory"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/stock-in" -ActorRole "Factory"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/activate" -ActorRole "Brand"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/activate-entangle" -ActorRole "Brand"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/activate-confirm" -ActorRole "Brand"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/legal-sell" -ActorRole "Brand" -ApprovalId "approval-demo-001" -PolicyVersion "policy-v1"
Invoke-ProtocolAction -Path "/assets/$MainAssetId/transfer" -ActorRole "Consumer"
Invoke-Verify -AssetId $MainAssetId

Write-Host "=== RCProtocol freeze / recover flow ==="
Invoke-ProtocolAction -Path "/assets/$FreezeAssetId/freeze" -ActorRole "Moderator" -ApprovalId "approval-freeze-001" -PolicyVersion "policy-v1"
Invoke-ProtocolAction -Path "/assets/$FreezeAssetId/recover" -ActorRole "Moderator" -ApprovalId "approval-recover-001" -PolicyVersion "policy-v1"
Invoke-Verify -AssetId $FreezeAssetId

Write-Host "=== RCProtocol terminal actions ==="
Invoke-ProtocolAction -Path "/assets/$TransferAssetId/consume" -ActorRole "Consumer"
Invoke-ProtocolAction -Path "/assets/asset-terminal-001/legacy" -ActorRole "Consumer"
Invoke-Verify -AssetId $TransferAssetId
Invoke-Verify -AssetId "asset-terminal-001"

Write-Host "All scripted local flows completed."
