import type { LoginRequest, SelectOrgRequest } from './endpoints'
import type { LoginResponse } from '@rcprotocol/utils'
import type { ApiResponse } from './types'
import { unwrapData } from './interceptors'

/**
 * 认证相关 API
 */
export function createAuthApi(request: ReturnType<typeof import('./request').createRequest>) {
  return {
    /**
     * 用户登录
     */
    async login(data: LoginRequest): Promise<LoginResponse> {
      const response = await request.post<ApiResponse<LoginResponse> | LoginResponse>('/auth/login', data)
      return unwrapData(response)
    },

    /**
     * 选择组织（多组织场景）
     */
    async selectOrg(data: SelectOrgRequest & Partial<LoginRequest>): Promise<LoginResponse> {
      const response = await request.post<ApiResponse<LoginResponse> | LoginResponse>('/auth/select-org', data)
      return unwrapData(response)
    },

    /**
     * 登出
     */
    async logout(): Promise<void> {
      await request.post('/auth/logout', {})
    },
  }
}
