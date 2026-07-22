<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { invoke } from '@tauri-apps/api/core'
import RTabs from '@/components/RTabs.vue'
import RTabPane from '@/components/RTabPane.vue'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'
import AddonInstaller from '@/components/AddonInstaller.vue'
import AddonUpdateDialog from '@/components/AddonUpdateDialog.vue'
import StagingPool from './StagingPool.vue'
import StoryList from './StoryList.vue'
import { createStory, addStoryEntries, getStory, listStories, type CreateStoryEntryRequest, type Story, type StoryFilterParams } from '@/api/story'
import { listTags, addStoryTag, type Tag } from '@/api/tag'
import { getAddonManifest } from '@/api/addon'
import { getGuild, type Guild } from '@/api/guild'
import { useToast } from '@/composables/useToast'
import type { ChatRecord, IdentityEndpoint, ProfileSnapshot } from '@/types/chatLog'

interface InstalledAddonInfo {
  installed: boolean
  version?: string | null
}

const mounted = ref(false)
const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const toast = useToast()
const activeTab = ref('staging')
const wowPath = ref(localStorage.getItem('wow_path') || '')

// 公会筛选相关
const filterGuildId = ref<number | null>(null)
const currentGuild = ref<Guild | null>(null)
const storyFilter = computed<StoryFilterParams | undefined>(() => {
  return filterGuildId.value ? { guild_id: String(filterGuildId.value) } : undefined
})

// 创建剧情对话框
const showCreateModal = ref(false)
const newStoryTitle = ref('')
const newStoryDesc = ref('')
const newStoryRegion = ref('')
const newStoryAddress = ref('')
const creating = ref(false)
const storyListRef = ref<InstanceType<typeof StoryList> | null>(null)
const stagingPoolRef = ref<InstanceType<typeof StagingPool> | null>(null)

// 标签选择
const allTags = ref<Tag[]>([])
const selectedTagIds = ref<number[]>([])

// 归档模式：create（创建新剧情）或 append（追加到已有剧情）
const archiveMode = ref<'create' | 'append'>('create')
const userStories = ref<Story[]>([])
const selectedStoryId = ref<number | null>(null)
const loadingStories = ref(false)
const storySearch = ref('')
const archiveError = ref('')
const archiveWarning = ref('')
const archiveStage = ref<'idle' | 'creating' | 'tagging' | 'checking' | 'entries' | 'finalizing'>('idle')
const createdArchiveStoryId = ref<number | null>(null)
const appliedArchiveTagIds = ref<number[]>([])
const entrySubmissionAttempted = ref(false)

const RECENT_ARCHIVE_STORY_KEY = 'rpbox_recent_archive_story_id'
const STORY_OPTION_RENDER_LIMIT = 80
const LARGE_ARCHIVE_THRESHOLD = 800

// 待归档的记录
const pendingRecords = ref<ChatRecord[]>([])

const recentArchiveStoryId = ref<number | null>((() => {
  const value = Number(localStorage.getItem(RECENT_ARCHIVE_STORY_KEY))
  return Number.isFinite(value) && value > 0 ? value : null
})())

const filteredUserStories = computed(() => {
  const query = storySearch.value.trim().toLocaleLowerCase()
  const stories = query
    ? userStories.value.filter((story) => {
      const tags = story.tag_list?.map(tag => tag.name).join(' ') || story.tags || ''
      return [story.title, story.description, story.region, story.address, tags]
        .filter(Boolean)
        .some(value => String(value).toLocaleLowerCase().includes(query))
    })
    : [...userStories.value]

  const recentId = recentArchiveStoryId.value
  return stories.sort((left, right) => {
    if (left.id === recentId) return -1
    if (right.id === recentId) return 1
    return new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime()
  })
})

const displayedUserStories = computed(() => filteredUserStories.value.slice(0, STORY_OPTION_RENDER_LIMIT))
const hiddenStoryOptionCount = computed(() => Math.max(0, filteredUserStories.value.length - displayedUserStories.value.length))
const selectedExistingStory = computed(() => userStories.value.find(story => story.id === selectedStoryId.value) || null)
const archiveTargetLocked = computed(() => creating.value || createdArchiveStoryId.value !== null)

const archiveSummary = computed(() => {
  const records = [...pendingRecords.value].sort((left, right) => left.timestamp - right.timestamp)
  const speakerNames = new Set<string>()
  const channels = new Set<string>()
  let identityEventCount = 0
  let emptyEntryCount = 0

  records.forEach((record) => {
    const speaker = archiveRecordSpeaker(record)
    if (speaker) speakerNames.add(speaker)
    if (record.event || record.mark === 'S') identityEventCount += 1
    else if (record.channel) channels.add(record.channel.toUpperCase())
    if (!archiveRecordContent(record)) emptyEntryCount += 1
  })

  const first = records[0]
  const last = records.at(-1)
  return {
    count: records.length,
    start: first ? new Date(first.timestamp * 1000) : null,
    end: last ? new Date(last.timestamp * 1000) : null,
    speakerNames: [...speakerNames],
    channelCount: channels.size,
    identityEventCount,
    emptyEntryCount,
    isLarge: records.length >= LARGE_ARCHIVE_THRESHOLD,
  }
})

const archiveTimeRange = computed(() => formatArchiveTimeRange(archiveSummary.value.start, archiveSummary.value.end))
const archiveStageText = computed(() => archiveStage.value === 'idle'
  ? ''
  : t(`archives.modal.archiveStage.${archiveStage.value}`))
const archiveDismissLocked = computed(() => Boolean(
  creating.value
  || (createdArchiveStoryId.value !== null && pendingRecords.value.length > 0),
))
const archiveSubmitDisabled = computed(() => {
  if (creating.value) return true
  if (archiveMode.value === 'append' && !selectedStoryId.value) {
    return true
  }
  return archiveSummary.value.emptyEntryCount > 0
})

// 插件状态
const showAddonInstaller = ref(false)
const addonInstalled = ref(false)
const addonVersion = ref<string | null>(null)
const addonChecking = ref(false)
const selectedFlavor = ref('_retail_')
const addonUpdateDialogRef = ref<InstanceType<typeof AddonUpdateDialog> | null>(null)

