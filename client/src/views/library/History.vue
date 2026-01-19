<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listMyPostLikes, listMyPostViews, type PostWithAuthor, POST_CATEGORIES } from '@/api/post'
import { listMyItemLikes, listMyItemViews, type Item, getImageUrl } from '@/api/item'
import LazyBgImage from '@/components/LazyBgImage.vue'

const router = useRouter()
const mounted = ref(false)
const loading = ref(false)
const recordType = ref<'liked' | 'viewed'>('liked')
const contentType = ref<'posts' | 'items'>('posts')
const posts = ref<PostWithAuthor[]>([])
const items = ref<Item[]>([])

const typeMap = {
  'item': '道具',
  'campaign': '剧本',
  'artwork': '画作'
}

onMounted(() => {
  setTimeout(() => mounted.value = true, 50)
  loadHistory()
})

watch([recordType, contentType], () => {
  loadHistory()
})

async function loadHistory() {
  loading.value = true
  try {
    if (contentType.value === 'posts') {
      const res = recordType.value === 'liked' ? await listMyPostLikes() : await listMyPostViews()
      posts.value = res.posts || []
    } else {
      const res: any = recordType.value === 'liked' ? await listMyItemLikes() : await listMyItemViews()
      if (res.code === 0) {
        items.value = res.data.items || []
      }
    }
  } catch (error) {
    console.error('加载历史记录失败:', error)
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.back()
}

function goToPost(id: number) {
  router.push({ name: 'post-detail', params: { id } })
}

function goToItem(id: number) {
  router.push({ name: 'item-detail', params: { id } })
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : '其他'
}
</script>

<template>
  <div class="library-page" :class="{ 'animate-in': mounted }">
    <div class="header anim-item" style="--delay: 0">
      <div class="header-left">
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-line"></i>
          返回
        </button>
        <div>
          <h1 class="page-title">历史记录</h1>
          <p class="page-subtitle">查看你点赞与浏览过的内容</p>
        </div>
      </div>
      <div class="tab-group">
        <button
          class="tab-btn"
          :class="{ active: recordType === 'liked' }"
          @click="recordType = 'liked'"
        >
          <i class="ri-heart-3-line"></i>
          点赞
        </button>
        <button
          class="tab-btn"
          :class="{ active: recordType === 'viewed' }"
          @click="recordType = 'viewed'"
        >
          <i class="ri-eye-line"></i>
          浏览
        </button>
      </div>
    </div>

    <div class="filter-bar anim-item" style="--delay: 1">
      <button
        class="filter-btn"
        :class="{ active: contentType === 'posts' }"
        @click="contentType = 'posts'"
      >
        帖子
      </button>
      <button
        class="filter-btn"
        :class="{ active: contentType === 'items' }"
        @click="contentType = 'items'"
      >
        作品
      </button>
    </div>

    <div class="content anim-item" style="--delay: 2">
      <div v-if="loading" class="loading-state">加载中...</div>

      <template v-else>
        <div v-if="contentType === 'posts'">
          <div v-if="posts.length === 0" class="empty-state">
            <i class="ri-history-line"></i>
            <p>暂无相关记录</p>
          </div>
          <div v-else class="post-grid">
            <div
              v-for="post in posts"
              :key="post.id"
              class="post-card"
              @click="goToPost(post.id)"
            >
              <div v-if="post.cover_image_url" class="post-cover">
                <img :src="getImageUrl('post-cover', post.id, { w: 480, q: 80 })" alt="" loading="lazy" />
              </div>
              <div class="post-body">
                <div class="post-meta">
                  <span class="category-tag">{{ getCategoryLabel(post.category) }}</span>
                  <span class="author-name">{{ post.author_name }}</span>
                </div>
                <h3 class="post-title">{{ post.title }}</h3>
                <div class="post-stats">
                  <span><i class="ri-eye-line"></i> {{ post.view_count }}</span>
                  <span><i class="ri-heart-3-line"></i> {{ post.like_count }}</span>
                  <span><i class="ri-chat-3-line"></i> {{ post.comment_count }}</span>
                  <span><i class="ri-star-line"></i> {{ post.favorite_count }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else>
          <div v-if="items.length === 0" class="empty-state">
            <i class="ri-history-line"></i>
            <p>暂无相关记录</p>
          </div>
          <div v-else class="item-grid">
            <div
              v-for="item in items"
              :key="item.id"
              class="item-card"
              @click="goToItem(item.id)"
            >
              <LazyBgImage
                class="item-image"
                :src="item.preview_image_url ? getImageUrl('item-preview', item.id, { w: 400, q: 80 }) : undefined"
                fallback-gradient="linear-gradient(135deg, #D4A373 0%, #8C7B70 100%)"
              >
                <div v-if="!item.preview_image_url" class="placeholder-icon">
                  <i class="ri-box-3-line"></i>
                </div>
              </LazyBgImage>
              <div class="item-body">
                <h3 class="item-title">{{ item.name }}</h3>
                <div class="item-meta">
                  <span class="item-type">{{ typeMap[item.type as keyof typeof typeMap] || item.type }}</span>
                  <span class="item-author">{{ item.author_username || '匿名' }}</span>
                </div>
                <div class="item-stats">
                  <span><i class="ri-heart-3-line"></i> {{ item.like_count }}</span>
                  <span><i class="ri-star-line"></i> {{ item.favorite_count }}</span>
                  <span><i class="ri-star-fill"></i> {{ item.rating.toFixed(1) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.library-page {
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
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
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.back-btn:hover {
  background: #F5EFE7;
}

.page-title {
  font-size: 32px;
  color: #4B3621;
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-size: 14px;
  color: #8D7B68;
  margin: 0;
}

.tab-group {
  display: flex;
  gap: 12px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 18px;
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 10px;
  color: #4B3621;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active {
  background: #2C1810;
  border-color: #2C1810;
  color: #fff;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.filter-btn {
  padding: 8px 16px;
  border-radius: 10px;
  border: 1px solid #E5D4C1;
  background: #fff;
  color: #4B3621;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-btn.active {
  background: #B87333;
  border-color: #B87333;
  color: #fff;
}

.content {
  min-height: 320px;
}

.loading-state {
  text-align: center;
  padding: 48px 0;
  color: #8D7B68;
}

.empty-state {
  text-align: center;
  padding: 64px 0;
  color: #8D7B68;
}

.empty-state i {
  font-size: 36px;
  margin-bottom: 12px;
  display: block;
}

.post-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 20px;
}

.post-card {
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 4px 10px rgba(75, 54, 33, 0.06);
}

.post-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(75, 54, 33, 0.12);
}

.post-cover img {
  width: 100%;
  height: 150px;
  object-fit: cover;
  display: block;
}

.post-body {
  padding: 16px;
}

.post-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #8D7B68;
  margin-bottom: 8px;
}

.category-tag {
  padding: 2px 8px;
  border-radius: 12px;
  background: #F6EFE6;
  color: #6B4E36;
  font-weight: 600;
}

.post-title {
  font-size: 16px;
  color: #2C1810;
  margin: 0 0 12px 0;
}

.post-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: #8D7B68;
}

.post-stats i {
  margin-right: 4px;
}

.item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 20px;
}

.item-card {
  background: #fff;
  border: 1px solid #E5D4C1;
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 4px 10px rgba(75, 54, 33, 0.06);
}

.item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(75, 54, 33, 0.12);
}

.item-image {
  height: 160px;
}

.placeholder-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: rgba(255, 255, 255, 0.8);
  font-size: 36px;
}

.item-body {
  padding: 14px 16px 16px;
}

.item-title {
  font-size: 16px;
  color: #2C1810;
  margin: 0 0 10px 0;
}

.item-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #8D7B68;
  margin-bottom: 10px;
}

.item-type {
  padding: 2px 8px;
  border-radius: 12px;
  background: #F6EFE6;
  color: #6B4E36;
  font-weight: 600;
}

.item-stats {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #8D7B68;
}

.item-stats i {
  margin-right: 4px;
}

@media (max-width: 768px) {
  .header {
    flex-direction: column;
    align-items: flex-start;
  }

  .tab-group {
    width: 100%;
  }

  .tab-btn {
    flex: 1;
    justify-content: center;
  }

  .filter-bar {
    width: 100%;
  }

  .filter-btn {
    flex: 1;
  }
}
</style>
