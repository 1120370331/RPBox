<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { dialog } from '@/composables/useDialog'
import { getPost } from '@/api/post'
import { getItem } from '@/api/item'
import { getGuild } from '@/api/guild'
import * as echarts from 'echarts'
import {
  getModeratorStats,
  getPendingPosts,
  getPendingItems,
  reviewPost,
  reviewItem,
  getAllPosts,
  getAllItems,
  deletePostByMod,
  deleteItemByMod,
  hidePostByMod,
  hideItemByMod,
  pinPost,
  featurePost,
  getPendingGuilds,
  reviewGuild,
  getAllGuilds,
  changeGuildOwner,
  deleteGuildByMod,
  getUsers,
  setUserRole,
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
  type ModeratorStats,
  type ReviewRequest,
  type SafeUser,
  type AdminActionLog,
  type DailyMetrics,
  type MetricsSummary
} from '@/api/moderator'

const router = useRouter()
const userStore = useUserStore()
const mounted = ref(false)

// 权限检查
const hasAccess = computed(() => userStore.isModerator)
const isAdmin = computed(() => userStore.isAdmin)

// 标签页
const activeTab = ref<'review' | 'manage' | 'admin' | 'logs' | 'metrics'>('review')
const activeSubTab = ref<'posts' | 'items' | 'guilds' | 'users'>('posts')
const adminSubTab = ref<'moderators' | 'guilds'>('guilds')

// 数据
const stats = ref<ModeratorStats | null>(null)
const pendingPosts = ref<any[]>([])
const pendingItems = ref<any[]>([])
const pendingGuilds = ref<any[]>([])
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
const filterRole = ref('')
const filterGuildStatus = ref('')

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

// 更换会长弹窗
const showChangeOwnerModal = ref(false)
const changeOwnerTarget = ref<{ id: number; name: string; currentOwnerId: number } | null>(null)
const newOwnerIdentifier = ref('')  // 用户名或邮箱

// 设置角色弹窗
const showRoleModal = ref(false)
const roleTarget = ref<{ userId: number; username: string; currentRole: string; newRole: 'user' | 'moderator' } | null>(null)

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
    const res = await getAllPosts({
      page: page.value,
      page_size: pageSize.value,
      status: filterStatus.value || undefined,
      review_status: filterReviewStatus.value || undefined,
      keyword: filterKeyword.value || undefined
    })
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

function switchTab(tab: 'review' | 'manage' | 'admin' | 'logs' | 'metrics') {
  activeTab.value = tab
  page.value = 1
  if (tab === 'review') {
    if (activeSubTab.value === 'posts') loadPendingPosts()
    else if (activeSubTab.value === 'items') loadPendingItems()
    else loadPendingGuilds()
  } else if (tab === 'manage') {
    if (activeSubTab.value === 'posts') loadAllPosts()
    else if (activeSubTab.value === 'items') loadAllItems()
    else loadAllGuilds()
  } else if (tab === 'admin') {
    if (adminSubTab.value === 'moderators') loadUsers()
    else loadAllGuilds()
  } else if (tab === 'logs') {
    loadActionLogs()
  } else if (tab === 'metrics') {
    loadMetrics()
  }
}

function switchSubTab(subTab: 'posts' | 'items' | 'guilds' | 'users') {
  activeSubTab.value = subTab
  page.value = 1
  if (activeTab.value === 'review') {
    if (subTab === 'posts') loadPendingPosts()
    else if (subTab === 'items') loadPendingItems()
    else loadPendingGuilds()
  } else {
    if (subTab === 'posts') loadAllPosts()
    else if (subTab === 'items') loadAllItems()
    else if (subTab === 'guilds') loadAllGuilds()
    else if (subTab === 'users') loadModeratorUsers()
  }
}

function switchAdminSubTab(subTab: 'moderators' | 'guilds') {
  adminSubTab.value = subTab
  page.value = 1
  if (subTab === 'moderators') loadUsers()
  else loadAllGuilds()
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
    'delete_post': '删除帖子',
    'hide_post': '屏蔽帖子',
    'pin_post': '置顶帖子',
    'feature_post': '精华帖子',
    'review_item': '审核作品',
    'delete_item': '删除作品',
    'hide_item': '屏蔽作品',
    'review_guild': '审核公会',
    'delete_guild': '删除公会',
    'change_guild_owner': '更换会长',
    'mute_user': '禁言用户',
    'unmute_user': '解除禁言',
    'ban_user': '封禁用户',
    'unban_user': '解除封禁',
    'set_role': '设置角色',
    'disable_posts': '禁用帖子',
    'delete_posts': '删除用户帖子'
  }
  return map[type] || type
}

