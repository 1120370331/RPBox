<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listPosts, deletePost, type PostWithAuthor } from '@/api/post'

const router = useRouter()
const mounted = ref(false)
const loading = ref(false)
const posts = ref<PostWithAuthor[]>([])
const currentUserId = ref<number>(0)
const filterStatus = ref<'published' | 'pending' | 'draft'>('published')

// 获取当前用户ID
const userStr = localStorage.getItem('user')
if (userStr) {
  try {
    const user = JSON.parse(userStr)
    currentUserId.value = user.id
  } catch (e) {
    console.error('解析用户信息失败:', e)
  }
}

// 过滤后的帖子列表
const filteredPosts = computed(() => {
  if (filterStatus.value === 'pending') {
    // 审核中：status为pending或review_status为pending
    return posts.value.filter(p => p.status === 'pending' || p.review_status === 'pending')
  }
  if (filterStatus.value === 'draft') {
    // 草稿
    return posts.value.filter(p => p.status === 'draft')
  }
  // 我发布的：已发布且审核通过
  return posts.value.filter(p => p.status === 'published' && p.review_status === 'approved')
})

// 统计数据
const stats = computed(() => {
  const published = posts.value.filter(p => p.status === 'published' && p.review_status === 'approved').length
  const pending = posts.value.filter(p => p.status === 'pending' || p.review_status === 'pending').length
  const draft = posts.value.filter(p => p.status === 'draft').length
  return { published, pending, draft }
})

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadMyPosts()
})

async function loadMyPosts() {
  loading.value = true
  try {
    // 传 status: 'all' 获取所有状态的帖子（包括草稿）
    const res = await listPosts({ author_id: currentUserId.value, status: 'all' })
    posts.value = res.posts || []
  } catch (error) {
    console.error('加载我的帖子失败:', error)
  } finally {
    loading.value = false
  }
}

function goToDetail(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function goToEdit(id: number) {
  router.push({ name: 'post-edit', params: { id } })
}

async function handleDelete(post: PostWithAuthor) {
  if (!confirm(`确定要删除帖子"${post.title}"吗？`)) {
    return
  }

  try {
    await deletePost(post.id)
    alert('删除成功')
    await loadMyPosts()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败，请重试')
  }
}

function goToCreate() {
  router.push({ name: 'post-create' })
}

function goBack() {
  router.push({ name: 'community' })
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 去除HTML标签，只保留纯文本
function stripHtml(html: string) {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}
</script>

<template>
  <div class="my-posts-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          返回
        </button>
        <h1 class="page-title">我的帖子</h1>
      </div>
      <button class="create-btn" @click="goToCreate">
        <i class="ri-add-line"></i>
        创建帖子
      </button>
    </div>

    <div class="stats anim-item" style="--delay: 1">
      <div class="stat-item">
        <div class="stat-value">{{ stats.published }}</div>
        <div class="stat-label">已发布</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.pending }}</div>
        <div class="stat-label">审核中</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.draft }}</div>
        <div class="stat-label">草稿</div>
      </div>
    </div>

    <div class="filters anim-item" style="--delay: 2">
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'published' }"
        @click="filterStatus = 'published'"
      >
        我发布的
      </button>
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'pending' }"
        @click="filterStatus = 'pending'"
      >
        审核中
      </button>
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'draft' }"
        @click="filterStatus = 'draft'"
      >
        草稿箱
      </button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="filteredPosts.length === 0" class="empty anim-item" style="--delay: 3">
      <i class="ri-file-list-3-line"></i>
      <p>{{ filterStatus === 'draft' ? '没有草稿' : filterStatus === 'pending' ? '没有审核中的帖子' : '没有已发布的帖子' }}</p>
      <button class="create-btn-large" @click="goToCreate">
        <i class="ri-add-line"></i>
        创建第一篇帖子
      </button>
    </div>

    <div v-else class="posts-list">
      <div
        v-for="(post, index) in filteredPosts"
        :key="post.id"
        class="post-card anim-item"
        :style="`--delay: ${index + 3}`"
      >
        <div class="post-header">
          <h2 class="post-title" @click="goToDetail(post.id)">{{ post.title }}</h2>
          <span v-if="post.status === 'draft'" class="draft-badge">草稿</span>
          <span v-else-if="post.status === 'pending' || post.review_status === 'pending'" class="pending-badge">审核中</span>
        </div>

        <div class="post-content">{{ stripHtml(post.content).substring(0, 150) }}{{ stripHtml(post.content).length > 150 ? '...' : '' }}</div>

        <div class="post-footer">
          <div class="post-meta">
            <span class="meta-item">
              <i class="ri-eye-line"></i>
              {{ post.view_count }}
            </span>
            <span class="meta-item">
              <i class="ri-heart-line"></i>
              {{ post.like_count }}
            </span>
            <span class="meta-item">
              <i class="ri-chat-3-line"></i>
              {{ post.comment_count }}
            </span>
            <span class="meta-item">
              <i class="ri-time-line"></i>
              {{ formatDate(post.updated_at) }}
            </span>
          </div>

          <div class="post-actions">
            <button class="action-btn edit" @click="goToEdit(post.id)">
              <i class="ri-edit-line"></i>
              编辑
            </button>
            <button class="action-btn delete" @click="handleDelete(post)">
              <i class="ri-delete-bin-line"></i>
              删除
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.my-posts-page {
  max-width: 1000px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #4B3621;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.back-btn:hover {
  background: #F5EFE7;
}

