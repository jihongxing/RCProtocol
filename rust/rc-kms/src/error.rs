use thiserror::Error;

#[derive(Debug, Error)]
pub enum KmsError {
    #[error("RC_ROOT_KEY_HEX environment variable is not set")]
    RootKeyMissing,

    #[error("RC_ROOT_KEY_HEX is not a valid 64-char hex string")]
    RootKeyInvalidFormat,

    #[error("RC_SYSTEM_ID environment variable is not set")]
    SystemIdMissing,

    #[error("brand key derivation failed")]
    BrandKeyDerivationFailed,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_display_no_secrets() {
        let cases: Vec<(KmsError, &str)> = vec![
            (
                KmsError::RootKeyMissing,
                "RC_ROOT_KEY_HEX environment variable is not set",
            ),
            (
                KmsError::RootKeyInvalidFormat,
                "RC_ROOT_KEY_HEX is not a valid 64-char hex string",
            ),
            (
                KmsError::SystemIdMissing,
                "RC_SYSTEM_ID environment variable is not set",
            ),
            (
                KmsError::BrandKeyDerivationFailed,
                "brand key derivation failed",
            ),
        ];

        // Patterns that must never appear in error messages
        let forbidden = [
            "0123456789abcdef0123456789abcdef", // example key material
            "secret",
            "key=",
            "password",
        ];

        for (err, expected_msg) in &cases {
            let display = err.to_string();
            assert_eq!(&display, *expected_msg);

            // Verify no key content leaks into display output
            for pattern in &forbidden {
                assert!(
                    !display.to_lowercase().contains(pattern),
                    "Error display for {:?} must not contain '{}'",
                    err,
                    pattern
                );
            }
        }
    }
}
