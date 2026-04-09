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
        .merge(rc_api::routes::assets::router())
        .merge(rc_api::routes::batch::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware))
        .with_state(state)
}

fn request_with_actor(
    method: &str,
    uri: &str,
    actor_id: &str,
    role: &str,
    brand_id: Option<&str>,
    body: Option<Value>,
) -> Request<Body> {
    let mut builder = Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", actor_id)
        .header("X-Actor-Role", role);

    if let Some(brand) = brand_id {
        builder = builder.header("X-Brand-Id", brand);
    }

    match body {
        Some(payload) => builder
            .header("Content-Type", "application/json")
            .body(Body::from(payload.to_string()))
            .unwrap(),
        None => builder.body(Body::empty()).unwrap(),
    }
}

async fn parse_json(response: axum::response::Response) -> Value {
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    serde_json::from_slice(&bytes).unwrap()
}

#[tokio::test]
async fn test_asset_detail_and_history_contract() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let router = test_router(db.clone(), setup_kms());

    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    let trace_id = Uuid::new_v4();
    let batch_id = "batch_contract_001";
    seed_brand(&db, &brand_id, "Asset Contract Brand").await;
    sqlx::query("INSERT INTO batches (batch_id, brand_id, status, actual_count) VALUES ($1, $2, 'Closed', 1)")
        .bind(batch_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, uid, batch_id, external_product_id, external_product_name, external_product_url, current_state, previous_state, owner_id, key_epoch, activated_at, sold_at) \
         VALUES ($1, $2, '04A31B2C3D4E5F', $3, 'sku-001', 'Product 001', 'https://example.test/p/1', 'LegallySold', 'Activated', 'buyer-001', 2, NOW(), NOW())"
    )
    .bind(&asset_id)
    .bind(&brand_id)
    .bind(batch_id)
    .execute(&db)
    .await
    .unwrap();

    sqlx::query(
        "INSERT INTO asset_state_events (event_id, asset_id, action, from_state, to_state, actor_id, actor_role, trace_id, idempotency_key, occurred_at) \
         VALUES ($1, $2, 'LegalSell', 'Activated', 'LegallySold', 'brand-ops-1', 'Brand', $3, 'idem-contract-history', NOW())"
    )
    .bind(Uuid::new_v4())
    .bind(&asset_id)
    .bind(trace_id)
    .execute(&db)
    .await
    .unwrap();

    let detail_resp = router
        .clone()
        .oneshot(request_with_actor(
            "GET",
            &format!("/assets/{asset_id}"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(detail_resp.status(), StatusCode::OK);
    let detail = parse_json(detail_resp).await;

    assert_eq!(detail["asset_id"], asset_id);
    assert_eq!(detail["brand_id"], brand_id);
    assert_eq!(detail["uid"], "04A31B2C3D4E5F");
    assert_eq!(detail["batch_id"], batch_id);
    assert_eq!(detail["external_product_id"], "sku-001");
    assert_eq!(detail["external_product_name"], "Product 001");
    assert_eq!(detail["external_product_url"], "https://example.test/p/1");
    assert_eq!(detail["current_state"], "LegallySold");
    assert_eq!(detail["previous_state"], "Activated");
    assert_eq!(detail["owner_id"], "buyer-001");
    assert_eq!(detail["key_epoch"], 2);
    assert!(detail.get("created_at").is_some());
    assert!(detail.get("updated_at").is_some());
    assert!(detail.get("activated_at").is_some());
    assert!(detail.get("sold_at").is_some());
    assert!(detail.get("status").is_none(), "asset detail should not expose legacy status alias");

    let history_resp = router
        .clone()
        .oneshot(request_with_actor(
            "GET",
            &format!("/assets/{asset_id}/history"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(history_resp.status(), StatusCode::OK);
    let history = parse_json(history_resp).await;

    assert_eq!(history["asset_id"], asset_id);
    assert_eq!(history["total"], 1);
    let events = history["events"].as_array().unwrap();
    assert_eq!(events.len(), 1);
    assert_eq!(events[0]["action"], "LegalSell");
    assert_eq!(events[0]["from_state"], "Activated");
    assert_eq!(events[0]["to_state"], "LegallySold");
    assert_eq!(events[0]["actor_id"], "brand-ops-1");
    assert_eq!(events[0]["actor_role"], "Brand");
    assert_eq!(events[0]["trace_id"], trace_id.to_string());
    assert!(events[0].get("occurred_at").is_some());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_asset_list_contract_and_brand_scope() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let router = test_router(db.clone(), setup_kms());

    let brand_id = generate_test_brand_id();
    let other_brand_id = generate_test_brand_id();
    let batch_id = generate_test_batch_id();
    let asset_id = generate_test_asset_id();
    seed_brand(&db, &brand_id, "Asset List Brand").await;
    seed_brand(&db, &other_brand_id, "Other Brand").await;

    sqlx::query("INSERT INTO batches (batch_id, brand_id, status, actual_count) VALUES ($1, $2, 'Open', 1)")
        .bind(&batch_id)
        .bind(&brand_id)
        .execute(&db)
        .await
        .unwrap();

    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, batch_id, uid, current_state, key_epoch) VALUES ($1, $2, $3, '04FEED11223344', 'Activated', 1)"
    )
    .bind(&asset_id)
    .bind(&brand_id)
    .bind(&batch_id)
    .execute(&db)
    .await
    .unwrap();

    let list_resp = router
        .clone()
        .oneshot(request_with_actor(
            "GET",
            &format!("/assets?brand_id={brand_id}&batch_id={batch_id}&current_state=Activated&page=1&page_size=10"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(list_resp.status(), StatusCode::OK);
    let list = parse_json(list_resp).await;

    assert_eq!(list["total"], 1);
    assert_eq!(list["page"], 1);
    assert_eq!(list["page_size"], 10);
    let items = list["items"].as_array().unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0]["asset_id"], asset_id);
    assert_eq!(items[0]["brand_id"], brand_id);
    assert_eq!(items[0]["batch_id"], batch_id);
    assert_eq!(items[0]["current_state"], "Activated");

    let forbidden_resp = router
        .oneshot(request_with_actor(
            "GET",
            &format!("/assets?brand_id={other_brand_id}"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(forbidden_resp.status(), StatusCode::FORBIDDEN);

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_batch_create_get_list_close_contract() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let router = test_router(db.clone(), setup_kms());

    let brand_id = generate_test_brand_id();
    seed_brand(&db, &brand_id, "Batch Contract Brand").await;

    let create_resp = router
        .clone()
        .oneshot(request_with_actor(
            "POST",
            "/batches",
            "factory-bot-1",
            "Factory",
            Some(&brand_id),
            Some(json!({
                "brand_id": brand_id,
                "batch_name": "Batch A",
                "factory_id": "factory-001",
                "expected_count": 120
            })),
        ))
        .await
        .unwrap();
    assert_eq!(create_resp.status(), StatusCode::CREATED);
    let created = parse_json(create_resp).await;

    let batch_id = created["batch_id"].as_str().unwrap().to_string();
    assert!(batch_id.starts_with("batch_"));
    assert_eq!(created["brand_id"], brand_id);
    assert_eq!(created["batch_name"], "Batch A");
    assert_eq!(created["factory_id"], "factory-001");
    assert_eq!(created["status"], "Open");
    assert_eq!(created["expected_count"], 120);
    assert_eq!(created["actual_count"], 0);
    assert!(created.get("created_at").is_some());
    assert!(created["closed_at"].is_null());

    let get_resp = router
        .clone()
        .oneshot(request_with_actor(
            "GET",
            &format!("/batches/{batch_id}"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(get_resp.status(), StatusCode::OK);
    let fetched = parse_json(get_resp).await;
    assert_eq!(fetched["batch_id"], batch_id);
    assert_eq!(fetched["brand_id"], brand_id);
    assert_eq!(fetched["status"], "Open");

    let list_resp = router
        .clone()
        .oneshot(request_with_actor(
            "GET",
            "/batches?page=1&page_size=5",
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            None,
        ))
        .await
        .unwrap();
    assert_eq!(list_resp.status(), StatusCode::OK);
    let list = parse_json(list_resp).await;
    assert_eq!(list["page"], 1);
    assert_eq!(list["page_size"], 5);
    assert!(list["total"].as_i64().unwrap() >= 1);
    let items = list["items"].as_array().unwrap();
    assert!(items.iter().any(|item| item["batch_id"] == batch_id));

    let close_resp = router
        .clone()
        .oneshot(request_with_actor(
            "POST",
            &format!("/batches/{batch_id}/close"),
            "brand-ops-1",
            "Brand",
            Some(&brand_id),
            Some(json!({})),
        ))
        .await
        .unwrap();
    assert_eq!(close_resp.status(), StatusCode::OK);
    let closed = parse_json(close_resp).await;
    assert_eq!(closed["batch_id"], batch_id);
    assert_eq!(closed["status"], "Closed");
    assert!(closed.get("closed_at").is_some());
    assert!(!closed["closed_at"].is_null());

    test_db.cleanup().await;
}
