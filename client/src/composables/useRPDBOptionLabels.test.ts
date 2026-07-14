import { beforeEach, describe, expect, it } from 'vitest'
import i18n from '@/i18n'
import { useRPDBOptionLabels } from './useRPDBOptionLabels'

describe('useRPDBOptionLabels', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
  })

  it('localizes RPDB API enum values without leaking raw codes', () => {
    const { availabilityLabel, bindTypeLabel, factionLabel } = useRPDBOptionLabels()

    expect(availabilityLabel('available')).toBe('可获取')
    expect(availabilityLabel('limited')).toBe('限时获取')
    expect(bindTypeLabel('no')).toBe('否')
    expect(bindTypeLabel('yes')).toBe('是')
    expect(factionLabel('neutral')).toBe('不限')
    expect(factionLabel('neutra')).toBe('不限')
    expect(factionLabel('alliance')).toBe('联盟')
    expect(availabilityLabel('future_status')).toBe('未知')
  })
})
