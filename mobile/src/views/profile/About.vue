<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useMobileUpdater } from '@/composables/useMobileUpdater'
import { clearImageCache } from '@/utils/imageCache'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const {
  currentVersion,
  currentTarget,
  checking,
  updateAvailable,
  updateInfo,
  lastError,
  checkForUpdate,
  openUpdate,
  refreshRuntimeInfo,
} = useMobileUpdater()

const platformLabel = computed(() => {
  if (currentTarget.value === 'android') {
    return t('profile.about.update.platform.android')
  }
  if (currentTarget.value === 'ios') {
    return t('profile.about.update.platform.ios')
  }
  return t('profile.about.update.platform.unsupported')
})

const currentVersionText = computed(() => t('profile.about.version', { v: currentVersion.value || '0.0.0' }))

async function handleCheckUpdate() {
  const update = await checkForUpdate()
  if (lastError.value) {
    toast.error(lastError.value || t('profile.about.update.checkFailed'))
    return
  }
  if (update) {
    toast.success(t('profile.about.update.available', { v: update.version }))
    return
  }
  toast.info(t('profile.about.update.noUpdate'))
}

function handleOpenUpdate() {
  if (openUpdate()) {
    toast.info(t('profile.about.update.redirecting'))
    return
  }
  toast.error(t('profile.about.update.checkManually'))
}

async function handleClearCache() {
  const ok = await clearImageCache()
  if (ok) {
    toast.success(t('profile.about.cache.cleared'))
    return
  }
  toast.info(t('profile.about.cache.empty'))
}

onMounted(async () => {
  await refreshRuntimeInfo()
  await checkForUpdate({ silent: true })
})
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.about.title') }}</h1>
    </header>
    <div class="sub-body about-body">
      <div class="about-card">
        <div class="app-icon"><i class="ri-box-3-fill" /></div>
        <h2>RPBox Mobile</h2>
        <p class="version">{{ currentVersionText }}</p>
      </div>

      <div class="about-card">
        <p>{{ $t('profile.about.description') }}</p>
      </div>

      <div class="about-card">
        <h3 class="section-title">{{ $t('profile.about.update.title') }}</h3>
        <div class="about-row">
          <span>{{ $t('profile.about.update.platformLabel') }}</span>
          <strong>{{ platformLabel }}</strong>
        </div>
        <div class="about-row">
          <span>{{ $t('profile.about.update.currentVersion') }}</span>
          <strong>v{{ currentVersion || '0.0.0' }}</strong>
        </div>
        <div v-if="updateAvailable && updateInfo" class="about-row">
          <span>{{ $t('profile.about.update.latestVersion') }}</span>
          <strong>v{{ updateInfo.version }}</strong>
        </div>
        <p v-if="updateInfo?.notes" class="update-notes">
          {{ updateInfo.notes }}
        </p>
        <p v-else-if="updateAvailable" class="update-hint">{{ $t('profile.about.update.noNotes') }}</p>
        <p v-else class="update-hint">{{ $t('profile.about.update.noUpdate') }}</p>

        <div class="update-actions">
          <button class="action-btn secondary" :disabled="checking" @click="handleCheckUpdate">
            {{ checking ? $t('profile.about.update.checking') : $t('profile.about.update.checkUpdate') }}
          </button>
          <button
            v-if="updateAvailable && updateInfo"
            class="action-btn primary"
            @click="handleOpenUpdate"
          >
            {{ $t('profile.about.update.openUpdate') }}
          </button>
        </div>
      </div>

      <div class="about-card">
        <div class="about-row">
          <span>{{ $t('profile.about.features.sync') }}</span>
        </div>
        <div class="about-row">
          <span>{{ $t('profile.about.features.stories') }}</span>
        </div>
        <div class="about-row">
          <span>{{ $t('profile.about.features.community') }}</span>
        </div>
        <div class="about-row">
          <span>{{ $t('profile.about.features.market') }}</span>
        </div>
      </div>

      <div class="about-card">
        <h3 class="section-title">{{ $t('profile.about.cache.title') }}</h3>
        <p class="update-hint">{{ $t('profile.about.cache.desc') }}</p>
        <div class="update-actions">
          <button class="action-btn secondary" @click="handleClearCache">
            {{ $t('profile.about.cache.clear') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.about-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-bottom: calc(20px + var(--safe-bottom, 0px));
}
.about-card {
  background: var(--color-card-bg); border-radius: var(--radius-md); padding: 20px 18px;
  box-shadow: var(--shadow-sm); text-align: center;
}
.app-icon { font-size: 48px; color: var(--color-accent); margin-bottom: 8px; }
.about-card h2 { font-size: 18px; font-weight: 600; margin-bottom: 4px; }
.version { font-size: 13px; color: var(--color-text-secondary); }
.about-card p { font-size: 14px; line-height: 1.6; color: var(--color-text-secondary); text-align: left; }
.section-title {
  font-size: 15px; font-weight: 600; margin-bottom: 10px; text-align: left; color: var(--text-dark);
}
.about-row {
  padding: 10px 0; border-bottom: 1px solid var(--color-border-light);
  font-size: 14px; color: var(--text-dark); text-align: left;
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
}
.about-row strong {
  color: var(--color-text-secondary);
  font-size: 13px;
}
.about-row:last-child { border-bottom: none; }
.update-notes {
  margin-top: 12px;
  white-space: pre-line;
}
.update-hint {
  margin-top: 12px;
  color: var(--color-text-muted);
}
.update-actions {
  margin-top: 14px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.action-btn {
  border: none;
  border-radius: 10px;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
}
.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.action-btn.primary {
  background: var(--color-primary);
  color: var(--btn-primary-text);
}
.action-btn.secondary {
  background: var(--color-primary-light);
  color: var(--text-dark);
}

@media (max-width: 380px) {
  .about-card {
    padding: 16px 14px;
  }

  .about-row {
    font-size: 13px;
  }
}
</style>
