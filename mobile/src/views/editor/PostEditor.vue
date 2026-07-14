<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  POST_CATEGORIES,
  addPostTag,
  createPost,
  createPostDraft,
  deletePost,
  getPost,
  getPostTags,
  removePostTag,
  savePostDraft,
  type PostWithAuthor,
  type SavePostDraftRequest,
  type PostCategory,
  updatePost,
} from '@/api/post'
import { listGuilds, type Guild } from '@/api/guild'
import { resolveApiUrl } from '@/api/image'
import { uploadImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import {
  addPostToCollection,
  getPostCollection,
  removePostFromCollection,
} from '@/api/collection'
import MobileCollectionSelector from '@/components/MobileCollectionSelector.vue'
import MobileQuickJump from '@/components/MobileQuickJump.vue'
import MobileRichEditor from '@/components/MobileRichEditor.vue'
import MobilePostDraftBox from '@/components/MobilePostDraftBox.vue'
import NativeImageSourceDialog from '@/components/NativeImageSourceDialog.vue'
import {
  canUseNativeImagePicker,
  pickSingleNativeImageFile,
  type NativeImageSource,
} from '@/utils/nativeImagePicker'

interface PostEditorForm {
  title: string
  content: string
  content_type: 'markdown' | 'html'
  category: PostCategory
  region: string
  address: string
  guild_id?: number
  story_id?: number
  cover_image?: string
  is_public: boolean
  event_type?: 'server' | 'guild'
  event_start_time?: string
  event_end_time?: string
  event_color?: string
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const showDeleteDialog = ref(false)
const guilds = ref<Guild[]>([])
const coverUploading = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const coverPreview = ref('')
const editorRef = ref<InstanceType<typeof MobileRichEditor> | null>(null)
const quickJumpOpen = ref(false)
const tags = ref<Tag[]>([])
const selectedTags = ref<number[]>([])
const originalTags = ref<number[]>([])
const selectedCollectionId = ref<number | null>(null)
const originalCollectionId = ref<number | null>(null)
const useNativeImagePicker = canUseNativeImagePicker()
const showImageSourceDialog = ref(false)
const activeDraftId = ref<number | null>(null)
const loadedStatus = ref<'draft' | 'pending' | 'published' | null>(null)
const draftSaveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const draftRefreshKey = ref(0)
let draftTimer: ReturnType<typeof setTimeout> | null = null
let cloudSaveRunning = false
let cloudSaveQueued = false
let initializing = false
let skipUnmountSave = false

const form = ref<PostEditorForm>({
  title: '',
  content: '',
  content_type: 'markdown',
  category: 'other',
  region: '',
  address: '',
  guild_id: undefined,
  story_id: undefined,
  cover_image: '',
  is_public: true,
  event_type: undefined,
  event_start_time: undefined,
  event_end_time: undefined,
  event_color: '#D97706',
})

const postId = computed(() => Number(route.params.id))
const isEdit = computed(() => Number.isFinite(postId.value) && postId.value > 0)
const currentDraftId = computed(() => activeDraftId.value || (loadedStatus.value === 'draft' && isEdit.value ? postId.value : null))
const pageTitle = computed(() => isEdit.value ? t('community.editor.editTitle') : t('community.editor.createTitle'))
const isEventCategory = computed(() => form.value.category === 'event')

function resetEditor() {
  form.value = {
    title: '', content: '', content_type: 'html', category: 'other', region: '', address: '',
    guild_id: undefined, story_id: undefined, cover_image: '', is_public: true,
    event_type: undefined, event_start_time: undefined, event_end_time: undefined, event_color: '#D97706',
  }
  selectedTags.value = []
  originalTags.value = []
  selectedCollectionId.value = null
  originalCollectionId.value = null
  coverPreview.value = ''
  activeDraftId.value = null
  loadedStatus.value = null
  draftSaveState.value = 'idle'
}

watch(() => form.value.category, (category) => {
  if (category !== 'event') {
    form.value.event_type = undefined
    form.value.event_start_time = undefined
    form.value.event_end_time = undefined
  }
})

async function loadGuilds() {
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } catch (error) {
    console.error('Failed to load guilds', error)
  }
}

async function loadTags() {
  try {
    const res = await getPresetTags('post')
    tags.value = res.tags || []
  } catch (error) {
    console.error('Failed to load tags', error)
  }
}

async function loadPostForEdit() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res = await getPost(postId.value)
    form.value.title = res.post.title || ''
    form.value.content = res.post.content || ''
    form.value.content_type = (res.post.content_type as 'markdown' | 'html') || 'markdown'
    form.value.category = (res.post.category as PostCategory) || 'other'
    form.value.region = res.post.region || ''
    form.value.address = res.post.address || ''
    form.value.guild_id = res.post.guild_id
    form.value.story_id = res.post.story_id
    form.value.event_type = res.post.event_type
    form.value.event_start_time = res.post.event_start_time?.slice(0, 16)
    form.value.event_end_time = res.post.event_end_time?.slice(0, 16)
    form.value.event_color = res.post.event_color || '#D97706'
    form.value.cover_image = res.post.cover_image || ''
    form.value.is_public = res.post.is_public ?? true
    loadedStatus.value = res.post.status
    activeDraftId.value = res.post.status === 'draft' ? res.post.id : null
    draftSaveState.value = res.post.status === 'draft' ? 'saved' : 'idle'
    coverPreview.value = resolveApiUrl(res.post.cover_image || '')

    const collectionRes = await getPostCollection(postId.value)
    const collectionId = collectionRes.collection?.id ?? null
    originalCollectionId.value = collectionId
    selectedCollectionId.value = collectionId

    const tagRes = await getPostTags(postId.value)
    originalTags.value = tagRes.tags?.map((tag) => tag.id) || []
    selectedTags.value = [...originalTags.value]
  } catch (error) {
    console.error('Failed to load post detail', error)
    toast.error((error as Error)?.message || t('community.editor.loadFailed'))
    router.replace({ name: 'my-posts' })
  } finally {
    loading.value = false
  }
}

