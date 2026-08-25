import type {
  CharacterCard,
  CharacterCardAdditionalInfo,
  CharacterCardImpression,
  CharacterCardPersonalityTrait,
  CharacterCardTRP3Color,
  UpdateCharacterCardRequest,
} from '@/api/characterCard'

export type CharacterCardEditorTab = 'basic' | 'traits' | 'background' | 'impression' | 'other'

export interface CharacterCardDraft extends Omit<UpdateCharacterCardRequest, 'impressions'> {
  impressions: CharacterCardImpression[]
}

export const CHARACTER_CARD_IMPRESSION_SLOTS = 5

function cleanString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

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
      if (
        Number.isInteger(slot)
        && slot >= 1
        && slot <= CHARACTER_CARD_IMPRESSION_SLOTS
        && !bySlot.has(slot)
      ) {
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
      title: cleanString(impression.title),
      text: cleanString(impression.text),
      trp3_icon: cleanString(impression.trp3_icon),
      icon_image_url: cleanString(impression.icon_image_url),
      icon_image_updated_at: impression.icon_image_updated_at || null,
      image_url: cleanString(impression.image_url),
      image_updated_at: impression.image_updated_at || null,
    }
  })
}

export function normalizeCharacterCardAdditionalInfo(
  items: CharacterCardAdditionalInfo[] | null | undefined,
): CharacterCardAdditionalInfo[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    id: Number.isInteger(item?.id) && item.id >= 1 && item.id <= 11 ? item.id : 1,
    name: cleanString(item?.name),
    value: cleanString(item?.value),
    icon: cleanString(item?.icon),
  }))
}

function normalizeTRP3Color(color: CharacterCardTRP3Color | null | undefined): CharacterCardTRP3Color | null {
  if (!color) return null
  const values = [Number(color.r), Number(color.g), Number(color.b)]
  if (values.some((value) => !Number.isFinite(value))) return null
  return {
    r: Math.min(1, Math.max(0, values[0])),
    g: Math.min(1, Math.max(0, values[1])),
    b: Math.min(1, Math.max(0, values[2])),
  }
}

export function normalizeCharacterCardPersonalityTraits(
  items: CharacterCardPersonalityTrait[] | null | undefined,
): CharacterCardPersonalityTrait[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    preset_id: Number.isInteger(item?.preset_id)
      && Number(item.preset_id) >= 1
      && Number(item.preset_id) <= 11
      ? Number(item.preset_id)
      : null,
    left_text: cleanString(item?.left_text),
    right_text: cleanString(item?.right_text),
    left_icon: cleanString(item?.left_icon),
    right_icon: cleanString(item?.right_icon),
    left_color: normalizeTRP3Color(item?.left_color),
    right_color: normalizeTRP3Color(item?.right_color),
    value: Number.isFinite(Number(item?.value))
      ? Math.round(Math.min(20, Math.max(0, Number(item.value))))
      : 10,
  }))
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
    character_id: card.character_id ?? undefined,
    first_name: cleanString(card.first_name),
    last_name: cleanString(card.last_name),
    display_name: cleanString(card.display_name),
    title: cleanString(card.title),
    full_title: cleanString(card.full_title),
    race: cleanString(card.race),
    class: cleanString(card.class),
    eye_color: cleanString(card.eye_color),
    eye_color_hex: cleanString(card.eye_color_hex),
    age: cleanString(card.age),
    height: cleanString(card.height),
    weight: cleanString(card.weight),
    birthplace: cleanString(card.birthplace),
    residence: cleanString(card.residence),
    relationship_status: cleanString(card.relationship_status),
    icon: cleanString(card.icon),
    class_color: cleanString(card.class_color),
    name_color: cleanString(card.name_color),
    additional_info: normalizeCharacterCardAdditionalInfo(card.additional_info),
    personality_traits: normalizeCharacterCardPersonalityTraits(card.personality_traits),
    summary: cleanString(card.summary),
    background_story: cleanString(card.background_story),
    first_impression: cleanString(card.first_impression),
    impressions: normalizeCharacterCardImpressions(card.impressions),
    other_content: cleanString(card.other_content),
    portrait_image_url: cleanString(card.portrait_image_url),
    status: card.status === 'published' ? 'published' : 'draft',
    visibility: card.visibility === 'public' ? 'public' : 'private',
    sort_order: card.sort_order,
  }
}

export function getCharacterCardDisplayName(
  card: Pick<CharacterCardDraft, 'display_name' | 'first_name' | 'last_name'>,
  fallback = '',
): string {
  const explicit = card.display_name.trim()
  if (explicit) return explicit
  return [card.first_name, card.last_name]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(' ') || fallback
}
