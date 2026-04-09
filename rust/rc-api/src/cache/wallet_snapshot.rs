use redis::AsyncCommands;
use sqlx::PgPool;

/// 钱包快照——资产状态变更后异步更新 Redis 中的持有者视图，
/// 让前端查询 wallet 列表时走 Redis 而非每次穿透 PG。
///
/// Redis 数据结构：
///   Hash  `wallet:{user_id}`         → asset_count, updated_at
///   ZSet  `wallet:{user_id}:assets`  → member=asset_id, score=created_at timestamp
#[derive(Clone)]
pub struct WalletSnapshot {
    redis: Option<redis::aio::MultiplexedConnection>,
    pub(crate) db: PgPool,
}

impl WalletSnapshot {
    pub fn new(
        redis: Option<redis::aio::MultiplexedConnection>,
        db: PgPool,
    ) -> Self {
        Self { redis, db }
    }

    /// 资产状态变更后更新钱包快照——从旧持有者移除、向新持有者添加。
    /// best-effort：Redis 不可用时仅记录 warn，不影响主流程。
    pub async fn on_asset_state_change(
        &self,
        asset_id: &str,
        old_owner: Option<&str>,
        new_owner: Option<&str>,
        created_at: i64,
    ) {
        if let Some(old) = old_owner {
            self.remove_asset(old, asset_id).await;
        }
        if let Some(new) = new_owner {
            self.add_asset(new, asset_id, created_at).await;
        }
    }

    /// 将资产加入持有者的 Sorted Set 并更新 Hash 计数
    async fn add_asset(&self, user_id: &str, asset_id: &str, score: i64) {
        let Some(mut conn) = self.redis.clone() else { return };

        let set_key = format!("wallet:{user_id}:assets");
        let hash_key = format!("wallet:{user_id}");

        if let Err(e) = conn.zadd::<_, _, _, ()>(&set_key, asset_id, score).await {
            tracing::warn!(user_id = user_id, asset_id = asset_id, error = %e, "wallet snapshot ZADD failed");
            return;
        }
        if let Err(e) = conn.hincr::<_, _, _, ()>(&hash_key, "asset_count", 1i64).await {
            tracing::warn!(user_id = user_id, error = %e, "wallet snapshot HINCRBY asset_count failed");
        }
        if let Err(e) = conn
            .hset::<_, _, _, ()>(&hash_key, "updated_at", chrono::Utc::now().to_rfc3339())
            .await
        {
            tracing::warn!(user_id = user_id, error = %e, "wallet snapshot HSET updated_at failed");
        }
    }

    /// 从持有者的 Sorted Set 移除资产并递减 Hash 计数
    async fn remove_asset(&self, user_id: &str, asset_id: &str) {
        let Some(mut conn) = self.redis.clone() else { return };

        let set_key = format!("wallet:{user_id}:assets");
        let hash_key = format!("wallet:{user_id}");

        if let Err(e) = conn.zrem::<_, _, ()>(&set_key, asset_id).await {
            tracing::warn!(user_id = user_id, asset_id = asset_id, error = %e, "wallet snapshot ZREM failed");
            return;
        }
        if let Err(e) = conn.hincr::<_, _, _, ()>(&hash_key, "asset_count", -1i64).await {
            tracing::warn!(user_id = user_id, error = %e, "wallet snapshot HINCRBY asset_count -1 failed");
        }
    }

    /// 分页查询钱包资产列表（Read-Through）：
    /// 1. 优先从 Redis Sorted Set 读取（ZREVRANGE 分页，按 created_at 降序）
    /// 2. Redis 命中 → 直接返回
    /// 3. Redis 未命中（set 不存在 or 页内无数据）→ 从 PG 查询
    /// 4. PG 结果直接返回给调用者，异步 tokio::spawn 回填 Redis
    ///
    /// page 为 0-indexed，page_size > 0
    pub async fn get_assets(
        &self,
        user_id: &str,
        page: i64,
        page_size: i64,
    ) -> Vec<String> {
        // ── 尝试 Redis ──
        if let Some(mut conn) = self.redis.clone() {
            let set_key = format!("wallet:{user_id}:assets");

            // 先用 ZCARD 判断 set 是否存在——空 set 视为未命中
            let card: i64 = conn.zcard(&set_key).await.unwrap_or(0);

            if card > 0 {
                let start = page * page_size;
                let stop = start + page_size - 1;

                match conn
                    .zrevrange::<_, Vec<String>>(&set_key, start as isize, stop as isize)
                    .await
                {
                    Ok(result) => {
                        // 即使当前页���出范围返回空 vec，只要 set 存在就算命中
                        return result;
                    }
                    Err(e) => {
                        // Redis 读取异常——降级到 PG
                        tracing::warn!(
                            user_id = user_id,
                            error = %e,
                            "wallet snapshot ZREVRANGE failed, falling back to PG"
                        );
                    }
                }
            }
            // card == 0 表示 set 不存在——穿透到 PG
        }

        // ── 穿透到 PG ──
        self.fetch_assets_from_pg(user_id, page, page_size).await
    }

