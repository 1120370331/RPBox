<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getPost, likePost, unlikePost, favoritePost, unfavoritePost, deletePost, POST_CATEGORIES } from '@/api/post'
import { listComments, createComment, deleteComment, likeComment, unlikeComment, type CommentWithAuthor } from '@/api/post'
import EmojiPicker from '@/components/EmojiPicker.vue'
import EmoteEditor from '@/components/EmoteEditor.vue'
import ImageViewer from '@/components/ImageViewer.vue'
import { attachImagePreview } from '@/utils/imagePreview'
import { buildNameStyle } from '@/utils/userNameStyle'
import { resolveApiUrl } from '@/api/item'
import { renderEmoteContent } from '@/utils/emote'
import { handleJumpLinkClick, sanitizeJumpLinks, hydrateJumpCardImages } from '@/utils/jumpLink'
import { handleAttachmentClick } from '@/utils/download'
import { useToast } from '@/composables/useToast'
import { useDialog } from '@/composables/useDialog'
import { useEmoteStore } from '@/stores/emote'
import CollectionBanner from '@/components/CollectionBanner.vue'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const toast = useToast()
const dialog = useDialog()
const emoteStore = useEmoteStore()
const mounted = ref(false)
const loading = ref(false)
const submittingComment = ref(false)
const actionLoading = ref(false)

const post = ref<any>(null)
const authorAvatar = ref('')
const comments = ref<CommentWithAuthor[]>([])
const liked = ref(false)
const favorited = ref(false)
const commentContent = ref('')
const currentUserId = ref<number>(0)
const currentUserRole = ref<string>('')

// 评论分页
const commentPage = ref(1)
const commentPageSize = 10
const commentTotal = ref(0)

// 回复功能
const replyingTo = ref<CommentWithAuthor | null>(null)
const replyContent = ref('')
const submittingReply = ref(false)

// Emoji选择器
const showEmojiPicker = ref(false)
const showReplyEmojiPicker = ref(false)
const emojiButtonRef = ref<HTMLElement | null>(null)
const replyEmojiTrigger = ref<HTMLElement | null>(null)
const commentEditorRef = ref<any>(null)
const replyEditorRef = ref<any>(null)

const errorMessage = ref('')
const commentError = ref('')
const articleContentRef = ref<HTMLElement | null>(null)
const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)

// 评论点赞状态
const commentLikes = reactive(new Map<number, boolean>())


// 获取当前用户ID和角色
const userStr = localStorage.getItem('user')
if (userStr) {
  try {
    const user = JSON.parse(userStr)
    currentUserId.value = user.id
    currentUserRole.value = user.role || 'user'
  } catch (e) {
    console.error('解析用户信息失败:', e)
  }
}

const canManagePost = computed(() => {
  if (!post.value) return false
  if (currentUserId.value === post.value.author_id) return true
  return currentUserRole.value === 'moderator' || currentUserRole.value === 'admin'
})

// 检查是否可以删除评论
function canDeleteComment(comment: CommentWithAuthor): boolean {
  const isCommentAuthor = comment.author_id === currentUserId.value
  const isPostAuthor = post.value && post.value.author_id === currentUserId.value
  const isModerator = currentUserRole.value === 'moderator' || currentUserRole.value === 'admin'
  return isCommentAuthor || isPostAuthor || isModerator
}

// 将评论组织成树形结构
interface CommentWithReplies extends CommentWithAuthor {
  replies: CommentWithAuthor[]
  replyToName?: string  // 回复的目标用户名
}

const organizedComments = computed(() => {
  const allComments = comments.value
  // 创建评论ID到评论的映射
  const commentMap = new Map<number, CommentWithAuthor>()
  allComments.forEach(c => commentMap.set(c.id, c))

  // 顶级评论（没有parent_id的）
  const topLevel: CommentWithReplies[] = []
  // 回复（有parent_id的）
  const replies: CommentWithAuthor[] = []

  allComments.forEach(c => {
    if (!c.parent_id) {
      topLevel.push({ ...c, replies: [] })
    } else {
      replies.push(c)
    }
  })

  // 将回复挂载到对应的顶级评论下
  replies.forEach(reply => {
    // 找到父评论
    const parentComment = commentMap.get(reply.parent_id!)
    const replyToName = parentComment?.author_name

    // 找到顶级评论（可能是直接父评论，也可能需要向上查找）
    let topLevelParent = topLevel.find(t => t.id === reply.parent_id)
    if (!topLevelParent) {
      // 父评论可能也是回复，找到其所属的顶级评论
      for (const top of topLevel) {
        if (top.replies.some(r => r.id === reply.parent_id)) {
          topLevelParent = top
          break
        }
      }
    }

    if (topLevelParent) {
      topLevelParent.replies.push({ ...reply, replyToName } as any)
    } else {
      // 如果找不到父评论，作为顶级评论显示
      topLevel.push({ ...reply, replies: [], replyToName } as any)
    }
  })

  return topLevel
})

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await emoteStore.loadPacks()
  await refreshPostData()
})

