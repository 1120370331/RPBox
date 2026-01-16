<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { useUserStore } from '@/stores/user'
import { useToastStore } from '@/stores/toast'
import { uploadAvatar } from '@/api/user'
import { useUpdater } from '@/composables/useUpdater'

interface WowInstallation {
  path: string
  version: string
  flavor: string
}

const router = useRouter()
const userStore = useUserStore()
const toast = useToastStore()
const { checking, updateAvailable, updateInfo, checkForUpdate, downloadAndInstall, downloading, downloadProgress } = useUpdater()

const mounted = ref(false)
const wowPath = ref('')
const detectedPaths = ref<WowInstallation[]>([])
const isScanning = ref(false)
const autoSync = ref(false)
const syncOnStartup = ref(true)
const avatarUploading = ref(false)
const avatarInputRef = ref<HTMLInputElement | null>(null)

onMounted(() => {
  wowPath.value = localStorage.getItem('wow_path') || ''
  autoSync.value = localStorage.getItem('auto_sync') === 'true'
  syncOnStartup.value = localStorage.getItem('sync_on_startup') !== 'false'
  setTimeout(() => mounted.value = true, 50)
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

async function handleCheckUpdate() {
  try {
    const update = await checkForUpdate()
    if (update) {
      toast.success(`发现新版本 ${update.version}`)
    } else {
      toast.info('当前已是最新版本')
    }
  } catch (e) {
    console.error('检查更新失败:', e)
    toast.error('检查更新失败')
  }
}

function triggerAvatarUpload() {
  avatarInputRef.value?.click()
}

async function handleAvatarChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  if (file.size > 20 * 1024 * 1024) {
    toast.warning('头像文件不能超过20MB')
    return
  }

  avatarUploading.value = true
  try {
    const res = await uploadAvatar(file)
    userStore.updateAvatar(res.avatar)
    toast.success('头像更新成功')
  } catch (error: any) {
    toast.error(error.message || '上传失败')
  } finally {
    avatarUploading.value = false
    input.value = ''
  }
}
</script>

<template>
  <div class="settings-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="breadcrumbs">
        <i class="ri-settings-3-line"></i>
        <span class="current">系统设置</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary" @click="router.back()">
          <i class="ri-arrow-left-line"></i> 返回
        </button>
      </div>
    </div>

    <!-- 设置内容区 -->
    <div class="settings-content">
      <!-- 个人资料 -->
      <div class="setting-card anim-item" style="--delay: 1">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-user-line"></i>
          </div>
          <div class="card-title">
            <h3>个人资料</h3>
            <p>管理您的头像和账户信息</p>
          </div>
        </div>
        <div class="card-body">
          <div class="avatar-section">
            <div class="avatar-preview" @click="triggerAvatarUpload">
              <img v-if="userStore.user?.avatar" :src="userStore.user.avatar" alt="头像" />
              <span v-else class="avatar-placeholder">
                {{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
              </span>
              <div class="avatar-overlay">
                <i :class="avatarUploading ? 'ri-loader-4-line spin' : 'ri-camera-line'"></i>
              </div>
            </div>
            <div class="avatar-info">
              <h4>{{ userStore.user?.username || '未登录' }}</h4>
              <p>点击头像更换，支持 JPG、PNG 格式，最大 20MB</p>
            </div>
            <input
              ref="avatarInputRef"
              type="file"
              accept="image/*"
              style="display: none"
              @change="handleAvatarChange"
            />
          </div>
        </div>
      </div>

      <!-- WoW 安装路径 -->
      <div class="setting-card anim-item" style="--delay: 2">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-folder-3-line"></i>
          </div>
          <div class="card-title">
            <h3>WoW 安装路径</h3>
            <p>配置魔兽世界的安装目录</p>
          </div>
        </div>
        <div class="card-body">
          <div class="path-input-group">
            <input
              v-model="wowPath"
              type="text"
              placeholder="请输入或选择魔兽世界安装路径"
              class="path-input"
            />
            <button class="btn btn-primary" @click="detectPaths" :disabled="isScanning">
              <i :class="isScanning ? 'ri-loader-4-line spin' : 'ri-search-line'"></i>
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
              <div class="path-info">
                <i class="ri-gamepad-line"></i>
                <span class="path-text">{{ p.path }}</span>
              </div>
              <span class="flavor-tag">{{ p.flavor }} · {{ p.version }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 同步设置 -->
      <div class="setting-card anim-item" style="--delay: 3">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-refresh-line"></i>
          </div>
          <div class="card-title">
            <h3>同步设置</h3>
            <p>配置数据同步行为</p>
          </div>
        </div>
        <div class="card-body">
          <label class="switch-item" @click="syncOnStartup = !syncOnStartup; saveSettings()">
            <div class="switch-info">
              <span class="switch-label">启动时自动同步</span>
              <span class="switch-desc">应用启动后自动检查并同步数据</span>
            </div>
            <div class="switch" :class="{ active: syncOnStartup }">
              <div class="switch-thumb"></div>
            </div>
          </label>
          <label class="switch-item" @click="autoSync = !autoSync; saveSettings()">
            <div class="switch-info">
              <span class="switch-label">变更时自动同步</span>
              <span class="switch-desc">检测到本地文件变更时自动上传</span>
            </div>
            <div class="switch" :class="{ active: autoSync }">
              <div class="switch-thumb"></div>
            </div>
          </label>
        </div>
      </div>

      <!-- 数据管理 -->
      <div class="setting-card anim-item" style="--delay: 4">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-database-2-line"></i>
          </div>
          <div class="card-title">
            <h3>数据管理</h3>
            <p>管理本地缓存和配置</p>
          </div>
        </div>
        <div class="card-body">
          <div class="action-buttons">
            <button class="btn btn-outline" @click="clearCache">
              <i class="ri-delete-bin-line"></i>
              清除本地缓存
            </button>
            <button class="btn btn-danger" @click="resetSetup">
              <i class="ri-restart-line"></i>
              重新配置
            </button>
          </div>
        </div>
      </div>

      <!-- 关于 -->
      <div class="setting-card about-card anim-item" style="--delay: 5">
        <div class="about-content">
          <div class="about-logo">
            <i class="ri-box-3-line"></i>
          </div>
          <div class="about-info">
            <h3>RPBox</h3>
            <p class="version">v0.1.0</p>
            <p class="desc">魔兽世界 RP 玩家的工具箱</p>
          </div>
          <div class="about-actions">
            <button
              v-if="!updateAvailable"
              class="btn btn-about"
              @click="handleCheckUpdate"
              :disabled="checking"
            >
              <i :class="checking ? 'ri-loader-4-line spin' : 'ri-refresh-line'"></i>
              {{ checking ? '检查中...' : '检查更新' }}
            </button>
            <template v-else>
              <div class="update-info">
                <span class="new-version">新版本 {{ updateInfo?.version }}</span>
              </div>
              <button
                class="btn btn-update"
                @click="downloadAndInstall"
                :disabled="downloading"
              >
                <i :class="downloading ? 'ri-loader-4-line spin' : 'ri-download-line'"></i>
                {{ downloading ? `下载中 ${Math.round(downloadProgress)}%` : '立即更新' }}
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 顶部工具栏 */
.top-toolbar {
  background-color: #FFFFFF;
  border-radius: 16px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #8C7B70;
}

.breadcrumbs i {
  font-size: 18px;
  color: #804030;
}

.breadcrumbs .current {
  color: #804030;
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

/* 设置内容区 */
.settings-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 720px;
}

/* 设置卡片 */
.setting-card {
  background: #FFFFFF;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #E8DCCF;
}

.card-icon {
  width: 48px;
  height: 48px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon i {
  font-size: 24px;
  color: #804030;
}

.card-title h3 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
}

.card-title p {
  margin: 0;
  font-size: 13px;
  color: #8C7B70;
}

/* 头像区域 */
.avatar-section {
  display: flex;
  align-items: center;
  gap: 20px;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  position: relative;
  cursor: pointer;
  overflow: hidden;
  flex-shrink: 0;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: 700;
  color: #FFF;
}

.avatar-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s;
}

.avatar-preview:hover .avatar-overlay {
  opacity: 1;
}

.avatar-overlay i {
  font-size: 24px;
  color: #FFF;
}

.avatar-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
}

.avatar-info p {
  margin: 0;
  font-size: 13px;
  color: #8C7B70;
}

/* 路径输入 */
.path-input-group {
  display: flex;
  gap: 12px;
}

.path-input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid #E8DCCF;
  border-radius: 10px;
  background: #FDFBF9;
  color: #2C1810;
  font-size: 14px;
  transition: all 0.2s;
}

.path-input:focus {
  outline: none;
  border-color: #804030;
  box-shadow: 0 0 0 3px rgba(128, 64, 48, 0.1);
}

.path-input::placeholder {
  color: #8C7B70;
}

/* 检测到的路径 */
.detected-paths {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.path-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  background: #FDFBF9;
  border: 1px solid #E8DCCF;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.path-item:hover {
  background: rgba(128, 64, 48, 0.05);
  border-color: #D4A373;
}

.path-item.selected {
  background: rgba(128, 64, 48, 0.08);
  border-color: #804030;
}

.path-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.path-info i {
  font-size: 18px;
  color: #804030;
}

.path-text {
  font-size: 13px;
  color: #2C1810;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.flavor-tag {
  font-size: 12px;
  color: #8C7B70;
  background: rgba(128, 64, 48, 0.1);
  padding: 4px 10px;
  border-radius: 6px;
  white-space: nowrap;
}

/* 开关项 */
.switch-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #FDFBF9;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 10px;
}

