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
  visitor_can_view_stories: boolean  // 访客可查看剧情
  visitor_can_view_posts: boolean    // 访客可查看帖子
  member_can_view_stories: boolean   // 成员可查看剧情
  member_can_view_posts: boolean     // 成员可查看帖子
  auto_approve: boolean              // 自动审核（无需审核直接加入）
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
  visitor_can_view_stories?: boolean
  visitor_can_view_posts?: boolean
  member_can_view_stories?: boolean
  member_can_view_posts?: boolean
  auto_approve?: boolean
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

export interface GuildStoryWithUploader {
  id: number
  title: string
  description: string
  start_time: string
  end_time: string
  status: string
  created_at: string
  updated_at: string
  added_by: number
  added_by_username: string
  added_by_avatar: string
}

export async function listGuildStories(guildId: number, addedBy?: number): Promise<{ stories: GuildStoryWithUploader[] }> {
  const params = addedBy ? { added_by: addedBy } : {}
  return request.get(`/guilds/${guildId}/stories`, { params })
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

// ========== 公会申请系统 ==========

export interface GuildApplication {
  id: number
  guild_id: number
  user_id: number
  message: string
  status: 'pending' | 'approved' | 'rejected'
  reviewer_id?: number
  review_comment?: string
  reviewed_at?: string
  created_at: string
  updated_at: string
  // 扩展字段
  username?: string
  avatar?: string
  guild_name?: string
  guild_icon?: string
}

export async function applyGuild(guildId: number, message?: string): Promise<{ application: GuildApplication }> {
  return request.post(`/guilds/${guildId}/apply`, { message })
}

export async function listGuildApplications(guildId: number, status?: string): Promise<{ applications: GuildApplication[] }> {
  const params = status ? `?status=${status}` : ''
  return request.get(`/guilds/${guildId}/applications${params}`)
}

export async function reviewGuildApplication(guildId: number, appId: number, action: 'approve' | 'reject', comment?: string): Promise<{ application: GuildApplication }> {
  return request.post(`/guilds/${guildId}/applications/${appId}/review`, { action, comment })
}

export async function listMyApplications(): Promise<{ applications: GuildApplication[] }> {
  return request.get('/user/guild-applications')
}

export async function cancelApplication(guildId: number, appId: number): Promise<void> {
  return request.delete(`/guilds/${guildId}/applications/${appId}`)
}
