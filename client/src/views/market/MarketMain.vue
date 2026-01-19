<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listItems, type Item, getImageUrl } from '@/api/item'
import { getPresetTags, type Tag } from '@/api/tag'
import LazyBgImage from '@/components/LazyBgImage.vue'

const router = useRouter()
const mounted = ref(false)
const loading = ref(false)
const items = ref<Item[]>([])
const total = ref(0)
const searchText = ref('')
const activeType = ref<'item' | 'campaign' | 'artwork' | ''>('')
const sortBy = ref<'created_at' | 'downloads' | 'rating'>('downloads')
const currentPage = ref(1)
const authorName = ref('')
const itemTags = ref<Tag[]>([])
const activeTagId = ref<number | null>(null)

const typeMap = {
  '': '全部',
  'item': '道具',
  'campaign': '剧本',
  'artwork': '画作'
}

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  loadTags()
  loadItems()
})

// 加载作品标签
async function loadTags() {
  try {
    const res: any = await getPresetTags('item')
    if (res.tags) {
      itemTags.value = res.tags
    }
  } catch (error) {
    console.error('加载标签失败:', error)
  }
}

// 加载作品列表
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

    if (authorName.value) {
      params.author_name = authorName.value
    }

    if (activeTagId.value) {
      params.tag_id = activeTagId.value
    }

    const res: any = await listItems(params)
    if (res.code === 0) {
      items.value = res.data.items || []
      total.value = res.data.total || 0
    }
  } catch (error) {
    console.error('加载作品列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 切换类型
function changeType(type: '' | 'item' | 'campaign' | 'artwork') {
  activeType.value = type
  currentPage.value = 1
  // 切换类型时清除标签筛选（因为标签只对道具和剧本有效）
  if (type === 'artwork') {
    activeTagId.value = null
  }
  loadItems()
}

// 搜索
function handleSearch() {
  currentPage.value = 1
  loadItems()
}

// 切换标签
function changeTag(tagId: number | null) {
  activeTagId.value = tagId
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
        <h1>创意市场</h1>
        <div class="header-actions">
          <button class="my-items-btn" @click="goToMyItems">
            <i class="ri-folder-user-line"></i> 我的作品
          </button>
          <button class="upload-btn" @click="goToUpload">
            <i class="ri-upload-line"></i> 上传作品
          </button>
        </div>
      </div>
      <div class="search-box">
        <input
          v-model="searchText"
          type="text"
          placeholder="搜索作品名称、类型或标签..."
          @keyup.enter="handleSearch"
        />
        <i class="ri-search-line" @click="handleSearch"></i>
      </div>
    </div>

    <!-- 筛选 -->
    <div class="filter-bar anim-item" style="--delay: 1">
      <div class="filter-types">
        <span
          v-for="(label, type) in typeMap"
          :key="type"
          class="tag"
          :class="{ active: activeType === type }"
          @click="changeType(type as any)"
        >{{ label }}</span>
      </div>

      <!-- 道具分类标签筛选（仅对道具和剧本类型显示） -->
      <div class="filter-tags" v-if="itemTags.length > 0 && activeType !== 'artwork'">
        <span class="filter-label">分类：</span>
        <span
          class="tag-item"
          :class="{ active: activeTagId === null }"
          @click="changeTag(null)"
        >全部</span>
        <span
          v-for="tag in itemTags"
          :key="tag.id"
          class="tag-item"
          :class="{ active: activeTagId === tag.id }"
          :style="{ '--tag-color': '#' + tag.color }"
          @click="changeTag(tag.id)"
        >{{ tag.name }}</span>
      </div>

      <div class="filter-controls">
        <input
          v-model="authorName"
          type="text"
          placeholder="按发布者筛选..."
          class="author-input"
          @keyup.enter="handleSearch"
        />
        <select v-model="sortBy" class="sort-select">
          <option value="created_at">最新</option>
          <option value="downloads">最热</option>
          <option value="rating">评分</option>
        </select>
      </div>
    </div>

    <!-- 卡片网格 -->
    <div class="card-grid anim-item" style="--delay: 2">
      <div v-if="loading" class="loading-state">加载中...</div>
      <div v-else-if="items.length === 0" class="empty-state">暂无作品</div>
      <div v-else v-for="item in items" :key="item.id" class="card" @click="viewDetail(item.id)">
        <LazyBgImage
          class="card-image"
          :src="item.preview_image_url ? getImageUrl('item-preview', item.id, { w: 400, q: 80 }) : undefined"
          fallback-gradient="linear-gradient(135deg, #D4A373 0%, #8C7B70 100%)"
        >
          <div v-if="!item.preview_image_url" class="placeholder-icon">
            <i class="ri-box-3-line"></i>
          </div>
        </LazyBgImage>
        <div class="card-content">
          <h3>{{ item.name }}</h3>
          <div class="card-meta">
            <span class="type-badge">{{ typeMap[item.type as keyof typeof typeMap] || item.type }}</span>
            <div class="card-author">
              <span v-if="item.author_avatar" class="author-avatar">
                <img :src="item.author_avatar" alt="" loading="lazy" />
              </span>
              <span v-else class="author-avatar placeholder">
                {{ (item.author_username || 'U').charAt(0) }}
              </span>
              <span class="author-name">{{ item.author_username || '匿名' }}</span>
            </div>
          </div>
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
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
  padding: 16px 24px;
  background: rgba(255,255,255,0.8);
  border-radius: 16px;
  box-shadow: 0 4px 12px rgba(184,115,51,0.1);
}

.filter-types {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-tags {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  width: 100%;
  padding-top: 12px;
  border-top: 1px solid rgba(229, 212, 193, 0.5);
}

.filter-label {
  font-size: 14px;
  color: #5D4037;
  font-weight: 500;
}

.tag-item {
  padding: 6px 14px;
  border-radius: 16px;
  background: rgba(255,255,255,0.6);
  cursor: pointer;
  font-size: 13px;
  color: #5D4037;
  border: 1px solid transparent;
  transition: all 0.3s;
}

.tag-item:hover {
  background: rgba(255,255,255,0.9);
  border-color: var(--tag-color, #B87333);
}

.tag-item.active {
  background: var(--tag-color, #B87333);
  color: #fff;
  border-color: var(--tag-color, #B87333);
}

.author-input {
  padding: 8px 16px;
  border-radius: 20px;
  background: rgba(255,255,255,0.9);
  border: 1px solid #E5D4C1;
  font-size: 14px;
  color: #5D4037;
  min-width: 180px;
  transition: all 0.3s;
}

.author-input:focus {
  outline: none;
  border-color: #B87333;
  box-shadow: 0 0 0 3px rgba(184,115,51,0.1);
}

.author-input::placeholder {
  color: #999;
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
  position: relative;
}

.placeholder-icon {
  font-size: 48px;
  color: rgba(255, 255, 255, 0.5);
}

/* 卡片元信息（类型+作者） */
.card-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

/* 卡片作者信息 */
.card-author {
  display: flex;
  align-items: center;
  gap: 6px;
}

.author-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
}

.author-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.author-avatar.placeholder {
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 10px;
}

.author-name {
  font-size: 12px;
  color: #795548;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-content {
  padding: 20px;
}

.card-content h3 {
  font-size: 18px;
  color: #3E2723;
  margin-bottom: 4px;
}

.type-badge {
  display: inline-block;
  font-size: 12px;
  color: #795548;
  background: #F5EFE7;
  padding: 3px 8px;
  border-radius: 4px;
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
