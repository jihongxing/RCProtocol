use clap::{Subcommand, ValueEnum};
use redis::AsyncCommands;
use sqlx::PgPool;

/// CTR 校准 CLI 子命令——检查和修正 Redis L2 与 PG L3 之间 CTR 值的不一致
#[derive(Subcommand, Debug)]
pub enum CtrAction {
    /// 检查所有 UID 的 Redis vs PG CTR 一致性
    Check,
    /// 使用 UseMax 策略自动校准（差值 > 1000 的跳过）
    Auto,
    /// 校准单个 UID
    Uid {
        #[arg(long)]
        uid: String,
        #[arg(long, value_enum)]
        action: CalibrateAction,
        /// 仅 SetValue 策略需要——指定要写入的 CTR 值
        #[arg(long)]
        value: Option<u32>,
    },
}

/// 单 UID 校准策略
#[derive(ValueEnum, Clone, Debug)]
pub enum CalibrateAction {
    /// 以 Redis 值为准，更新 PG
    UseRedis,
    /// 以 PG 值为准，更新 Redis
    UsePg,
    /// 取两方较大值，更新较小方（CTR 单调递增保护）
    UseMax,
    /// 使用指定值覆盖 Redis 和 PG
    SetValue,
}

/// 差值超过此阈值时 Auto 模式跳过，需人工审核
const AUTO_SKIP_THRESHOLD: u64 = 1000;

/// CTR 校准入口——由 main.rs CLI 分发调用
pub async fn run(
    action: CtrAction,
    db: &PgPool,
    redis: &mut redis::aio::MultiplexedConnection,
) {
    match action {
        CtrAction::Check => run_check(db, redis).await,
        CtrAction::Auto => run_auto(db, redis).await,
        CtrAction::Uid { uid, action: strategy, value } => {
            run_uid(db, redis, &uid, strategy, value).await;
        }
    }
}

/// Check：遍历所有 UID，逐个对比 Redis 与 PG 的 CTR 值
async fn run_check(db: &PgPool, redis: &mut redis::aio::MultiplexedConnection) {
    println!("Checking CTR consistency...");

    let all_uids = fetch_all_uids_with_ctr(db).await;
    let total = all_uids.len();
    let mut consistent: u64 = 0;
    let mut mismatches: Vec<(String, u32, i32)> = Vec::new();

    for (uid, pg_ctr) in &all_uids {
        match get_redis_ctr(redis, uid).await {
            Some(redis_ctr) => {
                if redis_ctr == *pg_ctr as u32 {
                    consistent += 1;
                } else {
                    mismatches.push((uid.clone(), redis_ctr, *pg_ctr));
                }
            }
            // Redis 中无记录——可能从未缓存过，不计为 mismatch
            None => {
                consistent += 1;
            }
        }
    }

    println!("Total UIDs: {total}");
    println!("Consistent: {consistent}");
    println!("Mismatches: {}", mismatches.len());
    for (uid, redis_val, pg_val) in &mismatches {
        let suggestion = if *redis_val > *pg_val as u32 {
            "UseRedis"
        } else {
            "UsePg"
        };
        println!("  UID: {uid}  Redis: {redis_val}  PG: {pg_val}  Suggested: {suggestion}");
    }
}

/// Auto：UseMax 策略批量校准——较大方覆盖较小方，差值 > 1000 时跳过
async fn run_auto(db: &PgPool, redis: &mut redis::aio::MultiplexedConnection) {
    println!("Auto-calibrating CTR values (UseMax strategy)...");

    let all_uids = fetch_all_uids_with_ctr(db).await;
    let mut corrected: u64 = 0;
    let mut skipped: u64 = 0;

    for (uid, pg_ctr) in &all_uids {
        let redis_ctr = match get_redis_ctr(redis, uid).await {
            Some(v) => v,
            None => continue,
        };

        let pg_u32 = *pg_ctr as u32;
        if redis_ctr == pg_u32 {
            continue;
        }

        let diff = (redis_ctr as i64 - pg_u32 as i64).unsigned_abs();
        if diff > AUTO_SKIP_THRESHOLD {
            println!("  SKIP (manual review): {uid}  Redis: {redis_ctr}  PG: {pg_ctr}  diff={diff}");
            skipped += 1;
            continue;
        }

        if redis_ctr > pg_u32 {
            // Redis 更大——更新 PG 到 Redis 值
            update_pg_ctr(db, uid, redis_ctr).await;
            println!("  UPDATED PG: {uid}  {pg_ctr} -> {redis_ctr}");
        } else {
            // PG 更大——更新 Redis 到 PG 值
            set_redis_ctr(redis, uid, pg_u32).await;
            println!("  UPDATED Redis: {uid}  {redis_ctr} -> {pg_u32}");
        }
        corrected += 1;
    }

    println!("Corrected: {corrected} UIDs, Skipped: {skipped} UIDs");
}

