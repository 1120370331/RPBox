<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { getPublicStory, type Story, type StoryEntry } from '@/api/story'
import { type Character } from '@/api/character'
import WowIcon from '@/components/WowIcon.vue'
import CharacterCard from '@/components/CharacterCard.vue'
import ImageViewer from '@/components/ImageViewer.vue'

const route = useRoute()

const loading = ref(true)
const error = ref('')
const story = ref<Story | null>(null)
const entries = ref<StoryEntry[]>([])
const characters = ref<Record<number, Character>>({})
const author = ref('')

// 角色卡片弹窗
const showCharacterCard = ref(false)
const selectedEntry = ref<StoryEntry | null>(null)
const characterCardPosition = ref({ x: 0, y: 0 })

const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)

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

// 播放控制
const isPlaying = ref(false)
const currentIndex = ref(0)
const playSpeed = ref(1)
const playTimer = ref<number | null>(null)

const shareCode = computed(() => route.params.code as string)

const visibleEntries = computed(() => {
  if (isPlaying.value) {
    return entries.value.slice(0, currentIndex.value + 1)
  }
  return entries.value
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
  const character = getEntryCharacter(entry)
  if (character) {
    return getCharacterDisplayName(character)
  }
  return entry.speaker || '未知'
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

onMounted(loadStory)
onUnmounted(stopPlay)
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
}

.playback-header h1 {
  font-size: 32px;
  color: #4B3621;
  margin: 0 0 12px 0;
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
</style>
