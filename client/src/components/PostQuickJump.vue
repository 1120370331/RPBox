<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import type { Story } from '@/api/story'
import { listGuilds, listGuildStories, type Guild, type GuildStoryWithUploader } from '@/api/guild'
import { listPosts, type PostWithAuthor, POST_CATEGORIES } from '@/api/post'
import { getImageUrl, resolveApiUrl } from '@/api/item'

const props = defineProps<{
  modelValue: boolean
  onInsert: (html: string) => void
  excludePostId?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

type GuildStoryCard = {
  guild: Guild
  story: GuildStoryWithUploader
}

const activeTab = ref<'guild' | 'post' | 'guildHome'>('guild')
const loadingGuildStories = ref(false)
const loadingPosts = ref(false)
const loadingGuilds = ref(false)
const hasLoaded = ref(false)

const guilds = ref<Guild[]>([])
const guildStories = ref<GuildStoryCard[]>([])
const publicPosts = ref<PostWithAuthor[]>([])
const postSearch = ref('')

const open = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const filteredPosts = computed(() => {
  const keyword = postSearch.value.trim().toLowerCase()
  return publicPosts.value.filter((post) => {
    if (props.excludePostId && post.id === props.excludePostId) {
      return false
    }
    if (!keyword) return true
    const authorName = resolveAuthorName(post).toLowerCase()
    const categoryLabel = post.category === 'event'
      ? getEventTypeLabel(post.event_type).toLowerCase()
      : getCategoryLabel(post.category).toLowerCase()
    const eventLabel = post.category === 'event'
      ? getEventTypeLabel(post.event_type).toLowerCase()
      : ''
    return post.title.toLowerCase().includes(keyword) ||
      authorName.includes(keyword) ||
      categoryLabel.includes(keyword) ||
      eventLabel.includes(keyword)
  })
})

watch(() => props.modelValue, (isOpen) => {
  if (isOpen) {
    void loadAll()
  }
})

async function loadAll() {
  if (hasLoaded.value) return
  hasLoaded.value = true
  await loadGuildsAndStories()
  await loadPublicPosts()
}

async function loadGuildsAndStories() {
  loadingGuilds.value = true
  loadingGuildStories.value = true
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } catch (error) {
    console.error('加载公会失败:', error)
    guilds.value = []
  } finally {
    loadingGuilds.value = false
  }

  if (guilds.value.length === 0) {
    guildStories.value = []
    loadingGuildStories.value = false
    return
  }

  try {
    const storyResults = await Promise.all(
      guilds.value.map(async (guild) => {
        try {
          const res = await listGuildStories(guild.id)
          return (res.stories || []).map((story) => ({ guild, story }))
        } catch (error) {
          console.error('加载公会剧情失败:', error)
          return []
        }
      })
    )
    guildStories.value = storyResults.flat()
  } finally {
    loadingGuildStories.value = false
  }
}

async function loadPublicPosts() {
  loadingPosts.value = true
  try {
    const pageSize = 100
    const maxPages = 20
    let page = 1
    let total = 0
    const allPosts: PostWithAuthor[] = []

    while (page <= maxPages) {
      const res = await listPosts({
        page,
        page_size: pageSize,
        sort: 'created_at',
        order: 'desc',
        status: 'published',
      })
      const batch = res.posts || []
      if (!batch.length) break
      allPosts.push(...batch)
      total = res.total || allPosts.length
      if (allPosts.length >= total) break
      page += 1
    }

    publicPosts.value = allPosts
  } catch (error) {
    console.error('加载公开帖子失败:', error)
    publicPosts.value = []
  } finally {
    loadingPosts.value = false
  }
}

type JumpCardAttrs = {
  href: string
  label: string
  title: string
  type: string
  variant: string
  status?: string
  visibility?: string
  guild?: string
  guildId?: number
  author?: string
  avatar?: string
  members?: string
  image?: string
}

