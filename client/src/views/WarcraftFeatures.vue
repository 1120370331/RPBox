<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'
import { open as openExternal } from '@tauri-apps/plugin-shell'
import {
  getAddonDownloadUrl,
  getAddonLatest,
  getTRP3Latest,
  type TRP3LatestResponse,
} from '@/api/addon'
import { dialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { isAddonVersionNewer } from '@/utils/addonVersion'

interface WowInstallation {
  path: string
  version: string | Record<string, null>
  accounts: string[]
}

interface Trp3AddonInfo {
  id: Trp3PluginId
  name: string
  installed: boolean
  latestVersion: string
  requiresUpdate: boolean
  version: string | null
  path: string | null
  curseforgeUrl: string
  sourceUrl: string
  downloadUrl: string
}

interface Trp3AddonCheckResult {
  wowPath: string
  addonsDir: string
  latestCheckAvailable: boolean
  latestCheckNote: string
  addons: Trp3AddonInfo[]
}

interface InstalledAddonInfo {
  installed: boolean
  version: string | null
  path: string | null
}

type Trp3PluginId = 'total-rp-3' | 'total-rp-3-extended'
type PluginCardId = Trp3PluginId | 'rpbox'
type PluginState = 'locked' | 'checking' | 'idle' | 'missing' | 'update' | 'ready'

interface PluginCard {
  id: PluginCardId
  title: string
  desc: string
  icon: string
  state: PluginState
  status: string
  currentVersion: string
  latestVersion: string
  sourceUrl?: string
  action: string
  canInstall: boolean
  canUninstall: boolean
  busy: boolean
  progressLabel: string
  progressDetail: string
  progressPercent: number | null
}

interface PluginProgress {
  label: string
  detail: string
  percent: number | null
  downloadedBytes: number
  totalBytes: number | null
}

interface NativeAddonInstallProgress {
  pluginId: string
  label: string
  detail: string
  percent: number | null
  downloadedBytes: number
  totalBytes: number | null
}

const router = useRouter()
const toast = useToastStore()
let progressUnlisten: UnlistenFn | null = null

const mounted = ref(false)
const wowPath = ref(localStorage.getItem('wow_path') || '')
const detectedPaths = ref<WowInstallation[]>([])
const trp3Status = ref<Trp3AddonCheckResult | null>(null)
const trp3Latest = ref<TRP3LatestResponse | null>(null)
const rpboxInstalledInfo = ref<InstalledAddonInfo | null>(null)
const rpboxLatestVersion = ref('')
const selectedFlavor = ref(localStorage.getItem('selected_flavor') || '_retail_')
const checkingPaths = ref(false)
const checkingPlugins = ref(false)
const installingPluginIds = ref<PluginCardId[]>([])
const pluginProgresses = ref<Partial<Record<PluginCardId, PluginProgress>>>({})
const pathError = ref('')
const pluginError = ref('')
const installNotice = ref('')

const featureEntries = [
  {
    title: '人物卡备份',
    desc: '备份、同步和恢复 Total RP 3 人物卡与账号数据。',
    icon: 'ri-user-star-line',
    route: '/sync',
  },
  {
    title: '剧情故事',
    desc: '读取 RPBox 插件记录，将聊天日志整理成剧情故事。',
    icon: 'ri-book-open-line',
    route: '/archives',
  },
]

const hasWowPath = computed(() => !!wowPath.value)

const trp3AddonsById = computed(() => {
  const entries = (trp3Status.value?.addons || []).map(addon => [addon.id, addon] as const)
  return new Map<Trp3PluginId, Trp3AddonInfo>(entries)
})

const rpboxNeedsUpdate = computed(() => {
  if (!rpboxInstalledInfo.value?.installed || !rpboxLatestVersion.value) return false
  return isAddonVersionNewer(rpboxLatestVersion.value, rpboxInstalledInfo.value.version)
})

const pluginCards = computed<PluginCard[]>(() => [
  buildTrp3Card(
    'total-rp-3',
    'Total RP 3',
    '人物卡，自定义昵称，RP插件本体',
    'ri-id-card-line',
  ),
  buildTrp3Card(
    'total-rp-3-extended',
    'Total RP 3: Extended',
    '道具、剧本、背包等拓展功能',
    'ri-tools-line',
  ),
  buildRpboxCard(),
])

const pluginReadyCount = computed(() => pluginCards.value.filter(card => card.state === 'ready').length)
const activeInstallCount = computed(() => installingPluginIds.value.length)

const pluginStatusSummary = computed(() => {
  if (!hasWowPath.value) return '等待目录'
  if (checkingPlugins.value) return '检测中'
  if (activeInstallCount.value) return `${activeInstallCount.value} 个处理中`
  if (pluginReadyCount.value === pluginCards.value.length) return '全部就绪'
  return `${pluginReadyCount.value}/${pluginCards.value.length} 就绪`
})

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await bindProgressEvents()
  if (wowPath.value) {
    await refreshPluginStatus()
  } else {
    await detectPaths()
  }
})

