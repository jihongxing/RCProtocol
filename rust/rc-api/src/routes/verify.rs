use axum::{extract::{Query, State}, http::StatusCode, response::IntoResponse, routing::get, Json, Router};
use rc_common::errors::RcError;
use serde::{Deserialize, Serialize};

use crate::app::AppState;
use crate::attestation_brand::{BrandAttestationPayloadV1, load_brand_signing_key, verify_brand_attestation};
use crate::attestation_platform::{PlatformAttestationPayloadV1, load_platform_signing_key, verify_platform_attestation};
use crate::cache::ctr_cache::CtrCache;
use crate::db::{asset_commitments, assets, brand_attestations, platform_attestations, verification};

const E_AC_NOT_FOUND: &str = "ASSET_COMMITMENT_NOT_FOUND";
const E_BA_MISSING: &str = "BRAND_ATTESTATION_MISSING";
const E_BA_INVALID: &str = "BRAND_ATTESTATION_INVALID";
const E_PA_MISSING: &str = "PLATFORM_ATTESTATION_MISSING";
const E_PA_INVALID: &str = "PLATFORM_ATTESTATION_INVALID";
const E_REPLAY: &str = "REPLAY_SUSPECTED";
const E_TAG_FAILED: &str = "TAG_AUTHENTICATION_FAILED";

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
pub struct VerificationV2Response {
    pub verification_version: String,
    pub tag_authentication: String,
    pub attestation_status: VerificationAttestationStatus,
    pub protocol_state: VerificationProtocolState,
    pub verification_status: String,
    pub error_codes: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset: Option<VerifyAssetInfoV2>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scan_metadata: Option<ScanMetadata>,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_commitment_id: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct VerificationAttestationStatus {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_commitment_id: Option<String>,
    pub brand_attestation: String,
    pub platform_attestation: String,
}

#[derive(Debug, Serialize)]
pub struct VerificationProtocolState {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub current_state: Option<String>,
    pub risk_flags: Vec<String>,
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
        "Activated" | "LegallySold" | "Transferred" => ("display_allowed", vec![]),
        "Disputed" => ("restricted", vec!["frozen_asset".to_string()]),
        "Tampered" | "Compromised" | "Destructed" => ("restricted", vec![]),
        _ => ("display_allowed", vec![]),
    }
}

struct Summary {
    asset_commitment_id: Option<String>,
    brand: String,
    platform: String,
    error_codes: Vec<String>,
}

fn v2_error() -> VerificationV2Response {
    VerificationV2Response {
        verification_version: "v2".into(),
        tag_authentication: "error".into(),
        attestation_status: VerificationAttestationStatus { asset_commitment_id: None, brand_attestation: "unknown".into(), platform_attestation: "unknown".into() },
        protocol_state: VerificationProtocolState { current_state: None, risk_flags: vec![] },
        verification_status: "error".into(),
        error_codes: vec![],
        asset: None,
        scan_metadata: None,
    }
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
    if v2 { (StatusCode::BAD_REQUEST, Json(v2_error())).into_response() } else { (StatusCode::BAD_REQUEST, Json(VerifyResponse { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None })).into_response() }
}

