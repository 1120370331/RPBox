<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getPost, updatePost, getPostTags, type UpdatePostRequest, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { addPostTag, removePostTag } from '@/api/post'
import TiptapEditor from '@/components/TiptapEditor.vue'
import { useToast } from '@/composables/useToast'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const toast = useToast()
const userStore = useUserStore()
const route = useRoute()
const mounted = ref(false)
const loading = ref(false)

// 草稿 key 基于帖子 ID
const getDraftKey = () => `post_edit_draft_${route.params.id}`

const form = ref<UpdatePostRequest>({
  title: '',
  content: '',
  content_type: 'html',
  category: 'other',
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

// 是否为活动分区
const isEventCategory = computed(() => form.value.category === 'event')

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
const originalTags = ref<number[]>([])
let autoSaveTimer: ReturnType<typeof setInterval> | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null

// 保存草稿
function saveDraft() {
  const draft = {
    form: form.value,
    selectedTags: selectedTags.value,
    savedAt: Date.now()
  }
  localStorage.setItem(getDraftKey(), JSON.stringify(draft))
}

// 防抖保存
function debouncedSaveDraft() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(saveDraft, 1000)
}

// 恢复草稿
function loadDraft() {
  const saved = localStorage.getItem(getDraftKey())
  if (saved) {
    try {
      const draft = JSON.parse(saved)
      if (draft.form) {
        form.value = { ...form.value, ...draft.form }
      }
      if (draft.selectedTags) {
        selectedTags.value = draft.selectedTags
      }
      return true
    } catch (e) {
      console.error('恢复草稿失败:', e)
    }
  }
  return false
}

// 清除草稿
function clearDraft() {
  localStorage.removeItem(getDraftKey())
}

onMounted(async () => {
  // 检查登录状态
  if (!userStore.user || !userStore.token) {
    toast.warning('请先登录后再编辑帖子')
    router.push('/login')
    return
  }

  setTimeout(() => mounted.value = true, 50)

  // 先尝试恢复草稿
  const hasDraft = loadDraft()

  // 如果没有草稿，从服务器加载
  if (!hasDraft) {
    await loadPost()
  }

  await loadTags()
  await loadGuilds()
  await loadPostTags()

  // 每 10 秒自动保存
  autoSaveTimer = setInterval(saveDraft, 10000)
})

onUnmounted(() => {
  if (autoSaveTimer) clearInterval(autoSaveTimer)
  if (debounceTimer) clearTimeout(debounceTimer)
})

// 监听表单变化，自动保存
watch([() => form.value.title, () => form.value.content, selectedTags], debouncedSaveDraft, { deep: true })

async function loadPost() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res = await getPost(id)
    form.value.title = res.post.title
    form.value.content = res.post.content
    form.value.content_type = res.post.content_type
    form.value.category = res.post.category || 'other'
    form.value.guild_id = res.post.guild_id
    form.value.status = res.post.status
    form.value.is_public = res.post.is_public ?? true
    // 加载封面图
    if (res.post.cover_image) {
      form.value.cover_image = res.post.cover_image
      coverImagePreview.value = res.post.cover_image
    }
    // 加载活动字段
    form.value.event_type = res.post.event_type
    if (res.post.event_start_time) {
      form.value.event_start_time = res.post.event_start_time.slice(0, 16)
    }
    if (res.post.event_end_time) {
      form.value.event_end_time = res.post.event_end_time.slice(0, 16)
    }
    if (res.post.event_color) {
      form.value.event_color = res.post.event_color
    }
  } catch (error) {
    console.error('加载帖子失败:', error)
    toast.error('帖子不存在')
    router.back()
  } finally {
    loading.value = false
  }
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

