<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listPosts, listEvents, type PostWithAuthor, type EventItem, type ListPostsParams, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { getGuild, type Guild } from '@/api/guild'
import { getImageUrl, resolveApiUrl } from '@/api/item'
import { buildNameStyle } from '@/utils/userNameStyle'
import UserLevelBadge from '@/components/UserLevelBadge.vue'

type FeedTab = 'posts' | 'events'
type EventStatus = 'live' | 'upcoming' | 'ended'
type EventStatusFilter = 'active' | 'all' | EventStatus

const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const mounted = ref(false)
const loading = ref(false)
const posts = ref<PostWithAuthor[]>([])
const total = ref(0)
const pinnedPosts = ref<PostWithAuthor[]>([])

const events = ref<EventItem[]>([])
const eventsLoading = ref(false)

const feedTab = ref<FeedTab>('posts')
const sortBy = ref<'created_at' | 'view_count' | 'like_count'>('created_at')
const filterCategory = ref<PostCategory | ''>('')
const searchKeyword = ref('')
const filterGuildId = ref<number | null>(null)
const currentGuild = ref<Guild | null>(null)
const currentPage = ref(1)
const eventFilter = ref<'all' | 'server' | 'guild'>('all')
const eventStatusFilter = ref<EventStatusFilter>('active')
const bannerIndex = ref(0)
const SEARCH_DEBOUNCE_MS = 350
const BANNER_INTERVAL_MS = 6000
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
let bannerTimer: ReturnType<typeof setInterval> | null = null
let nowTick = ref(Date.now())
let nowTimer: ReturnType<typeof setInterval> | null = null

const postCategories = computed(() => POST_CATEGORIES.filter(cat => cat.value !== 'event'))

const eventTypeMeta = computed(() => ({
  server: { label: t('community.eventType.server'), color: '#804030' },
  guild: { label: t('community.eventType.guild'), color: '#B87333' },
  other: { label: t('community.eventType.other'), color: '#D97706' }
}))

async function bootstrapCommunity() {
  if (route.query.guild_id) {
    filterGuildId.value = Number(route.query.guild_id)
    await loadGuildInfo()
  } else {
    filterGuildId.value = null
    currentGuild.value = null
  }
  if (route.query.tab === 'events') {
    feedTab.value = 'events'
  }

  await Promise.all([loadPosts(), loadEvents(), loadPinnedPosts()])
  startBannerAutoplay()
}

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  nowTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 30000)
  await bootstrapCommunity()
})

// 从发帖页返回时重新拉取活动，避免 Banner 停留在旧数据
watch(
  () => route.fullPath,
  async (path, prev) => {
    if (!path.includes('/community') || path === prev) return
    if (route.name !== 'community') return
    await bootstrapCommunity()
  }
)

watch(() => route.query.guild_id, async (newGuildId) => {
  if (newGuildId) {
    filterGuildId.value = Number(newGuildId)
    await loadGuildInfo()
  } else {
    filterGuildId.value = null
    currentGuild.value = null
  }
  await Promise.all([loadPosts(), loadPinnedPosts(), loadEvents()])
})

watch(searchKeyword, () => {
  queueSearchReload()
})

watch(feedTab, (tab) => {
  currentPage.value = 1
  if (tab === 'posts') {
    void Promise.all([loadPosts(), loadPinnedPosts()])
  } else {
    void loadEvents()
  }
})

