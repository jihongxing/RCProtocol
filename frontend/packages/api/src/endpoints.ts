import type { ApiResponse, ListResponse } from './types'
import type {
  DashboardData,
  Brand,
  Product,
  ApiKeyItem,
  Asset,
  AssetVM,
  AssetDetailVM,
  VerifyResponse,
  TransferInfo,
  AuditLog,
} from '@rcprotocol/utils'

/**
 * API 请求/响应类型定义
 */

// ============================================
// Auth API
// ============================================

export interface LoginRequest {
  email: string
  password: string
  org_id?: string
}

export interface SelectOrgRequest {
  org_id: string
}

// ============================================
// Console API - Dashboard
// ============================================

export type GetDashboardResponse = ApiResponse<DashboardData>

// ============================================
// Console API - Brands
// ============================================

export interface CreateBrandRequest {
  brand_name: string
  contact_email: string
  contact_phone: string
}

export interface UpdateBrandRequest {
  brand_name?: string
  contact_email?: string
  contact_phone?: string
}

export type ListBrandsResponse = ListResponse<Brand>
export type GetBrandResponse = ApiResponse<Brand>
export type CreateBrandResponse = ApiResponse<Brand>
export type UpdateBrandResponse = ApiResponse<Brand>

// ============================================
// Console API - Products
// ============================================

export interface CreateProductRequest {
  product_name: string
  brand_id: string
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
}

export interface UpdateProductRequest {
  product_name?: string
  external_product_id?: string
  external_product_name?: string
  external_product_url?: string
}

export type ListProductsResponse = ListResponse<Product>
export type GetProductResponse = ApiResponse<Product>
export type CreateProductResponse = ApiResponse<Product>
export type UpdateProductResponse = ApiResponse<Product>

// ============================================
// Console API - API Keys
// ============================================

export interface CreateApiKeyRequest {
  description?: string
}

export interface CreateApiKeyResponse {
  key_id: string
  api_key: string
  description: string
  created_at: string
}

export type ListApiKeysResponse = ListResponse<ApiKeyItem>
export type RevokeApiKeyResponse = ApiResponse<{ success: boolean }>

// ============================================
// Console API - Assets / Audit Logs
// ============================================

export interface ListAssetsRequest {
  brand_id?: string
  state?: string
  page?: number
  page_size?: number
}

export type ListAssetsResponse = ListResponse<Asset>
export type GetAssetResponse = ApiResponse<AssetDetailVM>
export type ListAuditLogsResponse = ListResponse<AuditLog>

// ============================================
// App API - Verify
// ============================================

export interface VerifyRequest {
  uid: string
  ctr: number
  cmac: string
}

export type VerifyApiResponse = ApiResponse<VerifyResponse>

// ============================================
// App API - Assets
// ============================================

export type ListMyAssetsResponse = ListResponse<AssetVM>
export type GetMyAssetResponse = ApiResponse<AssetDetailVM>

// ============================================
// Transfer API
// ============================================

export interface InitiateTransferRequest {
  new_owner_id: string
  child_uid: string
  child_ctr: string
  child_cmac: string
  authority_proof:
    | {
        type: 'virtual_token'
        user_id: string
        credential_token: string
      }
    | {
        type: 'physical_nfc'
        uid: string
        ctr: string
        cmac: string
      }
}

export interface ConfirmTransferRequest {
  transfer_id: string
}

export type InitiateTransferResponse = ApiResponse<TransferInfo>
export type ConfirmTransferResponse = ApiResponse<{ success: boolean }>
export type GetTransferResponse = ApiResponse<TransferInfo>
