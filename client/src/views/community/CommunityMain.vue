<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { listPosts, listEvents, type PostWithAuthor, type EventItem, type ListPostsParams, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { getGuild, type Guild } from '@/api/guild'
import { getImageUrl } from '@/api/item'
import { buildNameStyle } from '@/utils/userNameStyle'

const router = useRouter()
const route = useRoute()
const mounted = ref(false)
const loading = ref(false)
const posts = ref<PostWithAuthor[]>([])
const total = ref(0)
const pinnedPosts = ref<PostWithAuthor[]>([])

// 活动日历
const events = ref<EventItem[]>([])
const eventsExpanded = ref(true)
const eventsLoading = ref(false)

const sortBy = ref<'created_at' | 'view_count' | 'like_count'>('created_at')
const filterCategory = ref<PostCategory | ''>('')
const searchKeyword = ref('')
const filterGuildId = ref<number | null>(null)
const currentGuild = ref<Guild | null>(null)
const currentPage = ref(1)

onMounted(async () => {
  // 从 URL query 读取公会筛选
  if (route.query.guild_id) {
    filterGuildId.value = Number(route.query.guild_id)
    await loadGuildInfo()
  }

  setTimeout(() => mounted.value = true, 50)
  await Promise.all([loadPosts(), loadEvents(), loadPinnedPosts()])
})

// 监听路由变化
watch(() => route.query.guild_id, async (newGuildId) => {
  if (newGuildId) {
    filterGuildId.value = Number(newGuildId)
    await loadGuildInfo()
  } else {
    filterGuildId.value = null
    currentGuild.value = null
  }
  await Promise.all([loadPosts(), loadPinnedPosts()])
})

async function loadGuildInfo() {
  if (!filterGuildId.value) return
  try {
    const res = await getGuild(filterGuildId.value)
    currentGuild.value = res.guild
  } catch (error) {
    console.error('加载公会信息失败:', error)
  }
}

async function loadEvents() {
  eventsLoading.value = true
  try {
    const res = await listEvents()
    events.value = res.events || []
  } catch (error) {
    console.error('加载活动失败:', error)
  } finally {
    eventsLoading.value = false
  }
}

async function loadPosts() {
  loading.value = true
  try {
    const params: ListPostsParams = {
      page: currentPage.value,
      page_size: 12,
      sort: sortBy.value,
      order: 'desc',
      status: 'published',
      is_pinned: false,
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    }
    if (filterGuildId.value) {
      params.guild_id = filterGuildId.value
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

async function loadPinnedPosts() {
  try {
    const params: ListPostsParams = {
      page: 1,
      page_size: 10,
      sort: 'created_at',
      order: 'desc',
      status: 'published',
      is_pinned: true,
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    }
    if (filterGuildId.value) {
      params.guild_id = filterGuildId.value
    }
    const res = await listPosts(params)
    pinnedPosts.value = res.posts || []
  } catch (error) {
    console.error('加载置顶公告失败:', error)
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

function goToFavorites() {
  router.push('/library/favorites')
}

function goToHistory() {
  router.push('/library/history')
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))

  if (hours < 1) return '刚刚'
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

async function changeCategoryFilter(category: PostCategory | '') {
  filterCategory.value = category
  currentPage.value = 1
  await Promise.all([loadPosts(), loadPinnedPosts()])
}

function changePage(page: number) {
  currentPage.value = page
  loadPosts()
}

function clearGuildFilter() {
  router.push({ name: 'community' })
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : '其他'
}

function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}

function formatEventTime(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', { month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function formatEventTimeShort(dateStr?: string) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function formatEventMonth(dateStr: string) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', { month: 'short' })
}

function formatEventDay(dateStr: string) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.getDate().toString()
}

function getCategoryClass(category: string) {
  const classMap: Record<string, string> = {
    profile: 'cat-profile',
    guild: 'cat-guild',
    report: 'cat-report',
    novel: 'cat-novel',
    item: 'cat-item',
    event: 'cat-event',
    other: 'cat-other'
  }
  return classMap[category] || 'cat-other'
}

function getEventTypeLabel(event: EventItem) {
  const typeKey = event.event_type || 'other'
  return eventTypeMeta[typeKey]?.label || eventTypeMeta.other.label
}

function resolveEventColor(event: EventItem) {
  const typeKey = event.event_type || 'other'
  return event.event_color || eventTypeMeta[typeKey]?.color || eventTypeMeta.other.color
}

function getEventPillStyle(event: EventItem) {
  return {
    '--pill-color': resolveEventColor(event)
  }
}

// 从内容中提取第一张图片
function extractFirstImage(html: string): string | null {
  const imgMatch = html.match(/<img[^>]+src=["']([^"']+)["']/)
  return imgMatch ? imgMatch[1] : null
}

// 从内容中提取所有图片
function extractAllImages(html: string): string[] {
  const imgRegex = /<img[^>]+src=["']([^"']+)["']/g
  const images: string[] = []
  let match
  while ((match = imgRegex.exec(html)) !== null) {
    images.push(match[1])
  }
  return images
}

// 获取帖子所有图片（优先使用 cover_image，否则从内容提取）
function getPostImages(post: PostWithAuthor): string[] {
  const images: string[] = []
  if (post.cover_image) images.push(post.cover_image)
  const contentImages = extractAllImages(post.content)
  return [...images, ...contentImages]
}

// ========== 日历视图相关 ==========
const currentMonth = ref(new Date())
const calendarView = ref(true) // true: 日历视图, false: 列表视图
const eventFilter = ref<'all' | 'server' | 'guild'>('all')
const expandedDays = ref<Record<string, boolean>>({})

const weekDays = ['日', '一', '二', '三', '四', '五', '六']

const eventTypeMeta: Record<string, { label: string; color: string }> = {
  server: { label: '服务器活动', color: '#804030' },
  guild: { label: '公会活动', color: '#B87333' },
  other: { label: '活动', color: '#D97706' }
}

// 筛选后的活动
const filteredEvents = computed(() => {
  if (eventFilter.value === 'all') return events.value
  return events.value.filter(event => event.event_type === eventFilter.value)
})

function getDateKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// 切换视图模式
function toggleCalendarView() {
  calendarView.value = !calendarView.value
}

function setEventFilter(filter: 'all' | 'server' | 'guild') {
  eventFilter.value = filter
}

function isDayExpanded(dayKey: string) {
  return !!expandedDays.value[dayKey]
}

function toggleDayExpanded(dayKey: string) {
  const next = { ...expandedDays.value }
  if (next[dayKey]) {
    delete next[dayKey]
  } else {
    next[dayKey] = true
  }
  expandedDays.value = next
}

// 获取当月的所有日期
const calendarDays = computed(() => {
  const year = currentMonth.value.getFullYear()
  const month = currentMonth.value.getMonth()

  // 获取当月第一天和最后一天
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)

  // 获取第一天是星期几（0=周日，1=周一...）
  const firstDayOfWeek = firstDay.getDay()

  // 生成日历数组
  const days: Array<{
    key: string
    date: Date
    isCurrentMonth: boolean
    events: EventItem[]
  }> = []

  // 填充上个月的日期
  const prevMonthLastDay = new Date(year, month, 0).getDate()
  for (let i = firstDayOfWeek - 1; i >= 0; i--) {
    const date = new Date(year, month - 1, prevMonthLastDay - i)
    days.push({
      key: getDateKey(date),
      date,
      isCurrentMonth: false,
      events: []
    })
  }

  // 填充当月日期
  for (let i = 1; i <= lastDay.getDate(); i++) {
    const date = new Date(year, month, i)
    const dayEvents = filteredEvents.value.filter(event => {
      if (!event.event_start_time) return false
      const eventStart = new Date(event.event_start_time)
      const eventEnd = event.event_end_time ? new Date(event.event_end_time) : null

      // 将日期设置为当天的开始时间（00:00:00）进行比较
      const currentDay = new Date(year, month, i)
      const startDay = new Date(eventStart.getFullYear(), eventStart.getMonth(), eventStart.getDate())

      if (eventEnd) {
        // 如果有结束时间，检查当前日期是否在活动期间内
        const endDay = new Date(eventEnd.getFullYear(), eventEnd.getMonth(), eventEnd.getDate())
        return currentDay >= startDay && currentDay <= endDay
      } else {
        // 如果没有结束时间，只在开始日期显示
        return currentDay.getTime() === startDay.getTime()
      }
    })
    const sortedEvents = [...dayEvents].sort((a, b) => {
      const aTime = a.event_start_time ? new Date(a.event_start_time).getTime() : 0
      const bTime = b.event_start_time ? new Date(b.event_start_time).getTime() : 0
      return aTime - bTime
    })
    days.push({
      key: getDateKey(date),
      date,
      isCurrentMonth: true,
      events: sortedEvents
    })
  }

  // 填充下个月的日期，补齐到42个格子（6周）
  const remainingDays = 42 - days.length
  for (let i = 1; i <= remainingDays; i++) {
    const date = new Date(year, month + 1, i)
    days.push({
      key: getDateKey(date),
      date,
      isCurrentMonth: false,
      events: []
    })
  }

  return days
})

const monthEvents = computed(() => {
  const year = currentMonth.value.getFullYear()
  const month = currentMonth.value.getMonth()
  return filteredEvents.value.filter(event => {
    if (!event.event_start_time) return false
    const start = new Date(event.event_start_time)
    return start.getFullYear() === year && start.getMonth() === month
  })
})

const calendarStats = computed(() => {
  const monthList = monthEvents.value
  const total = monthList.length
  const dayMap = new Map<string, number>()
  const typeMap = new Map<string, number>()

  monthList.forEach(event => {
    if (!event.event_start_time) return
    const start = new Date(event.event_start_time)
    const dateKey = `${start.getFullYear()}-${String(start.getMonth() + 1).padStart(2, '0')}-${String(start.getDate()).padStart(2, '0')}`
    dayMap.set(dateKey, (dayMap.get(dateKey) || 0) + 1)
    const typeKey = event.event_type || 'other'
    typeMap.set(typeKey, (typeMap.get(typeKey) || 0) + 1)
  })

  let peakLabel = '--'
  let peakCount = 0
  dayMap.forEach((count, dateKey) => {
    if (count > peakCount) {
      peakCount = count
      const [year, month, day] = dateKey.split('-').map(Number)
      peakLabel = new Date(year, month - 1, day).toLocaleString('zh-CN', { month: 'short', day: 'numeric' })
    }
  })

  let focusLabel = '--'
  let focusCount = 0
  typeMap.forEach((count, typeKey) => {
    if (count > focusCount) {
      focusCount = count
      focusLabel = eventTypeMeta[typeKey]?.label || eventTypeMeta.other.label
    }
  })

  return {
    total,
    activeDays: dayMap.size,
    peakLabel,
    peakCount,
    focusLabel,
    focusCount
  }
})

// 切换到上个月
function prevMonth() {
  currentMonth.value = new Date(currentMonth.value.getFullYear(), currentMonth.value.getMonth() - 1)
  expandedDays.value = {}
}

// 切换到下个月
function nextMonth() {
  currentMonth.value = new Date(currentMonth.value.getFullYear(), currentMonth.value.getMonth() + 1)
  expandedDays.value = {}
}

// 回到当前月
function goToToday() {
  currentMonth.value = new Date()
  expandedDays.value = {}
}

// 格式化月份标题
const monthTitle = computed(() => {
  return currentMonth.value.toLocaleString('zh-CN', { year: 'numeric', month: 'long' })
})

// 获取活动样式（使用自定义颜色）
function getEventStyle(event: EventItem) {
  const color = resolveEventColor(event)

  // 将十六进制颜色转换为 RGB
  const hex = color.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)

  // 生成浅色背景（添加透明度）
  const backgroundColor = `rgba(${r}, ${g}, ${b}, 0.15)`

  return {
    backgroundColor,
    color,
  }
}
</script>

