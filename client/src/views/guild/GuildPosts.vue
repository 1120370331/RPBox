<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listPosts, type PostWithAuthor, type ListPostsParams, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { getGuild, type Guild } from '@/api/guild'
import { resolveApiUrl } from '@/api/item'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import { buildNameStyle } from '@/utils/userNameStyle'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const guildId = Number(route.params.id)

const loading = ref(true)
const guild = ref<Guild | null>(null)
const posts = ref<PostWithAuthor[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 12
const noPermission = ref(false)

// 筛选条件
const searchKeyword = ref('')
const filterCategory = ref<PostCategory | ''>('')
const sortBy = ref<'created_at' | 'view_count' | 'like_count'>('created_at')

// 筛选后的帖子列表
const filteredPosts = computed(() => {
  if (!searchKeyword.value) return posts.value
  const keyword = searchKeyword.value.toLowerCase()
  return posts.value.filter(post =>
    post.title.toLowerCase().includes(keyword) ||
    post.content.toLowerCase().includes(keyword)
  )
})

async function loadGuild() {
  try {
    const res = await getGuild(guildId)
    guild.value = res.guild
  } catch (error) {
    console.error('加载公会信息失败:', error)
  }
}

async function loadPosts() {
  loading.value = true
  noPermission.value = false
  try {
    const params: ListPostsParams = {
      page: currentPage.value,
      page_size: pageSize,
      sort: sortBy.value,
      order: 'desc',
      status: 'published',
      guild_id: guildId
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    }
    const res = await listPosts(params)
    posts.value = res.posts || []
    total.value = res.total
  } catch (error: any) {
    console.error('加载帖子失败:', error)
    if (error.response?.status === 403 || error.message?.includes('403') || error.message?.includes('无权')) {
      noPermission.value = true
    }
  } finally {
    loading.value = false
  }
}

function viewPost(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function goBack() {
  router.push({ name: 'guild-detail', params: { id: guildId } })
}

function goToStories() {
  router.push({ name: 'guild-stories', params: { id: guildId } })
}

function changePage(page: number) {
  currentPage.value = page
  loadPosts()
}

async function changeCategory(category: PostCategory | '') {
  filterCategory.value = category
  currentPage.value = 1
  await loadPosts()
}

async function changeSort() {
  currentPage.value = 1
  await loadPosts()
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))

  if (hours < 1) return t('guild.posts.timeJustNow')
  if (hours < 24) return t('guild.posts.timeHoursAgo', { n: hours })
  const days = Math.floor(hours / 24)
  if (days < 7) return t('guild.posts.timeDaysAgo', { n: days })
  return date.toLocaleDateString()
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : t('guild.posts.categoryOther')
}

function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}

onMounted(async () => {
  await loadGuild()
  await loadPosts()
})
</script>

