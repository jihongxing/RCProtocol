use axum::{extract::{Query, State}, http::StatusCode, response::IntoResponse, routing::get, Json, Router};
use rc_common::errors::RcError;
use serde::{Deserialize, Serialize};

use crate::app::AppState;
use crate::cache::ctr_cache::CtrCache;
use crate::db::{assets, brand_attestations, platform_attestations, verification};

#[derive(Debug, Deserialize)]
pub struct VerifyParams {
    pub uid: Option<String>,
    pub ctr: Option<String>,
    pub cmac: Option<String>,
}

struct ParsedSunParams {
    uid: [u8; 7],
    ctr: [u8; 3],
    cmac: [u8; 8],
    ctr_value: u32,
}

#[derive(Debug, Serialize)]
pub struct VerifyResponse {
    pub verification_status: String,
    pub risk_flags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset: Option<VerifyAssetInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scan_metadata: Option<ScanMetadata>,
}

#[derive(Debug, Serialize)]
pub struct VerifyResponseV2 {
    pub verification_status: String,
    pub risk_flags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset: Option<VerifyAssetInfoV2>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scan_metadata: Option<ScanMetadata>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub attestation_summary: Option<AttestationSummary>,
}

#[derive(Debug, Serialize)]
pub struct VerifyAssetInfo {
    pub asset_id: String,
    pub brand_id: String,
    pub product_id: Option<String>,
    pub uid: String,
    pub current_state: String,
}