<template>
  <div class="community-page" :class="{ 'animate-in': mounted }">
    <!-- Header -->
    <header class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <h1 class="page-title">酒馆布告栏</h1>
        <p class="page-subtitle">"这里汇聚了来自艾泽拉斯各地的故事与委托..."</p>
      </div>
      <div class="header-actions">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          <input v-model="searchKeyword" type="text" placeholder="搜索帖子..." />
        </div>
        <button class="favorites-btn" @click="goToFavorites">
          <i class="ri-bookmark-3-line"></i>
          收藏夹
        </button>
        <button class="history-btn" @click="goToHistory">
          <i class="ri-history-line"></i>
          历史记录
        </button>
        <button class="my-posts-btn" @click="goToMyPosts">
          <i class="ri-file-list-3-line"></i>
          我的帖子
        </button>
        <button class="create-btn" @click="goToCreatePost">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/>
          </svg>
          发布
        </button>
      </div>
    </header>

    <!-- 公会筛选提示 -->
    <div v-if="currentGuild" class="guild-filter-banner anim-item" style="--delay: 1">
      <div class="banner-content">
        <i class="ri-shield-line"></i>
        <span>正在查看「<strong>{{ currentGuild.name }}</strong>」的相关内容</span>
      </div>
      <button class="clear-filter-btn" @click="clearGuildFilter">
        <i class="ri-close-line"></i>
        清除筛选
      </button>
    </div>

    <!-- Filters & Sort -->
    <div class="filter-section anim-item" style="--delay: 1">
      <div class="category-filter">
        <button
          :class="{ active: filterCategory === '' }"
          @click="changeCategoryFilter('')"
        >全部</button>
        <button
          v-for="cat in POST_CATEGORIES"
          :key="cat.value"
          :class="{ active: filterCategory === cat.value }"
          @click="changeCategoryFilter(cat.value)"
        >{{ cat.label }}</button>
      </div>
      <div class="sort-select">
        <span class="sort-label">排序:</span>
        <select v-model="sortBy" @change="loadPosts">
          <option value="created_at">最新发布</option>
          <option value="like_count">热门讨论</option>
          <option value="view_count">最多浏览</option>
        </select>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading anim-item" style="--delay: 2">加载中...</div>

    <template v-else>
      <!-- 置顶帖子区域 -->
      <div v-if="pinnedPosts.length > 0" class="pinned-section anim-item" style="--delay: 2">
        <div class="section-header">
          <i class="ri-pushpin-fill"></i>
          <span>置顶公告</span>
        </div>
        <div class="pinned-list">
          <div
            v-for="post in pinnedPosts"
            :key="post.id"
            class="pinned-item"
            @click="goToPost(post.id)"
          >
            <span class="pinned-tag">置顶</span>
            <span class="pinned-title">{{ post.title }}</span>
            <span class="pinned-time">{{ formatDate(post.created_at) }}</span>
          </div>
        </div>
      </div>

      <!-- 活动日历（可展开收缩） -->
      <div class="events-section anim-item" style="--delay: 2.5">
        <div class="events-header" @click="eventsExpanded = !eventsExpanded">
          <div class="events-title">
            <i class="ri-calendar-event-line"></i>
            <span>近期活动</span>
            <span class="events-count">{{ filteredEvents.length }}</span>
          </div>
          <div class="events-header-actions">
            <button class="view-toggle-btn" @click.stop="toggleCalendarView" :title="calendarView ? '切换到列表视图' : '切换到日历视图'">
              <i :class="calendarView ? 'ri-list-check' : 'ri-calendar-line'"></i>
            </button>
            <i :class="eventsExpanded ? 'ri-arrow-up-s-line' : 'ri-arrow-down-s-line'" class="expand-icon"></i>
          </div>
        </div>
        <div v-show="eventsExpanded" class="events-body">
          <div v-if="eventsLoading" class="events-loading">加载中...</div>
          <div v-else-if="events.length === 0" class="events-empty">暂无近期活动</div>

          <!-- 日历视图 -->
          <div v-else-if="calendarView" class="calendar-shell">
            <div class="calendar-head">
              <div class="calendar-head-left">
                <span class="calendar-kicker">活动日历</span>
                <div class="calendar-title-row">
                  <h2 class="calendar-title">{{ monthTitle }}</h2>
                  <span class="calendar-count">{{ calendarStats.total }} 场</span>
                </div>
                <p class="calendar-subtitle">点击活动查看详情。</p>
              </div>
              <div class="calendar-head-right">
                <div class="calendar-sync">
                  <span class="sync-dot"></span>
                  已同步
                </div>
                <div class="calendar-controls">
                  <button class="calendar-nav-btn" type="button" @click="prevMonth">
                    <i class="ri-arrow-left-s-line"></i>
                  </button>
                  <div class="calendar-month-title">{{ monthTitle }}</div>
                  <button class="calendar-nav-btn" type="button" @click="nextMonth">
                    <i class="ri-arrow-right-s-line"></i>
                  </button>
                  <button class="today-btn" type="button" @click="goToToday">今天</button>
                </div>
              </div>
            </div>

            <div class="calendar-stats">
              <div class="stat-card">
                <div class="stat-label">本月活动</div>
                <div class="stat-value">{{ calendarStats.total }}</div>
                <div class="stat-foot">覆盖 {{ calendarStats.activeDays }} 天</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">高峰日期</div>
                <div class="stat-value">{{ calendarStats.peakLabel }}</div>
                <div class="stat-foot">{{ calendarStats.peakCount || 0 }} 场</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">主要类型</div>
                <div class="stat-value">{{ calendarStats.focusLabel }}</div>
                <div class="stat-foot">{{ calendarStats.focusCount || 0 }} 场</div>
              </div>
            </div>

            <div class="calendar-filters">
              <span class="filter-label">类型筛选</span>
              <div class="filter-chips">
                <button class="filter-chip" :class="{ active: eventFilter === 'all' }" type="button" @click="setEventFilter('all')">全部</button>
                <button class="filter-chip" :class="{ active: eventFilter === 'server' }" type="button" @click="setEventFilter('server')">服务器活动</button>
                <button class="filter-chip" :class="{ active: eventFilter === 'guild' }" type="button" @click="setEventFilter('guild')">公会活动</button>
              </div>
            </div>

            <div class="calendar-board">
              <div class="calendar-weekdays">
                <div v-for="day in weekDays" :key="day">{{ day }}</div>
              </div>
              <div class="calendar-days">
                <div
                  v-for="day in calendarDays"
                  :key="day.key"
                  class="calendar-day"
                  :class="{
                    'other-month': !day.isCurrentMonth,
                    'has-events': day.events.length > 0,
                    'today': day.date.toDateString() === new Date().toDateString()
                  }"
                >
                  <div class="day-header">
                    <span class="day-number">{{ day.date.getDate() }}</span>
                    <span v-if="day.events.length > 0" class="day-count">{{ day.events.length }}场</span>
                  </div>
                  <div class="day-events">
                    <button
                      v-for="event in (isDayExpanded(day.key) ? day.events : day.events.slice(0, 2))"
                      :key="event.id"
                      class="day-pill"
                      type="button"
                      :style="getEventPillStyle(event)"
                      @click="goToPost(event.id)"
                    >
                      <span class="pill-title">{{ event.title }}</span>
                      <span class="pill-time">{{ formatEventTimeShort(event.event_start_time) }}</span>
                    </button>
                    <button
                      v-if="day.events.length > 2"
                      class="day-more"
                      type="button"
                      @click="toggleDayExpanded(day.key)"
                    >
                      {{ isDayExpanded(day.key) ? '收起' : `+${day.events.length - 2}` }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 列表视图 -->
          <div v-else class="events-list">
            <div
              v-for="event in filteredEvents"
              :key="event.id"
              class="event-item"
              @click="goToPost(event.id)"
            >
              <div class="event-date" :style="{ backgroundColor: resolveEventColor(event) }">
                <span class="event-month">{{ formatEventMonth(event.event_start_time) }}</span>
                <span class="event-day">{{ formatEventDay(event.event_start_time) }}</span>
              </div>
              <div class="event-info">
                <h4 class="event-title">{{ event.title }}</h4>
                <div class="event-meta">
                  <span class="event-type" :style="getEventStyle(event)">{{ getEventTypeLabel(event) }}</span>
                  <span class="event-time">{{ formatEventTime(event.event_start_time) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 帖子瀑布流 -->
      <div class="posts-grid anim-item" style="--delay: 3">
        <div
          v-for="post in posts"
          :key="post.id"
          class="post-card standard"
          @click="goToPost(post.id)"
        >
          <div class="card-content">
            <div class="card-tags">
              <span class="category-tag" :class="getCategoryClass(post.category)">
                {{ getCategoryLabel(post.category) }}
              </span>
              <span v-if="post.is_featured" class="featured-tag">
                <i class="ri-star-fill"></i>
                精华
              </span>
            </div>
            <h3 class="post-title">{{ post.title }}</h3>
            <p class="post-excerpt">{{ stripHtml(post.content).substring(0, 100) }}...</p>
            <!-- 封面图 -->
            <div v-if="post.cover_image_url" class="cover-image small">
              <img :src="getImageUrl('post-cover', post.id, { w: 400, q: 80, v: post.cover_image_updated_at || post.updated_at })" alt="" loading="lazy" />
            </div>
            <div class="card-footer">
              <div class="author-info">
                <div class="author-avatar small">
                  <img v-if="post.author_avatar" :src="post.author_avatar" alt="" loading="lazy" />
                  <span v-else>{{ post.author_name?.charAt(0) || 'U' }}</span>
                </div>
                <span class="author-name" :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</span>
              </div>
              <span class="comment-count">
                <i class="ri-chat-3-line"></i>
                {{ post.comment_count }}
              </span>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="posts.length === 0 && pinnedPosts.length === 0" class="empty-state">
          <i class="ri-article-line"></i>
          <p>暂无帖子</p>
          <button class="create-btn" @click="goToCreatePost">
            <i class="ri-add-line"></i>
            发布第一篇帖子
          </button>
        </div>
      </div>
    </template>

    <!-- Pagination -->
    <div v-if="posts.length > 0" class="pagination anim-item" style="--delay: 4">
      <button
        class="page-btn"
        :disabled="currentPage === 1"
        @click="changePage(currentPage - 1)"
      >
        上一页
      </button>
      <span class="page-info">
        第 {{ currentPage }} / {{ Math.ceil(total / 12) }} 页
      </span>
      <button
        class="page-btn"
        :disabled="currentPage >= Math.ceil(total / 12)"
        @click="changePage(currentPage + 1)"
      >
        下一页
      </button>
    </div>
  </div>
</template>

<style scoped>
.community-page {
  max-width: 1400px;
  margin: 0 auto;
}

/* ========== Header ========== */
.header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 40px;
}

.page-title {
  font-family: 'Cinzel', serif;
  font-size: 30px;
  font-weight: 700;
  color: #2C1810;
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-family: 'Merriweather', serif;
  font-style: italic;
  font-size: 14px;
  color: #8D7B68;
  margin: 0;
}

/* ========== 公会筛选横幅 ========== */
.guild-filter-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  margin-bottom: 24px;
  background: linear-gradient(135deg, #FFF5E6, #FFF9F0);
  border: 1px solid #E5D4C1;
  border-left: 4px solid #B87333;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(184, 115, 51, 0.08);
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: #4B3621;
}

.banner-content i {
  font-size: 20px;
  color: #B87333;
}

.banner-content strong {
  color: #804030;
  font-weight: 600;
}

.clear-filter-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  color: #8D7B68;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-filter-btn:hover {
  background: #FFF5E6;
  border-color: #B87333;
  color: #B87333;
}

.clear-filter-btn i {
  font-size: 14px;
}

.header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.search-box {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: #8D7B68;
}

.search-box input {
  padding: 8px 16px 8px 36px;
  width: 256px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  font-size: 14px;
  color: #4B3621;
  outline: none;
  box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  transition: all 0.2s;
}

.search-box input:focus {
  border-color: #B87333;
  box-shadow: 0 0 0 2px rgba(184, 115, 51, 0.1);
}

.my-posts-btn,
.favorites-btn,
.history-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  color: #4B3621;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.my-posts-btn:hover,
.favorites-btn:hover,
.history-btn:hover {
  border-color: #B87333;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 20px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(128, 64, 48, 0.2);
  transition: all 0.2s;
}

.create-btn svg {
  width: 16px;
  height: 16px;
}

.create-btn:hover {
  background: #6B3528;
}

/* ========== Filter Section ========== */
.filter-section {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
}

.category-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.category-filter button {
  padding: 8px 18px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #4B3621;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}

.category-filter button:hover {
  border-color: #B87333;
  color: #B87333;
}

.category-filter button.active {
  background: #2C1810;
  border-color: #2C1810;
  color: #fff;
  box-shadow: 0 2px 6px rgba(44, 24, 16, 0.2);
}

.sort-select {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #8D7B68;
}

.sort-select select {
  background: transparent;
  border: none;
  color: #2C1810;
  font-weight: 500;
  cursor: pointer;
  outline: none;
}

/* ========== Pinned Section ========== */
.pinned-section {
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #804030;
  margin-bottom: 12px;
}

.pinned-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pinned-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #F5EFE7;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.pinned-item:hover {
  background: #E5D4C1;
}

.pinned-tag {
  flex-shrink: 0;
  padding: 2px 6px;
  background: #804030;
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  border-radius: 3px;
}

/* ========== Events Section ========== */
.events-section {
  background: linear-gradient(135deg, #FFFDF9, #F8EFE6);
  border: 1px solid #E5D4C1;
  border-radius: 16px;
  margin-bottom: 24px;
  overflow: hidden;
  box-shadow: 0 16px 40px rgba(75, 54, 33, 0.08);
}

.events-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 24px;
  cursor: pointer;
  transition: background 0.2s;
  background: rgba(255, 255, 255, 0.75);
}

.events-header:hover {
  background: #fff;
}

.events-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #804030;
}

