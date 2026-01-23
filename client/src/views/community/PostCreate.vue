<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { createPost, type CreatePostRequest, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { uploadImage } from '@/api/item'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import TiptapEditor from '@/components/TiptapEditor.vue'
import PostQuickJump from '@/components/PostQuickJump.vue'

const DRAFT_KEY = 'post_create_draft'

const router = useRouter()
const toast = useToastStore()
const userStore = useUserStore()
const mounted = ref(false)
const loading = ref(false)

const form = ref<CreatePostRequest>({
  title: '',
  content: '',
  content_type: 'html',
  category: 'other',
  tag_ids: [],
  status: 'published',
  cover_image: '',
  is_public: true,  // 公会外成员可见（默认开启）
  event_type: undefined,
  event_start_time: undefined,
  event_end_time: undefined,
  event_color: '#D97706',
})

// 封面图相关
const coverImagePreview = ref('')
const coverImageLoading = ref(false)
const coverImageInput = ref<HTMLInputElement | null>(null)
const editorRef = ref<InstanceType<typeof TiptapEditor> | null>(null)
const quickJumpOpen = ref(false)

// 是否为活动分区
const isEventCategory = computed(() => form.value.category === 'event')

// 权限检查：是否可以发布服务器活动
const canPostServerEvent = computed(() => userStore.isModerator)

// 有管理权限的公会列表（owner 或 admin）
const adminGuilds = computed(() => {
  return guilds.value.filter(g => g.my_role === 'owner' || g.my_role === 'admin')
})

// 权限检查：是否可以发布公会活动
const canPostGuildEvent = computed(() => adminGuilds.value.length > 0)

// 监听分区变化，重置活动相关字段
watch(() => form.value.category, (newVal) => {
  if (newVal !== 'event') {
    form.value.event_type = undefined
    form.value.event_start_time = undefined
    form.value.event_end_time = undefined
  }
})

const tags = ref<Tag[]>([])
const guilds = ref<Guild[]>([])
const selectedTags = ref<number[]>([])
let autoSaveTimer: ReturnType<typeof setInterval> | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null

// 保存草稿到 localStorage
function saveDraft() {
  const draft = {
    form: form.value,
    selectedTags: selectedTags.value,
    savedAt: Date.now()
  }
  localStorage.setItem(DRAFT_KEY, JSON.stringify(draft))
}

// 防抖保存
function debouncedSaveDraft() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(saveDraft, 1000)
}

// 从 localStorage 恢复草稿
function loadDraft() {
  const saved = localStorage.getItem(DRAFT_KEY)
  if (saved) {
    try {
      const draft = JSON.parse(saved)
      if (draft.form) {
        form.value = { ...form.value, ...draft.form }
      }
      if (draft.selectedTags) {
        selectedTags.value = draft.selectedTags
      }
    } catch (e) {
      console.error('恢复草稿失败:', e)
    }
  }
}

// 清除草稿
function clearDraft() {
  localStorage.removeItem(DRAFT_KEY)
}

onMounted(async () => {
  // 检查登录状态
  if (!userStore.user || !userStore.token) {
    toast.warning('请先登录后再发帖')
    router.push('/login')
    return
  }

  setTimeout(() => mounted.value = true, 50)
  loadDraft()
  await loadTags()
  await loadGuilds()

  // 每 10 秒自动保存一次
  autoSaveTimer = setInterval(saveDraft, 10000)
})

onUnmounted(() => {
  if (autoSaveTimer) {
    clearInterval(autoSaveTimer)
  }
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
})

// 监听表单变化，自动保存草稿
watch([() => form.value.title, () => form.value.content, selectedTags], debouncedSaveDraft, { deep: true })

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
    toast.warning('请输入标题')
    return
  }
  if (!form.value.content.trim()) {
    toast.warning('请输入内容')
    return
  }

  // 活动分区权限验证
  if (form.value.category === 'event') {
    if (form.value.event_type === 'server' && !canPostServerEvent.value) {
      toast.error('你没有发布服务器活动的权限')
      return
    }
    if (form.value.event_type === 'guild') {
      if (!canPostGuildEvent.value) {
        toast.error('你没有发布公会活动的权限')
        return
      }
      if (!form.value.guild_id) {
        toast.warning('请选择要发布活动的公会')
        return
      }
    }
  }

  loading.value = true
  try {
    form.value.status = status
    form.value.tag_ids = selectedTags.value

    // 转换时间格式为 ISO8601/RFC3339
    if (form.value.event_start_time) {
      form.value.event_start_time = new Date(form.value.event_start_time).toISOString()
    }
    if (form.value.event_end_time) {
      form.value.event_end_time = new Date(form.value.event_end_time).toISOString()
    }

    await createPost(form.value)
    clearDraft() // 发布成功后清除草稿
    toast.success(status === 'published' ? '发布成功，等待审核' : '保存草稿成功')
    router.push({ name: 'my-posts' })
  } catch (error: any) {
    console.error('提交失败:', error)
    const msg = error?.message || '提交失败，请重试'
    toast.error(msg)
  } finally {
    loading.value = false
  }
}

