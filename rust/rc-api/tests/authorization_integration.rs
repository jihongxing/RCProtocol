use rc_api::auth::authorization::{verify_authority, AuthorityProof};
use rc_common::ids;
use rc_kms::{KeyProvider, SoftwareKms};
use rc_test_helpers::{fixtures::{generate_test_brand_id, seed_brand}, TestDb};
use std::sync::Arc;

const TEST_ROOT_KEY_HEX: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
const TEST_SYSTEM_ID: &str = "test-system";

fn setup_kms() -> Arc<dyn KeyProvider + Send + Sync> {
    std::env::set_var("RC_ROOT_KEY_HEX", TEST_ROOT_KEY_HEX);
    std::env::set_var("RC_SYSTEM_ID", TEST_SYSTEM_ID);
    Arc::new(SoftwareKms::from_env().expect("test KMS init"))
}

#[tokio::test]
async fn test_virtual_authority_valid() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let test_brand_id = generate_test_brand_id();
    assert!(test_brand_id.starts_with("brand_"));

    let asset_id = ids::generate_asset_id();
    assert!(asset_id.starts_with("asset_"));
    let authority_uid = format!("vauth-{}", nanoid::nanoid!(12));
    let user_id = "user-001";

    seed_brand(&db, &test_brand_id, "Test Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&test_brand_id)
        .execute(&db).await.unwrap();

    let k_chip_mother = kms.derive_mother_key(&test_brand_id, authority_uid.as_bytes(), 0).unwrap();
    let hash_bytes = rc_crypto::hmac_sha256::compute(k_chip_mother.as_bytes(), authority_uid.as_bytes());
    let credential_hash = hex::encode(hash_bytes);

    let authority_id: uuid::Uuid = sqlx::query_scalar(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id, created_by) \
         VALUES ($1, 'VIRTUAL_APP', $2, 0, $3, $4, $4) \
         RETURNING authority_id"
    )
    .bind(&authority_uid)
    .bind(&test_brand_id)
    .bind(&credential_hash)
    .bind(user_id)
    .fetch_one(&db)
    .await
    .unwrap();

    sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) \
         VALUES ($1, '04A31B2C3D4E5F', $2, $3, 'Active', $4)"
    )
    .bind(&asset_id)
    .bind(authority_id)
    .bind(&authority_uid)
    .bind(user_id)
    .execute(&db)
    .await
    .unwrap();

    let proof = AuthorityProof::VirtualToken {
        user_id: user_id.to_string(),
        credential_token: credential_hash.clone(),
    };

    let result = verify_authority(&db, kms.clone(), &asset_id, proof).await.unwrap();
    assert!(result.authorized, "should be authorized");
    assert_eq!(result.authority_type, "VIRTUAL_APP");
    assert!(result.risk_flags.is_empty());

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_virtual_authority_wrong_user() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let test_brand_id = generate_test_brand_id();

    let asset_id = ids::generate_asset_id();
    let authority_uid = format!("vauth-{}", nanoid::nanoid!(12));
    let bound_user = "user-001";
    let wrong_user = "user-002";

    seed_brand(&db, &test_brand_id, "Test Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&test_brand_id)
        .execute(&db).await.unwrap();

    let k_chip_mother = kms.derive_mother_key(&test_brand_id, authority_uid.as_bytes(), 0).unwrap();
    let hash_bytes = rc_crypto::hmac_sha256::compute(k_chip_mother.as_bytes(), authority_uid.as_bytes());
    let credential_hash = hex::encode(hash_bytes);

    let authority_id: uuid::Uuid = sqlx::query_scalar(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id, created_by) \
         VALUES ($1, 'VIRTUAL_APP', $2, 0, $3, $4, $4) \
         RETURNING authority_id"
    )
    .bind(&authority_uid)
    .bind(&test_brand_id)
    .bind(&credential_hash)
    .bind(bound_user)
    .fetch_one(&db)
    .await
    .unwrap();

    sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) \
         VALUES ($1, '04A31B2C3D4E5F', $2, $3, 'Active', $4)"
    )
    .bind(&asset_id)
    .bind(authority_id)
    .bind(&authority_uid)
    .bind(bound_user)
    .execute(&db)
    .await
    .unwrap();

    let proof = AuthorityProof::VirtualToken {
        user_id: wrong_user.to_string(),
        credential_token: credential_hash,
    };

    let result = verify_authority(&db, kms, &asset_id, proof).await.unwrap();
    assert!(!result.authorized, "should not be authorized");
    assert!(result.risk_flags.contains(&"user_mismatch".to_string()));

    test_db.cleanup().await;
}

#[tokio::test]
async fn test_virtual_authority_wrong_credential() {
    let test_db = TestDb::new().await;
    let db = test_db.pool().clone();
    let kms = setup_kms();
    let test_brand_id = generate_test_brand_id();

    let asset_id = ids::generate_asset_id();
    let authority_uid = format!("vauth-{}", nanoid::nanoid!(12));
    let user_id = "user-001";

    seed_brand(&db, &test_brand_id, "Test Brand").await;
    sqlx::query("INSERT INTO assets (asset_id, brand_id, uid, current_state, epoch) VALUES ($1, $2, '04A31B2C3D4E5F', 'Activated', 0)")
        .bind(&asset_id)
        .bind(&test_brand_id)
        .execute(&db).await.unwrap();

    let k_chip_mother = kms.derive_mother_key(&test_brand_id, authority_uid.as_bytes(), 0).unwrap();
    let hash_bytes = rc_crypto::hmac_sha256::compute(k_chip_mother.as_bytes(), authority_uid.as_bytes());
    let credential_hash = hex::encode(hash_bytes);

    let authority_id: uuid::Uuid = sqlx::query_scalar(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, virtual_credential_hash, bound_user_id, created_by) \
         VALUES ($1, 'VIRTUAL_APP', $2, 0, $3, $4, $4) \
         RETURNING authority_id"
    )
    .bind(&authority_uid)
    .bind(&test_brand_id)
    .bind(&credential_hash)
    .bind(user_id)
    .fetch_one(&db)
    .await
    .unwrap();

    sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) \
         VALUES ($1, '04A31B2C3D4E5F', $2, $3, 'Active', $4)"
    )
    .bind(&asset_id)
    .bind(authority_id)
    .bind(&authority_uid)
    .bind(user_id)
    .execute(&db)
    .await
    .unwrap();

    let wrong_credential = "0000000000000000000000000000000000000000000000000000000000000000";
    let proof = AuthorityProof::VirtualToken {
        user_id: user_id.to_string(),
        credential_token: wrong_credential.to_string(),
    };

    let result = verify_authority(&db, kms, &asset_id, proof).await.unwrap();
    assert!(!result.authorized, "should not be authorized");
    assert!(result.risk_flags.contains(&"credential_mismatch".to_string()));

    test_db.cleanup().await;
}
