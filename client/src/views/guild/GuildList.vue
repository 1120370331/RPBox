<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listGuilds, joinGuild, listPublicGuilds, applyGuild, listMyApplications, cancelApplication, type Guild, type GuildApplication } from '@/api/guild'
import { getImageUrl } from '@/api/item'
import { useToastStore } from '@/stores/toast'
import RButton from '@/components/RButton.vue'
import REmpty from '@/components/REmpty.vue'
import RModal from '@/components/RModal.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
const toast = useToastStore()
const loading = ref(false)
const activeTab = ref<'my' | 'public'>('public')
const myGuilds = ref<Guild[]>([])
const publicGuilds = ref<Guild[]>([])
const showJoinModal = ref(false)
const inviteCode = ref('')
const joining = ref(false)

// 申请相关
const myApplications = ref<GuildApplication[]>([])
const showApplyModal = ref(false)
const applyingGuildId = ref<number | null>(null)
const applyMessage = ref('')
const applying = ref(false)

// 筛选条件
const keyword = ref('')
const faction = ref('')

const displayGuilds = computed(() => {
  return activeTab.value === 'my' ? myGuilds.value : publicGuilds.value
})

async function loadMyGuilds() {
  try {
    const res = await listGuilds()
    myGuilds.value = res.guilds || []
  } catch (e) {
    console.error('加载我的公会失败:', e)
  }
}

async function loadPublicGuilds() {
  try {
    const res = await listPublicGuilds({
      keyword: keyword.value || undefined,
      faction: faction.value || undefined
    })
    publicGuilds.value = res.guilds || []
  } catch (e) {
    console.error('加载公会广场失败:', e)
  }
}

async function loadMyApplications() {
  try {
    const res = await listMyApplications()
    myApplications.value = res.applications || []
  } catch (e) {
    console.error('加载申请记录失败:', e)
  }
}

// 检查公会的申请状态
function getApplicationStatus(guildId: number): 'none' | 'pending' {
  const app = myApplications.value.find(a => a.guild_id === guildId && a.status === 'pending')
  return app ? 'pending' : 'none'
}

// 检查是否已加入公会
function isGuildMember(guildId: number): boolean {
  return myGuilds.value.some(g => g.id === guildId)
}

async function loadData() {
  loading.value = true
  try {
    await Promise.all([loadMyGuilds(), loadPublicGuilds(), loadMyApplications()])
  } finally {
    loading.value = false
  }
}

async function handleJoin() {
  if (!inviteCode.value.trim()) return
  joining.value = true
  try {
    await joinGuild(inviteCode.value)
    showJoinModal.value = false
    inviteCode.value = ''
    toast.success('加入成功')
    loadData()
  } catch (e: any) {
    console.error('加入失败:', e)
    toast.error(e.message || '加入失败')
  } finally {
    joining.value = false
  }
}

// 打开申请弹窗
function openApplyModal(guildId: number, event: Event) {
  event.stopPropagation()
  applyingGuildId.value = guildId
  applyMessage.value = ''
  showApplyModal.value = true
}

// 提交申请
async function handleApply() {
  if (!applyingGuildId.value) return
  applying.value = true
  try {
    await applyGuild(applyingGuildId.value, applyMessage.value)
    showApplyModal.value = false
    applyMessage.value = ''
    toast.success('申请已提交')
    loadMyApplications()
  } catch (e: any) {
    console.error('申请失败:', e)
    toast.error(e.message || '申请失败')
  } finally {
    applying.value = false
  }
}

// 获取申请对象
function getApplication(guildId: number): GuildApplication | undefined {
  return myApplications.value.find(a => a.guild_id === guildId)
}

// 取消申请
async function handleCancelApplication(guildId: number, event: Event) {
  event.stopPropagation()
  const app = getApplication(guildId)
  if (!app) return

  try {
    await cancelApplication(guildId, app.id)
    toast.success('已取消申请')
    loadMyApplications()
  } catch (e: any) {
    console.error('取消申请失败:', e)
    toast.error(e.message || '取消失败')
  }
}

