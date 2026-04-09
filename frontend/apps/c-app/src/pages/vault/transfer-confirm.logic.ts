import type { TransferInfo } from '@rcprotocol/utils'

export interface TransferConfirmError {
  status?: number
  code?: string
  message?: string
}

export function normalizeTransferInfo(transferInfo: TransferInfo) {
  if (transferInfo.status !== 'pending') {
    return { expired: true, transferInfo: null as TransferInfo | null }
  }

  return { expired: false, transferInfo }
}

export function resolveTransferLoadError(error: TransferConfirmError) {
  if (error.status === 404 || error.code === 'NOT_FOUND') {
    return { expired: true, loadError: '' }
  }
  if (error.code === 'NETWORK_ERROR') {
    return { expired: false, loadError: '网络连接失败' }
  }
  return { expired: false, loadError: error.message || '加载失败' }
}

export function resolveTransferActionError(error: TransferConfirmError, fallbackMessage: string) {
  if (error.status === 409 || error.code === 'CONFLICT') {
    return { message: '该转让请求已过期或已处理', expired: true, clearTransferInfo: true }
  }
  if (error.status === 403 || error.code === 'FORBIDDEN') {
    return { message: '您没有权限处理此资产转让', expired: false, clearTransferInfo: false }
  }
  if (error.code === 'NETWORK_ERROR') {
    return { message: '网络连接失败，请稍后重试', expired: false, clearTransferInfo: false }
  }
  return { message: error.message || fallbackMessage, expired: false, clearTransferInfo: false }
}
