<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { dialog } from '../../composables/useDialog'
import * as accountBackupApi from '../../api/accountBackup'

interface ProfileItem {
  id: string
  name: string
  icon?: string
  checksum: string
  raw_lua: string
  account_id: string
  saved_variables_path: string
  modified_at?: string
}

interface AccountInfo {
  account_id: string
  profiles: ProfileItem[]
}

type WorkflowStep = 'scan' | 'backup' | 'upload' | 'verify' | 'finish'

const router = useRouter()
const accounts = ref<AccountInfo[]>([])
const selectedAccount = ref('')
const isLoading = ref(false)
const isSyncing = ref(false)
const cloudBackups = ref<Map<string, accountBackupApi.AccountBackup>>(new Map())
const mounted = ref(false)
const wowPath = ref('')
const search = ref('')
const isAuthed = ref(!!localStorage.getItem('token'))
const authMessage = ref('')
const viewMode = ref<'upload' | 'restore' | 'cloud'>('upload')
const showConfirmModal = ref(false)
const isRestoring = ref(false)
const fullBackupData = ref<accountBackupApi.AccountBackup | null>(null)
const isLoadingCloudData = ref(false)

const currentProfiles = computed(() => {
  const acc = accounts.value.find(a => a.account_id === selectedAccount.value)
  return acc?.profiles || []
})

const hasCloudData = computed(() => cloudBackups.value.size > 0)

// 当前账号的同步状态
const currentBackup = computed(() => cloudBackups.value.get(selectedAccount.value))
const accountSyncStatus = computed<'synced' | 'pending' | 'conflict'>(() => {
  if (!selectedAccount.value) return 'pending'
  const backup = currentBackup.value
  if (!backup) return 'pending'
  // 计算本地所有profiles的checksum
  const localChecksum = computeLocalChecksum()
  console.log('[Sync] checksum比较:', {
    account: selectedAccount.value,
    local: localChecksum,
    cloud: backup.checksum,
    match: backup.checksum === localChecksum
  })
  if (backup.checksum === localChecksum) return 'synced'
  return 'conflict'
})

// 计算本地profiles的整体checksum
function computeLocalChecksum(): string {
  const profiles = currentProfiles.value
  if (profiles.length === 0) return ''
  // 简单拼接所有checksum
  return profiles.map(p => p.checksum).sort().join('')
}

const stats = computed(() => {
  const total = currentProfiles.value.length
  const status = accountSyncStatus.value
  return {
    synced: status === 'synced' ? total : 0,
    pending: status === 'pending' ? total : 0,
    conflict: status === 'conflict' ? total : 0,
    total
  }
})

const filteredProfiles = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return currentProfiles.value
  return currentProfiles.value.filter(p =>
    p.name.toLowerCase().includes(keyword) ||
    p.account_id.toLowerCase().includes(keyword) ||
    p.saved_variables_path.toLowerCase().includes(keyword)
  )
})

const overallProgress = computed(() => {
  if (!stats.value.total) return 0
  return Math.round((stats.value.synced / stats.value.total) * 100)
})

const workflowStep = computed<WorkflowStep>(() => {
  if (isSyncing.value) return 'upload'
  if (stats.value.conflict > 0) return 'verify'
  if (stats.value.pending > 0) return 'backup'
  if (stats.value.total > 0) return 'finish'
  return 'scan'
})

const toSyncList = computed(() =>
  currentProfiles.value.filter(p => getStatus(p.id) !== 'synced')
)

// 云端账号备份列表
const cloudBackupsList = computed(() => Array.from(cloudBackups.value.values()))

// 当前账号的云端人物卡列表（从 profiles_data 解析）
interface CloudProfileItem {
  id: string
  name: string
  data: any
}
const cloudProfilesList = computed<CloudProfileItem[]>(() => {
  const backup = fullBackupData.value
  if (!backup?.profiles_data) return []
  try {
    const data = JSON.parse(backup.profiles_data)
    return Object.entries(data).map(([id, profileData]: [string, any]) => ({
      id,
      name: profileData?.profileName || profileData?.player?.characteristics?.FN || id.slice(0, 8),
      data: profileData
    }))
  } catch {
    return []
  }
})

// 删除确认弹窗
const showDeleteModal = ref(false)
const pendingDeleteAccount = ref<string | null>(null)
const isDeleting = ref(false)

function isDoneStep(stepKey: WorkflowStep) {
  if (workflowStep.value === 'finish') return true
  const currentIndex = workflowSteps.findIndex(s => s.key === workflowStep.value)
  const targetIndex = workflowSteps.findIndex(s => s.key === stepKey)
  return targetIndex < currentIndex
}

