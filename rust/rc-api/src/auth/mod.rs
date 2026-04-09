pub mod api_key;
pub mod authorization;
pub mod claims;
pub mod extractor;
pub mod jwt;
pub mod middleware;

/// Auth-module internal error type.
/// Converted to 401 JSON response in the middleware layer.
#[derive(Debug)]
pub enum AuthError {
    MissingToken,
    InvalidSignature,
    TokenExpired,
    InvalidClaims(&'static str),
}
