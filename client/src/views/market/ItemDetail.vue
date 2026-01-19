<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getItem, downloadItem, getItemComments, addItemComment, likeItem, unlikeItem, favoriteItem, unfavoriteItem, getItemImageDownloadUrl, getItemImageUrl, type Item, type ItemComment, type ItemImage } from '@/api/item'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const loading = ref(false)
const item = ref<Item | null>(null)
const author = ref<any>(null)
const tags = ref<any[]>([])
const images = ref<ItemImage[]>([])  // 画作图片列表
const comments = ref<ItemComment[]>([])
const newRating = ref(0)
const hoverRating = ref(0)
const newComment = ref('')
const showImportCode = ref(false)
const isLiked = ref(false)
const isFavorited = ref(false)
const submitting = ref(false)

// 画作图片查看
const selectedImageIndex = ref(0)
const showImageViewer = ref(false)

// 预览模式
const isPreview = ref(false)
const previewFrom = ref('')

// 是否为画作类型
const isArtwork = computed(() => item.value?.type === 'artwork')

// 获取类型显示文本
function getTypeText(type: string) {
  const typeMap: Record<string, string> = {
    'item': '道具',
    'campaign': '剧本',
    'artwork': '画作'
  }
  return typeMap[type] || type
}

// 计算评论字数
const commentLength = computed(() => [...newComment.value].length)
// 提交条件：有评分时需要10字，无评分时只需要有内容
const canSubmit = computed(() => {
  if (newRating.value > 0) {
    return commentLength.value >= 10
  }
  return commentLength.value > 0
})

onMounted(() => {
  // 检测预览模式
  if (route.query.preview === '1') {
    isPreview.value = true
    previewFrom.value = sessionStorage.getItem('item_preview_from') || ''
    loadPreviewData()
    // 预览模式不加载评论
  } else {
    loadItemDetail()
    loadComments()
  }
})

// 加载预览数据
function loadPreviewData() {
  const previewDataStr = sessionStorage.getItem('item_preview_data')
  if (previewDataStr) {
    try {
      item.value = JSON.parse(previewDataStr)
    } catch (e) {
      console.error('解析预览数据失败:', e)
      loadItemDetail()
    }
  } else {
    loadItemDetail()
  }
}

// 返回编辑
function backToEdit() {
  if (previewFrom.value) {
    router.push(previewFrom.value)
  } else {
    router.back()
  }
  // 清理预览数据
  sessionStorage.removeItem('item_preview_data')
  sessionStorage.removeItem('item_preview_from')
}