// 加载完整的云端备份数据
async function loadFullBackup() {
  if (!selectedAccount.value || !currentBackup.value) {
    fullBackupData.value = null
    return
  }
  isLoadingCloudData.value = true
  try {
    fullBackupData.value = await accountBackupApi.getAccountBackup(selectedAccount.value)
  } catch {
    fullBackupData.value = null
  } finally {
    isLoadingCloudData.value = false
  }
}

// 切换到云端视图或账号变化时加载完整数据
watch([viewMode, selectedAccount], ([mode]) => {
  if (mode === 'cloud') {
    loadFullBackup()
  }
}, { immediate: false })

onMounted(async () => {
  const savedPath = localStorage.getItem('wow_path')
  if (!savedPath) {
    router.push('/sync/setup')
    return
  }
  if (!isAuthed.value) {
    authMessage.value = '请先登录以继续备份人物卡'
    router.push('/login?redirect=/sync')
    return
  }
  wowPath.value = savedPath
  await loadProfiles()
  setTimeout(() => mounted.value = true, 50)
})

async function loadProfiles() {
  if (!isAuthed.value) return
  isLoading.value = true
  try {
    console.log('[Sync] 开始加载...')
    const [localResult, backupList] = await Promise.all([
      invoke<{ accounts: AccountInfo[] }>('scan_profiles', { wowPath: localStorage.getItem('wow_path') || '' }),
      accountBackupApi.listAccountBackups().catch(() => [])
    ])
    console.log('[Sync] 本地扫描结果:', localResult.accounts.map(a => ({
      account: a.account_id,
      count: a.profiles.length
    })))
    console.log('[Sync] 云端备份:', backupList.map(b => ({
      account_id: b.account_id,
      count: b.profiles_count,
      checksum: b.checksum
    })))

    accounts.value = localResult.accounts
    const stillExists = localResult.accounts.find(a => a.account_id === selectedAccount.value)
    if (!stillExists && localResult.accounts.length > 0) {
      selectedAccount.value = localResult.accounts[0].account_id
    }
    cloudBackups.value.clear()
    backupList.forEach(b => cloudBackups.value.set(b.account_id, b))
  } finally {
    isLoading.value = false
  }
}

// getStatus 改为返回账号级别的状态
function getStatus(_id: string): 'synced' | 'pending' | 'conflict' {
  return accountSyncStatus.value
}

async function openConfirmModal() {
  if (!isAuthed.value) {
    await dialog.alert({ title: '提示', message: '请先登录再执行备份', type: 'warning' })
    router.push('/login?redirect=/sync')
    return
  }
  if (!selectedAccount.value || currentProfiles.value.length === 0) return
  showConfirmModal.value = true
}

function formatTime(time?: string) {
  if (!time) return '未知'
  const d = new Date(time)
  if (Number.isNaN(d.getTime())) return time
  return d.toLocaleString()
}

async function confirmUpload() {
  if (!selectedAccount.value || currentProfiles.value.length === 0) {
    showConfirmModal.value = false
    return
  }

  // 构建profiles_data JSON
  const profilesData: Record<string, any> = {}
  for (const p of currentProfiles.value) {
    try {
      profilesData[p.id] = JSON.parse(p.raw_lua)
    } catch {
      await dialog.alert({
        title: '数据损坏',
        message: `人物卡「${p.name}」数据损坏，无法上传`,
        type: 'error'
      })
      return
    }
  }

  isSyncing.value = true
  try {
    await accountBackupApi.upsertAccountBackup({
      account_id: selectedAccount.value,
      profiles_data: JSON.stringify(profilesData),
      profiles_count: currentProfiles.value.length,
      checksum: computeLocalChecksum()
    })
    await loadProfiles()
    await dialog.alert({ title: '成功', message: '账号备份完成', type: 'success' })
  } catch (e: any) {
    await dialog.alert({ title: '错误', message: `备份失败：${e?.message || e}`, type: 'error' })
  } finally {
    isSyncing.value = false
    showConfirmModal.value = false
  }
}

function goToDetail(id: string) {
  if (!isAuthed.value) {
    router.push('/login?redirect=/sync')
    return
  }
  router.push(`/sync/profile/${id}`)
}

function openSettings() {
  router.push('/settings')
}

// 打开删除确认弹窗
function openDeleteModal(accountId: string) {
  pendingDeleteAccount.value = accountId
  showDeleteModal.value = true
}

// 确认删除云端备份（账号级别）
async function confirmDelete() {
  if (!pendingDeleteAccount.value) return
  isDeleting.value = true
  try {
    await accountBackupApi.deleteAccountBackup(pendingDeleteAccount.value)
    cloudBackups.value.delete(pendingDeleteAccount.value)
    await dialog.alert({ title: '成功', message: '云端备份已删除', type: 'success' })
  } catch (e: any) {
    await dialog.alert({ title: '错误', message: `删除失败：${e?.message || e}`, type: 'error' })
  } finally {
    isDeleting.value = false
    showDeleteModal.value = false
    pendingDeleteAccount.value = null
  }
}

