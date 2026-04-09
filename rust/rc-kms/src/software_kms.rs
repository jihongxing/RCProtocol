use std::collections::HashMap;
use std::sync::RwLock;

use rc_crypto::kdf;
use rc_crypto::{SecretKey16, SecretKey32};

use crate::error::KmsError;
use crate::provider::KeyProvider;
use crate::root_key::load_root_key_from_env;

pub struct SoftwareKms {
    root_key: SecretKey32,
    system_id: String,
    brand_key_cache: RwLock<HashMap<String, SecretKey32>>,
}

impl SoftwareKms {
    /// Initialize SoftwareKms from environment variables.
    ///
    /// Reads `RC_ROOT_KEY_HEX` (via `load_root_key_from_env`) and `RC_SYSTEM_ID`.
    pub fn from_env() -> Result<Self, KmsError> {
        let root_key = load_root_key_from_env()?;

        let system_id = std::env::var("RC_SYSTEM_ID").map_err(|_| KmsError::SystemIdMissing)?;

        tracing::info!("SoftwareKms initialized (root key loaded, system_id present)");

        Ok(Self {
            root_key,
            system_id,
            brand_key_cache: RwLock::new(HashMap::new()),
        })
    }

    /// Obtain a Brand Key (cached or freshly derived) and execute callback with it.
    ///
    /// Uses double-check locking: read lock first, then write lock on miss.
    /// RwLock poison causes panic — KMS state is unrecoverable.
    fn with_brand_key<T, F>(&self, brand_id: &str, f: F) -> Result<T, KmsError>
    where
        F: FnOnce(&SecretKey32) -> T,
    {
        // Fast path: read lock cache lookup
        {
            let cache = self.brand_key_cache.read().unwrap();
            if let Some(key) = cache.get(brand_id) {
                return Ok(f(key));
            }
        }

        // Slow path: write lock, double-check, derive and cache
        {
            let mut cache = self.brand_key_cache.write().unwrap();
            // Double-check: another thread may have populated the cache
            if let Some(key) = cache.get(brand_id) {
                return Ok(f(key));
            }

            let brand_key = kdf::derive_brand_key(
                &self.root_key,
                brand_id.as_bytes(),
                self.system_id.as_bytes(),
            );
            cache.insert(brand_id.to_string(), brand_key);
            let key = cache.get(brand_id).unwrap();
            Ok(f(key))
        }
    }
}

impl Drop for SoftwareKms {
    fn drop(&mut self) {
        // 手动清零 brand_key_cache 中的所有密钥
        // HashMap 的 Drop 不会触发值的 ZeroizeOnDrop
        if let Ok(mut cache) = self.brand_key_cache.write() {
            cache.drain().for_each(drop);
        }
    }
}

impl std::fmt::Debug for SoftwareKms {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("SoftwareKms")
            .field("root_key", &"[REDACTED]")
            .field("system_id", &self.system_id)
            .field(
                "cached_brands",
                &self.brand_key_cache.read().unwrap().len(),
            )
            .finish()
    }
}

impl KeyProvider for SoftwareKms {
    fn derive_chip_key(
        &self,
        brand_id: &str,
        uid: &[u8; 7],
        epoch: u32,
    ) -> Result<SecretKey16, KmsError> {
        let epoch_le = epoch.to_le_bytes();
        self.with_brand_key(brand_id, |brand_key| {
            kdf::derive_chip_key(brand_key, uid, &epoch_le)
        })
    }

    fn derive_honey_key(
        &self,
        brand_id: &str,
        serial: &[u8],
    ) -> Result<SecretKey32, KmsError> {
        self.with_brand_key(brand_id, |brand_key| {
            kdf::derive_honey_key(brand_key, serial)
        })
    }

    fn derive_mother_key(
        &self,
        brand_id: &str,
        authority_uid: &[u8],
        epoch: u32,
    ) -> Result<SecretKey16, KmsError> {
        let epoch_le = epoch.to_le_bytes();
        let uid_7: [u8; 7] = if authority_uid.len() >= 7 {
            authority_uid[..7].try_into().unwrap()
        } else {
            let mut padded = [0u8; 7];
            padded[..authority_uid.len()].copy_from_slice(authority_uid);
            padded
        };
        self.with_brand_key(brand_id, |brand_key| {
            kdf::derive_chip_key(brand_key, &uid_7, &epoch_le)
        })
    }
}

