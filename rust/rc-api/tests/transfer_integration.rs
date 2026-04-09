use std::sync::Arc;

use axum::{
    body::Body,
    http::{Request, StatusCode},
    middleware as axum_mw,
    Router,
};
use dashmap::DashMap;
use http_body_util::BodyExt;
use rc_api::auth::middleware::{auth_middleware, AuthState};
use rc_common::ids;
use rc_kms::{KeyProvider, SoftwareKms};
use rc_test_helpers::{fixtures::{generate_test_brand_id, seed_brand}, TestDb};
use serde_json::{json, Value};
use sqlx::PgPool;
use tower::ServiceExt;

const TEST_ROOT_KEY_HEX: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const TEST_SYSTEM_ID: &str = "test-system";

fn setup_kms() -> Arc<dyn KeyProvider + Send + Sync> {
    std::env::set_var("RC_ROOT_KEY_HEX", TEST_ROOT_KEY_HEX);
    std::env::set_var("RC_SYSTEM_ID", TEST_SYSTEM_ID);
    Arc::new(SoftwareKms::from_env().expect("test KMS init"))
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
    let auth_state = AuthState::from(&state);
    Router::new()
        .merge(rc_api::routes::transfer::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware))
        .with_state(state)
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

#[tokio::test]
async fn test_transfer_with_virtual_authority_succeeds() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms.clone());

    let brand_id = generate_test_brand_id();
    let asset_id = ids::generate_asset_id();
    let authority_uid = format!("vauth-{}", nanoid::nanoid!(12));
    let current_owner = "user-001";
    let new_owner = "user-002";
    let child_uid_hex = "04A31B2C3D4E5F";
    let child_uid_bytes: [u8; 7] = [0x04, 0xA3, 0x1B, 0x2C, 0x3D, 0x4E, 0x5F];
    let child_ctr: [u8; 3] = [0x01, 0x00, 0x00];

    seed_brand(&db, &brand_id, "Transfer Test Brand").await;
    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, owner_id, epoch) \
         VALUES ($1, $2, $3, 'LegallySold', 'Activated', $4, 0)"
    )
    .bind(&asset_id)
    .bind(&brand_id)
    .bind(child_uid_hex)
    .bind(current_owner)
    .execute(&db)
    .await
    .unwrap();

    let k_chip_mother = kms.derive_mother_key(&brand_id, authority_uid.as_bytes(), 0).unwrap();
    let credential_hash = hex::encode(rc_crypto::hmac_sha256::compute(
        k_chip_mother.as_bytes(),
        authority_uid.as_bytes(),
    ));

    let authority_id: uuid::Uuid = sqlx::query_scalar(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id, created_by) \
         VALUES ($1, 'VIRTUAL_APP', $2, 0, $3, $4, $4) RETURNING authority_id"
    )
    .bind(&authority_uid)
    .bind(&brand_id)
    .bind(&credential_hash)
    .bind(current_owner)
    .fetch_one(&db)
    .await
    .unwrap();

    sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) \
         VALUES ($1, $2, $3, $4, 'Active', $5)"
    )
    .bind(&asset_id)
    .bind(child_uid_hex)
    .bind(authority_id)
    .bind(&authority_uid)
    .bind(current_owner)
    .execute(&db)
    .await
    .unwrap();

    let child_cmac = compute_test_cmac(kms.as_ref(), &brand_id, &child_uid_bytes, &child_ctr, 0);

    let req = Request::builder()
        .method("POST")
        .uri(format!("/assets/{}/transfer", asset_id))
        .header("Content-Type", "application/json")
        .header("Authorization", current_owner)
        .header("X-Actor-Role", "Consumer")
        .header("X-Idempotency-Key", format!("idem-{}", nanoid::nanoid!(12)))
        .header("X-Trace-Id", uuid::Uuid::new_v4().to_string())
        .body(Body::from(
            json!({
                "new_owner_id": new_owner,
                "child_uid": child_uid_hex,
                "child_ctr": hex::encode(child_ctr),
                "child_cmac": hex::encode(child_cmac),
                "authority_proof": {
                    "type": "virtual_token",
                    "user_id": current_owner,
                    "credential_token": credential_hash
                }
            })
            .to_string(),
        ))
        .unwrap();

    let response = router.oneshot(req).await.unwrap();
    let status = response.status();
    let body_bytes = response.into_body().collect().await.unwrap().to_bytes();
    let body_text = String::from_utf8(body_bytes.to_vec()).unwrap();
    assert_eq!(status, StatusCode::OK, "transfer failed body: {}", body_text);

    let body: Value = serde_json::from_str(&body_text).unwrap();
    assert_eq!(body["asset_id"], asset_id);
    assert_eq!(body["from_owner_id"], current_owner);
    assert_eq!(body["to_owner_id"], new_owner);
    assert_eq!(body["new_state"], "Transferred");

    let owner: Option<String> = sqlx::query_scalar("SELECT owner_id FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_optional(&db)
        .await
        .unwrap()
        .flatten();
    assert_eq!(owner.as_deref(), Some(new_owner));

    let transfer_row: Option<(String, String, String)> = sqlx::query_as(
        "SELECT asset_id, from_owner_id, to_owner_id FROM asset_transfers WHERE asset_id = $1"
    )
    .bind(&asset_id)
    .fetch_optional(&db)
    .await
    .unwrap();
    assert_eq!(transfer_row, Some((asset_id.clone(), current_owner.to_string(), new_owner.to_string())));

    test_db.cleanup().await;
}
