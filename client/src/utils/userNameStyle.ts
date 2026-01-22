export function buildNameStyle(color?: string, bold?: boolean) {
  const style: Record<string, string> = {}
  if (color) {
    style.color = color
  }
  if (bold) {
    style.fontWeight = '700'
  }
  return style
}
