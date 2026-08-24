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
const websiteUrl = 'https://totalrpbox.com'
const supportUrl = 'https://github.com/1120370331/RPBox/issues'
const publicPrivacyUrl = 'https://totalrpbox.com/privacy.html'
const {
  currentVersion,
  currentTarget,
  checking,
  updateAvailable,
  updateInfo,
  updateMode,
  updating,
  downloadProgress,
  downloadedBytes,
  totalBytes,
  installPermissionRequired,
  installPermissionGranted,
  lastError,
  checkForUpdate,
  installUpdate,
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

const updateActionLabel = computed(() => {
  if (updating.value) {
    return t('profile.about.update.downloading', { percent: downloadProgress.value })
  }
  if (updateMode.value === 'android-in-app') {
    return t('profile.about.update.downloadInstall')
  }
  if (updateMode.value === 'ios-store') {
    return t('profile.about.update.openStore')
  }
  return t('profile.about.update.openUpdate')
})

const updateModeHint = computed(() => {
  if (!updateAvailable.value) return ''
  if (
    updateMode.value === 'android-in-app'
    && installPermissionRequired.value
    && !installPermissionGranted.value
  ) {
    return t('profile.about.update.installPermissionRequired')
  }
  if (updateMode.value === 'android-in-app') {
    return t('profile.about.update.androidInAppHint')
  }
  if (updateMode.value === 'ios-store') {
    return t('profile.about.update.iosStoreHint')
  }
  return t('profile.about.update.externalHint')
})

const updateProgressText = computed(() => {
  if (totalBytes.value > 0) {
    return t('profile.about.update.downloadProgressBytes', {
      percent: downloadProgress.value,
      done: formatBytes(downloadedBytes.value),
      total: formatBytes(totalBytes.value),
    })
  }
  return t('profile.about.update.downloadProgress', { percent: downloadProgress.value })
})

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const digits = value >= 10 || unitIndex === 0 ? 0 : 1
  return `${value.toFixed(digits)} ${units[unitIndex]}`
}

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

async function handleInstallUpdate() {
  const result = await installUpdate()
  if (result === 'installer-opened') {
    toast.success(t('profile.about.update.installerOpening'))
    return
  }
  if (result === 'permission-required') {
    toast.warning(t('profile.about.update.installPermissionRequired'))
    return
  }
  if (result === 'opened-external') {
    toast.info(updateMode.value === 'ios-store'
      ? t('profile.about.update.openingStore')
      : t('profile.about.update.redirecting'))
    return
  }
  if (result === 'missing-update') {
    toast.error(t('profile.about.update.checkManually'))
    return
  }
  if (result === 'failed') {
    toast.error(lastError.value || t('profile.about.update.checkFailed'))
  }
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
        <p v-if="updateModeHint" class="update-hint">{{ updateModeHint }}</p>

        <div v-if="updating" class="update-progress">
          <div class="update-progress-track">
            <span class="update-progress-fill" :style="{ width: `${downloadProgress}%` }" />
          </div>
          <p>{{ updateProgressText }}</p>
        </div>

        <div class="update-actions">
          <button class="action-btn secondary" :disabled="checking || updating" @click="handleCheckUpdate">
            {{ checking ? $t('profile.about.update.checking') : $t('profile.about.update.checkUpdate') }}
          </button>
          <button
            v-if="updateAvailable && updateInfo"
            class="action-btn primary"
            :disabled="updating"
            @click="handleInstallUpdate"
          >
            {{ updateActionLabel }}
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

      <div class="about-card">
        <h3 class="section-title">{{ $t('profile.about.legal.title') }}</h3>
        <p class="update-hint">{{ $t('profile.about.legal.desc') }}</p>
        <div class="update-actions">
          <button class="action-btn secondary" @click="router.push('/legal/terms')">
            {{ $t('auth.register.terms') }}
          </button>
          <button class="action-btn secondary" @click="router.push('/legal/privacy')">
            {{ $t('auth.register.privacy') }}
          </button>
          <a
            class="action-btn secondary"
            :href="publicPrivacyUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('profile.about.legal.publicPrivacy') }}
          </a>
        </div>
      </div>

      <div class="about-card">
        <h3 class="section-title">{{ $t('profile.about.support.title') }}</h3>
        <p class="update-hint">{{ $t('profile.about.support.desc') }}</p>
        <div class="update-actions">
          <a
            class="action-btn secondary"
            :href="websiteUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('profile.about.support.website') }}
          </a>
          <a
            class="action-btn secondary"
            :href="supportUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('profile.about.support.issues') }}
          </a>
        </div>
      </div>

      <div class="about-card">
        <h3 class="section-title">{{ $t('profile.about.sponsors.title') }}</h3>
        <p class="update-hint">{{ $t('profile.about.sponsors.desc') }}</p>
        <div class="update-actions">
          <button class="action-btn secondary" @click="router.push('/about/sponsors')">
            {{ $t('profile.about.sponsors.view') }}
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
.update-progress {
  margin-top: 12px;
  text-align: left;
}
.update-progress-track {
  position: relative;
  width: 100%;
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--color-primary-light);
}
.update-progress-fill {
  position: absolute;
  inset: 0 auto 0 0;
  width: 0;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width 160ms ease;
}
.update-progress p {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.update-actions {
  margin-top: 14px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
}
.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 10px;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  text-decoration: none;
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
