<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { createItem, uploadImage } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { useToast } from '@/composables/useToast'
import TiptapEditor from '@/components/TiptapEditor.vue'

const router = useRouter()
const toast = useToast()
const loading = ref(false)
const uploadingImage = ref(false)
const itemTags = ref<Tag[]>([])
const previewImageInput = ref<HTMLInputElement | null>(null)

const DRAFT_KEY = 'item_upload_draft'

// 表单数据
const form = ref({
  name: '',
  type: 'item' as 'item' | 'script',
  icon: '',
  preview_image: '',
  description: '',
  detail_content: '',
  import_code: '',
  raw_data: '',
  tag_ids: [] as number[],
  status: 'draft' as 'draft' | 'published'
})

// 挂载时恢复草稿
onMounted(() => {
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

// 加载道具标签
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
  if (!form.value.name || !form.value.import_code) {
    toast.warning('请填写道具名称和导入代码')
    return
  }

  form.value.status = status
  loading.value = true
  try {
    const res: any = await createItem(form.value)
    if (res.code === 0 || res.data) {
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
    toast.warning('请先填写道具名称')
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
    toast.warning('图片大小不能超过5MB')
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
        <h1>上传道具</h1>
        <p>分享你的 TRP3 Extended 道具给其他玩家</p>
      </div>

      <!-- 上传表单 -->
      <form class="upload-form" @submit.prevent="handleSubmit">
        <!-- 道具名称 -->
        <div class="form-group">
          <label>道具名称 <span class="required">*</span></label>
          <input
            v-model="form.name"
            type="text"
            placeholder="请输入道具名称"
            required
          />
        </div>

        <!-- 道具类型 -->
        <div class="form-group">
          <label>道具类型 <span class="required">*</span></label>
          <select v-model="form.type" required>
            <option value="item">道具</option>
            <option value="script">剧本</option>
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
            placeholder="请简短描述这个道具的功能和特点..."
            rows="3"
          ></textarea>
        </div>

        <!-- 详细介绍（富文本） -->
        <div class="form-group">
          <label>详细介绍</label>
          <TiptapEditor
            v-model="form.detail_content"
            placeholder="可以添加图片、详细说明道具的使用方法..."
          />
          <p class="hint">支持插入图片，可以展示道具的详细效果</p>
        </div>

        <!-- 导入代码 -->
        <div class="form-group">
          <label>TRP3 导入代码 <span class="required">*</span></label>
          <textarea
            v-model="form.import_code"
            placeholder="请粘贴从 TRP3 Extended 导出的代码..."
            rows="6"
            required
          ></textarea>
          <p class="hint">从 TRP3 Extended 中导出道具，然后将代码粘贴到这里</p>
        </div>

        <!-- 标签选择 -->
        <div class="form-group" v-if="itemTags.length > 0">
          <label>道具分类标签</label>
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
</style>
