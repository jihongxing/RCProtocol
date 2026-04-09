use std::sync::Arc;
use std::time::Instant;

use dashmap::DashMap;
use rc_kms::{KeyProvider, SoftwareKms};
use sqlx::{postgres::PgPoolOptions, PgPool};

use crate::auth::jwt::JwtDecoder;

/// Redis 不可用时的降级策略
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum FallbackStrategy {
    /// Redis 可用时使用三级缓冲，不可用时自动降级到 PG
    Auto,
    /// 强制直接查询 PostgreSQL，完全跳过 Redis
    DirectPg,
}

/// L1 进程内缓存条目，记录 CTR 值和缓存时间以支持 TTL 过期淘汰
#[derive(Clone, Debug)]
pub struct CtrEntry {
    pub ctr: u32,
    pub cached_at: Instant,
}

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub kms: Arc<dyn KeyProvider + Send + Sync>,
    pub jwt_decoder: Arc<JwtDecoder>,
    pub auth_disabled: bool,
    pub redis: Option<redis::aio::MultiplexedConnection>,
    pub ctr_cache: Arc<DashMap<String, CtrEntry>>,
    pub fallback_strategy: FallbackStrategy,
    /// Server secret for HMAC-SHA256 hashing of API Keys (from RC_API_KEY_SECRET env var)
    pub api_key_secret: Vec<u8>,
}

/// 从 REDIS_URL 环境变量解析 Redis 连接地址，未设置时使用默认值
pub fn parse_redis_url() -> String {
    std::env::var("REDIS_URL")
        .unwrap_or_else(|_| "redis://redis:6379".to_string())
}

/// 从 RC_API_FALLBACK_STRATEGY 环境变量解析降级策略，未设置或无法识别时默认 Auto
pub fn parse_fallback_strategy() -> FallbackStrategy {
    match std::env::var("RC_API_FALLBACK_STRATEGY")
        .unwrap_or_else(|_| "Auto".to_string())
        .as_str()
    {
        "DirectPg" => FallbackStrategy::DirectPg,
        _ => FallbackStrategy::Auto,
    }
}

impl AppState {
    pub async fn from_env() -> Self {
        let database_url = std::env::var("DATABASE_URL")
            .unwrap_or_else(|_| "postgres://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol".to_string());

        let db = PgPoolOptions::new()
            .max_connections(5)
            .connect(&database_url)
            .await
            .expect("connect postgres");

        sqlx::migrate!("./migrations")
            .run(&db)
            .await
            .expect("database migration failed");
        tracing::info!("database migrations applied");

        if std::env::var("RC_SEED_DATA").map(|v| v == "true").unwrap_or(false) {
            crate::seed::run_seed(&db).await;
            tracing::info!("seed data injected");
        }

        let kms: Arc<dyn KeyProvider + Send + Sync> = Arc::new(
            SoftwareKms::from_env().expect("KMS initialization failed — check RC_ROOT_KEY_HEX and RC_SYSTEM_ID"),
        );

        let auth_disabled = std::env::var("RC_AUTH_DISABLED")
            .map(|v| v == "true")
            .unwrap_or(false);

        let jwt_secret = if auth_disabled {
            b"unused-dev-placeholder".to_vec()
        } else {
            std::env::var("RC_JWT_SECRET")
                .expect("RC_JWT_SECRET must be set when auth is enabled")
                .into_bytes()
        };

        let jwt_decoder = Arc::new(JwtDecoder::new(&jwt_secret));

        let redis_url = parse_redis_url();
        let fallback_strategy = parse_fallback_strategy();
        tracing::info!(redis_url = %redis_url, fallback_strategy = ?fallback_strategy, "config loaded");

        // DirectPg 策略下跳过 Redis 连接——运维显式选择不使用 Redis
        let redis = if fallback_strategy == FallbackStrategy::DirectPg {
            tracing::info!("fallback_strategy=DirectPg, skipping Redis connection");
            None
        } else {
            match redis::Client::open(redis_url.as_str()) {
                Ok(client) => match client.get_multiplexed_tokio_connection().await {
                    Ok(conn) => {
                        tracing::info!("Redis connected");
                        Some(conn)
                    }
                    Err(e) => {
                        // 连接失败不阻止启动，降级为 L3-only
                        tracing::warn!(error = %e, "Redis connection failed, running in L3-only mode");
                        None
                    }
                },
                Err(e) => {
                    tracing::warn!(error = %e, "Redis URL invalid, running in L3-only mode");
                    None
                }
            }
        };

        let ctr_cache = Arc::new(DashMap::new());

        let api_key_secret = std::env::var("RC_API_KEY_SECRET")
            .unwrap_or_else(|_| {
                tracing::warn!("RC_API_KEY_SECRET not set, using dev-only default — DO NOT use in production");
                "rc-dev-api-key-secret-do-not-use-in-prod".to_string()
            })
            .into_bytes();

        Self { db, kms, jwt_decoder, auth_disabled, redis, ctr_cache, fallback_strategy, api_key_secret }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;

    #[test]
    #[serial]
    fn config_redis_url_from_env() {
        // 显式设置环境变量后应返回设置值
        std::env::set_var("REDIS_URL", "redis://custom:6380/1");
        let url = parse_redis_url();
        assert_eq!(url, "redis://custom:6380/1");
        std::env::remove_var("REDIS_URL");
    }

    #[test]
    #[serial]
    fn config_redis_url_default() {
        // 未设置 REDIS_URL 时应返回默认值
        std::env::remove_var("REDIS_URL");
        let url = parse_redis_url();
        assert_eq!(url, "redis://redis:6379");
    }

    #[test]
    #[serial]
    fn config_fallback_strategy_direct_pg() {
        // 显式设置 DirectPg 应解析为对应枚举
        std::env::set_var("RC_API_FALLBACK_STRATEGY", "DirectPg");
        let strategy = parse_fallback_strategy();
        assert_eq!(strategy, FallbackStrategy::DirectPg);
        std::env::remove_var("RC_API_FALLBACK_STRATEGY");
    }

    #[test]
    #[serial]
    fn config_fallback_strategy_auto_default() {
        // 未设置时默认 Auto
        std::env::remove_var("RC_API_FALLBACK_STRATEGY");
        let strategy = parse_fallback_strategy();
        assert_eq!(strategy, FallbackStrategy::Auto);
    }

    #[test]
    #[serial]
    fn config_fallback_strategy_auto_explicit() {
        // 显式设置 Auto 应解析为 Auto
        std::env::set_var("RC_API_FALLBACK_STRATEGY", "Auto");
        let strategy = parse_fallback_strategy();
        assert_eq!(strategy, FallbackStrategy::Auto);
        std::env::remove_var("RC_API_FALLBACK_STRATEGY");
    }

    #[test]
    #[serial]
    fn config_fallback_strategy_unknown_defaults_to_auto() {
        // 无法识别的值默认为 Auto（不报错）
        std::env::set_var("RC_API_FALLBACK_STRATEGY", "SomeUnknownValue");
        let strategy = parse_fallback_strategy();
        assert_eq!(strategy, FallbackStrategy::Auto);
        std::env::remove_var("RC_API_FALLBACK_STRATEGY");
    }
}
