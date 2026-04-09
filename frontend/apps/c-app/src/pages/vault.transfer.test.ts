import { describe, it, expect } from 'vitest'
import { buildTransferPayload, getNfcErrorMessage, resolveTransferError } from './vault/transfer.logic'

describe('Transfer Page logic', () => {
  it('builds virtual token transfer payload', () => {
    expect(buildTransferPayload({
      authorityMode: 'VirtualToken',
      targetUserId: 'user-002',
      assetId: 'asset-1',
      userId: 'user-001',
      virtualCredential: 'credential-1',
    })).toEqual({
      new_owner_id: 'user-002',
      child_uid: 'asset-1',
      child_ctr: '000000',
      child_cmac: '0000000000000000',
      authority_proof: {
        type: 'virtual_token',
        user_id: 'user-001',
        credential_token: 'credential-1',
      }
    })
  })

  it('builds physical nfc transfer payload', () => {
    expect(buildTransferPayload({
      authorityMode: 'PhysicalNfc',
      targetUserId: 'user-002',
      assetId: 'asset-1',
      userId: 'user-001',
      virtualCredential: '',
      nfcResult: {
        uid: '04ABCD',
        ctr: '123456',
        cmac: '89ABCDEF01234567',
      }
    })).toEqual({
      new_owner_id: 'user-002',
      child_uid: 'asset-1',
      child_ctr: '000000',
      child_cmac: '0000000000000000',
      authority_proof: {
        type: 'physical_nfc',
        uid: '04ABCD',
        ctr: '123456',
        cmac: '89ABCDEF01234567',
      }
    })
  })

  it('resolves forbidden transfer error', () => {
    expect(resolveTransferError({ status: 403, code: 'FORBIDDEN' })).toEqual({
      message: '您没有权限执行此操作',
      resetNfc: false,
      restartScan: false,
    })
  })

  it('resolves ctr replay transfer error', () => {
    expect(resolveTransferError({ risk_flags: ['ctr_replay'] })).toEqual({
      message: '检测到重放攻击，请重新扫描母卡',
      resetNfc: true,
      restartScan: true,
    })
  })

  it('maps nfc errors to user-facing message', () => {
    expect(getNfcErrorMessage('SCAN_TIMEOUT')).toBe('扫描超时，请重试')
    expect(getNfcErrorMessage('SOMETHING_ELSE')).toBe('SOMETHING_ELSE')
  })
})