// 使用提示（可永久关闭）
const showUsageTips = ref(!localStorage.getItem('rpbox_usage_tips_dismissed'))

function dismissUsageTips() {
  showUsageTips.value = false
  localStorage.setItem('rpbox_usage_tips_dismissed', '1')
}

function normalizeAddonVersion(version: string | null | undefined) {
  return (version || '').trim().replace(/^v/i, '')
}

async function checkAddonStatus(): Promise<InstalledAddonInfo | null> {
  console.log('[ArchivesMain] checkAddonStatus 被调用')
  if (!wowPath.value) return null
  try {
    const info = await invoke<InstalledAddonInfo>('check_addon_installed', {
      wowPath: wowPath.value,
      flavor: selectedFlavor.value,
    })
    console.log('[ArchivesMain] 检查插件结果:', info)
    addonInstalled.value = info.installed
    addonVersion.value = info.installed ? (normalizeAddonVersion(info.version) || '未知') : null
    console.log('[ArchivesMain] 更新后的版本号:', addonVersion.value)
    return info
  } catch (e) {
    console.error('检测插件失败:', e)
    return null
  }
}

// 检查插件更新（自动）
async function checkAddonUpdate() {
  try {
    if (!wowPath.value) return

    const installedInfo = await checkAddonStatus()
    if (!installedInfo?.installed) return

    const manifest = await getAddonManifest()
    const latestVersion = normalizeAddonVersion(manifest.latest)
    const currentVersion = normalizeAddonVersion(installedInfo.version)

    if (!latestVersion || !currentVersion) return

    if (currentVersion === latestVersion) {
      localStorage.setItem('addon_last_checked_version', latestVersion)
      localStorage.removeItem('addon_update_prompt_key')
      return
    }

    const promptKey = `${currentVersion}->${latestVersion}`
    if (localStorage.getItem('addon_update_prompt_key') === promptKey) return

    // 查找最新版本的详细信息（包括 changelog）
    const latestVersionInfo = manifest.versions.find(v => normalizeAddonVersion(v.version) === latestVersion)
    const changelog = latestVersionInfo?.changelog || '暂无更新说明'

    addonUpdateDialogRef.value?.show(currentVersion, latestVersion, changelog, wowPath.value, selectedFlavor.value)
    localStorage.setItem('addon_update_prompt_key', promptKey)
  } catch (e) {
    console.error('检查插件更新失败:', e)
  }
}

// 手动检测更新
async function handleCheckAddonUpdate() {
  if (!addonVersion.value) {
    return
  }

  addonChecking.value = true
  try {
    const installedInfo = await checkAddonStatus()
    const currentVersion = normalizeAddonVersion(installedInfo?.version || addonVersion.value)
    if (!currentVersion) return

    const manifest = await getAddonManifest()
    const latestVersion = normalizeAddonVersion(manifest.latest)

    if (currentVersion === latestVersion) {
      // 使用 toast 提示（需要导入 toast）
      console.log('当前已是最新版本')
    } else {
      const latestVersionInfo = manifest.versions.find(v => normalizeAddonVersion(v.version) === latestVersion)
      const changelog = latestVersionInfo?.changelog || '暂无更新说明'
      addonUpdateDialogRef.value?.show(currentVersion, latestVersion, changelog, wowPath.value, selectedFlavor.value)
    }
  } catch (e) {
    console.error('检查插件更新失败:', e)
  } finally {
    addonChecking.value = false
  }
}

onMounted(() => {
  loadTags()
  // 从 URL query 读取公会筛选
  if (route.query.guild_id) {
    filterGuildId.value = Number(route.query.guild_id)
    loadGuildInfo()
    // 切换到"我的剧情"标签页
    activeTab.value = 'stories'
  }

  // 检查是否已设置魔兽路径，未设置时留在当前模块展示配置提示
  const savedPath = localStorage.getItem('wow_path')
  if (!savedPath) {
    setTimeout(() => mounted.value = true, 50)
    return
  }
  wowPath.value = savedPath
  setTimeout(() => mounted.value = true, 50)
  checkAddonUpdate()  // 检查插件更新
})

// 监听路由变化
watch(() => route.query.guild_id, async (newGuildId) => {
  if (newGuildId) {
    filterGuildId.value = Number(newGuildId)
    await loadGuildInfo()
    // 切换到"我的剧情"标签页
    activeTab.value = 'stories'
  } else {
    filterGuildId.value = null
    currentGuild.value = null
    // 清除筛选后，如果当前在"我的剧情"，切换回"待归档池"
    if (activeTab.value === 'stories') {
      activeTab.value = 'staging'
    }
  }
})

async function loadGuildInfo() {
  if (!filterGuildId.value) return
  try {
    const res = await getGuild(filterGuildId.value)
    currentGuild.value = res.guild
  } catch (error) {
    console.error('加载公会信息失败:', error)
  }
}

// 监听标签页切换，每次打开时检查插件更新
watch(activeTab, (newTab) => {
  if (newTab === 'staging' || newTab === 'stories') {
    checkAddonUpdate()
  }
})

async function loadTags() {
  try {
    const res = await listTags('story')
    allTags.value = res.tags || []
  } catch (e) {
    console.error('加载标签失败:', e)
  }
}

async function loadUserStories() {
  loadingStories.value = true
  try {
    const res = await listStories({ sort: 'updated_at', order: 'desc' })
    userStories.value = res.stories || []
    archiveWarning.value = ''
  } catch (e) {
    console.error('加载剧情列表失败:', e)
    userStories.value = []
    archiveWarning.value = t('archives.modal.storyListLoadFailed')
  } finally {
    loadingStories.value = false
  }
}

// 清理TRP3特殊格式字符
function cleanTRP3Content(content: string): string {
  return content
    .replace(/\{[^}]+\}/g, '') // 移除 {icon:xxx} {col:xxx} 等标记
    .replace(/\|c[0-9a-fA-F]{8}/g, '') // 移除颜色开始标记 |cFFFFFFFF
    .replace(/\|r/g, '') // 移除颜色结束标记 |r
    .replace(/\|T[^|]+\|t/g, '') // 移除纹理标记 |Txxx|t
    .replace(/\|H[^|]+\|h/g, '') // 移除超链接标记
    .replace(/\|h/g, '')
    .replace(/[\uE000-\uF8FF]/g, '') // 移除私用区Unicode字符
    .replace(/\uFFFD/g, '') // 移除替换字符 �
    .replace(/[\u0000-\u001F]/g, '') // 移除控制字符
    .trim()
}