async function setupArticleImagePreview() {
  await nextTick()
  attachImagePreview(articleContentRef.value, (imageList, index) => {
    viewerImages.value = imageList
    viewerStartIndex.value = index
    showImageViewer.value = true
  }, t('community.detail.viewImage'))
  sanitizeJumpLinks(articleContentRef.value)
  hydrateJumpCardImages(articleContentRef.value)
}

watch(() => post.value?.content, () => {
  if (!post.value?.content) return
  setupArticleImagePreview()
})

async function loadPost() {
  loading.value = true
  errorMessage.value = ''
  try {
    const id = Number(route.params.id)
    if (isNaN(id)) throw new Error(t('community.detail.invalidPostId'))
    const res = await getPost(id)
    post.value = res.post
    post.value.author_name = res.author_name  // author_name 在响应顶层
    post.value.author_name_color = res.author_name_color
    post.value.author_name_bold = res.author_name_bold
    authorAvatar.value = resolveApiUrl(res.author_avatar)
    liked.value = res.liked
    favorited.value = res.favorited
  } catch (error: any) {
    console.error('加载帖子失败:', error)
    errorMessage.value = error.response?.data?.error || error.message || t('community.detail.loadFailed')
    setTimeout(() => router.back(), 2000)
  } finally {
    loading.value = false
  }
}

async function loadComments() {
  try {
    const id = Number(route.params.id)
    const res = await listComments(id)
    comments.value = res.comments || []
    commentLikes.clear()
    comments.value.forEach((comment) => {
      if (comment.liked) {
        commentLikes.set(comment.id, true)
      }
    })
    await scrollToCommentFromRoute()
  } catch (error: any) {
    console.error('加载评论失败:', error)
  }
}

async function refreshPostData() {
  replyingTo.value = null
  commentContent.value = ''
  replyContent.value = ''
  commentError.value = ''
  commentPage.value = 1
  await loadPost()
  await loadComments()
}

async function handleLike() {
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    if (liked.value) {
      await unlikePost(post.value.id)
      liked.value = false
      post.value.like_count--
    } else {
      await likePost(post.value.id)
      liked.value = true
      post.value.like_count++
    }
  } catch (error: any) {
    console.error('点赞失败:', error)
  } finally {
    actionLoading.value = false
  }
}

async function handleFavorite() {
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    if (favorited.value) {
      await unfavoritePost(post.value.id)
      favorited.value = false
      post.value.favorite_count--
    } else {
      await favoritePost(post.value.id)
      favorited.value = true
      post.value.favorite_count++
    }
  } catch (error: any) {
    console.error('收藏失败:', error)
  } finally {
    actionLoading.value = false
  }
}

async function handleComment() {
  if (!commentContent.value.trim()) return
  if (submittingComment.value) return
  submittingComment.value = true
  commentError.value = ''
  try {
    await createComment(post.value.id, commentContent.value)
    commentContent.value = ''
    await loadComments()
    post.value.comment_count++
  } catch (error: any) {
    commentError.value = error.response?.data?.error || t('community.detail.commentFailed')
  } finally {
    submittingComment.value = false
  }
}

// 回复评论
function startReply(comment: CommentWithAuthor) {
  replyingTo.value = comment
  replyContent.value = ''
}

function cancelReply() {
  replyingTo.value = null
  replyContent.value = ''
}

async function submitReply() {
  if (!replyContent.value.trim() || !replyingTo.value) return
  if (submittingReply.value) return
  submittingReply.value = true
  try {
    await createComment(post.value.id, replyContent.value, replyingTo.value.id)
    replyContent.value = ''
    replyingTo.value = null
    await loadComments()
    post.value.comment_count++
  } catch (error: any) {
    console.error('回复失败:', error)
  } finally {
    submittingReply.value = false
  }
}

// Emoji选择处理
function handleEmojiSelect(emoji: string) {
  if (commentEditorRef.value?.insertToken) {
    commentEditorRef.value.insertToken(emoji)
  } else {
    appendEmoteToken(commentContent, emoji)
  }
  showEmojiPicker.value = false
}

