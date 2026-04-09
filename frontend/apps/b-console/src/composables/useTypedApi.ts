import { createRequest, createFetchAdapter, createAuthApi, createConsoleApi } from '@rcprotocol/api'
import { useAuth } from '@rcprotocol/state'
import router from '../router'

const { getToken } = useAuth()

const request = createRequest({
  baseURL: '/api',
  adapter: createFetchAdapter(),
  getToken,
  onAuthError: () => {
    const { logout } = useAuth()
    logout()
    router.replace('/login')
  }
})

export const authApi = createAuthApi(request)
export const consoleApi = createConsoleApi(request)

export type UseApiReturn = typeof consoleApi

export function useTypedApi() {
  return {
    auth: authApi,
    console: consoleApi,
  }
}
