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
use rc_kms::{KeyProvider, SoftwareKms};
use rc_test_helpers::{fixtures::{generate_test_asset_id, generate_test_brand_id, seed_brand}, TestDb};
use serde_json::{json, Value};
use sqlx::PgPool;
use tower::ServiceExt;
use uuid::Uuid;

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
        .merge(rc_api::routes::protocol::write_router())
        .merge(rc_api::routes::brand::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware))
        .with_state(state)
}

struct TestRequestOptions<'a> {
    actor_id: &'a str,
    role: &'a str,
    brand_id: Option<&'a str>,
    with_trace: bool,
    idempotency_key: Option<&'a str>,
    approval_id: Option<&'a str>,
}

fn request_with_actor(method: &str, uri: &str, opts: TestRequestOptions<'_>, body: Value) -> Request<Body> {
    let mut builder = Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", opts.actor_id)
        .header("X-Actor-Role", opts.role)
        .header("Content-Type", "application/json");

    if opts.with_trace {
        builder = builder.header("X-Trace-Id", Uuid::new_v4().to_string());
    }
    if let Some(key) = opts.idempotency_key {
        builder = builder.header("X-Idempotency-Key", key);
    }
    if let Some(approval_id) = opts.approval_id {
        builder = builder.header("X-Approval-Id", approval_id);
    }
    if let Some(brand) = opts.brand_id {
        builder = builder.header("X-Brand-Id", brand);
    }

    builder.body(Body::from(body.to_string())).unwrap()
}

async fn response_body(response: axum::response::Response) -> String {
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    String::from_utf8(bytes.to_vec()).unwrap()
}

#[tokio::test]
async fn test_brand_boundary_violation_returns_403() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let asset_brand = generate_test_brand_id();
    let actor_brand = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &asset_brand, "Asset Brand").await;
    seed_brand(&db, &actor_brand, "Actor Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&asset_brand)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        TestRequestOptions {
            actor_id: "brand-ops-1",
            role: "Brand",
            brand_id: Some(&actor_brand),
            with_trace: true,
            idempotency_key: Some("idem-brand-boundary"),
            approval_id: None,
        },
        json!({"buyer_id": "buyer-001"}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    let body = response_body(resp).await;
    assert!(body.contains("BrandBoundaryViolation") || body.contains("brand boundary") || body.contains("forbidden"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_missing_operational_header_returns_400() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Header Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        TestRequestOptions {
            actor_id: "brand-ops-1",
            role: "Brand",
            brand_id: Some(&brand_id),
            with_trace: false,
            idempotency_key: Some("idem-missing-trace"),
            approval_id: None,
        },
        json!({"buyer_id": "buyer-001"}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    let body = response_body(resp).await;
    assert!(body.contains("X-Trace-Id"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_invalid_state_transition_returns_400() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Transition Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'FactoryLogged', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        TestRequestOptions {
            actor_id: "platform-admin",
            role: "Platform",
            brand_id: None,
            with_trace: true,
            idempotency_key: Some("idem-invalid-transition"),
            approval_id: Some("approval-001"),
        },
        json!({"buyer_id": "buyer-001"}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_idempotency_conflict_returns_409() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Idempotency Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let idem = "idem-conflict-001";
    let req1 = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        TestRequestOptions {
            actor_id: "brand-ops-1",
            role: "Brand",
            brand_id: Some(&brand_id),
            with_trace: true,
            idempotency_key: Some(idem),
            approval_id: None,
        },
        json!({"buyer_id": "buyer-001"}),
    );
    let resp1 = router.clone().oneshot(req1).await.unwrap();
    assert_eq!(resp1.status(), StatusCode::OK);

    let req2 = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        TestRequestOptions {
            actor_id: "brand-ops-1",
            role: "Brand",
            brand_id: Some(&brand_id),
            with_trace: true,
            idempotency_key: Some(idem),
            approval_id: None,
        },
        json!({"buyer_id": "buyer-002"}),
    );
    let resp2 = router.oneshot(req2).await.unwrap();
    assert_eq!(resp2.status(), StatusCode::CONFLICT);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_api_key_hash_authenticates_brand_without_plaintext_key() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms.clone());

    let brand_id = generate_test_brand_id();
    seed_brand(&db, &brand_id, "API Key Hash Brand").await;
    let key_plain = rc_api::auth::api_key::generate_api_key();
    let key_hash = rc_api::auth::api_key::hash_api_key(&key_plain);
    let key_prefix = rc_api::auth::api_key::extract_key_prefix(&key_plain);
    let key_id = rc_api::auth::api_key::generate_key_id();

    let mut tx = db.begin().await.unwrap();
    rc_api::db::brands::create_api_key(&mut tx, &key_id, &brand_id, &key_hash, &key_prefix)
        .await
        .unwrap();
    tx.commit().await.unwrap();

    let req = Request::builder()
        .method("GET")
        .uri(format!("/brands/{}", brand_id))
        .header("X-Api-Key-Hash", key_hash)
        .header("X-Api-Key-Verified", "hash-only")
        .body(Body::empty())
        .unwrap();

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = response_body(resp).await;
    let json: Value = serde_json::from_str(&body).unwrap();
    assert_eq!(json["brand_id"], brand_id);

    test_db.cleanup().await;
}
