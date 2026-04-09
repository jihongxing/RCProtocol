import { describe, it, expect } from 'vitest'
import { normalizeTransferInfo, resolveTransferActionError, resolveTransferLoadError } from './vault/transfer-confirm.logic'

describe('Transfer Confirm Page logic', () => {
  it('keeps pending transfer info', () => {
    const transferInfo = {
      transfer_id: 'tr-001',
      asset_id: 'asset-1',
      from_user_id: 'user-001',
      to_user_id: 'user-002',
      status: 'pending',
      created_at: '2025-01-01T00:00:00Z'
    }

    expect(normalizeTransferInfo(transferInfo as never)).toEqual({
      expired: false,
      transferInfo,
    })
  })

  it('marks processed transfer as expired', () => {
    expect(normalizeTransferInfo({ status: 'rejected' } as never)).toEqual({
      expired: true,
      transferInfo: null,
    })
  })

  it('resolves transfer load errors', () => {
    expect(resolveTransferLoadError({ status: 404, code: 'NOT_FOUND' })).toEqual({
      expired: true,
      loadError: '',
    })

    expect(resolveTransferLoadError({ code: 'NETWORK_ERROR' })).toEqual({
      expired: false,
      loadError: '网络连接失败',
    })
  })

  it('resolves conflict action error', () => {
    expect(resolveTransferActionError({ status: 409, code: 'CONFLICT' }, '确认失败')).toEqual({
      message: '该转让请求已过期或已处理',
      expired: true,
      clearTransferInfo: true,
    })
  })

  it('resolves forbidden action error', () => {
    expect(resolveTransferActionError({ status: 403, code: 'FORBIDDEN' }, '确认失败')).toEqual({
      message: '您没有权限处理此资产转让',
      expired: false,
      clearTransferInfo: false,
    })
  })
})
