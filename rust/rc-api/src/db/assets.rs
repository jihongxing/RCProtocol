use chrono::{DateTime, Utc};
use rc_common::errors::RcError;
use serde::Serialize;
use sqlx::{PgPool, Row};

pub struct AssetVerifyRow {
    pub asset_id: String,
    pub brand_id: String,
    pub product_id: Option<String>,
    pub uid: String,
    pub current_state: String,
    pub last_verified_ctr: Option<i32>,
    pub epoch: i32,
    pub asset_commitment_id: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct VirtualMotherCardDetail {
    pub authority_uid: String,
    pub authority_type: String,
    pub credential_hash: Option<String>,
    pub epoch: i32,
}

#[derive(Debug, Serialize)]
pub struct AssetDetail {
    pub asset_id: String,
    pub uid: String,
    pub brand_id: String,
    pub batch_id: Option<String>,
    pub external_product_id: Option<String>,
    pub external_product_name: Option<String>,
    pub external_product_url: Option<String>,
    pub asset_commitment_id: Option<String>,
    pub brand_attestation_status: Option<String>,
    pub platform_attestation_status: Option<String>,
    pub current_state: String,
    pub previous_state: Option<String>,
    pub owner_id: Option<String>,
    pub key_epoch: i32,
    pub activated_at: Option<DateTime<Utc>>,
    pub sold_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub virtual_mother_card: Option<VirtualMotherCardDetail>,
}

#[derive(Debug, Serialize)]
pub struct AssetStateEvent {
    pub event_id: String,
    pub asset_id: String,
    pub action: String,
    pub from_state: Option<String>,
    pub to_state: String,
    pub actor_id: String,
    pub actor_role: String,
    pub trace_id: Option<String>,
    pub occurred_at: DateTime<Utc>,
}

fn attestation_status_from_count(count: i64) -> Option<String> {
    if count > 0 {
        Some("issued".to_string())
    } else {
        None
    }
}

pub async fn fetch_asset_by_uid(pool: &PgPool, uid: &str) -> Result<AssetVerifyRow, RcError> {
    let row = sqlx::query(
        "SELECT asset_id, brand_id, product_id, uid, current_state, last_verified_ctr, epoch, asset_commitment_id FROM assets WHERE uid = $1",
    )
    .bind(uid)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?;

    Ok(AssetVerifyRow {
        asset_id: row.get("asset_id"),
        brand_id: row.get("brand_id"),
        product_id: row.get("product_id"),
        uid: row.get("uid"),
        current_state: row.get("current_state"),
        last_verified_ctr: row.get("last_verified_ctr"),
        epoch: row.get("epoch"),
        asset_commitment_id: row.get("asset_commitment_id"),
    })
}

pub async fn update_asset_ctr(pool: &PgPool, asset_id: &str, ctr: i32) -> Result<(), RcError> {
    sqlx::query("UPDATE assets SET last_verified_ctr = $2, updated_at = NOW() WHERE asset_id = $1")
        .bind(asset_id)
        .bind(ctr)
        .execute(pool)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(())
}

async fn fetch_virtual_mother_card_detail(
    pool: &PgPool,
    asset_id: &str,
) -> Result<Option<VirtualMotherCardDetail>, RcError> {
    let row = sqlx::query(
        r#"
        SELECT ad.authority_uid, ad.authority_type, ad.virtual_credential_hash, ad.key_epoch
        FROM asset_entanglements ae
        INNER JOIN authority_devices ad ON ad.authority_id = ae.authority_id
        WHERE ae.asset_id = $1
          AND ae.entanglement_state = 'Active'
          AND ad.authority_type = 'VIRTUAL_APP'
        ORDER BY ae.created_at DESC
        LIMIT 1
        "#,
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(row.map(|row| VirtualMotherCardDetail {
        authority_uid: row.get("authority_uid"),
        authority_type: row.get("authority_type"),
        credential_hash: row.get("virtual_credential_hash"),
        epoch: row.get("key_epoch"),
    }))
}

/// Fetch asset detail by asset_id
pub async fn fetch_asset_detail(pool: &PgPool, asset_id: &str) -> Result<AssetDetail, RcError> {
    let row = sqlx::query(
        r#"
        SELECT asset_id, uid, brand_id, batch_id, external_product_id, external_product_name,
               external_product_url, asset_commitment_id, current_state, previous_state, owner_id, key_epoch,
               activated_at, sold_at, created_at, updated_at
        FROM assets
        WHERE asset_id = $1
        "#,
    )
    .bind(asset_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::AssetNotFound)?;

    let commitment_id: Option<String> = row.get("asset_commitment_id");
    let (brand_attestation_status, platform_attestation_status) = if let Some(commitment_id) = commitment_id.as_deref() {
        let brand_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM brand_attestations WHERE asset_commitment_id = $1",
        )
        .bind(commitment_id)
        .fetch_one(pool)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

        let platform_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM platform_attestations WHERE asset_commitment_id = $1",
        )
        .bind(commitment_id)
        .fetch_one(pool)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

        (
            attestation_status_from_count(brand_count),
            attestation_status_from_count(platform_count),
        )
    } else {
        (None, None)
    };

