import type { RPDBGuideStep, RPDBWork, RPDBWorkType } from '@/api/rpdb'

const typeLabels: Record<RPDBWorkType, string> = {
  item_showcase: '魔兽物品',
  transmog: '幻化方案',
  home_showcase: '家宅分享',
  musician_midi: 'Musician MIDI',
}

const typeIcons: Record<RPDBWorkType, string> = {
  item_showcase: 'ri-magic-line',
  transmog: 'ri-shirt-line',
  home_showcase: 'ri-home-heart-line',
  musician_midi: 'ri-music-2-line',
}

export function getRPDBTypeLabel(type: RPDBWorkType) {
  return typeLabels[type]
}

export function getRPDBTypeIcon(type: RPDBWorkType) {
  return typeIcons[type]
}

export function getRPDBSummary(work: Pick<RPDBWork, 'summary' | 'effect_description' | 'rp_use_cases'>) {
  return work.summary?.trim()
    || work.effect_description?.trim()
    || work.rp_use_cases?.trim()
    || '作者尚未填写作品摘要'
}

export function parseRPDBExtra(value?: string): Record<string, string> {
  if (!value) return {}
  try {
    const parsed = JSON.parse(value)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? Object.fromEntries(Object.entries(parsed).map(([key, item]) => [key, String(item ?? '')]))
      : {}
  } catch {
    return {}
  }
}

export function buildTomTomCommand(step: RPDBGuideStep) {
  if (!Number.isFinite(step.x) || !Number.isFinite(step.y)) return ''
  const map = step.map_id?.trim()
  const label = step.label?.trim() || step.title.trim()
  return `/way${map ? ` ${map}` : ''} ${step.x} ${step.y}${label ? ` ${label}` : ''}`
}

export function qualityClass(quality?: string) {
  const normalized = String(quality || 'common').toLowerCase().replace(/[^a-z0-9_-]/g, '')
  return `quality-${normalized || 'common'}`
}
