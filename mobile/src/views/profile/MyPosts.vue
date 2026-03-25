<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@shared/stores/user'
import { listPosts, type PostWithAuthor, type ListPostsParams } from '@/api/post'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'
import MobilePagination from '@/components/MobilePagination.vue'

const router = useRouter()
const userStore = useUserStore()
const posts = ref<PostWithAuthor[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = 12
const requestSerial = ref(0)
const switchingPage = ref(false)

async function loadMyPosts() {
  const serial = ++requestSerial.value
  loading.value = true
  if (switchingPage.value) {
    posts.value = []
  }
  try {
    const params: ListPostsParams & { author_id?: number } = {
      page: currentPage.value,
      page_size: pageSize,
      sort: 'created_at',
      order: 'desc',
    }
    if (userStore.user?.id) (params as any).author_id = userStore.user.id
    const res = await listPosts(params as any)
    if (serial !== requestSerial.value) return
    posts.value = res.posts || []
    total.value = res.total || 0
  } catch (e) {
    console.error('Failed to load my posts', e)
  } finally {
    if (serial === requestSerial.value) {
      loading.value = false
      switchingPage.value = false
    }
  }
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const totalPages = () => Math.max(1, Math.ceil(total.value / pageSize))

function goToPage(page: number) {
  if (page < 1 || page > totalPages() || page === currentPage.value) return
  switchingPage.value = true
  currentPage.value = page
  document.querySelector('.mobile-content')?.scrollTo({ top: 0, behavior: 'smooth' })
  loadMyPosts()
}

onMounted(loadMyPosts)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myPosts.title') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading && posts.length === 0" class="post-list">
        <div v-for="i in 4" :key="`skeleton-${i}`" class="post-card skeleton-card">
          <div class="post-cover skeleton-block" />
          <div class="post-info">
            <div class="skeleton-line w-30" />
            <div class="skeleton-line w-85" />
            <div class="skeleton-line w-65" />
          </div>
        </div>
      </div>
      <div v-else-if="posts.length === 0" class="hint">{{ $t('profile.myPosts.empty') }}</div>
      <div v-else class="post-list">
        <button
          v-for="post in posts"
          :key="post.id"
          class="post-card"
          @click="router.push({ name: 'post-detail', params: { id: post.id } })"
        >
          <div v-if="post.cover_image_url" class="post-cover">
            <CachedImage :src="resolveApiUrl(post.cover_image_url)" alt="" />
          </div>
          <div class="post-info">
            <span v-if="post.category" class="cat">{{ post.category }}</span>
            <h3>{{ post.title }}</h3>
            <div class="meta">
              <span><i class="ri-eye-line" /> {{ post.view_count }}</span>
              <span><i class="ri-heart-line" /> {{ post.like_count }}</span>
              <span>{{ formatDate(post.created_at) }}</span>
            </div>
          </div>
        </button>
      </div>
      <MobilePagination
        v-if="total > pageSize"
        :model-value="currentPage"
        :total-pages="totalPages()"
        :disabled="loading || switchingPage"
        @change="goToPage"
      />
    </div>
  </div>
</template>

<style scoped>
.skeleton-card {
  pointer-events: none;
}

.skeleton-block,
.skeleton-line {
  background: linear-gradient(90deg, #f2ece6 0%, #ffffff 50%, #f2ece6 100%);
  background-size: 220% 100%;
  animation: skeletonShimmer 1.1s linear infinite;
}

.skeleton-line {
  height: 11px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.skeleton-line.w-30 { width: 30%; }
.skeleton-line.w-65 { width: 65%; }
.skeleton-line.w-85 { width: 85%; }

@keyframes skeletonShimmer {
  from { background-position: 200% 0; }
  to { background-position: -20% 0; }
}
</style>
