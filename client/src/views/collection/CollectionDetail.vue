<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../../stores/user'
import { getCollection, deleteCollection, favoriteCollection, unfavoriteCollection, reorderCollectionPosts, reorderCollectionItems, type CollectionDetail } from '../../api/collection'
import { useDialog } from '../../composables/useDialog'
import { useToast } from '../../composables/useToast'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const dialog = useDialog()
const toast = useToast()

const collection = ref<CollectionDetail | null>(null)
const loading = ref(true)
const isFavorited = ref(false)
const favoriteLoading = ref(false)
const reorderMode = ref(false)
const reordering = ref(false)
const draggedIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)
const dragType = ref<'post' | 'item' | null>(null)

const isOwner = computed(() => {
  return collection.value?.author_id === userStore.user?.id
})

const canReorder = computed(() => {
  return isOwner.value && collection.value?.allow_reorder
})

async function loadCollection() {
  const id = Number(route.params.id)
  if (!id) {
    router.push({ name: 'community' })
    return
  }

  loading.value = true
  try {
    collection.value = await getCollection(id)
    isFavorited.value = collection.value.is_favorited || false
  } catch (e) {
    toast.error(t('collection.detail.loadFailed'))
    router.push({ name: 'community' })
  } finally {
    loading.value = false
  }
}

function goToPost(postId: number) {
  router.push({ name: 'post-detail', params: { id: postId } })
}

function goToItem(itemId: number) {
  router.push({ name: 'item-detail', params: { id: itemId } })
}

async function handleDelete() {
  if (!collection.value) return

  const confirmed = await dialog.confirm({
    title: t('collection.delete.title'),
    message: t('collection.delete.confirm'),
    type: 'warning',
  })

  if (!confirmed) return

  try {
    await deleteCollection(collection.value.id)
    toast.success(t('collection.delete.success'))
    router.push({ name: 'community' })
  } catch (e) {
    toast.error(t('collection.delete.failed'))
  }
}

async function toggleFavorite() {
  if (!collection.value || favoriteLoading.value) return
  if (!userStore.user) {
    toast.warning(t('common.message.loginRequired'))
    return
  }

  favoriteLoading.value = true
  try {
    if (isFavorited.value) {
      await unfavoriteCollection(collection.value.id)
      isFavorited.value = false
      toast.success(t('collection.favorite.removed'))
    } else {
      await favoriteCollection(collection.value.id)
      isFavorited.value = true
      toast.success(t('collection.favorite.added'))
    }
  } catch (e) {
    toast.error(t('collection.favorite.failed'))
  } finally {
    favoriteLoading.value = false
  }
}

// 拖拽排序相关
function onDragStart(index: number, type: 'post' | 'item') {
  if (!reorderMode.value) return
  draggedIndex.value = index
  dragType.value = type
}

function onDragOver(e: DragEvent, index: number, type: 'post' | 'item') {
  if (!reorderMode.value || dragType.value !== type) return
  e.preventDefault()
  dragOverIndex.value = index
}

function onDragLeave() {
  dragOverIndex.value = null
}

function onDrop(index: number, type: 'post' | 'item') {
  if (!reorderMode.value || dragType.value !== type) return
  if (draggedIndex.value === null || draggedIndex.value === index) {
    resetDragState()
    return
  }

  if (type === 'post' && collection.value?.posts) {
    const items = [...collection.value.posts]
    const [removed] = items.splice(draggedIndex.value, 1)
    items.splice(index, 0, removed)
    collection.value.posts = items
  } else if (type === 'item' && collection.value?.items) {
    const items = [...collection.value.items]
    const [removed] = items.splice(draggedIndex.value, 1)
    items.splice(index, 0, removed)
    collection.value.items = items
  }

  resetDragState()
}

function resetDragState() {
  draggedIndex.value = null
  dragOverIndex.value = null
  dragType.value = null
}

async function saveReorder() {
  if (!collection.value) return
  reordering.value = true

  try {
    if (collection.value.posts && collection.value.posts.length > 0) {
      const postIds = collection.value.posts.map(p => p.id)
      await reorderCollectionPosts(collection.value.id, postIds)
    }
    if (collection.value.items && collection.value.items.length > 0) {
      const itemIds = collection.value.items.map(i => i.id)
      await reorderCollectionItems(collection.value.id, itemIds)
    }
    toast.success(t('collection.detail.reorderSuccess'))
    reorderMode.value = false
  } catch (e) {
    toast.error(t('collection.detail.reorderFailed'))
  } finally {
    reordering.value = false
  }
}

function toggleReorderMode() {
  if (reorderMode.value) {
    saveReorder()
  } else {
    reorderMode.value = true
  }
}

onMounted(loadCollection)
</script>

