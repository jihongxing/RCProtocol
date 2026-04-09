use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

/// 计算 HMAC-SHA256，返回 32 字节结果。
/// 密钥可以是任意长度，消息可以是任意长度。
pub fn compute(key: &[u8], message: &[u8]) -> [u8; 32] {
    let mut mac = HmacSha256::new_from_slice(key)
        .expect("HMAC can take key of any size");
    mac.update(message);
    let result = mac.finalize();
    result.into_bytes().into()
}

#[cfg(test)]
mod tests {
    use super::*;

    // RFC 4231 Test Case 1
    #[test]
    fn test_rfc4231_tc1() {
        let key = [0x0b_u8; 20];
        let data = b"Hi There";
        let expected = hex_to_bytes("b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7");
        assert_eq!(compute(&key, data), expected.as_slice());
    }

    // RFC 4231 Test Case 2
    #[test]
    fn test_rfc4231_tc2() {
        let key = b"Jefe";
        let data = b"what do ya want for nothing?";
        let expected = hex_to_bytes("5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843");
        assert_eq!(compute(key, data), expected.as_slice());
    }

    // RFC 4231 Test Case 3
    #[test]
    fn test_rfc4231_tc3() {
        let key = [0xaa_u8; 20];
        let data = [0xdd_u8; 50];
        let expected = hex_to_bytes("773ea91e36800e46854db8ebd09181a72959098b3ef8c122d9635514ced565fe");
        assert_eq!(compute(&key, &data), expected.as_slice());
    }

    fn hex_to_bytes(hex: &str) -> Vec<u8> {
        (0..hex.len())
            .step_by(2)
            .map(|i| u8::from_str_radix(&hex[i..i + 2], 16).unwrap())
            .collect()
    }
}
