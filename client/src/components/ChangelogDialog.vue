<script setup lang="ts">
import { ref, onMounted } from 'vue'
import RDialog from './RDialog.vue'
import { useDialog } from '@/composables/useDialog'

const dialog = useDialog()
const showChangelog = ref(false)

// 当前版本（从 package.json 读取）
const CURRENT_VERSION = '0.2.7'

// 更新日志内容
const changelog = {
  '0.2.7': {
    date: '2026-01-23',
    features: [
      '插件更新 V1.0.8，适配 12.0 前夕版本',
      '帖子新增"内部链接"，可跳转到其他帖子、公会剧情、公会主页',
      '帖子作品评论区新增表情包功能，后续会持续更新',
      '公会头像支持修改',
    ],
  },
  '0.2.6': {
    date: '2026-01-22',
    features: [
      '修复社区广场公告恢复显示',
      '精华帖子仅标记不再提高排序',
      '删除帖子与评论统一使用系统确认弹窗',
      '删除帖子/作品后自动清理 OSS 图片',
      '版主/管理员可编辑删除他人帖子',
      '优化版主中心新增帖子筛选：置顶/精华/普通',
      '优化管理中心帖子列表新增编辑入口',
    ],
  },
  // 可以继续添加历史版本的更新日志
}

onMounted(() => {
  checkVersion()
})

function checkVersion() {
  const lastViewedVersion = localStorage.getItem('last_viewed_version')

  // 如果是首次使用或版本更新，显示更新日志
  if (!lastViewedVersion || lastViewedVersion !== CURRENT_VERSION) {
    showChangelog.value = true
  }
}

function closeChangelog() {
  // 记录已查看的版本
  localStorage.setItem('last_viewed_version', CURRENT_VERSION)
  showChangelog.value = false
}

// 暴露方法供外部调用（手动打开更新日志）
defineExpose({
  open: () => { showChangelog.value = true }
})
</script>

<template>
  <Teleport to="body">
    <div v-if="showChangelog" class="changelog-overlay" @click.self="closeChangelog">
      <div class="changelog-dialog">
        <div class="changelog-header">
          <div class="header-content">
            <i class="ri-gift-line"></i>
            <h2>RPBox 更新日志</h2>
          </div>
          <button class="close-btn" @click="closeChangelog">
            <i class="ri-close-line"></i>
          </button>
        </div>

        <div class="changelog-body">
          <div v-for="(log, version) in changelog" :key="version" class="version-section">
            <div class="version-header">
              <span class="version-tag">v{{ version }}</span>
              <span class="version-date">{{ log.date }}</span>
            </div>

            <div class="features-list">
              <div v-for="(feature, index) in log.features" :key="index" class="feature-item">
                <i class="ri-checkbox-circle-fill"></i>
                <span>{{ feature }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="changelog-footer">
          <button class="btn-primary" @click="closeChangelog">
            知道了
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.changelog-overlay {
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

.changelog-dialog {
  background: #fff;
  border-radius: 16px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
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

.changelog-header {
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

.changelog-header h2 {
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

.changelog-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.version-section {
  margin-bottom: 24px;
}

.version-section:last-child {
  margin-bottom: 0;
}

.version-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.version-tag {
  padding: 6px 12px;
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 600;
}

.version-date {
  color: #999;
  font-size: 14px;
}

.features-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  background: #F5F0EB;
  border-radius: 8px;
  line-height: 1.6;
}

.feature-item i {
  font-size: 18px;
  color: #4CAF50;
  flex-shrink: 0;
  margin-top: 2px;
}

.feature-item span {
  color: #5D4037;
  font-size: 14px;
}

.changelog-footer {
  padding: 20px 24px;
  border-top: 1px solid #E0E0E0;
  display: flex;
  justify-content: center;
}

.btn-primary {
  padding: 12px 32px;
  background: linear-gradient(135deg, #B87333 0%, #D4A373 100%);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(184, 115, 51, 0.4);
}
</style>
