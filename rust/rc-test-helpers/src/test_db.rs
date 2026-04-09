use sqlx::{postgres::PgPoolOptions, PgPool};
use uuid::Uuid;

pub struct TestDb {
    pool: PgPool,
    db_name: String,
    admin_url: String,
}

impl TestDb {
    pub async fn new() -> Self {
        let admin_url = std::env::var("TEST_DATABASE_URL")
            .or_else(|_| std::env::var("DATABASE_URL").map(to_admin_database_url))
            .unwrap_or_else(|_| "postgres://rcprotocol:rcprotocol_dev@localhost:5433/postgres".to_string());

        let db_name = format!("rctest_{}", Uuid::new_v4().simple());

        let admin_pool = PgPoolOptions::new()
            .acquire_timeout(std::time::Duration::from_secs(5))
            .max_connections(2)
            .connect(&admin_url)
            .await
            .expect("failed to connect admin database for TestDb");

        sqlx::query(&format!("CREATE DATABASE \"{db_name}\""))
            .execute(&admin_pool)
            .await
            .expect("failed to create test database");

        admin_pool.close().await;

        let test_url = {
            let (base, _) = admin_url
                .rsplit_once('/')
                .expect("TEST_DATABASE_URL must contain a '/' before database name");
            format!("{base}/{db_name}")
        };

        let pool = PgPoolOptions::new()
            .acquire_timeout(std::time::Duration::from_secs(5))
            .max_connections(5)
            .connect(&test_url)
            .await
            .expect("failed to connect test database");

        sqlx::migrate!("../rc-api/migrations")
            .run(&pool)
            .await
            .expect("failed to run migrations on test database");

        Self { pool, db_name, admin_url }
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    pub fn db_name(&self) -> &str {
        &self.db_name
    }

    pub async fn cleanup(self) {
        self.pool.close().await;

        let admin_pool = PgPoolOptions::new()
            .acquire_timeout(std::time::Duration::from_secs(5))
            .max_connections(2)
            .connect(&self.admin_url)
            .await
            .expect("failed to reconnect admin for cleanup");

        let _ = sqlx::query(&format!(
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '{}'",
            self.db_name
        ))
        .execute(&admin_pool)
        .await;

        let _ = sqlx::query(&format!("DROP DATABASE IF EXISTS \"{}\"", self.db_name))
            .execute(&admin_pool)
            .await;

        admin_pool.close().await;
    }
}

fn to_admin_database_url(url: String) -> String {
    let (base, _) = url
        .rsplit_once('/')
        .expect("DATABASE_URL must contain a '/' before database name");
    format!("{base}/postgres")
}