function hasDraftContent() {
  return Boolean(form.value.title.trim() || form.value.content.trim() || form.value.cover_image || form.value.region.trim() || form.value.address.trim() || selectedTags.value.length)
}

function buildDraftPayload(): SavePostDraftRequest {
  const payload: SavePostDraftRequest = {
    ...form.value,
    title: form.value.title.trim(),
    content: form.value.content.trim(),
    region: form.value.region.trim(),
    address: form.value.address.trim(),
    content_type: 'html',
    tag_ids: [...selectedTags.value],
  }
  if (payload.event_start_time) payload.event_start_time = new Date(payload.event_start_time).toISOString()
  if (payload.event_end_time) payload.event_end_time = new Date(payload.event_end_time).toISOString()
  return payload
}

async function saveDraftToCloud(force = false) {
  if (initializing || saving.value || (!force && !hasDraftContent())) return currentDraftId.value
  if (loadedStatus.value && loadedStatus.value !== 'draft') return null
  if (cloudSaveRunning) {
    cloudSaveQueued = true
    return currentDraftId.value
  }

  cloudSaveRunning = true
  draftSaveState.value = 'saving'
  try {
    const payload = buildDraftPayload()
    const saved = currentDraftId.value
      ? await savePostDraft(currentDraftId.value, payload)
      : await createPostDraft(payload)
    activeDraftId.value = saved.id
    loadedStatus.value = 'draft'
    await syncPostCollection(saved.id)
    originalTags.value = [...selectedTags.value]
    draftSaveState.value = 'saved'
    draftRefreshKey.value++
    return saved.id
  } catch (error) {
    console.error('Failed to autosave post draft', error)
    draftSaveState.value = 'error'
    return currentDraftId.value
  } finally {
    cloudSaveRunning = false
    if (cloudSaveQueued) {
      cloudSaveQueued = false
      void saveDraftToCloud(force)
    }
  }
}

