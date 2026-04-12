import { Capacitor } from '@capacitor/core'
import { Clipboard } from '@capacitor/clipboard'
import { Directory, Encoding, Filesystem } from '@capacitor/filesystem'
import { Share } from '@capacitor/share'
import { buildPublicSitePathUrl } from './appLink'

interface ShareTextFileOptions {
  filename: string
  content: string
  title?: string
  text?: string
  dialogTitle?: string
}

interface ShareRouteLinkOptions {
  path: string
  title?: string
  text?: string
  dialogTitle?: string
}

function sanitizeFilenamePart(input: string) {
  const collapsed = input
    .replace(/[<>:"/\\|?*\u0000-\u001f]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  return collapsed || 'RPBox'
}

function ensureTxtFilename(input: string) {
  const base = sanitizeFilenamePart(input)
  return /\.txt$/i.test(base) ? base : `${base}.txt`
}

function downloadTextFile(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function writeClipboardFallback(text: string) {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.top = '-9999px'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  const succeeded = document.execCommand('copy')
  document.body.removeChild(textarea)
  if (!succeeded) {
    throw new Error('Clipboard copy failed')
  }
}

export async function copyTextToClipboard(text: string, label?: string) {
  if (Capacitor.isNativePlatform()) {
    await Clipboard.write({
      string: text,
      label,
    })
    return
  }

  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  writeClipboardFallback(text)
}

export async function shareTextFile(options: ShareTextFileOptions) {
  const filename = ensureTxtFilename(options.filename)

  if (Capacitor.isNativePlatform()) {
    const filePath = `shares/${Date.now()}-${filename}`
    const result = await Filesystem.writeFile({
      path: filePath,
      data: options.content,
      directory: Directory.Cache,
      encoding: Encoding.UTF8,
      recursive: true,
    })

    await Share.share({
      title: options.title,
      text: options.text,
      files: [result.uri],
      dialogTitle: options.dialogTitle || options.title,
    })
    return
  }

  const file = new File([options.content], filename, {
    type: 'text/plain;charset=utf-8',
  })

  if (navigator.canShare?.({ files: [file] }) && navigator.share) {
    await navigator.share({
      title: options.title,
      text: options.text,
      files: [file],
    })
    return
  }

  downloadTextFile(filename, options.content)
}

export async function shareRouteLink(options: ShareRouteLinkOptions) {
  const url = buildPublicSitePathUrl(options.path)

  if (Capacitor.isNativePlatform()) {
    await Share.share({
      title: options.title,
      text: options.text,
      url,
      dialogTitle: options.dialogTitle || options.title,
    })
    return url
  }

  if (navigator.share) {
    await navigator.share({
      title: options.title,
      text: options.text,
      url,
    })
    return url
  }

  await copyTextToClipboard(url, options.title)
  return url
}
