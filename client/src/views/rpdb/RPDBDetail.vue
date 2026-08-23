<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  addRPDBWorkToList,
  createRPDBComment,
  createRPDBList,
  deleteRPDBComment,
  favoriteRPDBWork,
  getRPDBWork,
  likeRPDBWork,
  listRPDBComments,
  listRPDBLists,
  listRPDBWorkRecommendations,
  unfavoriteRPDBWork,
  unlikeRPDBWork,
  type RPDBComment,
  type RPDBList,
  type RPDBRecommendation,
  type RPDBWork,
} from '@/api/rpdb'
import EmojiPicker from '@/components/EmojiPicker.vue'
import EmoteEditor from '@/components/EmoteEditor.vue'
import CommentReplyBox from '@/components/CommentReplyBox.vue'
import CommentImagePicker from '@/components/CommentImagePicker.vue'
import ImageViewer from '@/components/ImageViewer.vue'
import UserLevelBadge from '@/components/UserLevelBadge.vue'
import UserAvatarPopover from '@/components/UserAvatarPopover.vue'
import SafetyReportDialog from '@/components/SafetyReportDialog.vue'
import RPDBMediaGallery from '@/components/rpdb/RPDBMediaGallery.vue'
import RPDBWorkCard from '@/components/rpdb/RPDBWorkCard.vue'
import RPDBWorkContent from '@/components/rpdb/RPDBWorkContent.vue'
import { createContentReport, createUserBlock, type ReportTargetType } from '@/api/safety'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { useEmoteStore } from '@/stores/emote'
import { useUserStore } from '@/stores/user'
import { renderEmoteContent } from '@/utils/emote'
import { attachImagePreview } from '@/utils/imagePreview'
import { resolveApiUrl } from '@/api/item'
import { buildNameStyle } from '@/utils/userNameStyle'
import { sortRPDBStyleTags } from '@/constants/rpdbStyles'
import { hasTomTomCoordinates } from '@/utils/tomtom'
import { useRPDBOptionLabels } from '@/composables/useRPDBOptionLabels'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const dialog = useDialog()
const emoteStore = useEmoteStore()
const userStore = useUserStore()
const { availabilityLabel, bindTypeLabel, factionLabel } = useRPDBOptionLabels()
const work = ref<RPDBWork>()
const comments = ref<RPDBComment[]>([])
const recommendations = ref<RPDBRecommendation[]>([])
const loading = ref(true)
const comment = ref('')
const commentImageURL = ref('')
const commentImageUploading = ref(false)
const submittingComment = ref(false)
const articleContentRef = ref<HTMLElement | null>(null)
const commentEditorRef = ref<any>(null)
const replyEditorRef = ref<{ insertToken?: (token: string) => void } | null>(null)
const emojiButtonRef = ref<HTMLElement | null>(null)
const replyEmojiTrigger = ref<HTMLElement | null>(null)
const showEmojiPicker = ref(false)
const showReplyEmojiPicker = ref(false)
const replyingTo = ref<RPDBComment | null>(null)
const replyContent = ref('')
const replyImageURL = ref('')
const submittingReply = ref(false)
const showImageViewer = ref(false)
const viewerImages = ref<string[]>([])
const viewerStartIndex = ref(0)
const copiedHomeShareCode = ref(false)
const copiedTransmogShareCode = ref(false)
const FLOATING_TOC_WIDE_VIEWPORT = 2160
const tocCollapsed = ref(typeof window !== 'undefined' && window.innerWidth < FLOATING_TOC_WIDE_VIEWPORT)
const listPickerOpen = ref(false)
const listPickerLoading = ref(false)
const creatingList = ref(false)
const newListName = ref('')
const collectionLists = ref<RPDBList[]>([])
const commentLikes = reactive(new Map<number, boolean>())
const reportDialogOpen = ref(false)
const reportSubmitting = ref(false)
const reportContext = ref<{
  targetType: Extract<ReportTargetType, 'rpdb_work' | 'rpdb_comment'>
  targetId: number
  targetLabel: string
  dialogTitle: string
} | null>(null)

const typeLabel = computed(() => {
  if (work.value?.type === 'item_showcase') return '魔兽物品'
  if (work.value?.type === 'transmog') return '幻化方案'
  return '家宅分享'
})
const homeDetails = computed<Record<string, string>>(() => {
  if (work.value?.type !== 'home_showcase') return {}
  try {
    return JSON.parse(work.value.extra || '{}')
  } catch {
    return {}
  }
})
const homeShareCode = computed(() => String(homeDetails.value.share_code || '').trim())
const transmogShareCode = computed(() => {
  if (work.value?.type !== 'transmog') return ''
  try {
    const details = JSON.parse(work.value.extra || '{}') as Record<string, unknown>
    return String(details.share_code || '').trim()
  } catch {
    return ''
  }
})
const tableOfContents = computed(() => {
  if (!work.value) return []
  const result: Array<{ label: string; target: string }> = [
    {
      label: work.value.type === 'home_showcase' ? '空间故事' : '作品介绍',
      target: 'rpdb-section-overview',
    },
  ]
  if (work.value.transmog_slots?.length) result.push({ label: '幻化部件', target: 'rpdb-section-transmog' })
  if (work.value.guide_steps?.length && work.value.type !== 'home_showcase') {
    result.push({ label: '获取攻略', target: 'rpdb-section-guide' })
  }
  if (work.value.type === 'home_showcase') result.push({ label: '家宅资料', target: 'rpdb-section-home' })
  if (recommendations.value.length) result.push({ label: '相关推荐', target: 'rpdb-section-recommendations' })
  result.push({ label: '玩家讨论', target: 'rpdb-section-discussion' })
  return result
})
const coordinateCount = computed(() => work.value?.guide_steps?.filter(step => (
  hasTomTomCoordinates(step)
)).length || 0)
const styleTags = computed(() => {
  if (work.value?.type === 'home_showcase') return []
  return sortRPDBStyleTags((work.value?.tags || []).filter(tag => tag.name.endsWith('风格')))
})
const canReportWork = computed(() => Boolean(
  work.value
  && userStore.user?.id
  && userStore.user.id !== work.value.author_id,
))
const canEditWork = computed(() => Boolean(
  work.value
  && userStore.user
  && (userStore.user.id === work.value.author_id || userStore.user.role === 'admin'),
))

function canUseCommentSafetyActions(item: RPDBComment) {
  return Boolean(userStore.user?.id && item.author_id !== userStore.user.id)
}

function canDeleteComment(item: RPDBComment) {
  if (!work.value || !userStore.user) return false
  return item.author_id === userStore.user.id
    || work.value.author_id === userStore.user.id
    || userStore.user.role === 'moderator'
    || userStore.user.role === 'admin'
}

function formatCount(value?: number) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

function referenceTypeLabel(value?: string) {
  const labels: Record<string, string> = {
    item: '物品',
    equipment: '装备',
    toy: '玩具',
    quest_item: '任务道具',
    transmog: '幻化',
    furniture: '家具',
  }
  return labels[value || ''] || value || '物品'
}
const organizedComments = computed(() => {
  const commentMap = new Map<number, RPDBComment & { replies?: Array<RPDBComment & { replyToName?: string }> }>()
  const roots: Array<RPDBComment & { replies?: Array<RPDBComment & { replyToName?: string }> }> = []

  comments.value.forEach((item) => {
    commentMap.set(item.id, { ...item, replies: [] })
  })

  comments.value.forEach((item) => {
    const current = commentMap.get(item.id)
    if (!current) return
    if (!item.parent_id) {
      roots.push(current)
      return
    }

    const parent = commentMap.get(item.parent_id)
    if (!parent) {
      roots.push(current)
      return
    }

    let root = parent
    while (root.parent_id && commentMap.get(root.parent_id)) {
      root = commentMap.get(root.parent_id)!
    }
    if (root.id === current.id) {
      roots.push(current)
      return
    }
    root.replies = root.replies || []
    root.replies.push({
      ...current,
      replyToName: parent.author_name || '匿名玩家',
    })
  })

  return roots
})

