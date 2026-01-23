<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@/stores/toast'
import { useDialog } from '@/composables/useDialog'
import { useUserStore } from '@/stores/user'
import {
  getGuild,
  listGuildMembers,
  updateMemberRole,
  removeMember,
  listGuildApplications,
  reviewGuildApplication,
  uploadGuildAvatar,
  type Guild,
  type GuildMember,
  type GuildApplication
} from '@/api/guild'
import { getImageUrl } from '@/api/item'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import { buildNameStyle } from '@/utils/userNameStyle'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const { confirm } = useDialog()
const userStore = useUserStore()

const guildId = computed(() => Number(route.params.id))
const guild = ref<Guild | null>(null)
const myRole = ref<string>('')
const loading = ref(false)
const activeTab = ref<'members' | 'applications'>('members')
const avatarUploading = ref(false)
const avatarInputRef = ref<HTMLInputElement | null>(null)

const guildAvatarUrl = computed(() => {
  if (!guild.value || (!guild.value.avatar_url && !guild.value.avatar)) return ''
  return getImageUrl('guild-avatar', guild.value.id, {
    w: 160,
    q: 80,
    v: guild.value.avatar_updated_at || guild.value.updated_at,
  })
})

// 成员管理
const members = ref<GuildMember[]>([])
const loadingMembers = ref(false)

// 申请管理
const applications = ref<GuildApplication[]>([])
const loadingApplications = ref(false)
const applicationFilter = ref<'pending' | 'all'>('pending')

// 权限检查
const isAdmin = computed(() => {
  return myRole.value === 'owner' || myRole.value === 'admin'
})

const isOwner = computed(() => {
  return myRole.value === 'owner'
})

// 加载公会信息
async function loadGuild() {
  try {
    loading.value = true
    const res = await getGuild(guildId.value)
    guild.value = res.guild
    myRole.value = res.my_role

    // 权限检查
    if (!isAdmin.value) {
      toast.error('无权访问此页面')
      router.push(`/guild/${guildId.value}`)
      return
    }
  } catch (e: any) {
    console.error('加载公会信息失败:', e)
    toast.error(e.message || '加载失败')
    router.push('/guild')
  } finally {
    loading.value = false
  }
}

function triggerAvatarUpload() {
  avatarInputRef.value?.click()
}

async function handleAvatarUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !guild.value) return

  if (file.size > 10 * 1024 * 1024) {
    toast.error('头像文件不能超过10MB')
    input.value = ''
    return
  }

  avatarUploading.value = true
  try {
    const res = await uploadGuildAvatar(guildId.value, file)
    guild.value.avatar = res.avatar
    if (res.avatar_updated_at) {
      guild.value.avatar_updated_at = res.avatar_updated_at
    }
    toast.success('头像更新成功')
  } catch (e: any) {
    console.error('头像上传失败:', e)
    toast.error(e.message || '头像上传失败')
  } finally {
    avatarUploading.value = false
    input.value = ''
  }
}

// 加载成员列表
async function loadMembers() {
  try {
    loadingMembers.value = true
    const res = await listGuildMembers(guildId.value)
    members.value = res.members || []
  } catch (e: any) {
    console.error('加载成员列表失败:', e)
    toast.error(e.message || '加载失败')
  } finally {
    loadingMembers.value = false
  }
}

// 加载申请列表
async function loadApplications() {
  try {
    loadingApplications.value = true
    const status = applicationFilter.value === 'pending' ? 'pending' : undefined
    const res = await listGuildApplications(guildId.value, status)
    applications.value = res.applications || []
  } catch (e: any) {
    console.error('加载申请列表失败:', e)
    toast.error(e.message || '加载失败')
  } finally {
    loadingApplications.value = false
  }
}