function buildJumpCard(attrs: JumpCardAttrs) {
  const classes = ['jump-card']
  if (attrs.variant) {
    classes.push(`jump-card--${attrs.variant}`)
  }
  const dataAttrs = [
    ['data-jump-href', attrs.href],
    ['data-jump-type', attrs.type],
    ['data-jump-label', attrs.label],
    ['data-jump-title', attrs.title],
    ['data-jump-variant', attrs.variant],
    ['data-jump-status', attrs.status],
    ['data-jump-visibility', attrs.visibility],
    ['data-jump-guild', attrs.guild],
    ['data-jump-guild-id', attrs.guildId],
    ['data-jump-author', attrs.author],
    ['data-jump-avatar', attrs.avatar],
    ['data-jump-members', attrs.members],
    ['data-jump-image', attrs.image],
  ]
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => ` ${key}="${escapeHtml(String(value))}"`)
    .join('')

  return `<span class="${classes.join(' ')}" role="link" tabindex="0"${dataAttrs}></span>`
}

function resolveGuildBanner(guild: Guild) {
  if (!guild.banner_url) return ''
  return getImageUrl('guild-banner', guild.id, {
    w: 600,
    q: 80,
    v: guild.banner_updated_at || guild.updated_at,
  })
}

function resolveGuildAvatar(guild: Guild) {
  if (!guild.avatar_url && !guild.avatar) return ''
  return getImageUrl('guild-avatar', guild.id, {
    w: 200,
    q: 80,
  })
}

function resolvePostCover(post: PostWithAuthor) {
  if (!post.cover_image_url && !post.cover_image) return ''
  return getImageUrl('post-cover', post.id, {
    w: 800,
    q: 80,
    v: post.cover_image_updated_at || post.updated_at,
  })
}

function resolveAuthorName(post: PostWithAuthor) {
  if (post.author_name?.trim()) return post.author_name
  if (typeof post.author_id === 'number' && !Number.isNaN(post.author_id)) {
    return `用户#${post.author_id}`
  }
  return '未知作者'
}

function getCategoryLabel(category: string) {
  const normalized = String(category || '').trim().toLowerCase()
  if (!normalized) return '其他'
  const found = POST_CATEGORIES.find((item) => item.value === normalized)
  return found ? found.label : normalized
}

function getEventTypeLabel(eventType?: string) {
  if (eventType === 'server') return '服务器'
  if (eventType === 'guild') return '公会'
  return '活动'
}

function formatShortDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function insertStory(story: Story | GuildStoryWithUploader, guild?: Guild) {
  const isGuildStory = Boolean(guild)
  const label = isGuildStory ? '公会剧情' : '我的剧情'
  const href = `/archives/story/${story.id}`
  const isPublic = 'is_public' in story ? story.is_public : true
  props.onInsert(buildJumpCard({
    href,
    label,
    title: story.title || '未命名剧情',
    type: 'story',
    variant: isGuildStory ? 'story-guild' : 'story-mine',
    status: story.status,
    visibility: isGuildStory ? undefined : (isPublic ? 'public' : 'private'),
    guild: guild?.name,
    guildId: guild?.id,
    image: guild ? resolveGuildAvatar(guild) : '',
  }))
  emit('update:modelValue', false)
}

function insertPost(post: PostWithAuthor) {
  const href = `/community/post/${post.id}`
  const label = post.category === 'event'
    ? getEventTypeLabel(post.event_type)
    : '公开帖子'
  props.onInsert(buildJumpCard({
    href,
    label,
    title: post.title || '未命名帖子',
    type: 'post',
    variant: 'post-public',
    author: resolveAuthorName(post),
    avatar: resolveApiUrl(post.author_avatar),
    image: resolvePostCover(post),
  }))
  emit('update:modelValue', false)
}

function insertGuild(guild: Guild) {
  const href = `/guild/${guild.id}`
  props.onInsert(buildJumpCard({
    href,
    label: '公会主页',
    title: guild.name || '未知公会',
    type: 'guild',
    variant: 'guild-home',
    members: String(guild.member_count || 0),
    guildId: guild.id,
    avatar: resolveGuildAvatar(guild),
    image: resolveGuildBanner(guild),
  }))
  emit('update:modelValue', false)
}