function qualityClass(quality?: string) {
  const normalized = String(quality || 'common').toLowerCase().replace(/[^a-z0-9_-]/g, '')
  return `quality-${normalized || 'common'}`
}

async function load() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const [detail, discussion, related] = await Promise.all([
      getRPDBWork(id),
      listRPDBComments(id),
      listRPDBWorkRecommendations(id).catch(() => ({ recommendations: [] })),
    ])
    work.value = detail.work
    comments.value = discussion.comments || []
    recommendations.value = related.recommendations || []
    commentLikes.clear()
    comments.value.forEach((item) => {
      if (item.liked) commentLikes.set(item.id, true)
    })
    await nextTick()
    setupArticleImagePreview()
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    loading.value = false
    await scrollToCommentFromRoute()
  }
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

async function toggleLike() {
  if (!work.value) return
  try {
    await (work.value.is_liked ? unlikeRPDBWork(work.value.id) : likeRPDBWork(work.value.id))
    work.value.is_liked = !work.value.is_liked
    work.value.like_count = Math.max(0, work.value.like_count + (work.value.is_liked ? 1 : -1))
    toast.success(work.value.is_liked ? '已点赞' : '已取消点赞')
  } catch (error) {
    toast.error((error as Error).message)
  }
}

async function toggleFavorite() {
  if (!work.value) return
  try {
    await (work.value.is_favorited ? unfavoriteRPDBWork(work.value.id) : favoriteRPDBWork(work.value.id))
    work.value.is_favorited = !work.value.is_favorited
    work.value.favorite_count = Math.max(0, work.value.favorite_count + (work.value.is_favorited ? 1 : -1))
    toast.success(work.value.is_favorited ? '已收藏' : '已取消收藏')
  } catch (error) {
    toast.error((error as Error).message)
  }
}

async function addList() {
  if (!work.value) return
  await openListPicker()
}

async function openListPicker() {
  if (!work.value) return
  listPickerOpen.value = true
  listPickerLoading.value = true
  try {
    const result = await listRPDBLists()
    collectionLists.value = result.lists || []
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    listPickerLoading.value = false
  }
}

async function addToSelectedList(list: RPDBList) {
  if (!work.value) return
  try {
    await addRPDBWorkToList(work.value.id, 'wanted', list.id)
    if (!work.value.in_collection_list) {
      work.value.list_count += 1
    }
    work.value.in_collection_list = true
    listPickerOpen.value = false
    toast.success(`已加入「${list.name}」`)
  } catch (error) {
    toast.error((error as Error).message)
  }
}

async function createListAndAdd() {
  const name = newListName.value.trim()
  if (!work.value || !name) return
  creatingList.value = true
  try {
    const result = await createRPDBList(name)
    const list = result.list
    collectionLists.value = [
      list,
      ...collectionLists.value.filter(item => item.id !== list.id),
    ]
    newListName.value = ''
    await addToSelectedList(list)
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    creatingList.value = false
  }
}

function buildCommentReportLabel(item: RPDBComment) {
  const content = item.content.replace(/\s+/g, ' ').trim()
  const excerpt = content.length > 36 ? `${content.slice(0, 36)}...` : content || '评论'
  return `${item.author_name || '匿名玩家'}：${excerpt}`
}

function openWorkReport() {
  if (!work.value) return
  reportContext.value = {
    targetType: 'rpdb_work',
    targetId: work.value.id,
    targetLabel: work.value.title,
    dialogTitle: '举报 RP 数据库作品',
  }
  reportDialogOpen.value = true
}

function openCommentReport(item: RPDBComment) {
  reportContext.value = {
    targetType: 'rpdb_comment',
    targetId: item.id,
    targetLabel: buildCommentReportLabel(item),
    dialogTitle: '举报评论',
  }
  reportDialogOpen.value = true
}

function closeReportDialog() {
  reportDialogOpen.value = false
  reportContext.value = null
}

async function refreshComments() {
  if (!work.value) return
  const discussion = await listRPDBComments(work.value.id)
  comments.value = discussion.comments || []
  commentLikes.clear()
  comments.value.forEach((item) => {
    if (item.liked) commentLikes.set(item.id, true)
  })
}

async function handleDeleteComment(item: RPDBComment) {
  const confirmed = await dialog.confirm({
    title: '删除评论',
    message: '删除后无法恢复，确认删除这条评论吗？',
    type: 'warning',
    confirmText: '确认删除',
    cancelText: '取消',
  })
  if (!confirmed) return

  try {
    await deleteRPDBComment(item.id)
    if (work.value) {
      work.value.comment_count = Math.max(0, work.value.comment_count - 1)
    }
    await refreshComments()
    toast.success('评论已删除')
  } catch (error) {
    toast.error((error as Error).message || '删除评论失败')
  }
}

async function handleBlockCommentAuthor(item: RPDBComment) {
  const confirmed = await dialog.confirm({
    title: '屏蔽该评论作者',
    message: '屏蔽后该作者的评论会立即从当前页面隐藏，你稍后仍可在设置中取消屏蔽。',
    type: 'warning',
    confirmText: '确认屏蔽',
    cancelText: '取消',
  })
  if (!confirmed) return

  try {
    await createUserBlock(item.author_id, `RP 数据库评论：${buildCommentReportLabel(item)}`)
    await refreshComments()
    toast.success('已屏蔽该作者，相关评论已隐藏')
  } catch (error) {
    toast.error((error as Error).message || '屏蔽作者失败')
  }
}

async function submitSafetyReport(payload: { reason: string; detail: string; hideTarget: boolean; blockAuthor: boolean; submitReport: boolean }) {
  if (!reportContext.value || reportSubmitting.value) return
  reportSubmitting.value = true
  try {
    const context = reportContext.value
    const result = await createContentReport({
      target_type: context.targetType,
      target_id: context.targetId,
      reason: payload.reason,
      detail: payload.detail,
      hide_target: payload.hideTarget,
      block_author: payload.blockAuthor,
      submit_report: payload.submitReport,
    })
    closeReportDialog()
    toast.success(result.message || (payload.submitReport ? '举报已提交，版主会尽快处理' : '已按你的设置完成处理'))
    if (context.targetType === 'rpdb_work' && (payload.hideTarget || payload.blockAuthor)) {
      await router.push('/rpdb')
      return
    }
    if (context.targetType === 'rpdb_comment' && (payload.hideTarget || payload.blockAuthor)) {
      await refreshComments()
    }
  } catch (error) {
    toast.error((error as Error).message || '举报提交失败')
  } finally {
    reportSubmitting.value = false
  }
}

async function copyHomeShareCode() {
  if (!homeShareCode.value) {
    toast.warning('作者暂未提供住宅分享代码')
    return
  }
  try {
    await navigator.clipboard?.writeText(homeShareCode.value)
    copiedHomeShareCode.value = true
    toast.success('住宅分享代码已复制')
    window.setTimeout(() => {
      copiedHomeShareCode.value = false
    }, 1600)
  } catch {
    toast.error('复制失败，请手动选择复制')
  }
}

