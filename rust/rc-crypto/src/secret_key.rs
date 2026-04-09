use zeroize::{Zeroize, ZeroizeOnDrop};

/// 32 字节安全密钥容器，用于 HMAC 密钥/输出、Brand Key、Root Key。
/// Drop 时自动清零内存，Debug 输出不泄漏密钥内容。
#[derive(Zeroize, ZeroizeOnDrop)]
pub struct SecretKey32 {
    bytes: [u8; 32],
}

impl SecretKey32 {
    pub fn new(bytes: [u8; 32]) -> Self {
        Self { bytes }
    }

    /// 返回密钥的只读引用。
    ///
    /// # 安全约束
    /// 调用方禁止通过 `*key.as_bytes()` 解引用复制密钥到栈上。
    /// 栈上的副本不受 ZeroizeOnDrop 保护，会在内存中残留明文。
    /// 正确用法：始终以 `&[u8]` 引用形式传递，避免拷贝。
    pub fn as_bytes(&self) -> &[u8; 32] {
        &self.bytes
    }
}

impl std::fmt::Debug for SecretKey32 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("SecretKey32([REDACTED])")
    }
}

/// 16 字节安全密钥容器，用于 AES-128 密钥（K_chip）。
/// Drop 时自动清零内存，Debug 输出不泄漏密钥内容。
#[derive(Zeroize, ZeroizeOnDrop)]
pub struct SecretKey16 {
    bytes: [u8; 16],
}

impl SecretKey16 {
    pub fn new(bytes: [u8; 16]) -> Self {
        Self { bytes }
    }

    /// 返回密钥的只读引用。
    ///
    /// # 安全约束
    /// 调用方禁止通过 `*key.as_bytes()` 解引用复制密钥到栈上。
    /// 栈上的副本不受 ZeroizeOnDrop 保护，会在内存中残留明文。
    /// 正确用法：始终以 `&[u8]` 引用形式传递，避免拷贝。
    pub fn as_bytes(&self) -> &[u8; 16] {
        &self.bytes
    }
}

impl std::fmt::Debug for SecretKey16 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("SecretKey16([REDACTED])")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_secret_key32_debug_redacted() {
        let key = SecretKey32::new([0x41; 32]);
        let debug = format!("{:?}", key);
        assert!(debug.contains("REDACTED"), "Debug should contain REDACTED");
        assert!(!debug.contains("41"), "Debug should not contain key bytes");
    }

    #[test]
    fn test_secret_key16_debug_redacted() {
        let key = SecretKey16::new([0x42; 16]);
        let debug = format!("{:?}", key);
        assert!(debug.contains("REDACTED"), "Debug should contain REDACTED");
        assert!(!debug.contains("42"), "Debug should not contain key bytes");
    }

    #[test]
    fn test_secret_key_as_bytes() {
        let bytes32 = [0xAB; 32];
        let key32 = SecretKey32::new(bytes32);
        assert_eq!(key32.as_bytes(), &bytes32);

        let bytes16 = [0xCD; 16];
        let key16 = SecretKey16::new(bytes16);
        assert_eq!(key16.as_bytes(), &bytes16);
    }
}