function stripNpcPrefix(content: string): string {
  if (!content || content.startsWith('|c')) return content
  return content.replace(/^\|+\s*/, '')
}

function snapshotDisplayName(snapshot?: ProfileSnapshot): string {
  if (!snapshot) return ''
  if (snapshot.n) return cleanTRP3Content(snapshot.n)
  if (snapshot.FN) {
    return cleanTRP3Content(snapshot.LN ? `${snapshot.FN} ${snapshot.LN}` : snapshot.FN)
  }
  return ''
}

function identityEndpointName(endpoint?: IdentityEndpoint): string {
  if (!endpoint) return t('archives.staging.unknownProfile')
  return cleanTRP3Content(
    endpoint.display_name
    || endpoint.profile_name
    || endpoint.ref_id
    || t('archives.staging.unknownProfile'),
  )
}

function buildIdentityEventContent(record: ChatRecord): string {
  const eventLabel = record.event?.kind === 'profile_update'
    ? t('archives.staging.identityUpdated')
    : t('archives.staging.identitySwitched')
  const certainty = record.event?.certainty === 'exact'
    ? t('archives.staging.identityExact')
    : t('archives.staging.identityObserved')
  return `${eventLabel}：${identityEndpointName(record.event?.from)} → ${identityEndpointName(record.event?.to)}（${certainty}）`
}

function archiveRecordSpeaker(record: ChatRecord): string {
  if (record.event || record.mark === 'S' || record.mark === 'B' || (record.mark === 'N' && !record.npc)) return ''
  if (record.mark === 'N' && record.npc) return cleanTRP3Content(record.npc)

  const historicalName = snapshotDisplayName(record.profile_snapshot)
  if (historicalName) return historicalName
  const trp3 = record.sender.trp3
  if (trp3?.FN) return cleanTRP3Content(trp3.LN ? `${trp3.FN} ${trp3.LN}` : trp3.FN)
  return cleanTRP3Content(record.sender.gameID.split('-')[0])
}

function archiveRecordContent(record: ChatRecord): string {
  if (record.event || record.mark === 'S') return cleanTRP3Content(buildIdentityEventContent(record))
  const content = record.mark === 'B' || (record.mark === 'N' && !record.npc)
    ? stripNpcPrefix(record.content)
    : record.content
  return cleanTRP3Content(content)
}

function formatArchiveTimeRange(start: Date | null, end: Date | null): string {
  if (!start || !end) return '—'
  const dateFormatter = new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric' })
  const timeFormatter = new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit' })
  const sameDay = start.getFullYear() === end.getFullYear()
    && start.getMonth() === end.getMonth()
    && start.getDate() === end.getDate()
  if (sameDay) return `${dateFormatter.format(start)} ${timeFormatter.format(start)}–${timeFormatter.format(end)}`
  return `${dateFormatter.format(start)} ${timeFormatter.format(start)} – ${dateFormatter.format(end)} ${timeFormatter.format(end)}`
}

function suggestedArchiveTitle(records: ChatRecord[]): string {
  const sorted = [...records].sort((left, right) => left.timestamp - right.timestamp)
  const start = sorted[0] ? new Date(sorted[0].timestamp * 1000) : new Date()
  const date = new Intl.DateTimeFormat(locale.value, { month: '2-digit', day: '2-digit' }).format(start)
  const speakers = [...new Set(sorted.map(archiveRecordSpeaker).filter(Boolean))].slice(0, 2)
  return speakers.length > 0
    ? t('archives.modal.suggestedTitle', { date, speakers: speakers.join(locale.value.startsWith('zh') ? '、' : ' & ') })
    : t('archives.modal.suggestedTitleNoSpeaker', { date })
}

function resetCreateDialog(clearPending = true) {
  if (clearPending) pendingRecords.value = []
  newStoryTitle.value = ''
  newStoryDesc.value = ''
  newStoryRegion.value = ''
  newStoryAddress.value = ''
  selectedTagIds.value = []
  selectedStoryId.value = null
  archiveMode.value = 'create'
  storySearch.value = ''
  archiveError.value = ''
  archiveWarning.value = ''
  archiveStage.value = 'idle'
  createdArchiveStoryId.value = null
  appliedArchiveTagIds.value = []
  entrySubmissionAttempted.value = false
}

function openCreateStoryModal() {
  resetCreateDialog()
  showCreateModal.value = true
}

function closeCreateStoryModal() {
  if (archiveDismissLocked.value) return
  showCreateModal.value = false
  resetCreateDialog()
}

function setArchiveMode(mode: 'create' | 'append') {
  if (archiveTargetLocked.value || archiveMode.value === mode) return
  archiveMode.value = mode
  archiveError.value = ''
  archiveWarning.value = ''
  entrySubmissionAttempted.value = false
  if (mode === 'create') selectedStoryId.value = null
}

function selectArchiveStory(id: number) {
  if (creating.value) return
  if (selectedStoryId.value !== id) entrySubmissionAttempted.value = false
  selectedStoryId.value = id
  archiveError.value = ''
}

function toggleArchiveTag(id: number) {
  if (archiveTargetLocked.value) return
  selectedTagIds.value = selectedTagIds.value.includes(id)
    ? selectedTagIds.value.filter(tagId => tagId !== id)
    : [...selectedTagIds.value, id]
}

async function handleArchive(records: ChatRecord[]) {
  resetCreateDialog()
  pendingRecords.value = records
  newStoryTitle.value = suggestedArchiveTitle(records)
  showCreateModal.value = true
  // 加载用户剧情列表供追加选择
  loadUserStories()
}

