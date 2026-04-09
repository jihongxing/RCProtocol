use rc_common::errors::RcError;
use sqlx::{postgres::PgRow, PgPool, Row};

use crate::asset_commitment::AssetCommitmentRecord;

fn map_row_to_record(row: PgRow) -> AssetCommitmentRecord {
    AssetCommitmentRecord {
        commitment_id: row.get("commitment_id"),
        payload_version: row.get("payload_version"),
        brand_id: row.get("brand_id"),
        asset_uid: row.get("asset_uid"),
        chip_binding: row.get("chip_binding"),
        epoch: row.get("epoch"),
        metadata_hash: row.get("metadata_hash"),
        canonical_payload: row.get("canonical_payload"),
        created_at: row.get("created_at"),
    }
}

pub async fn insert_asset_commitment(
    pool: &PgPool,
    record: &AssetCommitmentRecord,
) -> Result<(), RcError> {
    sqlx::query(
        r#"
        INSERT INTO asset_commitments (
            commitment_id,
            payload_version,
            brand_id,
            asset_uid,
            chip_binding,
            epoch,
            metadata_hash,
            canonical_payload
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (commitment_id) DO NOTHING
        "#,
    )
    .bind(&record.commitment_id)
    .bind(&record.payload_version)
    .bind(&record.brand_id)
    .bind(&record.asset_uid)
    .bind(&record.chip_binding)
    .bind(record.epoch)
    .bind(&record.metadata_hash)
    .bind(&record.canonical_payload)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}

pub async fn fetch_asset_commitment_by_id(
    pool: &PgPool,
    commitment_id: &str,
) -> Result<AssetCommitmentRecord, RcError> {
    let row = sqlx::query(
        r#"
        SELECT commitment_id, payload_version, brand_id, asset_uid, chip_binding, epoch, metadata_hash, canonical_payload, created_at
        FROM asset_commitments
        WHERE commitment_id = $1
        "#,
    )
    .bind(commitment_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or_else(|| RcError::NotFound(format!("asset commitment not found: {commitment_id}")))?;

    Ok(map_row_to_record(row))
}

pub async fn fetch_asset_commitment_by_uid_epoch(
    pool: &PgPool,
    asset_uid: &str,
    epoch: i32,
) -> Result<Option<AssetCommitmentRecord>, RcError> {
    let row = sqlx::query(
        r#"
        SELECT commitment_id, payload_version, brand_id, asset_uid, chip_binding, epoch, metadata_hash, canonical_payload, created_at
        FROM asset_commitments
        WHERE asset_uid = $1 AND epoch = $2
        ORDER BY created_at DESC
        LIMIT 1
        "#,
    )
    .bind(asset_uid)
    .bind(epoch)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.map(map_row_to_record))
}

pub async fn bind_asset_commitment_to_asset(
    pool: &PgPool,
    asset_id: &str,
    commitment_id: &str,
) -> Result<(), RcError> {
    sqlx::query(
        r#"
        UPDATE assets
        SET asset_commitment_id = $2,
            updated_at = NOW()
        WHERE asset_id = $1
        "#,
    )
    .bind(asset_id)
    .bind(commitment_id)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}
