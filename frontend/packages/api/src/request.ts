import type { RequestConfig, ApiError, HttpAdapter, RequestOptions } from './types'
import { generateTraceId, handleErrorResponse } from './interceptors'

function buildUrl(baseURL: string, path: string, params?: Record<string, any>): string {
  const url = baseURL + path
  if (!params || Object.keys(params).length === 0) {
    return url
  }
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null) {
      query.append(key, String(value))
    }
  }
  const queryString = query.toString()
  return queryString ? `${url}?${queryString}` : url
}

export function createRequest(config: RequestConfig & { adapter: HttpAdapter }) {
  const { baseURL, getToken, onAuthError, adapter } = config

  async function request<T>(method: string, path: string, data?: unknown, options?: RequestOptions): Promise<T> {
    const token = getToken()
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Trace-Id': generateTraceId()
    }
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    // 合并调用方传入的自定义请求头（如 X-Idempotency-Key）
    if (options?.headers) {
      Object.assign(headers, options.headers)
    }

    let res: { statusCode: number; data: T }
    try {
      res = await adapter.request<T>(method, baseURL + path, headers, data)
    } catch {
      const error: ApiError = { code: 'NETWORK_ERROR', message: 'Network request failed', status: 0 }
      throw error
    }

    if (res.statusCode >= 200 && res.statusCode < 300) {
      return res.data
    }

    if (res.statusCode === 401) {
      onAuthError()
    }

    throw handleErrorResponse(res.statusCode, res.data as Record<string, unknown>)
  }

  return {
    get: <T>(path: string, params?: Record<string, any>) => request<T>('GET', buildUrl('', path, params)),
    post: <T>(path: string, data?: unknown, options?: RequestOptions) => request<T>('POST', path, data, options),
    put: <T>(path: string, data?: unknown, options?: RequestOptions) => request<T>('PUT', path, data, options),
    del: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, undefined, options),
    delete: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, undefined, options)
  }
}
