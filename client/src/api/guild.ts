import request from './request'

export interface Guild {
  id: number
  name: string
  description: string
  icon: string
  color: string
  banner: string
  slogan: string
  lore: string
  faction: 'alliance' | 'horde' | 'neutral' | ''
  layout: 1 | 2 | 3 | 4
  owner_id: number
  member_count: number
  story_count: number
  is_public: boolean
  invite_code: string
  status?: 'pending' | 'approved' | 'rejected'
  my_role?: 'owner' | 'admin' | 'member'
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
}

export interface CreateGuildRequest {
  name: string
  description?: string
  icon?: string
  color?: string
  banner?: string
  slogan?: string
  lore?: string
  faction?: string
  layout?: 1 | 2 | 3 | 4
}

export async function listGuilds(): Promise<{ guilds: Guild[] }> {
  return request.get('/guilds')
}

export async function createGuild(data: CreateGuildRequest): Promise<Guild> {
  return request.post('/guilds', data)
}

export async function getGuild(id: number): Promise<{ guild: Guild; my_role: string }> {
  return request.get(`/guilds/${id}`)
}

export async function updateGuild(id: number, data: Partial<CreateGuildRequest>): Promise<Guild> {
  return request.put(`/guilds/${id}`, data)
}

export async function deleteGuild(id: number): Promise<void> {
  return request.delete(`/guilds/${id}`)
}

export async function joinGuild(inviteCode: string): Promise<{ guild: Guild }> {
  return request.post('/guilds/join', { invite_code: inviteCode })
}

export async function leaveGuild(id: number): Promise<void> {
  return request.post(`/guilds/${id}/leave`)
}

export async function listGuildMembers(id: number): Promise<{ members: GuildMember[] }> {
  return request.get(`/guilds/${id}/members`)
}

export async function updateMemberRole(guildId: number, userId: number, role: string): Promise<void> {
  return request.put(`/guilds/${guildId}/members/${userId}`, { role })
}

export async function removeMember(guildId: number, userId: number): Promise<void> {
  return request.delete(`/guilds/${guildId}/members/${userId}`)
}

// ========== 剧情归档 ==========

export async function listGuildStories(guildId: number): Promise<{ stories: any[] }> {
  return request.get(`/guilds/${guildId}/stories`)
}

export async function archiveStoryToGuild(guildId: number, storyId: number): Promise<void> {
  return request.post(`/guilds/${guildId}/stories/${storyId}`)
}

export async function removeStoryFromGuild(guildId: number, storyId: number): Promise<void> {
  return request.delete(`/guilds/${guildId}/stories/${storyId}`)
}

export async function getStoryGuilds(storyId: number): Promise<{ guilds: Guild[] }> {
  return request.get(`/stories/${storyId}/guilds`)
}

// ========== 公开公会 ==========

export interface PublicGuildsQuery {
  keyword?: string
  faction?: string
  server?: string
}

export async function listPublicGuilds(query?: PublicGuildsQuery): Promise<{ guilds: Guild[] }> {
  const params = new URLSearchParams()
  if (query?.keyword) params.append('keyword', query.keyword)
  if (query?.faction) params.append('faction', query.faction)
  if (query?.server) params.append('server', query.server)
  const queryStr = params.toString()
  return request.get(`/public/guilds${queryStr ? '?' + queryStr : ''}`)
}

export async function uploadGuildBanner(guildId: number, file: File): Promise<{ banner: string }> {
  const formData = new FormData()
  formData.append('banner', file)
  return request.post(`/guilds/${guildId}/banner`, formData)
}
