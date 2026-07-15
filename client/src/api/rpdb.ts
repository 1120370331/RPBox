import request from './request'

const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.DEV ? '/api/v1' : 'http://localhost:8080/api/v1')
const API_HOST = API_BASE.replace(/\/api\/v1\/?$/, '')

export type RPDBWorkType = 'item_showcase' | 'transmog' | 'home_showcase'
export type RPDBVerificationStatus = 'unverified' | 'verified' | 'stale' | 'disputed'
export type RPDBListStatus = 'wanted' | 'farming' | 'owned' | 'paused'
export type RPDBVisibility = 'public' | 'guild' | 'private'

export interface RPDBReference {
  id?: number
  work_id?: number
  external_type: string
  external_id: string
  name: string
  icon?: string
  quality?: string
  description?: string
  acquisition_method?: string
  source?: string
  url?: string
  locale?: string
  is_primary?: boolean
  sort_order?: number
}

export interface RPDBMedia {
  id?: number
  work_id?: number
  type: 'image' | 'gif' | 'video' | 'embed'
  url: string
  thumbnail_url?: string
  caption?: string
  sort_order?: number
  review_status?: string
}

export interface RPDBTransmogSlot {
  id?: number
  work_id?: number
  reference_id?: number
  slot: string
  role?: 'unused' | 'required' | 'optional' | 'variant'
  name?: string
  description?: string
  source?: string
  wowhead_url?: string
  variant?: string
  note?: string
  sort_order?: number
}

export interface RPDBGuideStep {
  id?: number
  work_id?: number
  sort_order: number
  title: string
  body?: string
  zone?: string
  map_id?: string
  x?: number
  y?: number
  label?: string
  prerequisite?: string
}

export interface RPDBTag {
  id: number
  name: string
  color: string
}

export interface RPDBWork {
  id: number
  author_id: number
  author_name?: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  type: RPDBWorkType
  title: string
  slug: string
  summary: string
  content: string
  content_type: string
  cover_image: string
  cover_image_updated_at?: string
  rp_use_cases: string
  effect_description: string
  restrictions: string
  extra: string
  game_version: string
  expansion: string
  availability_status: string
  bind_type: string
  item_type?: string
  faction: string
  armor_type: string
  verification_status: RPDBVerificationStatus
  last_verified_at?: string
  verified_count: number
  outdated_count: number
  status: string
  is_public: boolean
  visibility: RPDBVisibility
  guild_id?: number
  guild_ids?: number[]
  review_status: string
  review_comment?: string
  version: number
  view_count: number
  like_count: number
  favorite_count: number
  comment_count: number
  list_count: number
  media_count: number
  is_liked: boolean
  is_favorited: boolean
  in_collection_list: boolean
  references?: RPDBReference[]
  media?: RPDBMedia[]
  transmog_slots?: RPDBTransmogSlot[]
  guide_steps?: RPDBGuideStep[]
  tags?: RPDBTag[]
  created_at: string
  updated_at: string
}

export interface RPDBComment {
  id: number
  work_id: number
  author_id: number
  author_name?: string
  author_avatar?: string
  author_name_color?: string
  author_name_bold?: boolean
  author_forum_level?: number
  author_forum_level_name?: string
  author_forum_level_color?: string
  author_forum_level_bold?: boolean
  parent_id?: number
  content: string
  like_count: number
  liked?: boolean
  created_at: string
}

export interface RPDBListEntry {
  id: number
  list_id: number
  work_id: number
  status: RPDBListStatus
  priority: number
  quantity: number
  note: string
  character_id?: number
  work: RPDBWork
}

export interface RPDBList {
  id: number
  user_id: number
  name: string
  description: string
  is_default: boolean
  is_public: boolean
  item_count: number
  entries: RPDBListEntry[]
}

export interface ListRPDBWorksParams {
  search?: string
  type?: RPDBWorkType | ''
  availability_status?: string
  verification_status?: RPDBVerificationStatus | ''
  faction?: string
  armor_type?: string
  tag_id?: number
  tag_search?: string
  author_id?: number
  sort?: 'updated_at' | 'created_at' | 'popular' | 'favorite' | 'comments' | 'verified'
  page?: number
  page_size?: number
}

export interface RPDBWorkPayload {
  type: RPDBWorkType
  title: string
  summary?: string
  content?: string
  content_type?: string
  cover_image?: string
  rp_use_cases?: string
  effect_description?: string
  restrictions?: Record<string, unknown>
  extra?: Record<string, unknown>
  availability_status?: string
  bind_type?: string
  faction?: string
  armor_type?: string
  status?: 'draft' | 'published'
  is_public?: boolean
  visibility?: RPDBVisibility
  guild_id?: number
  guild_ids?: number[]
  references?: RPDBReference[]
  media?: RPDBMedia[]
  transmog_slots?: RPDBTransmogSlot[]
  guide_steps?: RPDBGuideStep[]
  tag_ids?: number[]
  tag_names?: string[]
  change_summary?: string
}