// 设置成员角色
async function handleSetRole(member: GuildMember, newRole: 'admin' | 'member') {
  if (member.role === 'owner') {
    toast.warning('无法修改会长的角色')
    return
  }

  const roleLabel = newRole === 'admin' ? '管理员' : '普通成员'
  const confirmed = await confirm({
    title: '确认修改角色',
    message: `确定要将 ${member.username} 设置为${roleLabel}吗？`
  })

  if (!confirmed) return

  try {
    await updateMemberRole(guildId.value, member.user_id, newRole)
    toast.success('角色修改成功')
    loadMembers()
  } catch (e: any) {
    console.error('修改角色失败:', e)
    toast.error(e.message || '修改失败')
  }
}

// 移除成员
async function handleRemoveMember(member: GuildMember) {
  if (member.role === 'owner') {
    toast.warning('无法移除会长')
    return
  }

  const confirmed = await confirm({
    title: '确认移除成员',
    message: `确定要将 ${member.username} 移出公会吗？此操作不可撤销。`
  })

  if (!confirmed) return

  try {
    await removeMember(guildId.value, member.user_id)
    toast.success('成员已移除')
    loadMembers()
  } catch (e: any) {
    console.error('移除成员失败:', e)
    toast.error(e.message || '移除失败')
  }
}

// 审批申请
async function handleReviewApplication(app: GuildApplication, action: 'approve' | 'reject') {
  const actionLabel = action === 'approve' ? '通过' : '拒绝'
  const confirmed = await confirm({
    title: `确认${actionLabel}申请`,
    message: `确定要${actionLabel} ${app.username} 的加入申请吗？`
  })

  if (!confirmed) return

  try {
    await reviewGuildApplication(guildId.value, app.id, action)
    toast.success(`已${actionLabel}申请`)
    loadApplications()
    if (action === 'approve') {
      loadMembers() // 通过后刷新成员列表
    }
  } catch (e: any) {
    console.error('审批失败:', e)
    toast.error(e.message || '审批失败')
  }
}

// 获取角色标签
function getRoleLabel(role: string): string {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role] || ''
}

// 获取申请状态标签
function getStatusLabel(status: string): string {
  const map: Record<string, string> = { pending: '待审核', approved: '已通过', rejected: '已拒绝' }
  return map[status] || ''
}

// 格式化日期
function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 生命周期
onMounted(async () => {
  await loadGuild()
  if (isAdmin.value) {
    await Promise.all([loadMembers(), loadApplications()])
  }
})
</script>

