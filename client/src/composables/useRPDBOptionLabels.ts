import type { RPDBWorkType } from '@/api/rpdb'
import i18n from '@/i18n'

const availabilityValues = new Set(['available', 'limited', 'removed', 'unknown'])
const visitValues = new Set(['friend_only', 'closed'])
const factionAliases: Record<string, 'neutral' | 'alliance' | 'horde'> = {
  neutral: 'neutral',
  neutra: 'neutral',
  alliance: 'alliance',
  horde: 'horde',
}

function normalize(value?: string) {
  return String(value || '').trim().toLowerCase()
}

function preserveLocalizedValue(value: string | undefined, fallback: string) {
  const displayValue = String(value || '').trim()
  if (!displayValue || /^[a-z0-9_-]+$/i.test(displayValue)) return fallback
  return displayValue
}

export function useRPDBOptionLabels() {
  const t = (key: string) => i18n.global.t(key)

  function availabilityLabel(value?: string, type?: RPDBWorkType) {
    const normalized = normalize(value)
    if (type === 'home_showcase' && visitValues.has(normalized)) {
      return t(`rpdb.editor.options.visit.${normalized}.label`)
    }
    if (availabilityValues.has(normalized)) {
      return t(`rpdb.editor.options.availability.${normalized}.label`)
    }
    return preserveLocalizedValue(value, t('rpdb.editor.options.availability.unknown.label'))
  }

  function bindTypeLabel(value?: string) {
    const normalized = normalize(value)
    if (normalized === 'yes' || normalized === 'no') {
      return t(`rpdb.editor.options.bind.${normalized}.label`)
    }
    if (normalized === 'account' || normalized === 'pickup' || normalized === 'use') {
      return t(`rpdb.editor.options.bind.${normalized}.label`)
    }
    return preserveLocalizedValue(value, t('rpdb.editor.options.bind.unknown.label'))
  }

  function factionLabel(value?: string) {
    const normalized = factionAliases[normalize(value)]
    if (normalized) return t(`rpdb.editor.options.faction.${normalized}.label`)
    return preserveLocalizedValue(value, t('rpdb.editor.options.faction.neutral.label'))
  }

  return {
    availabilityLabel,
    bindTypeLabel,
    factionLabel,
  }
}
