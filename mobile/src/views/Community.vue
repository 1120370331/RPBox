<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { listPosts, type PostWithAuthor, type ListPostsParams } from '@/api/post'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'

const { t } = useI18n()
const router = useRouter()

const posts = ref<PostWithAuthor[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 12
const requestSerial = ref(0)
const switchingPage = ref(false)

const activeCategory = ref('')
const sortBy = ref<'created_at' | 'like_count' | 'view_count'>('created_at')

const categories = computed(() => [
  { key: '', label: t('community.categories.all') },
  { key: 'discussion', label: t('community.categories.discussion') },
  { key: 'story', label: t('community.categories.story') },
  { key: 'guide', label: t('community.categories.guide') },
  { key: 'showcase', label: t('community.categories.showcase') },
  { key: 'question', label: t('community.categories.question') },
  { key: 'other', label: t('community.categories.other') },
])

const sortOptions = computed(() => [
  { key: 'created_at', label: t('community.sort.latest') },
  { key: 'like_count', label: t('community.sort.hot') },
  { key: 'view_count', label: t('community.sort.mostViewed') },
])

async function loadPosts() {
  const serial = ++requestSerial.value
  loading.value = true
  if (switchingPage.value) {
    posts.value = []
  }
  try {
    const params: ListPostsParams = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
      order: 'desc',
    }
    if (activeCategory.value) params.category = activeCategory.value
    const res = await listPosts(params)
    if (serial !== requestSerial.value) return
    posts.value = res.posts || []
    total.value = res.total || 0
  } catch (e) {
    console.error('Failed to load posts', e)
  } finally {
    if (serial === requestSerial.value) {
      loading.value = false
      switchingPage.value = false
    }
  }
}

function selectCategory(key: string) {
  activeCategory.value = key
  currentPage.value = 1
}

function changeSort(key: string) {
  sortBy.value = key as typeof sortBy.value
  currentPage.value = 1
}

function prevPage() {
  if (currentPage.value <= 1) return
  switchingPage.value = true
  currentPage.value--
  document.querySelector('.mobile-content')?.scrollTo({ top: 0, behavior: 'smooth' })
}

