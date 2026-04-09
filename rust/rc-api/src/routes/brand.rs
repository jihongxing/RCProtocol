use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use chrono::{DateTime, Utc};
use rc_common::errors::RcError;
use rc_common::types::ActorRole;
use serde::{Deserialize, Serialize};

use crate::app::AppState;
use crate::auth::extractor::ActorContext;

// --- Request types ---

#[derive(Debug, Deserialize)]
pub struct RegisterBrandRequest {
    pub brand_name: String,
    pub contact_email: String,
    pub industry: String,
}

#[derive(Debug, Deserialize)]
pub struct RotateApiKeyRequest {
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateBrandRequest {
    pub brand_name: Option<String>,
    pub brand_logo: Option<String>,
    pub brand_website: Option<String>,
    pub webhook_url: Option<String>,
}

// --- Response types ---

/// Used for list/detail endpoints — does NOT contain api_key or api_key_hash.
#[derive(Debug, Serialize)]
pub struct BrandDetailResponse {
    pub brand_id: String,
    pub brand_name: String,
    pub contact_email: String,
    pub industry: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct ApiKeyInfo {
    pub key_id: String,
    pub api_key: String,
    pub created_at: String,
    pub expires_at: Option<String>,
    pub note: String,
}

/// Returned only from the register endpoint — contains the plaintext api_key (shown once).
#[derive(Debug, Serialize)]
pub struct RegisterBrandResponse {
    pub brand_id: String,
    pub brand_name: String,
    pub contact_email: String,
    pub industry: String,
    pub status: String,
    pub api_key: ApiKeyInfo,
    pub created_at: String,
}

/// Returned from the rotate-api-key endpoint — contains the new plaintext api_key (shown once).
#[derive(Debug, Serialize)]
pub struct RotateApiKeyResponse {
    pub key_id: String,
    pub api_key: String,
    pub created_at: String,
    pub expires_at: Option<String>,
    pub revoked_key_id: String,
    pub note: String,
}

#[derive(Debug, Serialize)]
pub struct ApiKeyListItem {
    pub key_id: String,
    pub key_prefix: String,
    pub status: String,
    pub created_at: String,
    pub last_used_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct ApiKeyListResponse {
    pub keys: Vec<ApiKeyListItem>,
}

// Legacy response type for compatibility
#[derive(Debug, Serialize)]
pub struct BrandResponse {
    pub brand_id: String,
    pub brand_name: String,
    pub brand_logo: Option<String>,
    pub brand_website: Option<String>,
    pub webhook_url: Option<String>,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct PaginatedResponse<T: Serialize> {
    pub items: Vec<T>,
    pub total: i64,
    pub page: i64,
    pub page_size: i64,
}

// --- Pagination ---

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

// --- Validation ---

pub fn validate_name(name: &str, field: &str) -> Result<(), RcError> {
    let trimmed = name.trim();
    if trimmed.is_empty() {
        return Err(RcError::InvalidInput(format!("{field} cannot be empty")));
    }
    if trimmed.len() > 200 {
        return Err(RcError::InvalidInput(format!(
            "{field} too long (max 200 chars)"
        )));
    }
    Ok(())
}

pub fn validate_email(email: &str) -> Result<(), RcError> {
    let trimmed = email.trim();
    if trimmed.is_empty() {
        return Err(RcError::InvalidInput("email cannot be empty".into()));
    }
    if !trimmed.contains('@') {
        return Err(RcError::InvalidInput("invalid email format".into()));
    }
    if trimmed.len() > 255 {
        return Err(RcError::InvalidInput("email too long (max 255 chars)".into()));
    }
    Ok(())
}

pub fn validate_industry(industry: &str) -> Result<(), RcError> {
    let valid_industries = ["Watches", "Fashion", "Wine", "Jewelry", "Art", "Other"];
    if !valid_industries.contains(&industry) {
        return Err(RcError::InvalidInput(format!(
            "invalid industry, must be one of: {}",
            valid_industries.join(", ")
        )));
    }
    Ok(())
}

// --- Access control ---

pub fn check_role_allowed(actor: &ActorContext, allowed: &[ActorRole]) -> Result<(), RcError> {
    if allowed.contains(&actor.actor_role) {
        Ok(())
    } else {
        Err(RcError::PermissionDenied)
    }
}

pub fn check_brand_read_access(actor: &ActorContext, brand_id: &str) -> Result<(), RcError> {
    match actor.actor_role {
        ActorRole::Platform => Ok(()),
        ActorRole::Brand | ActorRole::Factory => match actor.brand_id.as_deref() {
            Some(id) if id == brand_id => Ok(()),
            _ => Err(RcError::BrandBoundaryViolation),
        },
        _ => Err(RcError::PermissionDenied),
    }
}

pub fn check_brand_write_access(actor: &ActorContext, brand_id: &str) -> Result<(), RcError> {
    match actor.actor_role {
        ActorRole::Platform => Ok(()),
        ActorRole::Brand => match actor.brand_id.as_deref() {
            Some(id) if id == brand_id => Ok(()),
            _ => Err(RcError::BrandBoundaryViolation),
        },
        _ => Err(RcError::PermissionDenied),
    }
}

// --- Brand handlers ---

async fn register_brand(
    State(state): State<AppState>,
    actor: ActorContext,
    Json(payload): Json<RegisterBrandRequest>,
) -> Result<impl IntoResponse, axum::response::Response> {
    // 1. Permission check: only Platform role can register brands
    check_role_allowed(&actor, &[ActorRole::Platform])
        .map_err(super::error_response)?;

    // 2. Validate input
    validate_name(&payload.brand_name, "brand_name")
        .map_err(super::error_response)?;
    validate_email(&payload.contact_email)
        .map_err(super::error_response)?;
    validate_industry(&payload.industry)
        .map_err(super::error_response)?;

    // 3. Check email uniqueness
    if let Some(_) = crate::db::brands::fetch_brand_by_email(&state.db, &payload.contact_email)
        .await
        .map_err(super::error_response)?
    {
        return Err(super::error_response(RcError::Conflict(
            "品牌邮箱已存在".into(),
        )));
    }

    // 4. Generate brand_id and API key
    let brand_id = crate::auth::api_key::generate_brand_id();
    let api_key_plain = crate::auth::api_key::generate_api_key();
    let key_hash = crate::auth::api_key::hash_api_key(&api_key_plain);
    let key_prefix = crate::auth::api_key::extract_key_prefix(&api_key_plain);
    let key_id = crate::auth::api_key::generate_key_id();

    // 5. Transaction: insert brand and API key
    let mut tx = state
        .db
        .begin()
        .await
        .map_err(|e| super::error_response(RcError::Database(e.to_string())))?;

    let brand = crate::db::brands::create_brand(
        &mut tx,
        &brand_id,
        payload.brand_name.trim(),
        payload.contact_email.trim(),
        &payload.industry,
    )
    .await
    .map_err(super::error_response)?;

    let api_key_record = crate::db::brands::create_api_key(&mut tx, &key_id, &brand_id, &key_hash, &key_prefix)
        .await
        .map_err(super::error_response)?;

    tx.commit()
        .await
        .map_err(|e| super::error_response(RcError::Database(e.to_string())))?;

    // 6. Return response with plaintext API key (shown only once)
    Ok((
        StatusCode::CREATED,
        Json(RegisterBrandResponse {
            brand_id: brand.brand_id,
            brand_name: brand.brand_name,
            contact_email: brand.contact_email,
            industry: brand.industry,
            status: brand.status,
            api_key: ApiKeyInfo {
                key_id: api_key_record.key_id,
                api_key: api_key_plain,
                created_at: api_key_record.created_at.to_rfc3339(),
                expires_at: None,
                note: "⚠️ 此 API Key 仅显示一次，请妥善保管".into(),
            },
            created_at: brand.created_at.to_rfc3339(),
        }),
    ))
}

async fn list_brands(
    State(state): State<AppState>,
    actor: ActorContext,
    Query(params): Query<PaginationParams>,
) -> Result<Json<PaginatedResponse<BrandResponse>>, axum::response::Response> {
    check_role_allowed(
        &actor,
        &[ActorRole::Platform, ActorRole::Brand, ActorRole::Factory],
    )
    .map_err(super::error_response)?;

    let brand_filter = match actor.actor_role {
        ActorRole::Platform => None,
        _ => actor.brand_id.as_deref().map(String::from),
    };

    let (items, total) = crate::db::brands::list_brands(
        &state.db,
        brand_filter.as_deref(),
        &params,
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

async fn get_brand(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(brand_id): Path<String>,
) -> Result<Json<BrandDetailResponse>, axum::response::Response> {
    check_brand_read_access(&actor, &brand_id)
        .map_err(super::error_response)?;

    let brand = crate::db::brands::fetch_brand_detail(&state.db, &brand_id)
        .await
        .map_err(super::error_response)?;

    Ok(Json(brand))
}

async fn update_brand(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(brand_id): Path<String>,
    Json(payload): Json<UpdateBrandRequest>,
) -> Result<Json<BrandResponse>, axum::response::Response> {
    check_brand_write_access(&actor, &brand_id)
        .map_err(super::error_response)?;

    if let Some(ref name) = payload.brand_name {
        validate_name(name, "brand_name")
            .map_err(super::error_response)?;
    }

    let brand = crate::db::brands::update_brand(&state.db, &brand_id, &payload)
        .await
        .map_err(super::error_response)?;

    Ok(Json(brand))
}

async fn rotate_api_key(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(brand_id): Path<String>,
    Json(payload): Json<RotateApiKeyRequest>,
) -> Result<impl IntoResponse, axum::response::Response> {
    // 1. Permission check
    check_brand_write_access(&actor, &brand_id)
        .map_err(super::error_response)?;

    // 2. Generate new API key
    let new_api_key = crate::auth::api_key::generate_api_key();
    let new_hash = crate::auth::api_key::hash_api_key(&new_api_key);
    let new_prefix = crate::auth::api_key::extract_key_prefix(&new_api_key);
    let new_key_id = crate::auth::api_key::generate_key_id();

    // 3. Transaction: revoke old key and create new key
    let mut tx = state
        .db
        .begin()
        .await
        .map_err(|e| super::error_response(RcError::Database(e.to_string())))?;

    // Get current active key
    let old_key = crate::db::brands::fetch_active_api_key(&mut tx, &brand_id)
        .await
        .map_err(super::error_response)?;

    // Revoke old key
    crate::db::brands::revoke_api_key(&mut tx, &old_key.key_id, payload.reason.as_deref())
        .await
        .map_err(super::error_response)?;

    // Create new key
    let new_key_record = crate::db::brands::create_api_key(&mut tx, &new_key_id, &brand_id, &new_hash, &new_prefix)
        .await
        .map_err(super::error_response)?;

    tx.commit()
        .await
        .map_err(|e| super::error_response(RcError::Database(e.to_string())))?;

    // 4. Return response
    Ok(Json(RotateApiKeyResponse {
        key_id: new_key_record.key_id,
        api_key: new_api_key,
        created_at: new_key_record.created_at.to_rfc3339(),
        expires_at: None,
        revoked_key_id: old_key.key_id,
        note: "⚠️ 旧 API Key 已失效，请使用新密钥".into(),
    }))
}

async fn list_api_keys(
    State(state): State<AppState>,
    actor: ActorContext,
    Path(brand_id): Path<String>,
) -> Result<Json<ApiKeyListResponse>, axum::response::Response> {
    // 1. Permission check
    check_brand_read_access(&actor, &brand_id)
        .map_err(super::error_response)?;

    // 2. Fetch all API keys for this brand
    let keys = crate::db::brands::list_api_keys(&state.db, &brand_id)
        .await
        .map_err(super::error_response)?;

    // 3. Convert to response format
    let items = keys
        .into_iter()
        .map(|k| ApiKeyListItem {
            key_id: k.key_id,
            key_prefix: k.key_prefix,
            status: k.status,
            created_at: k.created_at.to_rfc3339(),
            last_used_at: k.last_used_at.map(|t| t.to_rfc3339()),
            revoked_at: k.revoked_at.map(|t| t.to_rfc3339()),
        })
        .collect();

    Ok(Json(ApiKeyListResponse { keys: items }))
}

// --- Batch query params ---

#[derive(Debug, Deserialize)]
pub struct BatchIdsParams {
    pub ids: String,
}

/// M12: GET /brands/batch?ids=brand-1,brand-2 — 批量查询品牌名称
async fn batch_brands(
    State(state): State<AppState>,
    _actor: ActorContext,
    Query(params): Query<BatchIdsParams>,
) -> Result<Json<Vec<BrandResponse>>, axum::response::Response> {
    let ids: Vec<String> = params
        .ids
        .split(',')
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect();

    if ids.is_empty() {
        return Ok(Json(vec![]));
    }
    if ids.len() > 100 {
        return Err(super::error_response(RcError::InvalidInput(
            "batch size exceeds 100".into(),
        )));
    }

    let brands = crate::db::brands::fetch_brands_batch(&state.db, &ids)
        .await
        .map_err(super::error_response)?;

    Ok(Json(brands))
}

// --- Router ---

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/brands", post(register_brand).get(list_brands))
        .route("/brands/batch", get(batch_brands))
        .route("/brands/:brand_id", get(get_brand).put(update_brand))
        .route("/brands/:brand_id/api-keys/rotate", post(rotate_api_key))
        .route("/brands/:brand_id/api-keys", get(list_api_keys))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn actor(role: ActorRole, brand_id: Option<&str>) -> ActorContext {
        ActorContext {
            actor_id: "test-user".into(),
            actor_role: role,
            actor_org: None,
            brand_id: brand_id.map(String::from),
        }
    }

    // --- validate_name tests ---

    #[test]
    fn test_validate_name_empty() {
        let err = validate_name("", "brand_name").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_name_too_long() {
        let long = "a".repeat(201);
        let err = validate_name(&long, "brand_name").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_name_whitespace_only() {
        let err = validate_name("   ", "brand_name").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_name_ok() {
        assert!(validate_name("Acme Corp", "brand_name").is_ok());
    }

    // --- validate_email tests ---

    #[test]
    fn test_validate_email_empty() {
        let err = validate_email("").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_email_no_at() {
        let err = validate_email("notanemail").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_email_too_long() {
        let long = format!("{}@example.com", "a".repeat(250));
        let err = validate_email(&long).unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_email_ok() {
        assert!(validate_email("test@example.com").is_ok());
    }

    // --- validate_industry tests ---

    #[test]
    fn test_validate_industry_invalid() {
        let err = validate_industry("InvalidIndustry").unwrap_err();
        assert!(matches!(err, RcError::InvalidInput(_)));
    }

    #[test]
    fn test_validate_industry_ok() {
        assert!(validate_industry("Watches").is_ok());
        assert!(validate_industry("Fashion").is_ok());
        assert!(validate_industry("Wine").is_ok());
        assert!(validate_industry("Other").is_ok());
    }

    // --- check_role_allowed tests ---

    #[test]
    fn test_check_role_allowed_pass() {
        let a = actor(ActorRole::Platform, None);
        assert!(check_role_allowed(&a, &[ActorRole::Platform, ActorRole::Brand]).is_ok());
    }

    #[test]
    fn test_check_role_allowed_reject() {
        let a = actor(ActorRole::Consumer, None);
        let err = check_role_allowed(&a, &[ActorRole::Platform, ActorRole::Brand]).unwrap_err();
        assert!(matches!(err, RcError::PermissionDenied));
    }

    // --- check_brand_read_access tests ---

    #[test]
    fn test_brand_read_access_platform() {
        let a = actor(ActorRole::Platform, None);
        assert!(check_brand_read_access(&a, "brand-abc").is_ok());
    }

    #[test]
    fn test_brand_read_access_brand_match() {
        let a = actor(ActorRole::Brand, Some("brand-abc"));
        assert!(check_brand_read_access(&a, "brand-abc").is_ok());
    }

    #[test]
    fn test_brand_read_access_brand_mismatch() {
        let a = actor(ActorRole::Brand, Some("brand-abc"));
        let err = check_brand_read_access(&a, "brand-xyz").unwrap_err();
        assert!(matches!(err, RcError::BrandBoundaryViolation));
    }

    #[test]
    fn test_brand_read_access_consumer() {
        let a = actor(ActorRole::Consumer, None);
        let err = check_brand_read_access(&a, "brand-abc").unwrap_err();
        assert!(matches!(err, RcError::PermissionDenied));
    }

    // --- check_brand_write_access tests ---

    #[test]
    fn test_brand_write_access_factory() {
        let a = actor(ActorRole::Factory, Some("brand-abc"));
        let err = check_brand_write_access(&a, "brand-abc").unwrap_err();
        assert!(matches!(err, RcError::PermissionDenied));
    }

    // --- Request/Response serialization tests ---

    #[test]
    fn test_register_brand_request_deserialization() {
        let json = r#"{"brand_name":"Acme Corp","contact_email":"contact@acme.com","industry":"Watches"}"#;
        let req: RegisterBrandRequest = serde_json::from_str(json).expect("deserialize RegisterBrandRequest");
        assert_eq!(req.brand_name, "Acme Corp");
        assert_eq!(req.contact_email, "contact@acme.com");
        assert_eq!(req.industry, "Watches");
    }

    #[test]
    fn test_rotate_api_key_request_optional_reason() {
        let json = r#"{}"#;
        let req: RotateApiKeyRequest = serde_json::from_str(json).expect("deserialize without reason");
        assert!(req.reason.is_none());

        let json = r#"{"reason":"Security rotation"}"#;
        let req: RotateApiKeyRequest = serde_json::from_str(json).expect("deserialize with reason");
        assert_eq!(req.reason.as_deref(), Some("Security rotation"));
    }

    #[test]
    fn test_brand_detail_response_no_api_key() {
        let resp = BrandDetailResponse {
            brand_id: "brand-abc123".into(),
            brand_name: "Acme Corp".into(),
            contact_email: "contact@acme.com".into(),
            industry: "Watches".into(),
            status: "Active".into(),
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };
        let json = serde_json::to_string(&resp).expect("serialize BrandDetailResponse");
        assert!(!json.contains("api_key"), "BrandDetailResponse must not contain api_key");
        assert!(!json.contains("api_key_hash"), "BrandDetailResponse must not contain api_key_hash");
        assert!(json.contains("contact_email"), "BrandDetailResponse must contain contact_email");
        assert!(json.contains("industry"), "BrandDetailResponse must contain industry");
    }

    #[test]
    fn test_update_brand_partial() {
        let payload = UpdateBrandRequest {
            brand_name: None,
            brand_logo: None,
            brand_website: None,
            webhook_url: Some("https://example.com/hook".into()),
        };

        assert!(payload.brand_name.is_none());
        assert!(payload.brand_logo.is_none());
        assert!(payload.brand_website.is_none());
        assert_eq!(payload.webhook_url.as_deref(), Some("https://example.com/hook"));
    }
}
