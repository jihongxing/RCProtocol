use thiserror::Error;

#[derive(Debug, Error)]
pub enum RcError {
    #[error("invalid state transition")]
    InvalidStateTransition,
    #[error("permission denied")]
    PermissionDenied,
    #[error("frozen asset blocks normal business action")]
    FrozenAsset,
    #[error("terminal state cannot transition")]
    TerminalState,
    #[error("missing previous state for recover")]
    MissingPreviousState,
    #[error("security check failed")]
    SecurityCheckFailed,
    #[error("asset not found")]
    AssetNotFound,
    #[error("missing required header: {0}")]
    MissingRequiredHeader(&'static str),
    #[error("invalid header: {0}")]
    InvalidHeader(&'static str),
    #[error("idempotency conflict")]
    IdempotencyConflict,
    #[error("database error: {0}")]
    Database(String),
    #[error("invalid input: {0}")]
    InvalidInput(String),
    #[error("unauthorized")]
    Unauthorized,
    #[error("token expired")]
    TokenExpired,
    #[error("invalid signature")]
    InvalidSignature,
    #[error("invalid claims: {0}")]
    InvalidClaims(&'static str),
    #[error("brand boundary violation")]
    BrandBoundaryViolation,
    #[error("forbidden: {0}")]
    Forbidden(String),
    #[error("brand not found")]
    BrandNotFound,
    #[error("product not found")]
    ProductNotFound,
    #[error("duplicate resource: {0}")]
    DuplicateResource(String),
    #[error("not found: {0}")]
    NotFound(String),
    #[error("conflict: {0}")]
    Conflict(String),
}
