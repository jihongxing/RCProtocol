use clap::Subcommand;
use redis::AsyncCommands;
use sqlx::PgPool;
use std::collections::HashMap;

/// 钱包快照 CLI 子命令——重建、清除、查看 Redis 中的钱包快照数据
#[derive(Subcommand, Debug)]
pub enum SnapshotAction {
    /// 重建指定用户（或所有用户）的钱包快照
    Rebuild {
        /// 指定用户 ID；省略时重建所有用户
        #[arg(long)]
        user_id: Option<String>,
    },
    /// 清除指定用户的钱包快照（DEL Hash + Sorted Set）
    Purge {
        #[arg(long)]
        user_id: String,
    },
    /// 查看指定用户的钱包快照统计信息
    Stats {
        #[arg(long)]
        user_id: String,
    },
}

/// 钱包快照 CLI 入口——由 main.rs CLI 分发调用
pub async fn run(
    action: SnapshotAction,
    db: &PgPool,
    redis: &mut redis::aio::MultiplexedConnection,
) {
    match action {
        SnapshotAction::Rebuild { user_id } => run_rebuild(db, redis, user_id).await,
        SnapshotAction::Purge { user_id } => run_purge(redis, &user_id).await,
        SnapshotAction::Stats { user_id } => run_stats(redis, &user_id).await,
    }
}

/// Rebuild：从 PG 查询资产数据并重建 Redis 快照
async fn run_rebuild(
    db: &PgPool,
    redis: &mut redis::aio::MultiplexedConnection,
    user_id: Option<String>,
) {
    let snapshot = crate::cache::wallet_snapshot::WalletSnapshot::new(
        Some(redis.clone()),
        db.clone(),
    );

    match user_id {
        Some(uid) => {
            println!("Rebuilding wallet snapshot for user: {uid}");
            match snapshot.rebuild_from_pg(&uid).await {
                Ok(()) => println!("  OK — rebuilt snapshot for {uid}"),
                Err(e) => println!("  ERROR — {e}"),
            }
            println!("Rebuilt: 1 user");
        }
        None => {
            // 查询所有有 owner_id 的用户
            println!("Rebuilding wallet snapshots for all users...");
            let owner_ids = fetch_all_owner_ids(db).await;
            let total = owner_ids.len();
            let mut success = 0u64;
            let mut failed = 0u64;

            for (i, uid) in owner_ids.iter().enumerate() {
                match snapshot.rebuild_from_pg(uid).await {
                    Ok(()) => {
                        success += 1;
                    }
                    Err(e) => {
                        println!("  ERROR user {uid}: {e}");
                        failed += 1;
                    }
                }
                // 每 100 个输出进度
                if (i + 1) % 100 == 0 {
                    println!("  progress: {}/{total}", i + 1);
                }
            }

            println!("Rebuilt: {success} users, Failed: {failed} users (Total: {total})");
        }
    }
}

/// Purge：删除指定用户的 Redis 钱包快照 key
async fn run_purge(redis: &mut redis::aio::MultiplexedConnection, user_id: &str) {
    let (hash_key, set_key) = wallet_keys(user_id);

    println!("Purging wallet snapshot for user: {user_id}");

    if let Err(e) = redis.del::<_, ()>(&hash_key).await {
        println!("  ERROR deleting {hash_key}: {e}");
    }
    if let Err(e) = redis.del::<_, ()>(&set_key).await {
        println!("  ERROR deleting {set_key}: {e}");
    }

    println!("  Purged: {hash_key}, {set_key}");
}

/// Stats：输出指定用户的钱包快照统计信息
async fn run_stats(redis: &mut redis::aio::MultiplexedConnection, user_id: &str) {
    let (hash_key, set_key) = wallet_keys(user_id);

    println!("Wallet snapshot stats for user: {user_id}");

    // HGETALL wallet:{user_id}
    match redis.hgetall::<_, HashMap<String, String>>(&hash_key).await {
        Ok(fields) => {
            if fields.is_empty() {
                println!("  Hash {hash_key}: (not found)");
            } else {
                let asset_count = fields.get("asset_count").map(|s| s.as_str()).unwrap_or("N/A");
                let updated_at = fields.get("updated_at").map(|s| s.as_str()).unwrap_or("N/A");
                println!("  asset_count: {asset_count}");
                println!("  updated_at:  {updated_at}");
            }
        }
        Err(e) => {
            println!("  ERROR HGETALL {hash_key}: {e}");
        }
    }

    // ZCARD wallet:{user_id}:assets
    match redis.zcard::<_, i64>(&set_key).await {
        Ok(card) => {
            println!("  sorted_set_size: {card}");
        }
        Err(e) => {
            println!("  ERROR ZCARD {set_key}: {e}");
        }
    }
}

// ────────────────────────────────────────────────────────
// 辅助函数
// ────────────────────────────────────────────────────────

/// 构造钱包快照的 Redis key 对——Hash key 和 Sorted Set key
pub fn wallet_keys(user_id: &str) -> (String, String) {
    let hash_key = format!("wallet:{user_id}");
    let set_key = format!("wallet:{user_id}:assets");
    (hash_key, set_key)
}

/// 从 PG 查询所有有 owner_id 的去重用户列表
async fn fetch_all_owner_ids(db: &PgPool) -> Vec<String> {
    sqlx::query_scalar::<_, String>(
        "SELECT DISTINCT owner_id FROM assets WHERE owner_id IS NOT NULL",
    )
    .fetch_all(db)
    .await
    .unwrap_or_else(|e| {
        tracing::error!(error = %e, "failed to fetch distinct owner_ids from PG");
        Vec::new()
    })
}

