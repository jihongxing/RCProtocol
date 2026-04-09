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
use rc_test_helpers::{fixtures::seed_brand, TestDb};
use serde_json::Value;
use sqlx::PgPool;
use tower::ServiceExt;

fn test_router_for_actions(db: PgPool, kms: Arc<dyn rc_kms::KeyProvider + Send + Sync>) -> Router {
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

#[tokio::test]
async fn test_transfer_get_and_reject_flow() {
    std::env::set_var("RC_ROOT_KEY_HEX", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef");
    std::env::set_var("RC_SYSTEM_ID", "test-system");

    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = Arc::new(rc_kms::SoftwareKms::from_env().expect("test KMS init"));
    let router = test_router_for_actions(db.clone(), kms);

    seed_brand(&db, "brand-001", "Transfer Reject Test Brand").await;

    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, owner_id, epoch) \
         VALUES ('asset-001', 'brand-001', '04A31B2C3D4E5F', 'Transferred', 'LegallySold', 'user-002', 0)"
    )
    .execute(&db)
    .await
    .unwrap();

    let transfer_id: uuid::Uuid = sqlx::query_scalar(
        "INSERT INTO asset_transfers (asset_id, from_user_id, to_user_id, from_owner_id, to_owner_id, transfer_type, idempotency_key, trace_id, metadata) \
         VALUES ('asset-001', 'user-001', 'user-002', 'user-001', 'user-002', 'CONSUMER_TO_CONSUMER', 'idem-001', 'trace-001', jsonb_build_object('status', 'pending')) \
         RETURNING transfer_id"
    )
    .fetch_one(&db)
    .await
    .unwrap();

    let get_req = Request::builder()
        .method("GET")
        .uri(format!("/transfers/{}", transfer_id))
        .header("Authorization", "user-002")
        .header("X-Actor-Role", "Consumer")
        .body(Body::empty())
        .unwrap();
    let get_resp = router.clone().oneshot(get_req).await.unwrap();
    assert_eq!(get_resp.status(), StatusCode::OK);

    let reject_req = Request::builder()
        .method("POST")
        .uri("/transfers/reject")
        .header("Content-Type", "application/json")
        .header("Authorization", "user-002")
        .header("X-Actor-Role", "Consumer")
        .body(Body::from(format!(r#"{{"transfer_id":"{}"}}"#, transfer_id)))
        .unwrap();
    let reject_resp = router.clone().oneshot(reject_req).await.unwrap();
    assert_eq!(reject_resp.status(), StatusCode::OK);
    let reject_body = reject_resp.into_body().collect().await.unwrap().to_bytes();
    let reject_json: Value = serde_json::from_slice(&reject_body).unwrap();
    assert_eq!(reject_json["status"], "rejected");

    let confirm_req = Request::builder()
        .method("POST")
        .uri("/transfers/confirm")
        .header("Content-Type", "application/json")
        .header("Authorization", "user-002")
        .header("X-Actor-Role", "Consumer")
        .body(Body::from(format!(r#"{{"transfer_id":"{}"}}"#, transfer_id)))
        .unwrap();
    let confirm_resp = router.oneshot(confirm_req).await.unwrap();
    assert_eq!(confirm_resp.status(), StatusCode::CONFLICT);

    test_db.cleanup().await;
}