function scheduleDraftSave() {
  if (initializing) return
  if (draftTimer) clearTimeout(draftTimer)
  draftTimer = setTimeout(() => void saveDraftToCloud(), 1400)
}

function toggleTag(tagId: number) {
  const index = selectedTags.value.indexOf(tagId)
  if (index >= 0) {
    selectedTags.value.splice(index, 1)
    return
  }
  selectedTags.value.push(tagId)
}

async function uploadCoverFile(file: File) {
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning(t('community.editor.invalidImage'))
    return
  }

  coverUploading.value = true
  try {
    const res = await uploadImage(file)
    const url = res.url || ''
    if (!url) throw new Error(t('community.editor.uploadFailed'))
    form.value.cover_image = url
    coverPreview.value = resolveApiUrl(url)
    toast.success(t('community.editor.uploadSuccess'))
  } catch (error) {
    console.error('Failed to upload cover image', error)
    toast.error((error as Error)?.message || t('community.editor.uploadFailed'))
  } finally {
    coverUploading.value = false
  }
}

async function onCoverFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  try {
    await uploadCoverFile(file as File)
  } finally {
    input.value = ''
  }
}

async function triggerCoverPicker() {
  if (!useNativeImagePicker) {
    coverInput.value?.click()
    return
  }

  showImageSourceDialog.value = true
}

function removeCoverImage() {
  form.value.cover_image = ''
  coverPreview.value = ''
}

async function handleImageSourceSelect(source: NativeImageSource) {
  try {
    const file = await pickSingleNativeImageFile(source)
    if (file) {
      await uploadCoverFile(file)
    }
  } catch (error) {
    console.error('Failed to pick cover image', error)
    toast.error((error as Error)?.message || t('community.editor.uploadFailed'))
  }
}

async function syncPostCollection(targetPostId: number) {
  if (selectedCollectionId.value === originalCollectionId.value) return

  if (originalCollectionId.value) {
    try {
      await removePostFromCollection(originalCollectionId.value, targetPostId)
    } catch (error) {
      console.warn('Failed to remove post from old collection', error)
    }
  }

  if (selectedCollectionId.value) {
    await addPostToCollection(selectedCollectionId.value, targetPostId)
  }

  originalCollectionId.value = selectedCollectionId.value
}

async function syncPostTags(targetPostId: number) {
  const addedTags = selectedTags.value.filter((tagId) => !originalTags.value.includes(tagId))
  const removedTags = originalTags.value.filter((tagId) => !selectedTags.value.includes(tagId))

  for (const tagId of addedTags) {
    await addPostTag(targetPostId, tagId)
  }
  for (const tagId of removedTags) {
    await removePostTag(targetPostId, tagId)
  }

  originalTags.value = [...selectedTags.value]
}

function validateForm() {
  if (!form.value.title.trim()) {
    toast.warning(t('community.editor.titleRequired'))
    return false
  }
  if (!form.value.content.trim()) {
    toast.warning(t('community.editor.contentRequired'))
    return false
  }
  if (form.value.category === 'event' && form.value.event_type === 'guild' && !form.value.guild_id) {
    toast.warning(t('community.editor.guildRequired'))
    return false
  }
  return true
}

function handleContentChange(value: string) {
  form.value.content = value
  form.value.content_type = 'html'
}