onBeforeUnmount(() => {
  if (progressUnlisten) {
    progressUnlisten()
    progressUnlisten = null
  }
})

async function bindProgressEvents() {
  try {
    progressUnlisten = await listen<NativeAddonInstallProgress>('addon-install-progress', event => {
      const progress = event.payload
      if (!isKnownPluginId(progress.pluginId)) return
      setPluginProgress(progress.pluginId, {
        label: progress.label,
        detail: progress.detail,
        percent: progress.percent,
        downloadedBytes: progress.downloadedBytes,
        totalBytes: progress.totalBytes,
      })
    })
  } catch {
    progressUnlisten = null
  }
}

function isKnownPluginId(pluginId: string): pluginId is PluginCardId {
  return pluginId === 'total-rp-3' || pluginId === 'total-rp-3-extended' || pluginId === 'rpbox'
}

function getNativeErrorMessage(e: any, fallback: string) {
  const message = typeof e === 'string' ? e : e?.message
  if (
    message?.includes('invoke')
    || message?.includes('__TAURI')
    || message?.includes('not allowed')
  ) {
    return `${fallback}：请在 RPBox 桌面客户端中运行，本地浏览器无法访问魔兽世界目录。`
  }
  return message || fallback
}

function mergeTrp3Latest(
  localStatus: Trp3AddonCheckResult,
  latest: TRP3LatestResponse | null,
): Trp3AddonCheckResult {
  const latestById = new Map((latest?.addons || []).map(addon => [addon.id, addon]))

  return {
    ...localStatus,
    latestCheckAvailable: !!latest,
    latestCheckNote: latest?.note || localStatus.latestCheckNote,
    addons: localStatus.addons.map(addon => {
      const latestAddon = latestById.get(addon.id)
      const latestVersion = latestAddon?.latestVersion || addon.latestVersion
      return {
        ...addon,
        name: latestAddon?.name || addon.name,
        latestVersion,
        curseforgeUrl: latestAddon?.curseforgeUrl || addon.curseforgeUrl,
        sourceUrl: latestAddon?.sourceUrl || addon.sourceUrl,
        downloadUrl: latestAddon?.downloadUrl || addon.downloadUrl,
        requiresUpdate: !addon.installed || isAddonVersionNewer(latestVersion, addon.version),
      }
    }),
  }
}

function getStateText(state: PluginState) {
  switch (state) {
    case 'locked':
      return '等待目录'
    case 'checking':
      return '检测中'
    case 'missing':
      return '未安装'
    case 'update':
      return '可更新'
    case 'ready':
      return '最新版'
    default:
      return '待检测'
  }
}

function getActionText(state: PluginState, installed: boolean, busy: boolean) {
  if (busy) return '处理中'
  if (state === 'checking') return '检测中'
  if (!installed) return '安装'
  if (state === 'update') return '更新'
  if (state === 'ready') return '已最新'
  return '安装'
}

function getActionIcon(card: PluginCard) {
  if (card.busy) return 'ri-loader-4-line spinning'
  if (card.state === 'ready') return 'ri-checkbox-circle-line'
  if (card.state === 'update') return 'ri-arrow-up-circle-line'
  return 'ri-download-cloud-2-line'
}

function isPluginInstalling(id: PluginCardId) {
  return installingPluginIds.value.includes(id)
}

function startPluginInstall(id: PluginCardId) {
  if (isPluginInstalling(id)) return
  installingPluginIds.value = [...installingPluginIds.value, id]
}

function finishPluginInstall(id: PluginCardId) {
  installingPluginIds.value = installingPluginIds.value.filter(pluginId => pluginId !== id)
}

function getPluginProgress(id: PluginCardId) {
  return pluginProgresses.value[id] || null
}

function setPluginProgress(id: PluginCardId, patch: Partial<PluginProgress>) {
  const current = pluginProgresses.value[id] || {
    label: '准备中',
    detail: '',
    percent: null,
    downloadedBytes: 0,
    totalBytes: null,
  }
  pluginProgresses.value = {
    ...pluginProgresses.value,
    [id]: {
      ...current,
      ...patch,
    },
  }
}

function clearPluginProgress(id: PluginCardId) {
  const next = { ...pluginProgresses.value }
  delete next[id]
  pluginProgresses.value = next
}

function clearPluginProgressSoon(id: PluginCardId) {
  window.setTimeout(() => clearPluginProgress(id), 1800)
}

function progressWidth(card: PluginCard) {
  if (card.progressPercent === null) return '42%'
  return `${Math.max(4, Math.min(100, card.progressPercent))}%`
}