async function restoreAll() {
  if (!isAuthed.value) {
    await dialog.alert({ title: '提示', message: '请先登录再执行写回', type: 'warning' })
    router.push('/login?redirect=/sync')
    return
  }
  const backup = currentBackup.value
  if (!backup) {
    await dialog.alert({ title: '提示', message: '当前账号在云端暂无备份', type: 'info' })
    return
  }
  const ok = await dialog.confirm({
    title: '确认写回',
    message: `将从云端写回账号 ${selectedAccount.value} 的 ${backup.profiles_count} 个人物卡到本地，需保证游戏已关闭。是否继续？`,
    type: 'warning'
  })
  if (!ok) return

  isRestoring.value = true
  try {
    // 获取完整备份数据
    const fullBackup = await accountBackupApi.getAccountBackup(selectedAccount.value)
    if (!fullBackup.profiles_data) {
      await dialog.alert({ title: '错误', message: '云端备份数据为空', type: 'error' })
      return
    }
    // 调用 Tauri 命令写回整个账号
    await invoke('apply_account_backup', {
      wowPath: wowPath.value,
      accountId: selectedAccount.value,
      profilesJson: fullBackup.profiles_data
    })
    await loadProfiles()
    await dialog.alert({ title: '成功', message: '写回完成，重启游戏后生效', type: 'success' })
  } catch (e: any) {
    await dialog.alert({ title: '错误', message: `写回失败：${e?.message || e}`, type: 'error' })
  } finally {
    isRestoring.value = false
  }
}

const workflowSteps = [
  { key: 'scan', label: '选择子账号', desc: '选择WOW子账号', icon: 'ri-search-line' },
  { key: 'backup', label: '自动备份', desc: '本地数据防护', icon: 'ri-shield-check-line' },
  { key: 'upload', label: '上传云端', desc: '增量同步+进度', icon: 'ri-cloud-upload-line' },
  { key: 'verify', label: '校验/冲突', desc: 'checksum/版本对比', icon: 'ri-loop-left-line' },
  { key: 'finish', label: '完成', desc: '版本归档，可回滚', icon: 'ri-checkbox-circle-line' }
] satisfies { key: WorkflowStep; label: string; desc: string; icon: string }[]
</script>

