use rc_common::errors::RcError;
use sqlx::{PgPool, Row};
use uuid::Uuid;

pub struct AuthorityDeviceRow {
    pub authority_id: Uuid,
    pub authority_uid: String,
    pub authority_type: String,
    pub brand_id: String,
    pub status: String,
    pub key_epoch: i32,
    pub virtual_credential_hash: Option<String>,
    pub bound_user_id: Option<String>,
    pub physical_chip_uid: Option<String>,
    pub last_known_ctr: Option<i32>,
}

/// 插入母卡设备记录，返回生成的 authority_id
#[allow(clippy::too_many_arguments)]
pub async fn insert_authority_device(
    pool: &PgPool,
    authority_uid: &str,
    authority_type: &str,
    brand_id: &str,
    key_epoch: i32,
    virtual_credential_hash: Option<&str>,
    bound_user_id: Option<&str>,
    created_by: Option<&str>,
) -> Result<Uuid, RcError> {
    let row = sqlx::query(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id, created_by) \
         VALUES ($1, $2, $3, $4, $5, $6, $7) \
         RETURNING authority_id",
    )
    .bind(authority_uid)
    .bind(authority_type)
    .bind(brand_id)
    .bind(key_epoch)
    .bind(virtual_credential_hash)
    .bind(bound_user_id)
    .bind(created_by)
    .fetch_one(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.get("authority_id"))
}

/// 通过 asset_entanglements 联查当前资产的 Active 母卡设备
pub async fn fetch_authority_device_by_asset(
    pool: &PgPool,
    asset_id: &str,
) -> Result<AuthorityDeviceRow, RcError> {
    let row = sqlx::query(
        "SELECT ad.authority_id, ad.authority_uid, ad.authority_type, ad.brand_id, \
                ad.status, ad.key_epoch, ad.virtual_credential_hash, ad.bound_user_id, \
                ad.physical_chip_uid, ad.last_known_ctr \
         FROM authority_devices ad \
         INNER JOIN asset_entanglements ae ON ae.authority_id = ad.authority_id \
         WHERE ae.asset_id = $1 AND ae.entanglement_state = 'Active' \
         LIMIT 1",
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?;

    Ok(AuthorityDeviceRow {
        authority_id: row.get("authority_id"),
        authority_uid: row.get("authority_uid"),
        authority_type: row.get("authority_type"),
        brand_id: row.get("brand_id"),
        status: row.get("status"),
        key_epoch: row.get("key_epoch"),
        virtual_credential_hash: row.get("virtual_credential_hash"),
        bound_user_id: row.get("bound_user_id"),
        physical_chip_uid: row.get("physical_chip_uid"),
        last_known_ctr: row.get("last_known_ctr"),
    })
}

/// 原子更新 CTR，仅当新 CTR 大于旧 CTR 或旧 CTR 为 NULL 时更新
/// 返回 true 表示更新成功，false 表示并发冲突或 CTR 不递增
pub async fn atomic_update_ctr(
    pool: &PgPool,
    authority_id: Uuid,
    new_ctr: i32,
) -> Result<bool, RcError> {
    let result = sqlx::query(
        "UPDATE authority_devices \
         SET last_known_ctr = $1 \
         WHERE authority_id = $2 \
           AND (last_known_ctr < $1 OR last_known_ctr IS NULL) \
         RETURNING authority_id",
    )
    .bind(new_ctr)
    .bind(authority_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(result.is_some())
}
