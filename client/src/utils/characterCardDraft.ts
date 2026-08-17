import type {
  CharacterCard,
  CharacterCardAdditionalInfo,
  CharacterCardImpression,
  CharacterCardPersonalityTrait,
  UpdateCharacterCardRequest,
} from '@/api/characterCard'

export type CharacterCardEditorTab = 'basic' | 'traits' | 'background' | 'impression' | 'other'

export interface CharacterCardDraft extends UpdateCharacterCardRequest {}

const CHARACTER_CARD_IMPRESSION_SLOTS = 5

function createEmptyCharacterCardImpression(slot: number): CharacterCardImpression {
  return {
    slot,
    active: false,
    title: '',
    text: '',
    trp3_icon: '',
    icon_image_url: '',
    icon_image_updated_at: null,
    image_url: '',
    image_updated_at: null,
  }
}

export function createEmptyCharacterCardImpressions(): CharacterCardImpression[] {
  return Array.from(
    { length: CHARACTER_CARD_IMPRESSION_SLOTS },
    (_, index) => createEmptyCharacterCardImpression(index + 1),
  )
}

export function normalizeCharacterCardImpressions(
  impressions: CharacterCardImpression[] | null | undefined,
): CharacterCardImpression[] {
  const bySlot = new Map<number, CharacterCardImpression>()
  if (Array.isArray(impressions)) {
    for (const impression of impressions) {
      const slot = Number(impression?.slot)
      if (Number.isInteger(slot) && slot >= 1 && slot <= CHARACTER_CARD_IMPRESSION_SLOTS && !bySlot.has(slot)) {
        bySlot.set(slot, impression)
      }
    }
  }

  return createEmptyCharacterCardImpressions().map((empty) => {
    const impression = bySlot.get(empty.slot)
    if (!impression) return empty
    return {
      slot: empty.slot,
      active: impression.active === true,
      title: impression.title || '',
      text: impression.text || '',
      trp3_icon: impression.trp3_icon || '',
      icon_image_url: impression.icon_image_url || '',
      icon_image_updated_at: impression.icon_image_updated_at || null,
      image_url: impression.image_url || '',
      image_updated_at: impression.image_updated_at || null,
    }
  })
}

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
    class_color: '',
    name_color: '',
    additional_info: [],
    personality_traits: [],
    summary: '',
    background_story: '',
    first_impression: '',
    impressions: createEmptyCharacterCardImpressions(),
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
    class_color: card.class_color || card.name_color || '',
    name_color: card.name_color || '',
    additional_info: normalizeCharacterCardAdditionalInfo(card.additional_info),
    personality_traits: normalizeCharacterCardPersonalityTraits(card.personality_traits),
    summary: card.summary || '',
    background_story: card.background_story || '',
    first_impression: card.first_impression || '',
    impressions: normalizeCharacterCardImpressions(card.impressions),
    other_content: card.other_content || '',
    portrait_image_url: card.portrait_image_url || '',
    status: card.status === 'published' ? 'published' : 'draft',
    visibility: card.visibility === 'public' ? 'public' : 'private',
    sort_order: card.sort_order,
  }
}

function normalizeCharacterCardAdditionalInfo(items: CharacterCardAdditionalInfo[] | null | undefined): CharacterCardAdditionalInfo[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    id: Number.isInteger(item?.id) ? item.id : 1,
    name: item?.name || '',
    value: item?.value || '',
    icon: item?.icon || '',
  }))
}

function normalizeCharacterCardPersonalityTraits(items: CharacterCardPersonalityTrait[] | null | undefined): CharacterCardPersonalityTrait[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    preset_id: Number.isInteger(item?.preset_id) ? item.preset_id : null,
    left_text: item?.left_text || '',
    right_text: item?.right_text || '',
    left_icon: item?.left_icon || '',
    right_icon: item?.right_icon || '',
    left_color: item?.left_color ? { ...item.left_color } : null,
    right_color: item?.right_color ? { ...item.right_color } : null,
    value: Number.isInteger(item?.value) ? Math.min(20, Math.max(0, item.value)) : 10,
  }))
}

export function getCharacterCardDisplayName(
  card: Pick<CharacterCardDraft, 'display_name' | 'first_name' | 'last_name'>,
): string {
  const explicit = card.display_name.trim()
  if (explicit) return explicit
  return [card.first_name, card.last_name].map((part) => part.trim()).filter(Boolean).join(' ') || '未命名人物'
}
