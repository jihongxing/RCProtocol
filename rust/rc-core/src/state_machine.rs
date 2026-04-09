use rc_common::types::{AssetAction, AssetState};

pub fn next_state_for_action(current: AssetState, action: AssetAction) -> Option<AssetState> {
    next_state_for_action_with_previous(current, action, None)
}

/// 状态机核心：根据当前状态和动作计算目标状态。
/// Recover 需要 previous_state 参数（从 PG 读取，非客户端传入）。
pub fn next_state_for_action_with_previous(
    current: AssetState,
    action: AssetAction,
    previous_state: Option<AssetState>,
) -> Option<AssetState> {
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
        // Recover: 从 Disputed 恢复到 PG 记录的 previous_state（非终态、非 Disputed）
        (AssetState::Disputed, AssetAction::Recover) => {
            previous_state.filter(|s| !s.is_terminal() && *s != AssetState::Disputed)
        }
        (AssetState::Disputed, AssetAction::MarkCompromised) => Some(AssetState::Compromised),
        (state, AssetAction::MarkTampered) if !state.is_terminal() => Some(AssetState::Tampered),
        (state, AssetAction::MarkCompromised) if !state.is_terminal() => Some(AssetState::Compromised),
        // MarkDestructed: 非终态均可进入 Destructed
        (state, AssetAction::MarkDestructed) if !state.is_terminal() => Some(AssetState::Destructed),
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
        || next_state_for_action(from, AssetAction::Recover) == Some(to)
        || next_state_for_action(from, AssetAction::MarkTampered) == Some(to)
        || next_state_for_action(from, AssetAction::MarkCompromised) == Some(to)
        || next_state_for_action(from, AssetAction::MarkDestructed) == Some(to)
}
