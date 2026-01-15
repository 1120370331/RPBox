<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listItems, deleteItem, type Item } from '@/api/item'
import { useToast } from '@/composables/useToast'
import { useDialog } from '@/composables/useDialog'

const router = useRouter()
const toast = useToast()
const dialog = useDialog()
const mounted = ref(false)
const loading = ref(false)
const items = ref<Item[]>([])
const currentUserId = ref<number>(0)
const filterStatus = ref<'all' | 'draft' | 'pending' | 'published'>('all')

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

// 判断道具的显示状态
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

// 过滤后的道具列表
const filteredItems = computed(() => {
  if (filterStatus.value === 'all') {
    return items.value
  }
  return items.value.filter(item => getItemDisplayStatus(item) === filterStatus.value)
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
    console.error('加载我的道具失败:', error)
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
    title: '确认删除',
    message: `确定要删除道具"${item.name}"吗？此操作不可恢复。`,
    confirmText: '删除',
    cancelText: '取消',
    type: 'danger'
  })

  if (!confirmed) return

  try {
    await deleteItem(item.id)
    toast.success('删除成功')
    await loadMyItems()
  } catch (error: any) {
    console.error('删除失败:', error)
    toast.error(error.message || '删除失败，请重试')
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
      return { text: '审核拒绝', class: 'rejected' }
    case 'pending':
      return { text: '待审核', class: 'pending' }
    case 'published':
      return { text: '已发布', class: 'published' }
    case 'draft':
    default:
      return { text: '草稿', class: 'draft' }
  }
}

// 获取类型显示
function getTypeText(type: string) {
  return type === 'item' ? '道具' : '剧本'
}
</script>

<template>
  <div class="my-items-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          返回
        </button>
        <h1 class="page-title">我的道具</h1>
      </div>
      <button class="create-btn" @click="goToUpload">
        <i class="ri-add-line"></i>
        上传道具
      </button>
    </div>

    <div class="stats anim-item" style="--delay: 1">
      <div class="stat-item">
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">全部</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.published }}</div>
        <div class="stat-label">已发布</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.pending }}</div>
        <div class="stat-label">待审核</div>
      </div>
      <div class="stat-item">
        <div class="stat-value">{{ stats.draft }}</div>
        <div class="stat-label">草稿</div>
      </div>
    </div>

    <div class="filters anim-item" style="--delay: 2">
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'all' }"
        @click="filterStatus = 'all'"
      >
        全部
      </button>
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'published' }"
        @click="filterStatus = 'published'"
      >
        已发布
      </button>
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'pending' }"
        @click="filterStatus = 'pending'"
      >
        待审核
      </button>
      <button
        class="filter-btn"
        :class="{ active: filterStatus === 'draft' }"
        @click="filterStatus = 'draft'"
      >
        草稿
      </button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="filteredItems.length === 0" class="empty anim-item" style="--delay: 3">
      <i class="ri-box-3-line"></i>
      <p>{{ filterStatus === 'all' ? '还没有上传任何道具' : `没有${filterStatus === 'draft' ? '草稿' : filterStatus === 'pending' ? '待审核' : '已发布'}的道具` }}</p>
      <button class="create-btn-large" @click="goToUpload">
        <i class="ri-add-line"></i>
        上传第一个道具
      </button>
    </div>

    <div v-else class="items-list">
      <div
        v-for="(item, index) in filteredItems"
        :key="item.id"
        class="item-card anim-item"
        :style="`--delay: ${index + 3}`"
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
          拒绝原因：{{ item.review_comment }}
        </div>

        <div class="item-content">{{ item.description || '暂无描述' }}</div>

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
              编辑
            </button>
            <button class="action-btn delete" @click="handleDelete(item)">
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
