<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listGuilds, listGuildStories, type Guild, type GuildStoryWithUploader } from '@/api/guild'
import { listPosts, POST_CATEGORIES, type PostWithAuthor } from '@/api/post'
import { getImageUrl, resolveApiUrl } from '@/api/image'
import {
  isApprovedPublicCharacterCard,
  listMyCharacterCards,
  type CharacterCard,
} from '@/api/characterCard'
import { buildCharacterCardJumpPlaceholder } from '@/utils/jumpLink'

const props = defineProps<{
  modelValue: boolean
  onInsert: (html: string) => void
  excludePostId?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

type JumpTab = 'guild' | 'post' | 'characterCard' | 'guildHome'
type GuildStoryCard = {
  guild: Guild
  story: GuildStoryWithUploader
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

const { t, locale } = useI18n()
const open = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const activeTab = ref<JumpTab>('guild')
const loading = ref(false)
const hasLoaded = ref(false)
const guilds = ref<Guild[]>([])
const guildStories = ref<GuildStoryCard[]>([])
const publicPosts = ref<PostWithAuthor[]>([])
const publicCharacterCards = ref<CharacterCard[]>([])
const postSearch = ref('')
const characterCardSearch = ref('')

const filteredPosts = computed(() => {
  const keyword = postSearch.value.trim().toLowerCase()
  return publicPosts.value.filter((post) => {
    if (props.excludePostId && post.id === props.excludePostId) return false
    if (!keyword) return true
    const author = resolveAuthorName(post).toLowerCase()
    const category = getPostLabel(post).toLowerCase()
    return post.title.toLowerCase().includes(keyword) || author.includes(keyword) || category.includes(keyword)
  })
})

const filteredCharacterCards = computed(() => {
  const keyword = characterCardSearch.value.trim().toLowerCase()
  return publicCharacterCards.value.filter((characterCard) => {
    if (!isApprovedPublicCharacterCard(characterCard)) return false
    if (!keyword) return true
    return [
      getCharacterCardName(characterCard),
      characterCard.title,
      characterCard.full_title,
      characterCard.race,
      characterCard.class,
      characterCard.summary,
    ].some(value => String(value || '').toLowerCase().includes(keyword))
  })
})

watch(open, (next) => {
  if (next) void loadAll()
})

async function loadAll() {
  if (hasLoaded.value) return
  hasLoaded.value = true
  loading.value = true
  try {
    const guildRes = await listGuilds()
    guilds.value = guildRes.guilds || []
    const storyResults = await Promise.all(
      guilds.value.map(async (guild) => {
        try {
          const res = await listGuildStories(guild.id)
          return (res.stories || []).map((story) => ({ guild, story }))
        } catch {
          return []
        }
      }),
    )
    guildStories.value = storyResults.flat()

    const [posts, characterCards] = await Promise.all([
      listPosts({
        page: 1,
        page_size: 100,
        sort: 'created_at',
        order: 'desc',
        status: 'published',
      }),
      listMyCharacterCards().catch((error) => {
        console.error('Failed to load character cards for quick jump', error)
        return { character_cards: [] }
      }),
    ])
    publicPosts.value = posts.posts || []
    publicCharacterCards.value = characterCards.character_cards || []
  } catch (error) {
    console.error('Failed to load quick jump data', error)
    guilds.value = []
    guildStories.value = []
    publicPosts.value = []
    publicCharacterCards.value = []
  } finally {
    loading.value = false
  }
}

function buildJumpCard(attrs: JumpCardAttrs) {
  const classes = ['jump-card']
  if (attrs.variant) classes.push(`jump-card--${attrs.variant}`)
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
  if (!guild.banner_url && !guild.banner) return ''
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
    v: guild.avatar_updated_at || guild.updated_at,
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
  return post.author_name?.trim() || t('community.editor.quickJumpSheet.userFallback', { id: post.author_id })
}

function getPostLabel(post: PostWithAuthor) {
  if (post.category === 'event') {
    if (post.event_type === 'server') return t('community.editor.quickJumpSheet.serverEvent')
    if (post.event_type === 'guild') return t('community.editor.quickJumpSheet.guildEvent')
    return t('community.editor.quickJumpSheet.event')
  }
  return t(`community.categories.${post.category}`, POST_CATEGORIES.find((item) => item.value === post.category)?.label || t('community.editor.quickJumpSheet.publicPost'))
}

function formatShortDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

function getCharacterCardName(characterCard: CharacterCard) {
  return characterCard.display_name?.trim()
    || [characterCard.first_name, characterCard.last_name].filter(Boolean).join(' ')
    || t('community.editor.quickJumpSheet.unnamedCharacterCard')
}

function getCharacterCardSummary(characterCard: CharacterCard) {
  return characterCard.summary?.trim()
    || [characterCard.race, characterCard.class].filter(Boolean).join(' · ')
    || t('community.editor.quickJumpSheet.characterCardSummaryMissing')
}

function insertGuildStory(item: GuildStoryCard) {
  props.onInsert(buildJumpCard({
    href: `/archives/story/${item.story.id}`,
    label: t('community.editor.quickJumpSheet.guildStory'),
    title: item.story.title || t('community.editor.quickJumpSheet.untitledStory'),
    type: 'story',
    variant: 'story-guild',
    status: item.story.status,
    guild: item.guild.name,
    guildId: item.guild.id,
    image: resolveGuildAvatar(item.guild),
  }))
  open.value = false
}

function insertPost(post: PostWithAuthor) {
  props.onInsert(buildJumpCard({
    href: `/community/post/${post.id}`,
    label: getPostLabel(post),
    title: post.title || t('community.editor.quickJumpSheet.untitledPost'),
    type: 'post',
    variant: 'post-public',
    author: resolveAuthorName(post),
    avatar: resolveApiUrl(post.author_avatar),
    image: resolvePostCover(post),
  }))
  open.value = false
}

function insertGuild(guild: Guild) {
  props.onInsert(buildJumpCard({
    href: `/guild/${guild.id}`,
    label: t('community.editor.quickJumpSheet.guildHome'),
    title: guild.name || t('community.editor.quickJumpSheet.unknownGuild'),
    type: 'guild',
    variant: 'guild-home',
    members: String(guild.member_count || 0),
    guildId: guild.id,
    avatar: resolveGuildAvatar(guild),
    image: resolveGuildBanner(guild),
  }))
  open.value = false
}

function insertCharacterCard(characterCard: CharacterCard) {
  if (!isApprovedPublicCharacterCard(characterCard)) return
  props.onInsert(buildCharacterCardJumpPlaceholder(characterCard.id))
  open.value = false
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
  <Teleport to="body">
    <Transition name="sheet-fade">
      <div v-if="open" class="quick-jump-mask" @click.self="open = false">
        <Transition name="sheet-slide">
          <section class="quick-jump-sheet">
            <div class="sheet-handle" />
            <header class="sheet-header">
              <div>
                <h3>{{ $t('community.editor.quickJumpSheet.title') }}</h3>
                <p>{{ $t('community.editor.quickJumpSheet.subtitle') }}</p>
              </div>
              <button type="button" class="icon-btn" :aria-label="$t('common.button.close')" @click="open = false">
                <i class="ri-close-line" />
              </button>
            </header>

            <div class="tab-row">
              <button type="button" :class="{ active: activeTab === 'guild' }" @click="activeTab = 'guild'">{{ $t('community.editor.quickJumpSheet.guildStory') }}</button>
              <button type="button" :class="{ active: activeTab === 'post' }" @click="activeTab = 'post'">{{ $t('community.editor.quickJumpSheet.publicPost') }}</button>
              <button type="button" :class="{ active: activeTab === 'characterCard' }" @click="activeTab = 'characterCard'">{{ $t('community.editor.quickJumpSheet.characterCards') }}</button>
              <button type="button" :class="{ active: activeTab === 'guildHome' }" @click="activeTab = 'guildHome'">{{ $t('community.editor.quickJumpSheet.guildHome') }}</button>
            </div>

            <div v-if="activeTab === 'post'" class="jump-search">
              <i class="ri-search-line" />
              <input v-model="postSearch" type="text" :placeholder="$t('community.editor.quickJumpSheet.searchPosts')">
            </div>
            <div v-else-if="activeTab === 'characterCard'" class="jump-search">
              <i class="ri-search-line" />
              <input v-model="characterCardSearch" type="search" :placeholder="$t('community.editor.quickJumpSheet.searchCharacterCards')">
            </div>

            <div class="sheet-body">
              <div v-if="loading" class="jump-empty">{{ $t('community.editor.quickJumpSheet.loading') }}</div>
              <template v-else-if="activeTab === 'guild'">
                <div v-if="guildStories.length === 0" class="jump-empty">{{ $t('community.editor.quickJumpSheet.emptyGuildStories') }}</div>
                <button
                  v-for="item in guildStories"
                  v-else
                  :key="`${item.guild.id}-${item.story.id}`"
                  type="button"
                  class="jump-item"
                  @click="insertGuildStory(item)"
                >
                  <strong>{{ item.story.title || $t('community.editor.quickJumpSheet.untitledStory') }}</strong>
                  <span>{{ item.guild.name }} · {{ item.story.status === 'draft' ? $t('community.editor.quickJumpSheet.draft') : $t('community.editor.quickJumpSheet.published') }}</span>
                </button>
              </template>

              <template v-else-if="activeTab === 'post'">
                <div v-if="filteredPosts.length === 0" class="jump-empty">{{ $t('community.editor.quickJumpSheet.emptyPosts') }}</div>
                <button
                  v-for="post in filteredPosts"
                  v-else
                  :key="post.id"
                  type="button"
                  class="jump-item"
                  @click="insertPost(post)"
                >
                  <strong>{{ post.title || $t('community.editor.quickJumpSheet.untitledPost') }}</strong>
                  <span>{{ resolveAuthorName(post) }} · {{ getPostLabel(post) }} · {{ formatShortDate(post.created_at) }}</span>
                </button>
              </template>

              <template v-else-if="activeTab === 'characterCard'">
                <div v-if="filteredCharacterCards.length === 0" class="jump-empty">
                  {{ $t('community.editor.quickJumpSheet.emptyCharacterCards') }}
                </div>
                <button
                  v-for="characterCard in filteredCharacterCards"
                  v-else
                  :key="characterCard.id"
                  type="button"
                  class="jump-item jump-item--character-card"
                  @click="insertCharacterCard(characterCard)"
                >
                  <i class="ri-id-card-line" aria-hidden="true" />
                  <span>
                    <strong>{{ getCharacterCardName(characterCard) }}</strong>
                    <small>{{ getCharacterCardSummary(characterCard) }}</small>
                  </span>
                  <i class="ri-add-circle-line" aria-hidden="true" />
                </button>
              </template>

              <template v-else>
                <div v-if="guilds.length === 0" class="jump-empty">{{ $t('community.editor.quickJumpSheet.emptyGuilds') }}</div>
                <button
                  v-for="guild in guilds"
                  v-else
                  :key="guild.id"
                  type="button"
                  class="jump-item"
                  @click="insertGuild(guild)"
                >
                  <strong>{{ guild.name }}</strong>
                  <span>{{ $t('community.editor.quickJumpSheet.memberCount', { count: guild.member_count || 0 }) }}</span>
                </button>
              </template>
            </div>
          </section>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.quick-jump-mask {
  position: fixed;
  inset: 0;
  z-index: 2500;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.52);
}

.quick-jump-sheet {
  width: 100%;
  max-height: min(82vh, 680px);
  display: flex;
  flex-direction: column;
  border-radius: 22px 22px 0 0;
  background: var(--color-card-bg);
  padding: 10px 16px calc(18px + var(--safe-bottom, 0px));
  box-shadow: 0 -18px 40px rgba(0, 0, 0, 0.18);
}

.sheet-handle {
  width: 54px;
  height: 5px;
  margin: 0 auto 14px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.45);
}

.sheet-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
}

