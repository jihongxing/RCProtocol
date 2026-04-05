mod app;
mod db;
mod routes;

use app::AppState;
use axum::Router;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();

    let state = AppState::from_env().await;

    let app = Router::new()
        .route("/healthz", axum::routing::get(routes::health::healthz))
        .merge(routes::protocol::router())
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8081")
        .await
        .expect("bind rc-api listener");

    tracing::info!("rc-api listening on 0.0.0.0:8081");
    axum::serve(listener, app).await.expect("serve rc-api");
}
