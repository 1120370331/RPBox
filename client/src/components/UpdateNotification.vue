<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUpdater } from '@/composables/useUpdater'
import RButton from './RButton.vue'

const {
  updateAvailable,
  updateInfo,
  downloading,
  downloadProgress,
  checkForUpdate,
  downloadAndInstall,
  lastError,
} = useUpdater()

const showModal = ref(false)
const installError = ref<string | null>(null)

onMounted(async () => {
  try {
    const update = await checkForUpdate()
    if (update) {
      showModal.value = true
    }
  } catch (e) {
    // 自动检查失败时静默处理，用户可以在设置中手动检查
    console.log('[UpdateNotification] 自动检查更新失败，静默处理')
  }
})

function handleClose() {
  showModal.value = false
}

async function handleInstall() {
  installError.value = null
  try {
    await downloadAndInstall()
  } catch (e: any) {
    installError.value = e?.message || '下载失败，请稍后重试'
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="showModal && updateAvailable" class="update-overlay">
        <div class="update-modal">
          <!-- 图标 -->
          <div class="update-icon">
            <i class="ri-rocket-2-line"></i>
          </div>

          <!-- 标题 -->
          <h2 class="update-title">发现新版本</h2>
          <div class="update-version">
            <span class="version-badge">v{{ updateInfo?.version }}</span>
          </div>

          <!-- 更新说明 -->
          <div v-if="updateInfo?.notes" class="update-notes">
            <div class="notes-content">{{ updateInfo.notes }}</div>
          </div>

          <!-- 下载进度 -->
          <div v-if="downloading" class="download-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
            </div>
            <span class="progress-text">正在下载... {{ Math.round(downloadProgress) }}%</span>
          </div>

          <!-- 错误提示 -->
          <div v-if="installError" class="error-message">
            <i class="ri-error-warning-line"></i>
            {{ installError }}
          </div>

          <!-- 操作按钮 -->
          <div class="update-actions">
            <RButton v-if="!downloading" size="large" @click="handleClose">
              稍后再说
            </RButton>
            <RButton
              type="primary"
              size="large"
              :loading="downloading"
              :disabled="downloading"
              @click="handleInstall"
            >
              <i v-if="!downloading" class="ri-download-line"></i>
              {{ downloading ? '下载中...' : '立即更新' }}
            </RButton>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.update-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  backdrop-filter: blur(4px);
}

.update-modal {
  background: #fff;
  border-radius: 20px;
  padding: 40px 48px;
  max-width: 420px;
  width: 90%;
  text-align: center;
  box-shadow: 0 20px 60px rgba(75, 54, 33, 0.3);
}

.update-icon {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #B87333, #D4A373);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 24px;
}

.update-icon i {
  font-size: 40px;
  color: #fff;
}

.update-title {
  font-size: 24px;
  font-weight: 700;
  color: #4B3621;
  margin: 0 0 12px;
}

.update-version {
  margin-bottom: 20px;
}

.version-badge {
  display: inline-block;
  padding: 6px 16px;
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
  border-radius: 20px;
  font-size: 16px;
  font-weight: 600;
}

.update-notes {
  margin-bottom: 24px;
  padding: 16px;
  background: #f9f6f3;
  border-radius: 12px;
  max-height: 150px;
  overflow-y: auto;
}

.notes-content {
  font-size: 14px;
  color: #665242;
  line-height: 1.6;
  text-align: left;
  white-space: pre-wrap;
}

.download-progress {
  margin-bottom: 24px;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #f0e6dc;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #B87333, #D4A373);
  transition: width 0.3s;
}

.progress-text {
  font-size: 13px;
  color: #856a52;
}

.error-message {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  background: #fff5f5;
  border: 1px solid #ffccc7;
  border-radius: 8px;
  color: #cf1322;
  font-size: 13px;
  margin-bottom: 20px;
}

.update-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.update-actions :deep(.r-button) {
  min-width: 120px;
}

/* 动画 */
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.fade-enter-from .update-modal,
.fade-leave-to .update-modal {
  transform: scale(0.9);
}
</style>
