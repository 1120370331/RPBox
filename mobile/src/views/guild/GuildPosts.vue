<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getGuild, type Guild } from '@/api/guild'
import { listPosts, POST_CATEGORIES, type ListPostsParams, type PostWithAuthor } from '@/api/post'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'
import MobilePagination from '@/components/MobilePagination.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const guildId = computed(() => Number(route.params.id))

const loading = ref(false)
const noPermission = ref(false)
const guild = ref<Guild | null>(null)
const posts = ref<PostWithAuthor[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 12
const requestSerial = ref(0)
const switchingPage = ref(false)

const searchKeyword = ref('')
const filterCategory = ref('')
const sortBy = ref<'created_at' | 'view_count' | 'like_count'>('created_at')

const filteredPosts = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return posts.value
  return posts.value.filter((post) =>
    `${post.title} ${stripHtml(post.content)}`.toLowerCase().includes(keyword),
  )
})

const categoryOptions = computed(() => {
  return [
    { value: '', label: t('guild.posts.all') },
    ...POST_CATEGORIES.map((cat) => ({ value: cat.value, label: cat.label })),
  ]
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

async function loadGuild() {
  if (!guildId.value) return
  const res = await getGuild(guildId.value)
  guild.value = res.guild
}

async function loadPosts() {
  const serial = ++requestSerial.value
  if (!guildId.value) return
  loading.value = true
  noPermission.value = false
  if (switchingPage.value) {
    posts.value = []
  }
  try {
    const params: ListPostsParams = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
      order: 'desc',
      status: 'published',
      guild_id: guildId.value,
    }
    if (filterCategory.value) params.category = filterCategory.value
    const res = await listPosts(params)
    if (serial !== requestSerial.value) return
    posts.value = res.posts || []
    total.value = res.total || 0
  } catch (error: any) {
    posts.value = []
    total.value = 0
    const status = error?.response?.status
    const message = String(error?.message || '')
    if (status === 403 || message.includes('403') || message.includes('无权')) {
      noPermission.value = true
    }
  } finally {
    if (serial === requestSerial.value) {
      loading.value = false
      switchingPage.value = false
    }
  }
}

function goBack() {
  router.push({ name: 'guild-detail', params: { id: guildId.value } })
}

function goToStories() {
  router.push({ name: 'guild-stories', params: { id: guildId.value } })
}

function viewPost(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html || ''
  return div.textContent || div.innerText || ''
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return '--'
  const now = new Date()
  const diffHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60))
  if (diffHours < 1) return t('guild.posts.timeJustNow')
  if (diffHours < 24) return t('guild.posts.timeHoursAgo', { n: diffHours })
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 7) return t('guild.posts.timeDaysAgo', { n: diffDays })
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function formatLocation(region?: string, address?: string) {
  const parts = [region, address].map((part) => part?.trim()).filter(Boolean)
  return parts.join(' · ')
}

function onPageChange(page: number) {
  if (page === currentPage.value) return
  switchingPage.value = true
  currentPage.value = page
  document.querySelector('.mobile-content')?.scrollTo({ top: 0, behavior: 'smooth' })
}

watch([currentPage, sortBy, filterCategory], loadPosts)
onMounted(async () => {
  await loadGuild()
  await loadPosts()
})
</script>

