import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { getAddonLatest, getTRP3Latest } from '@/api/addon'
import { listItems } from '@/api/item'
import { listEvents, listPosts } from '@/api/post'
import { listRPDBWorks } from '@/api/rpdb'

export type SidebarContentBadge = 'community' | 'events' | 'market' | 'rpdb'
export type SidebarMenuBadge = SidebarContentBadge | 'warcraft' | 'settings'

interface SidebarContentTotals {
  community: number
  events: number
  market: number
  rpdb: number
}

interface SidebarReadState {
  totals: Partial<SidebarContentTotals>
  addonSignature: string
}

interface InstalledAddonInfo {
  installed: boolean
  version: string | null
}

interface Trp3AddonInfo {
  id: string
  installed: boolean
  version: string | null
}

interface Trp3AddonCheckResult {
  addons: Trp3AddonInfo[]
}

const STORAGE_PREFIX = 'rpbox:sidebar-badges:v1'
const SYSTEM_UPDATE_READ_KEY = 'rpbox:sidebar-system-update-read:v1'
const EMPTY_TOTALS: SidebarContentTotals = {
  community: 0,
  events: 0,
  market: 0,
  rpdb: 0,
}

function normalizeVersion(version: string | null | undefined) {
  return (version || '').trim().replace(/^v/i, '')
}

function safeCount(value: unknown) {
  const count = Number(value)
  return Number.isFinite(count) && count > 0 ? Math.floor(count) : 0
}

function itemTotal(response: unknown) {
  const payload = response as { total?: unknown; data?: { total?: unknown } } | null
  return safeCount(payload?.data?.total ?? payload?.total)
}

function readStorage(key: string): SidebarReadState {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return { totals: {}, addonSignature: '' }
    const parsed = JSON.parse(raw) as Partial<SidebarReadState>
    return {
      totals: parsed.totals && typeof parsed.totals === 'object' ? parsed.totals : {},
      addonSignature: typeof parsed.addonSignature === 'string' ? parsed.addonSignature : '',
    }
  } catch {
    return { totals: {}, addonSignature: '' }
  }
}

function writeStorage(key: string, state: SidebarReadState) {
  localStorage.setItem(key, JSON.stringify(state))
}

