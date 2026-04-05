use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum AssetState {
    PreMinted,
    FactoryLogged,
    Unassigned,
    RotatingKeys,
    EntangledPending,
    Activated,
    LegallySold,
    Transferred,
    Consumed,
    Legacy,
    Tampered,
    Compromised,
    Destructed,
    Disputed,
}

impl AssetState {
    pub fn is_terminal(self) -> bool {
        matches!(
            self,
            Self::Consumed | Self::Legacy | Self::Tampered | Self::Compromised | Self::Destructed
        )
    }

    pub fn is_frozen(self) -> bool {
        matches!(self, Self::Disputed)
    }

    pub fn as_db_str(self) -> &'static str {
        match self {
            Self::PreMinted => "PreMinted",
            Self::FactoryLogged => "FactoryLogged",
            Self::Unassigned => "Unassigned",
            Self::RotatingKeys => "RotatingKeys",
            Self::EntangledPending => "EntangledPending",
            Self::Activated => "Activated",
            Self::LegallySold => "LegallySold",
            Self::Transferred => "Transferred",
            Self::Consumed => "Consumed",
            Self::Legacy => "Legacy",
            Self::Tampered => "Tampered",
            Self::Compromised => "Compromised",
            Self::Destructed => "Destructed",
            Self::Disputed => "Disputed",
        }
    }

    pub fn from_db_str(value: &str) -> Option<Self> {
        match value {
            "PreMinted" => Some(Self::PreMinted),
            "FactoryLogged" => Some(Self::FactoryLogged),
            "Unassigned" => Some(Self::Unassigned),
            "RotatingKeys" => Some(Self::RotatingKeys),
            "EntangledPending" => Some(Self::EntangledPending),
            "Activated" => Some(Self::Activated),
            "LegallySold" => Some(Self::LegallySold),
            "Transferred" => Some(Self::Transferred),
            "Consumed" => Some(Self::Consumed),
            "Legacy" => Some(Self::Legacy),
            "Tampered" => Some(Self::Tampered),
            "Compromised" => Some(Self::Compromised),
            "Destructed" => Some(Self::Destructed),
            "Disputed" => Some(Self::Disputed),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum ActorRole {
    Platform,
    Factory,
    Brand,
    Consumer,
    Moderator,
}

impl ActorRole {
    pub fn as_db_str(self) -> &'static str {
        match self {
            Self::Platform => "Platform",
            Self::Factory => "Factory",
            Self::Brand => "Brand",
            Self::Consumer => "Consumer",
            Self::Moderator => "Moderator",
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssetRecord {
    pub asset_id: String,
    pub brand_id: String,
    pub current_state: AssetState,
    pub previous_state: Option<AssetState>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum AssetAction {
    BlindLog,
    StockIn,
    ActivateRotateKeys,
    ActivateEntangle,
    ActivateConfirm,
    LegalSell,
    Transfer,
    Consume,
    Legacy,
    Freeze,
    Recover,
    MarkTampered,
    MarkCompromised,
}

impl AssetAction {
    pub fn as_db_str(self) -> &'static str {
        match self {
            Self::BlindLog => "BlindLog",
            Self::StockIn => "StockIn",
            Self::ActivateRotateKeys => "ActivateRotateKeys",
            Self::ActivateEntangle => "ActivateEntangle",
            Self::ActivateConfirm => "ActivateConfirm",
            Self::LegalSell => "LegalSell",
            Self::Transfer => "Transfer",
            Self::Consume => "Consume",
            Self::Legacy => "Legacy",
            Self::Freeze => "Freeze",
            Self::Recover => "Recover",
            Self::MarkTampered => "MarkTampered",
            Self::MarkCompromised => "MarkCompromised",
        }
    }
}
