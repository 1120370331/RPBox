import { defineStore } from 'pinia'
import { ref } from 'vue'

interface UserData {
  id: number
  username: string
  avatar?: string
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

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return { token, user, setAuth, updateAvatar, logout }
})
