<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getPost, updatePost, getPostTags, type UpdatePostRequest, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { addPostTag, removePostTag } from '@/api/post'
import TiptapEditor from '@/components/TiptapEditor.vue'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()
const route = useRoute()
const mounted = ref(false)
const loading = ref(false)

const form = ref<UpdatePostRequest>({
  title: '',
  content: '',
  content_type: 'html',
  category: 'other',
  status: 'published',
  event_type: undefined,
  event_start_time: undefined,
  event_end_time: undefined,
})

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

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadPost()
  await loadTags()
  await loadGuilds()
  await loadPostTags()
})

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
    // 加载活动字段
    form.value.event_type = res.post.event_type
    if (res.post.event_start_time) {
      form.value.event_start_time = res.post.event_start_time.slice(0, 16)
    }
    if (res.post.event_end_time) {
      form.value.event_end_time = res.post.event_end_time.slice(0, 16)
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
</script>

<template>
  <div class="post-edit-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <h1 class="page-title">编辑帖子</h1>
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

    <div v-if="loading && !form.title" class="loading">加载中...</div>

    <div v-else class="form-container anim-item" style="--delay: 1">
      <div class="form-group">
        <label>标题</label>
        <input
          v-model="form.title"
          type="text"
          placeholder="输入帖子标题..."
          class="title-input"
        />
      </div>

      <div class="form-group">
        <label>分区</label>
        <div class="category-list">
          <div
            v-for="cat in POST_CATEGORIES"
            :key="cat.value"
            class="category-item"
            :class="{ selected: form.category === cat.value }"
            @click="form.category = cat.value"
          >
            <i :class="cat.icon"></i>
            <span>{{ cat.label }}</span>
          </div>
        </div>
      </div>

      <!-- 活动分区特殊字段 -->
      <div v-if="isEventCategory" class="form-group event-fields">
        <label>活动类型</label>
        <div class="event-type-list">
          <div
            class="event-type-item"
            :class="{ selected: form.event_type === 'server' }"
            @click="form.event_type = 'server'"
          >
            <i class="ri-global-line"></i>
            <span>服务器活动</span>
            <small>需要版主权限</small>
          </div>
          <div
            class="event-type-item"
            :class="{ selected: form.event_type === 'guild' }"
            @click="form.event_type = 'guild'"
          >
            <i class="ri-team-line"></i>
            <span>公会活动</span>
            <small>需要公会管理员权限</small>
          </div>
        </div>

        <div v-if="form.event_type" class="event-time-fields">
          <div class="time-field">
            <label>开始时间</label>
            <input
              v-model="form.event_start_time"
              type="datetime-local"
              class="time-input"
            />
          </div>
          <div class="time-field">
            <label>结束时间（可选）</label>
            <input
              v-model="form.event_end_time"
              type="datetime-local"
              class="time-input"
            />
          </div>
        </div>

        <div v-if="form.event_type === 'guild'" class="guild-required-notice">
          <i class="ri-information-line"></i>
          <span>公会活动需要选择关联公会</span>
        </div>
      </div>

      <div class="form-group">
        <label>内容</label>
        <TiptapEditor
          v-model="form.content"
          placeholder="分享你的故事..."
        />
      </div>

      <div class="form-group">
        <label>标签</label>
        <div class="tag-list">
          <div
            v-for="tag in tags"
            :key="tag.id"
            class="tag-item"
            :class="{ selected: selectedTags.includes(tag.id) }"
            @click="toggleTag(tag.id)"
          >
            {{ tag.name }}
          </div>
        </div>
      </div>

      <div class="form-group">
        <label>关联公会（可选）</label>
        <select v-model="form.guild_id" class="guild-select">
          <option :value="undefined">不关联公会</option>
          <option v-for="guild in guilds" :key="guild.id" :value="guild.id">
            {{ guild.name }}
          </option>
        </select>
      </div>
    </div>
  </div>
</template>

<style scoped>
.post-edit-page {
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
  color: #804030;
  border: 2px solid #804030;
}

.publish-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #804030;
  color: #fff;
}

.publish-btn:hover {
  background: #6B3528;
  transform: translateY(-2px);
}

.publish-btn:disabled,
.draft-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  border-color: #804030;
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
  border-color: #804030;
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
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.category-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.category-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  background: #F5EFE7;
  border: 2px solid #E5D4C1;
  border-radius: 10px;
  color: #4B3621;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.category-item:hover {
  background: #E5D4C1;
}

.category-item.selected {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.category-item i {
  font-size: 16px;
}

.guild-select {
  width: 100%;
  padding: 12px 16px;
  font-size: 16px;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #2C1810;
  cursor: pointer;
  transition: all 0.3s;
}

.guild-select:focus {
  outline: none;
  border-color: #804030;
}

/* 活动分区样式 */
.event-fields {
  background: #FFF9F0;
  border-radius: 12px;
  padding: 20px;
  border: 2px solid #E5D4C1;
}

.event-type-list {
  display: flex;
  gap: 16px;
}

.event-type-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
}

.event-type-item:hover {
  border-color: #804030;
}

.event-type-item.selected {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.event-type-item i {
  font-size: 28px;
}

.event-type-item span {
  font-size: 16px;
  font-weight: 600;
}

.event-type-item small {
  font-size: 12px;
  opacity: 0.7;
}

.event-time-fields {
  display: flex;
  gap: 16px;
  margin-top: 20px;
}

.time-field {
  flex: 1;
}

.time-field label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #4B3621;
  margin-bottom: 8px;
}

.time-input {
  width: 100%;
  padding: 12px 16px;
  font-size: 16px;
  border: 2px solid #E5D4C1;
  border-radius: 10px;
  color: #2C1810;
  transition: all 0.3s;
}

.time-input:focus {
  outline: none;
  border-color: #804030;
}

.guild-required-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 12px;
  background: #FFF3E0;
  border-radius: 8px;
  color: #E65100;
  font-size: 14px;
}

.guild-required-notice i {
  font-size: 18px;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