function handleReplyEmojiSelect(emoji: string) {
  const editor = Array.isArray(replyEditorRef.value) ? replyEditorRef.value[0] : replyEditorRef.value
  if (editor?.insertToken) {
    editor.insertToken(emoji)
  } else {
    appendEmoteToken(replyContent, emoji)
  }
  showReplyEmojiPicker.value = false
}

function openReplyEmojiPicker(event: MouseEvent) {
  replyEmojiTrigger.value = event.currentTarget as HTMLElement
  showReplyEmojiPicker.value = true
}

function appendEmoteToken(target: { value: string }, token: string) {
  const trimmed = target.value.trimEnd()
  const spacer = trimmed.length > 0 ? ' ' : ''
  target.value = `${trimmed}${spacer}${token} `
}

function renderCommentContent(content: string) {
  return renderEmoteContent(content, emoteStore.emoteMap)
}

// 删除评论
async function handleDeleteComment(comment: CommentWithAuthor) {
  const confirmed = await dialog.confirm({
    title: t('community.detail.deleteCommentTitle'),
    message: t('community.detail.deleteCommentMessage'),
    type: 'warning',
  })
  if (!confirmed) return

  try {
    await deleteComment(post.value.id, comment.id)
    await loadComments()
    if (post.value.comment_count > 0) {
      post.value.comment_count--
    }
  } catch (error: any) {
    console.error('删除评论失败:', error)
    toast.error(t('community.detail.deleteCommentFailed') + (error.response?.data?.error || error.message))
  }
}

// 评论点赞
async function handleCommentLike(comment: CommentWithAuthor) {
  const isLiked = commentLikes.get(comment.id) || false

  try {
    if (isLiked) {
      await unlikeComment(comment.id)
      commentLikes.set(comment.id, false)
      comment.like_count = (comment.like_count || 0) - 1
    } else {
      await likeComment(comment.id)
      commentLikes.set(comment.id, true)
      comment.like_count = (comment.like_count || 0) + 1
    }
  } catch (error: any) {
    console.error('点赞失败:', error)
    toast.error(error?.message || t('community.detail.likeFailed'))
  }
}

// 分页
const totalPages = computed(() => Math.ceil(commentTotal.value / commentPageSize))

function goToCommentPage(page: number) {
  if (page < 1 || page > totalPages.value) return
  commentPage.value = page
  loadComments()
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

function formatCommentTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  if (hours < 1) return t('community.time.justNow')
  if (hours < 24) return t('community.time.hoursAgo', { hours })
  const days = Math.floor(hours / 24)
  if (days < 7) return t('community.time.daysAgo', { days })
  return date.toLocaleDateString('zh-CN')
}

function getCategoryLabel(category: string) {
  const cat = POST_CATEGORIES.find(c => c.value === category)
  return cat ? cat.label : t('community.category.other')
}

function goBack() {
  router.back()
}

function goToEdit() {
  router.push({ name: 'post-edit', params: { id: post.value.id } })
}

function handleArticleClick(event: MouseEvent) {
  // 先处理附件下载
  if (handleAttachmentClick(event)) return

  handleJumpLinkClick(event, router, {
    returnTo: {
      type: 'post',
      path: route.fullPath,
      title: post.value?.title || t('community.post.title'),
    },
  })
}

function getCommentIdFromRoute() {
  const raw = route.query.comment
  if (!raw) return null
  const value = Array.isArray(raw) ? raw[0] : raw
  const id = Number(value)
  return Number.isFinite(id) && id > 0 ? id : null
}

async function scrollToCommentFromRoute() {
  const commentId = getCommentIdFromRoute()
  if (!commentId) return
  await nextTick()
  const target = document.getElementById(`comment-${commentId}`)
  if (!target) return
  target.classList.add('comment-highlight')
  target.scrollIntoView({ behavior: 'smooth', block: 'center' })
  window.setTimeout(() => target.classList.remove('comment-highlight'), 1600)
}

watch(() => route.query.comment, () => {
  scrollToCommentFromRoute()
})

watch(() => route.params.id, async (nextId, prevId) => {
  if (!nextId || nextId === prevId) return
  await refreshPostData()
})

async function handleDelete() {
  const confirmed = await dialog.confirm({
    title: t('community.edit.deleteTitle'),
    message: t('community.edit.deleteMessage'),
    type: 'warning',
  })
  if (!confirmed) return

  try {
    await deletePost(post.value.id)
    toast.success(t('community.edit.deleteSuccess'))
    router.push({ name: 'community' })
  } catch (error) {
    console.error('删除失败:', error)
    toast.error(t('community.edit.deleteFailed'))
  }
}
</script>

