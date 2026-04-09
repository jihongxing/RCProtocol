use axum::{
    body::Body,
    http::{Request, StatusCode},
    middleware as axum_mw,
    Router,
};
use http_body_util::BodyExt;
use jsonwebtoken::{encode, EncodingKey, Header};
use rc_api::{
    app::AppState,
    auth::{
        claims::Claims,
        middleware::{auth_middleware, AuthState},
    },
};
use rc_test_helpers::TestDb;
use serde_json::Value;
use tower::ServiceExt;

const TEST_JWT_SECRET: &[u8] = b"test-secret-at-least-32-bytes-long";

fn app(state: AppState) -> Router {
    let auth_state = AuthState::from(&state);

    let public = Router::new()
        .route("/healthz", axum::routing::get(rc_api::routes::health::healthz))
        .merge(rc_api::routes::verify::router())
        .route(
            "/verify/:asset_id",
            axum::routing::get(rc_api::routes::protocol::verify_asset),
        );

    let protected = rc_api::routes::protocol::write_router()
        .merge(rc_api::routes::assets::router())
        .merge(rc_api::routes::brand::router())
        .merge(rc_api::routes::batch::router())
        .merge(rc_api::routes::transfer::router())
        .merge(rc_api::routes::authority_devices::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware));

    public.merge(protected).with_state(state)
}

async fn response_json(response: axum::response::Response) -> Value {
    let status = response.status();
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    let body = String::from_utf8(bytes.to_vec()).unwrap();
    serde_json::from_str(&body).unwrap_or_else(|_| serde_json::json!({
        "status": status.as_u16(),
        "raw": body,
    }))
}

fn platform_token() -> String {
    let claims = Claims {
        sub: "platform-admin".into(),
        role: "Platform".into(),
        org_id: None,
        brand_id: None,
        scopes: vec![],
        exp: u64::MAX,
        iat: 0,
    };

    encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(TEST_JWT_SECRET),
    )
    .expect("encode test jwt")
}

fn platform_request(method: &str, uri: &str, body: Value) -> Request<Body> {
    Request::builder()
        .method(method)
        .uri(uri)
        .header("Authorization", format!("Bearer {}", platform_token()))
        .header("Content-Type", "application/json")
        .body(Body::from(body.to_string()))
        .unwrap()
}

fn brand_request(method: &str, uri: &str, api_key: &str, body: Option<Value>) -> Request<Body> {
    let builder = Request::builder()
        .method(method)
        .uri(uri)
        .header("X-Api-Key", api_key);

    match body {
        Some(payload) => builder
            .header("Content-Type", "application/json")
            .body(Body::from(payload.to_string()))
            .unwrap(),
        None => builder.body(Body::empty()).unwrap(),
    }
}

async fn build_test_state() -> (TestDb, AppState) {
    std::env::set_var("RC_ROOT_KEY_HEX", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef");
    std::env::set_var("RC_SYSTEM_ID", "brand-integration-test");

    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms: std::sync::Arc<dyn rc_kms::KeyProvider + Send + Sync> = std::sync::Arc::new(
        rc_kms::SoftwareKms::from_env().expect("test kms init"),
    );

    let state = AppState {
        db,
        kms,
        jwt_decoder: std::sync::Arc::new(rc_api::auth::jwt::JwtDecoder::new(TEST_JWT_SECRET)),
        auth_disabled: false,
        redis: None,
        ctr_cache: std::sync::Arc::new(dashmap::DashMap::new()),
        fallback_strategy: rc_api::app::FallbackStrategy::DirectPg,
        api_key_secret: b"test-api-key-secret".to_vec(),
    };

    (test_db, state)
}

#[tokio::test]
async fn brand_registration_and_rotation_flow() {
    let (test_db, state) = build_test_state().await;
    let app = app(state);

    let register_req = platform_request(
        "POST",
        "/brands",
        serde_json::json!({
            "brand_name": "Integration Brand",
            "contact_email": "integration-brand@example.com",
            "industry": "Watches"
        }),
    );

    let register_resp = app.clone().oneshot(register_req).await.unwrap();
    let register_status = register_resp.status();
    let register_json = response_json(register_resp).await;
    assert_eq!(register_status, StatusCode::CREATED, "register response: {register_json}");

    let brand_id = register_json["brand_id"].as_str().unwrap().to_string();
    let api_key = register_json["api_key"]["api_key"].as_str().unwrap().to_string();
    assert!(brand_id.starts_with("brand_"));
    assert!(api_key.starts_with("rcpk_live_"));

    let duplicate_req = platform_request(
        "POST",
        "/brands",
        serde_json::json!({
            "brand_name": "Duplicate Brand",
            "contact_email": "integration-brand@example.com",
            "industry": "Fashion"
        }),
    );
    let duplicate_resp = app.clone().oneshot(duplicate_req).await.unwrap();
    assert_eq!(duplicate_resp.status(), StatusCode::CONFLICT);

    let detail_req = brand_request("GET", &format!("/brands/{brand_id}"), &api_key, None);
    let detail_resp = app.clone().oneshot(detail_req).await.unwrap();
    assert_eq!(detail_resp.status(), StatusCode::OK);
    let detail_json = response_json(detail_resp).await;
    assert_eq!(detail_json["brand_id"], brand_id);
    assert!(detail_json.get("api_key").is_none());

    let rotate_req = brand_request(
        "POST",
        &format!("/brands/{brand_id}/api-keys/rotate"),
        &api_key,
        Some(serde_json::json!({"reason": "integration test rotation"})),
    );
    let rotate_resp = app.clone().oneshot(rotate_req).await.unwrap();
    assert_eq!(rotate_resp.status(), StatusCode::OK);
    let rotate_json = response_json(rotate_resp).await;
    let new_api_key = rotate_json["api_key"].as_str().unwrap().to_string();
    let revoked_key_id = rotate_json["revoked_key_id"].as_str().unwrap().to_string();
    assert!(new_api_key.starts_with("rcpk_live_"));
    assert!(revoked_key_id.starts_with("key_"));

    let old_key_req = brand_request("GET", &format!("/brands/{brand_id}"), &api_key, None);
    let old_key_resp = app.clone().oneshot(old_key_req).await.unwrap();
    assert_eq!(old_key_resp.status(), StatusCode::UNAUTHORIZED);

    let new_key_req = brand_request("GET", &format!("/brands/{brand_id}"), &new_api_key, None);
    let new_key_resp = app.clone().oneshot(new_key_req).await.unwrap();
    assert_eq!(new_key_resp.status(), StatusCode::OK);

    let key_list_req = brand_request("GET", &format!("/brands/{brand_id}/api-keys"), &new_api_key, None);
    let key_list_resp = app.clone().oneshot(key_list_req).await.unwrap();
    assert_eq!(key_list_resp.status(), StatusCode::OK);
    let key_list_json = response_json(key_list_resp).await;
    let keys = key_list_json["keys"].as_array().unwrap();
    assert_eq!(keys.len(), 2);
    assert!(keys.iter().any(|k| k["status"] == "Active"));
    assert!(keys.iter().any(|k| k["status"] == "Revoked"));
    assert!(keys.iter().all(|k| k.get("key_hash").is_none()));
    assert!(keys.iter().all(|k| k["key_prefix"].as_str().unwrap().ends_with("****")));

    test_db.cleanup().await;
}
