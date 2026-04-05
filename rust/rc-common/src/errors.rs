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
}
