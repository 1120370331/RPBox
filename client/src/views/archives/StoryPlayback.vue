<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { getPublicStory, type Story, type StoryEntry } from '@/api/story'
import WowIcon from '@/components/WowIcon.vue'

const route = useRoute()

const loading = ref(true)
const error = ref('')
const story = ref<Story | null>(null)
const entries = ref<StoryEntry[]>([])
const author = ref('')

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
    author.value = res.author
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
          <div class="entry-avatar">
            <WowIcon v-if="entry.speaker_ic" :icon="entry.speaker_ic" :size="44" :fallback="entry.speaker?.charAt(0) || '?'" />
            <span v-else>{{ entry.speaker?.charAt(0) || '?' }}</span>
          </div>
          <div class="entry-body">
            <div class="entry-speaker">
              {{ entry.speaker || '未知' }}
              <span v-if="entry.type === 'narration'" class="entry-type">旁白</span>
            </div>
            <div class="entry-text">{{ entry.content }}</div>
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

.entry-text {
  font-size: 15px;
  color: #665242;
  line-height: 1.6;
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
