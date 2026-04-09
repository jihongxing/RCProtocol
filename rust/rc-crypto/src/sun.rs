use crate::cmac_aes128;
use crate::constant_time;

/// NTAG 424 DNA SUN Mode A 动态消息校验。
///
/// 输入消息构造：`[0x3C, 0xC3] || uid(7) || ctr(3)` = 12 字节
/// CMAC = AES-128-CMAC(K_chip, message)[..8]
pub fn verify_sun_message(
    key: &[u8; 16],
    uid: &[u8; 7],
    ctr: &[u8; 3],
    cmac_received: &[u8; 8],
) -> bool {
    // SV2 固定头 + UID + CTR = 12 字节
    let mut message = [0u8; 12];
    message[0] = 0x3C;
    message[1] = 0xC3;
    message[2..9].copy_from_slice(uid);
    message[9..12].copy_from_slice(ctr);

    let expected = cmac_aes128::compute_truncated(key, &message);
    constant_time::eq(&expected, cmac_received)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cmac_aes128;

    #[test]
    fn test_sun_valid_cmac() {
        let key: [u8; 16] = [0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
                              0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10];
        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let ctr: [u8; 3] = [0x01, 0x00, 0x00];

        // 构造消息并计算正确 CMAC
        let mut msg = [0u8; 12];
        msg[0] = 0x3C;
        msg[1] = 0xC3;
        msg[2..9].copy_from_slice(&uid);
        msg[9..12].copy_from_slice(&ctr);
        let valid_cmac = cmac_aes128::compute_truncated(&key, &msg);

        assert!(verify_sun_message(&key, &uid, &ctr, &valid_cmac));
    }

    #[test]
    fn test_sun_second_valid_vector() {
        let key: [u8; 16] = [0xAA; 16];
        let uid: [u8; 7] = [0xBB; 7];
        let ctr: [u8; 3] = [0x05, 0x00, 0x00];

        let mut msg = [0u8; 12];
        msg[0] = 0x3C;
        msg[1] = 0xC3;
        msg[2..9].copy_from_slice(&uid);
        msg[9..12].copy_from_slice(&ctr);
        let valid_cmac = cmac_aes128::compute_truncated(&key, &msg);

        assert!(verify_sun_message(&key, &uid, &ctr, &valid_cmac));
    }

    #[test]
    fn test_sun_tampered_cmac() {
        let key: [u8; 16] = [0x01; 16];
        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let ctr: [u8; 3] = [0x01, 0x00, 0x00];

        let mut msg = [0u8; 12];
        msg[0] = 0x3C;
        msg[1] = 0xC3;
        msg[2..9].copy_from_slice(&uid);
        msg[9..12].copy_from_slice(&ctr);
        let mut tampered_cmac = cmac_aes128::compute_truncated(&key, &msg);
        tampered_cmac[0] ^= 0xFF; // 篡改一个字节

        assert!(!verify_sun_message(&key, &uid, &ctr, &tampered_cmac));
    }

    #[test]
    fn test_sun_wrong_ctr() {
        let key: [u8; 16] = [0x01; 16];
        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let ctr: [u8; 3] = [0x01, 0x00, 0x00];

        let mut msg = [0u8; 12];
        msg[0] = 0x3C;
        msg[1] = 0xC3;
        msg[2..9].copy_from_slice(&uid);
        msg[9..12].copy_from_slice(&ctr);
        let valid_cmac = cmac_aes128::compute_truncated(&key, &msg);

        // 用不同的 CTR 验证
        let wrong_ctr: [u8; 3] = [0x02, 0x00, 0x00];
        assert!(!verify_sun_message(&key, &uid, &wrong_ctr, &valid_cmac));
    }

    #[test]
    fn test_sun_wrong_uid() {
        let key: [u8; 16] = [0x01; 16];
        let uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66];
        let ctr: [u8; 3] = [0x01, 0x00, 0x00];

        let mut msg = [0u8; 12];
        msg[0] = 0x3C;
        msg[1] = 0xC3;
        msg[2..9].copy_from_slice(&uid);
        msg[9..12].copy_from_slice(&ctr);
        let valid_cmac = cmac_aes128::compute_truncated(&key, &msg);

        // 用不同的 UID 验证
        let wrong_uid: [u8; 7] = [0x04, 0x11, 0x22, 0x33, 0x44, 0x55, 0x77];
        assert!(!verify_sun_message(&key, &wrong_uid, &ctr, &valid_cmac));
    }
}