#[cfg(test)]
fn _assert_send_sync() {
    fn assert_impl<T: Send + Sync>() {}
    assert_impl::<SoftwareKms>();
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::provider::KeyProvider;
    use serial_test::serial;
    use std::sync::Arc;

    const ROOT_KEY_HEX: &str =
        "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";

    fn setup_env() {
        std::env::set_var("RC_ROOT_KEY_HEX", ROOT_KEY_HEX);
        std::env::set_var("RC_SYSTEM_ID", "test-system");
    }

    fn cleanup_env() {
        std::env::remove_var("RC_ROOT_KEY_HEX");
        std::env::remove_var("RC_SYSTEM_ID");
    }

    #[test]
    #[serial]
    fn test_from_env_success() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize from valid env");
        assert_eq!(kms.system_id, "test-system");
        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_from_env_missing_system_id() {
        std::env::set_var("RC_ROOT_KEY_HEX", ROOT_KEY_HEX);
        std::env::remove_var("RC_SYSTEM_ID");

        let result = SoftwareKms::from_env();
        assert!(result.is_err());
        let err = result.unwrap_err();
        assert!(
            matches!(err, KmsError::SystemIdMissing),
            "expected SystemIdMissing, got {:?}",
            err
        );

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_debug_redacted() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let debug_output = format!("{:?}", kms);
        assert!(
            debug_output.contains("REDACTED"),
            "Debug output should contain REDACTED, got: {}",
            debug_output
        );
        // Root Key hex bytes must not appear in debug output
        assert!(
            !debug_output.contains("000102"),
            "Debug output must not contain root key bytes"
        );
        assert!(
            !debug_output.contains("1e1f"),
            "Debug output must not contain root key bytes"
        );

        cleanup_env();
    }

    // --- Task 7 tests: KeyProvider derive_chip_key / derive_honey_key ---

    #[test]
    #[serial]
    fn test_chip_key_deterministic() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let epoch = 1u32;

        let k1 = kms.derive_chip_key("brand-A", &uid, epoch).unwrap();
        let k2 = kms.derive_chip_key("brand-A", &uid, epoch).unwrap();
        assert_eq!(k1.as_bytes(), k2.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_chip_key_uid_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let uid_a: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let uid_b: [u8; 7] = [0x04, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF];
        let epoch = 1u32;

        let k_a = kms.derive_chip_key("brand-A", &uid_a, epoch).unwrap();
        let k_b = kms.derive_chip_key("brand-A", &uid_b, epoch).unwrap();
        assert_ne!(k_a.as_bytes(), k_b.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_chip_key_epoch_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];

        let k1 = kms.derive_chip_key("brand-A", &uid, 1).unwrap();
        let k2 = kms.derive_chip_key("brand-A", &uid, 2).unwrap();
        assert_ne!(k1.as_bytes(), k2.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_honey_key_deterministic() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let serial = b"serial-001";

        let k1 = kms.derive_honey_key("brand-A", serial).unwrap();
        let k2 = kms.derive_honey_key("brand-A", serial).unwrap();
        assert_eq!(k1.as_bytes(), k2.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_honey_key_serial_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let k1 = kms.derive_honey_key("brand-A", b"serial-001").unwrap();
        let k2 = kms.derive_honey_key("brand-A", b"serial-002").unwrap();
        assert_ne!(k1.as_bytes(), k2.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_brand_cache_hit() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let epoch = 1u32;

        // First call — cache miss, derives brand key
        let k1 = kms.derive_chip_key("brand-A", &uid, epoch).unwrap();
        // Second call — cache hit
        let k2 = kms.derive_chip_key("brand-A", &uid, epoch).unwrap();
        assert_eq!(k1.as_bytes(), k2.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_brand_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let epoch = 1u32;

        let k_a = kms.derive_chip_key("brand-A", &uid, epoch).unwrap();
        let k_b = kms.derive_chip_key("brand-B", &uid, epoch).unwrap();
        assert_ne!(k_a.as_bytes(), k_b.as_bytes());

        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_concurrent_derive_chip_key() {
        setup_env();
        let kms = Arc::new(SoftwareKms::from_env().expect("should initialize"));
        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let epoch = 1u32;

        let mut handles = vec![];
        for i in 0..8 {
            let kms_clone = Arc::clone(&kms);
            let brand_id = if i % 2 == 0 { "brand-A" } else { "brand-B" };
            let brand_id_owned = brand_id.to_string();
            handles.push(std::thread::spawn(move || {
                kms_clone
                    .derive_chip_key(&brand_id_owned, &uid, epoch)
                    .unwrap()
            }));
        }

        let mut results_a = vec![];
        let mut results_b = vec![];
        for (i, handle) in handles.into_iter().enumerate() {
            let key = handle.join().expect("thread should not panic");
            if i % 2 == 0 {
                results_a.push(key);
            } else {
                results_b.push(key);
            }
        }

        // All same-brand results should be identical
        for k in &results_a {
            assert_eq!(k.as_bytes(), results_a[0].as_bytes());
        }
        for k in &results_b {
            assert_eq!(k.as_bytes(), results_b[0].as_bytes());
        }
        // Different brands should produce different keys
        assert_ne!(results_a[0].as_bytes(), results_b[0].as_bytes());

        cleanup_env();
    }

    // --- Task 9: 端到端链路测试 (Property 5) ---

    #[test]
    #[serial]
    fn test_kms_e2e_sun_verify() {
        setup_env();
        let kms = SoftwareKms::from_env().expect("should initialize");

        let brand_id = "brand-E2E";
        let uid: [u8; 7] = [0x04, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF];
        let epoch = 0u32;
        let ctr: [u8; 3] = [0x03, 0x00, 0x00];

        let chip_key = kms.derive_chip_key(brand_id, &uid, epoch).unwrap();

        // Build SUN message: [0x3C, 0xC3, uid(7), ctr(3)] = 12 bytes
        let mut sun_msg = [0u8; 12];
        sun_msg[0] = 0x3C;
        sun_msg[1] = 0xC3;
        sun_msg[2..9].copy_from_slice(&uid);
        sun_msg[9..12].copy_from_slice(&ctr);

        let valid_cmac =
            rc_crypto::cmac_aes128::compute_truncated(chip_key.as_bytes(), &sun_msg);

        // Verify with correct CMAC → true
        assert!(rc_crypto::sun::verify_sun_message(
            chip_key.as_bytes(),
            &uid,
            &ctr,
            &valid_cmac
        ));

        // Tamper one byte → false
        let mut tampered = valid_cmac;
        tampered[0] ^= 0xFF;
        assert!(!rc_crypto::sun::verify_sun_message(
            chip_key.as_bytes(),
            &uid,
            &ctr,
            &tampered
        ));

        cleanup_env();
    }

    // --- Task 12: derive_mother_key tests ---

    #[test]
    #[serial]
    fn test_mother_key_deterministic() {
        setup_env();
        let kms = SoftwareKms::from_env().unwrap();
        let authority_uid = b"mother01";
        let k1 = kms.derive_mother_key("brand-x", authority_uid, 1).unwrap();
        let k2 = kms.derive_mother_key("brand-x", authority_uid, 1).unwrap();
        assert_eq!(k1.as_bytes(), k2.as_bytes());
        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_mother_key_uid_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().unwrap();
        // UIDs must differ within the first 7 bytes (truncation boundary)
        let k1 = kms.derive_mother_key("brand-x", b"mthr_01", 1).unwrap();
        let k2 = kms.derive_mother_key("brand-x", b"mthr_02", 1).unwrap();
        assert_ne!(k1.as_bytes(), k2.as_bytes());
        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_mother_key_epoch_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().unwrap();
        let authority_uid = b"mother01";
        let k1 = kms.derive_mother_key("brand-x", authority_uid, 1).unwrap();
        let k2 = kms.derive_mother_key("brand-x", authority_uid, 2).unwrap();
        assert_ne!(k1.as_bytes(), k2.as_bytes());
        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_mother_key_chip_key_isolation() {
        setup_env();
        let kms = SoftwareKms::from_env().unwrap();
        let authority_uid = b"mother01"; // 8 bytes, truncated to first 7
        let chip_uid: [u8; 7] = [0x04, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06];
        let epoch = 1u32;
        let mother_key = kms.derive_mother_key("brand-x", authority_uid, epoch).unwrap();
        let chip_key = kms.derive_chip_key("brand-x", &chip_uid, epoch).unwrap();
        assert_ne!(mother_key.as_bytes(), chip_key.as_bytes());
        cleanup_env();
    }

    #[test]
    #[serial]
    fn test_mother_key_short_uid_zero_padded_contract() {
        setup_env();
        let kms = SoftwareKms::from_env().unwrap();
        let short_uid = b"abc";
        let mut padded = [0u8; 7];
        padded[..short_uid.len()].copy_from_slice(short_uid);

        let derived_from_short = kms.derive_mother_key("brand-x", short_uid, 7).unwrap();
        let derived_from_padded = kms.derive_chip_key("brand-x", &padded, 7).unwrap();

        assert_eq!(derived_from_short.as_bytes(), derived_from_padded.as_bytes());
        cleanup_env();
    }
}
