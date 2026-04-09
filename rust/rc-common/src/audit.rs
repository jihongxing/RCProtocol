use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::types::{ActorRole, AssetAction, AssetState};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditContext {
    pub trace_id: Uuid,
    pub actor_id: String,
    pub actor_role: ActorRole,
    pub actor_org: Option<String>,
    pub idempotency_key: String,
    pub approval_id: Option<String>,
    pub policy_version: Option<String>,
    pub buyer_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEvent {
    pub event_id: Uuid,
    pub asset_id: String,
    pub action: AssetAction,
    pub from_state: Option<AssetState>,
    pub to_state: AssetState,
    pub context: AuditContext,
}
