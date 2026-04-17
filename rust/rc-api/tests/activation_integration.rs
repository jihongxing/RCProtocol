use std::sync::Arc;

use axum::{
    body::Body,
    http::{Request, StatusCode},
    middleware as axum_mw,
    Router,
};
use dashmap::DashMap;
use http_body_util::BodyExt;
use rc_kms::{KeyProvider, SoftwareKms};
use rc_test_helpers::{fixtures::{generate_test_asset_id, generate_test_batch_id, generate_test_brand_id, seed_brand}, TestDb};
use rc_api::auth::middleware::{auth_middleware, AuthState};
use rc_api::db::asset_commitments::{fetch_asset_commitment_by_id, fetch_asset_commitment_by_uid_epoch};
use serde_json::{json, Value};
use sqlx::PgPool;
use tower::ServiceExt;
use uuid::Uuid;

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
    let auth_state = AuthState::from(&state);
    Router::new()
        .merge(rc_api::routes::protocol::write_router())
        .merge(rc_api::routes::assets::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware))
        .with_state(state)
}

const TEST_ROOT_KEY_HEX: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const TEST_SYSTEM_ID: &str = "test-system";

fn setup_kms() -> Arc<dyn KeyProvider + Send + Sync> {
    std::env::set_var("RC_ROOT_KEY_HEX", TEST_ROOT_KEY_HEX);
    std::env::set_var("RC_SYSTEM_ID", TEST_SYSTEM_ID);
    Arc::new(SoftwareKms::from_env().expect("test KMS init"))
}

