<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUserStore } from '../stores/user'
import { useRouter } from 'vue-router'

const userStore = useUserStore()
const router = useRouter()
const activeMenu = ref('home')
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

function handleLogout() {
  userStore.logout()
  router.push('/login')
}

const menuItems = [
  { id: 'home', icon: '🏠', label: '首页' },
  { id: 'profiles', icon: '👤', label: '人物卡', route: '/profiles' },
  { id: 'archives', icon: '📖', label: '剧情档案' },
  { id: 'community', icon: '🏰', label: '社区' },
  { id: 'market', icon: '⚔️', label: '道具市场' },
]
</script>

<template>
  <div class="app-layout" :class="{ 'animate-in': mounted }">
    <!-- 侧边栏 -->
    <aside class="sidebar anim-sidebar">
      <div class="sidebar-header">
        <span class="logo">RPBOX</span>
      </div>

      <nav class="sidebar-nav">
        <div
          v-for="(item, index) in menuItems"
          :key="item.id"
          class="nav-item anim-nav"
          :class="{ active: activeMenu === item.id }"
          :style="{ '--nav-delay': index }"
          @click="activeMenu = item.id; item.route && router.push(item.route)"
        >
          <span class="nav-icon">{{ item.icon }}</span>
          <span class="nav-label">{{ item.label }}</span>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div v-if="userStore.token" class="user-info" @click="handleLogout">
          <div class="user-avatar">{{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}</div>
          <span class="user-name">{{ userStore.user?.username }}</span>
        </div>
        <router-link v-else to="/login" class="login-link">登录</router-link>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content">
      <!-- 顶部栏 -->
      <header class="topbar">
        <h1 class="page-title">首页</h1>
        <div class="topbar-actions">
          <button class="btn-icon">⚙️</button>
        </div>
      </header>

      <!-- 内容区 -->
      <div class="content-area">
        <!-- 欢迎 -->
        <div class="welcome-section anim-item" style="--delay: 0">
          <h2>欢迎回来</h2>
          <p>艾泽拉斯的故事，由你书写</p>
        </div>

        <!-- 大图资讯 -->
        <div class="news-section anim-item" style="--delay: 1">
          <div class="news-scroll">
            <div class="news-card">
              <div class="news-image">📜</div>
              <div class="news-overlay">
                <div class="news-tag">公告</div>
                <h3 class="news-title">RPBox 1.0 正式发布</h3>
              </div>
            </div>
            <div class="news-card">
              <div class="news-image">⚔️</div>
              <div class="news-overlay">
                <div class="news-tag">活动</div>
                <h3 class="news-title">RP 剧情创作大赛开启</h3>
              </div>
            </div>
            <div class="news-card">
              <div class="news-image">🏰</div>
              <div class="news-overlay">
                <div class="news-tag">社区</div>
                <h3 class="news-title">本周热门人物卡精选</h3>
              </div>
            </div>
          </div>
        </div>

        <!-- 快捷跳转 -->
        <div class="quick-nav anim-item" style="--delay: 2">
          <div class="quick-item" @click="router.push('/profiles')">
            <span class="quick-icon">👤</span>
            <span class="quick-label">我的人物卡</span>
          </div>
          <div class="quick-item">
            <span class="quick-icon">📖</span>
            <span class="quick-label">剧情档案</span>
          </div>
          <div class="quick-item">
            <span class="quick-icon">🏰</span>
            <span class="quick-label">浏览社区</span>
          </div>
          <div class="quick-item">
            <span class="quick-icon">⚔️</span>
            <span class="quick-label">道具市场</span>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* 应用布局 */
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/* 侧边栏 */
.sidebar {
  width: 200px;
  background: var(--color-primary);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
}

.logo {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-light);
}

.sidebar-nav {
  flex: 1;
  padding: 12px 0;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  color: var(--text-light);
  cursor: pointer;
  transition: all 0.2s;
  opacity: 0.7;
}

