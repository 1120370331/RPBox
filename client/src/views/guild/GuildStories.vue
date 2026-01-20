<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listStories, type Story, type StoryFilterParams } from '@/api/story'
import { getGuild, listGuildMembers, removeStoryFromGuild, type Guild, type GuildStoryWithUploader } from '@/api/guild'
import { listGuildTags, listTags, type Tag } from '@/api/tag'
import { useDialog } from '@/composables/useDialog'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'

const { confirm, alert } = useDialog()

const route = useRoute()
const router = useRouter()
const guildId = Number(route.params.id)

const loading = ref(true)
const guild = ref<Guild | null>(null)
const stories = ref<Story[]>([])
const members = ref<GuildMember[]>([])
const searchKeyword = ref('')
const noPermission = ref(false)
const myRole = ref<string>('')

// 筛选相关
const tags = ref<Tag[]>([])
const selectedTagIds = ref<number[]>([])
const startDate = ref('')
const endDate = ref('')
const sortBy = ref<'created_at' | 'updated_at' | 'start_time'>('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')
const selectedUploaderId = ref<number | undefined>(undefined)
const showFilter = ref(false)

// 筛选后的剧情列表
const filteredStories = computed(() => {
  if (!searchKeyword.value) return stories.value
  const keyword = searchKeyword.value.toLowerCase()
  return stories.value.filter((story: GuildStoryWithUploader) =>
    story.title.toLowerCase().includes(keyword) ||
    story.description?.toLowerCase().includes(keyword)
  )
})

// 检查是否是管理员或会长
const isAdmin = computed(() => {
  return myRole.value === 'owner' || myRole.value === 'admin'
})

async function loadGuild() {
  try {
    const res = await getGuild(guildId)
    guild.value = res.guild
    myRole.value = res.my_role || ''
  } catch (error) {
    console.error('加载公会信息失败:', error)
  }
}

async function loadMembers() {
  try {
    const res = await listGuildMembers(guildId)
    members.value = res.members || []
  } catch (error) {
    console.error('加载成员列表失败:', error)
  }
}

async function loadStories() {
  loading.value = true
  noPermission.value = false
  try {
    const params: StoryFilterParams = {
      guild_id: String(guildId),
      tag_ids: selectedTagIds.value.length > 0 ? selectedTagIds.value.join(',') : undefined,
      start_date: startDate.value || undefined,
      end_date: endDate.value || undefined,
      sort: sortBy.value,
      order: sortOrder.value,
      added_by: selectedUploaderId.value
    }
    const res = await listStories(params)
    stories.value = res.stories || []
  } catch (error: any) {
    console.error('加载剧情失败:', error)
    if (error.response?.status === 403 || error.message?.includes('403') || error.message?.includes('无权')) {
      noPermission.value = true
    }
  } finally {
    loading.value = false
  }
}

function viewStory(id: number) {
  router.push({
    name: 'story-detail',
    params: { id },
    query: { from: 'guild', guildId: String(guildId) }
  })
}

function goBack() {
  router.push({ name: 'guild-detail', params: { id: guildId } })
}

function goToPosts() {
  router.push({ name: 'guild-posts', params: { id: guildId } })
}

async function loadTags() {
  try {
    const results = await Promise.allSettled([
      listTags('story'),
      listGuildTags(guildId)
    ])

    const merged = new Map<number, Tag>()
    if (results[0].status === 'fulfilled') {
      for (const tag of results[0].value.tags || []) {
        merged.set(tag.id, tag)
      }
    } else {
      console.error('加载预设标签失败:', results[0].reason)
    }

    if (results[1].status === 'fulfilled') {
      for (const tag of results[1].value.tags || []) {
        merged.set(tag.id, tag)
      }
    } else {
      console.error('加载公会标签失败:', results[1].reason)
    }

    tags.value = Array.from(merged.values())
  } catch (error) {
    console.error('加载标签失败:', error)
  }
}

function getStoryTags(story: Story): string[] {
  if (!story.tags) return []
  return story.tags.split(',').map(t => t.trim()).filter(Boolean)
}

const tagMap = computed(() => {
  const map = new Map<string, Tag>()
  for (const tag of tags.value) {
    map.set(tag.name, tag)
  }
  return map
})

