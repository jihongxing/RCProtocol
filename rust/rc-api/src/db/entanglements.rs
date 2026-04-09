use rc_common::errors::RcError;
use sqlx::{PgPool, Row};
use uuid::Uuid;

pub struct EntanglementRow {
    pub entanglement_id: Uuid,
    pub asset_id: String,
    pub child_uid: String,
    pub authority_id: Uuid,
    pub authority_uid: String,
    pub entanglement_state: String,
    pub bound_by: String,
}

/// 插入母子绑定记录，返回生成的 entanglement_id
pub async fn insert_entanglement(
    pool: &PgPool,
    asset_id: &str,
    child_uid: &str,
    authority_id: Uuid,
    authority_uid: &str,
    bound_by: &str,
) -> Result<Uuid, RcError> {
    let row = sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) \
         VALUES ($1, $2, $3, $4, 'Active', $5) \
         RETURNING entanglement_id",
    )
    .bind(asset_id)
    .bind(child_uid)
    .bind(authority_id)
    .bind(authority_uid)
    .bind(bound_by)
    .fetch_one(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.get("entanglement_id"))
}

/// 查询资产当前 Active 状态的绑定记录
pub async fn fetch_active_entanglement(
    pool: &PgPool,
    asset_id: &str,
) -> Result<Option<EntanglementRow>, RcError> {
    let row = sqlx::query(
        "SELECT entanglement_id, asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by \
         FROM asset_entanglements \
         WHERE asset_id = $1 AND entanglement_state = 'Active' \
         LIMIT 1",
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.map(|r| EntanglementRow {
        entanglement_id: r.get("entanglement_id"),
        asset_id: r.get("asset_id"),
        child_uid: r.get("child_uid"),
        authority_id: r.get("authority_id"),
        authority_uid: r.get("authority_uid"),
        entanglement_state: r.get("entanglement_state"),
        bound_by: r.get("bound_by"),
    }))
}
