use chrono::{DateTime, Utc};
use rc_common::errors::RcError;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sqlx::types::JsonValue;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssetCommitmentPayloadV1 {
    pub version: String,
    pub brand_id: String,
    pub asset_uid: String,
    pub chip_binding: String,
    pub epoch: u32,
    pub metadata_hash: String,
}

#[derive(Debug, Clone)]
pub struct AssetCommitmentRecord {
    pub commitment_id: String,
    pub payload_version: String,
    pub brand_id: String,
    pub asset_uid: String,
    pub chip_binding: String,
    pub epoch: i32,
    pub metadata_hash: String,
    pub canonical_payload: JsonValue,
    pub created_at: DateTime<Utc>,
}

pub fn normalize_uid(uid: &str) -> String {
    uid.trim().to_ascii_uppercase()
}

pub fn build_chip_binding(uid: &str, epoch: u32) -> String {
    let normalized_uid = normalize_uid(uid);
    let input = format!("{}|ntag424dna|{}", normalized_uid, epoch);
    hex_sha256(input.as_bytes())
}

pub fn build_metadata_hash(
    external_product_id: &str,
    external_product_name: Option<&str>,
    external_product_url: Option<&str>,
    batch_id: Option<&str>,
) -> String {
    let payload = serde_json::json!({
        "batch_id": batch_id.map(|v| v.trim()),
        "external_product_id": external_product_id.trim(),
        "external_product_name": external_product_name.map(|v| v.trim()),
        "external_product_url": external_product_url.map(|v| v.trim()),
    });

    let bytes = serde_json::to_vec(&payload).expect("metadata hash payload serialization");
    hex_sha256(&bytes)
}

pub fn build_asset_commitment_payload(
    brand_id: &str,
    asset_uid: &str,
    epoch: u32,
    metadata_hash: String,
) -> AssetCommitmentPayloadV1 {
    let normalized_uid = normalize_uid(asset_uid);
    let chip_binding = build_chip_binding(&normalized_uid, epoch);

    AssetCommitmentPayloadV1 {
        version: "ac_v1".to_string(),
        brand_id: brand_id.trim().to_string(),
        asset_uid: normalized_uid,
        chip_binding,
        epoch,
        metadata_hash,
    }
}

pub fn canonical_payload_json(payload: &AssetCommitmentPayloadV1) -> JsonValue {
    serde_json::json!({
        "asset_uid": payload.asset_uid,
        "brand_id": payload.brand_id,
        "chip_binding": payload.chip_binding,
        "epoch": payload.epoch,
        "metadata_hash": payload.metadata_hash,
        "version": payload.version,
    })
}

pub fn compute_asset_commitment_id(payload: &AssetCommitmentPayloadV1) -> Result<String, RcError> {
    let canonical = canonical_payload_json(payload);
    let bytes = serde_json::to_vec(&canonical)
        .map_err(|err| RcError::Database(format!("serialize asset commitment payload: {err}")))?;
    Ok(hex_sha256(&bytes))
}

pub fn build_asset_commitment_record(
    brand_id: &str,
    asset_uid: &str,
    epoch: u32,
    external_product_id: &str,
    external_product_name: Option<&str>,
    external_product_url: Option<&str>,
    batch_id: Option<&str>,
) -> Result<AssetCommitmentRecord, RcError> {
    let metadata_hash = build_metadata_hash(
        external_product_id,
        external_product_name,
        external_product_url,
        batch_id,
    );
    let payload = build_asset_commitment_payload(brand_id, asset_uid, epoch, metadata_hash);
    let commitment_id = compute_asset_commitment_id(&payload)?;
    let canonical_payload = canonical_payload_json(&payload);

    Ok(AssetCommitmentRecord {
        commitment_id,
        payload_version: payload.version,
        brand_id: payload.brand_id,
        asset_uid: payload.asset_uid,
        chip_binding: payload.chip_binding,
        epoch: payload.epoch as i32,
        metadata_hash: payload.metadata_hash,
        canonical_payload,
        created_at: Utc::now(),
    })
}

fn hex_sha256(input: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input);
    hex::encode(hasher.finalize())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalizes_uid_to_uppercase() {
        assert_eq!(normalize_uid(" 04a31b2c3d4e5f "), "04A31B2C3D4E5F");
    }

    #[test]
    fn metadata_hash_is_stable_for_same_inputs() {
        let h1 = build_metadata_hash("sku_001", Some(" Product "), Some("https://a"), Some("batch_1"));
        let h2 = build_metadata_hash("sku_001", Some("Product"), Some("https://a"), Some("batch_1"));
        assert_eq!(h1, h2);
    }

    #[test]
    fn chip_binding_changes_when_epoch_changes() {
        let a = build_chip_binding("04A31B2C3D4E5F", 0);
        let b = build_chip_binding("04A31B2C3D4E5F", 1);
        assert_ne!(a, b);
    }

    #[test]
    fn commitment_hash_is_stable_for_same_inputs() {
        let a = build_asset_commitment_record(
            "brand_1",
            "04A31B2C3D4E5F",
            0,
            "sku_001",
            Some("Product A"),
            Some("https://example.com/p/sku_001"),
            Some("batch_1"),
        )
        .unwrap();
        let b = build_asset_commitment_record(
            "brand_1",
            "04a31b2c3d4e5f",
            0,
            "sku_001",
            Some(" Product A "),
            Some("https://example.com/p/sku_001"),
            Some("batch_1"),
        )
        .unwrap();
        assert_eq!(a.commitment_id, b.commitment_id);
        assert_eq!(a.canonical_payload, b.canonical_payload);
    }

    #[test]
    fn commitment_changes_when_metadata_changes() {
        let a = build_asset_commitment_record(
            "brand_1",
            "04A31B2C3D4E5F",
            0,
            "sku_001",
            Some("Product A"),
            None,
            None,
        )
        .unwrap();
        let b = build_asset_commitment_record(
            "brand_1",
            "04A31B2C3D4E5F",
            0,
            "sku_002",
            Some("Product A"),
            None,
            None,
        )
        .unwrap();
        assert_ne!(a.commitment_id, b.commitment_id);
    }

    #[test]
    fn commitment_changes_when_chip_binding_changes() {
        let a = build_asset_commitment_record(
            "brand_1",
            "04A31B2C3D4E5F",
            0,
            "sku_001",
            Some("Product A"),
            None,
            None,
        )
        .unwrap();
        let b = build_asset_commitment_record(
            "brand_1",
            "04A31B2C3D4E5F",
            1,
            "sku_001",
            Some("Product A"),
            None,
            None,
        )
        .unwrap();
        assert_ne!(a.chip_binding, b.chip_binding);
        assert_ne!(a.commitment_id, b.commitment_id);
    }
}
