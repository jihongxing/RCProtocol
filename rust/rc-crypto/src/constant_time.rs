use subtle::ConstantTimeEq;

/// 常量时间字节切片比较。
/// 不等长时直接返回 false（长度本身不是秘密）。
/// 等长时使用 subtle::ConstantTimeEq 保证比较时间不随匹配位置变化。
pub fn eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    a.ct_eq(b).into()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_equal_slices() {
        assert!(eq(&[1, 2, 3], &[1, 2, 3]));
    }

    #[test]
    fn test_different_slices() {
        assert!(!eq(&[1, 2, 3], &[1, 2, 4]));
    }

    #[test]
    fn test_different_length() {
        assert!(!eq(&[1, 2], &[1, 2, 3]));
    }

    #[test]
    fn test_empty_slices() {
        assert!(eq(&[], &[]));
    }

    #[test]
    fn test_single_byte_diff() {
        assert!(!eq(&[0], &[1]));
    }
}