<template>
  <div class="guild-manage">
    <!-- 加载中 -->
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="guild" class="manage-container">
      <!-- 页面头部 -->
      <header class="page-header">
        <div class="header-left">
          <button class="back-btn" @click="router.push(`/guild/${guildId}`)">
            <i class="ri-arrow-left-line"></i>
            返回
          </button>
          <h1 class="page-title">{{ guild.name }} - 管理中心</h1>
        </div>
      </header>

      <div class="avatar-panel">
        <div class="avatar-preview" :class="{ editable: isAdmin }" @click="isAdmin && triggerAvatarUpload()">
          <img v-if="guildAvatarUrl" :src="guildAvatarUrl" alt="" />
          <span v-else>{{ guild.name?.charAt(0) || 'G' }}</span>
          <div v-if="isAdmin" class="avatar-overlay">
            <i class="ri-camera-line"></i>
          </div>
        </div>
        <div class="avatar-meta">
          <div class="avatar-title">公会头像</div>
          <div class="avatar-actions">
            <RButton size="small" :loading="avatarUploading" @click="triggerAvatarUpload">上传头像</RButton>
          </div>
        </div>
        <input ref="avatarInputRef" type="file" accept="image/*" hidden @change="handleAvatarUpload" />
      </div>

      <!-- 标签页导航 -->
      <div class="tabs">
        <button
          :class="{ active: activeTab === 'members' }"
          @click="activeTab = 'members'"
        >
          <i class="ri-team-line"></i>
          成员管理
        </button>
        <button
          :class="{ active: activeTab === 'applications' }"
          @click="activeTab = 'applications'"
        >
          <i class="ri-user-add-line"></i>
          申请审批
          <span v-if="applications.filter(a => a.status === 'pending').length > 0" class="badge">
            {{ applications.filter(a => a.status === 'pending').length }}
          </span>
        </button>
      </div>

      <!-- 成员管理标签页 -->
      <div v-show="activeTab === 'members'" class="tab-content">
        <div v-if="loadingMembers" class="loading">加载中...</div>
        <REmpty v-else-if="members.length === 0" description="暂无成员" />
        <div v-else class="members-list">
          <div v-for="member in members" :key="member.id" class="member-item">
            <div class="member-info">
              <div class="member-avatar">
                {{ member.username?.charAt(0)?.toUpperCase() || 'U' }}
              </div>
              <div class="member-details">
                <h3 :style="buildNameStyle(member.name_color, member.name_bold)">{{ member.username }}</h3>
                <p class="join-date">加入时间: {{ formatDate(member.joined_at) }}</p>
              </div>
            </div>
            <div class="member-actions">
              <span class="role-badge" :class="member.role">
                {{ getRoleLabel(member.role) }}
              </span>
              <template v-if="member.role !== 'owner' && isOwner">
                <RButton
                  v-if="member.role === 'member'"
                  size="small"
                  @click="handleSetRole(member, 'admin')"
                >
                  设为管理员
                </RButton>
                <RButton
                  v-else-if="member.role === 'admin'"
                  size="small"
                  @click="handleSetRole(member, 'member')"
                >
                  取消管理员
                </RButton>
                <RButton
                  size="small"
                  type="danger"
                  @click="handleRemoveMember(member)"
                >
                  移除
                </RButton>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- 申请审批标签页 -->
      <div v-show="activeTab === 'applications'" class="tab-content">
        <!-- 筛选器 -->
        <div class="filter-bar">
          <button
            :class="{ active: applicationFilter === 'pending' }"
            @click="applicationFilter = 'pending'; loadApplications()"
          >
            待审核
          </button>
          <button
            :class="{ active: applicationFilter === 'all' }"
            @click="applicationFilter = 'all'; loadApplications()"
          >
            全部
          </button>
        </div>

        <div v-if="loadingApplications" class="loading">加载中...</div>
        <REmpty v-else-if="applications.length === 0" description="暂无申请" />
        <div v-else class="applications-list">
          <div v-for="app in applications" :key="app.id" class="application-item">
            <div class="app-info">
              <div class="app-avatar">
                {{ app.username?.charAt(0)?.toUpperCase() || 'U' }}
              </div>
              <div class="app-details">
                <h3 :style="buildNameStyle(app.name_color, app.name_bold)">{{ app.username }}</h3>
                <p v-if="app.message" class="app-message">{{ app.message }}</p>
                <p class="app-date">申请时间: {{ formatDate(app.created_at) }}</p>
              </div>
            </div>
            <div class="app-actions">
              <span class="status-badge" :class="app.status">
                {{ getStatusLabel(app.status) }}
              </span>
              <template v-if="app.status === 'pending'">
                <RButton
                  size="small"
                  type="primary"
                  @click="handleReviewApplication(app, 'approve')"
                >
                  <i class="ri-check-line"></i>
                  通过
                </RButton>
                <RButton
                  size="small"
                  type="danger"
                  @click="handleReviewApplication(app, 'reject')"
                >
                  <i class="ri-close-line"></i>
                  拒绝
                </RButton>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.guild-manage {
  padding: 24px;
  min-height: 100vh;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #856a52;
}

.manage-container {
  max-width: 1200px;
  margin: 0 auto;
}

/* 页面头部 */
.page-header {
  margin-bottom: 32px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #E8DCC8;
  border-radius: 8px;
  color: #B87333;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.back-btn:hover {
  background: #FBF5EF;
  border-color: #D4A373;
}

