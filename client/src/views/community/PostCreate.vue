<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  createPost,
  createPostDraft,
  deletePost,
  getPost,
  savePostDraft,
  updatePost,
  type CreatePostRequest,
  type PostWithAuthor,
  type SavePostDraftRequest,
  POST_CATEGORIES,
  type PostCategory,
} from '@/api/post'
import { uploadImage } from '@/api/item'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import ImageCropperDialog from '@/components/ImageCropperDialog.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import PostQuickJump from '@/components/PostQuickJump.vue'
import PostDraftBox from '@/components/PostDraftBox.vue'
import CollectionSelector from '@/components/CollectionSelector.vue'
import { addPostToCollection, removePostFromCollection } from '@/api/collection'
import { useDialog } from '@/composables/useDialog'

const DRAFT_KEY = 'post_create_draft'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const toast = useToastStore()
const dialog = useDialog()
const userStore = useUserStore()
const mounted = ref(false)
const loading = ref(false)

function resolveInitialCategory(): PostCategory {
  const raw = String(route.query.category || '').trim()
  if (POST_CATEGORIES.some(cat => cat.value === raw)) {
    return raw as PostCategory
  }
  return 'other'
}

const initialCategory = resolveInitialCategory()

const form = ref<CreatePostRequest>({
  title: '',
  content: '',
  content_type: 'html',
  category: initialCategory,
  region: '',
  address: '',
  tag_ids: [],
  status: 'published',
  cover_image: '',
  is_public: true,  // 公会外成员可见（默认开启）
  event_type: initialCategory === 'event' ? 'server' : undefined,
  event_start_time: undefined,
  event_end_time: undefined,
  event_color: '#D97706',
})

// 封面图相关
const coverImagePreview = ref('')
const coverImageLoading = ref(false)
const coverImageInput = ref<HTMLInputElement | null>(null)
const coverCropperOpen = ref(false)
const coverCropperFile = ref<File | null>(null)
const editorRef = ref<InstanceType<typeof TiptapEditor> | null>(null)
const quickJumpOpen = ref(false)

// 是否为活动分区
const isEventCategory = computed(() => form.value.category === 'event')

// 监听分区变化，重置活动相关字段
watch(() => form.value.category, (newVal) => {
  if (newVal !== 'event') {
    form.value.event_type = undefined
    form.value.event_start_time = undefined
    form.value.event_end_time = undefined
  } else if (!form.value.event_type) {
    form.value.event_type = 'server'
  }
})

const tags = ref<Tag[]>([])
const guilds = ref<Guild[]>([])
const selectedTags = ref<number[]>([])
const selectedCollectionId = ref<number | null>(null)
const draftId = ref<number | null>(null)
const draftSaveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const draftRefreshKey = ref(0)
const syncedCollectionId = ref<number | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let cloudSaveRunning = false
let cloudSaveQueued = false
let skipUnmountSave = false

function hasDraftContent(formData = form.value, tagIds = selectedTags.value) {
  return Boolean(
    formData.title?.trim()
    || formData.content?.trim()
    || formData.cover_image
    || formData.region?.trim()
    || formData.address?.trim()
    || tagIds.length > 0
  )
}

function saveLocalDraft() {
  // 只缓存“独立新帖草稿”，且必须绑定合法 draft 状态云端 ID
  if (!hasDraftContent() || !draftId.value) return
  const draft = {
    form: form.value,
    selectedTags: selectedTags.value,
    draftId: draftId.value,
    savedAt: Date.now(),
  }
  localStorage.setItem(DRAFT_KEY, JSON.stringify(draft))
}

function debouncedSaveDraft() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => void saveDraftToCloud(), 1400)
}

// 清除草稿
function clearDraft() {
  localStorage.removeItem(DRAFT_KEY)
}

/** 仅当云端帖仍是 draft 时才允许绑定，防止挂到已发布帖 */
async function ensureValidDraftId(candidateId: number | null | undefined): Promise<number | null> {
  const id = Number(candidateId)
  if (!id) return null
  try {
    const res = await getPost(id)
    if (res.post?.status === 'draft' && Number(res.post.author_id) === Number(userStore.user?.id)) {
      return id
    }
  } catch (error) {
    console.warn('草稿 ID 校验失败，将重新创建草稿:', error)
  }
  return null
}