async function submit(status: 'draft' | 'published') {
  if (!validateForm()) return

  if (status === 'draft') {
    await saveDraftToCloud(true)
    if (draftSaveState.value === 'saved') toast.success(t('community.editor.draftSuccess'))
    return
  }

  if (loadedStatus.value === 'draft' || (!loadedStatus.value && hasDraftContent())) {
    await saveDraftToCloud(true)
  }

  saving.value = true
  try {
    const payload = {
      ...form.value,
      title: form.value.title.trim(),
      content: form.value.content.trim(),
      region: form.value.region.trim(),
      address: form.value.address.trim(),
      content_type: 'html' as const,
      tag_ids: selectedTags.value,
      status,
    }
    if (payload.event_start_time) {
      payload.event_start_time = new Date(payload.event_start_time).toISOString()
    }
    if (payload.event_end_time) {
      payload.event_end_time = new Date(payload.event_end_time).toISOString()
    }

    let targetId = currentDraftId.value || postId.value
    const updatingExisting = Boolean(currentDraftId.value || isEdit.value)
    if (updatingExisting) {
      await updatePost(targetId, payload)
    } else {
      const created = await createPost(payload)
      targetId = created.id
    }

    await syncPostCollection(targetId)
    if (updatingExisting) await syncPostTags(targetId)

    skipUnmountSave = true
    toast.success(status === 'published' ? t('community.editor.publishSuccess') : t('community.editor.draftSuccess'))
    router.replace({ name: 'post-detail', params: { id: targetId } })
  } catch (error) {
    console.error('Failed to submit post', error)
    toast.error((error as Error)?.message || t('community.editor.submitFailed'))
  } finally {
    saving.value = false
  }
}

function handleQuickInsert(html: string) {
  editorRef.value?.insertContent(html)
  quickJumpOpen.value = false
}

function openDeleteDialog() {
  deletingPost.value = null
  showDeleteDialog.value = true
}

const deletingPost = ref<PostWithAuthor | null>(null)

function openDraftDeleteDialog(post: PostWithAuthor) {
  deletingPost.value = post
  showDeleteDialog.value = true
}

async function confirmDelete() {
  const targetId = deletingPost.value?.id || currentDraftId.value || (isEdit.value ? postId.value : 0)
  if (!targetId || deleting.value) return
  deleting.value = true
  try {
    await deletePost(targetId)
    toast.success(t('community.editor.deleteSuccess'))
    draftRefreshKey.value++
    if (targetId === currentDraftId.value || targetId === postId.value) {
      skipUnmountSave = true
      resetEditor()
      await router.replace({ name: 'post-create' })
    }
  } catch (error) {
    console.error('Failed to delete post', error)
    toast.error((error as Error)?.message || t('community.editor.deleteFailed'))
  } finally {
    deleting.value = false
    showDeleteDialog.value = false
    deletingPost.value = null
  }
}

async function handleDraftSelect(id: number) {
  if (id === currentDraftId.value) return
  await saveDraftToCloud(true)
  await router.push({ name: 'post-edit', params: { id } })
}

async function handleNewDraft() {
  await saveDraftToCloud()
  resetEditor()
  if (route.name !== 'post-create') await router.push({ name: 'post-create' })
}

async function initializeEditor() {
  initializing = true
  skipUnmountSave = false
  resetEditor()
  await loadPostForEdit()
  initializing = false
}

onMounted(async () => {
  await Promise.all([loadGuilds(), loadTags(), initializeEditor()])
})

watch(() => route.params.id, async (id, oldId) => {
  if (id === oldId) return
  await initializeEditor()
})

watch([form, selectedTags, selectedCollectionId], scheduleDraftSave, { deep: true })

