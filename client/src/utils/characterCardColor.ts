const RGB_HEX = /^[0-9a-f]{6}$/i
const EIGHT_DIGIT_HEX = /^[0-9a-f]{8}$/i

/** Converts TRP3 RGB/ARGB values into CSS RGB/RGBA syntax. */
export function normalizeCharacterCardHexForCSS(value?: string | null): string {
  const color = value?.trim() || ''
  if (RGB_HEX.test(color)) return `#${color}`
  if (EIGHT_DIGIT_HEX.test(color)) {
    const alpha = color.slice(0, 2)
    return `#${color.slice(2)}${alpha}`
  }
  return color
}

/** Converts CSS RGB/RGBA values back to TRP3 RGB/ARGB syntax. */
export function normalizeCharacterCardHexForTRP3(value?: string | null): string {
  const color = value?.trim() || ''
  if (/^#[0-9a-f]{6}$/i.test(color)) return color.slice(1)
  if (/^#[0-9a-f]{8}$/i.test(color)) {
    const rgba = color.slice(1)
    const alpha = rgba.slice(6)
    return `${alpha}${rgba.slice(0, 6)}`
  }
  return color
}

/** Returns the RPBox/TRP3 name dye, preferring the explicit name field for legacy cards. */
export function getCharacterCardDisplayColor(
  card?: { name_color?: string | null; class_color?: string | null } | null,
): string {
  return normalizeCharacterCardHexForCSS(card?.name_color || card?.class_color)
}
