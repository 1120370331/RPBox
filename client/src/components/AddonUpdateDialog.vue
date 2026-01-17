<script setup lang="ts">
import { ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { getAddonDownloadUrl } from '@/api/addon'

const visible = ref(false)
const currentVersion = ref('')
const latestVersion = ref('')
const changelog = ref('')
const wowPath = ref('')
const flavor = ref('_retail_')
const loading = ref(false)
const error = ref('')

const emit = defineEmits(['installed'])

function show(current: string, latest: string, changelogText: string = '', path: string = '', flavorValue: string = '_retail_') {
  currentVersion.value = current
  latestVersion.value = latest
  changelog.value = changelogText
  wowPath.value = path || localStorage.getItem('wow_path') || ''
  flavor.value = flavorValue
  error.value = ''
  visible.value = true
}

function close() {
  visible.value = false
}

async function handleDownload() {
  console.log('[AddonUpdateDialog] handleDownload 开始')
  console.log('[AddonUpdateDialog] wowPath:', wowPath.value)
  console.log('[AddonUpdateDialog] flavor:', flavor.value)
  console.log('[AddonUpdateDialog] latestVersion:', latestVersion.value)

  if (!wowPath.value) {
    error.value = '未找到魔兽世界路径，请先设置'
    console.log('[AddonUpdateDialog] 错误: 未找到魔兽世界路径')
    return
  }

  loading.value = true
  error.value = ''
  try {
    const url = getAddonDownloadUrl(latestVersion.value)
    console.log('[AddonUpdateDialog] 下载 URL:', url)

    const response = await fetch(url)
    console.log('[AddonUpdateDialog] fetch 响应状态:', response.status)

    if (!response.ok) throw new Error('下载失败')

    const arrayBuffer = await response.arrayBuffer()
    const zipData = Array.from(new Uint8Array(arrayBuffer))
    console.log('[AddonUpdateDialog] 下载完成，大小:', zipData.length)

    await invoke('install_addon', {
      wowPath: wowPath.value,
      flavor: flavor.value,
      zipData,
    })
    console.log('[AddonUpdateDialog] 安装成功')

    // 安装成功，触发事件通知父组件
    console.log('[AddonUpdateDialog] 触发 installed 事件')
    emit('installed')
    console.log('[AddonUpdateDialog] 关闭对话框')
    close()
  } catch (e: any) {
    console.error('[AddonUpdateDialog] 错误:', e)
    error.value = e.message || '安装失败'
  } finally {
    loading.value = false
  }
}

defineExpose({
  show
})
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="addon-update-overlay" @click.self="close">
      <div class="addon-update-dialog">
        <div class="dialog-header">
          <div class="header-content">
            <i class="ri-download-cloud-line"></i>
            <h2>插件更新提示</h2>
          </div>
          <button class="close-btn" @click="close">
            <i class="ri-close-line"></i>
          </button>
        </div>

        <div class="dialog-body">
          <div class="version-info">
            <div class="version-item">
              <span class="label">当前版本：</span>
              <span class="version current">v{{ currentVersion }}</span>
            </div>
            <i class="ri-arrow-right-line arrow-icon"></i>
            <div class="version-item">
              <span class="label">最新版本：</span>
              <span class="version latest">v{{ latestVersion }}</span>
            </div>
          </div>

          <!-- 更新内容 -->
          <div v-if="changelog" class="changelog-section">
            <h3>更新内容</h3>
            <div class="changelog-content">{{ changelog }}</div>
          </div>

          <div class="update-message">
            <i class="ri-information-line"></i>
            <p>检测到 RPBox Addon 插件有新版本可用，点击"立即安装"将自动下载并安装到插件目录。</p>
          </div>

          <!-- 错误提示 -->
          <div v-if="error" class="error-message">
            <i class="ri-error-warning-line"></i>
            <p>{{ error }}</p>
          </div>
        </div>

        <div class="dialog-footer">
          <button class="btn-secondary" @click="close" :disabled="loading">
            稍后更新
          </button>
          <button class="btn-primary" @click="handleDownload" :disabled="loading">
            <i v-if="!loading" class="ri-download-line"></i>
            <i v-else class="ri-loader-4-line spinning"></i>
            {{ loading ? '安装中...' : '立即安装' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.addon-update-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  animation: fadeIn 0.2s;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.addon-update-dialog {
  background: #fff;
  border-radius: 16px;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
  border-bottom: 1px solid #E0E0E0;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-content i {
  font-size: 28px;
  color: #B87333;
}

.dialog-header h2 {
  font-size: 20px;
  color: #3E2723;
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.close-btn:hover {
  background: rgba(0, 0, 0, 0.05);
}

.close-btn i {
  font-size: 20px;
  color: #666;
}

.dialog-body {
  padding: 24px;
}

.version-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  margin-bottom: 24px;
  padding: 20px;
  background: #F5F0EB;
  border-radius: 12px;
}

.version-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.label {
  font-size: 13px;
  color: #999;
}

.version {
  font-size: 18px;
  font-weight: 600;
  padding: 8px 16px;
  border-radius: 8px;
}

.version.current {
  background: #E0E0E0;
  color: #666;
}

.version.latest {
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
}

.arrow-icon {
  font-size: 24px;
  color: #B87333;
}

.changelog-section {
  margin-bottom: 20px;
  padding: 16px;
  background: #FFF8E1;
  border-radius: 8px;
  border-left: 4px solid #FFB300;
}

.changelog-section h3 {
  margin: 0 0 12px 0;
  font-size: 15px;
  color: #F57C00;
  font-weight: 600;
}

.changelog-content {
  color: #5D4037;
  font-size: 14px;
  line-height: 1.8;
  white-space: pre-wrap;
}

.update-message {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #E3F2FD;
  border-radius: 8px;
  border-left: 4px solid #2196F3;
}

.update-message i {
  font-size: 20px;
  color: #2196F3;
  flex-shrink: 0;
  margin-top: 2px;
}

.update-message p {
  margin: 0;
  color: #1565C0;
  font-size: 14px;
  line-height: 1.6;
}

.error-message {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #FFEBEE;
  border-radius: 8px;
  border-left: 4px solid #F44336;
  margin-top: 16px;
}

.error-message i {
  font-size: 20px;
  color: #F44336;
  flex-shrink: 0;
  margin-top: 2px;
}

.error-message p {
  margin: 0;
  color: #C62828;
  font-size: 14px;
  line-height: 1.6;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.dialog-footer {
  padding: 20px 24px;
  border-top: 1px solid #E0E0E0;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-secondary {
  padding: 10px 20px;
  background: #fff;
  color: #666;
  border: 1px solid #E0E0E0;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: #F5F5F5;
  border-color: #B87333;
  color: #B87333;
}

.btn-primary {
  padding: 10px 20px;
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.4);
}
</style>
