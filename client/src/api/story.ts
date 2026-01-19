import request from './request'

export interface Story {
  id: number
  user_id: number
  title: string
  description: string
  participants: string
  tags: string
  start_time: string
  end_time: string
  status: 'draft' | 'published'
  is_public: boolean
  share_code: string
  view_count: number
  created_at: string
  updated_at: string
}

export interface StoryEntry {
  id: number
  story_id: number
  source_id: string
  type: 'dialogue' | 'narration' | 'image'
  character_id?: number  // 关联的角色ID
  speaker: string
  content: string
  channel: string
  timestamp: string
  sort_order: number
}

export interface CreateStoryRequest {
  title: string
  description?: string
  participants?: string[]
  tags?: string[]
  start_time?: string
  end_time?: string
}

export interface CreateStoryEntryRequest {
  source_id?: string
  type?: string
  speaker?: string
  content: string
  channel?: string
  timestamp?: string
  // 角色信息（用于创建/关联Character）
  ref_id?: string      // TRP3 ref ID
  game_id?: string     // 游戏内ID
  trp3_data?: string   // 完整TRP3 profile JSON
  is_npc?: boolean     // 是否NPC
}

export interface StoryFilterParams {
  tag_ids?: string      // 标签ID列表，逗号分隔
  guild_id?: string     // 公会ID
  search?: string       // 搜索关键词
  start_date?: string   // 开始日期 YYYY-MM-DD
  end_date?: string     // 结束日期 YYYY-MM-DD
  sort?: string         // 排序字段 created_at|updated_at|start_time
  order?: 'asc' | 'desc' // 排序方向
}

export async function listStories(params?: StoryFilterParams): Promise<{ stories: Story[] }> {
  // 构建查询字符串（GET请求不能有body）
  const searchParams = new URLSearchParams()
  if (params) {
    if (params.tag_ids) searchParams.set('tag_ids', params.tag_ids)
    if (params.guild_id) searchParams.set('guild_id', params.guild_id)
    if (params.search) searchParams.set('search', params.search)
    if (params.start_date) searchParams.set('start_date', params.start_date)
    if (params.end_date) searchParams.set('end_date', params.end_date)
    if (params.sort) searchParams.set('sort', params.sort)
    if (params.order) searchParams.set('order', params.order)
  }
  const query = searchParams.toString()
  return request.get(`/stories${query ? '?' + query : ''}`)
}

export async function getStory(id: number): Promise<{ story: Story; entries: StoryEntry[] }> {
  return request.get(`/stories/${id}`)
}

export async function createStory(data: CreateStoryRequest): Promise<Story> {
  return request.post('/stories', data)
}

export async function updateStory(id: number, data: CreateStoryRequest): Promise<Story> {
  return request.put(`/stories/${id}`, data)
}

export async function deleteStory(id: number): Promise<void> {
  return request.delete(`/stories/${id}`)
}

export async function addStoryEntries(id: number, entries: CreateStoryEntryRequest[]): Promise<void> {
  return request.post(`/stories/${id}/entries`, entries)
}

export interface UpdateStoryEntryRequest {
  content?: string
  speaker?: string
  channel?: string
  type?: string
  character_id?: number | null
  timestamp?: string
}

export async function updateStoryEntry(storyId: number, entryId: number, data: UpdateStoryEntryRequest): Promise<StoryEntry> {
  return request.put(`/stories/${storyId}/entries/${entryId}`, data)
}

export async function deleteStoryEntry(storyId: number, entryId: number): Promise<void> {
  return request.delete(`/stories/${storyId}/entries/${entryId}`)
}

export async function publishStory(id: number, isPublic: boolean): Promise<Story> {
  return request.post(`/stories/${id}/publish`, { is_public: isPublic })
}

export interface PublicStoryResponse {
  story: Story
  entries: StoryEntry[]
  characters: Record<number, import('./character').Character>
  author: string
}

export async function getPublicStory(code: string): Promise<PublicStoryResponse> {
  return request.get(`/public/stories/${code}`)
}
