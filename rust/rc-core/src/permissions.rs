use rc_common::types::{ActorRole, AssetAction, AssetState};

pub fn can_role_initiate(role: ActorRole, current_state: AssetState, action: AssetAction) -> bool {
    match role {
        ActorRole::Platform => true,
        ActorRole::Factory => matches!(
            (current_state, action),
            (AssetState::PreMinted, AssetAction::BlindLog)
                | (AssetState::FactoryLogged, AssetAction::StockIn)
        ),
        ActorRole::Brand => matches!(
            (current_state, action),
            (AssetState::Unassigned, AssetAction::ActivateRotateKeys)
                | (AssetState::RotatingKeys, AssetAction::ActivateEntangle)
                | (AssetState::EntangledPending, AssetAction::ActivateConfirm)
                | (AssetState::Activated, AssetAction::LegalSell)
        ),
        ActorRole::Consumer => matches!(
            (current_state, action),
            (AssetState::LegallySold, AssetAction::Transfer)
                | (AssetState::Transferred, AssetAction::Transfer)
                | (AssetState::LegallySold, AssetAction::Consume)
                | (AssetState::Transferred, AssetAction::Consume)
                | (AssetState::LegallySold, AssetAction::Legacy)
                | (AssetState::Transferred, AssetAction::Legacy)
        ),
        ActorRole::Moderator => matches!(
            action,
            AssetAction::Freeze | AssetAction::Recover | AssetAction::MarkTampered | AssetAction::MarkCompromised | AssetAction::MarkDestructed
        ),
    }
}
