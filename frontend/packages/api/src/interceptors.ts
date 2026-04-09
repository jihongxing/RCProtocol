import type { ApiError } from './types'

export function generateTraceId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 10)
}

export function handleErrorResponse(statusCode: number, data: Record<string, unknown>): ApiError {
  const errorData = data?.error as { code?: string; message?: string } | undefined
  const codeMap: Record<number, string> = {
    400: 'INVALID_INPUT',
    401: 'AUTH_REQUIRED',
    403: 'FORBIDDEN',
    404: 'NOT_FOUND',
    409: 'CONFLICT',
    422: 'UNPROCESSABLE'
  }

  const error: ApiError = {
    code: errorData?.code || codeMap[statusCode] || (statusCode >= 500 ? 'UPSTREAM_FAILURE' : 'UNKNOWN_ERROR'),
    message: errorData?.message || (data?.message as string) || `Request failed with status ${statusCode}`,
    status: statusCode
  }

  for (const [key, value] of Object.entries(data || {})) {
    if (key !== 'error') {
      error[key] = value
    }
  }

  return error
}

export function unwrapData<T>(payload: T | { data: T }): T {
  if (payload && typeof payload === 'object' && 'data' in (payload as Record<string, unknown>)) {
    return (payload as { data: T }).data
  }
  return payload as T
}
