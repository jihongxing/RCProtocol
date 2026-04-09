# RCProtocol Local Main Chain Test Script
# Purpose: Test the complete asset lifecycle from blind scan to transfer
# Usage: .\scripts\local-main-chain.ps1

$ErrorActionPreference = "Stop"

# Configuration
$BASE_URL = "http://localhost:8080"
$API_KEY = "rc_test_luxurywatch_abc123xyz"
$BRAND_ID = "brand_luxury_watch"

# Colors for output
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Info { Write-Host $args -ForegroundColor Cyan }
function Write-Error { Write-Host $args -ForegroundColor Red }
function Write-Step { Write-Host "`n==> $args" -ForegroundColor Yellow }

# Helper function to make API calls
function Invoke-RCApi {
    param(
        [string]$Method = "GET",
        [string]$Endpoint,
        [hashtable]$Body = $null,
        [string]$ApiKey = $API_KEY
    )

    $headers = @{
        "Content-Type" = "application/json"
        "X-API-Key" = $ApiKey
        "X-Trace-ID" = [guid]::NewGuid().ToString()
    }

    $params = @{
        Method = $Method
        Uri = "$BASE_URL$Endpoint"
        Headers = $headers
    }

    if ($Body) {
        $params.Body = ($Body | ConvertTo-Json -Depth 10)
    }

    try {
        $response = Invoke-RestMethod @params
        return $response
    }
    catch {
        Write-Error "API call failed: $_"
        Write-Error "Response: $($_.Exception.Response)"
        throw
    }
}

# Test data
$TEST_UID_1 = "04A1B2C3D4E5F611"
$TEST_UID_2 = "04A1B2C3D4E5F612"
$TEST_BATCH_NAME = "Test-Batch-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
$TEST_SKU_ID = "SKU-TEST-CHRONO-001"
$TEST_SKU_NAME = "Test Chronograph Watch"
$TEST_USER_ID = "test_user_$(Get-Random)"

Write-Host @"

╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║           RCProtocol Main Chain Integration Test             ║
║                                                               ║
║  Testing: Blind Scan → Activate → Verify → Sell → Transfer  ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

"@ -ForegroundColor Magenta

# ============================================================================
# Step 1: Health Check
# ============================================================================
Write-Step "Step 1: Health Check"

try {
    $health = Invoke-RestMethod -Uri "$BASE_URL/health" -Method GET
    Write-Success "✓ API is healthy: $($health.status)"
}
catch {
    Write-Error "✗ API health check failed. Is the service running?"
    Write-Info "  Run: docker-compose -f deploy/compose/docker-compose.yml up"
    exit 1
}

# ============================================================================
# Step 2: Create Batch
# ============================================================================
Write-Step "Step 2: Create Batch for Blind Scan"

$batchPayload = @{
    brand_id = $BRAND_ID
    batch_name = $TEST_BATCH_NAME
    factory_id = "factory_test_001"
    expected_count = 2
}

try {
    $batch = Invoke-RCApi -Method POST -Endpoint "/api/v1/batches" -Body $batchPayload
    $BATCH_ID = $batch.batch_id
    Write-Success "✓ Batch created: $BATCH_ID"
    Write-Info "  Batch name: $TEST_BATCH_NAME"
}
catch {
    Write-Error "✗ Failed to create batch"
    throw
}

# ============================================================================
# Step 3: Blind Scan Assets
# ============================================================================
Write-Step "Step 3: Blind Scan Assets (Factory Logging)"

$blindScanPayload1 = @{
    uid = $TEST_UID_1
    brand_id = $BRAND_ID
    batch_id = $BATCH_ID
    metadata = @{
        factory_line = "A1"
        operator = "test_operator"
    }
}

$blindScanPayload2 = @{
    uid = $TEST_UID_2
    brand_id = $BRAND_ID
    batch_id = $BATCH_ID
    metadata = @{
        factory_line = "A1"
        operator = "test_operator"
    }
}

