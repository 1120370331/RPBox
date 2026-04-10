<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'
import ImagePreviewDialog from '@/components/ImagePreviewDialog.vue'
import MobileEmojiPicker from '@/components/MobileEmojiPicker.vue'
import UserLevelBadge from '@/components/UserLevelBadge.vue'
import { ensureEmoteMapLoaded, renderTextWithEmotes } from '@/utils/emote'
import { shareRouteLink } from '@/utils/mobileShare'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
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
const userStore = useUserStore()

const loading = ref(false)
const submitting = ref(false)
const liking = ref(false)
const favoriting = ref(false)
const post = ref<Post | null>(null)
const comments = ref<PostComment[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentText = ref('')
const commentInputRef = ref<HTMLTextAreaElement | null>(null)
const emojiPickerOpen = ref(false)
const authorName = ref('')
const authorAvatar = ref('')
const authorNameColor = ref('')
const authorNameBold = ref(false)
const authorForumLevel = ref<number | null>(null)
const authorForumLevelName = ref('')
const authorForumLevelColor = ref('')
const authorForumLevelBold = ref(false)
const imagePreviewOpen = ref(false)
const imagePreviewSrc = ref('')
const emoteVersion = ref(0)

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
const canEdit = computed(() => {
  const currentUserId = userStore.user?.id
  if (!currentUserId || !post.value) return false
  if (post.value.author_id === currentUserId) return true
  return userStore.isModerator
})
const commentPreviewHtml = computed(() => {
  void emoteVersion.value
  return renderTextWithEmotes(commentText.value || '')
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
    authorForumLevel.value = postRes.author_forum_level || null
    authorForumLevelName.value = postRes.author_forum_level_name || ''
    authorForumLevelColor.value = postRes.author_forum_level_color || ''
    authorForumLevelBold.value = !!postRes.author_forum_level_bold
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

async function sharePostLink() {
  if (!Number.isFinite(postId.value) || postId.value <= 0) {
    toast.error(t('community.shareLinkFailed'))
    return
  }

  try {
    await shareRouteLink({
      path: `/posts/${postId.value}`,
      title: post.value?.title || authorName.value || 'RPBox Post',
      text: post.value?.title || authorName.value || '',
      dialogTitle: post.value?.title || authorName.value || 'RPBox Post',
    })
    toast.success(t('community.shareLinkSuccess'))
  } catch (error) {
    toast.error((error as Error)?.message || t('community.shareLinkFailed'))
    console.error('Failed to share post link', error)
  }
}

function appendEmoteToken(target: { value: string }, token: string) {
  const trimmed = target.value.trimEnd()
  const spacer = trimmed.length > 0 ? ' ' : ''
  target.value = `${trimmed}${spacer}${token} `
}

function insertEmoteToken(token: string) {
  const input = commentInputRef.value
  if (!input) {
    appendEmoteToken(commentText, token)
    return
  }

  const current = commentText.value
  const start = input.selectionStart ?? current.length
  const end = input.selectionEnd ?? start
  const before = current.slice(0, start)
  const after = current.slice(end)

  const needsHeadSpace = before.length > 0 && !/\s$/.test(before)
  const needsTailSpace = after.length > 0 && !/^\s/.test(after)
  const insert = `${needsHeadSpace ? ' ' : ''}${token}${needsTailSpace ? ' ' : ''}`
  const cursor = before.length + insert.length

  commentText.value = `${before}${insert}${after}`
  void nextTick(() => {
    input.focus()
    input.setSelectionRange(cursor, cursor)
  })
}

function handleEmojiSelect(token: string) {
  insertEmoteToken(token)
}

function formatTime(value: string) {
  const date = new Date(value)
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function normalizePostContentHtml(raw: string) {
  void emoteVersion.value
  let html = raw.trim()
  if (!html) return ''

  if (!/<[a-z][\s\S]*>/i.test(html) && /&lt;\/?[a-z]/i.test(html)) {
    const decoder = document.createElement('textarea')
    decoder.innerHTML = html
    html = decoder.value
  }

  if (!/<[a-z][\s\S]*>/i.test(html)) {
    return renderTextWithEmotes(html)
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

function renderCommentHtml(content: string) {
  void emoteVersion.value
  return renderTextWithEmotes(content || '')
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
  const image = target?.closest('img')
  if (image) {
    const src = (image as HTMLImageElement).currentSrc || image.getAttribute('src') || ''
    if (src) {
      event.preventDefault()
      imagePreviewSrc.value = src
      imagePreviewOpen.value = true
      return
    }
  }

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
  emojiPickerOpen.value = false
  loadPostDetail()
})

onMounted(async () => {
  await loadPostDetail()
  await ensureEmoteMapLoaded()
  emoteVersion.value += 1
})
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
          <button v-if="postCoverUrl" class="cover-btn" @click="imagePreviewSrc = postCoverUrl; imagePreviewOpen = true">
            <CachedImage :src="postCoverUrl" class="cover" alt="" />
          </button>
          <h2>{{ post.title }}</h2>
          <div class="author-row">
            <img v-if="authorAvatar" :src="resolveApiUrl(authorAvatar)" class="author-avatar" alt="" />
            <i v-else class="ri-user-3-fill avatar-icon" />
            <span class="author-identity">
              <span
                :style="{ color: authorNameColor || undefined, fontWeight: authorNameBold ? 'bold' : undefined }"
              >{{ authorName }}</span>
              <UserLevelBadge
                v-if="authorForumLevel"
                compact
                :level="authorForumLevel"
                :name="authorForumLevelName"
                :color="authorForumLevelColor"
                :bold="authorForumLevelBold"
              />
            </span>
            <time>{{ formatTime(post.created_at) }}</time>
          </div>
          <div class="content" v-html="postContentHtml" @click="handleContentClick" />
        </article>

        <section class="action-row" :class="{ 'with-edit': canEdit }">
          <button :class="{ active: liked }" :disabled="liking" @click="toggleLike">
            <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'" /> {{ post.like_count }}
          </button>
          <button :class="{ active: favorited }" :disabled="favoriting" @click="toggleFavorite">
            <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'" />
            {{ favorited ? $t('common.action.favorited') : $t('common.action.favorite') }}
          </button>
          <button v-if="canEdit" @click="router.push({ name: 'post-edit', params: { id: post.id } })">
            <i class="ri-edit-line" /> {{ $t('community.editPost') }}
          </button>
        </section>

        <section class="share-row single">
          <button class="share-btn primary" @click="sharePostLink">
            <i class="ri-share-forward-line" /> {{ $t('community.shareLink') }}
          </button>
        </section>

        <section class="comment-box">
          <h3>{{ $t('community.comments') }} ({{ comments.length }})</h3>
          <div class="comment-input-wrap">
            <textarea
              ref="commentInputRef"
              v-model="commentText"
              :placeholder="$t('community.commentPlaceholder')"
              rows="3"
            />
            <button class="emoji-trigger" type="button" @click="emojiPickerOpen = true">
              <i class="ri-emotion-line" />
            </button>
          </div>
          <div v-if="commentText.trim()" class="comment-preview" v-html="commentPreviewHtml" />
          <button class="comment-submit" :disabled="submitting || !commentText.trim()" @click="submitComment">
            {{ submitting ? $t('common.action.submitting') : $t('community.submitComment') }}
          </button>
        </section>

        <section v-if="comments.length" class="comment-list">
          <article v-for="comment in comments" :key="comment.id" class="comment-item">
            <header>
              <div class="comment-author">
                <img
                  v-if="comment.author_avatar"
                  :src="resolveApiUrl(comment.author_avatar)"
                  class="comment-avatar"
                  alt=""
                >
                <i v-else class="ri-user-3-fill comment-avatar-fallback" />
                <span class="comment-author-name">
                  <span
                    :style="{
                      color: comment.author_name_color || undefined,
                      fontWeight: comment.author_name_bold ? 'bold' : undefined,
                    }"
                  >{{ comment.author_name }}</span>
                  <UserLevelBadge
                    v-if="comment.author_forum_level"
                    compact
                    :level="comment.author_forum_level"
                    :name="comment.author_forum_level_name"
                    :color="comment.author_forum_level_color"
                    :bold="comment.author_forum_level_bold"
                  />
                </span>
              </div>
              <time>{{ formatTime(comment.created_at) }}</time>
            </header>
            <p v-html="renderCommentHtml(comment.content)" />
          </article>
        </section>
      </template>
    </div>

    <ImagePreviewDialog :open="imagePreviewOpen" :src="imagePreviewSrc" @close="imagePreviewOpen = false" />
    <MobileEmojiPicker
      :open="emojiPickerOpen"
      @close="emojiPickerOpen = false"
      @select="handleEmojiSelect"
    />
  </div>
</template>

<style scoped>
.sub-body {
  padding-bottom: calc(28px + var(--safe-bottom, 0px));
}

.post-main {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 16px;
  margin-bottom: 14px;
}

.cover {
  width: 100%;
  height: min(46vw, 220px);
  object-fit: cover;
  border-radius: var(--radius-sm);
  margin-bottom: 12px;
}

.cover-btn {
  width: 100%;
  border: none;
  padding: 0;
  background: transparent;
}

.post-main h2 {
  font-size: 19px;
  line-height: 1.5;
  margin-bottom: 10px;
}

.author-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 10px;
}

.author-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-icon {
  font-size: 24px;
}

.author-row > span {
  font-size: 15px;
}

.author-identity,
.comment-author-name {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.author-row time {
  margin-left: auto;
  font-size: 12px;
}

.content {
  font-size: 15px;
  line-height: 1.78;
  white-space: pre-wrap;
  word-break: break-word;
}

.content :deep(p) {
  margin: 0 0 12px;
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
  cursor: zoom-in;
}

.action-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 14px;
}

.action-row.with-edit {
  grid-template-columns: 1fr 1fr 1fr;
}

.action-row button {
  border: 1px solid var(--color-border);
  background: var(--color-card-bg);
  border-radius: var(--radius-sm);
  padding: 11px 0;
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

.share-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 14px;
}

.share-row.single {
  grid-template-columns: 1fr;
}

.share-btn {
  min-height: 42px;
  border: 1px solid var(--color-border);
  background: var(--color-card-bg);
  border-radius: var(--radius-sm);
  padding: 0 14px;
  font-size: 13px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.share-btn.primary {
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  border-color: transparent;
}

.comment-box {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 16px;
  margin-bottom: 14px;
}

.comment-box h3 {
  font-size: 15px;
  margin-bottom: 10px;
}

.comment-box textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  resize: vertical;
  min-height: 96px;
  background: var(--input-bg);
}

.comment-input-wrap {
  position: relative;
}

.comment-input-wrap textarea {
  padding-right: 44px;
}

.emoji-trigger {
  position: absolute;
  right: 8px;
  bottom: 10px;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  background: #fff;
  color: var(--color-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
}

.comment-box .comment-submit {
  width: 100%;
  margin-top: 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--btn-primary-text);
  padding: 10px 0;
  cursor: pointer;
}

.comment-box .comment-submit:disabled {
  opacity: 0.6;
  cursor: default;
}

.comment-preview {
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px dashed var(--color-border);
  background: rgba(255, 255, 255, 0.6);
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
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
  padding: 12px 14px;
}

.comment-item header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.comment-author {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.comment-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.comment-avatar-fallback {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

.comment-item p {
  font-size: 14px;
  line-height: 1.68;
  white-space: pre-wrap;
  word-break: break-word;
}

.comment-item p :deep(.inline-emote),
.comment-preview :deep(.inline-emote) {
  width: 52px;
  height: 52px;
  vertical-align: text-bottom;
  margin: 0 2px;
}

.content :deep(.inline-emote) {
  width: 26px;
  height: 26px;
  vertical-align: text-bottom;
  margin: 0 2px;
}

.comment-item p :deep(.inline-mention),
.content :deep(.inline-mention) {
  display: inline-block;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-size: 12px;
}

@media (max-width: 380px) {
  .post-main {
    padding: 14px;
  }

  .post-main h2 {
    font-size: 17px;
  }

  .content {
    font-size: 14px;
  }

  .action-row {
    gap: 8px;
  }
}
</style>
