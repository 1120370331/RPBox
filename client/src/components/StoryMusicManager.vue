<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import RButton from '@/components/RButton.vue'
import RModal from '@/components/RModal.vue'
import { useDialog } from '@/composables/useDialog'
import { useToast } from '@/composables/useToast'
import {
  addStoryMusicPlaylistTrack,
  attachStoryMusicTrack,
  createStoryMusicPlaylist,
  deleteStoryMusicPlaylist,
  deleteStoryMusicTrack,
  detachStoryMusicTrack,
  importPublicStoryMusicPlaylist,
  listPublicStoryMusicPlaylists,
  removeStoryMusicPlaylistTrack,
  shareStoryMusicPlaylist,
  updateStoryMusicPlaylist,
  updateStoryMusicTrack,
  uploadStoryMusicTrack,
  type StoryMusicPlaylist,
  type StoryMusicSegment,
  type StoryMusicTrack,
} from '@/api/story'

const props = defineProps<{
  modelValue: boolean
  storyId: number
  tracks: StoryMusicTrack[]
  segments: StoryMusicSegment[]
  playlists: StoryMusicPlaylist[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  refresh: []
  'enter-edit-mode': []
}>()

const toast = useToast()
const { confirm } = useDialog()

const searchKeyword = ref('')
const activeTab = ref<'story' | 'history' | 'playlists' | 'market'>('story')
const importing = ref(false)
const playlistName = ref('')
const playlistDescription = ref('')
const playlistColor = ref('#B87333')
const creatingPlaylist = ref(false)
const expandedPlaylistId = ref<number | null>(null)
const publicPlaylists = ref<StoryMusicPlaylist[]>([])
const loadingMarket = ref(false)
const previewingTrackId = ref<number | null>(null)
const previewLoadingTrackId = ref<number | null>(null)
const previewPopoverTrackId = ref<number | null>(null)
const previewCurrentTime = ref(0)
const previewDuration = ref(0)
const previewPaused = ref(false)
const previewError = ref('')
const activeTrackEditor = ref<{ trackId: number; field: TrackEditorField } | null>(null)
const activePlaylistEditor = ref<{ playlistId: number | null; field: PlaylistEditorField } | null>(null)
const trackEditorPosition = ref({ x: 0, y: 0 })
const playlistEditorPosition = ref({ x: 0, y: 0 })
const previewPopoverPosition = ref({ x: 0, y: 0 })
const inlineName = ref('')
const inlineColor = ref('#B87333')
const inlineVolume = ref(75)

type TrackEditorField = 'name' | 'color' | 'volume' | 'playlists'
type PlaylistEditorField = 'create' | 'name' | 'description' | 'color'

let previewAudio: HTMLAudioElement | null = null

const modalVisible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

function uniqueTracksById(tracks: StoryMusicTrack[]) {
  const seen = new Set<number>()
  return tracks.filter(track => {
    if (seen.has(track.id)) return false
    seen.add(track.id)
    return true
  })
}

const allTracks = computed(() => uniqueTracksById(props.tracks))

const storyTracks = computed(() => {
  return allTracks.value.filter(track => (track.storyIds || []).includes(props.storyId))
})

const historyTracks = computed(() => {
  return allTracks.value
})

const visibleTracks = computed(() => {
  const source = activeTab.value === 'story' ? storyTracks.value : historyTracks.value
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return source
  return source.filter(track => {
    const playlistNames = trackPlaylists(track.id)
      .map(playlist => `${playlist.name} ${playlist.description || ''}`)
      .join(' ')
      .toLowerCase()
    return track.name.toLowerCase().includes(keyword) ||
      (track.fileName || '').toLowerCase().includes(keyword) ||
      playlistNames.includes(keyword)
  })
})

const visiblePlaylists = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return props.playlists
  return props.playlists.filter(playlist =>
    playlist.name.toLowerCase().includes(keyword) ||
    (playlist.description || '').toLowerCase().includes(keyword)
  )
})

const previewSeekMax = computed(() => {
  return Math.max(previewDuration.value || previewCurrentTime.value || 0, 1)
})

const activeEditorTrack = computed(() => {
  const trackId = activeTrackEditor.value?.trackId
  if (!trackId) return null
  return allTracks.value.find(track => track.id === trackId) || null
})

const activeEditorField = computed(() => activeTrackEditor.value?.field || null)

const previewTrack = computed(() => {
  const trackId = previewPopoverTrackId.value || previewingTrackId.value || previewLoadingTrackId.value
  if (!trackId) return null
  return allTracks.value.find(track => track.id === trackId) || null
})

const activeEditorPlaylist = computed(() => {
  const playlistId = activePlaylistEditor.value?.playlistId
  if (!playlistId) return null
  return props.playlists.find(playlist => playlist.id === playlistId) || null
})

const activePlaylistField = computed(() => activePlaylistEditor.value?.field || null)

function isAudioFile(file: File) {
  return file.type.startsWith('audio/') || /\.(mp3|wav|ogg|m4a|aac|flac|webm)$/i.test(file.name)
}

function defaultNameFromFile(fileName: string) {
  return fileName.replace(/\.[^.]+$/, '').trim() || fileName
}

function normalizeVolume(value: number) {
  if (!Number.isFinite(value)) return 0.75
  return Math.min(Math.max(value / 100, 0), 1)
}

function volumePercent(track: StoryMusicTrack) {
  return Math.round((track.volume ?? 0.75) * 100)
}

