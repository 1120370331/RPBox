<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listItems, type Item } from '@/api/item'

const router = useRouter()
const mounted = ref(false)
const loading = ref(false)
const items = ref<Item[]>([])
const total = ref(0)
const searchText = ref('')
const activeType = ref<'item' | 'script' | ''>('')
const sortBy = ref<'created_at' | 'downloads' | 'rating'>('created_at')
const currentPage = ref(1)

const typeMap = {
  '': '全部',
  'item': '道具',
  'script': '剧本'
}

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  loadItems()
})

// 加载道具列表
async function loadItems() {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      page_size: 12,
      sort: sortBy.value,
      order: 'desc'
    }

    if (activeType.value) {
      params.type = activeType.value
    }

    if (searchText.value) {
      params.search = searchText.value
    }

    const res: any = await listItems(params)
    if (res.code === 0) {
      items.value = res.data.items || []
      total.value = res.data.total || 0
    }
  } catch (error) {
    console.error('加载道具列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 切换类型
function changeType(type: '' | 'item' | 'script') {
  activeType.value = type
  currentPage.value = 1
  loadItems()
}

// 搜索
function handleSearch() {
  currentPage.value = 1
  loadItems()
}

// 查看详情
function viewDetail(id: number) {
  router.push(`/market/${id}`)
}

// 跳转到上传页面
function goToUpload() {
  router.push('/market/upload')
}

// 跳转到我的道具
function goToMyItems() {
  router.push('/market/my-items')
}

// 翻页
function changePage(page: number) {
  currentPage.value = page
  loadItems()
}

watch([sortBy], () => {
  currentPage.value = 1
  loadItems()
})
</script>

<template>
  <div class="market-page" :class="{ 'animate-in': mounted }">
    <!-- 头部 -->
    <div class="header anim-item" style="--delay: 0">
      <div class="header-top">
        <h1>道具市场</h1>
        <div class="header-actions">
          <button class="my-items-btn" @click="goToMyItems">
            <i class="ri-folder-user-line"></i> 我的道具
          </button>
          <button class="upload-btn" @click="goToUpload">
            <i class="ri-upload-line"></i> 上传道具
          </button>
        </div>
      </div>
      <div class="search-box">
        <input
          v-model="searchText"
          type="text"
          placeholder="搜索道具名称、类型或标签..."
          @keyup.enter="handleSearch"
        />
        <i class="ri-search-line" @click="handleSearch"></i>
      </div>
    </div>

    <!-- 筛选 -->
    <div class="filter-bar anim-item" style="--delay: 1">
      <span
        v-for="(label, type) in typeMap"
        :key="type"
        class="tag"
        :class="{ active: activeType === type }"
        @click="changeType(type as any)"
      >{{ label }}</span>

      <select v-model="sortBy" class="sort-select">
        <option value="created_at">最新</option>
        <option value="downloads">最热</option>
        <option value="rating">评分</option>
      </select>
    </div>

    <!-- 卡片网格 -->
    <div class="card-grid anim-item" style="--delay: 2">
      <div v-if="loading" class="loading-state">加载中...</div>
      <div v-else-if="items.length === 0" class="empty-state">暂无道具</div>
      <div v-else v-for="item in items" :key="item.id" class="card" @click="viewDetail(item.id)">
        <div class="card-image" :style="item.preview_image ? { backgroundImage: `url(${item.preview_image})` } : {}">
          <div v-if="!item.preview_image" class="placeholder-icon">
            <i class="ri-box-3-line"></i>
          </div>
        </div>
        <div class="card-content">
          <h3>{{ item.name }}</h3>
          <p class="creator">{{ item.type === 'item' ? '道具' : '剧本' }}</p>
          <p class="desc">{{ item.description || '暂无描述' }}</p>
          <div class="card-footer">
            <span class="stat"><i class="ri-download-line"></i> {{ item.downloads }}</span>
            <span class="rating"><i class="ri-star-fill"></i> {{ item.rating.toFixed(1) }}</span>
          </div>
          <button class="import-btn" @click.stop="viewDetail(item.id)">
            <i class="ri-eye-line"></i> 查看详情
          </button>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="items.length > 0" class="pagination anim-item" style="--delay: 3">
      <button
        class="page-btn"
        :disabled="currentPage === 1"
        @click="changePage(currentPage - 1)"
      >
        <i class="ri-arrow-left-s-line"></i>
        上一页
      </button>
      <span class="page-info">
        第 {{ currentPage }} / {{ Math.ceil(total / 12) }} 页
      </span>
      <button
        class="page-btn"
        :disabled="currentPage >= Math.ceil(total / 12)"
        @click="changePage(currentPage + 1)"
      >
        下一页
        <i class="ri-arrow-right-s-line"></i>
      </button>
    </div>
  </div>
</template>

<style scoped>
.market-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header {
  text-align: center;
  padding: 20px 0;
}

.header-top {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 24px;
  margin-bottom: 20px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.header h1 {
  font-size: 36px;
  color: #3E2723;
  margin: 0;
}

.my-items-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #fff;
  color: #B87333;
  border: 2px solid #B87333;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  position: relative;
  z-index: 10;
}

.my-items-btn:hover {
  background: #FFF8F0;
}

.upload-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.3s;
  position: relative;
  z-index: 10;
}

.upload-btn:hover {
  background: #A66629;
}

.search-box {
  position: relative;
  max-width: 500px;
  margin: 0 auto;
}

.search-box input {
  width: 100%;
  height: 48px;
  border-radius: 24px;
  border: none;
  padding: 0 48px 0 20px;
  font-size: 15px;
  box-shadow: 0 4px 12px rgba(184,115,51,0.15);
}

.search-box i {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 20px;
  color: #B87333;
  cursor: pointer;
}

.filter-bar {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
}

.tag {
  padding: 8px 18px;
  border-radius: 20px;
  background: rgba(255,255,255,0.6);
  cursor: pointer;
  font-size: 14px;
  color: #5D4037;
  border: 1px solid transparent;
}

.tag.active {
  background: #B87333;
  color: #fff;
}

.sort-select {
  padding: 8px 18px;
  border-radius: 20px;
  background: rgba(255,255,255,0.6);
  border: 1px solid transparent;
  font-size: 14px;
  color: #5D4037;
  cursor: pointer;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  min-height: 200px;
}

.loading-state, .empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 60px 20px;
  color: #999;
  font-size: 16px;
}

