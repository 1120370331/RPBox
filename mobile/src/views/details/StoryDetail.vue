<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { resolveApiUrl } from '@/api/image'
import { getCharacter, type Character } from '@/api/character'
import {
  createBookmark,
  deleteBookmark,
  deleteStoryEntry,
  getStory,
  listBookmarks,
  updateBookmark,
  updateStoryEntry,
  type Story,
  type StoryBookmark,
  type StoryEntry,
} from '@/api/story'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const loading = ref(false)
const story = ref<Story | null>(null)
const entries = ref<StoryEntry[]>([])
const charactersMap = ref<Map<number, Character>>(new Map())
const bookmarks = ref<StoryBookmark[]>([])
const failedAvatarEntryIds = ref<Set<number>>(new Set())

const manageMode = ref(false)
const selectedEntryIds = ref<number[]>([])
const bookmarkPanelOpen = ref(false)

const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const showBatchDeleteDialog = ref(false)
const showBookmarkDialog = ref(false)
const showDeleteBookmarkDialog = ref(false)

const editingEntry = ref<StoryEntry | null>(null)
const deletingEntryId = ref<number | null>(null)
const editingBookmark = ref<StoryBookmark | null>(null)
const deletingBookmarkId = ref<number | null>(null)
const bookmarkTargetEntryId = ref<number | null>(null)

const saving = ref(false)
const deleting = ref(false)
const batchDeleting = ref(false)
const bookmarkSaving = ref(false)
const bookmarkDeleting = ref(false)

const editType = ref<StoryEntry['type']>('dialogue')
const editSpeaker = ref('')
const editContent = ref('')
const editChannel = ref('SAY')
const editTimestamp = ref('')

const bookmarkName = ref('')
const bookmarkColor = ref('')
const bookmarkIsPublic = ref(false)
const bookmarkColors = ['#E57373', '#F06292', '#BA68C8', '#64B5F6', '#4DB6AC', '#81C784', '#FFD54F', '#FFB74D']

const storyId = computed(() => Number(route.params.id))
const canManage = computed(() => !!story.value && !!userStore.user && (story.value.user_id === userStore.user.id || userStore.isAdmin || userStore.isModerator))
const isAllSelected = computed(() => entries.value.length > 0 && selectedEntryIds.value.length === entries.value.length)
const sortedBookmarks = computed(() => [...bookmarks.value].sort((a, b) => Number(b.is_favorite) - Number(a.is_favorite) || new Date(a.created_at).getTime() - new Date(b.created_at).getTime()))
const publicBookmarks = computed(() => sortedBookmarks.value.filter((b) => b.is_public))
const myBookmarks = computed(() => sortedBookmarks.value.filter((b) => !b.is_public))

const normalizedEntries = computed(() => entries.value.map((entry) => {
  const imageEntry = parseImageEntry(entry)
  const c = getEntryCharacter(entry)
  return {
    ...entry,
    speakerName: getEntrySpeakerName(entry),
    avatar: getEntryAvatar(entry),
    nameColor: c?.custom_color || c?.color || '',
    channelLabel: getChannelLabel(entry.channel || ''),
    channelTextColor: getChannelTextColor(entry.channel || ''),
    imageUrl: imageEntry?.image || '',
    imageDescription: imageEntry?.description || '',
  }
}))

const groupedEntries = computed(() => {
  const groups = new Map<string, { key: string; title: string; color: string; items: typeof normalizedEntries.value }>()
  for (const entry of normalizedEntries.value) {
    const gName = (entry.group_name || '').trim()
    const gColor = (entry.background_color || '').trim()
    const key = gName ? `name:${gName}` : (gColor ? `color:${gColor}` : 'ungrouped')
    if (!groups.has(key)) groups.set(key, { key, title: gName || (gColor ? `${t('stories.detail.grouping')} ${gColor}` : t('stories.detail.ungrouped')), color: gColor, items: [] as any })
    groups.get(key)!.items.push(entry as any)
  }
  return Array.from(groups.values())
})