.page-title {
  font-size: 42px;
  color: #4B3621;
  margin: 0;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.create-btn:hover {
  background: #6B3528;
  transform: translateY(-2px);
}

.stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.stat-item {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.stat-value {
  font-size: 36px;
  font-weight: 700;
  color: #804030;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #8D7B68;
}

.filters {
  display: flex;
  gap: 12px;
}

.filter-btn {
  padding: 10px 20px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #4B3621;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.filter-btn:hover {
  background: #F5EFE7;
}

.filter-btn.active {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.loading {
  text-align: center;
  padding: 60px;
  color: #8D7B68;
  font-size: 18px;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.empty i {
  font-size: 64px;
  color: #8D7B68;
  opacity: 0.3;
  margin-bottom: 16px;
}

.empty p {
  font-size: 16px;
  color: #8D7B68;
  margin-bottom: 24px;
}

.create-btn-large {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 28px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.create-btn-large:hover {
  background: #6B3528;
  transform: translateY(-2px);
}

.posts-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.post-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
  transition: all 0.3s;
}

.post-card:hover {
  box-shadow: 0 6px 16px rgba(75,54,33,0.1);
  transform: translateY(-2px);
}

.post-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.post-title {
  font-size: 22px;
  font-weight: 700;
  color: #2C1810;
  margin: 0;
  cursor: pointer;
  transition: color 0.3s;
  flex: 1;
}

.post-title:hover {
  color: #804030;
}

.pending-badge {
  padding: 4px 12px;
  background: #E6A23C;
  color: #fff;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.draft-badge {
  padding: 4px 12px;
  background: #909399;
  color: #fff;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.post-content {
  font-size: 15px;
  line-height: 1.6;
  color: #4B3621;
  margin-bottom: 16px;
}

.post-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 2px solid #F5EFE7;
}

.post-meta {
  display: flex;
  gap: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: #8D7B68;
}

.meta-item i {
  font-size: 16px;
}

.post-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.action-btn.edit {
  background: #fff;
  color: #4B3621;
}

.action-btn.edit:hover {
  background: #F5EFE7;
}

.action-btn.delete {
  background: #fff;
  border-color: #C44536;
  color: #C44536;
}

.action-btn.delete:hover {
  background: #C44536;
  color: #fff;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