.card {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 20px rgba(93,64,55,0.05);
  transition: transform 0.3s;
}

.card:hover {
  transform: translateY(-6px);
}

.card-image {
  height: 140px;
  background: linear-gradient(135deg, #D4A373 0%, #8C7B70 100%);
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-icon {
  font-size: 48px;
  color: rgba(255, 255, 255, 0.5);
}

.card-content {
  padding: 20px;
}

.card-content h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 4px;
}

.creator {
  font-size: 13px;
  color: #999;
  margin-bottom: 8px;
}

.desc {
  font-size: 14px;
  color: #666;
  line-height: 1.5;
  margin-bottom: 12px;
}

.tags { display: flex; gap: 6px; margin-bottom: 12px; }
.mini-tag {
  font-size: 12px;
  padding: 3px 8px;
  background: #F5F0EB;
  color: #795548;
  border-radius: 4px;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #999;
  margin-bottom: 12px;
}

.rating { color: #FFB300; }

.import-btn {
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 10px;
  background: #B87333;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.anim-item { opacity: 0; transform: translateY(20px); pointer-events: auto; }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }

/* 确保按钮始终可点击 */
.header-top button {
  pointer-events: auto !important;
  opacity: 1;
}

/* 分页 */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding: 20px 0;
}

.page-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 8px;
  color: #5D4037;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
}

.page-btn:hover:not(:disabled) {
  background: #FFF8F0;
  border-color: #B87333;
  color: #B87333;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 14px;
  color: #5D4037;
  font-weight: 500;
}
</style>
