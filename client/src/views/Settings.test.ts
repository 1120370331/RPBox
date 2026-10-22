import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import { ref } from 'vue'
import Settings from './Settings.vue'
import RDialog from '@/components/RDialog.vue'
import { dialog } from '@/composables/useDialog'
import zhSettings from '@/i18n/locales/zh-CN/settings'
import enSettings from '@/i18n/locales/en-US/settings'
import { setAddonAutoUpdateEnabled } from '@/services/addonAutoUpdate'

const mocks = vi.hoisted(() => ({
  deleteAccount: vi.fn(),
  logout: vi.fn(),
  removeAccountHistoryItem: vi.fn(),
  replace: vi.fn(),
  error: vi.fn(),
  success: vi.fn(),
}))

vi.mock('@/api/user', () => ({ deleteAccount: mocks.deleteAccount, redeemSponsorCode: vi.fn(), uploadAvatar: vi.fn() }))
vi.mock('@/api/addon', () => ({ getAddonManifest: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ replace: mocks.replace, push: vi.fn(), back: vi.fn() }) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ user: { id: 42, username: 'test' }, logout: mocks.logout, removeAccountHistoryItem: mocks.removeAccountHistoryItem }) }))
vi.mock('@/stores/toast', () => ({ useToastStore: () => ({ error: mocks.error, success: mocks.success, warning: vi.fn() }) }))
vi.mock('@/stores/theme', () => ({ themes: [], useThemeStore: () => ({ currentThemeId: 'classic' }) }))
vi.mock('@/stores/locale', () => ({ useLocaleStore: () => ({ currentLocale: 'zh-CN', setLocale: vi.fn() }) }))
vi.mock('@/utils/userNameStyle', () => ({ buildNameStyle: () => ({}) }))
vi.mock('@tauri-apps/api/app', () => ({ getVersion: async () => '1.0.0' }))
vi.mock('@tauri-apps/plugin-autostart', () => ({ enable: vi.fn(), disable: vi.fn(), isEnabled: async () => false }))
vi.mock('@/components/AddonUpdateDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/ImageCropperDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/composables/useUpdater', () => ({
  useUpdater: () => Object.fromEntries([
    ...['checking', 'updateAvailable', 'latestReleaseLoading', 'downloading', 'participateTesting', 'betaUpdatesAvailable', 'betaUpdatesEnabled'].map(key => [key, ref(false)]),
    ...['updateInfo', 'latestRelease', 'lastError'].map(key => [key, ref(null)]),
    ['downloadProgress', ref(0)],
    ['fetchLatestRelease', vi.fn()], ['checkForUpdate', vi.fn()], ['downloadAndInstall', vi.fn()], ['setParticipateTesting', vi.fn()],
  ]),
}))

let wrapper: VueWrapper
let dialogWrapper: VueWrapper

beforeEach(async () => {
  vi.clearAllMocks()
  localStorage.clear()
  setAddonAutoUpdateEnabled(false)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    missingWarn: false,
    fallbackWarn: false,
    messages: { 'zh-CN': { settings: zhSettings }, 'en-US': { settings: enSettings } },
  })
  wrapper = mount(Settings, { attachTo: document.body, global: { plugins: [i18n] } })
  dialogWrapper = mount(RDialog, { attachTo: document.body, global: { plugins: [i18n] } })
  await flushPromises()
})

afterEach(() => {
  dialog.close(false)
  wrapper.unmount()
  dialogWrapper.unmount()
  document.body.innerHTML = ''
})

async function openDeletion() {
  await wrapper.get('#settings-tab-account').trigger('click')
  await wrapper.get('.danger-zone button').trigger('click')
}

async function submitDeletion(password = ' secret123 ') {
  const input = document.querySelector<HTMLInputElement>('#delete-account-password')!
  input.value = password
  input.dispatchEvent(new Event('input', { bubbles: true }))
  document.querySelector('#delete-account-form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flushPromises()
}

describe('settings account deletion', () => {
  it('persists the automatic addon update switch in general settings', async () => {
    const toggle = wrapper.get('.addon-auto-update')
    expect(toggle.attributes('aria-checked')).toBe('false')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-checked')).toBe('true')
    expect(localStorage.getItem('addon_auto_update')).toBe('true')
    await toggle.trigger('click')
    expect(localStorage.getItem('addon_auto_update')).toBe('false')
  })
  it('separates general and account settings and supports keyboard switching', async () => {
    expect(wrapper.get('#settings-panel-general').isVisible()).toBe(true)
    expect(wrapper.get('#settings-panel-account').isVisible()).toBe(false)
    expect(wrapper.get('#settings-panel-general').text()).not.toContain('注销账号')
    await wrapper.get('#settings-tab-general').trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get('#settings-tab-account').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('#settings-panel-account').isVisible()).toBe(true)
    expect(wrapper.get('#settings-panel-account').text()).toContain('兑换码使用')
    expect(wrapper.get('#settings-panel-general').text()).not.toContain('兑换码使用')
  })

  it('requires a password and a separate destructive confirmation', async () => {
    await openDeletion()
    await submitDeletion('')
    expect(dialog.state.value.visible).toBe(false)
    expect(mocks.deleteAccount).not.toHaveBeenCalled()
    await submitDeletion()
    expect(dialog.state.value.options.message).toContain('所有帖子将被删除')
    expect(mocks.deleteAccount).not.toHaveBeenCalled()
    dialog.close(false)
    await flushPromises()
    expect(mocks.deleteAccount).not.toHaveBeenCalled()
    expect(mocks.logout).not.toHaveBeenCalled()
  })

  it('clears only the deleted account session and logs out after server success', async () => {
    let finish!: (value: { message: string }) => void
    mocks.deleteAccount.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    await openDeletion()
    await submitDeletion()
    dialog.close(true)
    await flushPromises()
    expect(mocks.deleteAccount).toHaveBeenCalledWith(' secret123 ')
    expect(mocks.logout).not.toHaveBeenCalled()
    await submitDeletion()
    expect(mocks.deleteAccount).toHaveBeenCalledTimes(1)
    finish({ message: '账号已删除' })
    await flushPromises()
    expect(mocks.removeAccountHistoryItem).toHaveBeenCalledWith(42)
    expect(mocks.logout).toHaveBeenCalledOnce()
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'login' })
    expect(document.querySelector('#delete-account-password')).toBeNull()
  })

  it('preserves the account and allows retry after an API failure', async () => {
    mocks.deleteAccount.mockRejectedValueOnce(new Error('密码错误'))
    await openDeletion()
    await submitDeletion('wrong-password')
    dialog.close(true)
    await flushPromises()
    expect(mocks.error).toHaveBeenCalledWith('密码错误')
    expect(mocks.logout).not.toHaveBeenCalled()
    expect(mocks.removeAccountHistoryItem).not.toHaveBeenCalled()
    expect(document.querySelector<HTMLInputElement>('#delete-account-password')!.value).toBe('')
    mocks.deleteAccount.mockResolvedValueOnce({ message: '账号已删除' })
    await submitDeletion('correct-password')
    dialog.close(true)
    await flushPromises()
    expect(mocks.logout).toHaveBeenCalledOnce()
  })
})
