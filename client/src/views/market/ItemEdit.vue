<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getItem, updateItem, getItemTags, addItemTag, removeItemTag, getItemImages, uploadItemImages, deleteItemImage, type Item, type UpdateItemRequest, type ItemImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { useToast } from '@/composables/useToast'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const userStore = useUserStore()
const mounted = ref(false)
const loading = ref(false)
const item = ref<Item | null>(null)

const form = ref<UpdateItemRequest>({
  name: '',
  description: '',
  icon: '',
  import_code: '',
  requires_permission: false,
  enable_watermark: true,
  status: 'published'
})

// 画作图片管理
const existingImages = ref<ItemImage[]>([])  // 已有的图片
const newImages = ref<{ file: File; preview: string }[]>([])  // 新添加的图片
const imagesToDelete = ref<number[]>([])  // 待删除的图片ID
const artworkImagesInput = ref<HTMLInputElement | null>(null)

// 是否为画作类型
const isArtwork = computed(() => item.value?.type === 'artwork')

const itemTags = ref<Tag[]>([])
const selectedTags = ref<number[]>([])
const originalTags = ref<number[]>([])

// 是否有待审核的编辑
const hasPendingEdit = ref(false)

onMounted(async () => {
  // 检查登录状态
  if (!userStore.user || !userStore.token) {
    toast.info('请先登录后再编辑作品')
    router.push('/login')
    return
  }

  setTimeout(() => mounted.value = true, 50)
  await loadItem()
  await loadTags()
  await loadItemTags()
})

async function loadItem() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res: any = await getItem(id)
    if (res.code === 0 && res.data?.item) {
      item.value = res.data.item
      form.value.name = res.data.item.name
      form.value.description = res.data.item.description
      form.value.icon = res.data.item.icon
      form.value.import_code = res.data.item.import_code || ''
      form.value.requires_permission = res.data.item.requires_permission || false
      form.value.enable_watermark = res.data.item.enable_watermark ?? true
      form.value.status = res.data.item.status
      hasPendingEdit.value = !!res.data.pending_edit

      // 如果是画作类型，加载图片
      if (res.data.item.type === 'artwork') {
        await loadArtworkImages(id)
      }
    } else {
      throw new Error('作品不存在')
    }
  } catch (error) {
    console.error('加载作品失败:', error)
    toast.error('作品不存在')
    router.back()
  } finally {
    loading.value = false
  }
}

// 加载画作图片
async function loadArtworkImages(itemId: number) {
  try {
    const res: any = await getItemImages(itemId)
    if (res.code === 0 && res.data) {
      existingImages.value = res.data
    }
  } catch (error) {
    console.error('加载画作图片失败:', error)
  }
}

async function loadTags() {
  try {
    const res: any = await getPresetTags('item')
    if (res.tags) {
      itemTags.value = res.tags
    }
  } catch (error) {
    console.error('加载标签失败:', error)
  }
}

