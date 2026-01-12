<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import * as profileApi from '../../api/profile'

interface LocalProfile {
  id: string
  name: string
  icon?: string
  checksum: string
  raw_lua?: string
  account_id?: string
  saved_variables_path?: string
  characteristics?: {
    firstName?: string
    lastName?: string
    title?: string
    race?: string
    class?: string
    age?: string
  }
  about?: { title?: string; text?: string }
}

const route = useRoute()
const router = useRouter()
const profileId = computed(() => route.params.id as string)

const profile = ref<LocalProfile | null>(null)
const cloudProfile = ref<profileApi.CloudProfile | null>(null)
const isLoading = ref(false)
const isSyncing = ref(false)
const mounted = ref(false)

const syncStatus = computed(() => {
  if (!profile.value) return 'unknown'
  if (!cloudProfile.value) return 'pending'
  return profile.value.checksum === cloudProfile.value.checksum ? 'synced' : 'conflict'
})

onMounted(async () => {
  await loadProfile()
  setTimeout(() => mounted.value = true, 50)
})

async function loadProfile() {
  isLoading.value = true
  try {
    const wowPath = localStorage.getItem('wow_path') || ''
    const [local, cloud] = await Promise.all([
      invoke<LocalProfile>('get_profile_detail', { wowPath: wowPath, profileId: profileId.value }),
      profileApi.getProfile(profileId.value).catch(() => null)
    ])
    profile.value = local
    cloudProfile.value = cloud
  } finally {
    isLoading.value = false
  }
}

async function syncProfile() {
  if (!profile.value) return
  isSyncing.value = true
  try {
    const data = {
      id: profile.value.id,
      account_id: profile.value.account_id || '',
      profile_name: profile.value.name,
      raw_lua: profile.value.raw_lua || '',
      checksum: profile.value.checksum
    }
    cloudProfile.value = cloudProfile.value
      ? await profileApi.updateProfile(profileId.value, data)
      : await profileApi.createProfile(data)
  } finally {
    isSyncing.value = false
  }
}

async function restoreFromCloud() {
  if (!cloudProfile.value?.raw_lua || !cloudProfile.value.account_id) return
  const wowPath = localStorage.getItem('wow_path') || ''
  isSyncing.value = true
  try {
    await invoke('apply_cloud_profile', {
      wowPath: wowPath,
      accountId: cloudProfile.value.account_id,
      profileId: profileId.value,
      profileJson: cloudProfile.value.raw_lua
    })
    await loadProfile()
  } finally {
    isSyncing.value = false
  }
}
</script>

<template>
  <div class="detail-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部导航 -->
    <header class="topbar anim-item" style="--delay: 0">
      <button class="back-btn" @click="router.back()">← 返回</button>
      <div class="actions">
        <button class="btn-primary" @click="restoreFromCloud" :disabled="isSyncing || !cloudProfile?.raw_lua">
          {{ isSyncing ? '处理中...' : '从云端恢复' }}
        </button>
        <button class="btn-primary" @click="syncProfile" :disabled="isSyncing || syncStatus === 'synced'">
          {{ isSyncing ? '处理中...' : '上传到云端' }}
        </button>
      </div>
    </header>

    <div v-if="isLoading" class="loading">加载中...</div>

    <template v-else-if="profile">
      <!-- 人物卡头部 -->
      <div class="profile-hero anim-item" style="--delay: 1">
        <div class="hero-icon">{{ profile.icon || '👤' }}</div>
        <div class="hero-info">
          <h1>{{ profile.name }}</h1>
          <span class="status-tag" :class="syncStatus">
            {{ syncStatus === 'synced' ? '已同步' : syncStatus === 'pending' ? '待同步' : '有冲突' }}
          </span>
        </div>
      </div>

      <!-- 基本信息 -->
      <div class="info-card anim-item" style="--delay: 2" v-if="profile.characteristics">
        <h3>基本信息</h3>
        <div class="info-grid">
          <div class="info-item" v-if="profile.characteristics.race">
            <label>种族</label>
            <span>{{ profile.characteristics.race }}</span>
          </div>
          <div class="info-item" v-if="profile.characteristics.class">
            <label>职业</label>
            <span>{{ profile.characteristics.class }}</span>
          </div>
          <div class="info-item" v-if="profile.characteristics.age">
            <label>年龄</label>
            <span>{{ profile.characteristics.age }}</span>
          </div>
          <div class="info-item" v-if="profile.characteristics.title">
            <label>头衔</label>
            <span>{{ profile.characteristics.title }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.detail-page {
  padding: 24px;
  max-width: 800px;
  margin: 0 auto;
  min-height: 100vh;
  background: var(--color-background);
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.back-btn {
  background: none;
  border: none;
  font-size: 15px;
  color: var(--color-primary);
  cursor: pointer;
}

.actions {
  display: flex;
  gap: 8px;
}

.profile-hero {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 32px;
  background: #fff;
  border-radius: var(--radius-lg);
  margin-bottom: 20px;
}

.hero-icon { font-size: 64px; }

.hero-info h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  color: var(--color-primary);
}

.status-tag {
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
}

.status-tag.synced { background: #e8f5e9; color: #2e7d32; }
.status-tag.pending { background: #fff3e0; color: #ed6c02; }
.status-tag.conflict { background: #ffebee; color: #d32f2f; }

.info-card {
  background: #fff;
  border-radius: var(--radius-md);
  padding: 24px;
  margin-bottom: 16px;
}

.info-card h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: var(--color-primary);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item label {
  display: block;
  font-size: 12px;
  color: var(--color-secondary);
  margin-bottom: 4px;
}

.info-item span {
  font-size: 15px;
  color: var(--color-primary);
}

.btn-primary {
  padding: 10px 20px;
  background: var(--color-accent);
  color: var(--color-primary);
  border: none;
  border-radius: var(--radius-sm);
  font-weight: 600;
  cursor: pointer;
}

.btn-primary:disabled { opacity: 0.5; }

.loading { text-align: center; padding: 60px; }

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