<template>
  <div class="post-detail-page" :class="{ 'animate-in': mounted }">
    <div v-if="loading" class="loading">{{ t('community.loading') }}</div>

    <div v-else-if="errorMessage" class="error-message">
      <i class="ri-error-warning-line"></i>
      <p>{{ errorMessage }}</p>
    </div>

    <div v-else-if="post" class="content-layout">
      <!-- 主内容区 -->
      <main class="main-area">
        <!-- 返回按钮 -->
        <div class="nav-bar anim-item" style="--delay: 0">
          <button class="back-btn" @click="goBack">
            <div class="back-icon">
              <i class="ri-arrow-left-s-line"></i>
            </div>
            <span>{{ t('community.detail.back') }}</span>
          </button>
        </div>

        <!-- 文章 -->
        <article class="article-card anim-item" style="--delay: 1">
          <div class="article-decoration"></div>

          <!-- 合集横幅 -->
          <CollectionBanner
            v-if="post"
            type="post"
            :content-id="post.id"
            :is-author="currentUserId === post.author_id"
          />

          <!-- 文章头部：作者 + 操作 -->
          <div class="article-top">
            <div class="author-section">
              <div class="author-avatar">
                <img v-if="authorAvatar" :src="authorAvatar" alt="" />
                <span v-else>{{ post.author_name?.charAt(0) || 'U' }}</span>
              </div>
              <div class="author-info">
                <h4 class="author-name" :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</h4>
                <span class="post-date">{{ formatDate(post.created_at) }}</span>
              </div>
            </div>
            <div class="action-buttons">
              <button class="action-btn" :class="{ active: liked }" @click="handleLike" :disabled="actionLoading">
                <i :class="liked ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                <span>{{ post.like_count }}</span>
              </button>
              <button class="action-btn" :class="{ active: favorited }" @click="handleFavorite" :disabled="actionLoading">
                <i :class="favorited ? 'ri-star-fill' : 'ri-star-line'"></i>
                <span>{{ post.favorite_count }}</span>
              </button>
              <span class="view-count">
                <i class="ri-eye-line"></i>
                {{ post.view_count }}
              </span>
            </div>
          </div>

          <!-- 文章内容 -->
          <div class="article-body">
            <header class="article-header">
              <div class="category-badge">
                <span class="badge-dot"></span>
                <span>{{ getCategoryLabel(post.category) }}</span>
              </div>
              <h1 class="article-title">{{ post.title }}</h1>
            </header>

            <div class="zen-divider"></div>

            <div ref="articleContentRef" class="article-content" v-html="post.content" @click="handleArticleClick"></div>
          </div>

          <!-- 作者操作 -->
          <div v-if="canManagePost" class="owner-actions">
            <button class="owner-btn" @click="goToEdit">
              <i class="ri-edit-line"></i> {{ t('community.action.edit') }}
            </button>
            <button class="owner-btn delete" @click="handleDelete">
              <i class="ri-delete-bin-line"></i> {{ t('community.action.delete') }}
            </button>
          </div>
        </article>

        <!-- 评论区 -->
        <section class="comments-section anim-item" style="--delay: 2">
          <h3 class="comments-title">
            {{ t('community.detail.discussion') }} <span class="comment-badge">{{ post.comment_count }}</span>
          </h3>

          <!-- 评论列表 -->
          <div class="comments-list">
            <div v-for="comment in organizedComments" :key="comment.id" class="comment-item" :id="`comment-${comment.id}`">
              <div class="comment-avatar">
                <img v-if="comment.author_avatar" :src="resolveApiUrl(comment.author_avatar)" alt="" />
                <span v-else>{{ comment.author_name.charAt(0) }}</span>
              </div>
              <div class="comment-body">
                <div class="comment-meta">
                  <span class="comment-author" :style="buildNameStyle(comment.author_name_color, comment.author_name_bold)">{{ comment.author_name }}</span>
                  <button class="like-btn-inline" :class="{ active: commentLikes.get(comment.id) }" type="button" @click.stop="handleCommentLike(comment)">
                    <i :class="commentLikes.get(comment.id) ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                    <span v-if="comment.like_count">{{ comment.like_count }}</span>
                  </button>
                  <span class="comment-time">{{ formatCommentTime(comment.created_at) }}</span>
                </div>
                <div class="comment-text" v-html="renderCommentContent(comment.content)"></div>
                <div class="comment-actions">
                  <button class="reply-btn" @click="startReply(comment)">
                    <i class="ri-reply-line"></i> {{ t('community.action.reply') }}
                  </button>
                  <button v-if="canDeleteComment(comment)" class="delete-btn" @click="handleDeleteComment(comment)">
                    <i class="ri-delete-bin-line"></i> {{ t('community.action.delete') }}
                  </button>
                </div>

                <!-- 回复输入框 -->
                <div v-if="replyingTo?.id === comment.id" class="reply-input-box">
                  <EmoteEditor
                    ref="replyEditorRef"
                    v-model="replyContent"
                    :placeholder="t('community.detail.replyTo', { name: comment.author_name })"
                    :disabled="submittingReply"
                  />
                  <div class="reply-actions">
                    <button class="emoji-btn-small" @click="openReplyEmojiPicker" type="button">
                      <i class="ri-emotion-line"></i>
                    </button>
                    <div class="reply-actions-right">
                      <button class="cancel-btn" @click="cancelReply">{{ t('community.create.cancel') }}</button>
                      <button class="submit-btn" :disabled="submittingReply" @click="submitReply">{{ t('community.action.reply') }}</button>
                    </div>
                  </div>
                </div>

                <div v-if="comment.replies && comment.replies.length > 0" class="replies-list">
                  <div v-for="reply in comment.replies" :key="reply.id" class="reply-item" :id="`comment-${reply.id}`">
                    <div class="reply-avatar">
                      <img v-if="reply.author_avatar" :src="resolveApiUrl(reply.author_avatar)" alt="" />
                      <span v-else>{{ reply.author_name.charAt(0) }}</span>
                    </div>
                    <div class="reply-body">
                      <div class="reply-meta">
                        <span class="reply-author" :style="buildNameStyle(reply.author_name_color, reply.author_name_bold)">{{ reply.author_name }}</span>
                        <span v-if="reply.replyToName" class="reply-to">
                          {{ t('community.detail.replyToLabel') }} <span class="reply-to-name">@{{ reply.replyToName }}</span>
                        </span>
                        <span class="reply-time">{{ formatCommentTime(reply.created_at) }}</span>
                        <button class="like-btn-inline" :class="{ active: commentLikes.get(reply.id) }" type="button" @click.stop="handleCommentLike(reply)">
                          <i :class="commentLikes.get(reply.id) ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                          <span v-if="reply.like_count">{{ reply.like_count }}</span>
                        </button>
                      </div>
                      <div class="reply-text" v-html="renderCommentContent(reply.content)"></div>
                      <div class="comment-actions">
                        <button class="reply-btn" @click="startReply(reply)">
                          <i class="ri-reply-line"></i> {{ t('community.action.reply') }}
                        </button>
                        <button v-if="canDeleteComment(reply)" class="delete-btn" @click="handleDeleteComment(reply)">
                          <i class="ri-delete-bin-line"></i> {{ t('community.action.delete') }}
                        </button>
                      </div>

                      <!-- 回复的回复输入框 -->
                      <div v-if="replyingTo?.id === reply.id" class="reply-input-box">
                        <EmoteEditor
                          ref="replyEditorRef"
                          v-model="replyContent"
                          :placeholder="t('community.detail.replyTo', { name: reply.author_name })"
                          :disabled="submittingReply"
                        />
                        <div class="reply-actions">
                          <button class="emoji-btn-small" @click="openReplyEmojiPicker" type="button">
                            <i class="ri-emotion-line"></i>
                          </button>
                          <div class="reply-actions-right">
                            <button class="cancel-btn" @click="cancelReply">{{ t('community.create.cancel') }}</button>
                            <button class="submit-btn" :disabled="submittingReply" @click="submitReply">{{ t('community.action.reply') }}</button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="organizedComments.length === 0" class="empty-comments">
              {{ t('community.detail.emptyComments') }}
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="totalPages > 1" class="comments-pagination">
            <button
              class="page-btn"
              :disabled="commentPage === 1"
              @click="goToCommentPage(commentPage - 1)"
            >{{ t('community.pagination.prev') }}</button>
            <span class="page-info">{{ commentPage }} / {{ totalPages }}</span>
            <button
              class="page-btn"
              :disabled="commentPage === totalPages"
              @click="goToCommentPage(commentPage + 1)"
            >{{ t('community.pagination.next') }}</button>
          </div>

          <!-- 评论输入（底部） -->
          <div class="comment-input-box">
            <EmoteEditor
              ref="commentEditorRef"
              v-model="commentContent"
              :placeholder="t('community.detail.commentPlaceholder')"
              :disabled="submittingComment"
            />
            <div class="input-footer">
              <button ref="emojiButtonRef" class="emoji-btn" @click="showEmojiPicker = true" type="button">
                <i class="ri-emotion-line"></i>
              </button>
              <button class="post-btn" :disabled="submittingComment" @click="handleComment">
                {{ t('community.detail.postComment') }}
              </button>
            </div>
          </div>
        </section>
      </main>
    </div>

    <!-- Emoji选择器 -->
    <EmojiPicker :show="showEmojiPicker" :trigger-element="emojiButtonRef" @select="handleEmojiSelect" @close="showEmojiPicker = false" />
    <EmojiPicker :show="showReplyEmojiPicker" :trigger-element="replyEmojiTrigger" @select="handleReplyEmojiSelect" @close="showReplyEmojiPicker = false" />

    <ImageViewer
      v-model="showImageViewer"
      :images="viewerImages"
      :start-index="viewerStartIndex"
    />
  </div>
