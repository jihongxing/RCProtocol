//! DEPRECATED: Product 管理已废弃，不再接受新的写入。
//!
//! 保留 `fetch_product`、`list_products`、`fetch_products_batch` 只读函数，兼容旧数据查询。

use chrono::{DateTime, Utc};
use rc_common::errors::RcError;
use serde::Serialize;
use sqlx::{PgPool, Row};

use crate::routes::brand::PaginationParams;

/// Read-only product response for backward-compatible queries.
#[derive(Debug, Serialize)]
pub struct ProductResponse {
    pub product_id: String,
    pub brand_id: String,
    pub product_name: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

pub async fn fetch_product(
    pool: &PgPool,
    brand_id: &str,
    product_id: &str,
) -> Result<ProductResponse, RcError> {
    let row = sqlx::query(
        "SELECT product_id, brand_id, product_name, created_at, updated_at \
         FROM products WHERE product_id = $1 AND brand_id = $2",
    )
    .bind(product_id)
    .bind(brand_id)
    .fetch_optional(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?
    .ok_or(RcError::ProductNotFound)?;

    Ok(ProductResponse {
        product_id: row.get::<String, _>("product_id"),
        brand_id: row.get::<String, _>("brand_id"),
        product_name: row.get::<String, _>("product_name"),
        created_at: row.get::<DateTime<Utc>, _>("created_at"),
        updated_at: row.get::<DateTime<Utc>, _>("updated_at"),
    })
}

pub async fn list_products(
    pool: &PgPool,
    brand_id: &str,
    params: &PaginationParams,
) -> Result<(Vec<ProductResponse>, i64), RcError> {
    let page_size = params.page_size();
    let offset = params.offset();

    let rows = sqlx::query(
        "SELECT product_id, brand_id, product_name, created_at, updated_at \
         FROM products WHERE brand_id = $1 \
         ORDER BY created_at DESC LIMIT $2 OFFSET $3",
    )
    .bind(brand_id)
    .bind(page_size)
    .bind(offset)
    .fetch_all(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    let count_row = sqlx::query(
        "SELECT COUNT(*) as total FROM products WHERE brand_id = $1",
    )
    .bind(brand_id)
    .fetch_one(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    let total: i64 = count_row.get::<i64, _>("total");

    let items = rows
        .into_iter()
        .map(|row| ProductResponse {
            product_id: row.get::<String, _>("product_id"),
            brand_id: row.get::<String, _>("brand_id"),
            product_name: row.get::<String, _>("product_name"),
            created_at: row.get::<DateTime<Utc>, _>("created_at"),
            updated_at: row.get::<DateTime<Utc>, _>("updated_at"),
        })
        .collect();

    Ok((items, total))
}

/// M12: 批量查询产品信息，消除 BFF N+1 调用
pub async fn fetch_products_batch(
    pool: &PgPool,
    product_ids: &[String],
) -> Result<Vec<ProductResponse>, RcError> {
    if product_ids.is_empty() {
        return Ok(vec![]);
    }
    let rows = sqlx::query(
        "SELECT product_id, brand_id, product_name, created_at, updated_at \
         FROM products WHERE product_id = ANY($1)",
    )
    .bind(product_ids)
    .fetch_all(pool)
    .await
    .map_err(|err| RcError::Database(err.to_string()))?;

    Ok(rows
        .into_iter()
        .map(|row| ProductResponse {
            product_id: row.get::<String, _>("product_id"),
            brand_id: row.get::<String, _>("brand_id"),
            product_name: row.get::<String, _>("product_name"),
            created_at: row.get::<DateTime<Utc>, _>("created_at"),
            updated_at: row.get::<DateTime<Utc>, _>("updated_at"),
        })
        .collect())
}
