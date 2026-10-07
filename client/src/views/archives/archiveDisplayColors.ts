import { useThemeStore } from '@/stores/theme'

import { compositeDisplayColor, displayColorCSS, parseDisplayColor, readableDisplayColor } from '@/utils/displayColor'

/** Display-only archive dyes; never write adjusted colors into saved records. */
export function useArchiveDisplayColors() {
  const theme = useThemeStore()

  function readableArchiveColor(value: string, background?: string): string {
    const colors = theme.currentTheme.colors
    const base = parseDisplayColor(colors.panelBg)!
    const back = compositeDisplayColor(parseDisplayColor(background || colors.panelBg) || base, base)
    const display = readableDisplayColor(value, [back], colors.textMain)
    return displayColorCSS(compositeDisplayColor(parseDisplayColor(display)!, back))
  }

  return { readableArchiveColor }
}
