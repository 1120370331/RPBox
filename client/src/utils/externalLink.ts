function isTauriRuntime() {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}

export function resolveExternalUrl(rawHref: string | null | undefined): string | null {
  const href = rawHref?.trim()
  if (!href) return null

  let url: URL
  try {
    url = new URL(href, window.location.href)
  } catch {
    return null
  }

  if (url.protocol !== 'http:' && url.protocol !== 'https:') return null
  if (url.origin === window.location.origin) return null
  return url.toString()
}

export async function openExternalUrl(url: string) {
  if (isTauriRuntime()) {
    const { open } = await import('@tauri-apps/plugin-shell')
    await open(url)
    return
  }

  window.open(url, '_blank', 'noopener,noreferrer')
}

export function handleExternalLinkClick(event: MouseEvent): boolean {
  const target = event.target
  const element = target instanceof Element ? target : (target instanceof Node ? target.parentElement : null)
  const anchor = element?.closest('a[href]') as HTMLAnchorElement | null
  if (!anchor) return false

  const url = resolveExternalUrl(anchor.getAttribute('href'))
  if (!url) return false

  event.preventDefault()
  event.stopPropagation()
  if (typeof event.stopImmediatePropagation === 'function') {
    event.stopImmediatePropagation()
  }
  void openExternalUrl(url).catch((error) => {
    console.error('Failed to open external URL:', error)
  })
  return true
}