/// 格式化 stats 输出——纯逻辑函数，供测试使用
pub fn format_stats_output(
    user_id: &str,
    hash_fields: &HashMap<String, String>,
    sorted_set_size: i64,
) -> String {
    let mut out = format!("Wallet snapshot stats for user: {user_id}\n");

    if hash_fields.is_empty() {
        let hash_key = format!("wallet:{user_id}");
        out.push_str(&format!("  Hash {hash_key}: (not found)\n"));
    } else {
        let asset_count = hash_fields.get("asset_count").map(|s| s.as_str()).unwrap_or("N/A");
        let updated_at = hash_fields.get("updated_at").map(|s| s.as_str()).unwrap_or("N/A");
        out.push_str(&format!("  asset_count: {asset_count}\n"));
        out.push_str(&format!("  updated_at:  {updated_at}\n"));
    }

    out.push_str(&format!("  sorted_set_size: {sorted_set_size}\n"));
    out
}

// ────────────────────────────────────────────────────────
// 单元测试
// ────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    // ── Purge key 构造正确性 ──
    #[test]
    fn purge_deletes_correct_keys() {
        let (hash_key, set_key) = wallet_keys("user_abc_123");
        assert_eq!(hash_key, "wallet:user_abc_123");
        assert_eq!(set_key, "wallet:user_abc_123:assets");
    }

    #[test]
    fn purge_keys_empty_user_id() {
        // 空 user_id 边界——key 结构仍合法（实际业务不应出现）
        let (hash_key, set_key) = wallet_keys("");
        assert_eq!(hash_key, "wallet:");
        assert_eq!(set_key, "wallet::assets");
    }

    #[test]
    fn purge_keys_special_characters() {
        // user_id 含特殊字符——Redis key 不做转义，原样拼接
        let (hash_key, set_key) = wallet_keys("user:with:colons");
        assert_eq!(hash_key, "wallet:user:with:colons");
        assert_eq!(set_key, "wallet:user:with:colons:assets");
    }

    // ── Stats 输出格式 ──
    #[test]
    fn stats_output_with_data() {
        let mut fields = HashMap::new();
        fields.insert("asset_count".to_string(), "42".to_string());
        fields.insert("updated_at".to_string(), "2026-03-17T10:00:00Z".to_string());

        let output = format_stats_output("user_001", &fields, 42);

        assert!(output.contains("Wallet snapshot stats for user: user_001"));
        assert!(output.contains("asset_count: 42"));
        assert!(output.contains("updated_at:  2026-03-17T10:00:00Z"));
        assert!(output.contains("sorted_set_size: 42"));
    }

    #[test]
    fn stats_output_empty_hash() {
        let fields = HashMap::new();
        let output = format_stats_output("user_002", &fields, 0);

        assert!(output.contains("Wallet snapshot stats for user: user_002"));
        assert!(output.contains("Hash wallet:user_002: (not found)"));
        assert!(output.contains("sorted_set_size: 0"));
    }

    #[test]
    fn stats_output_partial_fields() {
        // Hash 中只有 asset_count，没有 updated_at
        let mut fields = HashMap::new();
        fields.insert("asset_count".to_string(), "10".to_string());

        let output = format_stats_output("user_003", &fields, 10);

        assert!(output.contains("asset_count: 10"));
        assert!(output.contains("updated_at:  N/A"));
        assert!(output.contains("sorted_set_size: 10"));
    }

    // ── SnapshotAction 枚举解析（clap） ──
    #[test]
    fn snapshot_action_rebuild_with_user_id() {
        use clap::Parser;

        #[derive(Parser)]
        struct TestCli {
            #[command(subcommand)]
            action: SnapshotAction,
        }

        let cli = TestCli::parse_from(["test", "rebuild", "--user-id", "user_xyz"]);
        match cli.action {
            SnapshotAction::Rebuild { user_id } => {
                assert_eq!(user_id, Some("user_xyz".to_string()));
            }
            _ => panic!("expected Rebuild"),
        }
    }

    #[test]
    fn snapshot_action_rebuild_all() {
        use clap::Parser;

        #[derive(Parser)]
        struct TestCli {
            #[command(subcommand)]
            action: SnapshotAction,
        }

        // 不传 --user-id 表示重建所有用户
        let cli = TestCli::parse_from(["test", "rebuild"]);
        match cli.action {
            SnapshotAction::Rebuild { user_id } => {
                assert_eq!(user_id, None);
            }
            _ => panic!("expected Rebuild with None"),
        }
    }

    #[test]
    fn snapshot_action_purge() {
        use clap::Parser;

        #[derive(Parser)]
        struct TestCli {
            #[command(subcommand)]
            action: SnapshotAction,
        }

        let cli = TestCli::parse_from(["test", "purge", "--user-id", "user_del"]);
        match cli.action {
            SnapshotAction::Purge { user_id } => {
                assert_eq!(user_id, "user_del");
            }
            _ => panic!("expected Purge"),
        }
    }

    #[test]
    fn snapshot_action_stats() {
        use clap::Parser;

        #[derive(Parser)]
        struct TestCli {
            #[command(subcommand)]
            action: SnapshotAction,
        }

        let cli = TestCli::parse_from(["test", "stats", "--user-id", "user_stats"]);
        match cli.action {
            SnapshotAction::Stats { user_id } => {
                assert_eq!(user_id, "user_stats");
            }
            _ => panic!("expected Stats"),
        }
    }
}