function buildTrp3Card(
  id: Trp3PluginId,
  title: string,
  desc: string,
  icon: string,
): PluginCard {
  const addon = trp3AddonsById.value.get(id)
  const busy = isPluginInstalling(id)
  const progress = getPluginProgress(id)
  const state = getTrp3State(addon)
  const installed = !!addon?.installed

  return {
    id,
    title,
    desc,
    icon,
    state,
    status: getStateText(state),
    currentVersion: installed ? addon?.version || '未知' : '未安装',
    latestVersion: addon?.latestVersion || (hasWowPath.value ? '待获取' : '选择目录后检测'),
    sourceUrl: addon?.sourceUrl,
    action: progress?.label || getActionText(state, installed, busy),
    canInstall: hasWowPath.value && !checkingPlugins.value && !busy && (state === 'missing' || state === 'update') && !!addon?.downloadUrl,
    canUninstall: hasWowPath.value && !checkingPlugins.value && !busy && installed,
    busy,
    progressLabel: progress?.label || '',
    progressDetail: progress?.detail || '',
    progressPercent: progress?.percent ?? null,
  }
}

function buildRpboxCard(): PluginCard {
  const installed = !!rpboxInstalledInfo.value?.installed
  const busy = isPluginInstalling('rpbox')
  const progress = getPluginProgress('rpbox')
  const state = getRpboxState()

  return {
    id: 'rpbox',
    title: 'RPBox Addon',
    desc: 'RPBox自研增强插件，帮您保存您的RP记录',
    icon: 'ri-archive-drawer-line',
    state,
    status: getStateText(state),
    currentVersion: installed ? rpboxInstalledInfo.value?.version || '未知' : '未安装',
    latestVersion: rpboxLatestVersion.value || (hasWowPath.value ? '待获取' : '选择目录后检测'),
    action: progress?.label || getActionText(state, installed, busy),
    canInstall: hasWowPath.value && !checkingPlugins.value && !busy && (state === 'missing' || state === 'update') && !!rpboxLatestVersion.value,
    canUninstall: hasWowPath.value && !checkingPlugins.value && !busy && installed,
    busy,
    progressLabel: progress?.label || '',
    progressDetail: progress?.detail || '',
    progressPercent: progress?.percent ?? null,
  }
}

function getTrp3State(addon: Trp3AddonInfo | undefined): PluginState {
  if (!hasWowPath.value) return 'locked'
  if (checkingPlugins.value && !addon) return 'checking'
  if (!addon) return 'idle'
  if (!addon.installed) return 'missing'
  if (addon.requiresUpdate) return 'update'
  return 'ready'
}

function getRpboxState(): PluginState {
  if (!hasWowPath.value) return 'locked'
  if (checkingPlugins.value && !rpboxInstalledInfo.value) return 'checking'
  if (!rpboxInstalledInfo.value) return 'idle'
  if (!rpboxInstalledInfo.value.installed) return 'missing'
  if (rpboxNeedsUpdate.value) return 'update'
  return 'ready'
}

async function detectPaths() {
  checkingPaths.value = true
  pathError.value = ''
  try {
    detectedPaths.value = await invoke<WowInstallation[]>('detect_wow_paths')
  } catch (e: any) {
    pathError.value = getNativeErrorMessage(e, '扫描魔兽世界目录失败')
    detectedPaths.value = []
  } finally {
    checkingPaths.value = false
  }
}

async function selectWowPath(path: string) {
  wowPath.value = path
  localStorage.setItem('wow_path', path)
  detectedPaths.value = []
  pathError.value = ''
  await refreshPluginStatus()
}

function clearWowPath() {
  localStorage.removeItem('wow_path')
  wowPath.value = ''
  trp3Status.value = null
  trp3Latest.value = null
  rpboxInstalledInfo.value = null
  rpboxLatestVersion.value = ''
  installNotice.value = ''
  pluginError.value = ''
  void detectPaths()
}

function goToSetup() {
  router.push({ path: '/sync/setup', query: { redirect: '/warcraft' } })
}

function goToFeature(route: string) {
  if (!hasWowPath.value) return
  router.push(route)
}

async function refreshPluginStatus() {
  if (!wowPath.value) return

  checkingPlugins.value = true
  pluginError.value = ''
  installNotice.value = ''

  const results = await Promise.allSettled([
    checkTrp3Addons(),
    checkRpboxAddon(),
  ])

  const errors = results
    .filter((result): result is PromiseRejectedResult => result.status === 'rejected')
    .map(result => result.reason?.message || String(result.reason))

  if (errors.length) {
    pluginError.value = errors.join('；')
  }
  checkingPlugins.value = false
}

async function checkTrp3Addons() {
  const localStatus = await invoke<Trp3AddonCheckResult>('check_trp3_addons', {
    wowPath: wowPath.value,
  })

  try {
    const latest = await getTRP3Latest()
    trp3Latest.value = latest
    trp3Status.value = mergeTrp3Latest(localStatus, latest)
  } catch (e: any) {
    trp3Latest.value = null
    trp3Status.value = mergeTrp3Latest(localStatus, null)
    throw new Error(e?.message || '获取 TRP/TRPEx 最新版本失败')
  }
}

