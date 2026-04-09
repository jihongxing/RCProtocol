use rc_common::errors::RcError;
use rc_kms::KeyProvider;
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use std::sync::Arc;

/// 授权证明：虚拟凭证或物理 NFC
#[derive(Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum AuthorityProof {
    VirtualToken {
        user_id: String,
        credential_token: String,
    },
    PhysicalNfc {
        uid: String,
        ctr: String,
        cmac: String,
    },
}

/// 授权校验结果
#[derive(Debug, Serialize)]
pub struct AuthorizationResult {
    pub authorized: bool,
    pub authority_type: String,
    pub risk_flags: Vec<String>,
}

/// 分层授权校验：虚拟卡或物理卡
pub async fn verify_authority(
    pool: &PgPool,
    kms: Arc<dyn KeyProvider + Send + Sync>,
    asset_id: &str,
    proof: AuthorityProof,
) -> Result<AuthorizationResult, RcError> {
    match proof {
        AuthorityProof::VirtualToken { user_id, credential_token } => {
            verify_virtual_authority(pool, kms, asset_id, &user_id, &credential_token).await
        }
        AuthorityProof::PhysicalNfc { uid, ctr, cmac } => {
            verify_physical_authority(pool, kms, asset_id, &uid, &ctr, &cmac).await
        }
    }
}

/// 虚拟卡授权校验
async fn verify_virtual_authority(
    pool: &PgPool,
    kms: Arc<dyn KeyProvider + Send + Sync>,
    asset_id: &str,
    user_id: &str,
    credential_token: &str,
) -> Result<AuthorizationResult, RcError> {
    // 查询资产绑定的母卡设备
    let authority_device = crate::db::authority_devices::fetch_authority_device_by_asset(pool, asset_id).await?;

    // 校验设备类型
    if authority_device.authority_type != "VIRTUAL_APP" {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["wrong_authority_type".to_string()],
        });
    }

    // 校验设备状态
    if authority_device.status != "Active" {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["authority_device_inactive".to_string()],
        });
    }

    // 校验绑定用户
    match &authority_device.bound_user_id {
        Some(bound_user) if bound_user == user_id => {}
        Some(_) => {
            return Ok(AuthorizationResult {
                authorized: false,
                authority_type: authority_device.authority_type,
                risk_flags: vec!["user_mismatch".to_string()],
            });
        }
        None => {
            return Ok(AuthorizationResult {
                authorized: false,
                authority_type: authority_device.authority_type,
                risk_flags: vec!["no_bound_user".to_string()],
            });
        }
    }

    // 重新派生 K_chip_mother 并计算凭证哈希
    let k_chip_mother = kms
        .derive_mother_key(
            &authority_device.brand_id,
            authority_device.authority_uid.as_bytes(),
            authority_device.key_epoch as u32,
        )
        .map_err(|err| RcError::Database(format!("KMS derive_mother_key failed: {err}")))?;

    let computed_hash_bytes = rc_crypto::hmac_sha256::compute(
        k_chip_mother.as_bytes(),
        authority_device.authority_uid.as_bytes(),
    );
    let computed_hash = hex::encode(computed_hash_bytes);
    // k_chip_mother 在此作用域结束后 drop → ZeroizeOnDrop

    // 比对凭证哈希
    let stored_hash = authority_device
        .virtual_credential_hash
        .ok_or_else(|| RcError::Database("virtual_credential_hash is NULL".into()))?;

    if credential_token != stored_hash {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["credential_mismatch".to_string()],
        });
    }

    // 额外校验：重新计算的哈希应与存储的哈希一致（防止数据库篡改）
    if computed_hash != stored_hash {
        tracing::warn!(
            asset_id = %asset_id,
            authority_uid = %authority_device.authority_uid,
            "stored credential hash does not match recomputed hash — possible database tampering"
        );
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["credential_integrity_failure".to_string()],
        });
    }

    Ok(AuthorizationResult {
        authorized: true,
        authority_type: authority_device.authority_type,
        risk_flags: vec![],
    })
}

