<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useUserStore } from '@shared/stores/user'
import RButton from '@shared/components/RButton.vue'
import RAvatar from '@shared/components/RAvatar.vue'

const userStore = useUserStore()
</script>

<template>
  <div class="layout">
    <header class="header">
      <div class="header-content">
        <RouterLink to="/" class="logo">
          <span class="logo-text">RPBox</span>
        </RouterLink>

        <nav class="nav">
          <RouterLink to="/" class="nav-link">首页</RouterLink>
          <RouterLink to="/download" class="nav-link">下载</RouterLink>
          <RouterLink to="/community" class="nav-link">社区</RouterLink>
        </nav>

        <div class="header-actions">
          <template v-if="userStore.token">
            <RAvatar :src="userStore.user?.avatar" :size="32" />
            <span class="username">{{ userStore.user?.username }}</span>
          </template>
          <template v-else>
            <RouterLink to="/login">
              <RButton variant="ghost">登录</RButton>
            </RouterLink>
            <RouterLink to="/register">
              <RButton variant="primary">注册</RButton>
            </RouterLink>
          </template>
        </div>
      </div>
    </header>

    <main class="main">
      <RouterView />
    </main>

    <footer class="footer">
      <p>RPBox - 魔兽世界 RP 工具箱</p>
    </footer>
  </div>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.logo {
  text-decoration: none;
}

.logo-text {
  font-size: 24px;
  font-weight: bold;
  color: var(--primary-color);
}

.nav {
  display: flex;
  gap: 32px;
}

.nav-link {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 15px;
  transition: color 0.2s;
}

.nav-link:hover,
.nav-link.router-link-active {
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.username {
  color: var(--text-primary);
  font-size: 14px;
}

.main {
  flex: 1;
}

.footer {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 14px;
}
</style>