async function checkRpboxAddon() {
  rpboxInstalledInfo.value = await invoke<InstalledAddonInfo>('check_addon_installed', {
    wowPath: wowPath.value,
    flavor: selectedFlavor.value,
  })

  try {
    const latest = await getAddonLatest()
    rpboxLatestVersion.value = latest.version
  } catch (e: any) {
    rpboxLatestVersion.value = ''
    throw new Error(e?.message || '获取 RPBox 插件最新版本失败')
  }
}

async function installPlugin(card: PluginCard) {
  if (!card.canInstall || !wowPath.value) return
  if (card.id === 'rpbox') {
    await installRpboxAddon()
    return
  }

  const addon = trp3AddonsById.value.get(card.id)
  if (!addon) return
  await installTrp3Addon(addon)
}

async function uninstallPlugin(card: PluginCard) {
  if (!card.canUninstall || !wowPath.value) return

  const confirmed = await dialog.confirm({
    title: `卸载 ${card.title}`,
    message: `确定要卸载 ${card.title} 吗？\n\n此操作只会删除 Interface/AddOns 里的插件本体，不会删除人物卡、道具、剧本或 RP 记录数据。这些数据保存在 WTF 文件夹中，不会丢失。`,
    type: 'warning',
    confirmText: '卸载',
    cancelText: '取消',
  })
  if (!confirmed) return

  startPluginInstall(card.id)
  pluginError.value = ''
  installNotice.value = ''
  let succeeded = false
  try {
    setPluginProgress(card.id, {
      label: '卸载中',
      detail: '正在删除 AddOns 目录中的插件本体',
      percent: null,
      downloadedBytes: 0,
      totalBytes: null,
    })

    if (card.id === 'rpbox') {
      await invoke('uninstall_addon', {
        wowPath: wowPath.value,
        flavor: selectedFlavor.value,
      })
      await checkRpboxAddon()
    } else {
      const localStatus = await invoke<Trp3AddonCheckResult>('uninstall_trp3_addon', {
        wowPath: wowPath.value,
        addonId: card.id,
      })
      trp3Status.value = mergeTrp3Latest(localStatus, trp3Latest.value)
    }

    setPluginProgress(card.id, {
      label: '已卸载',
      detail: '插件本体已移除，WTF 数据已保留',
      percent: 100,
    })
    installNotice.value = `${card.title} 已卸载；人物卡、道具数据仍保留在 WTF 文件夹中。`
    toast.success(`${card.title} 已卸载`)
    succeeded = true
  } catch (e: any) {
    pluginError.value = getNativeErrorMessage(e, `${card.title} 卸载失败`)
    setPluginProgress(card.id, {
      label: '卸载失败',
      detail: pluginError.value,
      percent: null,
    })
    toast.error('插件卸载失败')
  } finally {
    finishPluginInstall(card.id)
    if (succeeded) clearPluginProgressSoon(card.id)
  }
}

async function installTrp3Addon(addon: Trp3AddonInfo) {
  if (!wowPath.value || isPluginInstalling(addon.id) || !addon.downloadUrl) return

  const operation = addon.installed ? '更新' : '安装'
  startPluginInstall(addon.id)
  pluginError.value = ''
  installNotice.value = ''
  let succeeded = false
  try {
    setPluginProgress(addon.id, {
      label: '连接中',
      detail: '正在请求下载源',
      percent: null,
      downloadedBytes: 0,
      totalBytes: null,
    })

    const localStatus = await invoke<Trp3AddonCheckResult>('install_trp3_addon_with_progress', {
      wowPath: wowPath.value,
      addonId: addon.id,
      downloadUrl: addon.downloadUrl,
    })
    setPluginProgress(addon.id, {
      label: '检测中',
      detail: '正在确认本地插件版本',
      percent: 100,
    })
    trp3Status.value = mergeTrp3Latest(localStatus, trp3Latest.value)
    setPluginProgress(addon.id, {
      label: '已完成',
      detail: `${operation}完成，检测已更新`,
      percent: 100,
    })
    installNotice.value = `${addon.name} 已${operation}到 AddOns 目录`
    toast.success(installNotice.value)
    succeeded = true
  } catch (e: any) {
    pluginError.value = getNativeErrorMessage(e, `${addon.name} 自动${operation}失败`)
    setPluginProgress(addon.id, {
      label: '失败',
      detail: pluginError.value,
      percent: null,
    })
    toast.error('插件安装失败')
  } finally {
    finishPluginInstall(addon.id)
    if (succeeded) clearPluginProgressSoon(addon.id)
  }
}