function formatFileSize(size: number) {
  if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function formatPreviewTime(seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'
  const total = Math.floor(seconds)
  const mins = Math.floor(total / 60)
  const secs = total % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

function segmentCount(trackId: number) {
  return props.segments.filter(segment => segment.trackId === trackId).length
}

function playlistShareUrl(playlist: StoryMusicPlaylist) {
  if (!playlist.shareCode) return ''
  return `${window.location.origin}/music/playlist/${playlist.shareCode}`
}

function trackPlaylists(trackId: number) {
  return props.playlists.filter(playlist => (playlist.trackIds || []).includes(trackId))
}

function trackPlaylistLabel(track: StoryMusicTrack) {
  const names = trackPlaylists(track.id).map(playlist => playlist.name)
  if (!names.length) return '未加入歌单'
  if (names.length <= 2) return names.join('、')
  return `${names.slice(0, 2).join('、')} +${names.length - 2}`
}

function playlistTracks(playlist: StoryMusicPlaylist) {
  const ids = new Set(playlist.trackIds || [])
  return allTracks.value.filter(track => ids.has(track.id))
}

function playlistTrackCount(playlist: StoryMusicPlaylist) {
  return playlistTracks(playlist).length
}

function isPlaylistExpanded(playlist: StoryMusicPlaylist) {
  return expandedPlaylistId.value === playlist.id
}

function togglePlaylistExpanded(playlist: StoryMusicPlaylist) {
  closePlaylistField()
  closeTrackField()
  expandedPlaylistId.value = isPlaylistExpanded(playlist) ? null : playlist.id
}

function isTrackInStory(track: StoryMusicTrack) {
  return (track.storyIds || []).includes(props.storyId)
}

function isTrackFieldOpen(track: StoryMusicTrack, field: TrackEditorField) {
  const editor = activeTrackEditor.value
  return editor?.trackId === track.id && editor.field === field
}

function popoverSizeForField(field: TrackEditorField) {
  if (field === 'name') return { width: 320, height: 160 }
  if (field === 'color') return { width: 190, height: 150 }
  if (field === 'volume') return { width: 210, height: 180 }
  return { width: 360, height: 300 }
}

function popoverSizeForPlaylistField(field: PlaylistEditorField) {
  if (field === 'create') return { width: 360, height: 300 }
  if (field === 'name') return { width: 320, height: 160 }
  if (field === 'description') return { width: 340, height: 190 }
  return { width: 190, height: 150 }
}

function placeFloatingPopover(event: MouseEvent, width: number, height: number) {
  const margin = 12
  const x = Math.max(margin, Math.min(event.clientX + 10, window.innerWidth - width - margin))
  const y = Math.max(margin, Math.min(event.clientY + 10, window.innerHeight - height - margin))
  return { x, y }
}

function openTrackField(track: StoryMusicTrack, field: TrackEditorField, event?: MouseEvent) {
  event?.stopPropagation()
  closePlaylistField()
  if (isTrackFieldOpen(track, field)) {
    closeTrackField()
    return
  }

  if (event) {
    const size = popoverSizeForField(field)
    trackEditorPosition.value = placeFloatingPopover(event, size.width, size.height)
  }
  activeTrackEditor.value = { trackId: track.id, field }
  inlineName.value = track.name
  inlineColor.value = track.color || '#B87333'
  inlineVolume.value = volumePercent(track)
}

function closeTrackField() {
  activeTrackEditor.value = null
}

function isPlaylistFieldOpen(playlist: StoryMusicPlaylist, field: PlaylistEditorField) {
  const editor = activePlaylistEditor.value
  return editor?.playlistId === playlist.id && editor.field === field
}

function openCreatePlaylist(event?: MouseEvent) {
  event?.stopPropagation()
  closeTrackField()
  if (activePlaylistEditor.value?.field === 'create') {
    closePlaylistField()
    return
  }
  if (event) {
    const size = popoverSizeForPlaylistField('create')
    playlistEditorPosition.value = placeFloatingPopover(event, size.width, size.height)
  }
  playlistName.value = ''
  playlistDescription.value = ''
  playlistColor.value = '#B87333'
  activePlaylistEditor.value = { playlistId: null, field: 'create' }
}

function openPlaylistField(playlist: StoryMusicPlaylist, field: PlaylistEditorField, event?: MouseEvent) {
  event?.stopPropagation()
  closeTrackField()
  if (isPlaylistFieldOpen(playlist, field)) {
    closePlaylistField()
    return
  }
  if (event) {
    const size = popoverSizeForPlaylistField(field)
    playlistEditorPosition.value = placeFloatingPopover(event, size.width, size.height)
  }
  playlistName.value = playlist.name
  playlistDescription.value = playlist.description || ''
  playlistColor.value = playlist.color || '#B87333'
  activePlaylistEditor.value = { playlistId: playlist.id, field }
}

function closePlaylistField() {
  activePlaylistEditor.value = null
}

function stopPreview() {
  const audio = previewAudio
  previewAudio = null
  if (audio) {
    audio.onloadedmetadata = null
    audio.ondurationchange = null
    audio.ontimeupdate = null
    audio.onplay = null
    audio.onpause = null
    audio.onended = null
    audio.onerror = null
    audio.pause()
    audio.removeAttribute('src')
    audio.load()
  }
  previewingTrackId.value = null
  previewLoadingTrackId.value = null
  previewPopoverTrackId.value = null
  previewCurrentTime.value = 0
  previewDuration.value = 0
  previewPaused.value = false
  previewError.value = ''
}

async function togglePreviewPlayback() {
  if (!previewAudio) return
  try {
    if (previewAudio.paused) {
      await previewAudio.play()
    } else {
      previewAudio.pause()
    }
  } catch (e) {
    console.error('切换试听播放失败:', e)
    toast.error('试听播放失败')
  }
}

function seekPreview() {
  if (!previewAudio) return
  const duration = previewDuration.value || previewAudio.duration || 0
  const next = Math.min(Math.max(previewCurrentTime.value, 0), duration || previewCurrentTime.value)
  previewAudio.currentTime = next
  previewCurrentTime.value = next
}

async function togglePreview(track: StoryMusicTrack, event?: MouseEvent) {
  event?.stopPropagation()
  if (previewingTrackId.value === track.id) {
    stopPreview()
    return
  }
  if (event) {
    previewPopoverPosition.value = placeFloatingPopover(event, 360, 170)
  }
  if (!track.url) {
    previewPopoverTrackId.value = track.id
    previewError.value = '音频地址不可用'
    toast.error('音频地址不可用')
    return
  }

  stopPreview()
  previewPopoverTrackId.value = track.id
  previewLoadingTrackId.value = track.id
  previewCurrentTime.value = 0
  previewDuration.value = 0
  previewPaused.value = false
  previewError.value = ''

  const audio = new Audio(track.url)
  previewAudio = audio
  audio.preload = 'auto'
  audio.volume = track.volume ?? 0.75
  previewingTrackId.value = track.id
  audio.onloadedmetadata = () => {
    if (previewAudio !== audio) return
    previewDuration.value = Number.isFinite(audio.duration) ? audio.duration : 0
  }
  audio.ondurationchange = audio.onloadedmetadata
  audio.ontimeupdate = () => {
    if (previewAudio !== audio) return
    previewCurrentTime.value = audio.currentTime
    if (Number.isFinite(audio.duration)) previewDuration.value = audio.duration
  }
  audio.onplay = () => {
    if (previewAudio === audio) previewPaused.value = false
  }
  audio.onpause = () => {
    if (previewAudio === audio && !audio.ended) previewPaused.value = true
  }
  audio.onended = () => {
    if (previewAudio === audio) stopPreview()
  }
  audio.onerror = () => {
    if (previewAudio !== audio) return
    const failedTrackId = track.id
    stopPreview()
    previewPopoverTrackId.value = failedTrackId
    previewError.value = '试听播放失败'
    toast.error('试听播放失败')
  }

  try {
    await audio.play()
    if (previewAudio === audio) {
      previewLoadingTrackId.value = null
    }
  } catch (e) {
    const failedTrackId = track.id
    if (previewAudio === audio) stopPreview()
    previewPopoverTrackId.value = failedTrackId
    previewError.value = '试听播放失败'
    console.error('试听播放失败:', e)
    toast.error('试听播放失败')
  }
}

async function saveTrackField(track: StoryMusicTrack, field: TrackEditorField) {
  const payload: { name?: string; color?: string; volume?: number } = {}

  if (field === 'name') {
    if (!inlineName.value.trim()) {
      toast.error('音乐名称不能为空')
      return
    }
    payload.name = inlineName.value.trim()
  } else if (field === 'color') {
    payload.color = inlineColor.value || '#B87333'
  } else if (field === 'volume') {
    payload.volume = normalizeVolume(inlineVolume.value)
  } else {
    return
  }

  try {
    await updateStoryMusicTrack(props.storyId, track.id, payload)
    if (field === 'volume' && previewAudio && previewingTrackId.value === track.id) {
      previewAudio.volume = payload.volume ?? previewAudio.volume
    }
    closeTrackField()
    emit('refresh')
    toast.success('音乐信息已保存')
  } catch (e) {
    console.error('保存音乐信息失败:', e)
    toast.error('保存音乐信息失败')
  }
}

async function saveTrackVolume(track: StoryMusicTrack, value: number, close = false) {
  inlineVolume.value = value
  try {
    const volume = normalizeVolume(value)
    await updateStoryMusicTrack(props.storyId, track.id, { volume })
    if (previewAudio && previewingTrackId.value === track.id) {
      previewAudio.volume = volume
    }
    if (close) closeTrackField()
    emit('refresh')
    toast.success('音量已更新')
  } catch (e) {
    console.error('更新音量失败:', e)
    toast.error('更新音量失败')
  }
}

async function handleImportAudio(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || []).filter(isAudioFile)
  if (!files.length) {
    input.value = ''
    toast.error('请选择音频文件')
    return
  }

  importing.value = true
  try {
    for (const file of files) {
      await uploadStoryMusicTrack(props.storyId, file, {
        name: defaultNameFromFile(file.name),
        color: '#B87333',
        volume: 0.75,
      })
    }
    activeTab.value = 'story'
    emit('refresh')
    toast.success(files.length === 1 ? '音乐已导入' : `已导入 ${files.length} 首音乐`)
  } catch (e) {
    console.error('导入音乐失败:', e)
    toast.error('导入音乐失败')
  } finally {
    importing.value = false
    input.value = ''
  }
}

async function attachToStory(track: StoryMusicTrack) {
  try {
    await attachStoryMusicTrack(props.storyId, track.id)
    activeTab.value = 'story'
    emit('refresh')
    toast.success('已加入本剧情音乐')
  } catch (e) {
    console.error('加入本剧情失败:', e)
    toast.error('加入本剧情失败')
  }
}

async function removeFromStory(track: StoryMusicTrack) {
  if (previewingTrackId.value === track.id) stopPreview()
  const ok = await confirm({
    title: '移除音乐',
    message: segmentCount(track.id) > 0
      ? '此音乐已有定位段落，移除后会同时清除这些定位。'
      : '确定要从本剧情移除这首音乐吗？',
    type: 'warning',
  })
  if (!ok) return

  try {
    await detachStoryMusicTrack(props.storyId, track.id)
    emit('refresh')
    toast.success('已移除音乐')
  } catch (e) {
    console.error('移除音乐失败:', e)
    toast.error('移除音乐失败')
  }
}

async function deleteTrack(track: StoryMusicTrack) {
  if (previewingTrackId.value === track.id) stopPreview()
  const ok = await confirm({
    title: '删除音乐',
    message: '确定要删除这首历史音乐吗？所有剧情中的音乐定位也会一并清除。',
    type: 'error',
  })
  if (!ok) return

  try {
    await deleteStoryMusicTrack(props.storyId, track.id)
    emit('refresh')
    toast.success('音乐已删除')
  } catch (e) {
    console.error('删除音乐失败:', e)
    toast.error('删除音乐失败')
  }
}

async function createPlaylist() {
  if (!playlistName.value.trim()) {
    toast.error('歌单名称不能为空')
    return
  }

  creatingPlaylist.value = true
  try {
    await createStoryMusicPlaylist({
      name: playlistName.value.trim(),
      description: playlistDescription.value.trim(),
      color: playlistColor.value,
      trackIds: [],
    })
    playlistName.value = ''
    playlistDescription.value = ''
    playlistColor.value = '#B87333'
    closePlaylistField()
    emit('refresh')
    toast.success('歌单已创建')
  } catch (e) {
    console.error('创建歌单失败:', e)
    toast.error('创建歌单失败')
  } finally {
    creatingPlaylist.value = false
  }
}

async function savePlaylistField(playlist: StoryMusicPlaylist, field: Exclude<PlaylistEditorField, 'create'>) {
  try {
    if (field === 'name') {
      if (!playlistName.value.trim()) {
        toast.error('歌单名称不能为空')
        return
      }
      await updateStoryMusicPlaylist(playlist.id, { name: playlistName.value.trim() })
    } else if (field === 'description') {
      await updateStoryMusicPlaylist(playlist.id, { description: playlistDescription.value.trim() })
    } else if (field === 'color') {
      await updateStoryMusicPlaylist(playlist.id, { color: playlistColor.value })
    }
    emit('refresh')
    closePlaylistField()
    toast.success('歌单已更新')
  } catch (e) {
    console.error('更新歌单失败:', e)
    toast.error('更新歌单失败')
  }
}

async function togglePlaylistTrack(playlist: StoryMusicPlaylist, track: StoryMusicTrack) {
  try {
    if ((playlist.trackIds || []).includes(track.id)) {
      await removeStoryMusicPlaylistTrack(playlist.id, track.id)
      toast.success('已移出歌单')
    } else {
      await addStoryMusicPlaylistTrack(playlist.id, track.id)
      toast.success('已加入歌单')
    }
    emit('refresh')
  } catch (e) {
    console.error('更新歌单曲目失败:', e)
    toast.error('更新歌单曲目失败')
  }
}

async function removeTrackFromPlaylist(playlist: StoryMusicPlaylist, track: StoryMusicTrack) {
  try {
    await removeStoryMusicPlaylistTrack(playlist.id, track.id)
    if (previewingTrackId.value === track.id) stopPreview()
    closeTrackField()
    emit('refresh')
    toast.success('已从歌单移除')
  } catch (e) {
    console.error('移出歌单失败:', e)
    toast.error('移出歌单失败')
  }
}

async function togglePlaylistPublic(playlist: StoryMusicPlaylist) {
  try {
    await updateStoryMusicPlaylist(playlist.id, { isPublic: !playlist.isPublic })
    emit('refresh')
    toast.success(!playlist.isPublic ? '歌单已公开到素材市场' : '歌单已取消公开')
  } catch (e) {
    console.error('更新歌单公开状态失败:', e)
    toast.error('更新歌单公开状态失败')
  }
}

async function copyPlaylistShareLink(playlist: StoryMusicPlaylist) {
  try {
    const shared = playlist.shareCode ? playlist : await shareStoryMusicPlaylist(playlist.id)
    emit('refresh')
    await navigator.clipboard.writeText(playlistShareUrl(shared))
    toast.success('歌单分享链接已复制')
  } catch (e) {
    console.error('复制歌单分享链接失败:', e)
    toast.error('复制歌单分享链接失败')
  }
}

async function handleDeletePlaylist(playlist: StoryMusicPlaylist) {
  const ok = await confirm({
    title: '删除歌单',
    message: `确定要删除歌单「${playlist.name}」吗？曲目文件不会被删除。`,
    type: 'warning',
  })
  if (!ok) return

  try {
    await deleteStoryMusicPlaylist(playlist.id)
    if (expandedPlaylistId.value === playlist.id) expandedPlaylistId.value = null
    closePlaylistField()
    emit('refresh')
    toast.success('歌单已删除')
  } catch (e) {
    console.error('删除歌单失败:', e)
    toast.error('删除歌单失败')
  }
}

async function loadMaterialMarket() {
  loadingMarket.value = true
  try {
    const res = await listPublicStoryMusicPlaylists(searchKeyword.value)
    publicPlaylists.value = res.playlists || []
  } catch (e) {
    console.error('加载素材市场失败:', e)
    toast.error('加载素材市场失败')
  } finally {
    loadingMarket.value = false
  }
}

async function importPlaylistFromMarket(playlist: StoryMusicPlaylist) {
  if (!playlist.shareCode) return
  try {
    const res = await importPublicStoryMusicPlaylist(props.storyId, playlist.shareCode)
    activeTab.value = 'story'
    emit('refresh')
    toast.success(`已导入 ${res.tracks?.length || 0} 首音乐`)
  } catch (e) {
    console.error('导入公开歌单失败:', e)
    toast.error('导入公开歌单失败')
  }
}

function enterEditMode() {
  emit('enter-edit-mode')
  emit('update:modelValue', false)
}

function handleGlobalPointerDown(event: PointerEvent) {
  const target = event.target as HTMLElement | null
  if (!target) return
  if (
    target.closest('.music-floating-popover') ||
    target.closest('.music-popover-trigger')
  ) {
    return
  }
  closeTrackField()
  closePlaylistField()
  if (previewPopoverTrackId.value || previewingTrackId.value || previewLoadingTrackId.value) {
    stopPreview()
  }
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  closeTrackField()
  closePlaylistField()
  if (previewPopoverTrackId.value || previewingTrackId.value || previewLoadingTrackId.value) {
    stopPreview()
  }
}

function handleViewportChange() {
  closeTrackField()
  closePlaylistField()
  if (previewPopoverTrackId.value || previewingTrackId.value || previewLoadingTrackId.value) {
    stopPreview()
  }
}

watch(() => props.modelValue, visible => {
  if (!visible) {
    stopPreview()
    closeTrackField()
    closePlaylistField()
  }
})

onMounted(() => {
  document.addEventListener('pointerdown', handleGlobalPointerDown, true)
  document.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('resize', handleViewportChange)
  window.addEventListener('scroll', handleViewportChange, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleGlobalPointerDown, true)
  document.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('resize', handleViewportChange)
  window.removeEventListener('scroll', handleViewportChange, true)
  stopPreview()
})
</script>

<template>
  <RModal v-model="modalVisible" title="背景音乐管理" width="860px">
    <div class="music-manager">
      <div class="music-toolbar">
        <label class="music-import" :class="{ disabled: importing }">
          <input
            type="file"
            accept="audio/*,.mp3,.wav,.ogg,.m4a,.aac,.flac,.webm"
            multiple
            :disabled="importing"
            @change="handleImportAudio"
          />
          <i :class="importing ? 'ri-loader-4-line spinning' : 'ri-upload-cloud-2-line'"></i>
          <span>{{ importing ? '导入中...' : '导入音频' }}</span>
        </label>

        <div class="music-search">
          <i class="ri-search-line"></i>
          <input
            v-model="searchKeyword"
            :placeholder="activeTab === 'market' ? '检索公开素材歌单' : '检索名称、文件名或歌单'"
            @keyup.enter="activeTab === 'market' && loadMaterialMarket()"
          />
        </div>

        <RButton type="primary" :disabled="storyTracks.length === 0" @click="enterEditMode">
          <i class="ri-timeline-view"></i>
          进入编辑模式
        </RButton>
      </div>

      <div class="music-tabs">
        <button
          class="music-tab"
          :class="{ active: activeTab === 'story' }"
          @click="activeTab = 'story'"
        >
          本剧情音乐
          <span>{{ storyTracks.length }}</span>
        </button>
        <button
          class="music-tab"
          :class="{ active: activeTab === 'history' }"
          @click="activeTab = 'history'"
        >
          用户历史音乐
          <span>{{ historyTracks.length }}</span>
        </button>
        <button
          class="music-tab"
          :class="{ active: activeTab === 'playlists' }"
          @click="activeTab = 'playlists'"
        >
          我的歌单
          <span>{{ playlists.length }}</span>
        </button>
        <button
          class="music-tab"
          :class="{ active: activeTab === 'market' }"
          @click="activeTab = 'market'; loadMaterialMarket()"
        >
          素材市场
          <span>{{ publicPlaylists.length }}</span>
        </button>
      </div>

      <div v-if="activeTab === 'playlists'" class="playlist-panel">
        <div class="playlist-toolbar">
          <RButton
            class="music-popover-trigger"
            size="small"
            type="primary"
            @click="openCreatePlaylist($event)"
          >
            <i class="ri-add-line"></i>
            新建歌单
          </RButton>
          <span>{{ visiblePlaylists.length }} 个歌单</span>
        </div>

        <div v-if="visiblePlaylists.length === 0" class="music-empty">
          <i class="ri-album-line"></i>
          <p>还没有歌单</p>
        </div>
        <div v-else class="playlist-list">
          <div v-for="playlist in visiblePlaylists" :key="playlist.id" class="music-row playlist-manage-row">
            <div class="music-command-row playlist-command-row">
              <button type="button" class="field-chip color-chip music-popover-trigger" @click="openPlaylistField(playlist, 'color', $event)">
                <span class="track-color-dot" :style="{ backgroundColor: playlist.color }"></span>
                代表色
              </button>
              <button type="button" class="field-chip name-chip playlist-name-chip music-popover-trigger" @click="openPlaylistField(playlist, 'name', $event)">
                <i class="ri-pencil-line"></i>
                <span>{{ playlist.name }}</span>
              </button>
              <button type="button" class="field-chip playlist-desc-chip music-popover-trigger" @click="openPlaylistField(playlist, 'description', $event)">
                <i class="ri-file-text-line"></i>
                <span>{{ playlist.description || '无描述' }}</span>
              </button>
              <button type="button" class="field-chip playlist-tracks-chip" @click="togglePlaylistExpanded(playlist)">
                <i class="ri-music-2-line"></i>
                <span>{{ isPlaylistExpanded(playlist) ? '收起' : '查看' }} {{ playlistTrackCount(playlist) }} 首</span>
              </button>
              <RButton size="small" type="ghost" @click="copyPlaylistShareLink(playlist)">
                <i class="ri-share-line"></i>
                分享
              </RButton>
              <RButton size="small" type="secondary" @click="togglePlaylistPublic(playlist)">
                {{ playlist.isPublic ? '取消公开' : '公开' }}
              </RButton>
              <RButton size="small" type="danger" @click="handleDeletePlaylist(playlist)">
                <i class="ri-delete-bin-line"></i>
              </RButton>
            </div>
            <div class="music-meta">
              <span>{{ playlist.isPublic ? '已公开到素材市场' : '私有歌单' }}</span>
              <span v-if="playlist.shareCode">分享码 {{ playlist.shareCode }}</span>
              <span>{{ playlist.updatedAt ? `更新 ${new Date(playlist.updatedAt).toLocaleDateString()}` : '' }}</span>
            </div>

            <Transition name="playlist-expand">
              <div v-if="isPlaylistExpanded(playlist)" class="playlist-expanded">
                <div v-if="playlistTracks(playlist).length === 0" class="playlist-empty-inline">
                  这个歌单还没有音乐。到“本剧情音乐”或“用户历史音乐”列表里，把音乐加入歌单。
                </div>
                <div v-else class="music-list playlist-track-list">
                  <div v-for="track in playlistTracks(playlist)" :key="track.id" class="music-row playlist-track-row">
                    <div class="music-command-row">
                      <div class="field-shell preview-field">
                        <RButton
                          class="music-popover-trigger"
                          size="small"
                          :type="previewingTrackId === track.id ? 'secondary' : 'ghost'"
                          :disabled="previewLoadingTrackId === track.id"
                          @click="togglePreview(track, $event)"
                        >
                          <i :class="previewingTrackId === track.id ? 'ri-pause-fill' : (previewLoadingTrackId === track.id ? 'ri-loader-4-line spinning' : 'ri-play-fill')"></i>
                          {{ previewingTrackId === track.id ? '停止' : '试听' }}
                        </RButton>
                      </div>

                      <div class="field-shell color-field">
                        <button type="button" class="field-chip color-chip music-popover-trigger" @click="openTrackField(track, 'color', $event)">
                          <span class="track-color-dot" :style="{ backgroundColor: track.color }"></span>
                          代表色
                        </button>
                      </div>

                      <div class="field-shell name-field">
                        <button type="button" class="field-chip name-chip music-popover-trigger" @click="openTrackField(track, 'name', $event)">
                          <i class="ri-pencil-line"></i>
                          <span>{{ track.name }}</span>
                        </button>
                      </div>

                      <div class="field-shell volume-field">
                        <button type="button" class="field-chip volume-chip music-popover-trigger" @click="openTrackField(track, 'volume', $event)">
                          <i class="ri-volume-up-line"></i>
                          {{ volumePercent(track) }}%
                        </button>
                      </div>

                      <div class="field-shell playlists-field">
                        <button type="button" class="field-chip playlists-chip music-popover-trigger" @click="openTrackField(track, 'playlists', $event)">
                          <i class="ri-album-line"></i>
                          <span>歌单：{{ trackPlaylistLabel(track) }}</span>
                        </button>
                      </div>

                      <RButton
                        size="small"
                        type="danger"
                        title="从当前歌单移除"
                        @click="removeTrackFromPlaylist(playlist, track)"
                      >
                        <i class="ri-delete-bin-line"></i>
                      </RButton>
                    </div>

                    <div class="music-meta">
                      <span>{{ track.fileName }}</span>
                      <span>{{ formatFileSize(track.size) }}</span>
                      <span v-if="isTrackInStory(track)">已在本剧情</span>
                      <span v-if="segmentCount(track.id) > 0">{{ segmentCount(track.id) }} 个定位</span>
                    </div>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'market'" class="playlist-panel">
        <div class="market-toolbar">
          <span>公开素材歌单可直接导入本剧情，复用同一音频文件。</span>
          <RButton size="small" type="secondary" :loading="loadingMarket" @click="loadMaterialMarket">
            刷新
          </RButton>
        </div>
        <div v-if="publicPlaylists.length === 0" class="music-empty">
          <i class="ri-store-2-line"></i>
          <p>{{ loadingMarket ? '正在加载素材市场' : '暂无公开素材歌单' }}</p>
        </div>
        <div v-else class="playlist-list">
          <div v-for="playlist in publicPlaylists" :key="playlist.id" class="playlist-row">
            <div class="music-color" :style="{ backgroundColor: playlist.color }"></div>
            <div class="playlist-info">
              <div class="music-title-line">
                <strong>{{ playlist.name }}</strong>
                <span>{{ playlist.trackCount }} 首</span>
              </div>
              <div class="music-meta">
                <span>{{ playlist.authorName || '匿名作者' }}</span>
                <span><i class="ri-eye-line"></i> {{ playlist.viewCount }}</span>
              </div>
              <p v-if="playlist.description" class="playlist-desc">{{ playlist.description }}</p>
            </div>
            <div class="music-actions">
              <RButton size="small" type="primary" @click="importPlaylistFromMarket(playlist)">
                导入本剧情
              </RButton>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="visibleTracks.length === 0" class="music-empty">
        <i class="ri-music-2-line"></i>
        <p>{{ activeTab === 'story' ? '本剧情还没有音乐' : '还没有历史音乐' }}</p>
      </div>

      <div v-else class="music-list">
        <div v-for="track in visibleTracks" :key="track.id" class="music-row">
          <div class="music-command-row">
            <div class="field-shell preview-field">
              <RButton
                class="music-popover-trigger"
                size="small"
                :type="previewingTrackId === track.id ? 'secondary' : 'ghost'"
                :disabled="previewLoadingTrackId === track.id"
                @click="togglePreview(track, $event)"
              >
                <i :class="previewingTrackId === track.id ? 'ri-pause-fill' : (previewLoadingTrackId === track.id ? 'ri-loader-4-line spinning' : 'ri-play-fill')"></i>
                {{ previewingTrackId === track.id ? '停止' : '试听' }}
              </RButton>
            </div>

            <div class="field-shell color-field">
              <button type="button" class="field-chip color-chip music-popover-trigger" @click="openTrackField(track, 'color', $event)">
                <span class="track-color-dot" :style="{ backgroundColor: track.color }"></span>
                代表色
              </button>
            </div>

            <div class="field-shell name-field">
              <button type="button" class="field-chip name-chip music-popover-trigger" @click="openTrackField(track, 'name', $event)">
                <i class="ri-pencil-line"></i>
                <span>{{ track.name }}</span>
              </button>
            </div>

            <div class="field-shell volume-field">
              <button type="button" class="field-chip volume-chip music-popover-trigger" @click="openTrackField(track, 'volume', $event)">
                <i class="ri-volume-up-line"></i>
                {{ volumePercent(track) }}%
              </button>
            </div>

            <div class="field-shell playlists-field">
              <button type="button" class="field-chip playlists-chip music-popover-trigger" @click="openTrackField(track, 'playlists', $event)">
                <i class="ri-album-line"></i>
                <span>歌单：{{ trackPlaylistLabel(track) }}</span>
              </button>
            </div>

            <RButton
              v-if="activeTab === 'history' && !isTrackInStory(track)"
              size="small"
              type="secondary"
              @click="attachToStory(track)"
            >
              加入本剧情
            </RButton>
            <RButton
              v-else-if="activeTab === 'history'"
              size="small"
              type="ghost"
              disabled
            >
              已在本剧情
            </RButton>
            <RButton
              size="small"
              type="danger"
              @click="activeTab === 'story' ? removeFromStory(track) : deleteTrack(track)"
            >
              <i class="ri-delete-bin-line"></i>
            </RButton>
          </div>

          <div class="music-meta">
            <span>{{ track.fileName }}</span>
            <span>{{ formatFileSize(track.size) }}</span>
            <span v-if="segmentCount(track.id) > 0">{{ segmentCount(track.id) }} 个定位</span>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="music-floating-popover">
        <div
          v-if="previewTrack"
          class="music-floating-popover preview-popover"
          :style="{ left: `${previewPopoverPosition.x}px`, top: `${previewPopoverPosition.y}px` }"
          @click.stop
        >
          <div class="preview-popover-head">
            <button
              type="button"
              class="preview-icon-button"
              :disabled="previewLoadingTrackId === previewTrack.id || !!previewError"
              @click="togglePreviewPlayback"
            >
              <i :class="previewPaused ? 'ri-play-fill' : 'ri-pause-fill'"></i>
            </button>
            <div class="preview-popover-main">
              <strong>{{ previewTrack.name }}</strong>
              <span>
                {{ previewError || (previewLoadingTrackId === previewTrack.id ? '加载中...' : `${formatPreviewTime(previewCurrentTime)} / ${formatPreviewTime(previewDuration)}`) }}
              </span>
            </div>
            <button type="button" class="preview-icon-button" @click="stopPreview">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <input
            v-model.number="previewCurrentTime"
            class="preview-seek"
            type="range"
            min="0"
            :max="previewSeekMax"
            step="0.1"
            :disabled="previewLoadingTrackId === previewTrack.id || !!previewError"
            @input="seekPreview"
          />
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="music-floating-popover">
        <div
          v-if="activeEditorTrack && activeEditorField"
          class="music-floating-popover field-popover"
          :class="`${activeEditorField}-popover`"
          :style="{ left: `${trackEditorPosition.x}px`, top: `${trackEditorPosition.y}px` }"
          @click.stop
        >
          <template v-if="activeEditorField === 'color'">
            <label>
              代表色
              <input v-model="inlineColor" type="color" />
            </label>
            <div class="field-popover-actions">
              <button type="button" @click="closeTrackField">取消</button>
              <button type="button" class="primary" @click="saveTrackField(activeEditorTrack, 'color')">保存</button>
            </div>
          </template>

          <template v-else-if="activeEditorField === 'name'">
            <label>
              音乐名称
              <input
                v-model="inlineName"
                type="text"
                @keyup.enter="saveTrackField(activeEditorTrack, 'name')"
                @keyup.esc="closeTrackField"
              />
            </label>
            <div class="field-popover-actions">
              <button type="button" @click="closeTrackField">取消</button>
              <button type="button" class="primary" @click="saveTrackField(activeEditorTrack, 'name')">保存</button>
            </div>
          </template>

          <template v-else-if="activeEditorField === 'volume'">
            <div class="volume-popover-title">音量 {{ inlineVolume }}%</div>
            <input
              v-model.number="inlineVolume"
              type="range"
              min="0"
              max="100"
              @change="saveTrackVolume(activeEditorTrack, inlineVolume)"
            />
            <div class="volume-steps">
              <button type="button" @click="saveTrackVolume(activeEditorTrack, 25, true)">25%</button>
              <button type="button" @click="saveTrackVolume(activeEditorTrack, 50, true)">50%</button>
              <button type="button" @click="saveTrackVolume(activeEditorTrack, 75, true)">75%</button>
              <button type="button" @click="saveTrackVolume(activeEditorTrack, 100, true)">100%</button>
            </div>
          </template>

          <template v-else-if="activeEditorField === 'playlists'">
            <div class="playlist-picker-title">加入歌单</div>
            <div v-if="playlists.length === 0" class="playlist-picker-empty">
              还没有歌单，先到“我的歌单”创建。
            </div>
            <div v-else class="playlist-picker-list">
              <button
                v-for="playlist in playlists"
                :key="playlist.id"
                type="button"
                :class="{ active: (playlist.trackIds || []).includes(activeEditorTrack.id) }"
                @click="togglePlaylistTrack(playlist, activeEditorTrack)"
              >
                <i :class="(playlist.trackIds || []).includes(activeEditorTrack.id) ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'"></i>
                <span>{{ playlist.name }}</span>
                <small>{{ playlist.trackCount }} 首</small>
              </button>
            </div>
          </template>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="music-floating-popover">
        <div
          v-if="activePlaylistField"
          class="music-floating-popover field-popover playlist-editor-popover"
          :class="`playlist-${activePlaylistField}-popover`"
          :style="{ left: `${playlistEditorPosition.x}px`, top: `${playlistEditorPosition.y}px` }"
          @click.stop
        >
          <template v-if="activePlaylistField === 'create'">
            <label>
              歌单名称
              <input
                v-model="playlistName"
                type="text"
                @keyup.enter="createPlaylist"
                @keyup.esc="closePlaylistField"
              />
            </label>
            <label>
              描述
              <textarea v-model="playlistDescription" rows="3" placeholder="可选"></textarea>
            </label>
            <label>
              代表色
              <input v-model="playlistColor" type="color" />
            </label>
            <div class="field-popover-actions split-actions">
              <button type="button" @click="closePlaylistField">取消</button>
              <button type="button" class="primary" :disabled="creatingPlaylist" @click="createPlaylist">创建歌单</button>
            </div>
          </template>

          <template v-else-if="activeEditorPlaylist && activePlaylistField === 'color'">
            <label>
              代表色
              <input v-model="playlistColor" type="color" />
            </label>
            <div class="field-popover-actions">
              <button type="button" @click="closePlaylistField">取消</button>
              <button type="button" class="primary" @click="savePlaylistField(activeEditorPlaylist, 'color')">保存</button>
            </div>
          </template>

          <template v-else-if="activeEditorPlaylist && activePlaylistField === 'name'">
            <label>
              歌单名称
              <input
                v-model="playlistName"
                type="text"
                @keyup.enter="savePlaylistField(activeEditorPlaylist, 'name')"
                @keyup.esc="closePlaylistField"
              />
            </label>
            <div class="field-popover-actions">
              <button type="button" @click="closePlaylistField">取消</button>
              <button type="button" class="primary" @click="savePlaylistField(activeEditorPlaylist, 'name')">保存</button>
            </div>
          </template>

          <template v-else-if="activeEditorPlaylist && activePlaylistField === 'description'">
            <label>
              描述
              <textarea v-model="playlistDescription" rows="4" placeholder="可选"></textarea>
            </label>
            <div class="field-popover-actions">
              <button type="button" @click="closePlaylistField">取消</button>
              <button type="button" class="primary" @click="savePlaylistField(activeEditorPlaylist, 'description')">保存</button>
            </div>
          </template>

        </div>
      </Transition>
    </Teleport>

    <template #footer>
      <RButton @click="modalVisible = false">关闭</RButton>
    </template>
  </RModal>
</template>

<style scoped>
.music-manager {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.music-toolbar {
  display: grid;
  grid-template-columns: auto minmax(220px, 1fr) auto;
  gap: 12px;
  align-items: center;
}

.music-import {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px dashed var(--color-border);
  border-radius: 8px;
  color: var(--color-primary);
  background: var(--color-card-bg, #f5f0eb);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.music-import:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.music-import.disabled {
  opacity: 0.6;
  cursor: wait;
}

.music-import input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.music-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  height: 42px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--input-bg, #fff);
  color: var(--color-secondary);
}

.music-search input {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
}

.music-tabs {
  display: inline-flex;
  width: fit-content;
  padding: 4px;
  border-radius: 8px;
  background: var(--color-card-bg, #f5f0eb);
}

.music-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-secondary);
  padding: 8px 12px;
  cursor: pointer;
  font: inherit;
  font-size: 13px;
}

.music-tab.active {
  color: var(--color-primary);
  background: var(--color-panel-bg, #fff);
  box-shadow: 0 1px 4px rgba(var(--shadow-base), 0.12);
}

.music-tab span {
  min-width: 20px;
  height: 20px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(184, 115, 51, 0.12);
  color: var(--color-accent);
  font-size: 12px;
}

.music-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 52vh;
  overflow-y: auto;
  padding-right: 4px;
}

.music-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--color-border-light, rgba(229, 212, 193, 0.7));
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
}

