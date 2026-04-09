use std::env;

use chrono::{DateTime, Utc};
use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use rc_common::errors::RcError;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sqlx::types::JsonValue;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlatformAttestationPayloadV1 {
    pub version: String,
    pub platform_id: String,
    pub asset_commitment_id: String,
    pub statement: String,
    pub issued_at: String,
    pub key_id: String,
}

#[derive(Debug, Clone)]
pub struct PlatformAttestationRecord {
    pub attestation_id: String,
    pub version: String,
    pub platform_id: String,
    pub asset_commitment_id: String,
    pub statement: String,
    pub key_id: String,
    pub canonical_payload: JsonValue,
    pub signature: String,
    pub issued_at: DateTime<Utc>,
}

pub fn build_platform_attestation_payload(
    platform_id: &str,
    asset_commitment_id: &str,
    issued_at: DateTime<Utc>,
    key_id: &str,
) -> PlatformAttestationPayloadV1 {
    PlatformAttestationPayloadV1 {
        version: "pa_v1".to_string(),
        platform_id: platform_id.trim().to_string(),
        asset_commitment_id: asset_commitment_id.trim().to_string(),
        statement: "platform_accepts_asset".to_string(),
        issued_at: issued_at.to_rfc3339(),
        key_id: key_id.trim().to_string(),
    }
}

pub fn canonical_payload_json(payload: &PlatformAttestationPayloadV1) -> JsonValue {
    serde_json::json!({
        "asset_commitment_id": payload.asset_commitment_id,
        "issued_at": payload.issued_at,
        "key_id": payload.key_id,
        "platform_id": payload.platform_id,
        "statement": payload.statement,
        "version": payload.version,
    })
}

pub fn canonical_payload_bytes(payload: &PlatformAttestationPayloadV1) -> Result<Vec<u8>, RcError> {
    serde_json::to_vec(&canonical_payload_json(payload))
        .map_err(|err| RcError::Database(format!("serialize platform attestation payload: {err}")))
}

fn derive_signing_key(secret: &str) -> Result<SigningKey, RcError> {
    let trimmed = secret.trim();
    let candidate_bytes = if trimmed.len() == 64 && trimmed.chars().all(|c| c.is_ascii_hexdigit()) {
        hex::decode(trimmed)
            .map_err(|err| RcError::InvalidInput(format!("invalid platform attestation secret hex: {err}")))?
    } else {
        let digest = Sha256::digest(trimmed.as_bytes());
        digest.to_vec()
    };

    let secret_key: [u8; 32] = candidate_bytes
        .try_into()
        .map_err(|_| RcError::InvalidInput("platform attestation secret must resolve to 32 bytes".to_string()))?;

    Ok(SigningKey::from_bytes(&secret_key))
}

pub fn load_platform_signing_key() -> Result<(String, String, SigningKey), RcError> {
    let platform_id = env::var("RC_PLATFORM_ID")
        .or_else(|_| env::var("RC_SYSTEM_ID"))
        .unwrap_or_else(|_| "rcprotocol-main".to_string());
    let key_id = env::var("RC_PLATFORM_ATTESTATION_KEY_ID")
        .unwrap_or_else(|_| "platform-key-2026-01".to_string());
    let secret = env::var("RC_PLATFORM_ATTESTATION_SECRET")
        .unwrap_or_else(|_| "rc-platform-attestation-dev-secret".to_string());
    let signing_key = derive_signing_key(&secret)?;
    Ok((platform_id, key_id, signing_key))
}

pub fn sign_platform_attestation(
    payload: &PlatformAttestationPayloadV1,
    signing_key: &SigningKey,
) -> Result<String, RcError> {
    let bytes = canonical_payload_bytes(payload)?;
    let signature: Signature = signing_key.sign(&bytes);
    Ok(hex::encode(signature.to_bytes()))
}

pub fn verify_platform_attestation(
    payload: &PlatformAttestationPayloadV1,
    signature_hex: &str,
    verifying_key: &VerifyingKey,
) -> Result<bool, RcError> {
    let bytes = canonical_payload_bytes(payload)?;
    let sig_bytes = hex::decode(signature_hex)
        .map_err(|err| RcError::InvalidInput(format!("invalid platform attestation signature hex: {err}")))?;
    let signature = Signature::from_slice(&sig_bytes)
        .map_err(|err| RcError::InvalidInput(format!("invalid platform attestation signature bytes: {err}")))?;

    Ok(verifying_key.verify(&bytes, &signature).is_ok())
}

pub fn build_platform_attestation_record(
    platform_id: &str,
    asset_commitment_id: &str,
    issued_at: DateTime<Utc>,
    key_id: &str,
    signing_key: &SigningKey,
) -> Result<PlatformAttestationRecord, RcError> {
    let payload = build_platform_attestation_payload(platform_id, asset_commitment_id, issued_at, key_id);
    let canonical_payload = canonical_payload_json(&payload);
    let signature = sign_platform_attestation(&payload, signing_key)?;
    let digest = Sha256::digest(serde_json::to_vec(&canonical_payload).map_err(|err| RcError::Database(err.to_string()))?);

    Ok(PlatformAttestationRecord {
        attestation_id: format!("pa_{}", hex::encode(digest)),
        version: payload.version,
        platform_id: payload.platform_id,
        asset_commitment_id: payload.asset_commitment_id,
        statement: payload.statement,
        key_id: payload.key_id,
        canonical_payload,
        signature,
        issued_at,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_signing_key() -> SigningKey {
        derive_signing_key("platform-test-secret").unwrap()
    }

    #[test]
    fn platform_attestation_sign_verify_roundtrip() {
        let signing_key = test_signing_key();
        let verifying_key = signing_key.verifying_key();
        let issued_at = DateTime::parse_from_rfc3339("2026-04-09T12:00:00Z").unwrap().with_timezone(&Utc);
        let payload = build_platform_attestation_payload("rcprotocol-main", "commitment_1", issued_at, "platform-key-2026-01");
        let signature = sign_platform_attestation(&payload, &signing_key).unwrap();
        assert!(verify_platform_attestation(&payload, &signature, &verifying_key).unwrap());
    }

    #[test]
    fn platform_attestation_verify_fails_after_tamper() {
        let signing_key = test_signing_key();
        let verifying_key = signing_key.verifying_key();
        let issued_at = DateTime::parse_from_rfc3339("2026-04-09T12:00:00Z").unwrap().with_timezone(&Utc);
        let payload = build_platform_attestation_payload("rcprotocol-main", "commitment_1", issued_at, "platform-key-2026-01");
        let signature = sign_platform_attestation(&payload, &signing_key).unwrap();

        let tampered = build_platform_attestation_payload("rcprotocol-main", "commitment_2", issued_at, "platform-key-2026-01");
        assert!(!verify_platform_attestation(&tampered, &signature, &verifying_key).unwrap());
    }
}