<template>
  <div class="guild-posts-page">
    <!-- 头部 -->
    <div class="page-header">
      <button class="back-btn" @click="goBack">
        <i class="ri-arrow-left-line"></i>
        {{ t('guild.manage.back') }}
      </button>
      <div class="header-content">
        <h1 class="page-title">{{ guild?.name }} - {{ t('guild.posts.title') }}</h1>
        <p class="page-desc">{{ t('guild.posts.pageDesc') }}</p>
      </div>
    </div>

    <!-- 快速跳转导航 -->
    <div class="quick-nav">
      <button class="nav-btn active">
        <i class="ri-article-line"></i>
        {{ t('guild.posts.guildPosts') }}
      </button>
      <button class="nav-btn" @click="goToStories">
        <i class="ri-book-2-line"></i>
        {{ t('guild.posts.guildStories') }}
      </button>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="category-filter">
        <button
          :class="{ active: filterCategory === '' }"
          @click="changeCategory('')"
        >{{ t('guild.posts.all') }}</button>
        <button
          v-for="cat in POST_CATEGORIES"
          :key="cat.value"
          :class="{ active: filterCategory === cat.value }"
          @click="changeCategory(cat.value)"
        >{{ cat.label }}</button>
      </div>
      <div class="sort-select">
        <span class="sort-label">{{ t('guild.posts.sort') }}:</span>
        <select v-model="sortBy" @change="changeSort">
          <option value="created_at">{{ t('guild.posts.sortLatest') }}</option>
          <option value="like_count">{{ t('guild.posts.sortPopular') }}</option>
          <option value="view_count">{{ t('guild.posts.sortViews') }}</option>
        </select>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-input">
        <i class="ri-search-line"></i>
        <input
          v-model="searchKeyword"
          type="text"
          :placeholder="t('guild.posts.searchPlaceholder')"
        />
      </div>
      <div class="post-count">
        {{ t('guild.posts.postCount', { n: filteredPosts.length }) }}
      </div>
    </div>

    <!-- 帖子列表 -->
    <div v-if="loading" class="loading">
      <i class="ri-loader-4-line rotating"></i>
      {{ t('guild.posts.loading') }}
    </div>

    <REmpty v-else-if="noPermission"
      icon="ri-lock-line"
      :message="t('guild.posts.noPermission')"
      :description="t('guild.posts.noPermissionDesc')"
    />

    <REmpty v-else-if="posts.length === 0"
      icon="ri-article-line"
      :message="t('guild.posts.empty')"
      :description="t('guild.posts.emptyDesc')"
    />

    <div v-else class="posts-list">
      <div
        v-for="post in filteredPosts"
        :key="post.id"
        class="post-card"
        @click="viewPost(post.id)"
      >
        <div class="post-header">
          <div class="header-tags">
            <span class="category-tag">{{ getCategoryLabel(post.category) }}</span>
            <span v-if="post.is_pinned" class="pinned-tag">{{ t('guild.posts.pinned') }}</span>
            <span v-if="post.is_featured" class="featured-tag">{{ t('guild.posts.featured') }}</span>
          </div>
        </div>
        <h3 class="post-title">{{ post.title }}</h3>
        <p class="post-excerpt">{{ stripHtml(post.content).substring(0, 100) }}...</p>
        <div class="post-footer">
          <div class="author-info">
            <div class="author-avatar">
              <img v-if="post.author_avatar" :src="resolveApiUrl(post.author_avatar)" alt="" />
              <span v-else>{{ post.author_name?.charAt(0) || 'U' }}</span>
            </div>
            <span class="author-name" :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</span>
            <span class="post-time">{{ formatDate(post.created_at) }}</span>
          </div>
          <div class="post-stats">
            <span class="stat-item">
              <i class="ri-eye-line"></i>
              {{ post.view_count }}
            </span>
            <span class="stat-item">
              <i class="ri-chat-3-line"></i>
              {{ post.comment_count }}
            </span>
          </div>
        </div>
      </div>

      <REmpty v-if="filteredPosts.length === 0 && searchKeyword"
        icon="ri-search-line"
        :message="t('guild.posts.noMatch')"
        :description="t('guild.posts.noMatchDesc', { keyword: searchKeyword })"
      />
    </div>

    <!-- 分页 -->
    <div v-if="posts.length > 0 && !searchKeyword" class="pagination">
      <button
        class="page-btn"
        :disabled="currentPage === 1"
        @click="changePage(currentPage - 1)"
      >
        <i class="ri-arrow-left-s-line"></i>
      </button>
      <span class="page-info">
        {{ t('guild.posts.pageInfo', { current: currentPage, total: Math.ceil(total / pageSize) }) }}
      </span>
      <button
        class="page-btn"
        :disabled="currentPage >= Math.ceil(total / pageSize)"
        @click="changePage(currentPage + 1)"
      >
        <i class="ri-arrow-right-s-line"></i>
      </button>
    </div>
  </div>
</template>

<style scoped>
/* 日式极简 + 混合圆角设计系统 */
.guild-posts-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 48px 24px;
  background: #EED9C4;
  min-height: 100vh;
  animation: fadeIn 0.5s ease-out;
}

/* 渐入动画 */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 头部 */
.page-header {
  display: flex;
  flex-direction: column;
  gap: 24px;
  margin-bottom: 40px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: none;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.3s ease;
  padding: 0;
}

