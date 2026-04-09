use axum::{
    extract::{Path, State},
    http::HeaderMap,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use rc_common::{
    errors::RcError,
    types::AssetState,
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::{
    app::AppState,
    auth::{authorization::{AuthorityProof, verify_authority}, extractor::ActorContext},
    db::{fetch_asset, transfers::{confirm_transfer, fetch_transfer, insert_transfer, reject_transfer}},
};

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/assets/:asset_id/transfer", post(transfer_handler))
        .route("/transfers/:transfer_id", get(get_transfer_handler))
        .route("/transfers/confirm", post(confirm_transfer_handler))
        .route("/transfers/reject", post(reject_transfer_handler))
}

#[derive(Debug, Deserialize)]
pub struct TransferRequest {
    pub new_owner_id: String,
    pub child_uid: String,
    pub child_ctr: String,
    pub child_cmac: String,
    pub authority_proof: AuthorityProof,
}

#[derive(Debug, Deserialize)]
pub struct TransferActionRequest {
    pub transfer_id: Uuid,
}

#[derive(Debug, Serialize)]
pub struct TransferResponse {
    pub transfer_id: Uuid,
    pub asset_id: String,
    pub from_owner_id: String,
    pub to_owner_id: String,
    pub new_state: String,
}

#[derive(Debug, Serialize)]
pub struct TransferDetailResponse {
    pub transfer_id: Uuid,
    pub asset_id: String,
    pub from_user_id: String,
    pub to_user_id: String,
    pub from_owner_id: String,
    pub to_owner_id: String,
    pub transfer_type: String,
    pub status: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

async fn transfer_handler(
    State(state): State<AppState>,
    Path(asset_id): Path<String>,
    actor: ActorContext,
    headers: HeaderMap,
    Json(payload): Json<TransferRequest>,
) -> impl IntoResponse {
    let idempotency_key = match required_header(&headers, "X-Idempotency-Key") {
        Ok(key) => key,
        Err(err) => return super::error_response(err),
    };

    let asset_record = match fetch_asset(&state.db, &asset_id).await {
        Ok(record) => record,
        Err(err) => return super::error_response(err),
    };

    if !matches!(asset_record.current_state, AssetState::LegallySold | AssetState::Transferred) {
        return super::error_response(RcError::InvalidInput(
            format!("asset in state {:?} cannot be transferred", asset_record.current_state)
        ));
    }

    let current_owner: Option<String> = match sqlx::query_scalar("SELECT owner_id FROM assets WHERE asset_id = $1")
        .bind(&asset_id)
        .fetch_optional(&state.db)
        .await
    {
        Ok(owner) => owner.flatten(),
        Err(err) => return super::error_response(RcError::Database(err.to_string())),
    };

    let current_owner = match current_owner {
        Some(owner) => owner,
        None => return super::error_response(RcError::InvalidInput("asset has no owner".into())),
    };

    if actor.actor_id != current_owner {
        return super::error_response(RcError::Forbidden("only current owner can transfer".into()));
    }

    if let Err(err) = verify_child_tag(&state, &asset_record.brand_id, &payload.child_uid, &payload.child_ctr, &payload.child_cmac).await {
        return super::error_response(err);
    }

    let auth_result = match verify_authority(&state.db, state.kms.clone(), &asset_id, payload.authority_proof).await {
        Ok(result) => result,
        Err(err) => return super::error_response(err),
    };

    if !auth_result.authorized {
        return super::error_response(RcError::Forbidden(
            format!("authority verification failed: {:?}", auth_result.risk_flags)
        ));
    }

    let new_state = AssetState::Transferred;
    if let Err(err) = sqlx::query(
        "UPDATE assets SET current_state = $2, previous_state = $3, owner_id = $4, updated_at = NOW() WHERE asset_id = $1"
    )
    .bind(&asset_id)
    .bind(new_state.as_db_str())
    .bind(asset_record.current_state.as_db_str())
    .bind(&payload.new_owner_id)
    .execute(&state.db)
    .await
    {
        return super::error_response(RcError::Database(err.to_string()));
    }

    let trace_id = optional_header(&headers, "X-Trace-Id")
        .and_then(|s| Uuid::parse_str(&s).ok())
        .unwrap_or_else(Uuid::new_v4);

    let transfer_id = match insert_transfer(
        &state.db,
        &asset_id,
        &current_owner,
        &payload.new_owner_id,
        "CONSUMER_TO_CONSUMER",
        &idempotency_key,
        &trace_id.to_string(),
    ).await {
        Ok(id) => id,
        Err(err) => return super::error_response(err),
    };

    let _ = sqlx::query(
        "INSERT INTO asset_state_events (event_id, asset_id, action, from_state, to_state, trace_id, actor_id, actor_role, actor_org, idempotency_key, approval_id, policy_version) \
         VALUES ($1, $2, 'Transfer', $3, $4, $5, $6, $7, $8, $9, $10, $11)"
    )
    .bind(Uuid::new_v4())
    .bind(&asset_id)
    .bind(asset_record.current_state.as_db_str())
    .bind(new_state.as_db_str())
    .bind(trace_id)
    .bind(&actor.actor_id)
    .bind(actor.actor_role.as_db_str())
    .bind(&actor.actor_org)
    .bind(&idempotency_key)
    .bind(optional_header(&headers, "X-Approval-Id"))
    .bind(optional_header(&headers, "X-Policy-Version"))
    .execute(&state.db)
    .await
    .map_err(|err| {
        tracing::warn!("failed to write transfer audit event: {err}");
    });

    Json(TransferResponse {
        transfer_id,
        asset_id,
        from_owner_id: current_owner,
        to_owner_id: payload.new_owner_id,
        new_state: new_state.as_db_str().to_string(),
    })
    .into_response()
}

async fn get_transfer_handler(
    State(state): State<AppState>,
    Path(transfer_id): Path<Uuid>,
    actor: ActorContext,
) -> impl IntoResponse {
    let transfer = match fetch_transfer(&state.db, transfer_id).await {
        Ok(transfer) => transfer,
        Err(err) => return super::error_response(err),
    };

    if actor.actor_id != transfer.from_user_id && actor.actor_id != transfer.to_user_id {
        return super::error_response(RcError::Forbidden("only transfer participants can view transfer".into()));
    }

    Json(to_transfer_detail_response(transfer)).into_response()
}

async fn confirm_transfer_handler(
    State(state): State<AppState>,
    actor: ActorContext,
    Json(payload): Json<TransferActionRequest>,
) -> impl IntoResponse {
    let existing = match fetch_transfer(&state.db, payload.transfer_id).await {
        Ok(transfer) => transfer,
        Err(err) => return super::error_response(err),
    };

    if actor.actor_id != existing.to_user_id {
        return super::error_response(RcError::Forbidden("only target owner can confirm transfer".into()));
    }

    let confirmed = match confirm_transfer(&state.db, payload.transfer_id).await {
        Ok(transfer) => transfer,
        Err(err) => return super::error_response(err),
    };

    Json(to_transfer_detail_response(confirmed)).into_response()
}

async fn reject_transfer_handler(
    State(state): State<AppState>,
    actor: ActorContext,
    Json(payload): Json<TransferActionRequest>,
) -> impl IntoResponse {
    let existing = match fetch_transfer(&state.db, payload.transfer_id).await {
        Ok(transfer) => transfer,
        Err(err) => return super::error_response(err),
    };

    if actor.actor_id != existing.to_user_id {
        return super::error_response(RcError::Forbidden("only target owner can reject transfer".into()));
    }

    let rejected = match reject_transfer(&state.db, payload.transfer_id).await {
        Ok(transfer) => transfer,
        Err(err) => return super::error_response(err),
    };

    Json(to_transfer_detail_response(rejected)).into_response()
}

fn to_transfer_detail_response(transfer: crate::db::transfers::TransferRecord) -> TransferDetailResponse {
    TransferDetailResponse {
        transfer_id: transfer.transfer_id,
        asset_id: transfer.asset_id,
        from_user_id: transfer.from_user_id,
        to_user_id: transfer.to_user_id,
        from_owner_id: transfer.from_owner_id,
        to_owner_id: transfer.to_owner_id,
        transfer_type: transfer.transfer_type,
        status: transfer.status,
        created_at: transfer.created_at,
    }
}

async fn verify_child_tag(
    state: &AppState,
    brand_id: &str,
    uid_hex: &str,
    ctr_hex: &str,
    cmac_hex: &str,
) -> Result<(), RcError> {
    let uid_bytes = hex::decode(uid_hex)
        .map_err(|_| RcError::InvalidInput("invalid child_uid hex".into()))?;
    let ctr_bytes = hex::decode(ctr_hex)
        .map_err(|_| RcError::InvalidInput("invalid child_ctr hex".into()))?;
    let cmac_bytes = hex::decode(cmac_hex)
        .map_err(|_| RcError::InvalidInput("invalid child_cmac hex".into()))?;

    if uid_bytes.len() != 7 {
        return Err(RcError::InvalidInput("child_uid must be 7 bytes".into()));
    }
    if ctr_bytes.len() != 3 {
        return Err(RcError::InvalidInput("child_ctr must be 3 bytes".into()));
    }
    if cmac_bytes.len() != 8 {
        return Err(RcError::InvalidInput("child_cmac must be 8 bytes".into()));
    }

    let uid: [u8; 7] = uid_bytes.try_into().unwrap();
    let ctr: [u8; 3] = ctr_bytes.try_into().unwrap();
    let cmac: [u8; 8] = cmac_bytes.try_into().unwrap();

    let epoch: i32 = sqlx::query_scalar("SELECT epoch FROM assets WHERE uid = $1")
        .bind(uid_hex)
        .fetch_optional(&state.db)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?
        .ok_or(RcError::AssetNotFound)?;

    let k_chip = state.kms
        .derive_chip_key(brand_id, &uid, epoch as u32)
        .map_err(|err| RcError::Database(format!("KMS derive_chip_key failed: {err}")))?;

    let cmac_valid = rc_crypto::sun::verify_sun_message(k_chip.as_bytes(), &uid, &ctr, &cmac);

    if !cmac_valid {
        return Err(RcError::InvalidInput("child tag CMAC verification failed".into()));
    }

    Ok(())
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