async function loadDetail() {
  if (!storyId.value) return
  loading.value = true
  try {
    const res = await getStory(storyId.value)
    story.value = res.story
    entries.value = (res.entries || []).sort((a, b) => a.sort_order - b.sort_order)
    failedAvatarEntryIds.value = new Set()
    await Promise.all([loadCharacters(entries.value), loadBookmarks()])
  } catch (error) {
    toast.error((error as Error)?.message || t('common.status.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadBookmarks() { const res = await listBookmarks(storyId.value); bookmarks.value = res.bookmarks || [] }

async function loadCharacters(list: StoryEntry[]) {
  const ids = Array.from(new Set(list.map((e) => e.character_id).filter((id): id is number => typeof id === 'number')))
  await Promise.all(ids.map(async (id) => { try { charactersMap.value.set(id, await getCharacter(id)) } catch {} }))
}

function getEntryCharacter(entry: StoryEntry) { return entry.character_id ? charactersMap.value.get(entry.character_id) : undefined }
function getEntrySpeakerName(entry: StoryEntry) {
  if (entry.type === 'narration') return t('stories.narrator')
  const c = getEntryCharacter(entry)
  if (!c) return entry.speaker || t('stories.narrator')
  if (c.custom_name) return c.custom_name
  if (c.first_name) return c.last_name ? `${c.first_name} ${c.last_name}` : c.first_name
  return c.game_id?.split('-')[0] || entry.speaker || t('stories.narrator')
}
function normalizeIconName(value: string) {
  const trimmed = value.trim(); if (!trimmed) return ''
  const textureMatch = trimmed.match(/\|T([^:|]+)(?::\d+)?\|t/i)
  const source = textureMatch ? textureMatch[1] : trimmed
  let name = source.replace(/\\/g, '/')
  if (name.toLowerCase().startsWith('interface/icons/')) name = name.slice('interface/icons/'.length)
  name = (name.split('/').pop() || name).replace(/\.(blp|tga|png|jpg|jpeg)$/i, '').toLowerCase().trim()
  return /^[a-z0-9_-]+$/.test(name) ? name : ''
}
function getEntryAvatar(entry: StoryEntry) {
  const c = getEntryCharacter(entry)
  if (c?.custom_avatar) {
    const custom = c.custom_avatar.trim()
    const isUrl =
      custom.startsWith('http://') ||
      custom.startsWith('https://') ||
      custom.startsWith('data:') ||
      custom.startsWith('blob:') ||
      custom.startsWith('file:') ||
      custom.startsWith('/')
    if (isUrl) return resolveApiUrl(custom)
    const customIcon = normalizeIconName(custom)
    if (customIcon) return resolveApiUrl(`/api/v1/icons/${customIcon}`)
  }
  const iconName = normalizeIconName(c?.icon || '')
  return iconName ? resolveApiUrl(`/api/v1/icons/${iconName}`) : ''
}

function handleEntryAvatarError(entryId: number) {
  failedAvatarEntryIds.value.add(entryId)
}
function getChannelLabel(channel: string) {
  const map: Record<string, string> = { SAY: t('stories.channel.say'), YELL: t('stories.channel.yell'), EMOTE: t('stories.channel.emote'), TEXT_EMOTE: t('stories.channel.emote'), PARTY: t('stories.channel.party'), RAID: t('stories.channel.raid'), WHISPER: t('stories.channel.whisper'), CHAT_MSG_SAY: t('stories.channel.say'), CHAT_MSG_YELL: t('stories.channel.yell'), CHAT_MSG_EMOTE: t('stories.channel.emote'), CHAT_MSG_TEXT_EMOTE: t('stories.channel.emote'), CHAT_MSG_PARTY: t('stories.channel.party'), CHAT_MSG_RAID: t('stories.channel.raid'), CHAT_MSG_WHISPER: t('stories.channel.whisper') }
  return map[channel] || channel
}
function getChannelTextColor(channel: string) { const map: Record<string, string> = { YELL: '#E14E4E', WHISPER: '#9A78C5', EMOTE: '#C77922', TEXT_EMOTE: '#C77922', PARTY: '#4A76C7', RAID: '#C17C17', CHAT_MSG_YELL: '#E14E4E', CHAT_MSG_WHISPER: '#9A78C5', CHAT_MSG_EMOTE: '#C77922', CHAT_MSG_TEXT_EMOTE: '#C77922', CHAT_MSG_PARTY: '#4A76C7', CHAT_MSG_RAID: '#C17C17' }; return map[channel] || '' }
function parseImageEntry(entry: StoryEntry) {
  const raw = String(entry.content || '').trim(); if (!raw) return null
  if (raw.startsWith('{') && raw.endsWith('}')) { try { const p = JSON.parse(raw) as any; const img = p.image || p.url || ''; if (img) return { image: img.startsWith('data:') ? img : resolveApiUrl(img), description: String(p.description || p.caption || p.text || '') } } catch {} }
  if (entry.type === 'image') return { image: raw.startsWith('data:') ? raw : resolveApiUrl(raw), description: '' }
  return null
}
function formatTime(value: string) { if (!value) return '--'; const d = new Date(value); return `${String(d.getMonth() + 1).padStart(2, '0')}/${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}` }
function formatDateTimeLocal(value: string) { if (!value) return ''; const d = new Date(value); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}T${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}` }
function toggleManageMode() { manageMode.value = !manageMode.value; selectedEntryIds.value = [] }
function toggleEntrySelection(id: number) { const i = selectedEntryIds.value.indexOf(id); if (i === -1) selectedEntryIds.value.push(id); else selectedEntryIds.value.splice(i, 1) }
function toggleSelectAll() { selectedEntryIds.value = isAllSelected.value ? [] : entries.value.map((e) => e.id) }
function openEditDialog(entry: StoryEntry) { editingEntry.value = entry; editType.value = entry.type; editSpeaker.value = entry.speaker || ''; editContent.value = entry.content || ''; editChannel.value = entry.channel || 'SAY'; editTimestamp.value = formatDateTimeLocal(entry.timestamp || entry.created_at || ''); showEditDialog.value = true }

async function submitEntryEdit() {
  if (!editingEntry.value || saving.value) return
  saving.value = true
  try {
    const payload: Parameters<typeof updateStoryEntry>[2] = { content: editContent.value, timestamp: editTimestamp.value ? new Date(editTimestamp.value).toISOString() : undefined }
    if (editType.value !== 'image') { payload.speaker = editSpeaker.value; payload.channel = editChannel.value } else { payload.speaker = ''; payload.channel = '' }
    const updated = await updateStoryEntry(storyId.value, editingEntry.value.id, payload)
    entries.value = entries.value.map((e) => e.id === updated.id ? updated : e)
    showEditDialog.value = false; editingEntry.value = null
    toast.success(t('stories.detail.editSuccess'))
  } catch (error) { toast.error((error as Error)?.message || t('stories.detail.editFailed')) }
  finally { saving.value = false }
}

function openDeleteDialog(entryId: number) { deletingEntryId.value = entryId; showDeleteDialog.value = true }
async function confirmDelete() {
  if (!deletingEntryId.value || deleting.value) return
  deleting.value = true
  try {
    await deleteStoryEntry(storyId.value, deletingEntryId.value)
    entries.value = entries.value.filter((e) => e.id !== deletingEntryId.value)
    bookmarks.value = bookmarks.value.filter((b) => b.entry_id !== deletingEntryId.value)
    showDeleteDialog.value = false; deletingEntryId.value = null
    toast.success(t('stories.detail.deleteSuccess'))
  } catch (error) { toast.error((error as Error)?.message || t('stories.detail.deleteFailed')) }
  finally { deleting.value = false }
}

async function confirmBatchDelete() {
  if (!selectedEntryIds.value.length || batchDeleting.value) return
  batchDeleting.value = true
  try {
    const targets = [...selectedEntryIds.value]
    const results = await Promise.allSettled(targets.map((id) => deleteStoryEntry(storyId.value, id)))
    const deletedIds = targets.filter((_, i) => results[i].status === 'fulfilled')
    entries.value = entries.value.filter((e) => !deletedIds.includes(e.id))
    bookmarks.value = bookmarks.value.filter((b) => !deletedIds.includes(b.entry_id))
    selectedEntryIds.value = []; showBatchDeleteDialog.value = false
    const failed = targets.length - deletedIds.length
    if (failed > 0) toast.warning(t('stories.detail.batchDeletePartial', { n: failed })); else toast.success(t('stories.detail.batchDeleteSuccess'))
  } catch (error) { toast.error((error as Error)?.message || t('stories.detail.deleteFailed')) }
  finally { batchDeleting.value = false }
}

function openAddBookmark(entryId: number) { editingBookmark.value = null; bookmarkTargetEntryId.value = entryId; bookmarkName.value = `${t('stories.detail.bookmarks')} ${formatTime(new Date().toISOString())}`; bookmarkColor.value = ''; bookmarkIsPublic.value = false; showBookmarkDialog.value = true }
function openEditBookmark(bookmark: StoryBookmark) { editingBookmark.value = bookmark; bookmarkTargetEntryId.value = bookmark.entry_id; bookmarkName.value = bookmark.name; bookmarkColor.value = bookmark.color || ''; bookmarkIsPublic.value = bookmark.is_public; showBookmarkDialog.value = true }
function openDeleteBookmark(bookmarkId: number) { deletingBookmarkId.value = bookmarkId; showDeleteBookmarkDialog.value = true }

async function saveBookmark() {
  if (!bookmarkTargetEntryId.value || !bookmarkName.value.trim() || bookmarkSaving.value) return
  bookmarkSaving.value = true
  try {
    if (editingBookmark.value) await updateBookmark(storyId.value, editingBookmark.value.id, { name: bookmarkName.value.trim(), color: bookmarkColor.value || '' })
    else await createBookmark(storyId.value, bookmarkTargetEntryId.value, bookmarkName.value.trim(), bookmarkColor.value || '', bookmarkIsPublic.value)
    await loadBookmarks()
    showBookmarkDialog.value = false
    toast.success(t('stories.detail.bookmarkSaved'))
  } catch (error) { toast.error((error as Error)?.message || t('stories.detail.bookmarkFailed')) }
  finally { bookmarkSaving.value = false }
}

async function toggleBookmarkFavorite(bookmark: StoryBookmark) {
  try {
    await updateBookmark(storyId.value, bookmark.id, { is_favorite: !bookmark.is_favorite })
    await loadBookmarks()
  } catch (error) {
    toast.error((error as Error)?.message || t('stories.detail.bookmarkFailed'))
  }
}

async function confirmDeleteBookmark() {
  if (!deletingBookmarkId.value || bookmarkDeleting.value) return
  bookmarkDeleting.value = true
  try {
    await deleteBookmark(storyId.value, deletingBookmarkId.value)
    bookmarks.value = bookmarks.value.filter((b) => b.id !== deletingBookmarkId.value)
    showDeleteBookmarkDialog.value = false; deletingBookmarkId.value = null
    toast.success(t('stories.detail.bookmarkDeleted'))
  } catch (error) { toast.error((error as Error)?.message || t('stories.detail.bookmarkFailed')) }
  finally { bookmarkDeleting.value = false }
}

function scrollToEntry(entryId: number) {
  const el = document.getElementById(`entry-${entryId}`)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function getBookmarkPreview(entryId: number) {
  const entry = entries.value.find((e) => e.id === entryId)
  if (!entry) return ''
  if (entry.type === 'image') return '[Image]'
  const text = String(entry.content || '').replace(/<[^>]+>/g, '').trim()
  return text.length > 22 ? `${text.slice(0, 22)}...` : text
}

onMounted(loadDetail)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('stories.detailTitle') }}</h1>
      <button v-if="canManage" class="manage-btn" @click="toggleManageMode">{{ manageMode ? $t('stories.detail.exitManage') : $t('stories.detail.manage') }}</button>
    </header>

    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!story" class="hint">{{ $t('stories.empty') }}</div>

      <template v-else>
        <section class="story-summary">
          <h2>{{ story.title }}</h2>
          <p v-if="story.description">{{ story.description }}</p>
          <div class="meta-row">
            <span><i class="ri-file-text-line" /> {{ $t('stories.entryCount', { n: entries.length }) }}</span>
            <span><i class="ri-eye-line" /> {{ story.view_count }}</span>
          </div>
        </section>

        <section v-if="manageMode" class="batch-bar">
          <button class="batch-btn" @click="toggleSelectAll">{{ isAllSelected ? $t('stories.detail.unselectAll') : $t('stories.detail.selectAll') }}</button>
          <span class="batch-count">{{ $t('stories.detail.selectedCount', { n: selectedEntryIds.length }) }}</span>
          <button class="batch-btn danger" :disabled="selectedEntryIds.length === 0" @click="showBatchDeleteDialog = true">{{ $t('stories.detail.batchDelete') }}</button>
        </section>

        <section v-for="group in groupedEntries" :key="group.key" class="group-block">
          <div class="group-title" :style="group.color ? { borderLeftColor: group.color } : {}">{{ group.title }}</div>
          <div class="entry-list">
            <article
              v-for="entry in group.items"
              :id="`entry-${entry.id}`"
              :key="entry.id"
              class="entry-item"
              :class="{ selected: manageMode && selectedEntryIds.includes(entry.id) }"
              :style="entry.background_color ? { backgroundColor: entry.background_color } : undefined"
              @click="manageMode ? toggleEntrySelection(entry.id) : null"
            >
              <div class="entry-avatar">
                <img
                  v-if="entry.avatar && !failedAvatarEntryIds.has(entry.id)"
                  :src="entry.avatar"
                  alt=""
                  @error="handleEntryAvatarError(entry.id)"
                />
                <span v-else>{{ entry.speakerName.slice(0, 1) }}</span>
              </div>
              <div class="entry-main">
                <header class="entry-head">
                  <div class="name-row">
                    <strong :style="entry.nameColor ? { color: '#' + entry.nameColor.replace('#', '') } : {}">{{ entry.speakerName }}</strong>
                    <span v-if="entry.channel && entry.type !== 'narration' && entry.type !== 'image'" class="channel-tag">[{{ entry.channelLabel }}]</span>
                  </div>
                  <time>{{ formatTime(entry.timestamp || entry.created_at || '') }}</time>
                </header>
                <div v-if="entry.imageUrl" class="entry-media"><img :src="entry.imageUrl" :alt="entry.imageDescription || 'story image'" loading="lazy" /><p v-if="entry.imageDescription" class="entry-image-desc">{{ entry.imageDescription }}</p></div>
                <p v-else class="entry-text" :style="entry.channelTextColor ? { color: entry.channelTextColor } : undefined">{{ entry.content }}</p>
              </div>
              <div v-if="manageMode" class="entry-checkbox"><input type="checkbox" :checked="selectedEntryIds.includes(entry.id)" @click.stop @change="toggleEntrySelection(entry.id)" /></div>
              <div v-else-if="canManage" class="entry-actions">
                <button class="icon-btn" @click.stop="openAddBookmark(entry.id)"><i class="ri-bookmark-line" /></button>
                <button class="icon-btn" @click.stop="openEditDialog(entry)"><i class="ri-edit-line" /></button>
                <button class="icon-btn danger" @click.stop="openDeleteDialog(entry.id)"><i class="ri-delete-bin-line" /></button>
              </div>
            </article>
          </div>
        </section>
      </template>
    </div>

    <button
      v-if="story"
      class="bookmark-fab"
      :class="{ active: bookmarkPanelOpen }"
      @click="bookmarkPanelOpen = !bookmarkPanelOpen"
    >
      <i class="ri-bookmark-line" />
      <span>{{ sortedBookmarks.length }}</span>
    </button>

    <div v-if="bookmarkPanelOpen" class="bookmark-popup-mask" @click="bookmarkPanelOpen = false">
      <section class="bookmark-popup" @click.stop>
        <header class="bookmark-popup-head">
          <strong>{{ $t('stories.detail.bookmarks') }}</strong>
          <button class="icon-btn" @click="bookmarkPanelOpen = false"><i class="ri-close-line" /></button>
        </header>

        <div class="bookmark-list">
          <div v-if="publicBookmarks.length > 0" class="bookmark-group-title">{{ $t('stories.detail.publicBookmarks') }}</div>
          <button v-for="bookmark in publicBookmarks" :key="`pub-${bookmark.id}`" class="bookmark-item" :style="bookmark.color ? { borderLeftColor: bookmark.color } : {}" @click="scrollToEntry(bookmark.entry_id)">
            <div class="bookmark-main">
              <strong>{{ bookmark.name }}</strong>
              <p>{{ getBookmarkPreview(bookmark.entry_id) }}</p>
            </div>
            <div class="bookmark-actions">
              <button class="icon-btn" @click.stop="openEditBookmark(bookmark)"><i class="ri-edit-line" /></button>
              <button class="icon-btn danger" @click.stop="openDeleteBookmark(bookmark.id)"><i class="ri-delete-bin-line" /></button>
            </div>
          </button>

          <div class="bookmark-group-title">{{ $t('stories.detail.myBookmarks') }}</div>
          <button v-for="bookmark in myBookmarks" :key="`mine-${bookmark.id}`" class="bookmark-item" :style="bookmark.color ? { borderLeftColor: bookmark.color } : {}" @click="scrollToEntry(bookmark.entry_id)">
            <div class="bookmark-main">
              <strong>{{ bookmark.name }}</strong>
              <p>{{ getBookmarkPreview(bookmark.entry_id) }}</p>
            </div>
            <div class="bookmark-actions">
              <button v-if="!bookmark.is_auto" class="icon-btn" @click.stop="toggleBookmarkFavorite(bookmark)">
                <i :class="bookmark.is_favorite ? 'ri-star-fill' : 'ri-star-line'" />
              </button>
              <button v-if="!bookmark.is_auto" class="icon-btn" @click.stop="openEditBookmark(bookmark)"><i class="ri-edit-line" /></button>
              <button v-if="!bookmark.is_auto" class="icon-btn danger" @click.stop="openDeleteBookmark(bookmark.id)"><i class="ri-delete-bin-line" /></button>
            </div>
          </button>
          <div v-if="sortedBookmarks.length === 0" class="bookmark-empty">{{ $t('stories.detail.noBookmarks') }}</div>
        </div>
      </section>
    </div>

    <div v-if="showEditDialog" class="dialog-mask"><div class="dialog"><h3>{{ $t('stories.detail.editEntry') }}</h3><div class="form-grid">
      <label v-if="editType !== 'image'"><span>{{ $t('stories.detail.speaker') }}</span><input v-model="editSpeaker" /></label>
      <label v-if="editType !== 'image'"><span>{{ $t('stories.detail.channel') }}</span><select v-model="editChannel"><option value="SAY">{{ $t('stories.channel.say') }}</option><option value="YELL">{{ $t('stories.channel.yell') }}</option><option value="WHISPER">{{ $t('stories.channel.whisper') }}</option><option value="EMOTE">{{ $t('stories.channel.emote') }}</option><option value="PARTY">{{ $t('stories.channel.party') }}</option><option value="RAID">{{ $t('stories.channel.raid') }}</option></select></label>
      <label><span>{{ $t('stories.detail.time') }}</span><input v-model="editTimestamp" type="datetime-local" /></label>
      <label class="full"><span>{{ $t('stories.detail.content') }}</span><textarea v-model="editContent" rows="5" /></label>
    </div><div class="dialog-actions"><button class="action-btn" @click="showEditDialog = false">{{ $t('stories.detail.cancel') }}</button><button class="action-btn primary" :disabled="saving" @click="submitEntryEdit">{{ $t('stories.detail.save') }}</button></div></div></div>

    <div v-if="showDeleteDialog" class="dialog-mask"><div class="dialog"><h3>{{ $t('stories.detail.deleteTitle') }}</h3><p>{{ $t('stories.detail.deleteMessage') }}</p><div class="dialog-actions"><button class="action-btn" @click="showDeleteDialog = false">{{ $t('stories.detail.cancel') }}</button><button class="action-btn danger" :disabled="deleting" @click="confirmDelete">{{ $t('stories.detail.confirm') }}</button></div></div></div>

    <div v-if="showBatchDeleteDialog" class="dialog-mask"><div class="dialog"><h3>{{ $t('stories.detail.batchDeleteTitle') }}</h3><p>{{ $t('stories.detail.batchDeleteMessage', { n: selectedEntryIds.length }) }}</p><div class="dialog-actions"><button class="action-btn" @click="showBatchDeleteDialog = false">{{ $t('stories.detail.cancel') }}</button><button class="action-btn danger" :disabled="batchDeleting" @click="confirmBatchDelete">{{ $t('stories.detail.confirm') }}</button></div></div></div>

    <div v-if="showBookmarkDialog" class="dialog-mask"><div class="dialog"><h3>{{ editingBookmark ? $t('stories.detail.editBookmark') : $t('stories.detail.addBookmark') }}</h3><div class="form-grid">
      <label class="full"><span>{{ $t('stories.detail.bookmarkName') }}</span><input v-model="bookmarkName" /></label>
      <label class="full"><span>{{ $t('stories.detail.bookmarkColor') }}</span><div class="bookmark-color-row"><button v-for="color in bookmarkColors" :key="color" class="bookmark-color" :style="{ backgroundColor: color }" :class="{ active: bookmarkColor === color }" @click="bookmarkColor = color" /><button class="bookmark-color none" :class="{ active: !bookmarkColor }" @click="bookmarkColor = ''">-</button></div></label>
      <label v-if="canManage && !editingBookmark" class="full bookmark-public-toggle"><span>{{ $t('stories.detail.publicBookmarks') }}</span><input v-model="bookmarkIsPublic" type="checkbox" /></label>
    </div><div class="dialog-actions"><button class="action-btn" @click="showBookmarkDialog = false">{{ $t('stories.detail.cancel') }}</button><button class="action-btn primary" :disabled="bookmarkSaving" @click="saveBookmark">{{ $t('stories.detail.save') }}</button></div></div></div>

    <div v-if="showDeleteBookmarkDialog" class="dialog-mask"><div class="dialog"><h3>{{ $t('stories.detail.deleteBookmark') }}</h3><p>{{ $t('stories.detail.deleteBookmarkMessage') }}</p><div class="dialog-actions"><button class="action-btn" @click="showDeleteBookmarkDialog = false">{{ $t('stories.detail.cancel') }}</button><button class="action-btn danger" :disabled="bookmarkDeleting" @click="confirmDeleteBookmark">{{ $t('stories.detail.confirm') }}</button></div></div></div>
  </div>
</template>
<style scoped>
.sub-header { gap: 8px; }
.sub-header h1 { flex: 1; }
.manage-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); color: var(--text-dark); border-radius: 999px; padding: 4px 10px; font-size: 12px; }
.story-summary, .batch-bar, .group-block { background: var(--color-card-bg); border-radius: var(--radius-md); box-shadow: var(--shadow-sm); padding: 12px; margin-bottom: 12px; }
.story-summary h2 { font-size: 17px; margin-bottom: 8px; }
.story-summary p { font-size: 14px; color: var(--color-text-secondary); line-height: 1.6; }
.meta-row { margin-top: 10px; display: flex; gap: 12px; font-size: 12px; color: var(--color-text-secondary); }
.bookmark-fab {
  position: fixed;
  left: 14px;
  bottom: calc(var(--tab-bar-height) + var(--safe-bottom, 0px) + 12px);
  width: 52px;
  height: 52px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  background: var(--color-primary);
  color: var(--text-light);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.2);
  z-index: 900;
}
.bookmark-fab i { font-size: 20px; }
.bookmark-fab span {
  position: absolute;
  right: -2px;
  top: -3px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 999px;
  background: #fff;
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}
.bookmark-fab.active {
  background: var(--color-secondary);
}
.bookmark-popup-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.28);
  z-index: 920;
}
.bookmark-popup {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: calc(var(--tab-bar-height) + var(--safe-bottom, 0px) + 72px);
  max-height: min(58vh, 480px);
  overflow: auto;
  background: var(--color-card-bg);
  border-radius: 12px;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.22);
  padding: 10px;
}
.bookmark-popup-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.bookmark-popup-head strong {
  font-size: 14px;
}
.bookmark-list { display: flex; flex-direction: column; gap: 8px; }
.bookmark-group-title { font-size: 12px; color: var(--color-text-secondary); margin-top: 2px; margin-bottom: 2px; }
.bookmark-item { border: 1px solid var(--color-border-light); border-left: 4px solid var(--color-border); border-radius: 8px; background: var(--color-panel-bg); padding: 8px; display: flex; justify-content: space-between; gap: 8px; text-align: left; }
.bookmark-main strong { font-size: 12px; }
.bookmark-main p { font-size: 12px; color: var(--color-text-secondary); margin-top: 4px; }
.bookmark-empty { font-size: 12px; color: var(--color-text-secondary); text-align: center; padding: 8px; }
.batch-bar { display: flex; align-items: center; gap: 8px; }
.batch-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); border-radius: 8px; padding: 6px 8px; font-size: 12px; color: var(--text-dark); }
.batch-btn.danger { border-color: var(--btn-danger-bg); color: var(--btn-danger-bg); }
.batch-count { margin-left: auto; font-size: 12px; color: var(--color-text-secondary); }
.group-title { font-size: 12px; color: var(--color-text-secondary); border-left: 3px solid var(--color-border); padding-left: 8px; margin-bottom: 10px; }
.entry-list { display: flex; flex-direction: column; gap: 10px; }
.entry-item { display: grid; grid-template-columns: 38px minmax(0, 1fr) auto; gap: 10px; align-items: start; background: rgba(255,255,255,0.65); border-radius: var(--radius-md); padding: 10px; }
.entry-item.selected { outline: 1px solid var(--color-primary); }
.entry-avatar { width: 38px; height: 38px; border-radius: 50%; overflow: hidden; background: var(--icon-bg); display: flex; align-items: center; justify-content: center; color: var(--icon-color); font-size: 12px; font-weight: 700; }
.entry-avatar img { width: 100%; height: 100%; object-fit: cover; }
.entry-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.name-row { display: flex; align-items: center; gap: 6px; min-width: 0; }
.name-row strong { font-size: 13px; }
.channel-tag { font-size: 11px; color: var(--color-text-secondary); }
.entry-head time { font-size: 11px; color: var(--color-text-secondary); white-space: nowrap; }
.entry-text { margin-top: 6px; font-size: 14px; line-height: 1.65; color: var(--text-dark); white-space: pre-wrap; word-break: break-word; }
.entry-media { margin-top: 8px; }
.entry-media img { width: 100%; max-height: 360px; object-fit: contain; border-radius: var(--radius-sm); background: var(--input-bg); }
.entry-image-desc { margin-top: 8px; font-size: 12px; color: var(--color-text-secondary); }
.entry-actions, .bookmark-actions { display: flex; gap: 6px; }
.icon-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); color: var(--text-dark); border-radius: 8px; width: 28px; height: 28px; }
.icon-btn.danger { color: var(--btn-danger-bg); border-color: rgba(220,53,69,0.4); }
.entry-checkbox { padding-top: 4px; }
.dialog-mask { position: fixed; inset: 0; background: rgba(0,0,0,0.48); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.dialog { width: 100%; max-width: 360px; border-radius: var(--radius-md); background: var(--color-panel-bg); padding: 14px; }
.dialog h3 { font-size: 16px; }
.dialog p { margin-top: 8px; font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }
.form-grid { margin-top: 10px; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.form-grid label { display: flex; flex-direction: column; gap: 4px; }
.form-grid label.full { grid-column: 1 / -1; }
.form-grid span { font-size: 12px; color: var(--color-text-secondary); }
.form-grid input, .form-grid select, .form-grid textarea { border: 1px solid var(--color-border); border-radius: 8px; background: var(--input-bg); color: var(--text-dark); padding: 8px; font-size: 13px; }
.bookmark-public-toggle { flex-direction: row !important; justify-content: space-between; align-items: center; }
.bookmark-public-toggle input { width: 16px; height: 16px; }
.bookmark-color-row { display: flex; gap: 6px; flex-wrap: wrap; }
.bookmark-color { width: 20px; height: 20px; border-radius: 50%; border: 1px solid rgba(0,0,0,0.12); }
.bookmark-color.active { box-shadow: 0 0 0 2px var(--color-primary-light); }
.bookmark-color.none { display: inline-flex; align-items: center; justify-content: center; font-size: 12px; background: #f3f3f3; }
.dialog-actions { margin-top: 12px; display: flex; justify-content: flex-end; gap: 8px; }
.action-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); color: var(--text-dark); border-radius: 8px; padding: 8px 10px; font-size: 13px; }
.action-btn.primary { border-color: var(--color-primary); background: var(--color-primary); color: var(--text-light); }
.action-btn.danger { border-color: var(--btn-danger-bg); background: var(--btn-danger-bg); color: #fff; }
</style>
