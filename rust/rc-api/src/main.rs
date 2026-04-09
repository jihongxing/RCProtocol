use clap::{Parser, Subcommand};
use rc_api::app::AppState;
use rc_api::auth::middleware::{auth_middleware, AuthState};
use axum::{middleware as axum_mw, Router};

#[derive(Parser)]
#[command(name = "rc-api", about = "RCProtocol API server and maintenance CLI")]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Start the API server (default when no subcommand is given)
    Serve,
    /// CTR calibration utilities — check/auto/uid
    CtrCalibrate {
        #[command(subcommand)]
        action: rc_api::cli::ctr_calibrate::CtrAction,
    },
    /// Wallet snapshot utilities — rebuild/purge/stats
    WalletSnapshot {
        #[command(subcommand)]
        action: rc_api::cli::wallet_snapshot_cli::SnapshotAction,
    },
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();

    match cli.command.unwrap_or(Commands::Serve) {
        Commands::Serve => run_serve().await,
        Commands::CtrCalibrate { action } => run_ctr_calibrate(action).await,
        Commands::WalletSnapshot { action } => run_wallet_snapshot(action).await,
    }
}

/// 启动 API 服务——保留原有完整逻辑
async fn run_serve() {
    validate_security_config();
    let state = AppState::from_env().await;
    let auth_state = AuthState::from(&state);

    let public = Router::new()
        .route("/healthz", axum::routing::get(rc_api::routes::health::healthz))
        .merge(rc_api::routes::verify::router())
        .route(
            "/verify/:asset_id",
            axum::routing::get(rc_api::routes::protocol::verify_asset),
        );

    let protected = rc_api::routes::protocol::write_router()
        .merge(rc_api::routes::assets::router())
        .merge(rc_api::routes::brand::router())
        .merge(rc_api::routes::batch::router())
        .merge(rc_api::routes::transfer::router())
        .merge(rc_api::routes::authority_devices::router())
        .layer(axum_mw::from_fn_with_state(auth_state, auth_middleware));

    let app = public.merge(protected).with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8081")
        .await
        .expect("bind rc-api listener");

    tracing::info!("rc-api listening on 0.0.0.0:8081");
    axum::serve(listener, app).await.expect("serve rc-api");
}

async fn run_ctr_calibrate(action: rc_api::cli::ctr_calibrate::CtrAction) {
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol".to_string());

    let db = sqlx::postgres::PgPoolOptions::new()
        .max_connections(2)
        .connect(&database_url)
        .await
        .expect("connect postgres for ctr-calibrate");

    let redis_url = rc_api::app::parse_redis_url();
    let client = redis::Client::open(redis_url.as_str())
        .expect("invalid REDIS_URL");
    let mut redis_conn = client
        .get_multiplexed_tokio_connection()
        .await
        .expect("connect redis for ctr-calibrate");

    rc_api::cli::ctr_calibrate::run(action, &db, &mut redis_conn).await;
}

async fn run_wallet_snapshot(action: rc_api::cli::wallet_snapshot_cli::SnapshotAction) {
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol".to_string());

    let db = sqlx::postgres::PgPoolOptions::new()
        .max_connections(2)
        .connect(&database_url)
        .await
        .expect("connect postgres for wallet-snapshot");

    let redis_url = rc_api::app::parse_redis_url();
    let client = redis::Client::open(redis_url.as_str())
        .expect("invalid REDIS_URL");
    let mut redis_conn = client
        .get_multiplexed_tokio_connection()
        .await
        .expect("connect redis for wallet-snapshot");

    rc_api::cli::wallet_snapshot_cli::run(action, &db, &mut redis_conn).await;
}

fn validate_security_config() {
    if let Ok(secret) = std::env::var("RC_JWT_SECRET") {
        if secret.len() < 32 {
            panic!(
                "RC_JWT_SECRET 长度不足 32 字节（当前 {} 字节）。生产环境必须使用强密钥。",
                secret.len()
            );
        }
    }

    if let Ok(hex) = std::env::var("RC_ROOT_KEY_HEX") {
        let bytes: Vec<u8> = (0..hex.len())
            .step_by(2)
            .filter_map(|i| hex.get(i..i + 2).and_then(|s| u8::from_str_radix(s, 16).ok()))
            .collect();

        if !bytes.is_empty() {
            if bytes.iter().all(|&b| b == 0) {
                panic!("RC_ROOT_KEY_HEX 为全零，生产环境必须使用安全随机密钥。");
            }
            let is_sequential = bytes.iter().enumerate().all(|(i, &b)| b == i as u8);
            if is_sequential {
                panic!("RC_ROOT_KEY_HEX 为递增序列，生产环境必须使用安全随机密钥。");
            }
        }
    }
}
