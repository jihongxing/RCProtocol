pub mod asset_commitments;
pub mod assets;
pub mod authority_devices;
pub mod batches;
pub mod brand_attestations;
pub mod brands;
pub mod entanglements;
pub mod platform_attestations;
pub mod products;
pub mod transfers;
pub mod verification;

use rc_common::{
    audit::AuditEvent,
    errors::RcError,
    types::{AssetAction, AssetRecord, AssetState},
};
use serde::Serialize;
use serde_json::Value;
use sqlx::{PgPool, Row};

#[derive(Debug)]
pub struct IdempotencyRecord {
    pub request_hash: String,
    pub response_snapshot: Value,
}

#[derive(Debug, Serialize)]
pub struct VerifyAssetView {
    pub asset_id: String,
    pub brand_id: String,
    pub product_id: Option<String>,
    pub uid: Option<String>,
    pub current_state: String,
    pub previous_state: Option<String>,
    pub event_count: i64,
}

pub async fn fetch_asset(pool: &PgPool, asset_id: &str) -> Result<AssetRecord, RcError> {
    let row = sqlx::query(
        "SELECT asset_id, brand_id, current_state, previous_state FROM assets WHERE asset_id = $1",
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?;

    let current_state_raw: String = row.get("current_state");
    let previous_state_raw: Option<String> = row.get("previous_state");
    let previous_state = match previous_state_raw {
        Some(value) => Some(
            AssetState::from_db_str(&value)
                .ok_or_else(|| RcError::Database(format!("unknown previous_state: {value}")))?,
        ),
        None => None,
    };

    Ok(AssetRecord {
        asset_id: row.get("asset_id"),
        brand_id: row.get("brand_id"),
        current_state: AssetState::from_db_str(&current_state_raw)
            .ok_or_else(|| RcError::Database(format!("unknown current_state: {current_state_raw}")))?,
        previous_state,
    })
}

pub async fn fetch_verify_view(pool: &PgPool, asset_id: &str) -> Result<VerifyAssetView, RcError> {
    let row = sqlx::query(
        "SELECT a.asset_id, a.brand_id, a.product_id, a.uid, a.current_state, a.previous_state, COUNT(e.event_id) AS event_count FROM assets a LEFT JOIN asset_state_events e ON e.asset_id = a.asset_id WHERE a.asset_id = $1 GROUP BY a.asset_id, a.brand_id, a.product_id, a.uid, a.current_state, a.previous_state",
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?;

    Ok(VerifyAssetView {
        asset_id: row.get("asset_id"),
        brand_id: row.get("brand_id"),
        product_id: row.get("product_id"),
        uid: row.get("uid"),
        current_state: row.get("current_state"),
        previous_state: row.get("previous_state"),
        event_count: row.get("event_count"),
    })
}

pub async fn load_idempotency_record(pool: &PgPool, idempotency_key: &str) -> Result<Option<IdempotencyRecord>, RcError> {
    let row = sqlx::query(
        "SELECT request_hash, response_snapshot FROM idempotency_records WHERE idempotency_key = $1",
    )
    .bind(idempotency_key)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.map(|row| IdempotencyRecord {
        request_hash: row.get("request_hash"),
        response_snapshot: row.get("response_snapshot"),
    }))
}

pub async fn persist_action(
    pool: &PgPool,
    next_record: &AssetRecord,
    audit_event: &AuditEvent,
    request_hash: &str,
    response_snapshot: Value,
    asset_commitment_id: Option<&str>,
    redis: Option<redis::aio::MultiplexedConnection>,
) -> Result<(), RcError> {
    let mut tx = pool
        .begin()
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

    let old_owner: Option<String> = match audit_event.action {
        AssetAction::Transfer | AssetAction::Consume | AssetAction::Legacy => {
            sqlx::query_scalar("SELECT owner_id FROM assets WHERE asset_id = $1")
                .bind(&next_record.asset_id)
                .fetch_optional(&mut *tx)
                .await
                .map_err(|err| RcError::Database(err.to_string()))?
                .flatten()
        }
        _ => None,
    };

    sqlx::query(
        "UPDATE assets SET current_state = $2, previous_state = $3, updated_at = NOW() WHERE asset_id = $1",
    )
    .bind(&next_record.asset_id)
    .bind(next_record.current_state.as_db_str())
    .bind(next_record.previous_state.map(|state| state.as_db_str().to_string()))
    .execute(&mut *tx)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    match audit_event.action {
        AssetAction::LegalSell => {
            let buyer = audit_event.context.buyer_id.as_ref()
                .expect("LegalSell buyer_id validated in apply_action");
            sqlx::query(
                "UPDATE assets SET owner_id = $2 WHERE asset_id = $1",
            )
            .bind(&next_record.asset_id)
            .bind(buyer)
            .execute(&mut *tx)
            .await
            .map_err(|err| RcError::Database(err.to_string()))?;
        }
        AssetAction::Transfer => {
            sqlx::query(
                "UPDATE assets SET owner_id = $2 WHERE asset_id = $1",
            )
            .bind(&next_record.asset_id)
            .bind(&audit_event.context.actor_id)
            .execute(&mut *tx)
            .await
            .map_err(|err| RcError::Database(err.to_string()))?;
        }
        AssetAction::Consume | AssetAction::Legacy => {
            sqlx::query(
                "UPDATE assets SET owner_id = NULL WHERE asset_id = $1",
            )
            .bind(&next_record.asset_id)
            .execute(&mut *tx)
            .await
            .map_err(|err| RcError::Database(err.to_string()))?;
        }
        _ => {}
    }

    sqlx::query(
        "INSERT INTO asset_state_events (event_id, asset_id, action, from_state, to_state, trace_id, actor_id, actor_role, actor_org, idempotency_key, approval_id, policy_version, asset_commitment_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
    )
    .bind(audit_event.event_id)
    .bind(&audit_event.asset_id)
    .bind(audit_event.action.as_db_str())
    .bind(audit_event.from_state.map(|state| state.as_db_str().to_string()))
    .bind(audit_event.to_state.as_db_str())
    .bind(audit_event.context.trace_id)
    .bind(&audit_event.context.actor_id)
    .bind(audit_event.context.actor_role.as_db_str())
    .bind(&audit_event.context.actor_org)
    .bind(&audit_event.context.idempotency_key)
    .bind(&audit_event.context.approval_id)
    .bind(&audit_event.context.policy_version)
    .bind(asset_commitment_id)
    .execute(&mut *tx)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    sqlx::query(
        "INSERT INTO idempotency_records (idempotency_key, resource_type, resource_id, request_hash, response_snapshot) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (idempotency_key, resource_type) DO NOTHING",
    )
    .bind(&audit_event.context.idempotency_key)
    .bind("asset_action")
    .bind(&audit_event.asset_id)
    .bind(request_hash)
    .bind(response_snapshot)
    .execute(&mut *tx)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    tx.commit()
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

    let snapshot_params = match audit_event.action {
        AssetAction::LegalSell => Some((
            None,
            audit_event.context.buyer_id.clone(),
        )),
        AssetAction::Transfer => Some((
            old_owner.clone(),
            Some(audit_event.context.actor_id.clone()),
        )),
        AssetAction::Consume | AssetAction::Legacy => Some((
            old_owner.clone(),
            None,
        )),
        _ => None,
    };

    let _ = (snapshot_params, redis);

    Ok(())
}
