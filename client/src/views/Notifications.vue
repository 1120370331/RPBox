<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '../stores/notification'
import { getNotifications, markNotificationAsRead, markAllNotificationsAsRead, deleteNotification, deleteAllNotifications, type Notification } from '../api/notification'
import { buildNameStyle } from '@/utils/userNameStyle'

const router = useRouter()
const notificationStore = useNotificationStore()
const mounted = ref(false)
const activeTab = ref('all')
const notifications = ref<Notification[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = 20

const tabs = [
  { id: 'all', label: '全部', icon: 'ri-notification-3-line' },
  { id: 'like', label: '点赞', icon: 'ri-heart-line' },
  { id: 'comment', label: '评论', icon: 'ri-chat-3-line' },
  { id: 'guild', label: '公会', icon: 'ri-shield-line' },
  { id: 'system', label: '系统', icon: 'ri-information-line' },
]

const hasMore = computed(() => notifications.value.length < total.value)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  loadNotifications()
  notificationStore.loadUnreadCount()
})

async function loadNotifications(append = false) {
  loading.value = true
  try {
    const res = await getNotifications(activeTab.value, page.value, pageSize)
    if (append) {
      notifications.value = [...notifications.value, ...res.notifications]
    } else {
      notifications.value = res.notifications
    }
    total.value = res.total
  } catch (error) {
    console.error('加载消息失败:', error)
  } finally {
    loading.value = false
  }
}

function handleTabChange(tabId: string) {
  activeTab.value = tabId
  page.value = 1
  loadNotifications()
}

async function handleMarkAsRead(notification: Notification) {
  if (notification.is_read) return

  try {
    await markNotificationAsRead(notification.id)
    notification.is_read = true
    notificationStore.decrementUnreadCount()
  } catch (error) {
    console.error('标记已读失败:', error)
  }
}

async function handleMarkAllAsRead() {
  try {
    await markAllNotificationsAsRead()
    notifications.value.forEach(n => n.is_read = true)
    notificationStore.resetUnreadCount()
  } catch (error) {
    console.error('标记全部已读失败:', error)
  }
}

async function handleDeleteNotification(notification: Notification) {
  try {
    await deleteNotification(notification.id)
    notifications.value = notifications.value.filter(n => n.id !== notification.id)
    total.value--
    if (!notification.is_read) {
      notificationStore.decrementUnreadCount()
    }
  } catch (error) {
    console.error('删除通知失败:', error)
  }
}

async function handleDeleteAll() {
  try {
    await deleteAllNotifications()
    notifications.value = []
    total.value = 0
    notificationStore.resetUnreadCount()
  } catch (error) {
    console.error('清空通知失败:', error)
  }
}

function handleLoadMore() {
  page.value++
  loadNotifications(true)
}

function formatTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  if (hours < 1) return '刚刚'
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

function handleNotificationClick(notification: Notification) {
  handleMarkAsRead(notification)

  // 根据通知类型跳转到对应页面
  if (notification.target_type === 'comment') {
    if (notification.target_post_id) {
      router.push({
        path: `/community/post/${notification.target_post_id}`,
        query: { comment: String(notification.target_id) }
      })
    }
  } else if (notification.target_type === 'post') {
    router.push(`/community/post/${notification.target_id}`)
  } else if (notification.target_type === 'item') {
    router.push(`/market/${notification.target_id}`)
  } else if (notification.target_type === 'guild') {
    router.push(`/guild/${notification.target_id}`)
  }
}

function getTypeBadge(type: string): string {
  const badges: Record<string, string> = {
    'post_like': 'LIKE',
    'item_like': 'LIKE',
    'post_comment': 'REPLY',
    'item_comment': 'REPLY',
    'mention': 'AT',
    'guild_application': 'GUILD',
    'system': 'SYS'
  }
  return badges[type] || 'INFO'
}
</script>

<template>
  <div class="notifications-page" :class="{ 'animate-in': mounted }">
    <div class="page-header">
      <div class="header-left">
        <h1>消息中心</h1>
        <p class="subtitle">查看您的点赞、评论、公会通知等消息</p>
      </div>
      <div class="header-actions">
        <button v-if="notificationStore.unreadCount > 0" class="mark-all-btn" @click="handleMarkAllAsRead">
          <i class="ri-check-double-line"></i>
          <span>全部标记已读</span>
        </button>
        <button v-if="total > 0" class="clear-all-btn" @click="handleDeleteAll">
          <i class="ri-delete-bin-line"></i>
          <span>清空所有</span>
        </button>
      </div>
    </div>

    <!-- 标签页 -->
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-btn"
        :class="{ active: activeTab === tab.id }"
        @click="handleTabChange(tab.id)"
      >
        <i :class="tab.icon"></i>
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <!-- 消息列表 -->
    <div class="notifications-list">
      <div v-if="loading" class="loading">
        <i class="ri-loader-4-line spin"></i>
        <span>加载中...</span>
      </div>

      <div v-else-if="notifications.length === 0" class="empty-state">
        <i class="ri-notification-off-line"></i>
        <p>暂无消息</p>
      </div>

      <div v-else class="notification-items">
        <div
          v-for="notification in notifications"
          :key="notification.id"
          class="notification-item"
          :class="{ unread: !notification.is_read }"
        >
          <!-- 未读标记三角形 -->
          <div v-if="!notification.is_read" class="unread-corner"></div>

          <div class="notification-inner" @click="handleNotificationClick(notification)">
            <!-- 头像区域 -->
            <div class="notification-avatar-wrapper">
              <div class="notification-avatar">
                <img v-if="notification.actor_avatar" :src="notification.actor_avatar" alt="" />
                <span v-else>{{ notification.actor_name?.charAt(0)?.toUpperCase() || '?' }}</span>
              </div>
              <div class="notification-type-badge">{{ getTypeBadge(notification.type) }}</div>
            </div>

            <!-- 内容区域 -->
            <div class="notification-content">
              <div class="notification-header">
                <h3 class="notification-title">
                  <span class="username" :style="buildNameStyle(notification.actor_name_color, notification.actor_name_bold)">{{ notification.actor_name || '系统' }}</span>
                  {{ notification.content }}
                </h3>
                <time class="notification-time">{{ formatTime(notification.created_at) }}</time>
              </div>

              <!-- 悬停操作按钮 -->
              <div class="notification-actions">
                <button class="action-btn primary" @click.stop="handleNotificationClick(notification)">
                  查看详情
                </button>
                <button v-if="!notification.is_read" class="action-btn secondary" @click.stop="handleMarkAsRead(notification)">
                  标为已读
                </button>
                <button class="action-btn secondary" @click.stop="handleDeleteNotification(notification)">
                  删除
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 加载更多 -->
        <div v-if="hasMore && !loading" class="load-more">
          <button @click="handleLoadMore" class="load-more-btn">
            <span class="line"></span>
            <span class="text">加载更多</span>
            <span class="line"></span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notifications-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 24px;
}

