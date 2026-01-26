<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

interface DetectedPath {
  path: string
  version: string | { Retail?: null; Classic?: null }
  accounts: string[]
}

const router = useRouter()
const { t } = useI18n()
const currentStep = ref(1)
const isLoading = ref(false)
const isScanning = ref(false)
const mounted = ref(false)

// Step 1
const detectedPaths = ref<DetectedPath[]>([])
const selectedPath = ref('')
const manualPath = ref('')
const pathError = ref('')

// Step 2
const profileCount = ref(0)

onMounted(async () => {
  setTimeout(() => mounted.value = true, 50)
  await autoDetect()
})

async function autoDetect() {
  isScanning.value = true
  try {
    const { invoke } = await import('@tauri-apps/api/core')
    const result = await invoke<DetectedPath[]>('detect_wow_paths')
    detectedPaths.value = result || []
    if (detectedPaths.value.length > 0) {
      selectedPath.value = detectedPaths.value[0].path
    }
  } catch (e) {
    console.error('自动检测失败:', e)
    detectedPaths.value = []
  } finally {
    isScanning.value = false
  }
}

function selectPath(path: string) {
  selectedPath.value = path
  pathError.value = ''
}

function useManualPath() {
  if (manualPath.value.trim()) {
    selectedPath.value = manualPath.value.trim()
    pathError.value = ''
  }
}

async function browseFolder() {
  try {
    const { open } = await import('@tauri-apps/plugin-dialog')
    const selected = await open({
      directory: true,
      title: t('sync.setup.selectWowDirTitle')
    })
    if (selected) {
      // 智能规范化路径
      const { invoke } = await import('@tauri-apps/api/core')
      const normalized = await invoke<string | null>('normalize_wow_path', { path: selected })
      if (normalized) {
        selectedPath.value = normalized
        pathError.value = ''
      } else {
        pathError.value = t('sync.setup.errorInvalidPath')
      }
    }
  } catch (e) {
    console.error('打开文件夹选择器失败:', e)
    pathError.value = t('sync.setup.errorOpenDialog')
  }
}

async function validateAndNext() {
  if (!selectedPath.value) {
    pathError.value = t('sync.setup.errorNoPath')
    return
  }

  isLoading.value = true
  pathError.value = ''

  try {
    const { invoke } = await import('@tauri-apps/api/core')
    const result = await invoke<{ accounts: any[] }>('scan_profiles', {
      wowPath: selectedPath.value
    })
    profileCount.value = result.accounts.reduce((sum, a) => sum + (a.profiles?.length || 0), 0)
    currentStep.value = 2
  } catch (e) {
    pathError.value = t('sync.setup.errorAccessPath')
  } finally {
    isLoading.value = false
  }
}

function complete() {
  localStorage.setItem('wow_path', selectedPath.value)
  router.push('/sync')
}
</script>

