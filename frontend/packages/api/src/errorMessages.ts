import type { ApiError } from './types'

/**
 * 错误码到用户可读消息的映射表
 */
const ERROR_MESSAGES: Record<string, string> = {
  NETWORK_ERROR: '网络连接失败',
  AUTH_REQUIRED: '认证已过期，请重新登录',
  FORBIDDEN: '没有权限执行此操作',
  NOT_FOUND: '请求的资源不存在',
  CONFLICT: '操作冲突，请刷新后重试',
  UNPROCESSABLE: '请求参数无效',
  UPSTREAM_FAILURE: '服务暂时不可用，请稍后重试',
}

/**
 * 将 API 错误转换为用户可读的消息
 *
 * @param error - API 错误对象
 * @param context - 可选的操作上下文（如 "创建 API Key"）
 * @returns 用户可读的错误消息
 *
 * @example
 * ```ts
 * getErrorMessage({ code: 'FORBIDDEN', message: 'Access denied', status: 403 })
 * // => "没有权限执行此操作"
 *
 * getErrorMessage({ code: 'FORBIDDEN', message: 'Access denied', status: 403 }, '创建 API Key')
 * // => "没有权限创建 API Key"
 * ```
 */
export function getErrorMessage(error: ApiError, context?: string): string {
  const baseMessage = ERROR_MESSAGES[error.code]

  if (!baseMessage) {
    return error.message
  }

  if (!context) {
    return baseMessage
  }

  // 为特定错误码添加上下文
  switch (error.code) {
    case 'FORBIDDEN':
      return `没有权限${context}`
    case 'NOT_FOUND':
      return `${context}不存在`
    case 'CONFLICT':
      return `${context}冲突，请刷新后重试`
    case 'UNPROCESSABLE':
      return `${context}参数无效`
    default:
      return baseMessage
  }
}