// 加载道具详情
async function loadItemDetail() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res: any = await getItem(id)
    if (res.code === 0) {
      item.value = res.data.item
      author.value = res.data.author
      tags.value = res.data.tags || []
      // 画作类型加载图片列表，并转换为完整 URL
      if (res.data.images) {
        images.value = res.data.images.map((img: any) => ({
          ...img,
          image_url: getItemImageUrl(id, img.id)
        }))
      }
    }
  } catch (error) {
    console.error('加载作品详情失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载评论
async function loadComments() {
  try {
    const id = Number(route.params.id)
    const res: any = await getItemComments(id)
    if (res.code === 0) {
      comments.value = res.data || []
    }
  } catch (error) {
    console.error('加载评论失败:', error)
  }
}

// 下载道具
async function handleDownload() {
  try {
    const id = Number(route.params.id)
    const res: any = await downloadItem(id)
    if (res.code === 0) {
      showImportCode.value = true
    }
  } catch (error) {
    console.error('下载失败:', error)
  }
}

// 添加评论（可选评分）
async function handleAddComment() {
  if (!canSubmit.value) {
    if (newRating.value > 0 && commentLength.value < 10) {
      toast.error('带评分的评价至少需要10个字符')
    } else {
      toast.error('请输入评论内容')
    }
    return
  }

  submitting.value = true
  try {
    const id = Number(route.params.id)
    await addItemComment(id, newRating.value, newComment.value)
    newComment.value = ''
    newRating.value = 0
    loadComments()
    loadItemDetail()
    toast.success('评价发表成功')
  } catch (error: any) {
    console.error('添加评论失败:', error)
    toast.error(error.message || '添加评论失败')
  } finally {
    submitting.value = false
  }
}

// 点赞
async function handleLike() {
  try {
    const id = Number(route.params.id)
    if (isLiked.value) {
      await unlikeItem(id)
      isLiked.value = false
      if (item.value) item.value.like_count--
      toast.success('已取消点赞')
    } else {
      await likeItem(id)
      isLiked.value = true
      if (item.value) item.value.like_count++
      toast.success('点赞成功')
    }
  } catch (error) {
    console.error('点赞操作失败:', error)
    toast.error('操作失败，请重试')
  }
}

// 收藏
async function handleFavorite() {
  try {
    const id = Number(route.params.id)
    if (isFavorited.value) {
      await unfavoriteItem(id)
      isFavorited.value = false
      if (item.value) item.value.favorite_count--
      toast.success('已取消收藏')
    } else {
      await favoriteItem(id)
      isFavorited.value = true
      if (item.value) item.value.favorite_count++
      toast.success('收藏成功')
    }
  } catch (error) {
    console.error('收藏操作失败:', error)
    toast.error('操作失败，请重试')
  }
}

// 复制导入代码
async function copyImportCode() {
  if (!item.value?.import_code) return

  try {
    // 调用下载API记录下载（每用户每道具最多贡献1次）
    const id = Number(route.params.id)
    await downloadItem(id)

    // 复制到剪贴板
    navigator.clipboard.writeText(item.value.import_code)
    toast.success('导入代码已复制到剪贴板')

    // 更新本地下载数（如果是首次下载，后端会+1）
    loadItemDetail()
  } catch (error) {
    console.error('复制失败:', error)
    // 即使API调用失败，仍然复制到剪贴板
    navigator.clipboard.writeText(item.value.import_code)
    toast.success('导入代码已复制到剪贴板')
  }
}

// 返回列表
function goBack() {
  router.push('/market')
}

// 打开图片查看器
function openImageViewer(index: number) {
  selectedImageIndex.value = index
  showImageViewer.value = true
}

// 关闭图片查看器
function closeImageViewer() {
  showImageViewer.value = false
}

// 上一张图片
function prevImage() {
  if (selectedImageIndex.value > 0) {
    selectedImageIndex.value--
  } else {
    selectedImageIndex.value = images.value.length - 1
  }
}

// 下一张图片
function nextImage() {
  if (selectedImageIndex.value < images.value.length - 1) {
    selectedImageIndex.value++
  } else {
    selectedImageIndex.value = 0
  }
}

// 下载当前图片
function downloadCurrentImage() {
  if (!item.value || images.value.length === 0) return
  const img = images.value[selectedImageIndex.value]
  const url = getItemImageDownloadUrl(item.value.id, img.id, true)
  window.open(url, '_blank')
  toast.success('开始下载图片')
}

// 下载所有图片
function downloadAllImages() {
  if (!item.value || images.value.length === 0) return
  for (const img of images.value) {
    const url = getItemImageDownloadUrl(item.value.id, img.id, true)
    window.open(url, '_blank')
  }
  toast.success(`开始下载 ${images.value.length} 张图片`)
}
</script>

<template>
  <div class="item-detail-page">
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="item" class="detail-container">
      <!-- 返回按钮 -->
      <button class="back-btn" @click="goBack" v-if="!isPreview">
        <i class="ri-arrow-left-line"></i> 返回列表
      </button>

      <!-- 预览模式横幅 -->
      <div v-if="isPreview" class="preview-banner">
        <div class="preview-info">
          <i class="ri-eye-line"></i>
          <span>预览模式 - 这是道具发布后的效果预览</span>
        </div>
        <button class="back-edit-btn" @click="backToEdit">
          <i class="ri-arrow-left-line"></i> 返回编辑
        </button>
      </div>

      <!-- 道具信息 -->
      <div class="item-info">
        <!-- 画作图片画廊（仅画作类型） -->
        <div v-if="isArtwork && images.length > 0" class="artwork-gallery">
          <div class="gallery-main">
            <img
              :src="images[selectedImageIndex]?.image_url"
              alt="画作"
              @click="openImageViewer(selectedImageIndex)"
            />
            <div class="gallery-nav" v-if="images.length > 1">
              <button class="nav-btn prev" @click.stop="prevImage">
                <i class="ri-arrow-left-s-line"></i>
              </button>
              <button class="nav-btn next" @click.stop="nextImage">
                <i class="ri-arrow-right-s-line"></i>
              </button>
            </div>
            <div class="gallery-counter">{{ selectedImageIndex + 1 }} / {{ images.length }}</div>
          </div>
          <div class="gallery-thumbs" v-if="images.length > 1">
            <div
              v-for="(img, index) in images"
              :key="img.id"
              class="thumb-item"
              :class="{ active: index === selectedImageIndex }"
              @click="selectedImageIndex = index"
            >
              <img :src="img.image_url" alt="" />
            </div>
          </div>
        </div>

        <!-- 预览图（非画作类型） -->
        <div v-else-if="item.preview_image" class="item-preview">
          <img :src="item.preview_image" alt="预览图" />
        </div>

        <div class="item-header">
          <h1>{{ item.name }}</h1>
          <div class="item-meta">
            <span class="type-badge">{{ getTypeText(item.type) }}</span>
            <span class="author">作者: {{ author?.username || '未知' }}</span>
            <span class="permission-badge" v-if="item.requires_permission && !isArtwork">
              <i class="ri-shield-keyhole-line"></i> 需要权限
              <div class="permission-tooltip">
                <p><strong>道具作者：</strong>{{ author?.username || '未知' }}</p>
                <p>这个道具需要你在游戏内对道具 <strong>Shift+右键点击</strong> 来调整安全性设置后才能正常使用。</p>
              </div>
            </span>
          </div>
        </div>

        <div class="item-stats">
          <div class="stat-item">
            <i class="ri-download-line"></i>
            <span>{{ item.downloads }} 下载</span>
          </div>
          <div class="stat-item">
            <i class="ri-star-fill"></i>
            <span>{{ item.rating.toFixed(1) }} ({{ item.rating_count }} 评价)</span>
          </div>
          <div class="stat-item">
            <i class="ri-heart-fill"></i>
            <span>{{ item.like_count || 0 }} 点赞</span>
          </div>
          <div class="stat-item">
            <i class="ri-bookmark-fill"></i>
            <span>{{ item.favorite_count || 0 }} 收藏</span>
          </div>
        </div>

        <div class="item-description">
          <h3>描述</h3>
          <p>{{ item.description || '暂无描述' }}</p>
        </div>

        <!-- 详细介绍 -->
        <div v-if="item.detail_content" class="item-detail-content">
          <h3>详细介绍</h3>
          <div class="rich-content" v-html="item.detail_content"></div>
        </div>

        <div class="item-tags" v-if="tags.length > 0">
          <span v-for="tag in tags" :key="tag.id" class="tag">{{ tag.name }}</span>
        </div>

        <div class="action-buttons">
          <!-- 画作类型：下载图片按钮 -->
          <template v-if="isArtwork">
            <button class="download-images-btn" @click="downloadAllImages">
              <i class="ri-download-line"></i> 下载全部图片 ({{ images.length }})
            </button>
          </template>
          <!-- 非画作类型：复制导入代码按钮 -->
          <template v-else>
            <button class="copy-code-btn" @click="copyImportCode">
              <i class="ri-file-copy-line"></i> 一键复制导入代码
            </button>
          </template>
          <button class="like-btn" :class="{ active: isLiked }" @click="handleLike">
            <i :class="isLiked ? 'ri-heart-fill' : 'ri-heart-line'"></i>
            {{ isLiked ? '已点赞' : '点赞' }}
          </button>
          <button class="favorite-btn" :class="{ active: isFavorited }" @click="handleFavorite">
            <i :class="isFavorited ? 'ri-bookmark-fill' : 'ri-bookmark-line'"></i>
            {{ isFavorited ? '已收藏' : '收藏' }}
          </button>
        </div>

        <div class="secondary-actions" v-if="!isArtwork">
          <button class="download-btn" @click="handleDownload">
            <i class="ri-eye-line"></i> 查看导入代码
          </button>
        </div>

        <!-- 导入代码显示区域（非画作类型） -->
        <div v-if="showImportCode && !isArtwork" class="import-code-section">
          <div class="import-code-header">
            <h3>导入代码</h3>
            <button class="copy-btn" @click="copyImportCode">
              <i class="ri-file-copy-line"></i> 复制代码
            </button>
          </div>
          <textarea
            :value="item?.import_code"
            readonly
            class="import-code-textarea"
            rows="8"
          ></textarea>
          <p class="import-hint">
            在游戏中打开 TRP3 Extended，点击导入按钮，粘贴上面的代码即可
          </p>
        </div>
      </div>

      <!-- 评价区域（评分+评论合并） -->
      <div class="comments-section">
        <h3>评价 ({{ comments.length }})</h3>

        <!-- 评价输入框 -->
        <div class="review-input-box">
          <div class="rating-input">
            <span class="rating-label">评分：</span>
            <div class="rating-stars">
              <i
                v-for="star in 5"
                :key="star"
                class="ri-star-fill"
                :class="{ active: star <= (hoverRating || newRating) }"
                @click="newRating = star"
                @mouseenter="hoverRating = star"
                @mouseleave="hoverRating = 0"
              ></i>
            </div>
            <span class="rating-value" v-if="newRating">{{ newRating }} 星</span>
            <button v-if="newRating" class="clear-rating-btn" @click="newRating = 0" type="button">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <textarea
            v-model="newComment"
            placeholder="写下你的评价...（带评分需至少10字）"
            rows="4"
          ></textarea>
          <div class="review-footer">
            <span class="char-count" :class="{ warning: newRating > 0 && commentLength < 10 }">
              {{ commentLength }}{{ newRating > 0 ? '/10' : '' }} 字
            </span>
            <button
              class="submit-review-btn"
              @click="handleAddComment"
              :disabled="!canSubmit || submitting"
            >
              {{ submitting ? '提交中...' : '发表评价' }}
            </button>
          </div>
        </div>

        <!-- 评论列表 -->
        <div class="comments-list">
          <div v-if="comments.length === 0" class="empty-comments">
            暂无评价，快来抢沙发吧！
          </div>
          <div v-else v-for="comment in comments" :key="comment.id" class="comment-item">
            <div class="comment-avatar">
              <img v-if="comment.avatar" :src="comment.avatar" alt="" />
              <span v-else>{{ comment.username?.charAt(0) || 'U' }}</span>
            </div>
            <div class="comment-body">
              <div class="comment-header">
                <div class="comment-user-info">
                  <span class="comment-author">{{ comment.username || '匿名用户' }}</span>
                  <div class="comment-rating">
                    <i v-for="star in 5" :key="star" class="ri-star-fill" :class="{ active: star <= comment.rating }"></i>
                  </div>
                </div>
                <span class="comment-time">{{ new Date(comment.created_at).toLocaleDateString() }}</span>
              </div>
              <p class="comment-content">{{ comment.content }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 图片查看器（全屏模态框） -->
    <div v-if="showImageViewer && isArtwork" class="image-viewer-modal" @click.self="closeImageViewer">
      <button class="viewer-close" @click="closeImageViewer">
        <i class="ri-close-line"></i>
      </button>
      <button class="viewer-nav prev" @click="prevImage" v-if="images.length > 1">
        <i class="ri-arrow-left-s-line"></i>
      </button>
      <div class="viewer-content">
        <img :src="images[selectedImageIndex]?.image_url" alt="画作" />
      </div>
      <button class="viewer-nav next" @click="nextImage" v-if="images.length > 1">
        <i class="ri-arrow-right-s-line"></i>
      </button>
      <div class="viewer-footer">
        <span class="viewer-counter">{{ selectedImageIndex + 1 }} / {{ images.length }}</span>
        <button class="viewer-download" @click="downloadCurrentImage">
          <i class="ri-download-line"></i> 下载此图
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.item-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}

.loading {
  text-align: center;
  padding: 60px 20px;
  color: #999;
  font-size: 16px;
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

.preview-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(135deg, #FFF3E0 0%, #FFE0B2 100%);
  border: 2px solid #FFB74D;
  border-radius: 12px;
  margin-bottom: 24px;
}

.preview-info {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #E65100;
  font-size: 15px;
  font-weight: 600;
}

.preview-info i {
  font-size: 20px;
}

.back-edit-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.back-edit-btn:hover {
  background: #A66629;
}

.item-info {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 8px 20px rgba(93,64,55,0.05);
}

.item-preview {
  margin-bottom: 24px;
  text-align: center;
}

.item-preview img {
  max-width: 100%;
  max-height: 400px;
  border-radius: 12px;
  object-fit: cover;
}

.item-detail-content {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #E0E0E0;
}

.item-detail-content h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 16px;
}

.rich-content {
  line-height: 1.8;
  color: #5D4037;
}

.rich-content img {
  max-width: 100%;
  border-radius: 8px;
  margin: 16px 0;
}

.item-header h1 {
  font-size: 32px;
  color: #3E2723;
  margin-bottom: 12px;
}

.item-meta {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-bottom: 24px;
}

.type-badge {
  padding: 6px 16px;
  background: #B87333;
  color: #fff;
  border-radius: 20px;
  font-size: 14px;
}

.author {
  color: #999;
  font-size: 14px;
}

.permission-badge {
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #FFF3E0;
  color: #E65100;
  border-radius: 20px;
  font-size: 13px;
  cursor: help;
}

.permission-badge i {
  font-size: 14px;
}

.permission-tooltip {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-top: 8px;
  padding: 12px 16px;
  background: #3E2723;
  color: #fff;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.6;
  width: 280px;
  opacity: 0;
  visibility: hidden;
  transition: all 0.2s;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0,0,0,0.2);
}

.permission-tooltip::before {
  content: '';
  position: absolute;
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-bottom: 6px solid #3E2723;
}

.permission-tooltip p {
  margin: 0 0 8px 0;
}

.permission-tooltip p:last-child {
  margin-bottom: 0;
}

.permission-badge:hover .permission-tooltip {
  opacity: 1;
  visibility: visible;
}

.item-stats {
  display: flex;
  gap: 24px;
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid #f0f0f0;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #666;
  font-size: 15px;
}

.stat-item i {
  color: #B87333;
  font-size: 18px;
}

.item-description {
  margin-bottom: 24px;
}

.item-description h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 12px;
}

