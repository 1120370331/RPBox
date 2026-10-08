import { contrastingTextColor } from '@/utils/displayColor'

// Display-only styles for saved guild colors; never normalize the saved value.
export function guildColorStyle(value?: string) {
  const hex = value?.replace(/^#/, '') || ''
  if (!/^[\da-f]{6}$/i.test(hex)) return {}

  return {
    background: `#${hex}`,
    color: contrastingTextColor(`#${hex}`),
  }
}

export function guildTagStyle(value?: string, selected = false) {
  const style = guildColorStyle(value)
  if (!style.background) return {}
  return {
    '--guild-tag-color': style.background,
    ...(selected ? { background: style.background, color: style.color } : {}),
  }
}
