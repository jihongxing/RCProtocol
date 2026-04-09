import { createRequest, createUniAdapter, createAuthApi, createAppApi, createTransferApi } from '@rcprotocol/api'
import { useAuth } from '@rcprotocol/state'

const { getToken } = useAuth()

const request = createRequest({
  baseURL: '/api',
  adapter: createUniAdapter(),
  getToken,
  onAuthError: () => {
    const { logout } = useAuth()
    logout()
    uni.reLaunch({ url: '/pages/login' })
  }
})

export const authApi = createAuthApi(request)
export const appApi = createAppApi(request)
export const transferApi = createTransferApi(request)

export function useTypedApi() {
  return {
    auth: authApi,
    app: appApi,
    transfer: transferApi,
  }
}