.item-description p {
  color: #666;
  line-height: 1.6;
  font-size: 15px;
}

.item-tags {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.tag {
  padding: 6px 14px;
  background: #F5F0EB;
  color: #795548;
  border-radius: 20px;
  font-size: 13px;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.copy-code-btn {
  flex: 2;
  height: 48px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.3s;
}

.copy-code-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(184,115,51,0.4);
}

.secondary-actions {
  margin-bottom: 24px;
}

.download-btn {
  width: 100%;
  height: 40px;
  border: 1px solid #E0E0E0;
  border-radius: 8px;
  background: #fff;
  color: #666;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.3s;
}

.download-btn:hover {
  border-color: #B87333;
  color: #B87333;
}

.like-btn, .favorite-btn {
  flex: 1;
  height: 48px;
  border: 1px solid #E0E0E0;
  border-radius: 12px;
  background: #fff;
  color: #666;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.3s;
}

.like-btn:hover {
  border-color: #FF6B6B;
  color: #FF6B6B;
}

.like-btn.active {
  background: #FFF0F0;
  border-color: #FF6B6B;
  color: #FF6B6B;
}

.favorite-btn:hover {
  border-color: #FFB300;
  color: #FFB300;
}

.favorite-btn.active {
  background: #FFF8E1;
  border-color: #FFB300;
  color: #FFB300;
}

.import-code-section {
  margin-top: 24px;
  padding: 20px;
  background: #F5F0EB;
  border-radius: 12px;
}

.import-code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.import-code-header h3 {
  font-size: 16px;
  color: #3E2723;
  margin: 0;
}

.copy-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.3s;
}