.page-header {
  margin-bottom: 32px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.header-left {
  flex: 1;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #2C1810;
  margin: 0 0 8px 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.subtitle {
  font-size: 12px;
  color: #8D7B68;
  margin: 0;
  font-family: 'Courier New', monospace;
  opacity: 0.8;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.mark-all-btn,
.clear-all-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fff;
  color: #2C1810;
  border: 2px solid #2C1810;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  box-shadow: 4px 4px 0px 0px rgba(44, 24, 16, 0.1);
}

.mark-all-btn:hover,
.clear-all-btn:hover {
  box-shadow: 2px 2px 0px 0px rgba(44, 24, 16, 0.2);
}

.mark-all-btn:active,
.clear-all-btn:active {
  box-shadow: 0px 0px 0px 0px rgba(44, 24, 16, 0.2);
  transform: translate(2px, 2px);
}

.mark-all-btn i,
.clear-all-btn i {
  font-size: 16px;
}

/* 标签页 */
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  overflow-x: auto;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: transparent;
  border: none;
  border-left: 4px solid transparent;
  color: #2C1810;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.tab-btn:hover {
  background: #fff;
  border-left-color: #D4A373;
  transform: translateX(4px);
}

.tab-btn.active {
  background: #804030;
  color: #fff;
  border-left-color: #2C1810;
  box-shadow: 4px 4px 0px 0px rgba(44, 24, 16, 0.1);
}

.tab-btn i {
  font-size: 18px;
}

/* 消息列表 */
.notifications-list {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  min-height: 400px;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  color: #8D7B68;
  gap: 12px;
}

.loading i {
  font-size: 32px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  color: #8D7B68;
}

.empty-state i {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state p {
  font-size: 16px;
  margin: 0;
}

/* 消息项 */
.notification-items {
  padding: 12px;
}

.notification-item {
  position: relative;
  background: #fff;
  border: 1px solid rgba(44, 24, 16, 0.2);
  padding: 20px;
  margin-bottom: 12px;
  transition: all 0.2s;
}

.notification-item:hover {
  border-color: #804030;
}

.notification-item.unread {
  background: #FFF9F0;
  border-color: rgba(44, 24, 16, 0.2);
}

.notification-item.unread:hover {
  border-color: #804030;
}

/* 未读标记三角形 */
.unread-corner {
  position: absolute;
  top: 0;
  right: 0;
  width: 0;
  height: 0;
  border-top: 20px solid #B87333;
  border-left: 20px solid transparent;
}

.notification-inner {
  display: flex;
  gap: 20px;
  cursor: pointer;
}

.notification-avatar-wrapper {
  position: relative;
  flex-shrink: 0;
}

.notification-avatar {
  width: 48px;
  height: 48px;
  min-width: 48px;
  border: 1px solid #2C1810;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #804030);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  overflow: hidden;
}

.notification-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.notification-type-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  background: #2C1810;
  color: #F5E6D3;
  font-size: 8px;
  font-weight: 700;
  padding: 2px 4px;
  border-radius: 3px;
  border: 1px solid #F5E6D3;
  letter-spacing: 0.5px;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 8px;
}

.notification-title {
  font-size: 14px;
  color: #2C1810;
  line-height: 1.5;
  margin: 0;
}

.username {
  font-weight: 600;
  color: #804030;
}

.notification-time {
  font-size: 12px;
  color: #8D7B68;
  white-space: nowrap;
}

/* 悬停操作按钮 */
.notification-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.notification-item:hover .notification-actions {
  opacity: 1;
}

.action-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn.primary {
  background: #804030;
  color: #fff;
}

.action-btn.primary:hover {
  background: #6B3426;
}

.action-btn.secondary {
  background: transparent;
  color: #804030;
  border: 1px solid #E5D4C1;
}

.action-btn.secondary:hover {
  background: rgba(128, 64, 48, 0.05);
  border-color: #804030;
}

/* 加载更多 */
.load-more {
  padding: 20px;
  text-align: center;
}

.load-more-btn {
  display: inline-flex;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  background: transparent;
  border: none;
  color: #804030;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
}

.load-more-btn:hover {
  color: #6B3426;
}

.load-more-btn .line {
  width: 60px;
  height: 1px;
  background: #E5D4C1;
  transition: all 0.3s;
}

.load-more-btn:hover .line {
  background: #804030;
  width: 80px;
}

.load-more-btn .text {
  white-space: nowrap;
}

/* 动画 */
.animate-in {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
