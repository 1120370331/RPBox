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
  summary: string
  background_story: string
  first_impression: string
  impressions: CharacterCardImpression[]
  other_content: string
  portrait_image_url?: string | null
  portraits?: CharacterCardPortrait[]
  status: 'draft' | 'published'
  visibility: 'private' | 'public'
  review_status?: 'pending' | 'approved' | 'rejected' | null
}

interface CharacterCardResponse {
  character_card: CharacterCard
}

export async function getCharacterCard(id: number): Promise<CharacterCard> {
  const response = await request.get<CharacterCardResponse>(`/character-cards/${id}`)
  return response.character_card
}