</template>

<style scoped>
.post-detail-page {
  max-width: 1200px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  padding: 80px;
  color: var(--color-text-muted);
  font-size: 16px;
}

.error-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px;
  color: #C53030;
}

.error-message i {
  font-size: 48px;
  margin-bottom: 16px;
}

/* ========== 单栏布局 ========== */
.content-layout {
  max-width: 1000px;
  margin: 0 auto;
}

/* ========== 导航栏 ========== */
.nav-bar {
  margin-bottom: 8px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: color 0.3s;
}

.back-btn:hover {
  color: var(--color-secondary);
}

.back-icon {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-panel-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(var(--shadow-base), 0.04);
  transition: all 0.3s;
}

.back-btn:hover .back-icon {
  border-color: var(--color-secondary);
  background: var(--color-primary-light);
}

.back-icon i {
  font-size: 18px;
}

/* ========== 主内容区 ========== */
.main-area {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

/* ========== 文章卡片 ========== */
.article-card {
  background: var(--color-panel-bg);
  box-shadow: 0 4px 20px -2px rgba(var(--shadow-base), 0.05);
  position: relative;
}

.article-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, transparent, var(--color-secondary), transparent);
  opacity: 0.3;
}

/* ========== 文章头部 ========== */
.article-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 32px;
  border-bottom: 1px solid var(--color-border-light);
}

