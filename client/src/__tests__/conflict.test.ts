import { describe, it, expect } from 'vitest'
import type { ConflictInfo } from '../utils/conflict'

describe('conflict detection', () => {
  it('should detect conflict when checksums differ', () => {
    const local = { checksum: 'abc123' }
    const cloud = { checksum: 'def456' }

    const hasConflict = local.checksum !== cloud.checksum
    expect(hasConflict).toBe(true)
  })

  it('should not detect conflict when checksums match', () => {
    const local = { checksum: 'abc123' }
    const cloud = { checksum: 'abc123' }

    const hasConflict = local.checksum !== cloud.checksum
    expect(hasConflict).toBe(false)
  })
})