.copy-btn:hover {
  background: #A66629;
}

.import-code-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #E0E0E0;
  border-radius: 8px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  background: #fff;
  resize: vertical;
}

.import-hint {
  margin-top: 12px;
  font-size: 13px;
  color: #795548;
  text-align: center;
}

.comments-section {
  margin-top: 32px;
  padding: 24px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 8px 20px rgba(93,64,55,0.05);
}

.comments-section h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 20px;
}

.review-input-box {
  margin-bottom: 24px;
  padding: 20px;
  background: #F5F0EB;
  border-radius: 12px;
}

.rating-input {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.rating-label {
  font-size: 14px;
  color: #5D4037;
  font-weight: 600;
}

.rating-stars {
  display: flex;
  gap: 4px;
}

.rating-stars i {
  font-size: 24px;
  color: #E0E0E0;
  cursor: pointer;
  transition: all 0.2s;
}

.rating-stars i.active {
  color: #FFB300;
}

.rating-stars i:hover {
  transform: scale(1.1);
}

.rating-value {
  font-size: 14px;
  color: #FFB300;
  font-weight: 600;
}

.clear-rating-btn {
  background: none;
  border: none;
  color: #999;
  cursor: pointer;
  padding: 2px 6px;
  font-size: 14px;
  border-radius: 4px;
  transition: all 0.2s;
}

.clear-rating-btn:hover {
  color: #E53935;
  background: rgba(229, 57, 53, 0.1);
}

.review-input-box textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #E0E0E0;
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  background: #fff;
}

