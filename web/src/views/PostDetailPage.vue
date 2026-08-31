<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getPost } from '@shared/api/post'
import RCard from '@shared/components/RCard.vue'
import RLoading from '@shared/components/RLoading.vue'
import { sanitizeRichHtml } from '@shared/utils/sanitizeHtml'

const route = useRoute()
const post = ref<any>(null)
const sanitizedPostContent = computed(() => sanitizeRichHtml(post.value?.content || ''))
const loading = ref(true)

onMounted(async () => {
  try {
    const id = Number(route.params.id)
    post.value = await getPost(id)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="post-detail-page">
    <RLoading v-if="loading" />

    <RCard v-else-if="post" class="post-card">
      <h1>{{ post.title }}</h1>
      <div class="post-meta">
        <span>{{ post.author?.username }}</span>
        <span>{{ post.view_count }} 浏览</span>
      </div>
      <div class="post-content" v-html="sanitizedPostContent"></div>
    </RCard>
  </div>
</template>

<style scoped>
.post-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 24px;
}

.post-card {
  padding: 32px;
}

.post-card h1 {
  margin-bottom: 16px;
}

.post-meta {
  display: flex;
  gap: 16px;
  color: var(--text-tertiary);
  font-size: 14px;
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
}

.post-content {
  line-height: 1.8;
}
</style>
