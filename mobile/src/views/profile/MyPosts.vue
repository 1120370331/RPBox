<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { deletePost, listPosts, type PostWithAuthor } from '@/api/post'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const loading = ref(false)
const deleting = ref(false)
const posts = ref<PostWithAuthor[]>([])
const showDeleteDialog = ref(false)
const deletingPost = ref<PostWithAuthor | null>(null)

const publishedPosts = computed(() => posts.value.filter((post) => post.status === 'published' && post.review_status === 'approved'))
const draftPosts = computed(() => posts.value.filter((post) => post.status === 'draft'))
const pendingPosts = computed(() => posts.value.filter((post) => post.status === 'pending' || post.review_status === 'pending'))

async function loadMyPosts() {
  const authorId = userStore.user?.id
  if (!authorId) return
  loading.value = true
  try {
    const res = await listPosts({
      author_id: authorId,
      status: 'all',
      sort: 'created_at',
      order: 'desc',
      page: 1,
      page_size: 100,
    })
    posts.value = res.posts || []
  } catch (error) {
    console.error('Failed to load my posts', error)
    toast.error((error as Error)?.message || t('profile.myPosts.loadFailed'))
  } finally {
    loading.value = false
  }
}

function formatDate(value: string) {
  const date = new Date(value)
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function statusText(post: PostWithAuthor) {
  if (post.status === 'draft') return t('profile.myPosts.statusDraft')
  if (post.status === 'pending' || post.review_status === 'pending') return t('profile.myPosts.statusPending')
  if (post.review_status === 'rejected') return t('profile.myPosts.statusRejected')
  return t('profile.myPosts.statusPublished')
}

function openDeleteDialog(post: PostWithAuthor) {
  deletingPost.value = post
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingPost.value || deleting.value) return
  deleting.value = true
  try {
    await deletePost(deletingPost.value.id)
    posts.value = posts.value.filter((post) => post.id !== deletingPost.value?.id)
    toast.success(t('profile.myPosts.deleteSuccess'))
    showDeleteDialog.value = false
    deletingPost.value = null
  } catch (error) {
    console.error('Failed to delete post', error)
    toast.error((error as Error)?.message || t('profile.myPosts.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

onMounted(loadMyPosts)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('profile.myPosts.title') }}</h1>
      <button class="add-btn" @click="router.push({ name: 'post-create' })"><i class="ri-add-line" /></button>
    </header>

    <div class="sub-body">
      <div class="stats-row">
        <span>{{ $t('profile.myPosts.statsPublished', { n: publishedPosts.length }) }}</span>
        <span>{{ $t('profile.myPosts.statsPending', { n: pendingPosts.length }) }}</span>
        <span>{{ $t('profile.myPosts.statsDraft', { n: draftPosts.length }) }}</span>
      </div>

      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="posts.length === 0" class="hint">{{ $t('profile.myPosts.empty') }}</div>
      <div v-else class="post-list">
        <article v-for="post in posts" :key="post.id" class="post-card">
          <button class="post-main" @click="router.push({ name: 'post-detail', params: { id: post.id } })">
            <div v-if="post.cover_image_url || post.cover_image" class="post-cover">
              <CachedImage :src="resolveApiUrl(post.cover_image_url || post.cover_image || '')" alt="" />
            </div>
            <div class="post-info">
              <div class="title-row">
                <h3>{{ post.title }}</h3>
                <span class="status-tag">{{ statusText(post) }}</span>
              </div>
              <div class="meta">
                <span><i class="ri-eye-line" /> {{ post.view_count }}</span>
                <span><i class="ri-heart-line" /> {{ post.like_count }}</span>
                <span><i class="ri-chat-3-line" /> {{ post.comment_count }}</span>
                <span>{{ formatDate(post.updated_at) }}</span>
              </div>
            </div>
          </button>
          <div class="actions">
            <button class="action-btn" @click="router.push({ name: 'post-edit', params: { id: post.id } })">
              {{ $t('profile.myPosts.edit') }}
            </button>
            <button class="action-btn danger" @click="openDeleteDialog(post)">
              {{ $t('profile.myPosts.delete') }}
            </button>
          </div>
        </article>
      </div>
    </div>

    <div v-if="showDeleteDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('profile.myPosts.deleteTitle') }}</h3>
        <p>{{ $t('profile.myPosts.deleteMessage') }}</p>
        <div class="dialog-actions">
          <button class="action-btn" @click="showDeleteDialog = false">{{ $t('profile.myPosts.cancel') }}</button>
          <button class="action-btn danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? $t('common.status.loading') : $t('profile.myPosts.confirm') }}
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

.post-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.post-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.post-main {
  width: 100%;
  border: none;
  background: transparent;
  text-align: left;
}

.post-cover {
  width: 100%;
  height: 140px;
  overflow: hidden;
}

.post-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.post-info {
  padding: 10px 12px;
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
  margin-top: 8px;
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
