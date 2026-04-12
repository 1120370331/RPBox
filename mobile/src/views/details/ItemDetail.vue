<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'
import ImagePreviewDialog from '@/components/ImagePreviewDialog.vue'
import MobileEmojiPicker from '@/components/MobileEmojiPicker.vue'
import UserLevelBadge from '@/components/UserLevelBadge.vue'
import { ensureEmoteMapLoaded, renderTextWithEmotes } from '@/utils/emote'
import { shareRouteLink, shareTextFile } from '@/utils/mobileShare'
import { useUserStore } from '@shared/stores/user'
import { useToastStore } from '@shared/stores/toast'
import {
  createItemComment,
  downloadItem,
  favoriteItem,
  getItem,
  likeItem,
  listItemComments,
  type Item,
  type ItemAuthor,
  type ItemComment,
  unfavoriteItem,
  unlikeItem,
} from '@/api/item'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const toast = useToastStore()

const loading = ref(false)
const submitting = ref(false)
const item = ref<Item | null>(null)
const author = ref<ItemAuthor | null>(null)
const comments = ref<ItemComment[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentText = ref('')
const commentInputRef = ref<HTMLTextAreaElement | null>(null)
const emojiPickerOpen = ref(false)
const rating = ref(0)
const imagePreviewOpen = ref(false)
const imagePreviewSrc = ref('')
const emoteVersion = ref(0)

const itemId = computed(() => Number(route.params.id))
const exportableImportCode = computed(() => item.value?.import_code?.trim() || item.value?.raw_data?.trim() || '')
const previewUrl = computed(() => {
  if (!item.value) return ''
  if (item.value.preview_image_url) return resolveApiUrl(item.value.preview_image_url)
  return resolveApiUrl(`/api/v1/images/item-preview/${item.value.id}?w=900&q=80`)
})
const itemDescriptionHtml = computed(() => {
  void emoteVersion.value
  return renderTextWithEmotes(item.value?.description || '')
})
const itemDetailContentHtml = computed(() => {
  if (!item.value?.detail_content) return ''
  return normalizeRichContentHtml(item.value.detail_content)
})
const canEdit = computed(() => {
  const currentUserId = userStore.user?.id
  if (!currentUserId || !item.value) return false
  if (item.value.author_id === currentUserId) return true
  return userStore.isModerator
})
const commentPreviewHtml = computed(() => {
  void emoteVersion.value
  return renderTextWithEmotes(commentText.value || '')
})
function renderCommentHtml(content: string) {
  void emoteVersion.value
  return renderTextWithEmotes(content || '')
}

function normalizeRichContentHtml(raw: string) {
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
  doc.querySelectorAll('script,style,iframe').forEach((node) => node.remove())

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

function handleDetailContentClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const image = target?.closest('img')
  if (image) {
    const src = (image as HTMLImageElement).currentSrc || image.getAttribute('src') || ''
    if (src) {
      event.preventDefault()
      openImagePreview(src)
      return
    }
  }

  const link = target?.closest('a')
  if (!link) return
  const rawHref = link.getAttribute('href') || ''
  if (!rawHref) return
  const href = normalizeAppHref(rawHref)

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
  if (href.startsWith('/archives/story/')) {
    event.preventDefault()
    const id = href.replace('/archives/story/', '').split('/')[0]
    router.push({ name: 'story-detail', params: { id } })
    return
  }
  if (href.startsWith('/stories/')) {
    event.preventDefault()
    const id = href.replace('/stories/', '').split('/')[0]
    router.push({ name: 'story-detail', params: { id } })
  }
}

async function loadItemDetail() {
  if (!itemId.value) return
  loading.value = true
  try {
    const [itemRes, commentRes] = await Promise.all([
      getItem(itemId.value),
      listItemComments(itemId.value),
    ])
    item.value = itemRes.item
    author.value = itemRes.author
    liked.value = itemRes.liked
    favorited.value = itemRes.favorited
    comments.value = Array.isArray(commentRes) ? commentRes : ((commentRes as any)?.comments || [])
  } catch (error) {
    console.error('Failed to load item detail', error)
  } finally {
    loading.value = false
  }
}

function buildSharedItemFilename() {
  const rawName = item.value?.name?.trim() || 'RPBox Item'
  const safeName = rawName.replace(/[<>:"/\\|?*\u0000-\u001f]/g, ' ').replace(/\s+/g, ' ').trim()
  return `${safeName || 'RPBox Item'}.txt`
}

async function recordItemDownload() {
  if (!item.value?.id) return
  try {
    await downloadItem(item.value.id)
    await loadItemDetail()
  } catch (error) {
    console.error('Failed to record item download', error)
  }
}

async function toggleLike() {
  if (!item.value) return
  try {
    if (liked.value) {
      await unlikeItem(item.value.id)
      liked.value = false
      item.value.like_count = Math.max(0, item.value.like_count - 1)
      return
    }
    await likeItem(item.value.id)
    liked.value = true
    item.value.like_count += 1
  } catch (error) {
    console.error('Failed to toggle item like', error)
  }
}

async function toggleFavorite() {
  if (!item.value) return
  try {
    if (favorited.value) {
      await unfavoriteItem(item.value.id)
      favorited.value = false
      return
    }
    await favoriteItem(item.value.id)
    favorited.value = true
  } catch (error) {
    console.error('Failed to toggle item favorite', error)
  }
}

async function shareImportCodeFile() {
  if (!exportableImportCode.value) {
    toast.error(t('market.importCodeUnavailable'))
    return
  }

  try {
    await shareTextFile({
      filename: buildSharedItemFilename(),
      content: exportableImportCode.value,
      title: item.value?.name || 'RPBox Item',
      text: item.value?.name || '',
      dialogTitle: item.value?.name || 'RPBox Item',
    })
    toast.success(t('market.shareImportCodeSuccess'))
    void recordItemDownload()
  } catch (error) {
    console.error('Failed to share import code file', error)
    toast.error(t('market.shareImportCodeFailed'))
  }
}

async function shareItemLink() {
  if (!Number.isFinite(itemId.value) || itemId.value <= 0) {
    toast.error(t('market.shareItemLinkFailed'))
    return
  }

  try {
    await shareRouteLink({
      path: `/items/${itemId.value}`,
      title: item.value?.name || 'RPBox Item',
      text: item.value?.name || '',
      dialogTitle: item.value?.name || 'RPBox Item',
    })
    toast.success(t('market.shareItemLinkSuccess'))
  } catch (error) {
    console.error('Failed to share item link', error)
    toast.error(t('market.shareItemLinkFailed'))
  }
}

async function shareItemFile() {
  await shareImportCodeFile()
}

async function submitComment() {
  if (!item.value || !commentText.value.trim()) return
  submitting.value = true
  try {
    await createItemComment(item.value.id, commentText.value.trim(), rating.value)
    commentText.value = ''
    rating.value = 0
    await loadItemDetail()
  } catch (error) {
    console.error('Failed to create item comment', error)
  } finally {
    submitting.value = false
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

function openImagePreview(src: string) {
  if (!src) return
  imagePreviewSrc.value = src
  imagePreviewOpen.value = true
}

onMounted(async () => {
  await loadItemDetail()
  await ensureEmoteMapLoaded()
  emoteVersion.value += 1
})
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('market.detailTitle') }}</h1>
    </header>
    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="!item" class="hint">{{ $t('market.empty') }}</div>

      <template v-else>
        <article class="item-main">
          <button class="preview-btn" @click="openImagePreview(previewUrl)">
            <CachedImage :src="previewUrl" class="preview" alt="" :auth-fetch="true" />
          </button>
          <h2>{{ item.name }}</h2>
          <div v-if="author" class="author-row">
            <span class="author-identity">
              <span :style="{ color: author.name_color || undefined, fontWeight: author.name_bold ? 'bold' : undefined }">
                {{ author.username }}
              </span>
              <UserLevelBadge
                v-if="author.forum_level"
                compact
                :level="author.forum_level"
                :name="author.forum_level_name"
                :color="author.forum_level_color"
                :bold="author.forum_level_bold"
              />
            </span>
            <span class="type-tag">{{ $t('market.typeBadge.' + item.type) }}</span>
          </div>
          <p v-html="itemDescriptionHtml"></p>
          <div
            v-if="item.detail_content"
            class="detail-content"
            v-html="itemDetailContentHtml"
            @click="handleDetailContentClick"
          />
          <div class="stat-row">
            <span><i class="ri-download-line" /> {{ item.downloads }}</span>
            <span>★ {{ item.rating.toFixed(1) }} ({{ item.rating_count }})</span>
          </div>
        </article>

        <section class="action-row" :class="{ 'with-edit': canEdit }">
          <button @click="toggleLike">
            <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'" /> {{ item.like_count }}
          </button>
          <button @click="toggleFavorite">
            <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'" />
            {{ favorited ? $t('common.action.favorited') : $t('common.action.favorite') }}
          </button>
          <button v-if="canEdit" @click="router.push({ name: 'item-edit', params: { id: item.id } })">
            <i class="ri-edit-line" /> {{ $t('market.editItem') }}
          </button>
        </section>

        <section
          v-if="item && item.type !== 'artwork' && ((item.import_code && item.import_code.trim()) || (item.raw_data && item.raw_data.trim()))"
          class="share-row"
        >
          <button class="share-btn" @click="shareItemFile">
            <i class="ri-file-text-line" /> {{ $t('market.shareItemAsFile') }}
          </button>
          <button class="share-btn" @click="shareItemLink">
            <i class="ri-link" /> {{ $t('market.shareItemAsLink') }}
          </button>
        </section>

        <section class="comment-box">
          <h3 class="comment-title">
            <span><i class="ri-message-3-line" /> {{ $t('market.comments') }}</span>
            <em>{{ comments.length }}</em>
          </h3>
          <select v-model.number="rating">
            <option :value="0">{{ $t('market.ratingOptional') }}</option>
            <option :value="5">5</option>
            <option :value="4">4</option>
            <option :value="3">3</option>
            <option :value="2">2</option>
            <option :value="1">1</option>
          </select>
          <div class="comment-input-wrap">
            <textarea
              ref="commentInputRef"
              v-model="commentText"
              :placeholder="$t('market.commentPlaceholder')"
              rows="3"
            />
            <button class="emoji-trigger" type="button" @click="emojiPickerOpen = true">
              <i class="ri-emotion-line" />
            </button>
          </div>
          <div v-if="commentText.trim()" class="comment-preview" v-html="commentPreviewHtml" />
          <button class="comment-submit" :disabled="submitting || !commentText.trim()" @click="submitComment">
            {{ submitting ? $t('common.action.submitting') : $t('market.submitComment') }}
          </button>
        </section>

        <section v-if="comments.length" class="comment-list">
          <article v-for="comment in comments" :key="comment.id" class="comment-item">
            <header>
              <div class="comment-author">
                <img
                  v-if="comment.avatar"
                  :src="resolveApiUrl(comment.avatar)"
                  class="comment-avatar"
                  alt=""
                >
                <i v-else class="ri-user-3-fill comment-avatar-fallback" />
                <span class="comment-author-name">
                  <span :style="{ color: comment.name_color || undefined, fontWeight: comment.name_bold ? 'bold' : undefined }">
                    {{ comment.username }}
                  </span>
                  <UserLevelBadge
                    v-if="comment.forum_level"
                    compact
                    :level="comment.forum_level"
                    :name="comment.forum_level_name"
                    :color="comment.forum_level_color"
                    :bold="comment.forum_level_bold"
                  />
                </span>
              </div>
              <time>{{ formatTime(comment.created_at) }}</time>
            </header>
            <p v-html="renderCommentHtml(comment.content)" />
            <div class="rate" v-if="comment.rating > 0">★ {{ comment.rating }}</div>
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

.item-main {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 16px;
  margin-bottom: 14px;
}

.preview {
  width: 100%;
  height: min(46vw, 220px);
  border-radius: var(--radius-sm);
  object-fit: cover;
  margin-bottom: 12px;
}

.preview-btn {
  width: 100%;
  border: none;
  padding: 0;
  background: transparent;
}

.item-main h2 {
  font-size: 19px;
  margin-bottom: 10px;
}

.author-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.author-identity,
.comment-author-name {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.type-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 8px;
  background: var(--tag-bg);
  color: var(--tag-text);
}

.item-main p {
  font-size: 15px;
  line-height: 1.68;
  color: var(--color-text-secondary);
  margin-bottom: 10px;
}

.detail-content {
  font-size: 15px;
  line-height: 1.72;
  color: var(--color-text-main);
  margin-bottom: 12px;
  word-break: break-word;
}

.detail-content :deep(p) {
  margin: 0 0 12px;
}

.detail-content :deep(h1) {
  font-size: 21px;
  line-height: 1.35;
  margin: 0 0 12px;
}

.detail-content :deep(h2) {
  font-size: 18px;
  line-height: 1.4;
  margin: 14px 0 10px;
}

.detail-content :deep(h3) {
  font-size: 16px;
  line-height: 1.45;
  margin: 12px 0 8px;
}

.detail-content :deep(ul),
.detail-content :deep(ol) {
  padding-left: 20px;
  margin: 0 0 12px;
}

.detail-content :deep(li) {
  margin: 4px 0;
}

.detail-content :deep(blockquote) {
  margin: 0 0 12px;
  padding-left: 12px;
  border-left: 3px solid var(--color-accent);
  color: var(--color-text-secondary);
}

.detail-content :deep(a) {
  color: var(--color-secondary);
  text-decoration: underline;
  word-break: break-all;
}

.detail-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: var(--radius-sm);
  cursor: zoom-in;
}

.detail-content :deep(pre) {
  margin: 0 0 12px;
  padding: 12px;
  border-radius: var(--radius-sm);
  background: #2c1810;
  color: #fbf5ef;
  overflow-x: auto;
}

.detail-content :deep(code) {
  font-size: 13px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--color-text-secondary);
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
  padding: 10px 0;
  font-size: 13px;
  cursor: pointer;
}

.share-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 14px;
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

.comment-box {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 16px;
  margin-bottom: 14px;
}

.comment-box h3 {
  font-size: 14px;
  margin-bottom: 8px;
}
.comment-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.comment-title span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.comment-title i {
  font-size: 15px;
  color: var(--color-secondary);
}
.comment-title em {
  font-style: normal;
  min-width: 24px;
  height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
}

.comment-box select,
.comment-box textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  margin-bottom: 8px;
  background: var(--input-bg);
}

.comment-input-wrap {
  position: relative;
  margin-bottom: 8px;
}

.comment-input-wrap textarea {
  margin-bottom: 0;
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
  margin-bottom: 8px;
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
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 6px;
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
  line-height: 1.66;
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

.item-main p :deep(.inline-emote) {
  width: 26px;
  height: 26px;
  vertical-align: text-bottom;
  margin: 0 2px;
}

.comment-item p :deep(.inline-mention),
.item-main p :deep(.inline-mention) {
  display: inline-block;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-size: 12px;
}

.rate {
  margin-top: 6px;
  color: var(--color-accent);
  font-size: 12px;
}

@media (max-width: 380px) {
  .item-main {
    padding: 14px;
  }

  .item-main h2 {
    font-size: 17px;
  }

  .item-main p {
    font-size: 14px;
  }
}
</style>
