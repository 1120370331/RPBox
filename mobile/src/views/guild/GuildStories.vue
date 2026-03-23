<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  getGuild,
  listGuildMembers,
  listGuildStories,
  removeStoryFromGuild,
  type Guild,
  type GuildMember,
  type GuildStoryWithUploader,
} from '@/api/guild'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const guildId = computed(() => Number(route.params.id))

const loading = ref(false)
const noPermission = ref(false)
const guild = ref<Guild | null>(null)
const myRole = ref('')
const stories = ref<GuildStoryWithUploader[]>([])
const members = ref<GuildMember[]>([])

const searchKeyword = ref('')
const selectedUploaderId = ref<number | undefined>(undefined)
const sortBy = ref<'created_at' | 'updated_at'>('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')

const showRemoveConfirm = ref(false)
const removingStoryId = ref<number | null>(null)
const removing = ref(false)

const isAdmin = computed(() => myRole.value === 'owner' || myRole.value === 'admin')

const filteredStories = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return stories.value
  return stories.value.filter((story) =>
    `${story.title} ${story.description || ''}`.toLowerCase().includes(keyword),
  )
})

const sortedStories = computed(() => {
  const list = [...filteredStories.value]
  list.sort((a, b) => {
    const av = new Date(sortBy.value === 'updated_at' ? a.updated_at : a.created_at).getTime()
    const bv = new Date(sortBy.value === 'updated_at' ? b.updated_at : b.created_at).getTime()
    return sortOrder.value === 'desc' ? bv - av : av - bv
  })
  return list
})

async function loadGuild() {
  if (!guildId.value) return
  const res = await getGuild(guildId.value)
  guild.value = res.guild
  myRole.value = res.my_role || ''
}

async function loadMembers() {
  if (!guildId.value) return
  try {
    const res = await listGuildMembers(guildId.value)
    members.value = res.members || []
  } catch {
    members.value = []
  }
}

