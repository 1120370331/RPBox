import { ref } from 'vue'
import {
  type MobileUpdateMode,
  type MobileTarget,
  type MobileUpdateInfo,
  type UpdateDownloadProgress,
  checkMobileUpdate,
  detectMobileTarget,
  getAndroidInstallPermissionStatus,
  getCurrentAppVersion,
  getMobileUpdateMode,
  installAndroidUpdate,
  isMobileUpdaterSupported,
  openUpdateUrl,
  requestAndroidInstallPermission,
} from '@/api/updater'

const AUTO_CHECK_AT_KEY = 'rpbox.mobile.updater.last_auto_check_at'
const AUTO_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000

const currentVersion = ref('0.0.0')
const currentTarget = ref<MobileTarget | null>(null)
const checking = ref(false)
const updateAvailable = ref(false)
const updateInfo = ref<MobileUpdateInfo | null>(null)
const updateMode = ref<MobileUpdateMode>('external')
const updating = ref(false)
const downloadProgress = ref(0)
const downloadedBytes = ref(0)
const totalBytes = ref(0)
const installPermissionRequired = ref(false)
const installPermissionGranted = ref(true)
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
  updateMode.value = getMobileUpdateMode(currentTarget.value)
}

export function useMobileUpdater() {
  async function refreshInstallPermission() {
    const status = await getAndroidInstallPermissionStatus()
    installPermissionRequired.value = status.required
    installPermissionGranted.value = status.granted
    return status
  }

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
      await refreshInstallPermission()

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
    openUpdateUrl(updateInfo.value.url, currentTarget.value)
    return true
  }

  async function installUpdate() {
    if (!updateInfo.value?.url) {
      lastError.value = 'MISSING_UPDATE'
      return 'missing-update' as const
    }

    await refreshRuntimeInfo()
    lastError.value = null

    if (updateMode.value !== 'android-in-app') {
      openUpdate()
      return 'opened-external' as const
    }

    if (updating.value) {
      return 'updating' as const
    }

    const permission = await refreshInstallPermission()
    if (permission.required && !permission.granted) {
      const requested = await requestAndroidInstallPermission()
      const refreshed = await refreshInstallPermission()
      if (!requested.granted && !refreshed.granted) {
        lastError.value = 'INSTALL_PERMISSION_REQUIRED'
        return 'permission-required' as const
      }
    }

    updating.value = true
    downloadProgress.value = 0
    downloadedBytes.value = 0
    totalBytes.value = 0

    try {
      await installAndroidUpdate(updateInfo.value, (progress: UpdateDownloadProgress) => {
        downloadedBytes.value = progress.downloadedBytes
        totalBytes.value = progress.totalBytes
        downloadProgress.value = Math.max(0, Math.min(100, Math.round(progress.percent)))
      })
      downloadProgress.value = 100
      return 'installer-opened' as const
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error)
      lastError.value = message
      if (message.includes('INSTALL_PERMISSION_REQUIRED')) {
        installPermissionRequired.value = true
        installPermissionGranted.value = false
        return 'permission-required' as const
      }
      return 'failed' as const
    } finally {
      updating.value = false
    }
  }

  return {
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
    autoCheckForUpdate,
    installUpdate,
    openUpdate,
    refreshInstallPermission,
    refreshRuntimeInfo,
  }
}
