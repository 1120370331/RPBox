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
      <div class="mobile-content-inner">
        <router-view />
      </div>
    </main>

    <nav class="tab-bar" aria-label="Main navigation">
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
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: var(--app-height, 100dvh);
  height: var(--app-height, 100dvh);
  overflow: hidden;
  overscroll-behavior: none;
}

.mobile-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-top: var(--content-top-gap);
  padding-bottom: calc(var(--tab-bar-height) + var(--safe-bottom, 0px) + 18px);
  overscroll-behavior-y: contain;
  -webkit-overflow-scrolling: touch;
}

.mobile-content-inner {
  min-height: 100%;
}

.tab-bar {
  position: fixed;
  left: calc(var(--safe-left, 0px) + 10px);
  right: calc(var(--safe-right, 0px) + 10px);
  bottom: calc(var(--safe-bottom, 0px) + 8px);
  height: var(--tab-bar-height);
  padding: 4px 8px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid rgba(75, 54, 33, 0.14);
  box-shadow: 0 10px 24px rgba(44, 24, 16, 0.16);
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  justify-content: space-around;
  z-index: 120;
  transform: translateZ(0);
  will-change: transform;
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-width: 0;
  gap: 1px;
  height: 100%;
  border-radius: 12px;
  font-size: 10px;
  font-weight: 600;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s ease, transform 0.2s ease, background-color 0.2s ease;
}

.tab-item i {
  font-size: 20px;
}

.tab-item.active {
  color: var(--color-secondary);
  background: rgba(75, 54, 33, 0.08);
  transform: translateY(-1px);
}

@media (max-width: 360px) {
  .tab-bar {
    left: calc(var(--safe-left, 0px) + 8px);
    right: calc(var(--safe-right, 0px) + 8px);
  }

  .tab-item {
    font-size: 9px;
  }

  .tab-item i {
    font-size: 19px;
  }
}
</style>