async function loadStories() {
  if (!guildId.value) return
  loading.value = true
  noPermission.value = false
  try {
    const res = await listGuildStories(guildId.value, selectedUploaderId.value)
    stories.value = res.stories || []
  } catch (error: any) {
    stories.value = []
    const status = error?.response?.status
    const message = String(error?.message || '')
    if (status === 403 || message.includes('403') || message.includes('无权')) {
      noPermission.value = true
    }
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push({ name: 'guild-detail', params: { id: guildId.value } })
}

function goToPosts() {
  router.push({ name: 'guild-posts', params: { id: guildId.value } })
}

function viewStory(id: number) {
  router.push({ name: 'story-detail', params: { id }, query: { from: 'guild', guildId: String(guildId.value) } })
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function durationText(story: GuildStoryWithUploader) {
  if (!story.start_time || !story.end_time) return t('guild.stories.durationUnknown')
  const start = new Date(story.start_time)
  const end = new Date(story.end_time)
  const diff = end.getTime() - start.getTime()
  if (diff <= 0) return t('guild.stories.durationUnknown')
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  if (days > 0) return t('guild.stories.durationDays', { days })
  if (hours > 0) return t('guild.stories.durationHours', { hours })
  return t('guild.stories.durationMinutes', { minutes: Math.max(1, minutes) })
}

function getTags(story: GuildStoryWithUploader) {
  if (story.tag_list?.length) return story.tag_list
  const names = String(story.tags || '').split(',').map((v) => v.trim()).filter(Boolean)
  return names.map((name) => ({ name }))
}

function askRemove(storyId: number) {
  removingStoryId.value = storyId
  showRemoveConfirm.value = true
}

async function confirmRemove() {
  if (!guildId.value || !removingStoryId.value || removing.value) return
  removing.value = true
  try {
    await removeStoryFromGuild(guildId.value, removingStoryId.value)
    stories.value = stories.value.filter((story) => story.id !== removingStoryId.value)
    showRemoveConfirm.value = false
    removingStoryId.value = null
  } finally {
    removing.value = false
  }
}

watch(selectedUploaderId, () => {
  loadStories()
})

onMounted(async () => {
  await loadGuild()
  await loadMembers()
  await loadStories()
})
</script>

<template>
  <div class="sub-page guild-stories-page">
    <header class="sub-header">
      <button class="back-btn" @click="goBack"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('guild.stories.title') }}</h1>
    </header>

    <div class="sub-body">
      <div class="switch-row">
        <button class="switch-btn" @click="goToPosts"><i class="ri-article-line" /> {{ $t('guild.stories.guildPosts') }}</button>
        <button class="switch-btn active"><i class="ri-book-2-line" /> {{ $t('guild.stories.guildStories') }}</button>
      </div>

      <div class="toolbar-card">
        <p class="guild-name">{{ guild?.name || '-' }}</p>
        <div class="search-bar">
          <i class="ri-search-line" />
          <input v-model="searchKeyword" :placeholder="$t('guild.stories.searchPlaceholder')" />
        </div>

        <div class="filter-row">
          <select v-model="selectedUploaderId" class="filter-select">
            <option :value="undefined">{{ $t('guild.stories.allMembers') }}</option>
            <option v-for="member in members" :key="member.user_id" :value="member.user_id">{{ member.username }}</option>
          </select>

          <select v-model="sortBy" class="filter-select small">
            <option value="created_at">{{ $t('guild.stories.sortCreated') }}</option>
            <option value="updated_at">{{ $t('guild.stories.sortUpdated') }}</option>
          </select>

          <button class="sort-order-btn" @click="sortOrder = sortOrder === 'desc' ? 'asc' : 'desc'">
            <i :class="sortOrder === 'desc' ? 'ri-sort-desc' : 'ri-sort-asc'" />
          </button>
        </div>

        <p class="count-text">{{ $t('guild.stories.storyCount', { n: sortedStories.length }) }}</p>
      </div>

      <div v-if="loading" class="hint">{{ $t('guild.stories.loading') }}</div>
      <div v-else-if="noPermission" class="hint">{{ $t('guild.stories.noPermission') }}</div>
      <div v-else-if="stories.length === 0" class="hint">{{ $t('guild.stories.empty') }}</div>

      <div v-else class="story-list">
        <article
          v-for="story in sortedStories"
          :key="story.id"
          class="story-card"
          @click="viewStory(story.id)"
        >
          <div class="head">
            <h3 class="title">{{ story.title }}</h3>
            <span class="time">{{ formatDate(story.created_at) }}</span>
          </div>

          <p v-if="story.description" class="desc">{{ story.description }}</p>

          <div v-if="getTags(story).length" class="tags">
            <span v-for="tag in getTags(story)" :key="tag.name" class="tag" :style="tag.color ? { background: `#${tag.color}22`, color: `#${tag.color}` } : undefined">{{ tag.name }}</span>
          </div>

          <div class="meta-row">
            <span><i class="ri-message-3-line" /> {{ $t('guild.stories.entryCount', { n: story.entry_count || 0 }) }}</span>
            <span><i class="ri-time-line" /> {{ durationText(story) }}</span>
            <span class="uploader">{{ story.added_by_username }}</span>
          </div>

          <div v-if="isAdmin" class="actions" @click.stop>
            <button class="danger-link" @click="askRemove(story.id)">{{ $t('guild.stories.removeArchive') }}</button>
          </div>
        </article>

        <div v-if="sortedStories.length === 0" class="hint in-list">{{ $t('guild.stories.noMatch') }}</div>
      </div>
    </div>

    <div v-if="showRemoveConfirm" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('guild.stories.removeArchiveTitle') }}</h3>
        <p>{{ $t('guild.stories.removeArchiveMsg') }}</p>
        <div class="dialog-actions">
          <button class="action-btn" @click="showRemoveConfirm = false">{{ $t('guild.actions.cancel') }}</button>
          <button class="action-btn danger" :disabled="removing" @click="confirmRemove">{{ $t('guild.actions.confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.guild-stories-page .sub-body { padding-top: 8px; }
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
.filter-row {
  display: grid;
  grid-template-columns: 1fr 116px 36px;
  gap: 8px;
  margin-bottom: 8px;
}
.filter-select {
  border: 1px solid var(--color-border);
  background: var(--input-bg);
  border-radius: 10px;
  padding: 8px;
  font-size: 12px;
}
.sort-order-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 10px;
  font-size: 16px;
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
.story-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.story-card {
  border: none;
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  border-radius: var(--radius-md);
  text-align: left;
  padding: 12px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.title {
  font-size: 15px;
  flex: 1;
}
.time {
  font-size: 11px;
  color: var(--color-text-secondary);
}
.desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.55;
  margin-bottom: 8px;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.tag {
  font-size: 11px;
  background: var(--tag-bg);
  color: var(--tag-text);
  border-radius: 8px;
  padding: 2px 8px;
}
.meta-row {
  display: flex;
  gap: 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
  align-items: center;
  flex-wrap: wrap;
}
.meta-row .uploader {
  margin-left: auto;
}
.actions {
  margin-top: 6px;
  display: flex;
  justify-content: flex-end;
}
.danger-link {
  border: none;
  background: transparent;
  color: var(--btn-danger-bg);
  font-size: 12px;
  padding: 2px 0;
}
.dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 20px;
}
.dialog {
  width: 100%;
  max-width: 320px;
  background: var(--color-card-bg);
  border-radius: 12px;
  padding: 14px;
}
.dialog h3 {
  font-size: 16px;
  margin-bottom: 6px;
}
.dialog p {
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
.action-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 8px;
  padding: 6px 10px;
}
.action-btn.danger {
  background: var(--btn-danger-bg);
  border-color: var(--btn-danger-bg);
  color: #fff;
}
.action-btn:disabled {
  opacity: 0.6;
}
</style>
