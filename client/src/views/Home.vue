<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useToastStore } from '@/stores/toast'
import { signInDaily } from '@/api/user'

const router = useRouter()
const { t } = useI18n()
const mounted = ref(false)
const userStore = useUserStore()
const toast = useToastStore()
const signingIn = ref(false)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
})

const quickActions = computed(() => [
  { icon: 'ri-user-star-line', label: t('nav.menu.sync'), desc: t('home.quickActions.syncDesc'), route: '/sync' },
  { icon: 'ri-book-open-line', label: t('nav.menu.archives'), desc: t('home.quickActions.archivesDesc'), route: '/archives' },
  { icon: 'ri-sword-line', label: t('nav.menu.market'), desc: t('home.quickActions.marketDesc'), route: '/market' },
  { icon: 'ri-settings-3-line', label: t('nav.menu.settings'), desc: t('home.quickActions.settingsDesc'), route: '/settings' },
])

const signInHint = computed(() => (
  userStore.user?.signed_in_today
    ? '今日签到已完成，明天可继续领取 +10 积分 / +10 经验'
    : '每日签到可领取 +10 积分 / +10 经验'
))

async function handleDailySignIn() {
  if (signingIn.value || userStore.user?.signed_in_today) return
  signingIn.value = true
  try {
    const result = await signInDaily()
    userStore.mergeUser(result)
    if (result.granted) {
      toast.success(`签到成功，积分+${result.points_delta}，经验+${result.experience_delta}`)
    } else {
      toast.info(result.message)
    }
  } catch (error: any) {
    toast.error(error.message || '签到失败')
  } finally {
    signingIn.value = false
  }
}
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

    <section v-if="userStore.token && userStore.user" class="sign-in-panel anim-item" style="--delay: 1.5">
      <div class="sign-in-copy">
        <span class="sign-in-kicker">每日签到</span>
        <p>{{ signInHint }}</p>
      </div>
      <button
        class="sign-in-btn"
        :class="{ done: userStore.user.signed_in_today }"
        :disabled="signingIn || userStore.user.signed_in_today"
        @click="handleDailySignIn"
      >
        <i :class="userStore.user.signed_in_today ? 'ri-checkbox-circle-fill' : 'ri-rocket-line'"></i>
        <span v-if="userStore.user.signed_in_today">今日已签到</span>
        <span v-else-if="signingIn">签到中...</span>
        <span v-else>立即签到</span>
      </button>
    </section>

    <!-- 快捷入口 -->
    <div class="quick-grid anim-item" style="--delay: 2">
      <div
        v-for="action in quickActions"
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

.sign-in-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-radius: 18px;
  background:
    radial-gradient(circle at top right, rgba(255, 255, 255, 0.4), transparent 36%),
    linear-gradient(135deg, #fff9f2 0%, #f7e7d3 100%);
  border: 1px solid rgba(184, 115, 51, 0.18);
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
}

.sign-in-copy p {
  margin: 0;
  color: var(--color-text-main, #2C1810);
  font-size: 14px;
  line-height: 1.5;
}

.sign-in-kicker {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(184, 115, 51, 0.12);
  color: var(--color-accent, #B87333);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sign-in-btn {
  height: 44px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  background: linear-gradient(135deg, #b87333 0%, #804030 100%);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s, opacity 0.2s;
  box-shadow: 0 12px 24px rgba(128, 64, 48, 0.22);
}

.sign-in-btn:hover:not(:disabled) {
  transform: translateY(-2px);
}

.sign-in-btn:disabled {
  cursor: default;
  opacity: 0.9;
}

.sign-in-btn.done {
  background: linear-gradient(135deg, #5b8c5a 0%, #437347 100%);
}

/* 快捷入口 */
.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

@media (max-width: 900px) {
  .sign-in-panel {
    flex-direction: column;
    align-items: stretch;
  }
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
