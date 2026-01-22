<script setup lang="ts">
import { onMounted } from 'vue'
import { useUpdater } from '@/composables/useUpdater'
import RButton from './RButton.vue'

const {
  updateAvailable,
  updateInfo,
  downloading,
  downloadProgress,
  checkForUpdate,
  downloadAndInstall,
} = useUpdater()

onMounted(() => {
  checkForUpdate().catch(() => {
    // Ignore auto-check errors; user can retry in Settings.
  })
})
</script>

<template>
  <Teleport to="body">
    <Transition name="slide">
      <div v-if="updateAvailable" class="update-notification">
        <div class="update-content">
          <i class="ri-download-cloud-line"></i>
          <div class="update-info">
            <span class="update-title">发现新版本 {{ updateInfo?.version }}</span>
            <span v-if="updateInfo?.notes" class="update-notes">{{ updateInfo.notes }}</span>
          </div>
        </div>
        <div class="update-actions">
          <div v-if="downloading" class="progress-bar">
            <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
          </div>
          <RButton
            v-else
            type="primary"
            size="small"
            @click="downloadAndInstall"
          >
            立即更新
          </RButton>
          <button class="btn-close" @click="updateAvailable = false">
            <i class="ri-close-line"></i>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.update-notification {
  position: fixed;
  bottom: 24px;
  right: 24px;
  background: #fff;
  border-radius: 12px;
  padding: 16px 20px;
  box-shadow: 0 8px 32px rgba(75, 54, 33, 0.15);
  display: flex;
  align-items: center;
  gap: 16px;
  z-index: 9999;
  max-width: 400px;
}

.update-content {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.update-content > i {
  font-size: 28px;
  color: #B87333;
}

.update-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.update-title {
  font-weight: 600;
  color: #4B3621;
  font-size: 14px;
}

.update-notes {
  font-size: 12px;
  color: #856a52;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.update-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-bar {
  width: 100px;
  height: 6px;
  background: #f0e6dc;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #B87333;
  transition: width 0.3s;
}

.btn-close {
  background: none;
  border: none;
  color: #856a52;
  cursor: pointer;
  padding: 4px;
  font-size: 18px;
  display: flex;
}

.btn-close:hover {
  color: #4B3621;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateX(100px);
}
</style>
