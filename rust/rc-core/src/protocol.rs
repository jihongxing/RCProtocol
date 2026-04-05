use uuid::Uuid;

use rc_common::{
    audit::{AuditContext, AuditEvent},
    errors::RcError,
    types::{AssetAction, AssetRecord, AssetState},
};

use crate::{permissions::can_role_initiate, state_machine::next_state_for_action};

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

    if !can_role_initiate(context.actor_role, record.current_state, action) {
        return Err(RcError::PermissionDenied);
    }

    let target_state = match action {
        AssetAction::Recover => record.previous_state.ok_or(RcError::MissingPreviousState)?,
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
