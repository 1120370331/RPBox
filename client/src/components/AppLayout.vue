<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useUserStore } from '../stores/user'
import { useRouter, useRoute } from 'vue-router'
import RDialog from './RDialog.vue'
import RToast from './RToast.vue'

const userStore = useUserStore()
const router = useRouter()
const route = useRoute()
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

function handleLogout() {
  userStore.logout()
  router.push('/login')
}

const menuItems = [
  { id: 'home', icon: 'ri-home-4-line', label: '首页', route: '/' },
  { id: 'sync', icon: 'ri-user-star-line', label: '人物卡', route: '/sync' },
  { id: 'archives', icon: 'ri-book-open-line', label: '剧情故事', route: '/archives' },
  { id: 'market', icon: 'ri-sword-line', label: '道具物品', route: '/market' },
  { id: 'community', icon: 'ri-chat-smile-2-line', label: '社区广场', route: '/community' },
  { id: 'settings', icon: 'ri-settings-3-line', label: '系统设置', route: '/settings' },
]

const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/sync')) return 'sync'
  if (path.startsWith('/archives')) return 'archives'
  if (path.startsWith('/market')) return 'market'
  if (path.startsWith('/community')) return 'community'
  if (path.startsWith('/settings')) return 'settings'
  return 'home'
})
</script>

<template>
  <div class="app-layout" :class="{ 'animate-in': mounted }">
    <!-- 侧边栏 -->
    <aside class="sidebar">
      <div class="logo-area">
        <i class="ri-box-3-fill logo-icon"></i>
        <span>RPBox</span>
      </div>

      <nav class="menu">
        <RouterLink
          v-for="item in menuItems"
          :key="item.id"
          class="menu-item"
          :class="{ active: activeMenu === item.id }"
          :to="item.route"
        >
          <i :class="item.icon"></i>
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="user-profile">
        <template v-if="userStore.token">
          <div class="avatar">{{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}</div>
          <div class="user-info" @click="handleLogout">
            <h4>{{ userStore.user?.username }}</h4>
            <p>点击退出</p>
          </div>
        </template>
        <router-link v-else to="/login" class="login-btn">
          <i class="ri-login-box-line"></i>
          <span>登录</span>
        </router-link>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content">
      <router-view />
    </main>

    <!-- 全局弹窗 -->
    <RDialog />

    <!-- 全局消息通知 -->
    <RToast />
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: var(--color-main-bg, #EED9C4);
}

/* 侧边栏 */
.sidebar {
  width: 240px;
  background-color: var(--color-sidebar-bg, #4B3621);
  color: var(--color-text-light, #FBF5EF);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  box-shadow: 4px 0 12px rgba(0,0,0,0.1);
  z-index: 10;
}

.logo-area {
  height: 80px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 1px;
  border-bottom: 1px solid rgba(238, 217, 196, 0.1);
}

.logo-icon {
  margin-right: 12px;
  font-size: 28px;
  color: var(--color-accent, #D4A373);
}

.menu {
  flex: 1;
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 15px;
  color: rgba(251, 245, 239, 0.7);
  text-decoration: none;
}

.menu-item i {
  font-size: 20px;
  margin-right: 12px;
}

.menu-item:hover {
  background-color: rgba(238, 217, 196, 0.1);
  color: var(--color-text-light, #FBF5EF);
}

.menu-item.active {
  background-color: var(--color-main-bg, #EED9C4);
  color: var(--color-sidebar-bg, #4B3621);
  font-weight: bold;
  box-shadow: 0 4px 8px rgba(0,0,0,0.1);
}

.user-profile {
  padding: 24px;
  border-top: 1px solid rgba(238, 217, 196, 0.1);
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  color: #FFF;
  border: 2px solid rgba(255,255,255,0.2);
}

.user-info {
  cursor: pointer;
}

.user-info h4 {
  font-size: 14px;
  color: var(--color-text-light, #FBF5EF);
  margin: 0;
}

.user-info p {
  font-size: 12px;
  color: rgba(251, 245, 239, 0.5);
  margin: 2px 0 0 0;
}

.login-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(251, 245, 239, 0.7);
  text-decoration: none;
  font-size: 14px;
}

.login-btn:hover {
  color: var(--color-text-light, #FBF5EF);
}

/* 主内容区 */
.main-content {
  flex: 1;
  overflow-y: auto;
  background: var(--color-main-bg, #EED9C4);
  padding: 24px;
}
</style>
