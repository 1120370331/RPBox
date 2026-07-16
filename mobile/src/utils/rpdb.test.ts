import { describe, expect, it } from 'vitest'
import { buildTomTomCommand, getRPDBSummary, parseRPDBExtra, qualityClass } from './rpdb'

describe('rpdb helpers', () => {
  it('chooses the first useful work summary', () => {
    expect(getRPDBSummary({ summary: '', effect_description: '烛光效果', rp_use_cases: '仪式' })).toBe('烛光效果')
  })

  it('parses extra data without leaking malformed JSON errors', () => {
    expect(parseRPDBExtra('{"share_code":"ABC"}')).toEqual({ share_code: 'ABC' })
    expect(parseRPDBExtra('{broken')).toEqual({})
  })

  it('builds TomTom commands only for coordinate steps', () => {
    expect(buildTomTomCommand({ sort_order: 1, title: '入口', map_id: '84', x: 45.2, y: 61.8 })).toBe('/way 84 45.2 61.8 入口')
    expect(buildTomTomCommand({ sort_order: 2, title: '无坐标' })).toBe('')
  })

  it('sanitizes quality class values', () => {
    expect(qualityClass('Legendary')).toBe('quality-legendary')
    expect(qualityClass('"><script>')).toBe('quality-script')
  })
})
