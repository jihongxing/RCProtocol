use chrono::{DateTime, Utc};
use rc_common::errors::RcError;
use serde::Serialize;
use sqlx::{PgPool, Row};
use uuid::Uuid;

#[derive(Debug, Serialize)]
pub struct TransferRecord {
    pub transfer_id: Uuid,
    pub asset_id: String,
    pub from_user_id: String,
    pub to_user_id: String,
    pub from_owner_id: String,
    pub to_owner_id: String,
    pub transfer_type: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

/// 插入过户记录
pub async fn insert_transfer(
    pool: &PgPool,
    asset_id: &str,
    from_owner_id: &str,
    to_owner_id: &str,
    transfer_type: &str,
    idempotency_key: &str,
    trace_id: &str,
) -> Result<Uuid, RcError> {
    let row = sqlx::query(
        "INSERT INTO asset_transfers (asset_id, from_user_id, to_user_id, from_owner_id, to_owner_id, transfer_type, idempotency_key, trace_id, metadata) \
         VALUES ($1, $2, $3, $2, $3, $4, $5, $6, jsonb_build_object('status', 'pending')) \
         RETURNING transfer_id",
    )
    .bind(asset_id)
    .bind(from_owner_id)
    .bind(to_owner_id)
    .bind(transfer_type)
    .bind(idempotency_key)
    .bind(trace_id)
    .fetch_one(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.get("transfer_id"))
}

pub async fn fetch_transfer(pool: &PgPool, transfer_id: Uuid) -> Result<TransferRecord, RcError> {
    let row = sqlx::query(
        "SELECT transfer_id, asset_id, from_user_id, to_user_id, from_owner_id, to_owner_id, transfer_type, \
                COALESCE(metadata->>'status', 'pending') AS status, transferred_at \
         FROM asset_transfers \
         WHERE transfer_id = $1",
    )
    .bind(transfer_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::NotFound("transfer not found".into()))?;

    Ok(TransferRecord {
        transfer_id: row.get("transfer_id"),
        asset_id: row.get("asset_id"),
        from_user_id: row.get("from_user_id"),
        to_user_id: row.get("to_user_id"),
        from_owner_id: row.get("from_owner_id"),
        to_owner_id: row.get("to_owner_id"),
        transfer_type: row.get("transfer_type"),
        status: row.get("status"),
        created_at: row.get("transferred_at"),
    })
}

pub async fn update_transfer_status(
    pool: &PgPool,
    transfer_id: Uuid,
    target_status: &str,
) -> Result<TransferRecord, RcError> {
    let row = sqlx::query(
        r#"UPDATE asset_transfers
           SET metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{status}', to_jsonb($2::text), true)
         WHERE transfer_id = $1
           AND COALESCE(metadata->>'status', 'pending') = 'pending'
         RETURNING transfer_id, asset_id, from_user_id, to_user_id, from_owner_id, to_owner_id, transfer_type,
                   COALESCE(metadata->>'status', $2) AS status, transferred_at"#,
    )
    .bind(transfer_id)
    .bind(target_status)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    match row {
        Some(row) => Ok(TransferRecord {
            transfer_id: row.get("transfer_id"),
            asset_id: row.get("asset_id"),
            from_user_id: row.get("from_user_id"),
            to_user_id: row.get("to_user_id"),
            from_owner_id: row.get("from_owner_id"),
            to_owner_id: row.get("to_owner_id"),
            transfer_type: row.get("transfer_type"),
            status: row.get("status"),
            created_at: row.get("transferred_at"),
        }),
        None => {
            let existing = fetch_transfer(pool, transfer_id).await?;
            if existing.status != "pending" {
                return Err(RcError::Conflict("transfer already processed".into()));
            }
            Err(RcError::Conflict("transfer status update failed".into()))
        }
    }
}

pub async fn confirm_transfer(pool: &PgPool, transfer_id: Uuid) -> Result<TransferRecord, RcError> {
    update_transfer_status(pool, transfer_id, "confirmed").await
}

pub async fn reject_transfer(pool: &PgPool, transfer_id: Uuid) -> Result<TransferRecord, RcError> {
    update_transfer_status(pool, transfer_id, "rejected").await
}