.author-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.author-avatar {
  width: 48px;
  height: 48px;
  min-width: 48px;
  max-width: 48px;
  min-height: 48px;
  max-height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-accent), var(--color-secondary));
  border: 2px solid var(--color-panel-bg);
  box-shadow: 0 2px 8px rgba(var(--shadow-base), 0.2);
  display: block;
  text-align: center;
  line-height: 44px;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-text-light);
  overflow: hidden;
  flex-shrink: 0;
}

.author-avatar img {
  width: 48px;
  height: 48px;
  object-fit: cover;
  display: block;
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.author-name {
  font-family: 'Merriweather', serif;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main);
  margin: 0;
}

.post-date {
  font-size: 12px;
  color: var(--color-text-muted);
}

/* ========== 操作按钮 ========== */
.action-buttons {
  display: flex;
  align-items: center;
  gap: 16px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  color: var(--color-text-muted);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.action-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.action-btn.active {
  background: var(--color-primary-light);
  border-color: var(--color-secondary);
  color: var(--color-secondary);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn i {
  font-size: 16px;
}

.view-count {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.view-count i {
  font-size: 16px;
}

/* ========== 文章内容区 ========== */
.article-body {
  padding: 32px 48px 48px;
}

.article-header {
  text-align: center;
  margin-bottom: 32px;
}

.category-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background: var(--color-card-bg);
  margin-bottom: 20px;
}

.badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-accent);
}

.category-badge span:last-child {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-main);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.article-title {
  font-family: 'Merriweather', serif;
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text-main);
  line-height: 1.4;
  margin: 0 0 20px 0;
}

.article-meta {
  font-family: 'Merriweather', serif;
  font-style: italic;
  font-size: 14px;
  color: var(--color-text-muted);
}

.zen-divider {
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--color-border), transparent);
  margin: 32px 0;
}

/* 正文内容 */
.article-content {
  font-family: 'Merriweather', serif;
  font-size: 16px;
  line-height: 1.9;
  color: var(--color-primary);
}

.article-content :deep(img) {
  max-width: 100%;
  height: auto;
  display: inline-block;
  border-radius: 4px;
  margin: 0.8em 0.6em;
  vertical-align: middle;
}

.article-content :deep(.image-preview) {
  position: relative;
  display: inline-block;
  max-width: 100%;
  margin: 0.8em 0.6em;
  vertical-align: middle;
  cursor: zoom-in;
}

