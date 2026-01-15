<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { createItem } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()
const loading = ref(false)
const itemTags = ref<Tag[]>([])

// 表单数据
const form = ref({
  name: '',
  type: 'item' as 'item' | 'script',
  icon: '',
  description: '',
  import_code: '',
  raw_data: '',
  tag_ids: [] as number[],
  status: 'draft' as 'draft' | 'published'
})

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
async function handleSubmit() {
  if (!form.value.name || !form.value.import_code) {
    toast.warning('请填写道具名称和导入代码')
    return
  }

  loading.value = true
  try {
    const res: any = await createItem(form.value)
    if (res.code === 0) {
      toast.success('上传成功！')
      router.push('/market')
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

        <!-- 描述 -->
        <div class="form-group">
          <label>描述</label>
          <textarea
            v-model="form.description"
            placeholder="请描述这个道具的功能和特点..."
            rows="4"
          ></textarea>
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

        <!-- 发布状态 -->
        <div class="form-group">
          <label>发布状态</label>
          <select v-model="form.status">
            <option value="draft">草稿（仅自己可见）</option>
            <option value="published">发布（所有人可见）</option>
          </select>
        </div>

        <!-- 提交按钮 -->
        <div class="form-actions">
          <button type="button" class="cancel-btn" @click="goBack">取消</button>
          <button type="submit" class="submit-btn" :disabled="loading">
            {{ loading ? '上传中...' : '上传道具' }}
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
.submit-btn {
  flex: 1;
  height: 48px;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.cancel-btn {
  background: #F5F0EB;
  color: #795548;
}

.cancel-btn:hover {
  background: #E8DED3;
}

.submit-btn {
  background: #B87333;
  color: #fff;
  position: relative;
  z-index: 10;
}

.submit-btn:hover:not(:disabled) {
  background: #A66629;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-actions {
  position: relative;
  z-index: 5;
}
</style>
