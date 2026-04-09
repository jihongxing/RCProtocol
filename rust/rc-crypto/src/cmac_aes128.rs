use aes::Aes128;
use cmac::{Cmac, Mac};

type CmacAes128 = Cmac<Aes128>;

/// 计算 AES-128 CMAC 完整 16 字节输出。
pub fn compute_full(key: &[u8; 16], message: &[u8]) -> [u8; 16] {
    let mut mac = CmacAes128::new_from_slice(key)
        .expect("AES-128 key is exactly 16 bytes");
    mac.update(message);
    let result = mac.finalize();
    result.into_bytes().into()
}

/// 计算 AES-128 CMAC 截断为 8 字节（NTAG 424 DNA SUN 格式）。
/// 按 NXP AN12196 规范：取奇数索引字节 truncated[i] = full[i*2+1]
pub fn compute_truncated(key: &[u8; 16], message: &[u8]) -> [u8; 8] {
    let full = compute_full(key, message);
    let mut truncated = [0u8; 8];
    for i in 0..8 {
        truncated[i] = full[i * 2 + 1];
    }
    truncated
}

#[cfg(test)]
mod tests {
    use super::*;

    fn hex_to_bytes(hex: &str) -> Vec<u8> {
        (0..hex.len())
            .step_by(2)
            .map(|i| u8::from_str_radix(&hex[i..i + 2], 16).unwrap())
            .collect()
    }

    // NIST SP 800-38B AES-128 Example: empty message
    #[test]
    fn test_nist_800_38b_empty_message() {
        let key_bytes = hex_to_bytes("2b7e151628aed2a6abf7158809cf4f3c");
        let key: [u8; 16] = key_bytes.try_into().unwrap();
        let expected = hex_to_bytes("bb1d6929e95937287fa37d129b756746");
        assert_eq!(compute_full(&key, &[]), expected.as_slice());
    }

    // NIST SP 800-38B AES-128 Example: 16-byte message
    #[test]
    fn test_nist_800_38b_16byte_message() {
        let key_bytes = hex_to_bytes("2b7e151628aed2a6abf7158809cf4f3c");
        let key: [u8; 16] = key_bytes.try_into().unwrap();
        let msg = hex_to_bytes("6bc1bee22e409f96e93d7e117393172a");
        let expected = hex_to_bytes("070a16b46b4d4144f79bdd9dd04a287c");
        assert_eq!(compute_full(&key, &msg), expected.as_slice());
    }

    #[test]
    fn test_truncated_matches_nxp_an12196_odd_indices() {
        let key = [0x01_u8; 16];
        let msg = b"test message for truncation";
        let full = compute_full(&key, msg);
        let truncated = compute_truncated(&key, msg);
        // NXP AN12196: truncated[i] = full[i*2+1]
        for i in 0..8 {
            assert_eq!(truncated[i], full[i * 2 + 1], "index {i} mismatch");
        }
    }

    // ── Preservation 3.3: CMAC 校验通过时继续返回完整信息 ──
    // 验证 compute_full 对相同输入始终产生相同的确定性输出
    #[test]
    fn preservation_3_3_cmac_full_deterministic() {
        let key = [0x42_u8; 16];
        let msg = b"preservation test message";
        let r1 = compute_full(&key, msg);
        let r2 = compute_full(&key, msg);
        assert_eq!(r1, r2, "相同输入的 CMAC 结果应确定性相同");
    }

    // 验证 compute_full 返回 16 字节
    #[test]
    fn preservation_3_3_cmac_full_output_length() {
        let key = [0x01_u8; 16];
        let msg = b"test";
        let result = compute_full(&key, msg);
        assert_eq!(result.len(), 16, "CMAC full 输出应为 16 字节");
    }

    // 验证 compute_truncated 返回 8 字节
    #[test]
    fn preservation_3_3_cmac_truncated_output_length() {
        let key = [0x01_u8; 16];
        let msg = b"test";
        let result = compute_truncated(&key, msg);
        assert_eq!(result.len(), 8, "CMAC truncated 输出应为 8 字节");
    }

    // 验证不同密钥产生不同 CMAC
    #[test]
    fn preservation_3_3_different_keys_different_cmac() {
        let key1 = [0x01_u8; 16];
        let key2 = [0x02_u8; 16];
        let msg = b"same message";
        let r1 = compute_full(&key1, msg);
        let r2 = compute_full(&key2, msg);
        assert_ne!(r1, r2, "不同密钥的 CMAC 应不同");
    }

    // 验证不同消息产生不同 CMAC
    #[test]
    fn preservation_3_3_different_messages_different_cmac() {
        let key = [0x01_u8; 16];
        let r1 = compute_full(&key, b"message A");
        let r2 = compute_full(&key, b"message B");
        assert_ne!(r1, r2, "不同消息的 CMAC 应不同");
    }

    // 验证 NIST 向量仍然通过（基线不被破坏）
    #[test]
    fn preservation_3_3_nist_vectors_still_pass() {
        let key_bytes = hex_to_bytes("2b7e151628aed2a6abf7158809cf4f3c");
        let key: [u8; 16] = key_bytes.try_into().unwrap();

        // empty message
        let expected_empty = hex_to_bytes("bb1d6929e95937287fa37d129b756746");
        assert_eq!(compute_full(&key, &[]), expected_empty.as_slice());

        // 16-byte message
        let msg = hex_to_bytes("6bc1bee22e409f96e93d7e117393172a");
        let expected_16 = hex_to_bytes("070a16b46b4d4144f79bdd9dd04a287c");
        assert_eq!(compute_full(&key, &msg), expected_16.as_slice());
    }

    // ── BUG 1.15: CMAC 截断算法不符合 NXP AN12196 规范 ──
    // Bug: compute_truncated 取 full[..8]（前 8 字节），
    //      NXP AN12196 规范要求取奇数索引字节: truncated[i] = full[i*2+1] for i in 0..8
    // 期望行为: 截断结果应与 NXP 规范一致（取偶数索引字节）。
    //
    // **Validates: Requirements 1.15**
    #[test]
    fn bug_1_15_cmac_truncation_should_match_nxp_an12196() {
        let key = [0x01_u8; 16];
        let msg = b"test message for truncation";
        let full = compute_full(&key, msg);

        // NXP AN12196 规范: truncated[i] = full[i*2+1]
        let mut nxp_truncated = [0u8; 8];
        for i in 0..8 {
            nxp_truncated[i] = full[i * 2 + 1];
        }

        let actual_truncated = compute_truncated(&key, msg);

        assert_eq!(
            actual_truncated, nxp_truncated,
            "BUG 1.15: compute_truncated 使用 full[..8]，不符合 NXP AN12196 规范 (应取 full[i*2+1])"
        );
    }
}
