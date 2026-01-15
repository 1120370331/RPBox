<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getPost, likePost, unlikePost, favoritePost, unfavoritePost, deletePost, POST_CATEGORIES } from '@/api/post'
import { listComments, createComment, type CommentWithAuthor } from '@/api/post'

const router = useRouter()
const route = useRoute()
const mounted = ref(false)
const loading = ref(false)
const submittingComment = ref(false)
const actionLoading = ref(false)

const post = ref<any>(null)
const comments = ref<CommentWithAuthor[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentContent = ref('')
const currentUserId = ref<number>(0)

const errorMessage = ref('')
const commentError = ref('')

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

const isAuthor = computed(() => {
  return post.value && currentUserId.value === post.value.author_id
})

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadPost()
  await loadComments()
})

async function loadPost() {
  loading.value = true
  errorMessage.value = ''
  try {
    const id = Number(route.params.id)
    if (isNaN(id)) throw new Error('无效的帖子ID')
    const res = await getPost(id)
    post.value = res.post
    post.value.author_name = res.author_name  // author_name 在响应顶层
    liked.value = res.liked
    favorited.value = res.favorited
  } catch (error: any) {
    console.error('加载帖子失败:', error)
    errorMessage.value = error.response?.data?.error || error.message || '加载帖子失败'
    setTimeout(() => router.back(), 2000)
  } finally {
    loading.value = false
  }
}

async function loadComments() {
  try {
    const id = Number(route.params.id)
    const res = await listComments(id)
    comments.value = res.comments || []
  } catch (error: any) {
    console.error('加载评论失败:', error)
  }
}

async function handleLike() {
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    if (liked.value) {
      await unlikePost(post.value.id)
      liked.value = false
      post.value.like_count--
    } else {
      await likePost(post.value.id)
      liked.value = true
      post.value.like_count++
    }
  } catch (error: any) {
    console.error('点赞失败:', error)
  } finally {
    actionLoading.value = false
  }
}

async function handleFavorite() {
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    if (favorited.value) {
      await unfavoritePost(post.value.id)
      favorited.value = false
      post.value.favorite_count--
    } else {
      await favoritePost(post.value.id)
      favorited.value = true
      post.value.favorite_count++
    }
  } catch (error: any) {
    console.error('收藏失败:', error)
  } finally {
    actionLoading.value = false
  }
}

async function handleComment() {
  if (!commentContent.value.trim()) return
  if (submittingComment.value) return
  submittingComment.value = true
  commentError.value = ''
  try {
    await createComment(post.value.id, commentContent.value)
    commentContent.value = ''
    await loadComments()
    post.value.comment_count++
  } catch (error: any) {
    commentError.value = error.response?.data?.error || '评论失败'
  } finally {
    submittingComment.value = false
  }
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

function formatCommentTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  if (hours < 1) return '刚刚'
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : '其他'
}

function goBack() {
  router.back()
}

function goToEdit() {
  router.push({ name: 'post-edit', params: { id: post.value.id } })
}

async function handleDelete() {
  if (!confirm('确定要删除这篇帖子吗？')) return
  try {
    await deletePost(post.value.id)
    router.push({ name: 'community' })
  } catch (error) {
    console.error('删除失败:', error)
  }
}
</script>