async function loadItemTags() {
  try {
    const id = Number(route.params.id)
    const res: any = await getItemTags(id)
    if (res.code === 0) {
      originalTags.value = (res.data || []).map((t: any) => t.id)
      selectedTags.value = [...originalTags.value]
    }
  } catch (error) {
    console.error('加载作品标签失败:', error)
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

// 选择新的画作图片
function handleArtworkImagesSelect(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const maxImages = 20
  const currentCount = existingImages.value.filter(img => !imagesToDelete.value.includes(img.id)).length + newImages.value.length
  const remainingSlots = maxImages - currentCount

  if (remainingSlots <= 0) {
    toast.info('最多只能上传20张图片')
    return
  }

  const filesToAdd = Array.from(input.files).slice(0, remainingSlots)

  for (const file of filesToAdd) {
    if (file.size > 10 * 1024 * 1024) {
      toast.info(`图片 ${file.name} 超过10MB，已跳过`)
      continue
    }

    if (!file.type.startsWith('image/')) {
      toast.info(`文件 ${file.name} 不是图片，已跳过`)
      continue
    }

    // 创建预览
    const reader = new FileReader()
    reader.onload = (e) => {
      newImages.value.push({
        file,
        preview: e.target?.result as string
      })
    }
    reader.readAsDataURL(file)
  }

  // 清空 input 以允许重复选择相同文件
  input.value = ''
}

// 标记删除已有图片
function markImageForDeletion(imageId: number) {
  imagesToDelete.value.push(imageId)
}

// 撤销删除标记
function unmarkImageForDeletion(imageId: number) {
  const index = imagesToDelete.value.indexOf(imageId)
  if (index > -1) {
    imagesToDelete.value.splice(index, 1)
  }
}

// 移除新添加的图片
function removeNewImage(index: number) {
  newImages.value.splice(index, 1)
}

// 获取图片总数（排除待删除的）
function getTotalImageCount() {
  const existingCount = existingImages.value.filter(img => !imagesToDelete.value.includes(img.id)).length
  return existingCount + newImages.value.length
}

async function handleSubmit(status: 'draft' | 'published') {
  if (!form.value.name?.trim()) {
    toast.info('请输入作品名称')
    return
  }

  // 画作类型需要至少保留一张图片
  if (isArtwork.value && getTotalImageCount() === 0) {
    toast.info('画作类型至少需要一张图片')
    return
  }

  loading.value = true
  try {
    const id = Number(route.params.id)
    form.value.status = status
    await updateItem(id, form.value)

    // 更新标签
    const addedTags = selectedTags.value.filter(t => !originalTags.value.includes(t))
    const removedTags = originalTags.value.filter(t => !selectedTags.value.includes(t))

    for (const tagId of addedTags) {
      await addItemTag(id, tagId)
    }
    for (const tagId of removedTags) {
      await removeItemTag(id, tagId)
    }

    // 处理画作图片
    if (isArtwork.value) {
      // 删除标记的图片
      for (const imageId of imagesToDelete.value) {
        try {
          await deleteItemImage(id, imageId)
        } catch (err) {
          console.error('删除图片失败:', err)
        }
      }

      // 上传新图片
      if (newImages.value.length > 0) {
        try {
          const files = newImages.value.map(img => img.file)
          await uploadItemImages(id, files)
        } catch (err) {
          console.error('上传图片失败:', err)
          toast.info('部分图片上传失败')
        }
      }
    }

    if (item.value?.status === 'published' && status === 'published') {
      toast.success('编辑已提交，等待审核')
    } else {
      toast.success('更新成功')
    }
    router.push({ name: 'item-detail', params: { id } })
  } catch (error: any) {
    console.error('更新失败:', error)
    toast.error(error.message || '更新失败，请重试')
  } finally {
    loading.value = false
  }
}

function handleCancel() {
  router.back()
}

// 预览
function handlePreview() {
  if (!item.value) return

  // 构建预览数据
  const previewData = {
    ...item.value,
    name: form.value.name || item.value.name,
    description: form.value.description || item.value.description,
    icon: form.value.icon || item.value.icon,
  }

  sessionStorage.setItem('item_preview_data', JSON.stringify(previewData))
  sessionStorage.setItem('item_preview_from', route.fullPath)

  router.push('/market/preview')
}

function getTypeText(type: string) {
  const typeMap: Record<string, string> = {
    'item': '道具',
    'campaign': '剧本',
    'artwork': '画作'
  }
  return typeMap[type] || type
}
</script>

<template>
  <div class="item-edit-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <h1 class="page-title">编辑作品</h1>
      <div class="actions">
        <button class="cancel-btn" @click="handleCancel">取消</button>
        <button class="preview-btn" @click="handlePreview" :disabled="loading">
          <i class="ri-eye-line"></i>
          预览
        </button>
        <button class="draft-btn" @click="handleSubmit('draft')" :disabled="loading">
          保存草稿
        </button>
        <button class="publish-btn" @click="handleSubmit('published')" :disabled="loading">
          <i class="ri-save-line"></i>
          保存
        </button>
      </div>
    </div>

    <!-- 待审核提示 -->
    <div v-if="hasPendingEdit" class="pending-notice anim-item" style="--delay: 1">
      <i class="ri-time-line"></i>
      <span>您有一个编辑正在等待审核，再次提交将覆盖之前的编辑</span>
    </div>

    <div v-if="loading && !item" class="loading">加载中...</div>

    <div v-else-if="item" class="form-container anim-item" style="--delay: 2">
      <!-- 作品类型（只读） -->
      <div class="form-group">
        <label>作品类型</label>
        <div class="type-badge">{{ getTypeText(item.type) }}</div>
      </div>

      <!-- 作品名称 -->
      <div class="form-group">
        <label>作品名称 <span class="required">*</span></label>
        <input
          v-model="form.name"
          type="text"
          placeholder="请输入作品名称"
          class="title-input"
        />
      </div>

      <!-- 描述 -->
      <div class="form-group">
        <label>描述</label>
        <textarea
          v-model="form.description"
          placeholder="请描述这个作品的功能和特点..."
          rows="4"
          class="content-textarea"
        ></textarea>
      </div>

      <!-- 标签选择 -->
      <div class="form-group" v-if="itemTags.length > 0">
        <label>作品分类标签</label>
        <div class="tag-list">
          <div
            v-for="tag in itemTags"
            :key="tag.id"
            class="tag-item"
            :class="{ selected: selectedTags.includes(tag.id) }"
            @click="toggleTag(tag.id)"
          >
            {{ tag.name }}
          </div>
        </div>
      </div>

      <!-- 画作图片管理（仅画作类型） -->
      <div class="form-group" v-if="isArtwork">
        <label>画作图片 <span class="required">*</span></label>
        <div class="artwork-images">
          <div class="image-grid">
            <!-- 已有的图片 -->
            <div
              v-for="img in existingImages"
              :key="'existing-' + img.id"
              class="image-item"
              :class="{ 'marked-delete': imagesToDelete.includes(img.id) }"
            >
              <img :src="img.image_url" alt="画作图片" />
              <button
                v-if="!imagesToDelete.includes(img.id)"
                type="button"
                class="remove-btn"
                @click="markImageForDeletion(img.id)"
                title="删除图片"
              >
                <i class="ri-close-line"></i>
              </button>
              <button
                v-else
                type="button"
                class="undo-btn"
                @click="unmarkImageForDeletion(img.id)"
                title="撤销删除"
              >
                <i class="ri-arrow-go-back-line"></i>
              </button>
              <span v-if="imagesToDelete.includes(img.id)" class="delete-overlay">
                将被删除
              </span>
            </div>

            <!-- 新添加的图片 -->
            <div
              v-for="(img, index) in newImages"
              :key="'new-' + index"
              class="image-item new-image"
            >
              <img :src="img.preview" alt="新图片" />
              <button
                type="button"
                class="remove-btn"
                @click="removeNewImage(index)"
                title="移除图片"
              >
                <i class="ri-close-line"></i>
              </button>
              <span class="new-badge">新</span>
            </div>

            <!-- 添加更多按钮 -->
            <div
              v-if="getTotalImageCount() < 20"
              class="add-more-btn"
              @click="artworkImagesInput?.click()"
            >
              <i class="ri-add-line"></i>
              <span>添加图片</span>
            </div>
          </div>
          <input
            ref="artworkImagesInput"
            type="file"
            accept="image/*"
            multiple
            style="display: none"
            @change="handleArtworkImagesSelect"
          />
        </div>
        <p class="hint">支持 JPG、PNG 格式，单张最大 10MB，最多上传 20 张</p>
      </div>

      <!-- 水印设置（仅画作类型） -->
      <div class="form-group" v-if="isArtwork">
        <label>水印设置</label>
        <div class="watermark-toggle">
          <label class="toggle-switch">
            <input type="checkbox" v-model="form.enable_watermark" />
            <span class="slider"></span>
          </label>
          <span class="toggle-label">下载时添加水印（用户名）</span>
        </div>
        <p class="hint">开启后，其他用户下载图片时会自动在右下角添加你的用户名作为水印</p>
      </div>

      <!-- 导入代码（非画作类型可编辑） -->
      <div class="form-group" v-if="item.type !== 'artwork'">
        <label>TRP3 导入代码</label>
        <textarea
          v-model="form.import_code"
          placeholder="粘贴 TRP3 导入代码..."
          rows="6"
          class="code-textarea"
        ></textarea>
        <p class="hint">更新导入代码用于版本迭代，留空则保持原有代码不变</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.item-edit-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 900px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title {
  font-size: 42px;
  color: #4B3621;
  margin: 0;
}

.actions {
  display: flex;
  gap: 12px;
}

.cancel-btn,
.preview-btn,
.draft-btn,
.publish-btn {
  padding: 12px 24px;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.cancel-btn {
  background: #E5D4C1;
  color: #4B3621;
}

.preview-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  color: #5D4037;
  border: 2px solid #E5D4C1;
}

.preview-btn:hover {
  border-color: #B87333;
  color: #B87333;
}

.draft-btn {
  background: #fff;
  color: #B87333;
  border: 2px solid #B87333;
}

.publish-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #B87333;
  color: #fff;
}

.publish-btn:hover {
  background: #A66629;
  transform: translateY(-2px);
}

.publish-btn:disabled,
.draft-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pending-notice {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: #FFF3E0;
  border: 2px solid #FFB74D;
  border-radius: 12px;
  color: #E65100;
  font-size: 15px;
}

.pending-notice i {
  font-size: 20px;
}

.loading {
  text-align: center;
  padding: 60px;
  color: #8D7B68;
  font-size: 18px;
}

.form-container {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.form-group {
  margin-bottom: 24px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  margin-bottom: 12px;
}

.required {
  color: #DC143C;
}

.type-badge {
  display: inline-block;
  padding: 8px 16px;
  background: #F5EFE7;
  color: #8D7B68;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}

.title-input {
  width: 100%;
  padding: 16px;
  font-size: 18px;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #2C1810;
  transition: all 0.3s;
}

.title-input:focus {
  outline: none;
  border-color: #B87333;
}

.content-textarea {
  width: 100%;
  padding: 16px;
  font-size: 16px;
  line-height: 1.6;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #2C1810;
  resize: vertical;
  font-family: inherit;
  transition: all 0.3s;
}

.content-textarea:focus {
  outline: none;
  border-color: #B87333;
}

.code-textarea {
  width: 100%;
  padding: 16px;
  font-size: 13px;
  font-family: 'Consolas', 'Monaco', monospace;
  line-height: 1.5;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #2C1810;
  background: #FAFAFA;
  resize: vertical;
  transition: all 0.3s;
}

.code-textarea:focus {
  outline: none;
  border-color: #B87333;
  background: #fff;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  padding: 8px 16px;
  background: #F5EFE7;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  color: #4B3621;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.tag-item:hover {
  background: #E5D4C1;
}

.tag-item.selected {
  background: #B87333;
  border-color: #B87333;
  color: #fff;
}

.import-code-preview {
  padding: 16px;
  background: #F5F5F5;
  border-radius: 12px;
  overflow: hidden;
}

.import-code-preview code {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: #666;
  word-break: break-all;
}

.hint {
  font-size: 12px;
  color: #999;
  margin-top: 8px;
}

/* 画作图片管理样式 */
.artwork-images {
  width: 100%;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.image-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: 12px;
  overflow: hidden;
  background: #F5F0EB;
}

.image-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-item .remove-btn,
.image-item .undo-btn {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: rgba(0,0,0,0.6);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  opacity: 0;
  transition: opacity 0.2s;
}

.image-item:hover .remove-btn,
.image-item:hover .undo-btn {
  opacity: 1;
}

.image-item .remove-btn:hover {
  background: rgba(220,20,60,0.8);
}

.image-item .undo-btn {
  background: rgba(46,125,50,0.8);
}

.image-item .undo-btn:hover {
  background: rgba(46,125,50,1);
}

.image-item.marked-delete {
  opacity: 0.5;
}

.image-item.marked-delete img {
  filter: grayscale(1);
}

.delete-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: rgba(220,20,60,0.8);
  color: #fff;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.image-item.new-image {
  border: 2px solid #4CAF50;
}

.new-badge {
  position: absolute;
  bottom: 6px;
  left: 6px;
  background: #4CAF50;
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.add-more-btn {
  aspect-ratio: 1;
  border: 2px dashed #E0E0E0;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.3s;
  color: #999;
}

.add-more-btn:hover {
  border-color: #B87333;
  color: #B87333;
  background: #FFF8F0;
}

.add-more-btn i {
  font-size: 32px;
}

.add-more-btn span {
  font-size: 12px;
}

/* 水印开关样式 */
.watermark-toggle {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toggle-switch {
  position: relative;
  width: 48px;
  height: 26px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-switch .slider {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #E0E0E0;
  border-radius: 26px;
  transition: 0.3s;
}

.toggle-switch .slider:before {
  content: '';
  position: absolute;
  width: 20px;
  height: 20px;
  left: 3px;
  bottom: 3px;
  background: #fff;
  border-radius: 50%;
  transition: 0.3s;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
}

.toggle-switch input:checked + .slider {
  background: #B87333;
}

.toggle-switch input:checked + .slider:before {
  transform: translateX(22px);
}

.toggle-label {
  font-size: 14px;
  color: #5D4037;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