async function copyTransmogShareCode() {
  if (!transmogShareCode.value) {
    toast.warning('作者暂未提供幻化分享代码')
    return
  }
  try {
    await navigator.clipboard?.writeText(transmogShareCode.value)
    copiedTransmogShareCode.value = true
    toast.success('幻化分享代码已复制')
    window.setTimeout(() => {
      copiedTransmogShareCode.value = false
    }, 1600)
  } catch {
    toast.error('复制失败，请手动选择复制')
  }
}

async function submitComment() {
  if (!work.value || (!comment.value.trim() && !commentImageURL.value)) return
  submittingComment.value = true
  try {
    const hasImage = !!commentImageURL.value
    await createRPDBComment(work.value.id, comment.value.trim(), undefined, commentImageURL.value)
    comment.value = ''
    commentImageURL.value = ''
    await load()
    toast.success(hasImage ? '配图评论已提交审核，通过后将在评论区展示' : '评论已发布')
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    submittingComment.value = false
  }
}

function setReplyEditorRef(instance: { insertToken?: (token: string) => void } | null, commentId: number) {
  if (replyingTo.value?.id === commentId) {
    replyEditorRef.value = instance
  }
}

function startReply(item: RPDBComment) {
  replyingTo.value = item
  replyContent.value = ''
  replyImageURL.value = ''
}

function cancelReply() {
  replyingTo.value = null
  replyContent.value = ''
  replyImageURL.value = ''
  replyEditorRef.value = null
}

async function submitReply() {
  if (!work.value || !replyingTo.value || (!replyContent.value.trim() && !replyImageURL.value)) return
  submittingReply.value = true
  try {
    const hasImage = !!replyImageURL.value
    await createRPDBComment(work.value.id, replyContent.value.trim(), replyingTo.value.id, replyImageURL.value)
    cancelReply()
    await load()
    toast.success(hasImage ? '配图回复已提交审核，通过后将在评论区展示' : '回复已发布')
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    submittingReply.value = false
  }
}

function openImageViewer(images: string[], index: number) {
  viewerImages.value = images
  viewerStartIndex.value = index
  showImageViewer.value = images.length > 0
}

function openCommentImage(src: string) {
  if (!src) return
  openImageViewer([resolveApiUrl(src)], 0)
}

function setupArticleImagePreview() {
  attachImagePreview(articleContentRef.value, openImageViewer, '查看大图')
}

