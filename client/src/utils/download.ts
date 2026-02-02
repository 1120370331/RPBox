import { save } from '@tauri-apps/plugin-dialog'
import { invoke } from '@tauri-apps/api/core'

/**
 * 强制下载文件（使用 Tauri API）
 */
export async function downloadFile(url: string, filename: string): Promise<void> {
  console.log('[Download] Starting download:', url, filename)
  try {
    const response = await fetch(url)
    console.log('[Download] Fetch response:', response.status, response.ok)
    if (!response.ok) {
      throw new Error(`下载失败: ${response.status}`)
    }
    const blob = await response.blob()
    console.log('[Download] Blob size:', blob.size)

    // 使用 Tauri 保存对话框
    const filePath = await save({
      defaultPath: filename,
    })

    if (filePath) {
      const arrayBuffer = await blob.arrayBuffer()
      const uint8Array = new Uint8Array(arrayBuffer)
      await invoke('save_binary_file', {
        path: filePath,
        data: Array.from(uint8Array),
      })
      console.log('[Download] File saved to:', filePath)
    } else {
      console.log('[Download] User cancelled save dialog')
    }
  } catch (error) {
    console.error('[Download] 下载文件失败:', error)
  }
}

/**
 * 处理附件卡片点击事件
 */
export function handleAttachmentClick(event: MouseEvent): boolean {
  const target = event.target as HTMLElement
  console.log('[Download] Click target:', target.tagName, target.className)

  // 检查是否点击了下载按钮或其子元素
  const downloadBtn = target.closest('.attachment-card__download')
  console.log('[Download] Download btn:', downloadBtn)

  if (!downloadBtn) return false

  event.preventDefault()
  event.stopPropagation()

  const card = downloadBtn.closest('.attachment-card')
  console.log('[Download] Card:', card)

  if (!card) return false

  const href = card.getAttribute('data-href') || downloadBtn.getAttribute('href') || ''
  const filename = card.getAttribute('data-filename') || '下载文件'
  console.log('[Download] href:', href, 'filename:', filename)

  if (href) {
    downloadFile(href, filename)
  }
  return true
}

/**
 * 为容器元素绑定附件下载处理
 */
export function attachDownloadHandler(container: HTMLElement | null): void {
  if (!container) return
  container.addEventListener('click', (e) => handleAttachmentClick(e as MouseEvent))
}
