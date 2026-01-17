<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listStories, type Story } from '@/api/story'
import { getGuild, type Guild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'

const route = useRoute()
const router = useRouter()
const guildId = Number(route.params.id)

const loading = ref(true)
const guild = ref<Guild | null>(null)
const stories = ref<Story[]>([])
const searchKeyword = ref('')

// 筛选后的剧情列表
const filteredStories = computed(() => {
  if (!searchKeyword.value) return stories.value
  const keyword = searchKeyword.value.toLowerCase()
  return stories.value.filter(story =>
    story.title.toLowerCase().includes(keyword) ||
    story.description?.toLowerCase().includes(keyword)
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

async function loadStories() {
  loading.value = true
  try {
    const res = await listStories({ guild_id: String(guildId) })
    stories.value = res.stories || []
  } catch (error) {
    console.error('加载剧情失败:', error)
  } finally {
    loading.value = false
  }
}

function viewStory(id: number) {
  router.push({ name: 'story-detail', params: { id } })
}

function goBack() {
  router.back()
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

onMounted(async () => {
  await loadGuild()
  await loadStories()
})
</script>

<template>
  <div class="guild-stories-page">
    <!-- 头部 -->
    <div class="page-header">
      <button class="back-btn" @click="goBack">
        <i class="ri-arrow-left-line"></i>
        返回
      </button>
      <div class="header-content">
        <h1 class="page-title">{{ guild?.name }} - 剧情归档</h1>
        <p class="page-desc">查看公会成员归档的剧情记录</p>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-input">
        <i class="ri-search-line"></i>
        <input
          v-model="searchKeyword"
          type="text"
          placeholder="搜索剧情标题或描述..."
        />
      </div>
      <div class="story-count">
        共 {{ filteredStories.length }} 个剧情
      </div>
    </div>

    <!-- 剧情列表 -->
    <div v-if="loading" class="loading">
      <i class="ri-loader-4-line rotating"></i>
      加载中...
    </div>

    <REmpty v-else-if="stories.length === 0"
      icon="ri-book-2-line"
      message="暂无剧情归档"
      description="公会管理员可以将剧情归档到公会"
    />

    <div v-else class="stories-list">
      <div
        v-for="story in filteredStories"
        :key="story.id"
        class="story-card"
        @click="viewStory(story.id)"
      >
        <div class="story-header">
          <h3 class="story-title">{{ story.title }}</h3>
          <span class="story-date">{{ formatDate(story.created_at) }}</span>
        </div>
        <p v-if="story.description" class="story-desc">{{ story.description }}</p>
        <div class="story-meta">
          <span class="meta-item">
            <i class="ri-message-3-line"></i>
            {{ story.entry_count || 0 }} 条记录
          </span>
          <span class="meta-item">
            <i class="ri-time-line"></i>
            {{ story.duration || '未知' }}
          </span>
        </div>
      </div>

      <REmpty v-if="filteredStories.length === 0 && searchKeyword"
        icon="ri-search-line"
        message="未找到匹配的剧情"
        :description="`没有找到包含 '${searchKeyword}' 的剧情`"
      />
    </div>
  </div>
</template>

<style scoped>
.guild-stories-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

/* 头部 */
.page-header {
  margin-bottom: 32px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: none;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 16px;
}

.back-btn:hover {
  border-color: #B87333;
  color: #B87333;
  background: rgba(184, 115, 51, 0.05);
}

.header-content {
  text-align: center;
}

.page-title {
  font-size: 28px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 8px 0;
}

.page-desc {
  font-size: 14px;
  color: #8D7B68;
  margin: 0;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.search-input {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #F5EFE7;
  border-radius: 8px;
}

.search-input i {
  font-size: 18px;
  color: #8D7B68;
}

.search-input input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  font-size: 14px;
  color: #2C1810;
}

.search-input input::placeholder {
  color: #8D7B68;
}

.story-count {
  font-size: 14px;
  color: #8D7B68;
  white-space: nowrap;
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

/* 剧情列表 */
.stories-list {
  display: grid;
  gap: 16px;
}

.story-card {
  padding: 20px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.story-card:hover {
  border-color: #B87333;
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.1);
  transform: translateY(-2px);
}

.story-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 12px;
}

.story-title {
  flex: 1;
  font-size: 18px;
  font-weight: 600;
  color: #2C1810;
  margin: 0;
}

.story-date {
  font-size: 13px;
  color: #8D7B68;
  white-space: nowrap;
}

.story-desc {
  font-size: 14px;
  color: #4B3621;
  line-height: 1.6;
  margin: 0 0 12px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.story-meta {
  display: flex;
  gap: 16px;
  padding-top: 12px;
  border-top: 1px solid #F5EFE7;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #8D7B68;
}

.meta-item i {
  font-size: 16px;
}
</style>
