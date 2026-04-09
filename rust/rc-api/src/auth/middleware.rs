use std::sync::Arc;

use axum::{
    body::Body,
    extract::State,
    http::{Request, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use sqlx::PgPool;

use super::{claims::Claims, jwt::JwtDecoder};

#[derive(Clone)]
pub struct AuthState {
    pub jwt_decoder: Arc<JwtDecoder>,
    pub auth_disabled: bool,
    pub db: PgPool,
    pub api_key_secret: Vec<u8>,
}

impl From<&crate::app::AppState> for AuthState {
    fn from(state: &crate::app::AppState) -> Self {
        Self {
            jwt_decoder: state.jwt_decoder.clone(),
            auth_disabled: state.auth_disabled,
            db: state.db.clone(),
            api_key_secret: state.api_key_secret.clone(),
        }
    }
}

pub async fn auth_middleware(
    State(auth): State<AuthState>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response, Response> {
    let claims = if auth.auth_disabled {
        build_fallback_claims(&request)?
    } else if let Some(api_key_hash) = extract_api_key_hash(&request) {
        authenticate_api_key_hash(&auth, api_key_hash).await?
    } else if let Some(api_key) = extract_api_key(&request) {
        let hash = crate::auth::api_key::hash_api_key(api_key);
        authenticate_api_key_hash(&auth, &hash).await?
    } else {
        let token = extract_bearer_token(&request)
            .ok_or_else(|| unauthorized_response("missing or invalid Authorization header"))?;
        auth.jwt_decoder
            .decode(token)
            .map_err(|err| unauthorized_response(&format!("{err:?}")))?
    };

    request.extensions_mut().insert(claims);
    Ok(next.run(request).await)
}

fn extract_api_key(request: &Request<Body>) -> Option<&str> {
    request.headers().get("X-Api-Key")?.to_str().ok()
}

fn extract_api_key_hash(request: &Request<Body>) -> Option<&str> {
    request.headers().get("X-Api-Key-Hash")?.to_str().ok()
}

#[allow(clippy::result_large_err)]
async fn authenticate_api_key_hash(auth: &AuthState, api_key_hash: &str) -> Result<Claims, Response> {
    let brand = crate::db::brands::fetch_brand_by_api_key_hash(&auth.db, api_key_hash)
        .await
        .map_err(|_| unauthorized_response("invalid_api_key"))?
        .ok_or_else(|| unauthorized_response("invalid_api_key"))?;

    Ok(Claims {
        sub: format!("apikey:{}", brand.brand_id),
        role: "Brand".to_string(),
        org_id: None,
        brand_id: Some(brand.brand_id),
        scopes: vec![],
        exp: u64::MAX,
        iat: 0,
    })
}

fn extract_bearer_token(request: &Request<Body>) -> Option<&str> {
    request
        .headers()
        .get("Authorization")?
        .to_str()
        .ok()?
        .strip_prefix("Bearer ")
}

#[allow(clippy::result_large_err)]
fn build_fallback_claims(request: &Request<Body>) -> Result<Claims, Response> {
    let actor_id = request
        .headers()
        .get("Authorization")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("anonymous")
        .to_string();

    let role = request
        .headers()
        .get("X-Actor-Role")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("Platform")
        .to_string();

    let brand_id = request
        .headers()
        .get("X-Brand-Id")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());

    if (role == "Brand" || role == "Factory") && brand_id.is_none() {
        return Err(unauthorized_response("Brand/Factory role requires X-Brand-Id header"));
    }

    Ok(Claims {
        sub: actor_id,
        role,
        org_id: None,
        brand_id,
        scopes: vec![],
        exp: u64::MAX,
        iat: 0,
    })
}

fn unauthorized_response(message: &str) -> Response {
    (
        StatusCode::UNAUTHORIZED,
        Json(json!({"error": "unauthorized", "message": message})),
    )
        .into_response()
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::{middleware as axum_mw, routing::get, Router};
    use http_body_util::BodyExt;
    use rc_common::ids;
    use sqlx::postgres::PgPoolOptions;
    use tower::ServiceExt;

    const SECRET: &[u8] = b"test-secret-at-least-32-bytes-long";

    fn test_auth_state(auth_disabled: bool) -> AuthState {
        let db = PgPoolOptions::new()
            .max_connections(1)
            .connect_lazy("postgres://dummy:dummy@localhost:1/dummy")
            .expect("connect_lazy should not fail");
        AuthState {
            jwt_decoder: Arc::new(JwtDecoder::new(SECRET)),
            auth_disabled,
            db,
            api_key_secret: b"test-api-key-secret".to_vec(),
        }
    }

    fn app(auth_disabled: bool) -> Router {
        let state = test_auth_state(auth_disabled);
        Router::new()
            .route("/test", get(|| async { "ok" }))
            .layer(axum_mw::from_fn_with_state(state.clone(), auth_middleware))
            .with_state(state)
    }

    async fn body_string(response: Response) -> String {
        let bytes = response.into_body().collect().await.unwrap().to_bytes();
        String::from_utf8(bytes.to_vec()).unwrap()
    }

    #[tokio::test]
    async fn test_no_auth_header_returns_401() {
        let request = Request::builder()
            .uri("/test")
            .body(Body::empty())
            .unwrap();

        let response = app(false).oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
        let body = body_string(response).await;
        let json: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(json["error"], "unauthorized");
    }

    #[tokio::test]
    async fn test_invalid_token_returns_401() {
        let request = Request::builder()
            .uri("/test")
            .header("Authorization", "Bearer not-a-valid-jwt-token")
            .body(Body::empty())
            .unwrap();

        let response = app(false).oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
        let body = body_string(response).await;
        let json: serde_json::Value = serde_json::from_str(&body).unwrap();
        assert_eq!(json["error"], "unauthorized");
    }

    #[tokio::test]
    async fn test_valid_token_passes_through() {
        use crate::auth::jwt::encode_test_token;

        let claims = Claims {
            sub: "user-42".into(),
            role: "Platform".into(),
            org_id: None,
            brand_id: None,
            scopes: vec![],
            exp: u64::MAX,
            iat: 0,
        };
        let token = encode_test_token(&claims, SECRET);

        let request = Request::builder()
            .uri("/test")
            .header("Authorization", format!("Bearer {token}"))
            .body(Body::empty())
            .unwrap();

        let response = app(false).oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = body_string(response).await;
        assert_eq!(body, "ok");
    }

    #[tokio::test]
    async fn test_auth_disabled_fallback() {
        let brand_id = ids::generate_brand_id();
        assert!(brand_id.starts_with("brand_"));

        let request = Request::builder()
            .uri("/test")
            .header("Authorization", "factory-bot-1")
            .header("X-Actor-Role", "Factory")
            .header("X-Brand-Id", &brand_id)
            .body(Body::empty())
            .unwrap();

        let response = app(true).oneshot(request).await.unwrap();

        assert_eq!(response.status(), StatusCode::OK);
        let body = body_string(response).await;
        assert_eq!(body, "ok");
    }
}
