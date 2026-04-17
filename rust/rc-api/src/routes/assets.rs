use axum::{
    extract::{Path, Query, State},
    routing::get,
    Json, Router,
};
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::app::AppState;
use crate::auth::extractor::ActorContext;
use crate::db::assets::{self, AssetDetail, AssetStateEvent};
use crate::db::{brand_attestations, platform_attestations};
use rc_common::errors::RcError;
use rc_common::types::ActorRole;

#[derive(Debug, Deserialize)]
pub struct AssetListQuery {
    pub brand_id: Option<String>,
    pub batch_id: Option<String>,
    pub current_state: Option<String>,
    pub page: Option<i64>,
    pub page_size: Option<i64>,
}

impl AssetListQuery {
    pub fn offset(&self) -> i64 {
        (self.page() - 1) * self.page_size()
    }

    pub fn page_size(&self) -> i64 {
        self.page_size.unwrap_or(20).clamp(1, 100)
    }

    pub fn page(&self) -> i64 {
        self.page.unwrap_or(1).max(1)
    }
}

#[derive(Debug, Serialize)]
pub struct PaginatedResponse<T: Serialize> {
    pub items: Vec<T>,
    pub total: i64,
    pub page: i64,
    pub page_size: i64,
}

#[derive(Debug, Serialize)]
pub struct AssetHistoryResponse {
    pub asset_id: String,
    pub events: Vec<AssetStateEvent>,
    pub total: usize,
}

#[derive(Debug, Serialize)]
pub struct AttestationRecordResponse {
    pub attestation_id: String,
    pub version: String,
    pub statement: String,
    pub key_id: String,
    pub issued_at: String,
    pub signature: String,
    pub canonical_payload: Value,
}

#[derive(Debug, Serialize)]
pub struct AssetAttestationsResponse {
    pub asset_id: String,
    pub asset_commitment_id: Option<String>,
    pub brand_attestation_status: Option<String>,
    pub platform_attestation_status: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub brand_attestation: Option<AttestationRecordResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub platform_attestation: Option<AttestationRecordResponse>,
}

/// GET /assets/:asset_id - Get asset detail
async fn get_asset(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(asset_id): Path<String>,
) -> Result<Json<AssetDetail>, axum::response::Response> {
    let asset = assets::fetch_asset_detail(&state.db, &asset_id)
        .await
        .map_err(super::error_response)?;

    // Check brand access for Brand and Factory roles
    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == asset.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    Ok(Json(asset))
}

/// GET /assets/:asset_id/history - Get asset audit history
async fn get_asset_history(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(asset_id): Path<String>,
) -> Result<Json<AssetHistoryResponse>, axum::response::Response> {
    // First check if asset exists and user has access
    let asset = assets::fetch_asset_detail(&state.db, &asset_id)
        .await
        .map_err(super::error_response)?;

    // Check brand access
    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == asset.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    let events = assets::fetch_asset_history(&state.db, &asset_id)
        .await
        .map_err(super::error_response)?;

    let total = events.len();

    Ok(Json(AssetHistoryResponse {
        asset_id,
        events,
        total,
    }))
}

async fn get_asset_attestations(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(asset_id): Path<String>,
) -> Result<Json<AssetAttestationsResponse>, axum::response::Response> {
    let asset = assets::fetch_asset_detail(&state.db, &asset_id)
        .await
        .map_err(super::error_response)?;

    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == asset.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    let brand_attestation = if let Some(commitment_id) = asset.asset_commitment_id.as_deref() {
        brand_attestations::fetch_brand_attestation_by_commitment(&state.db, commitment_id)
            .await
            .map_err(super::error_response)?
            .map(|record| AttestationRecordResponse {
                attestation_id: record.attestation_id,
                version: record.version,
                statement: record.statement,
                key_id: record.key_id,
                issued_at: record.issued_at.to_rfc3339(),
                signature: record.signature,
                canonical_payload: record.canonical_payload,
            })
    } else {
        None
    };

    let platform_attestation = if let Some(commitment_id) = asset.asset_commitment_id.as_deref() {
        platform_attestations::fetch_platform_attestation_by_commitment(&state.db, commitment_id)
            .await
            .map_err(super::error_response)?
            .map(|record| AttestationRecordResponse {
                attestation_id: record.attestation_id,
                version: record.version,
                statement: record.statement,
                key_id: record.key_id,
                issued_at: record.issued_at.to_rfc3339(),
                signature: record.signature,
                canonical_payload: record.canonical_payload,
            })
    } else {
        None
    };

    Ok(Json(AssetAttestationsResponse {
        asset_id,
        asset_commitment_id: asset.asset_commitment_id,
        brand_attestation_status: asset.brand_attestation_status,
        platform_attestation_status: asset.platform_attestation_status,
        brand_attestation,
        platform_attestation,
    }))
}

/// GET /assets - List assets with filters
async fn list_assets(
    State(state): State<AppState>,
    actor: ActorContext,
    Query(query): Query<AssetListQuery>,
) -> Result<Json<PaginatedResponse<AssetDetail>>, axum::response::Response> {
    // Determine brand filter based on role
    let brand_filter = match actor.actor_role {
        ActorRole::Platform => query.brand_id.as_deref(),
        _ => {
            // Brand and Factory can only see their own assets
            match actor.brand_id.as_deref() {
                Some(id) => {
                    // If query specifies a different brand_id, reject
                    if let Some(ref qb) = query.brand_id {
                        if qb != id {
                            return Err(super::error_response(RcError::BrandBoundaryViolation));
                        }
                    }
                    Some(id)
                }
                None => return Err(super::error_response(RcError::PermissionDenied)),
            }
        }
    };

    let (items, total) = assets::list_assets(
        &state.db,
        brand_filter,
        query.batch_id.as_deref(),
        query.current_state.as_deref(),
        query.page_size(),
        query.offset(),
    )
    .await
    .map_err(super::error_response)?;

    Ok(Json(PaginatedResponse {
        items,
        total,
        page: query.page(),
        page_size: query.page_size(),
    }))
}

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/assets", get(list_assets))
        .route("/assets/:asset_id", get(get_asset))
        .route("/assets/:asset_id/history", get(get_asset_history))
        .route("/assets/:asset_id/attestations", get(get_asset_attestations))
}
