import { request } from '@shared/api/request'

export interface Character {
  id: number
  is_npc: boolean
  game_id?: string
  first_name?: string
  last_name?: string
  icon?: string
  color?: string
  custom_name?: string
  custom_color?: string
  custom_avatar?: string
}

export function getCharacter(id: number) {
  return request.get<Character>(`/characters/${id}`)
}
