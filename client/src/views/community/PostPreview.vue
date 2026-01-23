<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { POST_CATEGORIES } from '@/api/post'
import { useUserStore } from '@/stores/user'
import { buildNameStyle } from '@/utils/userNameStyle'
import { handleJumpLinkClick, sanitizeJumpLinks, hydrateJumpCardImages } from '@/utils/jumpLink'

const router = useRouter()
const userStore = useUserStore()
const mounted = ref(false)

interface PreviewData {
  title: string
  content: string
  category: string
  tag_ids: number[]
  guild_id?: number
  event_type?: string
  event_start_time?: string
  event_end_time?: string
  selectedTagNames: string[]
}

const previewData = ref<PreviewData | null>(null)
const articleContentRef = ref<HTMLElement | null>(null)

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  loadPreviewData()
})

function loadPreviewData() {
  const data = sessionStorage.getItem('post_preview')
  if (data) {
    try {
      previewData.value = JSON.parse(data)
    } catch (e) {
      console.error('解析预览数据失败:', e)
      goBackToEdit()
    }
  } else {
    goBackToEdit()
  }
}

watch(() => previewData.value?.content, async () => {
  await nextTick()
  sanitizeJumpLinks(articleContentRef.value)
  hydrateJumpCardImages(articleContentRef.value)
})

function goBackToEdit() {
  router.back()
}

function formatDate(dateStr?: string) {
  if (!dateStr) {
    return new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
  }
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : '其他'
}

function handlePreviewContentClick(event: MouseEvent) {
  handleJumpLinkClick(event, router, {
    returnTo: {
      type: 'post',
      path: router.currentRoute.value.fullPath,
      title: previewData.value?.title || '帖子',
    },
  })
}
</script>

<template>
  <div class="post-preview-page" :class="{ 'animate-in': mounted }">
    <div v-if="!previewData" class="loading">加载中...</div>

    <div v-else class="content-layout">
      <!-- 主内容区 -->
      <main class="main-area">
        <!-- 返回按钮 -->
        <div class="nav-bar anim-item" style="--delay: 0">
          <button class="back-btn" @click="goBackToEdit">
            <div class="back-icon">
              <i class="ri-arrow-left-s-line"></i>
            </div>
            <span>返回编辑</span>
          </button>
          <div class="preview-badge">
            <i class="ri-eye-line"></i>
            预览模式
          </div>
        </div>

        <!-- 文章 -->
        <article class="article-card anim-item" style="--delay: 1">
          <div class="article-decoration"></div>

          <!-- 文章头部：作者 + 操作 -->
          <div class="article-top">
            <div class="author-section">
              <div class="author-avatar">
                <img v-if="userStore.user?.avatar" :src="userStore.user.avatar" alt="" />
                <span v-else>{{ userStore.user?.username?.charAt(0) || 'U' }}</span>
              </div>
              <div class="author-info">
                <h4 class="author-name" :style="buildNameStyle(userStore.user?.name_color, userStore.user?.name_bold)">{{ userStore.user?.username || '未知用户' }}</h4>
                <span class="post-date">{{ formatDate() }}</span>
              </div>
            </div>
            <div class="action-buttons">
              <button class="action-btn" disabled>
                <i class="ri-heart-line"></i>
                <span>0</span>
              </button>
              <button class="action-btn" disabled>
                <i class="ri-star-line"></i>
                <span>0</span>
              </button>
              <span class="view-count">
                <i class="ri-eye-line"></i>
                0
              </span>
            </div>
          </div>

          <!-- 文章内容 -->
          <div class="article-body">
            <header class="article-header">
              <div class="category-badge">
                <span class="badge-dot"></span>
                <span>{{ getCategoryLabel(previewData.category) }}</span>
              </div>
              <h1 class="article-title">{{ previewData.title || '无标题' }}</h1>
              <!-- 标签 -->
              <div v-if="previewData.selectedTagNames?.length" class="tags-row">
                <span v-for="tag in previewData.selectedTagNames" :key="tag" class="tag-item">
                  {{ tag }}
                </span>
              </div>
            </header>

            <div class="zen-divider"></div>

            <div ref="articleContentRef" class="article-content" v-html="previewData.content || '<p>无内容</p>'" @click="handlePreviewContentClick"></div>
          </div>
        </article>

        <!-- 评论区预览 -->
        <section class="comments-section anim-item" style="--delay: 2">
          <h3 class="comments-title">
            讨论 <span class="comment-badge">0</span>
          </h3>
          <div class="empty-comments">
            发布后将显示评论区
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<style scoped>
.post-preview-page {
  max-width: 1200px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  padding: 80px;
  color: #8D7B68;
  font-size: 16px;
}

/* ========== 单栏布局 ========== */
.content-layout {
  max-width: 1000px;
  margin: 0 auto;
}

/* ========== 导航栏 ========== */
.nav-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.preview-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #FEF3C7;
  border: 1px solid #FDE68A;
  border-radius: 20px;
  color: #D97706;
  font-size: 13px;
  font-weight: 500;
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
  overflow: hidden;
}

.author-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
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
  cursor: not-allowed;
  opacity: 0.6;
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
  opacity: 0.6;
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

.tags-row {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.tag-item {
  padding: 4px 12px;
  background: rgba(128, 64, 48, 0.1);
  border: 1px solid rgba(128, 64, 48, 0.2);
  border-radius: 4px;
  font-size: 12px;
  color: #804030;
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

.article-content :deep(.mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.12);
  color: #804030;
  font-weight: 600;
  margin: 0 2px;
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
