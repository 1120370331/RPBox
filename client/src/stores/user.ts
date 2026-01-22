import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface UserData {
  id: number
  username: string
  avatar?: string
  role?: string // user|moderator|admin
  is_sponsor?: boolean
  sponsor_level?: number
  sponsor_color?: string
  sponsor_bold?: boolean
  name_color?: string
  name_bold?: boolean
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<UserData | null>(null)

  // 初始化时从 localStorage 恢复用户信息
  const savedUser = localStorage.getItem('user')
  if (savedUser) {
    try {
      user.value = JSON.parse(savedUser)
    } catch (e) {
      console.error('解析用户信息失败:', e)
    }
  }

  // 计算属性：是否为版主或管理员
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

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return { token, user, isModerator, isAdmin, setAuth, updateAvatar, updateRole, logout }
})
