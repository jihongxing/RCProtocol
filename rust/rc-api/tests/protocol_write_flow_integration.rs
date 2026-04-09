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
use rc_test_helpers::{
    fixtures::{generate_test_asset_id, generate_test_batch_id, generate_test_brand_id, seed_brand},
    TestDb,
};
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

fn request_with_actor(method: &str, uri: &str, actor_id: &str, role: &str, brand_id: Option<&str>, body: Value) -> Request<Body> {
    let mut builder = Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", actor_id)
        .header("X-Actor-Role", role)
        .header("X-Trace-Id", Uuid::new_v4().to_string())
        .header("X-Idempotency-Key", format!("idem-{}", nanoid::nanoid!(12)))
        .header("Content-Type", "application/json");

    if let Some(brand) = brand_id {
        builder = builder.header("X-Brand-Id", brand);
    }

    builder.body(Body::from(body.to_string())).unwrap()
}

async fn parse_json(response: axum::response::Response) -> Value {
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    serde_json::from_slice(&bytes).unwrap()
}

#[tokio::test]
async fn test_blind_scan_creates_factory_logged_asset() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let batch_id = generate_test_batch_id();
    seed_brand(&db, &brand_id, "Blind Scan Brand").await;
    sqlx::query("INSERT INTO batches (batch_id, brand_id, status, actual_count) VALUES ($1, $2, 'Open', 0)")
        .bind(&batch_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        "/assets/blind-scan",
        "factory-bot-1",
        "Factory",
        Some(&brand_id),
        json!({
            "uid": "04ABCD12345678",
            "brand_id": brand_id,
            "batch_id": batch_id,
            "metadata": {"line": "L1"}
        }),
    );

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::CREATED);
    let body = parse_json(resp).await;
    assert_eq!(body["current_state"], "FactoryLogged");

    let asset_id = body["asset_id"].as_str().unwrap();
    let row: (String, String, Option<String>) = sqlx::query_as(
        "SELECT uid, current_state, batch_id FROM assets WHERE asset_id = $1"
    )
    .bind(asset_id)
    .fetch_one(&db)
    .await
    .unwrap();
    assert_eq!(row.0, "04ABCD12345678");
    assert_eq!(row.1, "FactoryLogged");
    assert_eq!(row.2.as_deref(), Some(batch_id.as_str()));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_blind_log_transitions_preminted_to_factory_logged() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Blind Log Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04AA11BB22CC33', 'PreMinted', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/blind-log", asset_id),
        "factory-bot-1",
        "Factory",
        Some(&brand_id),
        json!({"previous_state": null}),
    );

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["from_state"], "PreMinted");
    assert_eq!(body["to_state"], "FactoryLogged");

    let state: String = sqlx::query_scalar("SELECT current_state FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(state, "FactoryLogged");

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_stock_in_transitions_factory_logged_to_unassigned() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Stock In Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04CC22DD33EE44', 'FactoryLogged', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/stock-in", asset_id),
        "factory-bot-1",
        "Factory",
        Some(&brand_id),
        json!({"previous_state": null}),
    );

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["from_state"], "FactoryLogged");
    assert_eq!(body["to_state"], "Unassigned");

    let state: String = sqlx::query_scalar("SELECT current_state FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(state, "Unassigned");

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_activate_confirm_transitions_to_activated() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Confirm Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'EntangledPending', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/activate-confirm", asset_id),
        "brand-ops-1",
        "Brand",
        Some(&brand_id),
        json!({"previous_state": null}),
    );

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["from_state"], "EntangledPending");
    assert_eq!(body["to_state"], "Activated");

    let state: String = sqlx::query_scalar("SELECT current_state FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(state, "Activated");

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_legal_sell_sets_owner_and_legally_sold_state() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    let buyer_id = "buyer-001";
    seed_brand(&db, &brand_id, "Sell Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let req = request_with_actor(
        "POST",
        &format!("/assets/{}/legal-sell", asset_id),
        "brand-ops-1",
        "Brand",
        Some(&brand_id),
        json!({"previous_state": null, "buyer_id": buyer_id}),
    );

    let resp = router.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::OK);
    let body = parse_json(resp).await;
    assert_eq!(body["to_state"], "LegallySold");

    let row: (String, Option<String>) = sqlx::query_as("SELECT current_state, owner_id FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(row.0, "LegallySold");
    assert_eq!(row.1.as_deref(), Some(buyer_id));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_freeze_then_recover_restores_previous_state() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let router = test_router(db.clone(), kms);

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Moderation Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    let freeze_req = request_with_actor(
        "POST",
        &format!("/assets/{}/freeze", asset_id),
        "mod-001",
        "Moderator",
        None,
        json!({"previous_state": null}),
    );
    let freeze_resp = router.clone().oneshot(freeze_req).await.unwrap();
    assert_eq!(freeze_resp.status(), StatusCode::OK);
    let freeze_body = parse_json(freeze_resp).await;
    assert_eq!(freeze_body["to_state"], "Disputed");

    let recover_req = request_with_actor(
        "POST",
        &format!("/assets/{}/recover", asset_id),
        "mod-001",
        "Moderator",
        None,
        json!({"previous_state": null}),
    );
    let recover_resp = router.oneshot(recover_req).await.unwrap();
    assert_eq!(recover_resp.status(), StatusCode::OK);
    let recover_body = parse_json(recover_resp).await;
    assert_eq!(recover_body["from_state"], "Disputed");
    assert_eq!(recover_body["to_state"], "Activated");

    let row: (String, Option<String>) = sqlx::query_as("SELECT current_state, previous_state FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_one(&db)
        .await
        .unwrap();
    assert_eq!(row.0, "Activated");
    assert!(row.1.is_none());

    test_db.cleanup().await;
}
