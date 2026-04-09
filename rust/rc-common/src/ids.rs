use crate::errors::RcError;
use serde::{Deserialize, Serialize};
use std::fmt;

fn generate_prefixed_id(prefix: &str) -> String {
    format!("{prefix}_{}", ulid::Ulid::new())
}

fn validate_prefixed_id(value: &str, prefix: &str) -> Result<(), RcError> {
    let expected_prefix = format!("{prefix}_");
    let suffix = value
        .strip_prefix(&expected_prefix)
        .ok_or_else(|| RcError::InvalidInput(format!("id must start with {expected_prefix}")))?;

    if suffix.len() != 26 {
        return Err(RcError::InvalidInput(format!(
            "id suffix for {prefix} must be 26 characters"
        )));
    }

    ulid::Ulid::from_string(suffix)
        .map_err(|_| RcError::InvalidInput(format!("id suffix for {prefix} must be a valid ULID")))?;

    Ok(())
}

macro_rules! define_resource_id {
    ($name:ident, $prefix:literal, $generator:ident) => {
        #[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
        pub struct $name(String);

        impl $name {
            pub fn new(value: String) -> Result<Self, RcError> {
                validate_prefixed_id(&value, $prefix)?;
                Ok(Self(value))
            }

            pub fn generate() -> Self {
                Self(generate_prefixed_id($prefix))
            }

            pub fn as_str(&self) -> &str {
                &self.0
            }
        }

        impl fmt::Display for $name {
            fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                self.0.fmt(f)
            }
        }

        impl AsRef<str> for $name {
            fn as_ref(&self) -> &str {
                self.as_str()
            }
        }

        impl TryFrom<String> for $name {
            type Error = RcError;

            fn try_from(value: String) -> Result<Self, Self::Error> {
                Self::new(value)
            }
        }

        impl TryFrom<&str> for $name {
            type Error = RcError;

            fn try_from(value: &str) -> Result<Self, Self::Error> {
                Self::new(value.to_string())
            }
        }

        impl From<$name> for String {
            fn from(value: $name) -> Self {
                value.0
            }
        }

        pub fn $generator() -> String {
            $name::generate().into()
        }
    };
}

define_resource_id!(BrandId, "brand", generate_brand_id);
define_resource_id!(ProductId, "product", generate_product_id);
define_resource_id!(AssetId, "asset", generate_asset_id);
define_resource_id!(BatchId, "batch", generate_batch_id);
define_resource_id!(SessionId, "session", generate_session_id);

#[cfg(test)]
mod tests {
    use super::*;

    fn assert_prefixed_id(id: &str, prefix: &str) {
        assert!(id.starts_with(&format!("{prefix}_")));
        let suffix = &id[prefix.len() + 1..];
        assert_eq!(suffix.len(), 26);
        assert!(ulid::Ulid::from_string(suffix).is_ok());
    }

    #[test]
    fn generates_brand_ids() {
        assert_prefixed_id(&generate_brand_id(), "brand");
    }

    #[test]
    fn generates_product_ids() {
        assert_prefixed_id(&generate_product_id(), "product");
    }

    #[test]
    fn generates_asset_ids() {
        assert_prefixed_id(&generate_asset_id(), "asset");
    }

    #[test]
    fn generates_batch_ids() {
        assert_prefixed_id(&generate_batch_id(), "batch");
    }

    #[test]
    fn generates_session_ids() {
        assert_prefixed_id(&generate_session_id(), "session");
    }

    #[test]
    fn strong_typed_wrappers_validate_prefix_and_ulid() {
        let brand = BrandId::generate();
        assert!(brand.as_str().starts_with("brand_"));
        assert!(matches!(BrandId::try_from("brand_bad"), Err(RcError::InvalidInput(_))));
        assert!(matches!(AssetId::try_from("batch_01ARZ3NDEKTSV4RRFFQ69G5FAV"), Err(RcError::InvalidInput(_))));
    }
}
