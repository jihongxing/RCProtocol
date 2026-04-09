/** 资产状态枚举（与 state-machine.md 14 态一致） */
export const ASSET_STATES = {
  PRE_MINTED: 'PreMinted',
  FACTORY_LOGGED: 'FactoryLogged',
  UNASSIGNED: 'Unassigned',
  ROTATING_KEYS: 'RotatingKeys',
  ENTANGLED_PENDING: 'EntangledPending',
  ACTIVATED: 'Activated',
  LEGALLY_SOLD: 'LegallySold',
  TRANSFERRED: 'Transferred',
  CONSUMED: 'Consumed',
  LEGACY: 'Legacy',
  TAMPERED: 'Tampered',
  COMPROMISED: 'Compromised',
  DESTRUCTED: 'Destructed',
  DISPUTED: 'Disputed'
} as const

export const ASSET_STATE_LABELS: Record<string, string> = {
  PreMinted: '预铸造',
  FactoryLogged: '工厂已登记',
  Unassigned: '待认领',
  RotatingKeys: '密钥轮换中',
  EntangledPending: '绑定待确认',
  Activated: '已激活',
  LegallySold: '已售出',
  Transferred: '已过户',
  Consumed: '已消耗',
  Legacy: '遗珍',
  Tampered: '已篡改',
  Compromised: '已失陷',
  Destructed: '已销毁',
  Disputed: '争议冻结'
}

export const ASSET_STATE_COLORS: Record<string, string> = {
  PreMinted: '#9E9E9E',
  FactoryLogged: '#9E9E9E',
  Unassigned: '#FF9800',
  RotatingKeys: '#2196F3',
  EntangledPending: '#2196F3',
  Activated: '#4CAF50',
  LegallySold: '#4CAF50',
  Transferred: '#8BC34A',
  Consumed: '#607D8B',
  Legacy: '#9C27B0',
  Tampered: '#F44336',
  Compromised: '#F44336',
  Destructed: '#212121',
  Disputed: '#FF5722'
}

export const TERMINAL_STATES = new Set([
  'Consumed', 'Legacy', 'Tampered', 'Compromised', 'Destructed'
])

export const ROLES = {
  PLATFORM: 'Platform',
  BRAND: 'Brand',
  FACTORY: 'Factory',
  CONSUMER: 'Consumer',
  MODERATOR: 'Moderator'
} as const

export const ROLE_LABELS: Record<string, string> = {
  Platform: '平台',
  Brand: '品牌',
  Factory: '工厂',
  Consumer: '消费者',
  Moderator: '审核员'
}
