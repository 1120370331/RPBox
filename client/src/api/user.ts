import request from './request'
import type { UserActivityInfo, UserData } from '@/types/user'

export interface UserInfo extends UserData {
  email: string
  avatar: string
  bio?: string
  location?: string
  website?: string
  post_count?: number
  story_count?: number
  profile_count?: number
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

// 获取当前用户信息
export async function getUserInfo(): Promise<UserInfo> {
  return request.get('/user/info')
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
  name_style_preference?: 'default' | 'level' | 'sponsor'
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
