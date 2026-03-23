<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { isFeatureEnabled } from '@/config/mobileFeatures'

const route = useRoute()
const { t } = useI18n()

const tabs = computed(() => {
  const allTabs = [
    { name: 'community', icon: 'ri-chat-3-line', activeIcon: 'ri-chat-3-fill', label: t('nav.tabs.community') },
    { name: 'stories', icon: 'ri-book-open-line', activeIcon: 'ri-book-open-fill', label: t('nav.tabs.stories') },
    { name: 'market', icon: 'ri-store-2-line', activeIcon: 'ri-store-2-fill', label: t('nav.tabs.market') },
    { name: 'guild', icon: 'ri-shield-line', activeIcon: 'ri-shield-fill', label: t('nav.tabs.guild') },
    { name: 'profile', icon: 'ri-user-line', activeIcon: 'ri-user-fill', label: t('nav.tabs.profile') },
  ]
  return allTabs.filter((tab) => isFeatureEnabled(tab.name))
})

const currentTab = computed(() => route.name as string)
</script>

<template>
  <div class="mobile-layout">
    <main class="mobile-content scroll-container">
      <router-view />
    </main>

    <nav class="tab-bar">
      <router-link
        v-for="tab in tabs"
        :key="tab.name"
        :to="{ name: tab.name }"
        class="tab-item"
        :class="{ active: currentTab === tab.name }"
      >
        <i :class="currentTab === tab.name ? tab.activeIcon : tab.icon" />
        <span>{{ tab.label }}</span>
      </router-link>
    </nav>
  </div>
</template>

<style scoped>
.mobile-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
}

.mobile-content {
  flex: 1;
  overflow-y: auto;
  padding-bottom: calc(var(--tab-bar-height) + var(--safe-bottom, 0px));
}

.tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: calc(var(--tab-bar-height) + var(--safe-bottom, 0px));
  padding-bottom: var(--safe-bottom, 0px);
  background: var(--color-panel-bg);
  border-top: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-around;
  z-index: 100;
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 8px;
  font-size: 10px;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s;
}

.tab-item i {
  font-size: 22px;
}

.tab-item.active {
  color: var(--color-secondary);
}
</style>