onUnmounted(() => {
  clearSearchDebounce()
  stopBannerAutoplay()
  if (nowTimer) {
    clearInterval(nowTimer)
    nowTimer = null
  }
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
  if (feedTab.value !== 'posts') return
  loading.value = true
  try {
    const normalizedSearch = searchKeyword.value.trim()
    const params: ListPostsParams = {
      page: currentPage.value,
      page_size: 12,
      sort: sortBy.value,
      order: 'desc',
      status: 'published',
      is_pinned: false,
    }
    if (normalizedSearch) {
      params.search = normalizedSearch
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    } else {
      params.exclude_category = 'event'
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
  if (feedTab.value !== 'posts') return
  try {
    const normalizedSearch = searchKeyword.value.trim()
    const params: ListPostsParams = {
      page: 1,
      page_size: 10,
      sort: 'created_at',
      order: 'desc',
      status: 'published',
      is_pinned: true,
    }
    if (normalizedSearch) {
      params.search = normalizedSearch
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    } else {
      params.exclude_category = 'event'
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

function clearSearchDebounce() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
}

function queueSearchReload(immediate = false) {
  currentPage.value = 1
  clearSearchDebounce()

  if (immediate) {
    void Promise.all([loadPosts(), loadPinnedPosts()])
    return
  }

  searchDebounceTimer = setTimeout(() => {
    void Promise.all([loadPosts(), loadPinnedPosts()])
  }, SEARCH_DEBOUNCE_MS)
}

function goToPost(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function goToCreatePost(options?: { category?: PostCategory }) {
  router.push({
    name: 'post-create',
    query: options?.category ? { category: options.category } : undefined,
  })
}

function goToCreateEvent() {
  goToCreatePost({ category: 'event' })
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

function switchFeedTab(tab: FeedTab) {
  feedTab.value = tab
  if (tab === 'events') {
    router.replace({ query: { ...route.query, tab: 'events' } })
  } else {
    const nextQuery = { ...route.query }
    delete nextQuery.tab
    router.replace({ query: nextQuery })
  }
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date(nowTick.value)
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))

  if (hours < 1) return t('community.time.justNow')
  if (hours < 24) return t('community.time.hoursAgo', { hours })
  const days = Math.floor(hours / 24)
  if (days < 7) return t('community.time.daysAgo', { days })
  return date.toLocaleDateString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US')
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
  return cat ? cat.label : t('community.category.other')
}

function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}

function formatLocation(region?: string, address?: string) {
  const parts = [region, address].map(part => part?.trim()).filter(Boolean)
  return parts.join(' · ')
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
  const typeKey = event.event_type || 'server'
  return eventTypeMeta.value[typeKey]?.label || eventTypeMeta.value.server.label
}

function resolveEventColor(event: EventItem) {
  const typeKey = event.event_type || 'server'
  return event.event_color || eventTypeMeta.value[typeKey]?.color || eventTypeMeta.value.server.color
}

function getEventEndTime(event: EventItem): Date | null {
  if (event.event_end_time) return new Date(event.event_end_time)
  if (!event.event_start_time) return null
  const end = new Date(event.event_start_time)
  end.setHours(23, 59, 59, 999)
  return end
}

function getEventStatus(event: EventItem): EventStatus {
  void nowTick.value
  const now = new Date()
  if (!event.event_start_time) return 'ended'
  const start = new Date(event.event_start_time)
  const end = getEventEndTime(event)
  if (end && end.getTime() <= now.getTime()) return 'ended'
  if (start.getTime() <= now.getTime()) return 'live'
  return 'upcoming'
}

function isEventEnded(event: EventItem) {
  return getEventStatus(event) === 'ended'
}

function getEventStatusLabel(status: EventStatus) {
  return t(`community.eventStatus.${status}`)
}

function formatEventRange(event: EventItem) {
  if (!event.event_start_time) return ''
  const start = new Date(event.event_start_time)
  const end = event.event_end_time ? new Date(event.event_end_time) : null
  const dateLocale = locale.value === 'zh-CN' ? 'zh-CN' : 'en-US'
  const startDate = start.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric', weekday: 'short' })
  const startTime = start.toLocaleTimeString(dateLocale, { hour: '2-digit', minute: '2-digit' })

  if (!end) return `${startDate} ${startTime}`

  const sameDay =
    start.getFullYear() === end.getFullYear() &&
    start.getMonth() === end.getMonth() &&
    start.getDate() === end.getDate()
  const endTime = end.toLocaleTimeString(dateLocale, { hour: '2-digit', minute: '2-digit' })
  if (sameDay) return `${startDate} ${startTime} – ${endTime}`

  const endDate = end.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric', weekday: 'short' })
  return `${startDate} ${startTime} – ${endDate} ${endTime}`
}

function formatCountdown(event: EventItem) {
  void nowTick.value
  const status = getEventStatus(event)
  const now = Date.now()
  if (status === 'upcoming' && event.event_start_time) {
    return formatDuration(new Date(event.event_start_time).getTime() - now, 'starts')
  }
  if (status === 'live') {
    const end = getEventEndTime(event)
    if (end) return formatDuration(end.getTime() - now, 'ends')
  }
  return ''
}

function formatDuration(ms: number, mode: 'starts' | 'ends') {
  if (ms <= 0) return ''
  const totalMinutes = Math.floor(ms / 60000)
  const days = Math.floor(totalMinutes / (60 * 24))
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60)
  const minutes = totalMinutes % 60

  let text = ''
  if (days > 0) text = t('community.countdown.daysHours', { days, hours })
  else if (hours > 0) text = t('community.countdown.hoursMinutes', { hours, minutes })
  else text = t('community.countdown.minutes', { minutes: Math.max(minutes, 1) })

  return mode === 'starts'
    ? t('community.countdown.startsIn', { time: text })
    : t('community.countdown.endsIn', { time: text })
}

function formatEventMonth(dateStr?: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', { month: 'short' })
}

function formatEventDay(dateStr?: string) {
  if (!dateStr) return ''
  return new Date(dateStr).getDate().toString()
}

function formatEventTimeShort(dateStr?: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleTimeString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    hour: '2-digit',
    minute: '2-digit'
  })
}

function getEventCover(event: EventItem) {
  if (event.cover_image_url) {
    return resolveApiUrl(event.cover_image_url)
  }
  if (event.cover_image) {
    if (event.cover_image.startsWith('http://') || event.cover_image.startsWith('https://') || event.cover_image.startsWith('data:')) {
      return event.cover_image
    }
    return getImageUrl('post-cover', event.id, {
      w: 1200,
      q: 80,
      v: event.cover_image_updated_at || event.updated_at
    })
  }
  return ''
}

function getEventStyle(event: EventItem) {
  const color = resolveEventColor(event)
  const hex = color.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return {
    backgroundColor: `rgba(${r}, ${g}, ${b}, 0.15)`,
    color,
  }
}

function sortEventsByStatus(list: EventItem[]) {
  const statusWeight: Record<EventStatus, number> = { live: 0, upcoming: 1, ended: 2 }
  return [...list].sort((a, b) => {
    const sa = getEventStatus(a)
    const sb = getEventStatus(b)
    if (statusWeight[sa] !== statusWeight[sb]) return statusWeight[sa] - statusWeight[sb]
    const at = a.event_start_time ? new Date(a.event_start_time).getTime() : 0
    const bt = b.event_start_time ? new Date(b.event_start_time).getTime() : 0
    if (sa === 'ended') return bt - at
    return at - bt
  })
}

// Banner 只跟随公会上下文，不受搜索/类型筛选影响
const bannerSourceEvents = computed(() => {
  let list = events.value
  if (filterGuildId.value) {
    list = list.filter(event => event.guild_id === filterGuildId.value || event.event_type === 'server' || !event.event_type)
  }
  return sortEventsByStatus(list)
})

const filteredEvents = computed(() => {
  let list = events.value
  if (filterGuildId.value) {
    list = list.filter(event => event.guild_id === filterGuildId.value || event.event_type === 'server' || !event.event_type)
  }
  if (eventFilter.value !== 'all') {
    list = list.filter(event => (event.event_type || 'server') === eventFilter.value)
  }
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (keyword) {
    list = list.filter(event => {
      const haystack = [
        event.title,
        event.author_name,
        event.guild_name,
        event.region,
        event.address,
      ].filter(Boolean).join(' ').toLowerCase()
      return haystack.includes(keyword)
    })
  }
  return sortEventsByStatus(list)
})

const displayEvents = computed(() => {
  if (eventStatusFilter.value === 'all') return filteredEvents.value
  if (eventStatusFilter.value === 'active') {
    return filteredEvents.value.filter(event => getEventStatus(event) !== 'ended')
  }
  return filteredEvents.value.filter(event => getEventStatus(event) === eventStatusFilter.value)
})

const bannerEvents = computed(() => {
  return bannerSourceEvents.value
    .filter(event => getEventStatus(event) !== 'ended')
    .slice(0, 5)
})

watch(bannerEvents, () => {
  bannerIndex.value = 0
  startBannerAutoplay()
})

const activeBanner = computed(() => bannerEvents.value[bannerIndex.value] || null)

const eventStats = computed(() => {
  const list = filteredEvents.value
  return {
    live: list.filter(e => getEventStatus(e) === 'live').length,
    upcoming: list.filter(e => getEventStatus(e) === 'upcoming').length,
    ended: list.filter(e => getEventStatus(e) === 'ended').length,
  }
})

function startBannerAutoplay() {
  stopBannerAutoplay()
  if (bannerEvents.value.length <= 1) return
  bannerTimer = setInterval(() => {
    bannerIndex.value = (bannerIndex.value + 1) % bannerEvents.value.length
  }, BANNER_INTERVAL_MS)
}

function stopBannerAutoplay() {
  if (bannerTimer) {
    clearInterval(bannerTimer)
    bannerTimer = null
  }
}

function goBanner(index: number) {
  if (!bannerEvents.value.length) return
  bannerIndex.value = ((index % bannerEvents.value.length) + bannerEvents.value.length) % bannerEvents.value.length
  startBannerAutoplay()
}

function prevBanner() {
  goBanner(bannerIndex.value - 1)
}

function nextBanner() {
  goBanner(bannerIndex.value + 1)
}

function setEventFilter(filter: 'all' | 'server' | 'guild') {
  eventFilter.value = filter
}

function setEventStatusFilter(filter: EventStatusFilter) {
  eventStatusFilter.value = filter
}
</script>

<template>
  <div class="community-page" :class="{ 'animate-in': mounted }">
    <header class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <h1 class="page-title">{{ t('community.pageTitle') }}</h1>
        <p class="page-subtitle">{{ t('community.pageSubtitle') }}</p>
      </div>
      <div class="header-actions">
        <div class="search-box">
          <div class="search-field keyword">
            <svg class="search-field-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            <input v-model="searchKeyword" type="text" :placeholder="t('community.filter.search')" @keyup.enter="queueSearchReload(true)" />
          </div>
        </div>
        <button class="favorites-btn" @click="goToFavorites">
          <i class="ri-bookmark-3-line"></i>
          {{ t('community.action.favorites') }}
        </button>
        <button class="history-btn" @click="goToHistory">
          <i class="ri-history-line"></i>
          {{ t('community.action.history') }}
        </button>
        <button class="my-posts-btn" @click="goToMyPosts">
          <i class="ri-file-list-3-line"></i>
          {{ t('community.action.myPosts') }}
        </button>
        <button type="button" class="create-btn" @click="goToCreatePost()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/>
          </svg>
          <span>{{ t('community.action.publish') }}</span>
        </button>
      </div>
    </header>

    <div v-if="currentGuild" class="guild-filter-banner anim-item" style="--delay: 1">
      <div class="banner-content">
        <i class="ri-shield-line"></i>
        <span>{{ t('community.guild.filterBanner', { name: currentGuild.name }) }}</span>
      </div>
      <button class="clear-filter-btn" @click="clearGuildFilter">
        <i class="ri-close-line"></i>
        {{ t('community.guild.clearFilter') }}
      </button>
    </div>

    <!-- Event Banner -->
    <section class="event-banner-section anim-item" style="--delay: 1">
      <div v-if="eventsLoading" class="event-banner empty loading-banner">
        <div class="banner-empty-inner">
          <i class="ri-loader-4-line spin"></i>
          <p>{{ t('community.loading') }}</p>
        </div>
      </div>

      <div
        v-else-if="activeBanner"
        class="event-banner"
        :class="{ 'has-cover': !!getEventCover(activeBanner) }"
        :style="{ '--event-color': resolveEventColor(activeBanner) }"
      >
        <div class="banner-media">
          <img
            v-if="getEventCover(activeBanner)"
            :src="getEventCover(activeBanner)"
            alt=""
            class="banner-cover"
            loading="lazy"
          />
          <div v-else class="banner-fallback" aria-hidden="true"></div>
          <div class="banner-color-fade" aria-hidden="true"></div>
        </div>

        <div class="banner-layout">
          <div class="banner-date-card" aria-hidden="true">
            <span class="date-month">{{ formatEventMonth(activeBanner.event_start_time) }}</span>
            <span class="date-day">{{ formatEventDay(activeBanner.event_start_time) }}</span>
            <span class="date-time">{{ formatEventTimeShort(activeBanner.event_start_time) }}</span>
          </div>

          <div class="banner-body">
            <div class="banner-top-row">
              <div class="banner-kicker">
                <span class="kicker-dot"></span>
                {{ t('community.banner.kicker') }}
                <span class="banner-count">{{ t('community.banner.count', { count: bannerEvents.length }) }}</span>
              </div>
              <div class="banner-chips">
                <span class="status-chip" :class="getEventStatus(activeBanner)">
                  <i :class="getEventStatus(activeBanner) === 'live' ? 'ri-broadcast-line' : 'ri-time-line'"></i>
                  {{ getEventStatusLabel(getEventStatus(activeBanner)) }}
                </span>
                <span class="type-chip" :style="getEventStyle(activeBanner)">{{ getEventTypeLabel(activeBanner) }}</span>
                <span v-if="activeBanner.guild_name" class="meta-chip">
                  <i class="ri-shield-star-line"></i>
                  {{ activeBanner.guild_name }}
                </span>
              </div>
            </div>

            <h2 class="banner-title" @click="goToPost(activeBanner.id)">{{ activeBanner.title }}</h2>

            <div class="banner-bottom-row">
              <div class="banner-meta-inline">
                <span class="meta-inline">
                  <i class="ri-time-line"></i>
                  {{ formatEventRange(activeBanner) }}
                </span>
                <span v-if="formatCountdown(activeBanner)" class="meta-inline countdown">
                  {{ formatCountdown(activeBanner) }}
                </span>
                <span v-if="formatLocation(activeBanner.region, activeBanner.address)" class="meta-inline">
                  <i class="ri-map-pin-2-fill"></i>
                  {{ formatLocation(activeBanner.region, activeBanner.address) }}
                </span>
              </div>

              <div class="banner-actions">
                <button type="button" class="banner-cta" @click="goToPost(activeBanner.id)">
                  {{ t('community.banner.viewDetail') }}
                  <i class="ri-arrow-right-line"></i>
                </button>
                <button type="button" class="banner-secondary" @click="switchFeedTab('events')">
                  {{ t('community.banner.viewAll') }}
                </button>
                <div v-if="bannerEvents.length > 1" class="banner-nav">
                  <button type="button" class="nav-btn" @click="prevBanner" :aria-label="t('community.banner.prev')">
                    <i class="ri-arrow-left-s-line"></i>
                  </button>
                  <div class="banner-dots">
                    <button
                      v-for="(event, index) in bannerEvents"
                      :key="event.id"
                      type="button"
                      class="dot"
                      :class="{ active: index === bannerIndex }"
                      @click="goBanner(index)"
                    />
                  </div>
                  <button type="button" class="nav-btn" @click="nextBanner" :aria-label="t('community.banner.next')">
                    <i class="ri-arrow-right-s-line"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="event-banner empty">
        <div class="banner-empty-inner">
          <i class="ri-calendar-event-line"></i>
          <div>
            <h3>{{ t('community.banner.emptyTitle') }}</h3>
            <p>{{ t('community.banner.emptyBody') }}</p>
          </div>
          <button type="button" class="create-btn banner-empty-action" @click="goToCreateEvent">
            <i class="ri-add-line" aria-hidden="true"></i>
            <span>{{ t('community.banner.emptyAction') }}</span>
          </button>
        </div>
      </div>
    </section>

    <!-- Feed Tabs -->
    <div class="feed-tabs anim-item" style="--delay: 2">
      <button
        type="button"
        class="feed-tab"
        :class="{ active: feedTab === 'posts' }"
        @click="switchFeedTab('posts')"
      >
        <i class="ri-article-line"></i>
        {{ t('community.feed.posts') }}
      </button>
      <button
        type="button"
        class="feed-tab"
        :class="{ active: feedTab === 'events' }"
        @click="switchFeedTab('events')"
      >
        <i class="ri-calendar-event-line"></i>
        {{ t('community.feed.events') }}
        <span v-if="eventStats.live + eventStats.upcoming > 0" class="tab-badge">
          {{ eventStats.live + eventStats.upcoming }}
        </span>
      </button>
    </div>

    <!-- Posts Tab -->
    <template v-if="feedTab === 'posts'">
      <div class="filter-section anim-item" style="--delay: 2.5">
        <div class="category-filter">
          <button
            :class="{ active: filterCategory === '' }"
            @click="changeCategoryFilter('')"
          >{{ t('community.filter.all') }}</button>
          <button
            v-for="cat in postCategories"
            :key="cat.value"
            :class="{ active: filterCategory === cat.value }"
            @click="changeCategoryFilter(cat.value)"
          >{{ cat.label }}</button>
        </div>
        <div class="sort-select">
          <span class="sort-label">{{ t('community.filter.sortLabel') }}</span>
          <select v-model="sortBy" @change="loadPosts">
            <option value="created_at">{{ t('community.filter.sortLatest') }}</option>
            <option value="like_count">{{ t('community.filter.sortHot') }}</option>
            <option value="view_count">{{ t('community.filter.sortViews') }}</option>
          </select>
        </div>
      </div>

      <div v-if="loading" class="loading anim-item" style="--delay: 3">{{ t('community.loading') }}</div>

      <template v-else>
        <div v-if="pinnedPosts.length > 0" class="pinned-section anim-item" style="--delay: 3">
          <div class="section-header">
            <i class="ri-pushpin-fill"></i>
            <span>{{ t('community.pinned.title') }}</span>
          </div>
          <div class="pinned-list">
            <div
              v-for="post in pinnedPosts"
              :key="post.id"
              class="pinned-item"
              @click="goToPost(post.id)"
            >
              <span class="pinned-tag">{{ t('community.pinned.tag') }}</span>
              <div class="pinned-content">
                <span class="pinned-title">{{ post.title }}</span>
                <span v-if="formatLocation(post.region, post.address)" class="pinned-location">
                  <i class="ri-map-pin-2-fill"></i>
                  {{ formatLocation(post.region, post.address) }}
                </span>
              </div>
              <span class="pinned-time">{{ formatDate(post.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="posts-grid anim-item" style="--delay: 3.5">
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
                  {{ t('community.post.featured') }}
                </span>
              </div>
              <p v-if="formatLocation(post.region, post.address)" class="post-location">
                <i class="ri-map-pin-2-fill"></i>
                {{ formatLocation(post.region, post.address) }}
              </p>
              <h3 class="post-title">{{ post.title }}</h3>
              <p class="post-excerpt">{{ stripHtml(post.content).substring(0, 100) }}...</p>
              <div v-if="post.cover_image_url" class="cover-image small">
                <img :src="getImageUrl('post-cover', post.id, { w: 400, q: 80, v: post.cover_image_updated_at || post.updated_at })" alt="" loading="lazy" />
              </div>
              <div class="card-footer">
                <div class="author-info">
                  <div class="author-avatar small">
                    <img v-if="post.author_avatar" :src="resolveApiUrl(post.author_avatar)" alt="" loading="lazy" />
                    <span v-else>{{ post.author_name?.charAt(0) || 'U' }}</span>
                  </div>
                  <span class="author-name" :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</span>
                  <UserLevelBadge
                    :level="post.author_forum_level"
                    :name="post.author_forum_level_name"
                    :color="post.author_forum_level_color"
                    :bold="post.author_forum_level_bold"
                    size="xs"
                  />
                </div>
                <span class="comment-count">
                  <i class="ri-chat-3-line"></i>
                  {{ post.comment_count }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="posts.length === 0 && pinnedPosts.length === 0" class="empty-state">
            <i class="ri-article-line"></i>
            <p>{{ t('community.empty') }}</p>
            <button type="button" class="create-btn" @click="goToCreatePost()">
              <i class="ri-add-line" aria-hidden="true"></i>
              <span>{{ t('community.emptyAction') }}</span>
            </button>
          </div>
        </div>
      </template>

      <div v-if="posts.length > 0" class="pagination anim-item" style="--delay: 4">
        <button class="page-btn" :disabled="currentPage === 1" @click="changePage(currentPage - 1)">
          {{ t('community.pagination.prev') }}
        </button>
        <span class="page-info">
          {{ t('community.pagination.pageInfo', { current: currentPage, total: Math.ceil(total / 12) || 1 }) }}
        </span>
        <button class="page-btn" :disabled="currentPage >= Math.ceil(total / 12)" @click="changePage(currentPage + 1)">
          {{ t('community.pagination.next') }}
        </button>
      </div>
    </template>

    <!-- Events Tab -->
    <template v-else>
      <div class="event-toolbar anim-item" style="--delay: 2.5">
        <div class="event-status-filters">
          <button type="button" :class="{ active: eventStatusFilter === 'active' }" @click="setEventStatusFilter('active')">
            {{ t('community.eventFilter.active') }}
            <span>{{ eventStats.live + eventStats.upcoming }}</span>
          </button>
          <button type="button" :class="{ active: eventStatusFilter === 'live' }" @click="setEventStatusFilter('live')">
            {{ t('community.eventStatus.live') }}
            <span>{{ eventStats.live }}</span>
          </button>
          <button type="button" :class="{ active: eventStatusFilter === 'upcoming' }" @click="setEventStatusFilter('upcoming')">
            {{ t('community.eventStatus.upcoming') }}
            <span>{{ eventStats.upcoming }}</span>
          </button>
          <button type="button" :class="{ active: eventStatusFilter === 'ended' }" @click="setEventStatusFilter('ended')">
            {{ t('community.eventStatus.ended') }}
            <span>{{ eventStats.ended }}</span>
          </button>
          <button type="button" :class="{ active: eventStatusFilter === 'all' }" @click="setEventStatusFilter('all')">
            {{ t('community.eventFilter.all') }}
          </button>
        </div>
        <div class="event-type-filters">
          <button type="button" class="filter-chip" :class="{ active: eventFilter === 'all' }" @click="setEventFilter('all')">{{ t('community.eventType.all') }}</button>
          <button type="button" class="filter-chip" :class="{ active: eventFilter === 'server' }" @click="setEventFilter('server')">{{ t('community.eventType.server') }}</button>
          <button type="button" class="filter-chip" :class="{ active: eventFilter === 'guild' }" @click="setEventFilter('guild')">{{ t('community.eventType.guild') }}</button>
        </div>
      </div>

      <div v-if="eventsLoading" class="loading anim-item" style="--delay: 3">{{ t('community.loading') }}</div>

      <div v-else-if="displayEvents.length === 0" class="events-empty-state anim-item" style="--delay: 3">
        <i class="ri-calendar-close-line"></i>
        <p>{{ t('community.events.empty') }}</p>
        <button type="button" class="create-btn" @click="goToCreateEvent">
          <i class="ri-add-line" aria-hidden="true"></i>
          <span>{{ t('community.banner.emptyAction') }}</span>
        </button>
      </div>

      <div v-else class="events-grid anim-item" style="--delay: 3">
        <article
          v-for="event in displayEvents"
          :key="event.id"
          class="event-card"
          :class="[getEventStatus(event), { faded: isEventEnded(event) }]"
          :style="{ '--event-color': resolveEventColor(event) }"
          @click="goToPost(event.id)"
        >
          <div class="event-card-visual">
            <img
              v-if="getEventCover(event)"
              :src="getEventCover(event)"
              alt=""
              class="event-card-cover"
              loading="lazy"
            />
            <div class="event-card-fallback"></div>
            <div class="event-date-badge">
              <span class="month">{{ formatEventMonth(event.event_start_time) }}</span>
              <span class="day">{{ formatEventDay(event.event_start_time) }}</span>
            </div>
            <span class="status-chip floating" :class="getEventStatus(event)">
              {{ getEventStatusLabel(getEventStatus(event)) }}
            </span>
          </div>

          <div class="event-card-body">
            <div class="event-card-chips">
              <span class="type-chip" :style="getEventStyle(event)">{{ getEventTypeLabel(event) }}</span>
              <span v-if="event.guild_name" class="meta-chip soft">
                <i class="ri-shield-star-line"></i>
                {{ event.guild_name }}
              </span>
            </div>
            <h3 class="event-card-title">{{ event.title }}</h3>
            <p class="event-card-time">
              <i class="ri-time-line"></i>
              {{ formatEventRange(event) }}
            </p>
            <p v-if="formatCountdown(event)" class="event-card-countdown">{{ formatCountdown(event) }}</p>
            <p v-if="formatLocation(event.region, event.address)" class="event-card-location">
              <i class="ri-map-pin-2-fill"></i>
              {{ formatLocation(event.region, event.address) }}
            </p>
            <div class="event-card-footer">
              <div class="author-info">
                <span class="author-name" :style="buildNameStyle(event.author_name_color, event.author_name_bold)">
                  {{ event.author_name }}
                </span>
              </div>
              <span class="comment-count">
                <i class="ri-chat-3-line"></i>
                {{ event.comment_count }}
              </span>
            </div>
          </div>
        </article>
      </div>
    </template>
  </div>
</template>

<style scoped>
.community-page {
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 28px;
}

.page-title {
  font-family: 'Cinzel', serif;
  font-size: 30px;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-family: 'Merriweather', serif;
  font-style: italic;
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
  margin: 0;
}

.guild-filter-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  margin-bottom: 24px;
  background: linear-gradient(135deg, var(--color-card-bg-hover, #FFF5E6), var(--color-card-bg, #FFF9F0));
  border: 1px solid var(--color-border, #E5D4C1);
  border-left: 4px solid var(--color-accent, #B87333);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(184, 115, 51, 0.08);
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: var(--color-primary, #4B3621);
}

.banner-content i {
  font-size: 20px;
  color: var(--color-accent, #B87333);
}

.clear-filter-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-filter-btn:hover {
  background: var(--color-card-bg-hover, #FFF5E6);
  border-color: var(--color-accent, #B87333);
  color: var(--color-accent, #B87333);
}

.header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.search-box {
  display: flex;
  align-items: center;
  flex: 1 1 360px;
  min-width: min(100%, 360px);
}

.search-field {
  position: relative;
  flex: 1 1 auto;
}

.search-field.keyword {
  min-width: min(100%, 320px);
}

.search-field-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: var(--color-text-secondary, #8D7B68);
}

.search-field input {
  padding: 8px 16px 8px 36px;
  width: 100%;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
  font-size: 14px;
  color: var(--color-primary, #4B3621);
  outline: none;
  box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  transition: all 0.2s;
}

.search-field input:focus {
  border-color: var(--color-accent, #B87333);
  box-shadow: 0 0 0 2px rgba(184, 115, 51, 0.1);
}

.my-posts-btn,
.favorites-btn,
.history-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
  color: var(--color-primary, #4B3621);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.my-posts-btn:hover,
.favorites-btn:hover,
.history-btn:hover {
  border-color: var(--color-accent, #B87333);
}

.create-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 20px;
  background: var(--color-secondary, #804030);
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(128, 64, 48, 0.2);
  transition: all 0.2s;
}

.create-btn svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.create-btn i {
  font-size: 16px;
  line-height: 1;
  flex-shrink: 0;
}

.create-btn span {
  line-height: 1.2;
}

.create-btn:hover {
  background: var(--color-secondary-hover, #6B3528);
}

.banner-empty-action {
  min-height: 40px;
  padding: 10px 18px;
}

/* ========== Event Banner ========== */
.event-banner-section {
  margin-bottom: 20px;
}

.event-banner {
  position: relative;
  min-height: 140px;
  height: 140px;
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid var(--color-border, #E5D4C1);
  box-shadow: 0 10px 24px rgba(75, 54, 33, 0.1);
  background: var(--event-color, #B87333);
  color: #fff;
}

.banner-media {
  position: absolute;
  inset: 0;
  z-index: 0;
}

/* 封面锚定左侧约 45% 区域，右侧留给标记色与文字 */
.banner-cover {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 52%;
  height: 100%;
  object-fit: cover;
  object-position: center;
  display: block;
}

.banner-fallback {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, #2c1810 0%, color-mix(in srgb, var(--event-color, #B87333) 55%, #2c1810) 100%);
}

/* 封面从左清晰 → 向右渐变成活动标记色 */
.banner-color-fade {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    90deg,
    rgba(0, 0, 0, 0.12) 0%,
    transparent 18%,
    transparent 30%,
    color-mix(in srgb, var(--event-color, #B87333) 42%, transparent) 46%,
    color-mix(in srgb, var(--event-color, #B87333) 82%, #1a100c) 68%,
    var(--event-color, #B87333) 100%
  );
}

.event-banner:not(.has-cover) .banner-color-fade {
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--event-color, #B87333) 70%, #1a100c) 0%,
    var(--event-color, #B87333) 100%
  );
}

.banner-layout {
  position: relative;
  z-index: 1;
  height: 100%;
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 14px;
  align-items: center;
  padding: 12px 16px 12px 14px;
}

.banner-date-card {
  width: 72px;
  height: 104px;
  padding: 10px 8px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.14);
  border: 1px solid rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(8px);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  flex-shrink: 0;
}

.date-month {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  opacity: 0.9;
  font-weight: 700;
}

.date-day {
  font-family: 'Cinzel', serif;
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.date-time {
  font-size: 11px;
  opacity: 0.92;
  margin-top: 2px;
}

.banner-body {
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
}

.banner-top-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}

.banner-kicker {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  font-weight: 700;
  color: rgba(255, 245, 230, 0.88);
}

.kicker-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 0 0 3px color-mix(in srgb, #fff 22%, transparent);
}

.banner-count {
  padding: 1px 7px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.14);
  letter-spacing: 0;
  text-transform: none;
  font-size: 10px;
}

.banner-chips,
.event-card-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.status-chip,
.type-chip,
.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
}

.status-chip.live {
  background: rgba(220, 38, 38, 0.22);
  color: #fecaca;
  border: 1px solid rgba(248, 113, 113, 0.45);
}

.status-chip.upcoming {
  background: rgba(245, 158, 11, 0.22);
  color: #fde68a;
  border: 1px solid rgba(251, 191, 36, 0.4);
}

.status-chip.ended {
  background: rgba(148, 163, 184, 0.22);
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.35);
}

.status-chip.floating {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  backdrop-filter: blur(8px);
}

.type-chip {
  border: 1px solid transparent;
  background: rgba(255, 255, 255, 0.16) !important;
  color: #fff !important;
}

.meta-chip {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.92);
}

.meta-chip.soft {
  background: var(--color-card-bg, #F5EFE7);
  color: var(--color-secondary, #804030);
}

.banner-title {
  margin: 0;
  font-family: 'Cinzel', serif;
  font-size: clamp(16px, 1.8vw, 22px);
  line-height: 1.25;
  cursor: pointer;
  transition: opacity 0.2s;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.25);
}

.banner-title:hover {
  opacity: 0.92;
}

.banner-bottom-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
}

.banner-meta-inline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 12px;
  min-width: 0;
  flex: 1 1 auto;
}

.meta-inline {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 248, 240, 0.92);
  white-space: nowrap;
}

.meta-inline i {
  font-size: 13px;
  opacity: 0.9;
}

.meta-inline.countdown {
  color: #fde68a;
}

.banner-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.banner-cta,
.banner-secondary {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 7px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.banner-cta {
  border: none;
  background: rgba(255, 255, 255, 0.95);
  color: color-mix(in srgb, var(--event-color, #B87333) 70%, #2c1810);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.banner-cta:hover {
  filter: brightness(1.04);
  transform: translateY(-1px);
}

.banner-secondary {
  border: 1px solid rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.banner-secondary:hover {
  background: rgba(255, 255, 255, 0.18);
}

.banner-nav {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: 2px;
}

.nav-btn {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.24);
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  display: grid;
  place-items: center;
  cursor: pointer;
  padding: 0;
}

.nav-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.banner-dots {
  display: flex;
  gap: 5px;
  align-items: center;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  border: none;
  background: rgba(255, 255, 255, 0.35);
  cursor: pointer;
  padding: 0;
  transition: all 0.2s;
}

.dot.active {
  width: 16px;
  background: #fff;
}

.event-banner.empty {
  height: auto;
  min-height: 120px;
  background:
    linear-gradient(135deg, rgba(255, 249, 240, 0.95), rgba(245, 239, 231, 0.98));
  color: var(--color-text-main, #2C1810);
  display: grid;
  place-items: center;
}

.banner-empty-inner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 16px 20px;
  text-align: left;
}

.banner-empty-inner i {
  font-size: 32px;
  color: var(--color-accent, #B87333);
}

.banner-empty-inner h3 {
  margin: 0 0 2px;
  font-size: 15px;
}

.banner-empty-inner p {
  margin: 0;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 13px;
}

.loading-banner .spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ========== Feed Tabs ========== */
.feed-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  padding: 6px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 14px;
  width: fit-content;
  max-width: 100%;
}

.feed-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.feed-tab:hover {
  color: var(--color-accent, #B87333);
  background: var(--color-card-bg-hover, #FFF5E6);
}

.feed-tab.active {
  background: var(--color-secondary, #804030);
  color: var(--btn-primary-text, #fff);
  box-shadow: 0 6px 16px rgba(128, 64, 48, 0.2);
}

.tab-badge {
  min-width: 20px;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
  font-size: 11px;
  line-height: 1.4;
}

.feed-tab:not(.active) .tab-badge {
  background: var(--color-primary-light, rgba(184, 115, 51, 0.15));
  color: var(--color-secondary, #804030);
}

/* ========== Filters ========== */
.filter-section {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.category-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.category-filter button {
  padding: 8px 18px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  color: var(--color-primary, #4B3621);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}

.category-filter button:hover {
  border-color: var(--color-accent, #B87333);
  color: var(--color-accent, #B87333);
}

.category-filter button.active {
  background: var(--color-accent, #B87333);
  border-color: var(--color-accent, #B87333);
  color: var(--btn-primary-text, #fff);
  box-shadow: 0 2px 6px rgba(44, 24, 16, 0.2);
}

.sort-select {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
}

.sort-select select {
  background: transparent;
  border: none;
  color: var(--color-text-main, #2C1810);
  font-weight: 500;
  cursor: pointer;
  outline: none;
}

/* ========== Pinned ========== */
.pinned-section {
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
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
  color: var(--color-secondary, #804030);
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
  background: var(--color-card-bg, #F5EFE7);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.pinned-item:hover {
  background: var(--color-border, #E5D4C1);
}

.pinned-tag {
  flex-shrink: 0;
  padding: 2px 6px;
  background: var(--color-secondary, #804030);
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  font-size: 10px;
  font-weight: 600;
  border-radius: 3px;
}

.pinned-title {
  font-size: 14px;
  color: var(--color-text-main, #2C1810);
  font-weight: 500;
  display: block;
}

.pinned-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pinned-location {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

.pinned-location i {
  color: var(--color-accent, #B87333);
}

.pinned-time {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

/* ========== Events Tab ========== */
.event-toolbar {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 22px;
}

.event-status-filters,
.event-type-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.event-status-filters button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-panel-bg, #fff);
  color: var(--color-primary, #4B3621);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.event-status-filters button span {
  min-width: 18px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--color-card-bg, #F5EFE7);
  font-size: 11px;
}

.event-status-filters button.active {
  background: var(--color-secondary, #804030);
  border-color: var(--color-secondary, #804030);
  color: #fff;
}

.event-status-filters button.active span {
  background: rgba(255, 255, 255, 0.18);
}

.filter-chip {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-panel-bg, #fff);
  color: var(--color-primary, #4B3621);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-chip:hover {
  border-color: var(--color-accent, #B87333);
  color: var(--color-accent, #B87333);
}

.filter-chip.active {
  background: var(--color-accent, #B87333);
  border-color: var(--color-accent, #B87333);
  color: var(--btn-primary-text, #fff);
}

.events-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.event-card {
  display: grid;
  grid-template-columns: 168px 1fr;
  min-height: 180px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.25s;
  box-shadow: 0 8px 24px rgba(75, 54, 33, 0.08);
}

.event-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--event-color, #B87333) 55%, var(--color-border, #E5D4C1));
  box-shadow: 0 14px 30px rgba(75, 54, 33, 0.14);
}

.event-card.faded {
  opacity: 0.72;
}

.event-card-visual {
  position: relative;
  min-height: 180px;
  background: linear-gradient(160deg, color-mix(in srgb, var(--event-color, #B87333) 70%, #2c1810), #2c1810);
  overflow: hidden;
}

.event-card-cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.event-card-fallback {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 30% 20%, color-mix(in srgb, var(--event-color, #B87333) 45%, transparent), transparent 55%),
    linear-gradient(160deg, rgba(44, 24, 16, 0.15), rgba(44, 24, 16, 0.55));
}

.event-date-badge {
  position: absolute;
  left: 12px;
  bottom: 12px;
  z-index: 2;
  min-width: 64px;
  padding: 8px 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.94);
  color: var(--color-text-main, #2C1810);
  text-align: center;
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.18);
}

.event-date-badge .month {
  display: block;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--event-color, #B87333);
}

.event-date-badge .day {
  display: block;
  font-family: 'Cinzel', serif;
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.event-card-body {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.event-card .status-chip.live {
  color: #b91c1c;
  background: rgba(254, 226, 226, 0.95);
  border-color: rgba(248, 113, 113, 0.45);
}

.event-card .status-chip.upcoming {
  color: #b45309;
  background: rgba(254, 243, 199, 0.95);
  border-color: rgba(251, 191, 36, 0.45);
}

.event-card .status-chip.ended {
  color: #64748b;
  background: rgba(241, 245, 249, 0.95);
  border-color: rgba(148, 163, 184, 0.4);
}

.event-card-title {
  margin: 0;
  font-family: 'Merriweather', serif;
  font-size: 17px;
  line-height: 1.35;
  color: var(--color-text-main, #2C1810);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.event-card-time,
.event-card-location,
.event-card-countdown {
  margin: 0;
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-secondary, #8D7B68);
}

.event-card-countdown {
  color: var(--color-secondary, #804030);
  font-weight: 600;
}

.event-card-footer {
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--color-border-light, #F5EFE7);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.events-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 72px 20px;
  color: var(--color-text-secondary, #8D7B68);
}

.events-empty-state i {
  font-size: 48px;
  opacity: 0.35;
}

/* ========== Posts Grid ========== */
.posts-grid {
  column-count: 3;
  column-gap: 24px;
}

@media (max-width: 1024px) {
  .posts-grid { column-count: 2; }
  .events-grid { grid-template-columns: 1fr; }
}

@media (max-width: 900px) {
  .event-banner {
    height: auto;
    min-height: 0;
  }

  .banner-layout {
    grid-template-columns: 64px minmax(0, 1fr);
    padding: 12px;
    gap: 10px;
  }

  .banner-date-card {
    width: 64px;
    height: 92px;
  }

  .date-day {
    font-size: 24px;
  }

  .banner-bottom-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .banner-cover {
    width: 58%;
  }

  .banner-color-fade {
    background: linear-gradient(
      90deg,
      rgba(0, 0, 0, 0.1) 0%,
      transparent 16%,
      color-mix(in srgb, var(--event-color, #B87333) 45%, transparent) 42%,
      color-mix(in srgb, var(--event-color, #B87333) 88%, #1a100c) 100%
    );
  }
}

@media (max-width: 760px) {
  .event-card {
    grid-template-columns: 1fr;
  }

  .event-card-visual {
    min-height: 140px;
  }

  .banner-layout {
    grid-template-columns: 1fr;
  }

  .banner-date-card {
    display: none;
  }

  .banner-title {
    -webkit-line-clamp: 2;
  }

  .meta-inline {
    white-space: normal;
  }
}

@media (max-width: 600px) {
  .posts-grid { column-count: 1; }
  .feed-tabs { width: 100%; }
  .feed-tab { flex: 1; justify-content: center; }
}

.post-card {
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
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

.category-tag {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  padding: 3px 8px;
  background: var(--color-card-bg, #F5EFE7);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  color: var(--color-accent, #B87333);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

.cat-guild { background: var(--cat-guild-bg, #EBF5FF); color: var(--cat-guild-color, #1D4ED8); border-color: var(--cat-guild-border, #BFDBFE); }
.cat-report { background: var(--cat-report-bg, #F5EFE7); color: var(--cat-report-color, #B87333); border-color: var(--cat-report-border, #E5D4C1); }
.cat-event { background: var(--cat-event-bg, #FEF3C7); color: var(--cat-event-color, #D97706); border-color: var(--cat-event-border, #FDE68A); }
.cat-profile { background: var(--cat-profile-bg, #F0FDF4); color: var(--cat-profile-color, #16A34A); border-color: var(--cat-profile-border, #BBF7D0); }
.cat-novel { background: var(--cat-novel-bg, #FDF4FF); color: var(--cat-novel-color, #A855F7); border-color: var(--cat-novel-border, #E9D5FF); }
.cat-item { background: var(--cat-item-bg, #FFF7ED); color: var(--cat-item-color, #EA580C); border-color: var(--cat-item-border, #FED7AA); }
.cat-other { background: var(--cat-other-bg, #F3F4F6); color: var(--cat-other-color, #6B7280); border-color: var(--cat-other-border, #E5E7EB); }

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

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid var(--color-card-bg, #F5EFE7);
  margin-top: auto;
}

.author-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.author-avatar {
  width: 32px;
  height: 32px;
  min-width: 32px;
  max-width: 32px;
  min-height: 32px;
  max-height: 32px;
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--color-accent, #B87333), var(--color-secondary, #804030));
  border-radius: 6px;
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: var(--btn-primary-text, var(--color-text-light, #fff));
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
  color: var(--color-text-main, #2C1810);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-card.standard {
  display: flex;
  flex-direction: column;
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
  background: var(--color-warning-light, rgba(230, 162, 60, 0.15));
  border: 1px solid var(--color-warning-border, rgba(217, 119, 6, 0.35));
  color: var(--color-warning, #B45309);
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
  color: var(--color-text-main, #2C1810);
  margin: 6px 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color 0.2s;
}

.post-card.standard:hover .post-title {
  color: var(--color-secondary, #804030);
}

.post-card.standard .post-excerpt {
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
  line-height: 1.5;
  margin-bottom: 10px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-location {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  margin-bottom: 10px;
  max-width: 100%;
  border-radius: 999px;
  border: 1px solid var(--color-border-light, #F5EFE7);
  background: var(--color-primary-light, rgba(184, 115, 51, 0.12));
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary, #804030);
  line-height: 1.4;
}

.post-location i {
  color: var(--color-accent, #B87333);
}

.post-card.standard .card-footer {
  padding-top: 10px;
  border-top: 1px solid var(--color-border-light, #F5EFE7);
}

.post-card.standard .author-name {
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

.comment-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

.empty-state {
  column-span: all;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--color-text-secondary, #8D7B68);
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

.pagination {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 40px;
}

.page-btn {
  padding: 8px 16px;
  background: var(--color-panel-bg, #fff);
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 6px;
  color: var(--color-primary, #4B3621);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  border-color: var(--color-accent, #B87333);
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 60px;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 16px;
}

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

@media (prefers-reduced-motion: reduce) {
  .anim-item,
  .animate-in .anim-item,
  .post-card,
  .event-card,
  .banner-cta {
    animation: none !important;
    transition: none !important;
    opacity: 1;
    transform: none;
  }
}
</style>
