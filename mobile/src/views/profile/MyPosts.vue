<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@shared/stores/user'
import { listPosts, type PostWithAuthor, type ListPostsParams } from '@/api/post'
import { resolveApiUrl } from '@/api/image'

const router = useRouter()
const userStore = useUserStore()
const posts = ref<PostWithAuthor[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = 12

async function loadMyPosts() {
  loading.value = true
  try {
    const params: ListPostsParams & { author_id?: number } = {
      page: currentPage.value,
      page_size: pageSize,
      sort: 'created_at',
      order: 'desc',
    }
    if (userStore.user?.id) (params as any).author_id = userStore.user.id
    const res = await listPosts(params as any)
    posts.value = res.posts || []
    total.value = res.total || 0
  } catch (e) {
    console.error('Failed to load my posts', e)
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const totalPages = () => Math.max(1, Math.ceil(total.value / pageSize))

onMounted(loadMyPosts)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myPosts.title') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="posts.length === 0" class="hint">{{ $t('profile.myPosts.empty') }}</div>
      <div v-else class="post-list">
        <button
          v-for="post in posts"
          :key="post.id"
          class="post-card"
          @click="router.push({ name: 'post-detail', params: { id: post.id } })"
        >
          <div v-if="post.cover_image_url" class="post-cover">
            <img :src="resolveApiUrl(post.cover_image_url)" alt="" loading="lazy" />
          </div>
          <div class="post-info">
            <span v-if="post.category" class="cat">{{ post.category }}</span>
            <h3>{{ post.title }}</h3>
            <div class="meta">
              <span><i class="ri-eye-line" /> {{ post.view_count }}</span>
              <span><i class="ri-heart-line" /> {{ post.like_count }}</span>
              <span>{{ formatDate(post.created_at) }}</span>
            </div>
          </div>
        </button>
      </div>
      <div v-if="total > pageSize" class="pagination">
        <button :disabled="currentPage <= 1" @click="currentPage--; loadMyPosts()">{{ $t('common.pagination.prev') }}</button>
        <span>{{ $t('common.pagination.pageInfo', { current: currentPage, total: totalPages() }) }}</span>
        <button :disabled="currentPage >= totalPages()" @click="currentPage++; loadMyPosts()">{{ $t('common.pagination.next') }}</button>
      </div>
    </div>
  </div>
</template>