    /// 从 PG 查询资产列表并异步回填 Redis
    async fn fetch_assets_from_pg(
        &self,
        user_id: &str,
        page: i64,
        page_size: i64,
    ) -> Vec<String> {
        let offset = page * page_size;

        let rows: Vec<String> = match sqlx::query_scalar::<_, String>(
            "SELECT asset_id FROM assets WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
        )
        .bind(user_id)
        .bind(page_size)
        .bind(offset)
        .fetch_all(&self.db)
        .await
        {
            Ok(r) => r,
            Err(e) => {
                tracing::warn!(user_id = user_id, error = %e, "wallet snapshot PG fallback query failed");
                return Vec::new();
            }
        };

        // 异步回填 Redis——不阻塞返回，Redis 不可用时静默忽略
        if !rows.is_empty() && self.redis.is_some() {
            let snapshot = self.clone();
            let uid = user_id.to_string();
            tokio::spawn(async move {
                if let Err(e) = snapshot.rebuild_from_pg(&uid).await {
                    tracing::warn!(
                        user_id = uid,
                        error = %e,
                        "wallet snapshot async rebuild failed"
                    );
                }
            });
        }

        rows
    }

    /// 重建单个用户的钱包快照——DEL 旧 key → ZADD 全部 → HSET asset_count + updated_at。
    ///
    /// 接收 (asset_id, score) 对列表，score 为 created_at 时间戳（秒级 epoch）。
    /// Redis 不可用时直接返回 Ok（best-effort）。
    pub async fn rebuild_for_user(
        &self,
        user_id: &str,
        asset_pairs: &[(String, f64)],
    ) -> Result<(), String> {
        let Some(mut conn) = self.redis.clone() else {
            return Ok(());
        };

        let set_key = format!("wallet:{user_id}:assets");
        let hash_key = format!("wallet:{user_id}");

        // 清除旧数据
        if let Err(e) = conn.del::<_, ()>(&set_key).await {
            return Err(format!("DEL set_key failed: {e}"));
        }
        if let Err(e) = conn.del::<_, ()>(&hash_key).await {
            return Err(format!("DEL hash_key failed: {e}"));
        }

        // ZADD 全部资产——逐条写入，Redis pipeline 在 MultiplexedConnection 上已自动 batch
        for (asset_id, score) in asset_pairs {
            if let Err(e) = conn.zadd::<_, _, _, ()>(&set_key, asset_id.as_str(), *score).await {
                return Err(format!("ZADD failed for {asset_id}: {e}"));
            }
        }

        // HSET asset_count + updated_at
        let count = asset_pairs.len() as i64;
        let now = chrono::Utc::now().to_rfc3339();
        if let Err(e) = conn
            .hset_multiple::<_, _, _, ()>(
                &hash_key,
                &[
                    ("asset_count", count.to_string()),
                    ("updated_at", now),
                ],
            )
            .await
        {
            return Err(format!("HSET failed: {e}"));
        }

        Ok(())
    }

    /// 从 PG 查询该用户全部资产（含 created_at）并重建 Redis 快照——
    /// 内部使用 rebuild_for_user 完成实际写入。
    /// pub 可见性：CLI wallet-snapshot rebuild 子命令需直接调用。
    pub async fn rebuild_from_pg(&self, user_id: &str) -> Result<(), String> {
        let rows = sqlx::query_as::<_, (String, i64)>(
            "SELECT asset_id, EXTRACT(EPOCH FROM created_at)::bigint as ts \
             FROM assets WHERE owner_id = $1 ORDER BY created_at DESC",
        )
        .bind(user_id)
        .fetch_all(&self.db)
        .await
        .map_err(|e| format!("PG query for rebuild failed: {e}"))?;

        let pairs: Vec<(String, f64)> = rows
            .into_iter()
            .map(|(id, ts)| (id, ts as f64))
            .collect();

        self.rebuild_for_user(user_id, &pairs).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use sqlx::postgres::PgPoolOptions;

    /// 创建不实际连接的 PgPool——redis=None 的测试不会触发 PG 查询，
    /// connect_lazy 保证在首次 query 前不建立 TCP 连接。
    fn dummy_pg_pool() -> PgPool {
        PgPoolOptions::new()
            .max_connections(1)
            .connect_lazy("postgres://dummy:dummy@localhost:1/dummy")
            .expect("connect_lazy should not fail")
    }

    // ────────────────────────────────────────────────────────
    // 测试：Redis 不可用（redis=None）→ get_assets 降级到 PG
    //
    // 因为 PgPool 是 connect_lazy 的 dummy，实际 fetch 会失败，
    // get_assets 内部 catch 住错误返回空 vec——验证降级路径不 panic。
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_get_assets_no_redis_returns_empty_on_pg_error() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // redis=None → 直接走 PG fallback → dummy PG 连接失败 → 返回空
        let result = snapshot.get_assets("user_001", 0, 10).await;
        assert!(result.is_empty(), "redis=None + PG 不可达时应返回空 vec");
    }