async function installRpboxAddon() {
  if (!wowPath.value || isPluginInstalling('rpbox')) return

  const operation = rpboxInstalledInfo.value?.installed ? '更新' : '安装'
  startPluginInstall('rpbox')
  pluginError.value = ''
  installNotice.value = ''
  let succeeded = false
  try {
    let latestVersion = rpboxLatestVersion.value
    if (!latestVersion) {
      const latest = await getAddonLatest()
      latestVersion = latest.version
      rpboxLatestVersion.value = latest.version
    }

    setPluginProgress('rpbox', {
      label: '连接中',
      detail: '正在请求下载源',
      percent: null,
      downloadedBytes: 0,
      totalBytes: null,
    })

    await invoke('install_addon_from_url', {
      wowPath: wowPath.value,
      flavor: selectedFlavor.value,
      downloadUrl: getAddonDownloadUrl(latestVersion),
      pluginId: 'rpbox',
    })

    setPluginProgress('rpbox', {
      label: '检测中',
      detail: '正在确认本地插件版本',
      percent: 100,
    })
    await checkRpboxAddon()
    setPluginProgress('rpbox', {
      label: '已完成',
      detail: `${operation}完成，检测已更新`,
      percent: 100,
    })
    installNotice.value = `RPBox Addon 已${operation}到 AddOns 目录`
    toast.success(installNotice.value)
    succeeded = true
  } catch (e: any) {
    pluginError.value = getNativeErrorMessage(e, `RPBox Addon 自动${operation}失败`)
    setPluginProgress('rpbox', {
      label: '失败',
      detail: pluginError.value,
      percent: null,
    })
    toast.error('插件安装失败')
  } finally {
    finishPluginInstall('rpbox')
    if (succeeded) clearPluginProgressSoon('rpbox')
  }
}

async function openSourcePage(card: PluginCard) {
  if (!card.sourceUrl) return
  try {
    await openExternal(card.sourceUrl)
  } catch (e: any) {
    toast.error(e?.message || '打开页面失败')
  }
}

async function openAddonFolder() {
  if (!wowPath.value) return
  try {
    await invoke<string>('open_addons_folder', {
      wowPath: wowPath.value,
    })
  } catch (e: any) {
    toast.error(getNativeErrorMessage(e, '打开 AddOns 目录失败'))
  }
}
</script>