.sheet-header h3 {
  font-size: 17px;
  margin: 0 0 5px;
}

.sheet-header p {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.icon-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-panel-bg);
  color: var(--text-dark);
}

.tab-row {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
  overflow-x: auto;
  scrollbar-width: none;
}

.tab-row::-webkit-scrollbar {
  display: none;
}

.tab-row button {
  flex: 0 0 auto;
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-panel-bg);
  color: var(--color-text-secondary);
  font-size: 12px;
}

.tab-row button.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--text-light);
}

.jump-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 9px 12px;
  background: var(--input-bg);
  min-height: 44px;
}

.jump-search input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-dark);
  font-size: 14px;
}

.sheet-body {
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  -webkit-overflow-scrolling: touch;
}

.jump-item {
  width: 100%;
  border: 1px solid var(--color-border-light);
  border-radius: 12px;
  background: var(--color-panel-bg);
  color: var(--text-dark);
  text-align: left;
  padding: 11px 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-height: 56px;
}

.jump-item strong {
  font-size: 14px;
  line-height: 1.4;
}

.jump-item span {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.4;
}

.jump-item--character-card {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 24px;
  align-items: center;
  gap: 10px;
  border-color: color-mix(in srgb, var(--color-accent) 35%, var(--color-border-light));
}

.jump-item--character-card > i:first-child {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 10px;
  background: var(--color-primary-light);
  color: var(--color-primary);
  font-size: 18px;
}

.jump-item--character-card > span {
  display: grid;
  min-width: 0;
  gap: 4px;
}

.jump-item--character-card strong,
.jump-item--character-card small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.jump-item--character-card small {
  color: var(--color-text-secondary);
  font-size: 11px;
}

.jump-item--character-card > i:last-child {
  color: var(--color-accent);
  font-size: 20px;
}

.jump-empty {
  padding: 28px 0;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.sheet-fade-enter-active,
.sheet-fade-leave-active {
  transition: opacity 0.2s ease;
}

.sheet-fade-enter-from,
.sheet-fade-leave-to {
  opacity: 0;
}

.sheet-slide-enter-active,
.sheet-slide-leave-active {
  transition: transform 0.2s ease;
}

.sheet-slide-enter-from,
.sheet-slide-leave-to {
  transform: translateY(100%);
}
</style>
