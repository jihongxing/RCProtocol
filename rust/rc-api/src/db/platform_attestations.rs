use rc_common::errors::RcError;
use sqlx::{postgres::PgRow, PgPool, Row};

use crate::attestation_platform::PlatformAttestationRecord;

fn map_row_to_record(row: PgRow) -> PlatformAttestationRecord {
    PlatformAttestationRecord {
        attestation_id: row.get("attestation_id"),
        version: row.get("version"),
        platform_id: row.get("platform_id"),
        asset_commitment_id: row.get("asset_commitment_id"),
        statement: row.get("statement"),
        key_id: row.get("key_id"),
        canonical_payload: row.get("canonical_payload"),
        signature: row.get("signature"),
        issued_at: row.get("issued_at"),
    }
}

pub async fn insert_platform_attestation(
    pool: &PgPool,
    record: &PlatformAttestationRecord,
) -> Result<(), RcError> {
    sqlx::query(
        r#"
        INSERT INTO platform_attestations (
            attestation_id,
            version,
            platform_id,
            asset_commitment_id,
            statement,
            key_id,
            canonical_payload,
            signature,
            issued_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (asset_commitment_id, statement) DO NOTHING
        "#,
    )
    .bind(&record.attestation_id)
    .bind(&record.version)
    .bind(&record.platform_id)
    .bind(&record.asset_commitment_id)
    .bind(&record.statement)
    .bind(&record.key_id)
    .bind(&record.canonical_payload)
    .bind(&record.signature)
    .bind(record.issued_at)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}

pub async fn fetch_platform_attestation_by_commitment(
    pool: &PgPool,
    asset_commitment_id: &str,
) -> Result<Option<PlatformAttestationRecord>, RcError> {
    let row = sqlx::query(
        r#"
        SELECT attestation_id, version, platform_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at
        FROM platform_attestations
        WHERE asset_commitment_id = $1
        ORDER BY created_at DESC
        LIMIT 1
        "#,
    )
    .bind(asset_commitment_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.map(map_row_to_record))
}

pub async fn fetch_platform_attestation_by_id(
    pool: &PgPool,
    attestation_id: &str,
) -> Result<PlatformAttestationRecord, RcError> {
    let row = sqlx::query(
        r#"
        SELECT attestation_id, version, platform_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at
        FROM platform_attestations
        WHERE attestation_id = $1
        "#,
    )
    .bind(attestation_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or_else(|| RcError::NotFound(format!("platform attestation not found: {attestation_id}")))?;

    Ok(map_row_to_record(row))
}
