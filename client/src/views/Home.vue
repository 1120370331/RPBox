<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
const mounted = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const quickActions = computed(() => [
  { icon: 'ri-user-star-line', label: t('nav.menu.sync'), desc: t('home.quickActions.syncDesc'), route: '/sync' },
  { icon: 'ri-book-open-line', label: t('nav.menu.archives'), desc: t('home.quickActions.archivesDesc'), route: '/archives' },
  { icon: 'ri-sword-line', label: t('nav.menu.market'), desc: t('home.quickActions.marketDesc'), route: '/market' },
  { icon: 'ri-settings-3-line', label: t('nav.menu.settings'), desc: t('home.quickActions.settingsDesc'), route: '/settings' },
])
</script>

<template>
  <div class="home-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="breadcrumbs">
        <i class="ri-home-4-line"></i>
        <span class="current">{{ t('home.title') }}</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary">
          <i class="ri-question-line"></i> {{ t('home.help') }}
        </button>
        <button class="btn btn-secondary" @click="router.push('/thanks')">
          <i class="ri-heart-3-line"></i> {{ t('home.thanks') }}
        </button>
      </div>
    </div>

    <!-- 欢迎面板 -->
    <div class="welcome-panel anim-item" style="--delay: 1">
      <div class="welcome-content">
        <h1>{{ t('home.welcome') }}</h1>
        <p>{{ t('home.slogan') }}</p>
      </div>
      <div class="welcome-decoration">
        <i class="ri-quill-pen-line"></i>
      </div>
    </div>

    <!-- 快捷入口 -->
    <div class="quick-grid anim-item" style="--delay: 2">
      <div
        v-for="(action, index) in quickActions"
        :key="action.route"
        class="quick-card"
        @click="router.push(action.route)"
      >
        <div class="card-icon">
          <i :class="action.icon"></i>
        </div>
        <div class="card-info">
          <span class="card-label">{{ action.label }}</span>
          <span class="card-desc">{{ action.desc }}</span>
        </div>
        <i class="ri-arrow-right-s-line card-arrow"></i>
      </div>
    </div>

  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 顶部工具栏 */
.top-toolbar {
  background-color: var(--color-panel-bg, #FFFFFF);
  border-radius: 16px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary, #8C7B70);
}

.breadcrumbs i {
  font-size: 18px;
}

.breadcrumbs .current {
  color: var(--color-secondary, #804030);
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: transform 0.2s;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn-secondary {
  background-color: var(--btn-secondary-bg, rgba(128, 64, 48, 0.1));
  color: var(--color-secondary, #804030);
}

/* 欢迎面板 */
.welcome-panel {
  background: linear-gradient(135deg, var(--color-secondary, #804030) 0%, var(--color-primary, #4B3621) 100%);
  border-radius: 16px;
  padding: 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: var(--shadow-lg, 0 4px 20px rgba(75, 54, 33, 0.15));
}

.welcome-content h1 {
  font-size: 28px;
  color: var(--color-text-light, #FBF5EF);
  margin: 0 0 8px 0;
}

.welcome-content p {
  color: var(--color-sidebar-text-muted, rgba(251, 245, 239, 0.7));
  font-size: 15px;
  margin: 0;
}

.welcome-decoration i {
  font-size: 64px;
  color: var(--color-accent-muted, rgba(212, 163, 115, 0.3));
}

/* 快捷入口 */
.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.quick-card {
  background: var(--color-panel-bg, #FFFFFF);
  border-radius: 16px;
  padding: 20px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
}

.quick-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg, 0 8px 30px rgba(75, 54, 33, 0.1));
}

.card-icon {
  width: 48px;
  height: 48px;
  background: var(--btn-secondary-bg, rgba(128, 64, 48, 0.1));
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon i {
  font-size: 24px;
  color: var(--color-secondary, #804030);
}

.card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.card-label {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
}

.card-desc {
  font-size: 13px;
  color: var(--color-text-secondary, #8C7B70);
}

.card-arrow {
  font-size: 20px;
  color: var(--color-accent, #D4A373);
}

.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}
</style>
