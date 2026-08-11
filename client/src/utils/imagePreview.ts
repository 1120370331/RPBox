export function attachImagePreview(
  container: HTMLElement | null,
  onOpen: (images: string[], index: number) => void,
  label: string
) {
  if (!container) return

  const entries = Array.from(container.querySelectorAll('img'))
    .map(img => ({ img, src: img.currentSrc || img.getAttribute('src') || '' }))
    .filter((entry): entry is { img: HTMLImageElement; src: string } => Boolean(entry.src))
  if (entries.length === 0) return

  const urls = entries.map(entry => entry.src)

  entries.forEach(({ img, src }, index) => {
    if (img.closest('.image-preview')) return

    const wrapper = document.createElement('span')
    wrapper.className = 'image-preview'
    wrapper.setAttribute('role', 'button')
    wrapper.setAttribute('tabindex', '0')
    wrapper.setAttribute('aria-label', label)

    const overlay = document.createElement('span')
    overlay.className = 'image-preview-overlay'
    overlay.textContent = label
    overlay.setAttribute('aria-hidden', 'true')

    const parent = img.parentNode
    if (!parent) return

    parent.insertBefore(wrapper, img)
    wrapper.appendChild(img)
    wrapper.appendChild(overlay)

    const openPreview = (event: Event) => {
      event.preventDefault()
      event.stopPropagation()
      onOpen(urls, index)
    }
    wrapper.addEventListener('click', openPreview)
    wrapper.addEventListener('keydown', (event) => {
      const keyboardEvent = event as KeyboardEvent
      if (keyboardEvent.key !== 'Enter' && keyboardEvent.key !== ' ') return
      openPreview(event)
    })
  })
}
