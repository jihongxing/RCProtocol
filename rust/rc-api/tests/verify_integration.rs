use std::sync::Arc;

use axum::{body::Body, http::Request, Router};
use dashmap::DashMap;
use http_body_util::BodyExt;
use rc_common::ids;
use rc_kms::{KeyProvider, SoftwareKms};
use rc_test_helpers::{fixtures::{generate_test_brand_id, generate_test_product_id, seed_brand, seed_product}, TestDb};
use serde_json::Value;
use sqlx::PgPool;
use tower::ServiceExt;

async fn ensure_attestation_tables(db: &PgPool) {
    sqlx::query("CREATE TABLE IF NOT EXISTS platform_attestations (attestation_id TEXT PRIMARY KEY, version TEXT NOT NULL, platform_id TEXT NOT NULL, asset_commitment_id TEXT NOT NULL, statement TEXT NOT NULL, key_id TEXT NOT NULL, canonical_payload JSONB NOT NULL, signature TEXT NOT NULL, issued_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(asset_commitment_id, statement))").execute(db).await.unwrap();
    sqlx::query("CREATE INDEX IF NOT EXISTS idx_platform_attestations_platform_id ON platform_attestations(platform_id)").execute(db).await.unwrap();
    sqlx::query("CREATE INDEX IF NOT EXISTS idx_platform_attestations_commitment ON platform_attestations(asset_commitment_id)").execute(db).await.unwrap();
}

fn test_router(db: PgPool, kms: Arc<dyn KeyProvider + Send + Sync>) -> Router {
    let jwt_decoder = Arc::new(rc_api::auth::jwt::JwtDecoder::new(b"test-placeholder"));
    let state = rc_api::app::AppState {
        db,
        kms,
        jwt_decoder,
        auth_disabled: true,
        redis: None,
        ctr_cache: Arc::new(DashMap::new()),
        fallback_strategy: rc_api::app::FallbackStrategy::DirectPg,
        api_key_secret: b"test-api-key-secret".to_vec(),
    };
    Router::new()
        .merge(rc_api::routes::verify::router())
        .with_state(state)
}

const TEST_ROOT_KEY_HEX: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const TEST_SYSTEM_ID: &str = "test-system";
const TEST_UID_HEX: &str = "04A31B2C3D4E5F";
const TEST_UID_BYTES: [u8; 7] = [0x04, 0xA3, 0x1B, 0x2C, 0x3D, 0x4E, 0x5F];

fn setup_kms() -> Arc<dyn KeyProvider + Send + Sync> {
    std::env::set_var("RC_ROOT_KEY_HEX", TEST_ROOT_KEY_HEX);
    std::env::set_var("RC_SYSTEM_ID", TEST_SYSTEM_ID);
    Arc::new(SoftwareKms::from_env().expect("test KMS init"))
}

fn compute_test_cmac(
    kms: &dyn KeyProvider,
    brand_id: &str,
    uid: &[u8; 7],
    ctr: &[u8; 3],
    epoch: u32,
) -> [u8; 8] {
    let k_chip = kms.derive_chip_key(brand_id, uid, epoch).expect("derive K_chip");
    let mut msg = [0u8; 12];
    msg[0] = 0x3C;
    msg[1] = 0xC3;
    msg[2..9].copy_from_slice(uid);
    msg[9..12].copy_from_slice(ctr);
    rc_crypto::cmac_aes128::compute_truncated(k_chip.as_bytes(), &msg)
}

async fn setup_db() -> (TestDb, PgPool, String, String) {
    let test_db = TestDb::new().await;
    let pool = test_db.pool().clone();

    let brand_id = generate_test_brand_id();
    let product_id = generate_test_product_id();
    assert!(brand_id.starts_with("brand_"));
    assert!(product_id.starts_with("product_"));

    seed_brand(&pool, &brand_id, "RC Verify Test Brand").await;
    seed_product(&pool, &product_id, &brand_id, "RC Verify Test Product").await;

    (test_db, pool, brand_id, product_id)
}

async fn seed_asset(
    pool: &PgPool,
    brand_id: &str,
    product_id: &str,
    asset_id: &str,
    uid: &str,
    state: &str,
    previous_state: Option<&str>,
) {
    sqlx::query("DELETE FROM verification_events WHERE uid = $1")
        .bind(uid)
        .execute(pool)
        .await
        .unwrap();
    sqlx::query("DELETE FROM assets WHERE asset_id = $1")
        .bind(asset_id)
        .execute(pool)
        .await
        .unwrap();

    sqlx::query("INSERT INTO assets (asset_id, brand_id, product_id, uid, current_state, previous_state, epoch, last_verified_ctr) VALUES ($1, $2, $3, $4, $5, $6, 0, NULL)")
        .bind(asset_id)
        .bind(brand_id)
        .bind(product_id)
        .bind(uid)
        .bind(state)
        .bind(previous_state)
        .execute(pool)
        .await
        .unwrap();
}

