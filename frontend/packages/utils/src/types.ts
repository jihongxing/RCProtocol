export interface Asset {
  asset_id: string
  state: string
  state_label: string
  brand_id: string
  product_id: string
  owner_id?: string
  risk_flags?: string[]
  allowed_actions?: string[]
  display_badges?: string[]
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
  created_at: string
  updated_at: string
}

export interface Brand {
  brand_id: string
  brand_name: string
  status: string
  created_at: string
}

export interface Product {
  product_id: string
  product_name: string
  brand_id: string
  status: string
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
  created_at: string
}

export interface Workorder {
  id: string
  type: string
  status: string
  title: string
  description?: string
  asset_id?: string
  assignee_id?: string
  conclusion?: string
  conclusion_type?: string
  created_at: string
  updated_at: string
}

export interface Approval {
  id: string
  type: string
  status: string
  applicant_id: string
  resource_type: string
  resource_id: string
  created_at: string
  expires_at: string
}

// ============================================
// B 端共享类型
// ============================================

export interface MenuItem {
  label: string
  path: string
  icon?: string
}

export interface DashboardData {
  total_assets: number
  active_assets: number
  total_brands: number
  pending_approvals: number
}

export interface ApiKey {
  key_id: string
  key_prefix: string
  description: string
  status: 'active' | 'revoked'
  created_at: string
  last_used_at?: string
  revoked_at?: string
}

export interface ApiKeyItem {
  key_id: string
  key_prefix: string
  description: string
  status: 'active' | 'revoked'
  created_at: string
  last_used_at?: string
  revoked_at?: string
}

export interface AuditLog {
  log_id: string
  event_type: string
  actor_id: string
  resource_type: string
  resource_id: string
  details: Record<string, unknown>
  created_at: string
}

export interface AuthorityDevice {
  device_id: string
  chip_uid: string
  brand_id: string
  key_epoch: number
  asset_id?: string
  status: string
  last_known_ctr?: number
  created_at: string
}

export interface LoginResponse {
  token: string
  user: {
    user_id: string
    email: string
    display_name: string
    role: string
    org_id?: string
    org_name?: string
    brand_id?: string
  }
  requires_org_selection?: boolean
  available_orgs?: Array<{
    org_id: string
    org_name: string
    role: string
  }>
}

// ============================================
// C 端共享类型
// ============================================

export interface VerifyResponse {
  valid?: boolean
  verification_status?: 'verified' | 'failed' | 'authentication_failed' | 'unknown' | 'unknown_tag' | 'restricted' | 'unverified'
  asset?: {
    asset_id?: string
    uid?: string
    brand_id?: string
    brand_name?: string
    product_id?: string
    product_name?: string
    state?: string
    current_state?: string
    state_label?: string
    risk_flags?: string[]
    external_product_id?: string
    external_product_name?: string
    external_product_url?: string
  }
  risk_flags?: string[]
  scan_metadata?: {
    ctr?: number
  }
  message?: string
}

export interface AssetVM {
  asset_id: string
  state: string
  state_label: string
  brand_id: string
  brand_name?: string
  product_id?: string
  product_name?: string
  uid?: string
  owner_id?: string
  risk_flags?: string[]
  display_badges?: string[]
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
  created_at: string
  updated_at: string
}

export interface AssetDetailVM extends AssetVM {
  allowed_actions?: string[]
  virtual_mother_card?: {
    authority_uid?: string
    authority_type?: string
    credential_hash?: string
    epoch?: number
  }
  transfer_history?: Array<{
    from_user_id: string
    to_user_id: string
    transferred_at: string
  }>
}

export interface AssetSummary {
  asset_id: string
  state: string
  state_label: string
  brand_name: string
  product_name?: string
  external_product_name?: string
}

export interface TransferInfo {
  transfer_id: string
  asset_id: string
  from_user_id: string
  to_user_id: string
  status: string
  created_at: string
  asset_summary: AssetSummary
}
