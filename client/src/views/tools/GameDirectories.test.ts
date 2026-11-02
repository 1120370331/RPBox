import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { dialog } from '@/composables/useDialog'
import { configuredGameFlavor, getGameInstallations, registerGameInstallation } from '@/utils/gameInstallations'
import GameDirectories from './GameDirectories.vue'

const mocks = vi.hoisted(() => ({ open: vi.fn(), invoke: vi.fn() }))
vi.mock('@tauri-apps/plugin-dialog', () => ({ open: mocks.open }))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
const retail = 'D:\\World of Warcraft\\_retail_'
const classic = 'D:\\World of Warcraft\\_classic_era_'
let wrapper: VueWrapper | undefined
beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
  Object.defineProperty(window, '__TAURI_INTERNALS__', { value: {}, configurable: true })
  i18n.global.locale.value = 'zh-CN'
  mocks.open.mockResolvedValue(retail)
  mocks.invoke.mockResolvedValue([{ path: retail, flavor: '_retail_', accounts: ['WOW1'] }])
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; dialog.close(false); vi.restoreAllMocks(); Reflect.deleteProperty(window, '__TAURI_INTERNALS__') })
async function openPage() {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/tools/directories', component: GameDirectories },
    { path: '/tools/rpbox/install', component: { template: '<div />' } },
  ] })
  await router.push('/tools/directories')
  await router.isReady()
  wrapper = mount(GameDirectories, { global: { plugins: [createPinia(), router, i18n], stubs: { teleport: true } } })
  await flushPromises()
  return wrapper
}
describe('directory registration and confirmation', () => {
  it('detects directories without a folder picker and waits for confirmation before saving', async () => {
    let finishDetection!: (directories: { path: string; flavor: string; accounts: string[] }[]) => void
    mocks.invoke.mockImplementationOnce(() => new Promise(resolve => { finishDetection = resolve }))
    const view = await openPage()
    await view.get('.directory-detect').trigger('click')
    expect(mocks.invoke).toHaveBeenCalledWith('detect_wow_paths')
    expect(mocks.open).not.toHaveBeenCalled()
    expect(view.get('.directory-detect').text()).toBe('正在检测…')
    expect(view.get('.directory-detect').attributes('disabled')).toBeDefined()
    expect(view.get('.directory-primary').attributes('disabled')).toBeDefined()
    finishDetection([{ path: retail, flavor: '_retail_', accounts: ['WOW1'] }])
    await flushPromises()
    expect(getGameInstallations()).toEqual([])
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_retail_')
    await view.get('select[aria-label="游戏版本"]').setValue('_classic_era_')
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: retail, flavor: '_classic_era_' }])
    expect(localStorage.getItem('selected_flavor')).toBe('_classic_era_')
  })
  it('requires selecting a detected client and preserves saved editions and the active directory', async () => {
    registerGameInstallation({ path: retail, flavor: '_classic_anniversary_' })
    mocks.invoke.mockResolvedValue([{ path: retail, flavor: '_retail_', accounts: [] }, { path: classic, flavor: '_classic_era_', accounts: [] }])
    const view = await openPage()
    await view.get('.directory-detect').trigger('click'); await flushPromises()
    expect(view.get('.r-dialog__btn--confirm').attributes('disabled')).toBeDefined()
    await view.get('select[aria-label="确认游戏目录"]').setValue(retail)
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_classic_anniversary_')
    await view.get('select[aria-label="确认游戏目录"]').setValue(classic)
    await view.get('.r-dialog__btn--cancel').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: retail, flavor: '_classic_anniversary_' }])
    await view.get('.directory-detect').trigger('click'); await flushPromises()
    await view.get('select[aria-label="确认游戏目录"]').setValue(classic)
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: retail, flavor: '_classic_anniversary_' }, { path: classic, flavor: '_classic_era_' }])
    expect(localStorage.getItem('wow_path')).toBe(retail)
    expect(localStorage.getItem('selected_flavor')).toBe('_classic_anniversary_')
  })
  it.each(['empty', 'failed'])('allows manual registration after %s detection', async result => {
    if (result === 'empty') mocks.invoke.mockResolvedValueOnce([])
    else mocks.invoke.mockRejectedValueOnce(new Error('扫描不可用'))
    const view = await openPage()
    await view.get('.directory-detect').trigger('click'); await flushPromises()
    expect(view.get('[role="alert"]').text()).toContain(result === 'empty' ? '未自动检测到游戏目录' : '自动检测失败：扫描不可用')
    expect(view.find('[role="dialog"]').exists()).toBe(false)
    expect(view.get('.directory-detect').attributes('disabled')).toBeUndefined()
    expect(getGameInstallations()).toEqual([])
    await view.get('.directory-primary').trigger('click'); await flushPromises()
    expect(view.find('[role="alert"]').exists()).toBe(false)
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: retail, flavor: '_retail_' }])
  })
  it('keeps native directory actions unavailable outside the desktop client', async () => {
    Reflect.deleteProperty(window, '__TAURI_INTERNALS__')
    const view = await openPage()
    expect(view.get('.directory-detect').attributes('disabled')).toBeDefined()
    expect(view.get('.directory-primary').attributes('disabled')).toBeDefined()
    await view.get('.directory-detect').trigger('click')
    expect(mocks.invoke).not.toHaveBeenCalled()
    expect(mocks.open).not.toHaveBeenCalled()
  })
  it('asks for a directory before inspecting and saves only the user-confirmed edition', async () => {
    let selectFolder!: (path: string) => void
    mocks.open.mockImplementationOnce(() => new Promise<string>(resolve => { selectFolder = resolve }))
    const view = await openPage()
    await view.get('.directory-primary').trigger('click')
    expect(mocks.invoke).not.toHaveBeenCalled()
    selectFolder(retail)
    await flushPromises()
    expect(mocks.invoke).toHaveBeenCalledWith('inspect_game_directories', { path: retail })
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_retail_')
    expect(getGameInstallations()).toEqual([])
    await view.get('select[aria-label="游戏版本"]').setValue('_classic_era_')
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: retail, flavor: '_classic_era_' }])
    expect(configuredGameFlavor(retail)).toBe('_classic_era_')
  })
  it('cancels without creating a directory entry', async () => {
    const view = await openPage()
    await view.get('.directory-primary').trigger('click'); await flushPromises()
    await view.get('.r-dialog__btn--cancel').trigger('click')
    expect(getGameInstallations()).toEqual([])
  })
  it('requires an explicit child directory when a selected root contains several clients', async () => {
    mocks.invoke.mockResolvedValue([{ path: retail, flavor: '_retail_', accounts: [] }, { path: classic, flavor: '_classic_era_', accounts: [] }])
    const view = await openPage()
    await view.get('.directory-primary').trigger('click'); await flushPromises()
    expect(view.get('.r-dialog__btn--confirm').attributes('disabled')).toBeDefined()
    await view.get('select[aria-label="确认游戏目录"]').setValue(classic)
    expect(view.get<HTMLSelectElement>('select[aria-label="游戏版本"]').element.value).toBe('_classic_era_')
    expect(getGameInstallations()).toEqual([])
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: classic, flavor: '_classic_era_' }])
  })
  it('allows manually confirming an unknown edition and editing it later', async () => {
    mocks.invoke.mockResolvedValue([{ path: classic, flavor: null, accounts: [] }])
    const view = await openPage()
    await view.get('.directory-primary').trigger('click'); await flushPromises()
    expect(view.get('.r-dialog__btn--confirm').attributes('disabled')).toBeDefined()
    await view.get('select[aria-label="游戏版本"]').setValue('_classic_era_')
    await view.get('.r-dialog__btn--confirm').trigger('click')
    await view.findAll('.directory-entry-actions button').find(button => button.text() === '修改版本')!.trigger('click')
    await view.get('select[aria-label="游戏版本"]').setValue('_classic_anniversary_')
    await view.get('.r-dialog__btn--confirm').trigger('click')
    expect(getGameInstallations()).toEqual([{ path: classic, flavor: '_classic_anniversary_' }])
    expect(localStorage.getItem('selected_flavor')).toBe('_classic_anniversary_')
  })
  it('confirms removal without invoking any native file deletion', async () => {
    registerGameInstallation({ path: retail, flavor: '_retail_' })
    const confirm = vi.spyOn(dialog, 'confirm').mockResolvedValueOnce(false).mockResolvedValueOnce(true)
    const view = await openPage()
    await view.get('.directory-remove').trigger('click'); await flushPromises()
    expect(getGameInstallations()).toHaveLength(1)
    await view.get('.directory-remove').trigger('click'); await flushPromises()
    expect(confirm).toHaveBeenCalledTimes(2)
    expect(getGameInstallations()).toEqual([])
    expect(mocks.invoke).not.toHaveBeenCalled()
  })
})
