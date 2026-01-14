import request from './request'

// Character 全局人物卡模型 (与TRP3 characteristics 1:1对应)
export interface Character {
  id: number
  user_id: number
  ref_id: string
  game_id: string
  is_npc: boolean
  created_at: string
  updated_at: string

  // TRP3 characteristics 字段
  trp3_version: number
  race: string
  class: string
  first_name: string
  last_name: string
  full_title: string
  title: string
  icon: string
  color: string
  eye_color: string
  age: string
  height: string
  residence: string
  birthplace: string
  misc_info: string   // JSON
  psycho: string      // JSON
  about_text: string  // JSON

  // 用户自定义覆盖字段
  custom_avatar: string
  custom_name: string
  custom_color: string

  // 原始TRP3数据备份
  raw_trp3_data: string
}

export interface CreateCharacterRequest {
  ref_id?: string
  game_id?: string
  is_npc?: boolean
  race?: string
  class?: string
  first_name?: string
  last_name?: string
  full_title?: string
  title?: string
  icon?: string
  color?: string
  eye_color?: string
  age?: string
  height?: string
  residence?: string
  birthplace?: string
  misc_info?: string
  psycho?: string
  about_text?: string
  raw_trp3_data?: string
}

export interface UpdateCharacterRequest {
  race?: string
  class?: string
  first_name?: string
  last_name?: string
  full_title?: string
  title?: string
  icon?: string
  color?: string
  eye_color?: string
  age?: string
  height?: string
  residence?: string
  birthplace?: string
  misc_info?: string
  psycho?: string
  about_text?: string
  custom_avatar?: string
  custom_name?: string
  custom_color?: string
}

export async function listCharacters(): Promise<{ characters: Character[] }> {
  return request.get('/characters')
}

export async function getCharacter(id: number): Promise<Character> {
  return request.get(`/characters/${id}`)
}

export async function createOrUpdateCharacter(data: CreateCharacterRequest): Promise<Character> {
  return request.post('/characters', data)
}

export async function updateCharacter(id: number, data: UpdateCharacterRequest): Promise<Character> {
  return request.put(`/characters/${id}`, data)
}

export async function deleteCharacter(id: number): Promise<void> {
  return request.delete(`/characters/${id}`)
}
