import type {
  VerifyRequest,
  VerifyApiResponse,
  ListMyAssetsResponse,
  GetMyAssetResponse,
} from './endpoints'
import { unwrapData } from './interceptors'

/**
 * C 端应用 API
 */
export function createAppApi(request: ReturnType<typeof import('./request').createRequest>) {
  return {
    async verify(data: VerifyRequest): Promise<VerifyApiResponse['data']> {
      const response = await request.post<VerifyApiResponse | VerifyApiResponse['data']>('/app/verify', data)
      return unwrapData(response)
    },

    async verifyByQuery(params: VerifyRequest): Promise<VerifyApiResponse['data']> {
      const response = await request.get<VerifyApiResponse | VerifyApiResponse['data']>('/app/verify', params as unknown as Record<string, unknown>)
      return unwrapData(response)
    },

    async listMyAssets(params?: { page?: number; page_size?: number }): Promise<ListMyAssetsResponse> {
      return request.get<ListMyAssetsResponse>('/app/assets', params)
    },

    async getMyAsset(assetId: string): Promise<GetMyAssetResponse['data']> {
      const response = await request.get<GetMyAssetResponse | GetMyAssetResponse['data']>(`/app/assets/${assetId}`)
      return unwrapData(response)
    },

    async consumeAsset(assetId: string, headers?: Record<string, string>) {
      return request.post(`/app/assets/${assetId}/consume`, {}, { headers })
    },

    async legacyAsset(assetId: string, headers?: Record<string, string>) {
      return request.post(`/app/assets/${assetId}/legacy`, {}, { headers })
    },
  }
}