<template>
  <div class="sync-page" :class="{ 'animate-in': mounted }">
    <div v-if="!isAuthed" class="auth-tip anim-item" style="--delay: 0">
      <i class="ri-information-line"></i>
      <span>{{ authMessage || '请先登录以继续备份人物卡' }}</span>
    </div>
    <div v-else-if="hasCloudData" class="cloud-tip anim-item" style="--delay: 0">
      <i class="ri-cloud-line"></i>
      <span>云端已有人物卡备份，上传时可选择覆盖或先查看详情；发生冲突时会提示确认。</span>
    </div>
    <!-- 顶部栏 -->
    <header class="topbar anim-item" style="--delay: 0">
      <div class="top-left">
        <div class="breadcrumbs">
          <i class="ri-home-4-line"></i>
          <span class="separator">/</span>
          <span>人物卡</span>
          <span class="separator">/</span>
          <span class="current">备份同步</span>
        </div>
        <div class="mode-tabs">
          <button
            class="tab-btn"
            :class="{ active: viewMode === 'upload' }"
            @click="viewMode = 'upload'"
          >
            <i class="ri-cloud-upload-line"></i> 云端备份
          </button>
          <button
            class="tab-btn"
            :class="{ active: viewMode === 'restore' }"
            @click="viewMode = 'restore'"
          >
            <i class="ri-download-2-line"></i> 写回本地
          </button>
          <button
            class="tab-btn"
            :class="{ active: viewMode === 'cloud' }"
            @click="viewMode = 'cloud'"
          >
            <i class="ri-cloud-line"></i> 查看云端
          </button>
        </div>
      </div>
      <div class="toolbar-actions">
        <div class="path-info">
          <span class="label">WoW 路径</span>
          <span class="value">{{ wowPath || '未配置' }}</span>
        </div>
        <div class="account-info">
          <span class="label">选择WOW子账号</span>
          <select v-model="selectedAccount" class="account-select">
            <option v-for="acc in accounts" :key="acc.account_id" :value="acc.account_id">
              {{ acc.account_id }}
            </option>
          </select>
        </div>
        <div class="refresh-info">
          <span class="label">刷新</span>
          <button class="btn-icon" @click="loadProfiles" :disabled="isLoading" title="刷新">
          <i class="ri-refresh-line"></i>
        </button>
        </div>
        <button
          class="btn-primary"
          @click="viewMode === 'upload' ? openConfirmModal() : restoreAll()"
          :disabled="viewMode === 'upload' ? (isSyncing || toSyncList.length === 0) : (isRestoring || !hasCloudData)"
        >
          <i v-if="isSyncing || isRestoring" class="ri-loader-4-line spin"></i>
          <i v-else :class="viewMode === 'upload' ? 'ri-save-3-line' : 'ri-download-2-line'"></i>
          {{ viewMode === 'upload' ? (isSyncing ? '同步中...' : '一键备份') : (isRestoring ? '写回中...' : '写回本地') }}
        </button>
      </div>
    </header>

    <!-- 总览卡片 -->
    <div class="overview-grid anim-item" style="--delay: 1">
      <div class="overview-card">
        <div class="title">账号 {{ selectedAccount || '未选择' }} 备份进度</div>
        <div class="progress">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: `${overallProgress}%` }"></div>
          </div>
          <span class="progress-text">{{ overallProgress }}%</span>
        </div>
        <div class="summary-row">
          <div class="pill">自动备份</div>
          <div class="pill">增量同步</div>
          <div class="pill" :class="{ danger: stats.conflict > 0 }">
            冲突 {{ stats.conflict }}
          </div>
        </div>
      </div>
      <div class="stat-card synced">
        <div class="stat-value">{{ stats.synced }}</div>
        <div class="stat-label">已同步</div>
      </div>
      <div class="stat-card pending">
        <div class="stat-value">{{ stats.pending }}</div>
        <div class="stat-label">待备份</div>
      </div>
      <div class="stat-card conflict">
        <div class="stat-value">{{ stats.conflict }}</div>
        <div class="stat-label">冲突待处理</div>
      </div>
  </div>

  <!-- 上传确认弹窗 -->
  <div v-if="showConfirmModal" class="modal-overlay">
    <div class="modal">
      <div class="modal-header">
        <h3>确认备份到云端</h3>
        <span class="tag" v-if="stats.conflict > 0">发现冲突</span>
      </div>
      <p class="muted">
        即将上传账号「{{ selectedAccount }}」的 {{ currentProfiles.length }} 个人物卡到云端。
        云端已有数据时将覆盖为本地版本。
      </p>
      <div class="confirm-info">
        <div class="info-row">
          <span class="label">账号</span>
          <span class="value">{{ selectedAccount }}</span>
        </div>
        <div class="info-row">
          <span class="label">人物卡数量</span>
          <span class="value">{{ currentProfiles.length }} 个</span>
        </div>
        <div class="info-row">
          <span class="label">同步状态</span>
          <span class="value status" :class="accountSyncStatus">
            {{ accountSyncStatus === 'synced' ? '已同步' : accountSyncStatus === 'pending' ? '待备份' : '有变更' }}
          </span>
        </div>
        <div class="info-row" v-if="currentBackup">
          <span class="label">云端版本</span>
          <span class="value">v{{ currentBackup.version }} · {{ formatTime(currentBackup.updated_at) }}</span>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary ghost" @click="showConfirmModal = false">取消</button>
        <button class="btn-primary" @click="confirmUpload" :disabled="isSyncing">
          <i class="ri-save-3-line"></i> {{ isSyncing ? '上传中...' : '确认备份' }}
        </button>
      </div>
    </div>
  </div>

  <!-- 删除确认弹窗 -->
  <div v-if="showDeleteModal" class="modal-overlay">
    <div class="modal delete-modal">
      <div class="modal-header">
        <h3>确认删除云端备份</h3>
      </div>
      <p class="muted">
        即将删除账号「{{ pendingDeleteAccount }}」的云端备份，此操作不可恢复。
      </p>
      <div class="delete-info" v-if="pendingDeleteAccount && cloudBackups.get(pendingDeleteAccount)">
        <div class="info-row">
          <span class="label">账号</span>
          <span class="value">{{ pendingDeleteAccount }}</span>
        </div>
        <div class="info-row">
          <span class="label">人物卡数量</span>
          <span class="value">{{ cloudBackups.get(pendingDeleteAccount)?.profiles_count }} 个</span>
        </div>
        <div class="info-row">
          <span class="label">版本</span>
          <span class="value">v{{ cloudBackups.get(pendingDeleteAccount)?.version }}</span>
        </div>
        <div class="info-row">
          <span class="label">更新时间</span>
          <span class="value">{{ formatTime(cloudBackups.get(pendingDeleteAccount)?.updated_at) }}</span>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary ghost" @click="showDeleteModal = false">取消</button>
        <button class="btn-danger" @click="confirmDelete" :disabled="isDeleting">
          <i v-if="isDeleting" class="ri-loader-4-line spin"></i>
          <i v-else class="ri-delete-bin-line"></i>
          {{ isDeleting ? '删除中...' : '确认删除' }}
        </button>
      </div>
    </div>
  </div>

    <!-- 主工作区 -->
    <div class="workspace">
      <!-- 左侧列表 (仅云端备份模式显示) -->
      <aside v-if="viewMode === 'upload'" class="panel left-panel anim-item" style="--delay: 1.2">
        <div class="panel-header">
          <div class="panel-title">
            <i class="ri-user-star-line"></i> 人物卡列表
          </div>
          <div class="badge">{{ currentProfiles.length }} 个</div>
        </div>

        <div class="panel-body">
          <div class="search-bar">
            <i class="ri-search-line"></i>
            <input v-model="search" type="text" placeholder="搜索角色..." />
          </div>

          <div v-if="isLoading" class="loading-state">
            <div class="loader"></div>
            <p>正在加载人物卡...</p>
          </div>

          <div v-else-if="currentProfiles.length === 0" class="empty-state">
            <div class="empty-icon">👤</div>
            <p>未找到人物卡，检查路径设置</p>
            <button class="btn-secondary small" @click="router.push('/sync/setup')">重新配置</button>
          </div>

          <div v-else class="task-list">
            <div
              v-for="(p, index) in filteredProfiles"
              :key="p.id"
              class="task-card anim-item"
              :class="[getStatus(p.id)]"
              :style="{ '--delay': 1.4 + index * 0.05 }"
              @click="goToDetail(p.id)"
            >
              <div class="avatar">
                <i class="ri-user-3-line"></i>
              </div>
              <div class="info">
                <div class="title-row">
                  <span class="name">{{ p.name }}</span>
                  <span class="path-tag">{{ p.account_id }}</span>
                </div>
                <div class="icon-pill" v-if="p.icon" :title="p.icon">{{ p.icon }}</div>
                <div class="status-line">
                  <span class="status" :class="getStatus(p.id)">
                    <template v-if="getStatus(p.id) === 'synced'">✓ 已同步</template>
                    <template v-else-if="getStatus(p.id) === 'pending'">○ 待备份</template>
                    <template v-else>⚠ 冲突</template>
                  </span>
                  <span class="hint">ID: {{ p.id.slice(0, 6) }}…</span>
                </div>
              </div>
              <div class="arrow">→</div>
            </div>
          </div>
        </div>
      </aside>

      <!-- 分隔线 (仅云端备份模式显示) -->
      <div v-if="viewMode === 'upload'" class="divider-handle anim-item" style="--delay: 1.3">
        <div class="divider-line"></div>
      </div>

      <!-- 右侧详情 -->
      <section class="panel right-panel anim-item" style="--delay: 1.4">
        <div class="panel-header">
          <div class="panel-title">
            <i class="ri-shield-star-line"></i>
            <span v-if="viewMode === 'upload'">备份工作流</span>
            <span v-else>写回本地</span>
          </div>
          <div class="tag" v-if="viewMode === 'upload'">覆盖 PRD: 自动备份 / 冲突检测 / 回滚</div>
          <div class="tag" v-else>PRD: 写回前自动备份 / 关闭游戏后写入</div>
        </div>

        <div class="panel-body right-body" v-if="viewMode === 'upload'">
          <!-- 流程 -->
          <div class="card steps-card">
            <div class="card-header">
              <div>
                <h3>流程进度</h3>
                <div class="muted">选择子账号 → 备份 → 上传 → 校验 → 完成</div>
              </div>
              <div class="step-summary">
                <span class="pill">
                  当前：{{
                    workflowStep === 'upload'
                      ? '上传中'
                      : workflowStep === 'verify'
                        ? '校验/冲突处理'
                        : workflowStep === 'finish'
                          ? '账号已备份完成'
                          : '已选择子账号'
                  }}
                </span>
                <span class="pill ghost" v-if="stats.conflict > 0">冲突待处理</span>
              </div>
            </div>
            <div class="steps-row">
              <div
                v-for="step in workflowSteps"
                :key="step.key"
                class="step-item"
                :class="{
                  done: isDoneStep(step.key),
                  active: workflowStep === step.key,
                  conflict: step.key === 'verify' && stats.conflict > 0
                }"
              >
                <div class="step-icon"><i :class="step.icon"></i></div>
                <div class="step-text">
                  <div class="label">{{ step.label }}</div>
                  <div class="desc">{{ step.desc }}</div>
                </div>
              </div>
            </div>

            <div class="next-actions">
              <div class="muted">下一步指引</div>
              <div class="actions-row">
                <span v-if="workflowStep === 'verify' && stats.conflict > 0">发现冲突，请先在详情页对比后再写回</span>
                <span v-else-if="workflowStep === 'upload'">正在上传，完成后会自动校验</span>
                <span v-else-if="workflowStep === 'backup'">准备备份，确认选中角色后点击一键备份</span>
                <span v-else-if="workflowStep === 'finish'">已完成，可查看版本历史或写回本地</span>
                <span v-else>请先选择WOW子账号</span>
              </div>
            </div>
          </div>

        </div>

        <div class="panel-body right-body" v-else-if="viewMode === 'restore'">
          <div class="card steps-card">
            <div class="card-header">
              <div>
                <h3>写回本地</h3>
                <div class="muted">账号 {{ selectedAccount || '未选择' }} · 关闭游戏后执行</div>
              </div>
            </div>
            <ul class="checklist">
              <li><i class="ri-shut-down-line"></i> 请先关闭魔兽世界</li>
              <li><i class="ri-checkbox-multiple-line"></i> 支持单角色/全量写回</li>
              <li><i class="ri-history-line"></i> 保留最近 10 个版本，可回滚</li>
            </ul>
            <div class="cta-row">
              <button class="btn-primary" :disabled="isRestoring || !hasCloudData" @click="restoreAll">
                <i v-if="isRestoring" class="ri-loader-4-line spin"></i>
                <i v-else class="ri-cloud-download-line"></i>
                {{ isRestoring ? '写回中...' : '从云端写回本地（账号）' }}
              </button>
            </div>
          </div>
        </div>

        <!-- 查看云端备份视图 -->
        <div class="panel-body right-body cloud-view" v-else>
          <div class="cloud-header">
            <div class="cloud-title">
              <i class="ri-cloud-line"></i>
              <span>云端备份管理</span>
            </div>
            <div class="cloud-stats">
              <span class="stat-pill" v-if="currentBackup">
                账号 {{ selectedAccount }} · v{{ currentBackup.version }} · {{ cloudProfilesList.length }} 个人物卡
              </span>
              <span class="stat-pill" v-else>当前账号暂无云端备份</span>
            </div>
          </div>

          <div v-if="!currentBackup" class="empty-state">
            <div class="empty-icon">☁️</div>
            <p>当前账号暂无云端备份</p>
            <button class="btn-secondary small" @click="viewMode = 'upload'">去备份</button>
          </div>

          <div v-else-if="isLoadingCloudData" class="loading-state">
            <div class="loader"></div>
            <p>正在加载云端数据...</p>
          </div>

          <div v-else class="cloud-list">
            <div
              v-for="p in cloudProfilesList"
              :key="p.id"
              class="cloud-card"
              @click="goToDetail(p.id)"
            >
              <div class="cloud-card-main">
                <div class="avatar">
                  <i class="ri-user-3-line"></i>
                </div>
                <div class="info">
                  <div class="title-row">
                    <span class="name">{{ p.name }}</span>
                  </div>
                  <div class="meta-row">
                    <span class="hint">ID: {{ p.id.slice(0, 8) }}…</span>
                  </div>
                </div>
              </div>
              <div class="arrow">→</div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.sync-page {
  padding: 24px;
  min-height: 100vh;
  background: var(--color-background);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.topbar {
  background: #fff;
  border-radius: 16px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-secondary);
}

