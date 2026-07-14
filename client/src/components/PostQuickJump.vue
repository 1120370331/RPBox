<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import type { Story } from '@/api/story'
import { listGuilds, listGuildStories, type Guild, type GuildStoryWithUploader } from '@/api/guild'
import { listPosts, type PostWithAuthor, POST_CATEGORIES } from '@/api/post'
import { getImageUrl, resolveApiUrl } from '@/api/item'
import {
  listRPDBWorks,
  resolveRPDBMediaURL,
  type RPDBWork,
  type RPDBWorkType,
} from '@/api/rpdb'

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

const activeTab = ref<'guild' | 'post' | 'rpdb' | 'guildHome'>('guild')
const loadingGuildStories = ref(false)
const loadingPosts = ref(false)
const loadingRPDBWorks = ref(false)
const loadingGuilds = ref(false)
const hasLoaded = ref(false)

const guilds = ref<Guild[]>([])
const guildStories = ref<GuildStoryCard[]>([])
const publicPosts = ref<PostWithAuthor[]>([])
const publicRPDBWorks = ref<RPDBWork[]>([])
const postSearch = ref('')
const rpdbSearch = ref('')
const rpdbType = ref<RPDBWorkType | ''>('')

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

const rpdbTypes: Array<{ value: RPDBWorkType | ''; label: string; icon: string }> = [
  { value: '', label: '全部', icon: 'ri-layout-grid-line' },
  { value: 'item_showcase', label: '魔兽物品', icon: 'ri-magic-line' },
  { value: 'transmog', label: '幻化方案', icon: 'ri-shirt-line' },
  { value: 'home_showcase', label: '家宅分享', icon: 'ri-home-heart-line' },
]

