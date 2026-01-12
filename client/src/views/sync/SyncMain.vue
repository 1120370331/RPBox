<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import * as profileApi from '../../api/profile'
import { uploadProfiles, type SyncProgress } from '../../services/syncService'

interface ProfileItem {
  id: string
  name: string
  icon?: string
  checksum: string
  raw_lua: string
  account_id: string
  saved_variables_path: string
}

interface AccountInfo {
  account_id: string
  profiles: ProfileItem[]
}

interface SyncStatus {
  [profileId: string]: 'synced' | 'pending' | 'conflict'
}

const router = useRouter()
const accounts = ref<AccountInfo[]>([])
const selectedAccount = ref('')
const selectedProfiles = ref<Set<string>>(new Set())
const isLoading = ref(false)
const isSyncing = ref(false)
const syncProgress = ref<SyncProgress | null>(null)
const syncStatus = ref<SyncStatus>({})
const cloudProfiles = ref<Map<string, profileApi.CloudProfile>>(new Map())
const mounted = ref(false)

const currentProfiles = computed(() => {
  const acc = accounts.value.find(a => a.account_id === selectedAccount.value)
  return acc?.profiles || []
})

const hasSelection = computed(() => selectedProfiles.value.size > 0)

const stats = computed(() => {
  let synced = 0, pending = 0, conflict = 0
  Object.values(syncStatus.value).forEach(s => {
    if (s === 'synced') synced++
    else if (s === 'pending') pending++
    else conflict++
  })
  return { synced, pending, conflict, total: synced + pending + conflict }
})

onMounted(async () => {
  const wowPath = localStorage.getItem('wow_path')
  if (!wowPath) {
    router.push('/sync/setup')
    return
  }
  await loadProfiles()
  setTimeout(() => mounted.value = true, 50)
})

async function loadProfiles() {
  isLoading.value = true
  try {
    const [localResult, cloudList] = await Promise.all([
      invoke<{ accounts: AccountInfo[] }>('scan_profiles', {
        wowPath: localStorage.getItem('wow_path') || ''
      }),
      profileApi.listProfiles().catch(() => [])
    ])
    accounts.value = localResult.accounts
    if (localResult.accounts.length > 0) {
      selectedAccount.value = localResult.accounts[0].account_id
    }
    cloudProfiles.value.clear()
    cloudList.forEach(p => cloudProfiles.value.set(p.id, p))
    updateSyncStatus()
  } finally {
    isLoading.value = false
  }
}

function updateSyncStatus() {
  const status: SyncStatus = {}
  for (const acc of accounts.value) {
    for (const p of acc.profiles) {
      const cloud = cloudProfiles.value.get(p.id)
      if (!cloud) status[p.id] = 'pending'
      else if (cloud.checksum === p.checksum) status[p.id] = 'synced'
      else status[p.id] = 'conflict'
    }
  }
  syncStatus.value = status
}

function getStatus(id: string): 'synced' | 'pending' | 'conflict' {
  return syncStatus.value[id] || 'pending'
}

function toggleSelect(id: string) {
  if (selectedProfiles.value.has(id)) {
    selectedProfiles.value.delete(id)
  } else {
    selectedProfiles.value.add(id)
  }
}

async function syncAll() {
  const toSync = currentProfiles.value.filter(p => getStatus(p.id) !== 'synced')
  if (toSync.length === 0) return

  const data = toSync.map(p => ({
    id: p.id,
    account_id: p.account_id,
    profile_name: p.name,
    raw_lua: p.raw_lua,
    checksum: p.checksum
  }))

  isSyncing.value = true
  try {
    await uploadProfiles(data, (progress) => {
      syncProgress.value = progress
    })
    await loadProfiles()
  } finally {
    isSyncing.value = false
    syncProgress.value = null
  }
}

function goToDetail(id: string) {
  router.push(`/sync/profile/${id}`)
}
</script>