function scrollToSection(target: string) {
  document.getElementById(target)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function goBack() {
  if (route.query.from === 'collection') {
    void router.push('/rpdb/lists')
    return
  }
  void router.push('/rpdb')
}

function openRecommendation(workID: number) {
  void router.push(`/rpdb/${workID}`)
}

function handleEmojiSelect(token: string) {
  commentEditorRef.value?.insertToken?.(token)
  showEmojiPicker.value = false
}

function handleReplyEmojiSelect(token: string) {
  replyEditorRef.value?.insertToken?.(token)
  showReplyEmojiPicker.value = false
}

function openReplyEmojiPicker(event: MouseEvent) {
  replyEmojiTrigger.value = event.currentTarget as HTMLElement
  showReplyEmojiPicker.value = true
}

function renderComment(value: string) {
  return renderEmoteContent(value, emoteStore.emoteMap)
}

function formatCommentTime(value: string) {
  const date = new Date(value)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  if (hours < 1) return '刚刚'
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days} 天前`
  return date.toLocaleDateString('zh-CN')
}

function collapseTocWhenSpaceIsTight() {
  if (window.innerWidth < FLOATING_TOC_WIDE_VIEWPORT) {
    tocCollapsed.value = true
  }
}

watch(() => work.value?.content, async () => {
  await nextTick()
  setupArticleImagePreview()
})

watch(() => route.query.comment, () => {
  void scrollToCommentFromRoute()
})

watch(() => route.params.id, (nextID, previousID) => {
  if (nextID === previousID) return
  void load()
  window.scrollTo?.({ top: 0, behavior: 'smooth' })
})

onMounted(async () => {
  window.addEventListener('resize', collapseTocWhenSpaceIsTight)
  await emoteStore.loadPacks()
  await load()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', collapseTocWhenSpaceIsTight)
})
</script>

<template>
  <div v-if="loading" class="loading">正在加载作品档案...</div>
  <article v-else-if="work" class="detail-page minimal-detail-shell">
    <button class="back" type="button" data-testid="detail-back-button" @click="goBack">
      <i class="ri-arrow-left-line"></i>
      {{ route.query.from === 'collection' ? '返回收集清单' : '返回发现' }}
    </button>

    <section class="detail-hero" data-testid="detail-hero">
      <div class="hero-gallery">
        <RPDBMediaGallery :cover="work.cover_image" :media="work.media" :title="work.title" @open-image="openImageViewer" />
      </div>
      <div class="hero-summary">
        <div class="hero-badges">
          <span>{{ typeLabel }}</span>
          <span v-for="tag in styleTags" :key="tag.id" class="style-badge" :style="{ '--tag-color': `#${tag.color || 'B87333'}` }">{{ tag.name }}</span>
        </div>
        <h1>{{ work.title }}</h1>
        <p class="summary">{{ work.summary }}</p>
        <div class="author">
          <UserAvatarPopover
            class="author-avatar"
            :user-id="work.author_id"
            :avatar-url="work.author_avatar ? resolveApiUrl(work.author_avatar) : ''"
            :username="work.author_name || '匿名贡献者'"
            :name-color="work.author_name_color"
            :name-bold="work.author_name_bold"
            :size="36"
          />
          <span>
            由 {{ work.author_name || '匿名贡献者' }} 发布
            <small>v{{ work.version }}</small>
          </span>
        </div>
        <dl class="hero-metadata">
          <div><dt>{{ work.type === 'home_showcase' ? '开放状态' : '获取状态' }}</dt><dd>{{ availabilityLabel(work.availability_status, work.type) }}</dd></div>
          <div><dt>是否绑定</dt><dd>{{ bindTypeLabel(work.bind_type) }}</dd></div>
          <div><dt>阵营</dt><dd>{{ factionLabel(work.faction) }}</dd></div>
        </dl>
        <dl class="hero-stats" aria-label="作品数据" data-testid="rpdb-detail-metrics">
          <div><dt><i class="ri-eye-line"></i>浏览</dt><dd title="登录用户每日计 1 次">{{ formatCount(work.view_count) }}</dd></div>
          <div><dt><i class="ri-heart-3-line"></i>点赞</dt><dd>{{ formatCount(work.like_count) }}</dd></div>
          <div><dt><i class="ri-bookmark-3-line"></i>收藏</dt><dd>{{ formatCount(work.favorite_count) }}</dd></div>
          <div><dt><i class="ri-list-check-3"></i>清单</dt><dd>{{ formatCount(work.list_count) }}</dd></div>
        </dl>
        <div class="hero-action-area">
          <div class="hero-actions">
            <button type="button" :class="{ active: work.is_liked }" @click="toggleLike">
              <i class="ri-heart-3-line"></i>
              {{ work.is_liked ? '已点赞' : '点赞' }}
            </button>
            <button type="button" :class="{ active: work.is_favorited }" @click="toggleFavorite">
              <i class="ri-bookmark-3-line"></i>
              {{ work.is_favorited ? '已收藏' : '收藏' }}
            </button>
            <button type="button" class="primary" @click="addList">
              <i class="ri-add-circle-line"></i>
              {{ work.in_collection_list ? '已在清单' : '加入清单' }}
            </button>
            <button v-if="canReportWork" type="button" class="report-action" data-testid="rpdb-report-button" @click="openWorkReport">
              <i class="ri-flag-line"></i>
              举报
            </button>
          </div>
          <button
            v-if="work.type === 'home_showcase'"
            type="button"
            class="home-copy-code hero-copy-code"
            data-testid="copy-home-share-code"
            :disabled="!homeShareCode"
            @click="copyHomeShareCode"
          >
            <i class="ri-file-copy-line"></i>
            {{ copiedHomeShareCode ? '已复制住宅分享代码' : '复制住宅分享代码' }}
          </button>
        </div>
      </div>
    </section>

    <aside class="floating-toc-panel" :class="{ 'is-collapsed': tocCollapsed }" data-testid="floating-toc">
      <button
        type="button"
        class="toc-collapse"
        data-testid="floating-toc-collapse"
        :aria-label="tocCollapsed ? '展开悬浮目录' : '收起悬浮目录'"
        :title="tocCollapsed ? '展开悬浮目录' : '收起悬浮目录'"
        @click="tocCollapsed = !tocCollapsed"
      >
        <i :class="tocCollapsed ? 'ri-arrow-left-s-line' : 'ri-arrow-right-s-line'"></i>
      </button>

      <div v-if="!tocCollapsed" class="floating-toc-content" data-testid="floating-toc-content">
        <section>
          <h3>文章目录</h3>
          <nav class="toc">
            <button v-for="(item, index) in tableOfContents" :key="item.target" type="button" :class="{ active: index === 0 }" @click="scrollToSection(item.target)">
              {{ item.label }}
            </button>
          </nav>
        </section>

        <section>
          <h3>快速操作</h3>
          <div class="floating-actions">
            <div class="floating-view" title="登录用户每日计 1 次"><i class="ri-eye-line"></i><span>浏览</span><b>{{ formatCount(work.view_count) }}</b></div>
            <button type="button" :class="{ active: work.is_liked }" data-testid="floating-like-button" @click="toggleLike">
              <i class="ri-heart-3-line"></i>
              <span>{{ work.is_liked ? '已点赞' : '点赞' }}</span>
              <b>{{ work.like_count }}</b>
            </button>
            <button type="button" :class="{ active: work.is_favorited }" data-testid="floating-favorite-button" @click="toggleFavorite">
              <i class="ri-bookmark-3-line"></i>
              <span>{{ work.is_favorited ? '已收藏' : '收藏' }}</span>
              <b>{{ work.favorite_count }}</b>
            </button>
            <button type="button" class="primary" data-testid="floating-list-button" @click="addList">
              <i class="ri-list-check-3"></i>
              <span>{{ work.in_collection_list ? '已在清单' : '加入清单' }}</span>
              <b>{{ work.list_count }}</b>
            </button>
          </div>
        </section>

        <section v-if="work.type !== 'home_showcase'">
          <h3>获取助手</h3>
          <p>{{ work.guide_steps?.length || 0 }} 个步骤，其中 {{ coordinateCount }} 个包含 TomTom 坐标。</p>
          <div class="assistant-actions">
            <button type="button" :disabled="!work.guide_steps?.length" @click="scrollToSection('rpdb-section-guide')"><i class="ri-route-line"></i>跳到攻略</button>
          </div>
        </section>

        <section v-if="work.type === 'transmog'">
          <h3>幻化代码</h3>
          <button
            type="button"
            class="transmog-copy-code"
            data-testid="copy-transmog-share-code"
            :disabled="!transmogShareCode"
            @click="copyTransmogShareCode"
          >
            <i class="ri-file-copy-line"></i>
            {{ copiedTransmogShareCode ? '已复制幻化分享代码' : '复制幻化分享代码' }}
          </button>
        </section>

        <section v-if="work.references?.length">
          <h3>引用对象</h3>
          <component
            :is="item.url ? 'a' : 'div'"
            v-for="item in work.references"
            :key="item.id || item.external_id"
            :href="item.url || undefined"
            :target="item.url ? '_blank' : undefined"
            :rel="item.url ? 'noopener noreferrer' : undefined"
            class="reference-object"
            :class="{ 'has-link': Boolean(item.url) }"
            data-testid="rpdb-reference-object"
          >
            <span class="reference-icon" :class="qualityClass(item.quality)" data-testid="rpdb-reference-icon">
              <img v-if="item.icon" :src="item.icon" :alt="item.name">
              <i v-else class="ri-archive-2-line"></i>
            </span>
            <span class="reference-copy">
              <b data-testid="rpdb-reference-name">{{ item.name }}</b>
              <small>{{ referenceTypeLabel(item.external_type) }}<template v-if="item.acquisition_method || item.source"> · {{ item.acquisition_method || item.source }}</template></small>
              <small v-if="item.description" class="reference-description">{{ item.description }}</small>
            </span>
          </component>
        </section>

      </div>
    </aside>

    <div class="detail-lower" data-testid="detail-lower">
      <main ref="articleContentRef" class="article-sheet">
        <RPDBWorkContent
          :work="work"
          :home-details="homeDetails"
          :transmog-share-code="transmogShareCode"
          :copied-transmog-share-code="copiedTransmogShareCode"
          @copy-transmog-share-code="copyTransmogShareCode"
        />

        <section v-if="recommendations.length" id="rpdb-section-recommendations" class="recommendations-section anim-item" data-testid="rpdb-recommendations">
          <header class="recommendations-heading">
            <div>
              <span>继续探索</span>
              <h3>相关推荐</h3>
            </div>
            <i class="ri-node-tree"></i>
          </header>
          <div class="recommendations-grid">
            <RPDBWorkCard
              v-for="item in recommendations"
              :key="item.id"
              :work="item"
              layout="mini"
              @open="openRecommendation(item.id)"
            />
          </div>
        </section>

        <section id="rpdb-section-discussion" class="comments-section anim-item">
          <h3 class="comments-title">
            玩家讨论 <span class="comment-badge">{{ comments.length }}</span>
          </h3>

          <div class="comments-list">
            <div v-for="item in organizedComments" :id="`comment-${item.id}`" :key="item.id" class="comment-item">
              <UserAvatarPopover
                class="comment-avatar"
                :user-id="item.author_id"
                :avatar-url="item.author_avatar ? resolveApiUrl(item.author_avatar) : ''"
                :username="item.author_name || '匿名玩家'"
                :name-color="item.author_name_color"
                :name-bold="item.author_name_bold"
                :size="32"
                :show-popover="false"
              />
              <div class="comment-body">
                <div class="comment-meta">
                  <span class="comment-author" :style="buildNameStyle(item.author_name_color, item.author_name_bold)">{{ item.author_name || '匿名玩家' }}</span>
                  <UserLevelBadge
                    :level="item.author_forum_level"
                    :name="item.author_forum_level_name"
                    :color="item.author_forum_level_color"
                    :bold="item.author_forum_level_bold"
                    size="xs"
                  />
                  <button class="like-btn-inline" :class="{ active: commentLikes.get(item.id) }" type="button">
                    <i :class="commentLikes.get(item.id) ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                    <span v-if="item.like_count">{{ item.like_count }}</span>
                  </button>
                  <span class="comment-time">{{ formatCommentTime(item.created_at) }}</span>
                </div>
                <div v-if="item.content" class="comment-text" v-html="renderComment(item.content)"></div>
                <button v-if="item.image_url" type="button" class="comment-image" @click="openCommentImage(item.image_url)">
                  <img :src="resolveApiUrl(item.image_url)" alt="评论配图" loading="lazy" />
                </button>
                <div class="comment-actions">
                  <button class="reply-btn" type="button" @click="startReply(item)">
                    <i class="ri-reply-line"></i> 回复
                  </button>
                  <button
                    v-if="canUseCommentSafetyActions(item)"
                    type="button"
                    class="comment-safety-btn"
                    :data-testid="`report-rpdb-comment-${item.id}`"
                    @click="openCommentReport(item)"
                  >
                    <i class="ri-alarm-warning-line"></i> 举报评论
                  </button>
                  <button
                    v-if="canUseCommentSafetyActions(item)"
                    type="button"
                    class="comment-safety-btn danger"
                    :data-testid="`block-rpdb-comment-author-${item.id}`"
                    @click="handleBlockCommentAuthor(item)"
                  >
                    <i class="ri-forbid-2-line"></i> 屏蔽作者
                  </button>
                  <button
                    v-if="canDeleteComment(item)"
                    type="button"
                    class="delete-btn"
                    :data-testid="`delete-rpdb-comment-${item.id}`"
                    @click="handleDeleteComment(item)"
                  >
                    <i class="ri-delete-bin-line"></i> 删除
                  </button>
                </div>

                <CommentReplyBox
                  v-if="replyingTo?.id === item.id"
                  :ref="(instance) => setReplyEditorRef(instance, item.id)"
                  v-model="replyContent"
                  :image-url="replyImageURL"
                  :placeholder="`回复 ${item.author_name || '匿名玩家'}`"
                  :disabled="submittingReply"
                  :auto-focus="true"
                  cancel-label="取消"
                  submit-label="回复"
                  @open-emoji="openReplyEmojiPicker"
                  @update:image-url="replyImageURL = $event"
                  @preview-image="openCommentImage"
                  @cancel="cancelReply"
                  @submit="submitReply"
                />

                <div v-if="item.replies && item.replies.length > 0" class="replies-list">
                  <div v-for="reply in item.replies" :id="`comment-${reply.id}`" :key="reply.id" class="reply-item">
                    <UserAvatarPopover
                      class="reply-avatar"
                      :user-id="reply.author_id"
                      :avatar-url="reply.author_avatar ? resolveApiUrl(reply.author_avatar) : ''"
                      :username="reply.author_name || '匿名玩家'"
                      :name-color="reply.author_name_color"
                      :name-bold="reply.author_name_bold"
                      :size="28"
                      :show-popover="false"
                    />
                    <div class="reply-body">
                      <div class="reply-meta">
                        <span class="reply-author" :style="buildNameStyle(reply.author_name_color, reply.author_name_bold)">{{ reply.author_name || '匿名玩家' }}</span>
                        <UserLevelBadge
                          :level="reply.author_forum_level"
                          :name="reply.author_forum_level_name"
                          :color="reply.author_forum_level_color"
                          :bold="reply.author_forum_level_bold"
                          size="xs"
                        />
                        <span v-if="reply.replyToName" class="reply-to">
                          回复 <span class="reply-to-name">@{{ reply.replyToName }}</span>
                        </span>
                        <span class="reply-time">{{ formatCommentTime(reply.created_at) }}</span>
                        <button class="like-btn-inline" :class="{ active: commentLikes.get(reply.id) }" type="button">
                          <i :class="commentLikes.get(reply.id) ? 'ri-heart-fill' : 'ri-heart-line'"></i>
                          <span v-if="reply.like_count">{{ reply.like_count }}</span>
                        </button>
                      </div>
                      <div v-if="reply.content" class="reply-text" v-html="renderComment(reply.content)"></div>
                      <button v-if="reply.image_url" type="button" class="comment-image reply-image" @click="openCommentImage(reply.image_url)">
                        <img :src="resolveApiUrl(reply.image_url)" alt="回复配图" loading="lazy" />
                      </button>
                      <div class="comment-actions">
                        <button class="reply-btn" type="button" @click="startReply(reply)">
                          <i class="ri-reply-line"></i> 回复
                        </button>
                        <button
                          v-if="canUseCommentSafetyActions(reply)"
                          type="button"
                          class="comment-safety-btn"
                          :data-testid="`report-rpdb-comment-${reply.id}`"
                          @click="openCommentReport(reply)"
                        >
                          <i class="ri-alarm-warning-line"></i> 举报评论
                        </button>
                        <button
                          v-if="canUseCommentSafetyActions(reply)"
                          type="button"
                          class="comment-safety-btn danger"
                          :data-testid="`block-rpdb-comment-author-${reply.id}`"
                          @click="handleBlockCommentAuthor(reply)"
                        >
                          <i class="ri-forbid-2-line"></i> 屏蔽作者
                        </button>
                        <button
                          v-if="canDeleteComment(reply)"
                          type="button"
                          class="delete-btn"
                          :data-testid="`delete-rpdb-comment-${reply.id}`"
                          @click="handleDeleteComment(reply)"
                        >
                          <i class="ri-delete-bin-line"></i> 删除
                        </button>
                      </div>

                      <CommentReplyBox
                        v-if="replyingTo?.id === reply.id"
                        :ref="(instance) => setReplyEditorRef(instance, reply.id)"
                        v-model="replyContent"
                        :image-url="replyImageURL"
                        :placeholder="`回复 ${reply.author_name || '匿名玩家'}`"
                        :disabled="submittingReply"
                        :auto-focus="true"
                        cancel-label="取消"
                        submit-label="回复"
                        @open-emoji="openReplyEmojiPicker"
                        @update:image-url="replyImageURL = $event"
                        @preview-image="openCommentImage"
                        @cancel="cancelReply"
                        @submit="submitReply"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="organizedComments.length === 0" class="empty-comments">还没有讨论，欢迎补充第一条使用经验。</div>
          </div>

          <div class="comment-input-box">
            <EmoteEditor ref="commentEditorRef" v-model="comment" placeholder="补充使用经验、版本变化或替代获取方式" :disabled="submittingComment" />
            <CommentImagePicker
              v-model="commentImageURL"
              :disabled="submittingComment"
              @update:uploading="commentImageUploading = $event"
              @preview="openCommentImage"
            />
            <div class="input-footer">
              <button ref="emojiButtonRef" class="emoji-btn" type="button" @click="showEmojiPicker = true">
                <i class="ri-emotion-line"></i>
              </button>
              <button class="post-btn" type="button" :disabled="submittingComment || commentImageUploading || (!comment.trim() && !commentImageURL)" @click="submitComment">
                发表评论
              </button>
            </div>
          </div>
        </section>

        <footer v-if="canEditWork" class="work-edit-footer" data-testid="rpdb-edit-footer">
          <router-link :to="`/rpdb/${work.id}/edit`" data-testid="rpdb-edit-button">
            <i class="ri-edit-line"></i>
            编辑帖子
          </router-link>
        </footer>
      </main>

    </div>

    <EmojiPicker :show="showEmojiPicker" :trigger-element="emojiButtonRef" @select="handleEmojiSelect" @close="showEmojiPicker = false" />
    <EmojiPicker :show="showReplyEmojiPicker" :trigger-element="replyEmojiTrigger" @select="handleReplyEmojiSelect" @close="showReplyEmojiPicker = false" />
    <ImageViewer v-model="showImageViewer" :images="viewerImages" :start-index="viewerStartIndex" />
    <SafetyReportDialog
      :visible="reportDialogOpen"
      :submitting="reportSubmitting"
      :title="reportContext?.dialogTitle"
      :target-label="reportContext?.targetLabel"
      :target-type="reportContext?.targetType"
      @close="closeReportDialog"
      @submit="submitSafetyReport"
    />

    <Teleport to="body">
      <div v-if="listPickerOpen" class="list-picker-mask" data-testid="rpdb-list-picker" @click.self="listPickerOpen = false">
        <section class="list-picker-dialog">
          <header>
            <div>
              <span>收集清单</span>
              <h2>添加到收集清单</h2>
            </div>
            <button type="button" aria-label="关闭清单选择" @click="listPickerOpen = false"><i class="ri-close-line"></i></button>
          </header>
          <div v-if="listPickerLoading" class="list-picker-state">
            <i class="ri-loader-4-line spin"></i>
            <span>正在加载清单</span>
          </div>
          <div v-else class="list-picker-body">
            <form class="list-picker-create" data-testid="rpdb-list-picker-create" @submit.prevent="createListAndAdd">
              <label>
                <span>新建收集清单</span>
                <input v-model="newListName" data-testid="rpdb-list-picker-create-input" maxlength="128" placeholder="例如：夜巡道具清单">
              </label>
              <button type="submit" data-testid="rpdb-list-picker-create-button" :disabled="creatingList || !newListName.trim()">
                <i :class="creatingList ? 'ri-loader-4-line spin' : 'ri-add-line'"></i>
                {{ creatingList ? '创建中' : '创建并加入' }}
              </button>
            </form>

            <div v-if="collectionLists.length" class="list-picker-options">
              <button
                v-for="list in collectionLists"
                :key="list.id"
                type="button"
                data-testid="rpdb-list-picker-option"
                @click="addToSelectedList(list)"
              >
                <span>
                  <b>{{ list.name }}</b>
                  <small>{{ list.description || '个人收集清单' }}</small>
                </span>
                <em>{{ list.item_count }} 项</em>
              </button>
            </div>
            <div v-else class="list-picker-state">
              <i class="ri-list-check-3"></i>
              <span>还没有清单，可以直接在上方创建。</span>
            </div>
          </div>
          <footer>
            <router-link to="/rpdb/lists"><i class="ri-list-settings-line"></i>管理清单</router-link>
          </footer>
        </section>
      </div>
    </Teleport>
  </article>
</template>

<style scoped>
.detail-page{max-width:1380px;margin:auto;color:var(--color-text-main)}
.minimal-detail-shell{--rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 88%,#fff 12%);--rpdb-muted:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%);--rpdb-line:color-mix(in srgb,var(--color-border) 72%,transparent);--rpdb-soft:color-mix(in srgb,var(--color-accent) 8%,transparent)}
.loading{padding:80px;text-align:center;color:var(--color-text-secondary)}
.back{display:inline-flex;align-items:center;gap:6px;margin-bottom:14px;padding:0;border:0;background:none;color:var(--color-accent)}
.detail-hero{display:grid;grid-template-columns:minmax(0,1.38fr) minmax(340px,.62fr);overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.hero-gallery{min-width:0;padding:10px;border-right:1px solid var(--rpdb-line);background:#1b1511}
.hero-gallery :deep(.stage){min-height:370px;border:0;border-radius:8px}
.hero-summary{display:flex;min-width:0;flex-direction:column;padding:26px}
.hero-badges{display:flex;align-items:center;gap:8px}
.hero-badges>span{padding:5px 9px;border-radius:999px;background:var(--rpdb-soft);color:var(--color-accent);font-size:11px;font-weight:800}.style-badge{border:1px solid color-mix(in srgb,var(--tag-color) 55%,var(--rpdb-line));background:color-mix(in srgb,var(--tag-color) 12%,transparent)!important;color:var(--color-text-main)!important}
.hero-summary h1{margin:16px 0 9px;color:var(--color-text-main);font:700 34px/1.2 system-ui,'Microsoft YaHei',sans-serif}
.summary{margin:0;color:var(--color-text-secondary);font-size:14px;line-height:1.85}
.author{display:flex;align-items:center;gap:10px;margin:20px 0;color:var(--color-text-main)}
.author-avatar{position:relative;display:grid;width:36px;height:36px;overflow:hidden;place-items:center;border-radius:50%;background:var(--color-secondary);color:#fff;font-weight:800}.author-avatar img{position:absolute;inset:0;width:100%;height:100%;object-fit:cover}
.author>span:last-child{display:flex;flex-direction:column}
.author small{margin-top:4px;color:var(--color-text-secondary)}
.hero-metadata{display:grid;grid-template-columns:1fr 1fr;gap:0 18px;margin:0;border-top:1px solid var(--rpdb-line)}
.hero-metadata div{display:flex;justify-content:space-between;gap:10px;padding:10px 0;border-bottom:1px solid var(--rpdb-line)}
.hero-metadata dt{color:var(--color-text-secondary)}
.hero-metadata dd{margin:0;text-align:right}
.hero-stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));margin:14px 0 0;padding:11px 0;border-top:1px solid var(--rpdb-line);border-bottom:1px solid var(--rpdb-line)}.hero-stats div{display:grid;justify-items:center;gap:4px;border-right:1px solid var(--rpdb-line)}.hero-stats div:last-child{border-right:0}.hero-stats dt{display:inline-flex;align-items:center;gap:4px;color:var(--color-text-secondary);font-size:10px}.hero-stats dt i{color:var(--color-accent);font-size:13px}.hero-stats dd{margin:0;color:var(--color-text-main);font-size:15px;font-weight:800;font-variant-numeric:tabular-nums}
.hero-action-area{display:grid;gap:10px;margin-top:auto;padding-top:20px}
.hero-actions{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:8px}
.hero-actions button,.assistant-actions button,.verify-actions button,.home-copy-code,.transmog-copy-code,.floating-actions button{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:36px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main)}
.hero-actions button{min-width:0;padding:0 8px;font-size:12px;font-weight:800;white-space:nowrap}
.hero-actions button.active,.hero-actions .primary,.floating-actions button.active,.floating-actions button.primary{border-color:var(--color-accent);background:var(--color-accent);color:#fff}
.hero-actions .report-action{color:var(--color-text-secondary)}
.hero-actions .report-action:hover{border-color:var(--color-danger,#b83232);color:var(--color-danger,#b83232)}
.detail-lower{display:block;margin-top:14px}
.article-sheet,.floating-toc-content{overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.comments-section{padding:32px;border-top:1px solid var(--rpdb-line);background:var(--color-panel-bg);box-shadow:0 4px 20px -2px rgba(var(--shadow-base),.05)}
.recommendations-section{padding:22px 24px;border-top:1px solid var(--rpdb-line);background:color-mix(in srgb,var(--color-panel-bg) 82%,var(--color-main-bg))}
.recommendations-heading{display:flex;align-items:center;justify-content:space-between;margin-bottom:12px}.recommendations-heading>div{display:grid;gap:2px}.recommendations-heading span{color:var(--color-accent);font-size:9px;font-weight:900}.recommendations-heading h3{margin:0;color:var(--color-text-main);font:500 17px/1.35 Merriweather,serif}.recommendations-heading>i{color:var(--icon-color);font-size:20px}.recommendations-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:9px}
.comments-title{display:flex;align-items:center;gap:10px;margin:0 0 20px;color:var(--color-text-main);font:500 20px/1.35 Merriweather,serif}
.comment-badge{padding:4px 12px;border-radius:20px;background:var(--color-card-bg);color:var(--color-text-muted);font:400 13px/1 Inter,sans-serif}
.comment-input-box{margin-top:24px;padding:16px;border:1px solid var(--color-border);background:var(--color-panel-bg);box-shadow:0 2px 8px rgba(var(--shadow-base),.04);transition:box-shadow .3s}
.comment-input-box:focus-within{box-shadow:0 0 0 3px var(--color-primary-light)}
.comment-input-box :deep(.emote-editor-input){width:100%;min-height:80px;border:0;outline:0;background:transparent;color:var(--color-primary);font:inherit;font-size:14px;line-height:1.6;resize:none}
.comment-input-box :deep(.emote-editor-input)::before{color:var(--color-text-muted);opacity:.6}
.comment-input-box>.comment-image-picker{margin-top:12px}
.input-footer{display:flex;align-items:center;justify-content:space-between;margin-top:12px;padding-top:12px;border-top:1px solid var(--color-border-light)}
.post-btn{padding:8px 20px;border:0;background:var(--color-text-main);color:var(--btn-primary-text,var(--color-text-light));font-size:11px;font-weight:500;letter-spacing:1px;text-transform:uppercase;cursor:pointer;transition:background .3s}
.post-btn:hover{background:var(--color-secondary)}
.post-btn:disabled{cursor:not-allowed;opacity:.5}
.emoji-btn{display:flex;width:36px;height:36px;align-items:center;justify-content:center;border:1px solid var(--color-border);border-radius:8px;background:transparent;color:var(--color-text-muted);cursor:pointer;transition:all .2s}
.emoji-btn:hover{border-color:var(--color-accent);background:var(--color-card-bg);color:var(--color-accent)}
.emoji-btn i{font-size:18px}
.comments-list{display:flex;flex-direction:column;gap:24px;margin-top:24px}
.comment-item{display:flex;gap:12px}
.comment-item.comment-highlight,.reply-item.comment-highlight{background:var(--color-primary-light);outline:2px solid var(--color-accent);outline-offset:2px;border-radius:8px}
.comment-avatar,.reply-avatar{display:flex;flex-shrink:0;align-items:center;justify-content:center;overflow:hidden;border:1px solid var(--color-border);border-radius:50%;background:linear-gradient(135deg,var(--color-accent),var(--color-secondary));color:var(--btn-primary-text,var(--color-text-light));font-weight:600}
.comment-avatar{width:32px;height:32px;font-size:12px}
.reply-avatar{width:28px;height:28px;font-size:11px}
.comment-avatar img,.reply-avatar img{width:100%;height:100%;object-fit:cover}
.comment-body,.reply-body{min-width:0;flex:1}
.comment-meta{display:flex;align-items:baseline;gap:10px;margin-bottom:6px}
.reply-meta{display:flex;align-items:baseline;flex-wrap:wrap;gap:6px;margin-bottom:4px}
.comment-author{color:var(--color-text-main);font-size:14px;font-weight:500}
.reply-author{color:var(--color-text-main);font-size:13px;font-weight:500}
.comment-time{color:var(--color-text-muted);font-size:12px}
.reply-time,.reply-to{color:var(--color-text-muted);font-size:11px}
.reply-to-name{color:var(--color-secondary);font-weight:500}
.comment-text{margin:0;color:var(--color-primary);font-size:14px;line-height:1.6}
.reply-text{margin:0;color:var(--color-primary);font-size:13px;line-height:1.5}
.comment-image{display:block;width:min(100%,360px);max-height:280px;overflow:hidden;margin-top:10px;padding:0;border:1px solid var(--color-border);border-radius:8px;background:var(--color-card-bg);cursor:zoom-in}.comment-image img{display:block;width:100%;max-height:278px;object-fit:contain}.comment-image.reply-image{width:min(100%,300px);max-height:230px}
.comment-text :deep(.comment-emote),.reply-text :deep(.comment-emote){display:inline-block;width:64px;height:64px;margin:4px 6px 4px 0;object-fit:contain;vertical-align:middle}
.comment-text :deep(.comment-mention),.reply-text :deep(.comment-mention){display:inline-flex;align-items:center;margin:0 2px;padding:2px 8px;border-radius:999px;background:var(--color-primary-light);color:var(--color-secondary);font-weight:600}
.comment-actions{display:flex;flex-wrap:wrap;align-items:center;gap:12px;margin-top:8px}
.reply-btn,.like-btn-inline{display:inline-flex;align-items:center;border:0;background:none;color:var(--color-text-muted);cursor:pointer;transition:color .2s}
.reply-btn{gap:4px;padding:4px 8px;font-size:12px}
.reply-btn:hover,.like-btn-inline:hover{color:var(--color-secondary)}
.comment-safety-btn,.delete-btn{display:inline-flex;align-items:center;gap:4px;padding:4px 8px;border:0;background:none;font-size:12px;cursor:pointer;transition:color .2s,background .2s}
.comment-safety-btn{color:var(--color-text-muted)}
.comment-safety-btn:hover{color:var(--color-secondary)}
.comment-safety-btn.danger,.delete-btn{color:#c44536}
.comment-safety-btn.danger:hover,.delete-btn:hover{color:#dc2626;background:rgba(220,38,38,.05)}
.like-btn-inline{gap:3px;margin-left:8px;padding:2px 6px;font-size:11px}
.like-btn-inline.active,.like-btn-inline.active i{color:#dc2626}
.like-btn-inline i{font-size:13px}
.replies-list{margin-top:16px;padding-left:12px;border-left:2px solid var(--color-border-light)}
.reply-item{display:flex;gap:10px;padding:12px 0}
.reply-item:not(:last-child){border-bottom:1px solid var(--color-border-light)}
.empty-comments{padding:40px 16px;color:var(--color-text-muted);font-size:14px;text-align:center}
.work-edit-footer{display:flex;justify-content:flex-end;padding:18px 32px;border-top:1px solid var(--rpdb-line);background:var(--color-panel-bg)}
.work-edit-footer a{display:inline-flex;min-height:40px;align-items:center;justify-content:center;gap:7px;padding:0 16px;border:1px solid var(--color-accent);border-radius:10px;background:var(--color-accent);color:#fff;text-decoration:none;font-weight:800}
.work-edit-footer a:hover{filter:brightness(.96)}
.floating-toc-panel{position:fixed;z-index:25;top:92px;right:24px;width:240px}
.floating-toc-panel.is-collapsed{width:0}
.floating-toc-content{max-height:calc(100vh - 120px);overflow:auto;box-shadow:var(--shadow-md);backdrop-filter:blur(12px)}
.toc-collapse{position:absolute;top:12px;left:-38px;display:grid;width:34px;height:44px;place-items:center;border:1px solid var(--rpdb-line);border-radius:12px 0 0 12px;background:var(--rpdb-surface);color:var(--color-accent);box-shadow:0 10px 24px rgba(0,0,0,.12)}
.toc-collapse:hover{background:var(--rpdb-soft);color:var(--color-text-main)}
.floating-toc-panel.is-collapsed .toc-collapse{left:-38px;border-radius:12px;background:var(--color-accent);color:#fff}
.floating-toc-panel section{padding:16px;border-bottom:1px solid var(--rpdb-line)}
.floating-toc-panel section:last-child{border-bottom:0}
.floating-toc-panel h3{margin:0 0 10px;color:var(--color-text-main);font-size:14px}
.floating-toc-panel p{margin:0 0 11px;color:var(--color-text-secondary);font-size:12px;line-height:1.7}
.toc{display:grid;gap:3px}
.toc button{padding:8px 9px;border:0;border-radius:9px;background:transparent;color:var(--color-text-secondary);text-align:left}
.toc button.active,.toc button:hover{background:var(--rpdb-soft);color:var(--color-text-main);font-weight:700}
.assistant-actions{display:grid;gap:7px}
.assistant-actions button{width:100%}
.assistant-actions button:disabled{opacity:.45}
.floating-actions{display:grid;gap:7px}
.floating-view{display:flex;min-height:36px;align-items:center;gap:6px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-muted);color:var(--color-text-secondary);font-weight:800}.floating-view i{color:var(--color-accent)}.floating-view b{margin-left:auto;color:var(--color-text-main);font-size:11px}
.floating-actions button{width:100%;justify-content:flex-start;font-weight:800}
.floating-actions button b{margin-left:auto;font-size:11px}
.home-copy-code,.transmog-copy-code{width:100%;min-height:44px;margin-top:12px;border-color:var(--color-accent);background:var(--color-accent);color:#fff;font-weight:800}
.home-copy-code.hero-copy-code{min-height:48px;margin-top:0;font-size:14px;letter-spacing:.02em}
.home-copy-code:disabled,.transmog-copy-code:disabled{cursor:not-allowed;opacity:.45}
.floating-toc-panel a{display:flex;gap:8px;padding:9px 0;border-bottom:1px solid var(--rpdb-line);color:var(--color-text-main);text-decoration:none}
.floating-toc-panel a:last-child{border-bottom:0}
.floating-toc-panel a i{color:var(--color-accent)}
.floating-toc-panel a span{display:flex;min-width:0;flex-direction:column}
.floating-toc-panel a small{margin-top:3px;color:var(--color-text-secondary)}
.reference-object{display:flex;align-items:center;gap:8px;padding:9px 0;border-bottom:1px solid var(--rpdb-line);color:var(--color-text-main);text-decoration:none}
.reference-object:last-child{border-bottom:0}
.reference-object.has-link:hover b{color:var(--color-accent)}
.reference-icon{display:grid!important;flex:0 0 auto;width:38px;height:38px;place-items:center;overflow:hidden;border:2px solid #9d9d9d;border-radius:6px;background:#16110d;box-shadow:inset 0 0 0 1px rgba(255,255,255,.2),0 2px 8px rgba(0,0,0,.18)}
.reference-icon img{width:100%;height:100%;object-fit:cover}
.reference-icon i{color:#c8bfb6;font-size:18px}
.reference-copy{display:flex;min-width:0;flex-direction:column;gap:2px}
.reference-copy b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.reference-copy small{margin:0;color:var(--color-text-secondary)}
.reference-copy .reference-description{display:-webkit-box;overflow:hidden;-webkit-box-orient:vertical;-webkit-line-clamp:2;line-height:1.45}
.reference-icon.quality-poor{border-color:#9d9d9d}
.reference-icon.quality-common{border-color:#ffffff}
.reference-icon.quality-uncommon{border-color:#1eff00}
.reference-icon.quality-rare{border-color:#0070dd}
.reference-icon.quality-epic{border-color:#a335ee}
.reference-icon.quality-legendary{border-color:#ff8000}
.reference-icon.quality-artifact{border-color:#e6cc80}
.reference-icon.quality-heirloom{border-color:#00ccff}
.sidebar-metadata{display:grid;gap:8px;margin:0}
.sidebar-metadata div{display:flex;justify-content:space-between;gap:10px}
.sidebar-metadata dt{color:var(--color-text-secondary)}
.sidebar-metadata dd{margin:0;text-align:right}
.list-picker-mask{position:fixed;inset:0;z-index:2200;display:grid;place-items:center;background:rgba(var(--shadow-base),.42);backdrop-filter:blur(4px)}
.list-picker-dialog{width:min(480px,calc(100vw - 32px));overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--color-panel-bg);color:var(--color-text-main);box-shadow:var(--shadow-lg)}
.list-picker-dialog header{display:flex;align-items:flex-start;justify-content:space-between;gap:14px;padding:18px 18px 12px;border-bottom:1px solid var(--rpdb-line)}
.list-picker-dialog header span{color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.06em}
.list-picker-dialog h2{margin:5px 0 0;font:700 20px/1.25 system-ui,'Microsoft YaHei',sans-serif}
.list-picker-dialog header button{display:grid;width:32px;height:32px;place-items:center;border:1px solid var(--rpdb-line);border-radius:9px;background:var(--color-panel-bg);color:var(--color-text-main)}
.list-picker-body{display:grid;gap:12px;padding:14px}
.list-picker-create{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:10px;align-items:end;padding:12px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--rpdb-muted)}
.list-picker-create label{display:grid;gap:6px;color:var(--color-text-main);font-weight:800}
.list-picker-create label span{font-size:12px}
.list-picker-create input{min-width:0;height:38px;padding:0 10px;border:1px solid var(--rpdb-line);border-radius:9px;background:var(--color-panel-bg);color:var(--color-text-main);font:inherit}
.list-picker-create button{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:6px;padding:0 12px;border:1px solid var(--color-accent);border-radius:9px;background:var(--color-accent);color:#fff;font-weight:800}
.list-picker-create button:disabled{cursor:not-allowed;opacity:.45}
.list-picker-options{display:grid;gap:8px;padding:14px}
.list-picker-body .list-picker-options{padding:0}
.list-picker-options button{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-muted);color:var(--color-text-main);text-align:left}
.list-picker-options button:hover{border-color:var(--color-accent);background:var(--rpdb-soft)}
.list-picker-options span{display:flex;min-width:0;flex-direction:column}
.list-picker-options small{margin-top:4px;color:var(--color-text-secondary)}
.list-picker-options em{flex:0 0 auto;color:var(--color-text-secondary);font-size:11px;font-style:normal}
.list-picker-state{display:grid;min-height:150px;place-items:center;align-content:center;gap:8px;padding:20px;color:var(--color-text-secondary);text-align:center}
.list-picker-state i{color:var(--color-accent);font-size:28px}
.list-picker-state a,.list-picker-dialog footer a{color:var(--color-accent);text-decoration:none;font-weight:800}
.list-picker-dialog footer{display:flex;justify-content:flex-end;padding:12px 14px;border-top:1px solid var(--rpdb-line)}
.list-picker-dialog footer a{display:inline-flex;align-items:center;gap:6px}
@media(max-width:1180px){.floating-toc-panel{right:12px;width:226px}}
@media(max-width:1050px){.detail-hero{grid-template-columns:1fr}.hero-gallery{border-right:0;border-bottom:1px solid var(--rpdb-line)}}
@media(max-width:980px){.recommendations-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:680px){.hero-summary{padding:22px}.hero-summary h1{font-size:29px}.hero-metadata{grid-template-columns:1fr}.hero-actions{gap:4px}.hero-actions button{padding:0 4px;font-size:11px}.recommendations-section,.comments-section{padding:22px 18px}.recommendations-grid{grid-template-columns:1fr}.work-edit-footer{padding:16px 18px}.work-edit-footer a{width:100%}.comment-meta,.reply-meta{align-items:flex-start;flex-direction:column;gap:4px}.comment-actions{gap:8px}.floating-toc-panel{top:auto;right:12px;bottom:12px;left:12px;width:auto}.floating-toc-panel.is-collapsed{left:auto;width:0}.floating-toc-content{max-height:42vh}.toc-collapse{top:-48px;right:0;left:auto;border-radius:12px}.floating-toc-panel section{border-right:0}}
</style>