<template>
  <div class="warcraft-page" :class="{ 'animate-in': mounted }">
    <section class="title-row anim-item" style="--delay: 0">
      <div class="title-copy">
        <span class="eyebrow">Warcraft</span>
        <h1>魔兽功能</h1>
        <p>先定位游戏目录，再安装插件，最后进入人物卡备份和剧情故事。</p>
      </div>
      <div class="title-status" :class="{ ready: hasWowPath }">
        <span>当前状态</span>
        <strong>{{ pluginStatusSummary }}</strong>
      </div>
    </section>

    <section class="directory-row anim-item" style="--delay: 1">
      <div class="row-heading">
        <div class="heading-copy">
          <h2>选择游戏目录</h2>
          <p>需要定位到包含 WTF/Account 的魔兽世界版本目录。</p>
        </div>
        <div class="directory-actions">
          <button v-if="hasWowPath" type="button" class="icon-btn" title="打开 AddOns 目录" @click="openAddonFolder">
            <i class="ri-folder-open-line"></i>
          </button>
          <button
            v-if="hasWowPath"
            type="button"
            class="ghost-btn"
            :disabled="checkingPlugins"
            @click="refreshPluginStatus"
          >
            <i :class="checkingPlugins ? 'ri-loader-4-line spinning' : 'ri-refresh-line'"></i>
            {{ checkingPlugins ? '检测中' : '重新检测' }}
          </button>
          <button v-if="hasWowPath" type="button" class="text-btn danger" @click="clearWowPath">清除</button>
          <button type="button" class="primary-btn" @click="goToSetup">
            <i class="ri-folder-settings-line"></i>
            {{ hasWowPath ? '更换目录' : '设置目录' }}
          </button>
        </div>
      </div>

      <div class="directory-current" :class="{ empty: !hasWowPath }">
        <span class="directory-icon">
          <i :class="hasWowPath ? 'ri-hard-drive-3-line' : 'ri-lock-line'"></i>
        </span>
        <span class="directory-copy">
          <small>当前游戏目录</small>
          <strong>{{ trp3Status?.wowPath || wowPath || '未选择，下面功能已锁定' }}</strong>
        </span>
      </div>

      <div v-if="!hasWowPath" class="path-tools">
        <button type="button" class="ghost-btn" :disabled="checkingPaths" @click="detectPaths">
          <i :class="checkingPaths ? 'ri-loader-4-line spinning' : 'ri-search-line'"></i>
          {{ checkingPaths ? '扫描中' : '自动扫描' }}
        </button>
        <span v-if="pathError" class="inline-error">{{ pathError }}</span>
        <span v-else class="path-hint">选择目录后，插件安装和快捷入口会自动解锁。</span>
      </div>

      <div v-if="!hasWowPath && detectedPaths.length" class="detected-grid">
        <button
          v-for="install in detectedPaths"
          :key="install.path"
          type="button"
          class="detected-path"
          @click="selectWowPath(install.path)"
        >
          <span>
            <strong>{{ install.path }}</strong>
            <small>{{ install.accounts?.length || 0 }} 个账号</small>
          </span>
          <i class="ri-arrow-right-line"></i>
        </button>
      </div>
    </section>

    <section class="plugin-row lockable anim-item" :class="{ locked: !hasWowPath }" style="--delay: 2">
      <div class="row-heading">
        <div class="heading-copy">
          <h2>插件安装</h2>
          <p>安装RP必备插件</p>
        </div>
      </div>

      <div class="locked-content">
        <div v-if="installNotice" class="notice-box">
          <i class="ri-checkbox-circle-fill"></i>
          <span>{{ installNotice }}</span>
        </div>
        <div v-if="pluginError" class="error-box">
          <i class="ri-error-warning-line"></i>
          <span>{{ pluginError }}</span>
        </div>

        <div class="plugin-grid">
          <article v-for="card in pluginCards" :key="card.id" class="plugin-card" :class="card.state">
            <div class="plugin-card__top">
              <span class="plugin-icon"><i :class="card.icon"></i></span>
              <span class="state-pill" :class="card.state">{{ card.status }}</span>
            </div>

            <div class="plugin-card__copy">
              <h3>{{ card.title }}</h3>
              <p>{{ card.desc }}</p>
            </div>

            <div class="version-lines">
              <span><small>本地</small><strong>{{ card.currentVersion }}</strong></span>
              <span><small>最新</small><strong>{{ card.latestVersion }}</strong></span>
            </div>

            <div v-if="card.progressLabel" class="plugin-progress" :class="{ active: card.busy, unknown: card.progressPercent === null }">
              <div class="plugin-progress__text">
                <span>{{ card.progressLabel }}</span>
                <strong v-if="card.progressPercent !== null">{{ card.progressPercent }}%</strong>
              </div>
              <div class="plugin-progress__track">
                <span :style="{ width: progressWidth(card) }"></span>
              </div>
              <small>{{ card.progressDetail }}</small>
            </div>

            <div class="plugin-actions">
              <button
                type="button"
                class="primary-btn compact"
                :disabled="!card.canInstall"
                @click="installPlugin(card)"
              >
                <i :class="getActionIcon(card)"></i>
                {{ card.action }}
              </button>
              <button
                v-if="card.canUninstall"
                type="button"
                class="danger-btn compact"
                :disabled="card.busy"
                @click="uninstallPlugin(card)"
              >
                <i class="ri-delete-bin-line"></i>
                卸载
              </button>
              <button
                v-if="card.sourceUrl"
                type="button"
                class="icon-btn compact-icon"
                title="打开来源页面"
                :disabled="!hasWowPath"
                @click="openSourcePage(card)"
              >
                <i class="ri-github-line"></i>
              </button>
            </div>
          </article>
        </div>
      </div>

      <div v-if="!hasWowPath" class="lock-banner">
        <i class="ri-lock-line"></i>
        <span>先选择游戏目录，插件安装才会开放。</span>
      </div>
    </section>

    <section class="feature-row lockable anim-item" :class="{ locked: !hasWowPath }" style="--delay: 3">
      <div class="row-heading">
        <div class="heading-copy">
          <h2>功能快捷入口</h2>
        </div>
      </div>

      <div class="locked-content feature-grid">
        <button
          v-for="entry in featureEntries"
          :key="entry.route"
          type="button"
          class="feature-card"
          :disabled="!hasWowPath"
          @click="goToFeature(entry.route)"
        >
          <span class="feature-icon"><i :class="entry.icon"></i></span>
          <span class="feature-copy">
            <strong>{{ entry.title }}</strong>
            <small>{{ entry.desc }}</small>
          </span>
          <i class="ri-arrow-right-line"></i>
        </button>
      </div>

      <div v-if="!hasWowPath" class="lock-banner feature-lock">
        <i class="ri-lock-line"></i>
        <span>目录未设置，快捷入口暂不可用。</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.warcraft-page {
  min-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--color-text-main, #2C1810);
}

h1,
h2,
h3,
p {
  margin: 0;
}

button {
  font-family: inherit;
}

.title-row,
.directory-row,
.plugin-row,
.feature-row {
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
  box-shadow: var(--shadow-sm, 0 2px 10px rgba(75, 54, 33, 0.05));
}

