<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import { useNotificationStore } from '../stores/notification'
import { useRouter, useRoute } from 'vue-router'
import RDialog from './RDialog.vue'
import RToast from './RToast.vue'
import UserLevelBadge from './UserLevelBadge.vue'
import { buildNameStyle } from '@/utils/userNameStyle'
import { handleJumpLinkClick, getJumpReturn, clearJumpReturn, type JumpReturnInfo } from '@/utils/jumpLink'
import { getUserInfo } from '@/api/user'

const { t } = useI18n()
const userStore = useUserStore()
const notificationStore = useNotificationStore()
const router = useRouter()
const route = useRoute()
const mounted = ref(false)
const jumpReturn = ref<JumpReturnInfo | null>(null)
const mainContentRef = ref<HTMLElement | null>(null)
const pendingRestoreMenu = ref<string | null>(null)

interface MenuCacheState {
  path: string
  scrollTop: number
  scrollLeft: number
}

const menuCache = ref<Record<string, MenuCacheState>>({})

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  if (userStore.token) {
    void refreshCurrentUser()
    notificationStore.loadUnreadCount()
  }
  document.addEventListener('click', handleGlobalJumpLink, true)
  refreshJumpReturn()
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleGlobalJumpLink, true)
})

// 侧边栏菜单点击时刷新未读消息数量
function handleMenuClick() {
  if (userStore.token) {
    notificationStore.loadUnreadCount()
  }
}

function ensureMenuCache(menuId: string, path: string = route.fullPath) {
  if (!menuCache.value[menuId]) {
    menuCache.value[menuId] = {
      path,
      scrollTop: 0,
      scrollLeft: 0,
    }
  }
  return menuCache.value[menuId]
}

async function refreshCurrentUser() {
  try {
    const userInfo = await getUserInfo()
    userStore.mergeUser(userInfo)
  } catch (error) {
    console.error('刷新用户信息失败:', error)
  }
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}

function handleGlobalJumpLink(event: MouseEvent) {
  const returnTo = resolvePostReturnTarget(event)
  handleJumpLinkClick(event, router, { ignoreEditor: true, returnTo })
}

function resolvePostReturnTarget(event: MouseEvent) {
  const target = event.target
  const element = target instanceof Element ? target : (target instanceof Node ? target.parentElement : null)
  if (!element) return
  const inPostContent = element.closest('.post-detail-page .article-content, .post-preview-page .article-content')
  if (!inPostContent) return
  return {
    type: 'post' as const,
    path: route.fullPath,
  }
}

function refreshJumpReturn() {
  const value = getJumpReturn()
  if (!value) {
    jumpReturn.value = null
    return
  }
  if (value.path === route.fullPath) {
    clearJumpReturn()
    jumpReturn.value = null
    return
  }
  jumpReturn.value = value
}

function handleReturnToPost() {
  if (!jumpReturn.value) return
  const target = jumpReturn.value.path
  clearJumpReturn()
  jumpReturn.value = null
  router.push(target)
}

const menuItems = computed(() => [
  { id: 'home', icon: 'ri-home-4-line', label: t('nav.menu.home'), route: '/' },
  { id: 'sync', icon: 'ri-user-star-line', label: t('nav.menu.sync'), route: '/sync' },
  { id: 'archives', icon: 'ri-book-open-line', label: t('nav.menu.archives'), route: '/archives' },
  { id: 'market', icon: 'ri-sword-line', label: t('nav.menu.market'), route: '/market' },
  { id: 'community', icon: 'ri-chat-smile-2-line', label: t('nav.menu.community'), route: '/community' },
  { id: 'guild', icon: 'ri-shield-line', label: t('nav.menu.guild'), route: '/guild' },
  { id: 'settings', icon: 'ri-settings-3-line', label: t('nav.menu.settings'), route: '/settings' },
])