async fn get_json(app: Router, uri: &str) -> (u16, Value) {
    let req = Request::builder().uri(uri).body(Body::empty()).unwrap();
    let resp = app.oneshot(req).await.unwrap();
    let status = resp.status().as_u16();
    let body = resp.into_body().collect().await.unwrap().to_bytes();
    let json: Value = serde_json::from_slice(&body).unwrap();
    (status, json)
}

#[tokio::test]
async fn test_verify_valid_cmac() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let asset_id = ids::generate_asset_id();
    assert!(asset_id.starts_with("asset_"));
    seed_asset(&pool, &brand_id, &product_id, &asset_id, TEST_UID_HEX, "Activated", None).await;

    let ctr: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac = compute_test_cmac(kms.as_ref(), &brand_id, &TEST_UID_BYTES, &ctr, 0);
    let ctr_hex = hex::encode(ctr);
    let cmac_hex = hex::encode(cmac);

    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify?uid={TEST_UID_HEX}&ctr={ctr_hex}&cmac={cmac_hex}");
    let (status, json) = get_json(app, &uri).await;

    assert_eq!(status, 200);
    assert_eq!(json["verification_status"], "verified");
    assert!(json["risk_flags"].as_array().unwrap().is_empty());
    assert!(json["asset"].is_object());
    assert_eq!(json["asset"]["asset_id"], asset_id);
    assert_eq!(json["asset"]["brand_id"], brand_id);
    assert!(json["scan_metadata"].is_object());
    assert_eq!(json["scan_metadata"]["ctr"], 1);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_invalid_cmac() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let asset_id = ids::generate_asset_id();
    seed_asset(&pool, &brand_id, &product_id, &asset_id, TEST_UID_HEX, "Activated", None).await;

    let ctr: [u8; 3] = [0x01, 0x00, 0x00];
    let mut cmac = compute_test_cmac(kms.as_ref(), &brand_id, &TEST_UID_BYTES, &ctr, 0);
    cmac[0] ^= 0xFF;

    let ctr_hex = hex::encode(ctr);
    let cmac_hex = hex::encode(cmac);

    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify?uid={TEST_UID_HEX}&ctr={ctr_hex}&cmac={cmac_hex}");
    let (status, json) = get_json(app, &uri).await;

    assert_eq!(status, 200);
    assert_eq!(json["verification_status"], "authentication_failed");
    assert!(json["asset"].is_null());
    assert!(json["scan_metadata"].is_null());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_unknown_uid() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, _) = setup_db().await;

    let unknown_uid = "04FFFFFFFFFFFF";
    let uid_bytes: [u8; 7] = [0x04, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF];
    let ctr: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac = compute_test_cmac(kms.as_ref(), &brand_id, &uid_bytes, &ctr, 0);
    let ctr_hex = hex::encode(ctr);
    let cmac_hex = hex::encode(cmac);

    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify?uid={unknown_uid}&ctr={ctr_hex}&cmac={cmac_hex}");
    let (status, json) = get_json(app, &uri).await;

    assert_eq!(status, 200);
    assert_eq!(json["verification_status"], "unknown_tag");
    assert!(json["asset"].is_null());
    assert!(json["scan_metadata"].is_null());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_ctr_increment() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let uid_hex = "04C53D4E5F6071";
    let uid_bytes: [u8; 7] = [0x04, 0xC5, 0x3D, 0x4E, 0x5F, 0x60, 0x71];
    let asset_id = ids::generate_asset_id();
    seed_asset(&pool, &brand_id, &product_id, &asset_id, uid_hex, "Activated", None).await;

    let ctr1: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac1 = compute_test_cmac(kms.as_ref(), &brand_id, &uid_bytes, &ctr1, 0);
    let app = test_router(pool.clone(), kms.clone());
    let uri = format!("/verify?uid={uid_hex}&ctr={}&cmac={}", hex::encode(ctr1), hex::encode(cmac1));
    let (_, json1) = get_json(app, &uri).await;
    assert_eq!(json1["verification_status"], "verified");
    assert!(json1["scan_metadata"]["previous_ctr"].is_null());

    let ctr2: [u8; 3] = [0x02, 0x00, 0x00];
    let cmac2 = compute_test_cmac(kms.as_ref(), &brand_id, &uid_bytes, &ctr2, 0);
    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify?uid={uid_hex}&ctr={}&cmac={}", hex::encode(ctr2), hex::encode(cmac2));
    let (_, json2) = get_json(app, &uri).await;
    assert_eq!(json2["verification_status"], "verified");
    assert_eq!(json2["scan_metadata"]["previous_ctr"], 1);
    assert!(json2["risk_flags"].as_array().unwrap().is_empty());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_ctr_replay() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let uid_hex = "04D64E5F607182";
    let uid_bytes: [u8; 7] = [0x04, 0xD6, 0x4E, 0x5F, 0x60, 0x71, 0x82];
    let asset_id = ids::generate_asset_id();
    seed_asset(&pool, &brand_id, &product_id, &asset_id, uid_hex, "Activated", None).await;

    let ctr1: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac1 = compute_test_cmac(kms.as_ref(), &brand_id, &uid_bytes, &ctr1, 0);
    let app = test_router(pool.clone(), kms.clone());
    let uri = format!("/verify?uid={uid_hex}&ctr={}&cmac={}", hex::encode(ctr1), hex::encode(cmac1));
    let (_, json1) = get_json(app, &uri).await;
    assert_eq!(json1["verification_status"], "verified");

    let app = test_router(pool.clone(), kms);
    let (_, json2) = get_json(app, &uri).await;
    assert_eq!(json2["verification_status"], "replay_suspected");
    assert!(json2["risk_flags"].as_array().unwrap().iter().any(|f| f == "replay_suspected"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_disputed_asset() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let uid_hex = "04E75F60718293";
    let uid_bytes: [u8; 7] = [0x04, 0xE7, 0x5F, 0x60, 0x71, 0x82, 0x93];
    let asset_id = ids::generate_asset_id();
    seed_asset(&pool, &brand_id, &product_id, &asset_id, uid_hex, "Disputed", Some("Activated")).await;

    let ctr: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac = compute_test_cmac(kms.as_ref(), &brand_id, &uid_bytes, &ctr, 0);
    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify?uid={uid_hex}&ctr={}&cmac={}", hex::encode(ctr), hex::encode(cmac));
    let (_, json) = get_json(app, &uri).await;

    assert_eq!(json["verification_status"], "restricted");
    assert!(json["risk_flags"].as_array().unwrap().iter().any(|f| f == "frozen_asset"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_verify_missing_params() {
    let kms = setup_kms();
    let (test_db, pool, _, _) = setup_db().await;

    let app = test_router(pool, kms);
    let (status, _) = get_json(app, "/verify").await;
    assert_eq!(status, 400);

    test_db.cleanup().await;
}


#[tokio::test]
async fn test_verify_v2_returns_attestation_summary() {
    let kms = setup_kms();
    let (test_db, pool, brand_id, product_id) = setup_db().await;
    let asset_id = ids::generate_asset_id();
    let commitment_id = "commitment_test_v2";
    ensure_attestation_tables(&pool).await;
    seed_asset(&pool, &brand_id, &product_id, &asset_id, TEST_UID_HEX, "Activated", None).await;

    sqlx::query("UPDATE assets SET asset_commitment_id = $2 WHERE asset_id = $1")
        .bind(&asset_id)
        .bind(commitment_id)
        .execute(&pool)
        .await
        .unwrap();

    sqlx::query("INSERT INTO brand_attestations (attestation_id, version, brand_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at) VALUES ($1, 'ba_v1', $2, $3, 'brand_issues_asset', 'brand-key-2026-01', '{}'::jsonb, 'sig', NOW())")
        .bind("ba_test_v2")
        .bind(&brand_id)
        .bind(commitment_id)
        .execute(&pool)
        .await
        .unwrap();

    sqlx::query("INSERT INTO platform_attestations (attestation_id, version, platform_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at) VALUES ($1, 'pa_v1', 'test-system', $2, 'platform_accepts_asset', 'platform-key-2026-01', '{}'::jsonb, 'sig', NOW())")
        .bind("pa_test_v2")
        .bind(commitment_id)
        .execute(&pool)
        .await
        .unwrap();

    let ctr: [u8; 3] = [0x01, 0x00, 0x00];
    let cmac = compute_test_cmac(kms.as_ref(), &brand_id, &TEST_UID_BYTES, &ctr, 0);
    let app = test_router(pool.clone(), kms);
    let uri = format!("/verify/v2?uid={TEST_UID_HEX}&ctr={}&cmac={}", hex::encode(ctr), hex::encode(cmac));
    let (status, json) = get_json(app, &uri).await;

    assert_eq!(status, 200);
    assert_eq!(json["verification_status"], "verified");
    assert_eq!(json["asset"]["asset_commitment_id"], commitment_id);
    assert_eq!(json["attestation_summary"]["asset_commitment_id"], commitment_id);
    assert_eq!(json["attestation_summary"]["brand_attestation_status"], "issued");
    assert_eq!(json["attestation_summary"]["platform_attestation_status"], "issued");

    test_db.cleanup().await;
}
