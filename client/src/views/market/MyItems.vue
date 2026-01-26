<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listItems, deleteItem, type Item } from '@/api/item'
import { useToast } from '@/composables/useToast'
import { useDialog } from '@/composables/useDialog'

const router = useRouter()
const { t } = useI18n()
const toast = useToast()
const dialog = useDialog()
const mounted = ref(false)
const loading = ref(false)
const items = ref<Item[]>([])
const currentUserId = ref<number>(0)
const filterStatus = ref<'all' | 'draft' | 'pending' | 'published'>('all')
const searchKeyword = ref('')
const typeFilter = ref<'all' | 'item' | 'campaign' | 'artwork'>('all')

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

// 判断作品的显示状态
function getItemDisplayStatus(item: Item): 'draft' | 'pending' | 'published' | 'rejected' {
  if (item.review_status === 'rejected') {
    return 'rejected'
  }
  if (item.status === 'published' && item.review_status === 'pending') {
    return 'pending'
  }
  if (item.status === 'published' && item.review_status === 'approved') {
    return 'published'
  }
  return 'draft'
}

// 过滤后的作品列表
const filteredItems = computed(() => {
  let list = items.value

  if (filterStatus.value !== 'all') {
    list = list.filter(item => getItemDisplayStatus(item) === filterStatus.value)
  }

  if (typeFilter.value !== 'all') {
    list = list.filter(item => item.type === typeFilter.value)
  }

  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) {
    return list
  }

  return list.filter(item => {
    const name = (item.name || '').toLowerCase()
    const description = (item.description || '').toLowerCase()
    const typeText = getTypeText(item.type).toLowerCase()
    return name.includes(keyword) || description.includes(keyword) || typeText.includes(keyword)
  })
})

// 统计数据
const stats = computed(() => {
  return {
    total: items.value.length,
    published: items.value.filter(i => getItemDisplayStatus(i) === 'published').length,
    pending: items.value.filter(i => getItemDisplayStatus(i) === 'pending').length,
    draft: items.value.filter(i => getItemDisplayStatus(i) === 'draft').length,
  }
})

const emptyMessage = computed(() => {
  const hasKeyword = searchKeyword.value.trim().length > 0
  const hasTypeFilter = typeFilter.value !== 'all'
  if (hasKeyword || hasTypeFilter) {
    return t('market.myItems.empty.noMatch')
  }
  if (filterStatus.value === 'all') {
    return t('market.myItems.empty.noItems')
  }
  if (filterStatus.value === 'draft') {
    return t('market.myItems.empty.noDraft')
  }
  if (filterStatus.value === 'pending') {
    return t('market.myItems.empty.noPending')
  }
  return t('market.myItems.empty.noPublished')
})

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await loadMyItems()
})

async function loadMyItems() {
  loading.value = true
  try {
    const res: any = await listItems({ author_id: currentUserId.value, status: 'all' })
    items.value = res.data?.items || []
  } catch (error) {
    console.error('加载我的作品失败:', error)
  } finally {
    loading.value = false
  }
}

function goToDetail(id: number) {
  router.push({ name: 'item-detail', params: { id } })
}

function goToEdit(id: number) {
  router.push({ name: 'item-edit', params: { id } })
}

async function handleDelete(item: Item) {
  const confirmed = await dialog.confirm({
    title: t('market.myItems.deleteConfirm.title'),
    message: t('market.myItems.deleteConfirm.message', { name: item.name }),
    confirmText: t('market.myItems.deleteConfirm.confirm'),
    cancelText: t('market.myItems.deleteConfirm.cancel'),
    type: 'danger'
  })

  if (!confirmed) return

  try {
    await deleteItem(item.id)
    toast.success(t('market.myItems.messages.deleteSuccess'))
    await loadMyItems()
  } catch (error: any) {
    console.error('删除失败:', error)
    toast.error(error.message || t('market.myItems.messages.deleteFailed'))
  }
}

function goToUpload() {
  router.push({ name: 'item-upload' })
}