.events-title i {
  font-size: 20px;
}

.events-count {
  font-size: 12px;
  font-weight: 600;
  color: #8D7B68;
  background: rgba(184, 115, 51, 0.15);
  padding: 2px 10px;
  border-radius: 999px;
}

.events-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.view-toggle-btn {
  padding: 6px 10px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.view-toggle-btn:hover {
  background: rgba(184, 115, 51, 0.12);
  border-color: #B87333;
  color: #B87333;
}

.view-toggle-btn i {
  font-size: 16px;
}

.expand-icon {
  font-size: 20px;
  color: #8D7B68;
  transition: transform 0.3s;
}

.events-body {
  border-top: 1px solid #F3E7DA;
  padding: 20px 24px 24px;
  background: rgba(255, 255, 255, 0.6);
}

.events-loading,
.events-empty {
  text-align: center;
  padding: 24px;
  color: #8D7B68;
  font-size: 14px;
}

.calendar-shell {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.calendar-head {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
}

.calendar-head-left {
  max-width: 520px;
}

.calendar-kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: #8D7B68;
  font-weight: 600;
}

.calendar-title-row {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 6px;
}

.calendar-title {
  font-family: 'Cinzel', serif;
  font-size: 24px;
  font-weight: 700;
  color: #2C1810;
  margin: 0;
}

.calendar-count {
  font-size: 12px;
  font-weight: 600;
  color: #804030;
  background: rgba(128, 64, 48, 0.12);
  padding: 4px 10px;
  border-radius: 999px;
}

.calendar-subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  color: #8D7B68;
}

