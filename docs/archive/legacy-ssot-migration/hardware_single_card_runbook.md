# ACR122U + NTAG 424 DNA Single-Card Runbook

This runbook captures the stable, reusable hardware integration path that was verified on a real `ACR122U` reader with an `NTAG 424 DNA` tag.

## Scope

Use this runbook when you have:

- `ACR122U` connected over USB
- one `NTAG 424 DNA` test tag
- local Rust toolchain available
- local `rc-api` test environment available or startable

This runbook is intentionally biased toward reusing existing code paths instead of adding new test code.

## Preferred Entry Points

These repository entry points were validated and are the recommended starting points:

- `crates/rc-core/examples/nfc_diag_minimal.rs`
- `crates/rc-core/tests/hardware_e2e.rs`
- `start-api-dev.ps1`
- `demos/run_single_card_hardware_suite.ps1`

## Verified Stable Sequence

The following sequence was validated on real hardware and is currently the safest single-card path:

1. Minimal diagnostics:
   `cargo run -p rc-core --features nfc-hardware --example nfc_diag_minimal`
2. ATR / reader parsing:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_phase1_atqa_sak_from_atr -- --ignored --exact --test-threads=1 --nocapture`
3. Health check:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_phase3_check_health_on_hardware -- --ignored --exact --test-threads=1 --nocapture`
4. PN532 passthrough:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_phase3_pn532_passthrough_explicit -- --ignored --exact --test-threads=1 --nocapture`
5. Reset to transport baseline:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_reset_card_a_to_transport -- --ignored --exact --test-threads=1 --nocapture`
6. Provision + readback:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_provision_and_sun_read -- --ignored --exact --test-threads=1 --nocapture`
7. Blind scan API path:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_blind_scan_card_a -- --ignored --exact --test-threads=1 --nocapture`
8. Resolve / CMAC verification path:
   `cargo test -p rc-core --test hardware_e2e --features nfc-hardware test_e2e_cmac_verification_unassigned -- --ignored --exact --test-threads=1 --nocapture`

## One-Command Execution

To run the stable single-card suite:

```powershell
powershell -ExecutionPolicy Bypass -File .\demos\run_single_card_hardware_suite.ps1
```

If `rc-api` is already running and you only want the hardware-only subset:

```powershell
powershell -ExecutionPolicy Bypass -File .\demos\run_single_card_hardware_suite.ps1 -SkipApi
```

## Observed Hardware Baseline

The validated environment produced the following practical baseline:

- reader name matched `ACS ACR122 0`
- card ATR was readable and stable
- UID was readable as a 7-byte NXP-style UID
- `GetVersion` identified the tag as `NTAG 424 DNA`
- PN532 passthrough test path executed successfully on the real reader
- `check_health()` passed before and after APDU traffic

## rc-api Test Mode

For API-coupled tests, use the repository's existing test-mode startup script:

```powershell
.\start-api-dev.ps1
```

This script sets:

- `RC_TEST_MODE=1`
- `RC_API_JWT_SECRET`
- `RC_API_KEY_SERVER_SECRET`
- `RC_API_VERIFICATION_TOKEN_SECRET`

and starts:

```powershell
cargo run -p rc-api --features nfc,sqlite
```

## Reset / Recovery

`test_reset_card_a_to_transport` was updated to try multiple candidate key layouts before resetting the card:

- legacy static test key `[0x01..0x10]`
- `E2E-BRAND` / `MasterKey`
- `E2E-BRAND` / `SdmFileKey`
- `TEST_BRAND` / `TEST-ENV` / `MasterKey`
- `TEST_BRAND` / `TEST-ENV` / `SdmFileKey`
- `Transport Key`

This makes the card recovery path much more robust after mixed historical test runs.

## Current Known Issue

One advanced path is still not stable from a fresh transport-state card:

- `test_provision_card_a_first`

Observed behavior:

- `ChangeKey #0` can succeed
- `ChangeKey #2` can fail with `SW=911E`

Practical impact:

- the stable single-card regression suite should prefer the runbook sequence above
- advanced sovereign / replay flows are still reusable when the card is already in the expected E2E layout, but the fresh `Transport Key -> split-key E2E layout` path needs another fix

## Recommended Usage Policy

- Always run hardware tests with `--test-threads=1`
- Keep one dedicated recovery card for reset validation
- Use the single-card suite as the default lab smoke/regression set
- Treat multi-card Card A-E flows as a separate higher-level validation layer
