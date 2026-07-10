export interface ShareRouteResult {
  method: 'shared' | 'copied'
  url: string
}

const PUBLIC_SITE_ORIGIN = 'https://totalrpbox.com'

export function buildPublicPostUrl(postId: number): string {
  return `${PUBLIC_SITE_ORIGIN}/posts/${postId}`
}

export function buildPostShareText(html: string, maxLength = 120): string {
  const container = document.createElement('div')
  container.innerHTML = html
  const text = (container.textContent || container.innerText || '')
    .replace(/\s+/g, ' ')
    .trim()

  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength).trimEnd()}…`
}

async function copyTextToClipboard(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)

  try {
    textarea.select()
    if (!document.execCommand('copy')) {
      throw new Error('Failed to copy link')
    }
  } finally {
    textarea.remove()
  }
}

export async function shareRouteLink(options: {
  path: string
  title?: string
  text?: string
}): Promise<ShareRouteResult> {
  const url = new URL(options.path, PUBLIC_SITE_ORIGIN).toString()

  if (navigator.share) {
    await navigator.share({
      title: options.title,
      text: options.text,
      url,
    })
    return { method: 'shared', url }
  }

  await copyTextToClipboard(url)
  return { method: 'copied', url }
}