#[tokio::test]
async fn test_activation_entangle_creates_virtual_mother_card() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    ensure_attestation_tables(&db).await;
    let router = test_router(db.clone(), kms.clone());

    let test_brand_id = generate_test_brand_id();
    assert!(test_brand_id.starts_with("brand_"));
    let asset_id = generate_test_asset_id();
    assert!(asset_id.starts_with("asset_"));
    let test_uid = "04A31B2C3D4E5F";
    let actor_id = "test-user-001";

    seed_brand(&db, &test_brand_id, "Test Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, $3, 'Unassigned', 0)")
        .bind(&asset_id)
        .bind(&test_brand_id)
        .bind(test_uid)
        .execute(&db).await.unwrap();

    let activate_req = Request::builder()
        .method("POST")
        .uri(format!("/assets/{}/activate", asset_id))
        .header("Content-Type", "application/json")
        .header("X-Trace-Id", Uuid::new_v4().to_string())
        .header("X-Idempotency-Key", format!("idem-{}", nanoid::nanoid!(12)))
        .header("Authorization", actor_id)
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::from(json!({
            "external_product_id": "sku-entangle-001",
            "external_product_name": "Entangle Product",
            "external_product_url": "https://example.com/p/entangle"
        }).to_string()))
        .unwrap();

    let activate_resp = router.clone().oneshot(activate_req).await.unwrap();
    assert_eq!(activate_resp.status(), StatusCode::OK);
    let activate_body = activate_resp.into_body().collect().await.unwrap().to_bytes();
    let activate_json: Value = serde_json::from_slice(&activate_body).unwrap();
    assert_eq!(activate_json["to_state"], "RotatingKeys");
    assert!(activate_json.get("virtual_mother_card").is_none());

    let detail_before_req = Request::builder()
        .method("GET")
        .uri(format!("/assets/{}", asset_id))
        .header("Authorization", actor_id)
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::empty())
        .unwrap();
    let detail_before_resp = router.clone().oneshot(detail_before_req).await.unwrap();
    assert_eq!(detail_before_resp.status(), StatusCode::OK);
    let detail_before_body = detail_before_resp.into_body().collect().await.unwrap().to_bytes();
    let detail_before_json: Value = serde_json::from_slice(&detail_before_body).unwrap();
    assert!(detail_before_json["virtual_mother_card"].is_null());

    let entangle_req = Request::builder()
        .method("POST")
        .uri(format!("/assets/{}/activate-entangle", asset_id))
        .header("Content-Type", "application/json")
        .header("X-Trace-Id", Uuid::new_v4().to_string())
        .header("X-Idempotency-Key", format!("idem-{}", nanoid::nanoid!(12)))
        .header("Authorization", actor_id)
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::from(json!({"previous_state": null}).to_string()))
        .unwrap();

    let response = router.clone().oneshot(entangle_req).await.unwrap();
    assert_eq!(response.status(), StatusCode::OK, "ActivateEntangle should succeed");

    let body_bytes = response.into_body().collect().await.unwrap().to_bytes();
    let body: Value = serde_json::from_slice(&body_bytes).unwrap();
    assert_eq!(body["to_state"], "EntangledPending");
    assert!(body["virtual_mother_card"].is_object());

    let authority_row: Option<(String, String, String, i32, Option<String>, Option<String>)> = sqlx::query_as(
        "SELECT authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id \
         FROM authority_devices \
         WHERE authority_uid LIKE 'vauth-%' AND brand_id = $1 \
         ORDER BY created_at DESC LIMIT 1"
    )
    .bind(&test_brand_id)
    .fetch_optional(&db)
    .await
    .unwrap();

    assert!(authority_row.is_some(), "authority_devices record should exist");
    let (authority_uid, authority_type, brand_id, key_epoch, credential_hash, bound_user_id) = authority_row.unwrap();
    assert_eq!(authority_type, "VIRTUAL_APP");
    assert_eq!(brand_id, test_brand_id);
    assert_eq!(key_epoch, 0);
    assert!(credential_hash.is_some(), "virtual_credential_hash should be set");
    assert_eq!(bound_user_id.as_deref(), Some(actor_id));

    let entanglement_row: Option<(String, String, String, String)> = sqlx::query_as(
        "SELECT asset_id, child_uid, authority_uid, entanglement_state \
         FROM asset_entanglements \
         WHERE asset_id = $1"
    )
    .bind(&asset_id)
    .fetch_optional(&db)
    .await
    .unwrap();

    assert!(entanglement_row.is_some(), "asset_entanglements record should exist");
    let (ent_asset_id, child_uid, ent_authority_uid, state) = entanglement_row.unwrap();
    assert_eq!(ent_asset_id, asset_id);
    assert_eq!(child_uid, test_uid);
    assert_eq!(ent_authority_uid, authority_uid);
    assert_eq!(state, "Active");

    let k_chip_mother = kms.derive_mother_key(&test_brand_id, authority_uid.as_bytes(), 0).unwrap();
    let expected_hash_bytes = rc_crypto::hmac_sha256::compute(k_chip_mother.as_bytes(), authority_uid.as_bytes());
    let expected_hash = hex::encode(expected_hash_bytes);
    assert_eq!(credential_hash.unwrap(), expected_hash, "credential hash should match HMAC-SHA256(K_chip_mother, authority_uid)");

    let detail_after_req = Request::builder()
        .method("GET")
        .uri(format!("/assets/{}", asset_id))
        .header("Authorization", actor_id)
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::empty())
        .unwrap();
    let detail_after_resp = router.oneshot(detail_after_req).await.unwrap();
    assert_eq!(detail_after_resp.status(), StatusCode::OK);
    let detail_after_body = detail_after_resp.into_body().collect().await.unwrap().to_bytes();
    let detail_after_json: Value = serde_json::from_slice(&detail_after_body).unwrap();
    assert_eq!(detail_after_json["current_state"], "EntangledPending");
    assert_eq!(detail_after_json["virtual_mother_card"]["authority_uid"], authority_uid);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_activate_generates_asset_commitment() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    ensure_attestation_tables(&db).await;
    let router = test_router(db.clone(), kms);

    let test_brand_id = generate_test_brand_id();
    let test_batch_id = generate_test_batch_id();
    let asset_id = generate_test_asset_id();
    let test_uid = "04B41C2D3E4F60";

    seed_brand(&db, &test_brand_id, "Test Brand").await;
    sqlx::query("INSERT INTO batches (batch_id, brand_id) VALUES ($1, $2)")
        .bind(&test_batch_id)
        .bind(&test_brand_id)
        .execute(&db)
        .await
        .unwrap();
    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, uid, batch_id, current_state, key_epoch, epoch) VALUES ($1, $2, $3, $4, 'Unassigned', 0, 0)"
    )
    .bind(&asset_id)
    .bind(&test_brand_id)
    .bind(test_uid)
    .bind(&test_batch_id)
    .execute(&db)
    .await
    .unwrap();

    let trace_id = Uuid::new_v4();
    let idempotency_key = format!("idem-{}", nanoid::nanoid!(12));

    let req = Request::builder()
        .method("POST")
        .uri(format!("/assets/{}/activate", asset_id))
        .header("Content-Type", "application/json")
        .header("X-Trace-Id", trace_id.to_string())
        .header("X-Idempotency-Key", &idempotency_key)
        .header("Authorization", "brand-user-001")
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::from(json!({
            "external_product_id": "sku_001",
            "external_product_name": "Asset Product",
            "external_product_url": "https://example.com/p/sku_001"
        }).to_string()))
        .unwrap();

    let response = router.clone().oneshot(req).await.unwrap();
    let status = response.status();
    if status != StatusCode::OK {
        let body_bytes = response.into_body().collect().await.unwrap().to_bytes();
        panic!("Activate should succeed, got {}: {}", status, String::from_utf8_lossy(&body_bytes));
    }

    let body_bytes = response.into_body().collect().await.unwrap().to_bytes();
    let body: Value = serde_json::from_slice(&body_bytes).unwrap();
    let commitment_id = body["asset_commitment_id"].as_str().unwrap().to_string();
    assert!(!commitment_id.is_empty());
    assert_eq!(body["to_state"], "RotatingKeys");
    assert!(body.get("virtual_mother_card").is_none());

    let asset_row: (Option<String>, Option<String>, Option<String>) = sqlx::query_as(
        "SELECT asset_commitment_id, external_product_id, batch_id FROM assets WHERE asset_id = $1"
    )
    .bind(&asset_id)
    .fetch_one(&db)
    .await
    .unwrap();
    assert_eq!(asset_row.0.as_deref(), Some(commitment_id.as_str()));
    assert_eq!(asset_row.1.as_deref(), Some("sku_001"));
    assert_eq!(asset_row.2.as_deref(), Some(test_batch_id.as_str()));

    let commitment_row: (String, String, String, i32, serde_json::Value) = sqlx::query_as(
        "SELECT brand_id, asset_uid, payload_version, epoch, canonical_payload FROM asset_commitments WHERE commitment_id = $1"
    )
    .bind(&commitment_id)
    .fetch_one(&db)
    .await
    .unwrap();

    assert_eq!(commitment_row.0, test_brand_id);
    assert_eq!(commitment_row.1, test_uid);
    assert_eq!(commitment_row.2, "ac_v1");
    assert_eq!(commitment_row.3, 0);
    assert_eq!(commitment_row.4["version"], "ac_v1");
    assert_eq!(commitment_row.4["asset_uid"], test_uid);
    assert_eq!(commitment_row.4["brand_id"], commitment_row.0);

    let by_uid_epoch = fetch_asset_commitment_by_uid_epoch(&db, test_uid, 0).await.unwrap().unwrap();
    assert_eq!(by_uid_epoch.commitment_id, commitment_id);

    let detail_req = Request::builder()
        .method("GET")
        .uri(format!("/assets/{}", asset_id))
        .header("Authorization", "brand-user-001")
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::empty())
        .unwrap();
    let detail_resp = router.clone().oneshot(detail_req).await.unwrap();
    assert_eq!(detail_resp.status(), StatusCode::OK);
    let detail_body = detail_resp.into_body().collect().await.unwrap().to_bytes();
    let detail_json: Value = serde_json::from_slice(&detail_body).unwrap();
    assert_eq!(detail_json["asset_commitment_id"], commitment_id);
    assert_eq!(detail_json["asset_commitment_payload"]["asset_uid"], test_uid);
    assert_eq!(detail_json["asset_commitment_payload"]["version"], "ac_v1");
    assert_eq!(detail_json["brand_attestation_status"], "issued");
    assert_eq!(detail_json["platform_attestation_status"], "issued");
    assert!(detail_json["virtual_mother_card"].is_null());

    let attestation_req = Request::builder()
        .method("GET")
        .uri(format!("/assets/{}/attestations", asset_id))
        .header("Authorization", "brand-user-001")
        .header("X-Actor-Role", "Brand")
        .header("X-Brand-Id", &test_brand_id)
        .body(Body::empty())
        .unwrap();
    let attestation_resp = router.clone().oneshot(attestation_req).await.unwrap();
    assert_eq!(attestation_resp.status(), StatusCode::OK);
    let attestation_body = attestation_resp.into_body().collect().await.unwrap().to_bytes();
    let attestation_json: Value = serde_json::from_slice(&attestation_body).unwrap();
    assert_eq!(attestation_json["asset_commitment_id"], commitment_id);
    assert_eq!(attestation_json["brand_attestation_status"], "issued");
    assert_eq!(attestation_json["platform_attestation_status"], "issued");
    assert_eq!(attestation_json["brand_attestation"]["statement"], "brand_issues_asset");
    assert_eq!(attestation_json["platform_attestation"]["statement"], "platform_accepts_asset");

    let event_commitment: Option<String> = sqlx::query_scalar(
        "SELECT asset_commitment_id FROM asset_state_events WHERE asset_id = $1 ORDER BY occurred_at DESC LIMIT 1"
    )
    .bind(&asset_id)
    .fetch_optional(&db)
    .await
    .unwrap()
    .flatten();
    assert_eq!(event_commitment.as_deref(), Some(commitment_id.as_str()));

    let commitment_row_full = fetch_asset_commitment_by_id(&db, &commitment_id).await.unwrap();
    assert_eq!(commitment_row_full.commitment_id, commitment_id);

    let brand_attestation: Option<(String, String, String)> = sqlx::query_as(
        "SELECT asset_commitment_id, statement, version FROM brand_attestations WHERE asset_commitment_id = $1"
    )
    .bind(&commitment_id)
    .fetch_optional(&db)
    .await
    .unwrap();
    assert_eq!(brand_attestation.as_ref().map(|r| r.0.as_str()), Some(commitment_id.as_str()));
    assert_eq!(brand_attestation.as_ref().map(|r| r.1.as_str()), Some("brand_issues_asset"));
    assert_eq!(brand_attestation.as_ref().map(|r| r.2.as_str()), Some("ba_v1"));

    let platform_attestation: Option<(String, String, String)> = sqlx::query_as(
        "SELECT asset_commitment_id, statement, version FROM platform_attestations WHERE asset_commitment_id = $1"
    )
    .bind(&commitment_id)
    .fetch_optional(&db)
    .await
    .unwrap();
    assert_eq!(platform_attestation.as_ref().map(|r| r.0.as_str()), Some(commitment_id.as_str()));
    assert_eq!(platform_attestation.as_ref().map(|r| r.1.as_str()), Some("platform_accepts_asset"));
    assert_eq!(platform_attestation.as_ref().map(|r| r.2.as_str()), Some("pa_v1"));

    test_db.cleanup().await;
}
