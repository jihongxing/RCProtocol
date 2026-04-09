use axum::{
    extract::{Path, State},
    http::HeaderMap,
    response::IntoResponse,
    routing::post,
    Json, Router,
};
use rc_common::{
    audit::AuditContext,
    errors::RcError,
    types::{ActorRole, AssetAction, AssetState},
};
use rc_core::protocol::apply_action;
use serde::{Deserialize, Serialize};
use sqlx::{PgPool, Row};
use uuid::Uuid;

use crate::{
    app::AppState,
    asset_commitment::build_asset_commitment_record,
    attestation_brand::{build_brand_attestation_record, load_brand_signing_key},
    attestation_platform::{build_platform_attestation_record, load_platform_signing_key},
    auth::extractor::ActorContext,
    db::{
        asset_commitments::{bind_asset_commitment_to_asset, insert_asset_commitment},
        authority_devices::insert_authority_device,
        brand_attestations::insert_brand_attestation,
        entanglements::insert_entanglement,
        fetch_asset, fetch_verify_view, load_idempotency_record, persist_action,
        platform_attestations::insert_platform_attestation,
    },
};

pub fn write_router() -> Router<AppState> {
    Router::new()
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
        .route("/assets/:asset_id/mark-destructed", post(mark_destructed_asset))
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

pub async fn verify_asset(
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
    pub buyer_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ActivateRequest {
    pub external_product_id: String,
    pub external_product_name: Option<String>,
    pub external_product_url: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct VirtualMotherCard {
    pub authority_uid: String,
    pub authority_type: String,
    pub credential_hash: String,
    pub epoch: i32,
}

#[derive(Debug, Serialize)]
pub struct ActivateResponse {
    pub asset_id: String,
    pub asset_commitment_id: String,
    pub brand_attestation_status: String,
    pub platform_attestation_status: String,
    pub action: String,
    pub from_state: AssetState,
    pub to_state: AssetState,
    pub audit_event_id: Uuid,
    pub virtual_mother_card: VirtualMotherCard,
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
    trace_id: Uuid,
    idempotency_key: String,
    approval_id: Option<String>,
    policy_version: Option<String>,
}

async fn blind_log_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::BlindLog, actor, headers, payload).await
}

async fn stock_in_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::StockIn, actor, headers, payload).await
}

