import { request } from '@shared/api/request'

export interface CharacterCardPortrait {
  id: number
  image_url: string
  image_updated_at?: string | null
  sort_order: number
  is_cover: boolean
}

export interface CharacterCardImpression {
  slot: number
  active: boolean
  title: string
  text: string
  trp3_icon: string
  icon_image_url: string
  image_url: string
}

export interface CharacterCardTRP3Color {
  r: number
  g: number
  b: number
}

export interface CharacterCardAdditionalInfo {
  id: number
  name: string
  value: string
  icon: string
}

export interface CharacterCardPersonalityTrait {
  preset_id: number | null
  left_text: string
  right_text: string
  left_icon: string
  right_icon: string
  left_color: CharacterCardTRP3Color | null
  right_color: CharacterCardTRP3Color | null
  value: number
}

export interface CharacterCard {
  id: number
  user_id: number
  first_name: string
  last_name: string
  display_name: string
  title: string
  full_title: string
  race: string
  class: string
  eye_color: string
  eye_color_hex: string
  age: string
  height: string
  weight: string
  birthplace: string
  residence: string
  relationship_status: string
  class_color: string
  name_color: string
  additional_info?: CharacterCardAdditionalInfo[]
  personality_traits?: CharacterCardPersonalityTrait[]
  summary: string
  background_story: string
  first_impression: string
  impressions: CharacterCardImpression[]
  other_content: string
  portrait_image_url?: string | null
  portrait_image_updated_at?: string | null
  portraits?: CharacterCardPortrait[]
  status: 'draft' | 'published'
  visibility: 'private' | 'public'
  review_status?: 'none' | 'pending' | 'approved' | 'rejected' | null
  updated_at: string
}

export interface CharacterCardShare {
  path: string
  title: string
  summary: string
  updated_at: string
}

interface CharacterCardResponse {
  character_card: CharacterCard
}

interface CharacterCardListResponse {
  character_cards: CharacterCard[]
}

export function isApprovedPublicCharacterCard(
  card: Pick<CharacterCard, 'status' | 'visibility' | 'review_status'> | null | undefined,
) {
  return card?.status === 'published'
    && card.visibility === 'public'
    && card.review_status === 'approved'
}

export async function listMyCharacterCards(): Promise<CharacterCardListResponse> {
  const response = await request.get<CharacterCardListResponse>('/character-cards')
  return {
    character_cards: Array.isArray(response?.character_cards) ? response.character_cards : [],
  }
}

export async function getCharacterCard(id: number): Promise<CharacterCard> {
  const response = await request.get<CharacterCardResponse>(`/character-cards/${id}`)
  return response.character_card
}

export async function getCharacterCardShare(id: number): Promise<CharacterCardShare> {
  const response = await request.get<CharacterCardShare>(`/character-cards/${id}/share`)
  if (!response?.path) throw new Error('Missing character card share path')
  return response
}
