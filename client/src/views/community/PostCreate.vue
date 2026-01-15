<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { createPost, type CreatePostRequest, POST_CATEGORIES, type PostCategory } from '@/api/post'
import { listTags, type Tag } from '@/api/tag'
import { listGuilds, type Guild } from '@/api/guild'
import { useToastStore } from '@/stores/toast'
import TiptapEditor from '@/components/TiptapEditor.vue'

const router = useRouter()
const toast = useToastStore()
const mounted = ref(false)
const loading = ref(false)

const form = ref<CreatePostRequest>({
  title: '',
  content: '',
  content_type: 'html',
  category: 'other',
  tag_ids: [],
  status: 'published',
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

  loading.value = true
  try {
    form.value.status = status
    form.value.tag_ids = selectedTags.value
    await createPost(form.value)
    toast.success(status === 'published' ? '发布成功' : '保存草稿成功')
    router.push({ name: 'community' })
  } catch (error) {
    console.error('提交失败:', error)
    toast.error('提交失败，请重试')
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

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
