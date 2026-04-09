use axum::extract::State;
use axum::Json;
use serde_json::{json, Value};

use crate::app::{AppState, FallbackStrategy};

pub async fn healthz(State(state): State<AppState>) -> Json<Value> {
    let redis_status = if state.fallback_strategy == FallbackStrategy::DirectPg {
        // 运维显式选择不使用 Redis
        "not_configured"
    } else if state.redis.is_some() {
        "connected"
    } else {
        "disconnected"
    };

    Json(json!({
        "status": "ok",
        "redis": redis_status
    }))
}