<template>
  <div class="setup-page" :class="{ 'animate-in': mounted }">
    <div class="setup-card anim-item" style="--delay: 0">
      <!-- 头部 -->
      <div class="card-header">
        <div class="logo">RPBOX</div>
        <h1>{{ $t('sync.setup.title') }}</h1>
        <div class="steps">
          <span :class="{ active: currentStep >= 1 }">1</span>
          <span class="line" :class="{ active: currentStep >= 2 }"></span>
          <span :class="{ active: currentStep >= 2 }">2</span>
        </div>
      </div>

      <!-- Step 1: 输入路径 -->
      <div v-if="currentStep === 1" class="step-content">
        <h2>{{ $t('sync.setup.selectWowDir') }}</h2>

        <!-- 自动检测结果 -->
        <div v-if="isScanning" class="scanning">
          <span class="spinner">↻</span> {{ $t('sync.setup.scanning') }}
        </div>

        <div v-else-if="detectedPaths.length > 0" class="detected-list">
          <p class="hint">{{ $t('sync.setup.detectedInstalls') }}</p>
          <div
            v-for="p in detectedPaths"
            :key="p.path"
            class="path-option"
            :class="{ selected: selectedPath === p.path }"
            @click="selectPath(p.path)"
          >
            <span class="path-name">{{ p.path }}</span>
            <span class="path-info">{{ $t('sync.setup.accountCount', { count: p.accounts?.length || 0 }) }}</span>
          </div>
        </div>

        <div v-else class="no-detect">
          <p>{{ $t('sync.setup.noDetect') }}</p>
        </div>

        <!-- 手动输入 -->
        <div class="manual-section">
          <p class="hint">{{ $t('sync.setup.manualHint') }}</p>
          <button class="btn-browse" @click="browseFolder">📁 {{ $t('sync.setup.browseFolder') }}</button>
          <div class="manual-row">
            <input
              v-model="manualPath"
              type="text"
              class="path-input"
              :placeholder="$t('sync.setup.inputPlaceholder')"
            />
            <button class="btn-small" @click="useManualPath">{{ $t('sync.setup.confirm') }}</button>
          </div>
        </div>

        <p v-if="pathError" class="error">{{ pathError }}</p>
        <p v-if="selectedPath" class="selected-hint">{{ $t('sync.setup.selected', { path: selectedPath }) }}</p>

        <button class="btn-primary" @click="validateAndNext" :disabled="isLoading || !selectedPath">
          {{ isLoading ? $t('sync.setup.validating') : $t('sync.setup.nextStep') }}
        </button>
      </div>

      <!-- Step 2: 完成 -->
      <div v-if="currentStep === 2" class="step-content success">
        <div class="success-icon">✓</div>
        <h2>{{ $t('sync.setup.complete') }}</h2>
        <p class="result" v-html="$t('sync.setup.foundProfiles', { count: `<strong>${profileCount}</strong>` })"></p>
        <button class="btn-primary" @click="complete">{{ $t('sync.setup.startUsing') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.setup-page {
  min-height: 100vh;
  background: var(--color-background);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.setup-card {
  background: #fff;
  border-radius: var(--radius-lg);
  padding: 48px;
  width: 100%;
  max-width: 480px;
  box-shadow: 0 8px 32px rgba(75, 54, 33, 0.1);
}

.card-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-accent);
  margin-bottom: 8px;
}

.card-header h1 {
  font-size: 20px;
  color: var(--color-primary);
  margin: 0 0 24px 0;
}

.steps {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.steps span {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(75, 54, 33, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: var(--color-secondary);
}

.steps span.active {
  background: var(--color-accent);
  color: #fff;
}

.steps .line {
  width: 40px;
  height: 2px;
  background: rgba(75, 54, 33, 0.1);
  border-radius: 0;
}

.steps .line.active {
  background: var(--color-accent);
}

.step-content h2 {
  font-size: 18px;
  color: var(--color-primary);
  margin: 0 0 8px 0;
}

.hint {
  color: var(--color-secondary);
  font-size: 14px;
  margin-bottom: 20px;
}

.path-input {
  width: 100%;
  padding: 14px 16px;
  border: 2px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  font-size: 14px;
  margin-bottom: 12px;
}

.path-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.error {
  color: #d32f2f;
  font-size: 13px;
  margin-bottom: 12px;
}

.btn-primary {
  width: 100%;
  padding: 14px;
  background: var(--color-accent);
  color: var(--color-primary);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.success { text-align: center; }

.success-icon {
  width: 64px;
  height: 64px;
  background: #2e7d32;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  margin: 0 auto 16px;
}

.result { color: var(--color-secondary); margin-bottom: 24px; }
.result strong { color: var(--color-primary); }

.scanning {
  text-align: center;
  padding: 20px;
  color: var(--color-secondary);
}

.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.detected-list {
  margin-bottom: 16px;
}

.path-option {
  padding: 12px 16px;
  background: var(--color-background);
  border: 2px solid transparent;
  border-radius: var(--radius-sm);
  margin-bottom: 8px;
  cursor: pointer;
}

.path-option.selected {
  border-color: var(--color-accent);
  background: #fff;
}

.path-name {
  display: block;
  font-size: 13px;
  color: var(--color-primary);
}

.path-info {
  font-size: 12px;
  color: var(--color-secondary);
}

.manual-section {
  margin: 16px 0;
  padding-top: 16px;
  border-top: 1px solid rgba(75,54,33,0.1);
}

.btn-browse {
  width: 100%;
  padding: 14px;
  background: var(--color-primary);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 15px;
  cursor: pointer;
  margin-bottom: 12px;
}

.btn-browse:hover {
  opacity: 0.9;
}

.manual-row {
  display: flex;
  gap: 8px;
}

.btn-small {
  padding: 10px 16px;
  background: var(--color-secondary);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.selected-hint {
  font-size: 13px;
  color: var(--color-accent);
  margin-bottom: 16px;
}

/* 动画 */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.6s ease forwards;
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}
</style>
