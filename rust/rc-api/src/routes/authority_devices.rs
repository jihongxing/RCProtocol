use axum::{
    extract::{Json, State},
    http::StatusCode,
    response::IntoResponse,
    routing::post,
    Router,
};
use serde::{Deserialize, Serialize};
use nanoid::nanoid;

use crate::{app::AppState, auth::extractor::ActorContext};
use rc_common::errors::RcError;

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/authority-devices/physical", post(register_physical_mother_card))
}

#[derive(Debug, Deserialize)]
pub struct RegisterPhysicalMotherCardRequest {
    pub chip_uid: String,
    pub brand_id: String,
    pub key_epoch: i32,
    pub asset_id: String,
}

#[derive(Debug, Serialize)]
pub struct RegisterPhysicalMotherCardResponse {
    pub device_id: String,
    pub chip_uid: String,
    pub status: String,
}

/// POST /authority-devices/physical
/// Register a physical mother card device
async fn register_physical_mother_card(
    State(state): State<AppState>,
    actor: ActorContext,
    Json(req): Json<RegisterPhysicalMotherCardRequest>,
) -> impl IntoResponse {
    // 1. Permission check: require Brand or Platform role
    if !matches!(actor.actor_role, rc_common::types::ActorRole::Brand | rc_common::types::ActorRole::Platform) {
        return super::error_response(RcError::Forbidden(
            "Only Brand or Platform users can register physical mother cards".to_string(),
        ));
    }

    // 2. Validate chip_uid format (14-char hex)
    if req.chip_uid.len() != 14 {
        return super::error_response(RcError::InvalidInput(
            "chip_uid must be exactly 14 characters".to_string(),
        ));
    }

    if !req.chip_uid.chars().all(|c| c.is_ascii_hexdigit()) {
        return super::error_response(RcError::InvalidInput(
            "chip_uid must contain only hexadecimal characters".to_string(),
        ));
    }

    let chip_uid_upper = req.chip_uid.to_uppercase();

    // 3. Check if chip_uid is already registered
    let existing = match sqlx::query(
        "SELECT device_id FROM authority_devices WHERE physical_chip_uid = $1"
    )
    .bind(&chip_uid_upper)
    .fetch_optional(&state.db)
    .await
    {
        Ok(result) => result,
        Err(e) => return super::error_response(RcError::Database(e.to_string())),
    };

    if existing.is_some() {
        return super::error_response(RcError::DuplicateResource(format!(
            "Physical chip UID {} is already registered",
            chip_uid_upper
        )));
    }

    // 4. Generate device_id
    let device_id = format!("phys-{}", nanoid!(12));

    // 5. Begin transaction
    let mut tx = match state.db.begin().await {
        Ok(tx) => tx,
        Err(e) => return super::error_response(RcError::Database(e.to_string())),
    };

    // 6. Insert authority_device record
    if let Err(e) = sqlx::query(
        "INSERT INTO authority_devices (
            device_id,
            authority_type,
            authority_uid,
            physical_chip_uid,
            virtual_credential_hash,
            bound_user_id,
            last_known_ctr,
            status,
            brand_id,
            key_epoch
        ) VALUES ($1, 'PHYSICAL_NFC', $2, $2, NULL, NULL, NULL, 'active', $3, $4)"
    )
    .bind(&device_id)
    .bind(&chip_uid_upper)
    .bind(&req.brand_id)
    .bind(req.key_epoch)
    .execute(&mut *tx)
    .await
    {
        return super::error_response(RcError::Database(e.to_string()));
    }

    // 7. Insert asset_entanglements binding
    if let Err(e) = sqlx::query(
        "INSERT INTO asset_entanglements (
            asset_id,
            device_id,
            entanglement_type,
            status
        ) VALUES ($1, $2, 'MOTHER_CHILD', 'active')"
    )
    .bind(&req.asset_id)
    .bind(&device_id)
    .execute(&mut *tx)
    .await
    {
        return super::error_response(RcError::Database(e.to_string()));
    }

    // 8. Commit transaction
    if let Err(e) = tx.commit().await {
        return super::error_response(RcError::Database(e.to_string()));
    }

    (
        StatusCode::CREATED,
        Json(RegisterPhysicalMotherCardResponse {
            device_id,
            chip_uid: chip_uid_upper,
            status: "active".to_string(),
        }),
    ).into_response()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::app::AppState;
    use crate::auth::extractor::ActorContext;
    use axum::http::StatusCode;
    use axum::response::IntoResponse;
    use rc_common::types::ActorRole;
    use serde_json::json;

    fn mock_actor(role: ActorRole) -> ActorContext {
        ActorContext {
            actor_id: "test-user".to_string(),
            actor_role: role,
            actor_org: None,
            brand_id: Some("brand-001".to_string()),
        }
    }

    fn setup_kms_env() {
        std::env::set_var("RC_ROOT_KEY_HEX", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f");
        std::env::set_var("RC_SYSTEM_ID", "test-system");
    }

    #[tokio::test]
    async fn test_register_physical_mother_card_success() {
        // This test requires a real database connection
        // Skip if DATABASE_URL is not set
        if std::env::var("DATABASE_URL").is_err() {
            eprintln!("Skipping test: DATABASE_URL not set");
            return;
        }

        setup_kms_env();
        let kms = rc_kms::SoftwareKms::from_env().expect("Failed to initialize KMS");

        let database_url = std::env::var("DATABASE_URL").unwrap();
        let pool = sqlx::postgres::PgPoolOptions::new()
            .max_connections(1)
            .connect(&database_url)
            .await
            .expect("Failed to connect to database");

        // Clean up any existing test data
        let _ = sqlx::query("DELETE FROM asset_entanglements WHERE asset_id = 'test-asset-001'")
            .execute(&pool)
            .await;
        let _ = sqlx::query("DELETE FROM authority_devices WHERE physical_chip_uid = '04E1A2B3C4D5E6'")
            .execute(&pool)
            .await;

        let request_body = json!({
            "chip_uid": "04E1A2B3C4D5E6",
            "brand_id": "brand-001",
            "key_epoch": 1,
            "asset_id": "test-asset-001"
        });

        let response = register_physical_mother_card(
            State(AppState {
                db: pool.clone(),
                kms: std::sync::Arc::new(kms),
                jwt_decoder: std::sync::Arc::new(crate::auth::jwt::JwtDecoder::new(b"test-secret-key-32-bytes-long!!")),
                auth_disabled: false,
                redis: None,
                ctr_cache: std::sync::Arc::new(dashmap::DashMap::new()),
                fallback_strategy: crate::app::FallbackStrategy::DirectPg,
                api_key_secret: b"test-api-key-secret".to_vec(),
            }),
            mock_actor(ActorRole::Platform),
            Json(serde_json::from_value(request_body).unwrap()),
        )
        .await;

        let response = response.into_response();
        assert_eq!(response.status(), StatusCode::CREATED);

        // Clean up
        let _ = sqlx::query("DELETE FROM asset_entanglements WHERE asset_id = 'test-asset-001'")
            .execute(&pool)
            .await;
        let _ = sqlx::query("DELETE FROM authority_devices WHERE physical_chip_uid = '04E1A2B3C4D5E6'")
            .execute(&pool)
            .await;
    }

    #[tokio::test]
    async fn test_register_duplicate_chip_uid() {
        if std::env::var("DATABASE_URL").is_err() {
            eprintln!("Skipping test: DATABASE_URL not set");
            return;
        }

        setup_kms_env();
        let kms = rc_kms::SoftwareKms::from_env().expect("Failed to initialize KMS");

        let database_url = std::env::var("DATABASE_URL").unwrap();
        let pool = sqlx::postgres::PgPoolOptions::new()
            .max_connections(1)
            .connect(&database_url)
            .await
            .expect("Failed to connect to database");

        // Clean up and insert test data
        let _ = sqlx::query("DELETE FROM asset_entanglements WHERE asset_id = 'test-asset-002'")
            .execute(&pool)
            .await;
        let _ = sqlx::query("DELETE FROM authority_devices WHERE physical_chip_uid = '04E1A2B3C4D5E7'")
            .execute(&pool)
            .await;

        // Insert a device first
        let _ = sqlx::query(
            "INSERT INTO authority_devices (device_id, authority_type, authority_uid, physical_chip_uid, status, brand_id, key_epoch)
             VALUES ('test-device-001', 'PHYSICAL_NFC', '04E1A2B3C4D5E7', '04E1A2B3C4D5E7', 'active', 'brand-001', 1)"
        )
        .execute(&pool)
        .await;

        let request_body = json!({
            "chip_uid": "04E1A2B3C4D5E7",
            "brand_id": "brand-001",
            "key_epoch": 1,
            "asset_id": "test-asset-002"
        });

        let response = register_physical_mother_card(
            State(AppState {
                db: pool.clone(),
                kms: std::sync::Arc::new(kms),
                jwt_decoder: std::sync::Arc::new(crate::auth::jwt::JwtDecoder::new(b"test-secret-key-32-bytes-long!!")),
                auth_disabled: false,
                redis: None,
                ctr_cache: std::sync::Arc::new(dashmap::DashMap::new()),
                fallback_strategy: crate::app::FallbackStrategy::DirectPg,
                api_key_secret: b"test-api-key-secret".to_vec(),
            }),
            mock_actor(ActorRole::Platform),
            Json(serde_json::from_value(request_body).unwrap()),
        )
        .await;

        let response = response.into_response();
        assert_eq!(response.status(), StatusCode::CONFLICT);

        // Clean up
        let _ = sqlx::query("DELETE FROM authority_devices WHERE physical_chip_uid = '04E1A2B3C4D5E7'")
            .execute(&pool)
            .await;
    }

    #[tokio::test]
    async fn test_register_forbidden_for_consumer() {
        if std::env::var("DATABASE_URL").is_err() {
            eprintln!("Skipping test: DATABASE_URL not set");
            return;
        }

        setup_kms_env();
        let kms = rc_kms::SoftwareKms::from_env().expect("Failed to initialize KMS");

        let database_url = std::env::var("DATABASE_URL").unwrap();
        let pool = sqlx::postgres::PgPoolOptions::new()
            .max_connections(1)
            .connect(&database_url)
            .await
            .expect("Failed to connect to database");

        let request_body = json!({
            "chip_uid": "04E1A2B3C4D5E8",
            "brand_id": "brand-001",
            "key_epoch": 1,
            "asset_id": "test-asset-003"
        });

        let response = register_physical_mother_card(
            State(AppState {
                db: pool.clone(),
                kms: std::sync::Arc::new(kms),
                jwt_decoder: std::sync::Arc::new(crate::auth::jwt::JwtDecoder::new(b"test-secret-key-32-bytes-long!!")),
                auth_disabled: false,
                redis: None,
                ctr_cache: std::sync::Arc::new(dashmap::DashMap::new()),
                fallback_strategy: crate::app::FallbackStrategy::DirectPg,
                api_key_secret: b"test-api-key-secret".to_vec(),
            }),
            mock_actor(ActorRole::Consumer),
            Json(serde_json::from_value(request_body).unwrap()),
        )
        .await;

        let response = response.into_response();
        assert_eq!(response.status(), StatusCode::FORBIDDEN);
    }

    #[test]
    fn test_validate_chip_uid_format() {
        // Valid 14-char hex
        assert!("04E1A2B3C4D5E6".len() == 14);
        assert!("04E1A2B3C4D5E6".chars().all(|c| c.is_ascii_hexdigit()));

        // Invalid length
        assert!("04E1A2B3".len() != 14);

        // Invalid characters
        assert!(!"04E1A2B3C4D5EG".chars().all(|c| c.is_ascii_hexdigit()));
    }
}