function formatLogDetails(log: AdminActionLog): string {
  if (!log.details) return '-'
  try {
    const d = JSON.parse(log.details)
    const parts: string[] = []

    // 审核操作
    if (d.action) {
      parts.push(d.action === 'approve' ? '通过' : '拒绝')
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

    // 影响数量
    if (d.affected_count !== undefined) {
      parts.push(`${d.affected_count} 条`)
    }

    return parts.length > 0 ? parts.join(' | ') : '-'
  } catch {
    return '-'
  }
}

// ========== 数据统计 ==========
async function loadMetrics() {
  try {
    const [historyRes, summaryRes] = await Promise.all([
      getMetricsHistory(metricsDays.value),
      getMetricsSummary()
    ])
    metricsHistory.value = historyRes.metrics
    metricsSummary.value = summaryRes
    // 下一帧渲染图表
    setTimeout(() => initMetricsChart(), 100)
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

function initMetricsChart() {
  if (!metricsChartRef.value) return

  if (!metricsChart) {
    metricsChart = echarts.init(metricsChartRef.value)
  }

  const dates = metricsHistory.value.map((m: DailyMetrics) => m.date.slice(5)) // 只显示月-日
  const newUsers = metricsHistory.value.map((m: DailyMetrics) => m.new_users)
  const newPosts = metricsHistory.value.map((m: DailyMetrics) => m.new_posts)
  const newItems = metricsHistory.value.map((m: DailyMetrics) => m.new_items)
  const newGuilds = metricsHistory.value.map((m: DailyMetrics) => m.new_guilds)

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' }
    },
    legend: {
      data: ['新增用户', '新增帖子', '新增作品', '新增公会'],
      textStyle: { color: '#8D7B68' }
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
      axisLine: { lineStyle: { color: '#E5D4C1' } },
      axisLabel: { color: '#8D7B68' }
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: '#E5D4C1' } },
      axisLabel: { color: '#8D7B68' },
      splitLine: { lineStyle: { color: '#F5EBE0' } }
    },
    series: [
      { name: '新增用户', type: 'line', data: newUsers, smooth: true, itemStyle: { color: '#804030' } },
      { name: '新增帖子', type: 'line', data: newPosts, smooth: true, itemStyle: { color: '#4682B4' } },
      { name: '新增作品', type: 'line', data: newItems, smooth: true, itemStyle: { color: '#9370DB' } },
      { name: '新增公会', type: 'line', data: newGuilds, smooth: true, itemStyle: { color: '#6B9B6B' } }
    ]
  }

  metricsChart.setOption(option)
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
        tags: tags || []
      }
    } else {
      const res = await getGuild(id)
      previewData.value = res.guild
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

function openRoleModal(userId: number, username: string, currentRole: string, newRole: 'user' | 'moderator') {
  roleTarget.value = { userId, username, currentRole, newRole }
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
          <span v-if="(stats?.pending_posts || 0) + (stats?.pending_items || 0) + (stats?.pending_guilds || 0) > 0" class="badge">
            {{ (stats?.pending_posts || 0) + (stats?.pending_items || 0) + (stats?.pending_guilds || 0) }}
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
              <span><i class="ri-user-line"></i> {{ post.author_name }}</span>
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
              <span><i class="ri-user-line"></i> {{ item.author_name }}</span>
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
              <span><i class="ri-user-line"></i> {{ guild.owner_name }}</span>
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
              <span><i class="ri-user-line"></i> {{ post.author_name }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(post.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="router.push({ name: 'post-detail', params: { id: post.id } })">
                <i class="ri-eye-line"></i> 查看
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
              <span><i class="ri-user-line"></i> {{ item.author_name }}</span>
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
              <span><i class="ri-user-line"></i> {{ guild.owner_name }}</span>
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
                <span class="item-title">{{ user.username }}</span>
              </div>
              <div class="user-status-tags">
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
                <span class="item-title">{{ user.username }}</span>
              </div>
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
                @click="openRoleModal(user.id, user.username, user.role, 'moderator')"
              >
                <i class="ri-shield-star-line"></i> 设为版主
              </button>
              <button
                v-else-if="user.role === 'moderator'"
                class="btn-warning"
                @click="openRoleModal(user.id, user.username, user.role, 'user')"
              >
                <i class="ri-shield-line"></i> 取消版主
              </button>
            </div>
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
              <span><i class="ri-user-line"></i> {{ guild.owner_name }}</span>
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
                  <span class="operator">{{ log.operator_name }}</span>
                  <span class="role-tag" :class="log.operator_role">{{ log.operator_role === 'admin' ? '管理员' : '版主' }}</span>
                </td>
                <td><span class="action-type">{{ formatActionType(log.action_type) }}</span></td>
                <td>
                  <span class="target-type">{{ log.target_type }}</span>
                  <span class="target-name">{{ log.target_name || `#${log.target_id}` }}</span>
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
      <div v-if="activeTab === 'metrics'" class="content-list anim-item" style="--delay: 3">
        <div class="filter-bar">
          <select v-model="metricsDays" @change="loadMetrics">
            <option :value="7">最近 7 天</option>
            <option :value="14">最近 14 天</option>
            <option :value="30">最近 30 天</option>
            <option :value="60">最近 60 天</option>
            <option :value="90">最近 90 天</option>
          </select>
        </div>

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
          <h4><i class="ri-line-chart-line"></i> 增长趋势</h4>
          <div ref="metricsChartRef" class="metrics-chart"></div>
        </div>
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
                <span class="username">{{ roleTarget?.username }}</span>
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
                <span class="username">{{ userActionTarget?.username }}</span>
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
                    <span><i class="ri-user-line"></i> {{ previewData.author_name }}</span>
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
                    <span><i class="ri-user-line"></i> {{ previewData.author_name }}</span>
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
                    <span><i class="ri-user-line"></i> 会长: {{ previewData.owner_name }}</span>
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
  color: #8D7B68;
}

.no-access i {
  font-size: 80px;
  margin-bottom: 24px;
  opacity: 0.3;
}

.no-access h2 {
  font-size: 24px;
  color: #4B3621;
  margin-bottom: 8px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title {
  font-size: 42px;
  color: #4B3621;
  margin: 0;
}

.role-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: linear-gradient(135deg, #B87333, #804030);
  color: #fff;
  border-radius: 20px;
  font-weight: 600;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.stat-card.pending {
  background: linear-gradient(135deg, #FFF5E6, #FFE4CC);
  border: 2px solid #FFB366;
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
  color: #804030;
}

.stat-card.pending .stat-icon {
  background: rgba(255, 153, 51, 0.2);
  color: #CC6600;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #2C1810;
}

.stat-label {
  font-size: 14px;
  color: #8D7B68;
}

.tab-container {
  background: #4B3621;
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
  color: #EED9C4;
  transition: all 0.3s;
}

.tab-item.active {
  background: #EED9C4;
  color: #4B3621;
}

.tab-item .badge {
  background: #FF6B6B;
  color: #fff;
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
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 10px;
  color: #8D7B68;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.sub-tab-container button.active {
  background: #804030;
  border-color: #804030;
  color: #fff;
}

.review-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  background: #FF6B6B;
  color: #fff;
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
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  font-size: 14px;
}

.filter-bar select {
  padding: 10px 16px;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  font-size: 14px;
  background: #fff;
}

.item-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.item-card {
  background: #fff;
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
  color: #2C1810;
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
  background: #FFF3E0;
  color: #E65100;
}

.status-badge.approved, .status-badge.published {
  background: #E8F5E9;
  color: #2E7D32;
}

.status-badge.rejected, .status-badge.draft {
  background: #FFEBEE;
  color: #C62828;
}

.item-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #8D7B68;
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
}

.btn-approve {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #4CAF50;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-approve:hover {
  background: #388E3C;
}

.btn-reject {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #FF5722;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-reject:hover {
  background: #E64A19;
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #F44336;
  color: #fff;
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
  background: #9C27B0;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-pin:hover {
  background: #7B1FA2;
}

.btn-pin.active {
  background: #E1BEE7;
  color: #7B1FA2;
}

/* 精华按钮 */
.btn-feature {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #FF9800;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-feature:hover {
  background: #F57C00;
}

.btn-feature.active {
  background: #FFE0B2;
  color: #E65100;
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
  background: #F3E5F5;
  color: #7B1FA2;
}

.mod-tag.featured {
  background: #FFF3E0;
  color: #E65100;
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
  background: #fff;
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
  border-bottom: 1px solid #E5D4C1;
}

.modal-header h3 {
  margin: 0;
  color: #2C1810;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: #8D7B68;
  cursor: pointer;
}

.modal-body {
  padding: 20px;
}

.review-title {
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
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

.radio-text.approve { color: #4CAF50; font-weight: 600; }
.radio-text.reject { color: #F44336; font-weight: 600; }

.modal-body textarea {
  width: 100%;
  padding: 12px;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid #E5D4C1;
}

.btn-cancel {
  padding: 10px 20px;
  background: #fff;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  color: #8D7B68;
  font-weight: 600;
  cursor: pointer;
}

.btn-submit {
  padding: 10px 20px;
  background: #804030;
  border: none;
  border-radius: 8px;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 20px;
  color: #8D7B68;
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
  color: #8D7B68;
}

.loading-spinner {
  font-size: 32px;
  color: #804030;
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
  background: #2196F3;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-edit:hover {
  background: #1976D2;
}

/* 警告按钮 */
.btn-warning {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #FF9800;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-warning:hover {
  background: #F57C00;
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
  background: #E5D4C1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #8D7B68;
}

/* 角色标签 */
.role-tag {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.role-tag.user {
  background: #E3F2FD;
  color: #1565C0;
}

.role-tag.moderator {
  background: #FFF3E0;
  color: #E65100;
}

.role-tag.admin {
  background: #FCE4EC;
  color: #C2185B;
}

/* 角色设置弹窗样式 */
.role-change-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #F5F0EB;
  border-radius: 12px;
  margin-bottom: 16px;
}

.user-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
  color: #2C1810;
}

.user-preview i {
  font-size: 24px;
  color: #804030;
}

.role-arrow {
  display: flex;
  align-items: center;
  gap: 12px;
}

.role-arrow i {
  font-size: 20px;
  color: #8D7B68;
}

.confirm-text {
  font-size: 14px;
  color: #5D4E37;
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
  color: #4B3621;
  margin-bottom: 8px;
}

.form-input {
  width: 100%;
  padding: 12px;
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  font-size: 14px;
}

.form-input:focus {
  outline: none;
  border-color: #804030;
}

.hint-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #8D7B68;
  margin-top: 8px;
}

/* 管理标签特殊样式 */
.tab-item.admin-tab {
  background: rgba(194, 24, 91, 0.1);
}

.tab-item.admin-tab.active {
  background: #C2185B;
  color: #fff;
}

/* 禁用按钮 */
.btn-submit:disabled {
  background: #ccc;
  cursor: not-allowed;
}

/* 预览按钮 */
.btn-preview {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #7C4DFF;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-preview:hover {
  background: #651FFF;
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
  color: #8D7B68;
}

.preview-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #E5D4C1;
}

.preview-title {
  font-size: 24px;
  color: #2C1810;
  margin: 0 0 12px 0;
}

.preview-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #8D7B68;
}

.preview-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.category-tag, .type-tag {
  padding: 2px 8px;
  background: #E5D4C1;
  border-radius: 4px;
  font-size: 12px;
}

.preview-content {
  line-height: 1.8;
  color: #4B3621;
  margin-bottom: 20px;
}

.review-section {
  background: #F5F0EB;
  border-radius: 12px;
  padding: 16px;
}

.review-section h4 {
  margin: 0 0 12px 0;
  color: #4B3621;
  font-size: 14px;
}

/* 危险按钮 */
.btn-danger {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: #D32F2F;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.btn-danger:hover {
  background: #B71C1C;
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
  background: #FFF3E0;
  color: #E65100;
}

.status-tag.banned {
  background: #FFEBEE;
  color: #C62828;
}

/* 封禁信息 */
.ban-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 12px;
  background: #FFF8E1;
  border-radius: 6px;
  font-size: 12px;
  color: #5D4037;
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
  background: #F5F0EB;
  border-radius: 12px;
  margin-bottom: 16px;
}

.action-description {
  font-size: 14px;
  color: #5D4E37;
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
  color: #4B3621;
}

.duration-input {
  width: 80px !important;
}

.duration-unit {
  font-size: 14px;
  color: #8D7B68;
}

/* 警告框 */
.warning-box {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  background: #FFF8E1;
  border: 1px solid #FFE082;
  border-radius: 8px;
  font-size: 14px;
  color: #5D4037;
}

.warning-box i {
  font-size: 20px;
  color: #FF9800;
  flex-shrink: 0;
}

.warning-box.danger {
  background: #FFEBEE;
  border-color: #EF9A9A;
  color: #B71C1C;
}

.warning-box.danger i {
  color: #D32F2F;
}

/* 危险提交按钮 */
.btn-submit.danger {
  background: #D32F2F;
}

.btn-submit.danger:hover {
  background: #B71C1C;
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
  background: #FFF3E0;
  color: #E65100;
  border: 1px solid #FFB74D;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.permission-warning-tag i {
  font-size: 12px;
}

/* 需要权限的道具卡片高亮 */
.item-card.has-permission {
  border-left: 3px solid #E65100;
  background: linear-gradient(90deg, #FFF8E1 0%, #fff 20%);
}

/* 权限警告横幅 - 预览弹窗 */
.permission-warning-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, #FFF3E0 0%, #FFE0B2 100%);
  border: 2px solid #FFB74D;
  border-radius: 12px;
  margin-bottom: 20px;
}

.permission-warning-banner > i {
  font-size: 24px;
  color: #E65100;
  flex-shrink: 0;
}

.permission-warning-banner .warning-content {
  flex: 1;
}

.permission-warning-banner .warning-content strong {
  display: block;
  font-size: 15px;
  color: #E65100;
  margin-bottom: 4px;
}

.permission-warning-banner .warning-content p {
  margin: 0;
  font-size: 13px;
  color: #5D4037;
  line-height: 1.5;
}

/* 操作日志表格样式 */
.logs-table-wrapper {
  overflow-x: auto;
  background: #fff;
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
  border-bottom: 1px solid #E5D4C1;
}

.logs-table th {
  background: #FAF7F2;
  font-weight: 600;
  color: #2C1810;
  white-space: nowrap;
}

.logs-table tr:hover {
  background: #FAF7F2;
}

.logs-table .operator {
  font-weight: 500;
  color: #2C1810;
  margin-right: 8px;
}

.logs-table .role-tag {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.logs-table .role-tag.admin {
  background: #FFE4E1;
  color: #C0392B;
}

.logs-table .role-tag.moderator {
  background: #E3F2FD;
  color: #1976D2;
}

.logs-table .action-type {
  color: #5D4037;
  font-weight: 500;
}

.logs-table .target-type {
  font-size: 11px;
  padding: 2px 6px;
  background: #E5D4C1;
  border-radius: 4px;
  margin-right: 8px;
  color: #5D4037;
}

.logs-table .target-name {
  color: #2C1810;
}

.logs-table .details-cell {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #7B6B5A;
}

.logs-table .ip-cell {
  font-family: monospace;
  font-size: 12px;
  color: #7B6B5A;
}

.logs-table .time-cell {
  white-space: nowrap;
  color: #7B6B5A;
  font-size: 13px;
}

/* 数据统计样式 */
.metrics-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.summary-card {
  background: #fff;
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
  color: #5D4037;
  padding: 4px 10px;
  background: #FAF7F2;
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
  color: #7B6B5A;
  margin-bottom: 4px;
}

.summary-item .value {
  font-size: 20px;
  font-weight: 700;
  color: #2C1810;
}

.metrics-chart-container {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
}

.metrics-chart-container h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  display: flex;
  align-items: center;
  gap: 8px;
}

.metrics-chart-container h4 i {
  color: #8B4513;
}

.metrics-chart {
  width: 100%;
  height: 400px;
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
  border: 2px solid #E5D4C1;
  border-radius: 8px;
  background: #fff;
  color: #5D4037;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.pagination button:hover:not(:disabled) {
  background: #FAF7F2;
  border-color: #8B4513;
  color: #8B4513;
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination span {
  font-size: 14px;
  color: #5D4037;
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
  border: 2px solid #E5D4C1;
}

.preview-description {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #E5D4C1;
}

.preview-description h4,
.preview-detail-content h4 {
  font-size: 14px;
  color: #8D7B68;
  margin: 0 0 12px 0;
  font-weight: 600;
}

.preview-description p {
  color: #4B3621;
  line-height: 1.7;
  margin: 0;
  white-space: pre-wrap;
}

.preview-detail-content {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #E5D4C1;
}

.preview-detail-content .rich-content {
  color: #4B3621;
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
  background: #F5EBE0;
  color: #8D7B68;
  border-radius: 12px;
  font-size: 12px;
}

.empty-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #8D7B68;
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