async function maybeRestoreDraft() {
  const saved = localStorage.getItem(DRAFT_KEY)
  if (!saved) return

  try {
    const draft = JSON.parse(saved)
    const draftForm = draft.form || {}
    const draftTags = Array.isArray(draft.selectedTags) ? draft.selectedTags : []
    if (!hasDraftContent(draftForm, draftTags)) {
      clearDraft()
      return
    }

    const confirmed = await dialog.confirm({
      title: t('community.drafts.resumeTitle'),
      message: t('community.drafts.resumeMessage'),
      type: 'warning',
      confirmText: t('community.drafts.resumeContinue'),
      cancelText: t('community.drafts.resumeDiscard'),
    })

    if (!confirmed) {
      clearDraft()
      return
    }

    const validId = await ensureValidDraftId(Number(draft.draftId) || null)
    if (!validId) {
      // 旧绑定已失效：只恢复本地内容，发布前会新建云端草稿
      form.value = { ...form.value, ...draftForm, status: 'published' }
      selectedTags.value = draftTags
      draftId.value = null
      coverImagePreview.value = form.value.cover_image || ''
      clearDraft()
      return
    }

    // 合法草稿：进入草稿编辑页，避免在 create 页误操作正式帖
    clearDraft()
    await router.replace({ name: 'post-edit', params: { id: validId } })
  } catch (e) {
    console.error('恢复草稿失败:', e)
    clearDraft()
  }
}

onMounted(async () => {
  // 检查登录状态
  if (!userStore.user || !userStore.token) {
    toast.warning(t('community.create.loginRequired'))
    router.push('/login')
    return
  }

  setTimeout(() => mounted.value = true, 50)
  await maybeRestoreDraft()
  // 若已跳转到草稿编辑页，后续初始化由 PostEdit 接管
  if (route.name !== 'post-create') return

  // 入口 query.category 始终生效（例如从 Banner 空态点「发布活动」）
  const preferredCategory = resolveInitialCategory()
  if (preferredCategory !== 'other') {
    form.value.category = preferredCategory
    if (preferredCategory === 'event' && !form.value.event_type) {
      form.value.event_type = 'server'
    }
  }

  await loadTags()
  await loadGuilds()
})

onUnmounted(() => {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
  if (!skipUnmountSave) {
    saveLocalDraft()
    if (hasDraftContent()) void saveDraftToCloud()
  }
})

// 监听表单变化，自动保存草稿
watch([form, selectedTags, selectedCollectionId], debouncedSaveDraft, { deep: true })

function buildDraftPayload(): SavePostDraftRequest {
  const payload: SavePostDraftRequest = {
    ...form.value,
    title: form.value.title.trim(),
    content: form.value.content.trim(),
    region: form.value.region?.trim() || '',
    address: form.value.address?.trim() || '',
    content_type: 'html',
    tag_ids: [...selectedTags.value],
  }
  // 明确携带封面字段，避免 undefined 跳过更新；空字符串表示清空当前草稿封面
  payload.cover_image = form.value.cover_image || ''
  if (payload.event_start_time) payload.event_start_time = new Date(payload.event_start_time).toISOString()
  if (payload.event_end_time) payload.event_end_time = new Date(payload.event_end_time).toISOString()
  return payload
}

async function syncDraftCollection(postId: number) {
  if (syncedCollectionId.value === selectedCollectionId.value) return
  if (syncedCollectionId.value) await removePostFromCollection(syncedCollectionId.value, postId)
  if (selectedCollectionId.value) await addPostToCollection(selectedCollectionId.value, postId)
  syncedCollectionId.value = selectedCollectionId.value
}

async function saveDraftToCloud(force = false) {
  if ((!force && !hasDraftContent()) || (loading.value && !force)) return draftId.value
  if (cloudSaveRunning) {
    cloudSaveQueued = true
    return draftId.value
  }

  cloudSaveRunning = true
  draftSaveState.value = 'saving'
  try {
    const payload = buildDraftPayload()
    let targetId = await ensureValidDraftId(draftId.value)
    if (draftId.value && !targetId) {
      // 绑定失效：断开旧 ID，改为新建草稿行
      draftId.value = null
    }

    const saved = targetId
      ? await savePostDraft(targetId, payload)
      : await createPostDraft(payload)

    // 仅接受云端返回的草稿 ID
    draftId.value = saved.id
    await syncDraftCollection(saved.id)
    saveLocalDraft()
    draftSaveState.value = 'saved'
    draftRefreshKey.value++
    return saved.id
  } catch (error) {
    console.error('云端草稿保存失败:', error)
    draftSaveState.value = 'error'
    // 草稿保存失败时不要继续拿着可能非法的 ID 去发布
    if (String((error as any)?.message || '').includes('只有草稿') || String((error as any)?.message || '').includes('Conflict')) {
      draftId.value = null
      clearDraft()
    }
    return draftId.value
  } finally {
    cloudSaveRunning = false
    if (cloudSaveQueued) {
      cloudSaveQueued = false
      void saveDraftToCloud(force)
    }
  }
}