#[derive(Debug, Serialize)]
pub struct VerifyAssetInfoV2 {
    pub asset_id: String,
    pub brand_id: String,
    pub product_id: Option<String>,
    pub uid: String,
    pub current_state: String,
    pub asset_commitment_id: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct AttestationSummary {
    pub asset_commitment_id: Option<String>,
    pub brand_attestation_status: Option<String>,
    pub platform_attestation_status: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct ScanMetadata {
    pub ctr: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previous_ctr: Option<u32>,
    pub verified_at: String,
}

fn parse_sun_params(params: &VerifyParams) -> Result<ParsedSunParams, RcError> {
    let uid_hex = params.uid.as_deref().ok_or_else(|| RcError::InvalidInput("missing parameter: uid".into()))?;
    let ctr_hex = params.ctr.as_deref().ok_or_else(|| RcError::InvalidInput("missing parameter: ctr".into()))?;
    let cmac_hex = params.cmac.as_deref().ok_or_else(|| RcError::InvalidInput("missing parameter: cmac".into()))?;
    let uid_bytes = hex::decode(uid_hex).map_err(|_| RcError::InvalidInput("invalid uid hex format".into()))?;
    let ctr_bytes = hex::decode(ctr_hex).map_err(|_| RcError::InvalidInput("invalid ctr hex format".into()))?;
    let cmac_bytes = hex::decode(cmac_hex).map_err(|_| RcError::InvalidInput("invalid cmac hex format".into()))?;
    let uid: [u8; 7] = uid_bytes.try_into().map_err(|_| RcError::InvalidInput("uid must be 7 bytes (14 hex chars)".into()))?;
    let ctr: [u8; 3] = ctr_bytes.try_into().map_err(|_| RcError::InvalidInput("ctr must be 3 bytes (6 hex chars)".into()))?;
    let cmac: [u8; 8] = cmac_bytes.try_into().map_err(|_| RcError::InvalidInput("cmac must be 8 bytes (16 hex chars)".into()))?;
    let ctr_value = u32::from_le_bytes([ctr[0], ctr[1], ctr[2], 0]);
    Ok(ParsedSunParams { uid, ctr, cmac, ctr_value })
}

fn is_degraded_mode(params: &VerifyParams) -> bool {
    params.uid.is_some() && params.ctr.is_none() && params.cmac.is_none()
}

fn evaluate_status(current_state: &str) -> (&'static str, Vec<String>) {
    match current_state {
        "Activated" | "LegallySold" | "Transferred" => ("verified", vec![]),
        "Disputed" => ("restricted", vec!["frozen_asset".to_string()]),
        "Tampered" | "Compromised" | "Destructed" => ("restricted", vec![]),
        _ => ("verified", vec![]),
    }
}

fn attestation_status(value: bool) -> Option<String> {
    if value { Some("issued".to_string()) } else { None }
}

pub async fn verify_handler(Query(params): Query<VerifyParams>, State(state): State<AppState>) -> impl IntoResponse {
    verify_impl(state, params, false).await
}

pub async fn verify_v2_handler(Query(params): Query<VerifyParams>, State(state): State<AppState>) -> impl IntoResponse {
    verify_impl(state, params, true).await
}

async fn verify_impl(state: AppState, params: VerifyParams, v2: bool) -> axum::response::Response {
    if params.uid.is_none() {
        return bad_request(v2);
    }
    let uid_hex = params.uid.as_deref().unwrap();
    if is_degraded_mode(&params) {
        return handle_degraded(&state, uid_hex, v2).await;
    }
    let parsed = match parse_sun_params(&params) {
        Ok(v) => v,
        Err(_) => return bad_request(v2),
    };
    handle_full_verify(&state, uid_hex, parsed, v2).await
}

fn bad_request(v2: bool) -> axum::response::Response {
    if v2 {
        (StatusCode::BAD_REQUEST, Json(VerifyResponseV2 { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: None })).into_response()
    } else {
        (StatusCode::BAD_REQUEST, Json(VerifyResponse { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None })).into_response()
    }
}

async fn handle_degraded(state: &AppState, uid_hex: &str, v2: bool) -> axum::response::Response {
    match assets::fetch_asset_by_uid(&state.db, uid_hex).await {
        Ok(row) => {
            let summary = if v2 { Some(build_attestation_summary(state, row.asset_commitment_id.as_deref()).await) } else { None };
            if v2 {
                Json(VerifyResponseV2 { verification_status: "unverified".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: summary }).into_response()
            } else {
                Json(VerifyResponse { verification_status: "unverified".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response()
            }
        }
        Err(RcError::AssetNotFound) => {
            if v2 {
                Json(VerifyResponseV2 { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: None }).into_response()
            } else {
                Json(VerifyResponse { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response()
            }
        }
        Err(_) => server_error(v2),
    }
}

async fn handle_full_verify(state: &AppState, uid_hex: &str, parsed: ParsedSunParams, v2: bool) -> axum::response::Response {
    let asset = match assets::fetch_asset_by_uid(&state.db, uid_hex).await {
        Ok(row) => row,
        Err(RcError::AssetNotFound) => {
            let _ = verification::insert_verification_event(&state.db, uid_hex, None, parsed.ctr_value as i32, "unknown_tag", &[], false, None).await;
            return if v2 { Json(VerifyResponseV2 { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: None }).into_response() } else { Json(VerifyResponse { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response() };
        }
        Err(_) => return server_error(v2),
    };

    let k_chip = match state.kms.derive_chip_key(&asset.brand_id, &parsed.uid, asset.epoch as u32) {
        Ok(key) => key,
        Err(_) => return server_error(v2),
    };
    let cmac_valid = rc_crypto::sun::verify_sun_message(k_chip.as_bytes(), &parsed.uid, &parsed.ctr, &parsed.cmac);
    if !cmac_valid {
        let _ = verification::insert_verification_event(&state.db, uid_hex, Some(&asset.asset_id), parsed.ctr_value as i32, "authentication_failed", &[], false, None).await;
        return if v2 { Json(VerifyResponseV2 { verification_status: "authentication_failed".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: None }).into_response() } else { Json(VerifyResponse { verification_status: "authentication_failed".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response() };
    }

    let ctr_cache = CtrCache::new(state.ctr_cache.clone(), state.redis.clone(), state.db.clone(), state.fallback_strategy);
    let new_ctr = parsed.ctr_value;
    let (cached_ctr, _) = match ctr_cache.get_ctr(uid_hex).await { Ok(v) => v, Err(_) => return server_error(v2) };
    let previous_ctr = if cached_ctr == 0 { None } else { Some(cached_ctr) };
    let mut risk_flags = Vec::new();
    if new_ctr <= cached_ctr && cached_ctr > 0 {
        risk_flags.push("replay_suspected".to_string());
    } else if assets::update_asset_ctr(&state.db, &asset.asset_id, new_ctr as i32).await.is_ok() {
        ctr_cache.update_ctr(uid_hex, new_ctr).await;
    } else {
        return server_error(v2);
    }

    let (status, mut state_flags) = evaluate_status(&asset.current_state);
    risk_flags.append(&mut state_flags);
    let verification_status = if risk_flags.iter().any(|f| f == "replay_suspected") { "replay_suspected".to_string() } else { status.to_string() };
    let now = chrono::Utc::now().to_rfc3339();

    let summary = if v2 { Some(build_attestation_summary(state, asset.asset_commitment_id.as_deref()).await) } else { None };
    if v2 {
        let s = summary.as_ref().unwrap();
        let _ = verification::insert_verification_event_v2(&state.db, uid_hex, Some(&asset.asset_id), new_ctr as i32, &verification_status, &risk_flags, true, None, s.asset_commitment_id.as_deref(), s.brand_attestation_status.as_deref(), s.platform_attestation_status.as_deref()).await;
        Json(VerifyResponseV2 {
            verification_status,
            risk_flags,
            asset: Some(VerifyAssetInfoV2 { asset_id: asset.asset_id, brand_id: asset.brand_id, product_id: asset.product_id, uid: asset.uid, current_state: asset.current_state, asset_commitment_id: asset.asset_commitment_id }),
            scan_metadata: Some(ScanMetadata { ctr: new_ctr, previous_ctr, verified_at: now }),
            attestation_summary: summary,
        }).into_response()
    } else {
        let _ = verification::insert_verification_event(&state.db, uid_hex, Some(&asset.asset_id), new_ctr as i32, &verification_status, &risk_flags, true, None).await;
        Json(VerifyResponse {
            verification_status,
            risk_flags,
            asset: Some(VerifyAssetInfo { asset_id: asset.asset_id, brand_id: asset.brand_id, product_id: asset.product_id, uid: asset.uid, current_state: asset.current_state }),
            scan_metadata: Some(ScanMetadata { ctr: new_ctr, previous_ctr, verified_at: now }),
        }).into_response()
    }
}

async fn build_attestation_summary(state: &AppState, asset_commitment_id: Option<&str>) -> AttestationSummary {
    let Some(commitment_id) = asset_commitment_id else {
        return AttestationSummary { asset_commitment_id: None, brand_attestation_status: None, platform_attestation_status: None };
    };
    let brand = brand_attestations::fetch_brand_attestation_by_commitment(&state.db, commitment_id).await.ok().flatten().is_some();
    let platform = platform_attestations::fetch_platform_attestation_by_commitment(&state.db, commitment_id).await.ok().flatten().is_some();
    AttestationSummary {
        asset_commitment_id: Some(commitment_id.to_string()),
        brand_attestation_status: attestation_status(brand),
        platform_attestation_status: attestation_status(platform),
    }
}

fn server_error(v2: bool) -> axum::response::Response {
    if v2 {
        (StatusCode::INTERNAL_SERVER_ERROR, Json(VerifyResponseV2 { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None, attestation_summary: None })).into_response()
    } else {
        (StatusCode::INTERNAL_SERVER_ERROR, Json(VerifyResponse { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None })).into_response()
    }
}

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/verify", get(verify_handler))
        .route("/verify/v2", get(verify_v2_handler))
}