.top-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.breadcrumbs .current {
  color: var(--color-primary);
  font-weight: 700;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-icon {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  border: 1px solid rgba(75, 54, 33, 0.2);
  background: #fff;
  cursor: pointer;
  font-size: 18px;
}

.path-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: rgba(128, 64, 48, 0.08);
  padding: 8px 10px;
  border-radius: 10px;
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.account-info .label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.refresh-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.refresh-info .label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.path-info .label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.path-info .value {
  font-size: 12px;
  color: var(--color-text-main);
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-select {
  padding: 10px 12px;
  border: 1px solid rgba(75, 54, 33, 0.2);
  border-radius: 10px;
  background: #fff;
  font-size: 14px;
}

.btn-primary,
.btn-secondary {
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.btn-primary {
  background: var(--color-primary);
  color: #fff;
}

.btn-secondary {
  background: rgba(128, 64, 48, 0.1);
  color: var(--color-primary);
}

.btn-primary:disabled,
.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mode-tabs {
  display: flex;
  gap: 8px;
}

.tab-btn {
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid var(--color-border, #E8DCCF);
  background: #fff;
  cursor: pointer;
  color: var(--color-text-main);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tab-btn.active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}

.auth-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff3e0;
  color: #ed6c02;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid rgba(237, 108, 2, 0.2);
}

.cloud-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #e8f4ff;
  color: #0b6daf;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid rgba(11, 109, 175, 0.2);
}

.btn-secondary.small {
  padding: 8px 12px;
  font-size: 13px;
}

.overview-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 12px;
}

.overview-card {
  background: #fff;
  border-radius: 16px;
  padding: 16px;
  border: 1px solid var(--color-border, #E8DCCF);
}

.overview-card .title {
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.progress {
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar {
  flex: 1;
  height: 10px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 8px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--color-accent, #D4A373);
  border-radius: 8px;
  transition: width 0.3s;
}

.progress-text {
  font-weight: 700;
  color: var(--color-primary);
}

.summary-row {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.pill {
  padding: 6px 10px;
  background: rgba(128, 64, 48, 0.08);
  border-radius: 10px;
  font-size: 12px;
  color: var(--color-text-main);
}

.pill.danger {
  background: #ffebee;
  color: #d32f2f;
}

.stat-card {
  background: #fff;
  border-radius: 16px;
  padding: 16px;
  text-align: center;
  border: 1px solid var(--color-border, #E8DCCF);
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
}

.stat-label {
  color: var(--color-text-secondary);
}

.stat-card.synced .stat-value { color: #2e7d32; }
.stat-card.pending .stat-value { color: #ed6c02; }
.stat-card.conflict .stat-value { color: #d32f2f; }

.workspace {
  display: flex;
  gap: 14px;
  min-height: 0;
}

.panel {
  background: #fff;
  border-radius: 16px;
  box-shadow: var(--shadow-md, 0 4px 20px rgba(75, 54, 33, 0.05));
  display: flex;
  flex-direction: column;
}

.left-panel {
  width: 32%;
  min-width: 320px;
  overflow: hidden;
}

.right-panel {
  flex: 1;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-border, #E8DCCF);
}

.panel-title {
  font-weight: 700;
  color: var(--color-text-main);
  display: flex;
  align-items: center;
  gap: 8px;
}

.tag {
  padding: 6px 10px;
  background: rgba(128, 64, 48, 0.08);
  border-radius: 10px;
  font-size: 12px;
  color: var(--color-primary);
}

.badge {
  padding: 6px 10px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 10px;
  font-size: 12px;
  color: var(--color-primary);
}

.panel-body {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.right-body {
  gap: 12px;
}

.search-bar {
  position: relative;
}

.search-bar i {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-secondary);
}

.search-bar input {
  width: 100%;
  padding: 12px 12px 12px 34px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 10px;
  background: #fffcf9;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  max-height: calc(100vh - 280px);
}

.task-card {
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  background: #fffdfb;
}

.task-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
}

.task-card.pending { border-color: rgba(237, 108, 2, 0.4); }
.task-card.conflict { border-color: #d32f2f; background: #fff2f2; }
.task-card.synced { border-color: #2e7d32; background: #f4faf4; }

.avatar {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
}

.info { flex: 1; }

.title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.name {
  font-weight: 700;
  color: var(--color-text-main);
}

.path-tag {
  font-size: 12px;
  background: rgba(128, 64, 48, 0.08);
  color: var(--color-primary);
  padding: 2px 8px;
  border-radius: 8px;
}

.icon-pill {
  display: inline-block;
  max-width: 100%;
  margin-top: 6px;
  padding: 4px 8px;
  border-radius: 8px;
  background: #fffaf5;
  border: 1px solid var(--color-border, #E8DCCF);
  font-size: 11px;
  color: #6f5b4b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.status-line {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}

.status {
  padding: 4px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
}

.status.pending { background: #fff3e0; color: #ed6c02; }
.status.synced { background: #e8f5e9; color: #2e7d32; }
.status.conflict { background: #ffebee; color: #d32f2f; }

.hint {
  font-size: 11px;
  color: #8c7b70;
}

.arrow {
  color: var(--color-text-secondary);
  font-size: 18px;
}

.divider-handle {
  width: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.divider-line {
  width: 4px;
  height: 48px;
  background: rgba(128, 64, 48, 0.2);
  border-radius: 2px;
}

.card {
  background: #fff;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 14px;
  padding: 14px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.card-header h3 {
  margin: 0;
  color: var(--color-text-main);
}

.muted {
  color: var(--color-text-secondary);
  font-size: 13px;
}

.steps-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 10px;
}

.step-item {
  display: flex;
  gap: 10px;
  padding: 12px;
  border-radius: 12px;
  border: 1px dashed var(--color-border, #E8DCCF);
  background: #fffcf9;
}

.step-item.done { border-color: #2e7d32; background: #f4faf4; }
.step-item.active { border-color: var(--color-primary); background: #fff5ed; }
.step-item.conflict { border-color: #d32f2f; background: #fff2f2; }

.step-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: rgba(128, 64, 48, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.step-text .label {
  font-weight: 700;
  color: var(--color-text-main);
}

.step-text .desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.step-summary {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.inline-progress {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar.slim { height: 6px; }

.toggle-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 10px;
}

.toggle-item {
  display: flex;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 12px;
  background: #fff;
  align-items: flex-start;
}

.toggle-item input { margin-top: 4px; }

.toggle-item .title {
  font-weight: 700;
  color: var(--color-text-main);
}

.toggle-item .desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.next-actions {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(128, 64, 48, 0.04);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.actions-row {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.checklist {
  list-style: none;
  padding: 0;
  margin: 0 0 12px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.checklist li {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-main);
  font-size: 13px;
}

.cta-row {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}

.ghost {
  background: rgba(128, 64, 48, 0.06);
  border: 1px solid rgba(128, 64, 48, 0.15);
}

.empty-state {
  text-align: center;
  padding: 40px 0;
  color: var(--color-text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 10px;
}

.loading-state {
  text-align: center;
  padding: 40px 0;
}

.loader {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 4px solid rgba(0, 0, 0, 0.08);
  border-top-color: var(--color-primary);
  animation: spin 1s linear infinite;
  margin: 0 auto 8px;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  width: 720px;
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 12px 40px rgba(0,0,0,0.12);
}

.modal-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.modal-header h3 { margin: 0; color: var(--color-text-main); }

.confirm-list {
  max-height: 320px;
  overflow-y: auto;
  margin: 12px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.confirm-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr auto;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 10px;
  background: #fffdfb;
}

.confirm-row .name { display: flex; flex-direction: column; }
.confirm-row .name .small { color: var(--color-text-secondary); font-size: 12px; }

.confirm-row .time {
  display: flex;
  flex-direction: column;
}

.confirm-row .time label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.confirm-row .time span {
  font-size: 13px;
  color: var(--color-text-main);
}

.confirm-row .status {
  align-self: center;
  padding: 4px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
}

.confirm-row .status.pending { background: #fff3e0; color: #ed6c02; }
.confirm-row .status.synced { background: #e8f5e9; color: #2e7d32; }
.confirm-row .status.conflict { background: #ffebee; color: #d32f2f; }

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.1s);
}

.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 云端备份视图样式 */
.cloud-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.cloud-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cloud-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  font-size: 16px;
  color: var(--color-text-main);
}

.cloud-title i {
  font-size: 20px;
  color: var(--color-primary);
}

.stat-pill {
  padding: 6px 12px;
  background: rgba(128, 64, 48, 0.08);
  border-radius: 10px;
  font-size: 13px;
  color: var(--color-primary);
}

.cloud-search {
  position: relative;
}

.cloud-search i {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-secondary);
}

.cloud-search input {
  width: 100%;
  padding: 12px 12px 12px 38px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 10px;
  background: #fffcf9;
  font-size: 14px;
}

.cloud-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: calc(100vh - 380px);
  overflow-y: auto;
}

.cloud-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 12px;
  background: #fffdfb;
  transition: all 0.2s;
}

.cloud-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(128, 64, 48, 0.1);
}

.cloud-card-main {
  flex: 1;
  display: flex;
  gap: 12px;
  cursor: pointer;
}

.cloud-card .avatar {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, #7eb8da, #5a9bc7);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
}

.cloud-card .info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cloud-card .title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cloud-card .name {
  font-weight: 700;
  color: var(--color-text-main);
}

.version-tag {
  padding: 2px 8px;
  background: #e3f2fd;
  color: #1976d2;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.account-tag {
  padding: 2px 8px;
  background: rgba(128, 64, 48, 0.08);
  border-radius: 6px;
  color: var(--color-primary);
}

.cloud-card-actions {
  display: flex;
  gap: 6px;
}

.btn-icon.small {
  width: 32px;
  height: 32px;
  font-size: 16px;
}

.btn-icon.danger {
  color: #d32f2f;
  border-color: rgba(211, 47, 47, 0.3);
}

.btn-icon.danger:hover {
  background: #ffebee;
  border-color: #d32f2f;
}

/* 删除弹窗样式 */
.delete-modal {
  width: 420px;
}

.delete-info {
  margin: 16px 0;
  padding: 12px;
  background: #fafafa;
  border-radius: 10px;
}

.delete-info .info-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
}

.delete-info .label {
  color: var(--color-text-secondary);
  font-size: 13px;
}

.delete-info .value {
  color: var(--color-text-main);
  font-size: 13px;
  font-weight: 500;
}

.confirm-info {
  margin: 16px 0;
  padding: 12px;
  background: #fafafa;
  border-radius: 10px;
}

.confirm-info .info-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
}

.confirm-info .label {
  color: var(--color-text-secondary);
  font-size: 13px;
}

.confirm-info .value {
  color: var(--color-text-main);
  font-size: 13px;
  font-weight: 500;
}

.confirm-info .value.status.synced { color: #2e7d32; }
.confirm-info .value.status.pending { color: #ed6c02; }
.confirm-info .value.status.conflict { color: #d32f2f; }

.btn-danger {
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #d32f2f;
  color: #fff;
}

.btn-danger:hover {
  background: #b71c1c;
}

.btn-danger:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 1280px) {
  .overview-grid { grid-template-columns: repeat(2, 1fr); }
  .workspace { flex-direction: column; }
  .left-panel { width: 100%; min-width: auto; }
  .divider-handle { display: none; }
}
</style>