.calendar-head-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
}

.calendar-sync {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(91, 140, 90, 0.15);
  color: #5B8C5A;
  font-size: 12px;
  font-weight: 600;
}

.sync-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #5B8C5A;
  box-shadow: 0 0 0 4px rgba(91, 140, 90, 0.2);
}

.calendar-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #E5D4C1;
  border-radius: 12px;
  background: #fff;
}

.calendar-nav-btn {
  width: 32px;
  height: 32px;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  background: #fff;
  color: #8D7B68;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.calendar-nav-btn:hover {
  background: rgba(184, 115, 51, 0.12);
  border-color: #B87333;
  color: #B87333;
}

.calendar-nav-btn i {
  font-size: 18px;
}

.calendar-month-title {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
  min-width: 120px;
  text-align: center;
}

.today-btn {
  padding: 6px 12px;
  border: 1px solid #B87333;
  border-radius: 8px;
  background: #B87333;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.today-btn:hover {
  background: #A66629;
  border-color: #A66629;
}

.calendar-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  background: #fff;
  border: 1px solid #F1E4D7;
  border-radius: 14px;
  padding: 14px 16px;
  box-shadow: 0 10px 24px rgba(75, 54, 33, 0.08);
}

.stat-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #8D7B68;
  margin-bottom: 6px;
  font-weight: 600;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: #2C1810;
}

