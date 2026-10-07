import { getActivePinia } from 'pinia'
import { getThemeById, useThemeStore, type ThemeColors } from '@/stores/theme'
import { compositeDisplayColor, parseDisplayColor, readableDisplayColor } from './displayColor'

export type NameBackground = keyof ThemeColors | string | readonly string[]

/** Adapts a saved dye to current display surfaces without modifying the saved value. */
export function readableNameColor(color: string, background: NameBackground = ['panelBg', 'cardBg', 'cardBgHover', 'primaryLight', 'btnOutlineHover']): string {
  const colors = getActivePinia() ? useThemeStore().currentTheme.colors : getThemeById('classic').colors
  const base = parseDisplayColor(colors.panelBg)!
  const surfaces = typeof background === 'string' ? [background] : background
  const backgrounds = surfaces.flatMap(surface => {
    const value = Object.prototype.hasOwnProperty.call(colors, surface) ? colors[surface as keyof ThemeColors] : surface
    const front = parseDisplayColor(value) || base
    const backdrops = surface === 'cardBgHover' ? [base, parseDisplayColor(colors.cardBg)!] : [base]
    return backdrops.map(backdrop => compositeDisplayColor(front, backdrop))
  })
  return readableDisplayColor(color, backgrounds, colors.textMain)
}

/** Builds display-only name styling; the optional third argument describes the actual background. */
export function buildNameStyle(color?: string, bold?: boolean, background?: NameBackground) {
  const style: Record<string, string> = {}
  if (color) {
    style.color = readableNameColor(color, background)
  }
  if (bold) {
    style.fontWeight = '700'
  }
  return style
}
