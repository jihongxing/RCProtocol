use uuid::Uuid;

use rc_common::{
    audit::{AuditContext, AuditEvent},
    errors::RcError,
    types::{ActorRole, AssetAction, AssetRecord, AssetState},
};

use crate::{permissions::can_role_initiate, state_machine::{next_state_for_action, next_state_for_action_with_previous}};

pub fn apply_action(record: &AssetRecord, action: AssetAction, context: AuditContext) -> Result<(AssetRecord, AuditEvent), RcError> {
    if record.current_state.is_terminal() {
        return Err(RcError::TerminalState);
    }

    let is_business_action = matches!(
        action,
        AssetAction::BlindLog
            | AssetAction::StockIn
            | AssetAction::ActivateRotateKeys
            | AssetAction::ActivateEntangle
            | AssetAction::ActivateConfirm
            | AssetAction::LegalSell
            | AssetAction::Transfer
            | AssetAction::Consume
            | AssetAction::Legacy
    );

    if record.current_state.is_frozen() && is_business_action {
        return Err(RcError::FrozenAsset);
    }

    // Platform 执行非治理类动作时强制要求 approval_id
    if context.actor_role == ActorRole::Platform && is_business_action && context.approval_id.is_none() {
        return Err(RcError::PermissionDenied);
    }

    if !can_role_initiate(context.actor_role, record.current_state, action) {
        return Err(RcError::PermissionDenied);
    }

    // LegalSell 必须携带 buyer_id
    if action == AssetAction::LegalSell && context.buyer_id.is_none() {
        return Err(RcError::InvalidInput("LegalSell requires buyer_id".into()));
    }

    let target_state = match action {
        // Recover: 从 PG 记录的 previous_state 恢复（非客户端传入）
        AssetAction::Recover => {
            next_state_for_action_with_previous(record.current_state, action, record.previous_state)
                .ok_or(RcError::InvalidStateTransition)?
        }
        _ => next_state_for_action(record.current_state, action).ok_or(RcError::InvalidStateTransition)?,
    };

    let next_record = AssetRecord {
        asset_id: record.asset_id.clone(),
        brand_id: record.brand_id.clone(),
        current_state: target_state,
        previous_state: if target_state == AssetState::Disputed {
            Some(record.current_state)
        } else {
            None
        },
    };

    let audit_event = AuditEvent {
        event_id: Uuid::new_v4(),
        asset_id: record.asset_id.clone(),
        action,
        from_state: Some(record.current_state),
        to_state: target_state,
        context,
    };

    Ok((next_record, audit_event))
}