function closeDialog() {
  emit('update:modelValue', false)
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
</script>

<template>
  <RModal v-model="open" title="快速跳转" width="760px">
    <div class="quick-jump-dialog">
      <div class="quick-jump__tabs">
        <button :class="{ active: activeTab === 'guild' }" @click="activeTab = 'guild'">公会剧情</button>
        <button :class="{ active: activeTab === 'post' }" @click="activeTab = 'post'">公开帖子</button>
        <button :class="{ active: activeTab === 'guildHome' }" @click="activeTab = 'guildHome'">公会主页</button>
      </div>

      <div v-if="activeTab === 'guild'" class="quick-jump__body">
        <div v-if="loadingGuildStories" class="jump-loading">加载中...</div>
        <div v-else-if="guildStories.length === 0" class="jump-empty">暂无公会剧情</div>
        <div v-else class="jump-list">
          <div v-for="item in guildStories" :key="`${item.guild.id}-${item.story.id}`" class="jump-item">
            <div class="jump-item__info">
              <div class="jump-item__title">{{ item.story.title || '未命名剧情' }}</div>
              <div class="jump-item__meta">
                <span>{{ item.guild.name }}</span>
                <span>{{ item.story.status === 'draft' ? '草稿' : '已发布' }}</span>
              </div>
            </div>
            <RButton size="sm" type="primary" @click="insertStory(item.story, item.guild)">插入</RButton>
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'post'" class="quick-jump__body">
        <div class="jump-search">
          <i class="ri-search-line"></i>
          <input v-model="postSearch" type="text" placeholder="搜索公开帖子或作者..." />
        </div>
        <div v-if="loadingPosts" class="jump-loading">加载中...</div>
        <div v-else-if="filteredPosts.length === 0" class="jump-empty">暂无匹配帖子</div>
        <div v-else class="jump-list">
          <div v-for="post in filteredPosts" :key="post.id" class="jump-item">
            <div class="jump-item__info">
              <div class="jump-item__title">{{ post.title || '未命名帖子' }}</div>
              <div class="jump-item__meta">
                <span>{{ resolveAuthorName(post) }}</span>
                <span>{{ post.category === 'event' ? getEventTypeLabel(post.event_type) : getCategoryLabel(post.category) }}</span>
                <span>{{ formatShortDate(post.created_at) }}</span>
              </div>
            </div>
            <RButton size="sm" type="primary" @click="insertPost(post)">插入</RButton>
          </div>
        </div>
      </div>

      <div v-else class="quick-jump__body">
        <div v-if="loadingGuilds" class="jump-loading">加载中...</div>
        <div v-else-if="guilds.length === 0" class="jump-empty">暂无公会</div>
        <div v-else class="jump-list">
          <div v-for="guild in guilds" :key="guild.id" class="jump-item">
            <div class="jump-item__info">
              <div class="jump-item__title">{{ guild.name }}</div>
              <div class="jump-item__meta">
                <span>{{ guild.member_count || 0 }} 名成员</span>
              </div>
            </div>
            <RButton size="sm" type="primary" @click="insertGuild(guild)">插入</RButton>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <RButton type="secondary" @click="closeDialog">关闭</RButton>
    </template>
  </RModal>
</template>

<style scoped>
.quick-jump-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.quick-jump__tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-jump__tabs button {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid #E5D4C1;
  background: #fff;
  font-size: 12px;
  color: #8D7B68;
  cursor: pointer;
  transition: all 0.2s;
}

.quick-jump__tabs button.active,
.quick-jump__tabs button:hover {
  border-color: #B87333;
  color: #B87333;
  background: rgba(184, 115, 51, 0.08);
}

.jump-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  background: #fff;
  margin-bottom: 12px;
  color: #8D7B68;
}

.jump-search input {
  border: none;
  outline: none;
  font-size: 12px;
  width: 100%;
  color: #4B3621;
  background: transparent;
}

.jump-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.jump-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border: 1px solid #F1E6DB;
  border-radius: 10px;
  background: #fff;
}

.jump-item__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.jump-item__title {
  font-size: 13px;
  font-weight: 600;
  color: #2C1810;
}

.jump-item__meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: #8D7B68;
  flex-wrap: wrap;
}

.jump-loading,
.jump-empty {
  font-size: 12px;
  color: #8D7B68;
  padding: 12px 0;
  text-align: center;
}
</style>