.back-btn i {
  font-size: 18px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: #4B3621;
  margin: 0;
}

.avatar-panel {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  margin-bottom: 24px;
}

.avatar-preview {
  width: 72px;
  height: 72px;
  border-radius: 16px;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  overflow: hidden;
  position: relative;
  border: 2px solid #fff;
  box-shadow: 0 4px 12px rgba(44, 24, 16, 0.2);
}

.avatar-preview.editable {
  cursor: pointer;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  color: #fff;
  font-size: 18px;
}

.avatar-preview.editable:hover .avatar-overlay {
  opacity: 1;
}

.avatar-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.avatar-title {
  font-size: 14px;
  font-weight: 600;
  color: #4B3621;
}

.avatar-actions {
  display: flex;
  gap: 8px;
}

/* 标签页导航 */
.tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  background: #FBF5EF;
  padding: 6px;
  border-radius: 12px;
}

.tabs button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 24px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: #856a52;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.tabs button:hover {
  background: rgba(212, 163, 115, 0.1);
}

.tabs button.active {
  background: #fff;
  color: #4B3621;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.tabs button i {
  font-size: 18px;
}

.tabs .badge {
  position: absolute;
  top: 6px;
  right: 6px;
  background: #FF9800;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}

/* 标签页内容 */
.tab-content {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.06);
}

/* 筛选栏 */
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  padding: 4px;
  background: #FBF5EF;
  border-radius: 8px;
}

.filter-bar button {
  flex: 1;
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #856a52;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-bar button:hover {
  background: rgba(212, 163, 115, 0.1);
}

.filter-bar button.active {
  background: #fff;
  color: #4B3621;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
}

/* 成员列表 */
.members-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.member-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #FBF5EF;
  border-radius: 10px;
  border: 1px solid #E8DCC8;
  transition: all 0.2s;
}

.member-item:hover {
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.member-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.member-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.member-details h3 {
  font-size: 15px;
  font-weight: 600;
  color: #4B3621;
  margin: 0 0 4px 0;
}

.join-date {
  font-size: 12px;
  color: #856a52;
  margin: 0;
}

.member-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 申请列表 */
.applications-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.application-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #FBF5EF;
  border-radius: 10px;
  border: 1px solid #E8DCC8;
  transition: all 0.2s;
}

.application-item:hover {
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.app-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex: 1;
}

.app-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.app-details {
  flex: 1;
  min-width: 0;
}

.app-details h3 {
  font-size: 15px;
  font-weight: 600;
  color: #4B3621;
  margin: 0 0 6px 0;
}

.app-message {
  font-size: 13px;
  color: #4B3621;
  margin: 0 0 6px 0;
  line-height: 1.5;
}

.app-date {
  font-size: 12px;
  color: #856a52;
  margin: 0;
}

.app-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 角色徽章 */
.role-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.role-badge.owner {
  background: linear-gradient(135deg, #B87333, #D4A373);
  color: #fff;
}

.role-badge.admin {
  background: rgba(184, 115, 51, 0.15);
  color: #B87333;
  border: 1px solid rgba(184, 115, 51, 0.3);
}

.role-badge.member {
  background: rgba(140, 123, 112, 0.1);
  color: #856a52;
  border: 1px solid rgba(140, 123, 112, 0.2);
}

/* 状态徽章 */
.status-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.status-badge.pending {
  background: rgba(255, 152, 0, 0.1);
  color: #FF9800;
  border: 1px solid rgba(255, 152, 0, 0.3);
}

.status-badge.approved {
  background: rgba(76, 175, 80, 0.1);
  color: #4CAF50;
  border: 1px solid rgba(76, 175, 80, 0.3);
}

.status-badge.rejected {
  background: rgba(244, 67, 54, 0.1);
  color: #F44336;
  border: 1px solid rgba(244, 67, 54, 0.3);
}
</style>
