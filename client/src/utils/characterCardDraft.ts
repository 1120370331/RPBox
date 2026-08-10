import type { CharacterCard, UpdateCharacterCardRequest } from '@/api/characterCard'

export type CharacterCardEditorTab = 'basic' | 'background' | 'impression' | 'other'

export interface CharacterCardDraft extends UpdateCharacterCardRequest {}

export function createEmptyCharacterCardDraft(): CharacterCardDraft {
  return {
    first_name: '',
    last_name: '',
    display_name: '',
    title: '',
    full_title: '',
    race: '',
    class: '',
    eye_color: '',
    eye_color_hex: '',
    age: '',
    height: '',
    weight: '',
    birthplace: '',
    residence: '',
    relationship_status: '',
    icon: '',
    name_color: '',
    summary: '',
    background_story: '',
    first_impression: '',
    other_content: '',
    portrait_image_url: '',
    status: 'draft',
    visibility: 'private',
  }
}

export function createCharacterCardDraft(card?: CharacterCard | null): CharacterCardDraft {
  const empty = createEmptyCharacterCardDraft()
  if (!card) return empty

  return {
    ...empty,
    first_name: card.first_name || '',
    last_name: card.last_name || '',
    display_name: card.display_name || '',
    title: card.title || '',
    full_title: card.full_title || '',
    race: card.race || '',
    class: card.class || '',
    eye_color: card.eye_color || '',
    eye_color_hex: card.eye_color_hex || '',
    age: card.age || '',
    height: card.height || '',
    weight: card.weight || '',
    birthplace: card.birthplace || '',
    residence: card.residence || '',
    relationship_status: card.relationship_status || '',
    icon: card.icon || '',
    name_color: card.name_color || '',
    summary: card.summary || '',
    background_story: card.background_story || '',
    first_impression: card.first_impression || '',
    other_content: card.other_content || '',
    portrait_image_url: card.portrait_image_url || '',
    status: card.status === 'published' ? 'published' : 'draft',
    visibility: card.visibility === 'public' ? 'public' : 'private',
    sort_order: card.sort_order,
  }
}

export function getCharacterCardDisplayName(
  card: Pick<CharacterCardDraft, 'display_name' | 'first_name' | 'last_name'>,
): string {
  const explicit = card.display_name.trim()
  if (explicit) return explicit
  return [card.first_name, card.last_name].map((part) => part.trim()).filter(Boolean).join(' ') || '未命名人物'
}
