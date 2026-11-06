import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useThemeStore } from '@shared/stores/theme'

describe('mobile system theme preference', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  afterEach(() => vi.unstubAllGlobals())

  it('changes its palette with the system unless a theme was chosen', () => {
    let onChange: ((event: { matches: boolean }) => void) | undefined
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: (_type: string, callback: typeof onChange) => { onChange = callback },
    })))

    const theme = useThemeStore()
    theme.initTheme()
    expect(theme.currentThemeId).toBe('system')
    expect(document.documentElement.dataset.theme).toBe('classic')
    onChange?.({ matches: true })
    expect(document.documentElement.dataset.theme).toBe('dark-coffee')
    expect(document.documentElement.style.colorScheme).toBe('dark')

    theme.setTheme('teal')
    onChange?.({ matches: false })
    expect(document.documentElement.dataset.theme).toBe('teal')
  })
})
