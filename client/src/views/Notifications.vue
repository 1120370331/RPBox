<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getNotifications, markNotificationAsRead, markAllNotificationsAsRead, getUnreadCount, type Notification } from '../api/notification'

const router = useRouter()
const mounted = ref(false)
const activeTab = ref('all')
const notifications = ref<Notification[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = 20
const unreadCount = ref(0)

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
  loadUnreadCount()
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

async function loadUnreadCount() {
  try {
    const res = await getUnreadCount()
    unreadCount.value = res.count
  } catch (error) {
    console.error('获取未读数量失败:', error)
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
    loadUnreadCount()
  } catch (error) {
    console.error('标记已读失败:', error)
  }
}

async function handleMarkAllAsRead() {
  try {
    await markAllNotificationsAsRead()
    notifications.value.forEach(n => n.is_read = true)
    loadUnreadCount()
  } catch (error) {
    console.error('标记全部已读失败:', error)
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
  if (notification.target_type === 'post') {
    router.push(`/community/post/${notification.target_id}`)
  } else if (notification.target_type === 'item') {
    router.push(`/market/${notification.target_id}`)
  } else if (notification.target_type === 'guild') {
    router.push(`/guild/${notification.target_id}`)
  }
}
</script>

<template>
  <div class="notifications-page" :class="{ 'animate-in': mounted }">
    <div class="page-header">
      <div class="header-left">
        <h1>消息中心</h1>
        <p class="subtitle">查看您的点赞、评论、公会通知等消息</p>
      </div>
      <button v-if="unreadCount > 0" class="mark-all-btn" @click="handleMarkAllAsRead">
        <i class="ri-check-double-line"></i>
        <span>全部标记已读</span>
      </button>
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
          @click="handleNotificationClick(notification)"
        >
          <div class="notification-avatar">
            <img v-if="notification.actor_avatar" :src="notification.actor_avatar" alt="" />
            <span v-else>{{ notification.actor_name?.charAt(0)?.toUpperCase() || '?' }}</span>
          </div>
          <div class="notification-content">
            <p class="notification-text">
              <span class="username">{{ notification.actor_name || '系统' }}</span>
              {{ notification.content }}
            </p>
            <span class="notification-time">{{ formatTime(notification.created_at) }}</span>
          </div>
          <div v-if="!notification.is_read" class="unread-badge"></div>
        </div>

        <!-- 加载更多 -->
        <div v-if="hasMore && !loading" class="load-more">
          <button @click="handleLoadMore" class="load-more-btn">
            <span>加载更多</span>
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
  font-size: 32px;
  font-weight: 700;
  color: #2C1810;
  margin: 0 0 8px 0;
}

.subtitle {
  font-size: 14px;
  color: #8D7B68;
  margin: 0;
}

.mark-all-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
  white-space: nowrap;
}

.mark-all-btn:hover {
  background: #6B3426;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(128, 64, 48, 0.3);
}

.mark-all-btn i {
  font-size: 18px;
}

/* 标签页 */
.tabs {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #E5D4C1;
  overflow-x: auto;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
  white-space: nowrap;
}

.tab-btn:hover {
  background: rgba(128, 64, 48, 0.05);
  color: #804030;
}

.tab-btn.active {
  background: #804030;
  color: #fff;
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
  padding: 16px;
}

.notification-item {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-radius: 8px;
  transition: background 0.2s;
  cursor: pointer;
  position: relative;
}

.notification-item:hover {
  background: #F5EFE7;
}

.notification-item:not(:last-child) {
  border-bottom: 1px solid #F5EFE7;
}

.notification-item.unread {
  background: #FFF9F0;
}

.notification-item.unread:hover {
  background: #FFF3E0;
}

.unread-badge {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 8px;
  height: 8px;
  background: #B87333;
  border-radius: 50%;
}

.notification-avatar {
  width: 40px;
  height: 40px;
  min-width: 40px;
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

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-text {
  font-size: 14px;
  color: #4B3621;
  margin: 0 0 4px 0;
  line-height: 1.5;
}

.username {
  font-weight: 600;
  color: #804030;
}

.notification-time {
  font-size: 12px;
  color: #8D7B68;
}

/* 加载更多 */
.load-more {
  padding: 20px;
  text-align: center;
}

.load-more-btn {
  padding: 10px 24px;
  background: transparent;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  color: #804030;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
}

.load-more-btn:hover {
  background: #804030;
  color: #fff;
  border-color: #804030;
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
