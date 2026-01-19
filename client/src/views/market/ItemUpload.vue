<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { createItem, uploadImage, uploadItemImages } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { useToast } from '@/composables/useToast'
import { useUserStore } from '@/stores/user'
import TiptapEditor from '@/components/TiptapEditor.vue'

const router = useRouter()
const toast = useToast()
const userStore = useUserStore()
const loading = ref(false)
const uploadingImage = ref(false)
const uploadingArtwork = ref(false)
const itemTags = ref<Tag[]>([])
const previewImageInput = ref<HTMLInputElement | null>(null)
const artworkImagesInput = ref<HTMLInputElement | null>(null)

const DRAFT_KEY = 'item_upload_draft'

// 表单数据
const form = ref({
  name: '',
  type: 'item' as 'item' | 'campaign' | 'artwork',
  icon: '',
  preview_image: '',
  description: '',
  detail_content: '',
  import_code: '',
  raw_data: '',
  enable_watermark: true,  // 画作水印开关，默认开启
  tag_ids: [] as number[],
  status: 'draft' as 'draft' | 'published'
})

// 画作图片（待上传）
const artworkImages = ref<{ file: File; preview: string }[]>([])

// 是否为画作类型
const isArtwork = computed(() => form.value.type === 'artwork')

// 挂载时恢复草稿
onMounted(() => {
  // 检查登录状态
  if (!userStore.user || !userStore.token) {
    toast.info('请先登录后再上传作品')
    router.push('/login')
    return
  }

  loadTags()
  restoreDraft()
})

// 监听表单变化，自动保存草稿
watch(form, () => {
  saveDraft()
}, { deep: true })

// 保存草稿到 sessionStorage
function saveDraft() {
  sessionStorage.setItem(DRAFT_KEY, JSON.stringify(form.value))
}

// 恢复草稿
function restoreDraft() {
  const draftStr = sessionStorage.getItem(DRAFT_KEY)
  if (draftStr) {
    try {
      const draft = JSON.parse(draftStr)
      Object.assign(form.value, draft)
    } catch (e) {
      console.error('恢复草稿失败:', e)
    }
  }
}

// 清除草稿
function clearDraft() {
  sessionStorage.removeItem(DRAFT_KEY)
}

// 加载作品标签
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

// 提交表单
async function handleSubmit(status: 'draft' | 'published') {
  // 画作类型需要至少一张图片
  if (isArtwork.value) {
    if (!form.value.name) {
      toast.info('请填写作品名称')
      return
    }
    if (artworkImages.value.length === 0) {
      toast.info('请至少上传一张图片')
      return
    }
  } else {
    // 非画作类型需要导入代码
    if (!form.value.name || !form.value.import_code) {
      toast.info('请填写作品名称和导入代码')
      return
    }
  }

  form.value.status = status
  loading.value = true
  try {
    const res: any = await createItem(form.value)
    if (res.code === 0 || res.data) {
      const itemId = res.data.id

      // 如果是画作类型，上传图片
      if (isArtwork.value && artworkImages.value.length > 0) {
        try {
          const files = artworkImages.value.map(img => img.file)
          await uploadItemImages(itemId, files)
        } catch (uploadError: any) {
          console.error('图片上传失败:', uploadError)
          toast.info('作品已创建，但部分图片上传失败')
        }
      }

      clearDraft()
      const msg = status === 'draft' ? '草稿已保存！' : '发布成功，等待审核！'
      toast.success(msg)
      router.push('/market/my-items')
    } else {
      toast.error(res.error || res.message || '保存失败')
    }
  } catch (error: any) {
    console.error('上传失败:', error)
    toast.error(error.message || '上传失败，请重试')
  } finally {
    loading.value = false
  }
}

// 返回列表
function goBack() {
  router.push('/market')
}

// 预览
function handlePreview() {
  if (!form.value.name) {
    toast.info('请先填写作品名称')
    return
  }

  // 构建预览数据
  const previewData = {
    id: 0,
    name: form.value.name,
    type: form.value.type,
    icon: form.value.icon,
    preview_image: form.value.preview_image,
    description: form.value.description,
    detail_content: form.value.detail_content,
    import_code: form.value.import_code,
    downloads: 0,
    rating: 0,
    rating_count: 0,
    like_count: 0,
    favorite_count: 0,
    status: 'draft',
    created_at: new Date().toISOString(),
  }

  sessionStorage.setItem('item_preview_data', JSON.stringify(previewData))
  sessionStorage.setItem('item_preview_from', '/market/upload')

  router.push('/market/preview')
}

