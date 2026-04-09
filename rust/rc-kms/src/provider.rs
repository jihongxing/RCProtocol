use rc_crypto::{SecretKey16, SecretKey32};
use crate::error::KmsError;

/// 密钥操作统一接口
///
/// Phase 1: SoftwareKms 实现（纯 CPU 计算）
/// Phase 2: CloudKms / HsmKms 实现（网络 IO，届时可能需要 async-trait）
pub trait KeyProvider: Send + Sync {
    /// 派生 Chip Key，用于 SUN CMAC 校验和标签认证
    fn derive_chip_key(
        &self,
        brand_id: &str,
        uid: &[u8; 7],
        epoch: u32,
    ) -> Result<SecretKey16, KmsError>;

    /// 派生 Honey Key，用于蜜标 / HBM 扩展校验
    fn derive_honey_key(
        &self,
        brand_id: &str,
        serial: &[u8],
    ) -> Result<SecretKey32, KmsError>;

    /// 派生 Mother Card Key，用于虚拟母卡凭证生成与校验
    ///
    /// - brand_id: 品牌标识
    /// - authority_uid: 母卡凭证 UID（虚拟母卡为系统生成标识，可变长度）
    /// - epoch: 密钥轮换周期标识
    fn derive_mother_key(
        &self,
        brand_id: &str,
        authority_uid: &[u8],
        epoch: u32,
    ) -> Result<SecretKey16, KmsError>;
}
