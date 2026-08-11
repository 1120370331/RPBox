import { getImageUrl } from './item'
import request from './request'

export type CharacterCardStatus = 'draft' | 'published'
export type CharacterCardVisibility = 'private' | 'public'
export type CharacterCardReviewStatus = 'pending' | 'approved' | 'rejected'
export type CharacterCardSourceType = 'blank' | 'backup'
export type CharacterCardImpressionImageKind = 'icon' | 'image'

export interface CharacterCardPortraitImage {
  id: number
  image_url: string
  image_updated_at: string | null
  sort_order: number
  is_cover: boolean
}

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
  readonly icon_image_updated_at: string | null
  readonly image_updated_at: string | null
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
  icon: string
  class_color: string
  name_color: string
  summary: string
  portrait_image_url?: string | null
  portrait_image_updated_at?: string | null
  portraits?: CharacterCardPortraitImage[]
  impressions?: CharacterCardImpression[]
  status: CharacterCardStatus
  visibility: CharacterCardVisibility
  review_status?: CharacterCardReviewStatus | null
  review_comment?: string | null
  reviewed_at?: string | null
  sort_order?: number
  created_at: string
  updated_at: string
}

export interface CharacterCard extends CharacterCardSummary {
  background_story: string
  first_impression: string
  impressions: CharacterCardImpression[]
  other_content: string
}

export interface CreateCharacterCardRequest {
  source_type: CharacterCardSourceType
  source_backup_id?: number
  source_profile_id?: string
}

export interface UpdateCharacterCardRequest {
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
  const body = unwrapData(response as CharacterCard | DataEnvelope<CharacterCard | { character_card: CharacterCard }>)
  if (body && typeof body === 'object' && 'character_card' in body) {
    return (body as { character_card: CharacterCard }).character_card
  }
  return body as CharacterCard
}

export async function getCharacterCardSources(): Promise<{ sources: CharacterCardSource[] }> {
  const response = await request.get<{ sources: CharacterCardSource[] } | DataEnvelope<{ sources: CharacterCardSource[] }>>(
    '/character-card-sources',
  )
  const body = unwrapData(response)
  return { sources: Array.isArray(body?.sources) ? body.sources : [] }
}

export async function listMyCharacterCards(): Promise<{ character_cards: CharacterCardSummary[] }> {
  const response = await request.get<
    { character_cards: CharacterCardSummary[] } | DataEnvelope<{ character_cards: CharacterCardSummary[] }>
  >('/character-cards')
  const body = unwrapData(response)
  return { character_cards: Array.isArray(body?.character_cards) ? body.character_cards : [] }
}

export async function listUserCharacterCards(userId: number): Promise<{ character_cards: CharacterCardSummary[] }> {
  const response = await request.get<
    { character_cards: CharacterCardSummary[] } | DataEnvelope<{ character_cards: CharacterCardSummary[] }>
  >(`/users/${userId}/character-cards`)
  const body = unwrapData(response)
  return { character_cards: Array.isArray(body?.character_cards) ? body.character_cards : [] }
}

export async function createCharacterCard(payload: CreateCharacterCardRequest): Promise<CharacterCard> {
  const response = await request.post<unknown>('/character-cards', payload)
  return unwrapCharacterCard(response)
}

export async function getCharacterCard(id: number): Promise<CharacterCard> {
  const response = await request.get<unknown>(`/character-cards/${id}`)
  return unwrapCharacterCard(response)
}

export async function getCharacterCardSharePath(id: number): Promise<string> {
  const response = await request.get<{ path: string } | DataEnvelope<{ path: string }>>(`/character-cards/${id}/share`)
  const body = unwrapData(response)
  if (!body?.path) throw new Error('Missing character card share path')
  return body.path
}

export async function updateCharacterCard(
  id: number,
  payload: Partial<UpdateCharacterCardRequest>,
): Promise<CharacterCard> {
  const response = await request.put<unknown>(`/character-cards/${id}`, payload)
  return unwrapCharacterCard(response)
}

