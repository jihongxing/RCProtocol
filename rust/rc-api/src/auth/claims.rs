use rc_common::types::ActorRole;
use serde::{Deserialize, Serialize};

use super::AuthError;

/// JWT payload claims carried through the request lifecycle.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub role: String,
    pub org_id: Option<String>,
    pub brand_id: Option<String>,
    #[serde(default)]
    pub scopes: Vec<String>,
    pub exp: u64,
    pub iat: u64,
}

impl Claims {
    /// Map the free-form `role` string to the canonical `ActorRole` enum.
    pub fn actor_role(&self) -> Result<ActorRole, AuthError> {
        match self.role.as_str() {
            "Platform" => Ok(ActorRole::Platform),
            "Factory" => Ok(ActorRole::Factory),
            "Brand" => Ok(ActorRole::Brand),
            "Consumer" => Ok(ActorRole::Consumer),
            "Moderator" => Ok(ActorRole::Moderator),
            _ => Err(AuthError::InvalidClaims("unknown role")),
        }
    }

    /// Semantic validation: Brand/Factory roles must carry a `brand_id`.
    pub fn validate(&self) -> Result<(), AuthError> {
        let role = self.actor_role()?;
        if matches!(role, ActorRole::Brand | ActorRole::Factory) && self.brand_id.is_none() {
            return Err(AuthError::InvalidClaims(
                "brand_id required for Brand/Factory role",
            ));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn base_claims(role: &str, brand_id: Option<&str>) -> Claims {
        Claims {
            sub: "user-1".into(),
            role: role.into(),
            org_id: None,
            brand_id: brand_id.map(String::from),
            scopes: vec![],
            exp: u64::MAX,
            iat: 0,
        }
    }

    #[test]
    fn test_platform_no_brand_id_ok() {
        let claims = base_claims("Platform", None);
        assert!(claims.validate().is_ok());
    }

    #[test]
    fn test_brand_missing_brand_id() {
        let claims = base_claims("Brand", None);
        let err = claims.validate().unwrap_err();
        assert!(matches!(err, AuthError::InvalidClaims(msg) if msg.contains("brand_id")));
    }

    #[test]
    fn test_brand_with_brand_id_ok() {
        let claims = base_claims("Brand", Some("brand-abc"));
        assert!(claims.validate().is_ok());
    }

    #[test]
    fn test_unknown_role() {
        let claims = base_claims("Admin", None);
        let err = claims.actor_role().unwrap_err();
        assert!(matches!(err, AuthError::InvalidClaims("unknown role")));
    }
}
