import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { check } from '@tauri-apps/plugin-updater'
import { canAccessBetaUpdates, useUpdater } from '@/composables/useUpdater'
import { useUserStore } from '@/stores/user'
import type { UserData } from '@/types/user'

vi.mock('@tauri-apps/plugin-updater', () => ({
  check: vi.fn(),
}))

vi.mock('@tauri-apps/plugin-process', () => ({
  relaunch: vi.fn(),
}))

function user(overrides: Partial<UserData> = {}): UserData {
  return {
    id: 1,
    username: 'tester',
    role: 'user',
    ...overrides,
  }
}

describe('desktop beta update eligibility', () => {
  const mockedCheck = vi.mocked(check)

  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    mockedCheck.mockReset()
    mockedCheck.mockResolvedValue(null)
  })

  it('does not allow normal users to enable beta updates', () => {
    expect(canAccessBetaUpdates(user())).toBe(false)
  })

  it('requires sponsor level 1 or above', () => {
    expect(canAccessBetaUpdates(user({ is_sponsor: true, sponsor_level: 0 }))).toBe(false)
    expect(canAccessBetaUpdates(user({ sponsor_level: 1 }))).toBe(true)
    expect(canAccessBetaUpdates(user({ sponsor_level: 2 }))).toBe(true)
  })

  it('keeps legacy sponsor flag as level 1 when sponsor_level is missing', () => {
    expect(canAccessBetaUpdates(user({ is_sponsor: true }))).toBe(true)
  })

  it('grants beta access to admins but not moderators', () => {
    expect(canAccessBetaUpdates(user({ role: 'moderator' }))).toBe(false)
    expect(canAccessBetaUpdates(user({ role: 'admin' }))).toBe(true)
  })

  it('sends beta update headers only after an eligible sponsor enables testing participation', async () => {
    const userStore = useUserStore()
    userStore.setAuth('sponsor-token', user({ sponsor_level: 1 }))

    const updater = useUpdater()
    updater.setParticipateTesting(true)
    await updater.checkForUpdate()

    expect(mockedCheck).toHaveBeenCalledWith({
      headers: {
        Authorization: 'Bearer sponsor-token',
        'X-RPBox-Update-Channel': 'beta',
      },
    })
  })

  it('sends beta update headers after an admin enables testing participation', async () => {
    const userStore = useUserStore()
    userStore.setAuth('admin-token', user({ role: 'admin' }))

    const updater = useUpdater()
    updater.setParticipateTesting(true)
    await updater.checkForUpdate()

    expect(mockedCheck).toHaveBeenCalledWith({
      headers: {
        Authorization: 'Bearer admin-token',
        'X-RPBox-Update-Channel': 'beta',
      },
    })
  })

  it('does not request beta updates for normal users even if the local flag is set', async () => {
    const userStore = useUserStore()
    userStore.setAuth('normal-token', user())

    const updater = useUpdater()
    updater.setParticipateTesting(true)
    await updater.checkForUpdate()

    expect(mockedCheck).toHaveBeenCalledWith(undefined)
  })

  it('treats a stable semver update as stable even when requested through beta headers', async () => {
    const userStore = useUserStore()
    userStore.setAuth('sponsor-token', user({ sponsor_level: 1 }))
    mockedCheck.mockResolvedValue({
      version: '0.2.39',
      body: '',
      date: '',
      rawJson: {},
    } as any)

    const updater = useUpdater()
    updater.setParticipateTesting(true)
    await updater.checkForUpdate()

    expect(updater.updateInfo.value?.channel).toBe('stable')
  })

  it('falls back to beta only for prerelease update versions when channel metadata is missing', async () => {
    const userStore = useUserStore()
    userStore.setAuth('sponsor-token', user({ sponsor_level: 1 }))
    mockedCheck.mockResolvedValue({
      version: '0.2.40-1',
      body: '',
      date: '',
      rawJson: {},
    } as any)

    const updater = useUpdater()
    updater.setParticipateTesting(true)
    await updater.checkForUpdate()

    expect(updater.updateInfo.value?.channel).toBe('beta')
  })
})