.title-row {
  padding: 20px 22px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.title-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.eyebrow {
  color: #155E91;
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0;
  text-transform: uppercase;
}

h1 {
  font-size: 28px;
  line-height: 1.15;
}

h2 {
  font-size: 18px;
  line-height: 1.3;
}

h3 {
  font-size: 16px;
  line-height: 1.3;
}

.title-copy p,
.row-heading p,
.path-hint,
.directory-copy small,
.plugin-card__copy p,
.version-lines small,
.detected-path small,
.feature-copy small {
  color: var(--color-text-secondary, #856a52);
  font-size: 12px;
  line-height: 1.5;
}

.title-status {
  min-width: 132px;
  padding: 11px 14px;
  border-radius: 8px;
  background: rgba(21, 94, 145, 0.07);
  border: 1px solid rgba(21, 94, 145, 0.15);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-status.ready {
  background: rgba(46, 125, 50, 0.08);
  border-color: rgba(46, 125, 50, 0.18);
}

.title-status span {
  color: var(--color-text-secondary, #856a52);
  font-size: 12px;
}

.title-status strong {
  font-size: 15px;
}

.directory-row,
.plugin-row,
.feature-row {
  padding: 18px;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.row-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.heading-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.directory-actions,
.plugin-actions,
.path-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.directory-actions {
  flex: 0 0 auto;
  justify-content: flex-end;
  flex-wrap: nowrap;
}

.directory-current {
  min-width: 0;
  padding: 13px 14px;
  border-radius: 8px;
  background: var(--color-bg-secondary, #F8EFE7);
  border: 1px solid var(--color-border-light, #EBDCCB);
  display: flex;
  align-items: center;
  gap: 12px;
}

.directory-current.empty {
  background: rgba(21, 94, 145, 0.07);
  border-color: rgba(21, 94, 145, 0.18);
}

.directory-icon,
.plugin-icon,
.feature-icon {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  background: rgba(21, 94, 145, 0.08);
  color: #155E91;
  font-size: 21px;
}

.directory-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.directory-copy strong,
.detected-path strong {
  overflow-wrap: anywhere;
}

.directory-copy strong {
  font-size: 13px;
}

.primary-btn,
.ghost-btn,
.icon-btn,
.text-btn,
.danger-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border-radius: 8px;
  border: 1px solid transparent;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.18s ease, border-color 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

.primary-btn {
  min-height: 38px;
  padding: 9px 14px;
  background: var(--btn-primary-bg, #804030);
  color: var(--btn-primary-text, #fff);
  border-color: var(--btn-primary-bg, #804030);
}

.primary-btn:hover:not(:disabled) {
  background: var(--btn-primary-hover, #6D3428);
  transform: translateY(-1px);
}

.ghost-btn {
  min-height: 38px;
  padding: 9px 14px;
  color: var(--btn-secondary-text, #4B3621);
  background: rgba(128, 64, 48, 0.07);
  border-color: var(--btn-outline-border, #D7C0AA);
}

.ghost-btn:hover:not(:disabled) {
  background: rgba(128, 64, 48, 0.12);
}

.danger-btn {
  min-height: 38px;
  padding: 9px 14px;
  color: #9B1C31;
  background: rgba(155, 28, 49, 0.07);
  border-color: rgba(155, 28, 49, 0.2);
}

.danger-btn:hover:not(:disabled) {
  background: rgba(155, 28, 49, 0.12);
  transform: translateY(-1px);
}

.icon-btn {
  width: 38px;
  height: 38px;
  color: var(--color-primary, #4B3621);
  background: var(--color-panel-bg, #fff);
  border-color: var(--btn-outline-border, #D7C0AA);
  font-size: 18px;
}

.icon-btn:hover:not(:disabled) {
  color: #155E91;
  border-color: rgba(21, 94, 145, 0.32);
  background: rgba(21, 94, 145, 0.07);
}

.text-btn {
  padding: 0;
  background: transparent;
  color: var(--color-accent, #B87333);
}

.text-btn.danger {
  color: #9B1C31;
}

.directory-actions .text-btn.danger {
  min-height: 38px;
  padding: 9px 12px;
  border-color: rgba(155, 28, 49, 0.18);
  background: rgba(155, 28, 49, 0.06);
}

.directory-actions .text-btn.danger:hover:not(:disabled) {
  background: rgba(155, 28, 49, 0.1);
  transform: translateY(-1px);
}

.compact {
  min-height: 34px;
  padding: 7px 11px;
  font-size: 12px;
}

.compact-icon {
  width: 34px;
  height: 34px;
}

.primary-btn:disabled,
.ghost-btn:disabled,
.icon-btn:disabled,
.danger-btn:disabled,
.feature-card:disabled {
  cursor: not-allowed;
  opacity: 0.58;
  transform: none;
}

.path-tools {
  justify-content: flex-start;
}

.inline-error,
.error-box {
  color: #9B1C31;
}

.inline-error {
  font-size: 12px;
  line-height: 1.5;
}

.detected-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
}

.detected-path {
  min-width: 0;
  min-height: 66px;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-panel-bg, #fff);
  color: inherit;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.detected-path:hover {
  border-color: rgba(21, 94, 145, 0.34);
  background: rgba(21, 94, 145, 0.045);
  transform: translateY(-1px);
}

.detected-path span,
.feature-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.state-pill {
  min-height: 24px;
  padding: 4px 8px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  color: var(--color-text-main, #2C1810);
  background: var(--color-bg-secondary, #F8EFE7);
  font-size: 12px;
  font-weight: 800;
  white-space: nowrap;
}

.plugin-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.plugin-card {
  min-width: 0;
  min-height: 278px;
  padding: 14px;
  border-radius: 8px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-panel-bg, #fff);
  display: flex;
  flex-direction: column;
  gap: 13px;
}

.plugin-card.ready {
  border-color: rgba(46, 125, 50, 0.26);
}

.plugin-card.update {
  border-color: rgba(21, 94, 145, 0.28);
}

.plugin-card.missing {
  border-color: rgba(220, 20, 60, 0.22);
}

.plugin-card.locked {
  border-style: dashed;
}

.plugin-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.plugin-card__copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.plugin-card__copy h3,
.feature-copy strong {
  overflow-wrap: anywhere;
}

.state-pill.ready {
  color: #2E7D32;
  background: rgba(46, 125, 50, 0.1);
}

.state-pill.update,
.state-pill.checking {
  color: #155E91;
  background: rgba(21, 94, 145, 0.09);
}

.state-pill.missing {
  color: #9B1C31;
  background: rgba(220, 20, 60, 0.08);
}

.state-pill.locked,
.state-pill.idle {
  color: var(--color-text-secondary, #856a52);
  background: rgba(133, 106, 82, 0.1);
}

.version-lines {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.version-lines span {
  min-width: 0;
  padding: 9px 10px;
  border-radius: 8px;
  background: var(--color-bg-secondary, #F8EFE7);
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.version-lines strong {
  font-size: 13px;
  overflow-wrap: anywhere;
}

.plugin-progress {
  padding: 10px;
  border-radius: 8px;
  background: rgba(21, 94, 145, 0.07);
  border: 1px solid rgba(21, 94, 145, 0.14);
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.plugin-progress__text {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #155E91;
  font-size: 12px;
  font-weight: 800;
}

.plugin-progress__text strong {
  font-size: 12px;
}

.plugin-progress__track {
  height: 7px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(21, 94, 145, 0.12);
}

.plugin-progress__track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #155E91;
  transition: width 0.18s ease;
}

.plugin-progress.active.unknown .plugin-progress__track span {
  width: 42%;
  animation: indeterminate 1.1s ease-in-out infinite;
}

.plugin-progress small {
  color: var(--color-text-secondary, #856a52);
  font-size: 12px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.plugin-actions {
  margin-top: auto;
}

.notice-box,
.error-box {
  padding: 12px 13px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.5;
}

.notice-box {
  color: #2E7D32;
  background: rgba(46, 125, 50, 0.09);
  border: 1px solid rgba(46, 125, 50, 0.18);
}

.error-box {
  background: rgba(220, 20, 60, 0.08);
  border: 1px solid rgba(220, 20, 60, 0.16);
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.feature-card {
  min-width: 0;
  min-height: 88px;
  padding: 14px;
  border-radius: 8px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-panel-bg, #fff);
  color: inherit;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.feature-card:hover:not(:disabled) {
  border-color: rgba(21, 94, 145, 0.34);
  background: rgba(21, 94, 145, 0.045);
  transform: translateY(-1px);
}

.lockable.locked .locked-content {
  pointer-events: none;
  opacity: 0.42;
  filter: grayscale(0.2);
}

.lock-banner {
  position: absolute;
  top: 18px;
  right: 18px;
  max-width: min(360px, calc(100% - 36px));
  padding: 9px 11px;
  border-radius: 8px;
  border: 1px solid rgba(21, 94, 145, 0.18);
  background: rgba(255, 255, 255, 0.94);
  color: #155E91;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  font-weight: 800;
  line-height: 1.4;
  box-shadow: var(--shadow-sm, 0 2px 10px rgba(75, 54, 33, 0.05));
}

.feature-lock {
  top: 16px;
}

.spinning {
  animation: spin 1s linear infinite;
}

.anim-item {
  opacity: 0;
  transform: translateY(12px);
}

.animate-in .anim-item {
  animation: fadeUp 0.42s ease forwards;
  animation-delay: calc(var(--delay) * 0.06s);
}

@keyframes fadeUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes indeterminate {
  0% {
    transform: translateX(-110%);
  }
  50% {
    transform: translateX(25%);
  }
  100% {
    transform: translateX(230%);
  }
}

@media (max-width: 1120px) {
  .plugin-grid {
    grid-template-columns: 1fr;
  }

  .plugin-card {
    min-height: auto;
  }
}

@media (max-width: 760px) {
  .title-row,
  .row-heading {
    flex-direction: column;
    align-items: stretch;
  }

  .directory-actions,
  .path-tools,
  .plugin-actions {
    align-items: stretch;
  }

  .directory-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .directory-actions .primary-btn,
  .directory-actions .ghost-btn,
  .path-tools .ghost-btn,
  .plugin-actions .primary-btn,
  .plugin-actions .danger-btn {
    flex: 1;
  }

  .detected-grid,
  .feature-grid,
  .version-lines {
    grid-template-columns: 1fr;
  }

  .feature-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .feature-card > .ri-arrow-right-line {
    grid-column: 1 / -1;
    justify-self: start;
  }

  .lock-banner {
    position: static;
    max-width: none;
  }
}
</style>