function getTagChips(story: Story): { name: string; color?: string }[] {
  if (story.tag_list && story.tag_list.length > 0) {
    return story.tag_list.map(tag => ({
      name: tag.name,
      color: tag.color || undefined
    }))
  }
  return getStoryTags(story).map((name) => {
    const tag = tagMap.value.get(name)
    return { name, color: tag?.color }
  })
}

function toggleTag(tagId: number) {
  const index = selectedTagIds.value.indexOf(tagId)
  if (index > -1) {
    selectedTagIds.value.splice(index, 1)
  } else {
    selectedTagIds.value.push(tagId)
  }
}

function applyFilter() {
  loadStories()
}

function resetFilter() {
  selectedTagIds.value = []
  startDate.value = ''
  endDate.value = ''
  sortBy.value = 'created_at'
  sortOrder.value = 'desc'
  selectedUploaderId.value = undefined
  loadStories()
}

async function handleRemoveArchive(storyId: number) {
  const confirmed = await confirm({
    title: '取消归档',
    message: '确定要将此剧情从公会归档中移除吗？',
    type: 'warning'
  })
  if (!confirmed) return

  try {
    await removeStoryFromGuild(guildId, storyId)
    stories.value = stories.value.filter(s => s.id !== storyId)
    await alert({
      title: '成功',
      message: '已取消归档',
      type: 'success'
    })
  } catch (e: any) {
    await alert({
      title: '操作失败',
      message: e.message || '取消归档失败',
      type: 'error'
    })
  }
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

function calculateDuration(startTime: string, endTime: string): string {
  if (!startTime || !endTime) return '未知'

  const start = new Date(startTime)
  const end = new Date(endTime)
  const diffMs = end.getTime() - start.getTime()

  if (diffMs < 0) return '未知'

  const diffMinutes = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffDays > 0) {
    const hours = diffHours % 24
    return hours > 0 ? `${diffDays}天${hours}小时` : `${diffDays}天`
  } else if (diffHours > 0) {
    const minutes = diffMinutes % 60
    return minutes > 0 ? `${diffHours}小时${minutes}分钟` : `${diffHours}小时`
  } else if (diffMinutes > 0) {
    return `${diffMinutes}分钟`
  } else {
    return '不到1分钟'
  }
}