async fn activate_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<ActivateRequest>,
) -> impl IntoResponse {
    if payload.external_product_id.is_empty() {
        return error_response(RcError::InvalidInput("external_product_id is required".into()));
    }
    let required_headers = match parse_required_headers(&headers) {
        Ok(parsed) => parsed,
        Err(err) => return error_response(err),
    };
    let request_hash = format!(
        "asset={asset_id}|action=ActivateRotateKeys|trace={}|role={}|product={}",
        required_headers.trace_id,
        actor.actor_role.as_db_str(),
        payload.external_product_id
    );
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
    let asset_record = match fetch_asset(&state.db, &asset_id).await {
        Ok(record) => record,
        Err(err) => return error_response(err),
    };
    if let Err(err) = check_brand_boundary(&actor, &asset_record.brand_id) {
        return error_response(err);
    }
    let context = AuditContext {
        trace_id: required_headers.trace_id,
        actor_id: actor.actor_id.clone(),
        actor_role: actor.actor_role,
        actor_org: actor.actor_org.clone(),
        idempotency_key: required_headers.idempotency_key.clone(),
        approval_id: required_headers.approval_id,
        policy_version: required_headers.policy_version,
        buyer_id: None,
    };
    let (next_record, audit_event) = match apply_action(&asset_record, AssetAction::ActivateRotateKeys, context) {
        Ok(v) => v, Err(err) => return error_response(err),
    };
    if let Err(err) = update_asset_product_mapping(&state.db, &asset_id, &payload.external_product_id, payload.external_product_name.as_deref(), payload.external_product_url.as_deref()).await {
        return error_response(err);
    }
    let asset_snapshot = match sqlx::query("SELECT uid, key_epoch, batch_id FROM assets WHERE asset_id = $1").bind(&asset_id).fetch_optional(&state.db).await {
        Ok(Some(row)) => row,
        Ok(None) => return error_response(RcError::AssetNotFound),
        Err(err) => return error_response(RcError::Database(err.to_string())),
    };
    let uid: String = match asset_snapshot.get::<Option<String>, _>("uid") {
        Some(value) => value,
        None => return error_response(RcError::InvalidInput("asset has no UID".into())),
    };
    let key_epoch: i32 = asset_snapshot.get("key_epoch");
    let batch_id: Option<String> = asset_snapshot.get("batch_id");
    let commitment_record = match build_asset_commitment_record(&asset_record.brand_id, &uid, key_epoch as u32, &payload.external_product_id, payload.external_product_name.as_deref(), payload.external_product_url.as_deref(), batch_id.as_deref()) {
        Ok(record) => record,
        Err(err) => return error_response(err),
    };
    if let Err(err) = insert_asset_commitment(&state.db, &commitment_record).await { return error_response(err); }
    if let Err(err) = bind_asset_commitment_to_asset(&state.db, &asset_id, &commitment_record.commitment_id).await { return error_response(err); }
    let issued_at = chrono::Utc::now();
    let (brand_key_id, brand_signing_key) = match load_brand_signing_key() { Ok(v) => v, Err(err) => return error_response(err) };
    let brand_attestation = match build_brand_attestation_record(&asset_record.brand_id, &commitment_record.commitment_id, issued_at, &brand_key_id, &brand_signing_key) { Ok(v) => v, Err(err) => return error_response(err) };
    if let Err(err) = insert_brand_attestation(&state.db, &brand_attestation).await { return error_response(err); }
    let (platform_id, platform_key_id, platform_signing_key) = match load_platform_signing_key() { Ok(v) => v, Err(err) => return error_response(err) };
    let platform_attestation = match build_platform_attestation_record(&platform_id, &commitment_record.commitment_id, issued_at, &platform_key_id, &platform_signing_key) { Ok(v) => v, Err(err) => return error_response(err) };
    if let Err(err) = insert_platform_attestation(&state.db, &platform_attestation).await { return error_response(err); }
    let virtual_mother_card = match generate_virtual_mother_card_with_result(&state, &asset_id, &asset_record.brand_id, &actor.actor_id).await { Ok(v) => v, Err(err) => return error_response(err) };
    let response = ActivateResponse {
        asset_id: asset_id.clone(),
        asset_commitment_id: commitment_record.commitment_id.clone(),
        brand_attestation_status: "issued".into(),
        platform_attestation_status: "issued".into(),
        action: AssetAction::ActivateRotateKeys.as_db_str().to_string(),
        from_state: asset_record.current_state,
        to_state: next_record.current_state,
        audit_event_id: audit_event.event_id,
        virtual_mother_card,
    };
    let response_snapshot = match serde_json::to_value(&response) { Ok(v) => v, Err(err) => return error_response(RcError::Database(err.to_string())) };
    match persist_action(&state.db, &next_record, &audit_event, &request_hash, response_snapshot, Some(&commitment_record.commitment_id), state.redis.clone()).await {
        Ok(()) => Json(response).into_response(),
        Err(err) => error_response(err),
    }
}

async fn activate_entangle_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::ActivateEntangle, actor, headers, payload).await
}

async fn activate_confirm_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::ActivateConfirm, actor, headers, payload).await
}

async fn legal_sell_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::LegalSell, actor, headers, payload).await
}

async fn transfer_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Transfer, actor, headers, payload).await
}

async fn consume_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Consume, actor, headers, payload).await
}

async fn legacy_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Legacy, actor, headers, payload).await
}

async fn freeze_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Freeze, actor, headers, payload).await
}

async fn recover_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::Recover, actor, headers, payload).await
}

async fn mark_tampered_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::MarkTampered, actor, headers, payload).await
}

async fn mark_compromised_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::MarkCompromised, actor, headers, payload).await
}

