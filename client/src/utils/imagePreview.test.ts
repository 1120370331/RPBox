import { describe, expect, it, vi } from 'vitest'
import { attachImagePreview } from './imagePreview'

describe('attachImagePreview', () => {
  it('opens image groups with mouse and keyboard access', () => {
    const container = document.createElement('div')
    container.innerHTML = '<img src="/one.png" alt="one"><img src="/two.png" alt="two">'
    const onOpen = vi.fn()

    attachImagePreview(container, onOpen, '查看大图')

    const triggers = container.querySelectorAll<HTMLElement>('.image-preview')
    const expectedImages = [
      new URL('/one.png', window.location.href).href,
      new URL('/two.png', window.location.href).href,
    ]
    expect(triggers).toHaveLength(2)
    expect(triggers[0].getAttribute('role')).toBe('button')
    expect(triggers[0].getAttribute('tabindex')).toBe('0')
    expect(triggers[0].getAttribute('aria-label')).toBe('查看大图')

    triggers[1].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(onOpen).toHaveBeenLastCalledWith(expectedImages, 1)

    triggers[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    expect(onOpen).toHaveBeenLastCalledWith(expectedImages, 0)
  })

  it('keeps the clicked index when the same image appears more than once', () => {
    const container = document.createElement('div')
    container.innerHTML = '<img src="/same.png"><img src="/same.png">'
    const onOpen = vi.fn()

    attachImagePreview(container, onOpen, '查看大图')
    container.querySelectorAll<HTMLElement>('.image-preview')[1]
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(onOpen).toHaveBeenCalledWith([
      new URL('/same.png', window.location.href).href,
      new URL('/same.png', window.location.href).href,
    ], 1)
  })
})