function resetForNewDraft() {
  const preferredCategory = resolveInitialCategory()
  form.value = {
    title: '',
    content: '',
    content_type: 'html',
    category: preferredCategory,
    region: '',
    address: '',
    tag_ids: [],
    status: 'published',
    cover_image: '',
    is_public: true,
    event_type: preferredCategory === 'event' ? 'server' : undefined,
    event_start_time: undefined,
    event_end_time: undefined,
    event_color: '#D97706',
  }
  selectedTags.value = []
  selectedCollectionId.value = null
  syncedCollectionId.value = null
  coverImagePreview.value = ''
  draftId.value = null
  draftSaveState.value = 'idle'
  clearDraft()
}

async function handleDraftSelect(id: number) {
  if (id === draftId.value) return
  // 当前内容先尽量落盘到云端草稿箱，再进入选中草稿
  if (hasDraftContent()) {
    await saveDraftToCloud(true)
  }
  const validId = await ensureValidDraftId(id)
  if (!validId) {
    toast.error(t('community.drafts.invalidDraft'))
    draftRefreshKey.value++
    return
  }
  await router.push({ name: 'post-edit', params: { id: validId } })
}

async function handleNewDraft() {
  // 当前内容先入草稿箱，再创建全新空白草稿并进入编辑
  if (hasDraftContent()) {
    await saveDraftToCloud(true)
  }
  try {
    const created = await createPostDraft({
      title: '',
      content: '',
      content_type: 'html',
      category: resolveInitialCategory() === 'other' ? 'other' : resolveInitialCategory(),
      tag_ids: [],
    })
    clearDraft()
    draftRefreshKey.value++
    await router.push({ name: 'post-edit', params: { id: created.id } })
  } catch (error) {
    console.error('创建草稿失败:', error)
    toast.error(t('community.drafts.error'))
    resetForNewDraft()
  }
}

async function handleDraftDelete(post: PostWithAuthor) {
  const confirmed = await dialog.confirm({
    title: t('community.drafts.deleteTitle'),
    message: t('community.drafts.deleteMessage', { title: post.title || t('community.drafts.untitled') }),
    type: 'warning',
  })
  if (!confirmed) return
  await deletePost(post.id)
  if (post.id === draftId.value) resetForNewDraft()
  draftRefreshKey.value++
}

async function loadTags() {
  try {
    // 只加载帖子类型的标签
    const res = await listTags('post')
    tags.value = res.tags || []
  } catch (error) {
    console.error('加载标签失败:', error)
  }
}

async function loadGuilds() {
  try {
    const res = await listGuilds()
    guilds.value = res.guilds || []
  } catch (error) {
    console.error('加载公会失败:', error)
  }
}

function toggleTag(tagId: number) {
  const index = selectedTags.value.indexOf(tagId)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tagId)
  }
}

async function handleSubmit(status: 'draft' | 'published') {
  if (!form.value.title.trim()) {
    toast.warning(t('community.create.titleRequired'))
    return
  }
  if (!form.value.content.trim()) {
    toast.warning(t('community.create.contentRequired'))
    return
  }

  // 活动分区基础验证
  if (form.value.category === 'event') {
    if (!form.value.event_type) {
      toast.warning(t('community.create.eventTypeRequired'))
      return
    }
    if (!form.value.event_start_time) {
      toast.warning(t('community.create.eventStartRequired'))
      return
    }
    if (form.value.event_type === 'guild' && !form.value.guild_id) {
      toast.warning(t('community.create.selectGuildForEvent'))
      return
    }
  }

  loading.value = true
  try {
    form.value.status = status
    form.value.tag_ids = selectedTags.value

    const payload: CreatePostRequest = { ...form.value }
    payload.region = form.value.region?.trim() || ''
    payload.address = form.value.address?.trim() || ''

    // 转换时间格式为 ISO8601/RFC3339
    if (payload.event_start_time) {
      payload.event_start_time = new Date(payload.event_start_time).toISOString()
    }
    if (payload.event_end_time) {
      payload.event_end_time = new Date(payload.event_end_time).toISOString()
    }

    if (status === 'draft') {
      const savedId = await saveDraftToCloud(true)
      if (savedId && draftSaveState.value === 'saved') {
        toast.success(t('community.create.draftSuccess'))
      } else {
        toast.error(t('community.drafts.error'))
      }
      return
    }

    // 发布：只允许把“云端 draft 行”转正；绝不能 update 已发布帖
    let publishedPostId: number | null = null
    const validDraftId = await ensureValidDraftId(draftId.value)
    if (validDraftId) {
      await savePostDraft(validDraftId, buildDraftPayload())
      const published = await updatePost(validDraftId, payload)
      publishedPostId = (published as any)?.id || validDraftId
    } else {
      draftId.value = null
      const created = await createPost(payload)
      publishedPostId = (created as any)?.id || (created as any)?.data?.id || null
    }

    if (selectedCollectionId.value && publishedPostId) {
      await addPostToCollection(selectedCollectionId.value, publishedPostId)
    }

    skipUnmountSave = true
    clearDraft()
    draftId.value = null
    toast.success(t('community.create.publishSuccess'))
    router.push({ name: 'my-posts' })
  } catch (error: any) {
    console.error('提交失败:', error)
    const msg = error?.message || t('community.create.submitFailed')
    toast.error(msg)
  } finally {
    loading.value = false
  }
}