.switch-item:last-child {
  margin-bottom: 0;
}

.switch-item:hover {
  background: rgba(128, 64, 48, 0.05);
}

.switch-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.switch-label {
  font-size: 14px;
  font-weight: 500;
  color: #2C1810;
}

.switch-desc {
  font-size: 12px;
  color: #8C7B70;
}

/* 开关组件 */
.switch {
  width: 44px;
  height: 24px;
  background: #D4D4D4;
  border-radius: 12px;
  position: relative;
  transition: all 0.3s;
  flex-shrink: 0;
}

.switch.active {
  background: #804030;
}

.switch-thumb {
  width: 20px;
  height: 20px;
  background: #FFFFFF;
  border-radius: 50%;
  position: absolute;
  top: 2px;
  left: 2px;
  transition: all 0.3s;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.switch.active .switch-thumb {
  left: 22px;
}

/* 操作按钮区 */
.action-buttons {
  display: flex;
  gap: 12px;
}

/* 按钮样式 */
.btn {
  padding: 10px 18px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-primary {
  background: #804030;
  color: #FFFFFF;
}

.btn-primary:hover:not(:disabled) {
  background: #6B3528;
  box-shadow: 0 4px 12px rgba(128, 64, 48, 0.3);
}

.btn-secondary {
  background: rgba(128, 64, 48, 0.1);
  color: #804030;
}

.btn-secondary:hover {
  background: rgba(128, 64, 48, 0.15);
}

.btn-outline {
  background: transparent;
  border: 1px solid #E8DCCF;
  color: #2C1810;
}

.btn-outline:hover {
  background: rgba(128, 64, 48, 0.05);
  border-color: #D4A373;
}

.btn-danger {
  background: #DC3545;
  color: #FFFFFF;
}

.btn-danger:hover {
  background: #C82333;
  box-shadow: 0 4px 12px rgba(220, 53, 69, 0.3);
}

/* 关于卡片 */
.about-card {
  background: linear-gradient(135deg, #804030 0%, #4B3621 100%);
}

.about-content {
  display: flex;
  align-items: center;
  gap: 20px;
}

.about-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-about {
  background: rgba(255, 255, 255, 0.15);
  color: #FBF5EF;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.btn-about:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.25);
}

.btn-update {
  background: #D4A373;
  color: #2C1810;
}

.btn-update:hover:not(:disabled) {
  background: #E5B584;
}

.update-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.new-version {
  font-size: 13px;
  color: #D4A373;
  font-weight: 600;
}

.about-logo {
  width: 64px;
  height: 64px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.about-logo i {
  font-size: 32px;
  color: #FBF5EF;
}

.about-info h3 {
  margin: 0 0 4px 0;
  font-size: 20px;
  font-weight: 700;
  color: #FBF5EF;
}

.about-info .version {
  margin: 0 0 4px 0;
  font-size: 13px;
  color: rgba(251, 245, 239, 0.7);
}

.about-info .desc {
  margin: 0;
  font-size: 13px;
  color: rgba(251, 245, 239, 0.6);
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
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 旋转动画 */
.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
