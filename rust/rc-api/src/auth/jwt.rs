use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};

use super::{claims::Claims, AuthError};

/// Reusable JWT decoder — constructed once at startup, shared via `Arc` in AppState.
pub struct JwtDecoder {
    decoding_key: DecodingKey,
    validation: Validation,
}

impl JwtDecoder {
    /// Build a decoder for HS256 tokens signed with `secret`.
    /// Requires `sub`, `exp`, `iat` in the payload; validates `exp` automatically.
    pub fn new(secret: &[u8]) -> Self {
        let mut validation = Validation::new(Algorithm::HS256);
        validation.set_required_spec_claims(&["sub", "exp", "iat"]);
        Self {
            decoding_key: DecodingKey::from_secret(secret),
            validation,
        }
    }

    /// Decode and validate a JWT token string.
    /// On success calls `Claims::validate()` for semantic checks (e.g. brand_id).
    pub fn decode(&self, token: &str) -> Result<Claims, AuthError> {
        let token_data =
            decode::<Claims>(token, &self.decoding_key, &self.validation).map_err(|err| {
                match err.kind() {
                    jsonwebtoken::errors::ErrorKind::ExpiredSignature => AuthError::TokenExpired,
                    jsonwebtoken::errors::ErrorKind::InvalidSignature => {
                        AuthError::InvalidSignature
                    }
                    _ => AuthError::InvalidClaims("token decode failed"),
                }
            })?;
        token_data.claims.validate()?;
        Ok(token_data.claims)
    }
}

/// Encode a JWT for testing purposes. Available to sibling test modules.
#[cfg(test)]
pub(crate) fn encode_test_token(claims: &Claims, secret: &[u8]) -> String {
    jsonwebtoken::encode(
        &jsonwebtoken::Header::new(Algorithm::HS256),
        claims,
        &jsonwebtoken::EncodingKey::from_secret(secret),
    )
    .unwrap()
}

#[cfg(test)]
mod tests {
    use super::*;

    const SECRET: &[u8] = b"test-secret-at-least-32-bytes-long";

    fn valid_claims() -> Claims {
        Claims {
            sub: "user-1".into(),
            role: "Platform".into(),
            org_id: None,
            brand_id: None,
            scopes: vec![],
            exp: u64::MAX,
            iat: 0,
        }
    }

    #[test]
    fn test_decode_valid_token() {
        let decoder = JwtDecoder::new(SECRET);
        let token = encode_test_token(&valid_claims(), SECRET);
        let claims = decoder.decode(&token).expect("should decode valid token");
        assert_eq!(claims.sub, "user-1");
        assert_eq!(claims.role, "Platform");
    }

    #[test]
    fn test_decode_tampered_signature() {
        let decoder = JwtDecoder::new(SECRET);
        let mut token = encode_test_token(&valid_claims(), SECRET);
        // Flip the last character to simulate signature tampering.
        let last = token.pop().unwrap();
        let replacement = if last == 'A' { 'B' } else { 'A' };
        token.push(replacement);

        let err = decoder.decode(&token).unwrap_err();
        assert!(
            matches!(err, AuthError::InvalidSignature),
            "expected InvalidSignature, got {:?}",
            err
        );
    }

    #[test]
    fn test_decode_expired_token() {
        let mut claims = valid_claims();
        claims.exp = 1000; // far in the past
        let decoder = JwtDecoder::new(SECRET);
        let token = encode_test_token(&claims, SECRET);
        let err = decoder.decode(&token).unwrap_err();
        assert!(
            matches!(err, AuthError::TokenExpired),
            "expected TokenExpired, got {:?}",
            err
        );
    }

    #[test]
    fn test_decode_missing_sub() {
        // Build a token without `sub` using a raw JSON value.
        #[derive(serde::Serialize)]
        struct NoSub {
            role: String,
            exp: u64,
            iat: u64,
        }
        let payload = NoSub {
            role: "Platform".into(),
            exp: u64::MAX,
            iat: 0,
        };
        let token = jsonwebtoken::encode(
            &jsonwebtoken::Header::new(Algorithm::HS256),
            &payload,
            &jsonwebtoken::EncodingKey::from_secret(SECRET),
        )
        .unwrap();

        let decoder = JwtDecoder::new(SECRET);
        let err = decoder.decode(&token).unwrap_err();
        assert!(
            matches!(err, AuthError::InvalidClaims(_)),
            "expected InvalidClaims, got {:?}",
            err
        );
    }

    #[test]
    fn test_decode_wrong_secret() {
        let token = encode_test_token(&valid_claims(), SECRET);
        let decoder = JwtDecoder::new(b"wrong-secret-completely-different");
        let err = decoder.decode(&token).unwrap_err();
        assert!(
            matches!(err, AuthError::InvalidSignature),
            "expected InvalidSignature, got {:?}",
            err
        );
    }
}