    let virtual_mother_card = fetch_virtual_mother_card_detail(pool, asset_id).await?;

    Ok(AssetDetail {
        asset_id: row.get("asset_id"),
        uid: row.get("uid"),
        brand_id: row.get("brand_id"),
        batch_id: row.get("batch_id"),
        external_product_id: row.get("external_product_id"),
        external_product_name: row.get("external_product_name"),
        external_product_url: row.get("external_product_url"),
        asset_commitment_id: commitment_id,
        brand_attestation_status,
        platform_attestation_status,
        current_state: row.get("current_state"),
        previous_state: row.get("previous_state"),
        owner_id: row.get("owner_id"),
        key_epoch: row.get("key_epoch"),
        activated_at: row.get("activated_at"),
        sold_at: row.get("sold_at"),
        created_at: row.get("created_at"),
        updated_at: row.get("updated_at"),
        virtual_mother_card,
    })
}

/// Fetch asset state events (audit history)
pub async fn fetch_asset_history(
    pool: &PgPool,
    asset_id: &str,
) -> Result<Vec<AssetStateEvent>, RcError> {
    let rows = sqlx::query(
        r#"
        SELECT event_id, asset_id, action, from_state, to_state, actor_id, actor_role, trace_id, occurred_at
        FROM asset_state_events
        WHERE asset_id = $1
        ORDER BY occurred_at ASC
        "#,
    )
    .bind(asset_id)
    .fetch_all(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    let events = rows
        .into_iter()
        .map(|row| {
            let event_id: uuid::Uuid = row.get("event_id");
            let trace_id: Option<uuid::Uuid> = row.get("trace_id");
            AssetStateEvent {
                event_id: event_id.to_string(),
                asset_id: row.get("asset_id"),
                action: row.get("action"),
                from_state: row.get("from_state"),
                to_state: row.get("to_state"),
                actor_id: row.get("actor_id"),
                actor_role: row.get("actor_role"),
                trace_id: trace_id.map(|id| id.to_string()),
                occurred_at: row.get("occurred_at"),
            }
        })
        .collect();

    Ok(events)
}

/// List assets with filters and pagination
pub async fn list_assets(
    pool: &PgPool,
    brand_id: Option<&str>,
    batch_id: Option<&str>,
    current_state: Option<&str>,
    limit: i64,
    offset: i64,
) -> Result<(Vec<AssetDetail>, i64), RcError> {
    let mut query = String::from("SELECT COUNT(*) FROM assets WHERE 1=1");
    let mut condition_strings = Vec::new();

    if brand_id.is_some() {
        condition_strings.push("brand_id = $1".to_string());
    }
    if batch_id.is_some() {
        let idx = condition_strings.len() + 1;
        condition_strings.push(format!("batch_id = ${}", idx));
    }
    if current_state.is_some() {
        let idx = condition_strings.len() + 1;
        condition_strings.push(format!("current_state = ${}", idx));
    }

    for condition in &condition_strings {
        query.push_str(" AND ");
        query.push_str(condition);
    }

    let mut count_query = sqlx::query_scalar::<_, i64>(&query);
    if let Some(brand) = brand_id {
        count_query = count_query.bind(brand);
    }
    if let Some(batch) = batch_id {
        count_query = count_query.bind(batch);
    }
    if let Some(state) = current_state {
        count_query = count_query.bind(state);
    }

    let total = count_query
        .fetch_one(pool)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

    let mut select_query = String::from(
        r#"
        SELECT asset_id, uid, brand_id, batch_id, external_product_id, external_product_name,
               external_product_url, asset_commitment_id, current_state, previous_state, owner_id, key_epoch,
               activated_at, sold_at, created_at, updated_at
        FROM assets WHERE 1=1
        "#
    );

    for condition in &condition_strings {
        select_query.push_str(" AND ");
        select_query.push_str(condition);
    }

    select_query.push_str(" ORDER BY created_at DESC LIMIT $");
    select_query.push_str(&(condition_strings.len() + 1).to_string());
    select_query.push_str(" OFFSET $");
    select_query.push_str(&(condition_strings.len() + 2).to_string());

    let mut fetch_query = sqlx::query(&select_query);
    if let Some(brand) = brand_id {
        fetch_query = fetch_query.bind(brand);
    }
    if let Some(batch) = batch_id {
        fetch_query = fetch_query.bind(batch);
    }
    if let Some(state) = current_state {
        fetch_query = fetch_query.bind(state);
    }
    fetch_query = fetch_query.bind(limit).bind(offset);

    let rows = fetch_query
        .fetch_all(pool)
        .await
        .map_err(|err| RcError::Database(err.to_string()))?;

    let mut items = Vec::with_capacity(rows.len());
    for row in rows {
        let asset_id: String = row.get("asset_id");
        let commitment_id: Option<String> = row.get("asset_commitment_id");
        let (brand_attestation_status, platform_attestation_status) = if let Some(commitment_id) = commitment_id.as_deref() {
            let brand_count: i64 = sqlx::query_scalar(
                "SELECT COUNT(*) FROM brand_attestations WHERE asset_commitment_id = $1",
            )
            .bind(commitment_id)
            .fetch_one(pool)
            .await
            .map_err(|err| RcError::Database(err.to_string()))?;

            let platform_count: i64 = sqlx::query_scalar(
                "SELECT COUNT(*) FROM platform_attestations WHERE asset_commitment_id = $1",
            )
            .bind(commitment_id)
            .fetch_one(pool)
            .await
            .map_err(|err| RcError::Database(err.to_string()))?;

            (
                attestation_status_from_count(brand_count),
                attestation_status_from_count(platform_count),
            )
        } else {
            (None, None)
        };

        let virtual_mother_card = fetch_virtual_mother_card_detail(pool, &asset_id).await?;
        items.push(AssetDetail {
            asset_id,
            uid: row.get("uid"),
            brand_id: row.get("brand_id"),
            batch_id: row.get("batch_id"),
            external_product_id: row.get("external_product_id"),
            external_product_name: row.get("external_product_name"),
            external_product_url: row.get("external_product_url"),
            asset_commitment_id: commitment_id,
            brand_attestation_status,
            platform_attestation_status,
            current_state: row.get("current_state"),
            previous_state: row.get("previous_state"),
            owner_id: row.get("owner_id"),
            key_epoch: row.get("key_epoch"),
            activated_at: row.get("activated_at"),
            sold_at: row.get("sold_at"),
            created_at: row.get("created_at"),
            updated_at: row.get("updated_at"),
            virtual_mother_card,
        });
    }

    Ok((items, total))
}
