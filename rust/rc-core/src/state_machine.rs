use rc_common::types::{AssetAction, AssetState};

pub fn next_state_for_action(current: AssetState, action: AssetAction) -> Option<AssetState> {
    match (current, action) {
        (AssetState::PreMinted, AssetAction::BlindLog) => Some(AssetState::FactoryLogged),
        (AssetState::FactoryLogged, AssetAction::StockIn) => Some(AssetState::Unassigned),
        (AssetState::Unassigned, AssetAction::ActivateRotateKeys) => Some(AssetState::RotatingKeys),
        (AssetState::RotatingKeys, AssetAction::ActivateEntangle) => Some(AssetState::EntangledPending),
        (AssetState::EntangledPending, AssetAction::ActivateConfirm) => Some(AssetState::Activated),
        (AssetState::Activated, AssetAction::LegalSell) => Some(AssetState::LegallySold),
        (AssetState::LegallySold, AssetAction::Transfer) => Some(AssetState::Transferred),
        (AssetState::Transferred, AssetAction::Transfer) => Some(AssetState::Transferred),
        (AssetState::LegallySold, AssetAction::Consume) | (AssetState::Transferred, AssetAction::Consume) => {
            Some(AssetState::Consumed)
        }
        (AssetState::LegallySold, AssetAction::Legacy) | (AssetState::Transferred, AssetAction::Legacy) => {
            Some(AssetState::Legacy)
        }
        (state, AssetAction::Freeze) if !state.is_terminal() && !state.is_frozen() => Some(AssetState::Disputed),
        (AssetState::Disputed, AssetAction::MarkCompromised) => Some(AssetState::Compromised),
        (state, AssetAction::MarkTampered) if !state.is_terminal() => Some(AssetState::Tampered),
        (state, AssetAction::MarkCompromised) if !state.is_terminal() => Some(AssetState::Compromised),
        _ => None,
    }
}

pub fn can_transition(from: AssetState, to: AssetState) -> bool {
    next_state_for_action(from, AssetAction::BlindLog) == Some(to)
        || next_state_for_action(from, AssetAction::StockIn) == Some(to)
        || next_state_for_action(from, AssetAction::ActivateRotateKeys) == Some(to)
        || next_state_for_action(from, AssetAction::ActivateEntangle) == Some(to)
        || next_state_for_action(from, AssetAction::ActivateConfirm) == Some(to)
        || next_state_for_action(from, AssetAction::LegalSell) == Some(to)
        || next_state_for_action(from, AssetAction::Transfer) == Some(to)
        || next_state_for_action(from, AssetAction::Consume) == Some(to)
        || next_state_for_action(from, AssetAction::Legacy) == Some(to)
        || next_state_for_action(from, AssetAction::Freeze) == Some(to)
        || next_state_for_action(from, AssetAction::MarkTampered) == Some(to)
        || next_state_for_action(from, AssetAction::MarkCompromised) == Some(to)
}
