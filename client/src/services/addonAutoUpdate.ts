import { ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { getAddonDownloadUrl } from '@/api/addon'
import { isAddonVersionNewer } from '@/utils/addonVersion'
import { useToastStore } from '@/stores/toast'
import i18n from '@/i18n'

const STORAGE_KEY = 'addon_auto_update'
export const ADDONS_UPDATED_EVENT = 'rpbox:addons-updated'
export const addonAutoUpdateEnabled = ref(localStorage.getItem(STORAGE_KEY) === 'true')
const busyKeys = ref(new Set<string>())
const operations = new Map<string, { action: string; promise: Promise<unknown> }>()

export interface AddonTarget {
  wowPath: string
  flavor: string
}

export interface AddonUpdateCandidate {
  id: string
  installed: boolean
  currentVersion?: string | null
  latestVersion?: string | null
  downloadUrl?: string
}

const addonNames: Record<string, string> = {
  rpbox: 'RPBox Addon',
  'total-rp-3': 'Total RP 3',
  'total-rp-3-extended': 'Total RP 3: Extended',
  musician: 'Musician',
}

function knownVersion(version?: string | null) {
  return /^v?\d+(?:\.\d+)*(?:-[\dA-Za-z.-]+)?(?:\+[\dA-Za-z.-]+)?$/i.test(version?.trim() || '')
}

function canAutoUpdate(target: AddonTarget) {
  const currentTarget = {
    wowPath: localStorage.getItem('wow_path') || '',
    flavor: localStorage.getItem('selected_flavor') || '_retail_',
  }
  return localStorage.getItem(STORAGE_KEY) === 'true' && !!target.wowPath && targetKey(target) === targetKey(currentTarget)
}

function targetKey(target: AddonTarget) {
  return `${target.wowPath.trim().replace(/\\/g, '/').replace(/\/+$/, '').toLowerCase()}|${target.flavor}`
}

function operationKey(target: AddonTarget, id: string) {
  return `${targetKey(target)}|${id}`
}

export function setAddonAutoUpdateEnabled(enabled: boolean) {
  localStorage.setItem(STORAGE_KEY, String(enabled))
  addonAutoUpdateEnabled.value = enabled
}

export function isAddonBusy(target: AddonTarget, id: string) {
  return busyKeys.value.has(operationKey(target, id))
}

/** Shares concurrent installs and excludes conflicting installs/uninstalls for the same target. */
export function runAddonOperation<T>(target: AddonTarget, id: string, action: 'install' | 'uninstall', operation: () => Promise<T>): Promise<T> {
  const key = operationKey(target, id)
  const existing = operations.get(key)
  if (existing) {
    if (existing.action === action) return existing.promise as Promise<T>
    return Promise.reject(new Error(i18n.global.t('settings.addon.busy')))
  }

  busyKeys.value.add(key)
  const promise = Promise.resolve().then(operation).then(result => {
    window.dispatchEvent(new CustomEvent(ADDONS_UPDATED_EVENT, { detail: { ...target, id } }))
    return result
  }).finally(() => {
    operations.delete(key)
    busyKeys.value.delete(key)
  })
  operations.set(key, { action, promise })
  return promise
}

/** Updates only installed, versioned add-ons for the currently selected game directory. */
export async function autoUpdateAddons(target: AddonTarget, candidates: AddonUpdateCandidate[]): Promise<string[]> {
  const updated: string[] = []
  for (const addon of candidates) {
    if (!canAutoUpdate(target)) break
    if (!addon.installed || !knownVersion(addon.currentVersion) || !knownVersion(addon.latestVersion)
      || !isAddonVersionNewer(addon.latestVersion, addon.currentVersion)) continue
    if (!Object.prototype.hasOwnProperty.call(addonNames, addon.id)) continue
    if (addon.id !== 'rpbox' && !addon.downloadUrl) continue
    const existing = operations.get(operationKey(target, addon.id))
    if (existing?.action === 'uninstall') continue

    try {
      const result = await runAddonOperation(target, addon.id, 'install', async () => {
        // A second detection or an uninstall may have changed the local state meanwhile.
        const local = addon.id === 'rpbox'
          ? await invoke<{ installed: boolean; version: string | null }>('check_addon_installed', { ...target })
          : (await invoke<{ addons: { id: string; installed: boolean; version: string | null }[] }>('check_trp3_addons', { wowPath: target.wowPath })).addons.find(item => item.id === addon.id)
        if (!canAutoUpdate(target) || !local?.installed || !knownVersion(local.version)) return false
        if (!isAddonVersionNewer(addon.latestVersion, local.version)) return false
        if (addon.id === 'rpbox') {
          await invoke('install_addon_from_url', { ...target, downloadUrl: getAddonDownloadUrl(addon.latestVersion!), pluginId: 'rpbox' })
        } else {
          await invoke('install_trp3_addon_with_progress', { wowPath: target.wowPath, addonId: addon.id, downloadUrl: addon.downloadUrl })
        }
        return true
      })
      if (result === false) continue
      updated.push(addon.id)
      if (addon.id === 'rpbox') {
        localStorage.setItem('addon_last_checked_version', addon.latestVersion!)
        localStorage.removeItem('addon_update_prompt_key')
      }
      if (!existing) useToastStore().success(i18n.global.t('settings.addon.autoUpdateSuccess', { name: addonNames[addon.id], version: addon.latestVersion }))
    } catch (error) {
      console.error('[AddonAutoUpdate]', addon.id, error)
      if (!existing) useToastStore().error(i18n.global.t('settings.addon.autoUpdateFailed', { name: addonNames[addon.id] }))
    }
  }
  return updated
}
