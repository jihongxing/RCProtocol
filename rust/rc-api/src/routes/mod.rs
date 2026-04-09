use axum::{http::StatusCode, response::IntoResponse};
use rc_common::errors::RcError;

pub mod assets;
pub mod authority_devices;
pub mod batch;
pub mod brand;
pub mod health;
pub mod protocol;
pub mod transfer;
pub mod verify;

pub fn error_response(err: RcError) -> axum::response::Response {
    let status = match err {
        RcError::AssetNotFound => StatusCode::NOT_FOUND,
        RcError::BrandNotFound | RcError::ProductNotFound | RcError::NotFound(_) => StatusCode::NOT_FOUND,
        RcError::InvalidInput(_) => StatusCode::BAD_REQUEST,
        RcError::DuplicateResource(_) | RcError::Conflict(_) => StatusCode::CONFLICT,
        RcError::MissingRequiredHeader(_) | RcError::InvalidHeader(_) => StatusCode::BAD_REQUEST,
        RcError::IdempotencyConflict => StatusCode::CONFLICT,
        RcError::Database(_) => StatusCode::INTERNAL_SERVER_ERROR,
        RcError::Unauthorized
        | RcError::TokenExpired
        | RcError::InvalidSignature
        | RcError::InvalidClaims(_) => StatusCode::UNAUTHORIZED,
        RcError::BrandBoundaryViolation | RcError::PermissionDenied | RcError::Forbidden(_) => StatusCode::FORBIDDEN,
        _ => StatusCode::BAD_REQUEST,
    };

    (status, err.to_string()).into_response()
}
