import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ADDONS_UPDATED_EVENT, addonAutoUpdateEnabled, autoUpdateAddons, isAddonBusy, runAddonOperation, setAddonAutoUpdateEnabled, type AddonUpdateCandidate } from './addonAutoUpdate'
import { registerGameInstallation } from '@/utils/gameInstallations'

const mocks = vi.hoisted(() => ({ invoke: vi.fn(), success: vi.fn(), error: vi.fn() }))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
vi.mock('@/api/addon', () => ({ getAddonDownloadUrl: (version: string) => `https://rpbox.test/addon/${version}` }))
vi.mock('@/stores/toast', () => ({ useToastStore: () => ({ success: mocks.success, error: mocks.error }) }))
vi.mock('@/i18n', () => ({ default: { global: { t: (key: string) => key } } }))

const target = { wowPath: 'C:\\Games\\WoW', flavor: '_retail_' }
const rpbox: AddonUpdateCandidate = { id: 'rpbox', installed: true, currentVersion: '1.0.0', latestVersion: '1.1.0' }
const trp3Ids = ['total-rp-3', 'total-rp-3-extended', 'musician']

function installations() {
  return mocks.invoke.mock.calls.filter(([command]) => String(command).startsWith('install_'))
}

beforeEach(() => {
  localStorage.clear()
  registerGameInstallation({ path: target.wowPath, flavor: target.flavor })
  setAddonAutoUpdateEnabled(true)
  vi.resetAllMocks()
  mocks.invoke.mockImplementation(async (command: string) => {
    if (command === 'check_addon_installed') return { installed: true, version: '1.0.0' }
    if (command === 'check_trp3_addons') return { addons: trp3Ids.map(id => ({ id, installed: true, version: '1.0.0' })) }
    return undefined
  })
})

afterEach(() => {
  setAddonAutoUpdateEnabled(false)
  vi.restoreAllMocks()
})

