use axum::{
    async_trait,
    extract::FromRequestParts,
    http::{request::Parts, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use rc_common::types::ActorRole;
use serde_json::json;

use super::claims::Claims;

/// Verified actor identity extracted from JWT Claims in request extensions.
/// Used by handlers to access actor_id, role, org, and brand_id
/// without manual header parsing.
#[derive(Debug, Clone)]
pub struct ActorContext {
    pub actor_id: String,
    pub actor_role: ActorRole,
    pub actor_org: Option<String>,
    pub brand_id: Option<String>,
}

#[async_trait]
impl<S: Send + Sync> FromRequestParts<S> for ActorContext {
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let claims = parts.extensions.get::<Claims>().cloned().ok_or_else(|| {
            (
                StatusCode::UNAUTHORIZED,
                Json(json!({"error": "unauthorized", "message": "missing claims"})),
            )
                .into_response()
        })?;

        let actor_role = claims.actor_role().map_err(|err| {
            (
                StatusCode::UNAUTHORIZED,
                Json(json!({"error": "unauthorized", "message": format!("{err:?}")})),
            )
                .into_response()
        })?;

        Ok(ActorContext {
            actor_id: claims.sub,
            actor_role,
            actor_org: claims.org_id,
            brand_id: claims.brand_id,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::auth::jwt::JwtDecoder;
    use crate::auth::middleware::{auth_middleware, AuthState};
    use axum::{body::Body, http::Request, middleware as axum_mw, routing::get, Router};
    use http_body_util::BodyExt;
    use std::sync::Arc;
    use tower::ServiceExt;

    #[tokio::test]
    async fn test_extractor_with_claims() {
        async fn handler(actor: ActorContext) -> String {
            format!("{}:{}", actor.actor_id, actor.actor_role.as_db_str())
        }

        let db = sqlx::postgres::PgPoolOptions::new()
            .max_connections(1)
            .connect_lazy("postgres://dummy:dummy@localhost:1/dummy")
            .expect("connect_lazy should not fail");
        let state = AuthState {
            jwt_decoder: Arc::new(JwtDecoder::new(b"unused-placeholder-key-32-bytes!")),
            auth_disabled: true,
            db,
            api_key_secret: b"test-api-key-secret".to_vec(),
        };

        let app = Router::new()
            .route("/test", get(handler))
            .layer(axum_mw::from_fn_with_state(state.clone(), auth_middleware))
            .with_state(state);

        let req = Request::builder()
            .uri("/test")
            .header("Authorization", "user-42")
            .header("X-Actor-Role", "Platform")
            .body(Body::empty())
            .unwrap();

        let resp = app.oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::OK);

        let body = resp.into_body().collect().await.unwrap().to_bytes();
        let body_str = String::from_utf8(body.to_vec()).unwrap();
        assert_eq!(body_str, "user-42:Platform");
    }

    #[tokio::test]
    async fn test_extractor_without_claims() {
        async fn handler(_actor: ActorContext) -> &'static str {
            "ok"
        }

        // No middleware → no Claims in extensions → 401
        let app = Router::new().route("/test", get(handler));

        let req = Request::builder()
            .uri("/test")
            .body(Body::empty())
            .unwrap();

        let resp = app.oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }
}
