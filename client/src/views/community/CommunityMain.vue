<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listPosts, type PostWithAuthor, type ListPostsParams, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { listGuilds, type Guild } from '@/api/guild'
import EventCalendar from '@/components/EventCalendar.vue'

const router = useRouter()
const mounted = ref(false)
const activeTab = ref('posts')
const loading = ref(false)
const posts = ref<PostWithAuthor[]>([])
const guilds = ref<Guild[]>([])
const total = ref(0)

const sortBy = ref<'created_at' | 'view_count' | 'like_count'>('created_at')
const filterGuildId = ref<number>()
const filterCategory = ref<PostCategory | ''>('')

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadPosts()
  await loadGuilds()
})

const tabs = [
  { id: 'guilds', icon: 'ri-team-line', label: '公会' },
  { id: 'posts', icon: 'ri-article-line', label: '帖子' },
]

async function loadPosts() {
  loading.value = true
  try {
    const params: ListPostsParams = {
      page: 1,
      page_size: 20,
      sort: sortBy.value,
      order: 'desc',
      status: 'published',
    }
    if (filterGuildId.value) {
      params.guild_id = filterGuildId.value
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    }
    const res = await listPosts(params)
    posts.value = res.posts || []
    total.value = res.total
  } catch (error) {
    console.error('加载帖子失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadGuilds() {
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } catch (error) {
    console.error('加载公会失败:', error)
  }
}

function goToPost(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function goToCreatePost() {
  router.push({ name: 'post-create' })
}

function goToMyPosts() {
  router.push({ name: 'my-posts' })
}

function goToGuild(id: number) {
  router.push({ name: 'guild-detail', params: { id } })
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

async function changeSortBy(sort: 'created_at' | 'view_count' | 'like_count') {
  sortBy.value = sort
  await loadPosts()
}

async function changeGuildFilter(guildId?: number) {
  filterGuildId.value = guildId
  await loadPosts()
}

async function changeCategoryFilter(category: PostCategory | '') {
  filterCategory.value = category
  await loadPosts()
}

// 获取分区标签
function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : '其他'
}

function getCategoryIcon(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.icon : 'ri-more-line'
}

// 去除HTML标签，只保留纯文本
function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}
</script>

<template>
  <div class="community-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <h1 class="page-title">社区广场</h1>
      <div class="header-actions">
        <button class="my-posts-btn" @click="goToMyPosts">
          <i class="ri-file-list-3-line"></i>
          我的帖子
        </button>
        <button class="create-btn" @click="goToCreatePost">
          <i class="ri-add-line"></i>
          发布帖子
        </button>
      </div>
    </div>

    <div class="tab-container anim-item" style="--delay: 1">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-item"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        <i :class="tab.icon"></i>
        <span>{{ tab.label }}</span>
      </div>
    </div>

    <!-- 活动日历 -->
    <EventCalendar class="anim-item" style="--delay: 2" />

    <!-- 公会列表 -->
    <div v-if="activeTab === 'guilds'" class="guild-list anim-item" style="--delay: 2">
      <div v-for="guild in guilds" :key="guild.id" class="guild-card" @click="goToGuild(guild.id)">
        <div class="guild-avatar">{{ guild.name.charAt(0) }}</div>
        <div class="guild-info">
          <h3>{{ guild.name }}</h3>
          <p>{{ guild.description || '暂无描述' }}</p>
          <div class="guild-stats">
            <span><i class="ri-group-line"></i> {{ guild.member_count }} 成员</span>
            <span><i class="ri-article-line"></i> {{ guild.story_count }} 剧情</span>
          </div>
        </div>
      </div>
      <div v-if="guilds.length === 0" class="empty-state">
        <i class="ri-team-line"></i>
        <p>暂无公会</p>
      </div>
    </div>

    <!-- 帖子列表 -->
    <div v-if="activeTab === 'posts'" class="posts-section">
      <!-- 分区筛选 -->
      <div class="category-filter anim-item" style="--delay: 2">
        <button
          :class="{ active: filterCategory === '' }"
          @click="changeCategoryFilter('')"
        >
          全部
        </button>
        <button
          v-for="cat in POST_CATEGORIES"
          :key="cat.value"
          :class="{ active: filterCategory === cat.value }"
          @click="changeCategoryFilter(cat.value)"
        >
          <i :class="cat.icon"></i>
          {{ cat.label }}
        </button>
      </div>

      <div class="filter-bar anim-item" style="--delay: 3">
        <div class="sort-buttons">
          <button
            :class="{ active: sortBy === 'created_at' }"
            @click="changeSortBy('created_at')"
          >
            最新
          </button>
          <button
            :class="{ active: sortBy === 'like_count' }"
            @click="changeSortBy('like_count')"
          >
            最热
          </button>
          <button
            :class="{ active: sortBy === 'view_count' }"
            @click="changeSortBy('view_count')"
          >
            浏览最多
          </button>
        </div>
        <select v-model="filterGuildId" @change="loadPosts" class="guild-filter">
          <option :value="undefined">全部公会</option>
          <option v-for="guild in guilds" :key="guild.id" :value="guild.id">
            {{ guild.name }}
          </option>
        </select>
      </div>

      <div v-if="loading" class="loading">加载中...</div>

      <div v-else class="post-list anim-item" style="--delay: 3">
        <div
          v-for="post in posts"
          :key="post.id"
          class="post-card"
          @click="goToPost(post.id)"
        >
          <div class="post-header">
            <div class="author-info">
              <div class="author-avatar">{{ post.author_name.charAt(0) }}</div>
              <div>
                <div class="author-name">{{ post.author_name }}</div>
                <div class="post-time">{{ formatDate(post.created_at) }}</div>
              </div>
            </div>
            <span class="category-badge">
              <i :class="getCategoryIcon(post.category)"></i>
              {{ getCategoryLabel(post.category) }}
            </span>
          </div>
          <h3 class="post-title">{{ post.title }}</h3>
          <div class="post-content">{{ stripHtml(post.content).substring(0, 150) }}...</div>
          <div class="post-stats">
            <span><i class="ri-eye-line"></i> {{ post.view_count }}</span>
            <span><i class="ri-heart-line"></i> {{ post.like_count }}</span>
            <span><i class="ri-chat-3-line"></i> {{ post.comment_count }}</span>
          </div>
        </div>

        <div v-if="posts.length === 0" class="empty-state">
          <i class="ri-article-line"></i>
          <p>暂无帖子</p>
          <button class="create-btn-secondary" @click="goToCreatePost">
            发布第一篇帖子
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.community-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title {
  font-size: 42px;
  color: #4B3621;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.my-posts-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: #fff;
  color: #4B3621;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.my-posts-btn:hover {
  background: #F5EFE7;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.create-btn:hover {
  background: #6B3528;
  transform: translateY(-2px);
}

.tab-container {
  background: #4B3621;
  border-radius: 16px;
  padding: 8px;
  display: flex;
  gap: 8px;
}

.tab-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
  color: #EED9C4;
  transition: all 0.3s;
}

