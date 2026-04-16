<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { dialog } from '@/composables/useDialog'
import { useToast } from '@/composables/useToast'
import { getPost } from '@/api/post'
import { getItem, resolveApiUrl } from '@/api/item'
import { getGuild } from '@/api/guild'
import { buildNameStyle } from '@/utils/userNameStyle'
import ImageViewer from '@/components/ImageViewer.vue'
import * as echarts from 'echarts'
import {
  getModeratorStats,
  getPendingPosts,
  getPendingItems,
  getModeratorReports,
  reviewPost,
  reviewItem,
  reviewModeratorReport,
  getAllPosts,
  type PostQueryParams,
  getAllItems,
  deletePostByMod,
  deleteItemByMod,
  hidePostByMod,
  hideItemByMod,
  pinPost,
  featurePost,
  getPendingGuilds,
  reviewGuild,
  getPendingPostCommentImages,
  reviewPostCommentImage,
  getPendingItemCommentImages,
  reviewItemCommentImage,
  getPendingUserAvatars,
  reviewUserAvatar,
  getAllGuilds,
  changeGuildOwner,
  deleteGuildByMod,
  getUsers,
  setUserRole,
  setUserSponsorLevel,
  setUserExperience,
  getModeratorUsers,
  muteUser,
  unmuteUser,
  banUser,
  unbanUser,
  disableUserPosts,
  deleteUserPosts,
  getActionLogs,
  getMetricsHistory,
  getMetricsSummary,
  getMetricsBasic,
  getMetricsBasicHistory,
  broadcastSystemMessage,
  type ModeratorStats,
  type ReviewRequest,
  type ReportReviewItem,
  type ReportReviewRequest,
  type ImageReviewStatus,
  type PostCommentImageReviewItem,
  type ItemCommentImageReviewItem,
  type UserAvatarReviewItem,
  type SafeUser,
  type AdminActionLog,
  type DailyMetrics,
  type MetricsSummary,
  type BasicMetrics,
  type BasicDailyMetrics
} from '@/api/moderator'

interface VisibleReportReason {
  id: number
  reporter_id?: number
  reporter_name?: string
  reason: string
  detail?: string
  created_at?: string
}

const router = useRouter()
const userStore = useUserStore()
const toast = useToast()
const mounted = ref(false)

// 权限检查
const hasAccess = computed(() => userStore.isModerator)
const isAdmin = computed(() => userStore.isAdmin)

// 标签页
const activeTab = ref<'review' | 'manage' | 'admin' | 'logs' | 'metrics'>('review')
type ReviewSubTab = 'posts' | 'items' | 'guilds' | 'reports' | 'postCommentImages' | 'itemCommentImages' | 'userAvatars'
type ManageSubTab = 'posts' | 'items' | 'guilds' | 'users'
type ModeratorSubTab = ReviewSubTab | ManageSubTab
const activeSubTab = ref<ModeratorSubTab>('posts')
const adminSubTab = ref<'moderators' | 'guilds' | 'sponsors' | 'experience' | 'system'>('guilds')

// 系统通知
const systemMessage = ref('')
const systemMessageSending = ref(false)
const systemMessageMaxLength = 512
const systemMessageLength = computed(() => systemMessage.value.length)
const systemMessageTrimmed = computed(() => systemMessage.value.trim())

// 数据
const stats = ref<ModeratorStats | null>(null)
const pendingPosts = ref<any[]>([])
const pendingItems = ref<any[]>([])
const pendingGuilds = ref<any[]>([])
const pendingReports = ref<ReportReviewItem[]>([])
const pendingPostCommentImages = ref<PostCommentImageReviewItem[]>([])
const pendingItemCommentImages = ref<ItemCommentImageReviewItem[]>([])
const pendingUserAvatars = ref<UserAvatarReviewItem[]>([])
const allPosts = ref<any[]>([])
const allItems = ref<any[]>([])
const allGuilds = ref<any[]>([])
const allUsers = ref<SafeUser[]>([])
const loading = ref(false)

// 分页
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 筛选
const filterStatus = ref('')
const filterReviewStatus = ref('')
const filterKeyword = ref('')
const filterPostFlag = ref('')
const filterRole = ref('')
const filterGuildStatus = ref('')
const filterSponsorLevel = ref('')
const reportStatus = ref<'pending' | 'resolved' | 'rejected' | 'all'>('pending')
const reportScope = ref<'user' | 'content' | 'comment'>('content')
const reportSort = ref<'report_count' | 'latest_reported_at'>('report_count')
const reportOrder = ref<'asc' | 'desc'>('desc')
type ReportReviewAction = ReportReviewRequest['action']
const selectedReportIds = ref<number[]>([])
const showReportActionModal = ref(false)
const reportActionType = ref<ReportReviewAction>('delete_content')
const reportActionDuration = ref(24)
const reportActionPermanent = ref(false)
const reportActionComment = ref('')
const reportActionSubmitting = ref(false)
const imageReviewStatus = ref<ImageReviewStatus>('pending')
const imageReviewCommentDrafts = ref<Record<string, string>>({})
const sponsorLevelDrafts = ref<Record<number, number>>({})
const experienceDrafts = ref<Record<number, number>>({})
const sponsorLevelOptions = [
  { value: 0, label: '无赞助' },
  { value: 1, label: 'Lv1 仅鸣谢' },
  { value: 2, label: 'Lv2 昵称样式' },
  { value: 3, label: 'Lv3 个性化' }
]
const selectedReportIdSet = computed(() => new Set(selectedReportIds.value))
const pendingSelectableReports = computed(() => pendingReports.value.filter(report => report.status === 'pending'))
const selectedPendingReports = computed(() => pendingSelectableReports.value.filter(report => selectedReportIdSet.value.has(report.id)))
const hasSelectedReports = computed(() => selectedPendingReports.value.length > 0)
const isAllPendingReportsSelected = computed(() => (
  pendingSelectableReports.value.length > 0
  && pendingSelectableReports.value.every(report => selectedReportIdSet.value.has(report.id))
))
const reportScopeSupportsDelete = computed(() => reportScope.value !== 'user')
const reportActionNeedsDuration = computed(() => (
  reportActionType.value === 'delete_and_mute_user'
  || reportActionType.value === 'delete_and_ban_user'
  || reportActionType.value === 'mute_user'
  || reportActionType.value === 'ban_user'
))
const reportActionDanger = computed(() => (
  reportActionType.value === 'delete_and_ban_user'
  || reportActionType.value === 'ban_user'
))

// 审核预览弹窗
const showPreviewModal = ref(false)
const previewType = ref<'post' | 'item' | 'guild'>('post')
const previewData = ref<any>(null)
const previewLoading = ref(false)
const previewReviewAction = ref<'approve' | 'reject'>('approve')
const previewReviewComment = ref('')

// 审核弹窗（简单版，保留兼容）
const showReviewModal = ref(false)
const reviewTarget = ref<{ type: 'post' | 'item' | 'guild'; id: number; title: string } | null>(null)
const reviewAction = ref<'approve' | 'reject'>('approve')
const reviewComment = ref('')

// 图片查看器
const showImageViewer = ref(false)
const imageViewerImages = ref<string[]>([])
const imageViewerStartIndex = ref(0)

// 更换会长弹窗
const showChangeOwnerModal = ref(false)
const changeOwnerTarget = ref<{ id: number; name: string; currentOwnerId: number } | null>(null)
const newOwnerIdentifier = ref('')  // 用户名或邮箱

// 设置角色弹窗
const showRoleModal = ref(false)
const roleTarget = ref<{
  userId: number
  username: string
  currentRole: string
  newRole: 'user' | 'moderator'
  name_color?: string
  name_bold?: boolean
} | null>(null)

// 用户管理弹窗
const showUserActionModal = ref(false)
const userActionType = ref<'mute' | 'ban' | 'disablePosts' | 'deletePosts'>('mute')
const userActionTarget = ref<SafeUser | null>(null)
const userActionDuration = ref(24) // 默认24小时
const userActionReason = ref('')
const userActionPermanent = ref(false)

// 操作日志
const actionLogs = ref<AdminActionLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsPageSize = ref(20)
const logsActionType = ref('')
const logsTargetType = ref('')

// 数据统计
const metricsHistory = ref<DailyMetrics[]>([])
const metricsSummary = ref<MetricsSummary | null>(null)
const metricsDays = ref(30)
const metricsSubTab = ref<'growth' | 'basic'>('growth')
const metricsLoading = ref(false)
const basicMetrics = ref<BasicMetrics | null>(null)
const basicMetricsHistory = ref<BasicDailyMetrics[]>([])
const basicMetricsLoading = ref(false)
const metricsChartRef = ref<HTMLDivElement | null>(null)
let metricsChart: echarts.ECharts | null = null

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  if (!hasAccess.value) {
    router.push({ name: 'home' })
    return
  }
  await loadStats()
  await loadPendingPosts()
})

onUnmounted(() => {
  disposeMetricsChart()
})

const reviewSubTabs: ReviewSubTab[] = ['posts', 'items', 'guilds', 'reports', 'postCommentImages', 'itemCommentImages', 'userAvatars']
const manageSubTabs: ManageSubTab[] = ['posts', 'items', 'guilds', 'users']

function isReviewSubTab(subTab: ModeratorSubTab): subTab is ReviewSubTab {
  return reviewSubTabs.includes(subTab as ReviewSubTab)
}

function isManageSubTab(subTab: ModeratorSubTab): subTab is ManageSubTab {
  return manageSubTabs.includes(subTab as ManageSubTab)
}

function getImageReviewDraftKey(type: 'post' | 'item' | 'avatar', id: number) {
  return `${type}-${id}`
}

function clearImageReviewDraft(type: 'post' | 'item' | 'avatar', id: number) {
  const key = getImageReviewDraftKey(type, id)
  const next = { ...imageReviewCommentDrafts.value }
  delete next[key]
  imageReviewCommentDrafts.value = next
}

function resolveImageUrl(url?: string | null) {
  return resolveApiUrl(url || '')
}

function openImageViewer(images: string[], startIndex: number = 0) {
  imageViewerImages.value = images.filter(Boolean).map(resolveImageUrl)
  imageViewerStartIndex.value = startIndex
  showImageViewer.value = imageViewerImages.value.length > 0
}

