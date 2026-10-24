import { gameFlavorLabel } from './gameInstallations'
import type { CharacterCardSource, CharacterCardSummary } from '@/api/characterCard'

export interface LocalCharacterSource {
  game_flavor?: string
  account_id: string
  profile_id: string
  profile_name: string
  profile: Record<string, unknown>
}
interface ChoiceBase { key: string; name: string; detail: string; searchText: string }
export type CharacterCardChoice = ChoiceBase & (
  { kind: 'archive'; card: CharacterCardSummary }
  | { kind: 'cloud'; source: CharacterCardSource }
  | { kind: 'local'; source: LocalCharacterSource }
)
function normalized(value: string) { return value.normalize('NFKC').replace(/\|c[0-9a-f]{8}|\|r/gi, '').toLocaleLowerCase() }
export function matchesCharacterKeywords(searchText: string, query: string) {
  const value = normalized(searchText)
  const compact = value.replace(/[\s·•]+/g, '')
  return normalized(query).trim().split(/\s+/).filter(Boolean).every(word => value.includes(word) || compact.includes(word.replace(/[·•]/g, '')))
}
export function buildCharacterChoices(cards: CharacterCardSummary[], cloud: CharacterCardSource[], local: LocalCharacterSource[]): CharacterCardChoice[] {
  const choices: CharacterCardChoice[] = cards.map(card => {
    const name = card.display_name || [card.first_name, card.last_name].filter(Boolean).join(' ') || '未命名人物'
    return { kind: 'archive', card, key: `rpbox:${card.id}`, name,
      detail: [card.source_account_id, card.race, card.class, `RPBox #${card.id}`].filter(Boolean).join(' · '),
      searchText: [name, card.first_name, card.last_name].join(' ') }
  })
  for (const source of cloud) {
    if (source.linked_card_id && cards.some(card => card.id === source.linked_card_id)) continue
    if (cards.some(card => card.source_backup_id === source.backup_id && card.source_profile_id === source.profile_id)) continue
    const name = source.display_name || [source.first_name, source.last_name].filter(Boolean).join(' ') || source.profile_name || '未命名人物'
    choices.push({ kind: 'cloud', source, key: `cloud:${JSON.stringify([source.backup_id, source.profile_id])}`, name,
      detail: [source.race, source.class, source.account_id, gameFlavorLabel(source.game_flavor)].filter(Boolean).join(' · '),
      searchText: [name, source.profile_name, source.first_name, source.last_name].join(' ') })
  }
  for (const source of local) {
    if (cards.some(card => !card.source_backup_id && (card.source_game_flavor || '') === (source.game_flavor || '') && card.source_account_id === source.account_id && card.source_profile_id === source.profile_id)) continue
    const player = source.profile.player as Record<string, unknown> | undefined
    const fields = player?.characteristics as Record<string, unknown> | undefined
    const text = (key: string) => typeof fields?.[key] === 'string' ? fields[key] as string : ''
    const name = [text('FN'), text('LN')].filter(Boolean).join(' ') || source.profile_name || '未命名人物'
    choices.push({ kind: 'local', source, key: `local:${JSON.stringify([source.game_flavor || '', source.account_id, source.profile_id])}`, name,
      detail: [text('RA'), text('CL'), source.account_id, gameFlavorLabel(source.game_flavor)].filter(Boolean).join(' · '),
      searchText: [name, source.profile_name].join(' ') })
  }
  return choices
}