function nextPage() {
  if (currentPage.value * pageSize >= total.value) return
  switchingPage.value = true
  currentPage.value++
  document.querySelector('.mobile-content')?.scrollTo({ top: 0, behavior: 'smooth' })
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return t('common.time.justNow')
  if (mins < 60) return t('common.time.minutesAgo', { n: mins })
  const hours = Math.floor(mins / 60)
  if (hours < 24) return t('common.time.hoursAgo', { n: hours })
  const days = Math.floor(hours / 24)
  if (days < 30) return t('common.time.daysAgo', { n: days })
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const totalPages = () => Math.max(1, Math.ceil(total.value / pageSize))

watch([activeCategory, sortBy, currentPage], loadPosts)
onMounted(loadPosts)
</script>

<template>
  <div class="page community-page">
    <header class="page-header">
      <h1>{{ $t('community.title') }}</h1>
      <div class="sort-row">
        <button
          v-for="opt in sortOptions" :key="opt.key"
          :class="['sort-btn', { active: sortBy === opt.key }]"
          @click="changeSort(opt.key)"
        >{{ opt.label }}</button>
      </div>
    </header>

    <div class="category-bar">
      <button
        v-for="cat in categories" :key="cat.key"
        :class="['cat-btn', { active: activeCategory === cat.key }]"
        @click="selectCategory(cat.key)"
      >{{ cat.label }}</button>
    </div>

    <div class="page-body">
      <div v-if="loading && posts.length === 0" class="post-list skeleton-list">
        <div v-for="i in 4" :key="`skeleton-${i}`" class="post-card skeleton-card">
          <div class="post-cover skeleton-block" />
          <div class="post-content">
            <div class="skeleton-line w-30" />
            <div class="skeleton-line w-85" />
            <div class="skeleton-line w-60" />
            <div class="skeleton-line w-75" />
          </div>
        </div>
      </div>

      <div v-else-if="posts.length === 0" class="empty-hint">{{ $t('community.empty') }}</div>

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
          <div class="post-content">
            <div class="post-meta-top">
              <span v-if="post.category" class="category-tag">{{ post.category }}</span>
              <span class="post-time">{{ formatDate(post.created_at) }}</span>
            </div>
            <h3 class="post-title">{{ post.title }}</h3>
            <div class="post-author">
              <img
                v-if="post.author_avatar"
                :src="resolveApiUrl(post.author_avatar)"
                class="author-avatar"
                alt=""
              />
              <i v-else class="ri-user-3-fill avatar-icon" />
              <span
                class="author-name"
                :style="{
                  color: post.author_name_color || undefined,
                  fontWeight: post.author_name_bold ? 'bold' : undefined,
                }"
              >{{ post.author_name }}</span>
            </div>
            <div class="post-stats">
              <span><i class="ri-eye-line" /> {{ post.view_count }}</span>
              <span><i class="ri-heart-line" /> {{ post.like_count }}</span>
              <span><i class="ri-chat-3-line" /> {{ post.comment_count }}</span>
            </div>
          </div>
        </button>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <button :disabled="currentPage <= 1" @click="prevPage">{{ $t('common.pagination.prev') }}</button>
        <span>{{ $t('common.pagination.pageInfo', { current: currentPage, total: totalPages() }) }}</span>
        <button :disabled="currentPage >= totalPages()" @click="nextPage">{{ $t('common.pagination.next') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 0 16px calc(20px + var(--safe-bottom, 0px)); }
.page-header { padding: 14px 0 10px; display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.page-header h1 { font-size: 22px; flex-shrink: 0; }

.sort-row { display: flex; gap: 6px; flex-wrap: wrap; justify-content: flex-end; }
.sort-btn {
  padding: 4px 10px; border: none; border-radius: 12px;
  background: transparent; color: var(--text-dark); font-size: 12px; cursor: pointer;
}
.sort-btn.active { background: var(--color-primary); color: var(--text-light); }

.category-bar {
  display: flex; gap: 8px; overflow-x: auto; padding: 0 0 10px;
  -webkit-overflow-scrolling: touch; scrollbar-width: none;
}
.category-bar::-webkit-scrollbar { display: none; }
.cat-btn {
  flex-shrink: 0; padding: 6px 14px; border: 1px solid var(--color-border);
  border-radius: 16px; background: transparent; color: var(--text-dark);
  font-size: 13px; cursor: pointer; white-space: nowrap;
}
.cat-btn.active { background: var(--color-primary); color: var(--text-light); border-color: var(--color-primary); }

.loading-hint, .empty-hint {
  text-align: center; padding: 60px 0; color: var(--color-accent); font-size: 14px;
}

.post-list { display: flex; flex-direction: column; gap: 12px; }

.post-card {
  width: 100%;
  border: none;
  text-align: left;
  cursor: pointer;
  background: var(--color-card-bg); border-radius: var(--radius-md); overflow: hidden;
  box-shadow: var(--shadow-sm);
}
.post-cover { width: 100%; height: 160px; overflow: hidden; }
.post-cover img { width: 100%; height: 100%; object-fit: cover; }

.post-content { padding: 14px 14px; }
.post-meta-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.category-tag {
  font-size: 11px; padding: 2px 8px; border-radius: 8px;
  background: var(--tag-bg); color: var(--tag-text);
}
.post-time { font-size: 11px; color: var(--color-text-secondary); }

.post-title { font-size: 15px; font-weight: 600; line-height: 1.4; margin-bottom: 8px; }

.post-author { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.author-avatar { width: 22px; height: 22px; border-radius: 50%; object-fit: cover; }
.avatar-icon { font-size: 18px; color: var(--color-secondary); }
.author-name { font-size: 12px; color: var(--color-text-secondary); }

.post-stats { display: flex; gap: 14px; font-size: 12px; color: var(--color-text-secondary); }
.post-stats i { margin-right: 2px; }

.pagination {
  display: flex; align-items: center; justify-content: center; gap: 16px;
  padding: 16px 0;
}
.pagination button {
  padding: 8px 16px; border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  background: transparent; color: var(--text-dark); font-size: 13px; cursor: pointer;
}
.pagination button:disabled { opacity: 0.4; cursor: default; }
.pagination span { font-size: 13px; color: var(--color-text-secondary); }

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
  height: 12px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.skeleton-line.w-30 { width: 30%; }
.skeleton-line.w-60 { width: 60%; }
.skeleton-line.w-75 { width: 75%; }
.skeleton-line.w-85 { width: 85%; }

@keyframes skeletonShimmer {
  from { background-position: 200% 0; }
  to { background-position: -20% 0; }
}
</style>
