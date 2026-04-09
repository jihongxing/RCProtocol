use zeroize::Zeroize;
use crate::hmac_sha256;
use crate::secret_key::{SecretKey16, SecretKey32};

/// 派生 Brand Key: HMAC-SHA256(Root_Key, Brand_ID || System_ID)
pub fn derive_brand_key(
    root_key: &SecretKey32,
    brand_id: &[u8],
    system_id: &[u8],
) -> SecretKey32 {
    let mut msg = Vec::with_capacity(brand_id.len() + system_id.len());
    msg.extend_from_slice(brand_id);
    msg.extend_from_slice(system_id);

    let result = hmac_sha256::compute(root_key.as_bytes(), &msg);
    msg.zeroize();

    SecretKey32::new(result)
}

/// 派生 Chip Key: HMAC-SHA256(Brand_Key, UID || Epoch_LE)[..16]
/// 关键：原始 32 字节 HMAC 结果在截断后立即 zeroize 清零。
pub fn derive_chip_key(
    brand_key: &SecretKey32,
    uid: &[u8],
    epoch_le: &[u8],
) -> SecretKey16 {
    let mut msg = Vec::with_capacity(uid.len() + epoch_le.len());
    msg.extend_from_slice(uid);
    msg.extend_from_slice(epoch_le);

    let mut full = hmac_sha256::compute(brand_key.as_bytes(), &msg);
    msg.zeroize();

    let mut key_bytes = [0u8; 16];
    key_bytes.copy_from_slice(&full[..16]);
    full.zeroize();

    SecretKey16::new(key_bytes)
}

/// 派生 Honey Key: HMAC-SHA256(Brand_Key, Serial_BE)
pub fn derive_honey_key(
    brand_key: &SecretKey32,
    serial_be: &[u8],
) -> SecretKey32 {
    let result = hmac_sha256::compute(brand_key.as_bytes(), serial_be);
    SecretKey32::new(result)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sun;

    #[test]
    fn test_kdf_deterministic() {
        let root = SecretKey32::new([0x11; 32]);
        let bk1 = derive_brand_key(&root, b"brand-A", b"sys-1");
        let bk2 = derive_brand_key(&root, b"brand-A", b"sys-1");
        assert_eq!(bk1.as_bytes(), bk2.as_bytes());
    }

    #[test]
    fn test_kdf_brand_isolation() {
        let root = SecretKey32::new([0x22; 32]);
        let bk_a = derive_brand_key(&root, b"brand-A", b"sys-1");
        let bk_b = derive_brand_key(&root, b"brand-B", b"sys-1");
        assert_ne!(bk_a.as_bytes(), bk_b.as_bytes());
    }

    #[test]
    fn test_kdf_chip_key_truncation() {
        let root = SecretKey32::new([0x33; 32]);
        let brand_key = derive_brand_key(&root, b"brand-X", b"sys-X");

        let uid = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let epoch_le = 1u32.to_le_bytes();

        // 手动计算期望值
        let mut msg = Vec::new();
        msg.extend_from_slice(&uid);
        msg.extend_from_slice(&epoch_le);
        let full = crate::hmac_sha256::compute(brand_key.as_bytes(), &msg);
        let expected: [u8; 16] = full[..16].try_into().unwrap();

        let chip_key = derive_chip_key(&brand_key, &uid, &epoch_le);
        assert_eq!(chip_key.as_bytes(), &expected);
    }

    #[test]
    fn test_kdf_e2e_sun_verify() {
        // Root Key → Brand Key → Chip Key → SUN CMAC → verify
        let root = SecretKey32::new([0x44; 32]);
        let brand_key = derive_brand_key(&root, b"brand-E2E", b"sys-E2E");

        let uid: [u8; 7] = [0x04, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF];
        let epoch_le = 0u32.to_le_bytes();
        let chip_key = derive_chip_key(&brand_key, &uid, &epoch_le);

        let ctr: [u8; 3] = [0x03, 0x00, 0x00];

        // 用 chip_key 计算正确的 SUN CMAC
        let mut sun_msg = [0u8; 12];
        sun_msg[0] = 0x3C;
        sun_msg[1] = 0xC3;
        sun_msg[2..9].copy_from_slice(&uid);
        sun_msg[9..12].copy_from_slice(&ctr);
        let valid_cmac = crate::cmac_aes128::compute_truncated(chip_key.as_bytes(), &sun_msg);

        // 验证 SUN 校验通过
        assert!(sun::verify_sun_message(chip_key.as_bytes(), &uid, &ctr, &valid_cmac));
    }
}
