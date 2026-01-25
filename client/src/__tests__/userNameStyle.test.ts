import { describe, it, expect } from 'vitest'
import { buildNameStyle } from '../utils/userNameStyle'

describe('buildNameStyle', () => {
  it('returns empty style when no args', () => {
    expect(buildNameStyle()).toEqual({})
  })

  it('applies color and bold', () => {
    expect(buildNameStyle('#fff', true)).toEqual({ color: '#fff', fontWeight: '700' })
  })
})
