<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { listPosts } from '@shared/api/post'
import RCard from '@shared/components/RCard.vue'
import RButton from '@shared/components/RButton.vue'
import RLoading from '@shared/components/RLoading.vue'
import REmpty from '@shared/components/REmpty.vue'

interface Post {
  id: number
  title: string
  author: { username: string }
  created_at: string
  view_count: number
  like_count: number
}

const posts = ref<Post[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await listPosts({ page: 1, page_size: 20 })
    posts.value = res.posts || []
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="community-page">
    <div class="page-header">
      <h1>社区</h1>
    </div>

    <RLoading v-if="loading" />

    <REmpty v-else-if="!posts.length" description="暂无帖子" />

    <div v-else class="post-list">
      <RouterLink
        v-for="post in posts"
        :key="post.id"
        :to="`/community/post/${post.id}`"
        class="post-item"
      >
        <RCard>
          <h3 class="post-title">{{ post.title }}</h3>
          <div class="post-meta">
            <span>{{ post.author.username }}</span>
            <span>{{ post.view_count }} 浏览</span>
            <span>{{ post.like_count }} 赞</span>
          </div>
        </RCard>
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.community-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 24px;
}

.page-header {
  margin-bottom: 32px;
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.post-item {
  text-decoration: none;
}

.post-title {
  margin-bottom: 8px;
}

.post-meta {
  display: flex;
  gap: 16px;
  color: var(--text-tertiary);
  font-size: 14px;
}
</style>