const filteredRPDBWorks = computed(() => {
  const keyword = rpdbSearch.value.trim().toLowerCase()
  return publicRPDBWorks.value.filter((work) => {
    if (rpdbType.value && work.type !== rpdbType.value) return false
    if (!keyword) return true
    return [work.title, work.summary, work.effect_description, work.author_name, getRPDBTypeLabel(work.type)]
      .some((value) => String(value || '').toLowerCase().includes(keyword))
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
  await Promise.all([
    loadGuildsAndStories(),
    loadPublicPosts(),
    loadPublicRPDBWorks(),
  ])
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

async function loadPublicRPDBWorks() {
  loadingRPDBWorks.value = true
  try {
    const pageSize = 12
    const maxPages = 20
    let page = 1
    let total = 0
    const allWorks: RPDBWork[] = []

    while (page <= maxPages) {
      const res = await listRPDBWorks({
        page,
        page_size: pageSize,
        sort: 'updated_at',
      })
      const batch = res.works || []
      if (!batch.length) break
      allWorks.push(...batch)
      total = res.total || allWorks.length
      if (allWorks.length >= total) break
      page += 1
    }

    publicRPDBWorks.value = allWorks
  } catch (error) {
    console.error('加载 RP 数据库作品失败:', error)
    publicRPDBWorks.value = []
  } finally {
    loadingRPDBWorks.value = false
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
  summary?: string
  rpdbType?: string
  views?: string
  likes?: string
  favorites?: string
  lists?: string
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
    ['data-jump-summary', attrs.summary],
    ['data-jump-rpdb-type', attrs.rpdbType],
    ['data-jump-views', attrs.views],
    ['data-jump-likes', attrs.likes],
    ['data-jump-favorites', attrs.favorites],
    ['data-jump-lists', attrs.lists],
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

function getRPDBTypeLabel(type: RPDBWorkType) {
  const labels: Record<RPDBWorkType, string> = {
    item_showcase: '魔兽物品',
    transmog: '幻化方案',
    home_showcase: '家宅分享',
  }
  return labels[type]
}

function getRPDBTypeIcon(type: RPDBWorkType) {
  const icons: Record<RPDBWorkType, string> = {
    item_showcase: 'ri-magic-line',
    transmog: 'ri-shirt-line',
    home_showcase: 'ri-home-heart-line',
  }
  return icons[type]
}

function getRPDBVariant(type: RPDBWorkType) {
  const variants: Record<RPDBWorkType, string> = {
    item_showcase: 'rpdb-item',
    transmog: 'rpdb-transmog',
    home_showcase: 'rpdb-home',
  }
  return variants[type]
}

function insertRPDBWork(work: RPDBWork) {
  props.onInsert(buildJumpCard({
    href: `/rpdb/${work.id}`,
    label: getRPDBTypeLabel(work.type),
    title: work.title || '未命名作品',
    type: 'rpdb_work',
    variant: getRPDBVariant(work.type),
    rpdbType: work.type,
    author: work.author_name || '匿名贡献者',
    avatar: resolveApiUrl(work.author_avatar),
    image: resolveRPDBMediaURL(work.cover_image),
    summary: work.summary || work.effect_description || '作者尚未填写作品摘要。',
    views: String(work.view_count || 0),
    likes: String(work.like_count || 0),
    favorites: String(work.favorite_count || 0),
    lists: String(work.list_count || 0),
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
  <RModal v-model="open" title="添加内部链接" width="820px">
    <div class="quick-jump-dialog">
      <div class="quick-jump__intro">
        <div class="quick-jump__intro-icon">
          <i class="ri-links-line"></i>
        </div>
        <div>
          <div class="quick-jump__intro-title">把站内内容插入正文</div>
          <div class="quick-jump__intro-text">选择公会剧情、公开帖子、RP 数据库作品或公会主页，生成可点击的站内内容卡片。</div>
        </div>
      </div>

      <div class="quick-jump__tabs">
        <button :class="{ active: activeTab === 'guild' }" @click="activeTab = 'guild'">公会剧情</button>
        <button :class="{ active: activeTab === 'post' }" @click="activeTab = 'post'">公开帖子</button>
        <button :class="{ active: activeTab === 'rpdb' }" @click="activeTab = 'rpdb'">RP数据库</button>
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
            <RButton size="sm" type="primary" @click="insertStory(item.story, item.guild)">插入链接</RButton>
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
            <RButton size="sm" type="primary" @click="insertPost(post)">插入链接</RButton>
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'rpdb'" class="quick-jump__body">
        <div class="jump-search">
          <i class="ri-search-line"></i>
          <input v-model="rpdbSearch" type="text" placeholder="搜索 RP 数据库作品或作者..." />
        </div>
        <div class="rpdb-type-filter" aria-label="RP数据库分类">
          <button
            v-for="item in rpdbTypes"
            :key="item.value || 'all'"
            type="button"
            :class="{ active: rpdbType === item.value }"
            @click="rpdbType = item.value"
          >
            <i :class="item.icon"></i>{{ item.label }}
          </button>
        </div>
        <div v-if="loadingRPDBWorks" class="jump-loading">加载中...</div>
        <div v-else-if="filteredRPDBWorks.length === 0" class="jump-empty">暂无匹配作品</div>
        <div v-else class="jump-list jump-list--rpdb">
          <div
            v-for="work in filteredRPDBWorks"
            :key="work.id"
            class="jump-item jump-item--rpdb"
            :class="`jump-item--${getRPDBVariant(work.type)}`"
          >
            <div class="jump-item__cover" :class="{ empty: !work.cover_image }">
              <img v-if="work.cover_image" :src="resolveRPDBMediaURL(work.cover_image)" alt="" loading="lazy" />
              <i v-else :class="getRPDBTypeIcon(work.type)"></i>
            </div>
            <div class="jump-item__info">
              <div class="jump-item__eyebrow"><i :class="getRPDBTypeIcon(work.type)"></i>{{ getRPDBTypeLabel(work.type) }}</div>
              <div class="jump-item__title">{{ work.title || '未命名作品' }}</div>
              <div class="jump-item__summary">{{ work.summary || work.effect_description || '作者尚未填写作品摘要。' }}</div>
              <div class="jump-item__meta">
                <span>{{ work.author_name || '匿名贡献者' }}</span>
                <span><i class="ri-eye-line"></i>{{ work.view_count || 0 }}</span>
                <span><i class="ri-heart-3-line"></i>{{ work.like_count || 0 }}</span>
              </div>
            </div>
            <RButton size="sm" type="primary" @click="insertRPDBWork(work)">插入链接</RButton>
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
            <RButton size="sm" type="primary" @click="insertGuild(guild)">插入链接</RButton>
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

.quick-jump__intro {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px 16px;
  border: 1px solid color-mix(in srgb, var(--color-secondary, #B87333) 28%, var(--color-border, #E5D4C1));
  border-radius: 14px;
  background:
    radial-gradient(circle at 10% 10%, color-mix(in srgb, var(--color-accent, #B87333) 16%, transparent), transparent 34%),
    linear-gradient(135deg,
      color-mix(in srgb, var(--color-secondary, #B87333) 11%, var(--color-card-bg, #fff)),
      var(--color-panel-bg, #fff));
}

.quick-jump__intro-icon {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: color-mix(in srgb, var(--color-secondary, #B87333) 18%, transparent);
  color: var(--color-primary, #804030);
  font-size: 19px;
}

.quick-jump__intro-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
  margin-bottom: 4px;
}

.quick-jump__intro-text {
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-secondary, #8D7B68);
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

.jump-list--rpdb {
  max-height: 430px;
  overflow-y: auto;
  padding-right: 4px;
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

.jump-item--rpdb {
  --rpdb-accent: #A65F2A;
  position: relative;
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr) auto;
  min-height: 92px;
  overflow: hidden;
  padding: 0 12px 0 0;
  border-left: 3px solid var(--rpdb-accent);
}

.jump-item--rpdb-transmog { --rpdb-accent: #55758B; }
.jump-item--rpdb-home { --rpdb-accent: #4F7A62; }

.jump-item__cover {
  align-self: stretch;
  min-height: 92px;
  background: color-mix(in srgb, var(--rpdb-accent) 12%, #F5EFE7);
}

.jump-item__cover img {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: block;
  object-fit: cover;
}

.jump-item__cover.empty {
  display: grid;
  place-items: center;
  color: var(--rpdb-accent);
  font-size: 26px;
}

.jump-item--rpdb .jump-item__info {
  padding: 10px 12px;
}

.jump-item__eyebrow {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--rpdb-accent);
  font-size: 10px;
  font-weight: 800;
}

.jump-item__summary {
  overflow: hidden;
  color: #8D7B68;
  font-size: 11px;
  line-height: 1.45;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.jump-item--rpdb .jump-item__meta i {
  margin-right: 3px;
  color: var(--rpdb-accent);
}

.rpdb-type-filter {
  display: flex;
  gap: 6px;
  margin-bottom: 12px;
  overflow-x: auto;
}

.rpdb-type-filter button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  background: #fff;
  color: #8D7B68;
  font-size: 11px;
  white-space: nowrap;
  cursor: pointer;
}

.rpdb-type-filter button:hover,
.rpdb-type-filter button.active {
  border-color: #B87333;
  background: rgba(184, 115, 51, 0.08);
  color: #804030;
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

@media (max-width: 640px) {
  .jump-item--rpdb {
    grid-template-columns: 64px minmax(0, 1fr);
    padding-right: 8px;
  }

  .jump-item--rpdb :deep(.r-button) {
    grid-column: 1 / -1;
    margin: 0 8px 8px;
  }
}
</style>
