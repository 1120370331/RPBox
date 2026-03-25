<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { deleteItem, listItems, type Item } from '@/api/item'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const loading = ref(false)
const deleting = ref(false)
const items = ref<Item[]>([])
const showDeleteDialog = ref(false)
const deletingItem = ref<Item | null>(null)

const publishedItems = computed(() => items.value.filter((item) => item.status === 'published' && item.review_status === 'approved'))
const draftItems = computed(() => items.value.filter((item) => item.status === 'draft'))
const pendingItems = computed(() => items.value.filter((item) => item.status === 'pending' || item.review_status === 'pending'))

async function loadMyItems() {
  const authorId = userStore.user?.id
  if (!authorId) return
  loading.value = true
  try {
    const res = await listItems({
      author_id: authorId,
      status: 'all',
      sort: 'created_at',
      order: 'desc',
      page: 1,
      page_size: 100,
    })
    items.value = res.items || []
  } catch (error) {
    console.error('Failed to load my items', error)
    toast.error((error as Error)?.message || t('profile.myItems.loadFailed'))
  } finally {
    loading.value = false
  }
}

function formatDate(value: string) {
  const date = new Date(value)
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function statusText(item: Item) {
  if (item.status === 'draft') return t('profile.myItems.statusDraft')
  if (item.status === 'pending' || item.review_status === 'pending') return t('profile.myItems.statusPending')
  if (item.review_status === 'rejected') return t('profile.myItems.statusRejected')
  return t('profile.myItems.statusPublished')
}

function openDeleteDialog(item: Item) {
  deletingItem.value = item
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingItem.value || deleting.value) return
  deleting.value = true
  try {
    await deleteItem(deletingItem.value.id)
    items.value = items.value.filter((item) => item.id !== deletingItem.value?.id)
    toast.success(t('profile.myItems.deleteSuccess'))
    showDeleteDialog.value = false
    deletingItem.value = null
  } catch (error) {
    console.error('Failed to delete item', error)
    toast.error((error as Error)?.message || t('profile.myItems.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

onMounted(loadMyItems)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myItems.title') }}</h1>
      <button class="add-btn" @click="router.push({ name: 'item-create' })"><i class="ri-add-line" /></button>
    </header>
    <div class="sub-body">
      <div class="stats-row">
        <span>{{ $t('profile.myItems.statsPublished', { n: publishedItems.length }) }}</span>
        <span>{{ $t('profile.myItems.statsPending', { n: pendingItems.length }) }}</span>
        <span>{{ $t('profile.myItems.statsDraft', { n: draftItems.length }) }}</span>
      </div>

      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="items.length === 0" class="hint">{{ $t('profile.myItems.empty') }}</div>
      <div v-else class="item-list">
        <article v-for="item in items" :key="item.id" class="item-card">
          <button class="item-main" @click="router.push({ name: 'item-detail', params: { id: item.id } })">
            <div class="item-icon">
              <CachedImage
                v-if="item.preview_image_url || item.preview_image"
                :src="resolveApiUrl(item.preview_image_url || item.preview_image || '')"
                alt=""
              />
              <i v-else class="ri-box-3-line" />
            </div>
            <div class="item-info">
              <div class="title-row">
                <h3>{{ item.name }}</h3>
                <span class="status-tag">{{ statusText(item) }}</span>
              </div>
              <div class="meta">
                <span>{{ $t(`market.types.${item.type}`) }}</span>
                <span><i class="ri-download-line" /> {{ item.downloads }}</span>
                <span>★ {{ item.rating.toFixed(1) }}</span>
                <span>{{ formatDate(item.updated_at) }}</span>
              </div>
            </div>
          </button>
          <div class="actions">
            <button class="action-btn" @click="router.push({ name: 'item-edit', params: { id: item.id } })">
              {{ $t('profile.myItems.edit') }}
            </button>
            <button class="action-btn danger" @click="openDeleteDialog(item)">
              {{ $t('profile.myItems.delete') }}
            </button>
          </div>
        </article>
      </div>
    </div>

    <div v-if="showDeleteDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('profile.myItems.deleteTitle') }}</h3>
        <p>{{ $t('profile.myItems.deleteMessage') }}</p>
        <div class="dialog-actions">
          <button class="action-btn" @click="showDeleteDialog = false">{{ $t('profile.myItems.cancel') }}</button>
          <button class="action-btn danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? $t('common.status.loading') : $t('profile.myItems.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.add-btn {
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--text-light);
}

.stats-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 10px;
}

.item-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.item-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.item-main {
  width: 100%;
  border: none;
  background: transparent;
  text-align: left;
  padding: 10px;
  display: flex;
  gap: 10px;
}

.item-icon {
  width: 56px;
  height: 56px;
  border-radius: 10px;
  overflow: hidden;
  background: var(--icon-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.item-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.item-icon i {
  font-size: 24px;
  color: var(--icon-color);
}

.item-info {
  flex: 1;
}

.title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.title-row h3 {
  font-size: 15px;
  font-weight: 600;
}

.status-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 8px;
  background: var(--tag-bg);
  color: var(--tag-text);
}

.meta {
  margin-top: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.actions {
  border-top: 1px solid var(--color-border-light);
  display: flex;
  gap: 8px;
  padding: 8px 10px 10px;
}

.action-btn {
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 6px 12px;
  font-size: 12px;
}

.action-btn.danger {
  border-color: #c44747;
  color: #c44747;
}

.dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.48);
}

.dialog {
  width: 100%;
  max-width: 360px;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 14px;
}

.dialog h3 {
  font-size: 16px;
}

.dialog p {
  margin-top: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
