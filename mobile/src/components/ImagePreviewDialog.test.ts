import { createApp, defineComponent, h, nextTick, ref, type App } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it } from 'vitest'
import ImagePreviewDialog from './ImagePreviewDialog.vue'

let app: App<Element> | null = null

async function flushUi() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

function provideTranslations(targetApp: App<Element>) {
  const i18n = createI18n({
    legacy: false,
    locale: 'en-US',
    messages: {
      'en-US': {
        common: {
          imagePreview: {
            label: 'Image preview',
            close: 'Close image preview',
          },
        },
      },
    },
  })
  targetApp.use(i18n)
}

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('ImagePreviewDialog accessibility', () => {
  it('labels the modal, focuses close, closes on Escape, and restores focus', async () => {
    let closeCount = 0
    const host = document.createElement('div')
    document.body.appendChild(host)

    const Root = defineComponent({
      setup() {
        const open = ref(false)
        return () => h('div', [
          h('button', {
            id: 'preview-opener',
            onClick: () => {
              open.value = true
            },
          }, 'Open image'),
          h(ImagePreviewDialog, {
            open: open.value,
            src: '/moonwell.jpg',
            alt: 'Moonwell at night',
            onClose: () => {
              closeCount += 1
              open.value = false
            },
          }),
        ])
      },
    })

    app = createApp(Root)
    provideTranslations(app)
    app.mount(host)

    const opener = document.querySelector<HTMLButtonElement>('#preview-opener')!
    opener.focus()
    opener.click()
    await flushUi()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')!
    const closeButton = dialog.querySelector<HTMLButtonElement>('.preview-close')!
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(dialog.getAttribute('aria-label')).toBe('Image preview: Moonwell at night')
    expect(closeButton.getAttribute('aria-label')).toBe('Close image preview')
    expect(document.activeElement).toBe(closeButton)

    opener.focus()
    opener.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(closeButton)
    opener.focus()
    opener.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }))
    expect(document.activeElement).toBe(closeButton)

    const image = dialog.querySelector<HTMLImageElement>('.preview-image')!
    image.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }))
    await nextTick()
    expect(image.style.transform).toContain('scale(2)')
    image.click()
    await nextTick()
    expect(document.querySelector('[role="dialog"]')).not.toBeNull()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushUi()

    expect(closeCount).toBe(1)
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(opener)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(closeCount).toBe(1)
  })

  it('restores external focus and removes its listener when unmounted while open', async () => {
    let closeCount = 0
    const externalButton = document.createElement('button')
    const host = document.createElement('div')
    document.body.append(externalButton, host)
    externalButton.focus()

    app = createApp(ImagePreviewDialog, {
      open: true,
      src: '/moonwell.jpg',
      onClose: () => {
        closeCount += 1
      },
    })
    provideTranslations(app)
    app.mount(host)
    await flushUi()
    expect(document.activeElement).toBe(document.querySelector('.preview-close'))

    app.unmount()
    app = null

    expect(document.activeElement).toBe(externalButton)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(closeCount).toBe(0)
  })
})
