<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getPost, likePost, unlikePost, favoritePost, unfavoritePost, deletePost } from '@/api/post'
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

// 获取当前用户ID（从localStorage）
const userStr = localStorage.getItem('user')
if (userStr) {
  try {
    const user = JSON.parse(userStr)
    currentUserId.value = user.id
  } catch (e) {
    console.error('解析用户信息失败:', e)
  }
}

// 判断是否是作者
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
    if (isNaN(id)) {
      throw new Error('无效的帖子ID')
    }
    const res = await getPost(id)
    post.value = res.post
    liked.value = res.liked
    favorited.value = res.favorited
  } catch (error: any) {
    console.error('加载帖子失败:', error)
    errorMessage.value = error.response?.data?.error || error.message || '加载帖子失败，请稍后重试'
    setTimeout(() => router.back(), 2000)
  } finally {
    loading.value = false
  }
}

async function loadComments() {
  commentError.value = ''
  try {
    const id = Number(route.params.id)
    const res = await listComments(id)
    comments.value = res.comments || []
  } catch (error: any) {
    console.error('加载评论失败:', error)
    commentError.value = '加载评论失败，请刷新页面重试'
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
    const message = error.response?.data?.error || '操作失败，请重试'
    alert(message)
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
    const message = error.response?.data?.error || '操作失败，请重试'
    alert(message)
  } finally {
    actionLoading.value = false
  }
}

async function handleComment() {
  if (!commentContent.value.trim()) {
    commentError.value = '请输入评论内容'
    return
  }

  if (submittingComment.value) return
  submittingComment.value = true
  commentError.value = ''

  try {
    await createComment(post.value.id, commentContent.value)
    commentContent.value = ''
    await loadComments()
    post.value.comment_count++
  } catch (error: any) {
    console.error('评论失败:', error)
    commentError.value = error.response?.data?.error || '评论失败，请重试'
  } finally {
    submittingComment.value = false
  }
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function goToEdit() {
  router.push({ name: 'post-edit', params: { id: post.value.id } })
}

async function handleDelete() {
  if (!confirm('确定要删除这篇帖子吗？')) {
    return
  }

  try {
    await deletePost(post.value.id)
    alert('删除成功')
    router.push({ name: 'community' })
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败，请重试')
  }
}

function goBack() {
  router.back()
}
</script>

<template>
  <div class="post-detail-page" :class="{ 'animate-in': mounted }">
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="errorMessage" class="error-message anim-item" style="--delay: 0">
      <i class="ri-error-warning-line"></i>
      <p>{{ errorMessage }}</p>
    </div>

    <div v-else-if="post" class="content-wrapper">
      <div class="header anim-item" style="--delay: 0">
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          返回
        </button>
        <div v-if="isAuthor" class="author-actions">
          <button class="edit-btn" @click="goToEdit">
            <i class="ri-edit-line"></i>
            编辑
          </button>
          <button class="delete-btn" @click="handleDelete">
            <i class="ri-delete-bin-line"></i>
            删除
          </button>
        </div>
      </div>

      <div class="post-container anim-item" style="--delay: 1">
        <h1 class="post-title">{{ post.title }}</h1>

        <div class="post-meta">
          <div class="author-info">
            <div class="author-avatar">{{ post.author_name?.charAt(0) || 'U' }}</div>
            <div>
              <div class="author-name">{{ post.author_name }}</div>
              <div class="post-time">{{ formatDate(post.created_at) }}</div>
            </div>
          </div>

          <div class="post-actions">
            <button
              class="action-btn"
              :class="{ active: liked }"
              :disabled="actionLoading"
              @click="handleLike"
            >
              <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'"></i>
              {{ post.like_count }}
            </button>
            <button
              class="action-btn"
              :class="{ active: favorited }"
              :disabled="actionLoading"
              @click="handleFavorite"
            >
              <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'"></i>
              {{ post.favorite_count }}
            </button>
            <span class="stat-item">
              <i class="ri-eye-line"></i>
              {{ post.view_count }}
            </span>
          </div>
        </div>

        <div class="post-content" v-html="post.content"></div>
      </div>

      <div class="comments-section anim-item" style="--delay: 2">
        <h2 class="section-title">评论 ({{ post.comment_count }})</h2>

        <div class="comment-input">
          <textarea
            v-model="commentContent"
            placeholder="写下你的评论..."
            rows="3"
            :disabled="submittingComment"
          ></textarea>
          <div v-if="commentError" class="comment-error">
            <i class="ri-error-warning-line"></i>
            {{ commentError }}
          </div>
          <button
            class="submit-btn"
            :disabled="submittingComment"
            @click="handleComment"
          >
            <span v-if="submittingComment">提交中...</span>
            <span v-else>发表评论</span>
          </button>
        </div>

        <div class="comment-list">
          <div v-for="comment in comments" :key="comment.id" class="comment-item">
            <div class="comment-avatar">{{ comment.author_name.charAt(0) }}</div>
            <div class="comment-body">
              <div class="comment-header">
                <span class="comment-author">{{ comment.author_name }}</span>
                <span class="comment-time">{{ formatDate(comment.created_at) }}</span>
              </div>
              <div class="comment-content">{{ comment.content }}</div>
            </div>
          </div>

          <div v-if="comments.length === 0" class="empty-comments">
            <i class="ri-chat-3-line"></i>
            <p>暂无评论，快来发表第一条评论吧</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.post-detail-page {
  max-width: 900px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  padding: 60px;
  color: #8D7B68;
  font-size: 18px;
}

.error-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  background: #FFF5F5;
  border: 2px solid #FEB2B2;
  border-radius: 16px;
  color: #C53030;
}