    // ────────────────────────────────────────────────────────
    // 测试：Redis 不可用 → rebuild_for_user 是 best-effort，直接返回 Ok
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_rebuild_no_redis_returns_ok() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        let pairs = vec![
            ("asset_a".to_string(), 1700000000.0),
            ("asset_b".to_string(), 1700000100.0),
        ];

        // redis=None 时 rebuild_for_user 应直接返回 Ok，不报错
        let result = snapshot.rebuild_for_user("user_001", &pairs).await;
        assert!(result.is_ok(), "redis=None 时 rebuild 应 best-effort 返回 Ok");
    }

    // ────────────────────────────────────────────────────────
    // 测试：on_asset_state_change 在 redis=None 时不 panic
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_state_change_no_redis_noop() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // redis=None 时 add_asset/remove_asset 内部 early return，不应 panic
        snapshot
            .on_asset_state_change(
                "asset_001",
                Some("old_owner"),
                Some("new_owner"),
                1700000000,
            )
            .await;
    }

    // ────────────────────────────────────────────────────────
    // 测试：get_assets 参数正确性——page=0, page_size=10 时
    // offset 应为 0，不应 panic
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_get_assets_page_zero() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // page=0 是合法的 0-indexed 首页
        let result = snapshot.get_assets("user_002", 0, 10).await;
        assert!(result.is_empty(), "dummy PG 不可达应返回空");
    }

    // ────────────────────────────────────────────────────────
    // 测试：get_assets 大 page 不 panic
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_get_assets_large_page() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // 超大 page 不应导致整数溢出或 panic
        let result = snapshot.get_assets("user_003", 9999, 50).await;
        assert!(result.is_empty());
    }

    // ────────────────────────────────────────────────────────
    // 测试：rebuild_for_user 空列表——应清除旧 key 并设置 count=0
    // redis=None 时直接 Ok
    // ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn wallet_snapshot_rebuild_empty_pairs() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        let result = snapshot.rebuild_for_user("user_004", &[]).await;
        assert!(result.is_ok(), "空 pairs 列表应正常完成");
    }

    // ════════════════════════════════════════════════════════════════
    // 降级策略测试 — `cargo test fallback` 可命中此节所有测试
    //
    // WalletSnapshot 的降级通过 AppState 层传播 redis=None 实现：
    //   - DirectPg → AppState.redis = None → WalletSnapshot 构造时 redis=None
    //   - Auto + Redis 宕机 → AppState.redis = None → 同上
    // 因此只需验证 redis=None 时各操作的降级行为即可。
    // ════════════════════════════════════════════════════════════════

    /// redis=None 时 on_asset_state_change 为 no-op——
    /// add_asset / remove_asset 内部 early return，不应 panic 或报错。
    #[tokio::test]
    async fn fallback_wallet_snapshot_no_redis_state_change_noop() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // 完整的状态变更场景：旧持有者移除 + 新持有者添加
        snapshot
            .on_asset_state_change("asset_fb_01", Some("old_user"), Some("new_user"), 1700000000)
            .await;

        // 仅移除（Consumed / Legacy 场景）
        snapshot
            .on_asset_state_change("asset_fb_02", Some("owner"), None, 1700000001)
            .await;

        // 仅添加（LegallySold 场景）
        snapshot
            .on_asset_state_change("asset_fb_03", None, Some("buyer"), 1700000002)
            .await;

        // 无新旧持有者——边界情况
        snapshot
            .on_asset_state_change("asset_fb_04", None, None, 1700000003)
            .await;
    }

    /// redis=None 时 get_assets 降级到 PG 路径——
    /// dummy PG 连接不可达，应返回空 vec 而非 panic。
    #[tokio::test]
    async fn fallback_wallet_snapshot_no_redis_get_assets_degrades() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        // redis=None → 跳过 Redis 分支 → 直接走 PG fallback
        let result = snapshot.get_assets("user_fallback_01", 0, 20).await;
        assert!(result.is_empty(), "redis=None + PG 不可达时应返回空 vec");
    }

    /// redis=None 时 rebuild_for_user 为 best-effort 返回 Ok——
    /// 降级场景下不应报错。
    #[tokio::test]
    async fn fallback_wallet_snapshot_no_redis_rebuild_ok() {
        let snapshot = WalletSnapshot::new(None, dummy_pg_pool());

        let pairs = vec![
            ("asset_x".to_string(), 1700000000.0),
            ("asset_y".to_string(), 1700000100.0),
        ];
        let result = snapshot.rebuild_for_user("user_fallback_02", &pairs).await;
        assert!(result.is_ok(), "redis=None 时 rebuild 应 best-effort 返回 Ok");
    }
}