<template>
  <div class="sub-page guild-posts-page">
    <header class="sub-header">
      <button class="back-btn" @click="goBack"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('guild.posts.title') }}</h1>
    </header>

    <div class="sub-body">
      <div class="switch-row">
        <button class="switch-btn active"><i class="ri-article-line" /> {{ $t('guild.posts.guildPosts') }}</button>
        <button class="switch-btn" @click="goToStories"><i class="ri-book-2-line" /> {{ $t('guild.posts.guildStories') }}</button>
      </div>

      <div class="toolbar-card">
        <p class="guild-name">{{ guild?.name || '-' }}</p>
        <div class="sort-row">
          <button :class="['sort-btn', { active: sortBy === 'created_at' }]" @click="sortBy = 'created_at'">{{ $t('guild.posts.sortLatest') }}</button>
          <button :class="['sort-btn', { active: sortBy === 'like_count' }]" @click="sortBy = 'like_count'">{{ $t('guild.posts.sortPopular') }}</button>
          <button :class="['sort-btn', { active: sortBy === 'view_count' }]" @click="sortBy = 'view_count'">{{ $t('guild.posts.sortViews') }}</button>
        </div>
        <div class="search-bar">
          <i class="ri-search-line" />
          <input v-model="searchKeyword" :placeholder="$t('guild.posts.searchPlaceholder')" />
        </div>
        <div class="category-row">
          <button
            v-for="item in categoryOptions"
            :key="item.value || 'all'"
            :class="['cat-btn', { active: filterCategory === item.value }]"
            @click="filterCategory = item.value"
          >{{ item.label }}</button>
        </div>
        <p class="count-text">{{ $t('guild.posts.postCount', { n: filteredPosts.length }) }}</p>
      </div>

      <div v-if="loading && posts.length === 0" class="post-list">
        <div v-for="i in 4" :key="`skeleton-${i}`" class="post-card skeleton-card">
          <div class="skeleton-line w-30" />
          <div class="skeleton-line w-85" />
          <div class="skeleton-line w-65" />
          <div class="skeleton-line w-55" />
        </div>
      </div>
      <div v-else-if="noPermission" class="hint">{{ $t('guild.posts.noPermission') }}</div>
      <div v-else-if="posts.length === 0" class="hint">{{ $t('guild.posts.empty') }}</div>

      <div v-else class="post-list">
        <button
          v-for="post in filteredPosts"
          :key="post.id"
          class="post-card"
          @click="viewPost(post.id)"
        >
          <div class="post-head">
            <span class="category-tag">{{ post.category || $t('guild.posts.categoryOther') }}</span>
            <span class="time">{{ formatDate(post.created_at) }}</span>
          </div>
          <div v-if="formatLocation(post.region, post.address)" class="location-row">
            <i class="ri-map-pin-2-fill" />
            <span>{{ formatLocation(post.region, post.address) }}</span>
          </div>
          <h3 class="title">{{ post.title }}</h3>
          <p class="excerpt">{{ stripHtml(post.content).slice(0, 120) }}</p>
          <div class="foot">
            <div class="author">
              <CachedImage v-if="post.author_avatar" :src="resolveApiUrl(post.author_avatar)" alt="" class="author-image" />
              <i v-else class="ri-user-3-fill" />
              <span :style="{ color: post.author_name_color || undefined, fontWeight: post.author_name_bold ? '700' : undefined }">{{ post.author_name }}</span>
            </div>
            <div class="stats">
              <span><i class="ri-eye-line" /> {{ post.view_count }}</span>
              <span><i class="ri-chat-3-line" /> {{ post.comment_count }}</span>
            </div>
          </div>
        </button>

        <div v-if="filteredPosts.length === 0" class="hint in-list">{{ $t('guild.posts.noMatch') }}</div>
      </div>

      <MobilePagination
        v-if="posts.length > 0 && !searchKeyword.trim()"
        :model-value="currentPage"
        :total-pages="totalPages"
        :disabled="loading || switchingPage"
        @change="onPageChange"
      />
    </div>
  </div>
</template>

<style scoped>
.guild-posts-page .sub-body { padding-top: 8px; }
.switch-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 10px;
}
.switch-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 10px;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.switch-btn i { margin-right: 4px; }
.switch-btn.active {
  background: var(--color-primary);
  color: var(--text-light);
  border-color: var(--color-primary);
}
.toolbar-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 12px;
  margin-bottom: 10px;
}
.guild-name {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}
.sort-row {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.sort-btn {
  border: none;
  background: var(--color-panel-bg);
  color: var(--text-dark);
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
}
.sort-btn.active {
  background: var(--color-primary);
  color: var(--text-light);
}
.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--input-bg);
  border-radius: 16px;
  padding: 8px 10px;
  margin-bottom: 8px;
}
.search-bar input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
}
.category-row {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  margin-bottom: 8px;
}
.cat-btn {
  flex-shrink: 0;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--text-dark);
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
}
.cat-btn.active {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-light);
}
.count-text {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.hint {
  text-align: center;
  color: var(--color-text-secondary);
  padding: 32px 0;
}
.hint.in-list {
  padding: 8px 0 0;
}
.post-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.post-card {
  border: none;
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  border-radius: var(--radius-md);
  text-align: left;
  padding: 12px;
}
.post-head {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
  gap: 8px;
}
.category-tag {
  font-size: 11px;
  color: var(--tag-text);
  background: var(--tag-bg);
  border-radius: 8px;
  padding: 2px 8px;
}
.time {
  font-size: 11px;
  color: var(--color-text-secondary);
}
.title {
  font-size: 15px;
  margin-bottom: 6px;
}
.excerpt {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.55;
  margin-bottom: 8px;
}

.location-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  margin-bottom: 8px;
  border-radius: 999px;
  border: 1px solid rgba(75, 54, 33, 0.1);
  background: var(--color-primary-light);
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
}

.location-row i {
  color: var(--color-primary);
  font-size: 14px;
}

.foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.author {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.author :deep(.author-image),
.author i {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  object-fit: cover;
  color: var(--color-secondary);
}
.author span {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.stats {
  font-size: 12px;
  color: var(--color-text-secondary);
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}
.skeleton-card {
  pointer-events: none;
}

.skeleton-line {
  height: 11px;
  border-radius: 6px;
  margin-bottom: 8px;
  background: linear-gradient(90deg, #f2ece6 0%, #ffffff 50%, #f2ece6 100%);
  background-size: 220% 100%;
  animation: skeletonShimmer 1.1s linear infinite;
}

.skeleton-line.w-30 { width: 30%; }
.skeleton-line.w-55 { width: 55%; }
.skeleton-line.w-65 { width: 65%; }
.skeleton-line.w-85 { width: 85%; }

@keyframes skeletonShimmer {
  from { background-position: 200% 0; }
  to { background-position: -20% 0; }
}
</style>