onBeforeUnmount(() => {
  if (draftTimer) clearTimeout(draftTimer)
  if (!skipUnmountSave && (loadedStatus.value === 'draft' || (!loadedStatus.value && hasDraftContent()))) void saveDraftToCloud()
})
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ pageTitle }}</h1>
      <MobilePostDraftBox
        :current-draft-id="currentDraftId"
        :save-state="draftSaveState"
        :refresh-key="draftRefreshKey"
        @select="handleDraftSelect"
        @create="handleNewDraft"
        @delete="openDraftDeleteDialog"
      />
    </header>

    <div class="sub-body editor-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <template v-else>
        <section class="editor-card">
          <label class="field">
            <span>{{ $t('community.editor.title') }}</span>
            <input v-model="form.title" type="text" :placeholder="$t('community.editor.titlePlaceholder')" maxlength="120">
          </label>

          <label class="field">
            <span>{{ $t('community.editor.category') }}</span>
            <select v-model="form.category">
              <option v-for="category in POST_CATEGORIES" :key="category.value" :value="category.value">
                {{ category.label }}
              </option>
            </select>
          </label>

          <label v-if="!isEventCategory" class="field">
            <span>{{ $t('community.editor.guild') }}</span>
            <select v-model="form.guild_id">
              <option :value="undefined">{{ $t('community.editor.guildNone') }}</option>
              <option v-for="guild in guilds" :key="guild.id" :value="guild.id">{{ guild.name }}</option>
            </select>
          </label>

          <label class="field">
            <span>{{ $t('community.editor.region') }}</span>
            <input v-model="form.region" type="text" :placeholder="$t('community.editor.regionPlaceholder')">
          </label>

          <label class="field">
            <span>{{ $t('community.editor.address') }}</span>
            <input v-model="form.address" type="text" :placeholder="$t('community.editor.addressPlaceholder')">
          </label>

          <label class="switch-field">
            <span>{{ $t('community.editor.publicVisible') }}</span>
            <input v-model="form.is_public" type="checkbox">
          </label>

          <div v-if="isEventCategory" class="field">
            <span>{{ $t('community.editor.eventType') }}</span>
            <div class="segmented-control">
              <button
                type="button"
                :class="{ active: form.event_type === 'server' }"
                @click="form.event_type = 'server'"
              >{{ $t('community.editor.eventTypeServer') }}</button>
              <button
                type="button"
                :class="{ active: form.event_type === 'guild' }"
                @click="form.event_type = 'guild'"
              >{{ $t('community.editor.eventTypeGuild') }}</button>
            </div>
          </div>

          <label v-if="isEventCategory && form.event_type === 'guild'" class="field">
            <span>{{ $t('community.editor.guild') }}</span>
            <select v-model="form.guild_id">
              <option :value="undefined">{{ $t('community.editor.guildSelect') }}</option>
              <option v-for="guild in guilds" :key="guild.id" :value="guild.id">{{ guild.name }}</option>
            </select>
          </label>

          <div v-if="isEventCategory && form.event_type" class="field">
            <span>{{ $t('community.editor.eventTime') }}</span>
            <div class="time-grid">
              <input v-model="form.event_start_time" type="datetime-local" :aria-label="$t('community.editor.eventStartTime')">
              <input v-model="form.event_end_time" type="datetime-local" :aria-label="$t('community.editor.eventEndTime')">
            </div>
          </div>

          <label v-if="isEventCategory && form.event_type" class="field">
            <span>{{ $t('community.editor.eventColor') }}</span>
            <div class="color-input-row">
              <input v-model="form.event_color" type="color">
              <span>{{ form.event_color }}</span>
            </div>
          </label>

          <div class="field">
            <span>{{ $t('community.editor.cover') }}</span>
            <div class="cover-box">
              <img v-if="coverPreview" :src="coverPreview" alt="">
              <button v-if="coverPreview" type="button" class="inline-btn" @click="removeCoverImage">
                {{ $t('community.editor.removeCover') }}
              </button>
              <button type="button" class="inline-btn" :disabled="coverUploading" @click="triggerCoverPicker">
                {{ coverUploading ? $t('common.status.loading') : $t('community.editor.uploadCover') }}
              </button>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="onCoverFileChange">
            </div>
          </div>

          <label class="field">
            <span>{{ $t('community.editor.content') }}</span>
            <MobileRichEditor
              ref="editorRef"
              :model-value="form.content"
              :placeholder="$t('community.editor.contentPlaceholder')"
              @update:modelValue="handleContentChange"
            >
              <template #toolbar>
                <button
                  type="button"
                  class="toolbar-slot toolbar-slot--featured"
                  :class="{ active: quickJumpOpen }"
                  :title="$t('community.editor.quickJump')"
                  :aria-label="$t('community.editor.quickJump')"
                  @mousedown.prevent
                  @click="quickJumpOpen = !quickJumpOpen"
                >
                  <i class="ri-links-line" />
                  <span>{{ $t('community.editor.quickJump') }}</span>
                </button>
              </template>
            </MobileRichEditor>
          </label>

          <div v-if="tags.length" class="field">
            <span>{{ $t('community.editor.tags') }}</span>
            <div class="tag-list">
              <button
                v-for="tag in tags"
                :key="tag.id"
                type="button"
                class="tag-chip"
                :class="{ selected: selectedTags.includes(tag.id) }"
                @click="toggleTag(tag.id)"
              >{{ tag.name }}</button>
            </div>
          </div>

          <MobileCollectionSelector
            v-model="selectedCollectionId"
            content-type="post"
          />
        </section>

        <section class="action-bar">
          <button v-if="loadedStatus !== 'published' && loadedStatus !== 'pending'" type="button" class="action-btn" :disabled="saving" @click="submit('draft')">
            {{ $t('community.editor.saveDraft') }}
          </button>
          <button type="button" class="action-btn primary" :disabled="saving" @click="submit('published')">
            {{ $t('community.editor.publish') }}
          </button>
          <button v-if="isEdit || currentDraftId" type="button" class="action-btn danger" :disabled="saving" @click="openDeleteDialog">
            {{ $t('community.editor.delete') }}
          </button>
        </section>
      </template>
    </div>

    <div v-if="showDeleteDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('community.editor.deleteTitle') }}</h3>
        <p>{{ $t('community.editor.deleteMessage') }}</p>
        <div class="dialog-actions">
          <button type="button" class="action-btn" @click="showDeleteDialog = false">{{ $t('community.editor.cancel') }}</button>
          <button type="button" class="action-btn danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? $t('common.status.loading') : $t('community.editor.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <NativeImageSourceDialog
      :model-value="showImageSourceDialog"
      @update:modelValue="showImageSourceDialog = $event"
      @select="handleImageSourceSelect"
    />
    <MobileQuickJump
      v-model="quickJumpOpen"
      :on-insert="handleQuickInsert"
      :exclude-post-id="isEdit ? postId : undefined"
    />
  </div>