<template>
  <div class="sync-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部栏 -->
    <header class="topbar anim-item" style="--delay: 0">
      <div class="topbar-left">
        <h1 class="page-title">人物卡同步</h1>
        <select v-model="selectedAccount" class="account-select">
          <option v-for="acc in accounts" :key="acc.account_id" :value="acc.account_id">
            {{ acc.account_id }}
          </option>
        </select>
      </div>
      <div class="topbar-actions">
        <button class="btn-icon" @click="loadProfiles" :disabled="isLoading">
          <span :class="{ spinning: isLoading }">↻</span>
        </button>
        <button class="btn-icon" @click="router.push('/settings')">⚙️</button>
      </div>
    </header>

    <!-- 统计卡片 -->
    <div class="stats-row anim-item" style="--delay: 1">
      <div class="stat-card">
        <span class="stat-value">{{ stats.total }}</span>
        <span class="stat-label">总人物卡</span>
      </div>
      <div class="stat-card synced">
        <span class="stat-value">{{ stats.synced }}</span>
        <span class="stat-label">已同步</span>
      </div>
      <div class="stat-card pending">
        <span class="stat-value">{{ stats.pending }}</span>
        <span class="stat-label">待同步</span>
      </div>
      <div class="stat-card conflict">
        <span class="stat-value">{{ stats.conflict }}</span>
        <span class="stat-label">有冲突</span>
      </div>
    </div>

    <!-- 同步操作栏 -->
    <div class="action-bar anim-item" style="--delay: 2">
      <span class="action-hint">点击卡片查看详情</span>
      <button class="btn-primary" @click="syncAll" :disabled="isSyncing || stats.pending === 0">
        {{ isSyncing ? '同步中...' : '一键同步全部' }}
      </button>
    </div>

    <!-- 同步进度 -->
    <div v-if="syncProgress" class="progress-bar">
      <div class="progress-fill" :style="{ width: `${(syncProgress.completed / syncProgress.total) * 100}%` }"></div>
      <span class="progress-text">{{ syncProgress.completed }}/{{ syncProgress.total }}</span>
    </div>

    <!-- 人物卡列表 -->
    <div v-if="isLoading" class="loading-state">
      <div class="loader"></div>
      <p>正在扫描人物卡...</p>
    </div>

    <div v-else-if="currentProfiles.length === 0" class="empty-state anim-item" style="--delay: 2">
      <div class="empty-icon">👤</div>
      <h3>未找到人物卡</h3>
      <p>请确认WoW路径配置正确</p>
      <button class="btn-secondary" @click="router.push('/sync/setup')">重新配置</button>
    </div>

    <div v-else class="profile-grid">
      <div
        v-for="(p, index) in currentProfiles"
        :key="p.id"
        class="profile-card anim-item"
        :class="getStatus(p.id)"
        :style="{ '--delay': 2 + index * 0.1 }"
        @click="goToDetail(p.id)"
      >
        <div class="card-icon">{{ p.icon || '👤' }}</div>
        <div class="card-info">
          <h4 class="card-name">{{ p.name }}</h4>
          <span class="card-status">
            <template v-if="getStatus(p.id) === 'synced'">✓ 已同步</template>
            <template v-else-if="getStatus(p.id) === 'pending'">○ 待同步</template>
            <template v-else>⚠ 有冲突</template>
          </span>
        </div>
        <div class="card-arrow">→</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sync-page {
  padding: 24px;
  min-height: 100vh;
  background: var(--color-background);
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-primary);
  margin: 0;
}

.account-select {
  padding: 8px 12px;
  border: 2px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  background: #fff;
  font-size: 14px;
}

.topbar-actions { display: flex; gap: 8px; }

.btn-icon {
  width: 40px;
  height: 40px;
  border: 1px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  background: #fff;
  cursor: pointer;
  font-size: 18px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: #fff;
  padding: 20px;
  border-radius: var(--radius-md);
  text-align: center;
  border: 1px solid rgba(75, 54, 33, 0.1);
}

.stat-value {
  display: block;
  font-size: 32px;
  font-weight: 700;
  color: var(--color-primary);
}

.stat-label {
  font-size: 14px;
  color: var(--color-secondary);
}

.stat-card.synced .stat-value { color: #2e7d32; }
.stat-card.pending .stat-value { color: #ed6c02; }
.stat-card.conflict .stat-value { color: #d32f2f; }

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.action-hint {
  color: var(--color-secondary);
  font-size: 14px;
}

.btn-primary {
  padding: 12px 24px;
  background: var(--color-accent);
  color: var(--color-primary);
  border: none;
  border-radius: var(--radius-sm);
  font-weight: 600;
  cursor: pointer;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.progress-bar {
  height: 8px;
  background: rgba(75, 54, 33, 0.1);
  border-radius: 4px;
  margin-bottom: 20px;
  position: relative;
}

.progress-fill {
  height: 100%;
  background: var(--color-accent);
  border-radius: 4px;
  transition: width 0.3s;
}

.profile-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: var(--radius-md);
  border: 1px solid rgba(75, 54, 33, 0.1);
  cursor: pointer;
  transition: all 0.2s;
}

.profile-card:hover {
  transform: translateX(4px);
  box-shadow: 0 4px 12px rgba(75, 54, 33, 0.1);
}

.card-icon { font-size: 32px; }

.card-info { flex: 1; }

.card-name {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-primary);
}

.card-status { font-size: 13px; }

.profile-card.synced .card-status { color: #2e7d32; }
.profile-card.pending .card-status { color: #ed6c02; }
.profile-card.conflict .card-status { color: #d32f2f; }

.card-arrow {
  color: var(--color-secondary);
  opacity: 0.5;
}

.empty-state, .loading-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-icon { font-size: 64px; margin-bottom: 16px; }

.empty-state h3 {
  color: var(--color-primary);
  margin-bottom: 8px;
}

.empty-state p {
  color: var(--color-secondary);
  margin-bottom: 20px;
}

.btn-secondary {
  padding: 10px 20px;
  background: #fff;
  border: 2px solid var(--color-accent);
  border-radius: var(--radius-sm);
  color: var(--color-primary);
  cursor: pointer;
}

/* 动画 */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.5s ease forwards;
  animation-delay: calc(var(--delay) * 0.1s);
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}
</style>
