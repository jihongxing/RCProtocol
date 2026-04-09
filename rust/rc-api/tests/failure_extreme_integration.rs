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
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware))
        .with_state(state)
}

struct ReqOpts<'a> {
    actor_id: &'a str,
    role: &'a str,
    brand_id: Option<&'a str>,
    trace_id: Option<&'a str>,
    idempotency_key: Option<&'a str>,
    approval_id: Option<&'a str>,
}

fn req(method: &str, uri: &str, opts: ReqOpts<'_>, body: Value) -> Request<Body> {
    let mut builder = Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", opts.actor_id)
        .header("X-Actor-Role", opts.role)
        .header("Content-Type", "application/json");

    if let Some(v) = opts.trace_id {
        builder = builder.header("X-Trace-Id", v);
    }
    if let Some(v) = opts.idempotency_key {
        builder = builder.header("X-Idempotency-Key", v);
    }
    if let Some(v) = opts.approval_id {
        builder = builder.header("X-Approval-Id", v);
    }
    if let Some(v) = opts.brand_id {
        builder = builder.header("X-Brand-Id", v);
    }

    builder.body(Body::from(body.to_string())).unwrap()
}

async fn body_text(response: axum::response::Response) -> String {
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    String::from_utf8(bytes.to_vec()).unwrap()
}

#[tokio::test]
async fn test_terminal_state_rejects_business_action() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Terminal Guard").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Consumed', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let resp = router.oneshot(req(
        "POST",
        &format!("/assets/{}/legacy", asset_id),
        ReqOpts {
            actor_id: "user-001",
            role: "Consumer",
            brand_id: None,
            trace_id: Some(&Uuid::new_v4().to_string()),
            idempotency_key: Some("idem-terminal"),
            approval_id: None,
        },
        json!({}),
    )).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    assert!(body_text(resp).await.contains("terminal"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_frozen_asset_blocks_business_action() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Frozen Guard").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Disputed', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let resp = router.oneshot(req(
        "POST",
        &format!("/assets/{}/consume", asset_id),
        ReqOpts {
            actor_id: "user-001",
            role: "Consumer",
            brand_id: None,
            trace_id: Some(&Uuid::new_v4().to_string()),
            idempotency_key: Some("idem-frozen"),
            approval_id: None,
        },
        json!({}),
    )).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    assert!(body_text(resp).await.contains("frozen"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_platform_business_action_without_approval_is_403() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Approval Guard").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let resp = router.oneshot(req(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        ReqOpts {
            actor_id: "platform-admin",
            role: "Platform",
            brand_id: None,
            trace_id: Some(&Uuid::new_v4().to_string()),
            idempotency_key: Some("idem-no-approval"),
            approval_id: None,
        },
        json!({"buyer_id": "buyer-001"}),
    )).await.unwrap();
    assert_eq!(resp.status(), StatusCode::FORBIDDEN);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_legal_sell_without_buyer_id_is_400() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Buyer Guard").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let resp = router.oneshot(req(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        ReqOpts {
            actor_id: "brand-ops-1",
            role: "Brand",
            brand_id: Some(&brand_id),
            trace_id: Some(&Uuid::new_v4().to_string()),
            idempotency_key: Some("idem-no-buyer"),
            approval_id: None,
        },
        json!({}),
    )).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    assert!(body_text(resp).await.contains("buyer_id"));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_brand_role_without_x_brand_id_is_401() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Header Guard").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let response = router.oneshot(Request::builder()
        .method("POST")
        .uri(format!("/assets/{}/legal-sell", asset_id))
        .header("Authorization", "brand-ops-1")
        .header("X-Actor-Role", "Brand")
        .header("X-Trace-Id", Uuid::new_v4().to_string())
        .header("X-Idempotency-Key", "idem-brand-header")
        .header("Content-Type", "application/json")
        .body(Body::from(json!({"buyer_id": "buyer-001"}).to_string()))
        .unwrap()).await.unwrap();

    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    assert!(body_text(response).await.contains("X-Brand-Id"));

    test_db.cleanup().await;
}
