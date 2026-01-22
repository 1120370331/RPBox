import request from './request'

export interface UserInfo {
  id: number
  username: string
  email: string
  avatar: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  sponsor_color?: string
  sponsor_bold?: boolean
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
}): Promise<void> {
  return request.put('/user/info', data)
}

// 上传头像
export async function uploadAvatar(file: File): Promise<{ avatar: string }> {
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

// 绑定邮箱
export async function bindEmail(email: string, verificationCode: string): Promise<{ message: string }> {
  return request.post('/user/bind-email', { email, verification_code: verificationCode })
}