.tab-item.active {
  background: #EED9C4;
  color: #4B3621;
}

.guild-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.guild-card {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
  cursor: pointer;
  transition: all 0.3s;
}

.guild-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(75,54,33,0.1);
}

.guild-avatar {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
}

.guild-info { flex: 1; }
.guild-info h3 { font-size: 18px; color: #2C1810; margin-bottom: 4px; }
.guild-info p { font-size: 14px; color: #8D7B68; margin-bottom: 8px; }

.guild-stats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #8D7B68;
}

.posts-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.category-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.category-filter button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: #F5EFE7;
  border: 2px solid transparent;
  border-radius: 8px;
  color: #4B3621;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.category-filter button:hover {
  background: #E5D4C1;
}

.category-filter button.active {
  background: #804030;
  color: #fff;
}

.category-filter button i {
  font-size: 14px;
}

.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.sort-buttons {
  display: flex;
  gap: 8px;
}

.sort-buttons button {
  padding: 8px 16px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.sort-buttons button.active {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.guild-filter {
  padding: 8px 16px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  color: #4B3621;
  font-size: 14px;
  cursor: pointer;
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.post-card {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
  cursor: pointer;
  transition: all 0.3s;
}

.post-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(75,54,33,0.1);
}

.post-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.category-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #804030;
}

.category-badge i {
  font-size: 12px;
}

.author-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.author-avatar {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: #fff;
}

.author-name {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
}

.post-time {
  font-size: 12px;
  color: #8D7B68;
}

.post-title {
  font-size: 20px;
  font-weight: 700;
  color: #2C1810;
  margin-bottom: 8px;
}

.post-content {
  font-size: 14px;
  color: #8D7B68;
  line-height: 1.6;
  margin-bottom: 12px;
}

.post-stats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #8D7B68;
}

.post-stats span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #8D7B68;
}

.empty-state i {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.empty-state p {
  font-size: 16px;
  margin-bottom: 16px;
}

.create-btn-secondary {
  padding: 10px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #8D7B68;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
