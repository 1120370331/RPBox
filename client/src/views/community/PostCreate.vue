<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { createPost, type CreatePostRequest, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import TiptapEditor from '@/components/TiptapEditor.vue'

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
  event_type: undefined,
  event_start_time: undefined,
  event_end_time: undefined,
})

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

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadTags()
  await loadGuilds()
})

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
    await createPost(form.value)
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
</script>

<template>
  <div class="post-create-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <h1 class="page-title">发布帖子</h1>
      <div class="actions">
        <button class="cancel-btn" @click="handleCancel">取消</button>
        <button class="draft-btn" @click="handleSubmit('draft')" :disabled="loading">
          保存草稿
        </button>
        <button class="publish-btn" @click="handleSubmit('published')" :disabled="loading">
          <i class="ri-send-plane-line"></i>
          发布
        </button>
      </div>
    </div>

    <div class="form-container anim-item" style="--delay: 1">
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
            :class="{ selected: form.event_type === 'server', disabled: !canPostServerEvent }"
            @click="canPostServerEvent && (form.event_type = 'server')"
          >
            <i class="ri-global-line"></i>
            <span>服务器活动</span>
            <small v-if="canPostServerEvent">需要版主权限</small>
            <small v-else class="no-permission">你没有发布的权限</small>
          </div>
          <div
            class="event-type-item"
            :class="{ selected: form.event_type === 'guild', disabled: !canPostGuildEvent }"
            @click="canPostGuildEvent && (form.event_type = 'guild')"
          >
            <i class="ri-team-line"></i>
            <span>公会活动</span>
            <small v-if="canPostGuildEvent">需要公会管理员权限</small>
            <small v-else class="no-permission">你没有发布的权限</small>
          </div>
        </div>

        <!-- 公会活动：选择公会 -->
        <div v-if="form.event_type === 'guild'" class="guild-select-section">
          <label>选择公会</label>
          <select v-model="form.guild_id" class="guild-select" required>
            <option :value="undefined" disabled>请选择要发布活动的公会</option>
            <option v-for="guild in adminGuilds" :key="guild.id" :value="guild.id">
              {{ guild.name }}（{{ guild.my_role === 'owner' ? '会长' : '管理员' }}）
            </option>
          </select>
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
.post-create-page {
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

.event-type-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: #F5F5F5;
  border-color: #E0E0E0;
}

.event-type-item.disabled:hover {
  border-color: #E0E0E0;
}

.event-type-item.disabled i {
  color: #999;
}

.event-type-item small.no-permission {
  color: #C44536;
  opacity: 1;
}

.guild-select-section {
  margin-top: 20px;
}

.guild-select-section label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #4B3621;
  margin-bottom: 8px;
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
