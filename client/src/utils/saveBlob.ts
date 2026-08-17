function isTauriRuntime() {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}

function browserDownload(blob: Blob, filename: string) {
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = filename
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 1000)
}

/** Saves a generated file through the native Tauri picker or browser download. */
export async function saveBlobAsFile(blob: Blob, filename: string): Promise<boolean> {
  if (!isTauriRuntime()) {
    browserDownload(blob, filename)
    return true
  }

  const [{ save }, { invoke }] = await Promise.all([
    import('@tauri-apps/plugin-dialog'),
    import('@tauri-apps/api/core'),
  ])
  const path = await save({
    defaultPath: filename,
    filters: [{ name: 'PNG image', extensions: ['png'] }],
  })
  if (!path) return false

  const data = Array.from(new Uint8Array(await blob.arrayBuffer()))
  await invoke('save_binary_file', { path, data })
  return true
}
