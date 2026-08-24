import { request } from '@shared/api/request'

export interface Story {
  id: number
  user_id: number
  title: string
  description: string
  region?: string
  address?: string
  participants: string
  start_time: string
  end_time: string
  status: string
  is_public: boolean
  view_count: number
  entry_count?: number
  tag_list?: { name: string; color?: string }[]
  created_at: string
  updated_at: string
}

export interface StoryEntry {
  id: number
  story_id: number
  source_id?: string
  type: 'dialogue' | 'narration' | 'image'
  character_id?: number | null
  character_card_id?: number | null
  speaker: string
  content: string
  channel: string
  timestamp: string
  sort_order: number
  background_color?: string
  group_name?: string
  created_at?: string
}

export interface CharacterCardSummary {
  id: number
  first_name: string
  last_name: string
  display_name: string
  icon?: string
  class_color?: string
  name_color?: string
  portrait_image_url?: string | null
  portrait_image_updated_at?: string | null
  updated_at?: string
}

export interface UpdateStoryEntryRequest {
  content?: string
  speaker?: string
  channel?: string
  type?: StoryEntry['type']
  character_id?: number | null
  character_card_id?: number | null
  timestamp?: string
}

export interface StoryBookmark {
  id: number
  story_id: number
  user_id: number
  entry_id: number
  name: string
  color?: string
  is_favorite: boolean
  is_auto: boolean
  is_public: boolean
  created_at: string
  updated_at: string
}

export function listStories(params?: Record<string, string>) {
  const query = params ? '?' + new URLSearchParams(params).toString() : ''
  return request.get<{ stories: Story[] }>(`/stories${query}`)
}

export function getStory(id: number) {
  return request.get<{
    story: Story
    entries: StoryEntry[]
    character_cards?: Record<number, CharacterCardSummary>
  }>(`/stories/${id}`)
}

export function updateStoryEntry(storyId: number, entryId: number, data: UpdateStoryEntryRequest) {
  return request.put<StoryEntry>(`/stories/${storyId}/entries/${entryId}`, data)
}

export function deleteStoryEntry(storyId: number, entryId: number) {
  return request.delete<void>(`/stories/${storyId}/entries/${entryId}`)
}

export function updateEntriesBackgroundColor(
  storyId: number,
  entryIds: number[],
  backgroundColor: string,
  groupName?: string,
) {
  return request.post<void>(`/stories/${storyId}/entries/batch-background`, {
    entry_ids: entryIds,
    background_color: backgroundColor,
    group_name: groupName,
  })
}

export function listBookmarks(storyId: number) {
  return request.get<{ bookmarks: StoryBookmark[] }>(`/stories/${storyId}/bookmarks`)
}

export function createBookmark(storyId: number, entryId: number, name: string, color?: string, isPublic?: boolean) {
  return request.post<StoryBookmark>(`/stories/${storyId}/bookmarks`, {
    entry_id: entryId,
    name,
    color,
    is_public: isPublic,
  })
}

export function updateBookmark(storyId: number, bookmarkId: number, data: { name?: string; color?: string; is_favorite?: boolean }) {
  return request.put<StoryBookmark>(`/stories/${storyId}/bookmarks/${bookmarkId}`, data)
}

export function deleteBookmark(storyId: number, bookmarkId: number) {
  return request.delete<void>(`/stories/${storyId}/bookmarks/${bookmarkId}`)
}

export function updateLastViewBookmark(storyId: number, entryId: number) {
  return request.put<StoryBookmark>(`/stories/${storyId}/bookmarks/last-view`, { entry_id: entryId })
}
