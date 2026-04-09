use rc_test_helpers::TestDb;
use rc_test_helpers::fixtures::{
    generate_test_asset_id, generate_test_brand_id, generate_test_product_id,
};

#[tokio::test]
#[serial_test::serial]
async fn test_testdb_creates_schema() {
    let db = TestDb::new().await;
    sqlx::query("SELECT 1 FROM brands LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM products LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM batches LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM factory_sessions LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM assets LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM asset_state_events LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM idempotency_records LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM authority_devices LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM asset_entanglements LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT 1 FROM asset_transfers LIMIT 0").execute(db.pool()).await.unwrap();
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_testdb_unique_names() {
    let db1 = TestDb::new().await;
    let db2 = TestDb::new().await;
    assert_ne!(db1.db_name(), db2.db_name());
    db1.cleanup().await;
    db2.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_demo_data() {
    let db = TestDb::new().await;
    rc_test_helpers::fixtures::seed_demo_data(db.pool()).await;
    let (brand_count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM brands").fetch_one(db.pool()).await.unwrap();
    let (asset_count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM assets").fetch_one(db.pool()).await.unwrap();
    assert_eq!(brand_count, 1);
    assert_eq!(asset_count, 4);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_idempotent() {
    let db = TestDb::new().await;
    rc_test_helpers::fixtures::seed_demo_data(db.pool()).await;
    rc_test_helpers::fixtures::seed_demo_data(db.pool()).await;
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM assets").fetch_one(db.pool()).await.unwrap();
    assert_eq!(count, 4);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_composable() {
    let db = TestDb::new().await;
    let brand_id = generate_test_brand_id();
    let product_id = generate_test_product_id();
    let asset_id = generate_test_asset_id();
    assert!(brand_id.starts_with("brand_"));
    assert!(product_id.starts_with("product_"));
    assert!(asset_id.starts_with("asset_"));
    rc_test_helpers::fixtures::seed_brand(db.pool(), &brand_id, "B1").await;
    rc_test_helpers::fixtures::seed_product(db.pool(), &product_id, &brand_id, "P1").await;
    rc_test_helpers::fixtures::seed_asset(db.pool(), &rc_test_helpers::fixtures::SeedAssetParams {
        asset_id: asset_id.clone(), brand_id: brand_id.clone(), product_id: Some(product_id.clone()), uid: Some("uid1".into()), current_state: "PreMinted".into(), previous_state: None, external_product_id: None, external_product_name: None, owner_id: None,
    }).await;
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM assets").fetch_one(db.pool()).await.unwrap();
    assert_eq!(count, 1);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_testdb_cleanup() {
    let db = TestDb::new().await;
    let name = db.db_name().to_string();
    db.cleanup().await;
    let admin_url = std::env::var("TEST_DATABASE_URL").unwrap_or_else(|_| "postgres://rcprotocol:rcprotocol_dev@localhost:5432/postgres".to_string());
    let admin_pool = sqlx::postgres::PgPoolOptions::new().max_connections(1).connect(&admin_url).await.unwrap();
    let (exists,): (bool,) = sqlx::query_as("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)").bind(&name).fetch_one(&admin_pool).await.unwrap();
    assert!(!exists, "database {} should have been dropped", name);
    admin_pool.close().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_authority_device() {
    let db = TestDb::new().await;
    let brand_id = generate_test_brand_id();
    rc_test_helpers::fixtures::seed_brand(db.pool(), &brand_id, "Test Brand").await;
    let authority_id = rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
        authority_uid: "vau-test-001".into(), authority_type: "VIRTUAL_APP".into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: Some("test_hash".into()), bound_user_id: Some("user-001".into()), physical_chip_uid: None, created_by: Some("test".into()),
    }).await;
    let authority_id2 = rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
        authority_uid: "vau-test-001".into(), authority_type: "VIRTUAL_APP".into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: Some("test_hash".into()), bound_user_id: Some("user-001".into()), physical_chip_uid: None, created_by: Some("test".into()),
    }).await;
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM authority_devices").fetch_one(db.pool()).await.unwrap();
    assert_eq!(authority_id, authority_id2);
    assert_eq!(count, 1);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_entanglement() {
    let db = TestDb::new().await;
    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    rc_test_helpers::fixtures::seed_brand(db.pool(), &brand_id, "Test Brand").await;
    rc_test_helpers::fixtures::seed_asset(db.pool(), &rc_test_helpers::fixtures::SeedAssetParams {
        asset_id: asset_id.clone(), brand_id: brand_id.clone(), product_id: None, uid: Some("uid-test-001".into()), current_state: "Activated".into(), previous_state: None, external_product_id: None, external_product_name: None, owner_id: None,
    }).await;
    let authority_id = rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
        authority_uid: "vau-test-002".into(), authority_type: "VIRTUAL_APP".into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: Some("test_hash".into()), bound_user_id: Some("user-002".into()), physical_chip_uid: None, created_by: Some("test".into()),
    }).await;
    rc_test_helpers::fixtures::seed_entanglement(db.pool(), &rc_test_helpers::fixtures::SeedEntanglementParams {
        asset_id: asset_id.clone(), child_uid: "uid-test-001".into(), authority_id, authority_uid: "vau-test-002".into(), entanglement_state: "Active".into(), bound_by: "test".into(),
    }).await;
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM asset_entanglements").fetch_one(db.pool()).await.unwrap();
    assert_eq!(count, 1);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_seed_demo_data_includes_authority() {
    let db = TestDb::new().await;
    rc_test_helpers::fixtures::seed_demo_data(db.pool()).await;
    let (authority_count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM authority_devices").fetch_one(db.pool()).await.unwrap();
    let (entanglement_count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM asset_entanglements").fetch_one(db.pool()).await.unwrap();
    assert_eq!(authority_count, 1);
    assert_eq!(entanglement_count, 1);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_authority_type_constraint() {
    let db = TestDb::new().await;
    let brand_id = generate_test_brand_id();
    rc_test_helpers::fixtures::seed_brand(db.pool(), &brand_id, "Test Brand").await;
    for auth_type in ["PHYSICAL_NFC", "VIRTUAL_QR", "VIRTUAL_APP", "VIRTUAL_BIOMETRIC"] {
        let uid = format!("vau-{}", auth_type);
        rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
            authority_uid: uid, authority_type: auth_type.into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: None, bound_user_id: None, physical_chip_uid: None, created_by: None,
        }).await;
    }
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM authority_devices").fetch_one(db.pool()).await.unwrap();
    assert_eq!(count, 4);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_unique_active_entanglement() {
    let db = TestDb::new().await;
    let brand_id = generate_test_brand_id();
    let asset_id = generate_test_asset_id();
    rc_test_helpers::fixtures::seed_brand(db.pool(), &brand_id, "Test Brand").await;
    rc_test_helpers::fixtures::seed_asset(db.pool(), &rc_test_helpers::fixtures::SeedAssetParams {
        asset_id: asset_id.clone(), brand_id: brand_id.clone(), product_id: None, uid: Some("uid-test-002".into()), current_state: "Activated".into(), previous_state: None, external_product_id: None, external_product_name: None, owner_id: None,
    }).await;
    let authority_id1 = rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
        authority_uid: "vau-test-003".into(), authority_type: "VIRTUAL_APP".into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: None, bound_user_id: None, physical_chip_uid: None, created_by: None,
    }).await;
    let authority_id2 = rc_test_helpers::fixtures::seed_authority_device(db.pool(), &rc_test_helpers::fixtures::SeedAuthorityDeviceParams {
        authority_uid: "vau-test-004".into(), authority_type: "VIRTUAL_APP".into(), brand_id: brand_id.clone(), key_epoch: 0, virtual_credential_hash: None, bound_user_id: None, physical_chip_uid: None, created_by: None,
    }).await;
    rc_test_helpers::fixtures::seed_entanglement(db.pool(), &rc_test_helpers::fixtures::SeedEntanglementParams {
        asset_id: asset_id.clone(), child_uid: "uid-test-002".into(), authority_id: authority_id1, authority_uid: "vau-test-003".into(), entanglement_state: "Active".into(), bound_by: "test".into(),
    }).await;
    let result = sqlx::query("INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, entanglement_state, bound_by) VALUES ($1, $2, $3, $4, $5, $6)")
        .bind(&asset_id).bind("uid-test-002").bind(authority_id2).bind("vau-test-004").bind("Active").bind("test")
        .execute(db.pool()).await;
    assert!(result.is_err(), "Should fail due to unique Active constraint");
    rc_test_helpers::fixtures::seed_entanglement(db.pool(), &rc_test_helpers::fixtures::SeedEntanglementParams {
        asset_id, child_uid: "uid-test-002".into(), authority_id: authority_id2, authority_uid: "vau-test-004".into(), entanglement_state: "Suspended".into(), bound_by: "test".into(),
    }).await;
    let (count,): (i64,) = sqlx::query_as("SELECT COUNT(*) FROM asset_entanglements").fetch_one(db.pool()).await.unwrap();
    assert_eq!(count, 2);
    db.cleanup().await;
}

#[tokio::test]
#[serial_test::serial]
async fn test_runtime_critical_columns_exist_via_migrations_only() {
    let db = TestDb::new().await;
    sqlx::query("SELECT batch_name, factory_id, status, expected_count, actual_count, closed_at FROM batches LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT ctr, cmac_valid, client_ip FROM verification_events LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT from_owner_id, to_owner_id, transfer_type, idempotency_key, metadata FROM asset_transfers LIMIT 0").execute(db.pool()).await.unwrap();
    sqlx::query("SELECT key_id, brand_id, key_hash, key_prefix, status, revoked_at, last_used_at, expires_at, metadata FROM api_keys LIMIT 0").execute(db.pool()).await.unwrap();
    db.cleanup().await;
}
