use rc_crypto::SecretKey32;
use zeroize::Zeroize;

use crate::error::KmsError;

/// 从 RC_ROOT_KEY_HEX 环境变量加载 Root Key
///
/// 安全约束：
/// - 原始 hex 字符串在解码后立即 zeroize
/// - 失败路径同样 zeroize hex 字符串
/// - 失败日志不包含密钥内容
pub(crate) fn load_root_key_from_env() -> Result<SecretKey32, KmsError> {
    let mut hex_str = match std::env::var("RC_ROOT_KEY_HEX") {
        Ok(val) => val,
        Err(_) => {
            tracing::error!("RC_ROOT_KEY_HEX environment variable is not set");
            return Err(KmsError::RootKeyMissing);
        }
    };

    if hex_str.len() != 64 {
        tracing::error!(
            "RC_ROOT_KEY_HEX has invalid length: expected 64, got {}",
            hex_str.len()
        );
        hex_str.zeroize();
        return Err(KmsError::RootKeyInvalidFormat);
    }

    let mut bytes = [0u8; 32];
    for i in 0..32 {
        let result = u8::from_str_radix(&hex_str[i * 2..i * 2 + 2], 16);
        if result.is_err() {
            tracing::error!("RC_ROOT_KEY_HEX contains invalid hex characters");
            hex_str.zeroize();
            return Err(KmsError::RootKeyInvalidFormat);
        }
        bytes[i] = result.unwrap();
    }

    hex_str.zeroize();

    Ok(SecretKey32::new(bytes))
}

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;

    #[test]
    #[serial]
    fn test_load_root_key_valid() {
        let hex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";
        std::env::set_var("RC_ROOT_KEY_HEX", hex);

        let key = load_root_key_from_env().expect("should load valid root key");

        let expected: [u8; 32] = [
            0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
            0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
            0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
            0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
        ];
        assert_eq!(key.as_bytes(), &expected);

        std::env::remove_var("RC_ROOT_KEY_HEX");
    }

    #[test]
    #[serial]
    fn test_load_root_key_missing() {
        std::env::remove_var("RC_ROOT_KEY_HEX");

        let result = load_root_key_from_env();
        assert!(result.is_err());

        let err = result.unwrap_err();
        assert!(
            matches!(err, KmsError::RootKeyMissing),
            "expected RootKeyMissing, got {:?}",
            err
        );
    }

    #[test]
    #[serial]
    fn test_load_root_key_invalid_length() {
        // 32 characters instead of 64
        std::env::set_var("RC_ROOT_KEY_HEX", "00112233445566778899aabbccddeeff");

        let result = load_root_key_from_env();
        assert!(result.is_err());

        let err = result.unwrap_err();
        assert!(
            matches!(err, KmsError::RootKeyInvalidFormat),
            "expected RootKeyInvalidFormat, got {:?}",
            err
        );

        std::env::remove_var("RC_ROOT_KEY_HEX");
    }

    #[test]
    #[serial]
    fn test_load_root_key_invalid_chars() {
        // 64 chars but contains "gg" which is not valid hex
        let bad_hex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1egg";
        assert_eq!(bad_hex.len(), 64);
        std::env::set_var("RC_ROOT_KEY_HEX", bad_hex);

        let result = load_root_key_from_env();
        assert!(result.is_err());

        let err = result.unwrap_err();
        assert!(
            matches!(err, KmsError::RootKeyInvalidFormat),
            "expected RootKeyInvalidFormat, got {:?}",
            err
        );

        std::env::remove_var("RC_ROOT_KEY_HEX");
    }

    #[test]
    #[serial]
    fn test_load_root_key_uppercase() {
        let hex = "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F";
        std::env::set_var("RC_ROOT_KEY_HEX", hex);

        let key = load_root_key_from_env().expect("should load uppercase hex");

        let expected: [u8; 32] = [
            0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
            0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
            0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
            0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
        ];
        assert_eq!(key.as_bytes(), &expected);

        std::env::remove_var("RC_ROOT_KEY_HEX");
    }
}