.review-input-box textarea:focus {
  outline: none;
  border-color: #B87333;
}

.review-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.char-count {
  font-size: 13px;
  color: #999;
}

.char-count.warning {
  color: #FF6B6B;
}

.submit-review-btn {
  padding: 10px 24px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.3s;
}

.submit-review-btn:hover:not(:disabled) {
  background: #A66629;
}

.submit-review-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.empty-comments {
  text-align: center;
  padding: 40px 20px;
  color: #999;
  font-size: 14px;
}

.comment-item {
  padding: 16px;
  background: #F5F0EB;
  border-radius: 8px;
  display: flex;
  gap: 12px;
}

.comment-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
  overflow: hidden;
}

.comment-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}

.comment-user-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.comment-author {
  font-weight: 600;
  color: #5D4037;
  font-size: 14px;
}

.comment-rating {
  display: flex;
  gap: 2px;
}

.comment-rating i {
  font-size: 12px;
  color: #E0E0E0;
}

.comment-rating i.active {
  color: #FFB300;
}

.comment-time {
  font-size: 12px;
  color: #999;
}

.comment-content {
  color: #666;
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
}

/* 画作图片画廊样式 */
.artwork-gallery {
  margin-bottom: 24px;
}

.gallery-main {
  position: relative;
  background: #000;
  border-radius: 12px;
  overflow: hidden;
  aspect-ratio: 16/10;
  cursor: pointer;
}

