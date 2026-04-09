import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UserData {
  id: number
  username: string
  avatar?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  sponsor_color?: string
  sponsor_bold?: boolean
  name_color?: string
  name_bold?: boolean
  activity_points?: number
  activity_experience?: number
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
  current_level_exp?: number
  next_level_exp?: number
  signed_in_today?: boolean
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<UserData | null>(null)

  const savedUser = localStorage.getItem('user')
  if (savedUser) {
    try {
      user.value = JSON.parse(savedUser)
    } catch (e) {
      console.error('解析用户信息失败:', e)
    }
  }

  const isModerator = computed(() => {
    return user.value?.role === 'moderator' || user.value?.role === 'admin'
  })

  const isAdmin = computed(() => {
    return user.value?.role === 'admin'
  })

  function setAuth(t: string, u: UserData) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
  }

  function updateAvatar(avatar: string) {
    if (user.value) {
      user.value.avatar = avatar
      localStorage.setItem('user', JSON.stringify(user.value))
    }
  }

  function updateRole(role: string) {
    if (user.value) {
      user.value.role = role
      localStorage.setItem('user', JSON.stringify(user.value))
    }
  }

  function mergeUser(patch: Partial<UserData>) {
    if (user.value) {
      user.value = { ...user.value, ...patch }
      localStorage.setItem('user', JSON.stringify(user.value))
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return { token, user, isModerator, isAdmin, setAuth, updateAvatar, updateRole, mergeUser, logout }
})