.error-message i {
  font-size: 48px;
  margin-bottom: 12px;
}

.error-message p {
  font-size: 16px;
  margin: 0;
}

.content-wrapper {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header {
  display: flex;
  align-items: center;
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

.author-actions {
  display: flex;
  gap: 12px;
  margin-left: auto;
}

.edit-btn,
.delete-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.edit-btn {
  background: #fff;
  color: #4B3621;
}

.edit-btn:hover {
  background: #F5EFE7;
}

.delete-btn {
  background: #fff;
  border-color: #C44536;
  color: #C44536;
}

.delete-btn:hover {
  background: #C44536;
  color: #fff;
}

.post-container {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.post-title {
  font-size: 32px;
  font-weight: 700;
  color: #2C1810;
  margin-bottom: 20px;
  line-height: 1.3;
}

.post-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 20px;
  border-bottom: 2px solid #F5EFE7;
  margin-bottom: 24px;
}

.author-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.author-avatar {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: #fff;
}

.author-name {
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
}

.post-time {
  font-size: 14px;
  color: #8D7B68;
}

.post-actions {
  display: flex;
  gap: 12px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
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

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #8B7355;
  font-size: 14px;
}

.action-btn:hover {
  background: #E5D4C1;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn.active {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.post-content {
  font-size: 16px;
  line-height: 1.8;
  color: #2C1810;
  white-space: pre-wrap;
}

.comments-section {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.section-title {
  font-size: 24px;
  font-weight: 700;
  color: #2C1810;
  margin-bottom: 20px;
}

.comment-input {
  margin-bottom: 24px;
}

.comment-input textarea {
  width: 100%;
  padding: 16px;
  font-size: 15px;
  line-height: 1.6;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  color: #2C1810;
  resize: vertical;
  font-family: inherit;
  margin-bottom: 12px;
}

.comment-input textarea:focus {
  outline: none;
  border-color: #804030;
}

.comment-input textarea:disabled {
  background: #F5F5F5;
  cursor: not-allowed;
  opacity: 0.6;
}

.comment-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #FFF5F5;
  border: 1px solid #FEB2B2;
  border-radius: 8px;
  color: #C53030;
  font-size: 14px;
  margin-bottom: 12px;
}

.comment-error i {
  font-size: 16px;
}

.submit-btn {
  padding: 10px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.submit-btn:hover {
  background: #6B3528;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background: #8D7B68;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.comment-item {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #F5EFE7;
  border-radius: 12px;
}

.comment-avatar {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #B87333, #804030);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.comment-body {
  flex: 1;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.comment-author {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
}

.comment-time {
  font-size: 12px;
  color: #8D7B68;
}

.comment-content {
  font-size: 14px;
  line-height: 1.6;
  color: #4B3621;
}

.empty-comments {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #8D7B68;
}

.empty-comments i {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.3;
}

.empty-comments p {
  font-size: 14px;
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }
</style>