.music-command-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  padding-bottom: 2px;
  scrollbar-width: thin;
}

.music-command-row :deep(.r-button) {
  flex: 0 0 auto;
  min-height: 30px;
  white-space: nowrap;
}

.preview-field,
.color-field,
.volume-field {
  flex: 0 0 auto;
}

.name-field {
  flex: 1 1 180px;
  max-width: 260px;
}

.playlists-field {
  flex: 1 1 170px;
  max-width: 240px;
}

.music-color {
  border-radius: 4px;
}

.music-title-line,
.music-meta,
.music-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.music-title-line {
  justify-content: space-between;
  margin-bottom: 4px;
  gap: 12px;
}

.music-title-line strong {
  color: var(--color-primary);
  font-size: 14px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-title-line span,
.music-meta {
  color: var(--color-secondary);
  font-size: 12px;
}

.music-meta {
  min-width: 0;
  flex-wrap: wrap;
  padding-left: 4px;
  line-height: 1.5;
}

.field-shell {
  position: relative;
  min-width: 0;
}

.field-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  max-width: 100%;
  min-height: 30px;
  padding: 5px 10px;
  border: 1px solid var(--color-border-light, rgba(229, 212, 193, 0.7));
  border-radius: 999px;
  background: rgba(184, 115, 51, 0.08);
  color: var(--color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  line-height: 1.2;
  text-align: left;
  transition: border-color 0.2s, color 0.2s, background 0.2s;
}

.field-chip span:not(.track-color-dot) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.field-chip:hover {
  border-color: var(--color-accent);
  color: var(--color-primary);
  background: rgba(184, 115, 51, 0.12);
}

.track-color-dot {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border: 1px solid rgba(64, 59, 51, 0.18);
  border-radius: 50%;
}

.color-chip {
  width: auto;
}

.name-chip,
.playlists-chip {
  justify-content: flex-start;
}

.volume-chip {
  width: auto;
  white-space: nowrap;
}

.music-floating-popover {
  position: fixed;
  z-index: 1600;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
  box-shadow: 0 12px 30px rgba(var(--shadow-base), 0.18);
}

.field-popover {
  width: min(280px, calc(100vw - 24px));
}

.name-popover {
  width: min(320px, calc(100vw - 24px));
}

.color-popover {
  width: 180px;
}

.volume-popover {
  width: 180px;
}

.playlists-popover {
  width: min(360px, calc(100vw - 24px));
}

.playlist-editor-popover {
  width: min(340px, calc(100vw - 24px));
}

.playlist-create-popover {
  width: min(360px, calc(100vw - 24px));
}

.playlist-color-popover {
  width: 180px;
}

.field-popover label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--color-secondary);
  font-size: 12px;
}

