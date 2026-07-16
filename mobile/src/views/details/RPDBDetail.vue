<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import {
  addRPDBWorkToList,
  createRPDBComment,
  createRPDBList,
  deleteRPDBComment,
  favoriteRPDBWork,
  getRPDBWork,
  likeRPDBComment,
  likeRPDBWork,
  listRPDBComments,
  listRPDBLists,
  listRPDBWorkRecommendations,
  resolveRPDBMediaUrl,
  unfavoriteRPDBWork,
  unlikeRPDBComment,
  unlikeRPDBWork,
  verifyRPDBWork,
  type RPDBComment,
  type RPDBList,
  type RPDBMedia,
  type RPDBTransmogSlot,
  type RPDBWork,
} from '@/api/rpdb'
import CachedImage from '@/components/CachedImage.vue'
import ImagePreviewDialog from '@/components/ImagePreviewDialog.vue'
import MobileEmojiPicker from '@/components/MobileEmojiPicker.vue'
import MobileRPDBWorkCard from '@/components/MobileRPDBWorkCard.vue'
import SafetyReportSheet from '@/components/SafetyReportSheet.vue'
import { createContentReport, type ReportTargetType } from '@/api/safety'
import { handleJumpLinkClick, sanitizeJumpLinks } from '@/utils/jumpLink'
import { copyTextToClipboard, shareRouteLink } from '@/utils/mobileShare'
import {
  buildTomTomCommand,
  getRPDBSummary,
  getRPDBTypeIcon,
  getRPDBTypeLabel,
  parseRPDBExtra,
  qualityClass,
} from '@/utils/rpdb'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const userStore = useUserStore()

const loading = ref(true)
const work = ref<RPDBWork | null>(null)
const comments = ref<RPDBComment[]>([])
const recommendations = ref<RPDBWork[]>([])
const activeMediaIndex = ref(0)
const previewOpen = ref(false)
const previewSrc = ref('')
const actionBusy = ref('')
const commentText = ref('')
const commentSubmitting = ref(false)
const emojiOpen = ref(false)
const commentInput = ref<HTMLTextAreaElement | null>(null)
const listSheetOpen = ref(false)
const listsLoading = ref(false)
const lists = ref<RPDBList[]>([])
const newListName = ref('')
const listSubmitting = ref(false)
const replyingTo = ref<RPDBComment | null>(null)
const verifyBusy = ref(false)
const reportOpen = ref(false)
const reportSubmitting = ref(false)
const reportContext = ref<{
  targetType: Extract<ReportTargetType, 'rpdb_work' | 'rpdb_comment'>
  targetId: number
  title: string
  dialogTitle: string
} | null>(null)
const deleteCommentTarget = ref<RPDBComment | null>(null)
const commentLikeBusy = ref(new Set<number>())

const workId = computed(() => Number(route.params.id))
const extra = computed(() => parseRPDBExtra(work.value?.extra))
const shareCode = computed(() => extra.value.share_code?.trim() || '')
const mediaItems = computed<Array<{ type: RPDBMedia['type']; url: string; caption: string }>>(() => {
  const result: Array<{ type: RPDBMedia['type']; url: string; caption: string }> = []
  if (work.value?.cover_image) {
    result.push({
      type: 'image',
      url: resolveRPDBMediaUrl(work.value.cover_image),
      caption: work.value.title,
    })
  }
  for (const item of work.value?.media || []) {
    const url = resolveRPDBMediaUrl(item.url)
    if (!url || result.some(existing => existing.url === url)) continue
    result.push({ type: item.type, url, caption: item.caption || work.value?.title || '' })
  }
  return result
})
const activeMedia = computed(() => mediaItems.value[activeMediaIndex.value])
const styleTags = computed(() => (work.value?.tags || []).filter(tag => tag.name.endsWith('风格')))
const orderedSlots = computed(() => {
  const rank = new Map([
    ['head', 1], ['shoulder', 2], ['back', 3], ['chest', 4], ['shirt', 5],
    ['tabard', 6], ['wrist', 7], ['hands', 8], ['waist', 9], ['legs', 10],
    ['feet', 11], ['main_hand', 12], ['off_hand', 13],
  ])
  return [...(work.value?.transmog_slots || [])]
    .filter(slot => slot.role !== 'unused')
    .sort((left, right) => (rank.get(left.slot) || 99) - (rank.get(right.slot) || 99))
})
const orderedGuideSteps = computed(() => [...(work.value?.guide_steps || [])].sort((a, b) => a.sort_order - b.sort_order))
const organizedComments = computed(() => {
  const roots = comments.value.filter(item => !item.parent_id)
  return roots.map(root => ({
    ...root,
    replies: comments.value.filter(item => item.parent_id === root.id),
  }))
})
const normalizedContent = computed(() => normalizeRichContent(work.value?.content || ''))
const canEdit = computed(() => Boolean(
  work.value
  && userStore.user
  && (userStore.user.id === work.value.author_id || userStore.user.role === 'admin'),
))
const canReportWork = computed(() => Boolean(work.value && userStore.user?.id && work.value.author_id !== userStore.user.id))

function availabilityLabel(value?: string) {
  const labels: Record<string, string> = {
    available: '可获取',
    limited: '限时获取',
    removed: '已绝版',
    unknown: '未知',
    friend_only: '好友可参观',
    closed: '暂不开放',
  }
  return labels[String(value || '').toLowerCase()] || value || '未知'
}

function bindLabel(value?: string) {
  const labels: Record<string, string> = {
    yes: '绑定',
    no: '不绑定',
    account: '战网通行证绑定',
    pickup: '拾取后绑定',
    use: '使用后绑定',
    unknown: '未知',
  }
  return labels[String(value || '').toLowerCase()] || value || '未知'
}

function factionLabel(value?: string) {
  const labels: Record<string, string> = {
    neutral: '中立',
    neutra: '中立',
    alliance: '联盟',
    horde: '部落',
  }
  return labels[String(value || '').toLowerCase()] || value || '中立'
}

function slotLabel(slot: string) {
  const labels: Record<string, string> = {
    head: '头部',
    shoulder: '肩部',
    back: '背部',
    chest: '胸甲',
    shirt: '衬衣',
    tabard: '战袍',
    wrist: '护腕',
    hands: '手套',
    waist: '腰带',
    legs: '腿部',
    feet: '脚部',
    main_hand: '主手',
    off_hand: '副手',
  }
  return labels[slot] || slot
}

function roleLabel(role?: RPDBTransmogSlot['role']) {
  return ({ required: '必选', optional: '可选', variant: '替代' } as Record<string, string>)[role || 'required'] || '必选'
}

