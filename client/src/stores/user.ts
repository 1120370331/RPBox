import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserData } from '@/types/user'

export type { UserData } from '@/types/user'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<UserData | null>(null)

  function persistUserState() {
    if (user.value) {
      localStorage.setItem('user', JSON.stringify(user.value))
    } else {
      localStorage.removeItem('user')
    }
  }

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
    persistUserState()
  }

  function updateAvatar(avatar: string) {
    if (user.value) {
      user.value.avatar = avatar
      persistUserState()
    }
  }

  function updateRole(role: string) {
    if (user.value) {
      user.value.role = role
      persistUserState()
    }
  }

  function mergeUser(patch: Partial<UserData>) {
    if (!user.value) {
      if (typeof patch.id !== 'number' || !patch.username) return
      user.value = patch as UserData
    } else {
      user.value = {
        ...user.value,
        ...patch,
      }
    }
    persistUserState()
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return { token, user, isModerator, isAdmin, setAuth, updateAvatar, updateRole, mergeUser, logout }
})
