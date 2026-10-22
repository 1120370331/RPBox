import type { CharacterCardSource, CharacterCardSummary } from '@/api/characterCard'

export type CardOrigin = 'local' | 'cloud' | 'rpbox'
export interface LocalCharacterSource {
  account_id: string
  profile_id: string
  profile_name: string
  profile: Record<string, unknown>
}
interface ChoiceBase { key: string; origin: CardOrigin; name: string; detail: string; searchText: string }
export type CharacterCardChoice = ChoiceBase & (
  { kind: 'archive'; card: CharacterCardSummary }
  | { kind: 'cloud'; source: CharacterCardSource }
  | { kind: 'local'; source: LocalCharacterSource }
)
export const cardOriginLabels: Record<CardOrigin, string> = { local: '本地', cloud: '云端', rpbox: 'RPBox 自定义' }

export function cardOrigin(card: CharacterCardSummary): CardOrigin {
  if (card.source_backup_id) return 'cloud'
  return card.source_account_id && card.source_profile_id ? 'local' : 'rpbox'
}
function normalized(value: string) { return value.normalize('NFKC').replace(/\|c[0-9a-f]{8}|\|r/gi, '').toLocaleLowerCase() }
export function matchesCharacterKeywords(searchText: string, query: string) {
  const value = normalized(searchText)
  const compact = value.replace(/[\s·•]+/g, '')
  return normalized(query).trim().split(/\s+/).filter(Boolean).every(word => value.includes(word) || compact.includes(word.replace(/[·•]/g, '')))
}
export function buildCharacterChoices(cards: CharacterCardSummary[], cloud: CharacterCardSource[], local: LocalCharacterSource[]): CharacterCardChoice[] {
  const choices: CharacterCardChoice[] = cards.map(card => {
    const name = card.display_name || [card.first_name, card.last_name].filter(Boolean).join(' ') || '未命名人物'
    return { kind: 'archive', card, key: `rpbox:${card.id}`, origin: cardOrigin(card), name,
      detail: [card.race, card.class, card.title, `RPBox #${card.id}`].filter(Boolean).join(' · '),
      searchText: [name, card.first_name, card.last_name, card.title, card.full_title, card.race, card.class, card.summary, card.source_account_id, card.source_profile_id, String(card.id)].join(' ') }
  })
  for (const source of cloud) {
    if (cards.some(card => card.source_backup_id === source.backup_id && card.source_profile_id === source.profile_id)) continue
    const name = source.display_name || [source.first_name, source.last_name].filter(Boolean).join(' ') || source.profile_name || '未命名人物'
    choices.push({ kind: 'cloud', source, key: `cloud:${JSON.stringify([source.backup_id, source.profile_id])}`, origin: 'cloud', name,
      detail: [source.race, source.class, source.account_id].filter(Boolean).join(' · '),
      searchText: [name, source.profile_name, source.first_name, source.last_name, source.title, source.race, source.class, source.account_id, source.profile_id].join(' ') })
  }
  for (const source of local) {
    if (cards.some(card => cardOrigin(card) === 'local' && card.source_account_id === source.account_id && card.source_profile_id === source.profile_id)) continue
    const player = source.profile.player as Record<string, unknown> | undefined
    const fields = player?.characteristics as Record<string, unknown> | undefined
    const text = (key: string) => typeof fields?.[key] === 'string' ? fields[key] as string : ''
    const name = [text('FN'), text('LN')].filter(Boolean).join(' ') || source.profile_name || '未命名人物'
    choices.push({ kind: 'local', source, key: `local:${JSON.stringify([source.account_id, source.profile_id])}`, origin: 'local', name,
      detail: [text('RA'), text('CL'), source.account_id].filter(Boolean).join(' · '),
      searchText: [name, source.profile_name, text('TI'), text('FT'), text('RA'), text('CL'), source.account_id, source.profile_id].join(' ') })
  }
  return choices
}