// 将待归档记录转换为条目请求
function buildEntriesFromRecords(records: ChatRecord[]): CreateStoryEntryRequest[] {
  return records.map(record => {
    let speaker = archiveRecordSpeaker(record)
    let type: string = 'dialogue'
    let channel: string = record.channel
    let isNpc: boolean = false
    const content = archiveRecordContent(record)

    if (record.event || record.mark === 'S') {
      type = 'narration'
      channel = 'SYSTEM'
    } else if (record.mark === 'N' && record.npc) {
      isNpc = true
      if (record.nt) {
        channel = record.nt.toUpperCase()
      }
    } else if (record.mark === 'B' || (record.mark === 'N' && !record.npc)) {
      type = 'narration'
    }

    const isIdentityEvent = Boolean(record.event) || record.mark === 'S'

    return {
      source_id: `chat_${record.record_key}`,
      type: type,
      speaker: speaker,
      content,
      channel: channel,
      timestamp: new Date(record.timestamp * 1000).toISOString(),
      ref_id: isIdentityEvent ? undefined : record.ref_id,
      game_id: isIdentityEvent ? undefined : record.sender.gameID,
      trp3_data: isIdentityEvent ? undefined : record.raw_profile,
      is_npc: isIdentityEvent ? false : isNpc,
    }
  })
}

async function handleCreateStory() {
  archiveError.value = ''
  archiveWarning.value = ''
  const title = newStoryTitle.value.trim()
  if (archiveMode.value === 'create' && !title) {
    archiveError.value = t('archives.modal.titleRequired')
    return
  }
  if (title.length > 256) {
    archiveError.value = t('archives.modal.titleTooLong')
    return
  }
  if (newStoryRegion.value.trim().length > 128) {
    archiveError.value = t('archives.modal.regionTooLong')
    return
  }
  if (newStoryAddress.value.trim().length > 256) {
    archiveError.value = t('archives.modal.addressTooLong')
    return
  }
  if (archiveMode.value === 'append' && !selectedStoryId.value) {
    archiveError.value = t('archives.modal.targetRequired')
    return
  }

  const entries = buildEntriesFromRecords(pendingRecords.value)
  if (entries.some(entry => !entry.content.trim())) {
    archiveError.value = t('archives.modal.emptyRecordWarning', { count: archiveSummary.value.emptyEntryCount })
    return
  }

  let failedTagCount = 0
  let skippedDuplicateCount = 0
  creating.value = true
  try {
    let storyId: number
    let targetTitle = ''

    if (archiveMode.value === 'create') {
      if (createdArchiveStoryId.value) {
        storyId = createdArchiveStoryId.value
        targetTitle = title
      } else {
        archiveStage.value = 'creating'
        const story = await createStory({
          title,
          description: newStoryDesc.value,
          region: newStoryRegion.value.trim(),
          address: newStoryAddress.value.trim(),
        })
        storyId = story.id
        targetTitle = story.title || title
        createdArchiveStoryId.value = story.id
      }

      const missingTagIds = selectedTagIds.value.filter(tagId => !appliedArchiveTagIds.value.includes(tagId))
      if (missingTagIds.length > 0) {
        archiveStage.value = 'tagging'
        for (const tagId of missingTagIds) {
          try {
            await addStoryTag(storyId, tagId)
            appliedArchiveTagIds.value = [...appliedArchiveTagIds.value, tagId]
          } catch (e) {
            console.error('添加标签失败:', e)
            failedTagCount += 1
          }
        }
      }
    } else {
      storyId = selectedStoryId.value!
      targetTitle = selectedExistingStory.value?.title || ''
    }

    let entriesToSubmit = entries
    if (entries.length > 0 && entrySubmissionAttempted.value) {
      archiveStage.value = 'checking'
      const existing = await getStory(storyId)
      const existingSourceIds = new Set(existing.entries.map(entry => entry.source_id).filter(Boolean))
      entriesToSubmit = entries.filter(entry => !entry.source_id || !existingSourceIds.has(entry.source_id))
      skippedDuplicateCount = entries.length - entriesToSubmit.length
    }

    if (entriesToSubmit.length > 0) {
      archiveStage.value = 'entries'
      entrySubmissionAttempted.value = true
      await addStoryEntries(storyId, entriesToSubmit)
    }

    if (pendingRecords.value.length > 0) {
      archiveStage.value = 'finalizing'
      const archivedRecordKeys = pendingRecords.value.map(record => record.record_key)
      stagingPoolRef.value?.removeArchivedRecords?.(archivedRecordKeys)
    }

    recentArchiveStoryId.value = storyId
    localStorage.setItem(RECENT_ARCHIVE_STORY_KEY, String(storyId))
    const archivedCount = pendingRecords.value.length
    showCreateModal.value = false
    resetCreateDialog()
    activeTab.value = 'stories'

    storyListRef.value?.loadStories?.()
    toast.success(archivedCount > 0
      ? t('archives.modal.archiveSuccess', { count: archivedCount, title: targetTitle })
      : t('archives.modal.createSuccess', { title: targetTitle }))
    if (failedTagCount > 0) toast.info(t('archives.modal.tagsPartialFailed', { count: failedTagCount }))
    if (skippedDuplicateCount > 0) toast.info(t('archives.modal.duplicatesSkipped', { count: skippedDuplicateCount }))
  } catch (e) {
    console.error('归档失败:', e)
    const stage = archiveStageText.value || t('archives.modal.archiveStage.entries')
    archiveError.value = t('archives.modal.archiveFailedAt', { stage })
    toast.error(archiveError.value)
  } finally {
    creating.value = false
    archiveStage.value = 'idle'
  }
}

function handleViewStory(id: number) {
  router.push({ name: 'story-detail', params: { id } })
}
</script>

