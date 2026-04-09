import type {
  InitiateTransferRequest,
  ConfirmTransferRequest,
  InitiateTransferResponse,
  ConfirmTransferResponse,
  GetTransferResponse,
} from './endpoints'
import { unwrapData } from './interceptors'

/**
 * 过户相关 API
 */
export function createTransferApi(request: ReturnType<typeof import('./request').createRequest>) {
  return {
    async initiate(assetId: string, data: InitiateTransferRequest, headers?: Record<string, string>): Promise<InitiateTransferResponse['data']> {
      const response = await request.post<InitiateTransferResponse | InitiateTransferResponse['data']>(`/app/assets/${assetId}/transfer`, data, { headers })
      return unwrapData(response)
    },

    async confirm(data: ConfirmTransferRequest, headers?: Record<string, string>): Promise<ConfirmTransferResponse['data']> {
      const response = await request.post<ConfirmTransferResponse | ConfirmTransferResponse['data']>('/app/transfers/confirm', data, { headers })
      return unwrapData(response)
    },

    async reject(data: ConfirmTransferRequest, headers?: Record<string, string>): Promise<ConfirmTransferResponse['data']> {
      const response = await request.post<ConfirmTransferResponse | ConfirmTransferResponse['data']>('/app/transfers/reject', data, { headers })
      return unwrapData(response)
    },

    async getTransfer(transferId: string): Promise<GetTransferResponse['data']> {
      const response = await request.get<GetTransferResponse | GetTransferResponse['data']>(`/app/transfers/${transferId}`)
      return unwrapData(response)
    },
  }
}