function handleCancel() {
  void saveDraftToCloud()
  router.back()
}

async function handlePreview() {
  await saveDraftToCloud()

  // 保存预览数据到 sessionStorage
  const previewData = {
    title: form.value.title,
    content: form.value.content,
    category: form.value.category,
    tag_ids: selectedTags.value,
    guild_id: form.value.guild_id,
    region: form.value.region,
    address: form.value.address,
    event_type: form.value.event_type,
    event_start_time: form.value.event_start_time,
    event_end_time: form.value.event_end_time,
    // 用于显示的额外信息
    selectedTagNames: tags.value.filter(t => selectedTags.value.includes(t.id)).map(t => t.name),
  }
  sessionStorage.setItem('post_preview', JSON.stringify(previewData))
  router.push({ name: 'post-preview' })
}

// 处理封面图上传
function handleCoverImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.error(t('community.create.selectImageFile'))
    input.value = ''
    return
  }

  coverCropperFile.value = file
  coverCropperOpen.value = true
  input.value = ''
}

async function handleCoverImageCropped(file: File) {
  coverImageLoading.value = true
  try {
    const res: any = await uploadImage(file)
    const url = res?.data?.url || res?.url
    if (!url) {
      throw new Error(t('community.create.noImageUrl'))
    }
    coverImagePreview.value = url
    form.value.cover_image = url
    toast.success(t('community.create.coverUploadSuccess'))
  } catch (error: any) {
    console.error('封面图上传失败:', error)
    toast.error(error?.message || t('community.create.coverUploadFailed'))
  } finally {
    coverImageLoading.value = false
    coverCropperFile.value = null
  }
}

function handleCoverCropperError(error: Error) {
  toast.error(error.message || t('community.create.coverUploadFailed'))
}

// 移除封面图
function removeCoverImage() {
  coverImagePreview.value = ''
  form.value.cover_image = ''
}

function handleQuickInsert(html: string) {
  editorRef.value?.insertContent(html)
  quickJumpOpen.value = false
}

function toggleQuickJump() {
  quickJumpOpen.value = !quickJumpOpen.value
}
</script>