function handleCancel() {
  router.back()
}

function handlePreview() {
  // 先保存草稿
  saveDraft()

  // 保存预览数据到 sessionStorage
  const previewData = {
    title: form.value.title,
    content: form.value.content,
    category: form.value.category,
    tag_ids: selectedTags.value,
    guild_id: form.value.guild_id,
    event_type: form.value.event_type,
    event_start_time: form.value.event_start_time,
    event_end_time: form.value.event_end_time,
    // 用于显示的额外信息
    selectedTagNames: tags.value.filter(t => selectedTags.value.includes(t.id)).map(t => t.name),
  }
  sessionStorage.setItem('post_preview', JSON.stringify(previewData))
  router.push({ name: 'post-preview' })
}

// 压缩图片到指定大小以内
async function compressImage(file: File, maxSizeKB: number = 1024): Promise<File> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const img = new Image()
      img.onload = () => {
        const canvas = document.createElement('canvas')
        let { width, height } = img

        // 限制最大尺寸
        const maxDimension = 1920
        if (width > maxDimension || height > maxDimension) {
          if (width > height) {
            height = (height / width) * maxDimension
            width = maxDimension
          } else {
            width = (width / height) * maxDimension
            height = maxDimension
          }
        }

        canvas.width = width
        canvas.height = height
        const ctx = canvas.getContext('2d')!
        ctx.drawImage(img, 0, 0, width, height)

        const toBlob = (quality: number) => new Promise<Blob>((resolveBlob, rejectBlob) => {
          canvas.toBlob((blob) => {
            if (!blob) {
              rejectBlob(new Error('图片处理失败'))
              return
            }
            resolveBlob(blob)
          }, 'image/jpeg', quality)
        })

        void (async () => {
          // 逐步降低质量直到满足大小要求
          let quality = 0.9
          let blob = await toBlob(quality)
          while (blob.size > maxSizeKB * 1024 && quality > 0.1) {
            quality -= 0.1
            blob = await toBlob(quality)
          }

          const baseName = file.name.replace(/\.[^.]+$/, '') || 'cover'
          resolve(new File([blob], `${baseName}.jpg`, { type: 'image/jpeg' }))
        })().catch(reject)
      }
      img.onerror = reject
      img.src = e.target?.result as string
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