// 版主菜单项（仅版主可见）
const moderatorMenuItem = computed(() => {
  if (userStore.isModerator) {
    return { id: 'moderator', icon: 'ri-shield-star-line', label: t('nav.menu.moderator'), route: '/moderator' }
  }
  return null
})

const lastMainMenu = ref<string>('home')

function resolveMenu(path: string): string | null {
  if (path.startsWith('/sync')) return 'sync'
  if (path.startsWith('/archives')) return 'archives'
  if (path.startsWith('/market')) return 'market'
  if (path.startsWith('/community')) return 'community'
  if (path.startsWith('/guild')) return 'guild'
  if (path.startsWith('/settings')) return 'settings'
  if (path.startsWith('/moderator')) return 'moderator'
  if (path === '/' || path === '') return 'home'
  return null
}

const currentMenu = computed(() => resolveMenu(route.path))

watch(currentMenu, (menu) => {
  if (menu) {
    lastMainMenu.value = menu
  }
}, { immediate: true })

const activeMenu = computed(() => {
  if (currentMenu.value) return currentMenu.value
  // 合集页面和收藏夹页面保持在上一个主菜单
  if (route.path.startsWith('/library') || route.path.startsWith('/collection')) {
    return lastMainMenu.value
  }
  return 'home'
})

function saveMenuState(menuId: string | null, path: string = route.fullPath) {
  if (!menuId) return

  const cache = ensureMenuCache(menuId, path)
  const mainContent = mainContentRef.value
  cache.path = path
  cache.scrollTop = mainContent?.scrollTop ?? 0
  cache.scrollLeft = mainContent?.scrollLeft ?? 0
}

async function restoreMenuState(menuId: string) {
  const cache = menuCache.value[menuId]
  if (!cache) return

  await nextTick()
  requestAnimationFrame(() => {
    const mainContent = mainContentRef.value
    if (mainContent) {
      mainContent.scrollTo({
        top: cache.scrollTop,
        left: cache.scrollLeft,
        behavior: 'auto',
      })
    } else {
      window.scrollTo({
        top: cache.scrollTop,
        left: cache.scrollLeft,
        behavior: 'auto',
      })
    }
  })
}

function handleMainContentScroll() {
  saveMenuState(activeMenu.value)
}

async function handleMenuNavigate(menuId: string, fallbackRoute: string) {
  handleMenuClick()
  if (activeMenu.value === menuId) return

  saveMenuState(activeMenu.value)

  const cache = menuCache.value[menuId]
  pendingRestoreMenu.value = menuId
  await router.push(cache?.path || fallbackRoute)
}

watch(() => route.fullPath, async (newPath, oldPath) => {
  saveMenuState(resolveMenu(oldPath || ''))

  if (activeMenu.value) {
    ensureMenuCache(activeMenu.value, newPath).path = newPath
  }

  refreshJumpReturn()

  if (pendingRestoreMenu.value && pendingRestoreMenu.value === activeMenu.value) {
    const menuId = pendingRestoreMenu.value
    pendingRestoreMenu.value = null
    await restoreMenuState(menuId)
  }
}, { flush: 'post' })

onMounted(() => {
  if (activeMenu.value) {
    ensureMenuCache(activeMenu.value, route.fullPath)
  }
  saveMenuState(activeMenu.value)
})

onBeforeUnmount(() => {
  saveMenuState(activeMenu.value)
})
</script>

