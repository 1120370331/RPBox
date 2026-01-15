<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { getPost } from '@/api/post'
import { getItem } from '@/api/item'
import { getGuild } from '@/api/guild'
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
  getPendingGuilds,
  reviewGuild,
  getAllGuilds,
  changeGuildOwner,
  deleteGuildByMod,
  getUsers,
  setUserRole,
  type ModeratorStats,
  type ReviewRequest,
  type SafeUser
} from '@/api/moderator'

const router = useRouter()
const userStore = useUserStore()
const mounted = ref(false)

// 权限检查
const hasAccess = computed(() => userStore.isModerator)
const isAdmin = computed(() => userStore.isAdmin)

// 标签页
const activeTab = ref<'review' | 'manage' | 'admin'>('review')
const activeSubTab = ref<'posts' | 'items' | 'guilds'>('posts')
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
const newOwnerId = ref('')

// 设置角色弹窗
const showRoleModal = ref(false)
const roleTarget = ref<{ userId: number; username: string; currentRole: string; newRole: 'user' | 'moderator' } | null>(null)

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

function switchTab(tab: 'review' | 'manage' | 'admin') {
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
  }
}

function switchSubTab(subTab: 'posts' | 'items' | 'guilds') {
  activeSubTab.value = subTab
  page.value = 1
  if (activeTab.value === 'review') {
    if (subTab === 'posts') loadPendingPosts()
    else if (subTab === 'items') loadPendingItems()
    else loadPendingGuilds()
  } else {
    if (subTab === 'posts') loadAllPosts()
    else if (subTab === 'items') loadAllItems()
    else loadAllGuilds()
  }
}

function switchAdminSubTab(subTab: 'moderators' | 'guilds') {
  adminSubTab.value = subTab
  page.value = 1
  if (subTab === 'moderators') loadUsers()
  else loadAllGuilds()
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
      const res = await getItem(id)
      previewData.value = res.item
    } else {
      const res = await getGuild(id)
      previewData.value = res.guild
    }
  } catch (error) {
    console.error('加载详情失败:', error)
    alert('加载详情失败')
    showPreviewModal.value = false
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
  if (!confirm('确定要删除这篇帖子吗？此操作不可恢复。')) return
  try {
    await deletePostByMod(id)
    await loadAllPosts()
    await loadStats()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败: ' + (error as Error).message)
  }
}

async function handleDeleteItem(id: number) {
  if (!confirm('确定要删除这个道具吗？此操作不可恢复。')) return
  try {
    await deleteItemByMod(id)
    await loadAllItems()
    await loadStats()
  } catch (error) {
    console.error('删除失败:', error)
    alert('删除失败: ' + (error as Error).message)
  }
}

async function handleDeleteGuild(id: number) {
  if (!confirm('确定要删除这个公会吗？此操作不可恢复。')) return
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
  newOwnerId.value = ''
  showChangeOwnerModal.value = true
}

async function submitChangeOwner() {
  if (!changeOwnerTarget.value || !newOwnerId.value) return
  try {
    await changeGuildOwner(changeOwnerTarget.value.id, parseInt(newOwnerId.value))
    showChangeOwnerModal.value = false
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
            <div class="stat-label">待审核道具</div>
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
            <div class="stat-label">总道具数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon"><i class="ri-group-line"></i></div>
          <div class="stat-info">
            <div class="stat-value">{{ stats?.total_guilds || 0 }}</div>
            <div class="stat-label">总公会数</div>
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
      </div>

      <!-- 子标签页 - 审核/管理中心 -->
      <div v-if="activeTab !== 'admin'" class="sub-tab-container anim-item" style="--delay: 3">
        <button
          :class="{ active: activeSubTab === 'posts' }"
          @click="switchSubTab('posts')"
        >
          <i class="ri-article-line"></i>
          帖子
        </button>
        <button
          :class="{ active: activeSubTab === 'items' }"
          @click="switchSubTab('items')"
        >
          <i class="ri-gift-line"></i>
          道具
        </button>
        <button
          :class="{ active: activeSubTab === 'guilds' }"
          @click="switchSubTab('guilds')"
        >
          <i class="ri-team-line"></i>
          公会
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
              <button class="btn-preview" @click="openPreview('post', post.id)">
                <i class="ri-eye-line"></i> 预览审核
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
          <p>暂无待审核道具</p>
        </div>
        <div v-else class="item-list">
          <div v-for="item in pendingItems" :key="item.id" class="item-card">
            <div class="item-header">
              <span class="item-title">{{ item.name }}</span>
              <span class="status-badge pending">待审核</span>
            </div>
            <div class="item-meta">
              <span><i class="ri-user-line"></i> {{ item.author_name }}</span>
              <span><i class="ri-time-line"></i> {{ formatDate(item.created_at) }}</span>
            </div>
            <div class="item-actions">
              <button class="btn-preview" @click="openPreview('item', item.id)">
                <i class="ri-eye-line"></i> 预览审核
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
              <button class="btn-preview" @click="openPreview('guild', guild.id)">
                <i class="ri-eye-line"></i> 预览审核
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
              <span class="item-title">{{ post.title }}</span>
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

      <!-- 审核弹窗 -->
      <div v-if="showReviewModal" class="modal-overlay" @click.self="showReviewModal = false">
        <div class="modal">
          <div class="modal-header">
            <h3>审核{{ reviewTarget?.type === 'post' ? '帖子' : reviewTarget?.type === 'item' ? '道具' : '公会' }}</h3>
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
              <label>新会长用户ID</label>
              <input
                v-model="newOwnerId"
                type="number"
                placeholder="请输入新会长的用户ID"
                class="form-input"
              />
            </div>
            <p class="hint-text">
              <i class="ri-information-line"></i>
              当前会长ID: {{ changeOwnerTarget?.currentOwnerId }}
            </p>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showChangeOwnerModal = false">取消</button>
            <button class="btn-submit" @click="submitChangeOwner" :disabled="!newOwnerId">确认</button>
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
                ? '确定要将该用户设为版主吗？版主可以审核帖子、道具和公会。'
                : '确定要取消该用户的版主权限吗？' }}
            </p>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="showRoleModal = false">取消</button>
            <button class="btn-submit" @click="submitRoleChange">确认</button>
          </div>
        </div>
      </div>

      <!-- 审核预览弹窗 -->
      <div v-if="showPreviewModal" class="modal-overlay preview-overlay" @click.self="showPreviewModal = false">
        <div class="modal preview-modal">
          <div class="modal-header">
            <h3>
              <i :class="previewType === 'post' ? 'ri-article-line' : previewType === 'item' ? 'ri-gift-line' : 'ri-team-line'"></i>
              {{ previewType === 'post' ? '帖子审核' : previewType === 'item' ? '道具审核' : '公会审核' }}
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

              <!-- 道具预览 -->
              <template v-else-if="previewType === 'item'">
                <div class="preview-header">
                  <h2 class="preview-title">{{ previewData.name }}</h2>
                  <div class="preview-meta">
                    <span><i class="ri-user-line"></i> {{ previewData.author_name }}</span>
                    <span><i class="ri-time-line"></i> {{ formatDate(previewData.created_at) }}</span>
                    <span class="type-tag">{{ previewData.type }}</span>
                  </div>
                </div>
                <div class="preview-content">
                  <p>{{ previewData.description }}</p>
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
</style>