.nav-item:hover {
  opacity: 1;
  background: rgba(255,255,255,0.05);
}

.nav-item.active {
  opacity: 1;
  background: var(--color-secondary);
}

.nav-icon {
  font-size: 18px;
}

.nav-label {
  font-size: 14px;
}

/* 侧边栏底部 */
.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid rgba(255,255,255,0.1);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.user-avatar {
  width: 32px;
  height: 32px;
  background: var(--color-accent);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  font-weight: 700;
  font-size: 14px;
}

.user-name {
  color: var(--text-light);
  font-size: 13px;
}

.login-link {
  color: var(--text-light);
  font-size: 14px;
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 顶部栏 */
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: #fff;
  border-bottom: 1px solid rgba(75, 54, 33, 0.1);
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-primary);
}

.btn-icon {
  width: 36px;
  height: 36px;
  background: transparent;
  border: 1px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 16px;
}

/* 内容区 */
.content-area {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

/* 欢迎 */
.welcome-section {
  margin-bottom: 24px;
}

.welcome-section h2 {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 4px;
}

.welcome-section p {
  font-size: 14px;
  color: var(--color-secondary);
}

/* 大图资讯 */
.news-section {
  margin-bottom: 24px;
}

.news-scroll {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding-bottom: 8px;
  scroll-snap-type: x mandatory;
}

.news-scroll::-webkit-scrollbar {
  height: 6px;
}

.news-scroll::-webkit-scrollbar-track {
  background: rgba(75, 54, 33, 0.1);
  border-radius: 3px;
}

.news-scroll::-webkit-scrollbar-thumb {
  background: var(--color-accent);
  border-radius: 3px;
}

.news-card {
  flex-shrink: 0;
  width: 320px;
  aspect-ratio: 16 / 9;
  border-radius: var(--radius-md);
  overflow: hidden;
  position: relative;
  cursor: pointer;
  scroll-snap-align: start;
}

.news-card .news-image {
  width: 100%;
  height: 100%;
  background: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 64px;
}

.news-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16px;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
}

.news-tag {
  display: inline-block;
  padding: 4px 10px;
  background: var(--color-accent);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
  border-radius: 4px;
  margin-bottom: 6px;
}

.news-title {
  font-size: 14px;
  color: #fff;
  font-weight: 500;
}

/* 快捷跳转 */
.quick-nav {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 24px 16px;
  background: #fff;
  border: 1px solid rgba(75, 54, 33, 0.1);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
}

.quick-item:hover {
  border-color: var(--color-accent);
  transform: translateY(-2px);
}

.quick-icon {
  font-size: 32px;
}

.quick-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary);
}

/* ========== 入场动画 ========== */

/* 侧边栏从左滑入 */
.anim-sidebar {
  opacity: 0;
  transform: translateX(-30px);
}

.animate-in .anim-sidebar {
  animation: slideRight 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

/* 导航项依次出现 */
.anim-nav {
  opacity: 0;
  transform: translateX(-20px);
}

.animate-in .anim-nav {
  animation: slideRight 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(0.3s + var(--nav-delay) * 0.05s);
}

/* 内容区元素向上键入 */
.anim-item {
  opacity: 0;
  transform: translateY(30px);
}

.animate-in .anim-item {
  animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: calc(0.2s + var(--delay) * 0.15s);
}

@keyframes slideRight {
  to { opacity: 1; transform: translateX(0); }
}

@keyframes slideUp {
  to { opacity: 1; transform: translateY(0); }
}

/* 快捷卡片悬浮效果 */
.quick-item {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.quick-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(75, 54, 33, 0.12);
}

.quick-item:active {
  transform: translateY(-2px) scale(0.98);
}

/* 资讯卡片悬浮 */
.news-card {
  transition: transform 0.3s ease;
}

.news-card:hover {
  transform: scale(1.02);
}
</style>
