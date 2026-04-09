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

fn request_with_actor(method: &str, uri: &str, actor_id: &str, role: &str, body: Value) -> Request<Body> {
    Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", actor_id)
        .header("X-Actor-Role", role)
        .header("X-Trace-Id", Uuid::new_v4().to_string())
        .header("X-Idempotency-Key", format!("idem-{}", nanoid::nanoid!(12)))
        .header("Content-Type", "application/json")
        .body(Body::from(body.to_string()))
        .unwrap()
}

async fn parse_json(response: axum::response::Response) -> Value {
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    serde_json::from_slice(&bytes).unwrap()
}

#[tokio::test]
async fn test_consume_transitions_legally_sold_to_consumed_and_clears_owner() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Terminal Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, owner_id, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'LegallySold', 'user-001', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/consume", asset_id),
        "user-001",
        "Consumer",
        json!({"previous_state": null}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "Consumed");

    let row: (String, Option<String>) = sqlx::query_as("SELECT current_state, owner_id FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(row.0, "Consumed");
    assert!(row.1.is_none());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_legacy_transitions_transferred_to_legacy_and_clears_owner() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Legacy Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, owner_id, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Transferred', 'user-001', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/legacy", asset_id),
        "user-001",
        "Consumer",
        json!({"previous_state": null}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "Legacy");

    let row: (String, Option<String>) = sqlx::query_as("SELECT current_state, owner_id FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(row.0, "Legacy");
    assert!(row.1.is_none());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_mark_tampered_from_activated() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Risk Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/mark-tampered", asset_id),
        "mod-001",
        "Moderator",
        json!({"previous_state": null}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "Tampered");

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_mark_compromised_from_disputed() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Compromised Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Disputed', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/mark-compromised", asset_id),
        "mod-001",
        "Moderator",
        json!({"previous_state": null}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "Compromised");

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_mark_destructed_from_activated() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Destructed Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/mark-destructed", asset_id),
        "mod-001",
        "Moderator",
        json!({"previous_state": null}),
    );
    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "Destructed");

    test_db.cleanup().await;
}
