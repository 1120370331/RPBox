import { computed, ref } from 'vue'
import { check, type CheckOptions, type DownloadOptions } from '@tauri-apps/plugin-updater'
import { relaunch } from '@tauri-apps/plugin-process'
import { getDesktopLatestRelease, normalizeDesktopLatestRelease, type NormalizedDesktopLatestRelease, type UpdateChannel } from '@/api/updater'
import { useUserStore } from '@/stores/user'
import type { UserData } from '@/types/user'

export interface UpdateInfo {
  version: string
  notes?: string
  date?: string
  channel: UpdateChannel
}

export type LatestReleaseInfo = NormalizedDesktopLatestRelease

const updateAvailable = ref(false)
const updateInfo = ref<UpdateInfo | null>(null)
const latestRelease = ref<NormalizedDesktopLatestRelease | null>(null)
const latestReleaseLoading = ref(false)
const latestReleaseError = ref<string | null>(null)
const checking = ref(false)
const downloading = ref(false)
const downloadProgress = ref(0)
const lastError = ref<string | null>(null)
const participateTesting = ref(localStorage.getItem('rpbox_participate_testing') === 'true')

const betaUpdateHeader = 'X-RPBox-Update-Channel'

export function canAccessBetaUpdates(user?: UserData | null): boolean {
  if (!user) return false
  if (user.role === 'admin') return true

  const sponsorLevel = typeof user.sponsor_level === 'number'
    ? user.sponsor_level
    : (user.is_sponsor ? 1 : 0)
  return sponsorLevel >= 1
}

function hasPrereleaseVersion(version: string): boolean {
  return version.trim().replace(/^v/i, '').split('+')[0].includes('-')
}

function resolveRawUpdateChannel(rawJson: Record<string, unknown> | undefined, version: string): UpdateChannel {
  if (rawJson?.channel === 'beta') return 'beta'
  if (rawJson?.channel === 'stable') return 'stable'
  return hasPrereleaseVersion(version) ? 'beta' : 'stable'
}

function setParticipateTesting(value: boolean) {
  participateTesting.value = value
  localStorage.setItem('rpbox_participate_testing', String(value))
}

export function useUpdater() {
  const userStore = useUserStore()
  const betaUpdatesAvailable = computed(() => canAccessBetaUpdates(userStore.user))
  const betaUpdatesEnabled = computed(() => participateTesting.value && betaUpdatesAvailable.value)

  function buildCheckOptions(): CheckOptions | undefined {
    if (!betaUpdatesEnabled.value) return undefined

    const token = userStore.token || localStorage.getItem('token')
    if (!token) return undefined

    return {
      headers: {
        Authorization: `Bearer ${token}`,
        [betaUpdateHeader]: 'beta',
      },
    }
  }

  function buildDownloadOptions(checkOptions?: CheckOptions): DownloadOptions | undefined {
    if (!checkOptions?.headers) return undefined
    return { headers: checkOptions.headers }
  }

  async function fetchLatestRelease(force = false): Promise<LatestReleaseInfo | null> {
    const channel: UpdateChannel = betaUpdatesEnabled.value ? 'beta' : 'stable'
    if (latestReleaseLoading.value) {
      return latestRelease.value
    }
    if (!force && latestRelease.value?.channel === channel) {
      return latestRelease.value
    }

    latestReleaseLoading.value = true
    latestReleaseError.value = null

    try {
      latestRelease.value = await getDesktopLatestRelease({ channel })
      return latestRelease.value
    } catch (e: any) {
      latestReleaseError.value = e?.message || e?.toString() || '未知错误'
      console.error('[Updater] 获取最新版本信息失败:', e)
      return null
    } finally {
      latestReleaseLoading.value = false
    }
  }

  async function checkForUpdate() {
    if (checking.value) return
    checking.value = true
    lastError.value = null

    try {
      console.log('[Updater] 开始检查更新...')
      console.log('[Updater] 当前配置的 endpoint:', import.meta.env.VITE_UPDATER_ENDPOINT || 'tauri.conf.json 中的配置')
      console.log('[Updater] 当前更新通道:', betaUpdatesEnabled.value ? 'beta' : 'stable')

      const checkOptions = buildCheckOptions()
      const update = await check(checkOptions)
      console.log('[Updater] 检查结果:', update)

      if (update) {
        const channel = resolveRawUpdateChannel(update.rawJson, update.version)
        updateAvailable.value = true
        updateInfo.value = {
          version: update.version,
          notes: update.body || '',
          date: update.date || '',
          channel,
        }
        latestRelease.value = normalizeDesktopLatestRelease({
          latest_version: update.version,
          version: update.version,
          notes: update.body || '',
          pub_date: update.date || '',
          channel,
        })
        console.log('[Updater] 发现新版本:', update.version, 'channel=', channel)
        console.log('[Updater] 更新说明:', update.body)
        return update
      } else {
        console.log('[Updater] 当前已是最新版本')
      }
    } catch (e: any) {
      const errorMsg = e?.message || e?.toString() || '未知错误'
      lastError.value = errorMsg
      console.error('[Updater] 检查更新失败:', e)
      console.error('[Updater] 错误详情:', {
        message: e?.message,
        stack: e?.stack,
        name: e?.name,
        toString: e?.toString(),
      })
      throw e // 重新抛出错误，让调用方处理
    } finally {
      checking.value = false
    }
    return null
  }

  async function downloadAndInstall() {
    if (downloading.value) return
    downloading.value = true
    downloadProgress.value = 0
    lastError.value = null

    try {
      console.log('[Updater] 开始下载更新...')
      const checkOptions = buildCheckOptions()
      const update = await check(checkOptions)

      if (!update) {
        console.log('[Updater] 没有可用的更新')
        return
      }

      console.log('[Updater] 开始下载版本:', update.version)
      console.log('[Updater] 下载地址:', update.downloadUrl)

      await update.downloadAndInstall((event) => {
        if (event.event === 'Progress') {
          const { contentLength, chunkLength } = event.data as {
            contentLength: number
            chunkLength: number
          }
          if (contentLength) {
            downloadProgress.value += (chunkLength / contentLength) * 100
          }
          console.log('[Updater] 下载进度:', downloadProgress.value.toFixed(2) + '%')
        }
      }, buildDownloadOptions(checkOptions))

      console.log('[Updater] 下载完成，准备重启应用...')
      // 安装完成后重启
      await relaunch()
    } catch (e: any) {
      const errorMsg = e?.message || e?.toString() || '未知错误'
      lastError.value = errorMsg
      console.error('[Updater] 下载更新失败:', e)
      console.error('[Updater] 错误详情:', {
        message: e?.message,
        stack: e?.stack,
        name: e?.name,
        toString: e?.toString(),
      })
      throw e // 重新抛出错误，让调用方处理
    } finally {
      downloading.value = false
    }
  }

  return {
    updateAvailable,
    updateInfo,
    latestRelease,
    latestReleaseLoading,
    latestReleaseError,
    checking,
    downloading,
    downloadProgress,
    lastError,
    participateTesting,
    betaUpdatesAvailable,
    betaUpdatesEnabled,
    setParticipateTesting,
    fetchLatestRelease,
    checkForUpdate,
    downloadAndInstall,
  }
}