.back-btn i {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px solid #E5D4C1;
  background: white;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.back-btn:hover {
  color: #804030;
}

.back-btn:hover i {
  border-color: #B87333;
  background: #F5EFE7;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.page-title {
  font-size: 2.5rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: #2C1810;
  margin: 0;
  line-height: 1.2;
}

.page-desc {
  font-size: 14px;
  font-weight: 300;
  color: #8D7B68;
  margin: 0;
  padding-left: 8px;
  border-left: 2px solid #D4A373;
}

/* 快速跳转导航 */
.quick-nav {
  display: flex;
  gap: 12px;
  margin-bottom: 32px;
}

.nav-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: white;
  border: 1px solid #E5D4C1;
  border-radius: 2px;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.nav-btn:hover {
  border-color: #B87333;
  color: #804030;
  background: #F5EFE7;
}

.nav-btn.active {
  background: #804030;
  color: white;
  border-color: #804030;
}

.nav-btn i {
  font-size: 16px;
}

/* 筛选栏 */
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  margin-bottom: 40px;
  padding: 16px;
  background: white;
  border-radius: 24px 2px 24px 2px; /* leaf-shape-reverse */
  box-shadow: 0 4px 20px -2px rgba(44, 24, 16, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.category-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.category-filter button {
  padding: 8px 20px;
  background: transparent;
  border: 1px solid #E5D4C1;
  border-radius: 2px;
  color: #8D7B68;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.3s ease;
}

.category-filter button:hover {
  border-color: #B87333;
  color: #B87333;
}

.category-filter button.active {
  background: #2C1810;
  color: white;
  border-color: #2C1810;
  border-radius: 12px 2px 12px 2px; /* 对角圆角 */
  box-shadow: 0 2px 8px rgba(44, 24, 16, 0.2);
}

.sort-select {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid #F5EFE7;
  width: 100%;
}

@media (min-width: 1024px) {
  .sort-select {
    border-top: none;
    padding-top: 0;
    width: auto;
  }
}

.sort-label {
  font-size: 11px;
  color: #B87333;
  font-family: monospace;
  font-weight: 600;
}

.sort-select select {
  background: #F5EFE7;
  border: none;
  color: #2C1810;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  outline: none;
  padding: 8px 32px 8px 12px;
  border-radius: 8px;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding: 20px;
  background: white;
  border-radius: 2px 24px 2px 24px; /* leaf-shape */
  box-shadow: 0 4px 20px -2px rgba(44, 24, 16, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.search-input {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid #E5D4C1;
  border-radius: 2px 24px 2px 24px; /* leaf-shape */
  transition: all 0.3s ease;
}

.search-input:focus-within {
  background: white;
  border-color: #B87333;
}

.search-input i {
  font-size: 18px;
  color: #D4A373;
}

.search-input input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  font-size: 14px;
  color: #2C1810;
  font-weight: 400;
}

.search-input input::placeholder {
  color: #D4A373;
}

.post-count {
  font-size: 11px;
  color: #B87333;
  white-space: nowrap;
  font-weight: 600;
  font-family: monospace;
  background: #F5EFE7;
  padding: 6px 12px;
  border-radius: 999px;
}

/* 加载状态 */
.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 60px;
  color: #8D7B68;
  font-size: 16px;
}

.rotating {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 帖子列表 */
.posts-list {
  display: grid;
  grid-template-columns: repeat(1, 1fr);
  gap: 24px;
}

@media (min-width: 768px) {
  .posts-list {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1024px) {
  .posts-list {
    grid-template-columns: repeat(3, 1fr);
  }
}

.post-card {
  position: relative;
  padding: 24px;
  background: white;
  border: 1px solid rgba(229, 212, 193, 0.5);
  border-radius: 2px 24px 2px 24px; /* leaf-shape */
  cursor: pointer;
  transition: transform 0.4s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.4s ease;
  box-shadow: 0 4px 20px -2px rgba(44, 24, 16, 0.08);
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  animation: fadeInUp 0.6s ease-out backwards;
}

/* 交错动画延迟 */
.post-card:nth-child(1) { animation-delay: 0.1s; }
.post-card:nth-child(2) { animation-delay: 0.15s; }
.post-card:nth-child(3) { animation-delay: 0.2s; }
.post-card:nth-child(4) { animation-delay: 0.25s; }
.post-card:nth-child(5) { animation-delay: 0.3s; }
.post-card:nth-child(6) { animation-delay: 0.35s; }
.post-card:nth-child(7) { animation-delay: 0.4s; }
.post-card:nth-child(8) { animation-delay: 0.45s; }
.post-card:nth-child(9) { animation-delay: 0.5s; }
.post-card:nth-child(10) { animation-delay: 0.55s; }
.post-card:nth-child(11) { animation-delay: 0.6s; }
.post-card:nth-child(12) { animation-delay: 0.65s; }
.post-card:nth-child(n+13) { animation-delay: 0.7s; }

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.post-card::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 96px;
  height: 96px;
  background: linear-gradient(to bottom right, #F5EFE7, transparent);
  opacity: 0.3;
  border-radius: 0 0 0 100%;
  pointer-events: none;
}

.post-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 20px 40px -4px rgba(44, 24, 16, 0.12);
}

.post-header {
  position: relative;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
  z-index: 1;
}

.header-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.category-tag {
  display: inline-block;
  padding: 4px 12px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 2px;
  font-size: 11px;
  font-weight: 700;
  color: #804030;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pinned-tag {
  display: inline-block;
  padding: 4px 12px;
  background: #804030;
  color: white;
  border-radius: 2px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.featured-tag {
  display: inline-block;
  padding: 4px 12px;
  background: linear-gradient(135deg, #E6A23C, #D97706);
  color: white;
  border-radius: 2px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.post-title {
  font-size: 20px;
  font-weight: 700;
  color: #2C1810;
  margin: 0 0 12px 0;
  line-height: 1.4;
  transition: color 0.3s ease;
}

.post-card:hover .post-title {
  color: #804030;
}

.post-excerpt {
  font-size: 14px;
  color: #8D7B68;
  line-height: 1.6;
  margin: 0 0 24px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-footer {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid #F5EFE7;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.author-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.author-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #804030);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  border: 1px solid #E5D4C1;
}

.author-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.author-name {
  font-size: 12px;
  font-weight: 700;
  color: #2C1810;
}

.post-time {
  font-size: 10px;
  color: #B87333;
}

.post-stats {
  display: flex;
  gap: 12px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #D4A373;
  font-family: monospace;
}

.stat-item i {
  font-size: 14px;
}

/* 分页 */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 48px;
  padding: 20px;
}

.page-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid #E5D4C1;
  border-radius: 50%;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
}

.page-btn:hover:not(:disabled) {
  background: #2C1810;
  color: white;
  border-color: #2C1810;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 18px;
  color: #2C1810;
  font-weight: 400;
}
</style>