export async function deleteCharacterCard(id: number): Promise<void> {
  await request.delete(`/character-cards/${id}`)
}

export async function syncCharacterCardFromTRP3(id: number): Promise<CharacterCard> {
  const response = await request.post<unknown>(`/character-cards/${id}/sync-from-trp3`)
  return unwrapCharacterCard(response)
}

export async function uploadCharacterCardPortrait(file: File): Promise<string> {
  const formData = new FormData()
  formData.append('image', file)
  const response = await request.post<unknown>('/upload/character-card-portrait', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  const body = unwrapData(
    response as { portrait_image_ref?: string } | DataEnvelope<{ portrait_image_ref?: string }>,
  )
  const portraitRef = body?.portrait_image_ref?.trim() || ''
  if (!portraitRef) throw new Error('上传服务没有返回图片引用')
  return portraitRef
}

export async function addCharacterCardPortrait(id: number, imageRef: string): Promise<CharacterCard> {
  const response = await request.post<unknown>(`/character-cards/${id}/portraits`, { image_ref: imageRef })
  return unwrapCharacterCard(response)
}

export async function reorderCharacterCardPortraits(id: number, portraitIds: number[]): Promise<CharacterCard> {
  const response = await request.put<unknown>(`/character-cards/${id}/portraits/order`, {
    portrait_ids: portraitIds,
  })
  return unwrapCharacterCard(response)
}

export async function setCharacterCardPortraitCover(id: number, portraitId: number): Promise<CharacterCard> {
  const response = await request.put<unknown>(`/character-cards/${id}/portraits/${portraitId}/cover`)
  return unwrapCharacterCard(response)
}

export async function deleteCharacterCardPortrait(id: number, portraitId: number): Promise<CharacterCard> {
  const response = await request.delete<unknown>(`/character-cards/${id}/portraits/${portraitId}`)
  return unwrapCharacterCard(response)
}

export interface CharacterCardTRP3Export {
  profile_id: string
  profile: Record<string, unknown>
  lua: string
}

export interface CharacterCardTRP3WriteBack extends CharacterCardTRP3Export {
  backup?: unknown
  snapshot?: unknown
}

export async function getCharacterCardTRP3Lua(id: number): Promise<CharacterCardTRP3Export> {
  const response = await request.get<CharacterCardTRP3Export | DataEnvelope<CharacterCardTRP3Export>>(
    `/character-cards/${id}/trp3-lua`,
  )
  return unwrapData(response)
}

export async function writeBackCharacterCardToTRP3(
  id: number,
  payload: { backup_id?: number; profile_id?: string; snapshot_name?: string },
): Promise<CharacterCardTRP3WriteBack> {
  const response = await request.post<CharacterCardTRP3WriteBack | DataEnvelope<CharacterCardTRP3WriteBack>>(
    `/character-cards/${id}/write-back-trp3`,
    payload,
  )
  return unwrapData(response)
}

export async function uploadCharacterCardImpressionImage(
  file: File,
  kind: CharacterCardImpressionImageKind,
): Promise<string> {
  const formData = new FormData()
  formData.append('image', file)
  formData.append('kind', kind)
  const response = await request.post<unknown>('/upload/character-card-impression-image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  const body = unwrapData(response as { image_ref?: string } | DataEnvelope<{ image_ref?: string }>)
  const imageRef = body?.image_ref?.trim() || ''
  if (!imageRef) throw new Error('上传服务没有返回图片引用')
  return imageRef
}

export function getCharacterCardPortraitUrl(
  card: Pick<CharacterCard, 'id' | 'portrait_image_url' | 'portrait_image_updated_at' | 'updated_at'>,
  options?: { w?: number; q?: number },
): string {
  if (!card.portrait_image_url) return ''
  return getImageUrl('character-card-portrait', card.id, {
    ...options,
    v: card.portrait_image_updated_at || card.updated_at,
  })
}