function goBack() {
  router.push({ name: 'market' })
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 获取状态显示信息
function getStatusInfo(item: Item) {
  const displayStatus = getItemDisplayStatus(item)
  switch (displayStatus) {
    case 'rejected':
      return { text: t('market.myItems.status.rejected'), class: 'rejected' }
    case 'pending':
      return { text: t('market.myItems.status.pending'), class: 'pending' }
    case 'published':
      return { text: t('market.myItems.status.published'), class: 'published' }
    case 'draft':
    default:
      return { text: t('market.myItems.status.draft'), class: 'draft' }
  }
}

// 获取类型显示
function getTypeText(type: string) {
  return t(`market.types.${type}`)
}
</script>

<template>
  <div class="my-items-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          {{ t('market.myItems.back') }}
        </button>
        <h1 class="page-title">{{ t('market.myItems.title') }}</h1>
      </div>
      <button class="create-btn" @click="goToUpload">
        <i class="ri-add-line"></i>
        {{ t('market.myItems.uploadItem') }}
      </button>
    </div>

    <div class="stats anim-item" style="--delay: 1">
      <div class="stat-item">
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">{{ t('market.myItems.stats.total') }}</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.published }}</div>
        <div class="stat-label">{{ t('market.myItems.stats.published') }}</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.pending }}</div>
        <div class="stat-label">{{ t('market.myItems.stats.pending') }}</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.draft }}</div>
        <div class="stat-label">{{ t('market.myItems.stats.draft') }}</div>
      </div>
    </div>

    <div class="filters anim-item" style="--delay: 2">
      <div class="filter-buttons">
        <button
          class="filter-btn"
          :class="{ active: filterStatus === 'all' }"
          @click="filterStatus = 'all'"
        >
          {{ t('market.myItems.filter.all') }}
        </button>
        <button
          class="filter-btn"
          :class="{ active: filterStatus === 'published' }"
          @click="filterStatus = 'published'"
        >
          {{ t('market.myItems.filter.published') }}
        </button>
        <button
          class="filter-btn"
          :class="{ active: filterStatus === 'pending' }"
          @click="filterStatus = 'pending'"
        >
          {{ t('market.myItems.filter.pending') }}
        </button>
        <button
          class="filter-btn"
          :class="{ active: filterStatus === 'draft' }"
          @click="filterStatus = 'draft'"
        >
          {{ t('market.myItems.filter.draft') }}
        </button>
      </div>
      <div class="filter-search">
        <i class="ri-search-line"></i>
        <input
          v-model="searchKeyword"
          type="text"
          :placeholder="t('market.myItems.filter.searchPlaceholder')"
        />
      </div>
      <select v-model="typeFilter" class="type-select">
        <option value="all">{{ t('market.myItems.filter.allTypes') }}</option>
        <option value="item">{{ t('market.types.item') }}</option>
        <option value="campaign">{{ t('market.types.campaign') }}</option>
        <option value="artwork">{{ t('market.types.artwork') }}</option>
      </select>
    </div>

    <div v-if="loading" class="loading">{{ t('market.myItems.loading') }}</div>

    <div v-else-if="filteredItems.length === 0" class="empty anim-item" style="--delay: 3">
      <i class="ri-box-3-line"></i>
      <p>{{ emptyMessage }}</p>
      <button class="create-btn-large" @click="goToUpload">
        <i class="ri-add-line"></i>
        {{ t('market.myItems.uploadFirst') }}
      </button>
    </div>

    <div v-else class="items-list">
      <div
        v-for="item in filteredItems"
        :key="item.id"
        class="item-card"
      >
        <div class="item-header">
          <div class="item-info">
            <h2 class="item-name" @click="goToDetail(item.id)">{{ item.name }}</h2>
            <span class="item-type">{{ getTypeText(item.type) }}</span>
          </div>
          <span class="status-badge" :class="getStatusInfo(item).class">
            {{ getStatusInfo(item).text }}
          </span>
        </div>

        <!-- 审核拒绝原因 -->
        <div v-if="item.review_status === 'rejected' && item.review_comment" class="reject-reason">
          <i class="ri-error-warning-line"></i>
          {{ t('market.myItems.rejectReason') }}{{ item.review_comment }}
        </div>

        <div class="item-content">{{ item.description || t('market.item.noDescription') }}</div>

        <div class="item-footer">
          <div class="item-meta">
            <span class="meta-item">
              <i class="ri-download-line"></i>
              {{ item.downloads }}
            </span>
            <span class="meta-item">
              <i class="ri-star-line"></i>
              {{ item.rating.toFixed(1) }}
            </span>
            <span class="meta-item">
              <i class="ri-heart-line"></i>
              {{ item.like_count }}
            </span>
            <span class="meta-item">
              <i class="ri-time-line"></i>
              {{ formatDate(item.updated_at) }}
            </span>
          </div>

          <div class="item-actions">
            <button class="action-btn edit" @click="goToEdit(item.id)">
              <i class="ri-edit-line"></i>
              {{ t('market.myItems.actions.edit') }}
            </button>
            <button class="action-btn delete" @click="handleDelete(item)">
              <i class="ri-delete-bin-line"></i>
              {{ t('market.myItems.actions.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.my-items-page {
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
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.create-btn:hover {
  background: #A66629;
  transform: translateY(-2px);
}

.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
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
  color: #B87333;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #8D7B68;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.filter-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  flex: 1;
  min-width: 220px;
}

.filter-search i {
  font-size: 16px;
  color: #8D7B68;
}

.filter-search input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 14px;
  color: #4B3621;
  background: transparent;
}

.filter-search input::placeholder {
  color: #8D7B68;
}

.type-select {
  padding: 10px 14px;
  border: 2px solid #E5D4C1;
  border-radius: 12px;
  background: #fff;
  font-size: 14px;
  color: #4B3621;
  min-width: 120px;
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
  background: #B87333;
  border-color: #B87333;
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
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.create-btn-large:hover {
  background: #A66629;
  transform: translateY(-2px);
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.item-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
  transition: all 0.3s;
}

.item-card:hover {
  box-shadow: 0 6px 16px rgba(75,54,33,0.1);
  transform: translateY(-2px);
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.item-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.item-name {
  font-size: 22px;
  font-weight: 700;
  color: #2C1810;
  margin: 0;
  cursor: pointer;
  transition: color 0.3s;
}

.item-name:hover {
  color: #B87333;
}

.item-type {
  padding: 4px 10px;
  background: #F5EFE7;
  color: #8D7B68;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.draft {
  background: #9E9E9E;
  color: #fff;
}

.status-badge.pending {
  background: #FFA500;
  color: #fff;
}

.status-badge.published {
  background: #4CAF50;
  color: #fff;
}

.status-badge.rejected {
  background: #F44336;
  color: #fff;
}

.reject-reason {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #FFF3F3;
  border: 1px solid #FFCDD2;
  border-radius: 8px;
  color: #C62828;
  font-size: 14px;
  margin-bottom: 12px;
}

.reject-reason i {
  font-size: 18px;
}

.item-content {
  font-size: 15px;
  line-height: 1.6;
  color: #4B3621;
  margin-bottom: 16px;
}

.item-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 2px solid #F5EFE7;
}

.item-meta {
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

.item-actions {
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