/// 解析 CTR hex 字符串为 u32 (24-bit little-endian)
fn parse_ctr(ctr_hex: &str) -> Result<u32, RcError> {
    // 验证长度为 6 字符（3 bytes）
    if ctr_hex.len() != 6 {
        return Err(RcError::InvalidInput("ctr must be 6 hex characters".into()));
    }

    // 解码 hex 为 3 bytes
    let ctr_bytes = hex::decode(ctr_hex)
        .map_err(|_| RcError::InvalidInput("invalid ctr hex".into()))?;

    if ctr_bytes.len() != 3 {
        return Err(RcError::InvalidInput("ctr must be 3 bytes".into()));
    }

    // 按 little-endian 顺序转换为 u32 (24-bit)
    let ctr_value = u32::from_le_bytes([ctr_bytes[0], ctr_bytes[1], ctr_bytes[2], 0]);

    Ok(ctr_value)
}

/// 物理卡授权校验：复用验真逻辑
async fn verify_physical_authority(
    pool: &PgPool,
    kms: Arc<dyn KeyProvider + Send + Sync>,
    asset_id: &str,
    uid_hex: &str,
    ctr_hex: &str,
    cmac_hex: &str,
) -> Result<AuthorizationResult, RcError> {
    // 查询资产绑定的母卡设备
    let authority_device = crate::db::authority_devices::fetch_authority_device_by_asset(pool, asset_id).await?;

    // 校验设备类型
    if authority_device.authority_type != "PHYSICAL_NFC" {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["wrong_authority_type".to_string()],
        });
    }

    // 校验设备状态
    if authority_device.status != "Active" {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["authority_device_inactive".to_string()],
        });
    }

    // 校验 physical_chip_uid 存在
    let physical_chip_uid = match &authority_device.physical_chip_uid {
        Some(uid) => uid,
        None => {
            return Ok(AuthorizationResult {
                authorized: false,
                authority_type: authority_device.authority_type,
                risk_flags: vec!["missing_physical_chip_uid".to_string()],
            });
        }
    };

    // 校验 UID 匹配（使用 physical_chip_uid，case-insensitive）
    if uid_hex.to_uppercase() != physical_chip_uid.to_uppercase() {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["uid_mismatch".to_string()],
        });
    }

    // 解析 CTR
    let new_ctr = match parse_ctr(ctr_hex) {
        Ok(ctr) => ctr,
        Err(_) => {
            return Ok(AuthorizationResult {
                authorized: false,
                authority_type: authority_device.authority_type,
                risk_flags: vec!["invalid_ctr_format".to_string()],
            });
        }
    };

    // CTR 防重放检测（fail fast）
    if let Some(last_ctr) = authority_device.last_known_ctr {
        if new_ctr as i32 <= last_ctr {
            return Ok(AuthorizationResult {
                authorized: false,
                authority_type: authority_device.authority_type,
                risk_flags: vec!["ctr_replay".to_string()],
            });
        }
    }
    // 如果 last_known_ctr 为 NULL，接受任意 CTR（首次使用）

    // 解析 hex 参数
    let uid_bytes = hex::decode(uid_hex)
        .map_err(|_| RcError::InvalidInput("invalid uid hex".into()))?;
    let ctr_bytes = hex::decode(ctr_hex)
        .map_err(|_| RcError::InvalidInput("invalid ctr hex".into()))?;
    let cmac_bytes = hex::decode(cmac_hex)
        .map_err(|_| RcError::InvalidInput("invalid cmac hex".into()))?;

    if uid_bytes.len() != 7 {
        return Err(RcError::InvalidInput("uid must be 7 bytes".into()));
    }
    if ctr_bytes.len() != 3 {
        return Err(RcError::InvalidInput("ctr must be 3 bytes".into()));
    }
    if cmac_bytes.len() != 8 {
        return Err(RcError::InvalidInput("cmac must be 8 bytes".into()));
    }

    let uid: [u8; 7] = uid_bytes.try_into().unwrap();
    let ctr: [u8; 3] = ctr_bytes.try_into().unwrap();
    let cmac: [u8; 8] = cmac_bytes.try_into().unwrap();

    // 派生 K_chip 并验证 CMAC
    let k_chip = kms
        .derive_chip_key(
            &authority_device.brand_id,
            &uid,
            authority_device.key_epoch as u32,
        )
        .map_err(|err| RcError::Database(format!("KMS derive_chip_key failed: {err}")))?;

    let cmac_valid = rc_crypto::sun::verify_sun_message(k_chip.as_bytes(), &uid, &ctr, &cmac);
    // k_chip 在此作用域结束后 drop → ZeroizeOnDrop

    if !cmac_valid {
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["cmac_invalid".to_string()],
        });
    }

    // 原子更新 CTR（防止并发重放攻击）
    let ctr_updated = crate::db::authority_devices::atomic_update_ctr(
        pool,
        authority_device.authority_id,
        new_ctr as i32,
    )
    .await?;

    if !ctr_updated {
        // 并发冲突或 CTR 不递增
        return Ok(AuthorizationResult {
            authorized: false,
            authority_type: authority_device.authority_type,
            risk_flags: vec!["ctr_replay".to_string()],
        });
    }

    Ok(AuthorizationResult {
        authorized: true,
        authority_type: authority_device.authority_type,
        risk_flags: vec![],
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_ctr_valid() {
        // 测试正确解析 little-endian 3-byte CTR
        assert_eq!(parse_ctr("010000").unwrap(), 1);
        assert_eq!(parse_ctr("000100").unwrap(), 256);
        assert_eq!(parse_ctr("000001").unwrap(), 65536);
        assert_eq!(parse_ctr("010203").unwrap(), 0x030201);
    }

    #[test]
    fn test_parse_ctr_boundary_values() {
        // 测试边界值
        assert_eq!(parse_ctr("000000").unwrap(), 0);
        assert_eq!(parse_ctr("FFFFFF").unwrap(), 0xFFFFFF);
        assert_eq!(parse_ctr("ffffff").unwrap(), 0xFFFFFF); // lowercase
    }

    #[test]
    fn test_parse_ctr_invalid_length() {
        // 测试拒绝无效长度
        assert!(parse_ctr("0100").is_err());
        assert!(parse_ctr("01000000").is_err());
        assert!(parse_ctr("").is_err());
    }

    #[test]
    fn test_parse_ctr_invalid_hex() {
        // 测试拒绝非 hex 字符
        assert!(parse_ctr("GGGGGG").is_err());
        assert!(parse_ctr("01000Z").is_err());
        assert!(parse_ctr("01 000").is_err());
    }

    #[test]
    fn test_authority_proof_deserialization() {
        let virtual_json = r#"{"type":"virtual_token","user_id":"user-001","credential_token":"abc123"}"#;
        let virtual_proof: AuthorityProof = serde_json::from_str(virtual_json).unwrap();
        match virtual_proof {
            AuthorityProof::VirtualToken { user_id, credential_token } => {
                assert_eq!(user_id, "user-001");
                assert_eq!(credential_token, "abc123");
            }
            _ => panic!("expected VirtualToken"),
        }

        let physical_json = r#"{"type":"physical_nfc","uid":"04A31B2C3D4E5F","ctr":"010000","cmac":"0102030405060708"}"#;
        let physical_proof: AuthorityProof = serde_json::from_str(physical_json).unwrap();
        match physical_proof {
            AuthorityProof::PhysicalNfc { uid, ctr, cmac } => {
                assert_eq!(uid, "04A31B2C3D4E5F");
                assert_eq!(ctr, "010000");
                assert_eq!(cmac, "0102030405060708");
            }
            _ => panic!("expected PhysicalNfc"),
        }
    }
}