/// Uid：按指定策略校准单个 UID
async fn run_uid(
    db: &PgPool,
    redis: &mut redis::aio::MultiplexedConnection,
    uid: &str,
    strategy: CalibrateAction,
    value: Option<u32>,
) {
    let pg_ctr = fetch_pg_ctr(db, uid).await;
    let redis_ctr = get_redis_ctr(redis, uid).await;

    println!("UID: {uid}  PG: {pg_ctr:?}  Redis: {redis_ctr:?}");

    match strategy {
        CalibrateAction::UseRedis => {
            let Some(r) = redis_ctr else {
                println!("  ERROR: Redis has no CTR for this UID");
                return;
            };
            update_pg_ctr(db, uid, r).await;
            println!("  UPDATED PG: -> {r}");
        }
        CalibrateAction::UsePg => {
            let Some(pg) = pg_ctr else {
                println!("  ERROR: PG has no CTR for this UID");
                return;
            };
            set_redis_ctr(redis, uid, pg as u32).await;
            println!("  UPDATED Redis: -> {pg}");
        }
        CalibrateAction::UseMax => {
            let r = redis_ctr.unwrap_or(0);
            let p = pg_ctr.unwrap_or(0) as u32;
            let max_val = r.max(p);
            if r < max_val {
                set_redis_ctr(redis, uid, max_val).await;
                println!("  UPDATED Redis: {r} -> {max_val}");
            }
            if p < max_val {
                update_pg_ctr(db, uid, max_val).await;
                println!("  UPDATED PG: {p} -> {max_val}");
            }
            if r == p {
                println!("  Already consistent: {r}");
            }
        }
        CalibrateAction::SetValue => {
            let Some(v) = value else {
                println!("  ERROR: --value is required for set-value strategy");
                return;
            };
            set_redis_ctr(redis, uid, v).await;
            update_pg_ctr(db, uid, v).await;
            println!("  SET both Redis and PG: -> {v}");
        }
    }
}

// ────────────────────────────────────────────────────────
// 数据访问辅助函数
// ────────────────────────────────────────────────────────

/// 从 PG 查询所有有 UID 的资产及其 last_verified_ctr
async fn fetch_all_uids_with_ctr(db: &PgPool) -> Vec<(String, i32)> {
    sqlx::query_as::<_, (String, i32)>(
        "SELECT uid, COALESCE(last_verified_ctr, 0) FROM assets WHERE uid IS NOT NULL",
    )
    .fetch_all(db)
    .await
    .unwrap_or_else(|e| {
        tracing::error!(error = %e, "failed to fetch UIDs from PG");
        Vec::new()
    })
}

/// 从 PG 查询单个 UID 的 last_verified_ctr
async fn fetch_pg_ctr(db: &PgPool, uid: &str) -> Option<i32> {
    sqlx::query_scalar::<_, Option<i32>>(
        "SELECT last_verified_ctr FROM assets WHERE uid = $1",
    )
    .bind(uid)
    .fetch_optional(db)
    .await
    .unwrap_or_else(|e| {
        tracing::error!(uid = uid, error = %e, "failed to fetch CTR from PG");
        None
    })
    .flatten()
}

/// 从 Redis 读取 ctr:{uid} 的值
async fn get_redis_ctr(conn: &mut redis::aio::MultiplexedConnection, uid: &str) -> Option<u32> {
    let key = format!("ctr:{uid}");
    conn.get::<_, Option<u32>>(&key).await.unwrap_or_else(|e| {
        tracing::warn!(uid = uid, error = %e, "Redis GET ctr:{uid} failed");
        None
    })
}

