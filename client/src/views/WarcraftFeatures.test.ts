import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getGameInstallations, registerGameInstallation } from '@/utils/gameInstallations'
import WarcraftFeatures from './WarcraftFeatures.vue'

const mocks = vi.hoisted(() => ({ invoke: vi.fn(), getAddonLatest: vi.fn(), getTRP3Latest: vi.fn() }))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
vi.mock('@tauri-apps/api/event', () => ({ listen: async () => () => {} }))
vi.mock('@tauri-apps/plugin-shell', () => ({ open: vi.fn() }))
vi.mock('@/api/addon', () => ({ getAddonLatest: mocks.getAddonLatest, getTRP3Latest: mocks.getTRP3Latest, getAddonDownloadUrl: () => 'https://example.test/addon.zip' }))
vi.mock('@/services/addonAutoUpdate', () => ({ ADDONS_UPDATED_EVENT: 'addon-test-event', autoUpdateAddons: async () => [], isAddonBusy: () => false, runAddonOperation: vi.fn() }))
const retail = { path: 'D:/WoW/_retail_', flavor: '_retail_' }
const classic = { path: 'D:/WoW/_classic_era_', flavor: '_classic_era_' }
let wrapper: VueWrapper | undefined
beforeEach(() => {
  localStorage.clear(); vi.clearAllMocks()
  Object.defineProperty(window, '__TAURI_INTERNALS__', { value: {}, configurable: true })
  mocks.invoke.mockImplementation(async (command: string) => command === 'check_trp3_addons' ? { addons: [] } : { installed: false, version: null, path: null })
  mocks.getAddonLatest.mockResolvedValue({ version: '1.0.15-beta.1' })
  mocks.getTRP3Latest.mockResolvedValue({ addons: [] })
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; Reflect.deleteProperty(window, '__TAURI_INTERNALS__') })
async function openPage() {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/tools/rpbox/install', component: WarcraftFeatures }, { path: '/tools/directories', component: { template: '<div />' } },
  ] })
  await router.push('/tools/rpbox/install'); await router.isReady()
  wrapper = mount(WarcraftFeatures, { global: { plugins: [createPinia(), router] } })
  await flushPromises()
  return { view: wrapper, router }
}
describe('installer registered-directory selection', () => {
  it('does not scan or register directories in the installer and navigates to the manager', async () => {
    const { view, router } = await openPage()
    expect(mocks.invoke).not.toHaveBeenCalled()
    expect(view.find('select[aria-label="安装游戏目录"]').exists()).toBe(false)
    expect(getGameInstallations()).toEqual([])
    await view.findAll('button').find(button => button.text() === '管理游戏目录')!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/tools/directories')
  })
  it('derives the version from the registered selection, including manual overrides', async () => {
    registerGameInstallation({ ...retail, flavor: '_classic_anniversary_' })
    registerGameInstallation(classic)
    const { view } = await openPage()
    expect(view.get('.installation-version strong').text()).toBe('周年怀旧服')
    expect(mocks.invoke).toHaveBeenCalledWith('check_addon_installed', { wowPath: retail.path, flavor: '_classic_anniversary_' })
    expect(mocks.getAddonLatest).not.toHaveBeenCalled()
    expect(view.find('select[aria-label="游戏版本"]').exists()).toBe(false)
    expect(view.get('select[aria-label="安装游戏目录"]').findAll('option')).toHaveLength(2)
    await view.get('select[aria-label="安装游戏目录"]').setValue(classic.path)
    await flushPromises()
    expect(view.get('.installation-version strong').text()).toBe('经典怀旧服')
    expect(mocks.invoke).toHaveBeenCalledWith('check_addon_installed', { wowPath: classic.path, flavor: classic.flavor })
    expect(mocks.getAddonLatest).not.toHaveBeenCalled()
    expect(getGameInstallations()).toHaveLength(2)
  })
  it('offers the dedicated beta addon for a registered Forever client', async () => {
    registerGameInstallation({ path: 'D:/WoW/_forever_', flavor: '_forever_' })
    const { view } = await openPage()
    expect(view.get('.installation-version strong').text()).toBe('魔兽：无限')
    expect(mocks.getAddonLatest).toHaveBeenCalledWith('_forever_')
    const rpboxCard = view.findAll('.plugin-card').find(card => card.text().includes('RPBox Addon'))
    expect(rpboxCard?.find('button.primary-btn').attributes('disabled')).toBeUndefined()
    expect(rpboxCard?.text()).toContain('1.0.15-beta.1')
  })
  it('offers the Classic TRP3 package while disabling unsupported Extended', async () => {
    registerGameInstallation(classic)
    mocks.invoke.mockImplementation(async (command: string) => command === 'check_trp3_addons'
      ? { addons: [
        { id: 'total-rp-3', installed: false, latestVersion: '', downloadUrl: '' },
        { id: 'total-rp-3-extended', installed: false, latestVersion: '', downloadUrl: '' },
      ] }
      : { installed: false, version: null, path: null })
    mocks.getTRP3Latest.mockResolvedValue({ addons: [
      { id: 'total-rp-3', latestVersion: '3.4.1', downloadUrl: 'https://rpbox.test/classic-trp3' },
      { id: 'total-rp-3-extended', latestVersion: '', downloadUrl: '', compatibilityNote: '怀旧服不支持' },
    ] })
    const { view } = await openPage()
    expect(mocks.getTRP3Latest).toHaveBeenCalledWith('_classic_era_')
    const trp3 = view.findAll('.plugin-card').find(card => card.text().includes('Total RP 3'))!
    const extended = view.findAll('.plugin-card').find(card => card.text().includes('Total RP 3: Extended'))!
    expect(trp3.text()).toContain('3.4.1')
    expect(trp3.find('button.primary-btn').attributes('disabled')).toBeUndefined()
    expect(extended.text()).toContain('怀旧服不支持')
    expect(extended.find('button.primary-btn').attributes('disabled')).toBeDefined()
  })
})