.gallery-main img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.gallery-nav {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 12px;
  pointer-events: none;
}

.gallery-nav .nav-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(0,0,0,0.5);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  pointer-events: auto;
  transition: all 0.2s;
}

.gallery-nav .nav-btn:hover {
  background: rgba(0,0,0,0.7);
}

.gallery-counter {
  position: absolute;
  bottom: 12px;
  right: 12px;
  padding: 4px 12px;
  background: rgba(0,0,0,0.6);
  color: #fff;
  border-radius: 20px;
  font-size: 13px;
}

.gallery-thumbs {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  overflow-x: auto;
  padding-bottom: 8px;
}

.thumb-item {
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  border: 3px solid transparent;
  transition: all 0.2s;
}

.thumb-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb-item:hover {
  border-color: rgba(184,115,51,0.5);
}

.thumb-item.active {
  border-color: #B87333;
}

/* 下载图片按钮 */
.download-images-btn {
  flex: 2;
  height: 48px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.3s;
}

.download-images-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(184,115,51,0.4);
}

/* 图片查看器模态框 */
.image-viewer-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.95);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.viewer-close {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: rgba(255,255,255,0.1);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  z-index: 10;
  transition: background 0.2s;
}

.viewer-close:hover {
  background: rgba(255,255,255,0.2);
}

.viewer-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: rgba(255,255,255,0.1);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  z-index: 10;
  transition: background 0.2s;
}

.viewer-nav:hover {
  background: rgba(255,255,255,0.2);
}

.viewer-nav.prev {
  left: 20px;
}

.viewer-nav.next {
  right: 20px;
}

.viewer-content {
  max-width: 90%;
  max-height: 80%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.viewer-content img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.viewer-footer {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 20px;
}

.viewer-counter {
  color: #fff;
  font-size: 14px;
  background: rgba(0,0,0,0.5);
  padding: 6px 16px;
  border-radius: 20px;
}

.viewer-download {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.3s;
}

.viewer-download:hover {
  background: #A66629;
}
</style>
