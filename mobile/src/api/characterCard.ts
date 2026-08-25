import { request } from '@shared/api/request'

export type CharacterCardStatus = 'draft' | 'published'
export type CharacterCardVisibility = 'private' | 'public'
export type CharacterCardReviewStatus = 'none' | 'pending' | 'approved' | 'rejected'
export type CharacterCardSourceType = 'blank' | 'backup'

export interface CharacterCardPortrait {
  id: number
  image_url: string
  image_updated_at?: string | null
  sort_order: number
  is_cover: boolean
}

export type CharacterCardPortraitImage = CharacterCardPortrait

export interface CharacterCardImpressionUpdate {
  slot: number
  active: boolean
  title: string
  text: string
  trp3_icon: string
  icon_image_url: string
  image_url: string
}

export interface CharacterCardImpression extends CharacterCardImpressionUpdate {
  icon_image_updated_at?: string | null
  image_updated_at?: string | null
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

export interface CharacterCardSource {
  backup_id: number
  account_id: string
  profile_id: string
  profile_name?: string
  display_name?: string
  first_name?: string
  last_name?: string
  title?: string
  race?: string
  class?: string
  icon?: string
  class_color?: string
  name_color?: string
  backup_updated_at?: string
}

export interface CharacterCardSummary {
  id: number
  user_id: number
  character_id?: number | null
  source_backup_id?: number | null
  source_account_id?: string | null
  source_profile_id?: string | null
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
  icon?: string
  class_color: string
  name_color: string
  additional_info?: CharacterCardAdditionalInfo[]
  personality_traits?: CharacterCardPersonalityTrait[]
  summary: string
  portrait_image_url?: string | null
  portrait_image_updated_at?: string | null
  portraits?: CharacterCardPortrait[]
  impressions?: CharacterCardImpression[]
  status: CharacterCardStatus
  visibility: CharacterCardVisibility
  review_status?: CharacterCardReviewStatus | null
  review_comment?: string | null
  reviewed_at?: string | null
  sort_order?: number
  created_at?: string
  updated_at: string
}

export interface CharacterCard extends CharacterCardSummary {
  background_story: string
  first_impression: string
  impressions: CharacterCardImpression[]
  other_content: string
}

export interface CharacterCardShare {
  path: string
  title: string
  summary: string
  updated_at: string
}

export interface CreateCharacterCardRequest {
  source_type: CharacterCardSourceType
  source_backup_id?: number
  source_profile_id?: string
}

export interface UpdateCharacterCardRequest {
  character_id?: number | null
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
  icon: string
  class_color: string
  name_color: string
  additional_info: CharacterCardAdditionalInfo[]
  personality_traits: CharacterCardPersonalityTrait[]
  summary: string
  background_story: string
  first_impression: string
  impressions: CharacterCardImpressionUpdate[]
  other_content: string
  portrait_image_url: string
  status: CharacterCardStatus
  visibility: CharacterCardVisibility
  sort_order?: number
}

interface DataEnvelope<T> {
  data?: T
}

function unwrapData<T>(response: T | DataEnvelope<T>): T {
  if (response && typeof response === 'object' && 'data' in response) {
    const data = (response as DataEnvelope<T>).data
    if (data !== undefined) return data
  }
  return response as T
}

function unwrapCharacterCard(response: unknown): CharacterCard {
  const body = unwrapData(
    response as CharacterCard | DataEnvelope<CharacterCard | { character_card: CharacterCard }>,
  )
  if (body && typeof body === 'object' && 'character_card' in body) {
    return (body as { character_card: CharacterCard }).character_card
  }
  return body as CharacterCard
}

export function isApprovedPublicCharacterCard(
  card: Pick<CharacterCardSummary, 'status' | 'visibility' | 'review_status'> | null | undefined,
) {
  return card?.status === 'published'
    && card.visibility === 'public'
    && card.review_status === 'approved'
}

export async function getCharacterCardSources(): Promise<{ sources: CharacterCardSource[] }> {
  const response = await request.get<
    { sources: CharacterCardSource[] } | DataEnvelope<{ sources: CharacterCardSource[] }>
  >('/character-card-sources')
  const body = unwrapData(response)
  return { sources: Array.isArray(body?.sources) ? body.sources : [] }
}

export async function listMyCharacterCards(): Promise<{ character_cards: CharacterCardSummary[] }> {
  const response = await request.get<
    { character_cards: CharacterCardSummary[] }
    | DataEnvelope<{ character_cards: CharacterCardSummary[] }>
  >('/character-cards')
  const body = unwrapData(response)
  return {
    character_cards: Array.isArray(body?.character_cards) ? body.character_cards : [],
  }
}

export async function createCharacterCard(payload: CreateCharacterCardRequest): Promise<CharacterCard> {
  const response = await request.post<unknown>('/character-cards', payload)
  return unwrapCharacterCard(response)
}

export async function getCharacterCard(id: number): Promise<CharacterCard> {
  const response = await request.get<unknown>(`/character-cards/${id}`)
  return unwrapCharacterCard(response)
}

export async function getCharacterCardShare(id: number): Promise<CharacterCardShare> {
  const response = await request.get<CharacterCardShare | DataEnvelope<CharacterCardShare>>(
    `/character-cards/${id}/share`,
  )
  const body = unwrapData(response)
  if (!body?.path) throw new Error('Missing character card share path')
  return body
}

export async function updateCharacterCard(
  id: number,
  payload: Partial<UpdateCharacterCardRequest>,
): Promise<CharacterCard> {
  const response = await request.put<unknown>(`/character-cards/${id}`, payload)
  return unwrapCharacterCard(response)
}

export async function publishCharacterCard(id: number): Promise<CharacterCard> {
  const response = await request.post<unknown>(`/character-cards/${id}/publish`)
  return unwrapCharacterCard(response)
}

export async function deleteCharacterCard(id: number): Promise<void> {
  await request.delete(`/character-cards/${id}`)
}

export async function uploadCharacterCardPortrait(file: File): Promise<string> {
  const formData = new FormData()
  formData.append('image', file)
  const response = await request.post<
    { portrait_image_ref?: string } | DataEnvelope<{ portrait_image_ref?: string }>
  >('/upload/character-card-portrait', formData)
  const body = unwrapData(response)
  const portraitRef = body?.portrait_image_ref?.trim() || ''
  if (!portraitRef) throw new Error('Missing character-card portrait reference')
  return portraitRef
}

export async function addCharacterCardPortrait(id: number, imageRef: string): Promise<CharacterCard> {
  const response = await request.post<unknown>(`/character-cards/${id}/portraits`, {
    image_ref: imageRef,
  })
  return unwrapCharacterCard(response)
}

export async function setCharacterCardPortraitCover(
  id: number,
  portraitId: number,
): Promise<CharacterCard> {
  const response = await request.put<unknown>(`/character-cards/${id}/portraits/${portraitId}/cover`)
  return unwrapCharacterCard(response)
}

export async function deleteCharacterCardPortrait(
  id: number,
  portraitId: number,
): Promise<CharacterCard> {
  const response = await request.delete<unknown>(`/character-cards/${id}/portraits/${portraitId}`)
  return unwrapCharacterCard(response)
}