function referenceTypeLabel(value: string) {
  return ({
    item: '物品',
    equipment: '装备',
    toy: '玩具',
    quest_item: '任务道具',
    transmog: '幻化',
    furniture: '家具',
  } as Record<string, string>)[value] || value || '物品'
}

function formatTime(value: string) {
  const date = new Date(value)
  const diff = Date.now() - date.getTime()
  const hours = Math.floor(diff / 3_600_000)
  if (hours < 1) return '刚刚'
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days} 天前`
  return date.toLocaleDateString('zh-CN')
}

function canDeleteComment(item: RPDBComment) {
  if (!work.value || !userStore.user) return false
  return item.author_id === userStore.user.id
    || work.value.author_id === userStore.user.id
    || userStore.user.role === 'moderator'
    || userStore.user.role === 'admin'
}

function canReportComment(item: RPDBComment) {
  return Boolean(userStore.user?.id && item.author_id !== userStore.user.id)
}

function normalizeRichContent(raw: string) {
  const html = raw.trim()
  if (!html || typeof DOMParser === 'undefined') return html
  if (!/<[a-z][\s\S]*>/i.test(html)) {
    return html.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>')
  }
  const doc = new DOMParser().parseFromString(html, 'text/html')
  doc.querySelectorAll('script, style, iframe, object, embed').forEach(node => node.remove())
  doc.querySelectorAll<HTMLElement>('*').forEach(node => {
    Array.from(node.attributes).forEach(attribute => {
      if (attribute.name.startsWith('on')) node.removeAttribute(attribute.name)
    })
  })
  doc.querySelectorAll('img').forEach(image => {
    const src = image.getAttribute('src') || ''
    if (!/^(https?:|data:image\/|\/|\.\/|\.\.\/)/i.test(src)) {
      image.remove()
      return
    }
    image.setAttribute('src', resolveRPDBMediaUrl(src))
    image.setAttribute('loading', 'lazy')
  })
  doc.querySelectorAll('a').forEach(link => {
    const href = link.getAttribute('href') || ''
    if (!/^(https?:|mailto:|\/|#)/i.test(href)) {
      link.removeAttribute('href')
      return
    }
    if (/^https?:\/\//.test(href)) {
      link.setAttribute('target', '_blank')
      link.setAttribute('rel', 'noopener noreferrer')
    }
  })
  sanitizeJumpLinks(doc.body)
  return doc.body.innerHTML
}

function handleContentClick(event: MouseEvent) {
  if (handleJumpLinkClick(event, router)) return
  const target = event.target instanceof Element ? event.target : null
  const image = target?.closest('img') as HTMLImageElement | null
  if (!image?.src) return
  event.preventDefault()
  openPreview(image.currentSrc || image.src)
}

function openPreview(src: string) {
  previewSrc.value = src
  previewOpen.value = true
}

async function load() {
  loading.value = true
  try {
    const [detail, discussion, related] = await Promise.all([
      getRPDBWork(workId.value),
      listRPDBComments(workId.value),
      listRPDBWorkRecommendations(workId.value).catch(() => ({ recommendations: [] })),
    ])
    work.value = detail.work
    comments.value = discussion.comments || []
    recommendations.value = related.recommendations || []
    activeMediaIndex.value = 0
  } catch (error) {
    toast.error((error as Error).message || '作品加载失败')
  } finally {
    loading.value = false
  }
}

async function toggleLike() {
  if (!work.value || actionBusy.value) return
  actionBusy.value = 'like'
  try {
    await (work.value.is_liked ? unlikeRPDBWork(work.value.id) : likeRPDBWork(work.value.id))
    work.value.is_liked = !work.value.is_liked
    work.value.like_count = Math.max(0, work.value.like_count + (work.value.is_liked ? 1 : -1))
    toast.success(work.value.is_liked ? '已点赞' : '已取消点赞')
  } catch (error) {
    toast.error((error as Error).message || '操作失败')
  } finally {
    actionBusy.value = ''
  }
}

async function toggleFavorite() {
  if (!work.value || actionBusy.value) return
  actionBusy.value = 'favorite'
  try {
    await (work.value.is_favorited ? unfavoriteRPDBWork(work.value.id) : favoriteRPDBWork(work.value.id))
    work.value.is_favorited = !work.value.is_favorited
    work.value.favorite_count = Math.max(0, work.value.favorite_count + (work.value.is_favorited ? 1 : -1))
    toast.success(work.value.is_favorited ? '已收藏' : '已取消收藏')
  } catch (error) {
    toast.error((error as Error).message || '操作失败')
  } finally {
    actionBusy.value = ''
  }
}

async function openListSheet() {
  if (!work.value) return
  listSheetOpen.value = true
  listsLoading.value = true
  try {
    const result = await listRPDBLists()
    lists.value = result.lists || []
  } catch (error) {
    toast.error((error as Error).message || '清单加载失败')
  } finally {
    listsLoading.value = false
  }
}

async function addToList(list: RPDBList) {
  if (!work.value || listSubmitting.value) return
  listSubmitting.value = true
  try {
    await addRPDBWorkToList(work.value.id, 'wanted', list.id)
    if (!work.value.in_collection_list) work.value.list_count += 1
    work.value.in_collection_list = true
    listSheetOpen.value = false
    toast.success(`已加入「${list.name}」`)
  } catch (error) {
    toast.error((error as Error).message || '加入清单失败')
  } finally {
    listSubmitting.value = false
  }
}

async function createListAndAdd() {
  const name = newListName.value.trim()
  if (!name || listSubmitting.value) {
    if (!name) toast.warning('请填写清单名称')
    return
  }
  listSubmitting.value = true
  try {
    const result = await createRPDBList(name)
    newListName.value = ''
    listSubmitting.value = false
    await addToList(result.list)
  } catch (error) {
    listSubmitting.value = false
    toast.error((error as Error).message || '创建清单失败')
  }
}

async function submitComment(parentId?: number) {
  const content = commentText.value.trim()
  if (!work.value || !content || commentSubmitting.value) return
  commentSubmitting.value = true
  try {
    await createRPDBComment(work.value.id, content, parentId)
    const result = await listRPDBComments(work.value.id)
    comments.value = result.comments || []
    work.value.comment_count = comments.value.length
    commentText.value = ''
    replyingTo.value = null
    toast.success('评论已发布')
  } catch (error) {
    toast.error((error as Error).message || '评论发布失败')
  } finally {
    commentSubmitting.value = false
  }
}

function startReply(item: RPDBComment) {
  replyingTo.value = item
  commentText.value = ''
  void nextTick(() => commentInput.value?.focus())
}

async function toggleCommentLike(item: RPDBComment) {
  if (commentLikeBusy.value.has(item.id)) return
  commentLikeBusy.value.add(item.id)
  try {
    await (item.liked ? unlikeRPDBComment(item.id) : likeRPDBComment(item.id))
    item.liked = !item.liked
    item.like_count = Math.max(0, item.like_count + (item.liked ? 1 : -1))
  } catch (error) {
    toast.error((error as Error).message || '评论点赞失败')
  } finally {
    commentLikeBusy.value.delete(item.id)
  }
}

async function confirmDeleteComment() {
  const target = deleteCommentTarget.value
  if (!target || !work.value) return
  try {
    await deleteRPDBComment(target.id)
    const result = await listRPDBComments(work.value.id)
    comments.value = result.comments || []
    work.value.comment_count = comments.value.length
    deleteCommentTarget.value = null
    toast.success('评论已删除')
  } catch (error) {
    toast.error((error as Error).message || '评论删除失败')
  }
}

async function verifyWork(result: 'valid' | 'outdated') {
  if (!work.value || verifyBusy.value) return
  verifyBusy.value = true
  try {
    await verifyRPDBWork(work.value.id, result)
    if (result === 'valid') work.value.verified_count += 1
    else work.value.outdated_count += 1
    toast.success(result === 'valid' ? '已确认当前信息有效' : '已反馈信息可能过期')
  } catch (error) {
    toast.error((error as Error).message || '验证提交失败')
  } finally {
    verifyBusy.value = false
  }
}

function openWorkReport() {
  if (!work.value) return
  reportContext.value = {
    targetType: 'rpdb_work',
    targetId: work.value.id,
    title: work.value.title,
    dialogTitle: '举报 RP 数据库作品',
  }
  reportOpen.value = true
}

function openCommentReport(item: RPDBComment) {
  reportContext.value = {
    targetType: 'rpdb_comment',
    targetId: item.id,
    title: `${item.author_name || '匿名玩家'}：${item.content.slice(0, 36)}`,
    dialogTitle: '举报评论',
  }
  reportOpen.value = true
}

async function submitReport(payload: { reason: string; detail: string; hideTarget: boolean; blockAuthor: boolean; submitReport: boolean }) {
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
    reportOpen.value = false
    reportContext.value = null
    toast.success(result.message || '处理已提交')
    if (context.targetType === 'rpdb_work' && (payload.hideTarget || payload.blockAuthor)) {
      await router.replace({ name: 'rpdb' })
    } else if (work.value) {
      const discussion = await listRPDBComments(work.value.id)
      comments.value = discussion.comments || []
    }
  } catch (error) {
    toast.error((error as Error).message || '举报提交失败')
  } finally {
    reportSubmitting.value = false
  }
}

function insertEmoji(token: string) {
  const input = commentInput.value
  const start = input?.selectionStart ?? commentText.value.length
  const end = input?.selectionEnd ?? start
  commentText.value = `${commentText.value.slice(0, start)}${token}${commentText.value.slice(end)}`
  emojiOpen.value = false
  void nextTick(() => {
    input?.focus()
    input?.setSelectionRange(start + token.length, start + token.length)
  })
}

async function copyShareCode() {
  if (!shareCode.value) return
  try {
    await copyTextToClipboard(shareCode.value, work.value?.title)
    toast.success(work.value?.type === 'home_showcase' ? '住宅分享代码已复制' : '幻化分享代码已复制')
  } catch {
    toast.error('复制失败')
  }
}

async function copyTomTom(step: (typeof orderedGuideSteps.value)[number]) {
  const command = buildTomTomCommand(step)
  if (!command) return
  try {
    await copyTextToClipboard(command, step.title)
    toast.success('TomTom 坐标已复制')
  } catch {
    toast.error('复制失败')
  }
}

async function shareWork() {
  if (!work.value) return
  try {
    await shareRouteLink({
      path: `/rpdb/${work.value.id}`,
      title: work.value.title,
      text: getRPDBSummary(work.value),
      dialogTitle: '分享 RP 数据库作品',
    })
  } catch {
    toast.error('分享失败')
  }
}

function goBack() {
  if (route.query.from === 'collection') {
    void router.push({ name: 'rpdb-lists' })
    return
  }
  void router.push({ name: 'rpdb' })
}

watch(workId, (next, previous) => {
  if (next === previous) return
  void load()
  window.scrollTo({ top: 0 })
})

onMounted(load)
</script>

<template>
  <div class="rpdb-detail sub-page">
    <header class="detail-header">
      <button type="button" class="icon-button" aria-label="返回 RP 数据库" @click="goBack">
        <i class="ri-arrow-left-line" />
      </button>
      <div>
        <small>RP 数据库</small>
        <b>{{ work?.title || '作品档案' }}</b>
      </div>
      <button type="button" class="icon-button" aria-label="分享作品" @click="shareWork">
        <i class="ri-share-forward-line" />
      </button>
    </header>

    <div v-if="loading" class="loading-state">
      <i class="ri-loader-4-line spin" />
      <span>正在打开作品档案</span>
    </div>

    <div v-else-if="!work" class="loading-state">
      <i class="ri-file-damage-line" />
      <span>作品不存在或暂时无法访问</span>
      <button type="button" @click="router.push({ name: 'rpdb' })">返回数据库</button>
    </div>

    <template v-else>
      <main class="detail-body">
        <section class="media-stage">
          <button
            v-if="activeMedia && (activeMedia.type === 'image' || activeMedia.type === 'gif')"
            type="button"
            class="active-media"
            @click="openPreview(activeMedia.url)"
          >
            <CachedImage :src="activeMedia.url" :alt="activeMedia.caption" loading="eager" />
            <span><i class="ri-zoom-in-line" />查看原图</span>
          </button>
          <video v-else-if="activeMedia?.type === 'video'" class="active-video" :src="activeMedia.url" controls playsinline />
          <a v-else-if="activeMedia" class="media-link" :href="activeMedia.url" target="_blank" rel="noopener">
            <i class="ri-play-circle-line" />
            <span>打开外部视频</span>
          </a>
          <div v-else class="media-empty"><i :class="getRPDBTypeIcon(work.type)" /></div>

          <div v-if="mediaItems.length > 1" class="media-thumbs">
            <button
              v-for="(item, index) in mediaItems"
              :key="`${item.url}-${index}`"
              type="button"
              :class="{ active: activeMediaIndex === index }"
              :aria-label="`查看媒体 ${index + 1}`"
              @click="activeMediaIndex = index"
            >
              <CachedImage v-if="item.type === 'image' || item.type === 'gif'" :src="item.url" :alt="item.caption" />
              <i v-else class="ri-play-circle-fill" />
            </button>
          </div>
        </section>

        <section class="hero-copy">
          <div class="badge-row">
            <span class="type-badge"><i :class="getRPDBTypeIcon(work.type)" />{{ getRPDBTypeLabel(work.type) }}</span>
            <span v-for="tag in styleTags" :key="tag.id" class="style-badge">{{ tag.name }}</span>
          </div>
          <h1>{{ work.title }}</h1>
          <p>{{ getRPDBSummary(work) }}</p>
          <div class="author-row">
            <span class="author-avatar">{{ (work.author_name || 'R').slice(0, 1).toUpperCase() }}</span>
            <span>
              <b :style="{ color: work.author_name_color || undefined, fontWeight: work.author_name_bold ? 'bold' : undefined }">
                {{ work.author_name || '匿名贡献者' }}
              </b>
              <small>版本 {{ work.version }} · {{ new Date(work.updated_at).toLocaleDateString('zh-CN') }}</small>
            </span>
          </div>
          <button v-if="canEdit" type="button" class="edit-work" @click="router.push({ name: 'rpdb-edit', params: { id: work.id } })">
            <i class="ri-edit-line" />编辑这份作品
          </button>
          <button v-if="canReportWork" type="button" class="report-work" @click="openWorkReport">
            <i class="ri-flag-line" />举报作品
          </button>
        </section>

        <dl class="metadata-strip">
          <div>
            <dt>{{ work.type === 'home_showcase' ? '开放状态' : '获取状态' }}</dt>
            <dd>{{ availabilityLabel(work.availability_status) }}</dd>
          </div>
          <div>
            <dt>绑定</dt>
            <dd>{{ bindLabel(work.bind_type) }}</dd>
          </div>
          <div>
            <dt>阵营</dt>
            <dd>{{ factionLabel(work.faction) }}</dd>
          </div>
        </dl>

        <dl class="stats-strip">
          <div><dt><i class="ri-eye-line" />浏览</dt><dd>{{ work.view_count }}</dd></div>
          <div><dt><i class="ri-heart-3-line" />点赞</dt><dd>{{ work.like_count }}</dd></div>
          <div><dt><i class="ri-bookmark-3-line" />收藏</dt><dd>{{ work.favorite_count }}</dd></div>
          <div><dt><i class="ri-list-check-3" />清单</dt><dd>{{ work.list_count }}</dd></div>
        </dl>

        <section class="verification-strip">
          <div>
            <span><i class="ri-shield-check-line" />玩家验证</span>
            <p>{{ work.verified_count }} 人确认有效 · {{ work.outdated_count }} 人反馈过期</p>
          </div>
          <div>
            <button type="button" :disabled="verifyBusy" @click="verifyWork('valid')"><i class="ri-check-line" />仍然有效</button>
            <button type="button" :disabled="verifyBusy" @click="verifyWork('outdated')"><i class="ri-time-line" />可能过期</button>
          </div>
        </section>

        <section class="content-section">
          <header><span>作品介绍</span><h2>{{ work.type === 'home_showcase' ? '空间故事与参观亮点' : '实际效果与 RP 用途' }}</h2></header>
          <p v-if="work.rp_use_cases" class="use-case"><b>适用场景</b>{{ work.rp_use_cases }}</p>
          <div v-if="normalizedContent" class="rich-content" @click="handleContentClick" v-html="normalizedContent" />
          <p v-else class="empty-copy">作者尚未补充完整正文。</p>
        </section>

        <section v-if="shareCode && (work.type === 'transmog' || work.type === 'home_showcase')" class="share-code-section">
          <div>
            <i :class="work.type === 'home_showcase' ? 'ri-home-heart-line' : 'ri-shirt-line'" />
            <span>
              <b>{{ work.type === 'home_showcase' ? '住宅分享代码' : '幻化分享代码' }}</b>
              <small>复制后可在游戏内使用</small>
            </span>
          </div>
          <button type="button" @click="copyShareCode"><i class="ri-file-copy-line" />复制</button>
        </section>

        <section v-if="orderedSlots.length" class="content-section">
          <header><span>幻化部件</span><h2>部件与替代方案</h2></header>
          <div class="slot-list">
            <article v-for="slot in orderedSlots" :key="slot.id || `${slot.slot}-${slot.sort_order}`">
              <span class="slot-icon"><i class="ri-shirt-line" /></span>
              <div>
                <h3>{{ slotLabel(slot.slot) }}<small>{{ roleLabel(slot.role) }}</small></h3>
                <b>{{ slot.name || slot.note || '未填写名称' }}</b>
                <p v-if="slot.description">{{ slot.description }}</p>
                <p v-if="slot.source"><strong>来源</strong>{{ slot.source }}</p>
                <a v-if="slot.wowhead_url" :href="slot.wowhead_url" target="_blank" rel="noopener">在 Wowhead 查看</a>
              </div>
            </article>
          </div>
        </section>

        <section v-if="orderedGuideSteps.length && work.type !== 'home_showcase'" class="content-section">
          <header><span>获取攻略</span><h2>按步骤完成收集</h2></header>
          <ol class="guide-list">
            <li v-for="(step, index) in orderedGuideSteps" :key="step.id || index">
              <span class="step-index">{{ String(index + 1).padStart(2, '0') }}</span>
              <div>
                <h3>{{ step.title }}</h3>
                <p v-if="step.prerequisite" class="prerequisite"><i class="ri-lock-line" />{{ step.prerequisite }}</p>
                <p v-if="step.body">{{ step.body }}</p>
                <small v-if="step.zone"><i class="ri-map-pin-line" />{{ step.zone }}</small>
                <button v-if="buildTomTomCommand(step)" type="button" @click="copyTomTom(step)">
                  <i class="ri-file-copy-line" />复制 TomTom 坐标
                </button>
              </div>
            </li>
          </ol>
        </section>

        <section v-if="work.type === 'home_showcase' && extra.visit_notes" class="content-section">
          <header><span>参观资料</span><h2>前往家宅前</h2></header>
          <p class="visit-note">{{ extra.visit_notes }}</p>
        </section>

        <section v-if="work.references?.length" class="content-section">
          <header><span>引用对象</span><h2>关联的游戏物品</h2></header>
          <div class="reference-list">
            <component
              :is="reference.url ? 'a' : 'div'"
              v-for="reference in work.references"
              :key="reference.id || reference.external_id"
              :href="reference.url || undefined"
              :target="reference.url ? '_blank' : undefined"
              :rel="reference.url ? 'noopener noreferrer' : undefined"
              class="reference-row"
            >
              <span class="reference-icon" :class="qualityClass(reference.quality)">
                <CachedImage v-if="reference.icon" :src="resolveRPDBMediaUrl(reference.icon)" :alt="reference.name" />
                <i v-else class="ri-archive-2-line" />
              </span>
              <span>
                <b>{{ reference.name }}</b>
                <small>{{ referenceTypeLabel(reference.external_type) }}<template v-if="reference.source"> · {{ reference.source }}</template></small>
                <p v-if="reference.description">{{ reference.description }}</p>
              </span>
              <i v-if="reference.url" class="ri-external-link-line" />
            </component>
          </div>
        </section>

        <section v-if="recommendations.length" class="content-section recommendations">
          <header><span>继续探索</span><h2>相关推荐</h2></header>
          <div class="recommendation-list">
            <MobileRPDBWorkCard
              v-for="item in recommendations"
              :key="item.id"
              :work="item"
              compact
              @open="router.push({ name: 'rpdb-detail', params: { id: item.id } })"
            />
          </div>
        </section>

        <section class="content-section discussion">
          <header><span>玩家讨论</span><h2>评论 {{ comments.length }}</h2></header>
          <div class="comment-composer">
            <div v-if="replyingTo" class="replying-banner">
              <span>回复 {{ replyingTo.author_name || '匿名玩家' }}</span>
              <button type="button" @click="replyingTo = null; commentText = ''"><i class="ri-close-line" /></button>
            </div>
            <textarea ref="commentInput" v-model="commentText" rows="3" :placeholder="replyingTo ? `回复 ${replyingTo.author_name || '匿名玩家'}` : '分享你的使用经验或获取建议'" />
            <button type="button" class="emoji-button" aria-label="选择表情" @click="emojiOpen = true"><i class="ri-emotion-line" /></button>
            <button type="button" class="comment-submit" :disabled="commentSubmitting || !commentText.trim()" @click="submitComment(replyingTo?.id)">
              {{ commentSubmitting ? '发布中' : '发表评论' }}
            </button>
          </div>

          <div v-if="organizedComments.length" class="comment-list">
            <article v-for="item in organizedComments" :key="item.id">
              <div class="comment-head">
                <span class="comment-avatar">
                  <CachedImage v-if="item.author_avatar" :src="resolveRPDBMediaUrl(item.author_avatar)" alt="" />
                  <b v-else>{{ (item.author_name || '匿').slice(0, 1) }}</b>
                </span>
                <span>
                  <b>{{ item.author_name || '匿名玩家' }}</b>
                  <small>{{ formatTime(item.created_at) }}</small>
                </span>
              </div>
              <p>{{ item.content }}</p>
              <div class="comment-actions">
                <button
                  type="button"
                  :class="{ active: item.liked }"
                  :disabled="commentLikeBusy.has(item.id)"
                  @click="toggleCommentLike(item)"
                >
                  <i :class="item.liked ? 'ri-heart-3-fill' : 'ri-heart-3-line'" />
                  {{ item.liked ? '已赞' : '点赞' }}<span v-if="item.like_count">{{ item.like_count }}</span>
                </button>
                <button type="button" @click="startReply(item)"><i class="ri-reply-line" />回复</button>
                <button v-if="canReportComment(item)" type="button" @click="openCommentReport(item)"><i class="ri-flag-line" />举报</button>
                <button v-if="canDeleteComment(item)" type="button" class="danger" @click="deleteCommentTarget = item"><i class="ri-delete-bin-line" />删除</button>
              </div>
              <div v-if="item.replies.length" class="reply-list">
                <div v-for="reply in item.replies" :key="reply.id" class="reply-item">
                  <p class="reply-content">
                    <b>{{ reply.author_name || '匿名玩家' }}</b>
                    {{ reply.content }}
                  </p>
                  <div class="reply-actions">
                    <button
                      type="button"
                      :class="{ active: reply.liked }"
                      :disabled="commentLikeBusy.has(reply.id)"
                      @click="toggleCommentLike(reply)"
                    >
                      <i :class="reply.liked ? 'ri-heart-3-fill' : 'ri-heart-3-line'" />
                      {{ reply.liked ? '已赞' : '点赞' }}<span v-if="reply.like_count">{{ reply.like_count }}</span>
                    </button>
                    <button v-if="canReportComment(reply)" type="button" @click="openCommentReport(reply)"><i class="ri-flag-line" />举报</button>
                    <button v-if="canDeleteComment(reply)" type="button" class="danger" @click="deleteCommentTarget = reply"><i class="ri-delete-bin-line" />删除</button>
                  </div>
                </div>
              </div>
            </article>
          </div>
          <p v-else class="empty-copy">还没有评论，留下第一条使用体验。</p>
        </section>
      </main>

      <nav class="action-dock" aria-label="作品操作">
        <button type="button" :class="{ active: work.is_liked }" :disabled="Boolean(actionBusy)" @click="toggleLike">
          <i :class="work.is_liked ? 'ri-heart-3-fill' : 'ri-heart-3-line'" />
          <span>{{ work.is_liked ? '已赞' : '点赞' }}</span>
        </button>
        <button type="button" :class="{ active: work.is_favorited }" :disabled="Boolean(actionBusy)" @click="toggleFavorite">
          <i :class="work.is_favorited ? 'ri-bookmark-3-fill' : 'ri-bookmark-3-line'" />
          <span>{{ work.is_favorited ? '已藏' : '收藏' }}</span>
        </button>
        <button type="button" class="primary" @click="openListSheet">
          <i class="ri-list-check-3" />
          <span>{{ work.in_collection_list ? '已在清单' : '加入清单' }}</span>
        </button>
        <button type="button" @click="shareWork">
          <i class="ri-share-forward-line" />
          <span>分享</span>
        </button>
      </nav>
    </template>

    <ImagePreviewDialog :open="previewOpen" :src="previewSrc" @close="previewOpen = false" />
    <MobileEmojiPicker :open="emojiOpen" @close="emojiOpen = false" @select="insertEmoji" />
    <SafetyReportSheet
      :open="reportOpen"
      :submitting="reportSubmitting"
      :title="reportContext?.dialogTitle"
      :target-label="reportContext?.title"
      :target-type="reportContext?.targetType"
      @close="reportOpen = false"
      @submit="submitReport"
    />

    <Teleport to="body">
      <div v-if="listSheetOpen" class="sheet-mask" @click.self="listSheetOpen = false">
        <section class="list-sheet" role="dialog" aria-modal="true" aria-label="加入收集清单">
          <header>
            <div><small>收集助手</small><h2>加入清单</h2></div>
            <button type="button" aria-label="关闭" @click="listSheetOpen = false"><i class="ri-close-line" /></button>
          </header>
          <div class="new-list">
            <input v-model="newListName" type="text" maxlength="80" placeholder="新清单名称">
            <button type="button" :disabled="listSubmitting || !newListName.trim()" @click="createListAndAdd">
              <i class="ri-add-line" />新建并加入
            </button>
          </div>
          <div v-if="listsLoading" class="sheet-state"><i class="ri-loader-4-line spin" />加载清单</div>
          <div v-else-if="lists.length" class="list-options">
            <button v-for="list in lists" :key="list.id" type="button" :disabled="listSubmitting" @click="addToList(list)">
              <span><b>{{ list.name }}</b><small>{{ list.description || (list.is_default ? '默认收集清单' : '个人收集清单') }}</small></span>
              <em>{{ list.item_count }}</em>
            </button>
          </div>
          <div v-else class="sheet-state"><i class="ri-list-check-3" />还没有清单，可以在上方新建</div>
        </section>
      </div>
    </Teleport>

    <div v-if="deleteCommentTarget" class="comment-delete-mask">
      <section class="comment-delete-dialog" role="dialog" aria-modal="true">
        <h2>删除评论</h2>
        <p>删除后无法恢复，确认删除这条评论吗？</p>
        <footer><button type="button" @click="deleteCommentTarget = null">取消</button><button type="button" class="danger" @click="confirmDeleteComment">确认删除</button></footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.rpdb-detail {
  --detail-line: rgba(75, 54, 33, 0.13);
  min-height: var(--app-height, 100dvh);
  padding-bottom: calc(84px + var(--safe-bottom, 0px));
}

.detail-header {
  position: sticky;
  top: 0;
  z-index: 30;
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) 44px;
  align-items: center;
  gap: 8px;
  padding:
    calc(var(--safe-top, 0px) + 7px)
    calc(var(--safe-right, 0px) + var(--page-gutter))
    8px
    calc(var(--safe-left, 0px) + var(--page-gutter));
  border-bottom: 1px solid var(--detail-line);
  background: rgba(238, 217, 196, 0.94);
  backdrop-filter: blur(12px);
}

.detail-header > div {
  min-width: 0;
  text-align: center;
}

.detail-header small,
.detail-header b {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-header small {
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.06em;
}

.detail-header b {
  margin-top: 2px;
  font-size: 13px;
}

.icon-button {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 1px solid var(--detail-line);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.62);
  color: var(--color-text-main);
  font-size: 19px;
}

.loading-state {
  display: grid;
  min-height: 72vh;
  place-items: center;
  align-content: center;
  gap: 10px;
  color: var(--color-text-secondary);
}

.loading-state > i {
  color: var(--color-secondary);
  font-size: 36px;
}

.loading-state button {
  min-height: 38px;
  padding: 0 14px;
  border: 0;
  border-radius: 6px;
  background: var(--color-primary);
  color: #fff;
}

.detail-body {
  display: flex;
  flex-direction: column;
}

.media-stage {
  position: relative;
  background: #241711;
}

.active-media,
.active-video,
.media-link,
.media-empty {
  width: 100%;
  height: min(72vw, 360px);
  border: 0;
}

.active-media {
  position: relative;
  display: block;
  padding: 0;
  background: #241711;
}

.active-media span {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  gap: 5px;
  padding: 0 9px;
  border-radius: 5px;
  background: rgba(0, 0, 0, 0.62);
  color: #fff;
  font-size: 10px;
}

.active-video {
  display: block;
  background: #000;
}

.media-link,
.media-empty {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  background: linear-gradient(145deg, #4b3621, #804030);
  color: #fff;
  text-decoration: none;
}

.media-link i,
.media-empty i {
  color: #e6b878;
  font-size: 48px;
}

.media-thumbs {
  display: flex;
  gap: 7px;
  overflow-x: auto;
  padding: 8px var(--page-gutter);
  scrollbar-width: none;
}

.media-thumbs::-webkit-scrollbar {
  display: none;
}

.media-thumbs button {
  display: grid;
  width: 54px;
  height: 45px;
  flex: 0 0 54px;
  place-items: center;
  overflow: hidden;
  border: 2px solid transparent;
  border-radius: 5px;
  background: #4b3621;
  color: #fff;
}

.media-thumbs button.active {
  border-color: #e6b878;
}

.hero-copy,
.content-section,
.share-code-section {
  padding: 20px var(--page-gutter);
  border-bottom: 1px solid var(--detail-line);
  background: var(--color-panel-bg);
}

.badge-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.type-badge,
.style-badge {
  display: inline-flex;
  min-height: 24px;
  align-items: center;
  gap: 4px;
  padding: 0 7px;
  border-radius: 4px;
  background: var(--tag-bg);
  color: var(--tag-text);
  font-size: 10px;
  font-weight: 700;
}

.type-badge {
  background: var(--color-primary);
  color: #fff;
}

.hero-copy h1 {
  margin: 12px 0 7px;
  font-family: Georgia, 'Microsoft YaHei', serif;
  font-size: 27px;
  line-height: 1.25;
}

.hero-copy > p {
  color: var(--color-text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.author-row {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 16px;
}

.author-avatar,
.comment-avatar {
  display: grid;
  overflow: hidden;
  place-items: center;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
}

.author-avatar {
  width: 38px;
  height: 38px;
}

.author-row > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.author-row small {
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 10px;
}

.edit-work {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  gap: 5px;
  margin-top: 14px;
  padding: 0 11px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-card-bg);
  color: var(--color-secondary);
  font-size: 11px;
  font-weight: 700;
}

.report-work {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  gap: 5px;
  margin: 14px 0 0 6px;
  padding: 0 11px;
  border: 1px solid rgba(182, 56, 45, 0.28);
  border-radius: 6px;
  background: rgba(182, 56, 45, 0.05);
  color: #a33b32;
  font-size: 11px;
}

.metadata-strip,
.stats-strip {
  display: grid;
  margin: 0;
  border-bottom: 1px solid var(--detail-line);
  background: rgba(255, 255, 255, 0.58);
}

.metadata-strip {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.stats-strip {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  background: var(--color-primary);
  color: #fff;
}

.metadata-strip div,
.stats-strip div {
  min-width: 0;
  padding: 12px 7px;
  border-right: 1px solid var(--detail-line);
  text-align: center;
}

.metadata-strip div:last-child,
.stats-strip div:last-child {
  border-right: 0;
}

.metadata-strip dt,
.stats-strip dt {
  color: var(--color-text-secondary);
  font-size: 9px;
}

.stats-strip dt {
  color: rgba(255, 255, 255, 0.66);
}

.metadata-strip dd,
.stats-strip dd {
  overflow: hidden;
  margin: 4px 0 0;
  font-size: 12px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.verification-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 13px var(--page-gutter);
  border-bottom: 1px solid var(--detail-line);
  background: #f7efe6;
}

.verification-strip span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-secondary);
  font-size: 11px;
  font-weight: 800;
}

.verification-strip p {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 9px;
}

.verification-strip > div:last-child {
  display: flex;
  gap: 5px;
}

.verification-strip button {
  min-height: 34px;
  padding: 0 8px;
  border: 1px solid var(--color-border);
  border-radius: 5px;
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  font-size: 9px;
}

.stats-strip dd {
  color: #f1c58c;
  font-size: 14px;
}

.content-section header {
  margin-bottom: 14px;
}

.content-section header span {
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.content-section header h2 {
  margin-top: 4px;
  font-family: Georgia, 'Microsoft YaHei', serif;
  font-size: 20px;
}

.use-case,
.visit-note {
  margin-bottom: 14px;
  padding: 11px 12px;
  border-left: 3px solid var(--color-accent);
  background: rgba(184, 115, 51, 0.08);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.use-case b {
  display: block;
  margin-bottom: 3px;
  color: var(--color-text-main);
}

.rich-content {
  color: var(--color-text-main);
  font-size: 14px;
  line-height: 1.8;
  overflow-wrap: anywhere;
}

.rich-content :deep(p) {
  margin: 0 0 12px;
}

.rich-content :deep(h1) {
  margin: 16px 0 10px;
  font-size: 22px;
}

.rich-content :deep(h2) {
  margin: 15px 0 9px;
  font-size: 19px;
}

.rich-content :deep(h3) {
  margin: 13px 0 8px;
  font-size: 16px;
}

.rich-content :deep(img) {
  max-width: 100%;
  height: auto;
  margin: 8px 0;
  border-radius: 6px;
}

.rich-content :deep(a) {
  color: var(--color-secondary);
}

.rich-content :deep(ul),
.rich-content :deep(ol) {
  margin: 0 0 12px;
  padding-left: 20px;
}

.empty-copy {
  color: var(--color-text-secondary);
  font-size: 13px;
}

.share-code-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  background: #f8efe5;
}

.share-code-section > div {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 9px;
}

.share-code-section > div > i {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  place-items: center;
  border-radius: 7px;
  background: var(--color-primary);
  color: #fff;
  font-size: 19px;
}

.share-code-section span {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.share-code-section small {
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 10px;
}

.share-code-section button {
  min-height: 38px;
  flex: 0 0 auto;
  padding: 0 12px;
  border: 0;
  border-radius: 6px;
  background: var(--color-secondary);
  color: #fff;
  font-weight: 800;
}

.slot-list,
.reference-list,
.recommendation-list,
.comment-list {
  display: grid;
  gap: 9px;
}

.slot-list article {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  gap: 10px;
  padding: 12px 0;
  border-top: 1px solid var(--detail-line);
}

.slot-list article:first-child {
  border-top: 0;
}

.slot-icon {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 6px;
  background: var(--tag-bg);
  color: var(--color-secondary);
}

.slot-list h3 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin: 0 0 5px;
  font-size: 13px;
}

.slot-list h3 small {
  color: var(--color-accent);
  font-size: 9px;
}

.slot-list p,
.slot-list a {
  display: block;
  margin-top: 5px;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.5;
}

.slot-list strong {
  margin-right: 5px;
  color: var(--color-text-main);
}

.slot-list a {
  color: var(--color-secondary);
}

.guide-list {
  display: grid;
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.guide-list li {
  position: relative;
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 10px;
  padding-bottom: 18px;
}

.guide-list li:not(:last-child)::after {
  position: absolute;
  top: 33px;
  bottom: 0;
  left: 18px;
  width: 1px;
  background: var(--color-border);
  content: '';
}

.step-index {
  display: grid;
  width: 36px;
  height: 32px;
  z-index: 1;
  place-items: center;
  border-radius: 5px;
  background: var(--color-primary);
  color: #f1c58c;
  font: 800 12px/1 Georgia, serif;
}

.guide-list h3 {
  margin: 5px 0 7px;
  font-size: 14px;
}

.guide-list p {
  margin: 0 0 7px;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
}

.guide-list .prerequisite {
  color: var(--color-secondary);
  font-size: 10px;
}

.guide-list small {
  display: block;
  margin-bottom: 8px;
  color: var(--color-text-secondary);
}

.guide-list button {
  min-height: 34px;
  padding: 0 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-card-bg);
  color: var(--color-secondary);
  font-size: 11px;
  font-weight: 700;
}

.reference-row {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) 18px;
  gap: 10px;
  align-items: center;
  padding: 9px 0;
  border-top: 1px solid var(--detail-line);
  color: var(--color-text-main);
  text-decoration: none;
}

.reference-row:first-child {
  border-top: 0;
}

.reference-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  overflow: hidden;
  border: 2px solid #fff;
  border-radius: 6px;
  background: #211914;
  color: #fff;
}

.reference-icon.quality-poor { border-color: #9d9d9d; }
.reference-icon.quality-common { border-color: #fff; }
.reference-icon.quality-uncommon { border-color: #1eff00; }
.reference-icon.quality-rare { border-color: #0070dd; }
.reference-icon.quality-epic { border-color: #a335ee; }
.reference-icon.quality-legendary { border-color: #ff8000; }
.reference-icon.quality-artifact { border-color: #e6cc80; }
.reference-icon.quality-heirloom { border-color: #00ccff; }

.reference-row > span:nth-child(2) {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.reference-row small,
.reference-row p {
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 10px;
  line-height: 1.45;
}

.recommendations {
  background: #f6ede4;
}

.discussion {
  padding-bottom: 28px;
}

.comment-composer {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  margin-bottom: 16px;
}

.comment-composer textarea {
  grid-column: 1 / -1;
  width: 100%;
  min-height: 86px;
  resize: vertical;
  padding: 11px 42px 11px 11px;
  border: 1px solid var(--input-border);
  border-radius: 7px;
  outline: 0;
  background: var(--input-bg);
  color: var(--color-text-main);
  font: inherit;
  font-size: 13px;
}

.replying-banner {
  display: flex;
  grid-column: 1 / -1;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 9px;
  border-left: 3px solid var(--color-accent);
  background: rgba(184, 115, 51, 0.08);
  color: var(--color-secondary);
  font-size: 10px;
}

.replying-banner button {
  border: 0;
  background: transparent;
  color: var(--color-text-secondary);
}

.emoji-button {
  position: absolute;
  top: 48px;
  right: 8px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 0;
  background: transparent;
  color: var(--color-secondary);
  font-size: 18px;
}

.comment-submit {
  grid-column: 2;
  min-height: 38px;
  padding: 0 14px;
  border: 0;
  border-radius: 6px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 700;
}

.comment-submit:disabled {
  opacity: 0.5;
}

.comment-list > article {
  padding: 12px 0;
  border-top: 1px solid var(--detail-line);
}

.comment-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.comment-avatar {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
}

.comment-head > span:last-child {
  display: flex;
  flex-direction: column;
}

.comment-head small {
  margin-top: 2px;
  color: var(--color-text-secondary);
  font-size: 9px;
}

.comment-list article > p {
  margin: 9px 0 0 40px;
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
}

.comment-actions {
  display: flex;
  gap: 12px;
  margin: 8px 0 0 40px;
}

.comment-actions button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  border: 0;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 10px;
}

.comment-actions button.active,
.reply-actions button.active {
  color: #a33b32;
  font-weight: 700;
}

.comment-actions button:disabled,
.reply-actions button:disabled {
  opacity: 0.55;
}

.comment-actions button.danger {
  color: #a33b32;
}

.reply-list {
  margin: 10px 0 0 40px;
  padding: 9px 10px;
  border-left: 2px solid var(--color-border);
  background: rgba(75, 54, 33, 0.04);
}

.reply-item {
  padding: 7px 0;
  border-bottom: 1px solid rgba(75, 54, 33, 0.09);
}

.reply-item:first-child {
  padding-top: 0;
}

.reply-item:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.reply-content {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.55;
  white-space: pre-wrap;
}

.reply-content b {
  margin-right: 4px;
  color: var(--color-secondary);
}

.reply-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 11px;
  margin-top: 5px;
}

.reply-actions button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 9px;
}

.reply-actions button.danger {
  color: #a33b32;
}

.action-dock {
  position: fixed;
  right: calc(var(--safe-right, 0px) + 10px);
  bottom: calc(var(--safe-bottom, 0px) + 8px);
  left: calc(var(--safe-left, 0px) + 10px);
  z-index: 80;
  display: grid;
  grid-template-columns: 56px 56px minmax(112px, 1fr) 56px;
  gap: 5px;
  min-height: 62px;
  padding: 5px;
  border: 1px solid rgba(75, 54, 33, 0.14);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 10px 28px rgba(44, 24, 16, 0.22);
  backdrop-filter: blur(14px);
}

.action-dock button {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 2px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--color-text-secondary);
}

.action-dock button i {
  font-size: 19px;
}

.action-dock button span {
  overflow: hidden;
  max-width: 100%;
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action-dock button.active {
  color: var(--color-secondary);
}

.action-dock button.primary {
  background: var(--color-primary);
  color: #fff;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: flex-end;
  background: rgba(44, 24, 16, 0.48);
  backdrop-filter: blur(3px);
}

.list-sheet {
  width: 100%;
  max-height: min(72vh, 620px);
  overflow-y: auto;
  padding: 16px var(--page-gutter) calc(18px + var(--safe-bottom, 0px));
  border-radius: 14px 14px 0 0;
  background: var(--color-panel-bg);
}

.list-sheet header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.list-sheet header small {
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.list-sheet h2 {
  margin-top: 3px;
  font-size: 21px;
}

.list-sheet header button {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid var(--color-border);
  border-radius: 7px;
  background: var(--color-card-bg);
  font-size: 18px;
}

.new-list {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  margin: 14px 0;
}

.new-list input {
  min-width: 0;
  height: 42px;
  padding: 0 10px;
  border: 1px solid var(--input-border);
  border-radius: 7px;
  background: var(--input-bg);
}

.new-list button {
  min-height: 42px;
  padding: 0 11px;
  border: 0;
  border-radius: 7px;
  background: var(--color-primary);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
}

.list-options {
  display: grid;
  gap: 7px;
}

.list-options button {
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 11px;
  border: 1px solid var(--color-border);
  border-radius: 7px;
  background: var(--color-card-bg);
  color: var(--color-text-main);
  text-align: left;
}

.list-options span {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.list-options small {
  overflow: hidden;
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-options em {
  display: grid;
  min-width: 28px;
  height: 24px;
  place-items: center;
  border-radius: 4px;
  background: var(--tag-bg);
  color: var(--tag-text);
  font-size: 10px;
  font-style: normal;
}

.sheet-state {
  display: grid;
  min-height: 140px;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: var(--color-text-secondary);
  font-size: 12px;
}

.sheet-state i {
  color: var(--color-secondary);
  font-size: 28px;
}

.comment-delete-mask {
  position: fixed;
  inset: 0;
  z-index: 2300;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgba(44, 24, 16, 0.5);
}

.comment-delete-dialog {
  width: min(100%, 350px);
  padding: 16px;
  border-radius: 8px;
  background: var(--color-panel-bg);
}

.comment-delete-dialog h2 {
  font-size: 18px;
}

.comment-delete-dialog p {
  margin-top: 8px;
  color: var(--color-text-secondary);
  font-size: 12px;
}

.comment-delete-dialog footer {
  display: flex;
  justify-content: flex-end;
  gap: 7px;
  margin-top: 16px;
}

.comment-delete-dialog button {
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-card-bg);
}

.comment-delete-dialog button.danger {
  border-color: #a33b32;
  background: #a33b32;
  color: #fff;
}

@media (max-width: 350px) {
  .hero-copy h1 {
    font-size: 24px;
  }

  .action-dock {
    grid-template-columns: 50px 50px minmax(100px, 1fr) 50px;
  }
}

@media (max-width: 420px) {
  .verification-strip {
    align-items: stretch;
    flex-direction: column;
  }

  .verification-strip > div:last-child {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .spin {
    animation: none;
  }
}
</style>
