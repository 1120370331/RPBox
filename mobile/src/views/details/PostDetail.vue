<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { resolveApiUrl } from '@/api/image'
import { useToastStore } from '@shared/stores/toast'
import {
  createPostComment,
  favoritePost,
  getPost,
  likePost,
  listPostComments,
  type Post,
  type PostComment,
  unfavoritePost,
  unlikePost,
} from '@/api/post'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const submitting = ref(false)
const liking = ref(false)
const favoriting = ref(false)
const post = ref<Post | null>(null)
const comments = ref<PostComment[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentText = ref('')
const authorName = ref('')
const authorAvatar = ref('')
const authorNameColor = ref('')
const authorNameBold = ref(false)

const postId = computed(() => Number(route.params.id))
const postCoverUrl = computed(() => {
  if (!post.value) return ''
  if (!post.value.cover_image) return ''
  return resolveApiUrl(post.value.cover_image)
})
const postContentHtml = computed(() => {
  if (!post.value?.content) return ''
  return normalizePostContentHtml(post.value.content)
})

async function loadPostDetail() {
  if (!postId.value) return
  loading.value = true
  try {
    const [postRes, commentRes] = await Promise.all([
      getPost(postId.value),
      listPostComments(postId.value),
    ])
    post.value = postRes.post
    comments.value = commentRes.comments || []
    liked.value = postRes.liked
    favorited.value = postRes.favorited
    authorName.value = postRes.author_name
    authorAvatar.value = postRes.author_avatar || ''
    authorNameColor.value = postRes.author_name_color || ''
    authorNameBold.value = !!postRes.author_name_bold
  } catch (error) {
    toast.error((error as Error)?.message || t('community.actionFailed'))
    console.error('Failed to load post detail', error)
  } finally {
    loading.value = false
  }
}

async function toggleLike() {
  if (!post.value || liking.value) return
  liking.value = true
  try {
    if (liked.value) {
      await unlikePost(post.value.id)
      liked.value = false
      post.value.like_count = Math.max(0, post.value.like_count - 1)
      toast.success(t('community.likeRemoved'))
      return
    }
    await likePost(post.value.id)
    liked.value = true
    post.value.like_count += 1
    toast.success(t('community.likeAdded'))
  } catch (error) {
    toast.error((error as Error)?.message || t('community.actionFailed'))
    console.error('Failed to toggle post like', error)
  } finally {
    liking.value = false
  }
}

async function toggleFavorite() {
  if (!post.value || favoriting.value) return
  favoriting.value = true
  try {
    if (favorited.value) {
      await unfavoritePost(post.value.id)
      favorited.value = false
      toast.success(t('community.favoriteRemoved'))
      return
    }
    await favoritePost(post.value.id)
    favorited.value = true
    toast.success(t('community.favoriteAdded'))
  } catch (error) {
    toast.error((error as Error)?.message || t('community.actionFailed'))
    console.error('Failed to toggle post favorite', error)
  } finally {
    favoriting.value = false
  }
}

async function submitComment() {
  if (!post.value || !commentText.value.trim()) return
  submitting.value = true
  try {
    await createPostComment(post.value.id, commentText.value.trim())
    commentText.value = ''
    await loadPostDetail()
    toast.success(t('community.commentPosted'))
  } catch (error) {
    toast.error((error as Error)?.message || t('community.commentFailed'))
    console.error('Failed to create post comment', error)
  } finally {
    submitting.value = false
  }
}

function formatTime(value: string) {
  const date = new Date(value)
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function normalizePostContentHtml(raw: string) {
  let html = raw.trim()
  if (!html) return ''

  if (!/<[a-z][\s\S]*>/i.test(html) && /&lt;\/?[a-z]/i.test(html)) {
    const decoder = document.createElement('textarea')
    decoder.innerHTML = html
    html = decoder.value
  }

  if (!/<[a-z][\s\S]*>/i.test(html)) {
    return html
  }

  const doc = new DOMParser().parseFromString(html, 'text/html')
  doc.querySelectorAll('script,style,iframe').forEach(node => node.remove())

  doc.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src')
    if (!src) return
    img.setAttribute('src', mapContentUrl(src))
    img.setAttribute('loading', 'lazy')
  })

  doc.querySelectorAll('a').forEach((a) => {
    const href = a.getAttribute('href')
    if (!href) return
    a.setAttribute('href', mapContentUrl(href))
    if (/^https?:\/\//.test(href)) {
      a.setAttribute('target', '_blank')
      a.setAttribute('rel', 'noopener noreferrer')
    }
  })

  return doc.body.innerHTML
}

function mapContentUrl(url: string) {
  if (!url) return url
  if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('data:') || url.startsWith('#')) {
    return url
  }

  if (
    url.startsWith('/archives/') ||
    url.startsWith('/community/') ||
    url.startsWith('/posts/') ||
    url.startsWith('/stories/')
  ) {
    return url
  }

  return resolveApiUrl(url)
}

function handleContentClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const link = target?.closest('a')
  if (!link) return
  const rawHref = link.getAttribute('href') || ''
  if (!rawHref) return
  const href = normalizeAppHref(rawHref)

  if (href.startsWith('/archives/story/')) {
    event.preventDefault()
    const id = href.replace('/archives/story/', '').split('/')[0]
    router.push({ name: 'story-detail', params: { id } })
    return
  }
  if (href.startsWith('/community/post/')) {
    event.preventDefault()
    const id = href.replace('/community/post/', '').split('/')[0]
    router.push({ name: 'post-detail', params: { id } })
    return
  }
  if (href.startsWith('/posts/')) {
    event.preventDefault()
    const id = href.replace('/posts/', '').split('/')[0]
    router.push({ name: 'post-detail', params: { id } })
    return
  }
  if (href.startsWith('/stories/')) {
    event.preventDefault()
    const id = href.replace('/stories/', '').split('/')[0]
    router.push({ name: 'story-detail', params: { id } })
  }
}

function normalizeAppHref(href: string) {
  if (href.startsWith('http://') || href.startsWith('https://')) {
    try {
      const url = new URL(href)
      if (url.origin === window.location.origin) {
        return `${url.pathname}${url.search}${url.hash}`
      }
    } catch {
      return href
    }
  }
  return href
}

watch(postId, () => {
  commentText.value = ''
  loadPostDetail()
})

onMounted(loadPostDetail)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('community.detailTitle') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!post" class="hint">{{ $t('community.empty') }}</div>

      <template v-else>
        <article class="post-main">
          <img v-if="postCoverUrl" :src="postCoverUrl" class="cover" alt="" />
          <h2>{{ post.title }}</h2>
          <div class="author-row">
            <img v-if="authorAvatar" :src="resolveApiUrl(authorAvatar)" class="author-avatar" alt="" />
            <i v-else class="ri-user-3-fill avatar-icon" />
            <span
              :style="{ color: authorNameColor || undefined, fontWeight: authorNameBold ? 'bold' : undefined }"
            >{{ authorName }}</span>
            <time>{{ formatTime(post.created_at) }}</time>
          </div>
          <div class="content" v-html="postContentHtml" @click="handleContentClick" />
        </article>

        <section class="action-row">
          <button :class="{ active: liked }" :disabled="liking" @click="toggleLike">
            <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'" /> {{ post.like_count }}
          </button>
          <button :class="{ active: favorited }" :disabled="favoriting" @click="toggleFavorite">
            <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'" />
            {{ favorited ? $t('common.action.favorited') : $t('common.action.favorite') }}
          </button>
        </section>

        <section class="comment-box">
          <h3>{{ $t('community.comments') }} ({{ comments.length }})</h3>
          <textarea v-model="commentText" :placeholder="$t('community.commentPlaceholder')" rows="3" />
          <button :disabled="submitting || !commentText.trim()" @click="submitComment">
            {{ submitting ? $t('common.action.submitting') : $t('community.submitComment') }}
          </button>
        </section>

        <section v-if="comments.length" class="comment-list">
          <article v-for="comment in comments" :key="comment.id" class="comment-item">
            <header>
              <span
                :style="{
                  color: comment.author_name_color || undefined,
                  fontWeight: comment.author_name_bold ? 'bold' : undefined,
                }"
              >{{ comment.author_name }}</span>
              <time>{{ formatTime(comment.created_at) }}</time>
            </header>
            <p>{{ comment.content }}</p>
          </article>
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>
.post-main {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  margin-bottom: 12px;
}

.cover {
  width: 100%;
  height: 180px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  margin-bottom: 10px;
}

.post-main h2 {
  font-size: 18px;
  line-height: 1.4;
  margin-bottom: 8px;
}

.author-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.author-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-icon {
  font-size: 18px;
}

.author-row time {
  margin-left: auto;
}

.content {
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.content :deep(p) {
  margin: 0 0 10px;
}

.content :deep(a) {
  color: var(--link-color);
  text-decoration: underline;
  word-break: break-all;
}

.content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: var(--radius-sm);
}

.action-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 12px;
}

.action-row button {
  border: 1px solid var(--color-border);
  background: var(--color-card-bg);
  border-radius: var(--radius-sm);
  padding: 10px 0;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-row button.active {
  border-color: var(--color-secondary);
  color: var(--color-secondary);
  background: var(--color-primary-light);
}

.action-row button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.comment-box {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  margin-bottom: 12px;
}

.comment-box h3 {
  font-size: 14px;
  margin-bottom: 8px;
}

.comment-box textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  resize: vertical;
  min-height: 84px;
  background: var(--input-bg);
}

.comment-box button {
  width: 100%;
  margin-top: 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  padding: 10px 0;
  cursor: pointer;
}

.comment-box button:disabled {
  opacity: 0.6;
  cursor: default;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.comment-item {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 10px 12px;
}

.comment-item header {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.comment-item p {
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
