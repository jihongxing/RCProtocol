use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    sub: String,
    role: String,
    brand_id: Option<String>,
    exp: u64,
    iat: u64,
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() < 2 {
        eprintln!("Usage: verify_jwt <token>");
        std::process::exit(1);
    }

    let token = &args[1];
    let secret = std::env::var("RC_JWT_SECRET")
        .unwrap_or_else(|_| "my-super-secret-jwt-key-for-testing-only".to_string());

    println!("使用密钥: {}", secret);
    println!("密钥长度: {} 字节", secret.len());
    println!("Token: {}", token);
    println!();

    let mut validation = Validation::new(Algorithm::HS256);
    validation.set_required_spec_claims(&["sub", "exp", "iat"]);

    match decode::<Claims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &validation,
    ) {
        Ok(token_data) => {
            println!("✅ Token 验证成功!");
            println!("Claims: {:#?}", token_data.claims);
        }
        Err(e) => {
            println!("❌ Token 验证失败: {:?}", e);
        }
    }
}