.field-popover input,
.field-popover textarea {
  width: 100%;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--color-border);
  border-radius: 7px;
  background: var(--input-bg, #fff);
  color: var(--color-primary);
  font: inherit;
}

.field-popover textarea {
  resize: vertical;
  line-height: 1.5;
}

.field-popover input[type="color"] {
  height: 36px;
  padding: 2px;
}

.field-popover input[type="range"] {
  padding: 0;
}

.field-popover-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.field-popover-actions button,
.volume-steps button {
  border: 1px solid var(--color-border);
  border-radius: 7px;
  padding: 6px 10px;
  background: transparent;
  color: var(--color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.field-popover-actions.split-actions {
  justify-content: space-between;
  flex-wrap: wrap;
}

.field-popover-actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.field-popover-actions button.primary,
.volume-steps button:hover {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: white;
}

.field-popover-actions button.primary:disabled {
  border-color: var(--color-border);
  background: transparent;
  color: var(--color-secondary);
}

.volume-popover-title {
  margin-bottom: 8px;
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 600;
}

.volume-steps {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
  margin-top: 10px;
}

.playlist-picker-title {
  margin-bottom: 8px;
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 600;
}

.playlist-picker-empty {
  color: var(--color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.playlist-picker-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
}

.playlist-picker-list button {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 8px;
  border: 1px solid var(--color-border-light, rgba(229, 212, 193, 0.7));
  border-radius: 7px;
  background: transparent;
  color: var(--color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  text-align: left;
}

.playlist-picker-list button span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.playlist-picker-list button small {
  color: var(--color-tertiary, #8b8072);
  font-size: 11px;
}

.playlist-picker-list button.active {
  border-color: var(--color-accent);
  background: rgba(184, 115, 51, 0.12);
  color: var(--color-primary);
}

.preview-popover {
  width: min(340px, calc(100vw - 24px));
}

.preview-popover-head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.preview-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--color-border-light, rgba(229, 212, 193, 0.7));
  border-radius: 7px;
  background: rgba(184, 115, 51, 0.1);
  color: var(--color-primary);
  cursor: pointer;
  font-size: 18px;
}

.preview-icon-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.preview-icon-button:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.preview-popover-main {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.preview-popover-main strong {
  min-width: 0;
  overflow: hidden;
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-popover-main span {
  color: var(--color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.preview-seek {
  width: 100%;
  margin-top: 12px;
  accent-color: var(--color-accent);
}

.music-floating-popover-enter-active,
.music-floating-popover-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.music-floating-popover-enter-from,
.music-floating-popover-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}

.music-actions {
  align-self: center;
}

.music-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 44px 16px;
  color: var(--color-secondary);
  border: 1px dashed var(--color-border);
  border-radius: 8px;
  background: var(--color-card-bg, #f5f0eb);
}

.music-empty i {
  font-size: 28px;
  color: var(--color-accent);
}

.music-empty p {
  margin: 0;
  font-size: 14px;
}

.playlist-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.playlist-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-secondary);
  font-size: 13px;
}

.playlist-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 420px;
  overflow: auto;
  padding-right: 4px;
}

.playlist-row {
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
}

.playlist-manage-row {
  padding: 10px 12px;
}

.playlist-command-row {
  align-items: stretch;
}

.playlist-command-row > .field-chip {
  width: auto;
}

.playlist-name-chip {
  flex: 0 1 auto;
  max-width: 180px;
}

.playlist-desc-chip {
  flex: 0 1 auto;
  max-width: 220px;
}

.playlist-tracks-chip {
  flex: 0 0 auto;
  white-space: nowrap;
}

.playlist-expanded {
  overflow: hidden;
  padding-top: 10px;
  border-top: 1px solid var(--color-border-light, rgba(229, 212, 193, 0.7));
}

.playlist-track-list {
  gap: 8px;
  max-height: min(340px, 42vh);
  padding-right: 2px;
}

.playlist-track-row {
  background: rgba(184, 115, 51, 0.04);
}

.playlist-empty-inline {
  padding: 12px;
  border: 1px dashed var(--color-border-light, rgba(229, 212, 193, 0.7));
  border-radius: 8px;
  background: rgba(184, 115, 51, 0.04);
  color: var(--color-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.playlist-expand-enter-active,
.playlist-expand-leave-active {
  max-height: 520px;
  overflow: hidden;
  transform-origin: top;
  transition: max-height 0.2s ease, opacity 0.2s ease, transform 0.2s ease;
}

.playlist-expand-enter-from,
.playlist-expand-leave-to {
  max-height: 0;
  opacity: 0;
  transform: scaleY(0.98);
}

.playlist-info {
  min-width: 0;
}

.playlist-desc {
  margin: 6px 0 0;
  color: var(--color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.market-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-secondary);
  font-size: 13px;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 760px) {
  .music-toolbar {
    grid-template-columns: 1fr;
  }

  .playlist-row {
    grid-template-columns: 1fr;
  }

  .music-command-row {
    flex-wrap: nowrap;
  }

  .music-title-line {
    align-items: flex-start;
    flex-direction: column;
  }

  .field-popover,
  .name-popover,
  .playlists-popover,
  .playlist-editor-popover,
  .preview-popover {
    width: min(320px, 86vw);
  }
}
</style>