.article-content :deep(.image-preview img) {
  max-width: 100%;
  height: auto;
  display: inline-block;
  border-radius: 4px;
  margin: 0;
}

.article-content :deep(.image-preview-overlay) {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0,0,0,0.35);
  color: #fff;
  font-size: 14px;
  opacity: 0;
  transition: opacity 0.2s;
  pointer-events: none;
}

.article-content :deep(.image-preview:hover .image-preview-overlay) {
  opacity: 1;
}

.article-content :deep(p) {
  margin-bottom: 1.5em;
}

.article-content :deep(h2),
.article-content :deep(h3) {
  color: var(--color-text-main);
  font-weight: 700;
  margin-top: 2em;
  margin-bottom: 1em;
  padding-left: 16px;
  border-left: 3px solid var(--color-accent);
}

.article-content :deep(blockquote) {
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  padding: 24px;
  margin: 2em 0;
  font-style: italic;
  text-align: center;
  color: var(--color-text-main);
  font-size: 18px;
}

.article-content :deep(strong) {
  color: var(--color-secondary);
}

.article-content :deep(.mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-weight: 600;
  margin: 0 2px;
}

/* ========== 附件卡片 ========== */
.article-content :deep(.attachment-card) {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin: 12px 0;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  transition: all 0.2s;
}

.article-content :deep(.attachment-card:hover) {
  border-color: var(--color-secondary);
  box-shadow: 0 2px 8px rgba(var(--shadow-base), 0.08);
}

.article-content :deep(.attachment-card__icon) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--color-primary-light);
  border-radius: 8px;
  flex-shrink: 0;
}

.article-content :deep(.attachment-card__icon i) {
  font-size: 20px;
  color: var(--color-secondary);
}

.article-content :deep(.attachment-card__info) {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.article-content :deep(.attachment-card__name) {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.article-content :deep(.attachment-card__size) {
  font-size: 12px;
  color: var(--color-text-muted);
}

.article-content :deep(.attachment-card__download) {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  background: var(--color-secondary);
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.2s;
  flex-shrink: 0;
}

.article-content :deep(.attachment-card__download:hover) {
  background: var(--color-secondary-dark, #6B3528);
  transform: translateY(-1px);
}

.article-content :deep(.attachment-card__download i) {
  font-size: 16px;
}

/* ========== 作者操作 ========== */
.owner-actions {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 20px 32px;
  border-top: 1px solid var(--color-border-light);
}

.owner-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-muted);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.3s;
}

.owner-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.owner-btn.delete {
  color: #C44536;
  border-color: rgba(196, 69, 54, 0.3);
  background: rgba(196, 69, 54, 0.05);
}

.owner-btn.delete:hover {
  border-color: #C44536;
  background: rgba(196, 69, 54, 0.1);
}

/* ========== 评论区 ========== */
.comments-section {
  background: var(--color-panel-bg);
  padding: 32px;
  box-shadow: 0 4px 20px -2px rgba(var(--shadow-base), 0.05);
}

.comments-title {
  font-family: 'Merriweather', serif;
  font-size: 20px;
  font-weight: 500;
  color: var(--color-text-main);
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 20px 0;
}

.comment-badge {
  font-family: 'Inter', sans-serif;
  font-size: 13px;
  font-weight: 400;
  color: var(--color-text-muted);
  background: var(--color-card-bg);
  padding: 4px 12px;
  border-radius: 20px;
}

/* 评论输入框 */
.comment-input-box {
  background: var(--color-panel-bg);
  border: 1px solid var(--color-border);
  padding: 16px;
  box-shadow: 0 2px 8px rgba(var(--shadow-base), 0.04);
  transition: box-shadow 0.3s;
}

.comment-input-box:focus-within {
  box-shadow: 0 0 0 3px var(--color-primary-light);
}

.comment-input-box :deep(.emote-editor-input) {
  width: 100%;
  background: transparent;
  border: none;
  outline: none;
  resize: none;
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-primary);
  font-family: inherit;
  min-height: 80px;
}

.comment-input-box :deep(.emote-editor-input)::before {
  color: var(--color-text-muted);
  opacity: 0.6;
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--color-border-light);
  margin-top: 12px;
}

.post-btn {
  background: var(--color-text-main);
  color: var(--color-text-light);
  border: none;
  padding: 8px 20px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: background 0.3s;
}

.post-btn:hover {
  background: var(--color-secondary);
}

.post-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Emoji按钮 */
.emoji-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s;
}

