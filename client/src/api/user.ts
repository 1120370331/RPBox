import request from './request'
import type { UserActivityInfo, UserData } from '@/types/user'

export interface UserInfo extends UserData {
  email: string
  avatar: string
  bio?: string
  location?: string
  website?: string
  post_count?: number
  guild_count?: number
  item_count?: number
  story_count?: number
  story_entry_count?: number
  profile_count?: number
  max_post_views?: number
  max_item_downloads?: number
  total_likes?: number
  total_item_downloads?: number
  total_sign_in_days?: number
  consecutive_sign_in_days?: number
  created_at?: string
}

export interface UserMentionItem {
  id: number
  username: string
  avatar?: string
  name_color?: string
  name_bold?: boolean
}

export interface SponsorUser {
  id: number
  username: string
  avatar?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  name_color?: string
  name_bold?: boolean
}

export interface PublicUserProfile {
  id: number
  username: string
  avatar?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  name_color?: string
  name_bold?: boolean
  bio?: string
  location?: string
  website?: string
  post_count?: number
  guild_count?: number
  item_count?: number
  story_count?: number
  created_at?: string
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
}

// 获取当前用户信息
export async function getUserInfo(): Promise<UserInfo> {
  return request.get('/user/info')
}

export async function getUserProfile(id: number): Promise<PublicUserProfile> {
  return request.get(`/users/${id}`)
}

// 更新用户信息
export async function updateUserInfo(data: {
  username?: string
  email?: string
  bio?: string
  location?: string
  website?: string
  sponsor_color?: string
  sponsor_bold?: boolean
  name_style_preference?: 'default' | 'sponsor'
}): Promise<void> {
  return request.put('/user/info', data)
}

// 上传头像
export async function uploadAvatar(file: File): Promise<UserActivityInfo & {
  avatar: string
  avatar_review_status: string
  message: string
}> {
  const formData = new FormData()
  formData.append('avatar', file)

  const token = localStorage.getItem('token')
  const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

  const res = await fetch(`${API_BASE}/user/avatar`, {
    method: 'POST',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: formData,
  })

  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error || '上传失败')
  }

  return res.json()
}

export async function signInDaily(): Promise<UserActivityInfo & {
  message: string
  granted: boolean
  points_delta: number
  experience_delta: number
}> {
  return request.post('/user/sign-in')
}

export async function redeemSponsorCode(code: string): Promise<{
  message: string
  user: Partial<UserData>
}> {
  return request.post('/sponsor-codes/redeem', { code })
}

// 绑定邮箱
export async function bindEmail(email: string, verificationCode: string): Promise<{ message: string }> {
  return request.post('/user/bind-email', { email, verification_code: verificationCode })
}

// 搜索用户（用于@提及）
export async function searchUsers(keyword: string, limit: number = 10): Promise<{ users: UserMentionItem[] }> {
  return request.get('/users/search', {
    params: { q: keyword, limit },
  })
}

export async function listSponsors(): Promise<{ users: SponsorUser[] }> {
  return request.get('/sponsors')
}