// 上传预览图
async function handlePreviewImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const file = input.files[0]
  if (file.size > 5 * 1024 * 1024) {
    toast.info('图片大小不能超过5MB')
    return
  }

  uploadingImage.value = true
  try {
    const res: any = await uploadImage(file)
    if (res.code === 0) {
      form.value.preview_image = res.data.url
      toast.success('预览图上传成功')
    }
  } catch (error: any) {
    toast.error(error.message || '上传失败')
  } finally {
    uploadingImage.value = false
  }
}

// 移除预览图
function removePreviewImage() {
  form.value.preview_image = ''
}

// 选择画作图片
function handleArtworkImagesSelect(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const maxImages = 20
  const remainingSlots = maxImages - artworkImages.value.length

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
      artworkImages.value.push({
        file,
        preview: e.target?.result as string
      })
    }
    reader.readAsDataURL(file)
  }

  // 清空 input 以允许重复选择相同文件
  input.value = ''
}

// 移除画作图片
function removeArtworkImage(index: number) {
  artworkImages.value.splice(index, 1)
}

loadTags()
</script>

<template>
  <div class="upload-page">
    <div class="upload-container">
      <!-- 返回按钮 -->
      <button class="back-btn" @click="goBack">
        <i class="ri-arrow-left-line"></i> 返回列表
      </button>

      <!-- 表单标题 -->
      <div class="form-header">
        <h1>上传作品</h1>
        <p>分享你的 TRP3 Extended 创意作品给其他玩家</p>
      </div>

      <!-- 上传表单 -->
      <form class="upload-form" @submit.prevent="handleSubmit">
        <!-- 作品名称 -->
        <div class="form-group">
          <label>作品名称 <span class="required">*</span></label>
          <input
            v-model="form.name"
            type="text"
            placeholder="请输入作品名称"
            required
          />
        </div>

        <!-- 作品类型 -->
        <div class="form-group">
          <label>作品类型 <span class="required">*</span></label>
          <select v-model="form.type" required>
            <option value="item">道具</option>
            <option value="campaign">剧本</option>
            <option value="artwork">画作</option>
          </select>
        </div>

        <!-- 预览图上传 -->
        <div class="form-group">
          <label>预览图</label>
          <div class="preview-upload">
            <div v-if="form.preview_image" class="preview-image">
              <img :src="form.preview_image" alt="预览图" />
              <button type="button" class="remove-btn" @click="removePreviewImage">
                <i class="ri-close-line"></i>
              </button>
            </div>
            <div v-else class="upload-area" @click="previewImageInput?.click()">
              <i class="ri-image-add-line"></i>
              <span>{{ uploadingImage ? '上传中...' : '点击上传预览图' }}</span>
            </div>
            <input
              ref="previewImageInput"
              type="file"
              accept="image/*"
              style="display: none"
              @change="handlePreviewImageUpload"
            />
          </div>
          <p class="hint">建议尺寸 400x300，最大 5MB</p>
        </div>

        <!-- 描述 -->
        <div class="form-group">
          <label>简短描述</label>
          <textarea
            v-model="form.description"
            placeholder="请简短描述这个作品的功能和特点..."
            rows="3"
          ></textarea>
        </div>

        <!-- 详细介绍（富文本） -->
        <div class="form-group">
          <label>详细介绍</label>
          <TiptapEditor
            v-model="form.detail_content"
            placeholder="可以添加图片、详细说明作品的使用方法..."
          />
          <p class="hint">支持插入图片，可以展示作品的详细效果</p>
        </div>

        <!-- 画作图片上传（仅画作类型） -->
        <div class="form-group" v-if="isArtwork">
          <label>画作图片 <span class="required">*</span></label>
          <div class="artwork-images">
            <!-- 已选择的图片列表 -->
            <div class="image-grid" v-if="artworkImages.length > 0">
              <div
                v-for="(img, index) in artworkImages"
                :key="index"
                class="image-item"
              >
                <img :src="img.preview" alt="画作图片" />
                <button type="button" class="remove-btn" @click="removeArtworkImage(index)">
                  <i class="ri-close-line"></i>
                </button>
                <span class="image-index">{{ index + 1 }}</span>
              </div>
              <!-- 添加更多按钮 -->
              <div
                v-if="artworkImages.length < 20"
                class="add-more-btn"
                @click="artworkImagesInput?.click()"
              >
                <i class="ri-add-line"></i>
                <span>添加更多</span>
              </div>
            </div>
            <!-- 空状态 -->
            <div v-else class="upload-area" @click="artworkImagesInput?.click()">
              <i class="ri-image-add-line"></i>
              <span>点击选择图片（最多20张）</span>
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

        <!-- 导入代码（非画作类型） -->
        <div class="form-group" v-if="!isArtwork">
          <label>TRP3 导入代码 <span class="required">*</span></label>
          <textarea
            v-model="form.import_code"
            placeholder="请粘贴从 TRP3 Extended 导出的代码..."
            rows="6"
            required
          ></textarea>
          <p class="hint">从 TRP3 Extended 中导出作品，然后将代码粘贴到这里</p>
        </div>

        <!-- 标签选择 -->
        <div class="form-group" v-if="itemTags.length > 0">
          <label>作品分类标签</label>
          <div class="tag-selector">
            <label
              v-for="tag in itemTags"
              :key="tag.id"
              class="tag-checkbox"
              :class="{ selected: form.tag_ids.includes(tag.id) }"
              :style="{ '--tag-color': '#' + tag.color }"
            >
              <input
                type="checkbox"
                :value="tag.id"
                v-model="form.tag_ids"
              />
              <span>{{ tag.name }}</span>
            </label>
          </div>
        </div>

        <!-- 提交按钮 -->
        <div class="form-actions">
          <button type="button" class="cancel-btn" @click="goBack">取消</button>
          <button type="button" class="preview-btn" @click="handlePreview" :disabled="loading">
            <i class="ri-eye-line"></i> 预览
          </button>
          <button type="button" class="draft-btn" @click="handleSubmit('draft')" :disabled="loading">
            保存草稿
          </button>
          <button type="button" class="publish-btn" @click="handleSubmit('published')" :disabled="loading">
            <i class="ri-upload-line"></i> 发布
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.upload-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: rgba(255,255,255,0.8);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #5D4037;
  margin-bottom: 24px;
}