// 处理封面图上传
async function handleCoverImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.error('请选择图片文件')
    return
  }

  coverImageLoading.value = true
  try {
    const compressed = await compressImage(file, 1024)
    const res: any = await uploadImage(compressed)
    const url = res?.data?.url || res?.url
    if (!url) {
      throw new Error('未获取到图片地址')
    }
    coverImagePreview.value = url
    form.value.cover_image = url
    toast.success('封面图上传成功')
  } catch (error: any) {
    console.error('封面图上传失败:', error)
    toast.error(error?.message || '封面图上传失败')
  } finally {
    coverImageLoading.value = false
    input.value = ''
  }
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
      <h1 class="page-title">发布新卷轴</h1>
    </div>

    <!-- 编辑区域 -->
    <div class="editor-container anim-item" style="--delay: 1">
      <!-- 标题输入 -->
      <div class="title-group">
        <input
          v-model="form.title"
          type="text"
          placeholder="输入一个吸引人的标题..."
          class="title-input"
        />
      </div>

      <!-- 封面图上传 -->
      <div class="cover-image-group">
        <label class="cover-label">封面图（可选）</label>
        <div class="cover-upload-area">
          <div v-if="coverImagePreview" class="cover-preview">
            <img :src="coverImagePreview" alt="封面预览" />
            <button class="remove-cover-btn" @click="removeCoverImage">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div v-else class="cover-placeholder" @click="coverImageInput?.click()">
            <i class="ri-image-add-line"></i>
            <span>{{ coverImageLoading ? '处理中...' : '点击上传封面图' }}</span>
            <span class="cover-hint">建议尺寸 16:9，自动压缩到 1MB 以内</span>
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

      <!-- 内容编辑器 -->
      <div class="content-group">
        <TiptapEditor
          ref="editorRef"
          v-model="form.content"
          placeholder="开始书写你的传奇..."
        >
          <template #toolbar>
            <button
              type="button"
              class="toolbar-slot"
              :class="{ active: quickJumpOpen }"
              title="快速跳转"
              @mousedown.prevent
              @click="toggleQuickJump"
            >
              <i class="ri-links-line"></i>
            </button>
          </template>
        </TiptapEditor>
      </div>

      <PostQuickJump v-model="quickJumpOpen" :on-insert="handleQuickInsert" />
    </div>

    <!-- 设置区域 -->
    <div class="settings-bar anim-item" style="--delay: 2">
      <!-- 分区选择 -->
      <div class="setting-item">
        <label class="setting-label">分区</label>
        <div class="category-select">
          <select v-model="form.category">
            <option v-for="cat in POST_CATEGORIES" :key="cat.value" :value="cat.value">
              {{ cat.label }}
            </option>
          </select>
        </div>
      </div>

      <!-- 活动设置 -->
      <div v-if="isEventCategory" class="setting-item setting-vertical">
        <label class="setting-label">活动类型</label>
        <div class="event-type-toggle">
          <button
            :class="{ active: form.event_type === 'server' }"
            :disabled="!canPostServerEvent"
            @click="form.event_type = 'server'"
          >服务器</button>
          <button
            :class="{ active: form.event_type === 'guild' }"
            :disabled="!canPostGuildEvent"
            @click="form.event_type = 'guild'"
          >公会</button>
        </div>
      </div>

      <!-- 标签 -->
      <div class="setting-item tags-setting">
        <label class="setting-label">标签</label>
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
        <label class="setting-label">公会</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">不关联</option>
          <option v-for="g in guilds" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>

      <!-- 公会外可见开关（当关联公会时显示） -->
      <div v-if="!isEventCategory && form.guild_id" class="setting-item setting-vertical visibility-setting">
        <label class="setting-label">公会外可见</label>
        <div class="visibility-toggle">
          <label class="switch">
            <input type="checkbox" v-model="form.is_public" />
            <span class="slider"></span>
          </label>
          <span class="visibility-hint">{{ form.is_public ? '帖子将同时显示在社区广场' : '仅公会成员可见' }}</span>
        </div>
      </div>

      <div v-if="isEventCategory && form.event_type === 'guild'" class="setting-item setting-vertical">
        <label class="setting-label">公会</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">请选择</option>
          <option v-for="g in adminGuilds" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
      </div>

      <div v-if="isEventCategory && form.event_type" class="setting-item setting-vertical event-time-group">
        <label class="setting-label">活动时间</label>
        <div class="time-inputs-row">
          <div class="time-input-wrapper">
            <label class="time-sub-label">开始时间</label>
            <input type="datetime-local" v-model="form.event_start_time" class="time-input" />
          </div>
          <div class="time-separator">
            <i class="ri-arrow-right-line"></i>
          </div>
          <div class="time-input-wrapper">
            <label class="time-sub-label">结束时间</label>
            <input type="datetime-local" v-model="form.event_end_time" class="time-input" />
          </div>
        </div>
      </div>

      <!-- 活动颜色选择 -->
      <div v-if="isEventCategory && form.event_type" class="setting-item setting-vertical event-color-group">
        <label class="setting-label">活动标记颜色</label>
        <div class="color-picker-wrapper">
          <div class="custom-color-input">
            <input type="color" v-model="form.event_color" class="color-input" />
            <span class="color-value">{{ form.event_color }}</span>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="actions-group">
        <button class="action-btn preview" @click="handlePreview">
          <i class="ri-eye-line"></i>
          预览
        </button>
        <button class="action-btn cancel" @click="handleCancel">
          取消
        </button>
        <button class="action-btn publish" @click="handleSubmit('published')" :disabled="loading">
          发布
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

.remove-cover-btn {
  position: absolute;
  top: 8px;
  right: 8px;
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

.remove-cover-btn:hover {
  background: rgba(0, 0, 0, 0.8);
}

.cover-placeholder {
  width: 100%;
  max-width: 400px;
  aspect-ratio: 16 / 9;
  border: 2px dashed #E5D4C1;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background: #FDFBF9;
}

.cover-placeholder:hover {
  border-color: #B87333;
  background: #FFF8F0;
}

.cover-placeholder i {
  font-size: 32px;
  color: #B87333;
}

.cover-placeholder span {
  font-size: 14px;
  color: #8D7B68;
}

.cover-hint {
  font-size: 12px !important;
  color: #A99B8D !important;
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
}

.setting-item.tags-setting {
  flex: 1;
  min-width: 200px;
}

.setting-item.setting-vertical {
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
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

.event-type-toggle button:hover:not(:disabled) {
  color: #4B3621;
}

.event-type-toggle button.active {
  background: #fff;
  color: #804030;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.event-type-toggle button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
