import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getUnreadCount } from '../api/notification'
import { WebSocketService, MessageType, type WebSocketMessage } from '../services/websocket'
import { useUserStore } from './user'

export const useNotificationStore = defineStore('notification', () => {
  const unreadCount = ref(0)
  const loading = ref(false)
  let wsService: WebSocketService | null = null

  // 连接 WebSocket
  function connectWebSocket() {
    try {
      const userStore = useUserStore()
      if (!userStore.token) {
        console.log('[Notification] 未登录，跳过 WebSocket 连接')
        return
      }

      // 如果已连接，先断开
      if (wsService) {
        wsService.disconnect()
      }

      // 构建 WebSocket URL
      const apiBase = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'
      const wsUrl = apiBase.replace(/^https?/, (match) => match === 'https' ? 'wss' : 'ws') + '/ws/notifications'

      // 创建 WebSocket 服务
      wsService = new WebSocketService(wsUrl, userStore.token)

      // 监听未读数量更新
      wsService.on(MessageType.UnreadCountUpdate, (message: WebSocketMessage) => {
        console.log('[Notification] 收到未读数量更新:', message.data)
        unreadCount.value = message.data.count
      })

      // 监听新通知
      wsService.on(MessageType.NewNotification, (message: WebSocketMessage) => {
        console.log('[Notification] 收到新通知:', message.data)
        // 新通知到达时，未读数量会通过 UnreadCountUpdate 消息更新
      })

      // 连接
      wsService.connect()
      console.log('[Notification] WebSocket 连接已启动')
    } catch (error) {
      console.error('[Notification] WebSocket 连接失败:', error)
      // 连接失败不影响应用正常使用
    }
  }

  // 断开 WebSocket
  function disconnectWebSocket() {
    if (wsService) {
      wsService.disconnect()
      wsService = null
      console.log('[Notification] WebSocket 已断开')
    }
  }

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
    connectWebSocket,
    disconnectWebSocket,
  }
})
