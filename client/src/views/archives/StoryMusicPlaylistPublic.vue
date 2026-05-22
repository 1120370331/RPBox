<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getPublicStoryMusicPlaylist, type StoryMusicPlaylist, type StoryMusicTrack } from '@/api/story'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const playlist = ref<StoryMusicPlaylist | null>(null)
const tracks = ref<StoryMusicTrack[]>([])
const code = computed(() => route.params.code as string)

async function loadPlaylist() {
  loading.value = true
  error.value = ''
  try {
    const res = await getPublicStoryMusicPlaylist(code.value)
    playlist.value = res.playlist
    tracks.value = res.tracks || []
  } catch (e: any) {
    error.value = e.message || '歌单加载失败'
  } finally {
    loading.value = false
  }
}

function formatFileSize(size: number) {
  if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

onMounted(loadPlaylist)
</script>

<template>
  <main class="playlist-page">
    <div v-if="loading" class="state">
      <i class="ri-loader-4-line spinning"></i>
      加载中...
    </div>
    <div v-else-if="error" class="state error">
      <i class="ri-error-warning-line"></i>
      {{ error }}
    </div>
    <template v-else-if="playlist">
      <section class="playlist-hero">
        <span class="playlist-color" :style="{ backgroundColor: playlist.color }"></span>
        <p class="eyebrow">RPBox 背景音乐歌单</p>
        <h1>{{ playlist.name }}</h1>
        <p v-if="playlist.description" class="description">{{ playlist.description }}</p>
        <div class="meta">
          <span>{{ playlist.authorName || '匿名作者' }}</span>
          <span>{{ tracks.length }} 首音乐</span>
          <span><i class="ri-eye-line"></i> {{ playlist.viewCount }}</span>
        </div>
      </section>

      <section class="track-list">
        <article v-for="track in tracks" :key="track.id" class="track-row">
          <span class="track-color" :style="{ backgroundColor: track.color }"></span>
          <div class="track-main">
            <div class="track-title">
              <strong>{{ track.name }}</strong>
              <span>{{ Math.round(track.volume * 100) }}%</span>
            </div>
            <div class="track-meta">
              <span>{{ track.fileName }}</span>
              <span>{{ formatFileSize(track.size) }}</span>
            </div>
            <audio controls preload="metadata" :src="track.url"></audio>
          </div>
        </article>
      </section>
    </template>
  </main>
</template>

<style scoped>
.playlist-page {
  min-height: 100vh;
  padding: 40px 20px 80px;
  background: #f5f0e8;
  color: #4B3621;
}

.state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 60vh;
  color: #856a52;
}

.state.error {
  color: #c0392b;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.playlist-hero,
.track-list {
  max-width: 820px;
  margin: 0 auto;
}

.playlist-hero {
  position: relative;
  padding: 28px 0 24px;
}

.playlist-color {
  display: block;
  width: 52px;
  height: 6px;
  border-radius: 999px;
  margin-bottom: 16px;
}

.eyebrow {
  margin: 0 0 8px;
  color: #856a52;
  font-size: 13px;
}

h1 {
  margin: 0;
  font-size: 32px;
  letter-spacing: 0;
}

.description {
  max-width: 680px;
  margin: 12px 0 0;
  color: #665242;
  line-height: 1.7;
}

.meta,
.track-meta,
.track-title {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.meta {
  margin-top: 14px;
  color: #856a52;
  font-size: 14px;
}

.track-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.track-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  border: 1px solid #e5d4c1;
  border-radius: 8px;
  background: #fff;
}

.track-color {
  border-radius: 999px;
}

.track-main {
  min-width: 0;
}

.track-title {
  justify-content: space-between;
}

.track-meta {
  margin-top: 4px;
  color: #856a52;
  font-size: 13px;
}

audio {
  width: 100%;
  margin-top: 10px;
}
</style>