</template>

<style scoped>
.editor-body {
  padding-bottom: calc(88px + var(--safe-bottom, 0px));
}

.sub-header h1 {
  flex: 1;
  min-width: 0;
}

.editor-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field > span {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.field input,
.field textarea,
.field select {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--input-bg);
  color: var(--text-dark);
}

.field textarea {
  resize: vertical;
}

.switch-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 13px;
  color: var(--text-dark);
}

.switch-field input {
  width: auto;
}

.segmented-control {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.segmented-control button {
  min-height: 38px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--color-text-secondary);
  font-size: 13px;
}

.segmented-control button.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--text-light);
}

.time-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}

.color-input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
}

.color-input-row input {
  width: 54px;
  height: 36px;
  padding: 0;
}

.color-input-row span {
  color: var(--color-text-secondary);
  font-size: 13px;
  font-family: monospace;
  text-transform: uppercase;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-chip {
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--input-border);
  border-radius: 999px;
  background: var(--input-bg);
  color: var(--text-dark);
  font-size: 12px;
}

.tag-chip.selected {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--text-light);
}

.cover-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cover-box img {
  width: 100%;
  border-radius: var(--radius-sm);
  max-height: 220px;
  object-fit: cover;
}

.inline-btn {
  width: fit-content;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 8px 12px;
  font-size: 13px;
}

.action-bar {
  position: fixed;
  left: calc(var(--safe-left, 0px) + 10px);
  right: calc(var(--safe-right, 0px) + 10px);
  bottom: calc(var(--safe-bottom, 0px) + 8px);
  z-index: 110;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 8px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(75, 54, 33, 0.12);
  box-shadow: 0 8px 20px rgba(44, 24, 16, 0.15);
}

.action-btn {
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  min-height: 40px;
  font-size: 13px;
  font-weight: 600;
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.action-btn.danger {
  grid-column: span 2;
  border-color: #c44747;
  color: #c44747;
}

.dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.48);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 16px;
}

.dialog {
  width: 100%;
  max-width: 360px;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 14px;
}

.dialog h3 {
  font-size: 16px;
}

.dialog p {
  margin-top: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
