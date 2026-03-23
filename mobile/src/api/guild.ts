import { request } from '@shared/api/request'

export interface Guild {
  id: number
  name: string
  description: string
  icon: string
  color: string
  avatar?: string
  avatar_url?: string
  avatar_updated_at?: string
  banner?: string
  banner_url?: string
  banner_updated_at?: string
  slogan: string
  lore: string
  faction: 'alliance' | 'horde' | 'neutral' | ''
  owner_id: number
  member_count: number
  story_count: number
  is_public: boolean
  invite_code: string
  my_role?: 'owner' | 'admin' | 'member'
  auto_approve?: boolean
  server?: string
  created_at: string
  updated_at: string
}

export interface GuildMember {
  id: number
  guild_id: number
  user_id: number
  username: string
  role: 'owner' | 'admin' | 'member'
  joined_at: string
  avatar?: string
  name_color?: string
  name_bold?: boolean
}

export interface GuildApplication {
  id: number
  guild_id: number
  status: 'pending' | 'approved' | 'rejected'
}

export interface GuildStoryWithUploader {
  id: number
  title: string
  description: string
  start_time: string
  end_time: string
  status: string
  tags?: string
  entry_count?: number
  created_at: string
  updated_at: string
  added_by: number
  added_by_username: string
  added_by_avatar?: string
  added_by_name_color?: string
  added_by_name_bold?: boolean
  tag_list?: { name: string; color?: string }[]
}

export function listGuilds() {
  return request.get<{ guilds: Guild[] }>('/guilds')
}

export function listPublicGuilds(query?: { keyword?: string; faction?: string }) {
  const params = new URLSearchParams()
  if (query?.keyword) params.append('keyword', query.keyword)
  if (query?.faction) params.append('faction', query.faction)
  const qs = params.toString()
  return request.get<{ guilds: Guild[] }>(`/public/guilds${qs ? '?' + qs : ''}`)
}

export function getGuild(id: number) {
  return request.get<{ guild: Guild; my_role: '' | 'owner' | 'admin' | 'member' }>(`/guilds/${id}`)
}

export function listGuildMembers(id: number) {
  return request.get<{ members: GuildMember[] }>(`/guilds/${id}/members`)
}

export function joinGuild(inviteCode: string) {
  return request.post<{ guild: Guild }>('/guilds/join', { invite_code: inviteCode })
}

export function applyGuild(guildId: number, message?: string) {
  return request.post<{ application?: GuildApplication; auto_approved?: boolean }>(`/guilds/${guildId}/apply`, { message })
}

export function leaveGuild(guildId: number) {
  return request.post<void>(`/guilds/${guildId}/leave`)
}

export function listMyApplications() {
  return request.get<{ applications: GuildApplication[] }>('/user/guild-applications')
}

export function cancelApplication(guildId: number, appId: number) {
  return request.delete<void>(`/guilds/${guildId}/applications/${appId}`)
}

export function listGuildStories(guildId: number, addedBy?: number) {
  const params = addedBy ? { added_by: addedBy } : {}
  return request.get<{ stories: GuildStoryWithUploader[] }>(`/guilds/${guildId}/stories`, { params })
}

export function removeStoryFromGuild(guildId: number, storyId: number) {
  return request.delete<void>(`/guilds/${guildId}/stories/${storyId}`)
}
