use rc_common::errors::RcError;
use sqlx::PgPool;

#[allow(clippy::too_many_arguments)]
pub async fn insert_verification_event(
    pool: &PgPool,
    uid: &str,
    asset_id: Option<&str>,
    ctr: i32,
    verification_status: &str,
    risk_flags: &[String],
    cmac_valid: bool,
    client_ip: Option<&str>,
) -> Result<(), RcError> {
    sqlx::query(
        "INSERT INTO verification_events (uid, asset_id, ctr, verification_status, risk_flags, cmac_valid, client_ip) VALUES ($1, $2, $3, $4, $5, $6, $7)",
    )
    .bind(uid)
    .bind(asset_id)
    .bind(ctr)
    .bind(verification_status)
    .bind(risk_flags)
    .bind(cmac_valid)
    .bind(client_ip)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub async fn insert_verification_event_v2(
    pool: &PgPool,
    uid: &str,
    asset_id: Option<&str>,
    ctr: i32,
    verification_status: &str,
    risk_flags: &[String],
    cmac_valid: bool,
    client_ip: Option<&str>,
    asset_commitment_id: Option<&str>,
    brand_attestation_status: Option<&str>,
    platform_attestation_status: Option<&str>,
) -> Result<(), RcError> {
    sqlx::query(
        "INSERT INTO verification_events (uid, asset_id, ctr, verification_status, risk_flags, cmac_valid, client_ip, asset_commitment_id, verification_version, brand_attestation_status, platform_attestation_status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'v2', $9, $10)",
    )
    .bind(uid)
    .bind(asset_id)
    .bind(ctr)
    .bind(verification_status)
    .bind(risk_flags)
    .bind(cmac_valid)
    .bind(client_ip)
    .bind(asset_commitment_id)
    .bind(brand_attestation_status)
    .bind(platform_attestation_status)
    .execute(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}
