export interface DisplayColor {
  r: number
  g: number
  b: number
  a: number
}

/** Parses CSS RGB/RGBA dyes. TRP3 ARGB must be normalized by the caller first. */
export function parseDisplayColor(value: string): DisplayColor | null {
  const named: Record<string, string> = { black: '#000000', white: '#ffffff', transparent: '#00000000' }
  const color = named[value.trim().toLowerCase()] || value.trim()
  const hex = color.match(/^#([\da-f]{3,4}|[\da-f]{6}|[\da-f]{8})$/i)
  if (hex) {
    const digits = hex[1]!.length <= 4 ? [...hex[1]!].map(digit => digit + digit).join('') : hex[1]!
    return {
      r: parseInt(digits.slice(0, 2), 16), g: parseInt(digits.slice(2, 4), 16),
      b: parseInt(digits.slice(4, 6), 16), a: digits.length === 8 ? parseInt(digits.slice(6), 16) / 255 : 1,
    }
  }
  const rgb = color.match(/^rgba?\(\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)(?:\s*,\s*(\d*(?:\.\d+)?))?\s*\)$/i)
  if (!rgb) return null
  return { r: Math.min(255, Number(rgb[1])), g: Math.min(255, Number(rgb[2])), b: Math.min(255, Number(rgb[3])), a: Math.min(1, Number(rgb[4] ?? 1)) }
}

/** Composites a display color over an opaque background. */
export function compositeDisplayColor(color: DisplayColor, background: DisplayColor): DisplayColor {
  return {
    r: Math.round(color.r * color.a + background.r * (1 - color.a)),
    g: Math.round(color.g * color.a + background.g * (1 - color.a)),
    b: Math.round(color.b * color.a + background.b * (1 - color.a)), a: 1,
  }
}

/** Returns WCAG relative luminance. */
export function displayLuminance(color: DisplayColor): number {
  const channel = (value: number) => {
    const normalized = value / 255
    return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4
  }
  return channel(color.r) * 0.2126 + channel(color.g) * 0.7152 + channel(color.b) * 0.0722
}

/** Returns the contrast ratio of opaque display colors. */
export function displayContrast(front: DisplayColor, back: DisplayColor): number {
  const a = displayLuminance(front)
  const b = displayLuminance(back)
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

/** Serializes an opaque display color. */
export function displayColorCSS(color: DisplayColor): string {
  return `rgb(${color.r}, ${color.g}, ${color.b})`
}

/** Chooses the higher-contrast foreground for a filled saved-color swatch. */
export function contrastingTextColor(background: string): string {
  const color = parseDisplayColor(background)
  return color && displayLuminance(color) > 0.179 ? '#000000' : '#ffffff'
}

/** Adjusts only the display value; backgrounds are alternate surfaces/gradient endpoints. */
export function readableDisplayColor(value: string, backgrounds: readonly DisplayColor[], fallback: string): string {
  const firstBackground = backgrounds[0]
  if (!firstBackground) return fallback
  const parsed = parseDisplayColor(value) || parseDisplayColor(fallback)
  if (!parsed) return fallback
  const worstContrast = (color: DisplayColor) => Math.min(...backgrounds.map(back => displayContrast(compositeDisplayColor(color, back), back)))
  if (worstContrast(parsed) >= 4.5) return value && parseDisplayColor(value) ? value : fallback

  const front = compositeDisplayColor(parsed, firstBackground)
  const black = { r: 0, g: 0, b: 0, a: 1 }
  const white = { r: 255, g: 255, b: 255, a: 1 }
  const target = worstContrast(black) > worstContrast(white) ? black : white
  let low = 0
  let high = 1
  let result = target
  for (let step = 0; step < 16; step++) {
    const amount = (low + high) / 2
    const candidate = compositeDisplayColor({ ...target, a: amount }, front)
    if (worstContrast(candidate) >= 4.5) { high = amount; result = candidate }
    else low = amount
  }
  return displayColorCSS(result)
}
