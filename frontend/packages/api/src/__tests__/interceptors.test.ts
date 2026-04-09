import { describe, it, expect } from 'vitest'
import { handleErrorResponse, unwrapData } from '../interceptors'

describe('api interceptors', () => {
  it('maps standard status codes to normalized error codes', () => {
    expect(handleErrorResponse(401, {})).toMatchObject({ code: 'AUTH_REQUIRED', status: 401 })
    expect(handleErrorResponse(403, {})).toMatchObject({ code: 'FORBIDDEN', status: 403 })
    expect(handleErrorResponse(404, {})).toMatchObject({ code: 'NOT_FOUND', status: 404 })
    expect(handleErrorResponse(409, {})).toMatchObject({ code: 'CONFLICT', status: 409 })
    expect(handleErrorResponse(422, {})).toMatchObject({ code: 'UNPROCESSABLE', status: 422 })
  })

  it('preserves backend error payload fields', () => {
    const err = handleErrorResponse(400, {
      error: { code: 'ORG_SELECTION_REQUIRED', message: 'choose org' },
      available_orgs: [{ org_id: 'o-1' }]
    })

    expect(err.code).toBe('ORG_SELECTION_REQUIRED')
    expect(err.message).toBe('choose org')
    expect(err.available_orgs).toEqual([{ org_id: 'o-1' }])
  })

  it('unwraps response data consistently', () => {
    expect(unwrapData({ data: { ok: true } })).toEqual({ ok: true })
    expect(unwrapData({ ok: true })).toEqual({ ok: true })
  })
})