<template>
  <div class="app-layout" :class="{ 'animate-in': mounted }">
    <!-- 侧边栏 -->
    <aside class="sidebar">
      <div class="logo-area">
        <div class="logo-brand">
          <i class="ri-box-3-fill logo-icon"></i>
          <span>RPBox</span>
        </div>
        <router-link
          v-if="userStore.token"
          to="/notifications"
          class="notification-btn top"
          :title="t('nav.user.notifications')"
        >
          <i class="ri-notification-3-line"></i>
          <span v-if="notificationStore.unreadCount > 0" class="notification-badge">{{ notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount }}</span>
        </router-link>
      </div>

      <nav class="menu">
        <button
          v-for="item in menuItems"
          :key="item.id"
          type="button"
          class="menu-item"
          :class="{ active: activeMenu === item.id }"
          @click="handleMenuNavigate(item.id, item.route)"
        >
          <i :class="item.icon"></i>
          <span>{{ item.label }}</span>
        </button>

        <!-- 版主中心（仅版主可见） -->
        <button
          v-if="moderatorMenuItem"
          type="button"
          class="menu-item moderator-item"
          :class="{ active: activeMenu === 'moderator' }"
          @click="handleMenuNavigate('moderator', moderatorMenuItem.route)"
        >
          <i :class="moderatorMenuItem.icon"></i>
          <span>{{ moderatorMenuItem.label }}</span>
        </button>
      </nav>

      <div class="user-profile">
        <template v-if="userStore.token">
          <router-link :to="`/user/${userStore.user?.id}`" class="avatar-link">
            <div class="avatar">
              <img v-if="userStore.user?.avatar" :src="userStore.user.avatar" alt="头像" />
              <span v-else>{{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}</span>
            </div>
          </router-link>
          <div class="user-info">
            <div class="user-name-row">
              <router-link :to="`/user/${userStore.user?.id}`" class="username-link">
                <h4 :style="buildNameStyle(userStore.user?.name_color, userStore.user?.name_bold)">{{ userStore.user?.username }}</h4>
              </router-link>
              <UserLevelBadge
                :level="userStore.user?.forum_level"
                :name="userStore.user?.forum_level_name"
                :color="userStore.user?.forum_level_color"
                :bold="userStore.user?.forum_level_bold"
                size="xs"
              />
            </div>
            <span class="user-points">积分 {{ userStore.user?.activity_points ?? 0 }}</span>
            <p class="logout-link" @click="handleLogout">{{ t('nav.user.logout') }}</p>
          </div>
        </template>
        <router-link v-else to="/login" class="login-btn">
          <i class="ri-login-box-line"></i>
          <span>{{ t('nav.user.login') }}</span>
        </router-link>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main ref="mainContentRef" class="main-content" @scroll.passive="handleMainContentScroll">
      <div v-if="jumpReturn?.type === 'post'" class="jump-return-bar">
        <button class="jump-return-btn" type="button" @click="handleReturnToPost">
          <i class="ri-arrow-left-line"></i>
          {{ t('nav.action.returnToPost') }}
        </button>
      </div>
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
  color: var(--color-sidebar-text, #FBF5EF);
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
  justify-content: space-between;
  gap: 12px;
  padding: 0 24px;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 1px;
  border-bottom: 1px solid rgba(238, 217, 196, 0.1);
}

.logo-brand {
  display: flex;
  align-items: center;
  min-width: 0;
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
  border: none;
  background: transparent;
  width: 100%;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 15px;
  text-align: left;
  color: var(--color-sidebar-text-muted, rgba(251, 245, 239, 0.7));
  text-decoration: none;
}

.menu-item i {
  font-size: 20px;
  margin-right: 12px;
}

.menu-item:hover {
  background-color: var(--color-sidebar-hover, rgba(238, 217, 196, 0.1));
  color: var(--color-sidebar-text, #FBF5EF);
}

.menu-item.active {
  background: linear-gradient(135deg, var(--color-accent, #D4A373), var(--color-accent-hover, #B87333));
  color: var(--btn-primary-text, var(--color-accent-contrast, #2C1810));
  font-weight: bold;
  box-shadow: 0 10px 24px -16px rgba(var(--shadow-base, 75, 54, 33), 0.8);
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
  background: linear-gradient(135deg, var(--color-accent, #D4A373), var(--color-text-secondary, #8C7B70));
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  color: var(--btn-primary-text, #FFF);
  border: 2px solid rgba(255,255,255,0.2);
  overflow: hidden;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-link {
  text-decoration: none;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.user-name-row {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  min-height: 18px;
  min-width: 0;
}

.user-points {
  font-size: 11px;
  color: rgba(251, 245, 239, 0.72);
  white-space: nowrap;
}

.notification-btn {
  position: relative;
  width: 40px;
  height: 40px;
  min-width: 40px;
  min-height: 40px;
  background: rgba(238, 217, 196, 0.2);
  border-radius: 8px;
  border: 1px solid rgba(238, 217, 196, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-sidebar-text-muted, rgba(251, 245, 239, 0.8));
  text-decoration: none;
  font-size: 20px;
  transition: all 0.3s;
  flex-shrink: 0;
}

.notification-btn.top {
  width: 36px;
  height: 36px;
  min-width: 36px;
  min-height: 36px;
  border-radius: 10px;
}

.notification-btn:hover {
  background: var(--color-sidebar-hover, rgba(238, 217, 196, 0.3));
  border-color: rgba(238, 217, 196, 0.5);
  color: var(--color-sidebar-text, #FBF5EF);
}

.notification-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  background: #DC143C;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--color-sidebar-bg, #4B3621);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.username-link {
  text-decoration: none;
  min-width: 0;
  flex: 0 1 auto;
}

.username-link h4 {
  font-size: 14px;
  color: var(--color-sidebar-text, #FBF5EF);
  margin: 0;
  transition: color 0.3s;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.username-link:hover h4 {
  color: var(--color-accent, #D4A373);
}

.logout-link {
  font-size: 12px;
  color: var(--color-sidebar-text-muted, rgba(251, 245, 239, 0.5));
  margin: 0;
  cursor: pointer;
  transition: color 0.3s;
}

.logout-link:hover {
  color: var(--color-sidebar-text, rgba(251, 245, 239, 0.8));
}

.login-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-sidebar-text-muted, rgba(251, 245, 239, 0.7));
  text-decoration: none;
  font-size: 14px;
}

.login-btn:hover {
  color: var(--color-sidebar-text, #FBF5EF);
}

/* 版主菜单项特殊样式 */
.menu-item.moderator-item {
  margin-top: auto;
  background: linear-gradient(135deg, rgba(184, 115, 51, 0.2), rgba(128, 64, 48, 0.2));
  border: 1px solid rgba(184, 115, 51, 0.3);
}

.menu-item.moderator-item:hover {
  background: linear-gradient(135deg, rgba(184, 115, 51, 0.3), rgba(128, 64, 48, 0.3));
}

.menu-item.moderator-item.active {
  background: linear-gradient(135deg, #B87333, #804030);
  color: var(--btn-primary-text, #fff);
}

/* 主内容区 */
.main-content {
  flex: 1;
  overflow-y: auto;
  background: var(--color-main-bg, #EED9C4);
  padding: 24px;
}

.jump-return-bar {
  position: sticky;
  top: 16px;
  z-index: 20;
  display: flex;
  justify-content: flex-start;
  pointer-events: none;
  margin-bottom: 12px;
}

.jump-return-btn {
  pointer-events: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 999px;
  border: 1px solid #E5D4C1;
  background: rgba(255, 255, 255, 0.92);
  color: #4B3621;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(44, 24, 16, 0.08);
  backdrop-filter: blur(6px);
  transition: all 0.2s ease;
}

.jump-return-btn:hover {
  border-color: #B87333;
  color: #B87333;
  transform: translateY(-1px);
}

@media (max-width: 767px) {
  .logo-area {
    padding: 0 16px;
  }

  .logo-brand {
    font-size: 22px;
  }

  .notification-btn.top {
    width: 34px;
    height: 34px;
    min-width: 34px;
    min-height: 34px;
  }

  .user-profile {
    padding: 16px;
    gap: 10px;
  }

  .user-name-row {
    gap: 3px;
  }

  .user-points,
  .logout-link {
    font-size: 11px;
  }
}
</style>
