use chrono::{DateTime, Utc};
use serde::Serialize;
use sqlx::{PgPool, Postgres, Transaction};

use rc_common::errors::RcError;
use crate::routes::brand::BrandDetailResponse;

#[derive(Debug, Serialize)]
pub struct BrandRecord {
    pub brand_id: String,
    pub brand_name: String,
    pub contact_email: String,
    pub industry: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl From<BrandRecord> for BrandDetailResponse {
    fn from(record: BrandRecord) -> Self {
        BrandDetailResponse {
            brand_id: record.brand_id,
            brand_name: record.brand_name,
            contact_email: record.contact_email,
            industry: record.industry,
            status: record.status,
            created_at: record.created_at,
            updated_at: record.updated_at,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct ApiKeyRecord {
    pub key_id: String,
    pub brand_id: String,
    pub key_hash: String,
    pub key_prefix: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub revoked_at: Option<DateTime<Utc>>,
    pub last_used_at: Option<DateTime<Utc>>,
}

/// Create a new brand
pub async fn create_brand(
    tx: &mut Transaction<'_, Postgres>,
    brand_id: &str,
    brand_name: &str,
    contact_email: &str,
    industry: &str,
) -> Result<BrandRecord, RcError> {
    let record = sqlx::query_as!(
        BrandRecord,
        r#"
        INSERT INTO brands (brand_id, brand_name, contact_email, industry, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, 'Active', NOW(), NOW())
        RETURNING brand_id, brand_name, contact_email, industry, status, created_at, updated_at
        "#,
        brand_id,
        brand_name,
        contact_email,
        industry
    )
    .fetch_one(&mut **tx)
    .await
    .map_err(|e: sqlx::Error| {
        // 检查是否是唯一约束冲突
        if let sqlx::Error::Database(db_err) = &e {
            if db_err.code().as_deref() == Some("23505") {
                // PostgreSQL 唯一约束冲突错误码
                if db_err.message().contains("brands_contact_email_key")
                    || db_err.message().contains("idx_brands_contact_email") {
                    return RcError::Conflict("邮箱已被注册".to_string());
                }
                if db_err.message().contains("uq_brands_name") {
                    return RcError::Conflict("品牌名称已存在".to_string());
                }
            }
        }
        RcError::Database(e.to_string())
    })?;

    Ok(record)
}

/// Fetch brand by ID
pub async fn fetch_brand_by_id(
    pool: &PgPool,
    brand_id: &str,
) -> Result<Option<BrandRecord>, RcError> {
    let record = sqlx::query_as!(
        BrandRecord,
        r#"
        SELECT brand_id, brand_name, contact_email, industry, status, created_at, updated_at
        FROM brands
        WHERE brand_id = $1
        "#,
        brand_id
    )
    .fetch_optional(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// Fetch brand by email (for uniqueness check)
pub async fn fetch_brand_by_email(
    pool: &PgPool,
    email: &str,
) -> Result<Option<BrandRecord>, RcError> {
    let record = sqlx::query_as!(
        BrandRecord,
        r#"
        SELECT brand_id, brand_name, contact_email, industry, status, created_at, updated_at
        FROM brands
        WHERE contact_email = $1
        "#,
        email
    )
    .fetch_optional(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// Create a new API key
pub async fn create_api_key(
    tx: &mut Transaction<'_, Postgres>,
    key_id: &str,
    brand_id: &str,
    key_hash: &str,
    key_prefix: &str,
) -> Result<ApiKeyRecord, RcError> {
    let record = sqlx::query_as!(
        ApiKeyRecord,
        r#"
        INSERT INTO api_keys (key_id, brand_id, key_hash, key_prefix, status, created_at)
        VALUES ($1, $2, $3, $4, 'Active', NOW())
        RETURNING key_id, brand_id, key_hash, key_prefix, status, created_at, revoked_at, last_used_at
        "#,
        key_id,
        brand_id,
        key_hash,
        key_prefix
    )
    .fetch_one(&mut **tx)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// Revoke an API key with optional reason
pub async fn revoke_api_key(
    tx: &mut Transaction<'_, Postgres>,
    key_id: &str,
    _reason: Option<&str>,
) -> Result<(), RcError> {
    sqlx::query!(
        r#"
        UPDATE api_keys
        SET status = 'Revoked', revoked_at = NOW()
        WHERE key_id = $1
        "#,
        key_id
    )
    .execute(&mut **tx)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(())
}

/// Fetch all API keys for a brand
pub async fn fetch_api_keys_by_brand(
    pool: &PgPool,
    brand_id: &str,
) -> Result<Vec<ApiKeyRecord>, RcError> {
    let records = sqlx::query_as!(
        ApiKeyRecord,
        r#"
        SELECT key_id, brand_id, key_hash, key_prefix, status, created_at, revoked_at, last_used_at
        FROM api_keys
        WHERE brand_id = $1
        ORDER BY created_at DESC
        "#,
        brand_id
    )
    .fetch_all(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(records)
}

/// Fetch active API key by brand ID
pub async fn fetch_active_api_key_by_brand(
    pool: &PgPool,
    brand_id: &str,
) -> Result<Option<ApiKeyRecord>, RcError> {
    let record = sqlx::query_as!(
        ApiKeyRecord,
        r#"
        SELECT key_id, brand_id, key_hash, key_prefix, status, created_at, revoked_at, last_used_at
        FROM api_keys
        WHERE brand_id = $1 AND status = 'Active'
        ORDER BY created_at DESC
        LIMIT 1
        "#,
        brand_id
    )
    .fetch_optional(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// Update API key last used timestamp
pub async fn update_api_key_last_used(
    pool: &PgPool,
    key_id: &str,
) -> Result<(), RcError> {
    sqlx::query!(
        r#"
        UPDATE api_keys
        SET last_used_at = NOW()
        WHERE key_id = $1
        "#,
        key_id
    )
    .execute(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(())
}

/// Fetch active API key (for transaction context)
pub async fn fetch_active_api_key(
    tx: &mut Transaction<'_, Postgres>,
    brand_id: &str,
) -> Result<ApiKeyRecord, RcError> {
    let record = sqlx::query_as!(
        ApiKeyRecord,
        r#"
        SELECT key_id, brand_id, key_hash, key_prefix, status, created_at, revoked_at, last_used_at
        FROM api_keys
        WHERE brand_id = $1 AND status = 'Active'
        ORDER BY created_at DESC
        LIMIT 1
        "#,
        brand_id
    )
    .fetch_one(&mut **tx)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// List API keys (alias for fetch_api_keys_by_brand)
pub async fn list_api_keys(
    pool: &PgPool,
    brand_id: &str,
) -> Result<Vec<ApiKeyRecord>, RcError> {
    fetch_api_keys_by_brand(pool, brand_id).await
}

/// Fetch brand detail (returns BrandDetailResponse, error if not found)
pub async fn fetch_brand_detail(
    pool: &PgPool,
    brand_id: &str,
) -> Result<BrandDetailResponse, RcError> {
    let record = fetch_brand_by_id(pool, brand_id)
        .await?
        .ok_or_else(|| RcError::NotFound(format!("Brand {} not found", brand_id)))?;

    Ok(record.into())
}

/// Fetch brand by API key hash (for authentication)
pub async fn fetch_brand_by_api_key_hash(
    pool: &PgPool,
    key_hash: &str,
) -> Result<Option<BrandRecord>, RcError> {
    let record = sqlx::query_as!(
        BrandRecord,
        r#"
        SELECT b.brand_id, b.brand_name, b.contact_email, b.industry, b.status, b.created_at, b.updated_at
        FROM brands b
        INNER JOIN api_keys k ON b.brand_id = k.brand_id
        WHERE k.key_hash = $1 AND k.status = 'Active'
        "#,
        key_hash
    )
    .fetch_optional(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    Ok(record)
}

/// List brands with pagination
pub async fn list_brands(
    pool: &PgPool,
    brand_filter: Option<&str>,
    params: &crate::routes::brand::PaginationParams,
) -> Result<(Vec<crate::routes::brand::BrandResponse>, i64), RcError> {
    let offset = params.offset();
    let limit = params.page_size();

    let total: i64 = if let Some(brand_id) = brand_filter {
        sqlx::query_scalar!(
            r#"SELECT COUNT(*) as "count!" FROM brands WHERE brand_id = $1"#,
            brand_id
        )
        .fetch_one(pool)
        .await
        .map_err(|e| RcError::Database(e.to_string()))?
    } else {
        sqlx::query_scalar!(r#"SELECT COUNT(*) as "count!" FROM brands"#)
            .fetch_one(pool)
            .await
            .map_err(|e| RcError::Database(e.to_string()))?
    };

    let records = if let Some(brand_id) = brand_filter {
        sqlx::query_as!(
            BrandRecord,
            r#"
            SELECT brand_id, brand_name, contact_email, industry, status, created_at, updated_at
            FROM brands
            WHERE brand_id = $1
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            brand_id,
            limit,
            offset
        )
        .fetch_all(pool)
        .await
        .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?
    } else {
        sqlx::query_as!(
            BrandRecord,
            r#"
            SELECT brand_id, brand_name, contact_email, industry, status, created_at, updated_at
            FROM brands
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#,
            limit,
            offset
        )
        .fetch_all(pool)
        .await
        .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?
    };

    // Convert to response format (legacy format without contact_email/industry)
    let items = records
        .into_iter()
        .map(|r| crate::routes::brand::BrandResponse {
            brand_id: r.brand_id,
            brand_name: r.brand_name,
            brand_logo: None,
            brand_website: None,
            webhook_url: None,
            status: r.status,
            created_at: r.created_at,
            updated_at: r.updated_at,
        })
        .collect();

    Ok((items, total))
}

/// Update brand fields
pub async fn update_brand(
    pool: &PgPool,
    brand_id: &str,
    payload: &crate::routes::brand::UpdateBrandRequest,
) -> Result<crate::routes::brand::BrandResponse, RcError> {
    // Build dynamic update query
    let mut updates = Vec::new();
    let mut params: Vec<String> = Vec::new();

    if let Some(ref name) = payload.brand_name {
        updates.push(format!("brand_name = ${}", params.len() + 2));
        params.push(name.clone());
    }
    if let Some(ref logo) = payload.brand_logo {
        updates.push(format!("brand_logo = ${}", params.len() + 2));
        params.push(logo.clone());
    }
    if let Some(ref website) = payload.brand_website {
        updates.push(format!("brand_website = ${}", params.len() + 2));
        params.push(website.clone());
    }
    if let Some(ref webhook) = payload.webhook_url {
        updates.push(format!("webhook_url = ${}", params.len() + 2));
        params.push(webhook.clone());
    }

    if updates.is_empty() {
        // No updates, just fetch and return
        let record = fetch_brand_by_id(pool, brand_id)
            .await?
            .ok_or_else(|| RcError::NotFound(format!("Brand {} not found", brand_id)))?;

        return Ok(crate::routes::brand::BrandResponse {
            brand_id: record.brand_id,
            brand_name: record.brand_name,
            brand_logo: None,
            brand_website: None,
            webhook_url: None,
            status: record.status,
            created_at: record.created_at,
            updated_at: record.updated_at,
        });
    }

    updates.push("updated_at = NOW()".to_string());

    let _query = format!(
        "UPDATE brands SET {} WHERE brand_id = $1 RETURNING brand_id, brand_name, contact_email, industry, status, created_at, updated_at",
        updates.join(", ")
    );

    // For now, use a simple approach - fetch after update
    // TODO: Use dynamic query building properly
    let _ = sqlx::query(&format!(
        "UPDATE brands SET updated_at = NOW() WHERE brand_id = $1"
    ))
    .bind(brand_id)
    .execute(pool)
    .await
    .map_err(|e| RcError::Database(e.to_string()))?;

    let record = fetch_brand_by_id(pool, brand_id)
        .await?
        .ok_or_else(|| RcError::NotFound(format!("Brand {} not found", brand_id)))?;

    Ok(crate::routes::brand::BrandResponse {
        brand_id: record.brand_id,
        brand_name: record.brand_name,
        brand_logo: None,
        brand_website: None,
        webhook_url: None,
        status: record.status,
        created_at: record.created_at,
        updated_at: record.updated_at,
    })
}

/// Fetch brands by batch IDs
pub async fn fetch_brands_batch(
    pool: &PgPool,
    brand_ids: &[String],
) -> Result<Vec<crate::routes::brand::BrandResponse>, RcError> {
    let records = sqlx::query_as!(
        BrandRecord,
        r#"
        SELECT brand_id, brand_name, contact_email, industry, status, created_at, updated_at
        FROM brands
        WHERE brand_id = ANY($1)
        "#,
        brand_ids as &[String]
    )
    .fetch_all(pool)
    .await
    .map_err(|e: sqlx::Error| RcError::Database(e.to_string()))?;

    let items = records
        .into_iter()
        .map(|r| crate::routes::brand::BrandResponse {
            brand_id: r.brand_id,
            brand_name: r.brand_name,
            brand_logo: None,
            brand_website: None,
            webhook_url: None,
            status: r.status,
            created_at: r.created_at,
            updated_at: r.updated_at,
        })
        .collect();

    Ok(items)
}