/// 更新 PG 中指定 UID 的 last_verified_ctr
async fn update_pg_ctr(db: &PgPool, uid: &str, ctr: u32) {
    if let Err(e) = sqlx::query("UPDATE assets SET last_verified_ctr = $1 WHERE uid = $2")
        .bind(ctr as i32)
        .bind(uid)
        .execute(db)
        .await
    {
        tracing::error!(uid = uid, ctr = ctr, error = %e, "failed to update PG CTR");
    }
}

/// 设置 Redis 中 ctr:{uid} 的值（24h TTL）
async fn set_redis_ctr(conn: &mut redis::aio::MultiplexedConnection, uid: &str, ctr: u32) {
    let key = format!("ctr:{uid}");
    if let Err(e) = conn.set_ex::<_, _, ()>(&key, ctr, 86400).await {
        tracing::error!(uid = uid, ctr = ctr, error = %e, "failed to SET Redis CTR");
    }
}

// ────────────────────────────────────────────────────────
// 单元测试
// ────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    // ── Check 输出格式正确 ──
    // 验证 run_check 的核心逻辑：一致/不一致分类和计数
    #[derive(Debug)]
    struct CheckResult {
        total: usize,
        consistent: u64,
        mismatches: Vec<(String, u32, i32)>,
    }

    /// 纯逻辑版 check——不依赖真实 DB/Redis，直接接收两组数据进行比对
    fn check_logic(
        pg_data: &[(String, i32)],
        redis_data: &[(String, Option<u32>)],
    ) -> CheckResult {
        let redis_map: std::collections::HashMap<&str, Option<u32>> = redis_data
            .iter()
            .map(|(uid, v)| (uid.as_str(), *v))
            .collect();

        let mut consistent = 0u64;
        let mut mismatches = Vec::new();

        for (uid, pg_ctr) in pg_data {
            match redis_map.get(uid.as_str()).copied().flatten() {
                Some(redis_ctr) => {
                    if redis_ctr == *pg_ctr as u32 {
                        consistent += 1;
                    } else {
                        mismatches.push((uid.clone(), redis_ctr, *pg_ctr));
                    }
                }
                None => {
                    consistent += 1;
                }
            }
        }

        CheckResult {
            total: pg_data.len(),
            consistent,
            mismatches,
        }
    }

    /// 纯逻辑版 auto（UseMax 策略）——返回需要执行的校准动作
    #[derive(Debug, PartialEq)]
    enum CalibrationOp {
        UpdatePg { uid: String, new_ctr: u32 },
        UpdateRedis { uid: String, new_ctr: u32 },
        Skip { uid: String, diff: u64 },
    }

    fn auto_logic(
        pg_data: &[(String, i32)],
        redis_data: &[(String, Option<u32>)],
    ) -> Vec<CalibrationOp> {
        let redis_map: std::collections::HashMap<&str, Option<u32>> = redis_data
            .iter()
            .map(|(uid, v)| (uid.as_str(), *v))
            .collect();

        let mut ops = Vec::new();

        for (uid, pg_ctr) in pg_data {
            let redis_ctr = match redis_map.get(uid.as_str()).copied().flatten() {
                Some(v) => v,
                None => continue,
            };

            let pg_u32 = *pg_ctr as u32;
            if redis_ctr == pg_u32 {
                continue;
            }

            let diff = (redis_ctr as i64 - pg_u32 as i64).unsigned_abs();
            if diff > AUTO_SKIP_THRESHOLD {
                ops.push(CalibrationOp::Skip {
                    uid: uid.clone(),
                    diff,
                });
                continue;
            }

            if redis_ctr > pg_u32 {
                ops.push(CalibrationOp::UpdatePg {
                    uid: uid.clone(),
                    new_ctr: redis_ctr,
                });
            } else {
                ops.push(CalibrationOp::UpdateRedis {
                    uid: uid.clone(),
                    new_ctr: pg_u32,
                });
            }
        }

        ops
    }

    #[test]
    fn check_all_consistent() {
        let pg = vec![
            ("UID_A".to_string(), 10),
            ("UID_B".to_string(), 20),
        ];
        let redis = vec![
            ("UID_A".to_string(), Some(10u32)),
            ("UID_B".to_string(), Some(20u32)),
        ];

        let result = check_logic(&pg, &redis);
        assert_eq!(result.total, 2);
        assert_eq!(result.consistent, 2);
        assert!(result.mismatches.is_empty());
    }

    #[test]
    fn check_with_mismatches() {
        let pg = vec![
            ("UID_A".to_string(), 10),
            ("UID_B".to_string(), 20),
            ("UID_C".to_string(), 30),
        ];
        let redis = vec![
            ("UID_A".to_string(), Some(10u32)),
            ("UID_B".to_string(), Some(99u32)),
            ("UID_C".to_string(), Some(30u32)),
        ];

        let result = check_logic(&pg, &redis);
        assert_eq!(result.total, 3);
        assert_eq!(result.consistent, 2);
        assert_eq!(result.mismatches.len(), 1);
        assert_eq!(result.mismatches[0].0, "UID_B");
        assert_eq!(result.mismatches[0].1, 99); // Redis
        assert_eq!(result.mismatches[0].2, 20); // PG
    }

    #[test]
    fn check_redis_missing_counts_as_consistent() {
        let pg = vec![("UID_A".to_string(), 10)];
        let redis: Vec<(String, Option<u32>)> = vec![("UID_A".to_string(), None)];

        let result = check_logic(&pg, &redis);
        assert_eq!(result.consistent, 1);
        assert!(result.mismatches.is_empty());
    }

    #[test]
    fn auto_use_max_updates_smaller_side() {
        let pg = vec![
            ("UID_A".to_string(), 10),  // PG < Redis → 更新 PG
            ("UID_B".to_string(), 50),  // PG > Redis → 更新 Redis
        ];
        let redis = vec![
            ("UID_A".to_string(), Some(20u32)),
            ("UID_B".to_string(), Some(30u32)),
        ];

        let ops = auto_logic(&pg, &redis);
        assert_eq!(ops.len(), 2);
        assert_eq!(ops[0], CalibrationOp::UpdatePg { uid: "UID_A".to_string(), new_ctr: 20 });
        assert_eq!(ops[1], CalibrationOp::UpdateRedis { uid: "UID_B".to_string(), new_ctr: 50 });
    }

    #[test]
    fn auto_skips_large_diff() {
        let pg = vec![("UID_A".to_string(), 100)];
        let redis = vec![("UID_A".to_string(), Some(5000u32))];

        let ops = auto_logic(&pg, &redis);
        assert_eq!(ops.len(), 1);
        match &ops[0] {
            CalibrationOp::Skip { uid, diff } => {
                assert_eq!(uid, "UID_A");
                assert_eq!(*diff, 4900);
            }
            other => panic!("expected Skip, got {other:?}"),
        }
    }

    #[test]
    fn auto_equal_values_no_op() {
        let pg = vec![("UID_A".to_string(), 42)];
        let redis = vec![("UID_A".to_string(), Some(42u32))];

        let ops = auto_logic(&pg, &redis);
        assert!(ops.is_empty(), "equal values should produce no operations");
    }

    #[test]
    fn auto_redis_missing_skipped() {
        let pg = vec![("UID_A".to_string(), 42)];
        let redis: Vec<(String, Option<u32>)> = vec![("UID_A".to_string(), None)];

        let ops = auto_logic(&pg, &redis);
        assert!(ops.is_empty(), "Redis 无记录时 auto 不应产生操作");
    }

    #[test]
    fn auto_diff_exactly_1000_not_skipped() {
        // 边界：diff == 1000 不超过阈值，应正常校准
        let pg = vec![("UID_A".to_string(), 0)];
        let redis = vec![("UID_A".to_string(), Some(1000u32))];

        let ops = auto_logic(&pg, &redis);
        assert_eq!(ops.len(), 1);
        assert_eq!(ops[0], CalibrationOp::UpdatePg { uid: "UID_A".to_string(), new_ctr: 1000 });
    }

    #[test]
    fn auto_diff_1001_skipped() {
        // 边界：diff == 1001 超过阈值，应跳过
        let pg = vec![("UID_A".to_string(), 0)];
        let redis = vec![("UID_A".to_string(), Some(1001u32))];

        let ops = auto_logic(&pg, &redis);
        assert_eq!(ops.len(), 1);
        match &ops[0] {
            CalibrationOp::Skip { uid, diff } => {
                assert_eq!(uid, "UID_A");
                assert_eq!(*diff, 1001);
            }
            other => panic!("expected Skip, got {other:?}"),
        }
    }
}