<template>
  <div class="archives-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="page-title">
        <h1>{{ $t('archives.pageTitle') }}</h1>
        <p>{{ $t('archives.pageSubtitle') }}</p>
      </div>
      <button class="btn-create" @click="openCreateStoryModal">
        <i class="ri-add-line"></i> {{ $t('archives.action.createNew') }}
      </button>
    </div>

    <div v-if="!wowPath" class="setup-required anim-item" style="--delay: 0.5">
      <div class="setup-icon">
        <i class="ri-folder-settings-line"></i>
      </div>
      <div class="setup-content">
        <h2>{{ $t('archives.setupRequired.title') }}</h2>
        <p>{{ $t('archives.setupRequired.desc') }}</p>
        <div class="setup-actions">
          <RButton type="primary" @click="router.push({ path: '/sync/setup', query: { redirect: '/archives' } })">
            <i class="ri-settings-3-line"></i>
            {{ $t('archives.setupRequired.action') }}
          </RButton>
          <RButton type="outline" @click="router.push('/guide')">
            <i class="ri-question-line"></i>
            {{ $t('archives.setupRequired.guide') }}
          </RButton>
        </div>
      </div>
    </div>

    <template v-else>
    <!-- 插件状态提示 -->
    <div v-if="wowPath" class="addon-notice anim-item" style="--delay: 0.5">
      <!-- 未安装状态 -->
      <template v-if="!addonInstalled">
        <i class="ri-plug-line"></i>
        <span>{{ $t('archives.addon.needInstall') }}</span>
        <RButton size="small" type="primary" @click="showAddonInstaller = true">
          {{ $t('archives.addon.install') }}
        </RButton>
      </template>

      <!-- 已安装状态 -->
      <template v-else>
        <i class="ri-checkbox-circle-fill addon-installed-icon"></i>
        <span>{{ $t('archives.addon.installed') }}</span>
        <span class="addon-version">v{{ addonVersion }}</span>
        <RButton
          size="small"
          @click="handleCheckAddonUpdate"
          :loading="addonChecking"
        >
          <i class="ri-refresh-line"></i>
          {{ addonChecking ? $t('archives.addon.checking') : $t('archives.addon.checkUpdate') }}
        </RButton>
      </template>
    </div>

    <!-- 使用提示（可永久关闭） -->
    <div v-if="showUsageTips" class="usage-tips-banner anim-item" style="--delay: 0.6">
      <div class="tips-icon">
        <i class="ri-lightbulb-flash-line"></i>
      </div>
      <div class="tips-content">
        <div class="tips-title">{{ $t('archives.tips.title') }}</div>
        <ul class="tips-list">
          <li><code>/rpbox</code> {{ $t('archives.tips.openPanel') }}</li>
          <li><code>/rpbox help</code> {{ $t('archives.tips.viewCommands') }}</li>
          <li>{{ $t('archives.tips.defaultListen') }}</li>
          <li>{{ $t('archives.tips.whitelistTip') }}</li>
        </ul>
      </div>
      <button class="tips-close-btn" @click="dismissUsageTips" :title="$t('archives.tips.dontShowAgain')">
        <i class="ri-close-line"></i>
      </button>
    </div>

    <!-- 公会筛选提示 -->
    <div v-if="currentGuild" class="guild-filter-banner anim-item" style="--delay: 0.7">
      <div class="banner-content">
        <i class="ri-shield-line"></i>
        <span v-html="$t('archives.filter.viewingGuild', { name: `<strong>${currentGuild.name}</strong>` })"></span>
      </div>
      <button class="clear-filter-btn" @click="router.push({ name: 'archives' })">
        <i class="ri-close-line"></i>
        {{ $t('archives.filter.clearFilter') }}
      </button>
    </div>

    <!-- Tab切换 -->
    <RTabs v-model="activeTab" class="anim-item" style="--delay: 1">
      <RTabPane v-if="!filterGuildId" name="staging" :label="$t('archives.tabs.staging')">
        <StagingPool ref="stagingPoolRef" @archive="handleArchive" />
      </RTabPane>
      <RTabPane name="stories" :label="$t('archives.tabs.stories')">
        <StoryList ref="storyListRef" :initialFilter="storyFilter" @create="openCreateStoryModal" @view="handleViewStory" />
      </RTabPane>
    </RTabs>
    </template>

    <!-- 创建/追加剧情对话框 -->
    <RModal
      v-model="showCreateModal"
      :title="pendingRecords.length > 0 ? $t('archives.modal.archiveTitle') : $t('archives.modal.createTitle')"
      width="680px"
      :closable="!archiveDismissLocked"
      :mask-closable="!archiveDismissLocked"
      @close="closeCreateStoryModal"
    >
      <div class="create-form">
        <section v-if="pendingRecords.length > 0" class="archive-manifest" aria-live="polite">
          <div class="manifest-heading">
            <span class="manifest-seal"><i class="ri-archive-drawer-line"></i></span>
            <div>
              <strong>{{ $t('archives.modal.archiveManifest') }}</strong>
              <span>{{ archiveTimeRange }}</span>
            </div>
            <b>{{ $t('archives.modal.recordUnit', { count: archiveSummary.count }) }}</b>
          </div>
          <div class="manifest-rule">
            <span>{{ $t('archives.modal.speakerSummary', { count: archiveSummary.speakerNames.length }) }}</span>
            <span>{{ $t('archives.modal.channelSummary', { count: archiveSummary.channelCount }) }}</span>
            <span v-if="archiveSummary.identityEventCount > 0">
              {{ $t('archives.modal.identitySummary', { count: archiveSummary.identityEventCount }) }}
            </span>
          </div>
          <p v-if="archiveSummary.speakerNames.length > 0" class="manifest-speakers">
            {{ archiveSummary.speakerNames.slice(0, 5).join(' · ') }}
            <span v-if="archiveSummary.speakerNames.length > 5">+{{ archiveSummary.speakerNames.length - 5 }}</span>
          </p>
          <p v-if="archiveSummary.isLarge" class="archive-notice warning">
            <i class="ri-timer-line"></i>{{ $t('archives.modal.largeArchiveWarning', { count: archiveSummary.count }) }}
          </p>
          <p v-if="archiveSummary.emptyEntryCount > 0" class="archive-notice danger">
            <i class="ri-error-warning-line"></i>{{ $t('archives.modal.emptyRecordWarning', { count: archiveSummary.emptyEntryCount }) }}
          </p>
        </section>

        <!-- 模式切换（仅在有待归档记录时显示） -->
        <div v-if="pendingRecords.length > 0" class="mode-switcher">
          <button
            class="mode-btn"
            :class="{ active: archiveMode === 'create' }"
            :disabled="archiveTargetLocked"
            @click="setArchiveMode('create')"
          >
            <i class="ri-add-line"></i> {{ $t('archives.mode.createNew') }}
          </button>
          <button
            class="mode-btn"
            :class="{ active: archiveMode === 'append' }"
            :disabled="archiveTargetLocked"
            @click="setArchiveMode('append')"
          >
            <i class="ri-file-add-line"></i> {{ $t('archives.mode.appendExisting') }}
          </button>
        </div>

        <p v-if="createdArchiveStoryId" class="archive-recovery-note">
          <i class="ri-shield-check-line"></i>
          {{ $t('archives.modal.safeRetryTarget', { id: createdArchiveStoryId }) }}
        </p>

        <!-- 创建模式：显示标题、描述、标签 -->
        <template v-if="archiveMode === 'create'">
          <div class="form-field">
            <label>{{ $t('archives.modal.storyTitle') }}</label>
            <RInput v-model="newStoryTitle" :disabled="archiveTargetLocked" :placeholder="$t('archives.modal.storyTitlePlaceholder')" />
            <span class="field-count" :class="{ invalid: newStoryTitle.length > 256 }">{{ newStoryTitle.length }}/256</span>
          </div>
          <div class="form-field">
            <label>{{ $t('archives.modal.storyDesc') }}</label>
            <textarea v-model="newStoryDesc" :disabled="archiveTargetLocked" :placeholder="$t('archives.modal.storyDescPlaceholder')" rows="3"></textarea>
          </div>
          <div class="form-field">
            <label>{{ $t('archives.modal.storyRegion') }}</label>
            <RInput v-model="newStoryRegion" :disabled="archiveTargetLocked" :placeholder="$t('archives.modal.storyRegionPlaceholder')" />
            <span class="field-count" :class="{ invalid: newStoryRegion.trim().length > 128 }">{{ newStoryRegion.trim().length }}/128</span>
          </div>
          <div class="form-field">
            <label>{{ $t('archives.modal.storyAddress') }}</label>
            <RInput v-model="newStoryAddress" :disabled="archiveTargetLocked" :placeholder="$t('archives.modal.storyAddressPlaceholder')" />
            <span class="field-count" :class="{ invalid: newStoryAddress.trim().length > 256 }">{{ newStoryAddress.trim().length }}/256</span>
          </div>
          <div class="form-field">
            <label>{{ $t('archives.modal.addTags') }}</label>
            <div class="tag-selector">
              <span
                v-for="tag in allTags"
                :key="tag.id"
                class="tag-option"
                :class="{ selected: selectedTagIds.includes(tag.id) }"
                :style="selectedTagIds.includes(tag.id) ? { background: `#${tag.color}`, color: 'var(--color-text-light)' } : { borderColor: `#${tag.color}`, color: `#${tag.color}` }"
                :aria-disabled="archiveTargetLocked"
                @click="toggleArchiveTag(tag.id)"
              >
                {{ tag.name }}
              </span>
            </div>
          </div>
        </template>

        <!-- 追加模式：显示剧情选择器 -->
        <template v-else>
          <div class="form-field">
            <label>{{ $t('archives.modal.selectStory') }}</label>
            <RInput
              v-model="storySearch"
              type="search"
              clearable
              :disabled="creating"
              :placeholder="$t('archives.modal.storySearchPlaceholder')"
            />
            <div v-if="loadingStories" class="loading-stories">
              <i class="ri-loader-4-line spinning"></i> {{ $t('archives.status.loading') }}
            </div>
            <div v-else-if="userStories.length === 0" class="no-stories">
              {{ $t('archives.empty.noStories') }}
            </div>
            <div v-else-if="filteredUserStories.length === 0" class="no-stories">
              {{ $t('archives.modal.noMatchingStories') }}
            </div>
            <div v-else class="story-selector">
              <button
                v-for="story in displayedUserStories"
                :key="story.id"
                type="button"
                class="story-option"
                :class="{ selected: selectedStoryId === story.id }"
                :disabled="creating"
                @click="selectArchiveStory(story.id)"
              >
                <div class="story-option-title">
                  <span>{{ story.title }}</span>
                  <em v-if="story.id === recentArchiveStoryId">{{ $t('archives.modal.recentTarget') }}</em>
                </div>
                <div class="story-option-meta">
                  <span>{{ $t('archives.modal.entryCount', { count: story.entry_count || 0 }) }}</span>
                  <span>{{ $t('archives.modal.updatedAt', { date: new Date(story.updated_at).toLocaleDateString() }) }}</span>
                </div>
                <div v-if="story.tag_list?.length" class="story-option-tags">
                  <span v-for="tag in story.tag_list.slice(0, 3)" :key="tag.name">{{ tag.name }}</span>
                </div>
              </button>
              <p v-if="hiddenStoryOptionCount > 0" class="story-options-limited">
                {{ $t('archives.modal.storyOptionsLimited', { count: hiddenStoryOptionCount }) }}
              </p>
            </div>
          </div>
        </template>

        <p v-if="archiveWarning" class="archive-notice warning">
          <i class="ri-information-line"></i>{{ archiveWarning }}
        </p>
        <p v-if="archiveError" class="archive-notice danger" role="alert">
          <i class="ri-error-warning-line"></i>{{ archiveError }}
        </p>
        <p v-if="creating && archiveStageText" class="archive-progress" aria-live="polite">
          <i class="ri-loader-4-line spinning"></i>{{ archiveStageText }}
        </p>
      </div>
      <template #footer>
        <RButton type="outline" :disabled="archiveDismissLocked" @click="closeCreateStoryModal">{{ $t('archives.action.cancel') }}</RButton>
        <RButton
          type="primary"
          :loading="creating"
          :disabled="archiveSubmitDisabled"
          @click="handleCreateStory"
        >
          {{ archiveMode === 'create' ? $t('archives.action.create') : $t('archives.action.append') }}
        </RButton>
      </template>
    </RModal>

    <!-- 插件安装器 -->
    <AddonInstaller
      v-model="showAddonInstaller"
      :wow-path="wowPath"
      @installed="checkAddonStatus"
    />

    <!-- 插件更新提示 -->
    <AddonUpdateDialog ref="addonUpdateDialogRef" @installed="checkAddonStatus" />
  </div>