export const useSidebarBadgesStore = defineStore('sidebarBadges', () => {
  const userId = ref(0)
  const currentTotals = ref<SidebarContentTotals>({ ...EMPTY_TOTALS })
  const unreadCounts = ref<SidebarContentTotals>({ ...EMPTY_TOTALS })
  const addonUpdateCount = ref(0)
  const addonUpdateSignature = ref('')
  const systemUpdateAvailable = ref(false)
  const systemUpdateVersion = ref('')
  const contentLoading = ref(false)
  const addonLoading = ref(false)
  let contentRefreshPromise: Promise<void> | null = null
  let addonRefreshPromise: Promise<void> | null = null

  const storageKey = computed(() => `${STORAGE_PREFIX}:${userId.value || 'anonymous'}`)

  function useUser(nextUserId?: number | null) {
    const normalized = Number(nextUserId) || 0
    if (normalized === userId.value) return
    userId.value = normalized
    currentTotals.value = { ...EMPTY_TOTALS }
    unreadCounts.value = { ...EMPTY_TOTALS }
    addonUpdateCount.value = 0
    addonUpdateSignature.value = ''
  }

  async function refreshContent() {
    if (contentRefreshPromise) return contentRefreshPromise

    contentRefreshPromise = (async () => {
      contentLoading.value = true
      const results = await Promise.allSettled([
        listPosts({ page: 1, page_size: 1, status: 'published', exclude_category: 'event' }),
        listEvents(),
        listItems({ page: 1, page_size: 1, sort: 'created_at', order: 'desc' }),
        listRPDBWorks({ page: 1, page_size: 1, sort: 'created_at' }),
      ])

      const nextTotals: SidebarContentTotals = { ...currentTotals.value }
      if (results[0].status === 'fulfilled') nextTotals.community = safeCount(results[0].value.total)
      if (results[1].status === 'fulfilled') nextTotals.events = safeCount(results[1].value.events?.length)
      if (results[2].status === 'fulfilled') nextTotals.market = itemTotal(results[2].value)
      if (results[3].status === 'fulfilled') nextTotals.rpdb = safeCount(results[3].value.total)

      const readState = readStorage(storageKey.value)
      let storageChanged = false
      const nextUnread: SidebarContentTotals = { ...unreadCounts.value }

      for (const key of Object.keys(nextTotals) as SidebarContentBadge[]) {
        const requestSucceeded = results[
          key === 'community' ? 0 : key === 'events' ? 1 : key === 'market' ? 2 : 3
        ].status === 'fulfilled'
        if (!requestSucceeded) continue

        const current = nextTotals[key]
        const baseline = readState.totals[key]
        if (typeof baseline !== 'number' || current < baseline) {
          readState.totals[key] = current
          nextUnread[key] = 0
          storageChanged = true
        } else {
          nextUnread[key] = Math.max(0, current - baseline)
        }
      }

      currentTotals.value = nextTotals
      unreadCounts.value = nextUnread
      if (storageChanged) writeStorage(storageKey.value, readState)
    })().finally(() => {
      contentLoading.value = false
      contentRefreshPromise = null
    })

    return contentRefreshPromise
  }

  async function refreshAddonUpdates() {
    if (addonRefreshPromise) return addonRefreshPromise

    addonRefreshPromise = (async () => {
      const wowPath = localStorage.getItem('wow_path')?.trim() || ''
      if (!wowPath) {
        addonUpdateCount.value = 0
        addonUpdateSignature.value = ''
        return
      }

      addonLoading.value = true
      const flavor = localStorage.getItem('selected_flavor') || '_retail_'
      const [trp3Local, trp3Latest, rpboxLocal, rpboxLatest] = await Promise.allSettled([
        invoke<Trp3AddonCheckResult>('check_trp3_addons', { wowPath }),
        getTRP3Latest(),
        invoke<InstalledAddonInfo>('check_addon_installed', { wowPath, flavor }),
        getAddonLatest(),
      ])

      const updates: string[] = []
      if (trp3Local.status === 'fulfilled' && trp3Latest.status === 'fulfilled') {
        const latestById = new Map(trp3Latest.value.addons.map(addon => [addon.id, addon]))
        for (const addon of trp3Local.value.addons || []) {
          const latest = latestById.get(addon.id)
          const localVersion = normalizeVersion(addon.version)
          const latestVersion = normalizeVersion(latest?.latestVersion)
          if (addon.installed && localVersion && latestVersion && localVersion !== latestVersion) {
            updates.push(`${addon.id}:${localVersion}>${latestVersion}`)
          }
        }
      }

      if (rpboxLocal.status === 'fulfilled' && rpboxLatest.status === 'fulfilled') {
        const localVersion = normalizeVersion(rpboxLocal.value.version)
        const latestVersion = normalizeVersion(rpboxLatest.value.version)
        if (rpboxLocal.value.installed && localVersion && latestVersion && localVersion !== latestVersion) {
          updates.push(`rpbox:${localVersion}>${latestVersion}`)
        }
      }

      const signature = updates.sort().join('|')
      addonUpdateSignature.value = signature
      if (!signature) {
        addonUpdateCount.value = 0
        return
      }

      const readState = readStorage(storageKey.value)
      addonUpdateCount.value = readState.addonSignature === signature ? 0 : updates.length
    })().finally(() => {
      addonLoading.value = false
      addonRefreshPromise = null
    })

    return addonRefreshPromise
  }

  async function initialize(nextUserId?: number | null) {
    useUser(nextUserId)
    await Promise.allSettled([refreshContent(), refreshAddonUpdates()])
  }

  function markContentRead(...keys: SidebarContentBadge[]) {
    const readState = readStorage(storageKey.value)
    for (const key of keys) {
      readState.totals[key] = currentTotals.value[key]
      unreadCounts.value[key] = 0
    }
    writeStorage(storageKey.value, readState)
  }

  function markMenuRead(menuId: string) {
    if (menuId === 'community') {
      markContentRead('community', 'events')
      return
    }
    if (menuId === 'market' || menuId === 'rpdb') {
      markContentRead(menuId)
      return
    }
    if (menuId === 'warcraft' && addonUpdateSignature.value) {
      const readState = readStorage(storageKey.value)
      readState.addonSignature = addonUpdateSignature.value
      addonUpdateCount.value = 0
      writeStorage(storageKey.value, readState)
      return
    }
    if (menuId === 'settings' && systemUpdateVersion.value) {
      localStorage.setItem(SYSTEM_UPDATE_READ_KEY, systemUpdateVersion.value)
      systemUpdateAvailable.value = false
    }
  }

  function syncSystemUpdate(version?: string | null) {
    const normalized = normalizeVersion(version)
    systemUpdateVersion.value = normalized
    systemUpdateAvailable.value = Boolean(
      normalized && localStorage.getItem(SYSTEM_UPDATE_READ_KEY) !== normalized,
    )
  }

  function reset() {
    userId.value = 0
    currentTotals.value = { ...EMPTY_TOTALS }
    unreadCounts.value = { ...EMPTY_TOTALS }
    addonUpdateCount.value = 0
    addonUpdateSignature.value = ''
  }

  return {
    unreadCounts,
    addonUpdateCount,
    systemUpdateAvailable,
    systemUpdateVersion,
    contentLoading,
    addonLoading,
    initialize,
    refreshContent,
    refreshAddonUpdates,
    syncSystemUpdate,
    markMenuRead,
    reset,
  }
})
