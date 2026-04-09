use jsonwebtoken::{encode, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    sub: String,
    role: String,
    brand_id: Option<String>,
    exp: usize,
    iat: usize,
}

fn main() {
    let secret = std::env::var("RC_JWT_SECRET")
        .unwrap_or_else(|_| "my-super-secret-jwt-key-for-testing-only".to_string());

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as usize;

    let claims = Claims {
        sub: "platform-admin".to_string(),
        role: "Platform".to_string(),
        brand_id: None,
        exp: now + (365 * 24 * 60 * 60), // 1年有效期
        iat: now,
    };

    let token = encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(secret.as_ref()),
    )
    .unwrap();

    println!("{}", token);
}