function handleSearch() {
  loadPublicGuilds()
}

function getRoleLabel(role?: string): string {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role || ''] || ''
}

function getFactionLabel(f: string): string {
  const map: Record<string, string> = { alliance: '联盟', horde: '部落', neutral: '中立' }
  return map[f] || ''
}

function getFactionClass(f: string): string {
  return f || 'neutral'
}

onMounted(loadData)
</script>

<template>
  <div class="guild-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="tabs">
        <button :class="{ active: activeTab === 'public' }" @click="activeTab = 'public'">公会广场</button>
        <button :class="{ active: activeTab === 'my' }" @click="activeTab = 'my'">我的公会</button>
      </div>
      <div class="header-actions">
        <RButton @click="showJoinModal = true">加入公会</RButton>
        <RButton type="primary" @click="router.push('/guild/create')">创建公会</RButton>
      </div>
    </div>

    <!-- 筛选栏（仅公会广场显示） -->
    <div v-if="activeTab === 'public'" class="filter-bar">
      <RInput v-model="keyword" placeholder="搜索公会名称..." class="search-input" @keyup.enter="handleSearch" />
      <select v-model="faction" @change="handleSearch">
        <option value="">全部阵营</option>
        <option value="alliance">联盟</option>
        <option value="horde">部落</option>
        <option value="neutral">中立</option>
      </select>
      <RButton @click="handleSearch">搜索</RButton>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading">加载中...</div>

    <!-- 空状态 -->
    <REmpty v-else-if="displayGuilds.length === 0" :description="activeTab === 'my' ? '暂未加入任何公会' : '暂无公会'" />

    <!-- 公会卡片列表 -->
    <div v-else class="guild-grid">
      <div v-for="guild in displayGuilds" :key="guild.id" class="guild-card" @click="router.push(`/guild/${guild.id}`)">
        <!-- 头图 -->
        <div class="card-banner" :style="{ background: guild.banner_url ? `url(${getImageUrl('guild-banner', guild.id, { w: 600, q: 80 })}) center/cover` : `linear-gradient(135deg, #${guild.color || 'B87333'}, #4B3621)` }">
          <div v-if="guild.faction" class="faction-badge" :class="getFactionClass(guild.faction)">
            {{ getFactionLabel(guild.faction) }}
          </div>
        </div>
        <!-- 卡片内容 -->
        <div class="card-body">
          <div class="guild-icon" :style="{ background: '#' + (guild.color || 'B87333') }">
            {{ guild.name.charAt(0) }}
          </div>
          <div class="guild-info">
            <h3>{{ guild.name }}</h3>
            <p class="slogan">{{ guild.slogan || guild.description || '暂无描述' }}</p>
            <div class="guild-meta">
              <span><i class="ri-user-line"></i> {{ guild.member_count }} 成员</span>
              <span><i class="ri-book-line"></i> {{ guild.story_count }} 剧情</span>
              <span v-if="guild.server" class="server"><i class="ri-server-line"></i> {{ guild.server }}</span>
            </div>
            <span v-if="guild.my_role" class="role-badge">{{ getRoleLabel(guild.my_role) }}</span>

            <!-- 申请状态和操作按钮（仅公会广场显示） -->
            <div v-if="activeTab === 'public' && !isGuildMember(guild.id)" class="guild-actions">
              <button
                v-if="getApplicationStatus(guild.id) === 'none'"
                class="apply-btn"
                @click="openApplyModal(guild.id, $event)"
              >
                <i class="ri-user-add-line"></i>
                申请加入
              </button>
              <button
                v-else-if="getApplicationStatus(guild.id) === 'pending'"
                class="cancel-btn"
                @click="handleCancelApplication(guild.id, $event)"
              >
                <i class="ri-close-line"></i>
                取消申请
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 加入公会弹窗 -->
    <RModal v-model="showJoinModal" title="加入公会" width="400px">
      <RInput v-model="inviteCode" placeholder="输入邀请码" />
      <template #footer>
        <RButton @click="showJoinModal = false">取消</RButton>
        <RButton type="primary" :loading="joining" @click="handleJoin">加入</RButton>
      </template>
    </RModal>

    <!-- 申请加入公会弹窗 -->
    <RModal v-model="showApplyModal" title="申请加入公会" width="500px">
      <div class="apply-form">
        <p class="apply-hint">请简单介绍一下自己，说明为什么想加入这个公会</p>
        <textarea
          v-model="applyMessage"
          placeholder="申请留言（可选）"
          rows="4"
          maxlength="500"
          class="apply-textarea"
        ></textarea>
      </div>
      <template #footer>
        <RButton @click="showApplyModal = false">取消</RButton>
        <RButton type="primary" :loading="applying" @click="handleApply">提交申请</RButton>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.guild-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.tabs {
  display: flex;
  gap: 4px;
  background: #f0e6dc;
  padding: 4px;
  border-radius: 10px;
}

