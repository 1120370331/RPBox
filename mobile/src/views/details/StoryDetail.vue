<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { getImageUrl, resolveApiUrl } from '@/api/image'
import { getCharacter, type Character } from '@/api/character'
import { createContentReport } from '@/api/safety'
import { ensureEmoteMapLoaded, renderTextWithEmotes } from '@/utils/emote'
import {
  createBookmark,
  deleteBookmark,
  deleteStoryEntry,
  getStory,
  listBookmarks,
  updateEntriesBackgroundColor,
  updateBookmark,
  updateLastViewBookmark,
  updateStoryEntry,
  type Story,
  type StoryBookmark,
  type CharacterCardSummary,
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
const characterCardsMap = ref<Map<number, CharacterCardSummary>>(new Map())
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
const showGroupDialog = ref(false)
const showReportSheet = ref(false)

const editingEntry = ref<StoryEntry | null>(null)
const deletingEntryId = ref<number | null>(null)
const editingBookmark = ref<StoryBookmark | null>(null)
const deletingBookmarkId = ref<number | null>(null)
const bookmarkTargetEntryId = ref<number | null>(null)

const saving = ref(false)
const deleting = ref(false)
const batchDeleting = ref(false)
const updatingGroup = ref(false)
const bookmarkSaving = ref(false)
const bookmarkDeleting = ref(false)
const reportSubmitting = ref(false)

const editType = ref<StoryEntry['type']>('dialogue')
const editSpeaker = ref('')
const editContent = ref('')
const editChannel = ref('SAY')
const editTimestamp = ref('')

const bookmarkName = ref('')
const bookmarkColor = ref('')
const bookmarkIsPublic = ref(false)
const bookmarkColors = ['#E57373', '#F06292', '#BA68C8', '#64B5F6', '#4DB6AC', '#81C784', '#FFD54F', '#FFB74D']
const defaultColors = [
  '#E57373', '#F06292', '#BA68C8', '#9575CD', '#7986CB',
  '#64B5F6', '#4FC3F7', '#4DD0E1', '#4DB6AC', '#81C784',
  '#AED581', '#DCE775', '#FFF176', '#FFD54F', '#FFB74D',
  '#FF8A65', '#A1887F', '#90A4AE',
]
const selectedEntryColor = ref('')
const selectedGroupName = ref('')
const emoteVersion = ref(0)
type StoryReportReason = 'story_content' | 'story_audio'
const reportReason = ref<StoryReportReason>('story_content')
const reportEntryId = ref<number>(0)
const reportDetail = ref('')
const reportReasonOptions: { value: StoryReportReason; label: string }[] = [
  { value: 'story_content', label: '剧情内容违规' },
  { value: 'story_audio', label: '音频违规' },
]

const storyId = computed(() => Number(route.params.id))
const locationRegionText = computed(() => story.value?.region?.trim() || '')
const locationAddressText = computed(() => story.value?.address?.trim() || '')
const locationText = computed(() => locationAddressText.value || locationRegionText.value)
const locationContext = computed(() => locationRegionText.value && locationAddressText.value ? locationRegionText.value : '')
const canManage = computed(() => !!story.value && !!userStore.user && (story.value.user_id === userStore.user.id || userStore.isAdmin || userStore.isModerator))
const canReportStory = computed(() => !!story.value && !!userStore.user && story.value.user_id !== userStore.user.id)
const isAllSelected = computed(() => entries.value.length > 0 && selectedEntryIds.value.length === entries.value.length)
const sortedBookmarks = computed(() => {
  const entryMap = new Map(entries.value.map((entry) => [entry.id, entry]))
  return [...bookmarks.value].sort((a, b) => {
    if (a.is_favorite !== b.is_favorite) return a.is_favorite ? -1 : 1
    if (a.is_auto !== b.is_auto) return a.is_auto ? 1 : -1

    const entryA = entryMap.get(a.entry_id)
    const entryB = entryMap.get(b.entry_id)
    const timeA = getEntrySortTime(entryA)
    const timeB = getEntrySortTime(entryB)
    if (timeA !== timeB) return timeA - timeB
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  })
})
const publicBookmarks = computed(() => sortedBookmarks.value.filter((b) => b.is_public))
const myBookmarks = computed(() => sortedBookmarks.value.filter((b) => !b.is_public))
const usedColorGroups = computed(() => {
  const colorMap = new Map<string, number>()
  for (const entry of entries.value) {
    const color = String(entry.background_color || '').trim()
    if (!color) continue
    colorMap.set(color, (colorMap.get(color) || 0) + 1)
  }
  return Array.from(colorMap.entries())
    .map(([color, count]) => ({ color, count }))
    .sort((a, b) => b.count - a.count)
})

const normalizedEntries = computed(() => entries.value.map((entry) => {
  const imageEntry = parseImageEntry(entry)
  const characterCard = getEntryCharacterCard(entry)
  return {
    ...entry,
    speakerName: getEntrySpeakerName(entry),
    avatar: getEntryAvatar(entry),
    nameColor: getEntryNameColor(entry),
    characterCardId: characterCard?.id || null,
    channelLabel: getChannelLabel(entry.channel || ''),
    channelTextColor: getChannelTextColor(entry.channel || ''),
    imageUrl: imageEntry?.image || '',
    imageDescription: imageEntry?.description || '',
  }
}))

const reportableEntries = computed(() => entries.value.map((entry, index) => ({
  entry,
  label: buildReportEntryLabel(entry, index),
})))

const selectedReportEntry = computed(() => {
  if (!reportEntryId.value) return null
  return entries.value.find((entry) => entry.id === reportEntryId.value) || null
})

const canSubmitReport = computed(() => reportEntryId.value > 0 || reportDetail.value.trim().length > 0)

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
    characterCardsMap.value = new Map(
      Object.values(res.character_cards || {}).map((card) => [card.id, card]),
    )
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
function getEntryCharacterCard(entry: StoryEntry) {
  return entry.character_card_id ? characterCardsMap.value.get(entry.character_card_id) : undefined
}
function getCharacterCardDisplayName(card: CharacterCardSummary) {
  return card.display_name?.trim()
    || [card.first_name, card.last_name].map((part) => part?.trim()).filter(Boolean).join(' ')
    || ''
}
function getEntrySpeakerName(entry: StoryEntry) {
  if (entry.type === 'narration') return t('stories.narrator')
  if (entry.speaker) return entry.speaker
  const card = getEntryCharacterCard(entry)
  const cardName = card ? getCharacterCardDisplayName(card) : ''
  if (cardName) return cardName
  const c = getEntryCharacter(entry)
  if (!c) return t('stories.narrator')
  if (c.custom_name) return c.custom_name
  if (c.first_name) return c.last_name ? `${c.first_name} ${c.last_name}` : c.first_name
  return c.game_id?.split('-')[0] || t('stories.narrator')
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
  const card = getEntryCharacterCard(entry)
  if (card?.portrait_image_url) {
    return getImageUrl('character-card-portrait', card.id, {
      w: 96,
      q: 82,
      v: card.portrait_image_updated_at || card.updated_at,
    })
  }
  const cardIcon = normalizeIconName(card?.icon || '')
  if (cardIcon) return resolveApiUrl(`/api/v1/icons/${cardIcon}`)

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

function normalizeNameColor(value: string | undefined) {
  const color = value?.trim() || ''
  if (/^[\da-f]{6}$/i.test(color)) return `#${color}`
  if (/^[\da-f]{8}$/i.test(color)) return `#${color.slice(2)}${color.slice(0, 2)}`
  if (/^#[\da-f]{6}(?:[\da-f]{2})?$/i.test(color)) return color
  return ''
}

function getEntryNameColor(entry: StoryEntry) {
  const card = getEntryCharacterCard(entry)
  if (card) return normalizeNameColor(card.name_color || card.class_color)
  const character = getEntryCharacter(entry)
  return normalizeNameColor(character?.custom_color || character?.color)
}

function openCharacterCard(cardId: number) {
  if (!Number.isInteger(cardId) || cardId <= 0) return
  void router.push({ name: 'character-card-detail', params: { id: cardId } })
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
function toggleManageMode() { manageMode.value = !manageMode.value; selectedEntryIds.value = []; showGroupDialog.value = false }
function toggleEntrySelection(id: number) { const i = selectedEntryIds.value.indexOf(id); if (i === -1) selectedEntryIds.value.push(id); else selectedEntryIds.value.splice(i, 1) }
function toggleSelectAll() { selectedEntryIds.value = isAllSelected.value ? [] : entries.value.map((e) => e.id) }

function selectByColor(color: string) {
  const ids = entries.value
    .filter(e => String(e.background_color || '').trim() === color)
    .map(e => e.id)
  if (ids.length === 0) return

  const allSelected = ids.every(id => selectedEntryIds.value.includes(id))
  if (allSelected) {
    selectedEntryIds.value = selectedEntryIds.value.filter(id => !ids.includes(id))
    return
  }

  selectedEntryIds.value = Array.from(new Set([...selectedEntryIds.value, ...ids]))
}

function openGroupDialog() {
  if (!selectedEntryIds.value.length) return
  selectedEntryColor.value = ''
  selectedGroupName.value = ''
  showGroupDialog.value = true
}

async function applyGroupSettings() {
  if (!selectedEntryIds.value.length || updatingGroup.value) return
  updatingGroup.value = true
  try {
    await updateEntriesBackgroundColor(
      storyId.value,
      selectedEntryIds.value,
      selectedEntryColor.value,
      selectedGroupName.value.trim(),
    )

    entries.value = entries.value.map((entry) => {
      if (!selectedEntryIds.value.includes(entry.id)) return entry
      return {
        ...entry,
        background_color: selectedEntryColor.value,
        group_name: selectedGroupName.value.trim(),
      }
    })

    selectedEntryIds.value = []
    manageMode.value = false
    showGroupDialog.value = false
    toast.success(t('stories.detail.groupApplySuccess'))
  } catch (error) {
    toast.error((error as Error)?.message || t('stories.detail.groupApplyFailed'))
  } finally {
    updatingGroup.value = false
  }
}

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

function jumpToBookmark(entryId: number) {
  scrollToEntry(entryId)
  bookmarkPanelOpen.value = false
}

function getBookmarkPreview(entryId: number) {
  const entry = entries.value.find((e) => e.id === entryId)
  if (!entry) return ''
  if (entry.type === 'image') return '[图片]'
  const text = String(entry.content || '').replace(/<[^>]+>/g, '').trim()
  return text.length > 22 ? `${text.slice(0, 22)}...` : text
}

function getEntrySortTime(entry?: StoryEntry) {
  if (!entry) return Number.MAX_SAFE_INTEGER
  const time = new Date(entry.timestamp || entry.created_at || '').getTime()
  return Number.isFinite(time) ? time : entry.sort_order
}

function getBookmarkTime(entryId: number) {
  const entry = entries.value.find((e) => e.id === entryId)
  if (!entry) return ''
  return formatTime(entry.timestamp || entry.created_at || '')
}

function getLastVisibleEntryId() {
  const entryElements = document.querySelectorAll<HTMLElement>('.entry-item[data-entry-id]')
  if (!entryElements.length) return null

  const viewportBottom = window.innerHeight
  let lastVisibleId: number | null = null

  entryElements.forEach((el) => {
    const rect = el.getBoundingClientRect()
    if (rect.top < viewportBottom && rect.bottom > 0) {
      const parsed = Number(el.dataset.entryId)
      if (Number.isFinite(parsed) && parsed > 0) lastVisibleId = parsed
    }
  })

  return lastVisibleId
}

async function saveLastViewBookmark() {
  if (!storyId.value) return
  const entryId = getLastVisibleEntryId()
  if (!entryId) return

  try {
    await updateLastViewBookmark(storyId.value, entryId)
  } catch (error) {
    console.error('Failed to save story last-view bookmark', error)
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    void saveLastViewBookmark()
  }
}

function truncateText(text: string, limit: number) {
  const normalized = String(text || '').replace(/\s+/g, ' ').trim()
  const chars = Array.from(normalized)
  if (chars.length <= limit) return normalized
  return `${chars.slice(0, limit).join('')}...`
}

function getReportEntryText(entry: StoryEntry) {
  if (entry.type === 'image') {
    const parsed = parseImageEntry(entry)
    return parsed?.description || '图片条目'
  }
  return entry.content || ''
}

function buildReportEntryLabel(entry: StoryEntry, index: number) {
  const channel = entry.channel && entry.type !== 'narration' && entry.type !== 'image' ? `[${getChannelLabel(entry.channel)}] ` : ''
  const speaker = getEntrySpeakerName(entry)
  const text = truncateText(getReportEntryText(entry), 56)
  return `第 ${index + 1} 条 #${entry.id} ${channel}${speaker}${text ? `：${text}` : ''}`
}

function resetReportForm() {
  reportReason.value = 'story_content'
  reportEntryId.value = 0
  reportDetail.value = ''
}

function openReportSheet() {
  if (!story.value) return
  if (!userStore.token) {
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  if (story.value.user_id === userStore.user?.id) {
    toast.error('不能举报自己的剧情')
    return
  }
  resetReportForm()
  showReportSheet.value = true
}

function closeReportSheet() {
  if (reportSubmitting.value) return
  showReportSheet.value = false
}

function buildStoryReportDetail() {
  const parts = [
    `违规类型：${reportReasonOptions.find((option) => option.value === reportReason.value)?.label || '剧情内容违规'}`,
  ]
  const entry = selectedReportEntry.value
  if (entry) {
    const entryIndex = entries.value.findIndex((item) => item.id === entry.id)
    parts.push(`辅助条目：${buildReportEntryLabel(entry, entryIndex >= 0 ? entryIndex : 0)}`)
  }
  const note = reportDetail.value.trim()
  if (note) {
    parts.push(`补充说明：${note}`)
  }
  return parts.join('\n')
}

async function submitStoryReport() {
  if (!story.value || !canSubmitReport.value || reportSubmitting.value) return
  reportSubmitting.value = true
  try {
    await createContentReport({
      target_type: 'story',
      target_id: story.value.id,
      reason: reportReason.value,
      detail: buildStoryReportDetail(),
      submit_report: true,
    })
    showReportSheet.value = false
    resetReportForm()
    toast.success('举报已提交，版主会尽快处理')
  } catch (error) {
    toast.error((error as Error)?.message || '举报提交失败')
  } finally {
    reportSubmitting.value = false
  }
}

function renderEntryTextHtml(content: string) {
  void emoteVersion.value
  return renderTextWithEmotes(content || '')
}

onMounted(async () => {
  await loadDetail()
  await ensureEmoteMapLoaded()
  emoteVersion.value += 1
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  void saveLastViewBookmark()
})
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('stories.detailTitle') }}</h1>
      <button v-if="canReportStory" class="report-btn" @click="openReportSheet"><i class="ri-alarm-warning-line" /> 举报</button>
      <button v-if="canManage" class="manage-btn" @click="toggleManageMode">{{ manageMode ? $t('stories.detail.exitManage') : $t('stories.detail.manage') }}</button>
    </header>

    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!story" class="hint">{{ $t('stories.empty') }}</div>

      <template v-else>
        <section class="story-summary">
          <h2>{{ story.title }}</h2>
          <p v-if="story.description">{{ story.description }}</p>
          <div v-if="locationText" class="story-location-banner">
            <div class="story-location-banner__icon">
              <i class="ri-map-pin-2-fill" />
            </div>
            <div class="story-location-banner__content">
              <div class="story-location-banner__top">
                <span class="story-location-banner__label">{{ $t('stories.locationLabel') }}</span>
                <span v-if="locationContext" class="story-location-banner__chip">{{ locationContext }}</span>
              </div>
              <strong class="story-location-banner__main">{{ locationText }}</strong>
            </div>
          </div>
          <div class="meta-row">
            <span><i class="ri-file-text-line" /> {{ $t('stories.entryCount', { n: entries.length }) }}</span>
            <span><i class="ri-eye-line" /> {{ story.view_count }}</span>
          </div>
        </section>

        <section v-if="manageMode" class="batch-bar">
          <div class="batch-top">
            <button class="batch-btn" @click="toggleSelectAll">{{ isAllSelected ? $t('stories.detail.unselectAll') : $t('stories.detail.selectAll') }}</button>
            <span class="batch-count">{{ $t('stories.detail.selectedCount', { n: selectedEntryIds.length }) }}</span>
            <button class="batch-btn" :disabled="selectedEntryIds.length === 0" @click="openGroupDialog">{{ $t('stories.detail.groupAction') }}</button>
            <button class="batch-btn danger" :disabled="selectedEntryIds.length === 0" @click="showBatchDeleteDialog = true">{{ $t('stories.detail.batchDelete') }}</button>
          </div>
          <div v-if="usedColorGroups.length > 0" class="color-groups">
            <span class="groups-label">{{ $t('stories.detail.selectByGroup') }}</span>
            <button
              v-for="group in usedColorGroups"
              :key="group.color"
              class="color-group-btn"
              :style="{ backgroundColor: group.color }"
              :title="`${group.count}`"
              @click="selectByColor(group.color)"
            >
              {{ group.count }}
            </button>
          </div>
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
              :data-entry-id="entry.id"
              :style="entry.background_color ? { backgroundColor: entry.background_color } : undefined"
              @click="manageMode ? toggleEntrySelection(entry.id) : null"
            >
              <button
                v-if="entry.characterCardId && !manageMode"
                type="button"
                class="entry-avatar character-card-link"
                :aria-label="$t('stories.detail.openCharacterCard', { name: entry.speakerName })"
                @click.stop="openCharacterCard(entry.characterCardId)"
              >
                <img
                  v-if="entry.avatar && !failedAvatarEntryIds.has(entry.id)"
                  :src="entry.avatar"
                  :alt="$t('stories.detail.characterCardPortraitAlt', { name: entry.speakerName })"
                  @error="handleEntryAvatarError(entry.id)"
                />
                <span v-else>{{ entry.speakerName.slice(0, 1) }}</span>
              </button>
              <div v-else class="entry-avatar">
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
                    <button
                      v-if="entry.characterCardId && !manageMode"
                      type="button"
                      class="character-card-name-link"
                      :aria-label="$t('stories.detail.openCharacterCard', { name: entry.speakerName })"
                      @click.stop="openCharacterCard(entry.characterCardId)"
                    >
                      <strong :style="entry.nameColor ? { color: entry.nameColor } : {}">{{ entry.speakerName }}</strong>
                      <i class="ri-profile-line" aria-hidden="true" />
                    </button>
                    <strong v-else :style="entry.nameColor ? { color: entry.nameColor } : {}">{{ entry.speakerName }}</strong>
                    <span v-if="entry.channel && entry.type !== 'narration' && entry.type !== 'image'" class="channel-tag">[{{ entry.channelLabel }}]</span>
                  </div>
                  <time>{{ formatTime(entry.timestamp || entry.created_at || '') }}</time>
                </header>
                <div v-if="entry.imageUrl" class="entry-media"><img :src="entry.imageUrl" :alt="entry.imageDescription || 'story image'" loading="lazy" /><p v-if="entry.imageDescription" class="entry-image-desc">{{ entry.imageDescription }}</p></div>
                <p v-else class="entry-text" :style="entry.channelTextColor ? { color: entry.channelTextColor } : undefined" v-html="renderEntryTextHtml(entry.content)" />
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
          <button v-for="bookmark in publicBookmarks" :key="`pub-${bookmark.id}`" class="bookmark-item" :class="{ 'is-auto': bookmark.is_auto, 'is-favorite': bookmark.is_favorite, 'is-public': bookmark.is_public }" :style="bookmark.color ? { borderLeftColor: bookmark.color } : {}" @click="jumpToBookmark(bookmark.entry_id)">
            <div class="bookmark-main">
              <strong><i v-if="bookmark.is_auto" class="ri-time-line" />{{ bookmark.name }}</strong>
              <p>{{ getBookmarkPreview(bookmark.entry_id) }}</p>
              <small v-if="getBookmarkTime(bookmark.entry_id)">{{ getBookmarkTime(bookmark.entry_id) }}</small>
            </div>
            <div v-if="canManage" class="bookmark-actions">
              <button class="icon-btn" @click.stop="openEditBookmark(bookmark)"><i class="ri-edit-line" /></button>
              <button class="icon-btn danger" @click.stop="openDeleteBookmark(bookmark.id)"><i class="ri-delete-bin-line" /></button>
            </div>
          </button>

          <div class="bookmark-group-title">{{ $t('stories.detail.myBookmarks') }}</div>
          <button v-for="bookmark in myBookmarks" :key="`mine-${bookmark.id}`" class="bookmark-item" :class="{ 'is-auto': bookmark.is_auto, 'is-favorite': bookmark.is_favorite }" :style="bookmark.color ? { borderLeftColor: bookmark.color } : {}" @click="jumpToBookmark(bookmark.entry_id)">
            <div class="bookmark-main">
              <strong><i v-if="bookmark.is_auto" class="ri-time-line" />{{ bookmark.name }}</strong>
              <p>{{ getBookmarkPreview(bookmark.entry_id) }}</p>
              <small v-if="getBookmarkTime(bookmark.entry_id)">{{ getBookmarkTime(bookmark.entry_id) }}</small>
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

    <Teleport to="body">
      <Transition name="sheet-fade">
        <div v-if="showReportSheet" class="report-sheet-mask" @click.self="closeReportSheet">
          <Transition name="sheet-slide">
            <section class="report-sheet">
              <div class="report-sheet-handle"></div>
              <header class="report-sheet-head">
                <div>
                  <h3>举报剧情</h3>
                  <p>{{ story?.title }}</p>
                </div>
                <button type="button" class="icon-btn" @click="closeReportSheet"><i class="ri-close-line" /></button>
              </header>

              <label class="report-field">
                <span>违规类型</span>
                <select v-model="reportReason">
                  <option v-for="option in reportReasonOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                </select>
              </label>

              <label class="report-field">
                <span>辅助定位条目</span>
                <select v-model.number="reportEntryId">
                  <option :value="0">不指定具体条目</option>
                  <option v-for="item in reportableEntries" :key="item.entry.id" :value="item.entry.id">{{ item.label }}</option>
                </select>
              </label>

              <label class="report-field">
                <span>补充说明</span>
                <textarea
                  v-model="reportDetail"
                  rows="4"
                  maxlength="500"
                  placeholder="请说明违规位置或问题表现，选择条目后也可以只填写简短说明"
                />
              </label>

              <p class="report-hint" :class="{ error: !canSubmitReport }">
                请选择具体条目或填写补充说明；音频违规建议选择触发音频的剧情条目。
              </p>

              <div class="report-actions">
                <button type="button" class="action-btn" :disabled="reportSubmitting" @click="closeReportSheet">取消</button>
                <button type="button" class="action-btn primary" :disabled="reportSubmitting || !canSubmitReport" @click="submitStoryReport">
                  {{ reportSubmitting ? '提交中...' : '提交举报' }}
                </button>
              </div>
            </section>
          </Transition>
        </div>
      </Transition>
    </Teleport>

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

    <div v-if="showGroupDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('stories.detail.groupDialogTitle') }}</h3>
        <div class="form-grid">
          <label class="full">
            <span>{{ $t('stories.detail.groupName') }}</span>
            <input v-model="selectedGroupName" :placeholder="$t('stories.detail.groupNamePlaceholder')" />
          </label>
          <label class="full">
            <span>{{ $t('stories.detail.groupColor') }}</span>
            <div class="bookmark-color-row">
              <button
                v-for="color in defaultColors"
                :key="color"
                class="bookmark-color group-color"
                :style="{ backgroundColor: color }"
                :class="{ active: selectedEntryColor === color }"
                @click.prevent="selectedEntryColor = color"
              />
              <button class="bookmark-color none group-color" :class="{ active: !selectedEntryColor }" @click.prevent="selectedEntryColor = ''">-</button>
            </div>
            <small class="group-hint">{{ $t('stories.detail.groupNameHint') }}</small>
          </label>
        </div>
        <div class="dialog-actions">
          <button class="action-btn" @click="showGroupDialog = false">{{ $t('stories.detail.cancel') }}</button>
          <button class="action-btn primary" :disabled="updatingGroup" @click="applyGroupSettings">{{ $t('stories.detail.apply') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
<style scoped>
.sub-header { gap: 8px; }
.sub-header h1 { flex: 1; }
.manage-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); color: var(--text-dark); border-radius: 999px; padding: 4px 10px; font-size: 12px; }
.report-btn { border: 1px solid rgba(220,53,69,0.36); background: rgba(220,53,69,0.08); color: var(--btn-danger-bg); border-radius: 999px; padding: 4px 9px; font-size: 12px; display: inline-flex; align-items: center; gap: 4px; white-space: nowrap; }
.story-summary, .batch-bar, .group-block { background: var(--color-card-bg); border-radius: var(--radius-md); box-shadow: var(--shadow-sm); padding: 12px; margin-bottom: 12px; }
.story-summary h2 { font-size: 17px; margin-bottom: 8px; }
.story-summary p { font-size: 14px; color: var(--color-text-secondary); line-height: 1.6; }
.story-location-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-top: 10px;
  padding: 14px;
  border-radius: 16px;
  border: 1px solid var(--color-border);
  background: linear-gradient(135deg, var(--color-primary-light) 0%, rgba(184, 115, 51, 0.12) 100%);
}
.story-location-banner__icon {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--color-primary);
  color: var(--text-light);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  flex-shrink: 0;
}
.story-location-banner__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.story-location-banner__top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.story-location-banner__label {
  font-size: 11px;
  letter-spacing: 1px;
  text-transform: uppercase;
  font-weight: 700;
  color: var(--color-accent);
}
.story-location-banner__chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 9px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.72);
  color: var(--text-dark);
  font-size: 12px;
  font-weight: 600;
}
.story-location-banner__main {
  font-size: 18px;
  line-height: 1.45;
  color: var(--text-dark);
  word-break: break-word;
}
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
.bookmark-item.is-public { background: rgba(184, 115, 51, 0.08); }
.bookmark-item.is-auto { opacity: 0.86; border-style: dashed; }
.bookmark-item.is-favorite { border-color: rgba(255, 178, 62, 0.62); box-shadow: inset 0 0 0 1px rgba(255, 178, 62, 0.1); }
.bookmark-main { min-width: 0; }
.bookmark-main strong { display: inline-flex; align-items: center; gap: 4px; max-width: 100%; font-size: 12px; }
.bookmark-main strong i { color: var(--color-accent); font-size: 13px; }
.bookmark-main p { font-size: 12px; color: var(--color-text-secondary); margin-top: 4px; }
.bookmark-main small { display: block; margin-top: 3px; color: var(--color-text-muted); font-size: 11px; }
.bookmark-empty { font-size: 12px; color: var(--color-text-secondary); text-align: center; padding: 8px; }
.batch-bar { display: flex; flex-direction: column; gap: 8px; }
.batch-top { display: flex; align-items: center; gap: 8px; width: 100%; flex-wrap: wrap; }
.batch-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); border-radius: 8px; padding: 6px 8px; font-size: 12px; color: var(--text-dark); }
.batch-btn.danger { border-color: var(--btn-danger-bg); color: var(--btn-danger-bg); }
.batch-count { margin-left: auto; font-size: 12px; color: var(--color-text-secondary); }
.color-groups { width: 100%; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.groups-label { font-size: 12px; color: var(--color-text-secondary); }
.color-group-btn { width: 26px; height: 26px; border-radius: 7px; border: 1px solid rgba(0,0,0,0.08); color: #fff; font-size: 11px; font-weight: 700; text-shadow: 0 1px 2px rgba(0,0,0,0.3); }
.group-title { font-size: 12px; color: var(--color-text-secondary); border-left: 3px solid var(--color-border); padding-left: 8px; margin-bottom: 10px; }
.entry-list { display: flex; flex-direction: column; gap: 10px; }
.entry-item { display: grid; grid-template-columns: 44px minmax(0, 1fr) auto; gap: 10px; align-items: start; background: rgba(255,255,255,0.65); border-radius: var(--radius-md); padding: 10px; }
.entry-item.selected { outline: 1px solid var(--color-primary); }
.entry-avatar { width: 44px; height: 44px; padding: 0; border: 0; border-radius: 50%; overflow: hidden; background: var(--icon-bg); display: flex; align-items: center; justify-content: center; color: var(--icon-color); font-size: 12px; font-weight: 700; }
.entry-avatar img { width: 100%; height: 100%; object-fit: cover; }
.character-card-link { cursor: pointer; box-shadow: 0 0 0 2px var(--color-primary-light); }
.character-card-link:focus-visible, .character-card-name-link:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.entry-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.name-row { display: flex; align-items: center; gap: 6px; min-width: 0; }
.name-row strong { font-size: 13px; }
.character-card-name-link { min-width: 0; min-height: 32px; margin: -5px 0; padding: 5px 4px 5px 0; border: 0; background: transparent; color: inherit; display: inline-flex; align-items: center; gap: 4px; text-align: left; }
.character-card-name-link strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.character-card-name-link i { flex-shrink: 0; color: var(--color-primary); font-size: 14px; }
.channel-tag { font-size: 11px; color: var(--color-text-secondary); }
.entry-head time { font-size: 11px; color: var(--color-text-secondary); white-space: nowrap; }
.entry-text { margin-top: 6px; font-size: 14px; line-height: 1.65; color: var(--text-dark); white-space: pre-wrap; word-break: break-word; }
.entry-text :deep(.inline-emote) { width: 26px; height: 26px; vertical-align: text-bottom; margin: 0 2px; }
.entry-text :deep(.inline-mention) { display: inline-block; padding: 0 6px; border-radius: 999px; background: var(--color-primary-light); color: var(--color-secondary); font-size: 12px; }
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
.group-color { width: 24px; height: 24px; }
.group-hint { margin-top: 4px; font-size: 11px; color: var(--color-text-secondary); }
.report-sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 2400;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgba(15, 23, 42, 0.52);
}
.report-sheet {
  width: 100%;
  max-width: 640px;
  border-radius: 22px 22px 0 0;
  background: var(--color-card-bg);
  padding: 10px 16px calc(20px + var(--safe-bottom, 0px));
  box-shadow: 0 -18px 40px rgba(0, 0, 0, 0.18);
}
.report-sheet-handle {
  width: 54px;
  height: 5px;
  margin: 0 auto 14px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.45);
}
.report-sheet-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
}
.report-sheet-head h3 {
  font-size: 17px;
  margin: 0;
}
.report-sheet-head p {
  margin-top: 5px;
  font-size: 12px;
  color: var(--color-text-secondary);
  word-break: break-word;
}
.report-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 14px;
}
.report-field span {
  font-size: 13px;
  font-weight: 600;
}
.report-field select,
.report-field textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 12px;
  font: inherit;
}
.report-field textarea {
  resize: vertical;
}
.report-hint {
  margin: -4px 0 10px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}
.report-hint.error {
  color: #c2410c;
}
.report-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
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
.dialog-actions { margin-top: 12px; display: flex; justify-content: flex-end; gap: 8px; }
.action-btn { border: 1px solid var(--color-border); background: var(--color-panel-bg); color: var(--text-dark); border-radius: 8px; padding: 8px 10px; font-size: 13px; }
.action-btn.primary { border-color: var(--color-primary); background: var(--color-primary); color: var(--text-light); }
.action-btn.danger { border-color: var(--btn-danger-bg); background: var(--btn-danger-bg); color: #fff; }
</style>
