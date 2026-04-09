use chrono::Utc;
use rc_common::types::AssetAction;
use serde_json::json;
use sqlx::{PgPool, Row};
use uuid::Uuid;

/// Trigger webhook for asset state change events
pub async fn trigger_webhook(
    pool: &PgPool,
    brand_id: &str,
    event_type: &str,
    _asset_id: &Uuid,
    event_data: serde_json::Value,
) {
    // Query active webhook configs for this brand that subscribe to this event type
    let configs = match sqlx::query(
        r#"
        SELECT webhook_id, url, secret
        FROM webhook_configs
        WHERE brand_id = $1 AND status = 'Active' AND $2 = ANY(events)
        "#,
    )
    .bind(brand_id)
    .bind(event_type)
    .fetch_all(pool)
    .await
    {
        Ok(rows) => rows,
        Err(err) => {
            tracing::warn!("Failed to fetch webhook configs for brand {}: {}", brand_id, err);
            return;
        }
    };

    if configs.is_empty() {
        tracing::debug!("No active webhooks for brand {} event {}", brand_id, event_type);
        return;
    }

    // Create webhook deliveries for each configured webhook
    for config in configs {
        let webhook_id: Uuid = config.get("webhook_id");

        let payload = json!({
            "event_id": Uuid::new_v4().to_string(),
            "event_type": event_type,
            "timestamp": Utc::now().to_rfc3339(),
            "data": event_data,
        });

        let result = sqlx::query(
            r#"
            INSERT INTO webhook_deliveries (webhook_id, event_type, payload, status, attempts)
            VALUES ($1, $2, $3, 'Pending', 0)
            "#,
        )
        .bind(webhook_id)
        .bind(event_type)
        .bind(payload)
        .execute(pool)
        .await;

        if let Err(err) = result {
            tracing::error!(
                "Failed to create webhook delivery for webhook {}: {}",
                webhook_id,
                err
            );
        } else {
            tracing::info!(
                "Webhook delivery created for brand {} event {} webhook {}",
                brand_id,
                event_type,
                webhook_id
            );
        }
    }
}

/// Helper to determine event type from action
pub fn event_type_from_action(action: AssetAction) -> Option<&'static str> {
    match action {
        AssetAction::ActivateConfirm => Some("asset.activated"),
        AssetAction::LegalSell => Some("asset.sold"),
        AssetAction::Transfer => Some("asset.transferred"),
        AssetAction::Freeze => Some("asset.disputed"),
        AssetAction::Recover => Some("asset.recovered"),
        _ => None,
    }
}

/// Trigger webhook after asset action (call from persist_action)
pub async fn trigger_asset_webhook(
    pool: &PgPool,
    brand_id: &str,
    asset_id: &Uuid,
    action: AssetAction,
    from_state: &str,
    to_state: &str,
    actor_id: &str,
    owner_id: Option<&str>,
) {
    if let Some(event_type) = event_type_from_action(action) {
        let event_data = json!({
            "asset_id": asset_id.to_string(),
            "brand_id": brand_id,
            "action": action.as_db_str(),
            "from_state": from_state,
            "to_state": to_state,
            "actor_id": actor_id,
            "owner_id": owner_id,
            "occurred_at": Utc::now().to_rfc3339(),
        });

        // Spawn async task to avoid blocking the main flow
        let pool_clone = pool.clone();
        let brand_id_owned = brand_id.to_string();
        let event_type_owned = event_type.to_string();
        let asset_id_owned = *asset_id;

        tokio::spawn(async move {
            trigger_webhook(
                &pool_clone,
                &brand_id_owned,
                &event_type_owned,
                &asset_id_owned,
                event_data,
            )
            .await;
        });
    }
}
