import { ref } from 'vue'
import {
  type MobileTarget,
  type MobileUpdateInfo,
  checkMobileUpdate,
  detectMobileTarget,
  getCurrentAppVersion,
  isMobileUpdaterSupported,
  openUpdateUrl,
} from '@/api/updater'

const AUTO_CHECK_AT_KEY = 'rpbox.mobile.updater.last_auto_check_at'
const AUTO_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000

const currentVersion = ref('0.0.0')
const currentTarget = ref<MobileTarget | null>(null)
const checking = ref(false)
const updateAvailable = ref(false)
const updateInfo = ref<MobileUpdateInfo | null>(null)
const lastError = ref<string | null>(null)

interface CheckOptions {
  silent?: boolean
}

interface AutoCheckOptions extends CheckOptions {
  force?: boolean
}

function now() {
  return Date.now()
}

function shouldAutoCheck(force?: boolean): boolean {
  if (force) return true
  const raw = localStorage.getItem(AUTO_CHECK_AT_KEY)
  if (!raw) return true
  const ts = Number(raw)
  if (!Number.isFinite(ts)) return true
  return now() - ts >= AUTO_CHECK_INTERVAL_MS
}

function markAutoCheckAt() {
  localStorage.setItem(AUTO_CHECK_AT_KEY, String(now()))
}

async function refreshRuntimeInfo() {
  currentTarget.value = detectMobileTarget()
  currentVersion.value = await getCurrentAppVersion()
}

export function useMobileUpdater() {
  async function checkForUpdate(options: CheckOptions = {}): Promise<MobileUpdateInfo | null> {
    if (checking.value) {
      return updateInfo.value
    }

    checking.value = true
    lastError.value = null

    try {
      await refreshRuntimeInfo()
      if (!currentTarget.value || !isMobileUpdaterSupported()) {
        updateAvailable.value = false
        updateInfo.value = null
        return null
      }

      const update = await checkMobileUpdate({
        target: currentTarget.value,
        currentVersion: currentVersion.value,
      })

      updateInfo.value = update
      updateAvailable.value = !!update
      return update
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error)
      lastError.value = message
      updateAvailable.value = false
      updateInfo.value = null
      if (!options.silent) {
        console.error('[MobileUpdater] check failed:', error)
      }
      return null
    } finally {
      checking.value = false
    }
  }

  async function autoCheckForUpdate(options: AutoCheckOptions = {}): Promise<MobileUpdateInfo | null> {
    if (!shouldAutoCheck(options.force)) {
      return null
    }

    try {
      return await checkForUpdate({ silent: true })
    } finally {
      markAutoCheckAt()
    }
  }

  function openUpdate() {
    if (!updateInfo.value?.url) {
      return false
    }
    openUpdateUrl(updateInfo.value.url)
    return true
  }

  return {
    currentVersion,
    currentTarget,
    checking,
    updateAvailable,
    updateInfo,
    lastError,
    checkForUpdate,
    autoCheckForUpdate,
    openUpdate,
    refreshRuntimeInfo,
  }
}

