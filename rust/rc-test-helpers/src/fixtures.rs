use rc_common::ids;
use sqlx::{PgPool, Row};
use uuid::Uuid;

pub struct SeedAssetParams {
    pub asset_id: String,
    pub brand_id: String,
    pub product_id: Option<String>,
    pub uid: Option<String>,
    pub current_state: String,
    pub previous_state: Option<String>,
    pub external_product_id: Option<String>,
    pub external_product_name: Option<String>,
    pub owner_id: Option<String>,
}

pub struct SeedAuthorityDeviceParams {
    pub authority_uid: String,
    pub authority_type: String,
    pub brand_id: String,
    pub key_epoch: i32,
    pub virtual_credential_hash: Option<String>,
    pub bound_user_id: Option<String>,
    pub physical_chip_uid: Option<String>,
    pub created_by: Option<String>,
}

pub struct SeedEntanglementParams {
    pub asset_id: String,
    pub child_uid: String,
    pub authority_id: Uuid,
    pub authority_uid: String,
    pub entanglement_state: String,
    pub bound_by: String,
}

pub async fn seed_brand(pool: &PgPool, brand_id: &str, brand_name: &str) {
    seed_brand_with_api_key(pool, brand_id, brand_name, None).await;
}

pub async fn seed_brand_with_api_key(
    pool: &PgPool,
    brand_id: &str,
    brand_name: &str,
    api_key_hash: Option<&str>,
) {
    let contact_email = format!("{}@test.local", brand_id.replace('_', "-"));
    sqlx::query(
        "INSERT INTO brands (brand_id, brand_name, contact_email, industry, api_key_hash, status) \
         VALUES ($1, $2, $3, 'Other', $4, 'Active') ON CONFLICT DO NOTHING"
    )
    .bind(brand_id)
    .bind(brand_name)
    .bind(contact_email)
    .bind(api_key_hash)
    .execute(pool)
    .await
    .expect("seed brand");
}

pub async fn seed_product(pool: &PgPool, product_id: &str, brand_id: &str, name: &str) {
    sqlx::query(
        "INSERT INTO products (product_id, brand_id, product_name) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING"
    )
    .bind(product_id)
    .bind(brand_id)
    .bind(name)
    .execute(pool)
    .await
    .expect("seed product");
}

pub async fn seed_asset(pool: &PgPool, params: &SeedAssetParams) {
    sqlx::query(
        "INSERT INTO assets (asset_id, brand_id, product_id, uid, current_state, previous_state, \
         external_product_id, external_product_name, owner_id) \
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT DO NOTHING"
    )
    .bind(&params.asset_id)
    .bind(&params.brand_id)
    .bind(&params.product_id)
    .bind(&params.uid)
    .bind(&params.current_state)
    .bind(&params.previous_state)
    .bind(&params.external_product_id)
    .bind(&params.external_product_name)
    .bind(&params.owner_id)
    .execute(pool)
    .await
    .expect("seed asset");
}

pub async fn seed_authority_device(pool: &PgPool, params: &SeedAuthorityDeviceParams) -> Uuid {
    let row = sqlx::query(
        "INSERT INTO authority_devices (authority_uid, authority_type, brand_id, key_epoch, \
         virtual_credential_hash, bound_user_id, physical_chip_uid, created_by) \
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8) \
         ON CONFLICT (authority_uid) DO NOTHING \
         RETURNING authority_id"
    )
    .bind(&params.authority_uid)
    .bind(&params.authority_type)
    .bind(&params.brand_id)
    .bind(params.key_epoch)
    .bind(&params.virtual_credential_hash)
    .bind(&params.bound_user_id)
    .bind(&params.physical_chip_uid)
    .bind(&params.created_by)
    .fetch_optional(pool)
    .await
    .expect("seed authority device");

    if let Some(row) = row {
        row.get("authority_id")
    } else {
        sqlx::query("SELECT authority_id FROM authority_devices WHERE authority_uid = $1")
            .bind(&params.authority_uid)
            .fetch_one(pool)
            .await
            .expect("fetch existing authority device")
            .get("authority_id")
    }
}

pub async fn seed_entanglement(pool: &PgPool, params: &SeedEntanglementParams) {
    sqlx::query(
        "INSERT INTO asset_entanglements (asset_id, child_uid, authority_id, authority_uid, \
         entanglement_state, bound_by) \
         VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING"
    )
    .bind(&params.asset_id)
    .bind(&params.child_uid)
    .bind(params.authority_id)
    .bind(&params.authority_uid)
    .bind(&params.entanglement_state)
    .bind(&params.bound_by)
    .execute(pool)
    .await
    .expect("seed entanglement");
}

/// 注入与 deploy/postgres/init/002_seed.sql 完全等价的确定性数据
///
/// 注意：`seed_demo_data` 属于历史兼容样例保留；新增测试数据应优先改用统一 ID 生成函数。
pub async fn seed_demo_data(pool: &PgPool) {
    seed_brand(pool, "brand-demo", "RC Demo Brand").await;
    seed_product(pool, "product-demo-001", "brand-demo", "RC Demo Product").await;

    sqlx::query(
        "INSERT INTO batches (batch_id, brand_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
    )
    .bind("batch-demo-001")
    .bind("brand-demo")
    .execute(pool)
    .await
    .expect("seed batch");

    sqlx::query(
        "INSERT INTO factory_sessions (session_id, batch_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
    )
    .bind("session-demo-001")
    .bind("batch-demo-001")
    .execute(pool)
    .await
    .expect("seed session");

    for (asset_id, uid, state) in [
        ("asset-main-001", "UID-DEMO-0001", "PreMinted"),
        ("asset-freeze-001", "UID-DEMO-0002", "Activated"),
        ("asset-transfer-001", "UID-DEMO-0003", "LegallySold"),
        ("asset-terminal-001", "UID-DEMO-0004", "Transferred"),
    ] {
        seed_asset(pool, &SeedAssetParams {
            asset_id: asset_id.into(),
            brand_id: "brand-demo".into(),
            product_id: Some("product-demo-001".into()),
            uid: Some(uid.into()),
            current_state: state.into(),
            previous_state: None,
            external_product_id: None,
            external_product_name: None,
            owner_id: None,
        }).await;
    }

    let authority_id = seed_authority_device(pool, &SeedAuthorityDeviceParams {
        authority_uid: "vau-demo-001".into(),
        authority_type: "VIRTUAL_APP".into(),
        brand_id: "brand-demo".into(),
        key_epoch: 0,
        virtual_credential_hash: Some("demo_hash_placeholder".into()),
        bound_user_id: Some("user-demo-001".into()),
        physical_chip_uid: None,
        created_by: Some("system".into()),
    }).await;

    seed_entanglement(pool, &SeedEntanglementParams {
        asset_id: "asset-freeze-001".into(),
        child_uid: "UID-DEMO-0002".into(),
        authority_id,
        authority_uid: "vau-demo-001".into(),
        entanglement_state: "Active".into(),
        bound_by: "system".into(),
    }).await;
}

pub fn generate_test_brand_id() -> String {
    ids::generate_brand_id()
}

pub fn generate_test_product_id() -> String {
    ids::generate_product_id()
}

pub fn generate_test_asset_id() -> String {
    ids::generate_asset_id()
}

pub fn generate_test_batch_id() -> String {
    ids::generate_batch_id()
}