describe('automatic addon updates', () => {
  it('rejects installs for unregistered directories and mismatched confirmed editions', async () => {
    const operation = vi.fn()
    await expect(runAddonOperation({ wowPath: 'D:/Unregistered', flavor: '_retail_' }, 'rpbox', 'install', operation)).rejects.toThrow('登记目录')
    registerGameInstallation({ path: target.wowPath, flavor: '_classic_era_' })
    await expect(runAddonOperation(target, 'rpbox', 'install', operation)).rejects.toThrow('确认游戏版本')
    expect(operation).not.toHaveBeenCalled()
  })

  it('keeps physical-directory operations exclusive across root/WTF aliases and edition edits', async () => {
    let finish!: () => void
    const pending = runAddonOperation(target, 'rpbox', 'install', () => new Promise<void>(resolve => { finish = resolve }))
    await Promise.resolve()
    const alias = { wowPath: target.wowPath + '/WTF', flavor: '_classic_era_' }
    expect(isAddonBusy(alias, 'rpbox')).toBe(true)
    await expect(runAddonOperation(alias, 'rpbox', 'uninstall', vi.fn())).rejects.toThrow()
    finish()
    await pending
    expect(isAddonBusy(alias, 'rpbox')).toBe(false)
  })
  it('persists the switch and performs no automatic work when disabled', async () => {
    setAddonAutoUpdateEnabled(false)
    expect(addonAutoUpdateEnabled.value).toBe(false)
    expect(localStorage.getItem('addon_auto_update')).toBe('false')
    expect(await autoUpdateAddons(target, [rpbox])).toEqual([])
    expect(mocks.invoke).not.toHaveBeenCalled()
  })

  it('updates all four installed managed addons using the existing native installers', async () => {
    const listener = vi.fn()
    window.addEventListener(ADDONS_UPDATED_EVENT, listener)
    try {
      const candidates = [rpbox, ...trp3Ids.map(id => ({ ...rpbox, id, downloadUrl: `https://download.test/${id}.zip` }))]
      expect(await autoUpdateAddons(target, candidates)).toEqual(['rpbox', ...trp3Ids])
      expect(installations()).toHaveLength(4)
      expect(mocks.invoke).toHaveBeenCalledWith('install_addon_from_url', { ...target, pluginId: 'rpbox', downloadUrl: 'https://rpbox.test/addon/1.1.0' })
      for (const id of trp3Ids) {
        expect(mocks.invoke).toHaveBeenCalledWith('install_trp3_addon_with_progress', { wowPath: target.wowPath, addonId: id, downloadUrl: `https://download.test/${id}.zip` })
      }
      expect(listener).toHaveBeenCalledTimes(4)
      expect(localStorage.getItem('addon_last_checked_version')).toBe('1.1.0')
    } finally {
      window.removeEventListener(ADDONS_UPDATED_EVENT, listener)
    }
  })

  it('does not install missing addons, downgrade versions, or act on unknown metadata', async () => {
    const candidates = [
      { ...rpbox, installed: false },
      { ...rpbox, latestVersion: '0.9.0' },
      { ...rpbox, latestVersion: '1.0.0' },
      { ...rpbox, currentVersion: '未知' },
      { ...rpbox, latestVersion: 'unknown' },
      { ...rpbox, id: 'unmanaged' },
      { ...rpbox, id: 'total-rp-3' },
    ]
    expect(await autoUpdateAddons(target, candidates)).toEqual([])
    expect(mocks.invoke).not.toHaveBeenCalled()
  })

  it('shares simultaneous detections and manual installs for the same target', async () => {
    let complete!: () => void
    mocks.invoke.mockImplementation(async (command: string) => {
      if (command === 'check_addon_installed') return { installed: true, version: '1.0.0' }
      return new Promise<void>(resolve => { complete = resolve })
    })
    const first = autoUpdateAddons(target, [rpbox])
    const second = autoUpdateAddons(target, [rpbox])
    await vi.waitFor(() => expect(installations()).toHaveLength(1))
    expect(isAddonBusy(target, 'rpbox')).toBe(true)
    const manual = vi.fn()
    const shared = runAddonOperation(target, 'rpbox', 'install', manual)
    complete()
    await Promise.all([first, second, shared])
    expect(manual).not.toHaveBeenCalled()
    expect(mocks.success).toHaveBeenCalledTimes(1)
    expect(isAddonBusy(target, 'rpbox')).toBe(false)
  })

  it('rechecks the switch and selected directory after an asynchronous local check', async () => {
    for (const change of ['disable', 'directory']) {
      setAddonAutoUpdateEnabled(true)
      localStorage.setItem('wow_path', target.wowPath)
      let finish!: (value: { installed: boolean; version: string }) => void
      mocks.invoke.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
      const task = autoUpdateAddons(target, [rpbox])
      await vi.waitFor(() => expect(finish).toBeTypeOf('function'))
      if (change === 'disable') setAddonAutoUpdateEnabled(false)
      else localStorage.setItem('wow_path', 'D:\\Other WoW')
      finish({ installed: true, version: '1.0.0' })
      expect(await task).toEqual([])
    }
    expect(installations()).toHaveLength(0)
  })

  it('avoids reinstalling an addon already updated or removed since detection', async () => {
    mocks.invoke.mockResolvedValueOnce({ installed: true, version: '1.2.0' })
    expect(await autoUpdateAddons(target, [rpbox])).toEqual([])
    mocks.invoke.mockResolvedValueOnce({ installed: false, version: null })
    expect(await autoUpdateAddons(target, [rpbox])).toEqual([])
    expect(installations()).toHaveLength(0)
  })

  it('releases a failed installation for retry and does not record false success', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    mocks.invoke.mockImplementationOnce(async () => ({ installed: true, version: '1.0.0' }))
      .mockRejectedValueOnce(new Error('download failed'))
    expect(await autoUpdateAddons(target, [rpbox])).toEqual([])
    expect(mocks.error).toHaveBeenCalledOnce()
    expect(mocks.success).not.toHaveBeenCalled()
    expect(localStorage.getItem('addon_last_checked_version')).toBeNull()
    expect(isAddonBusy(target, 'rpbox')).toBe(false)
    expect(await autoUpdateAddons(target, [rpbox])).toEqual(['rpbox'])
  })

  it('excludes updates while uninstalling the same addon', async () => {
    let finish!: () => void
    const uninstall = runAddonOperation(target, 'rpbox', 'uninstall', () => new Promise<void>(resolve => { finish = resolve }))
    await vi.waitFor(() => expect(finish).toBeTypeOf('function'))
    expect(await autoUpdateAddons(target, [rpbox])).toEqual([])
    await expect(runAddonOperation(target, 'rpbox', 'install', vi.fn())).rejects.toThrow()
    finish()
    await uninstall
    expect(installations()).toHaveLength(0)
  })
})
