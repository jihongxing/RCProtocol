use axum::{
    extract::{Path, State},
    http::HeaderMap,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use rc_common::{
    audit::AuditContext,
    errors::RcError,
    types::{ActorRole, AssetAction, AssetState},
};
use rc_core::protocol::apply_action;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::{
    app::AppState,
    db::{fetch_asset, fetch_verify_view, load_idempotency_record, persist_action},
};

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/verify/:asset_id", get(verify_asset))
        .route("/assets/:asset_id/blind-log", post(blind_log_asset))
        .route("/assets/:asset_id/stock-in", post(stock_in_asset))
        .route("/assets/:asset_id/activate", post(activate_asset))
        .route("/assets/:asset_id/activate-entangle", post(activate_entangle_asset))
        .route("/assets/:asset_id/activate-confirm", post(activate_confirm_asset))
        .route("/assets/:asset_id/legal-sell", post(legal_sell_asset))
        .route("/assets/:asset_id/transfer", post(transfer_asset))
        .route("/assets/:asset_id/consume", post(consume_asset))
        .route("/assets/:asset_id/legacy", post(legacy_asset))
        .route("/assets/:asset_id/freeze", post(freeze_asset))
        .route("/assets/:asset_id/recover", post(recover_asset))
        .route("/assets/:asset_id/mark-tampered", post(mark_tampered_asset))
        .route("/assets/:asset_id/mark-compromised", post(mark_compromised_asset))
}

#[derive(Debug, Serialize)]
struct VerifyResponse {
    asset_id: String,
    brand_id: String,
    product_id: Option<String>,
    uid: Option<String>,
    current_state: String,
    previous_state: Option<String>,
    event_count: i64,
    verification_result: String,
}

async fn verify_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
) -> impl IntoResponse {
    match fetch_verify_view(&state.db, &asset_id).await {
        Ok(view) => Json(VerifyResponse {
            verification_result: verification_result_label(&view.current_state).to_string(),
            asset_id: view.asset_id,
            brand_id: view.brand_id,
            product_id: view.product_id,
            uid: view.uid,
            current_state: view.current_state,
            previous_state: view.previous_state,
            event_count: view.event_count,
        })
        .into_response(),
        Err(err) => error_response(err),
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct AssetActionRequest {
    pub previous_state: Option<AssetState>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct AssetActionResponse {
    pub asset_id: String,
    pub action: String,
    pub from_state: AssetState,
    pub to_state: AssetState,
    pub audit_event_id: Uuid,
}

#[derive(Debug)]
struct RequiredHeaders {
    authorization: String,
    trace_id: Uuid,
    idempotency_key: String,
    actor_role: ActorRole,
    actor_org: Option<String>,
    approval_id: Option<String>,
    policy_version: Option<String>,
}

async fn blind_log_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::BlindLog, headers, payload).await
}

async fn stock_in_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::StockIn, headers, payload).await
}

async fn activate_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::ActivateRotateKeys, headers, payload).await
}

async fn activate_entangle_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::ActivateEntangle, headers, payload).await
}

async fn activate_confirm_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::ActivateConfirm, headers, payload).await
}

async fn legal_sell_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::LegalSell, headers, payload).await
}

async fn transfer_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Transfer, headers, payload).await
}

async fn consume_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Consume, headers, payload).await
}

async fn legacy_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Legacy, headers, payload).await
}

async fn freeze_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Freeze, headers, payload).await
}

async fn recover_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Recover, headers, payload).await
}

async fn mark_tampered_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::MarkTampered, headers, payload).await
}

async fn mark_compromised_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::MarkCompromised, headers, payload).await
}

