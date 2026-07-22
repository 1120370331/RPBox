<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getPublicStory, type Story, type StoryEntry, type StoryMusicSegment, type StoryMusicTrack } from '@/api/story'
import { createContentReport } from '@/api/safety'
import { type Character } from '@/api/character'
import { useUserStore } from '@/stores/user'
import { useToast } from '@/composables/useToast'
import RModal from '@/components/RModal.vue'
import WowIcon from '@/components/WowIcon.vue'
import CharacterCard from '@/components/CharacterCard.vue'
import ImageViewer from '@/components/ImageViewer.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const toast = useToast()

const loading = ref(true)
const error = ref('')
const story = ref<Story | null>(null)
const entries = ref<StoryEntry[]>([])
const characters = ref<Record<number, Character>>({})
const author = ref('')
const musicTracks = ref<StoryMusicTrack[]>([])
const musicSegments = ref<StoryMusicSegment[]>([])
const currentMusicSegmentId = ref<number | null>(null)
const musicBlocked = ref(false)
const musicControlOpen = ref(false)
const musicMasterVolume = ref(loadStoredBgmVolume())
const musicMuted = ref(loadStoredBgmBoolean('muted', false))
const musicDisabled = ref(loadStoredBgmBoolean('disabled', false))

// 角色卡片弹窗
const showCharacterCard = ref(false)
const selectedEntry = ref<StoryEntry | null>(null)
const characterCardPosition = ref({ x: 0, y: 0 })

const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)
type StoryReportReason = 'story_content' | 'story_audio'
const showStoryReportModal = ref(false)
const storyReportReason = ref<StoryReportReason>('story_content')
const storyReportEntryId = ref(0)
const storyReportDetail = ref('')
const storyReportSubmitting = ref(false)
const storyReportReasonOptions: { value: StoryReportReason; label: string }[] = [
  { value: 'story_content', label: '剧情内容违规' },
  { value: 'story_audio', label: '音频违规' },
]

const imageEntries = computed(() => {
  const result: { id: number; image: string }[] = []
  for (const entry of entries.value) {
    if (entry.type !== 'image') continue
    const parsed = parseImageEntry(entry)
    if (parsed?.image) {
      result.push({ id: entry.id, image: parsed.image })
    }
  }
  return result
})

const reportableEntries = computed(() => entries.value.map((entry, index) => ({
  entry,
  index,
  label: buildStoryReportEntryLabel(entry, index),
})))

const selectedStoryReportEntry = computed(() => {
  if (!storyReportEntryId.value) return null
  return entries.value.find(entry => entry.id === storyReportEntryId.value) || null
})

const storyReportCanSubmit = computed(() => (
  storyReportDetail.value.trim().length > 0 || storyReportEntryId.value > 0
))

// 播放控制
const isPlaying = ref(false)
const currentIndex = ref(0)
const playSpeed = ref(1)
const playTimer = ref<number | null>(null)
let musicAudio: HTMLAudioElement | null = null
let musicFadeTimer: number | null = null
let musicScrollRaf = 0

const shareCode = computed(() => route.params.code as string)

function loadStoredBgmVolume() {
  const fallback = 80
  try {
    const raw = localStorage.getItem('rpbox.storyBgm.volume')
    const parsed = raw === null ? fallback : Number(raw)
    return Number.isFinite(parsed) ? Math.min(Math.max(parsed, 0), 100) : fallback
  } catch {
    return fallback
  }
}

function loadStoredBgmBoolean(key: 'muted' | 'disabled', fallback: boolean) {
  try {
    const raw = localStorage.getItem(`rpbox.storyBgm.${key}`)
    return raw === null ? fallback : raw === 'true'
  } catch {
    return fallback
  }
}

function saveBgmPreference(key: 'volume' | 'muted' | 'disabled', value: string | number | boolean) {
  try {
    localStorage.setItem(`rpbox.storyBgm.${key}`, String(value))
  } catch {
    // Keep session-level control even if localStorage is unavailable.
  }
}

const visibleEntries = computed(() => {
  if (isPlaying.value) {
    return entries.value.slice(0, currentIndex.value + 1)
  }
  return entries.value
})

const entryIndexById = computed(() => {
  const map = new Map<number, number>()
  entries.value.forEach((entry, index) => {
    map.set(entry.id, index)
  })
  return map
})

const sortedMusicSegments = computed(() => {
  return [...musicSegments.value].sort((a, b) => {
    const startA = getEntrySortIndex(a.startEntryId)
    const startB = getEntrySortIndex(b.startEntryId)
    if (startA !== startB) return startA - startB
    return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
  })
})

const currentMusicSegment = computed(() => {
  if (!currentMusicSegmentId.value) return null
  return musicSegments.value.find(segment => segment.id === currentMusicSegmentId.value) || null
})

const currentMusicTrack = computed(() => {
  return currentMusicSegment.value ? getMusicTrack(currentMusicSegment.value.trackId) : null
})