try {
    $asset1 = Invoke-RCApi -Method POST -Endpoint "/api/v1/assets/blind-scan" -Body $blindScanPayload1
    $ASSET_ID_1 = $asset1.asset_id
    Write-Success "✓ Asset 1 blind scanned: $ASSET_ID_1"
    Write-Info "  UID: $TEST_UID_1"
    Write-Info "  State: $($asset1.current_state)"

    Start-Sleep -Milliseconds 500

    $asset2 = Invoke-RCApi -Method POST -Endpoint "/api/v1/assets/blind-scan" -Body $blindScanPayload2
    $ASSET_ID_2 = $asset2.asset_id
    Write-Success "✓ Asset 2 blind scanned: $ASSET_ID_2"
    Write-Info "  UID: $TEST_UID_2"
    Write-Info "  State: $($asset2.current_state)"
}
catch {
    Write-Error "✗ Failed to blind scan assets"
    throw
}

# ============================================================================
# Step 4: Activate Asset 1 (with External SKU Mapping)
# ============================================================================
Write-Step "Step 4: Activate Asset 1 (Brand Activation)"

$activatePayload = @{
    asset_id = $ASSET_ID_1
    brand_id = $BRAND_ID
    external_product_id = $TEST_SKU_ID
    external_product_name = $TEST_SKU_NAME
    external_product_url = "https://example.com/products/$TEST_SKU_ID"
    authority_type = "VIRTUAL_APP"
    metadata = @{
        activation_reason = "integration_test"
        activated_by = "test_script"
    }
}

try {
    $activated = Invoke-RCApi -Method POST -Endpoint "/api/v1/assets/activate" -Body $activatePayload
    Write-Success "✓ Asset 1 activated: $ASSET_ID_1"
    Write-Info "  State: $($activated.current_state)"
    Write-Info "  External SKU: $TEST_SKU_ID"
    Write-Info "  Virtual Mother Card: Generated"
}
catch {
    Write-Error "✗ Failed to activate asset"
    throw
}

# ============================================================================
# Step 5: Verify Asset 1 (Consumer Scan)
# ============================================================================
Write-Step "Step 5: Verify Asset 1 (Consumer Verification)"

# Simulate dynamic authentication parameters
$verifyPayload = @{
    uid = $TEST_UID_1
    ctr = 1
    cmac = "SIMULATED_CMAC_VALUE"
    metadata = @{
        scan_source = "test_script"
        ip_address = "127.0.0.1"
    }
}

try {
    $verified = Invoke-RCApi -Method POST -Endpoint "/api/v1/verify" -Body $verifyPayload
    Write-Success "✓ Asset 1 verified successfully"
    Write-Info "  Auth Result: $($verified.auth_result)"
    Write-Info "  Asset State: $($verified.asset_state)"
    Write-Info "  Brand: $($verified.brand_name)"
    Write-Info "  Product: $($verified.product_name)"
}
catch {
    Write-Error "✗ Failed to verify asset"
    throw
}

# ============================================================================
# Step 6: Legal Sell Asset 1
# ============================================================================
Write-Step "Step 6: Legal Sell Asset 1 (Brand Records Sale)"

$sellPayload = @{
    asset_id = $ASSET_ID_1
    buyer_id = $TEST_USER_ID
    sale_channel = "official_store"
    metadata = @{
        order_id = "ORDER-TEST-$(Get-Random)"
        sale_price = 50000
        currency = "CNY"
    }
}

try {
    $sold = Invoke-RCApi -Method POST -Endpoint "/api/v1/assets/sell" -Body $sellPayload
    Write-Success "✓ Asset 1 legally sold"
    Write-Info "  State: $($sold.current_state)"
    Write-Info "  Owner: $TEST_USER_ID"
}
catch {
    Write-Error "✗ Failed to sell asset"
    throw
}

# ============================================================================
# Step 7: Transfer Asset 1 (Consumer to Consumer)
# ============================================================================
Write-Step "Step 7: Transfer Asset 1 (Ownership Transfer)"

$NEW_OWNER_ID = "test_user_$(Get-Random)"