.stat-foot {
  font-size: 12px;
  color: #8D7B68;
  margin-top: 6px;
}

.calendar-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid #F1E4D7;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.7);
}

.filter-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #8D7B68;
  font-weight: 600;
}

.filter-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-chip {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid #E5D4C1;
  background: #fff;
  color: #4B3621;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-chip:hover {
  border-color: #B87333;
  color: #B87333;
}

.filter-chip.active {
  background: #2C1810;
  border-color: #2C1810;
  color: #fff;
}

.calendar-board {
  background: #fff;
  border: 1px solid #F1E4D7;
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 12px 28px rgba(75, 54, 33, 0.08);
}

.calendar-weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: #8D7B68;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  text-align: center;
  margin-bottom: 8px;
}

.calendar-weekdays div {
  padding: 6px 0;
}

.calendar-days {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 8px;
}

.calendar-day {
  min-height: 120px;
  padding: 8px;
  border: 1px solid #F1E4D7;
  border-radius: 12px;
  background: #fff;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.calendar-day.other-month {
  background: #FAF7F2;
  opacity: 0.4;
}

.calendar-day.today {
  border-color: #B87333;
  box-shadow: 0 0 0 2px rgba(184, 115, 51, 0.15);
}

.calendar-day.has-events {
  background: #FFF6EC;
}

