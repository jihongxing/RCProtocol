use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};

use crate::app::AppState;
use crate::auth::extractor::ActorContext;
use crate::db::batches::{self, Batch, CreateBatchRequest};
use rc_common::errors::RcError;
use rc_common::types::ActorRole;

#[derive(Debug, Deserialize)]
pub struct PaginationParams {
    pub page: Option<i64>,
    pub page_size: Option<i64>,
}

impl PaginationParams {
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

/// POST /batches - Create a new batch
async fn create_batch(
    State(state): State<AppState>,
    actor: ActorContext,
    Json(payload): Json<CreateBatchRequest>,
) -> Result<impl IntoResponse, axum::response::Response> {
    // Only Platform, Brand, and Factory can create batches
    if !matches!(
        actor.actor_role,
        ActorRole::Platform | ActorRole::Brand | ActorRole::Factory
    ) {
        return Err(super::error_response(RcError::PermissionDenied));
    }

    // Brand and Factory can only create batches for their own brand
    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == payload.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    let batch = batches::create_batch(&state.db, &payload)
        .await
        .map_err(super::error_response)?;

    Ok((StatusCode::CREATED, Json(batch)))
}

/// GET /batches/:batch_id - Get a batch by ID
async fn get_batch(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(batch_id): Path<String>,
) -> Result<Json<Batch>, axum::response::Response> {
    let batch = batches::fetch_batch(&state.db, &batch_id)
        .await
        .map_err(super::error_response)?;

    // Check brand access
    if matches!(actor.actor_role, ActorRole::Brand | ActorRole::Factory) {
        match actor.brand_id.as_deref() {
            Some(id) if id == batch.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    Ok(Json(batch))
}

/// POST /batches/:batch_id/close - Close a batch
async fn close_batch(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(batch_id): Path<String>,
) -> Result<Json<Batch>, axum::response::Response> {
    // Fetch batch first to check brand access
    let batch = batches::fetch_batch(&state.db, &batch_id)
        .await
        .map_err(super::error_response)?;

    // Only Platform and Brand can close batches
    if !matches!(actor.actor_role, ActorRole::Platform | ActorRole::Brand) {
        return Err(super::error_response(RcError::PermissionDenied));
    }

    // Brand can only close their own batches
    if matches!(actor.actor_role, ActorRole::Brand) {
        match actor.brand_id.as_deref() {
            Some(id) if id == batch.brand_id => {}
            _ => return Err(super::error_response(RcError::BrandBoundaryViolation)),
        }
    }

    let closed_batch = batches::close_batch(&state.db, &batch_id)
        .await
        .map_err(super::error_response)?;

    Ok(Json(closed_batch))
}

/// GET /batches - List batches with pagination
async fn list_batches(
    State(state): State<AppState>,
    actor: ActorContext,
    Query(params): Query<PaginationParams>,
) -> Result<Json<PaginatedResponse<Batch>>, axum::response::Response> {
    // Determine brand filter based on role
    let brand_filter = match actor.actor_role {
        ActorRole::Platform => None,
        _ => actor.brand_id.as_deref(),
    };

    let (items, total) = batches::list_batches(
        &state.db,
        brand_filter,
        params.page_size(),
        params.offset(),
    )
    .await
    .map_err(super::error_response)?;

    Ok(Json(PaginatedResponse {
        items,
        total,
        page: params.page(),
        page_size: params.page_size(),
    }))
}

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/batches", post(create_batch).get(list_batches))
        .route("/batches/:batch_id", get(get_batch))
        .route("/batches/:batch_id/close", post(close_batch))
}
