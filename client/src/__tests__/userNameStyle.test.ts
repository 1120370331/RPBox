import { beforeEach, describe, it, expect } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { computed } from 'vue'
import { buildNameStyle } from '../utils/userNameStyle'
import { themes, useThemeStore } from '../stores/theme'
import { compositeDisplayColor, displayContrast, parseDisplayColor } from '../utils/displayColor'
import { normalizeCharacterCardHexForCSS } from '../utils/characterCardColor'

function contrast(color: string, background: string, base = background) {
  const back = compositeDisplayColor(parseDisplayColor(background)!, parseDisplayColor(base)!)
  return displayContrast(compositeDisplayColor(parseDisplayColor(color)!, back), back)
}

describe('buildNameStyle', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })
  it('returns empty style when no args', () => {
    expect(buildNameStyle()).toEqual({})
  })

  it('applies color and bold', () => {
    expect(buildNameStyle('#fff', true, '#181818')).toEqual({ color: '#fff', fontWeight: '700' })
  })

  it('adapts extreme and alpha dyes to actual panel, card, hover, sidebar and gradient backgrounds', () => {
    const theme = useThemeStore()
    const saved = Object.freeze({ name_color: '#000000', name_bold: true, trp3: '80000000' })
    const surfaces = ['panelBg', 'cardBg', 'cardBgHover', 'sidebarBg', 'gradientStart', 'gradientEnd'] as const
    const dyes = ['#000000', '#FFFFFF', '#FFF468', '#804030', '#B87333', '#69CCF0', '#FF7D0A', 'rgba(0, 0, 0, 0.2)', normalizeCharacterCardHexForCSS(saved.trp3)]
    for (const preset of themes) {
      theme.setTheme(preset.id)
      for (const dye of dyes) {
        for (const surface of surfaces) {
          const style = buildNameStyle(dye, saved.name_bold, surface)
          expect(contrast(style.color!, preset.colors[surface], preset.colors.panelBg)).toBeGreaterThanOrEqual(4.5)
          expect(style.fontWeight).toBe('700')
        }
        const panel = buildNameStyle(dye).color!
        for (const surface of surfaces.slice(0, 3)) {
          expect(contrast(panel, preset.colors[surface], preset.colors.panelBg)).toBeGreaterThanOrEqual(4.5)
        }
        const gradient = buildNameStyle(dye, false, ['gradientStart', 'gradientEnd']).color!
        for (const surface of ['gradientStart', 'gradientEnd'] as const) {
          expect(contrast(gradient, preset.colors[surface])).toBeGreaterThanOrEqual(4.5)
        }
      }
    }
    expect(saved).toEqual({ name_color: '#000000', name_bold: true, trp3: '80000000' })
  })

  it('updates existing computed display styles when the theme changes while preserving readable dyes', () => {
    const theme = useThemeStore()
    const name = computed(() => buildNameStyle('#000000', true))
    expect(name.value.color).toBe('#000000')
    theme.setTheme('black-gold')
    expect(name.value.color).not.toBe('#000000')
    expect(contrast(name.value.color!, theme.currentTheme.colors.panelBg)).toBeGreaterThanOrEqual(4.5)
    expect(buildNameStyle('#FFFFFF', true).color).toBe('#FFFFFF')
    theme.setTheme('classic')
    expect(name.value).toEqual({ color: '#000000', fontWeight: '700' })
  })
})