.calendar-day.has-events:hover {
  border-color: #B87333;
}

.day-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.day-number {
  font-size: 13px;
  font-weight: 700;
  color: #2C1810;
}

.day-count {
  font-size: 10px;
  color: #8D7B68;
}

.day-events {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.day-pill {
  border: 1px solid rgba(128, 64, 48, 0.08);
  background: #fff;
  border-left: 3px solid var(--pill-color, #D97706);
  border-radius: 8px;
  padding: 4px 6px;
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  text-align: left;
}

.day-pill:hover {
  background: #FDF3E6;
}

.pill-title {
  flex: 1;
  min-width: 0;
  font-weight: 600;
  color: #4B3621;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pill-time {
  font-size: 10px;
  color: #8D7B68;
  margin-left: 6px;
}

.day-more {
  align-self: flex-start;
  font-size: 10px;
  color: #8D7B68;
  font-weight: 600;
  text-align: left;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}

.day-more:hover {
  color: #B87333;
}

.events-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-item {
  display: flex;
  gap: 16px;
  padding: 14px;
  background: #fff;
  border: 1px solid #F1E4D7;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.event-item:hover {
  border-color: #B87333;
  background: #FFF3E4;
}

.event-date {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-width: 56px;
  padding: 8px;
  border-radius: 10px;
  color: #fff;
}

.event-month {
  font-size: 10px;
  text-transform: uppercase;
  opacity: 0.9;
}

.event-day {
  font-size: 20px;
  font-weight: 700;
  line-height: 1;
}

.event-info {
  flex: 1;
  min-width: 0;
}

.event-title {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 6px 0;
}

.event-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
}

.event-type {
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.event-time {
  color: #8D7B68;
}

@media (max-width: 1100px) {
  .calendar-head-right {
    align-items: flex-start;
  }

  .calendar-controls {
    flex-wrap: wrap;
    justify-content: flex-start;
  }
}

@media (max-width: 860px) {
  .calendar-stats {
    grid-template-columns: 1fr;
  }

  .calendar-weekdays {
    letter-spacing: 0.12em;
  }

  .calendar-day {
    min-height: 100px;
  }
}

.pinned-title {
  flex: 1;
  font-size: 14px;
  color: #2C1810;
  font-weight: 500;
}

.pinned-time {
  flex-shrink: 0;
  font-size: 12px;
  color: #8D7B68;
}

/* ========== Posts Grid (Masonry) ========== */
.posts-grid {
  column-count: 3;
  column-gap: 24px;
}

@media (max-width: 1024px) {
  .posts-grid { column-count: 2; }
}
@media (max-width: 600px) {
  .posts-grid { column-count: 1; }
}

/* ========== Post Card Base ========== */
.post-card {
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.08);
  break-inside: avoid;
  margin-bottom: 20px;
  overflow: hidden;
}

.post-card:hover {
  box-shadow: 0 10px 25px -5px rgba(75, 54, 33, 0.15);
  transform: translateY(-2px);
}

/* ========== Featured Card (大卡片) ========== */
.post-card.featured {
  position: relative;
}

.featured-badge {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: linear-gradient(135deg, #E6A23C, #D97706);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  border-radius: 4px;
  z-index: 2;
}

.card-image {
  width: 100%;
  height: 180px;
  overflow: hidden;
}

.card-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.post-card:hover .card-image img {
  transform: scale(1.05);
}

.card-body {
  padding: 16px;
}

.card-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.category-tag {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  padding: 3px 8px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  color: #B87333;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

/* Category Colors */
.cat-guild { background: #EBF5FF; color: #1D4ED8; border-color: #BFDBFE; }
.cat-report { background: #F5EFE7; color: #B87333; border-color: #E5D4C1; }
.cat-event { background: #FEF3C7; color: #D97706; border-color: #FDE68A; }
.cat-profile { background: #F0FDF4; color: #16A34A; border-color: #BBF7D0; }
.cat-novel { background: #FDF4FF; color: #A855F7; border-color: #E9D5FF; }
.cat-item { background: #FFF7ED; color: #EA580C; border-color: #FED7AA; }
.cat-other { background: #F3F4F6; color: #6B7280; border-color: #E5E7EB; }

.post-time {
  font-size: 12px;
  color: #8D7B68;
}

.post-card.featured .post-title {
  font-family: 'Merriweather', serif;
  font-size: 18px;
  font-weight: 700;
  color: #2C1810;
  margin-bottom: 8px;
  line-height: 1.4;
  transition: color 0.3s;
}

.post-card.featured:hover .post-title {
  color: #804030;
}

.post-card.featured .post-excerpt {
  font-size: 13px;
  color: #4B3621;
  line-height: 1.6;
  margin-bottom: 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ========== Cover Image ========== */
.cover-image {
  width: 100%;
  border-radius: 10px;
  overflow: hidden;
  margin-bottom: 14px;
}

.cover-image img {
  width: 100%;
  height: auto;
  display: block;
  transition: transform 0.3s;
}

.post-card:hover .cover-image img {
  transform: scale(1.03);
}

.cover-image.small {
  border-radius: 8px;
  margin-bottom: 10px;
}

.cover-image.small img {
  max-height: 150px;
  object-fit: cover;
}

/* ========== Image Preview ========== */
.image-preview {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
  overflow: hidden;
}

.preview-item {
  width: 100px;
  height: 100px;
  border-radius: 10px;
  overflow: hidden;
  flex-shrink: 0;
}

.preview-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.post-card:hover .preview-item img {
  transform: scale(1.05);
}

.preview-more {
  width: 100px;
  height: 100px;
  border-radius: 10px;
  background: #F5EFE7;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  color: #8D7B68;
  flex-shrink: 0;
}

/* 小卡片的图片预览 */
.image-preview.small {
  gap: 8px;
}

.image-preview.small .preview-item,
.image-preview.small .preview-more {
  width: 72px;
  height: 72px;
  border-radius: 8px;
}

.image-preview.small .preview-more {
  font-size: 14px;
}

.post-card.featured .card-footer {
  padding-top: 12px;
  border-top: 1px solid #F5EFE7;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #F5EFE7;
  margin-top: auto;
}

.author-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.author-avatar {
  width: 32px;
  height: 32px;
  min-width: 32px;
  max-width: 32px;
  min-height: 32px;
  max-height: 32px;
  flex-shrink: 0;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 6px;
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  overflow: hidden;
  text-align: center;
  line-height: 32px;
}

.author-avatar img {
  width: 32px;
  height: 32px;
  object-fit: cover;
  display: block;
}

.author-avatar.small {
  width: 24px;
  height: 24px;
  min-width: 24px;
  max-width: 24px;
  min-height: 24px;
  max-height: 24px;
  font-size: 11px;
  border-radius: 4px;
  line-height: 24px;
}

.author-avatar.small img {
  width: 24px;
  height: 24px;
}

.author-name {
  font-size: 14px;
  font-weight: 500;
  color: #2C1810;
}


.post-stats {
  display: flex;
  gap: 16px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #8D7B68;
}

.stat-item svg {
  width: 16px;
  height: 16px;
}

/* ========== Standard Card (小卡片) ========== */
.post-card.standard {
  display: flex;
  flex-direction: column;
}

.card-thumb {
  width: 100%;
  height: 120px;
  overflow: hidden;
}

.card-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}

.post-card.standard:hover .card-thumb img {
  transform: scale(1.05);
}

.card-content {
  padding: 12px;
}

.card-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.featured-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(230, 162, 60, 0.15);
  border: 1px solid rgba(217, 119, 6, 0.35);
  color: #B45309;
  font-size: 10px;
  font-weight: 600;
  line-height: 1;
}

.featured-tag i {
  font-size: 11px;
}

.post-card.standard .post-title {
  font-family: 'Merriweather', serif;
  font-size: 14px;
  font-weight: 700;
  color: #2C1810;
  margin: 6px 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color 0.2s;
}

.post-card.standard:hover .post-title {
  color: #804030;
}

.post-card.standard .post-excerpt {
  font-size: 12px;
  color: #8D7B68;
  line-height: 1.5;
  margin-bottom: 10px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-card.standard .card-footer {
  padding-top: 10px;
  border-top: 1px solid #F5EFE7;
}

.post-card.standard .author-name {
  font-size: 12px;
  color: #8D7B68;
}

.comment-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #8D7B68;
}

/* ========== Empty State ========== */
.empty-state {
  column-span: all;
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

/* ========== Pagination ========== */
.pagination {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 40px;
}

.page-btn {
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  color: #4B3621;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  border-color: #B87333;
}

.page-btn.active {
  background: #2C1810;
  border-color: #2C1810;
  color: #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ========== Loading ========== */
.loading {
  column-span: all;
  text-align: center;
  padding: 60px;
  color: #8D7B68;
  font-size: 16px;
}

/* ========== Animation ========== */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.4s ease-out forwards;
  animation-delay: calc(var(--delay) * 0.1s);
}

@keyframes fadeUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