async fn handle_degraded(state: &AppState, uid_hex: &str, v2: bool) -> axum::response::Response {
    match assets::fetch_asset_by_uid(&state.db, uid_hex).await {
        Ok(row) if v2 => {
            let s = match build_summary(state, &row).await { Ok(v) => v, Err(_) => return server_error(true) };
            Json(VerificationV2Response { verification_version: "v2".into(), tag_authentication: "not_performed".into(), attestation_status: VerificationAttestationStatus { asset_commitment_id: s.asset_commitment_id.clone(), brand_attestation: s.brand.clone(), platform_attestation: s.platform.clone() }, protocol_state: VerificationProtocolState { current_state: Some(row.current_state.clone()), risk_flags: vec![] }, verification_status: "unverified".into(), error_codes: s.error_codes, asset: Some(VerifyAssetInfoV2 { asset_id: row.asset_id, brand_id: row.brand_id, product_id: row.product_id, uid: row.uid, current_state: row.current_state, asset_commitment_id: s.asset_commitment_id }), scan_metadata: None }).into_response()
        }
        Ok(_) => Json(VerifyResponse { verification_status: "unverified".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response(),
        Err(RcError::AssetNotFound) if v2 => Json(VerificationV2Response { verification_version: "v2".into(), tag_authentication: "unknown_tag".into(), attestation_status: VerificationAttestationStatus { asset_commitment_id: None, brand_attestation: "unknown".into(), platform_attestation: "unknown".into() }, protocol_state: VerificationProtocolState { current_state: None, risk_flags: vec![] }, verification_status: "unknown_tag".into(), error_codes: vec![], asset: None, scan_metadata: None }).into_response(),
        Err(RcError::AssetNotFound) => Json(VerifyResponse { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response(),
        Err(_) => server_error(v2),
    }
}

async fn handle_full_verify(state: &AppState, uid_hex: &str, parsed: ParsedSunParams, v2: bool) -> axum::response::Response {
    let asset = match assets::fetch_asset_by_uid(&state.db, uid_hex).await {
        Ok(row) => row,
        Err(RcError::AssetNotFound) => {
            let _ = verification::insert_verification_event(&state.db, uid_hex, None, parsed.ctr_value as i32, "unknown_tag", &[], false, None).await;
            return if v2 { Json(VerificationV2Response { verification_version: "v2".into(), tag_authentication: "unknown_tag".into(), attestation_status: VerificationAttestationStatus { asset_commitment_id: None, brand_attestation: "unknown".into(), platform_attestation: "unknown".into() }, protocol_state: VerificationProtocolState { current_state: None, risk_flags: vec![] }, verification_status: "unknown_tag".into(), error_codes: vec![], asset: None, scan_metadata: None }).into_response() } else { Json(VerifyResponse { verification_status: "unknown_tag".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response() };
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
        return if v2 { Json(VerificationV2Response { verification_version: "v2".into(), tag_authentication: "failed".into(), attestation_status: VerificationAttestationStatus { asset_commitment_id: None, brand_attestation: "unknown".into(), platform_attestation: "unknown".into() }, protocol_state: VerificationProtocolState { current_state: Some(asset.current_state.clone()), risk_flags: vec![] }, verification_status: "authentication_failed".into(), error_codes: vec![E_TAG_FAILED.into()], asset: None, scan_metadata: None }).into_response() } else { Json(VerifyResponse { verification_status: "authentication_failed".into(), risk_flags: vec![], asset: None, scan_metadata: None }).into_response() };
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

    let (state_status, mut state_flags) = evaluate_status(&asset.current_state);
    risk_flags.append(&mut state_flags);
    let now = chrono::Utc::now().to_rfc3339();

    if v2 {
        let s = match build_summary(state, &asset).await { Ok(v) => v, Err(_) => return server_error(true) };
        let tag_authentication = if risk_flags.iter().any(|f| f == "replay_suspected") { "replay_suspected".to_string() } else { "passed".to_string() };
        let mut error_codes = s.error_codes.clone();
        if risk_flags.iter().any(|f| f == "replay_suspected") { error_codes.push(E_REPLAY.into()); }
        let verification_status = if tag_authentication == "replay_suspected" { "replay_suspected".to_string() } else if s.brand != "valid" || s.platform != "valid" { "incomplete_attestation".to_string() } else if state_status == "restricted" { "restricted".to_string() } else { "authentic".to_string() };
        tracing::info!(verification_version = "v2", uid = uid_hex, asset_id = %asset.asset_id, asset_commitment_id = ?s.asset_commitment_id, brand_attestation_status = %s.brand, platform_attestation_status = %s.platform, tag_authentication = %tag_authentication, current_state = %asset.current_state, verification_status = %verification_status, risk_flags = ?risk_flags, error_codes = ?error_codes, "verification v2 completed");
        let _ = verification::insert_verification_event_v2(&state.db, uid_hex, Some(&asset.asset_id), new_ctr as i32, &verification_status, &risk_flags, true, None, s.asset_commitment_id.as_deref(), Some(&s.brand), Some(&s.platform)).await;
        Json(VerificationV2Response {
            verification_version: "v2".into(),
            tag_authentication,
            attestation_status: VerificationAttestationStatus { asset_commitment_id: s.asset_commitment_id.clone(), brand_attestation: s.brand.clone(), platform_attestation: s.platform.clone() },
            protocol_state: VerificationProtocolState { current_state: Some(asset.current_state.clone()), risk_flags: risk_flags.clone() },
            verification_status,
            error_codes,
            asset: Some(VerifyAssetInfoV2 { asset_id: asset.asset_id, brand_id: asset.brand_id, product_id: asset.product_id, uid: asset.uid, current_state: asset.current_state, asset_commitment_id: s.asset_commitment_id }),
            scan_metadata: Some(ScanMetadata { ctr: new_ctr, previous_ctr, verified_at: now }),
        }).into_response()
    } else {
        let verification_status = if risk_flags.iter().any(|f| f == "replay_suspected") { "replay_suspected".to_string() } else if state_status == "restricted" { "restricted".to_string() } else { "verified".to_string() };
        let _ = verification::insert_verification_event(&state.db, uid_hex, Some(&asset.asset_id), new_ctr as i32, &verification_status, &risk_flags, true, None).await;
        Json(VerifyResponse {
            verification_status,
            risk_flags,
            asset: Some(VerifyAssetInfo { asset_id: asset.asset_id, brand_id: asset.brand_id, product_id: asset.product_id, uid: asset.uid, current_state: asset.current_state }),
            scan_metadata: Some(ScanMetadata { ctr: new_ctr, previous_ctr, verified_at: now }),
        }).into_response()
    }
}

async fn resolve_asset_commitment_id(state: &AppState, asset: &assets::AssetVerifyRow) -> Result<Option<String>, RcError> {
    if let Some(id) = asset.asset_commitment_id.clone() { return Ok(Some(id)); }
    Ok(asset_commitments::fetch_asset_commitment_by_uid_epoch(&state.db, &asset.uid, asset.epoch).await?.map(|r| r.commitment_id))
}

async fn build_summary(state: &AppState, asset: &assets::AssetVerifyRow) -> Result<Summary, RcError> {
    let asset_commitment_id = resolve_asset_commitment_id(state, asset).await?;
    let Some(commitment_id) = asset_commitment_id.clone() else { return Ok(Summary { asset_commitment_id: None, brand: "missing".into(), platform: "missing".into(), error_codes: vec![E_AC_NOT_FOUND.into()] }); };
    let brand = validate_brand_attestation(state, asset, &commitment_id).await?;
    let platform = validate_platform_attestation(state, &commitment_id).await?;
    let mut error_codes = Vec::new();
    match brand.as_str() { "missing" => error_codes.push(E_BA_MISSING.into()), "invalid" => error_codes.push(E_BA_INVALID.into()), _ => {} }
    match platform.as_str() { "missing" => error_codes.push(E_PA_MISSING.into()), "invalid" => error_codes.push(E_PA_INVALID.into()), _ => {} }
    Ok(Summary { asset_commitment_id: Some(commitment_id), brand, platform, error_codes })
}

async fn validate_brand_attestation(state: &AppState, asset: &assets::AssetVerifyRow, commitment_id: &str) -> Result<String, RcError> {
    let Some(record) = brand_attestations::fetch_brand_attestation_by_commitment(&state.db, commitment_id).await? else { return Ok("missing".into()); };
    let payload: BrandAttestationPayloadV1 = match serde_json::from_value(record.canonical_payload.clone()) { Ok(v) => v, Err(_) => return Ok("invalid".into()) };
    let (_, signing_key) = load_brand_signing_key()?;
    let valid = verify_brand_attestation(&payload, &record.signature, &signing_key.verifying_key())?;
    let fields = payload.asset_commitment_id == commitment_id && payload.brand_id == asset.brand_id && payload.key_id == record.key_id && payload.statement == record.statement && payload.version == record.version;
    Ok(if valid && fields { "valid" } else { "invalid" }.into())
}

async fn validate_platform_attestation(state: &AppState, commitment_id: &str) -> Result<String, RcError> {
    let Some(record) = platform_attestations::fetch_platform_attestation_by_commitment(&state.db, commitment_id).await? else { return Ok("missing".into()); };
    let payload: PlatformAttestationPayloadV1 = match serde_json::from_value(record.canonical_payload.clone()) { Ok(v) => v, Err(_) => return Ok("invalid".into()) };
    let (platform_id, _, signing_key) = load_platform_signing_key()?;
    let valid = verify_platform_attestation(&payload, &record.signature, &signing_key.verifying_key())?;
    let fields = payload.asset_commitment_id == commitment_id && payload.platform_id == platform_id && payload.key_id == record.key_id && payload.statement == record.statement && payload.version == record.version;
    Ok(if valid && fields { "valid" } else { "invalid" }.into())
}

fn server_error(v2: bool) -> axum::response::Response {
    if v2 {
        (StatusCode::INTERNAL_SERVER_ERROR, Json(v2_error())).into_response()
    } else {
        (StatusCode::INTERNAL_SERVER_ERROR, Json(VerifyResponse { verification_status: "error".into(), risk_flags: vec![], asset: None, scan_metadata: None })).into_response()
    }
}

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/verify", get(verify_handler))
        .route("/verify/v2", get(verify_v2_handler))
}
