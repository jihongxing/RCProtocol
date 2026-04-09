use sqlx::PgPool;

/// 注入开发环境种子数据，等价于 deploy/postgres/init/002_seed.sql
///
/// 所有 INSERT 使用 ON CONFLICT DO NOTHING，保证幂等。
///
/// 注意：本文件保留的 `*-demo*` / `*-001` 风格值属于历史兼容的 legacy sample，
/// 不应再作为新增正式资源 ID 的模板。新增资源应统一使用 `prefix + ULID`。
/// 此模块独立于 db/ 模块，直接使用 sqlx raw SQL。
pub async fn run_seed(pool: &PgPool) {
    sqlx::query(
        "INSERT INTO brands (brand_id, brand_name) VALUES ($1, $2) ON CONFLICT DO NOTHING",
    )
    .bind("brand-demo")
    .bind("RC Demo Brand")
    .execute(pool)
    .await
    .expect("seed brand");

    sqlx::query(
        "INSERT INTO products (product_id, brand_id, product_name) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
    )
    .bind("product-demo-001")
    .bind("brand-demo")
    .bind("RC Demo Product")
    .execute(pool)
    .await
    .expect("seed product");

    sqlx::query(
        "INSERT INTO batches (batch_id, brand_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
    )
    .bind("batch-demo-001")
    .bind("brand-demo")
    .execute(pool)
    .await
    .expect("seed batch");

    sqlx::query(
        "INSERT INTO factory_sessions (session_id, batch_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
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
        sqlx::query(
            "INSERT INTO assets (asset_id, brand_id, product_id, uid, current_state, previous_state) \
             VALUES ($1, $2, $3, $4, $5, NULL) ON CONFLICT DO NOTHING",
        )
        .bind(asset_id)
        .bind("brand-demo")
        .bind("product-demo-001")
        .bind(uid)
        .bind(state)
        .execute(pool)
        .await
        .expect("seed asset");
    }
}