async fn execute_asset_action(
    state: AppState,
    asset_id: String,
    action: AssetAction,
    headers: HeaderMap,
    payload: AssetActionRequest,
) -> impl IntoResponse {
    let required_headers = match parse_required_headers(&headers) {
        Ok(parsed) => parsed,
        Err(err) => return error_response(err),
    };

    let request_hash = build_request_hash(&asset_id, action, &payload, &required_headers);

    match load_idempotency_record(&state.db, &required_headers.idempotency_key).await {
        Ok(Some(record)) => {
            if record.request_hash != request_hash {
                return error_response(RcError::IdempotencyConflict);
            }
            return Json(record.response_snapshot).into_response();
        }
        Ok(None) => {}
        Err(err) => return error_response(err),
    }

    let mut asset_record = match fetch_asset(&state.db, &asset_id).await {
        Ok(record) => record,
        Err(err) => return error_response(err),
    };

    if payload.previous_state.is_some() {
        asset_record.previous_state = payload.previous_state;
    }

    let context = AuditContext {
        trace_id: required_headers.trace_id,
        actor_id: required_headers.authorization,
        actor_role: required_headers.actor_role,
        actor_org: required_headers.actor_org,
        idempotency_key: required_headers.idempotency_key,
        approval_id: required_headers.approval_id,
        policy_version: required_headers.policy_version,
    };

    match apply_action(&asset_record, action, context) {
        Ok((next_record, audit_event)) => {
            let response = AssetActionResponse {
                asset_id,
                action: action.as_db_str().to_string(),
                from_state: asset_record.current_state,
                to_state: next_record.current_state,
                audit_event_id: audit_event.event_id,
            };

            let response_snapshot = match serde_json::to_value(&response) {
                Ok(value) => value,
                Err(err) => return error_response(RcError::Database(err.to_string())),
            };

            match persist_action(&state.db, &next_record, &audit_event, &request_hash, response_snapshot).await {
                Ok(()) => Json(response).into_response(),
                Err(err) => error_response(err),
            }
        }
        Err(err) => error_response(err),
    }
}

fn parse_required_headers(headers: &HeaderMap) -> Result<RequiredHeaders, RcError> {
    let authorization = required_header(headers, "Authorization")?;
    let trace_id = required_header(headers, "X-Trace-Id")?;
    let idempotency_key = required_header(headers, "X-Idempotency-Key")?;
    let actor_role = required_header(headers, "X-Actor-Role")?;

    Ok(RequiredHeaders {
        authorization,
        trace_id: Uuid::parse_str(&trace_id).map_err(|_| RcError::InvalidHeader("X-Trace-Id"))?,
        idempotency_key,
        actor_role: parse_actor_role(&actor_role)?,
        actor_org: optional_header(headers, "X-Actor-Org"),
        approval_id: optional_header(headers, "X-Approval-Id"),
        policy_version: optional_header(headers, "X-Policy-Version"),
    })
}

fn required_header(headers: &HeaderMap, key: &'static str) -> Result<String, RcError> {
    headers
        .get(key)
        .ok_or(RcError::MissingRequiredHeader(key))?
        .to_str()
        .map(|value| value.to_string())
        .map_err(|_| RcError::InvalidHeader(key))
}

fn optional_header(headers: &HeaderMap, key: &'static str) -> Option<String> {
    headers.get(key).and_then(|value| value.to_str().ok()).map(ToOwned::to_owned)
}

fn parse_actor_role(value: &str) -> Result<ActorRole, RcError> {
    match value {
        "Platform" => Ok(ActorRole::Platform),
        "Factory" => Ok(ActorRole::Factory),
        "Brand" => Ok(ActorRole::Brand),
        "Consumer" => Ok(ActorRole::Consumer),
        "Moderator" => Ok(ActorRole::Moderator),
        _ => Err(RcError::InvalidHeader("X-Actor-Role")),
    }
}

fn build_request_hash(
    asset_id: &str,
    action: AssetAction,
    payload: &AssetActionRequest,
    headers: &RequiredHeaders,
) -> String {
    format!(
        "asset={asset_id}|action={}|trace={}|role={}|approval={}|policy={}|prev={:?}",
        action.as_db_str(),
        headers.trace_id,
        headers.actor_role.as_db_str(),
        headers.approval_id.as_deref().unwrap_or(""),
        headers.policy_version.as_deref().unwrap_or(""),
        payload.previous_state.map(|state| state.as_db_str())
    )
}

fn verification_result_label(state: &str) -> &'static str {
    match state {
        "Activated" | "LegallySold" | "Transferred" => "verified",
        "Disputed" | "Tampered" | "Compromised" | "Destructed" => "restricted",
        _ => "pending",
    }
}

fn error_response(err: RcError) -> axum::response::Response {
    let status = match err {
        RcError::AssetNotFound => axum::http::StatusCode::NOT_FOUND,
        RcError::MissingRequiredHeader(_) | RcError::InvalidHeader(_) => axum::http::StatusCode::BAD_REQUEST,
        RcError::IdempotencyConflict => axum::http::StatusCode::CONFLICT,
        RcError::Database(_) => axum::http::StatusCode::INTERNAL_SERVER_ERROR,
        _ => axum::http::StatusCode::BAD_REQUEST,
    };

    (status, err.to_string()).into_response()
}
