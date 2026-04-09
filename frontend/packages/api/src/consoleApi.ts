import type {
  GetDashboardResponse,
  CreateBrandRequest,
  UpdateBrandRequest,
  ListBrandsResponse,
  GetBrandResponse,
  CreateBrandResponse,
  UpdateBrandResponse,
  CreateProductRequest,
  UpdateProductRequest,
  ListProductsResponse,
  GetProductResponse,
  CreateProductResponse,
  UpdateProductResponse,
  CreateApiKeyRequest,
  CreateApiKeyResponse,
  ListApiKeysResponse,
  RevokeApiKeyResponse,
  ListAssetsRequest,
  ListAssetsResponse,
  GetAssetResponse,
  ListAuditLogsResponse,
} from './endpoints'
import { unwrapData } from './interceptors'

/**
 * B 端控制台 API
 */
export function createConsoleApi(request: ReturnType<typeof import('./request').createRequest>) {
  return {
    async getDashboard(): Promise<GetDashboardResponse['data']> {
      const response = await request.get<GetDashboardResponse | GetDashboardResponse['data']>('/console/dashboard')
      return unwrapData(response)
    },

    async listBrands(params?: { page?: number; page_size?: number }): Promise<ListBrandsResponse> {
      return request.get<ListBrandsResponse>('/console/brands', params)
    },

    async getBrand(brandId: string): Promise<GetBrandResponse['data']> {
      const response = await request.get<GetBrandResponse | GetBrandResponse['data']>(`/console/brands/${brandId}`)
      return unwrapData(response)
    },

    async createBrand(data: CreateBrandRequest): Promise<CreateBrandResponse['data']> {
      const response = await request.post<CreateBrandResponse | CreateBrandResponse['data']>('/console/brands', data)
      return unwrapData(response)
    },

    async updateBrand(brandId: string, data: UpdateBrandRequest): Promise<UpdateBrandResponse['data']> {
      const response = await request.put<UpdateBrandResponse | UpdateBrandResponse['data']>(`/console/brands/${brandId}`, data)
      return unwrapData(response)
    },

    async listProducts(params?: { brand_id?: string; page?: number; page_size?: number }): Promise<ListProductsResponse> {
      return request.get<ListProductsResponse>('/console/products', params)
    },

    async listBrandProducts(brandId: string, params?: { page?: number; page_size?: number }): Promise<ListProductsResponse> {
      return request.get<ListProductsResponse>(`/console/brands/${brandId}/products`, params)
    },

    async getProduct(productId: string): Promise<GetProductResponse['data']> {
      const response = await request.get<GetProductResponse | GetProductResponse['data']>(`/console/products/${productId}`)
      return unwrapData(response)
    },

    async createProduct(data: CreateProductRequest): Promise<CreateProductResponse['data']> {
      const response = await request.post<CreateProductResponse | CreateProductResponse['data']>('/console/products', data)
      return unwrapData(response)
    },

    async createBrandProduct(brandId: string, data: Omit<CreateProductRequest, 'brand_id'>): Promise<CreateProductResponse['data']> {
      const response = await request.post<CreateProductResponse | CreateProductResponse['data']>(`/console/brands/${brandId}/products`, data)
      return unwrapData(response)
    },

    async updateProduct(productId: string, data: UpdateProductRequest): Promise<UpdateProductResponse['data']> {
      const response = await request.put<UpdateProductResponse | UpdateProductResponse['data']>(`/console/products/${productId}`, data)
      return unwrapData(response)
    },

    async listApiKeys(brandId: string): Promise<ListApiKeysResponse> {
      return request.get<ListApiKeysResponse>(`/console/brands/${brandId}/api-keys`)
    },

    async createApiKey(brandId: string, data: CreateApiKeyRequest): Promise<CreateApiKeyResponse> {
      return unwrapData(await request.post<CreateApiKeyResponse | { data: CreateApiKeyResponse }>(`/console/brands/${brandId}/api-keys`, data))
    },

    async revokeApiKey(brandId: string, keyId: string): Promise<RevokeApiKeyResponse['data']> {
      const response = await request.delete<RevokeApiKeyResponse | RevokeApiKeyResponse['data']>(`/console/brands/${brandId}/api-keys/${keyId}`)
      return unwrapData(response)
    },

    async listAssets(params?: ListAssetsRequest): Promise<ListAssetsResponse> {
      return request.get<ListAssetsResponse>('/console/assets', params)
    },

    async getAsset(assetId: string): Promise<GetAssetResponse['data']> {
      const response = await request.get<GetAssetResponse | GetAssetResponse['data']>(`/console/assets/${assetId}`)
      return unwrapData(response)
    },

    async activateAsset(assetId: string, data: Record<string, unknown>, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/activate`, data, { headers })
    },

    async entangleAsset(assetId: string, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/entangle`, {}, { headers })
    },

    async confirmAssetActivation(assetId: string, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/activate-confirm`, {}, { headers })
    },

    async blindLogAsset(assetId: string, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/blind-log`, {}, { headers })
    },

    async stockInAsset(assetId: string, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/stock-in`, {}, { headers })
    },

    async legalSellAsset(assetId: string, data: Record<string, unknown>, headers?: Record<string, string>) {
      return request.post(`/console/assets/${assetId}/legal-sell`, data, { headers })
    },

    async listAuditLogs(params?: { resource_type?: string; resource_id?: string; page?: number; page_size?: number }): Promise<ListAuditLogsResponse> {
      return request.get<ListAuditLogsResponse>('/console/audit-logs', params)
    },

    async registerPhysicalAuthorityDevice(data: Record<string, unknown>) {
      return request.post('/console/authority-devices/physical', data)
    },
  }
}
