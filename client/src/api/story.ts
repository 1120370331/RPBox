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
  speaker: string
  speaker_ic: string
  speaker_color: string
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
  speaker_ic?: string
  speaker_color?: string
  content: string
  channel?: string
  timestamp?: string
}

export async function listStories(): Promise<{ stories: Story[] }> {
  return request.get('/stories')
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

export async function publishStory(id: number, isPublic: boolean): Promise<Story> {
  return request.post(`/stories/${id}/publish`, { is_public: isPublic })
}

export interface PublicStoryResponse {
  story: Story
  entries: StoryEntry[]
  author: string
}

export async function getPublicStory(code: string): Promise<PublicStoryResponse> {
  return request.get(`/public/stories/${code}`)
}
