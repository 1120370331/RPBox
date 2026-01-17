import { ref } from 'vue'
import { check } from '@tauri-apps/plugin-updater'
import { relaunch } from '@tauri-apps/plugin-process'

export interface UpdateInfo {
  version: string
  notes?: string
  date?: string
}

const updateAvailable = ref(false)
const updateInfo = ref<UpdateInfo | null>(null)
const checking = ref(false)
const downloading = ref(false)
const downloadProgress = ref(0)
const lastError = ref<string | null>(null)

export function useUpdater() {
  async function checkForUpdate() {
    if (checking.value) return
    checking.value = true
    lastError.value = null

    try {
      console.log('[Updater] 开始检查更新...')
      console.log('[Updater] 当前配置的 endpoint:', import.meta.env.VITE_UPDATER_ENDPOINT || 'tauri.conf.json 中的配置')

      const update = await check()
      console.log('[Updater] 检查结果:', update)

      if (update) {
        updateAvailable.value = true
        updateInfo.value = {
          version: update.version,
          notes: update.body || '',
          date: update.date || '',
        }
        console.log('[Updater] 发现新版本:', update.version)
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
      const update = await check()

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
      })

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
    checking,
    downloading,
    downloadProgress,
    lastError,
    checkForUpdate,
    downloadAndInstall,
  }
}
