import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getUnreadCount } from '../api/notification'

export const useNotificationStore = defineStore('notification', () => {
  const unreadCount = ref(0)
  const loading = ref(false)

  // 加载未读数量
  async function loadUnreadCount() {
    loading.value = true
    try {
      const res = await getUnreadCount()
      unreadCount.value = res.count
    } catch (error) {
      console.error('获取未读数量失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 减少未读数量
  function decrementUnreadCount(amount: number = 1) {
    unreadCount.value = Math.max(0, unreadCount.value - amount)
  }

  // 重置未读数量
  function resetUnreadCount() {
    unreadCount.value = 0
  }

  return {
    unreadCount,
    loading,
    loadUnreadCount,
    decrementUnreadCount,
    resetUnreadCount,
  }
})