.back-btn:hover {
  background: rgba(255,255,255,1);
}

.upload-container {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 8px 20px rgba(93,64,55,0.05);
}

.form-header {
  text-align: center;
  margin-bottom: 32px;
}

.form-header h1 {
  font-size: 28px;
  color: #3E2723;
  margin-bottom: 8px;
}

.form-header p {
  color: #999;
  font-size: 14px;
}

.upload-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  font-weight: 600;
  color: #5D4037;
}

.required {
  color: #DC143C;
}

.form-group input,
.form-group select,
.form-group textarea {
  padding: 12px 16px;
  border: 1px solid #E0E0E0;
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  transition: border-color 0.3s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #B87333;
}

.form-group textarea {
  resize: vertical;
  min-height: 100px;
}

.hint {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.preview-upload {
  width: 100%;
}

.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px;
  border: 2px dashed #E0E0E0;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
}

.upload-area:hover {
  border-color: #B87333;
  background: #FFF8F0;
}

.upload-area i {
  font-size: 48px;
  color: #B87333;
}

.upload-area span {
  color: #999;
  font-size: 14px;
}

.preview-image {
  position: relative;
  display: inline-block;
}

.preview-image img {
  max-width: 100%;
  max-height: 300px;
  border-radius: 12px;
  object-fit: cover;
}

.preview-image .remove-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(0,0,0,0.6);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.preview-image .remove-btn:hover {
  background: rgba(0,0,0,0.8);
}

.tag-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.tag-checkbox {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: #F5F0EB;
  border-radius: 20px;
  cursor: pointer;
  font-size: 13px;
  color: #795548;
  transition: all 0.3s;
  border: 2px solid transparent;
}

.tag-checkbox:hover {
  background: #E8DED3;
}

.tag-checkbox.selected {
  background: var(--tag-color, #B87333);
  color: #fff;
  border-color: var(--tag-color, #B87333);
}

.tag-checkbox input[type="checkbox"] {
  display: none;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.cancel-btn,
.preview-btn,
.draft-btn,
.publish-btn {
  flex: 1;
  height: 48px;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.cancel-btn {
  background: #F5F0EB;
  color: #795548;
  flex: 0.8;
}

.cancel-btn:hover {
  background: #E8DED3;
}

.preview-btn {
  background: #fff;
  color: #5D4037;
  border: 2px solid #E5D4C1;
  flex: 0.8;
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

.draft-btn:hover {
  background: #FFF8F0;
}

.publish-btn {
  background: #B87333;
  color: #fff;
}

.publish-btn:hover:not(:disabled) {
  background: #A66629;
}

.publish-btn:disabled,
.draft-btn:disabled,
.preview-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 画作图片上传样式 */
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

.image-item .remove-btn {
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

.image-item:hover .remove-btn {
  opacity: 1;
}

.image-item .remove-btn:hover {
  background: rgba(220,20,60,0.8);
}

.image-index {
  position: absolute;
  bottom: 6px;
  left: 6px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(0,0,0,0.5);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
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
</style>
