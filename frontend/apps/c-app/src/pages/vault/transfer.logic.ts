import type { InitiateTransferRequest } from '@rcprotocol/api'

export type AuthorityMode = 'VirtualToken' | 'PhysicalNfc'

export interface PhysicalNfcResult {
  uid: string
  ctr: string
  cmac: string
}

export interface TransferPageError {
  status?: number
  code?: string
  message?: string
  risk_flags?: string[]
}

interface BuildTransferPayloadInput {
  authorityMode: AuthorityMode
  targetUserId: string
  assetId: string
  userId: string
  virtualCredential: string
  nfcResult?: PhysicalNfcResult | null
}

export function buildTransferPayload(input: BuildTransferPayloadInput): InitiateTransferRequest {
  if (input.authorityMode === 'VirtualToken') {
    return {
      new_owner_id: input.targetUserId.trim(),
      child_uid: input.assetId,
      child_ctr: '000000',
      child_cmac: '0000000000000000',
      authority_proof: {
        type: 'virtual_token',
        user_id: input.userId,
        credential_token: input.virtualCredential,
      }
    }
  }

  const nfcResult = input.nfcResult!
  return {
    new_owner_id: input.targetUserId.trim(),
    child_uid: input.assetId,
    child_ctr: '000000',
    child_cmac: '0000000000000000',
    authority_proof: {
      type: 'physical_nfc',
      uid: nfcResult.uid,
      ctr: nfcResult.ctr,
      cmac: nfcResult.cmac,
    }
  }
}

export function resolveTransferError(error: TransferPageError) {
  if (error.risk_flags?.includes('cmac_invalid')) {
    return { message: '母卡验证失败，请确认使用正确的物理母卡', resetNfc: false, restartScan: false }
  }
  if (error.risk_flags?.includes('uid_mismatch')) {
    return { message: '母卡与当前资产不匹配', resetNfc: false, restartScan: false }
  }
  if (error.risk_flags?.includes('ctr_replay')) {
    return { message: '检测到重放攻击，请重新扫描母卡', resetNfc: true, restartScan: true }
  }
  if (error.risk_flags?.includes('authority_device_inactive')) {
    return { message: '母卡设备已停用', resetNfc: false, restartScan: false }
  }
  if (error.status === 403 || error.code === 'FORBIDDEN') {
    return { message: '您没有权限执行此操作', resetNfc: false, restartScan: false }
  }
  if (error.status === 409 || error.code === 'CONFLICT') {
    return { message: '该资产当前状态不允许转让', resetNfc: false, restartScan: false }
  }
  if (error.status === 422 || error.code === 'UNPROCESSABLE') {
    return { message: '接收方信息无效或不满足转让条件', resetNfc: false, restartScan: false }
  }
  if (error.code === 'NETWORK_ERROR') {
    return { message: '网络连接失败，请稍后重试', resetNfc: false, restartScan: false }
  }
  return { message: error.message || '转让失败', resetNfc: false, restartScan: false }
}

export function getNfcErrorMessage(error: string): string {
  const errorMap: Record<string, string> = {
    NFC_NOT_SUPPORTED: '您的设备不支持 NFC 功能',
    NFC_DISABLED: '请在系统设置中开启 NFC 功能',
    SCAN_TIMEOUT: '扫描超时，请重试',
    MALFORMED_DATA: '数据格式错误，请重新扫描',
    UNKNOWN_ERROR: '未知错误，请重试'
  }
  return errorMap[error] || error
}
