export function attachImagePreview(
  container: HTMLElement | null,
  onOpen: (images: string[], index: number) => void,
  label: string
) {
  if (!container) return

  const images = Array.from(container.querySelectorAll('img'))
  if (images.length === 0) return

  const urls = images
    .map((img) => img.currentSrc || img.getAttribute('src') || '')
    .filter((src) => src)

  images.forEach((img, index) => {
    const src = img.currentSrc || img.getAttribute('src') || ''
    if (!src) return
    if (img.closest('.image-preview')) return

    const wrapper = document.createElement('span')
    wrapper.className = 'image-preview'

    const overlay = document.createElement('span')
    overlay.className = 'image-preview-overlay'
    overlay.textContent = label

    const parent = img.parentNode
    if (!parent) return

    parent.insertBefore(wrapper, img)
    wrapper.appendChild(img)
    wrapper.appendChild(overlay)

    wrapper.addEventListener('click', (event) => {
      event.preventDefault()
      event.stopPropagation()
      onOpen(urls, index)
    })
  })
}