onMounted(async () => {
  await loadGuild()
  await loadTags()
  await loadMembers()
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

    <!-- 快速跳转导航 -->
    <div class="quick-nav">
      <button class="nav-btn" @click="goToPosts">
        <i class="ri-article-line"></i>
        公会帖子
      </button>
      <button class="nav-btn active">
        <i class="ri-book-2-line"></i>
        公会剧情
      </button>
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
      <button class="filter-toggle-btn" @click="showFilter = !showFilter">
        <i class="ri-filter-3-line"></i>
        筛选
      </button>
      <div class="story-count">
        共 {{ filteredStories.length }} 个剧情
      </div>
    </div>

    <!-- 筛选面板 -->
    <div v-if="showFilter" class="filter-panel">
      <!-- 标签筛选 -->
      <div class="filter-section">
        <label class="filter-label">标签筛选</label>
        <div class="tag-filters">
          <span
            v-for="tag in tags"
            :key="tag.id"
            class="filter-tag"
            :class="{ active: selectedTagIds.includes(tag.id) }"
            :style="selectedTagIds.includes(tag.id) ? { background: `#${tag.color}`, color: '#fff' } : { borderColor: `#${tag.color}`, color: `#${tag.color}` }"
            @click="toggleTag(tag.id)"
          >
            {{ tag.name }}
          </span>
          <span v-if="tags.length === 0" class="empty-hint">暂无标签</span>
        </div>
      </div>

      <!-- 日期范围 -->
      <div class="filter-section">
        <label class="filter-label">日期范围</label>
        <div class="date-range">
          <input v-model="startDate" type="date" class="date-input" />
          <span class="date-separator">至</span>
          <input v-model="endDate" type="date" class="date-input" />
        </div>
      </div>

      <!-- 排序 -->
      <div class="filter-section">
        <label class="filter-label">排序</label>
        <div class="sort-controls">
          <select v-model="sortBy" class="filter-select">
            <option value="created_at">创建时间</option>
            <option value="updated_at">更新时间</option>
            <option value="start_time">剧情时间</option>
          </select>
          <button
            class="sort-order-btn"
            @click="sortOrder = sortOrder === 'desc' ? 'asc' : 'desc'"
          >
            <i :class="sortOrder === 'desc' ? 'ri-sort-desc' : 'ri-sort-asc'"></i>
          </button>
        </div>
      </div>

      <!-- 上传者筛选 -->
      <div class="filter-section">
        <label class="filter-label">按上传者筛选</label>
        <select v-model="selectedUploaderId" class="filter-select">
          <option :value="undefined">全部成员</option>
          <option v-for="member in members" :key="member.user_id" :value="member.user_id">
            {{ member.username }}
          </option>
        </select>
      </div>

      <!-- 操作按钮 -->
      <div class="filter-actions">
        <button class="filter-btn reset-btn" @click="resetFilter">
          <i class="ri-refresh-line"></i>
          重置
        </button>
        <button class="filter-btn apply-btn" @click="applyFilter">
          <i class="ri-check-line"></i>
          应用筛选
        </button>
      </div>
    </div>

    <!-- 剧情列表 -->
    <div v-if="loading" class="loading">
      <i class="ri-loader-4-line rotating"></i>
      加载中...
    </div>

    <REmpty v-else-if="noPermission"
      icon="ri-lock-line"
      message="无权限查看"
      description="您没有权限查看该公会的剧情归档"
    />

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
        <div v-if="getTagChips(story).length" class="story-tags">
          <span
            v-for="tag in getTagChips(story)"
            :key="tag.name"
            class="story-tag"
            :style="tag.color ? { background: `#${tag.color}20`, color: `#${tag.color}` } : {}"
          >
            {{ tag.name }}
          </span>
        </div>
        <div class="story-meta">
          <span class="meta-item">
            <i class="ri-message-3-line"></i>
            {{ story.entry_count || 0 }} 条记录
          </span>
          <span class="meta-item">
            <i class="ri-time-line"></i>
            {{ calculateDuration(story.start_time, story.end_time) }}
          </span>
          <span class="meta-item uploader">
            <div class="uploader-avatar">
              <img v-if="story.added_by_avatar" :src="story.added_by_avatar" alt="" loading="lazy" />
              <span v-else>{{ story.added_by_username?.charAt(0) || '?' }}</span>
            </div>
            <span class="uploader-name">{{ story.added_by_username }}</span>
          </span>
          <button v-if="isAdmin" class="remove-archive-btn" @click.stop="handleRemoveArchive(story.id)">
            <i class="ri-close-circle-line"></i>
            取消归档
          </button>
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
/* 日式极简 + 混合圆角设计系统 */
.guild-stories-page {
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

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 40px;
}

.search-input {
  position: relative;
  flex: 1;
  max-width: 256px;
}

.search-input input {
  width: 100%;
  background: transparent;
  border: none;
  border-bottom: 2px solid #E5D4C1;
  color: #2C1810;
  padding: 8px 0;
  font-size: 14px;
  outline: none;
  transition: border-color 0.3s ease;
}

.search-input input::placeholder {
  color: rgba(212, 163, 115, 0.7);
}

.search-input input:focus {
  border-bottom-color: #804030;
}

.search-input i {
  position: absolute;
  right: 0;
  top: 8px;
  font-size: 20px;
  color: #D4A373;
  transition: color 0.3s ease;
}

.search-input:hover i {
  color: #804030;
}

.story-count {
  font-size: 11px;
  color: #B87333;
  white-space: nowrap;
  font-weight: 600;
  font-family: monospace;
  background: #F5EFE7;
  padding: 6px 12px;
  border-radius: 999px;
}

/* 筛选按钮 */
.filter-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: white;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.filter-toggle-btn:hover {
  border-color: #B87333;
  color: #804030;
  background: #F5EFE7;
}

.filter-toggle-btn i {
  font-size: 16px;
}

/* 筛选面板 */
.filter-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(44, 24, 16, 0.08);
  margin-bottom: 32px;
  animation: slideDown 0.3s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-label {
  font-size: 13px;
  font-weight: 600;
  color: #804030;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* 标签筛选 */
.tag-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-tag {
  padding: 6px 14px;
  border: 2px solid;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
}

.filter-tag:hover {
  opacity: 0.8;
  transform: translateY(-1px);
}

.filter-tag.active {
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.empty-hint {
  font-size: 13px;
  color: #8D7B68;
  font-style: italic;
}

/* 日期范围 */
.date-range {
  display: flex;
  align-items: center;
  gap: 12px;
}

.date-input {
  flex: 1;
  padding: 10px 14px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #2C1810;
  font-size: 13px;
  font-family: inherit;
  outline: none;
  transition: all 0.2s ease;
}

.date-input:focus {
  border-color: #B87333;
  background: white;
}

.date-separator {
  font-size: 13px;
  color: #8D7B68;
  font-weight: 500;
}

/* 排序控制 */
.sort-controls {
  display: flex;
  gap: 8px;
}

.filter-select {
  flex: 1;
  padding: 10px 14px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #2C1810;
  font-size: 13px;
  font-family: inherit;
  outline: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-select:focus {
  border-color: #B87333;
  background: white;
}

.sort-order-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #804030;
  font-size: 18px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.sort-order-btn:hover {
  border-color: #B87333;
  background: white;
}

/* 筛选操作按钮 */
.filter-actions {
  display: flex;
  gap: 12px;
  padding-top: 8px;
  border-top: 1px solid #F5EFE7;
}

.filter-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.reset-btn {
  background: #F5EFE7;
  color: #8D7B68;
}

.reset-btn:hover {
  background: #E5D4C1;
  color: #804030;
}

.apply-btn {
  background: #804030;
  color: white;
}

.apply-btn:hover {
  background: #6B3626;
}

.filter-btn i {
  font-size: 16px;
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
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.story-card {
  position: relative;
  padding: 24px 32px;
  background: white;
  border-left: 4px solid #804030;
  border-radius: 0 48px 48px 0;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 20px -2px rgba(44, 24, 16, 0.08);
  overflow: hidden;
  animation: slideInLeft 0.6s ease-out backwards;
}

/* 交错动画延迟 */
.story-card:nth-child(1) { animation-delay: 0.1s; }
.story-card:nth-child(2) { animation-delay: 0.2s; }
.story-card:nth-child(3) { animation-delay: 0.3s; }
.story-card:nth-child(4) { animation-delay: 0.4s; }
.story-card:nth-child(5) { animation-delay: 0.5s; }
.story-card:nth-child(6) { animation-delay: 0.6s; }
.story-card:nth-child(7) { animation-delay: 0.7s; }
.story-card:nth-child(8) { animation-delay: 0.8s; }
.story-card:nth-child(n+9) { animation-delay: 0.9s; }

@keyframes slideInLeft {
  from {
    opacity: 0;
    transform: translateX(-30px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.story-card:hover {
  box-shadow: 0 20px 40px -4px rgba(44, 24, 16, 0.12);
}

.story-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.story-title {
  flex: 1;
  font-size: 20px;
  font-weight: 700;
  color: #2C1810;
  margin: 0;
  line-height: 1.4;
}

.story-date {
  font-size: 12px;
  color: #B87333;
  white-space: nowrap;
  font-weight: 600;
}

.story-desc {
  font-size: 14px;
  color: #8D7B68;
  line-height: 1.6;
  margin: 0 0 16px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.story-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 16px;
}

.story-tag {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  background: rgba(184, 115, 51, 0.15);
  color: #B87333;
  letter-spacing: 0.02em;
}

.story-meta {
  display: flex;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid #F5EFE7;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #D4A373;
  font-family: monospace;
}

.meta-item i {
  font-size: 16px;
}

/* 上传者信息 */
.meta-item.uploader {
  margin-left: auto;
}

.uploader-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  overflow: hidden;
}

.uploader-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.uploader-name {
  font-size: 12px;
  color: #8D7B68;
  font-family: inherit;
}

/* 取消归档按钮 */
.remove-archive-btn {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #fff3e0;
  border: 1px solid #ffb74d;
  border-radius: 6px;
  color: #f57c00;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.remove-archive-btn:hover {
  background: #ffe0b2;
  border-color: #ff9800;
  color: #e65100;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(255, 152, 0, 0.2);
}

.remove-archive-btn i {
  font-size: 14px;
}
</style>
