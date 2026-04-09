use chrono::{DateTime, Utc};
use rc_common::{errors::RcError, ids};
use serde::{Deserialize, Serialize};
use sqlx::{PgPool, Row};

#[derive(Debug, Serialize, Deserialize)]
pub struct Batch {
    pub batch_id: String,
    pub brand_id: String,
    pub batch_name: Option<String>,
    pub factory_id: Option<String>,
    pub status: String,
    pub expected_count: Option<i32>,
    pub actual_count: i32,
    pub created_at: DateTime<Utc>,
    pub closed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Deserialize)]
pub struct CreateBatchRequest {
    pub brand_id: String,
    pub batch_name: Option<String>,
    pub factory_id: Option<String>,
    pub expected_count: Option<i32>,
}

/// Create a new batch
pub async fn create_batch(
    pool: &PgPool,
    req: &CreateBatchRequest,
) -> Result<Batch, RcError> {
    let batch_id = ids::generate_batch_id();

    let row = sqlx::query(
        r#"
        INSERT INTO batches (batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count)
        VALUES ($1, $2, $3, $4, 'Open', $5, 0)
        RETURNING batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count, created_at, closed_at
        "#
    )
    .bind(&batch_id)
    .bind(&req.brand_id)
    .bind(&req.batch_name)
    .bind(&req.factory_id)
    .bind(req.expected_count)
    .fetch_one(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(Batch {
        batch_id: row.get("batch_id"),
        brand_id: row.get("brand_id"),
        batch_name: row.get("batch_name"),
        factory_id: row.get("factory_id"),
        status: row.get("status"),
        expected_count: row.get("expected_count"),
        actual_count: row.get("actual_count"),
        created_at: row.get("created_at"),
        closed_at: row.get("closed_at"),
    })
}

/// Fetch a batch by ID
pub async fn fetch_batch(pool: &PgPool, batch_id: &str) -> Result<Batch, RcError> {
    let row = sqlx::query(
        r#"
        SELECT batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count, created_at, closed_at
        FROM batches
        WHERE batch_id = $1
        "#
    )
    .bind(batch_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?; // Reuse AssetNotFound for now

    Ok(Batch {
        batch_id: row.get("batch_id"),
        brand_id: row.get("brand_id"),
        batch_name: row.get("batch_name"),
        factory_id: row.get("factory_id"),
        status: row.get("status"),
        expected_count: row.get("expected_count"),
        actual_count: row.get("actual_count"),
        created_at: row.get("created_at"),
        closed_at: row.get("closed_at"),
    })
}

/// Close a batch
pub async fn close_batch(pool: &PgPool, batch_id: &str) -> Result<Batch, RcError> {
    let row = sqlx::query(
        r#"
        UPDATE batches
        SET status = 'Closed', closed_at = NOW()
        WHERE batch_id = $1 AND status = 'Open'
        RETURNING batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count, created_at, closed_at
        "#
    )
    .bind(batch_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or_else(|| RcError::InvalidInput("Batch not found or already closed".to_string()))?;

    Ok(Batch {
        batch_id: row.get("batch_id"),
        brand_id: row.get("brand_id"),
        batch_name: row.get("batch_name"),
        factory_id: row.get("factory_id"),
        status: row.get("status"),
        expected_count: row.get("expected_count"),
        actual_count: row.get("actual_count"),
        created_at: row.get("created_at"),
        closed_at: row.get("closed_at"),
    })
}

/// List batches for a brand with pagination
pub async fn list_batches(
    pool: &PgPool,
    brand_id: Option<&str>,
    limit: i64,
    offset: i64,
) -> Result<(Vec<Batch>, i64), RcError> {
    let count_query = if let Some(brand) = brand_id {
        sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM batches WHERE brand_id = $1")
            .bind(brand)
            .fetch_one(pool)
            .await
    } else {
        sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM batches")
            .fetch_one(pool)
            .await
    };

    let total = count_query.map_err(|err| RcError::Database(err.to_string()))?;

    let rows = if let Some(brand) = brand_id {
        sqlx::query(
            r#"
            SELECT batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count, created_at, closed_at
            FROM batches
            WHERE brand_id = $1
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#
        )
        .bind(brand)
        .bind(limit)
        .bind(offset)
        .fetch_all(pool)
        .await
    } else {
        sqlx::query(
            r#"
            SELECT batch_id, brand_id, batch_name, factory_id, status, expected_count, actual_count, created_at, closed_at
            FROM batches
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(pool)
        .await
    };

    let batches = rows
        .map_err(|err| RcError::Database(err.to_string()))?
        .into_iter()
        .map(|row| Batch {
            batch_id: row.get("batch_id"),
            brand_id: row.get("brand_id"),
            batch_name: row.get("batch_name"),
            factory_id: row.get("factory_id"),
            status: row.get("status"),
            expected_count: row.get("expected_count"),
            actual_count: row.get("actual_count"),
            created_at: row.get("created_at"),
            closed_at: row.get("closed_at"),
        })
        .collect();

    Ok((batches, total))
}
