<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'

interface WowInstallation {
  path: string
  version: string
  flavor: string
}

const router = useRouter()
const wowPath = ref('')
const detectedPaths = ref<WowInstallation[]>([])
const isScanning = ref(false)
const autoSync = ref(false)
const syncOnStartup = ref(true)

onMounted(() => {
  wowPath.value = localStorage.getItem('wow_path') || ''
  autoSync.value = localStorage.getItem('auto_sync') === 'true'
  syncOnStartup.value = localStorage.getItem('sync_on_startup') !== 'false'
})

async function detectPaths() {
  isScanning.value = true
  try {
    detectedPaths.value = await invoke<WowInstallation[]>('detect_wow_paths')
  } finally {
    isScanning.value = false
  }
}

function selectPath(path: string) {
  wowPath.value = path
  saveSettings()
}

function saveSettings() {
  localStorage.setItem('wow_path', wowPath.value)
  localStorage.setItem('auto_sync', String(autoSync.value))
  localStorage.setItem('sync_on_startup', String(syncOnStartup.value))
}

async function clearCache() {
  if (confirm('确定要清除本地缓存吗？')) {
    await invoke('clear_sync_cache')
    alert('缓存已清除')
  }
}

function resetSetup() {
  localStorage.removeItem('wow_path')
  router.push('/sync/setup')
}
</script>

<template>
  <div class="settings">
    <div class="header">
      <button class="back-btn" @click="router.back()">&larr; 返回</button>
      <h2>设置</h2>
    </div>

    <div class="section">
      <h3>WoW 安装路径</h3>
      <div class="path-input">
        <input v-model="wowPath" type="text" placeholder="魔兽世界安装路径" />
        <button class="btn" @click="detectPaths" :disabled="isScanning">
          {{ isScanning ? '扫描中...' : '自动检测' }}
        </button>
      </div>
      <div v-if="detectedPaths.length > 0" class="detected-paths">
        <div
          v-for="p in detectedPaths"
          :key="p.path"
          class="path-item"
          :class="{ selected: wowPath === p.path }"
          @click="selectPath(p.path)"
        >
          <span class="path">{{ p.path }}</span>
          <span class="flavor">{{ p.flavor }} ({{ p.version }})</span>
        </div>
      </div>
    </div>

    <div class="section">
      <h3>同步设置</h3>
      <label class="checkbox-item">
        <input type="checkbox" v-model="syncOnStartup" @change="saveSettings" />
        <span>启动时自动同步</span>
      </label>
      <label class="checkbox-item">
        <input type="checkbox" v-model="autoSync" @change="saveSettings" />
        <span>检测到变更时自动同步</span>
      </label>
    </div>

    <div class="section">
      <h3>数据管理</h3>
      <div class="action-buttons">
        <button class="btn btn-secondary" @click="clearCache">清除本地缓存</button>
        <button class="btn btn-danger" @click="resetSetup">重新配置</button>
      </div>
    </div>

    <div class="section">
      <h3>关于</h3>
      <p class="about-text">RPBox v0.1.0</p>
      <p class="about-text">人物卡备份同步工具</p>
    </div>
  </div>
</template>

<style scoped>
.settings {
  padding: 1.5rem;
  max-width: 600px;
  margin: 0 auto;
}

.header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.header h2 {
  margin: 0;
}

.back-btn {
  background: none;
  border: none;
  font-size: 1rem;
  cursor: pointer;
  color: var(--color-primary);
}

.section {
  background: var(--color-bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1rem;
}

.section h3 {
  margin: 0 0 1rem 0;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.5rem;
}

.path-input {
  display: flex;
  gap: 0.5rem;
}

.path-input input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg);
  color: var(--color-text);
}

.detected-paths {
  margin-top: 1rem;
}

.path-item {
  display: flex;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--color-bg);
  border-radius: 4px;
  margin-bottom: 0.5rem;
  cursor: pointer;
}

.path-item.selected {
  outline: 2px solid var(--color-primary);
}

.path-item .flavor {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.checkbox-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  cursor: pointer;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  background: var(--color-primary);
  color: white;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-bg-tertiary);
  color: var(--color-text);
}

.btn-danger {
  background: #c00;
  color: white;
}

.about-text {
  margin: 0.25rem 0;
  color: var(--color-text-secondary);
}
</style>
