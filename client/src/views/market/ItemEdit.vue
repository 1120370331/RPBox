<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getItem, updateItem, getItemTags, addItemTag, removeItemTag, type Item, type UpdateItemRequest } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const mounted = ref(false)
const loading = ref(false)
const item = ref<Item | null>(null)

const form = ref<UpdateItemRequest>({
  name: '',
  description: '',
  icon: '',
  status: 'published'
})

const itemTags = ref<Tag[]>([])
const selectedTags = ref<number[]>([])
const originalTags = ref<number[]>([])

// 是否有待审核的编辑
const hasPendingEdit = ref(false)

onMounted(async () => {
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
    item.value = res.item
    form.value.name = res.item.name
    form.value.description = res.item.description
    form.value.icon = res.item.icon
    form.value.status = res.item.status
    hasPendingEdit.value = !!res.pending_edit
  } catch (error) {
    console.error('加载道具失败:', error)
    toast.error('道具不存在')
    router.back()
  } finally {
    loading.value = false
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
    originalTags.value = (res.tags || []).map((t: any) => t.id)
    selectedTags.value = [...originalTags.value]
  } catch (error) {
    console.error('加载道具标签失败:', error)
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
  if (!form.value.name?.trim()) {
    toast.warning('请输入道具名称')
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

function getTypeText(type: string) {
  return type === 'item' ? '道具' : '剧本'
}
</script>

<template>
  <div class="item-edit-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <h1 class="page-title">编辑道具</h1>
      <div class="actions">
        <button class="cancel-btn" @click="handleCancel">取消</button>
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
      <!-- 道具类型（只读） -->
      <div class="form-group">
        <label>道具类型</label>
        <div class="type-badge">{{ getTypeText(item.type) }}</div>
      </div>

      <!-- 道具名称 -->
      <div class="form-group">
        <label>道具名称 <span class="required">*</span></label>
        <input
          v-model="form.name"
          type="text"
          placeholder="请输入道具名称"
          class="title-input"
        />
      </div>

      <!-- 描述 -->
      <div class="form-group">
        <label>描述</label>
        <textarea
          v-model="form.description"
          placeholder="请描述这个道具的功能和特点..."
          rows="4"
          class="content-textarea"
        ></textarea>
      </div>

      <!-- 标签选择 -->
      <div class="form-group" v-if="itemTags.length > 0">
        <label>道具分类标签</label>
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

      <!-- 导入代码（只读显示） -->
      <div class="form-group">
        <label>TRP3 导入代码</label>
        <div class="import-code-preview">
          <code>{{ item.import_code?.substring(0, 200) }}{{ item.import_code?.length > 200 ? '...' : '' }}</code>
        </div>
        <p class="hint">导入代码不可修改，如需更改请重新上传</p>
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

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
