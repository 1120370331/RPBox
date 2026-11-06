import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useThemeStore } from './theme'

describe('system theme preference', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  afterEach(() => vi.unstubAllGlobals())

  it('follows system changes by default and preserves a manual override', () => {
    let onChange: ((event: { matches: boolean }) => void) | undefined
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      addEventListener: (_type: string, callback: typeof onChange) => { onChange = callback },
    })))

    const theme = useThemeStore()
    theme.initTheme()
    expect(theme.currentThemeId).toBe('system')
    expect(document.documentElement.dataset.theme).toBe('dark-coffee')

    onChange?.({ matches: false })
    expect(document.documentElement.dataset.theme).toBe('classic')

    theme.setTheme('teal')
    onChange?.({ matches: true })
    expect(document.documentElement.dataset.theme).toBe('teal')

    theme.setTheme('system')
    expect(document.documentElement.dataset.theme).toBe('dark-coffee')
    expect(localStorage.getItem('theme')).toBe('system')
  })
})