async function loadStory() {
  loading.value = true
  error.value = ''
  try {
    const res = await getPublicStory(shareCode.value)
    story.value = res.story
    entries.value = res.entries || []
    characters.value = res.characters || {}
    author.value = res.author
    musicTracks.value = res.music_tracks || []
    musicSegments.value = res.music_segments || []
    scheduleMusicScrollCheck()
    console.log('[StoryPlayback] entries:', entries.value)
    console.log('[StoryPlayback] characters:', characters.value)
    console.log('[StoryPlayback] 第一条entry:', entries.value[0])
    if (entries.value[0]?.character_id) {
      console.log('[StoryPlayback] 第一条entry的character_id:', entries.value[0].character_id)
      console.log('[StoryPlayback] 对应角色:', getEntryCharacter(entries.value[0]))
    }
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function startPlay() {
  if (entries.value.length === 0) return
  isPlaying.value = true
  currentIndex.value = 0
  scheduleNext()
}

function stopPlay() {
  isPlaying.value = false
  if (playTimer.value) {
    clearTimeout(playTimer.value)
    playTimer.value = null
  }
}

function scheduleNext() {
  if (!isPlaying.value) return
  const delay = 2000 / playSpeed.value
  playTimer.value = window.setTimeout(() => {
    if (currentIndex.value < entries.value.length - 1) {
      currentIndex.value++
      scheduleNext()
    } else {
      isPlaying.value = false
    }
  }, delay)
}

function skipToStart() {
  stopPlay()
  currentIndex.value = 0
}

function skipToEnd() {
  stopPlay()
  currentIndex.value = entries.value.length - 1
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function getParticipants(): string[] {
  if (!story.value?.participants) return []
  try {
    return JSON.parse(story.value.participants)
  } catch {
    return []
  }
}

function getEntrySortIndex(entryId?: number) {
  if (!entryId) return Number.MAX_SAFE_INTEGER
  return entryIndexById.value.get(entryId) ?? Number.MAX_SAFE_INTEGER
}

function getMusicTrack(trackId?: number | null) {
  if (!trackId) return null
  return musicTracks.value.find(track => track.id === trackId) || null
}

function getMusicControlTrackLabel() {
  if (musicDisabled.value) return '已禁用'
  if (musicBlocked.value) return '点击启用播放'
  if (currentMusicTrack.value) return currentMusicTrack.value.name
  if (musicSegments.value.length) return '等待滚动触发'
  return '暂无 BGM'
}

function getMusicEffectiveVolume(segment: StoryMusicSegment) {
  if (musicDisabled.value || musicMuted.value) return 0
  const segmentVolume = Math.min(Math.max(segment.volume, 0), 1)
  return segmentVolume * (musicMasterVolume.value / 100)
}

function applyMusicVolumePreference(fadeSeconds = 0.12) {
  if (!musicAudio) return
  musicAudio.muted = musicMuted.value
  if (musicDisabled.value) {
    stopMusicPlayback(0.15)
    return
  }

  const segment = currentMusicSegment.value
  if (segment && !musicAudio.paused) {
    fadeMusicAudioTo(getMusicEffectiveVolume(segment), fadeSeconds)
  }
}

function updateMusicMasterVolume() {
  musicMasterVolume.value = Math.min(Math.max(Number(musicMasterVolume.value) || 0, 0), 100)
  saveBgmPreference('volume', musicMasterVolume.value)
  applyMusicVolumePreference()
}

function toggleMusicMuted() {
  musicMuted.value = !musicMuted.value
  saveBgmPreference('muted', musicMuted.value)
  applyMusicVolumePreference()
}

function toggleMusicDisabled() {
  musicDisabled.value = !musicDisabled.value
  saveBgmPreference('disabled', musicDisabled.value)
  if (musicDisabled.value) {
    musicBlocked.value = false
    stopMusicPlayback(0.15)
  } else {
    enableStoryMusic()
  }
}

function getMusicSegmentForEntry(entryId: number) {
  const entryIndex = getEntrySortIndex(entryId)
  if (entryIndex === Number.MAX_SAFE_INTEGER) return null

  let matched: StoryMusicSegment | null = null
  let matchedStart = -1
  for (const segment of sortedMusicSegments.value) {
    const startIndex = getEntrySortIndex(segment.startEntryId)
    if (startIndex > entryIndex) continue
    const endIndex = segment.endEntryId ? getEntrySortIndex(segment.endEntryId) : entries.value.length - 1
    if (entryIndex <= endIndex && startIndex >= matchedStart) {
      matched = segment
      matchedStart = startIndex
    }
  }
  return matched
}

function getLastVisibleEntryId(): number | null {
  const nodes = document.querySelectorAll<HTMLElement>('.entry-item[data-entry-id]')
  if (!nodes.length) return null

  const viewportBottom = window.innerHeight
  let lastVisibleId: number | null = null
  for (const node of nodes) {
    const rect = node.getBoundingClientRect()
    if (rect.top < viewportBottom && rect.bottom > 0) {
      lastVisibleId = Number(node.dataset.entryId)
    }
  }
  return lastVisibleId
}

function getMusicAudio() {
  if (!musicAudio) {
    musicAudio = new Audio()
    musicAudio.preload = 'auto'
  }
  return musicAudio
}

function clearMusicFadeTimer() {
  if (musicFadeTimer !== null) {
    window.clearTimeout(musicFadeTimer)
    musicFadeTimer = null
  }
}

function fadeMusicAudioTo(targetVolume: number, seconds: number, done?: () => void) {
  const audio = getMusicAudio()
  clearMusicFadeTimer()

  if (seconds <= 0) {
    audio.volume = targetVolume
    done?.()
    return
  }

  const startVolume = audio.volume
  const startedAt = performance.now()
  const duration = seconds * 1000

  const step = () => {
    const progress = Math.min((performance.now() - startedAt) / duration, 1)
    audio.volume = startVolume + (targetVolume - startVolume) * progress
    if (progress < 1) {
      musicFadeTimer = window.setTimeout(step, 40)
    } else {
      musicFadeTimer = null
      done?.()
    }
  }

  step()
}

function stopMusicPlayback(fadeSeconds = 1) {
  if (!musicAudio) return
  const audio = musicAudio
  const stop = () => {
    audio.pause()
    audio.currentTime = 0
    currentMusicSegmentId.value = null
  }

  if (audio.paused || fadeSeconds <= 0) {
    stop()
  } else {
    fadeMusicAudioTo(0, fadeSeconds, stop)
  }
}

async function playMusicSegment(segment: StoryMusicSegment) {
  if (musicDisabled.value) return
  const track = getMusicTrack(segment.trackId)
  if (!track?.url) return

  const audio = getMusicAudio()
  const targetVolume = getMusicEffectiveVolume(segment)

  if (currentMusicSegmentId.value === segment.id && !audio.paused) {
    audio.loop = segment.loop
    audio.muted = musicMuted.value
    fadeMusicAudioTo(targetVolume, 0.2)
    return
  }

  clearMusicFadeTimer()
  audio.pause()
  audio.src = track.url
  audio.currentTime = 0
  audio.loop = segment.loop
  audio.muted = musicMuted.value
  audio.volume = 0
  currentMusicSegmentId.value = segment.id

  try {
    await audio.play()
    musicBlocked.value = false
    fadeMusicAudioTo(targetVolume, segment.fadeInSeconds)
  } catch (e) {
    currentMusicSegmentId.value = null
    musicBlocked.value = true
    console.warn('背景音乐自动播放失败:', e)
  }
}

function updateMusicPlaybackByScroll() {
  if (musicDisabled.value) return
  if (!musicSegments.value.some(segment => segment.autoPlay)) return

  const entryId = getLastVisibleEntryId()
  const segment = entryId ? getMusicSegmentForEntry(entryId) : null
  if (segment?.autoPlay) {
    playMusicSegment(segment)
    return
  }

  if (currentMusicSegment.value?.autoPlay) {
    stopMusicPlayback(currentMusicSegment.value.fadeOutSeconds)
  }
}

function scheduleMusicScrollCheck() {
  if (musicScrollRaf) return
  musicScrollRaf = window.requestAnimationFrame(() => {
    musicScrollRaf = 0
    updateMusicPlaybackByScroll()
  })
}

function enableStoryMusic() {
  musicDisabled.value = false
  saveBgmPreference('disabled', false)
  musicBlocked.value = false
  const entryId = getLastVisibleEntryId()
  const segment = entryId ? getMusicSegmentForEntry(entryId) : null
  if (segment?.autoPlay) {
    playMusicSegment(segment)
  } else {
    scheduleMusicScrollCheck()
  }
}

function parseImageEntry(entry: StoryEntry): { image: string; description: string } | null {
  if (entry.type !== 'image') return null
  try {
    return JSON.parse(entry.content)
  } catch {
    return null
  }
}

function openImageViewer(entryId: number) {
  const images = imageEntries.value
  if (!images.length) return
  const index = images.findIndex((image) => image.id === entryId)
  if (index < 0) return
  viewerImages.value = images.map((image) => image.image)
  viewerStartIndex.value = index
  showImageViewer.value = true
}

function truncateText(text: string, limit: number) {
  const normalized = text.replace(/\s+/g, ' ').trim()
  const chars = Array.from(normalized)
  if (chars.length <= limit) return normalized
  return `${chars.slice(0, limit).join('')}...`
}

function getEntryReportText(entry: StoryEntry) {
  if (entry.type === 'image') {
    const parsed = parseImageEntry(entry)
    return parsed?.description || '图片条目'
  }
  return entry.content || ''
}

function buildStoryReportEntryLabel(entry: StoryEntry, index: number) {
  const speaker = getEntrySpeakerName(entry)
  const channel = entry.channel ? `[${getChannelLabel(entry.channel)}]` : ''
  const text = truncateText(getEntryReportText(entry), 64)
  return `第 ${index + 1} 条 #${entry.id} ${channel} ${speaker}${text ? `：${text}` : ''}`
}

function resetStoryReportForm() {
  storyReportReason.value = 'story_content'
  storyReportEntryId.value = 0
  storyReportDetail.value = ''
}

function openStoryReportModal() {
  if (!story.value) return
  if (!userStore.token) {
    toast.error('请先登录后再举报剧情')
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  if (userStore.user?.id === story.value.user_id) {
    toast.error('不能举报自己的剧情')
    return
  }
  resetStoryReportForm()
  showStoryReportModal.value = true
}

function closeStoryReportModal() {
  if (storyReportSubmitting.value) return
  showStoryReportModal.value = false
}

function buildStoryReportDetail() {
  const parts = [
    `违规类型：${storyReportReasonOptions.find(option => option.value === storyReportReason.value)?.label || '剧情内容违规'}`,
  ]
  const entry = selectedStoryReportEntry.value
  if (entry) {
    const entryIndex = entryIndexById.value.get(entry.id) ?? entries.value.findIndex(item => item.id === entry.id)
    parts.push(`辅助条目：${buildStoryReportEntryLabel(entry, entryIndex >= 0 ? entryIndex : 0)}`)
    if (storyReportReason.value === 'story_audio') {
      const segment = getMusicSegmentForEntry(entry.id)
      const track = segment ? getMusicTrack(segment.trackId) : null
      if (track) {
        parts.push(`关联音频：${track.name}`)
      }
    }
  } else if (storyReportReason.value === 'story_audio' && musicTracks.value.length > 0) {
    parts.push(`剧情音频数量：${musicTracks.value.length}`)
  }
  const note = storyReportDetail.value.trim()
  if (note) {
    parts.push(`补充说明：${note}`)
  }
  return parts.join('\n')
}

async function submitStoryReport() {
  if (!story.value || !storyReportCanSubmit.value || storyReportSubmitting.value) return
  storyReportSubmitting.value = true
  try {
    await createContentReport({
      target_type: 'story',
      target_id: story.value.id,
      reason: storyReportReason.value,
      detail: buildStoryReportDetail(),
      submit_report: true,
    })
    toast.success('举报已提交，版主会尽快处理')
    showStoryReportModal.value = false
    resetStoryReportForm()
  } catch (e: any) {
    toast.error(e?.message || '举报提交失败')
  } finally {
    storyReportSubmitting.value = false
  }
}

// 获取条目对应的角色
function getEntryCharacter(entry: StoryEntry): Character | undefined {
  if (entry.character_id) {
    // Go map[uint] 序列化后 key 是字符串
    return characters.value[entry.character_id] || characters.value[String(entry.character_id) as any]
  }
  return undefined
}

function getCharacterDisplayName(character: Character): string {
  if (character.custom_name) return character.custom_name
  if (character.first_name) {
    return character.last_name
      ? `${character.first_name} ${character.last_name}`
      : character.first_name
  }
  return character.game_id?.split('-')[0] || '未知角色'
}

function getEntrySpeakerName(entry: StoryEntry): string {
  if (entry.type === 'narration') return '旁白'
  // Keep playback faithful to the name captured on this historical entry.
  if (entry.speaker) return entry.speaker
  const character = getEntryCharacter(entry)
  if (character) {
    return getCharacterDisplayName(character)
  }
  return '未知'
}

function getEntrySpeakerInitial(entry: StoryEntry): string {
  const name = getEntrySpeakerName(entry)
  return name ? name.charAt(0) : '?'
}

// 获取条目的头像图标
function getEntryIcon(entry: StoryEntry): string {
  const character = getEntryCharacter(entry)
  if (character) {
    return character.custom_avatar || character.icon || ''
  }
  return ''
}

// 获取条目的名字颜色
function getEntryColor(entry: StoryEntry): string {
  const character = getEntryCharacter(entry)
  if (character) {
    return character.custom_color || character.color || ''
  }
  return ''
}

// 获取频道标签
function getChannelLabel(channel: string): string {
  const map: Record<string, string> = {
    // 新格式（简写）
    'SAY': '说',
    'YELL': '喊',
    'EMOTE': '表情',
    'TEXT_EMOTE': '表情',
    'PARTY': '小队',
    'RAID': '团队',
    'WHISPER': '密语',
    // 旧格式（完整事件名）
    'CHAT_MSG_SAY': '说',
    'CHAT_MSG_YELL': '喊',
    'CHAT_MSG_EMOTE': '表情',
    'CHAT_MSG_TEXT_EMOTE': '表情',
    'CHAT_MSG_PARTY': '小队',
    'CHAT_MSG_RAID': '团队',
    'CHAT_MSG_WHISPER': '密语',
  }
  return map[channel] || channel
}

// 获取频道对应的文字颜色
function getChannelTextColor(channel: string): string {
  const colorMap: Record<string, string> = {
    'SAY': '',
    'YELL': '#FF3333',
    'WHISPER': '#B39DDB',
    'EMOTE': '#FF8C00',
    'TEXT_EMOTE': '#FF8C00',
    'PARTY': '#AAAAFF',
    'RAID': '#FF7F00',
    'CHAT_MSG_SAY': '',
    'CHAT_MSG_YELL': '#FF3333',
    'CHAT_MSG_WHISPER': '#B39DDB',
    'CHAT_MSG_EMOTE': '#FF8C00',
    'CHAT_MSG_TEXT_EMOTE': '#FF8C00',
    'CHAT_MSG_PARTY': '#AAAAFF',
    'CHAT_MSG_RAID': '#FF7F00',
  }
  return colorMap[channel] || ''
}

// 获取频道CSS类
function getChannelClass(channel: string): string {
  if (channel === 'YELL' || channel === 'CHAT_MSG_YELL') return 'channel-yell'
  if (channel === 'WHISPER' || channel === 'CHAT_MSG_WHISPER') return 'channel-whisper'
  return ''
}

// 判断是否是NPC消息
function isNpcEntry(entry: StoryEntry): boolean {
  const character = getEntryCharacter(entry)
  return character?.is_npc || false
}

// 点击头像显示角色卡片
function showCharacterInfo(entry: StoryEntry, event: MouseEvent) {
  if (entry.type === 'narration' || entry.type === 'image') return
  if (isNpcEntry(entry)) return  // NPC不显示角色卡片
  if (!getEntryCharacter(entry)) return

  selectedEntry.value = entry
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  characterCardPosition.value = {
    x: rect.right,
    y: rect.top
  }
  showCharacterCard.value = true
}

function handleMusicControlDocumentClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (target.closest('.story-bgm-control')) return
  musicControlOpen.value = false
}

onMounted(() => {
  loadStory()
  window.addEventListener('scroll', scheduleMusicScrollCheck, { passive: true })
  window.addEventListener('resize', scheduleMusicScrollCheck)
  document.addEventListener('click', handleMusicControlDocumentClick)
})

onUnmounted(() => {
  stopPlay()
  window.removeEventListener('scroll', scheduleMusicScrollCheck)
  window.removeEventListener('resize', scheduleMusicScrollCheck)
  document.removeEventListener('click', handleMusicControlDocumentClick)
  if (musicScrollRaf) {
    window.cancelAnimationFrame(musicScrollRaf)
  }
  stopMusicPlayback(0)
  clearMusicFadeTimer()
})
</script>

<template>
  <div class="playback-page">
    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <i class="ri-loader-4-line spinning"></i> 加载中...
    </div>

    <!-- 错误 -->
    <div v-else-if="error" class="error-state">
      <i class="ri-error-warning-line"></i>
      <p>{{ error }}</p>
    </div>

    <!-- 内容 -->
    <template v-else-if="story">
      <!-- 头部 -->
      <div class="playback-header">
        <button class="story-report-button" type="button" title="举报剧情" @click="openStoryReportModal">
          <i class="ri-alarm-warning-line"></i>
          <span>举报</span>
        </button>
        <h1>{{ story.title }}</h1>
        <div class="story-meta">
          <span>作者: {{ author }}</span>
          <span>{{ formatDate(story.created_at) }}</span>
          <span>参与: {{ getParticipants().length || '?' }}人</span>
          <span><i class="ri-eye-line"></i> {{ story.view_count }}</span>
        </div>
        <p v-if="story.description" class="story-desc">{{ story.description }}</p>
      </div>

      <!-- 对话列表 -->
      <div class="entries-container">
        <div
          v-for="(entry, idx) in visibleEntries"
          :key="entry.id"
          :data-entry-id="entry.id"
          class="entry-item"
          :class="[entry.type, { 'fade-in': isPlaying && idx === currentIndex }]"
        >
          <div
            v-if="entry.type !== 'image'"
            class="entry-avatar"
            :class="{ clickable: entry.type !== 'narration' && !isNpcEntry(entry) && !!getEntryCharacter(entry) }"
            @click="showCharacterInfo(entry, $event)"
          >
            <template v-if="entry.type === 'narration'">
              <span class="avatar-narration">旁白</span>
            </template>
            <template v-else-if="isNpcEntry(entry)">
              <span class="avatar-npc">NPC</span>
            </template>
            <template v-else>
              <WowIcon v-if="getEntryIcon(entry)" :icon="getEntryIcon(entry)" :size="44" :fallback="getEntrySpeakerInitial(entry)" />
              <span v-else>{{ getEntrySpeakerInitial(entry) }}</span>
            </template>
          </div>
          <div class="entry-body">
            <div v-if="entry.type !== 'image'" class="entry-speaker">
              <span :style="entry.type !== 'narration' && getEntryColor(entry) ? { color: '#' + getEntryColor(entry) } : {}">
                {{ getEntrySpeakerName(entry) }}
              </span>
              <span v-if="entry.channel && entry.type !== 'narration'" class="entry-channel" :class="getChannelClass(entry.channel)">[{{ getChannelLabel(entry.channel) }}]</span>
            </div>
            <div v-if="entry.type !== 'image'" class="entry-text" :style="getChannelTextColor(entry.channel) ? { color: getChannelTextColor(entry.channel) } : {}">{{ entry.content }}</div>
            <div v-else-if="parseImageEntry(entry)" class="entry-image-content">
              <div class="entry-image-wrapper" @click="openImageViewer(entry.id)" title="查看图像">
                <img :src="parseImageEntry(entry)!.image" alt="剧情图片" class="entry-image" />
                <div class="entry-image-hover">
                  <i class="ri-zoom-in-line"></i>
                  <span>查看图像</span>
                </div>
              </div>
              <p v-if="parseImageEntry(entry)!.description" class="image-description">
                {{ parseImageEntry(entry)!.description }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- 播放控制 -->
      <div class="playback-controls">
        <div v-if="currentMusicTrack" class="bgm-now-playing">
          <span :style="{ backgroundColor: currentMusicTrack.color }"></span>
          {{ currentMusicTrack.name }}
        </div>
        <button class="ctrl-btn" @click="skipToStart" title="跳到开头">
          <i class="ri-skip-back-line"></i>
        </button>
        <button class="ctrl-btn play-btn" @click="isPlaying ? stopPlay() : startPlay()">
          <i :class="isPlaying ? 'ri-pause-fill' : 'ri-play-fill'"></i>
        </button>
        <button class="ctrl-btn" @click="skipToEnd" title="跳到结尾">
          <i class="ri-skip-forward-line"></i>
        </button>
        <div class="speed-control">
          <span>速度:</span>
          <select v-model="playSpeed">
            <option :value="0.5">0.5x</option>
            <option :value="1">1x</option>
            <option :value="2">2x</option>
            <option :value="3">3x</option>
          </select>
        </div>
        <div class="progress-info">
          {{ currentIndex + 1 }} / {{ entries.length }}
        </div>
      </div>

      <div v-if="musicSegments.length > 0" class="story-bgm-control" :class="{ open: musicControlOpen }" @click.stop>
        <Transition name="story-bgm-panel">
          <div v-if="musicControlOpen" class="story-bgm-panel">
            <div class="story-bgm-panel-head">
              <div>
                <strong>BGM</strong>
                <span>{{ getMusicControlTrackLabel() }}</span>
              </div>
              <span
                v-if="currentMusicTrack"
                class="story-bgm-track-dot"
                :style="{ backgroundColor: currentMusicTrack.color }"
              ></span>
            </div>
            <label class="story-bgm-volume">
              <span>音量 {{ musicMasterVolume }}%</span>
              <input
                v-model.number="musicMasterVolume"
                type="range"
                min="0"
                max="100"
                :disabled="musicDisabled"
                @input="updateMusicMasterVolume"
              />
            </label>
            <div class="story-bgm-actions">
              <button v-if="musicBlocked && !musicDisabled" type="button" class="primary" @click="enableStoryMusic">
                <i class="ri-play-circle-line"></i>
                开启
              </button>
              <button type="button" :disabled="musicDisabled" @click="toggleMusicMuted">
                <i :class="musicMuted ? 'ri-volume-mute-line' : 'ri-volume-up-line'"></i>
                {{ musicMuted ? '取消静音' : '静音' }}
              </button>
              <button type="button" class="danger" @click="toggleMusicDisabled">
                <i :class="musicDisabled ? 'ri-play-circle-line' : 'ri-forbid-2-line'"></i>
                {{ musicDisabled ? '启用 BGM' : '禁用 BGM' }}
              </button>
            </div>
          </div>
        </Transition>
        <button
          type="button"
          class="story-bgm-fab"
          :class="{ disabled: musicDisabled, muted: musicMuted || musicBlocked }"
          :title="musicDisabled ? 'BGM 已禁用' : 'BGM 控制'"
          @click="musicControlOpen = !musicControlOpen"
        >
          <i class="ri-music-2-line"></i>
        </button>
      </div>
    </template>
  </div>

  <!-- 角色信息卡片（只读） -->
  <CharacterCard
    v-model:visible="showCharacterCard"
    :character="selectedEntry ? getEntryCharacter(selectedEntry) : undefined"
    :speaker="selectedEntry ? getEntrySpeakerName(selectedEntry) : undefined"
    :position="characterCardPosition"
    :editable="false"
  />

  <ImageViewer
    v-model="showImageViewer"
    :images="viewerImages"
    :start-index="viewerStartIndex"
  />

  <RModal
    v-model="showStoryReportModal"
    title="举报剧情"
    width="560px"
    :mask-closable="!storyReportSubmitting"
  >
    <div class="story-report-form">
      <label class="story-report-field">
        <span>违规类型</span>
        <select v-model="storyReportReason">
          <option v-for="option in storyReportReasonOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </label>
      <label class="story-report-field">
        <span>辅助定位条目</span>
        <select v-model.number="storyReportEntryId">
          <option :value="0">不指定具体条目</option>
          <option v-for="item in reportableEntries" :key="item.entry.id" :value="item.entry.id">
            {{ item.label }}
          </option>
        </select>
      </label>
      <label class="story-report-field">
        <span>补充说明</span>
        <textarea
          v-model="storyReportDetail"
          rows="4"
          maxlength="500"
          placeholder="请说明违规位置或问题表现，选择条目后也可以只填写简短说明"
        ></textarea>
      </label>
      <p class="story-report-hint" :class="{ error: !storyReportCanSubmit }">
        请选择具体条目或填写补充说明；音频违规建议选择触发音频的剧情条目。
      </p>
    </div>
    <template #footer>
      <button class="story-report-modal-btn ghost" type="button" :disabled="storyReportSubmitting" @click="closeStoryReportModal">
        取消
      </button>
      <button
        class="story-report-modal-btn primary"
        type="button"
        :disabled="storyReportSubmitting || !storyReportCanSubmit"
        @click="submitStoryReport"
      >
        {{ storyReportSubmitting ? '提交中...' : '提交举报' }}
      </button>
    </template>
  </RModal>
</template>

<style scoped>
.playback-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #f5f0e8 0%, #e8dfd3 100%);
  padding: 40px 20px;
}

.loading-state,
.error-state {
  text-align: center;
  padding: 80px 20px;
  color: #856a52;
}

.error-state i {
  font-size: 48px;
  color: #dc3545;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.playback-header {
  max-width: 800px;
  margin: 0 auto 32px;
  text-align: center;
  position: relative;
  padding: 0 96px;
}

.playback-header h1 {
  font-size: 32px;
  color: #4B3621;
  margin: 0 0 12px 0;
}

.story-report-button {
  position: absolute;
  top: 0;
  right: 0;
  height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(133, 106, 82, 0.32);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.78);
  color: #6f5846;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, color 0.2s, border-color 0.2s;
}

.story-report-button:hover {
  background: #fff;
  color: #b42318;
  border-color: rgba(180, 35, 24, 0.32);
}

.story-meta {
  display: flex;
  justify-content: center;
  gap: 20px;
  font-size: 14px;
  color: #856a52;
  margin-bottom: 16px;
}

.story-desc {
  font-size: 15px;
  color: #665242;
  line-height: 1.6;
  margin: 0;
}

.entries-container {
  max-width: 800px;
  margin: 0 auto;
  padding-bottom: 100px;
}

.entry-item {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #fff;
  border-radius: 12px;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(75, 54, 33, 0.08);
}

.entry-item.image {
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.entry-item.image .entry-body {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.entry-item.fade-in {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.entry-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: #B87333;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 18px;
  flex-shrink: 0;
}

.entry-avatar.clickable {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.entry-avatar.clickable:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.avatar-narration {
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  background: #a98467;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.avatar-npc {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  background: #9b59b6;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.entry-item.narration .entry-avatar {
  background: #a98467;
}

.entry-item.narration .entry-speaker {
  color: #856a52;
}

.entry-body {
  flex: 1;
}

.entry-speaker {
  font-weight: 600;
  color: #4B3621;
  margin-bottom: 4px;
}

.entry-type {
  font-size: 12px;
  color: #856a52;
  font-weight: normal;
  margin-left: 8px;
}

.entry-channel {
  font-size: 12px;
  color: #856a52;
  font-weight: normal;
  margin-left: 8px;
}

.entry-channel.channel-yell {
  color: #FF3333;
  font-weight: bold;
}

.entry-channel.channel-whisper {
  color: #B39DDB;
}

.entry-text {
  font-size: 15px;
  color: #665242;
  line-height: 1.6;
}

.entry-image-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.entry-image-wrapper {
  position: relative;
  display: inline-flex;
  max-width: 100%;
  cursor: zoom-in;
  margin-bottom: 8px;
  border-radius: 12px;
  overflow: hidden;
}

.entry-image-hover {
  position: absolute;
  inset: 0;
  background: rgba(44, 24, 16, 0.45);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.2s ease;
  pointer-events: none;
  font-size: 13px;
}

.entry-image-wrapper:hover .entry-image-hover {
  opacity: 1;
}

.entry-image {
  max-width: 100%;
  height: auto;
  border-radius: 12px;
  border: 2px solid #e5d4c1;
  display: block;
}

.image-description {
  font-size: 14px;
  color: #665242;
  line-height: 1.6;
  margin: 0;
  padding: 8px 12px;
  background: #f5f0eb;
  border-radius: 6px;
  border-left: 3px solid #d4a373;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.entry-item.narration {
  background: rgba(184, 115, 51, 0.08);
  border-left: 3px solid #B87333;
}

.story-report-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.story-report-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.story-report-field span {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-main);
}

.story-report-field select,
.story-report-field textarea {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-card-bg);
  color: var(--color-text-main);
  padding: 10px 12px;
  font: inherit;
}

.story-report-field textarea {
  resize: vertical;
}

.story-report-hint {
  margin: -4px 0 0;
  font-size: 12px;
  color: var(--color-text-muted);
  line-height: 1.5;
}

.story-report-hint.error {
  color: #b45309;
}

.story-report-modal-btn {
  min-width: 96px;
  height: 38px;
  border-radius: 8px;
  border: 1px solid var(--color-border);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.story-report-modal-btn.ghost {
  background: var(--color-card-bg);
  color: var(--color-text-main);
}

.story-report-modal-btn.primary {
  border-color: var(--color-secondary);
  background: var(--color-secondary);
  color: var(--btn-primary-text, #fff);
}

.story-report-modal-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.playback-controls {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: #fff;
  padding: 16px 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  box-shadow: 0 -4px 20px rgba(75, 54, 33, 0.1);
}

.bgm-now-playing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 180px;
  padding: 8px 10px;
  border-radius: 999px;
  background: #f5f0e8;
  color: #665242;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bgm-now-playing span {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.story-bgm-control {
  position: fixed;
  right: 20px;
  bottom: 92px;
  z-index: 80;
}

.story-bgm-fab {
  width: 48px;
  height: 48px;
  border: none;
  border-radius: 50%;
  background: #4B3621;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 21px;
  box-shadow: 0 8px 24px rgba(75, 54, 33, 0.24);
  transition: transform 0.2s, box-shadow 0.2s, background 0.2s, opacity 0.2s;
}

.story-bgm-fab:hover,
.story-bgm-control.open .story-bgm-fab {
  transform: scale(1.05);
  box-shadow: 0 10px 28px rgba(75, 54, 33, 0.3);
}

.story-bgm-fab.muted {
  background: #856a52;
}

.story-bgm-fab.disabled {
  background: #9b8b7d;
  opacity: 0.82;
}

.story-bgm-panel {
  position: absolute;
  right: 0;
  bottom: 60px;
  width: 260px;
  padding: 12px;
  border: 1px solid #e5d4c1;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 12px 36px rgba(75, 54, 33, 0.22);
}

.story-bgm-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.story-bgm-panel-head div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.story-bgm-panel-head strong {
  color: #4B3621;
  font-size: 14px;
}

.story-bgm-panel-head span {
  color: #856a52;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.story-bgm-track-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 0 3px rgba(184, 115, 51, 0.14);
}

.story-bgm-volume {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: #4B3621;
  font-size: 12px;
}

.story-bgm-volume input {
  width: 100%;
  accent-color: #B87333;
}

.story-bgm-volume input:disabled {
  opacity: 0.45;
}

.story-bgm-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(72px, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.story-bgm-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 32px;
  border: 1px solid #e5d4c1;
  border-radius: 8px;
  background: #f5f0e8;
  color: #4B3621;
  cursor: pointer;
  font-size: 12px;
}

.story-bgm-actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.story-bgm-actions button:not(:disabled):hover {
  border-color: #B87333;
  color: #B87333;
}

.story-bgm-actions button.primary:not(:disabled) {
  border-color: #B87333;
  background: #B87333;
  color: #fff;
}

.story-bgm-actions button.danger:not(:disabled):hover {
  border-color: #dc3545;
  color: #dc3545;
}

.story-bgm-panel-enter-active,
.story-bgm-panel-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.story-bgm-panel-enter-from,
.story-bgm-panel-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}

.ctrl-btn {
  width: 44px;
  height: 44px;
  border: none;
  border-radius: 50%;
  background: #f5f0e8;
  color: #4B3621;
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ctrl-btn:hover {
  background: #e8dfd3;
}

.ctrl-btn.play-btn {
  width: 56px;
  height: 56px;
  background: #B87333;
  color: #fff;
  font-size: 24px;
}

.speed-control {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #856a52;
}

.speed-control select {
  padding: 6px 10px;
  border: 1px solid #d1bfa8;
  border-radius: 6px;
  background: #fff;
}

.progress-info {
  font-size: 14px;
  color: #856a52;
}

@media (max-width: 640px) {
  .playback-header {
    padding: 0;
  }

  .playback-header h1 {
    font-size: 26px;
  }

  .story-report-button {
    position: static;
    margin: 0 auto 14px;
  }

  .story-meta {
    flex-wrap: wrap;
    gap: 8px 14px;
  }
}
</style>