</template>

<style scoped>
.archives-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.top-toolbar {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  padding: 24px 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.page-title h1 {
  font-size: 28px;
  color: var(--color-primary, #4B3621);
  margin: 0 0 4px 0;
}

.page-title p {
  font-size: 14px;
  color: var(--color-text-secondary, #856a52);
  margin: 0;
}

.setup-required {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 28px 32px;
  background:
    radial-gradient(circle at 10% 10%, color-mix(in srgb, var(--color-accent, #B87333) 16%, transparent), transparent 34%),
    var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 18px;
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
}

.setup-icon {
  width: 58px;
  height: 58px;
  border-radius: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--icon-bg, rgba(128, 64, 48, 0.1));
  color: var(--icon-color, var(--color-primary));
  font-size: 28px;
  flex-shrink: 0;
}

.setup-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.setup-content h2 {
  margin: 0;
  color: var(--color-text-main, #2C1810);
  font-size: 20px;
}

.setup-content p {
  margin: 0;
  color: var(--color-text-secondary, #856a52);
  line-height: 1.6;
}

.setup-actions {
  display: flex;
  gap: 10px;
  margin-top: 4px;
}

.btn-create {
  background: var(--color-secondary, #804030);
  color: var(--color-text-light, #fff);
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-bar {
  display: flex;
  gap: 12px;
}

.filter-item {
  background: var(--color-card-bg, rgba(255,255,255,0.6));
  padding: 10px 16px;
  border-radius: 20px;
  border: 1px solid var(--color-border, #d1bfa8);
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.timeline-section {
  position: relative;
  padding: 40px 0;
}

.timeline-line {
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--color-primary, #4B3621);
  transform: translateX(-50%);
  opacity: 0.3;
}

.timeline-item {
  display: flex;
  margin-bottom: 40px;
  position: relative;
}

.timeline-item.left { justify-content: flex-start; padding-right: 52%; }
.timeline-item.right { justify-content: flex-end; padding-left: 52%; }

.timeline-dot {
  width: 16px;
  height: 16px;
  background: var(--color-background, #EED9C4);
  border: 3px solid var(--color-primary, #4B3621);
  border-radius: 50%;
  position: absolute;
  left: 50%;
  top: 24px;
  transform: translateX(-50%);
  z-index: 2;
}

.timeline-item.highlight .timeline-dot {
  border-color: var(--color-accent, #B87333);
  box-shadow: 0 0 0 4px var(--color-primary-light, rgba(184,115,51,0.2));
}

.story-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  padding: 24px;
  box-shadow: var(--shadow-md, 0 8px 24px rgba(75,54,33,0.08));
  max-width: 400px;
}

.card-date {
  display: inline-block;
  background: var(--color-primary-light, rgba(184,115,51,0.1));
  color: var(--color-accent, #B87333);
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.card-title {
  font-size: 18px;
  color: var(--color-text-main, #2c1e12);
  font-weight: 600;
  margin-bottom: 12px;
}

.card-body {
  font-size: 14px;
  color: var(--color-text-secondary, #665242);
  line-height: 1.7;
  margin-bottom: 16px;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid var(--color-border-light, #f0e6dc);
  padding-top: 16px;
}

.avatars { display: flex; }
.avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  border: 2px solid var(--color-panel-bg, #fff);
  margin-left: -8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-light, #fff);
}
.avatar:nth-child(1) { background: var(--color-accent, #D4A373); margin-left: 0; }
.avatar:nth-child(2) { background: var(--avatar-color-2, #A98467); }
.avatar:nth-child(3) { background: var(--avatar-color-3, #ADC178); }
.avatar:nth-child(4) { background: var(--avatar-color-4, #A9D6E5); }

.view-detail {
  color: var(--color-accent, #B87333);
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }

.create-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.archive-manifest {
  position: relative;
  overflow: hidden;
  padding: 16px 18px 14px;
  border: 1px solid var(--color-border, #d1bfa8);
  border-left: 4px solid var(--color-accent, #B87333);
  border-radius: 10px;
  color: var(--color-primary, #4B3621);
  background:
    repeating-linear-gradient(0deg, transparent 0 29px, color-mix(in srgb, var(--color-border, #d1bfa8) 42%, transparent) 30px),
    color-mix(in srgb, var(--color-panel-bg, #fff) 90%, var(--color-accent, #B87333));
}

.manifest-heading {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
}

.manifest-heading > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.manifest-heading strong {
  font-size: 15px;
  letter-spacing: 0.04em;
}

.manifest-heading span,
.manifest-rule,
.manifest-speakers {
  color: var(--color-text-secondary, #856a52);
  font-size: 12px;
}

.manifest-heading b {
  color: var(--color-accent, #B87333);
  font-family: ui-monospace, 'Consolas', monospace;
  font-size: 15px;
}

.manifest-seal {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--color-accent, #B87333) 55%, transparent);
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-accent, #B87333) 13%, transparent);
  color: var(--color-accent, #B87333) !important;
  font-size: 18px !important;
}

.manifest-rule {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed color-mix(in srgb, var(--color-border, #d1bfa8) 75%, transparent);
}

.manifest-speakers {
  margin: 8px 0 0;
  line-height: 1.5;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field :deep(.r-input) {
  width: 100%;
}

.form-field label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary, #4B3621);
}

.field-count {
  align-self: flex-end;
  margin-top: -3px;
  color: var(--color-text-secondary, #856a52);
  font-size: 11px;
  font-family: ui-monospace, 'Consolas', monospace;
}

.field-count.invalid {
  color: var(--color-danger, #b42318);
  font-weight: 700;
}

.form-field textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border, #d1bfa8);
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
  background: var(--color-panel-bg, #fff);
  color: var(--color-primary, #4B3621);
}

.form-field textarea:focus {
  outline: none;
  border-color: var(--color-accent, #B87333);
  box-shadow: 0 0 0 2px var(--color-primary-light, rgba(184, 115, 51, 0.1));
}

.form-field textarea:disabled {
  opacity: 0.62;
  cursor: not-allowed;
  background: var(--color-card-bg, #f5efe7);
}

.archive-notice,
.archive-progress,
.archive-recovery-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  padding: 9px 11px;
  border-radius: 7px;
  font-size: 12px;
  line-height: 1.5;
}

.archive-notice.warning {
  margin-top: 10px;
  color: var(--color-warning-dark, #8a4b08);
  background: color-mix(in srgb, var(--color-warning, #d69028) 12%, transparent);
}

.archive-notice.danger {
  margin-top: 10px;
  color: var(--color-danger, #a3342d);
  background: color-mix(in srgb, var(--color-danger, #a3342d) 10%, transparent);
}

.archive-progress,
.archive-recovery-note {
  color: var(--color-secondary, #804030);
  background: color-mix(in srgb, var(--color-accent, #B87333) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-accent, #B87333) 28%, transparent);
}

.archive-recovery-note i {
  color: var(--color-success, #4f7a50);
}

.pending-info {
  font-size: 13px;
  color: var(--color-accent, #B87333);
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  padding: 8px 12px;
  border-radius: 6px;
  margin: 0;
}

.addon-notice {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: linear-gradient(135deg, var(--color-primary-light, rgba(184, 115, 51, 0.12)) 0%, var(--color-card-bg, #f5efe7) 100%);
  border: 1px solid var(--color-border-light, rgba(184, 115, 51, 0.35));
  border-radius: 8px;
  color: var(--color-text-main, #2c1e12);
  font-size: 14px;
}

.addon-notice i {
  font-size: 18px;
  color: var(--color-accent, #B87333);
}

.addon-notice span {
  flex: 1;
}

.addon-installed-icon {
  color: var(--color-success, #4CAF50) !important;
}

.addon-version {
  flex: none !important;
  padding: 4px 10px;
  background: var(--btn-secondary-bg, rgba(128, 64, 48, 0.1));
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-main, #2c1e12);
}

/* 使用提示横幅 */
.usage-tips-banner {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 20px;
  background: var(--color-warning-bg, linear-gradient(135deg, #FFF8E1 0%, #FFF3E0 100%));
  border: 1px solid var(--color-warning-border, #FFE0B2);
  border-left: 4px solid var(--color-warning, #FFB300);
  border-radius: 8px;
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(255, 179, 0, 0.1));
}

.tips-icon {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, var(--color-warning, #FFB300), var(--color-warning-dark, #FF9800));
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tips-icon i {
  font-size: 20px;
  color: var(--btn-primary-text, var(--color-text-light, #fff));
}

.tips-content {
  flex: 1;
}

.tips-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-warning-dark, #E65100);
  margin-bottom: 8px;
}

.tips-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  color: var(--color-text-main, #5D4037);
  line-height: 1.8;
}

.tips-list code {
  padding: 2px 6px;
  background: var(--color-primary-light, rgba(184, 115, 51, 0.15));
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: var(--color-accent, #B87333);
}

.tips-close-btn {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--color-accent, #BF8040);
  font-size: 18px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.tips-close-btn:hover {
  background: var(--color-primary-light, rgba(191, 128, 64, 0.15));
  color: var(--color-warning-dark, #E65100);
}

/* 公会筛选横幅 */
.guild-filter-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(135deg, var(--color-card-bg, #FFF5E6), var(--color-panel-bg, #FFF9F0));
  border: 1px solid var(--color-border, #E5D4C1);
  border-left: 4px solid var(--color-accent, #B87333);
  border-radius: 8px;
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(184, 115, 51, 0.08));
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: var(--color-primary, #4B3621);
}

.banner-content i {
  font-size: 20px;
  color: var(--color-accent, #B87333);
}

.banner-content strong {
  color: var(--color-secondary, #804030);
  font-weight: 600;
}

.clear-filter-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-filter-btn:hover {
  background: var(--color-card-bg, #FFF5E6);
  border-color: var(--color-accent, #B87333);
  color: var(--color-accent, #B87333);
}

.clear-filter-btn i {
  font-size: 14px;
}

.tag-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-option {
  padding: 6px 12px;
  border: 1.5px solid;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tag-option:hover {
  opacity: 0.8;
}

.tag-option.selected {
  font-weight: 600;
}

.tag-option[aria-disabled='true'] {
  opacity: 0.58;
  cursor: not-allowed;
}

/* 模式切换器 */
.mode-switcher {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.mode-btn {
  flex: 1;
  padding: 10px 16px;
  border: 1.5px solid var(--color-border, #d1bfa8);
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
  color: var(--color-text-secondary, #665242);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: var(--color-accent, #B87333);
  color: var(--color-accent, #B87333);
}

.mode-btn.active {
  background: var(--color-secondary, #804030);
  border-color: var(--color-secondary, #804030);
  color: var(--color-text-light, #fff);
}

.mode-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

/* 剧情选择器 */
.story-selector {
  max-height: 240px;
  overflow-y: auto;
  border: 1px solid var(--color-border, #d1bfa8);
  border-radius: 8px;
}

.story-option {
  display: block;
  width: 100%;
  padding: 12px 14px;
  border: none;
  cursor: pointer;
  background: transparent;
  text-align: left;
  color: inherit;
  border-bottom: 1px solid var(--color-border-light, #f0e6dc);
  transition: background 0.2s;
}

.story-option:last-child {
  border-bottom: none;
}

.story-option:hover {
  background: var(--color-card-bg-hover, rgba(184, 115, 51, 0.05));
}

.story-option.selected {
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  border-left: 3px solid var(--color-accent, #B87333);
}

.story-option:disabled {
  cursor: wait;
}

.story-option-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary, #4B3621);
  margin-bottom: 4px;
}

.story-option-title em {
  flex: none;
  padding: 2px 7px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-accent, #B87333) 13%, transparent);
  color: var(--color-accent, #B87333);
  font-size: 10px;
  font-style: normal;
  font-weight: 700;
}

.story-option-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 12px;
  font-size: 12px;
  color: var(--color-text-secondary, #856a52);
}

.story-option-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 7px;
}

.story-option-tags span {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--color-card-bg, #f5efe7);
  color: var(--color-text-secondary, #856a52);
  font-size: 10px;
}

.story-options-limited {
  margin: 0;
  padding: 9px 12px;
  border-top: 1px solid var(--color-border-light, #f0e6dc);
  color: var(--color-text-secondary, #856a52);
  background: var(--color-card-bg, #f5efe7);
  font-size: 11px;
  text-align: center;
}

.loading-stories,
.no-stories {
  padding: 24px;
  text-align: center;
  color: var(--color-text-secondary, #856a52);
  font-size: 14px;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 640px) {
  .manifest-heading {
    grid-template-columns: auto 1fr;
  }

  .manifest-heading b {
    grid-column: 2;
  }

  .mode-switcher {
    flex-direction: column;
  }

  .story-selector {
    max-height: 210px;
  }
}
</style>