<template>
  <div class="post-detail-page" :class="{ 'animate-in': mounted }">
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="errorMessage" class="error-message">
      <i class="ri-error-warning-line"></i>
      <p>{{ errorMessage }}</p>
    </div>

    <div v-else-if="post" class="content-layout">
      <!-- 主内容区 -->
      <main class="main-area">
        <!-- 返回按钮 -->
        <div class="nav-bar anim-item" style="--delay: 0">
          <button class="back-btn" @click="goBack">
            <div class="back-icon">
              <i class="ri-arrow-left-s-line"></i>
            </div>
            <span>返回</span>
          </button>
        </div>

        <!-- 文章 -->
        <article class="article-card anim-item" style="--delay: 1">
          <div class="article-decoration"></div>

          <!-- 文章头部：作者 + 操作 -->
          <div class="article-top">
            <div class="author-section">
              <div class="author-avatar">
                {{ post.author_name?.charAt(0) || 'U' }}
              </div>
              <div class="author-info">
                <h4 class="author-name">{{ post.author_name }}</h4>
                <span class="post-date">{{ formatDate(post.created_at) }}</span>
              </div>
            </div>
            <div class="action-buttons">
              <button class="action-btn" :class="{ active: liked }" @click="handleLike" :disabled="actionLoading">
                <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                <span>{{ post.like_count }}</span>
              </button>
              <button class="action-btn" :class="{ active: favorited }" @click="handleFavorite" :disabled="actionLoading">
                <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'"></i>
                <span>{{ post.favorite_count }}</span>
              </button>
              <span class="view-count">
                <i class="ri-eye-line"></i>
                {{ post.view_count }}
              </span>
            </div>
          </div>

          <!-- 文章内容 -->
          <div class="article-body">
            <header class="article-header">
              <div class="category-badge">
                <span class="badge-dot"></span>
                <span>{{ getCategoryLabel(post.category) }}</span>
              </div>
              <h1 class="article-title">{{ post.title }}</h1>
            </header>

            <div class="zen-divider"></div>

            <div class="article-content" v-html="post.content"></div>
          </div>

          <!-- 作者操作 -->
          <div v-if="isAuthor" class="owner-actions">
            <button class="owner-btn" @click="goToEdit">
              <i class="ri-edit-line"></i> 编辑
            </button>
            <button class="owner-btn delete" @click="handleDelete">
              <i class="ri-delete-bin-line"></i> 删除
            </button>
          </div>
        </article>

        <!-- 评论区 -->
        <section class="comments-section anim-item" style="--delay: 2">
          <h3 class="comments-title">
            讨论 <span class="comment-badge">{{ post.comment_count }}</span>
          </h3>

          <!-- 评论输入 -->
          <div class="comment-input-box">
            <textarea
              v-model="commentContent"
              placeholder="分享你的想法..."
              :disabled="submittingComment"
            ></textarea>
            <div class="input-footer">
              <div></div>
              <button class="post-btn" :disabled="submittingComment" @click="handleComment">
                发表
              </button>
            </div>
          </div>

          <!-- 评论列表 -->
          <div class="comments-list">
            <div v-for="comment in comments" :key="comment.id" class="comment-item">
              <div class="comment-avatar">
                {{ comment.author_name.charAt(0) }}
              </div>
              <div class="comment-body">
                <div class="comment-meta">
                  <span class="comment-author">{{ comment.author_name }}</span>
                  <span class="comment-time">{{ formatCommentTime(comment.created_at) }}</span>
                </div>
                <p class="comment-text">{{ comment.content }}</p>
              </div>
            </div>

            <div v-if="comments.length === 0" class="empty-comments">
              暂无评论，快来发表第一条评论吧
            </div>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<style scoped>
.post-detail-page {
  max-width: 1200px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  padding: 80px;
  color: #8D7B68;
  font-size: 16px;
}

.error-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px;
  color: #C53030;
}

.error-message i {
  font-size: 48px;
  margin-bottom: 16px;
}

/* ========== 单栏布局 ========== */
.content-layout {
  max-width: 1000px;
  margin: 0 auto;
}

/* ========== 导航栏 ========== */
.nav-bar {
  margin-bottom: 8px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  background: none;
  border: none;
  color: #8D7B68;
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: color 0.3s;
}

.back-btn:hover {
  color: #804030;
}

.back-icon {
  width: 40px;
  height: 40px;
  border: 1px solid #E5D4C1;
  border-radius: 50%;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  transition: all 0.3s;
}

.back-btn:hover .back-icon {
  border-color: #804030;
  background: rgba(128, 64, 48, 0.05);
}

.back-icon i {
  font-size: 18px;
}

/* ========== 主内容区 ========== */
.main-area {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

/* ========== 文章卡片 ========== */
.article-card {
  background: #fff;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  position: relative;
}

.article-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, transparent, #804030, transparent);
  opacity: 0.3;
}

/* ========== 文章头部 ========== */
.article-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 32px;
  border-bottom: 1px solid #F5EFE7;
}

.author-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.author-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #804030);
  border: 2px solid #fff;
  box-shadow: 0 2px 8px rgba(128, 64, 48, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.author-name {
  font-family: 'Merriweather', serif;
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  margin: 0;
}

.post-date {
  font-size: 12px;
  color: #8D7B68;
}

/* ========== 操作按钮 ========== */
.action-buttons {
  display: flex;
  align-items: center;
  gap: 16px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 20px;
  color: #8D7B68;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.action-btn:hover {
  border-color: #B87333;
  color: #B87333;
}

.action-btn.active {
  background: rgba(128, 64, 48, 0.1);
  border-color: #804030;
  color: #804030;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn i {
  font-size: 16px;
}

.view-count {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #8D7B68;
  font-size: 14px;
}

.view-count i {
  font-size: 16px;
}

/* ========== 文章内容区 ========== */
.article-body {
  padding: 32px 48px 48px;
}

.article-header {
  text-align: center;
  margin-bottom: 32px;
}

.category-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background: #F5EFE7;
  margin-bottom: 20px;
}

.badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #B87333;
}

.category-badge span:last-child {
  font-size: 11px;
  font-weight: 600;
  color: #2C1810;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.article-title {
  font-family: 'Merriweather', serif;
  font-size: 32px;
  font-weight: 700;
  color: #2C1810;
  line-height: 1.4;
  margin: 0 0 20px 0;
}

.article-meta {
  font-family: 'Merriweather', serif;
  font-style: italic;
  font-size: 14px;
  color: #8D7B68;
}

.zen-divider {
  height: 1px;
  background: linear-gradient(90deg, transparent, #E5D4C1, transparent);
  margin: 32px 0;
}

/* 正文内容 */
.article-content {
  font-family: 'Merriweather', serif;
  font-size: 16px;
  line-height: 1.9;
  color: #4B3621;
}

.article-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 1.5em 0;
}

.article-content :deep(p) {
  margin-bottom: 1.5em;
}

.article-content :deep(h2),
.article-content :deep(h3) {
  color: #2C1810;
  font-weight: 700;
  margin-top: 2em;
  margin-bottom: 1em;
  padding-left: 16px;
  border-left: 3px solid #B87333;
}

.article-content :deep(blockquote) {
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  padding: 24px;
  margin: 2em 0;
  font-style: italic;
  text-align: center;
  color: #2C1810;
  font-size: 18px;
}

.article-content :deep(strong) {
  color: #804030;
}

/* ========== 作者操作 ========== */
.owner-actions {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 20px 32px;
  border-top: 1px solid #F5EFE7;
}

.owner-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #F5EFE7;
  border: 1px solid #E5D4C1;
  border-radius: 6px;
  color: #8D7B68;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.3s;
}

.owner-btn:hover {
  border-color: #B87333;
  color: #B87333;
}

.owner-btn.delete {
  color: #C44536;
  border-color: rgba(196, 69, 54, 0.3);
  background: rgba(196, 69, 54, 0.05);
}

.owner-btn.delete:hover {
  border-color: #C44536;
  background: rgba(196, 69, 54, 0.1);
}

/* ========== 评论区 ========== */
.comments-section {
  background: #fff;
  padding: 32px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
}

.comments-title {
  font-family: 'Merriweather', serif;
  font-size: 20px;
  font-weight: 500;
  color: #2C1810;
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 20px 0;
}

.comment-badge {
  font-family: 'Inter', sans-serif;
  font-size: 13px;
  font-weight: 400;
  color: #8D7B68;
  background: #F5EFE7;
  padding: 4px 12px;
  border-radius: 20px;
}

/* 评论输入框 */
.comment-input-box {
  background: #fff;
  border: 1px solid #E5D4C1;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(75, 54, 33, 0.04);
  transition: box-shadow 0.3s;
}

.comment-input-box:focus-within {
  box-shadow: 0 0 0 3px rgba(128, 64, 48, 0.1);
}

.comment-input-box textarea {
  width: 100%;
  background: transparent;
  border: none;
  outline: none;
  resize: none;
  font-size: 14px;
  line-height: 1.6;
  color: #4B3621;
  font-family: inherit;
  min-height: 80px;
}

.comment-input-box textarea::placeholder {
  color: rgba(141, 123, 104, 0.6);
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid #F5EFE7;
  margin-top: 12px;
}

.post-btn {
  background: #2C1810;
  color: #fff;
  border: none;
  padding: 8px 20px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: background 0.3s;
}

.post-btn:hover {
  background: #804030;
}

.post-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 评论列表 */
.comments-list {
  display: flex;
  flex-direction: column;
  gap: 24px;
  margin-top: 24px;
}

.comment-item {
  display: flex;
  gap: 12px;
}

.comment-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #804030);
  border: 1px solid #E5D4C1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
  filter: grayscale(100%);
  transition: filter 0.5s;
}

.comment-item:hover .comment-avatar {
  filter: grayscale(0%);
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-meta {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 6px;
}

.comment-author {
  font-size: 14px;
  font-weight: 500;
  color: #2C1810;
}

.comment-time {
  font-size: 12px;
  color: #8D7B68;
}

.comment-text {
  font-size: 14px;
  line-height: 1.6;
  color: #4B3621;
  margin: 0;
}

.empty-comments {
  text-align: center;
  padding: 40px 16px;
  color: #8D7B68;
  font-size: 14px;
}

/* ========== 动画 ========== */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}

@keyframes fadeUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
