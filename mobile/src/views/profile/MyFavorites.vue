<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { request } from '@shared/api/request'
import { resolveApiUrl } from '@/api/image'
import type { PostWithAuthor } from '@/api/post'

const { t } = useI18n()
const router = useRouter()
const posts = ref<PostWithAuthor[]>([])
const loading = ref(false)

async function loadFavorites() {
  loading.value = true
  try {
    const res = await request.get<{ posts: PostWithAuthor[]; total: number }>('/posts/favorites')
    posts.value = res.posts || []
  } catch (e) {
    console.error('Failed to load favorites', e)
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  const now = new Date()
  const days = Math.floor((now.getTime() - d.getTime()) / 86400000)
  if (days < 1) return t('common.time.today')
  if (days < 30) return t('common.time.daysAgo', { n: days })
  return `${d.getMonth() + 1}/${d.getDate()}`
}

onMounted(loadFavorites)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myFavorites.title') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="posts.length === 0" class="hint">{{ $t('profile.myFavorites.empty') }}</div>
      <div v-else class="post-list">
        <button
          v-for="post in posts"
          :key="post.id"
          class="post-card"
          @click="router.push({ name: 'post-detail', params: { id: post.id } })"
        >
          <div v-if="post.cover_image_url" class="post-cover">
            <img :src="resolveApiUrl(post.cover_image_url)" alt="" loading="lazy" />
          </div>
          <div class="post-info">
            <span v-if="post.category" class="cat">{{ post.category }}</span>
            <h3>{{ post.title }}</h3>
            <div class="meta">
              <span class="author">{{ post.author_name }}</span>
              <span>{{ formatDate(post.created_at) }}</span>
            </div>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>
