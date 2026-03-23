import { describe, expect, it } from 'vitest'
import { unwrapApiPayload } from '@shared/api/request'

describe('unwrapApiPayload', () => {
  it('returns normal JSON payload directly', () => {
    const payload = { posts: [{ id: 1 }], total: 1 }
    expect(unwrapApiPayload(payload)).toEqual(payload)
  })

  it('unwraps { code: 0, data } envelope payload', () => {
    const payload = { code: 0, data: { items: [{ id: 2 }], total: 1 } }
    expect(unwrapApiPayload(payload)).toEqual({ items: [{ id: 2 }], total: 1 })
  })

  it('throws on non-zero business code', () => {
    expect(() => unwrapApiPayload({ code: 5001, message: 'forbidden' })).toThrowError()
  })
})
