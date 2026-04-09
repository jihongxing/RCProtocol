use rand::Rng;
use sha2::{Digest, Sha256};

/// Generate a new API key in format: rcpk_live_<32 hex chars>
/// Example: rcpk_live_1234567890abcdef1234567890abcdef
pub fn generate_api_key() -> String {
    let random_bytes: [u8; 16] = rand::thread_rng().gen();
    let random_str = hex::encode(random_bytes);
    format!("rcpk_live_{}", random_str)
}

/// Hash an API key using SHA-256
/// Returns 64-character hex string
pub fn hash_api_key(api_key: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(api_key.as_bytes());
    hex::encode(hasher.finalize())
}

/// Extract key prefix for display (first 16 chars + ****)
/// Example: rcpk_live_1234****
pub fn extract_key_prefix(api_key: &str) -> String {
    if api_key.len() >= 16 {
        format!("{}****", &api_key[..16])
    } else {
        format!("{}****", api_key)
    }
}

/// Generate a ULID-based key ID
/// Format: key_01HQZX3K4M5N6P7Q8R9S0T1U2V
pub fn generate_key_id() -> String {
    format!("key_{}", ulid::Ulid::new())
}

/// Generate a ULID-based brand ID
/// Format: brand_01HQZX3K4M5N6P7Q8R9S0T1U2V
pub fn generate_brand_id() -> String {
    rc_common::ids::generate_brand_id()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_api_key_format() {
        let key = generate_api_key();
        assert!(key.starts_with("rcpk_live_"));
        assert_eq!(key.len(), 42); // "rcpk_live_" (10) + 32 hex chars
    }

    #[test]
    fn test_hash_api_key() {
        let key = "rcpk_live_1234567890abcdef1234567890abcdef";
        let hash = hash_api_key(key);
        assert_eq!(hash.len(), 64); // SHA-256 produces 64 hex chars

        // Hash should be deterministic
        let hash2 = hash_api_key(key);
        assert_eq!(hash, hash2);
    }

    #[test]
    fn test_extract_key_prefix() {
        let key = "rcpk_live_1234567890abcdef";
        let prefix = extract_key_prefix(key);
        assert_eq!(prefix, "rcpk_live_123456****");
    }

    #[test]
    fn test_generate_key_id_format() {
        let key_id = generate_key_id();
        assert!(key_id.starts_with("key_"));
        assert!(key_id.len() > 4);
    }

    #[test]
    fn test_generate_brand_id_format() {
        let brand_id = generate_brand_id();
        assert!(brand_id.starts_with("brand_"));
        assert_eq!(brand_id.len(), "brand_".len() + 26);
    }
}