export function listRPDBWorks(params: ListRPDBWorksParams = {}) {
  return request.get<{ works: RPDBWork[]; total: number; page: number; page_size: number }>('/rpdb/works', { params })
}

export function listRPDBHotWorks(params: { type?: RPDBWorkType | ''; limit?: number } = {}) {
  return request.get<{ works: RPDBWork[]; window_days: number; limit: number }>('/rpdb/works/hot', { params })
}

export function getRPDBWork(id: number) {
  return request.get<{ work: RPDBWork }>(`/rpdb/works/${id}`)
}

export function getRPDBWorkPreview(id: number) {
  return request.get<{ work: RPDBWork }>(`/rpdb/works/${id}/preview`)
}

export function listMyRPDBWorks() {
  return request.get<{ works: RPDBWork[] }>('/rpdb/my/works')
}

export function listMyRPDBFavorites() {
  return request.get<{ works: RPDBWork[] }>('/rpdb/my/favorites')
}

export function createRPDBWork(payload: RPDBWorkPayload) {
  return request.post<{ work: RPDBWork }>('/rpdb/works', payload)
}

export function updateRPDBWork(id: number, payload: Partial<RPDBWorkPayload>) {
  return request.put<{ work?: RPDBWork; revision?: unknown }>(`/rpdb/works/${id}`, payload)
}

export function deleteRPDBWork(id: number) {
  return request.delete<void>(`/rpdb/works/${id}`)
}

export function likeRPDBWork(id: number) {
  return request.post<{ active: boolean }>(`/rpdb/works/${id}/like`)
}

export function unlikeRPDBWork(id: number) {
  return request.delete<{ active: boolean }>(`/rpdb/works/${id}/like`)
}

export function favoriteRPDBWork(id: number) {
  return request.post<{ active: boolean }>(`/rpdb/works/${id}/favorite`)
}

export function unfavoriteRPDBWork(id: number) {
  return request.delete<{ active: boolean }>(`/rpdb/works/${id}/favorite`)
}

export function listRPDBComments(id: number) {
  return request.get<{ comments: RPDBComment[] }>(`/rpdb/works/${id}/comments`)
}

export function createRPDBComment(id: number, content: string, parentId?: number) {
  return request.post<{ comment: RPDBComment }>(`/rpdb/works/${id}/comments`, { content, parent_id: parentId })
}

export function deleteRPDBComment(commentId: number) {
  return request.delete<void>(`/rpdb/comments/${commentId}`)
}

export function verifyRPDBWork(id: number, result: 'valid' | 'outdated', comment = '') {
  return request.post(`/rpdb/works/${id}/verify`, { result, comment })
}

export function addRPDBWorkToList(id: number, status: RPDBListStatus = 'wanted', listId?: number) {
  return request.post(`/rpdb/works/${id}/list`, { status, list_id: listId })
}

export function updateRPDBWorkVisibility(id: number, visibility: RPDBVisibility, guildIds: number[] = []) {
  return request.put<{ work: RPDBWork }>(`/rpdb/works/${id}/visibility`, {
    visibility,
    guild_ids: visibility === 'guild' ? guildIds : [],
    guild_id: visibility === 'guild' ? guildIds[0] : undefined,
  })
}

export function listRPDBLists() {
  return request.get<{ lists: RPDBList[] }>('/rpdb/lists')
}

export function createRPDBList(name: string, description = '', isPublic = false) {
  return request.post<{ list: RPDBList }>('/rpdb/lists', { name, description, is_public: isPublic })
}

export function updateRPDBListEntry(listId: number, workId: number, payload: Partial<RPDBListEntry>) {
  return request.put<void>(`/rpdb/lists/${listId}/works/${workId}`, payload)
}

export function removeRPDBListEntry(listId: number, workId: number) {
  return request.delete<void>(`/rpdb/lists/${listId}/works/${workId}`)
}

export function exportRPDBList(listId: number, format: 'json' | 'csv' | 'tomtom') {
  return request.get<{ format: string; content?: string; list?: RPDBList; missing_coordinates?: Array<{ work_id: number; title: string }> }>(
    `/rpdb/lists/${listId}/export`,
    { params: { format } },
  )
}

export function resolveRPDBMediaURL(value?: string) {
  if (!value) return ''
  if (/^https?:\/\//.test(value) || value.startsWith('data:')) return value
  if (value.startsWith('/')) return `${API_HOST}${value}`
  return `${API_HOST}/${value}`
}
