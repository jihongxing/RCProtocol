export interface ApiError {
  code: string
  message: string
  status: number
  [key: string]: unknown
}

export interface ApiResponse<T> {
  data: T
}

export interface ListResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface RequestConfig {
  baseURL: string
  getToken: () => string | null
  onAuthError: () => void
}

/** 单次请求的可选配置（如自定义请求头） */
export interface RequestOptions {
  headers?: Record<string, string>
}

/** 平台无关的 HTTP 适配器接口 */
export interface HttpAdapter {
  request<T>(method: string, url: string, headers: Record<string, string>, data?: unknown): Promise<{ statusCode: number; data: T }>
}