async function loadPostTags() {
  try {
    const id = Number(route.params.id)
    const res = await getPostTags(id)
    originalTags.value = res.tags.map((t: any) => t.id)
    selectedTags.value = [...originalTags.value]
  } catch (error) {
    console.error('加载帖子标签失败:', error)
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
  if (!form.value.title?.trim()) {
    toast.warning('请输入标题')
    return
  }
  if (!form.value.content?.trim()) {
    toast.warning('请输入内容')
    return
  }

  loading.value = true
  try {
    const id = Number(route.params.id)
    form.value.status = status
    await updatePost(id, form.value)

    // 更新标签
    const addedTags = selectedTags.value.filter(t => !originalTags.value.includes(t))
    const removedTags = originalTags.value.filter(t => !selectedTags.value.includes(t))

    for (const tagId of addedTags) {
      await addPostTag(id, tagId)
    }
    for (const tagId of removedTags) {
      await removePostTag(id, tagId)
    }

    clearDraft() // 保存成功后清除草稿
    toast.success('更新成功')
    router.push({ name: 'post-detail', params: { id } })
  } catch (error) {
    console.error('更新失败:', error)
    toast.error('更新失败，请重试')
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

  const previewData = {
    title: form.value.title,
    content: form.value.content,
    category: form.value.category,
    tag_ids: selectedTags.value,
    guild_id: form.value.guild_id,
    event_type: form.value.event_type,
    event_start_time: form.value.event_start_time,
    event_end_time: form.value.event_end_time,
    selectedTagNames: tags.value.filter(t => selectedTags.value.includes(t.id)).map(t => t.name),
  }
  sessionStorage.setItem('post_preview', JSON.stringify(previewData))
  router.push({ name: 'post-preview' })
}

// 压缩图片到指定大小以内
async function compressImage(file: File, maxSizeKB: number = 1024): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const img = new Image()
      img.onload = () => {
        const canvas = document.createElement('canvas')
        let { width, height } = img

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

        let quality = 0.9
        let result = canvas.toDataURL('image/jpeg', quality)

        while (result.length > maxSizeKB * 1024 * 1.37 && quality > 0.1) {
          quality -= 0.1
          result = canvas.toDataURL('image/jpeg', quality)
        }

        resolve(result)
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
    coverImagePreview.value = compressed
    form.value.cover_image = compressed
    toast.success('封面图上传成功')
  } catch (error) {
    console.error('图片压缩失败:', error)
    toast.error('图片处理失败')
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
</script>

<template>
  <div class="post-edit-page" :class="{ 'animate-in': mounted }">
    <!-- 头部 -->
    <div class="page-header anim-item" style="--delay: 0">
      <h1 class="page-title">编辑卷轴</h1>
    </div>

    <div v-if="loading && !form.title" class="loading">加载中...</div>

    <!-- 编辑区域 -->
    <div v-else class="editor-container anim-item" style="--delay: 1">
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
          v-model="form.content"
          placeholder="开始书写你的传奇..."
        />
      </div>
    </div>

    <!-- 设置区域 -->
    <div v-if="!loading || form.title" class="settings-bar anim-item" style="--delay: 2">
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

      <!-- 活动设置 -->
      <div v-if="isEventCategory" class="setting-item setting-vertical">
        <label class="setting-label">活动类型</label>
        <div class="event-type-toggle">
          <button
            :class="{ active: form.event_type === 'server' }"
            @click="form.event_type = 'server'"
          >服务器</button>
          <button
            :class="{ active: form.event_type === 'guild' }"
            @click="form.event_type = 'guild'"
          >公会</button>
        </div>
      </div>

      <div v-if="isEventCategory && form.event_type === 'guild'" class="setting-item setting-vertical">
        <label class="setting-label">公会</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">请选择</option>
          <option v-for="g in guilds" :key="g.id" :value="g.id">{{ g.name }}</option>
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
          <div class="preset-colors">
            <button
              v-for="color in ['#D97706', '#2196F3', '#16A34A', '#DC2626', '#9333EA', '#EA580C']"
              :key="color"
              class="color-preset"
              :class="{ active: form.event_color === color }"
              :style="{ backgroundColor: color }"
              @click="form.event_color = color"
              type="button"
            ></button>
          </div>
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
          保存
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.post-edit-page {
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

.loading {
  text-align: center;
  padding: 60px;
  color: #8D7B68;
  font-size: 18px;
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
.event-type-toggle {
  display: flex;
  background: #F5EFE7;
  padding: 4px;
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

.time-input {
  padding: 8px 12px;
  background: #fff;
  border: 1px solid #E5D4C1;
  font-size: 13px;
  color: #4B3621;
  outline: none;
}

.time-input:focus {
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

/* Event Color Picker */
.event-color-group {
  width: 100%;
}

.color-picker-wrapper {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preset-colors {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.color-preset {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  border: 3px solid transparent;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.color-preset:hover {
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.color-preset.active {
  border-color: #2C1810;
  box-shadow: 0 0 0 2px #fff, 0 0 0 4px #2C1810;
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