<template>
  <div class="collection-detail">
    <div v-if="loading" class="loading">
      <i class="ri-loader-4-line spin"></i>
      {{ $t('common.status.loading') }}
    </div>

    <template v-else-if="collection">
      <!-- Header -->
      <div class="detail-header">
        <div class="header-info">
          <h1 class="collection-title">
            <i class="ri-folder-line"></i>
            {{ collection.name }}
          </h1>
          <div class="collection-meta">
            <span class="author">{{ collection.author_name }}</span>
            <span class="divider">·</span>
            <span class="count">{{ collection.item_count }} {{ $t('collection.detail.items') }}</span>
          </div>
          <p v-if="collection.description" class="collection-desc">
            {{ collection.description }}
          </p>
        </div>
        <div class="header-actions">
          <button
            v-if="canReorder"
            class="btn-reorder"
            :class="{ active: reorderMode }"
            :disabled="reordering"
            @click="toggleReorderMode"
          >
            <i :class="reorderMode ? 'ri-check-line' : 'ri-drag-move-line'"></i>
            {{ reorderMode ? $t('collection.detail.reorderDone') : $t('collection.detail.reorderMode') }}
          </button>
          <button
            class="btn-favorite"
            :class="{ active: isFavorited }"
            :disabled="favoriteLoading"
            @click="toggleFavorite"
          >
            <i :class="isFavorited ? 'ri-star-fill' : 'ri-star-line'"></i>
            {{ isFavorited ? $t('collection.favorite.favorited') : $t('collection.favorite.favorite') }}
          </button>
          <button v-if="isOwner" class="btn-delete" @click="handleDelete">
            <i class="ri-delete-bin-line"></i>
            {{ $t('common.button.delete') }}
          </button>
        </div>
      </div>

      <!-- Content List -->
      <div class="content-list">
        <!-- Posts -->
        <div v-if="collection.posts && collection.posts.length > 0" class="content-section">
          <h2 class="section-title">
            <i class="ri-article-line"></i>
            {{ $t('collection.detail.posts') }}
            <span v-if="reorderMode" class="reorder-tip">{{ $t('collection.detail.reorderTip') }}</span>
          </h2>
          <div class="content-items">
            <div
              v-for="(post, index) in collection.posts"
              :key="post.id"
              class="content-item"
              :class="{
                draggable: reorderMode,
                dragging: draggedIndex === index && dragType === 'post',
                'drag-over': dragOverIndex === index && dragType === 'post'
              }"
              :draggable="reorderMode"
              @dragstart="onDragStart(index, 'post')"
              @dragover="onDragOver($event, index, 'post')"
              @dragleave="onDragLeave"
              @drop="onDrop(index, 'post')"
              @dragend="resetDragState"
              @click="!reorderMode && goToPost(post.id)"
            >
              <i v-if="reorderMode" class="ri-drag-move-2-line drag-handle"></i>
              <span class="item-index">{{ index + 1 }}</span>
              <span class="item-title">{{ post.title }}</span>
              <i v-if="!reorderMode" class="ri-arrow-right-s-line"></i>
            </div>
          </div>
        </div>

        <!-- Items -->
        <div v-if="collection.items && collection.items.length > 0" class="content-section">
          <h2 class="section-title">
            <i class="ri-box-3-line"></i>
            {{ $t('collection.detail.works') }}
            <span v-if="reorderMode" class="reorder-tip">{{ $t('collection.detail.reorderTip') }}</span>
          </h2>
          <div class="content-items">
            <div
              v-for="(item, index) in collection.items"
              :key="item.id"
              class="content-item"
              :class="{
                draggable: reorderMode,
                dragging: draggedIndex === index && dragType === 'item',
                'drag-over': dragOverIndex === index && dragType === 'item'
              }"
              :draggable="reorderMode"
              @dragstart="onDragStart(index, 'item')"
              @dragover="onDragOver($event, index, 'item')"
              @dragleave="onDragLeave"
              @drop="onDrop(index, 'item')"
              @dragend="resetDragState"
              @click="!reorderMode && goToItem(item.id)"
            >
              <i v-if="reorderMode" class="ri-drag-move-2-line drag-handle"></i>
              <span class="item-index">{{ index + 1 }}</span>
              <span class="item-title">{{ item.name }}</span>
              <i v-if="!reorderMode" class="ri-arrow-right-s-line"></i>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="(!collection.posts || collection.posts.length === 0) && (!collection.items || collection.items.length === 0)" class="empty-state">
          <i class="ri-folder-open-line"></i>
          <p>{{ $t('collection.detail.empty') }}</p>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.collection-detail {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px;
  color: var(--color-text-secondary);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--color-border);
}

.collection-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-main);
  margin: 0 0 12px;
}

.collection-title i {
  color: var(--color-primary);
}

.collection-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.divider {
  opacity: 0.5;
}

.collection-desc {
  color: var(--color-text-secondary);
  line-height: 1.6;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

.btn-favorite {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg);
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-favorite:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.btn-favorite.active {
  background: rgba(255, 193, 7, 0.1);
  border-color: #ffc107;
  color: #ffc107;
}

.btn-favorite:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-reorder {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg);
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-reorder:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.btn-reorder.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #fff;
}

.btn-reorder:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border: none;
  border-radius: 8px;
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-delete:hover {
  background: rgba(220, 53, 69, 0.2);
}

.content-section {
  margin-bottom: 32px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main);
  margin: 0 0 16px;
}

.section-title i {
  color: var(--color-primary);
}

.content-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.content-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--color-panel-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.content-item:hover {
  border-color: var(--color-primary);
  background: var(--color-primary-light, rgba(128, 64, 48, 0.05));
}

.content-item.draggable {
  cursor: grab;
}

.content-item.draggable:active {
  cursor: grabbing;
}

.content-item.dragging {
  opacity: 0.5;
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
}

.content-item.drag-over {
  border-color: var(--color-primary);
  border-style: dashed;
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
}

.drag-handle {
  color: var(--color-text-secondary);
  font-size: 18px;
  cursor: grab;
}

.reorder-tip {
  font-size: 12px;
  font-weight: 400;
  color: var(--color-text-secondary);
  margin-left: auto;
}

.item-index {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  flex-shrink: 0;
}

.item-title {
  flex: 1;
  font-size: 15px;
  color: var(--color-text-main);
}

.content-item i {
  color: var(--color-text-secondary);
  font-size: 18px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px;
  color: var(--color-text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}
</style>
