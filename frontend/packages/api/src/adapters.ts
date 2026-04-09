import type { HttpAdapter } from './types'

declare const uni: {
  request(options: {
    url: string
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
    data?: string | object | ArrayBuffer
    header?: Record<string, string>
    success?: (res: { statusCode: number; data: unknown }) => void
    fail?: (err: { errMsg?: string }) => void
  }): void
}

/** uni-app 适配器，供 c-app 使用 */
export function createUniAdapter(): HttpAdapter {
  return {
    request<T>(method: string, url: string, headers: Record<string, string>, data?: unknown) {
      return new Promise<{ statusCode: number; data: T }>((resolve, reject) => {
        uni.request({
          url,
          method: method as 'GET' | 'POST' | 'PUT' | 'DELETE',
          data: data as string | object | ArrayBuffer | undefined,
          header: headers,
          success(res: { statusCode: number; data: unknown }) {
            resolve({ statusCode: res.statusCode, data: res.data as T })
          },
          fail(err: { errMsg?: string }) {
            reject(new Error(err.errMsg || 'Network request failed'))
          }
        })
      })
    }
  }
}

/** fetch 适配器，供 b-console 使用 */
export function createFetchAdapter(): HttpAdapter {
  return {
    async request<T>(method: string, url: string, headers: Record<string, string>, data?: unknown) {
      const init: RequestInit = { method, headers }
      if (data !== undefined && method !== 'GET') {
        init.body = JSON.stringify(data)
      }
      try {
        const res = await fetch(url, init)
        const body = await res.json().catch(() => ({}))
        return { statusCode: res.status, data: body as T }
      } catch {
        throw new Error('Network request failed')
      }
    }
  }
}