function downloadImageFromViewer(index: number) {
  const url = imageViewerImages.value[index]
  if (!url) return
  const cleaned = url.split('?')[0]
  const match = cleaned.match(/\/([^/]+)$/)
  const fileName = match?.[1] || `image-${Date.now()}.jpg`
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.target = '_blank'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

async function loadStats() {
  try {
    stats.value = await getModeratorStats()
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

async function loadPendingPosts() {
  loading.value = true
  try {
    const res = await getPendingPosts({ page: page.value, page_size: pageSize.value })
    pendingPosts.value = res.posts || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核帖子失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingItems() {
  loading.value = true
  try {
    const res = await getPendingItems({ page: page.value, page_size: pageSize.value })
    pendingItems.value = res.items || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核道具失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadAllPosts() {
  loading.value = true
  try {
    const params: PostQueryParams = {
      page: page.value,
      page_size: pageSize.value,
      status: filterStatus.value || undefined,
      review_status: filterReviewStatus.value || undefined,
      keyword: filterKeyword.value || undefined
    }
    if (filterPostFlag.value === 'pinned') {
      params.is_pinned = true
    } else if (filterPostFlag.value === 'featured') {
      params.is_featured = true
    } else if (filterPostFlag.value === 'normal') {
      params.is_pinned = false
      params.is_featured = false
    }
    const res = await getAllPosts(params)
    allPosts.value = res.posts || []
    total.value = res.total
  } catch (error) {
    console.error('加载帖子失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadAllItems() {
  loading.value = true
  try {
    const res = await getAllItems({
      page: page.value,
      page_size: pageSize.value,
      status: filterStatus.value || undefined,
      review_status: filterReviewStatus.value || undefined,
      keyword: filterKeyword.value || undefined
    })
    allItems.value = res.items || []
    total.value = res.total
  } catch (error) {
    console.error('加载道具失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingGuilds() {
  loading.value = true
  try {
    const res = await getPendingGuilds({ page: page.value, page_size: pageSize.value })
    pendingGuilds.value = res.guilds || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核公会失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingReports() {
  loading.value = true
  try {
    const res = await getModeratorReports({
      page: page.value,
      page_size: pageSize.value,
      status: reportStatus.value,
      target_scope: reportScope.value,
      sort: reportSort.value,
      order: reportOrder.value,
    })
    pendingReports.value = res.reports || []
    const availableIds = new Set(pendingReports.value.filter(report => report.status === 'pending').map(report => report.id))
    selectedReportIds.value = selectedReportIds.value.filter(id => availableIds.has(id))
    total.value = res.total
  } catch (error) {
    console.error('加载举报列表失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingPostCommentImages() {
  loading.value = true
  try {
    const res = await getPendingPostCommentImages({
      page: page.value,
      page_size: pageSize.value,
      status: imageReviewStatus.value
    })
    pendingPostCommentImages.value = res.comments || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核帖子评论图片失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingItemCommentImages() {
  loading.value = true
  try {
    const res = await getPendingItemCommentImages({
      page: page.value,
      page_size: pageSize.value,
      status: imageReviewStatus.value
    })
    pendingItemCommentImages.value = res.comments || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核道具评论图片失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadPendingUserAvatars() {
  loading.value = true
  try {
    const res = await getPendingUserAvatars({
      page: page.value,
      page_size: pageSize.value,
      status: imageReviewStatus.value
    })
    pendingUserAvatars.value = res.users || []
    total.value = res.total
  } catch (error) {
    console.error('加载待审核头像失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadAllGuilds() {
  loading.value = true
  try {
    const res = await getAllGuilds({
      page: page.value,
      page_size: pageSize.value,
      status: filterGuildStatus.value || undefined,
      keyword: filterKeyword.value || undefined
    })
    allGuilds.value = res.guilds || []
    total.value = res.total
  } catch (error) {
    console.error('加载公会失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  loading.value = true
  try {
    const res = await getUsers({
      page: page.value,
      page_size: pageSize.value,
      role: filterRole.value || undefined,
      keyword: filterKeyword.value || undefined,
      sponsor_level: filterSponsorLevel.value === '' ? undefined : Number(filterSponsorLevel.value)
    })
    allUsers.value = res.users || []
    total.value = res.total
    syncSponsorLevelDrafts()
    syncExperienceDrafts()
  } catch (error) {
    console.error('加载用户失败:', error)
  } finally {
    loading.value = false
  }
}

function resolveSponsorLevel(user: SafeUser): number {
  if (typeof user.sponsor_level === 'number') return user.sponsor_level
  return user.is_sponsor ? 2 : 0
}

function syncSponsorLevelDrafts() {
  const drafts: Record<number, number> = {}
  for (const user of allUsers.value) {
    drafts[user.id] = resolveSponsorLevel(user)
  }
  sponsorLevelDrafts.value = drafts
}

function syncExperienceDrafts() {
  const drafts: Record<number, number> = {}
  for (const user of allUsers.value) {
    drafts[user.id] = typeof user.activity_experience === 'number' ? user.activity_experience : 0
  }
  experienceDrafts.value = drafts
}

function formatSponsorLevel(level: number): string {
  if (level <= 0) return '无赞助'
  if (level === 1) return 'Lv1 仅鸣谢'
  if (level === 2) return 'Lv2 昵称样式'
  return 'Lv3 个性化'
}

function formatForumLevel(user: SafeUser): string {
  const level = typeof user.forum_level === 'number' ? user.forum_level : 1
  const name = user.forum_level_name || '新人'
  return `Lv${level} ${name}`
}

function loadReviewSubTab(subTab: ReviewSubTab) {
  if (subTab === 'posts') loadPendingPosts()
  else if (subTab === 'items') loadPendingItems()
  else if (subTab === 'guilds') loadPendingGuilds()
  else if (subTab === 'reports') loadPendingReports()
  else if (subTab === 'postCommentImages') loadPendingPostCommentImages()
  else if (subTab === 'itemCommentImages') loadPendingItemCommentImages()
  else loadPendingUserAvatars()
}

function loadManageSubTab(subTab: ManageSubTab) {
  if (subTab === 'posts') loadAllPosts()
  else if (subTab === 'items') loadAllItems()
  else if (subTab === 'guilds') loadAllGuilds()
  else loadModeratorUsers()
}

function switchTab(tab: 'review' | 'manage' | 'admin' | 'logs' | 'metrics') {
  if (tab === 'metrics' && !isAdmin.value) return
  const previousTab = activeTab.value
  activeTab.value = tab
  page.value = 1
  if (previousTab === 'metrics' && tab !== 'metrics') {
    disposeMetricsChart()
  }
  if (tab === 'review') {
    if (!isReviewSubTab(activeSubTab.value)) {
      activeSubTab.value = 'posts'
    }
    loadReviewSubTab(activeSubTab.value)
  } else if (tab === 'manage') {
    if (!isManageSubTab(activeSubTab.value)) {
      activeSubTab.value = 'posts'
    }
    loadManageSubTab(activeSubTab.value)
  } else if (tab === 'admin') {
    if (adminSubTab.value === 'moderators' || adminSubTab.value === 'sponsors' || adminSubTab.value === 'experience') {
      loadUsers()
    } else if (adminSubTab.value === 'guilds') {
      loadAllGuilds()
    }
  } else if (tab === 'logs') {
    loadActionLogs()
  } else if (tab === 'metrics') {
    if (metricsSubTab.value === 'basic') loadBasicMetrics()
    else loadMetrics()
  }
}

function switchSubTab(subTab: ModeratorSubTab) {
  activeSubTab.value = subTab
  page.value = 1
  if (activeTab.value === 'review') {
    if (isReviewSubTab(subTab)) {
      loadReviewSubTab(subTab)
    }
  } else if (activeTab.value === 'manage') {
    if (isManageSubTab(subTab)) {
      loadManageSubTab(subTab)
    }
  }
}

function switchAdminSubTab(subTab: 'moderators' | 'guilds' | 'sponsors' | 'experience' | 'system') {
  adminSubTab.value = subTab
  page.value = 1
  if (subTab === 'moderators') {
    filterSponsorLevel.value = ''
    loadUsers()
  } else if (subTab === 'sponsors') {
    filterRole.value = ''
    loadUsers()
  } else if (subTab === 'experience') {
    filterRole.value = ''
    filterSponsorLevel.value = ''
    loadUsers()
  } else if (subTab === 'system') {
    filterRole.value = ''
    filterSponsorLevel.value = ''
  } else {
    filterRole.value = ''
    filterSponsorLevel.value = ''
    loadAllGuilds()
  }
}

async function sendSystemMessage() {
  if (!isAdmin.value) return
  const content = systemMessageTrimmed.value
  if (!content) {
    toast.error('请输入系统通知内容')
    return
  }

  const confirmed = await dialog.confirm({
    title: '发送系统通知',
    message: '该通知将发送给所有用户，确认继续？',
    type: 'warning'
  })
  if (!confirmed) return

  systemMessageSending.value = true
  try {
    const res = await broadcastSystemMessage(content)
    toast.success(`已发送给 ${res.count} 位用户`)
    systemMessage.value = ''
  } catch (error) {
    toast.error('发送失败: ' + (error as Error).message)
  } finally {
    systemMessageSending.value = false
  }
}

// ========== 操作日志 ==========
async function loadActionLogs() {
  try {
    const res = await getActionLogs({
      page: logsPage.value,
      page_size: logsPageSize.value,
      action_type: logsActionType.value || undefined,
      target_type: logsTargetType.value || undefined
    })
    actionLogs.value = res.logs
    logsTotal.value = res.total
  } catch (error) {
    console.error('加载操作日志失败:', error)
  }
}

function formatActionType(type: string): string {
  const map: Record<string, string> = {
    'review_post': '审核帖子',
    'review_report': '处理举报',
    'delete_post': '删除帖子',
    'hide_post': '屏蔽帖子',
    'pin_post': '置顶帖子',
    'feature_post': '精华帖子',
    'review_item': '审核作品',
    'delete_item': '删除作品',
    'hide_item': '屏蔽作品',
    'review_guild': '审核公会',
    'review_post_comment_image': '审核帖子评论图片',
    'review_item_comment_image': '审核道具评论图片',
    'review_user_avatar': '审核用户头像',
    'delete_guild': '删除公会',
    'change_guild_owner': '更换会长',
    'mute_user': '禁言用户',
    'unmute_user': '解除禁言',
    'ban_user': '封禁用户',
    'unban_user': '解除封禁',
    'set_role': '设置角色',
    'set_sponsor': '设置赞助',
    'set_experience': '设置经验',
    'disable_posts': '禁用帖子',
    'delete_posts': '删除用户帖子',
    'broadcast_notification': '系统通知'
  }
  return map[type] || type
}

function getReportTargetLabel(type: string) {
  if (type === 'post') return '帖子'
  if (type === 'item') return '作品'
  if (type === 'user') return '用户'
  if (type === 'comment') return '帖子评论'
  if (type === 'item_comment') return '作品评论'
  return type
}

function getReportStatusLabel(status: string) {
  if (status === 'resolved') return '已处置'
  if (status === 'rejected') return '已驳回'
  return '待处理'
}

function getReportScopeLabel(scope: 'user' | 'content' | 'comment') {
  if (scope === 'user') return '举报用户'
  if (scope === 'comment') return '举报评论'
  return '举报帖子/道具'
}

function getReportEmptyText() {
  return `暂无${getReportScopeLabel(reportScope.value)}记录`
}

function switchReportScope(scope: 'user' | 'content' | 'comment') {
  if (reportScope.value === scope) return
  reportScope.value = scope
  selectedReportIds.value = []
  closeReportActionModal()
  page.value = 1
  loadPendingReports()
}

function handleReportFilterChange() {
  selectedReportIds.value = []
  closeReportActionModal()
  page.value = 1
  loadPendingReports()
}

function formatReportReason(reason: string) {
  const reasonMap: Record<string, string> = {
    spam: '垃圾信息或刷屏',
    abuse: '辱骂、人身攻击',
    fraud: '诈骗或恶意引流',
    sexual: '色情或不适内容',
    illegal: '违法违规内容',
    other: '其他问题',
    block_user: '用户已执行屏蔽',
  }
  return reasonMap[reason] || reason
}

function normalizeVisibleReportReason(reason: Partial<VisibleReportReason> & { reporterId?: number }, fallbackCreatedAt?: string): VisibleReportReason {
  const reporterId = typeof reason.reporter_id === 'number'
    ? reason.reporter_id
    : (typeof reason.reporterId === 'number' ? reason.reporterId : undefined)

  return {
    id: typeof reason.id === 'number' ? reason.id : 0,
    reporter_id: reporterId,
    reporter_name: reason.reporter_name?.trim() || '',
    reason: reason.reason || 'other',
    detail: reason.detail || '',
    created_at: reason.created_at || fallbackCreatedAt,
  }
}

function getVisibleReportReasons(report: ReportReviewItem): VisibleReportReason[] {
  const reportAny = report as ReportReviewItem & {
    reason?: string
    detail?: string
    reporter_id?: number
    reporter_name?: string
    created_at?: string
    reporterId?: number
    reports?: Array<VisibleReportReason & { reporterId?: number }>
  }

  if (Array.isArray(report.reasons) && report.reasons.length > 0) {
    return report.reasons.map(reason => normalizeVisibleReportReason(reason, report.latest_reported_at))
  }
  if (Array.isArray(reportAny.reports) && reportAny.reports.length > 0) {
    return reportAny.reports.map(reason => normalizeVisibleReportReason(reason, report.latest_reported_at))
  }
  if (reportAny.reason || reportAny.detail) {
    return [normalizeVisibleReportReason({
      id: report.id,
      reporter_id: reportAny.reporter_id,
      reporterId: reportAny.reporterId,
      reporter_name: reportAny.reporter_name,
      reason: reportAny.reason || 'other',
      detail: reportAny.detail || '',
      created_at: reportAny.created_at || report.latest_reported_at,
    }, report.latest_reported_at)]
  }
  return []
}

function getReportReporterLabel(reason: VisibleReportReason) {
  const reporterName = reason.reporter_name?.trim()
  if (reporterName) return reporterName
  if (typeof reason.reporter_id === 'number' && reason.reporter_id > 0) return `用户#${reason.reporter_id}`
  return '未知用户'
}

function isBlockUserReason(reason: string) {
  return reason === 'block_user'
}

function getReportReasonSummaryLabel(reason: VisibleReportReason) {
  return isBlockUserReason(reason.reason) ? '触发类型' : '举报原因'
}

function getReportReasonSummaryValue(reason: VisibleReportReason) {
  return isBlockUserReason(reason.reason) ? '用户主动屏蔽' : formatReportReason(reason.reason)
}

function getReportReasonDetailLabel(reason: VisibleReportReason) {
  return isBlockUserReason(reason.reason) ? '用户备注' : '备注说明'
}

function hasReportPreview(report: ReportReviewItem) {
  return Boolean(report.target_preview_image || report.target_preview_text)
}

function toggleReportSelection(reportId: number) {
  const next = new Set(selectedReportIds.value)
  if (next.has(reportId)) next.delete(reportId)
  else next.add(reportId)
  selectedReportIds.value = Array.from(next)
}

function toggleSelectAllReports() {
  if (isAllPendingReportsSelected.value) {
    selectedReportIds.value = []
    return
  }
  selectedReportIds.value = pendingSelectableReports.value.map(report => report.id)
}

function clearReportSelection() {
  selectedReportIds.value = []
}

function getReportMuteActionLabel() {
  return reportScope.value === 'user' ? '禁言用户' : '删除并禁言用户'
}

function getReportBanActionLabel() {
  return reportScope.value === 'user' ? '封禁用户' : '删除并封禁用户'
}

function getReportActionTitle(action: ReportReviewAction = reportActionType.value) {
  if (action === 'delete_content') return '删除内容'
  if (action === 'delete_and_mute_user') return '删除并禁言用户'
  if (action === 'delete_and_ban_user') return '删除并封禁用户'
  if (action === 'mute_user') return '禁言用户'
  if (action === 'ban_user') return '封禁用户'
  return '驳回举报'
}

function getReportActionDescription(action: ReportReviewAction = reportActionType.value) {
  const count = selectedPendingReports.value.length
  if (action === 'delete_content') return `将删除选中的 ${count} 条被举报内容，并把对应举报标记为已处置。`
  if (action === 'delete_and_mute_user') return `将删除选中的 ${count} 条被举报内容，并对对应作者执行禁言。`
  if (action === 'delete_and_ban_user') return `将删除选中的 ${count} 条被举报内容，并对对应作者执行封禁。`
  if (action === 'mute_user') return `将对选中的 ${count} 位被举报用户执行禁言。`
  if (action === 'ban_user') return `将对选中的 ${count} 位被举报用户执行封禁。`
  return `将驳回选中的 ${count} 条举报，不会删除内容，也不会处罚用户。`
}

function getReportActionWarning() {
  if (reportActionType.value === 'delete_content') return '删除后内容会立即从前台消失，请确认举报依据充分。'
  if (reportActionType.value === 'delete_and_mute_user') return '删除内容后，作者将无法继续发帖和评论。'
  if (reportActionType.value === 'delete_and_ban_user') return '删除内容后，作者将被禁止登录；如果选择永久，需谨慎执行。'
  if (reportActionType.value === 'mute_user') return '禁言用户后，对方将无法继续发帖和评论。'
  if (reportActionType.value === 'ban_user') return '封禁用户后，对方将无法继续登录。'
  return '驳回后举报将被关闭，内容与账号状态不会发生变化。'
}

function closeReportActionModal() {
  showReportActionModal.value = false
  reportActionComment.value = ''
  reportActionDuration.value = 24
  reportActionPermanent.value = false
  reportActionSubmitting.value = false
}

function openReportActionModal(action: ReportReviewAction, report?: ReportReviewItem) {
  if (report) {
    selectedReportIds.value = report.status === 'pending' ? [report.id] : []
  }
  if (!hasSelectedReports.value) {
    toast.error('请先勾选要处理的举报记录')
    return
  }
  reportActionType.value = action
  reportActionDuration.value = 24
  reportActionPermanent.value = false
  reportActionComment.value = ''
  showReportActionModal.value = true
}

function openReportTarget(report: ReportReviewItem) {
  if (report.target_type === 'post' || report.target_type === 'item') {
    openPreview(report.target_type, report.target_id)
    return
  }
  if (report.target_type === 'comment' && report.parent_target_id) {
    router.push({
      name: 'post-detail',
      params: { id: report.parent_target_id },
      query: { comment: String(report.target_id) },
    })
    return
  }
  if (report.target_type === 'item_comment' && report.parent_target_id) {
    router.push({
      name: 'item-detail',
      params: { id: report.parent_target_id },
    })
    return
  }
  router.push({ name: 'user-profile', params: { id: report.target_id } })
}

function getReportReasonDetail(detail?: string) {
  const trimmed = detail?.trim()
  return trimmed || '未填写补充说明'
}

async function submitReportAction() {
  if (!hasSelectedReports.value) {
    toast.error('请先勾选要处理的举报记录')
    return
  }
  if (reportActionNeedsDuration.value && !reportActionPermanent.value && (!Number.isFinite(reportActionDuration.value) || reportActionDuration.value < 1)) {
    toast.error('请输入有效的处理时长')
    return
  }

  reportActionSubmitting.value = true
  const duration = reportActionPermanent.value ? 0 : reportActionDuration.value
  const comment = reportActionComment.value.trim()
  let successCount = 0
  let failureMessage = ''

  for (const report of selectedPendingReports.value) {
    try {
      await reviewModeratorReport(report.id, {
        action: reportActionType.value,
        duration: reportActionNeedsDuration.value ? duration : undefined,
        comment: comment || undefined,
      })
      successCount += 1
    } catch (error) {
      failureMessage = (error as Error).message
      break
    }
  }

  reportActionSubmitting.value = false
  if (successCount === 0 && failureMessage) {
    toast.error('处理举报失败: ' + failureMessage)
    return
  }

  const totalCount = selectedPendingReports.value.length
  closeReportActionModal()
  clearReportSelection()
  await loadStats()
  await loadPendingReports()
  if (failureMessage) {
    toast.error(`已处理 ${successCount}/${totalCount} 条举报，剩余失败：${failureMessage}`)
    return
  }
  toast.success(`已处理 ${successCount} 条举报`)
}

function formatLogDetails(log: AdminActionLog): string {
  if (!log.details) return '-'
  try {
    const d = JSON.parse(log.details)
    const parts: string[] = []

    // 审核操作
    if (d.action) {
      const actionMap: Record<string, string> = {
        approve: '通过',
        reject: log.action_type === 'review_report' ? '驳回举报' : '拒绝',
        delete_content: '删除内容',
        delete_and_mute_user: '删除并禁言用户',
        delete_and_ban_user: '删除并封禁用户',
        mute_user: '禁言用户',
        ban_user: '封禁用户',
      }
      parts.push(actionMap[d.action] || d.action)
    }

    // 封禁/禁言时长
    if (d.duration) {
      parts.push(d.duration)
    }

    // 原因
    if (d.reason) {
      parts.push(`原因: ${d.reason}`)
    }

    // 审核意见
    if (d.comment) {
      parts.push(`意见: ${d.comment}`)
    }
    if (d.review_comment) {
      parts.push(`处理结果: ${d.review_comment}`)
    }

    // 置顶/精华状态
    if (d.is_pinned !== undefined) {
      parts.push(d.is_pinned ? '置顶' : '取消置顶')
    }
    if (d.is_featured !== undefined) {
      parts.push(d.is_featured ? '加精' : '取消加精')
    }

    // 更换会长
    if (d.new_owner_name) {
      parts.push(`新会长: ${d.new_owner_name}`)
    }

    // 设置角色
    if (d.new_role) {
      parts.push(`新角色: ${d.new_role === 'moderator' ? '版主' : '普通用户'}`)
    }

    // 赞助者权限
    if (d.sponsor_level !== undefined) {
      parts.push(`赞助等级: ${formatSponsorLevel(d.sponsor_level)}`)
    } else if (d.is_sponsor !== undefined) {
      parts.push(`赞助者: ${d.is_sponsor ? '是' : '否'}`)
    }

    if (d.before_experience !== undefined || d.after_experience !== undefined) {
      parts.push(`经验: ${d.before_experience ?? '-'} -> ${d.after_experience ?? '-'}`)
    }
    if (d.before_level !== undefined || d.after_level !== undefined) {
      parts.push(`等级: Lv${d.before_level ?? '-'} -> Lv${d.after_level ?? '-'}`)
    }

    // 影响数量
    if (d.affected_count !== undefined) {
      parts.push(`${d.affected_count} 条`)
    }
    if (d.recipient_count !== undefined) {
      parts.push(`发送 ${d.recipient_count} 人`)
    }
    if (d.content_preview) {
      parts.push(`内容: ${d.content_preview}`)
    }

    return parts.length > 0 ? parts.join(' | ') : '-'
  } catch {
    return '-'
  }
}

// ========== 数据统计 ==========
function disposeMetricsChart() {
  if (metricsChart) {
    metricsChart.dispose()
    metricsChart = null
  }
}

async function loadMetrics() {
  if (!isAdmin.value) return
  metricsLoading.value = true
  try {
    const [historyRes, summaryRes] = await Promise.all([
      getMetricsHistory(metricsDays.value),
      getMetricsSummary()
    ])
    metricsHistory.value = historyRes.metrics
    metricsSummary.value = summaryRes
    // 下一帧渲染图表
    if (activeTab.value === 'metrics' && metricsSubTab.value === 'growth') {
      setTimeout(() => initMetricsChart(), 100)
    }
  } catch (error) {
    console.error('加载统计数据失败:', error)
  } finally {
    metricsLoading.value = false
  }
}

async function loadBasicMetrics() {
  if (!isAdmin.value) return
  basicMetricsLoading.value = true
  try {
    const [basicRes, historyRes] = await Promise.all([
      getMetricsBasic(),
      getMetricsBasicHistory(metricsDays.value)
    ])
    basicMetrics.value = basicRes
    basicMetricsHistory.value = historyRes.metrics
    if (activeTab.value === 'metrics' && metricsSubTab.value === 'basic') {
      setTimeout(() => initMetricsChart(), 100)
    }
  } catch (error) {
    console.error('加载基础监控数据失败:', error)
  } finally {
    basicMetricsLoading.value = false
  }
}

function switchMetricsSubTab(subTab: 'growth' | 'basic') {
  if (metricsSubTab.value === subTab) return
  metricsSubTab.value = subTab
  disposeMetricsChart()
  if (subTab === 'growth') {
    loadMetrics()
  } else {
    loadBasicMetrics()
  }
}

function initMetricsChart() {
  if (!metricsChartRef.value) return

  if (metricsChart) {
    const currentDom = metricsChart.getDom()
    if (metricsChart.isDisposed() || currentDom !== metricsChartRef.value) {
      metricsChart.dispose()
      metricsChart = null
    }
  }

  if (!metricsChart) {
    metricsChart = echarts.init(metricsChartRef.value)
  }

  const isGrowth = metricsSubTab.value === 'growth'
  let dates: string[] = []
  let legendData: string[] = []
  let series: echarts.SeriesOption[] = []
  const palette = {
    linePrimary: getComputedStyle(document.documentElement).getPropertyValue('--color-secondary').trim() || '#804030',
    lineAccent: getComputedStyle(document.documentElement).getPropertyValue('--color-accent').trim() || '#4682B4',
    lineAux: getComputedStyle(document.documentElement).getPropertyValue('--color-primary').trim() || '#9370DB',
    lineSuccess: getComputedStyle(document.documentElement).getPropertyValue('--color-success').trim() || '#6B9B6B',
    textSecondary: getComputedStyle(document.documentElement).getPropertyValue('--color-text-secondary').trim() || '#8D7B68',
    border: getComputedStyle(document.documentElement).getPropertyValue('--color-border').trim() || '#E5D4C1',
    borderLight: getComputedStyle(document.documentElement).getPropertyValue('--color-border-light').trim() || '#F5EBE0',
  }

  if (isGrowth) {
    dates = metricsHistory.value.map((m: DailyMetrics) => m.date.slice(5)) // 只显示月-日
    const newUsers = metricsHistory.value.map((m: DailyMetrics) => m.new_users)
    const newPosts = metricsHistory.value.map((m: DailyMetrics) => m.new_posts)
    const newItems = metricsHistory.value.map((m: DailyMetrics) => m.new_items)
    const newGuilds = metricsHistory.value.map((m: DailyMetrics) => m.new_guilds)
    legendData = ['新增用户', '新增帖子', '新增作品', '新增公会']
    series = [
      { name: '新增用户', type: 'line', data: newUsers, smooth: true, itemStyle: { color: palette.linePrimary } },
      { name: '新增帖子', type: 'line', data: newPosts, smooth: true, itemStyle: { color: palette.lineAccent } },
      { name: '新增作品', type: 'line', data: newItems, smooth: true, itemStyle: { color: palette.lineAux } },
      { name: '新增公会', type: 'line', data: newGuilds, smooth: true, itemStyle: { color: palette.lineSuccess } }
    ]
  } else {
    dates = basicMetricsHistory.value.map((m: BasicDailyMetrics) => m.date.slice(5))
    const newArchives = basicMetricsHistory.value.map((m: BasicDailyMetrics) => m.new_story_archives)
    const newEntries = basicMetricsHistory.value.map((m: BasicDailyMetrics) => m.new_story_entries)
    const newBackups = basicMetricsHistory.value.map((m: BasicDailyMetrics) => m.new_profile_backups)
    legendData = ['新增剧情归档', '新增归档条目', '新增人物卡备份']
    series = [
      { name: '新增剧情归档', type: 'line', data: newArchives, smooth: true, itemStyle: { color: palette.linePrimary } },
      { name: '新增归档条目', type: 'line', data: newEntries, smooth: true, itemStyle: { color: palette.lineAccent } },
      { name: '新增人物卡备份', type: 'line', data: newBackups, smooth: true, itemStyle: { color: palette.lineSuccess } }
    ]
  }

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' }
    },
    legend: {
      data: legendData,
      textStyle: { color: palette.textSecondary }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: dates,
      axisLine: { lineStyle: { color: palette.border } },
      axisLabel: { color: palette.textSecondary }
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: palette.border } },
      axisLabel: { color: palette.textSecondary },
      splitLine: { lineStyle: { color: palette.borderLight } }
    },
    series
  }

  metricsChart.setOption(option)
  metricsChart.resize()
}

// 打开预览弹窗
async function openPreview(type: 'post' | 'item' | 'guild', id: number) {
  previewType.value = type
  previewData.value = null
  previewReviewAction.value = 'approve'
  previewReviewComment.value = ''
  showPreviewModal.value = true
  previewLoading.value = true

  try {
    if (type === 'post') {
      const res = await getPost(id)
      previewData.value = res.post
    } else if (type === 'item') {
      const res: any = await getItem(id)
      // 处理不同的 API 返回格式
      let item, author, tags
      if (res.code === 0 && res.data) {
        // 格式: { code: 0, data: { item, author, tags } }
        item = res.data.item
        author = res.data.author
        tags = res.data.tags
      } else if (res.item) {
        // 格式: { item, author, tags }
        item = res.item
        author = res.author
        tags = res.tags
      } else {
        // 直接是 item 对象
        item = res
      }
      previewData.value = {
        ...item,
        author_name: author?.username || item?.author_name,
        author_name_color: author?.name_color || item?.author_name_color,
        author_name_bold: author?.name_bold ?? item?.author_name_bold,
        tags: tags || []
      }
    } else {
      const res = await getGuild(id)
      const guildFromList = pendingGuilds.value.find(g => g.id === id) || allGuilds.value.find(g => g.id === id)
      previewData.value = {
        ...res.guild,
        owner_name: guildFromList?.owner_name,
        owner_name_color: guildFromList?.owner_name_color,
        owner_name_bold: guildFromList?.owner_name_bold
      }
    }
  } catch (error: any) {
    console.error('加载详情失败:', error)
    // 如果是 404 错误，说明内容已被删除，刷新列表
    if (error.message?.includes('不存在')) {
      showPreviewModal.value = false
      // 刷新对应的列表
      if (type === 'item') {
        loadPendingItems()
        loadAllItems()
      } else if (type === 'post') {
        loadPendingPosts()
        loadAllPosts()
      } else {
        loadPendingGuilds()
        loadAllGuilds()
      }
    } else {
      alert('加载详情失败: ' + error.message)
      showPreviewModal.value = false
    }
  } finally {
    previewLoading.value = false
  }
}

// 提交预览审核
async function submitPreviewReview() {
  if (!previewData.value) return

  const data: ReviewRequest = {
    action: previewReviewAction.value,
    comment: previewReviewComment.value
  }

  try {
    if (previewType.value === 'post') {
      await reviewPost(previewData.value.id, data)
    } else if (previewType.value === 'item') {
      await reviewItem(previewData.value.id, data)
    } else {
      await reviewGuild(previewData.value.id, data)
    }
    showPreviewModal.value = false
    await loadStats()
    // 刷新对应列表
    if (previewType.value === 'post') await loadPendingPosts()
    else if (previewType.value === 'item') await loadPendingItems()
    else await loadPendingGuilds()
  } catch (error) {
    console.error('审核失败:', error)
    alert('审核失败: ' + (error as Error).message)
  }
}

// 快速审核（直接通过/拒绝）
async function quickReview(type: 'post' | 'item' | 'guild', id: number, action: 'approve' | 'reject') {
  const actionText = action === 'approve' ? '通过' : '拒绝'
  const typeText = type === 'post' ? '帖子' : type === 'item' ? '作品' : '公会'

  const confirmed = await dialog.confirm({
    title: `${actionText}${typeText}`,
    message: `确定要${actionText}这个${typeText}吗？`,
    type: action === 'approve' ? 'success' : 'warning',
    confirmText: actionText,
    cancelText: '取消'
  })

  if (!confirmed) return

  const data: ReviewRequest = { action, comment: '' }

  try {
    if (type === 'post') {
      await reviewPost(id, data)
    } else if (type === 'item') {
      await reviewItem(id, data)
    } else {
      await reviewGuild(id, data)
    }
    await loadStats()
    if (type === 'post') await loadPendingPosts()
    else if (type === 'item') await loadPendingItems()
    else await loadPendingGuilds()
  } catch (error) {
    console.error('审核失败:', error)
    alert('审核失败: ' + (error as Error).message)
  }
}

function openReviewModal(type: 'post' | 'item' | 'guild', id: number, title: string) {
  reviewTarget.value = { type, id, title }
  reviewAction.value = 'approve'
  reviewComment.value = ''
  showReviewModal.value = true
}

async function submitReview() {
  if (!reviewTarget.value) return

  const data: ReviewRequest = {
    action: reviewAction.value,
    comment: reviewComment.value
  }

  try {
    if (reviewTarget.value.type === 'post') {
      await reviewPost(reviewTarget.value.id, data)
    } else if (reviewTarget.value.type === 'item') {
      await reviewItem(reviewTarget.value.id, data)
    } else {
      await reviewGuild(reviewTarget.value.id, data)
    }
    showReviewModal.value = false
    await loadStats()
    if (activeSubTab.value === 'posts') await loadPendingPosts()
    else if (activeSubTab.value === 'items') await loadPendingItems()
    else await loadPendingGuilds()
  } catch (error) {
    console.error('审核失败:', error)
    alert('审核失败: ' + (error as Error).message)
  }
}

function handleImageReviewStatusChange() {
  page.value = 1
  if (activeTab.value === 'review' && isReviewSubTab(activeSubTab.value)) {
    loadReviewSubTab(activeSubTab.value)
  }
}

async function quickReviewImage(
  type: 'postComment' | 'itemComment' | 'avatar',
  id: number,
  action: 'approve' | 'reject',
  comment?: string
) {
  const actionText = action === 'approve' ? '通过' : '拒绝'
  const typeText = type === 'postComment' ? '帖子评论图片' : type === 'itemComment' ? '道具评论图片' : '头像'

  const confirmed = await dialog.confirm({
    title: `${actionText}${typeText}`,
    message: `确定要${actionText}该${typeText}吗？`,
    type: action === 'approve' ? 'success' : 'warning',
    confirmText: actionText,
    cancelText: '取消'
  })
  if (!confirmed) return

  const payload: ReviewRequest = {
    action,
    comment: (comment || '').trim()
  }

  try {
    if (type === 'postComment') {
      await reviewPostCommentImage(id, payload)
      clearImageReviewDraft('post', id)
      await loadPendingPostCommentImages()
    } else if (type === 'itemComment') {
      await reviewItemCommentImage(id, payload)
      clearImageReviewDraft('item', id)
      await loadPendingItemCommentImages()
    } else {
      await reviewUserAvatar(id, payload)
      clearImageReviewDraft('avatar', id)
      await loadPendingUserAvatars()
    }
    toast.success(`${actionText}成功`)
  } catch (error) {
    console.error('图片审核失败:', error)
    toast.error('审核失败: ' + (error as Error).message)
  }
}

async function handleDeletePost(id: number) {
  const confirmed = await dialog.confirm({
    title: '删除帖子',
    message: '确定要删除这篇帖子吗？此操作不可恢复。',
    type: 'error',
    confirmText: '删除',
    cancelText: '取消'
  })
  if (!confirmed) return
  try {
    await deletePostByMod(id)
    await loadAllPosts()
    await loadStats()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败: ' + (error as Error).message)
  }
}

async function handleHidePost(id: number) {
  const confirmed = await dialog.confirm({
    title: '屏蔽帖子',
    message: '确定要屏蔽这篇帖子吗？帖子将被打回待审核状态。',
    type: 'warning',
    confirmText: '屏蔽',
    cancelText: '取消'
  })
  if (!confirmed) return
  try {
    await hidePostByMod(id)
    await loadAllPosts()
    await loadStats()
  } catch (error) {
    console.error('屏蔽失败:', error)
    alert('屏蔽失败: ' + (error as Error).message)
  }
}

async function handlePinPost(id: number, isPinned: boolean) {
  try {
    await pinPost(id)
    await loadAllPosts()
  } catch (error) {
    console.error('操作失败:', error)
    alert('操作失败: ' + (error as Error).message)
  }
}

async function handleFeaturePost(id: number, isFeatured: boolean) {
  try {
    await featurePost(id)
    await loadAllPosts()
  } catch (error) {
    console.error('操作失败:', error)
    alert('操作失败: ' + (error as Error).message)
  }
}

async function handleDeleteItem(id: number) {
  const confirmed = await dialog.confirm({
    title: '删除作品',
    message: '确定要删除这个作品吗？此操作不可恢复。',
    type: 'error',
    confirmText: '删除',
    cancelText: '取消'
  })
  if (!confirmed) return
  try {
    await deleteItemByMod(id)
    await loadAllItems()
    await loadStats()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败: ' + (error as Error).message)
  }
}

async function handleHideItem(id: number) {
  const confirmed = await dialog.confirm({
    title: '屏蔽作品',
    message: '确定要屏蔽这个作品吗？作品将被打回待审核状态。',
    type: 'warning',
    confirmText: '屏蔽',
    cancelText: '取消'
  })
  if (!confirmed) return
  try {
    await hideItemByMod(id)
    await loadAllItems()
    await loadStats()
  } catch (error) {
    console.error('屏蔽失败:', error)
    alert('屏蔽失败: ' + (error as Error).message)
  }
}

async function handleDeleteGuild(id: number) {
  const confirmed = await dialog.confirm({
    title: '删除公会',
    message: '确定要删除这个公会吗？此操作不可恢复。',
    type: 'error',
    confirmText: '删除',
    cancelText: '取消'
  })
  if (!confirmed) return
  try {
    await deleteGuildByMod(id)
    await loadAllGuilds()
    await loadStats()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败: ' + (error as Error).message)
  }
}

function openChangeOwnerModal(id: number, name: string, currentOwnerId: number) {
  changeOwnerTarget.value = { id, name, currentOwnerId }
  newOwnerIdentifier.value = ''
  showChangeOwnerModal.value = true
}

async function submitChangeOwner() {
  if (!changeOwnerTarget.value || !newOwnerIdentifier.value) return
  try {
    const result = await changeGuildOwner(changeOwnerTarget.value.id, { new_owner_name: newOwnerIdentifier.value })
    showChangeOwnerModal.value = false
    alert(`会长已更换为 ${result.new_owner.username}`)
    await loadAllGuilds()
  } catch (error) {
    console.error('更换会长失败:', error)
    alert('更换会长失败: ' + (error as Error).message)
  }
}

function openRoleModal(
  userId: number,
  username: string,
  currentRole: string,
  newRole: 'user' | 'moderator',
  nameColor?: string,
  nameBold?: boolean
) {
  roleTarget.value = { userId, username, currentRole, newRole, name_color: nameColor, name_bold: nameBold }
  showRoleModal.value = true
}

async function submitRoleChange() {
  if (!roleTarget.value) return
  try {
    await setUserRole(roleTarget.value.userId, roleTarget.value.newRole)
    showRoleModal.value = false
    await loadUsers()
  } catch (error) {
    console.error('设置角色失败:', error)
    alert('设置角色失败: ' + (error as Error).message)
  }
}

async function applySponsorLevel(user: SafeUser) {
  const currentLevel = resolveSponsorLevel(user)
  const nextLevel = sponsorLevelDrafts.value[user.id] ?? currentLevel
  if (nextLevel === currentLevel) return

  const nextLabel = formatSponsorLevel(nextLevel)
  const actionText = nextLevel === 0 ? '取消赞助' : `设置为 ${nextLabel}`
  const confirmed = await dialog.confirm({
    title: actionText,
    message: `确定要将 ${user.username} ${actionText} 吗？`,
    type: nextLevel === 0 ? 'warning' : 'success',
    confirmText: actionText,
    cancelText: '取消'
  })

  if (!confirmed) {
    sponsorLevelDrafts.value[user.id] = currentLevel
    return
  }

  try {
    await setUserSponsorLevel(user.id, nextLevel)
    await loadUsers()
  } catch (error) {
    sponsorLevelDrafts.value[user.id] = currentLevel
    console.error('设置赞助等级失败:', error)
    alert('设置赞助等级失败: ' + (error as Error).message)
  }
}

async function applyUserExperience(user: SafeUser) {
  const currentExperience = typeof user.activity_experience === 'number' ? user.activity_experience : 0
  const nextExperienceRaw = experienceDrafts.value[user.id]
  const nextExperience = Number.isFinite(nextExperienceRaw) ? Math.max(0, Math.floor(nextExperienceRaw)) : currentExperience
  experienceDrafts.value[user.id] = nextExperience

  if (nextExperience === currentExperience) return

  const confirmed = await dialog.confirm({
    title: '设置社区经验值',
    message: `确定要将 ${user.username} 的社区经验值从 ${currentExperience} 调整为 ${nextExperience} 吗？`,
    type: nextExperience >= currentExperience ? 'success' : 'warning',
    confirmText: '应用',
    cancelText: '取消'
  })

  if (!confirmed) {
    experienceDrafts.value[user.id] = currentExperience
    return
  }

  try {
    await setUserExperience(user.id, nextExperience)
    toast.success('社区经验值已更新')
    await loadUsers()
  } catch (error) {
    experienceDrafts.value[user.id] = currentExperience
    console.error('设置社区经验值失败:', error)
    toast.error('设置社区经验值失败: ' + (error as Error).message)
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN')
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    draft: '草稿',
    pending: '待发布',
    published: '已发布',
    removed: '已移除'
  }
  return map[status] || status
}

function getReviewStatusLabel(status: string) {
  const map: Record<string, string> = {
    pending: '待审核',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return map[status] || status
}

function getGuildStatusLabel(status: string) {
  const map: Record<string, string> = {
    pending: '待审核',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return map[status] || status
}

function getRoleLabel(role: string) {
  const map: Record<string, string> = {
    user: '普通用户',
    moderator: '版主',
    admin: '管理员'
  }
  return map[role] || role
}

function getItemTypeLabel(type: string) {
  const map: Record<string, string> = {
    item: '道具',
    document: '文档',
    campaign: '剧本',
    artwork: '画作'
  }
  return map[type] || type
}

// ========== 用户管理功能 ==========

function openUserActionModal(user: SafeUser, action: 'mute' | 'ban' | 'disablePosts' | 'deletePosts') {
  userActionTarget.value = user
  userActionType.value = action
  userActionDuration.value = 24
  userActionReason.value = ''
  userActionPermanent.value = false
  showUserActionModal.value = true
}

function getUserActionTitle() {
  const titles: Record<string, string> = {
    mute: '禁言用户',
    ban: '禁止登录',
    disablePosts: '禁用所有帖子',
    deletePosts: '删除所有帖子'
  }
  return titles[userActionType.value] || ''
}

function getUserActionDescription() {
  const descriptions: Record<string, string> = {
    mute: '禁言后，该用户将无法发帖和评论。',
    ban: '禁止登录后，该用户将无法登录账号。',
    disablePosts: '将该用户的所有帖子设为不可见状态。',
    deletePosts: '永久删除该用户的所有帖子，此操作不可恢复！'
  }
  return descriptions[userActionType.value] || ''
}

async function submitUserAction() {
  if (!userActionTarget.value) return

  const userId = userActionTarget.value.id
  const duration = userActionPermanent.value ? 0 : userActionDuration.value

  try {
    if (userActionType.value === 'mute') {
      await muteUser(userId, { duration, reason: userActionReason.value })
    } else if (userActionType.value === 'ban') {
      await banUser(userId, { duration, reason: userActionReason.value })
    } else if (userActionType.value === 'disablePosts') {
      await disableUserPosts(userId)
    } else if (userActionType.value === 'deletePosts') {
      await deleteUserPosts(userId)
    }
    showUserActionModal.value = false
    await loadModeratorUsers()
  } catch (error) {
    console.error('操作失败:', error)
    alert('操作失败: ' + (error as Error).message)
  }
}

async function handleUnmute(userId: number) {
  try {
    await unmuteUser(userId)
    await loadModeratorUsers()
  } catch (error) {
    console.error('解除禁言失败:', error)
    alert('解除禁言失败: ' + (error as Error).message)
  }
}

async function handleUnban(userId: number) {
  try {
    await unbanUser(userId)
    await loadModeratorUsers()
  } catch (error) {
    console.error('解除封禁失败:', error)
    alert('解除封禁失败: ' + (error as Error).message)
  }
}

async function loadModeratorUsers() {
  loading.value = true
  try {
    const res = await getModeratorUsers({
      page: page.value,
      page_size: pageSize.value,
      role: filterRole.value || undefined,
      keyword: filterKeyword.value || undefined
    })
    allUsers.value = res.users || []
    total.value = res.total
  } catch (error) {
    console.error('加载用户失败:', error)
  } finally {
    loading.value = false
  }
}

function formatBanTime(dateStr: string | null) {
  if (!dateStr) return '永久'
  return new Date(dateStr).toLocaleString('zh-CN')
}
</script>

<template>
  <div class="moderator-page" :class="{ 'animate-in': mounted }">
    <!-- 无权限提示 -->
    <div v-if="!hasAccess" class="no-access">
      <i class="ri-shield-keyhole-line"></i>
      <h2>无权访问</h2>
      <p>您没有版主权限，无法访问此页面</p>
    </div>

    <template v-else>
      <!-- 头部 -->
      <div class="header anim-item" style="--delay: 0">
        <h1 class="page-title">版主中心</h1>
        <div class="role-badge">
          <i class="ri-shield-star-line"></i>
          {{ userStore.user?.role === 'admin' ? '管理员' : '版主' }}
        </div>
      </div>

      <!-- 统计卡片 -->
      <div class="stats-grid anim-item" style="--delay: 1">
        <div class="stat-card pending">
          <div class="stat-icon"><i class="ri-time-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.pending_posts || 0 }}</div>
            <div class="stat-label">待审核帖子</div>
          </div>
        </div>
        <div class="stat-card pending">
          <div class="stat-icon"><i class="ri-gift-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.pending_items || 0 }}</div>
            <div class="stat-label">待审核作品</div>
          </div>
        </div>
        <div class="stat-card pending">
          <div class="stat-icon"><i class="ri-team-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.pending_guilds || 0 }}</div>
            <div class="stat-label">待审核公会</div>
          </div>
        </div>
        <div class="stat-card pending">
          <div class="stat-icon"><i class="ri-alarm-warning-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.pending_reports || 0 }}</div>
            <div class="stat-label">待处理举报</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon"><i class="ri-article-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.total_posts || 0 }}</div>
            <div class="stat-label">总帖子数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon"><i class="ri-box-3-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.total_items || 0 }}</div>
            <div class="stat-label">总作品数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon"><i class="ri-group-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.total_guilds || 0 }}</div>
            <div class="stat-label">总公会数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon"><i class="ri-user-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.total_users || 0 }}</div>
            <div class="stat-label">总用户数</div>
          </div>
        </div>
      </div>

      <!-- 主标签页 -->
      <div class="tab-container anim-item" style="--delay: 2">
        <div
          class="tab-item"
          :class="{ active: activeTab === 'review' }"
          @click="switchTab('review')"
        >
          <i class="ri-checkbox-circle-line"></i>
          <span>审核中心</span>
          <span v-if="(stats?.pending_posts || 0) + (stats?.pending_items || 0) + (stats?.pending_guilds || 0) + (stats?.pending_reports || 0) > 0" class="badge">
            {{ (stats?.pending_posts || 0) + (stats?.pending_items || 0) + (stats?.pending_guilds || 0) + (stats?.pending_reports || 0) }}
          </span>
        </div>
        <div
          class="tab-item"
          :class="{ active: activeTab === 'manage' }"
          @click="switchTab('manage')"
        >
          <i class="ri-settings-3-line"></i>
          <span>社区管理</span>
        </div>
        <div
          v-if="isAdmin"
          class="tab-item admin-tab"
          :class="{ active: activeTab === 'admin' }"
          @click="switchTab('admin')"
        >
          <i class="ri-admin-line"></i>
          <span>管理</span>
        </div>
        <div
          class="tab-item"
          :class="{ active: activeTab === 'logs' }"
          @click="switchTab('logs')"
        >
          <i class="ri-file-list-3-line"></i>
          <span>操作日志</span>
        </div>
        <div
          v-if="isAdmin"
          class="tab-item"
          :class="{ active: activeTab === 'metrics' }"
          @click="switchTab('metrics')"
        >
          <i class="ri-line-chart-line"></i>
          <span>数据统计</span>
        </div>
      </div>

      <!-- 子标签页 - 审核/管理中心 -->
      <div v-if="activeTab === 'review' || activeTab === 'manage'" class="sub-tab-container anim-item" style="--delay: 3">
        <button
          :class="{ active: activeSubTab === 'posts' }"
          @click="switchSubTab('posts')"
        >
          <i class="ri-article-line"></i>
          帖子
          <span v-if="activeTab === 'review' && (stats?.pending_posts || 0) > 0" class="review-badge">
            {{ stats?.pending_posts }}
          </span>
        </button>
        <button
          :class="{ active: activeSubTab === 'items' }"
          @click="switchSubTab('items')"
        >
          <i class="ri-gift-line"></i>
          作品
          <span v-if="activeTab === 'review' && (stats?.pending_items || 0) > 0" class="review-badge">
            {{ stats?.pending_items }}
          </span>
        </button>
        <button
          :class="{ active: activeSubTab === 'guilds' }"
          @click="switchSubTab('guilds')"
        >
          <i class="ri-team-line"></i>
          公会
          <span v-if="activeTab === 'review' && (stats?.pending_guilds || 0) > 0" class="review-badge">
            {{ stats?.pending_guilds }}
          </span>
        </button>
        <button
          v-if="activeTab === 'review'"
          :class="{ active: activeSubTab === 'reports' }"
          @click="switchSubTab('reports')"
        >
          <i class="ri-alarm-warning-line"></i>
          举报
          <span v-if="(stats?.pending_reports || 0) > 0" class="review-badge">
            {{ stats?.pending_reports }}
          </span>
        </button>
        <button
          v-if="activeTab === 'review'"
          :class="{ active: activeSubTab === 'postCommentImages' }"
          @click="switchSubTab('postCommentImages')"
        >
          <i class="ri-image-2-line"></i>
          帖子评论图
        </button>
        <button
          v-if="activeTab === 'review'"
          :class="{ active: activeSubTab === 'itemCommentImages' }"
          @click="switchSubTab('itemCommentImages')"
        >
          <i class="ri-image-line"></i>
          道具评论图
        </button>
        <button
          v-if="activeTab === 'review'"
          :class="{ active: activeSubTab === 'userAvatars' }"
          @click="switchSubTab('userAvatars')"
        >
          <i class="ri-user-3-line"></i>
          头像审核
        </button>
        <button
          v-if="activeTab === 'manage'"
          :class="{ active: activeSubTab === 'users' }"
          @click="switchSubTab('users')"
        >
          <i class="ri-user-settings-line"></i>
          用户
        </button>
      </div>

      <!-- 子标签页 - 管理标签 -->
      <div v-if="activeTab === 'admin'" class="sub-tab-container anim-item" style="--delay: 3">
        <button
          v-if="isAdmin"
          :class="{ active: adminSubTab === 'moderators' }"
          @click="switchAdminSubTab('moderators')"
        >
          <i class="ri-user-star-line"></i>
          版主管理
        </button>
        <button
          v-if="isAdmin"
          :class="{ active: adminSubTab === 'sponsors' }"
          @click="switchAdminSubTab('sponsors')"
        >
          <i class="ri-vip-crown-2-line"></i>
          赞助管理
        </button>
          <button
            v-if="isAdmin"
            :class="{ active: adminSubTab === 'experience' }"
            @click="switchAdminSubTab('experience')"
          >
            <i class="ri-medal-line"></i>
            等级管理
          </button>
          <button
            v-if="isAdmin"
            :class="{ active: adminSubTab === 'system' }"
            @click="switchAdminSubTab('system')"
          >
            <i class="ri-notification-3-line"></i>
            系统通知
          </button>
          <button
            :class="{ active: adminSubTab === 'guilds' }"
            @click="switchAdminSubTab('guilds')"
          >
            <i class="ri-team-line"></i>
          公会管理
        </button>
      </div>

      <!-- 审核中心 - 帖子列表 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'posts'" class="content-list anim-item" style="--delay: 4">
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingPosts.length === 0" class="empty-state">
          <i class="ri-checkbox-circle-line"></i>
          <p>暂无待审核帖子</p>
        </div>
        <div v-else class="item-list">
          <div v-for="post in pendingPosts" :key="post.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ post.title }}</span>
              <span class="status-badge pending">待审核</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</span>
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(post.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-approve" @click="quickReview('post', post.id, 'approve')">
                <i class="ri-checkbox-circle-line"></i> 通过
              </button>
              <button class="btn-reject" @click="quickReview('post', post.id, 'reject')">
                <i class="ri-close-circle-line"></i> 拒绝
              </button>
              <button class="btn-preview" @click="openPreview('post', post.id)">
                <i class="ri-eye-line"></i> 预览
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 审核中心 - 道具列表 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'items'" class="content-list anim-item" style="--delay: 4">
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingItems.length === 0" class="empty-state">
          <i class="ri-checkbox-circle-line"></i>
          <p>暂无待审核作品</p>
        </div>
        <div v-else class="item-list">
          <div v-for="item in pendingItems" :key="item.id" class="item-card" :class="{ 'has-permission': item.requires_permission }">
            <div class="item-header">
              <div class="title-with-tags">
                <span class="item-title">{{ item.name }}</span>
                <span v-if="item.requires_permission" class="permission-warning-tag">
                  <i class="ri-shield-keyhole-line"></i> 需要权限
                </span>
              </div>
              <span class="status-badge pending">待审核</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(item.author_name_color, item.author_name_bold)">{{ item.author_name }}</span>
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(item.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-approve" @click="quickReview('item', item.id, 'approve')">
                <i class="ri-checkbox-circle-line"></i> 通过
              </button>
              <button class="btn-reject" @click="quickReview('item', item.id, 'reject')">
                <i class="ri-close-circle-line"></i> 拒绝
              </button>
              <button class="btn-preview" @click="openPreview('item', item.id)">
                <i class="ri-eye-line"></i> 预览
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 审核中心 - 公会列表 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'guilds'" class="content-list anim-item" style="--delay: 4">
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingGuilds.length === 0" class="empty-state">
          <i class="ri-checkbox-circle-line"></i>
          <p>暂无待审核公会</p>
        </div>
        <div v-else class="item-list">
          <div v-for="guild in pendingGuilds" :key="guild.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ guild.name }}</span>
              <span class="status-badge pending">待审核</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(guild.owner_name_color, guild.owner_name_bold)">{{ guild.owner_name }}</span>
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(guild.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-approve" @click="quickReview('guild', guild.id, 'approve')">
                <i class="ri-checkbox-circle-line"></i> 通过
              </button>
              <button class="btn-reject" @click="quickReview('guild', guild.id, 'reject')">
                <i class="ri-close-circle-line"></i> 拒绝
              </button>
              <button class="btn-preview" @click="openPreview('guild', guild.id)">
                <i class="ri-eye-line"></i> 预览
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="activeTab === 'review' && activeSubTab === 'reports'" class="content-list anim-item" style="--delay: 4">
          <div class="report-scope-tabs">
            <button :class="{ active: reportScope === 'content' }" @click="switchReportScope('content')">
              <i class="ri-article-line"></i> 举报帖子/道具
            </button>
            <button :class="{ active: reportScope === 'comment' }" @click="switchReportScope('comment')">
              <i class="ri-message-3-line"></i> 举报评论
            </button>
            <button :class="{ active: reportScope === 'user' }" @click="switchReportScope('user')">
              <i class="ri-user-shared-line"></i> 举报用户
            </button>
          </div>
          <div class="filter-bar">
            <select v-model="reportStatus" @change="handleReportFilterChange">
              <option value="pending">待处理</option>
              <option value="resolved">已处置</option>
              <option value="rejected">已驳回</option>
              <option value="all">全部状态</option>
            </select>
            <select v-model="reportSort" @change="handleReportFilterChange">
              <option value="report_count">按举报数排序</option>
              <option value="latest_reported_at">按最新举报时间</option>
            </select>
            <select v-model="reportOrder" @change="handleReportFilterChange">
              <option value="desc">降序</option>
              <option value="asc">升序</option>
            </select>
          </div>
          <div v-if="pendingSelectableReports.length" class="report-batch-toolbar">
            <label class="report-select-toggle">
              <input
                type="checkbox"
                :checked="isAllPendingReportsSelected"
                @change="toggleSelectAllReports"
              />
              <span>全选当前页待处理</span>
            </label>
            <div class="report-batch-info">
              <span v-if="hasSelectedReports">已选择 {{ selectedPendingReports.length }} 条举报</span>
              <span v-else>勾选后可批量处置</span>
            </div>
            <div class="report-batch-actions">
              <button
                v-if="reportScopeSupportsDelete"
                class="btn-danger"
                :disabled="!hasSelectedReports"
                @click="openReportActionModal('delete_content')"
              >
                <i class="ri-delete-bin-line"></i> 删除内容
              </button>
              <button
                class="btn-warning"
                :disabled="!hasSelectedReports"
                @click="openReportActionModal(reportScope === 'user' ? 'mute_user' : 'delete_and_mute_user')"
              >
                <i class="ri-forbid-2-line"></i> {{ getReportMuteActionLabel() }}
              </button>
              <button
                class="btn-danger"
                :disabled="!hasSelectedReports"
                @click="openReportActionModal(reportScope === 'user' ? 'ban_user' : 'delete_and_ban_user')"
              >
                <i class="ri-shield-user-line"></i> {{ getReportBanActionLabel() }}
              </button>
              <button
                class="btn-reject"
                :disabled="!hasSelectedReports"
                @click="openReportActionModal('reject')"
              >
                <i class="ri-close-circle-line"></i> 驳回举报
              </button>
            </div>
          </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingReports.length === 0" class="empty-state">
          <i class="ri-shield-check-line"></i>
          <p>{{ getReportEmptyText() }}</p>
        </div>
        <div v-else class="item-list">
          <div v-for="report in pendingReports" :key="report.id" class="item-card report-card">
            <div class="item-header report-card-header">
              <label v-if="report.status === 'pending'" class="report-select-toggle card-toggle" @click.stop>
                <input
                  type="checkbox"
                  :checked="selectedReportIdSet.has(report.id)"
                  @change="toggleReportSelection(report.id)"
                />
                <span>选择</span>
              </label>
              <div class="report-title-wrap">
                <span class="item-title">{{ report.target_title || `#${report.target_id}` }}</span>
                <span class="report-type-tag">{{ getReportTargetLabel(report.target_type) }}</span>
              </div>
              <div class="report-tags">
                <span class="report-count-tag">
                  <i class="ri-flag-line"></i> {{ report.report_count }}
                </span>
                <span class="status-badge" :class="report.status">{{ getReportStatusLabel(report.status) }}</span>
              </div>
            </div>
            <div class="item-meta">
              <span v-if="report.target_author_name && report.target_type !== 'user'">
                <i class="ri-user-line"></i> {{ report.target_author_name }}
              </span>
              <span v-if="report.parent_target_title">
                <i class="ri-links-line"></i> 所属：{{ report.parent_target_title }}
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(report.latest_reported_at) }}</span>
            </div>
            <div v-if="getVisibleReportReasons(report).length" class="report-reason-section">
              <div class="report-section-title">
                <i class="ri-file-list-3-line"></i>
                <span>举报依据（{{ getVisibleReportReasons(report).length }}）</span>
              </div>
              <div class="report-reason-list">
              <div v-for="reason in getVisibleReportReasons(report)" :key="reason.id" class="report-reason-item">
                <div class="report-reason-head">
                  <div class="report-reason-main">
                    <span class="report-reason-meta">
                      举报人：{{ getReportReporterLabel(reason) }}
                    </span>
                    <div class="report-reason-summary">
                      {{ getReportReasonSummaryLabel(reason) }}：{{ getReportReasonSummaryValue(reason) }}
                    </div>
                  </div>
                  <span class="report-reason-time">
                    {{ reason.created_at ? formatDate(reason.created_at) : '-' }}
                  </span>
                </div>
                <div v-if="reason.detail?.trim()" class="report-evidence-row detail-row">
                  <div class="report-evidence-title">{{ getReportReasonDetailLabel(reason) }}</div>
                  <div class="report-evidence-content">{{ getReportReasonDetail(reason.detail) }}</div>
                </div>
              </div>
              </div>
            </div>
            <div v-if="hasReportPreview(report)" class="report-preview-card">
              <img
                v-if="report.target_preview_image"
                class="report-preview-image"
                :src="resolveImageUrl(report.target_preview_image)"
                :alt="report.target_title"
              />
              <div class="report-preview-main">
                <div class="report-preview-label">被举报内容预览</div>
                <p v-if="report.target_preview_text" class="report-preview-text">{{ report.target_preview_text }}</p>
                <p v-else class="report-preview-text muted">该目标当前没有可展示的正文摘要</p>
              </div>
            </div>
            <div v-if="report.review_comment" class="report-review-text">
              <i class="ri-chat-check-line"></i> 处理结果：{{ report.review_comment }}
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="openReportTarget(report)">
                <i class="ri-eye-line"></i> 查看目标
              </button>
              <button
                v-if="report.status === 'pending' && reportScopeSupportsDelete"
                class="btn-danger"
                @click="openReportActionModal('delete_content', report)"
              >
                <i class="ri-delete-bin-line"></i> 删除内容
              </button>
              <button
                v-if="report.status === 'pending'"
                class="btn-warning"
                @click="openReportActionModal(reportScope === 'user' ? 'mute_user' : 'delete_and_mute_user', report)"
              >
                <i class="ri-forbid-2-line"></i> {{ getReportMuteActionLabel() }}
              </button>
              <button
                v-if="report.status === 'pending'"
                class="btn-danger"
                @click="openReportActionModal(reportScope === 'user' ? 'ban_user' : 'delete_and_ban_user', report)"
              >
                <i class="ri-shield-user-line"></i> {{ getReportBanActionLabel() }}
              </button>
              <button v-if="report.status === 'pending'" class="btn-reject" @click="openReportActionModal('reject', report)">
                <i class="ri-close-circle-line"></i> 驳回举报
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 审核中心 - 帖子评论图片 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'postCommentImages'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <select v-model="imageReviewStatus" @change="handleImageReviewStatusChange">
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingPostCommentImages.length === 0" class="empty-state">
          <i class="ri-image-2-line"></i>
          <p>暂无帖子评论图片</p>
        </div>
        <div v-else class="item-list">
          <div v-for="comment in pendingPostCommentImages" :key="comment.id" class="item-card image-review-card">
            <div class="item-header">
              <span class="item-title">{{ comment.post_title || `帖子 #${comment.post_id}` }}</span>
              <span class="status-badge" :class="comment.image_review_status">{{ getReviewStatusLabel(comment.image_review_status) }}</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                {{ comment.author_name }}
              </span>
              <span><i class="ri-hashtag"></i> 评论 #{{ comment.id }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(comment.created_at) }}</span>
            </div>
            <div class="image-review-body">
              <img class="image-review-thumb" :src="resolveImageUrl(comment.image_url)" alt="帖子评论图片" @click="openImageViewer([comment.image_url])" />
              <div class="image-review-main">
                <p class="image-review-text">{{ comment.content || '（无评论文字）' }}</p>
                <textarea
                  v-if="comment.image_review_status === 'pending'"
                  v-model="imageReviewCommentDrafts[getImageReviewDraftKey('post', comment.id)]"
                  class="image-review-comment"
                  rows="2"
                  placeholder="审核备注（可选）"
                />
                <p v-else-if="comment.image_review_comment" class="image-review-history">
                  审核备注：{{ comment.image_review_comment }}
                </p>
              </div>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="openImageViewer([comment.image_url])">
                <i class="ri-zoom-in-line"></i> 查看大图
              </button>
              <template v-if="comment.image_review_status === 'pending'">
                <button
                  class="btn-approve"
                  @click="quickReviewImage('postComment', comment.id, 'approve', imageReviewCommentDrafts[getImageReviewDraftKey('post', comment.id)])"
                >
                  <i class="ri-checkbox-circle-line"></i> 通过
                </button>
                <button
                  class="btn-reject"
                  @click="quickReviewImage('postComment', comment.id, 'reject', imageReviewCommentDrafts[getImageReviewDraftKey('post', comment.id)])"
                >
                  <i class="ri-close-circle-line"></i> 拒绝
                </button>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- 审核中心 - 道具评论图片 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'itemCommentImages'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <select v-model="imageReviewStatus" @change="handleImageReviewStatusChange">
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingItemCommentImages.length === 0" class="empty-state">
          <i class="ri-image-line"></i>
          <p>暂无道具评论图片</p>
        </div>
        <div v-else class="item-list">
          <div v-for="comment in pendingItemCommentImages" :key="comment.id" class="item-card image-review-card">
            <div class="item-header">
              <span class="item-title">{{ comment.item_name || `道具 #${comment.item_id}` }}</span>
              <span class="status-badge" :class="comment.image_review_status">{{ getReviewStatusLabel(comment.image_review_status) }}</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                {{ comment.author_name }}
              </span>
              <span><i class="ri-hashtag"></i> 评论 #{{ comment.id }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(comment.created_at) }}</span>
            </div>
            <div class="image-review-body">
              <img class="image-review-thumb" :src="resolveImageUrl(comment.image_url)" alt="道具评论图片" @click="openImageViewer([comment.image_url])" />
              <div class="image-review-main">
                <p class="image-review-text">{{ comment.content || '（无评论文字）' }}</p>
                <textarea
                  v-if="comment.image_review_status === 'pending'"
                  v-model="imageReviewCommentDrafts[getImageReviewDraftKey('item', comment.id)]"
                  class="image-review-comment"
                  rows="2"
                  placeholder="审核备注（可选）"
                />
                <p v-else-if="comment.image_review_comment" class="image-review-history">
                  审核备注：{{ comment.image_review_comment }}
                </p>
              </div>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="openImageViewer([comment.image_url])">
                <i class="ri-zoom-in-line"></i> 查看大图
              </button>
              <template v-if="comment.image_review_status === 'pending'">
                <button
                  class="btn-approve"
                  @click="quickReviewImage('itemComment', comment.id, 'approve', imageReviewCommentDrafts[getImageReviewDraftKey('item', comment.id)])"
                >
                  <i class="ri-checkbox-circle-line"></i> 通过
                </button>
                <button
                  class="btn-reject"
                  @click="quickReviewImage('itemComment', comment.id, 'reject', imageReviewCommentDrafts[getImageReviewDraftKey('item', comment.id)])"
                >
                  <i class="ri-close-circle-line"></i> 拒绝
                </button>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- 审核中心 - 用户头像 -->
      <div v-if="activeTab === 'review' && activeSubTab === 'userAvatars'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <select v-model="imageReviewStatus" @change="handleImageReviewStatusChange">
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="pendingUserAvatars.length === 0" class="empty-state">
          <i class="ri-user-3-line"></i>
          <p>暂无头像审核记录</p>
        </div>
        <div v-else class="item-list">
          <div v-for="user in pendingUserAvatars" :key="user.id" class="item-card image-review-card">
            <div class="item-header">
              <span class="item-title">{{ user.username }}</span>
              <span class="status-badge" :class="user.avatar_review_status">{{ getReviewStatusLabel(user.avatar_review_status) }}</span>
            </div>
            <div class="item-meta">
              <span><i class="ri-shield-user-line"></i> {{ getRoleLabel(user.role) }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(user.updated_at) }}</span>
            </div>
            <div class="image-review-body">
              <img class="image-review-thumb avatar" :src="resolveImageUrl(user.avatar_url)" alt="用户头像" @click="openImageViewer([user.avatar_url])" />
              <div class="image-review-main">
                <textarea
                  v-if="user.avatar_review_status === 'pending'"
                  v-model="imageReviewCommentDrafts[getImageReviewDraftKey('avatar', user.id)]"
                  class="image-review-comment"
                  rows="2"
                  placeholder="审核备注（可选）"
                />
                <p v-else-if="user.avatar_review_comment" class="image-review-history">
                  审核备注：{{ user.avatar_review_comment }}
                </p>
              </div>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="openImageViewer([user.avatar_url])">
                <i class="ri-zoom-in-line"></i> 查看大图
              </button>
              <template v-if="user.avatar_review_status === 'pending'">
                <button
                  class="btn-approve"
                  @click="quickReviewImage('avatar', user.id, 'approve', imageReviewCommentDrafts[getImageReviewDraftKey('avatar', user.id)])"
                >
                  <i class="ri-checkbox-circle-line"></i> 通过
                </button>
                <button
                  class="btn-reject"
                  @click="quickReviewImage('avatar', user.id, 'reject', imageReviewCommentDrafts[getImageReviewDraftKey('avatar', user.id)])"
                >
                  <i class="ri-close-circle-line"></i> 拒绝
                </button>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理中心 - 帖子列表 -->
      <div v-if="activeTab === 'manage' && activeSubTab === 'posts'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索标题..." @keyup.enter="loadAllPosts" />
          <select v-model="filterStatus" @change="loadAllPosts">
            <option value="">全部状态</option>
            <option value="draft">草稿</option>
            <option value="pending">待发布</option>
            <option value="published">已发布</option>
          </select>
          <select v-model="filterReviewStatus" @change="loadAllPosts">
            <option value="">全部审核状态</option>
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
          <select v-model="filterPostFlag" @change="loadAllPosts">
            <option value="">全部帖子</option>
            <option value="pinned">置顶</option>
            <option value="featured">精华</option>
            <option value="normal">普通</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else class="item-list">
          <div v-for="post in allPosts" :key="post.id" class="item-card">
            <div class="item-header">
              <div class="title-with-tags">
                <span class="item-title">{{ post.title }}</span>
                <span v-if="post.is_pinned" class="mod-tag pinned"><i class="ri-pushpin-fill"></i> 置顶</span>
                <span v-if="post.is_featured" class="mod-tag featured"><i class="ri-star-fill"></i> 精华</span>
              </div>
              <div class="status-badges">
                <span class="status-badge" :class="post.status">{{ getStatusLabel(post.status) }}</span>
                <span class="status-badge" :class="post.review_status">{{ getReviewStatusLabel(post.review_status) }}</span>
              </div>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(post.author_name_color, post.author_name_bold)">{{ post.author_name }}</span>
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(post.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="router.push({ name: 'post-detail', params: { id: post.id } })">
                <i class="ri-eye-line"></i> 查看
              </button>
              <button class="btn-edit" @click="router.push({ name: 'post-edit', params: { id: post.id } })">
                <i class="ri-edit-line"></i> 编辑
              </button>
              <button class="btn-pin" :class="{ active: post.is_pinned }" @click="handlePinPost(post.id, post.is_pinned)">
                <i class="ri-pushpin-line"></i> {{ post.is_pinned ? '取消置顶' : '置顶' }}
              </button>
              <button class="btn-feature" :class="{ active: post.is_featured }" @click="handleFeaturePost(post.id, post.is_featured)">
                <i class="ri-star-line"></i> {{ post.is_featured ? '取消精华' : '精华' }}
              </button>
              <button class="btn-warning" @click="handleHidePost(post.id)">
                <i class="ri-eye-off-line"></i> 屏蔽
              </button>
              <button class="btn-delete" @click="handleDeletePost(post.id)">
                <i class="ri-delete-bin-line"></i> 删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理中心 - 道具列表 -->
      <div v-if="activeTab === 'manage' && activeSubTab === 'items'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索名称..." @keyup.enter="loadAllItems" />
          <select v-model="filterStatus" @change="loadAllItems">
            <option value="">全部状态</option>
            <option value="draft">草稿</option>
            <option value="pending">待发布</option>
            <option value="published">已发布</option>
          </select>
          <select v-model="filterReviewStatus" @change="loadAllItems">
            <option value="">全部审核状态</option>
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else class="item-list">
          <div v-for="item in allItems" :key="item.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ item.name }}</span>
              <div class="status-badges">
                <span class="status-badge" :class="item.status">{{ getStatusLabel(item.status) }}</span>
                <span class="status-badge" :class="item.review_status">{{ getReviewStatusLabel(item.review_status) }}</span>
              </div>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(item.author_name_color, item.author_name_bold)">{{ item.author_name }}</span>
              </span>
              <span><i class="ri-time-line"></i> {{ formatDate(item.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-warning" @click="handleHideItem(item.id)">
                <i class="ri-eye-off-line"></i> 屏蔽
              </button>
              <button class="btn-delete" @click="handleDeleteItem(item.id)">
                <i class="ri-delete-bin-line"></i> 删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理中心 - 公会列表 -->
      <div v-if="activeTab === 'manage' && activeSubTab === 'guilds'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索公会名..." @keyup.enter="loadAllGuilds" />
          <select v-model="filterGuildStatus" @change="loadAllGuilds">
            <option value="">全部状态</option>
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else class="item-list">
          <div v-for="guild in allGuilds" :key="guild.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ guild.name }}</span>
              <span class="status-badge" :class="guild.status">{{ getGuildStatusLabel(guild.status) }}</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(guild.owner_name_color, guild.owner_name_bold)">{{ guild.owner_name }}</span>
              </span>
              <span><i class="ri-group-line"></i> {{ guild.member_count }} 成员</span>
              <span><i class="ri-time-line"></i> {{ formatDate(guild.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-edit" @click="openChangeOwnerModal(guild.id, guild.name, guild.owner_id)">
                <i class="ri-user-settings-line"></i> 换会长
              </button>
              <button class="btn-delete" @click="handleDeleteGuild(guild.id)">
                <i class="ri-delete-bin-line"></i> 删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理中心 - 用户列表 -->
      <div v-if="activeTab === 'manage' && activeSubTab === 'users'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索用户名或邮箱..." @keyup.enter="loadModeratorUsers" />
          <select v-model="filterRole" @change="loadModeratorUsers">
            <option value="">全部角色</option>
            <option value="user">普通用户</option>
            <option value="moderator">版主</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="allUsers.length === 0" class="empty-state">
          <i class="ri-user-search-line"></i>
          <p>暂无用户数据</p>
        </div>
        <div v-else class="item-list">
          <div v-for="user in allUsers" :key="user.id" class="item-card user-card">
            <div class="item-header">
              <div class="user-info">
                <img v-if="user.avatar" :src="user.avatar" class="user-avatar" />
                <i v-else class="ri-user-line user-avatar-placeholder"></i>
                <span class="item-title" :style="buildNameStyle(user.name_color, user.name_bold)">{{ user.username }}</span>
              </div>
              <div class="user-status-tags">
                <span v-if="resolveSponsorLevel(user) > 0" class="sponsor-tag">赞助 Lv{{ resolveSponsorLevel(user) }}</span>
                <span class="role-tag" :class="user.role">{{ getRoleLabel(user.role) }}</span>
                <span v-if="user.is_muted" class="status-tag muted">禁言中</span>
                <span v-if="user.is_banned" class="status-tag banned">已封禁</span>
              </div>
            </div>
            <div class="item-meta">
              <span><i class="ri-mail-line"></i> {{ user.email }}</span>
              <span><i class="ri-article-line"></i> {{ user.post_count }} 帖子</span>
              <span><i class="ri-time-line"></i> {{ formatDate(user.created_at) }}</span>
            </div>
            <div v-if="user.is_muted || user.is_banned" class="ban-info">
              <span v-if="user.is_muted">
                <i class="ri-volume-mute-line"></i> 禁言至: {{ formatBanTime(user.muted_until) }}
                <template v-if="user.mute_reason"> - {{ user.mute_reason }}</template>
              </span>
              <span v-if="user.is_banned">
                <i class="ri-forbid-line"></i> 封禁至: {{ formatBanTime(user.banned_until) }}
                <template v-if="user.ban_reason"> - {{ user.ban_reason }}</template>
              </span>
            </div>
            <div class="item-actions" v-if="user.role !== 'admin' && user.role !== 'moderator'">
              <button v-if="!user.is_muted" class="btn-warning" @click="openUserActionModal(user, 'mute')">
                <i class="ri-volume-mute-line"></i> 禁言
              </button>
              <button v-else class="btn-approve" @click="handleUnmute(user.id)">
                <i class="ri-volume-up-line"></i> 解除禁言
              </button>
              <button v-if="!user.is_banned" class="btn-danger" @click="openUserActionModal(user, 'ban')">
                <i class="ri-forbid-line"></i> 封禁
              </button>
              <button v-else class="btn-approve" @click="handleUnban(user.id)">
                <i class="ri-checkbox-circle-line"></i> 解除封禁
              </button>
              <button class="btn-warning" @click="openUserActionModal(user, 'disablePosts')">
                <i class="ri-eye-off-line"></i> 禁用帖子
              </button>
              <button class="btn-delete" @click="openUserActionModal(user, 'deletePosts')">
                <i class="ri-delete-bin-line"></i> 删除帖子
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理标签 - 版主管理 -->
      <div v-if="activeTab === 'admin' && adminSubTab === 'moderators'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索用户名或邮箱..." @keyup.enter="loadUsers" />
          <select v-model="filterRole" @change="loadUsers">
            <option value="">全部角色</option>
            <option value="user">普通用户</option>
            <option value="moderator">版主</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="allUsers.length === 0" class="empty-state">
          <i class="ri-user-search-line"></i>
          <p>暂无用户数据</p>
        </div>
        <div v-else class="item-list">
          <div v-for="user in allUsers" :key="user.id" class="item-card user-card">
            <div class="item-header">
              <div class="user-info">
                <img v-if="user.avatar" :src="user.avatar" class="user-avatar" />
                <i v-else class="ri-user-line user-avatar-placeholder"></i>
                <span class="item-title" :style="buildNameStyle(user.name_color, user.name_bold)">{{ user.username }}</span>
              </div>
              <span v-if="resolveSponsorLevel(user) > 0" class="sponsor-tag">赞助 Lv{{ resolveSponsorLevel(user) }}</span>
              <span class="role-tag" :class="user.role">{{ getRoleLabel(user.role) }}</span>
            </div>
            <div class="item-meta">
              <span><i class="ri-mail-line"></i> {{ user.email }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(user.created_at) }}</span>
            </div>
            <div class="item-actions" v-if="user.role !== 'admin'">
              <button
                v-if="user.role === 'user'"
                class="btn-approve"
                @click="openRoleModal(user.id, user.username, user.role, 'moderator', user.name_color, user.name_bold)"
              >
                <i class="ri-shield-star-line"></i> 设为版主
              </button>
              <button
                v-else-if="user.role === 'moderator'"
                class="btn-warning"
                @click="openRoleModal(user.id, user.username, user.role, 'user', user.name_color, user.name_bold)"
              >
                <i class="ri-shield-line"></i> 取消版主
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理标签 - 赞助管理 -->
      <div v-if="activeTab === 'admin' && adminSubTab === 'sponsors'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索用户名或邮箱..." @keyup.enter="loadUsers" />
          <select v-model="filterSponsorLevel" @change="loadUsers">
            <option value="">全部赞助等级</option>
            <option value="1">Lv1 仅鸣谢</option>
            <option value="2">Lv2 昵称样式</option>
            <option value="3">Lv3 个性化</option>
            <option value="0">非赞助者</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="allUsers.length === 0" class="empty-state">
          <i class="ri-user-search-line"></i>
          <p>暂无用户数据</p>
        </div>
        <div v-else class="item-list">
          <div v-for="user in allUsers" :key="user.id" class="item-card user-card">
            <div class="item-header">
              <div class="user-info">
                <img v-if="user.avatar" :src="user.avatar" class="user-avatar" />
                <i v-else class="ri-user-line user-avatar-placeholder"></i>
                <span class="item-title" :style="buildNameStyle(user.name_color, user.name_bold)">{{ user.username }}</span>
              </div>
              <div class="user-status-tags">
                <span v-if="resolveSponsorLevel(user) > 0" class="sponsor-tag">赞助 Lv{{ resolveSponsorLevel(user) }}</span>
                <span class="role-tag" :class="user.role">{{ getRoleLabel(user.role) }}</span>
              </div>
            </div>
            <div class="item-meta">
              <span><i class="ri-mail-line"></i> {{ user.email }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(user.created_at) }}</span>
            </div>
            <div class="item-actions sponsor-level-actions">
              <select v-model.number="sponsorLevelDrafts[user.id]" class="sponsor-level-select">
                <option v-for="option in sponsorLevelOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
              <button class="btn-sponsor" @click="applySponsorLevel(user)">
                <i class="ri-vip-crown-2-line"></i> 应用
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="activeTab === 'admin' && adminSubTab === 'experience'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索用户名或邮箱..." @keyup.enter="loadUsers" />
          <select v-model="filterRole" @change="loadUsers">
            <option value="">全部角色</option>
            <option value="user">普通用户</option>
            <option value="moderator">版主</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="allUsers.length === 0" class="empty-state">
          <i class="ri-medal-line"></i>
          <p>暂无用户数据</p>
        </div>
        <div v-else class="item-list">
          <div v-for="user in allUsers" :key="user.id" class="item-card user-card">
            <div class="item-header">
              <div class="user-info">
                <img v-if="user.avatar" :src="user.avatar" class="user-avatar" />
                <i v-else class="ri-user-line user-avatar-placeholder"></i>
                <span class="item-title" :style="buildNameStyle(user.name_color, user.name_bold)">{{ user.username }}</span>
              </div>
              <div class="user-status-tags">
                <span class="level-tag">{{ formatForumLevel(user) }}</span>
                <span class="role-tag" :class="user.role">{{ getRoleLabel(user.role) }}</span>
              </div>
            </div>
            <div class="item-meta">
              <span><i class="ri-mail-line"></i> {{ user.email }}</span>
              <span><i class="ri-sparkling-line"></i> 总经验 {{ user.activity_experience || 0 }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(user.created_at) }}</span>
            </div>
            <div class="item-actions experience-actions">
              <label class="experience-label">
                <span>总经验</span>
                <input
                  v-model.number="experienceDrafts[user.id]"
                  class="experience-input"
                  type="number"
                  min="0"
                  step="1"
                />
              </label>
              <button class="btn-sponsor" @click="applyUserExperience(user)">
                <i class="ri-medal-line"></i> 应用
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 管理标签 - 系统通知 -->
      <div v-if="activeTab === 'admin' && adminSubTab === 'system'" class="content-list anim-item" style="--delay: 4">
        <div class="item-card system-message-card">
          <div class="item-header">
            <span class="item-title">系统通知</span>
            <span class="system-message-hint">发送给所有用户</span>
          </div>
          <textarea
            v-model="systemMessage"
            class="system-message-textarea"
            :maxlength="systemMessageMaxLength"
            rows="6"
            placeholder="请输入系统通知内容..."
          ></textarea>
          <div class="system-message-actions">
            <span class="char-count">{{ systemMessageLength }}/{{ systemMessageMaxLength }}</span>
            <button class="btn-submit" :disabled="systemMessageSending || !systemMessageTrimmed" @click="sendSystemMessage">
              <i class="ri-send-plane-2-line"></i>
              {{ systemMessageSending ? '发送中...' : '发送通知' }}
            </button>
          </div>
          <div class="warning-box">
            <i class="ri-alert-line"></i>
            <span>系统通知会进入所有用户的消息中心，请确认内容无误。</span>
          </div>
        </div>
      </div>

      <!-- 管理标签 - 公会管理 -->
      <div v-if="activeTab === 'admin' && adminSubTab === 'guilds'" class="content-list anim-item" style="--delay: 4">
        <div class="filter-bar">
          <input v-model="filterKeyword" placeholder="搜索公会名..." @keyup.enter="loadAllGuilds" />
          <select v-model="filterGuildStatus" @change="loadAllGuilds">
            <option value="">全部状态</option>
            <option value="pending">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="allGuilds.length === 0" class="empty-state">
          <i class="ri-team-line"></i>
          <p>暂无公会数据</p>
        </div>
        <div v-else class="item-list">
          <div v-for="guild in allGuilds" :key="guild.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ guild.name }}</span>
              <span class="status-badge" :class="guild.status">{{ getGuildStatusLabel(guild.status) }}</span>
            </div>
            <div class="item-meta">
              <span>
                <i class="ri-user-line"></i>
                <span :style="buildNameStyle(guild.owner_name_color, guild.owner_name_bold)">{{ guild.owner_name }}</span>
              </span>
              <span><i class="ri-group-line"></i> {{ guild.member_count }} 成员</span>
              <span><i class="ri-time-line"></i> {{ formatDate(guild.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-edit" @click="openChangeOwnerModal(guild.id, guild.name, guild.owner_id)">
                <i class="ri-user-settings-line"></i> 换会长
              </button>
              <button class="btn-delete" @click="handleDeleteGuild(guild.id)">
                <i class="ri-delete-bin-line"></i> 删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作日志 -->
      <div v-if="activeTab === 'logs'" class="content-list anim-item" style="--delay: 3">
        <div class="filter-bar">
          <select v-model="logsActionType" @change="loadActionLogs">
            <option value="">全部操作</option>
            <option value="review_post">审核帖子</option>
            <option value="review_item">审核作品</option>
            <option value="review_guild">审核公会</option>
            <option value="delete_post">删除帖子</option>
            <option value="delete_item">删除作品</option>
            <option value="delete_guild">删除公会</option>
            <option value="hide_post">屏蔽帖子</option>
            <option value="hide_item">屏蔽作品</option>
            <option value="pin_post">置顶帖子</option>
            <option value="feature_post">设为精华</option>
            <option value="change_guild_owner">更换会长</option>
            <option value="mute_user">禁言用户</option>
            <option value="unmute_user">解除禁言</option>
            <option value="ban_user">封禁用户</option>
            <option value="unban_user">解除封禁</option>
            <option value="set_role">设置角色</option>
            <option value="set_sponsor">设置赞助</option>
            <option value="set_experience">设置经验</option>
          </select>
          <select v-model="logsTargetType" @change="loadActionLogs">
            <option value="">全部类型</option>
            <option value="post">帖子</option>
            <option value="item">作品</option>
            <option value="guild">公会</option>
            <option value="user">用户</option>
          </select>
        </div>
        <div v-if="loading" class="loading">
          <i class="ri-loader-4-line loading-spinner"></i>
          <span>加载中...</span>
        </div>
        <div v-else-if="actionLogs.length === 0" class="empty-state">
          <i class="ri-file-list-3-line"></i>
          <p>暂无操作日志</p>
        </div>
        <div v-else class="logs-table-wrapper">
          <table class="logs-table">
            <thead>
              <tr>
                <th>操作者</th>
                <th>操作类型</th>
                <th>目标</th>
                <th>详情</th>
                <th>IP地址</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in actionLogs" :key="log.id">
                <td>
                  <span class="operator" :style="buildNameStyle(log.operator_name_color, log.operator_name_bold)">{{ log.operator_name }}</span>
                  <span class="role-tag" :class="log.operator_role">{{ log.operator_role === 'admin' ? '管理员' : '版主' }}</span>
                </td>
                <td><span class="action-type">{{ formatActionType(log.action_type) }}</span></td>
                <td>
                  <span class="target-type">{{ log.target_type }}</span>
                  <span class="target-name" :style="buildNameStyle(log.target_name_color, log.target_name_bold)">{{ log.target_name || `#${log.target_id}` }}</span>
                </td>
                <td class="details-cell">{{ formatLogDetails(log) }}</td>
                <td class="ip-cell">{{ log.ip_address }}</td>
                <td class="time-cell">{{ formatDate(log.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="logsTotal > logsPageSize" class="pagination">
          <button :disabled="logsPage <= 1" @click="logsPage--; loadActionLogs()">
            <i class="ri-arrow-left-line"></i>
          </button>
          <span>{{ logsPage }} / {{ Math.ceil(logsTotal / logsPageSize) }}</span>
          <button :disabled="logsPage >= Math.ceil(logsTotal / logsPageSize)" @click="logsPage++; loadActionLogs()">
            <i class="ri-arrow-right-line"></i>
          </button>
        </div>
      </div>

      <!-- 数据统计 -->
      <div v-if="activeTab === 'metrics' && isAdmin" class="content-list anim-item" style="--delay: 3">
        <div class="sub-tab-container metrics-subtabs">
          <button
            :class="{ active: metricsSubTab === 'growth' }"
            @click="switchMetricsSubTab('growth')"
          >
            <i class="ri-line-chart-line"></i>
            社区统计
          </button>
          <button
            :class="{ active: metricsSubTab === 'basic' }"
            @click="switchMetricsSubTab('basic')"
          >
            <i class="ri-dashboard-2-line"></i>
            基础监控
          </button>
        </div>

        <template v-if="metricsSubTab === 'growth'">
          <div class="filter-bar">
            <select v-model="metricsDays" @change="loadMetrics">
              <option :value="7">最近 7 天</option>
              <option :value="14">最近 14 天</option>
              <option :value="30">最近 30 天</option>
              <option :value="60">最近 60 天</option>
              <option :value="90">最近 90 天</option>
            </select>
          </div>

          <div v-if="metricsLoading" class="loading">
            <i class="ri-loader-4-line loading-spinner"></i>
            <span>加载中...</span>
          </div>

          <template v-else>
            <!-- 摘要卡片 -->
            <div v-if="metricsSummary" class="metrics-summary">
              <div class="summary-card">
                <div class="summary-header">
                  <span class="period">今日</span>
                </div>
                <div class="summary-body">
                  <div class="summary-item">
                    <span class="label">新用户</span>
                    <span class="value">{{ metricsSummary.today.users }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新帖子</span>
                    <span class="value">{{ metricsSummary.today.posts }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新作品</span>
                    <span class="value">{{ metricsSummary.today.items }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新公会</span>
                    <span class="value">{{ metricsSummary.today.guilds }}</span>
                  </div>
                </div>
              </div>
              <div class="summary-card">
                <div class="summary-header">
                  <span class="period">昨日</span>
                </div>
                <div class="summary-body">
                  <div class="summary-item">
                    <span class="label">新用户</span>
                    <span class="value">{{ metricsSummary.yesterday.users }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新帖子</span>
                    <span class="value">{{ metricsSummary.yesterday.posts }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新作品</span>
                    <span class="value">{{ metricsSummary.yesterday.items }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新公会</span>
                    <span class="value">{{ metricsSummary.yesterday.guilds }}</span>
                  </div>
                </div>
              </div>
              <div class="summary-card">
                <div class="summary-header">
                  <span class="period">本周</span>
                </div>
                <div class="summary-body">
                  <div class="summary-item">
                    <span class="label">新用户</span>
                    <span class="value">{{ metricsSummary.week.users }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新帖子</span>
                    <span class="value">{{ metricsSummary.week.posts }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新作品</span>
                    <span class="value">{{ metricsSummary.week.items }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新公会</span>
                    <span class="value">{{ metricsSummary.week.guilds }}</span>
                  </div>
                </div>
              </div>
              <div class="summary-card">
                <div class="summary-header">
                  <span class="period">本月</span>
                </div>
                <div class="summary-body">
                  <div class="summary-item">
                    <span class="label">新用户</span>
                    <span class="value">{{ metricsSummary.month.users }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新帖子</span>
                    <span class="value">{{ metricsSummary.month.posts }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新作品</span>
                    <span class="value">{{ metricsSummary.month.items }}</span>
                  </div>
                  <div class="summary-item">
                    <span class="label">新公会</span>
                    <span class="value">{{ metricsSummary.month.guilds }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 趋势图表 -->
            <div class="metrics-chart-container">
              <h4><i class="ri-line-chart-line"></i> 社区增长趋势</h4>
              <div ref="metricsChartRef" class="metrics-chart"></div>
            </div>
          </template>
        </template>

        <template v-else>
          <div class="filter-bar">
            <select v-model="metricsDays" @change="loadBasicMetrics">
              <option :value="7">最近 7 天</option>
              <option :value="14">最近 14 天</option>
              <option :value="30">最近 30 天</option>
              <option :value="60">最近 60 天</option>
              <option :value="90">最近 90 天</option>
            </select>
          </div>

          <div v-if="basicMetricsLoading" class="loading">
            <i class="ri-loader-4-line loading-spinner"></i>
            <span>加载中...</span>
          </div>
          <template v-else>
            <div v-if="basicMetrics" class="basic-metrics-grid">
              <div class="basic-metric-card">
                <div class="basic-metric-icon"><i class="ri-book-open-line"></i></div>
                <div class="basic-metric-info">
                  <div class="basic-metric-value">{{ basicMetrics.story_archives || 0 }}</div>
                  <div class="basic-metric-label">系统剧情归档数</div>
                </div>
              </div>
              <div class="basic-metric-card">
                <div class="basic-metric-icon"><i class="ri-chat-3-line"></i></div>
                <div class="basic-metric-info">
                  <div class="basic-metric-value">{{ basicMetrics.story_entries || 0 }}</div>
                  <div class="basic-metric-label">归档条目数</div>
                </div>
              </div>
              <div class="basic-metric-card">
                <div class="basic-metric-icon"><i class="ri-save-3-line"></i></div>
                <div class="basic-metric-info">
                  <div class="basic-metric-value">{{ basicMetrics.profile_backups || 0 }}</div>
                  <div class="basic-metric-label">人物卡备份数</div>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <i class="ri-bar-chart-2-line"></i>
              <p>暂无监控数据</p>
            </div>

            <div v-if="basicMetrics" class="metrics-chart-container">
              <h4><i class="ri-line-chart-line"></i> 基础监控趋势</h4>
              <div ref="metricsChartRef" class="metrics-chart"></div>
            </div>
          </template>
        </template>
      </div>

      <!-- 审核弹窗 -->
      <div v-if="showReviewModal" class="modal-overlay" @click.self="showReviewModal = false">
        <div class="modal">
          <div class="modal-header">
            <h3>审核{{ reviewTarget?.type === 'post' ? '帖子' : reviewTarget?.type === 'item' ? '作品' : '公会' }}</h3>
            <button class="close-btn" @click="showReviewModal = false">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body">
            <p class="review-title">{{ reviewTarget?.title }}</p>
            <div class="review-actions">
              <label class="radio-label">
                <input type="radio" v-model="reviewAction" value="approve" />
                <span class="radio-text approve">通过</span>
              </label>
              <label class="radio-label">
                <input type="radio" v-model="reviewAction" value="reject" />
                <span class="radio-text reject">拒绝</span>
              </label>
            </div>
            <textarea
              v-model="reviewComment"
              placeholder="审核意见（可选）"
              rows="3"
            ></textarea>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showReviewModal = false">取消</button>
            <button class="btn-submit" @click="submitReview">确认</button>
          </div>
        </div>
      </div>

      <!-- 更换会长弹窗 -->
      <div v-if="showChangeOwnerModal" class="modal-overlay" @click.self="showChangeOwnerModal = false">
        <div class="modal">
          <div class="modal-header">
            <h3>更换会长</h3>
            <button class="close-btn" @click="showChangeOwnerModal = false">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body">
            <p class="review-title">公会: {{ changeOwnerTarget?.name }}</p>
            <div class="form-group">
              <label>新会长（用户名或邮箱）</label>
              <input
                v-model="newOwnerIdentifier"
                type="text"
                placeholder="请输入新会长的用户名或邮箱"
                class="form-input"
              />
            </div>
            <p class="hint-text">
              <i class="ri-information-line"></i>
              支持通过用户名或注册邮箱查找用户
            </p>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showChangeOwnerModal = false">取消</button>
            <button class="btn-submit" @click="submitChangeOwner" :disabled="!newOwnerIdentifier">确认</button>
          </div>
        </div>
      </div>

      <!-- 设置角色弹窗 -->
      <div v-if="showRoleModal" class="modal-overlay" @click.self="showRoleModal = false">
        <div class="modal">
          <div class="modal-header">
            <h3>{{ roleTarget?.newRole === 'moderator' ? '设为版主' : '取消版主' }}</h3>
            <button class="close-btn" @click="showRoleModal = false">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body">
            <div class="role-change-info">
              <div class="user-preview">
                <i class="ri-user-line"></i>
                <span class="username" :style="buildNameStyle(roleTarget?.name_color, roleTarget?.name_bold)">{{ roleTarget?.username }}</span>
              </div>
              <div class="role-arrow">
                <span class="role-tag" :class="roleTarget?.currentRole">{{ getRoleLabel(roleTarget?.currentRole || '') }}</span>
                <i class="ri-arrow-right-line"></i>
                <span class="role-tag" :class="roleTarget?.newRole">{{ getRoleLabel(roleTarget?.newRole || '') }}</span>
              </div>
            </div>
            <p class="confirm-text">
              {{ roleTarget?.newRole === 'moderator'
                ? '确定要将该用户设为版主吗？版主可以审核帖子、作品和公会。'
                : '确定要取消该用户的版主权限吗？' }}
            </p>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showRoleModal = false">取消</button>
            <button class="btn-submit" @click="submitRoleChange">确认</button>
          </div>
        </div>
      </div>

      <!-- 用户操作弹窗 -->
      <div v-if="showUserActionModal" class="modal-overlay" @click.self="showUserActionModal = false">
        <div class="modal">
          <div class="modal-header">
            <h3>{{ getUserActionTitle() }}</h3>
            <button class="close-btn" @click="showUserActionModal = false">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body">
            <div class="user-action-info">
              <div class="user-preview">
                <img v-if="userActionTarget?.avatar" :src="userActionTarget.avatar" class="user-avatar" />
                <i v-else class="ri-user-line user-avatar-placeholder"></i>
                <span class="username" :style="buildNameStyle(userActionTarget?.name_color, userActionTarget?.name_bold)">{{ userActionTarget?.username }}</span>
              </div>
              <p class="action-description">{{ getUserActionDescription() }}</p>
            </div>

            <!-- 禁言/封禁需要设置时长和原因 -->
            <template v-if="userActionType === 'mute' || userActionType === 'ban'">
              <div class="form-group">
                <label>时长</label>
                <div class="duration-options">
                  <label class="checkbox-label">
                    <input type="checkbox" v-model="userActionPermanent" />
                    <span>永久</span>
                  </label>
                  <input
                    v-if="!userActionPermanent"
                    v-model.number="userActionDuration"
                    type="number"
                    min="1"
                    placeholder="小时数"
                    class="form-input duration-input"
                  />
                  <span v-if="!userActionPermanent" class="duration-unit">小时</span>
                </div>
              </div>
              <div class="form-group">
                <label>原因</label>
                <textarea
                  v-model="userActionReason"
                  placeholder="请输入原因（可选）"
                  rows="3"
                ></textarea>
              </div>
            </template>

            <!-- 禁用/删除帖子的警告 -->
            <template v-else>
              <div class="warning-box" :class="{ danger: userActionType === 'deletePosts' }">
                <i :class="userActionType === 'deletePosts' ? 'ri-error-warning-line' : 'ri-information-line'"></i>
                <span v-if="userActionType === 'disablePosts'">
                  此操作将把该用户的所有帖子设为不可见状态。
                </span>
                <span v-else>
                  此操作将永久删除该用户的所有帖子，包括相关的评论、点赞和收藏数据。此操作不可恢复！
                </span>
              </div>
            </template>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showUserActionModal = false">取消</button>
            <button
              class="btn-submit"
              :class="{ danger: userActionType === 'deletePosts' || userActionType === 'ban' }"
              @click="submitUserAction"
            >
              确认{{ getUserActionTitle() }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="showReportActionModal" class="modal-overlay" @click.self="closeReportActionModal()">
        <div class="modal">
          <div class="modal-header">
            <h3>{{ getReportActionTitle() }}</h3>
            <button class="close-btn" @click="closeReportActionModal()">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body">
            <div class="report-action-summary">
              <p class="action-description">{{ getReportActionDescription() }}</p>
              <div class="report-action-chip-list">
                <span v-for="report in selectedPendingReports.slice(0, 6)" :key="report.id" class="report-action-chip">
                  {{ report.target_title || `#${report.target_id}` }}
                </span>
                <span v-if="selectedPendingReports.length > 6" class="report-action-chip more-chip">
                  另 {{ selectedPendingReports.length - 6 }} 条
                </span>
              </div>
            </div>

            <div v-if="reportActionNeedsDuration" class="form-group">
              <label>处理时长</label>
              <div class="duration-options">
                <label class="checkbox-label">
                  <input type="checkbox" v-model="reportActionPermanent" />
                  <span>永久</span>
                </label>
                <input
                  v-if="!reportActionPermanent"
                  v-model.number="reportActionDuration"
                  type="number"
                  min="1"
                  placeholder="小时数"
                  class="form-input duration-input"
                />
                <span v-if="!reportActionPermanent" class="duration-unit">小时</span>
              </div>
            </div>

            <div class="form-group">
              <label>版主备注</label>
              <textarea
                v-model="reportActionComment"
                placeholder="可选，作为审核备注记录在举报处理结果里"
                rows="3"
              ></textarea>
            </div>

            <div class="warning-box" :class="{ danger: reportActionDanger }">
              <i :class="reportActionDanger ? 'ri-error-warning-line' : 'ri-information-line'"></i>
              <span>{{ getReportActionWarning() }}</span>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="closeReportActionModal()">取消</button>
            <button
              class="btn-submit"
              :class="{ danger: reportActionDanger }"
              :disabled="reportActionSubmitting"
              @click="submitReportAction"
            >
              {{ reportActionSubmitting ? '处理中...' : `确认${getReportActionTitle()}` }}
            </button>
          </div>
        </div>
      </div>

      <!-- 审核预览弹窗 -->
      <div v-if="showPreviewModal" class="modal-overlay preview-overlay" @click.self="showPreviewModal = false">
        <div class="modal preview-modal">
          <div class="modal-header">
            <h3>
              <i :class="previewType === 'post' ? 'ri-article-line' : previewType === 'item' ? 'ri-gift-line' : 'ri-team-line'"></i>
              {{ previewType === 'post' ? '帖子审核' : previewType === 'item' ? '作品审核' : '公会审核' }}
            </h3>
            <button class="close-btn" @click="showPreviewModal = false">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="modal-body preview-body">
            <div v-if="previewLoading" class="preview-loading">
              <i class="ri-loader-4-line loading-spinner"></i>
              <span>加载中...</span>
            </div>
            <template v-else-if="previewData">
              <!-- 帖子预览 -->
              <template v-if="previewType === 'post'">
                <div class="preview-header">
                  <h2 class="preview-title">{{ previewData.title }}</h2>
                  <div class="preview-meta">
                    <span>
                      <i class="ri-user-line"></i>
                      <span :style="buildNameStyle(previewData.author_name_color, previewData.author_name_bold)">{{ previewData.author_name }}</span>
                    </span>
                    <span><i class="ri-time-line"></i> {{ formatDate(previewData.created_at) }}</span>
                    <span class="category-tag">{{ previewData.category }}</span>
                  </div>
                </div>
                <div class="preview-content" v-html="previewData.content"></div>
              </template>

              <!-- 作品预览 -->
              <template v-else-if="previewType === 'item'">
                <!-- 权限警告横幅 -->
                <div v-if="previewData.requires_permission" class="permission-warning-banner">
                  <i class="ri-shield-keyhole-line"></i>
                  <div class="warning-content">
                    <strong>此作品需要 TRP3 权限授权</strong>
                    <p>用户需要在游戏内对道具 Shift+右键点击 来调整安全性设置后才能正常使用。请确认作品功能是否需要此权限。</p>
                  </div>
                </div>

                <!-- 预览图 -->
                <div v-if="previewData.preview_image" class="item-preview-image">
                  <img :src="previewData.preview_image" alt="预览图" />
                </div>

                <div class="preview-header">
                  <div class="item-title-row">
                    <img v-if="previewData.icon" :src="previewData.icon" class="item-icon" />
                    <h2 class="preview-title">{{ previewData.name }}</h2>
                  </div>
                  <div class="preview-meta">
                    <span>
                      <i class="ri-user-line"></i>
                      <span :style="buildNameStyle(previewData.author_name_color, previewData.author_name_bold)">{{ previewData.author_name }}</span>
                    </span>
                    <span><i class="ri-time-line"></i> {{ formatDate(previewData.created_at) }}</span>
                    <span class="type-tag">{{ getItemTypeLabel(previewData.type) }}</span>
                  </div>
                </div>

                <!-- 简介 -->
                <div v-if="previewData.description" class="preview-description">
                  <h4>简介</h4>
                  <p>{{ previewData.description }}</p>
                </div>

                <!-- 详细介绍 -->
                <div v-if="previewData.detail_content" class="preview-detail-content">
                  <h4>详细介绍</h4>
                  <div class="rich-content" v-html="previewData.detail_content"></div>
                </div>

                <!-- 标签 -->
                <div v-if="previewData.tags && previewData.tags.length > 0" class="preview-tags">
                  <span v-for="tag in previewData.tags" :key="tag.id" class="preview-tag">{{ tag.name }}</span>
                </div>

                <!-- 无内容提示 -->
                <div v-if="!previewData.description && !previewData.detail_content && !previewData.preview_image" class="empty-preview">
                  <i class="ri-file-text-line"></i>
                  <p>暂无详细内容</p>
                </div>
              </template>

              <!-- 公会预览 -->
              <template v-else>
                <div class="preview-header">
                  <h2 class="preview-title">{{ previewData.name }}</h2>
                  <div class="preview-meta">
                    <span>
                      <i class="ri-user-line"></i>
                      会长:
                      <span :style="buildNameStyle(previewData.owner_name_color, previewData.owner_name_bold)">{{ previewData.owner_name }}</span>
                    </span>
                    <span><i class="ri-time-line"></i> {{ formatDate(previewData.created_at) }}</span>
                  </div>
                </div>
                <div class="preview-content">
                  <p>{{ previewData.description || '暂无简介' }}</p>
                </div>
              </template>

              <!-- 审核操作区 -->
              <div class="review-section">
                <h4>审核操作</h4>
                <div class="review-actions">
                  <label class="radio-label">
                    <input type="radio" v-model="previewReviewAction" value="approve" />
                    <span class="radio-text approve">通过</span>
                  </label>
                  <label class="radio-label">
                    <input type="radio" v-model="previewReviewAction" value="reject" />
                    <span class="radio-text reject">拒绝</span>
                  </label>
                </div>
                <textarea
                  v-model="previewReviewComment"
                  placeholder="审核意见（可选）"
                  rows="2"
                ></textarea>
              </div>
            </template>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showPreviewModal = false">取消</button>
            <button class="btn-submit" @click="submitPreviewReview" :disabled="previewLoading">
              {{ previewReviewAction === 'approve' ? '通过' : '拒绝' }}
            </button>
          </div>
        </div>
      </div>

      <ImageViewer
        v-model="showImageViewer"
        :images="imageViewerImages"
        :start-index="imageViewerStartIndex"
        :show-download="true"
        download-label="下载图片"
        @download="downloadImageFromViewer"
      />
    </template>
  </div>
</template>

<style scoped>
.moderator-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.no-access {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100px 20px;
  color: var(--color-text-secondary, #8D7B68);
}

.no-access i {
  font-size: 80px;
  margin-bottom: 24px;
  opacity: 0.3;
}

.no-access h2 {
  font-size: 24px;
  color: var(--color-primary, #4B3621);
  margin-bottom: 8px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title {
  font-size: 42px;
  color: var(--color-primary, #4B3621);
  margin: 0;
}

.role-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: linear-gradient(135deg, var(--color-accent, #B87333), var(--color-secondary, #804030));
  color: var(--btn-primary-text, #fff);
  border-radius: 20px;
  font-weight: 600;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.stat-card.pending {
  background: linear-gradient(135deg, var(--color-warning-light, #FFF5E6), var(--color-warning-light, #FFE4CC));
  border: 2px solid var(--color-warning-border, #FFB366);
}

.stat-icon {
  width: 48px;
  height: 48px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: var(--color-secondary, #804030);
}

.stat-card.pending .stat-icon {
  background: rgba(255, 153, 51, 0.2);
  color: var(--color-warning-dark, #CC6600);
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
}

.tab-container {
  background: var(--color-primary, #4B3621);
  border-radius: 16px;
  padding: 8px;
  display: flex;
  gap: 8px;
}

.tab-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-main-bg, #EED9C4);
  transition: all 0.3s;
}

.tab-item.active {
  background: var(--color-main-bg, #EED9C4);
  color: var(--color-primary, #4B3621);
}

.tab-item .badge {
  background: var(--btn-danger-bg, #FF6B6B);
  color: var(--btn-primary-text, #fff);
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
}

.sub-tab-container {
  display: flex;
  gap: 8px;
}

.sub-tab-container button {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: var(--color-panel-bg, #fff);
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 10px;
  color: var(--color-text-secondary, #8D7B68);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.sub-tab-container button.active {
  background: var(--color-secondary, #804030);
  border-color: var(--color-secondary, #804030);
  color: var(--btn-primary-text, #fff);
}

.review-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  background: var(--btn-danger-bg, #FF6B6B);
  color: var(--btn-primary-text, #fff);
  border-radius: 9px;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.content-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-bar {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-bar input {
  flex: 1;
  min-width: 200px;
  padding: 10px 16px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 14px;
}

.filter-bar select {
  padding: 10px 16px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 14px;
  background: var(--color-panel-bg, #fff);
}

.item-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.item-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.item-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
}

.status-badges {
  display: flex;
  gap: 8px;
}

.status-badge {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.pending {
  background: var(--color-warning-light, #FFF3E0);
  color: var(--color-warning-dark, #E65100);
}

.status-badge.approved, .status-badge.published {
  background: var(--color-success-light, #E8F5E9);
  color: var(--color-success, #2E7D32);
}

.status-badge.rejected, .status-badge.draft {
  background: var(--color-warning-light, #FFEBEE);
  color: var(--btn-danger-bg, #C62828);
}

.item-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--color-text-secondary, #8D7B68);
  margin-bottom: 12px;
}

.item-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.item-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.image-review-card {
  overflow: hidden;
}

.image-review-body {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.image-review-thumb {
  width: 108px;
  height: 108px;
  border-radius: 8px;
  object-fit: cover;
  border: 1px solid var(--color-border, #E5D4C1);
  cursor: pointer;
  flex-shrink: 0;
}

.image-review-thumb.avatar {
  border-radius: 50%;
}

.image-review-main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-review-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary, #5D4E37);
  white-space: pre-wrap;
  word-break: break-word;
}

.image-review-comment {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 13px;
  color: var(--btn-primary-text, var(--color-primary, #4B3621));
  resize: vertical;
  background: var(--color-panel-bg, #fff);
}

.image-review-history {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

@media (max-width: 768px) {
  .image-review-body {
    flex-direction: column;
  }

  .image-review-thumb {
    width: 100%;
    height: 180px;
  }

  .image-review-thumb.avatar {
    width: 120px;
    height: 120px;
  }
}

.btn-approve {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-success, #4CAF50);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-approve:hover {
  background: var(--color-success, #388E3C);
}

.btn-sponsor {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: linear-gradient(135deg, var(--color-warning, #E7C67D), var(--color-accent, #D6A645));
  color: var(--btn-primary-text, var(--color-primary, #4B3621));
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

.btn-sponsor:hover {
  background: linear-gradient(135deg, var(--color-warning-light, #EED79A), var(--color-accent, #E1B256));
}

.sponsor-level-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.experience-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.sponsor-level-select {
  min-width: 160px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--color-border, #E2D3C3);
  background: var(--input-bg, #FFFDFB);
  font-size: 13px;
  color: var(--btn-primary-text, var(--color-primary, #4B3621));
}

.experience-label {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-secondary, #6D5B48);
}

.experience-input {
  width: 140px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--color-border, #E2D3C3);
  background: var(--input-bg, #FFFDFB);
  font-size: 13px;
  color: var(--btn-primary-text, var(--color-primary, #4B3621));
}

.btn-reject {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--btn-danger-bg, #FF5722);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-reject:hover {
  background: var(--btn-danger-hover, #E64A19);
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--btn-danger-bg, #F44336);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

/* 置顶按钮 */
.btn-pin {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-secondary, #9C27B0);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-pin:hover {
  background: var(--color-secondary, #7B1FA2);
}

.btn-pin.active {
  background: var(--color-primary-light, #E1BEE7);
  color: var(--color-secondary, #7B1FA2);
}

/* 精华按钮 */
.btn-feature {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-warning, #FF9800);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-feature:hover {
  background: var(--color-warning, #F57C00);
}

.btn-feature.active {
  background: var(--color-warning-light, #FFE0B2);
  color: var(--color-warning-dark, #E65100);
}

/* 标题带标签 */
.title-with-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mod-tag {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.mod-tag.pinned {
  background: var(--color-primary-light, #F3E5F5);
  color: var(--color-secondary, #7B1FA2);
}

.mod-tag.featured {
  background: var(--color-warning-light, #FFF3E0);
  color: var(--color-warning-dark, #E65100);
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  width: 90%;
  max-width: 480px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--color-border, #E5D4C1);
}

.modal-header h3 {
  margin: 0;
  color: var(--color-text-main, #2C1810);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--color-text-secondary, #8D7B68);
  cursor: pointer;
}

.modal-body {
  padding: 20px;
}

.review-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  margin-bottom: 16px;
}

.review-actions {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.radio-text.approve { color: var(--color-success, #4CAF50); font-weight: 600; }
.radio-text.reject { color: var(--btn-danger-bg, #F44336); font-weight: 600; }

.modal-body textarea {
  width: 100%;
  padding: 12px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid var(--color-border, #E5D4C1);
}

.btn-cancel {
  padding: 10px 20px;
  background: var(--color-panel-bg, #fff);
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  color: var(--color-text-secondary, #8D7B68);
  font-weight: 600;
  cursor: pointer;
}

.btn-submit {
  padding: 10px 20px;
  background: var(--color-secondary, #804030);
  border: none;
  border-radius: 8px;
  color: var(--btn-primary-text, #fff);
  font-weight: 600;
  cursor: pointer;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 20px;
  color: var(--color-text-secondary, #8D7B68);
}

.empty-state i {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 60px 20px;
  color: var(--color-text-secondary, #8D7B68);
}

.loading-spinner {
  font-size: 32px;
  color: var(--color-secondary, #804030);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.anim-item { opacity: 0; transform: translateY(20px); }
.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.15s);
}
@keyframes fadeUp { to { opacity: 1; transform: translateY(0); } }

/* 新增样式 - 编辑按钮 */
.btn-edit {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-secondary, #2196F3);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-edit:hover {
  background: var(--color-secondary-hover, #1976D2);
}

/* 警告按钮 */
.btn-warning {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-warning, #FF9800);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-warning:hover {
  background: var(--color-warning, #F57C00);
}

/* 用户卡片样式 */
.user-card .item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.user-avatar-placeholder {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-border, #E5D4C1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--color-text-secondary, #8D7B68);
}

/* 角色标签 */
.sponsor-tag {
  padding: 5px 12px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-warning, #E7C67D), var(--color-accent, #D6A645));
  color: var(--btn-primary-text, var(--color-primary, #4B3621));
  border: 1px solid rgba(214, 166, 69, 0.4);
}

.level-tag {
  padding: 5px 12px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 700;
  background: var(--color-primary-light, #F3E8D8);
  color: var(--color-primary, #5C432B);
  border: 1px solid rgba(92, 67, 43, 0.12);
}

.role-tag {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.role-tag.user {
  background: var(--color-primary-light, #E3F2FD);
  color: var(--color-accent, #1565C0);
}

.role-tag.moderator {
  background: var(--color-warning-light, #FFF3E0);
  color: var(--color-warning-dark, #E65100);
}

.role-tag.admin {
  background: var(--color-primary-light, #FCE4EC);
  color: var(--color-accent, #C2185B);
}

/* 角色设置弹窗样式 */
.role-change-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-card-bg, #F5F0EB);
  border-radius: 12px;
  margin-bottom: 16px;
}

.user-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
}

.user-preview i {
  font-size: 24px;
  color: var(--color-secondary, #804030);
}

.role-arrow {
  display: flex;
  align-items: center;
  gap: 12px;
}

.role-arrow i {
  font-size: 20px;
  color: var(--color-text-secondary, #8D7B68);
}

.confirm-text {
  font-size: 14px;
  color: var(--color-text-secondary, #5D4E37);
  text-align: center;
  line-height: 1.6;
}

/* 表单样式 */
.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary, #4B3621);
  margin-bottom: 8px;
}

.form-input {
  width: 100%;
  padding: 12px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 14px;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-secondary, #804030);
}

.hint-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-secondary, #8D7B68);
  margin-top: 8px;
}

/* 管理标签特殊样式 */
.tab-item.admin-tab {
  background: rgba(194, 24, 91, 0.1);
}

.tab-item.admin-tab.active {
  background: var(--color-accent, #C2185B);
  color: var(--btn-primary-text, #fff);
}

/* 禁用按钮 */
.btn-submit:disabled {
  background: var(--color-border, #ccc);
  cursor: not-allowed;
}

/* 预览按钮 */
.btn-preview {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-secondary, #7C4DFF);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-preview:hover {
  background: var(--color-secondary-hover, #651FFF);
}

/* 编辑按钮 */
.btn-edit {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--color-success, #2F855A);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-edit:hover {
  background: var(--color-success, #276749);
}

/* 预览弹窗 */
.preview-overlay {
  z-index: 1001;
}

.preview-modal {
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.preview-body {
  flex: 1;
  overflow-y: auto;
  max-height: 60vh;
}

.preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  gap: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

.preview-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border, #E5D4C1);
}

.preview-title {
  font-size: 24px;
  color: var(--color-text-main, #2C1810);
  margin: 0 0 12px 0;
}

.preview-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--color-text-secondary, #8D7B68);
}

.preview-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.category-tag, .type-tag {
  padding: 2px 8px;
  background: var(--color-border, #E5D4C1);
  border-radius: 4px;
  font-size: 12px;
}

.preview-content {
  line-height: 1.8;
  color: var(--color-primary, #4B3621);
  margin-bottom: 20px;
}

.review-section {
  background: var(--color-card-bg, #F5F0EB);
  border-radius: 12px;
  padding: 16px;
}

.review-section h4 {
  margin: 0 0 12px 0;
  color: var(--color-primary, #4B3621);
  font-size: 14px;
}

/* 危险按钮 */
.btn-danger {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: var(--btn-danger-bg, #D32F2F);
  color: var(--btn-primary-text, #fff);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-danger:hover {
  background: var(--btn-danger-hover, #B71C1C);
}

/* 用户状态标签 */
.user-status-tags {
  display: flex;
  gap: 8px;
  align-items: center;
}

.status-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.status-tag.muted {
  background: var(--color-warning-light, #FFF3E0);
  color: var(--color-warning-dark, #E65100);
}

.status-tag.banned {
  background: var(--color-warning-light, #FFEBEE);
  color: var(--btn-danger-bg, #C62828);
}

/* 封禁信息 */
.ban-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 12px;
  background: var(--color-warning-light, #FFF8E1);
  border-radius: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #5D4037);
  margin-bottom: 12px;
}

.ban-info i {
  margin-right: 4px;
}

/* 用户操作弹窗样式 */
.user-action-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--color-card-bg, #F5F0EB);
  border-radius: 12px;
  margin-bottom: 16px;
}

.action-description {
  font-size: 14px;
  color: var(--color-text-secondary, #5D4E37);
  text-align: center;
  margin: 0;
}

.duration-options {
  display: flex;
  align-items: center;
  gap: 12px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 14px;
  color: var(--color-primary, #4B3621);
}

.duration-input {
  width: 80px !important;
}

.duration-unit {
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
}

/* 系统通知 */
.system-message-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.system-message-textarea {
  width: 100%;
  min-height: 140px;
  padding: 12px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  font-size: 14px;
  color: var(--color-primary, #4B3621);
  background: var(--input-bg, #FFFDFB);
  resize: vertical;
}

.system-message-textarea:focus {
  outline: none;
  border-color: var(--color-secondary, #804030);
}

.system-message-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.system-message-hint {
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

.char-count {
  font-size: 12px;
  color: var(--color-text-secondary, #8D7B68);
}

/* 警告框 */
.warning-box {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  background: var(--color-warning-light, #FFF8E1);
  border: 1px solid var(--color-warning-border, #FFE082);
  border-radius: 8px;
  font-size: 14px;
  color: var(--color-text-secondary, #5D4037);
}

.warning-box i {
  font-size: 20px;
  color: var(--color-warning, #FF9800);
  flex-shrink: 0;
}

.warning-box.danger {
  background: var(--color-warning-light, #FFEBEE);
  border-color: var(--btn-danger-bg, #EF9A9A);
  color: var(--btn-danger-hover, #B71C1C);
}

.warning-box.danger i {
  color: var(--btn-danger-bg, #D32F2F);
}

/* 危险提交按钮 */
.btn-submit.danger {
  background: var(--btn-danger-bg, #D32F2F);
}

.btn-submit.danger:hover {
  background: var(--btn-danger-hover, #B71C1C);
}

/* 权限警告标签 - 道具列表 */
.title-with-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.permission-warning-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: var(--color-warning-light, #FFF3E0);
  color: var(--color-warning-dark, #E65100);
  border: 1px solid var(--color-warning-border, #FFB74D);
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.permission-warning-tag i {
  font-size: 12px;
}

/* 需要权限的道具卡片高亮 */
.item-card.has-permission {
  border-left: 3px solid var(--color-warning-dark, #E65100);
  background: linear-gradient(90deg, var(--color-warning-light, #FFF8E1) 0%, var(--color-panel-bg, #fff) 20%);
}

/* 权限警告横幅 - 预览弹窗 */
.permission-warning-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, var(--color-warning-light, #FFF3E0) 0%, var(--color-warning-light, #FFE0B2) 100%);
  border: 2px solid var(--color-warning-border, #FFB74D);
  border-radius: 12px;
  margin-bottom: 20px;
}

.permission-warning-banner > i {
  font-size: 24px;
  color: var(--color-warning-dark, #E65100);
  flex-shrink: 0;
}

.permission-warning-banner .warning-content {
  flex: 1;
}

.permission-warning-banner .warning-content strong {
  display: block;
  font-size: 15px;
  color: var(--color-warning-dark, #E65100);
  margin-bottom: 4px;
}

.permission-warning-banner .warning-content p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary, #5D4037);
  line-height: 1.5;
}

/* 操作日志表格样式 */
.logs-table-wrapper {
  overflow-x: auto;
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.logs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.logs-table th,
.logs-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid var(--color-border, #E5D4C1);
}

.logs-table th {
  background: var(--color-card-bg, #FAF7F2);
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  white-space: nowrap;
}

.logs-table tr:hover {
  background: var(--color-card-bg, #FAF7F2);
}

.logs-table .operator {
  font-weight: 500;
  color: var(--color-text-main, #2C1810);
  margin-right: 8px;
}

.logs-table .role-tag {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.logs-table .role-tag.admin {
  background: var(--color-warning-light, #FFE4E1);
  color: var(--btn-danger-bg, #C0392B);
}

.logs-table .role-tag.moderator {
  background: var(--color-primary-light, #E3F2FD);
  color: var(--color-secondary-hover, #1976D2);
}

.logs-table .action-type {
  color: var(--color-text-secondary, #5D4037);
  font-weight: 500;
}

.logs-table .target-type {
  font-size: 11px;
  padding: 2px 6px;
  background: var(--color-border, #E5D4C1);
  border-radius: 4px;
  margin-right: 8px;
  color: var(--color-text-secondary, #5D4037);
}

.logs-table .target-name {
  color: var(--color-text-main, #2C1810);
}

.logs-table .details-cell {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-muted, #7B6B5A);
}

.logs-table .ip-cell {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text-muted, #7B6B5A);
}

.logs-table .time-cell {
  white-space: nowrap;
  color: var(--color-text-muted, #7B6B5A);
  font-size: 13px;
}

.report-card {
  gap: 14px;
}

.report-card-header {
  gap: 12px;
}

.report-scope-tabs {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.report-scope-tabs button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-card-bg, #FFFFFF);
  color: var(--color-text-secondary, #8D7B68);
  cursor: pointer;
  transition: all 0.2s ease;
}

.report-scope-tabs button.active {
  border-color: rgba(154, 52, 18, 0.26);
  background: rgba(251, 146, 60, 0.12);
  color: #9A3412;
}

.report-batch-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  padding: 14px 16px;
  margin-bottom: 14px;
  border-radius: 14px;
  background: rgba(251, 146, 60, 0.08);
  border: 1px solid rgba(154, 52, 18, 0.12);
}

.report-batch-info {
  color: var(--color-text-secondary, #8D7B68);
  font-size: 13px;
  font-weight: 600;
}

.report-batch-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.report-batch-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.report-select-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-main, #2C1810);
  font-size: 13px;
  font-weight: 600;
  user-select: none;
}

.report-select-toggle input {
  width: 16px;
  height: 16px;
  accent-color: #9A3412;
}

.report-select-toggle.card-toggle {
  flex-shrink: 0;
}

.report-title-wrap,
.report-tags {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.report-type-tag,
.report-count-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--color-card-bg, #FAF7F2);
  color: var(--color-text-muted, #7B6B5A);
  font-size: 12px;
}

.report-count-tag {
  color: #9A3412;
  background: rgba(251, 146, 60, 0.12);
}

.report-preview-card {
  display: flex;
  gap: 14px;
  padding: 14px;
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.04);
  align-items: flex-start;
}

.report-preview-image {
  width: 84px;
  height: 84px;
  object-fit: cover;
  border-radius: 12px;
  border: 1px solid var(--color-border, #E5D4C1);
  background: var(--color-card-bg, #FFFFFF);
}

.report-preview-main {
  flex: 1;
  min-width: 0;
}

.report-preview-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary, #8D7B68);
  margin-bottom: 6px;
}

.report-preview-text {
  margin: 0;
  color: var(--color-text-main, #2C1810);
  line-height: 1.7;
  white-space: pre-wrap;
}

.report-preview-text.muted {
  color: var(--color-text-muted, #7B6B5A);
}

.report-reason-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: 16px;
  background: linear-gradient(180deg, var(--color-card-bg, #FDFBF9) 0%, var(--color-panel-bg, #FFFFFF) 100%);
  border: 1px solid var(--color-border, #E8DCCF);
}

.report-section-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-main, #2C1810);
  font-size: 14px;
  font-weight: 700;
}

.report-reason-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.report-reason-item {
  padding: 14px;
  border-radius: 14px;
  background: var(--color-panel-bg, #FFFFFF);
  border: 1px solid var(--color-border, #E8DCCF);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.05);
}

.report-reason-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.report-reason-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.report-reason-meta {
  font-size: 13px;
  color: var(--color-text-main, #2C1810);
  font-weight: 700;
}

.report-reason-summary {
  color: var(--btn-secondary-text, var(--color-text-main, #2C1810));
  font-size: 13px;
  font-weight: 700;
  line-height: 1.6;
}

.report-review-text {
  padding: 12px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.7;
}

.report-reason-time {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
  font-weight: 600;
}

.report-evidence-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  background: var(--color-panel-bg, #FFFFFF);
}

.report-evidence-row + .report-evidence-row {
  margin-top: 10px;
}

.report-evidence-row.detail-row {
  background: var(--color-card-bg, #FDFBF9);
  border-color: var(--color-border-light, #F0E6DC);
}

.report-evidence-row:not(.detail-row) {
  border-color: var(--color-border-hover, #D4A373);
  background: var(--btn-secondary-bg, rgba(128, 64, 48, 0.1));
}

.report-evidence-title {
  color: var(--btn-secondary-text, var(--color-text-main, #2C1810));
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.report-evidence-content {
  color: var(--color-text-main, #2C1810);
  font-size: 14px;
  font-weight: 700;
  white-space: pre-wrap;
  line-height: 1.7;
}

.report-evidence-content.strong {
  color: var(--btn-secondary-text, var(--color-text-main, #2C1810));
}

.report-card .item-meta,
.report-card .item-meta span {
  color: var(--color-text-secondary, #8C7B70);
}

.report-card .item-title {
  color: var(--color-text-main, #2C1810);
}

@media (max-width: 768px) {
  .report-batch-toolbar {
    align-items: stretch;
  }

  .report-batch-actions {
    width: 100%;
  }
}

.report-action-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.report-action-chip-list {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.report-action-chip {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.05);
  color: var(--color-text-main, #2C1810);
  font-size: 12px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.report-action-chip.more-chip {
  color: var(--color-text-secondary, #8D7B68);
}

.report-review-text {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: rgba(34, 197, 94, 0.08);
  color: #166534;
}

/* 数据统计样式 */
.metrics-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.summary-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.summary-header {
  margin-bottom: 12px;
}

.summary-header .period {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-secondary, #5D4037);
  padding: 4px 10px;
  background: var(--color-card-bg, #FAF7F2);
  border-radius: 6px;
}

.summary-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.summary-item {
  display: flex;
  flex-direction: column;
}

.summary-item .label {
  font-size: 12px;
  color: var(--color-text-muted, #7B6B5A);
  margin-bottom: 4px;
}

.summary-item .value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
}

.metrics-chart-container {
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.metrics-chart-container h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  display: flex;
  align-items: center;
  gap: 8px;
}

.metrics-chart-container h4 i {
  color: var(--color-secondary, #8B4513);
}

.metrics-chart {
  width: 100%;
  height: 400px;
}

.metrics-subtabs {
  margin-bottom: 8px;
}

.basic-metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.basic-metric-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.basic-metric-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: rgba(128, 64, 48, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: var(--color-secondary, #804030);
  flex-shrink: 0;
}

.basic-metric-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.basic-metric-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
}

.basic-metric-label {
  font-size: 13px;
  color: var(--color-text-secondary, #8D7B68);
}

/* 分页样式 */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
}

.pagination button {
  width: 36px;
  height: 36px;
  border: 2px solid var(--color-border, #E5D4C1);
  border-radius: 8px;
  background: var(--color-panel-bg, #fff);
  color: var(--color-text-secondary, #5D4037);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.pagination button:hover:not(:disabled) {
  background: var(--color-card-bg, #FAF7F2);
  border-color: var(--color-secondary, #8B4513);
  color: var(--color-secondary, #8B4513);
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination span {
  font-size: 14px;
  color: var(--color-text-secondary, #5D4037);
}

/* 作品预览样式 */
.item-preview-image {
  margin-bottom: 20px;
  text-align: center;
}

.item-preview-image img {
  max-width: 100%;
  max-height: 300px;
  border-radius: 12px;
  object-fit: cover;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.item-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.item-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
  border: 2px solid var(--color-border, #E5D4C1);
}

.preview-description {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border, #E5D4C1);
}

.preview-description h4,
.preview-detail-content h4 {
  font-size: 14px;
  color: var(--color-text-secondary, #8D7B68);
  margin: 0 0 12px 0;
  font-weight: 600;
}

.preview-description p {
  color: var(--color-primary, #4B3621);
  line-height: 1.7;
  margin: 0;
  white-space: pre-wrap;
}

.preview-detail-content {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border, #E5D4C1);
}

.preview-detail-content .rich-content {
  color: var(--color-primary, #4B3621);
  line-height: 1.8;
}

.preview-detail-content .rich-content img {
  max-width: 100%;
  border-radius: 8px;
  margin: 12px 0;
}

.preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.preview-tag {
  padding: 4px 12px;
  background: var(--color-card-bg, #F5EBE0);
  color: var(--color-text-secondary, #8D7B68);
  border-radius: 12px;
  font-size: 12px;
}

.empty-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--color-text-secondary, #8D7B68);
}

.empty-preview i {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.4;
}

.empty-preview p {
  margin: 0;
  font-size: 14px;
}
</style>