.tabs button {
  padding: 8px 20px;
  border: none;
  background: transparent;
  color: #856a52;
  font-size: 14px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.tabs button.active {
  background: #fff;
  color: #4B3621;
  font-weight: 600;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.header-actions {
  display: flex;
  gap: 12px;
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  align-items: center;
}

.filter-bar .search-input {
  flex: 1;
  max-width: 300px;
}

.filter-bar select {
  padding: 8px 12px;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  background: #fff;
  color: #4B3621;
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #856a52;
}

.guild-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.guild-card {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.guild-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.12);
}

.card-banner {
  height: 120px;
  position: relative;
}

.faction-badge {
  position: absolute;
  top: 12px;
  right: 12px;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
}

.faction-badge.alliance {
  background: linear-gradient(135deg, #1e5aa8, #3b82f6);
}

.faction-badge.horde {
  background: linear-gradient(135deg, #991b1b, #dc2626);
}

.faction-badge.neutral {
  background: linear-gradient(135deg, #6b7280, #9ca3af);
}

.card-body {
  padding: 24px;
  display: flex;
  gap: 12px;
}

.guild-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
  margin-top: -32px;
  position: relative;
  z-index: 10;
  border: 3px solid #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.guild-info {
  flex: 1;
  min-width: 0;
}

.guild-info h3 {
  font-size: 16px;
  color: #4B3621;
  margin: 0 0 4px 0;
  font-weight: 600;
}

.guild-info .slogan {
  font-size: 13px;
  color: #856a52;
  margin: 0 0 8px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.guild-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  color: #856a52;
}

.guild-meta i {
  margin-right: 4px;
}

.role-badge {
  display: inline-block;
  background: rgba(184, 115, 51, 0.15);
  color: #B87333;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  margin-top: 8px;
}

/* 申请弹窗样式 */
.apply-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.apply-hint {
  font-size: 13px;
  color: #856a52;
  margin: 0;
  line-height: 1.5;
}

.apply-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  font-size: 14px;
  color: #4B3621;
  font-family: inherit;
  resize: vertical;
  transition: border-color 0.2s;
}

.apply-textarea:focus {
  outline: none;
  border-color: #B87333;
}

/* 公会卡片操作区域 */
.guild-actions {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0e6dc;
}

.apply-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 8px 16px;
  background: linear-gradient(135deg, #B87333, #D4A373);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.apply-btn:hover {
  background: linear-gradient(135deg, #4B3621, #856a52);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.3);
}

.apply-btn i {
  font-size: 16px;
}

.cancel-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 8px 16px;
  background: rgba(255, 152, 0, 0.1);
  color: #FF9800;
  border: 1px solid rgba(255, 152, 0, 0.3);
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.cancel-btn:hover {
  background: rgba(255, 152, 0, 0.2);
  border-color: #FF9800;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(255, 152, 0, 0.3);
}

.cancel-btn i {
  font-size: 16px;
}

.pending-status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 16px;
  background: rgba(255, 152, 0, 0.1);
  color: #FF9800;
  border: 1px solid rgba(255, 152, 0, 0.3);
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
}

.pending-status i {
  font-size: 16px;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
