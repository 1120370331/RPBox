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

export function useUpdater() {
  async function checkForUpdate() {
    if (checking.value) return
    checking.value = true

    try {
      const update = await check()
      if (update) {
        updateAvailable.value = true
        updateInfo.value = {
          version: update.version,
          notes: update.body || '',
          date: update.date || '',
        }
        return update
      }
    } catch (e) {
      console.error('检查更新失败:', e)
    } finally {
      checking.value = false
    }
    return null
  }

  async function downloadAndInstall() {
    if (downloading.value) return
    downloading.value = true
    downloadProgress.value = 0

    try {
      const update = await check()
      if (!update) return

      await update.downloadAndInstall((event) => {
        if (event.event === 'Progress') {
          const { contentLength, chunkLength } = event.data as {
            contentLength: number
            chunkLength: number
          }
          if (contentLength) {
            downloadProgress.value += (chunkLength / contentLength) * 100
          }
        }
      })

      // 安装完成后重启
      await relaunch()
    } catch (e) {
      console.error('下载更新失败:', e)
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
    checkForUpdate,
    downloadAndInstall,
  }
}
