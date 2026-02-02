<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Item } from '@/api/item'
import { handleJumpLinkClick, sanitizeJumpLinks, hydrateJumpCardImages } from '@/utils/jumpLink'

const router = useRouter()
const { t } = useI18n()
const item = ref<Item | null>(null)
const previewFrom = ref('')
const detailContentRef = ref<HTMLElement | null>(null)

onMounted(() => {
  loadPreviewData()
})

function loadPreviewData() {
  const previewDataStr = sessionStorage.getItem('item_preview_data')
  previewFrom.value = sessionStorage.getItem('item_preview_from') || '/market/upload'

  if (previewDataStr) {
    try {
      item.value = JSON.parse(previewDataStr)
    } catch (e) {
      console.error('解析预览数据失败:', e)
      router.push('/market')
    }
  } else {
    router.push('/market')
  }
}

function backToEdit() {
  router.push(previewFrom.value)
}

function copyImportCode() {
  if (item.value?.import_code) {
    navigator.clipboard.writeText(item.value.import_code)
  }
}

watch(() => item.value?.detail_content, async () => {
  await nextTick()
  sanitizeJumpLinks(detailContentRef.value)
  hydrateJumpCardImages(detailContentRef.value)
})

function handlePreviewContentClick(event: MouseEvent) {
  handleJumpLinkClick(event, router)
}
</script>

<template>
  <div class="item-preview-page">
    <!-- 预览横幅 -->
    <div class="preview-banner">
      <div class="preview-info">
        <i class="ri-eye-line"></i>
        <span>{{ t('market.detail.preview.banner') }}</span>
      </div>
      <button class="back-edit-btn" @click="backToEdit">
        <i class="ri-arrow-left-line"></i> {{ t('market.detail.preview.backToEdit') }}
      </button>
    </div>

    <div v-if="item" class="detail-container">
      <!-- 道具信息 -->
      <div class="item-info">
        <!-- 预览图 -->
        <div v-if="item.preview_image" class="item-preview">
          <img :src="item.preview_image" :alt="t('market.detail.previewImage')" />
        </div>

        <div class="item-header">
          <h1>{{ item.name }}</h1>
          <div class="item-meta">
            <span class="type-badge">{{ t(`market.types.${item.type}`) }}</span>
            <span class="author">{{ t('market.detail.author') }}: {{ t('market.preview.me') }}</span>
          </div>
        </div>

        <div class="item-stats">
          <div class="stat-item">
            <i class="ri-download-line"></i>
            <span>0 {{ t('market.detail.stats.downloads') }}</span>
          </div>
          <div class="stat-item">
            <i class="ri-star-fill"></i>
            <span>0.0 (0 {{ t('market.detail.stats.rating') }})</span>
          </div>
          <div class="stat-item">
            <i class="ri-heart-fill"></i>
            <span>0 {{ t('market.detail.stats.likes') }}</span>
          </div>
          <div class="stat-item">
            <i class="ri-bookmark-fill"></i>
            <span>0 {{ t('market.detail.stats.favorites') }}</span>
          </div>
        </div>

        <div class="item-description">
          <h3>{{ t('market.detail.description') }}</h3>
          <p>{{ item.description || t('market.item.noDescription') }}</p>
        </div>

        <!-- 详细介绍 -->
        <div v-if="item.detail_content" class="item-detail-content">
          <h3>{{ t('market.detail.detailContent') }}</h3>
          <div ref="detailContentRef" class="rich-content" v-html="item.detail_content" @click="handlePreviewContentClick"></div>
        </div>

        <div class="action-buttons">
          <button class="copy-code-btn" @click="copyImportCode">
            <i class="ri-file-copy-line"></i> {{ t('market.detail.actions.copyImportCode') }}
          </button>
          <button class="like-btn">
            <i class="ri-heart-line"></i> {{ t('market.detail.actions.like') }}
          </button>
          <button class="favorite-btn">
            <i class="ri-bookmark-line"></i> {{ t('market.detail.actions.favorite') }}
          </button>
        </div>
      </div>

      <!-- 评价区域 -->
      <div class="comments-section">
        <h3>{{ t('market.detail.comments.title') }} (0)</h3>
        <div class="empty-comments">
          {{ t('market.detail.comments.empty') }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.item-preview-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
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

.rich-content :deep(.mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.12);
  color: #804030;
  font-weight: 600;
  margin: 0 2px;
}

/* 附件卡片 */
.rich-content :deep(.attachment-card) {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin: 12px 0;
  background: #FDFBF9;
  border: 1px solid #E5D4C1;
  border-radius: 10px;
  transition: all 0.2s;
}

.rich-content :deep(.attachment-card:hover) {
  border-color: #804030;
  box-shadow: 0 2px 8px rgba(75, 54, 33, 0.08);
}

.rich-content :deep(.attachment-card__icon) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: rgba(128, 64, 48, 0.08);
  border-radius: 8px;
  flex-shrink: 0;
}

.rich-content :deep(.attachment-card__icon i) {
  font-size: 20px;
  color: #804030;
}

.rich-content :deep(.attachment-card__info) {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.rich-content :deep(.attachment-card__name) {
  font-size: 14px;
  font-weight: 500;
  color: #2C1810;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rich-content :deep(.attachment-card__size) {
  font-size: 12px;
  color: #8D7B68;
}

.rich-content :deep(.attachment-card__download) {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  background: #804030;
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.2s;
  flex-shrink: 0;
}

.rich-content :deep(.attachment-card__download:hover) {
  background: #6B3528;
  transform: translateY(-1px);
}

.rich-content :deep(.attachment-card__download i) {
  font-size: 16px;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin-top: 24px;
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

.empty-comments {
  text-align: center;
  padding: 40px 20px;
  color: #999;
  font-size: 14px;
}
</style>