<template>
  <div class="post-create-page" :class="{ 'animate-in': mounted }">
    <!-- 头部 -->
    <div class="page-header anim-item" style="--delay: 0">
      <h1 class="page-title">{{ t('community.create.pageTitle') }}</h1>
      <PostDraftBox
        :current-draft-id="draftId"
        :save-state="draftSaveState"
        :refresh-key="draftRefreshKey"
        @select="handleDraftSelect"
        @create="handleNewDraft"
        @delete="handleDraftDelete"
      />
    </div>

    <!-- 编辑区域 -->
    <div class="editor-container anim-item" style="--delay: 1">
      <!-- 标题输入 -->
      <div class="title-group">
        <input
          v-model="form.title"
          type="text"
          :placeholder="t('community.create.titlePlaceholder')"
          class="title-input"
        />
      </div>

      <!-- 封面图上传 -->
      <div class="cover-image-group">
        <label class="cover-label">{{ t('community.create.coverLabel') }}</label>
        <div class="cover-upload-area">
          <div v-if="coverImagePreview" class="cover-preview">
            <img :src="coverImagePreview" :alt="t('community.create.coverPreview')" />
            <div class="cover-preview-actions">
              <button class="edit-cover-btn" @click="coverImageInput?.click()">
                <i class="ri-crop-line"></i>
              </button>
              <button class="remove-cover-btn" @click="removeCoverImage">
                <i class="ri-close-line"></i>
              </button>
            </div>
          </div>
          <div v-else class="cover-placeholder" @click="coverImageInput?.click()">
            <i class="ri-image-add-line"></i>
            <span>{{ coverImageLoading ? t('community.create.coverProcessing') : t('community.create.coverUpload') }}</span>
            <span class="cover-hint">{{ t('community.create.coverHint') }}</span>
          </div>
          <input
            ref="coverImageInput"
            type="file"
            accept="image/*"
            style="display: none"
            @change="handleCoverImageUpload"
          />
        </div>
      </div>

      <ImageCropperDialog
        v-model="coverCropperOpen"
        :file="coverCropperFile"
        :aspect-ratio="16 / 9"
        :output-width="1600"
        :output-height="900"
        :max-size-k-b="1024"
        title="调整封面图"
        @cropped="handleCoverImageCropped"
        @error="handleCoverCropperError"
      />

      <!-- 内容编辑器 -->
      <div class="content-group">
        <TiptapEditor
          ref="editorRef"
          v-model="form.content"
          :placeholder="t('community.create.contentPlaceholder')"
        >
          <template #toolbar>
            <button
              type="button"
              class="toolbar-slot toolbar-slot--featured"
              :class="{ active: quickJumpOpen }"
              :title="t('community.create.quickJump')"
              :aria-label="t('community.create.quickJump')"
              @mousedown.prevent
              @click="toggleQuickJump"
            >
              <i class="ri-links-line"></i>
              <span>{{ t('community.create.quickJump') }}</span>
            </button>
          </template>
        </TiptapEditor>
      </div>

      <PostQuickJump v-model="quickJumpOpen" :on-insert="handleQuickInsert" />
    </div>

    <!-- 设置区域 -->
    <div class="settings-bar anim-item" style="--delay: 2">
      <!-- 分区与位置 -->
      <div class="setting-block setting-block-primary">
        <div class="setting-item setting-vertical setting-item--category">
          <label class="setting-label">{{ t('community.create.category') }}</label>
          <div class="category-select">
            <select v-model="form.category">
              <option v-for="cat in POST_CATEGORIES" :key="cat.value" :value="cat.value">
                {{ cat.label }}
              </option>
            </select>
          </div>
        </div>
        <div class="location-fields">
          <div class="setting-item setting-vertical location-setting">
            <label class="setting-label">{{ t('community.create.region') }}</label>
            <input
              v-model="form.region"
              type="text"
              class="location-text-input"
              :placeholder="t('community.create.regionPlaceholder')"
              autocomplete="off"
            />
          </div>

          <div class="setting-item setting-vertical location-setting">
            <label class="setting-label">{{ t('community.create.address') }}</label>
            <input
              v-model="form.address"
              type="text"
              class="location-text-input"
              :placeholder="t('community.create.addressPlaceholder')"
              autocomplete="off"
            />
          </div>
        </div>
      </div>

      <!-- 活动设置 -->
      <div v-if="isEventCategory" class="setting-item setting-vertical">
        <label class="setting-label">{{ t('community.create.eventType') }}</label>
        <div class="event-type-toggle">
          <button
            :class="{ active: form.event_type === 'server' }"
            @click="form.event_type = 'server'"
          >{{ t('community.create.eventTypeServer') }}</button>
          <button
            :class="{ active: form.event_type === 'guild' }"
            @click="form.event_type = 'guild'"
          >{{ t('community.create.eventTypeGuild') }}</button>
        </div>
        <div class="event-calendar-guide">
          <p class="event-calendar-guide-title">{{ t('community.create.eventCalendarGuideTitle') }}</p>
          <p class="event-calendar-guide-text">{{ t('community.create.eventCalendarGuideBody') }}</p>
          <p class="event-calendar-guide-text">{{ t('community.create.eventCalendarGuideGuild') }}</p>
        </div>
      </div>

      <!-- 标签 -->
      <div class="setting-item tags-setting">
        <label class="setting-label">{{ t('community.create.tags') }}</label>
        <div class="tags-list">
          <span
            v-for="tag in tags"
            :key="tag.id"
            class="tag-chip"
            :class="{ selected: selectedTags.includes(tag.id) }"
            @click="toggleTag(tag.id)"
          >{{ tag.name }}</span>
        </div>
      </div>

      <!-- 关联公会 -->
      <div v-if="!isEventCategory" class="setting-item setting-vertical">
        <label class="setting-label">{{ t('community.create.guild') }}</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">{{ t('community.create.guildNone') }}</option>
          <option v-for="g in guilds" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>

      <!-- 公开可见开关 -->
      <div v-if="!isEventCategory" class="setting-item setting-vertical visibility-setting">
        <label class="setting-label">{{ t('community.create.visibility') }}</label>
        <div class="visibility-toggle">
          <label class="switch">
            <input type="checkbox" v-model="form.is_public" />
            <span class="slider"></span>
          </label>
          <span class="visibility-hint">{{ form.guild_id
            ? (form.is_public ? t('community.create.visibilityGuildPublic') : t('community.create.visibilityGuildPrivate'))
            : (form.is_public ? t('community.create.visibilityPublic') : t('community.create.visibilityPrivate')) }}</span>
        </div>
      </div>

      <div v-if="isEventCategory && form.event_type === 'guild'" class="setting-item setting-vertical">
        <label class="setting-label">{{ t('community.create.guild') }}</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">{{ t('community.create.guildSelect') }}</option>
          <option v-for="g in guilds" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>

      <div v-if="isEventCategory && form.event_type" class="setting-item setting-vertical event-time-group">
        <label class="setting-label">{{ t('community.create.eventTime') }}</label>
        <div class="time-inputs-row">
          <div class="time-input-wrapper">
            <label class="time-sub-label">{{ t('community.create.eventStartTime') }}</label>
            <input type="datetime-local" v-model="form.event_start_time" class="time-input" />
          </div>
          <div class="time-separator">
            <i class="ri-arrow-right-line"></i>
          </div>
          <div class="time-input-wrapper">
            <label class="time-sub-label">{{ t('community.create.eventEndTime') }}</label>
            <input type="datetime-local" v-model="form.event_end_time" class="time-input" />
          </div>
        </div>
      </div>

      <!-- 活动颜色选择 -->
      <div v-if="isEventCategory && form.event_type" class="setting-item setting-vertical event-color-group">
        <label class="setting-label">{{ t('community.create.eventColor') }}</label>
        <div class="color-picker-wrapper">
          <div class="custom-color-input">
            <input type="color" v-model="form.event_color" class="color-input" />
            <span class="color-value">{{ form.event_color }}</span>
          </div>
        </div>
      </div>

      <!-- 合集选择 -->
      <CollectionSelector
        v-model="selectedCollectionId"
        content-type="post"
      />

      <!-- 操作按钮 -->
      <div class="actions-group">
        <button class="action-btn draft" @click="handleSubmit('draft')" :disabled="loading">
          <i class="ri-save-line"></i>
          {{ t('community.create.saveDraft') }}
        </button>
        <button class="action-btn preview" @click="handlePreview">
          <i class="ri-eye-line"></i>
          {{ t('community.create.preview') }}
        </button>
        <button class="action-btn cancel" @click="handleCancel">
          {{ t('community.create.cancel') }}
        </button>
        <button class="action-btn publish" @click="handleSubmit('published')" :disabled="loading">
          {{ t('community.create.publish') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.post-create-page {
  max-width: 1000px;
  margin: 0 auto;
}

/* ========== Page Header ========== */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.page-title {
  font-family: 'Merriweather', serif;
  font-size: 24px;
  font-weight: 700;
  color: #2C1810;
  margin: 0;
}

/* ========== Editor Container ========== */
.editor-container {
  background: #fff;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  padding: 32px 48px;
  margin-bottom: 20px;
}

.title-group {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid #F5EFE7;
}

/* ========== Cover Image ========== */
.cover-image-group {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid #F5EFE7;
}

.cover-label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #5D4037;
  margin-bottom: 12px;
}

.cover-upload-area {
  width: 100%;
}

.cover-preview {
  position: relative;
  width: 100%;
  max-height: 300px;
  border-radius: 12px;
  overflow: hidden;
  background: #f5f5f5;
}

.cover-preview img {
  width: 100%;
  height: auto;
  max-height: 300px;
  object-fit: contain;
  display: block;
}

.cover-preview-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 8px;
}