$transferPayload = @{
    asset_id = $ASSET_ID_1
    from_user_id = $TEST_USER_ID
    to_user_id = $NEW_OWNER_ID
    authority_proof = @{
        type = "VIRTUAL_APP"
        credential_hash = "SIMULATED_VIRTUAL_TOKEN"
        biometric_verified = $true
    }
    metadata = @{
        transfer_reason = "secondary_sale"
        transfer_price = 45000
    }
}

try {
    $transferred = Invoke-RCApi -Method POST -Endpoint "/api/v1/assets/transfer" -Body $transferPayload
    Write-Success "✓ Asset 1 transferred successfully"
    Write-Info "  State: $($transferred.current_state)"
    Write-Info "  Previous Owner: $TEST_USER_ID"
    Write-Info "  New Owner: $NEW_OWNER_ID"
}
catch {
    Write-Error "✗ Failed to transfer asset"
    throw
}

# ============================================================================
# Step 8: Query Asset History
# ============================================================================
Write-Step "Step 8: Query Asset History (Audit Trail)"

try {
    $history = Invoke-RCApi -Method GET -Endpoint "/api/v1/assets/$ASSET_ID_1/history"
    Write-Success "✓ Asset history retrieved"
    Write-Info "  Total events: $($history.events.Count)"

    Write-Host "`n  Event Timeline:" -ForegroundColor Cyan
    foreach ($event in $history.events) {
        Write-Host "    • $($event.action): $($event.from_state) → $($event.to_state)" -ForegroundColor Gray
        Write-Host "      Actor: $($event.actor_id) ($($event.actor_role))" -ForegroundColor Gray
        Write-Host "      Time: $($event.occurred_at)" -ForegroundColor Gray
    }
}
catch {
    Write-Error "✗ Failed to query asset history"
    throw
}

# ============================================================================
# Step 9: Query Batch Status
# ============================================================================
Write-Step "Step 9: Query Batch Status"

try {
    $batchStatus = Invoke-RCApi -Method GET -Endpoint "/api/v1/batches/$BATCH_ID"
    Write-Success "✓ Batch status retrieved"
    Write-Info "  Batch: $($batchStatus.batch_name)"
    Write-Info "  Expected: $($batchStatus.expected_count)"
    Write-Info "  Actual: $($batchStatus.actual_count)"
    Write-Info "  Status: $($batchStatus.status)"
}
catch {
    Write-Error "✗ Failed to query batch status"
    throw
}

# ============================================================================
# Step 10: Close Batch
# ============================================================================
Write-Step "Step 10: Close Batch"

try {
    $closedBatch = Invoke-RCApi -Method POST -Endpoint "/api/v1/batches/$BATCH_ID/close"
    Write-Success "✓ Batch closed successfully"
    Write-Info "  Final count: $($closedBatch.actual_count)"
}
catch {
    Write-Error "✗ Failed to close batch"
    throw
}

# ============================================================================
# Summary
# ============================================================================
Write-Host @"

╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║                    ✓ ALL TESTS PASSED                        ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

Test Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Health Check
✓ Batch Creation
✓ Blind Scan (2 assets)
✓ Activation (with external SKU mapping)
✓ Verification (consumer scan)
✓ Legal Sale
✓ Transfer (ownership change)
✓ Audit Trail Query
✓ Batch Management

Test Assets:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Asset 1: $ASSET_ID_1
  UID: $TEST_UID_1
  State: Transferred
  Owner: $NEW_OWNER_ID

Asset 2: $ASSET_ID_2
  UID: $TEST_UID_2
  State: FactoryLogged

Batch: $BATCH_ID
  Name: $TEST_BATCH_NAME
  Status: Closed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Next Steps:
  1. Check database: docker exec -it rcprotocol-postgres-1 psql -U rcprotocol -d rcprotocol
  2. Query assets: SELECT * FROM assets WHERE uid IN ('$TEST_UID_1', '$TEST_UID_2');
  3. Check events: SELECT * FROM asset_state_events WHERE asset_id = '$ASSET_ID_1';
  4. View webhooks: SELECT * FROM webhook_deliveries ORDER BY created_at DESC LIMIT 5;

"@ -ForegroundColor Green

Write-Host "Test completed at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