async fn execute_asset_action(
    state: AppState,
    asset_id: String,
    action: AssetAction,
    actor: ActorContext,
    headers: HeaderMap,
    payload: AssetActionRequest,
) -> impl IntoResponse {
    if action == AssetAction::LegalSell && payload.buyer_id.as_ref().is_none_or(|s| s.is_empty()) {
        return error_response(RcError::InvalidInput("LegalSell requires buyer_id".into()));
    }
    let required_headers = match parse_required_headers(&headers) {
        Ok(parsed) => parsed,
        Err(err) => return error_response(err),
    };

    let request_hash = build_request_hash(&asset_id, action, &payload, &actor, &required_headers);

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
        actor_id: actor.actor_id.clone(),
        actor_role: actor.actor_role,
        actor_org: actor.actor_org.clone(),
        idempotency_key: required_headers.idempotency_key,
        approval_id: required_headers.approval_id,
        policy_version: required_headers.policy_version,
        buyer_id: payload.buyer_id.clone(),
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

            match persist_action(&state.db, &next_record, &audit_event, &request_hash, response_snapshot, None, state.redis.clone()).await {
                Ok(()) => {
                    if action == AssetAction::ActivateEntangle {
                        if let Err(err) = generate_virtual_mother_card_with_result(&state, &response.asset_id, &asset_record.brand_id, &actor.actor_id).await {
                            return error_response(err);
                        }
                    }
                    Json(response).into_response()
                }
                Err(err) => error_response(err),
            }
        }
        Err(err) => error_response(err),
    }
}

fn parse_required_headers(headers: &HeaderMap) -> Result<RequiredHeaders, RcError> {
    let trace_id = required_header(headers, "X-Trace-Id")?;
    let idempotency_key = required_header(headers, "X-Idempotency-Key")?;
    Ok(RequiredHeaders {
        trace_id: Uuid::parse_str(&trace_id).map_err(|_| RcError::InvalidHeader("X-Trace-Id"))?,
        idempotency_key,
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

fn build_request_hash(
    asset_id: &str,
    action: AssetAction,
    payload: &AssetActionRequest,
    actor: &ActorContext,
    headers: &RequiredHeaders,
) -> String {
    format!(
        "asset={asset_id}|action={}|trace={}|role={}|approval={}|policy={}|prev={:?}",
        action.as_db_str(),
        headers.trace_id,
        actor.actor_role.as_db_str(),
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


async fn mark_destructed_asset(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<AssetActionRequest>,
) -> impl IntoResponse {
    execute_asset_action(state, asset_id, AssetAction::MarkDestructed, actor, headers, payload).await
}

async fn generate_virtual_mother_card_with_result(
    state: &AppState,
    asset_id: &str,
    brand_id: &str,
    actor_id: &str,
) -> Result<VirtualMotherCard, RcError> {
    let row = sqlx::query("SELECT uid, epoch FROM assets WHERE asset_id = $1")
        .bind(asset_id)
        .fetch_optional(&state.db)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?
        .ok_or(RcError::AssetNotFound)?;

    let child_uid: String = row
        .get::<Option<String>, _>("uid")
        .ok_or_else(|| RcError::InvalidInput("asset has no UID".into()))?;
    let epoch: i32 = row.get("epoch");

    let authority_uid = format!("vauth-{}", nanoid::nanoid!(12));
    let k_chip_mother = state
        .kms
        .derive_mother_key(brand_id, authority_uid.as_bytes(), epoch as u32)
        .map_err(|err| RcError::Database(format!("KMS derive_mother_key failed: {err}")))?;

    let hash_bytes = rc_crypto::hmac_sha256::compute(k_chip_mother.as_bytes(), authority_uid.as_bytes());
    let credential_hash = hex::encode(hash_bytes);
    let _authority_id = insert_authority_device(
        &state.db,
        &authority_uid,
        "VIRTUAL_APP",
        brand_id,
        epoch,
        Some(&credential_hash),
        Some(actor_id),
        Some(actor_id),
    )
    .await?;

    insert_entanglement(&state.db, asset_id, &child_uid, _authority_id, &authority_uid, actor_id).await?;

    Ok(VirtualMotherCard {
        authority_uid,
        authority_type: "VIRTUAL_APP".to_string(),
        credential_hash,
        epoch,
    })
}

async fn update_asset_product_mapping(
    pool: &PgPool,
    asset_id: &str,
    external_product_id: &str,
    external_product_name: Option<&str>,
    external_product_url: Option<&str>,
) -> Result<(), RcError> {
    sqlx::query(
        "UPDATE assets SET external_product_id = $2, external_product_name = $3, external_product_url = $4, updated_at = NOW() WHERE asset_id = $1",
    )
    .bind(asset_id)
    .bind(external_product_id)
    .bind(external_product_name)
    .bind(external_product_url)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}

fn check_brand_boundary(actor: &ActorContext, brand_id: &str) -> Result<(), RcError> {
    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == brand_id => Ok(()),
            _ => Err(RcError::BrandBoundaryViolation),
        }
    } else {
        Ok(())
    }
}