.edit-cover-btn,
.remove-cover-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: background 0.2s;
}

.edit-cover-btn:hover,
.remove-cover-btn:hover {
  background: rgba(0, 0, 0, 0.8);
}

.cover-placeholder {
  width: 100%;
  max-width: 400px;
  aspect-ratio: 16 / 9;
  border: 2px dashed var(--color-border-hover, #E5D4C1);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background:
    radial-gradient(circle at 28% 22%, rgba(255, 255, 255, 0.74), transparent 34%),
    linear-gradient(135deg, var(--color-card-bg, #FDFBF9), var(--color-primary-light, #FFF8F0));
}

.cover-placeholder:hover {
  border-color: var(--color-accent, #B87333);
  background:
    radial-gradient(circle at 28% 22%, rgba(255, 255, 255, 0.78), transparent 34%),
    linear-gradient(135deg, var(--color-primary-light, #FFF8F0), var(--color-card-bg-hover, #FFF8F0));
}

.cover-placeholder i {
  font-size: 32px;
  color: var(--color-accent, #B87333);
}

.cover-placeholder span {
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
}

.cover-hint {
  font-size: 12px !important;
  color: var(--color-text-muted, #A99B8D) !important;
}

.title-input {
  width: 100%;
  padding: 8px 0;
  font-family: 'Merriweather', serif;
  font-size: 28px;
  font-weight: 700;
  color: #2C1810;
  background: transparent;
  border: none;
  outline: none;
}

.title-input::placeholder {
  color: #E5D4C1;
}

.content-group {
  min-height: 400px;
}

/* ========== Settings Bar ========== */
.settings-bar {
  background: #fff;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  padding: 20px 24px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.setting-item.tags-setting {
  flex: 1;
  min-width: 200px;
}

.setting-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.setting-block-primary {
  flex: 1 1 460px;
  min-width: 320px;
}

.setting-item.setting-vertical {
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

.setting-item--category {
  width: 100%;
}

.setting-item--category .category-select {
  width: min(360px, 100%);
}

.location-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(220px, 1fr));
  gap: 12px;
}

.setting-item.location-setting {
  min-width: 0;
}

.location-text-input {
  width: 100%;
  padding: 12px 14px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 10px;
  font-size: 14px;
  color: #4B3621;
  outline: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: all 0.2s;
}

.location-text-input::placeholder {
  color: #A99B8D;
}

.location-text-input:hover {
  border-color: #B87333;
}

.location-text-input:focus {
  border-color: #804030;
  box-shadow: 0 0 0 2px rgba(128, 64, 48, 0.1);
}

@media (max-width: 900px) {
  .setting-block-primary {
    min-width: 100%;
  }

  .location-fields {
    grid-template-columns: 1fr;
  }
}

.setting-label {
  font-size: 12px;
  font-weight: 500;
  color: #8D7B68;
  white-space: nowrap;
}

/* Category Select */
.category-select {
  position: relative;
}

.category-select select {
  width: 100%;
  appearance: none;
  background: #fff;
  border: 1px solid #E5D4C1;
  padding: 12px 36px 12px 16px;
  font-size: 14px;
  color: #4B3621;
  cursor: pointer;
  outline: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: all 0.2s;
}

.category-select select:hover {
  border-color: #B87333;
}

.category-select select:focus {
  border-color: #804030;
  box-shadow: 0 0 0 2px rgba(128, 64, 48, 0.1);
}

.category-select::after {
  content: '';
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  width: 0;
  height: 0;
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-top: 5px solid #8D7B68;
  pointer-events: none;
}

/* Event Settings */
.event-settings {
  background: #fff;
  padding: 20px;
  border: 1px solid rgba(184, 115, 51, 0.3);
  box-shadow: inset 0 2px 4px 0 rgba(75, 54, 33, 0.02);
}

.event-header {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #B87333;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 16px;
}

.event-header i {
  font-size: 16px;
}

.event-type-toggle {
  display: flex;
  background: #F5EFE7;
  padding: 4px;
  margin-bottom: 16px;
}

.event-type-toggle button {
  flex: 1;
  padding: 8px 12px;
  background: transparent;
  border: none;
  font-size: 12px;
  font-weight: 500;
  color: #8D7B68;
  cursor: pointer;
  transition: all 0.2s;
}

.event-type-toggle button:hover {
  color: #4B3621;
}

.event-type-toggle button.active {
  background: #fff;
  color: #804030;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.event-calendar-guide {
  padding: 10px 12px;
  background: rgba(184, 115, 51, 0.08);
  border: 1px solid rgba(184, 115, 51, 0.2);
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.event-calendar-guide-title {
  font-size: 12px;
  font-weight: 600;
  color: #804030;
  margin: 0;
}

.event-calendar-guide-text {
  font-size: 12px;
  color: #5D4037;
  margin: 0;
  line-height: 1.45;
}

.event-guild {
  margin-bottom: 16px;
}

.event-guild label {
  display: block;
  font-size: 10px;
  text-transform: uppercase;
  color: #8D7B68;
  margin-bottom: 6px;
}

.event-guild select {
  width: 100%;
  padding: 10px 12px;
  background: #fff;
  border: 1px solid #E5D4C1;
  font-size: 13px;
  color: #4B3621;
  outline: none;
}

.event-guild select:focus {
  border-color: #804030;
}

.event-time label {
  display: block;
  font-size: 10px;
  text-transform: uppercase;
  color: #8D7B68;
  margin-bottom: 6px;
}

.event-time input {
  width: 100%;
  padding: 10px 12px;
  background: #fff;
  border: 1px solid #E5D4C1;
  font-size: 13px;
  color: #4B3621;
  outline: none;
}

.event-time input:focus {
  border-color: #804030;
}

/* Tags */
.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-chip {
  padding: 6px 12px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  font-size: 12px;
  color: #4B3621;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-chip:hover {
  border-color: #B87333;
  color: #B87333;
}

.tag-chip.selected {
  background: rgba(128, 64, 48, 0.1);
  border-color: rgba(128, 64, 48, 0.2);
  color: #804030;
}

/* Guild Select */
.guild-select {
  width: 100%;
  appearance: none;
  background: #fff;
  border: 1px solid #E5D4C1;
  padding: 12px 16px;
  font-size: 14px;
  color: #4B3621;
  cursor: pointer;
  outline: none;
  transition: all 0.2s;
}

.guild-select:hover {
  border-color: #B87333;
}

.guild-select:focus {
  border-color: #804030;
}

/* Event Time Inputs */
.event-time-group {
  width: 100%;
}

.time-inputs-row {
  display: flex;
  align-items: flex-end;
  gap: 16px;
  width: 100%;
}

.time-input-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.time-sub-label {
  font-size: 11px;
  font-weight: 500;
  color: #8D7B68;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.time-separator {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #B87333;
  font-size: 18px;
  padding-bottom: 8px;
}

.time-input {
  width: 100%;
  padding: 10px 12px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  font-size: 13px;
  color: #4B3621;
  outline: none;
  transition: all 0.2s;
  font-family: inherit;
}

.time-input:hover {
  border-color: #B87333;
}

.time-input:focus {
  border-color: #804030;
  box-shadow: 0 0 0 2px rgba(128, 64, 48, 0.1);
}

.time-input::-webkit-calendar-picker-indicator {
  cursor: pointer;
  filter: opacity(0.6);
  transition: filter 0.2s;
}

.time-input::-webkit-calendar-picker-indicator:hover {
  filter: opacity(1);
}

/* ========== Event Color Picker ========== */
.event-color-group {
  width: 100%;
}

.color-picker-wrapper {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.custom-color-input {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #F5EFE7;
  border-radius: 8px;
}

.color-input {
  width: 60px;
  height: 40px;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.color-input:hover {
  border-color: #B87333;
}

.color-value {
  font-size: 13px;
  font-weight: 600;
  color: #4B3621;
  font-family: 'Courier New', monospace;
  text-transform: uppercase;
}

/* ========== Animation ========== */
.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }

/* ========== Actions Group ========== */
.actions-group {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}

.actions-group .action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.actions-group .action-btn.preview {
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  color: #4B3621;
}

.actions-group .action-btn.draft {
  color: #76512c;
  border-color: rgba(118, 81, 44, 0.28);
  background: #fbf8f3;
}

.actions-group .action-btn.preview:hover {
  border-color: #B87333;
  color: #B87333;
}

.actions-group .action-btn.cancel {
  background: transparent;
  border: none;
  color: #8D7B68;
}

.actions-group .action-btn.cancel:hover {
  color: #2C1810;
}

.actions-group .action-btn.publish {
  background: #804030;
  border: none;
  color: #fff;
  box-shadow: 0 2px 8px rgba(128, 64, 48, 0.2);
}

.actions-group .action-btn.publish:hover {
  background: #6B3528;
}

.actions-group .action-btn.publish:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ========== Visibility Toggle ========== */
.visibility-setting {
  margin-top: 12px;
}

.visibility-toggle {
  display: flex;
  align-items: center;
  gap: 12px;
}

.visibility-hint {
  font-size: 13px;
  color: #8D7B68;
}

/* Switch Toggle */
.switch {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #E5D4C1;
  transition: 0.3s;
  border-radius: 26px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

input:checked + .slider {
  background-color: #804030;
}

input:checked + .slider:before {
  transform: translateX(22px);
}
</style>
