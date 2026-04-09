use rc_common::errors::RcError;
use sqlx::{postgres::PgRow, PgPool, Row};

use crate::attestation_brand::BrandAttestationRecord;

fn map_row_to_record(row: PgRow) -> BrandAttestationRecord {
    BrandAttestationRecord {
        attestation_id: row.get("attestation_id"),
        version: row.get("version"),
        brand_id: row.get("brand_id"),
        asset_commitment_id: row.get("asset_commitment_id"),
        statement: row.get("statement"),
        key_id: row.get("key_id"),
        canonical_payload: row.get("canonical_payload"),
        signature: row.get("signature"),
        issued_at: row.get("issued_at"),
    }
}

pub async fn insert_brand_attestation(
    pool: &PgPool,
    record: &BrandAttestationRecord,
) -> Result<(), RcError> {
    sqlx::query(
        r#"
        INSERT INTO brand_attestations (
            attestation_id,
            version,
            brand_id,
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
    .bind(&record.brand_id)
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

pub async fn fetch_brand_attestation_by_commitment(
    pool: &PgPool,
    asset_commitment_id: &str,
) -> Result<Option<BrandAttestationRecord>, RcError> {
    let row = sqlx::query(
        r#"
        SELECT attestation_id, version, brand_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at
        FROM brand_attestations
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

pub async fn fetch_brand_attestation_by_id(
    pool: &PgPool,
    attestation_id: &str,
) -> Result<BrandAttestationRecord, RcError> {
    let row = sqlx::query(
        r#"
        SELECT attestation_id, version, brand_id, asset_commitment_id, statement, key_id, canonical_payload, signature, issued_at
        FROM brand_attestations
        WHERE attestation_id = $1
        "#,
    )
    .bind(attestation_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or_else(|| RcError::NotFound(format!("brand attestation not found: {attestation_id}")))?;

    Ok(map_row_to_record(row))
}