.emoji-btn:hover {
  background: var(--color-card-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.emoji-btn i {
  font-size: 18px;
}

/* 评论列表 */
.comments-list {
  display: flex;
  flex-direction: column;
  gap: 24px;
  margin-top: 24px;
}

.comment-item {
  display: flex;
  gap: 12px;
}

.comment-item.comment-highlight,
.reply-item.comment-highlight {
  background: var(--color-primary-light);
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
  border-radius: 8px;
}

.comment-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-accent), var(--color-secondary));
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-light);
  flex-shrink: 0;
  overflow: hidden;
}

.comment-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-meta {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 6px;
}

.comment-author {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
}

.comment-time {
  font-size: 12px;
  color: var(--color-text-muted);
}

.comment-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-primary);
  margin: 0;
}

.comment-text :deep(.comment-emote),
.reply-text :deep(.comment-emote) {
  width: 64px;
  height: 64px;
  object-fit: contain;
  display: inline-block;
  margin: 4px 6px 4px 0;
  vertical-align: middle;
}

.comment-text :deep(.comment-mention),
.reply-text :deep(.comment-mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-weight: 600;
  margin: 0 2px;
}

.empty-comments {
  text-align: center;
  padding: 40px 16px;
  color: var(--color-text-muted);
  font-size: 14px;
}

/* ========== 回复功能 ========== */
.reply-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color 0.2s;
}

.reply-btn:hover {
  color: var(--color-secondary);
}

.comment-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
}

.like-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color 0.2s;
}

.like-btn:hover {
  color: var(--color-secondary);
}

.like-btn.active {
  color: #DC2626;
}

.like-btn.active i {
  color: #DC2626;
}

.like-btn-inline {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 6px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  font-size: 11px;
  cursor: pointer;
  transition: color 0.2s;
  margin-left: 8px;
}

.like-btn-inline:hover {
  color: var(--color-secondary);
}

.like-btn-inline.active {
  color: #DC2626;
}

.like-btn-inline.active i {
  color: #DC2626;
}

.like-btn-inline i {
  font-size: 13px;
}

.delete-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: none;
  border: none;
  color: #C44536;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.delete-btn:hover {
  color: #DC2626;
  background: rgba(220, 38, 38, 0.05);
}

.reply-input-box {
  margin-top: 12px;
  padding: 12px;
  background: var(--color-card-bg);
  border-radius: 6px;
}

.reply-input-box :deep(.emote-editor-input) {
  width: 100%;
  min-height: 60px;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 13px;
  resize: none;
  outline: none;
}

.reply-input-box :deep(.emote-editor-input:focus) {
  border-color: var(--color-accent);
}

.reply-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}

.reply-actions-right {
  display: flex;
  gap: 8px;
}

.emoji-btn-small {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s;
}

.emoji-btn-small:hover {
  background: var(--color-card-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.emoji-btn-small i {
  font-size: 14px;
}

.cancel-btn {
  padding: 6px 12px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
}

.submit-btn {
  padding: 6px 12px;
  background: var(--color-secondary);
  border: none;
  border-radius: 4px;
  color: var(--color-text-light);
  font-size: 12px;
  cursor: pointer;
}

.submit-btn:disabled {
  opacity: 0.5;
}

/* ========== 子回复列表 ========== */
.replies-list {
  margin-top: 16px;
  padding-left: 12px;
  border-left: 2px solid var(--color-border-light);
}

.reply-item {
  display: flex;
  gap: 10px;
  padding: 12px 0;
}

.reply-item:not(:last-child) {
  border-bottom: 1px solid var(--color-border-light);
}

.reply-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-accent), var(--color-secondary));
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-light);
  flex-shrink: 0;
  overflow: hidden;
}

.reply-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.reply-body {
  flex: 1;
  min-width: 0;
}

.reply-meta {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 4px;
}

.reply-author {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-main);
}

.reply-to {
  font-size: 12px;
  color: var(--color-text-muted);
}

.reply-to-name {
  color: var(--color-secondary);
  font-weight: 500;
}

.reply-time {
  font-size: 11px;
  color: var(--color-text-muted);
}

.reply-text {
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-primary);
  margin: 0;
}

/* ========== 评论分页 ========== */
.comments-pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding: 20px 0;
  border-top: 1px solid var(--color-border-light);
  margin-top: 20px;
}

.comments-pagination .page-btn {
  padding: 6px 14px;
  background: var(--color-panel-bg);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.comments-pagination .page-btn:hover:not(:disabled) {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.comments-pagination .page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  font-size: 13px;
  color: var(--color-text-muted);
}

/* ========== 动画 ========== */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}

@keyframes fadeUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
