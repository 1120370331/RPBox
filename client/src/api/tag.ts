import request from './request'

export interface Tag {
  id: number
  name: string
  color: string
  type: 'preset' | 'custom' | 'guild'
  guild_id?: number
  creator_id: number
  is_public: boolean
  usage_count: number
  created_at: string
  updated_at: string
}

export interface CreateTagRequest {
  name: string
  color?: string
}

export async function listTags(category?: 'story' | 'item' | 'post'): Promise<{ tags: Tag[] }> {
  const params = category ? { category } : {}
  return request.get('/tags', { params })
}

export async function getPresetTags(category?: 'story' | 'item' | 'post'): Promise<{ tags: Tag[] }> {
  const params = category ? { category } : {}
  return request.get('/tags/preset', { params })
}

export async function createTag(data: CreateTagRequest): Promise<Tag> {
  return request.post('/tags', data)
}

export async function updateTag(id: number, data: Partial<CreateTagRequest>): Promise<Tag> {
  return request.put(`/tags/${id}`, data)
}

export async function deleteTag(id: number): Promise<void> {
  return request.delete(`/tags/${id}`)
}

// ========== 剧情标签 ==========

export async function getStoryTags(storyId: number): Promise<{ tags: Tag[] }> {
  return request.get(`/stories/${storyId}/tags`)
}

export async function addStoryTag(storyId: number, tagId: number): Promise<void> {
  return request.post(`/stories/${storyId}/tags`, { tag_id: tagId })
}

export async function removeStoryTag(storyId: number, tagId: number): Promise<void> {
  return request.delete(`/stories/${storyId}/tags/${tagId}`)
}

// ========== 公会标签 ==========

export async function listGuildTags(guildId: number): Promise<{ tags: Tag[] }> {
  return request.get(`/guilds/${guildId}/tags`)
}

export async function createGuildTag(guildId: number, data: CreateTagRequest): Promise<Tag> {
  return request.post(`/guilds/${guildId}/tags`, data)
}

export async function deleteGuildTag(guildId: number, tagId: number): Promise<void> {
  return request.delete(`/guilds/${guildId}/tags/${tagId}`)
}
