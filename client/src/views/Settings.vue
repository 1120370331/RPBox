<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { invoke } from '@tauri-apps/api/core'
import { getVersion } from '@tauri-apps/api/app'
import { useUserStore } from '@/stores/user'
import { useToastStore } from '@/stores/toast'
import { useThemeStore, themes, type Theme } from '@/stores/theme'
import { useLocaleStore, type LocaleType } from '@/stores/locale'
import { uploadAvatar } from '@/api/user'
import { useUpdater } from '@/composables/useUpdater'
import { getAddonManifest } from '@/api/addon'
import { bumpImageCacheVersion } from '@/utils/imageCache'
import { buildNameStyle } from '@/utils/userNameStyle'
import AddonUpdateDialog from '@/components/AddonUpdateDialog.vue'
import RModal from '@/components/RModal.vue'

interface WowInstallation {
  path: string
  version: string
  flavor: string
}

const router = useRouter()
const { t, locale } = useI18n()
const userStore = useUserStore()
const toast = useToastStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()
const { checking, updateAvailable, updateInfo, checkForUpdate, downloadAndInstall, downloading, downloadProgress, lastError } = useUpdater()

// 检查是否为 LV3+ 赞助者
const sponsorLevel = computed(() => {
  const level = userStore.user?.sponsor_level
  if (typeof level === 'number') return level
  return userStore.user?.is_sponsor ? 2 : 0
})
const canUseTheme = computed(() => sponsorLevel.value >= 3)

const mounted = ref(false)
const wowPath = ref('')
const detectedPaths = ref<WowInstallation[]>([])
const isScanning = ref(false)
const autoSync = ref(false)
const syncOnStartup = ref(true)
const avatarUploading = ref(false)
const avatarInputRef = ref<HTMLInputElement | null>(null)
const appVersion = ref('0.0.0')
const addonVersion = ref<string | null>(null)
const addonChecking = ref(false)
const addonUpdateDialogRef = ref<InstanceType<typeof AddonUpdateDialog> | null>(null)
const showChangelogModal = ref(false)

const changelogEntries = [
  {
    title: 'RPBox 更新日志 v0.2.7',
    content: `插件更新 V1.0.8
适配了12.0前夕版本
新功能追加
1. 在帖子内加入了"内部链接"功能，可以链接到其他帖子、公会剧情、公会主页
2. 在帖子作品评论区增加了表情包功能，后续会持续更新
3. 增加了公会头像，现在公会可以修改自己的头像了`,
  },
  {
    title: 'RPBox 更新日志 v0.2.2',
    content: `插件更新 V1.0.5
修复 NPC 密语开头乱码与颜色显示问题；清理旁白/无名 NPC 前缀残留符号
新功能追加
1. 剧情归档升级，现在，可以在剧情中插入图片，并且时间轴上可以显示彩色标签；
[图片]

[图片]
优化与修复
1. 修复了编辑道具无法编辑详情的问题
2. 优化了我的帖子、我的作品列表加载速度
3. 邮箱验证已上线，请绑定自己的邮箱！
4. 修复了道具帖子图片的显示问题
5. 修复了记录格式的一些问题`,
  },
  {
    title: 'RPBox 更新日志 v0.2.0-0.2.1',
    content: `新功能追加
1. 新增消息区域， 现在可以看到别人的和你的互动消息了。

2.公会公会社区和公会帖子已上线，在这里，包括访客可以看到所有归档到公会的剧情和帖子（可配置）
现在可以通过将剧情归档到公会来分享给其他人观看，后续将更新独立链接和网页版

3. 道具市场更名创意市场，并添加了新分类画作，支持无损上传图像作品。优化了筛选逻辑。

4. 现在创意市场和社区广场都添加了收藏夹和历史记录，可以看到自己收藏/浏览过的帖子。
优化与修复
1. 修复了社区广场和创意市场的分页逻辑失效问题
2. 增强了RPBox插件对于/表情的采集逻辑，现在会将第一视角 你笑得很开心 替换为 玩家名笑得很开心，请更新RPBOX插件
3. 优化了社区加载速度
4. 添加了RPBox插件使用提示教程在剧情故事标签
5. 美化了剧情故事查看的页面
6. 现在在个人中心可以更改自己的用户名了（重新登录后生效），请记住自己的用户名！`,
  },
]

onMounted(async () => {
  wowPath.value = localStorage.getItem('wow_path') || ''
  autoSync.value = localStorage.getItem('auto_sync') === 'true'
  syncOnStartup.value = localStorage.getItem('sync_on_startup') !== 'false'
  setTimeout(() => mounted.value = true, 50)

  // 获取应用版本
  try {
    appVersion.value = await getVersion()
  } catch (e) {
    console.error('获取版本失败:', e)
  }

  // 检查插件版本
  await checkAddonInstalled()
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
  if (confirm(t('settings.data.confirmClearCache'))) {
    await invoke('clear_sync_cache')
    toast.success(t('settings.data.cacheCleared'))
  }
}

function clearImageCache() {
  if (!confirm(t('settings.data.confirmClearImageCache'))) return
  bumpImageCacheVersion()
  sessionStorage.removeItem('tiptap_image_upload_cache')
  toast.success(t('settings.data.imageCacheCleared'))
}

function resetSetup() {
  localStorage.removeItem('wow_path')
  router.push('/sync/setup')
}

async function handleCheckUpdate() {
  try {
    const update = await checkForUpdate()
    if (update) {
      toast.success(`${t('settings.about.foundNewVersion')} ${update.version}`)
    } else {
      toast.info(t('settings.about.alreadyLatest'))
    }
  } catch (e: any) {
    console.error('[Settings] 检查更新失败:', e)
    const errorMsg = lastError.value || e?.message || 'Unknown error'
    toast.error(`${t('settings.about.checkFailed')}: ${errorMsg}`)
  }
}

async function handleDownloadAndInstall() {
  try {
    toast.info(t('settings.about.startDownload'))
    await downloadAndInstall()
    toast.success(t('settings.about.downloadComplete'))
  } catch (e: any) {
    console.error('[Settings] 下载更新失败:', e)
    const errorMsg = lastError.value || e?.message || 'Unknown error'
    toast.error(`${t('settings.about.downloadFailed')}: ${errorMsg}`)
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
    toast.warning(t('settings.profile.avatarSizeLimit'))
    return
  }

  avatarUploading.value = true
  try {
    const res = await uploadAvatar(file)
    userStore.updateAvatar(res.avatar)
    toast.success(t('settings.profile.avatarUpdated'))
  } catch (error: any) {
    toast.error(error.message || t('settings.profile.uploadFailed'))
  } finally {
    avatarUploading.value = false
    input.value = ''
  }
}

async function checkAddonInstalled() {
  console.log('[Settings] checkAddonInstalled 被调用')
  if (!wowPath.value) {
    addonVersion.value = null
    return
  }

  try {
    // 从 localStorage 读取 flavor，默认使用 _retail_
    const flavor = localStorage.getItem('selected_flavor') || '_retail_'

    const info = await invoke<{ installed: boolean; version?: string }>('check_addon_installed', {
      wowPath: wowPath.value,
      flavor: flavor,
    })
    console.log('[Settings] 检查插件结果:', info)
    addonVersion.value = info.installed ? (info.version || '未知') : null
    console.log('[Settings] 更新后的版本号:', addonVersion.value)
  } catch (e) {
    console.error('检查插件失败:', e)
    addonVersion.value = null
  }
}

async function handleCheckAddonUpdate() {
  console.log('[Settings] 开始检查插件更新, 当前版本:', addonVersion.value)

  if (!addonVersion.value) {
    toast.warning(t('settings.addon.installFirst'))
    return
  }

  addonChecking.value = true
  try {
    console.log('[Settings] 调用 getAddonManifest API...')
    const manifest = await getAddonManifest()
    console.log('[Settings] 获取到 manifest:', manifest)
    const latestVersion = manifest.latest

    if (addonVersion.value === latestVersion) {
      toast.success(t('settings.addon.alreadyLatest'))
    } else {
      console.log('[Settings] 发现新版本:', latestVersion)
      const latestVersionInfo = manifest.versions.find(v => v.version === latestVersion)
      const changelog = latestVersionInfo?.changelog || t('settings.addon.noChangelog')
      const flavor = localStorage.getItem('selected_flavor') || '_retail_'
      console.log('[Settings] 显示更新对话框')
      addonUpdateDialogRef.value?.show(addonVersion.value, latestVersion, changelog, wowPath.value, flavor)
    }
  } catch (e) {
    console.error('检查插件更新失败:', e)
    toast.error(t('settings.addon.checkFailed'))
  } finally {
    addonChecking.value = false
  }
}

// 生成主题预览样式
function getThemePreviewStyle(theme: Theme) {
  return {
    '--preview-sidebar': theme.colors.sidebarBg,
    '--preview-bg': theme.colors.background,
    '--preview-panel': theme.colors.panelBg,
    '--preview-accent': theme.colors.accent,
  }
}

// 语言选项
const languageOptions = [
  { value: 'zh-CN' as LocaleType, label: '简体中文' },
  { value: 'en-US' as LocaleType, label: 'English' },
]

// 切换语言
function changeLanguage(lang: LocaleType) {
  localeStore.setLocale(lang)
  locale.value = lang
}

// 同步 locale store 到 i18n
watch(() => localeStore.currentLocale, (newLocale) => {
  locale.value = newLocale
}, { immediate: true })
</script>

<template>
  <div class="settings-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar anim-item" style="--delay: 0">
      <div class="breadcrumbs">
        <i class="ri-settings-3-line"></i>
        <span class="current">{{ $t('settings.title') }}</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary" @click="router.back()">
          <i class="ri-arrow-left-line"></i> {{ $t('settings.back') }}
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
            <h3>{{ $t('settings.profile.title') }}</h3>
            <p>{{ $t('settings.profile.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <div class="avatar-section">
            <div class="avatar-preview" @click="triggerAvatarUpload">
              <img v-if="userStore.user?.avatar" :src="userStore.user.avatar" alt="avatar" />
              <span v-else class="avatar-placeholder">
                {{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
              </span>
              <div class="avatar-overlay">
                <i :class="avatarUploading ? 'ri-loader-4-line spin' : 'ri-camera-line'"></i>
              </div>
            </div>
            <div class="avatar-info">
              <h4 :style="buildNameStyle(userStore.user?.name_color, userStore.user?.name_bold)">{{ userStore.user?.username || $t('settings.profile.notLoggedIn') }}</h4>
              <p>{{ $t('settings.profile.avatarHint') }}</p>
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

      <!-- 主题设置 -->
      <div class="setting-card anim-item" style="--delay: 1.5">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-palette-line"></i>
          </div>
          <div class="card-title">
            <h3>{{ $t('settings.appearance.theme') }}</h3>
            <p>{{ $t('settings.appearance.themeDesc') }}</p>
          </div>
          <span v-if="!canUseTheme" class="sponsor-badge">
            <i class="ri-vip-crown-line"></i> {{ $t('settings.appearance.sponsorOnly') }}
          </span>
        </div>
        <div class="card-body">
          <div v-if="canUseTheme" class="theme-grid">
            <div
              v-for="theme in themes"
              :key="theme.id"
              class="theme-item"
              :class="{ active: themeStore.currentThemeId === theme.id }"
              @click="themeStore.setTheme(theme.id)"
            >
              <div class="theme-preview" :style="getThemePreviewStyle(theme)">
                <div class="preview-sidebar"></div>
                <div class="preview-content">
                  <div class="preview-header"></div>
                  <div class="preview-card"></div>
                </div>
              </div>
              <span class="theme-name">{{ theme.name }}</span>
              <i v-if="themeStore.currentThemeId === theme.id" class="ri-checkbox-circle-fill theme-check"></i>
            </div>
          </div>
          <div v-else class="theme-locked">
            <div class="locked-content">
              <i class="ri-lock-line"></i>
              <p>{{ $t('settings.appearance.sponsorHint') }}</p>
              <button class="btn btn-outline" @click="router.push('/thanks')">
                <i class="ri-heart-3-line"></i> {{ $t('settings.appearance.learnSponsor') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 语言设置 -->
      <div class="setting-card anim-item" style="--delay: 1.8">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-translate-2"></i>
          </div>
          <div class="card-title">
            <h3>{{ t('settings.language.title') }}</h3>
            <p>{{ t('settings.language.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <div class="language-options">
            <div
              v-for="lang in languageOptions"
              :key="lang.value"
              class="language-item"
              :class="{ active: localeStore.currentLocale === lang.value }"
              @click="changeLanguage(lang.value)"
            >
              <span class="language-label">{{ lang.label }}</span>
              <i v-if="localeStore.currentLocale === lang.value" class="ri-checkbox-circle-fill"></i>
            </div>
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
            <h3>{{ $t('settings.game.title') }}</h3>
            <p>{{ $t('settings.game.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <div class="path-input-group">
            <input
              v-model="wowPath"
              type="text"
              :placeholder="$t('settings.game.wowPathPlaceholder')"
              class="path-input"
            />
            <button class="btn btn-primary" @click="detectPaths" :disabled="isScanning">
              <i :class="isScanning ? 'ri-loader-4-line spin' : 'ri-search-line'"></i>
              {{ isScanning ? $t('settings.game.scanning') : $t('settings.game.autoDetect') }}
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
            <h3>{{ $t('settings.sync.title') }}</h3>
            <p>{{ $t('settings.sync.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <label class="switch-item" @click="syncOnStartup = !syncOnStartup; saveSettings()">
            <div class="switch-info">
              <span class="switch-label">{{ $t('settings.sync.autoSyncOnStartup') }}</span>
              <span class="switch-desc">{{ $t('settings.sync.autoSyncOnStartupDesc') }}</span>
            </div>
            <div class="switch" :class="{ active: syncOnStartup }">
              <div class="switch-thumb"></div>
            </div>
          </label>
          <label class="switch-item" @click="autoSync = !autoSync; saveSettings()">
            <div class="switch-info">
              <span class="switch-label">{{ $t('settings.sync.autoSyncOnChange') }}</span>
              <span class="switch-desc">{{ $t('settings.sync.autoSyncOnChangeDesc') }}</span>
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
            <h3>{{ $t('settings.data.title') }}</h3>
            <p>{{ $t('settings.data.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <div class="action-buttons">
            <button class="btn btn-outline" @click="clearImageCache">
              <i class="ri-brush-line"></i>
              {{ $t('settings.data.clearImageCache') }}
            </button>
            <button class="btn btn-outline" @click="clearCache">
              <i class="ri-delete-bin-line"></i>
              {{ $t('settings.data.clearLocalCache') }}
            </button>
            <button class="btn btn-danger" @click="resetSetup">
              <i class="ri-restart-line"></i>
              {{ $t('settings.data.reconfigure') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 插件管理 -->
      <div class="setting-card anim-item" style="--delay: 4.5">
        <div class="card-header">
          <div class="card-icon">
            <i class="ri-puzzle-line"></i>
          </div>
          <div class="card-title">
            <h3>{{ $t('settings.addon.title') }}</h3>
            <p>{{ $t('settings.addon.description') }}</p>
          </div>
        </div>
        <div class="card-body">
          <div v-if="addonVersion" class="addon-info">
            <div class="addon-status">
              <i class="ri-checkbox-circle-fill status-icon installed"></i>
              <div class="status-text">
                <span class="status-label">{{ $t('settings.addon.installed') }}</span>
                <span class="status-version">v{{ addonVersion }}</span>
              </div>
            </div>
            <button
              class="btn btn-primary"
              @click="handleCheckAddonUpdate"
              :disabled="addonChecking"
            >
              <i :class="addonChecking ? 'ri-loader-4-line spin' : 'ri-refresh-line'"></i>
              {{ addonChecking ? $t('settings.addon.checking') : $t('settings.addon.checkUpdate') }}
            </button>
          </div>
          <div v-else class="addon-info">
            <div class="addon-status">
              <i class="ri-error-warning-fill status-icon not-installed"></i>
              <div class="status-text">
                <span class="status-label">{{ $t('settings.addon.notInstalled') }}</span>
                <span class="status-desc">{{ $t('settings.addon.notInstalledDesc') }}</span>
              </div>
            </div>
            <button class="btn btn-outline" @click="router.push('/archives')">
              <i class="ri-download-line"></i>
              {{ $t('settings.addon.goToInstall') }}
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
            <p class="version">v{{ appVersion }}</p>
            <p class="desc">{{ $t('settings.about.appDesc') }}</p>
          </div>
          <div class="about-actions">
            <button
              v-if="!updateAvailable"
              class="btn btn-about"
              @click="handleCheckUpdate"
              :disabled="checking"
            >
              <i :class="checking ? 'ri-loader-4-line spin' : 'ri-refresh-line'"></i>
              {{ checking ? $t('settings.about.checking') : $t('settings.about.checkUpdate') }}
            </button>
            <template v-else>
              <div class="update-info">
                <span class="new-version">{{ $t('settings.about.newVersion') }} {{ updateInfo?.version }}</span>
              </div>
              <button
                class="btn btn-update"
                @click="handleDownloadAndInstall"
                :disabled="downloading"
              >
                <i :class="downloading ? 'ri-loader-4-line spin' : 'ri-download-line'"></i>
                {{ downloading ? `${$t('settings.about.downloading')} ${Math.round(downloadProgress)}%` : $t('settings.about.downloadUpdate') }}
              </button>
            </template>
          </div>
        </div>
      </div>

      <!-- 其他信息 -->
      <div class="setting-card anim-item" style="--delay: 5.5">
        <div class="action-buttons extra-actions">
          <button class="btn btn-outline" @click="showChangelogModal = true">
            <i class="ri-file-list-3-line"></i>
            {{ $t('settings.about.viewChangelog') }}
          </button>
          <button class="btn btn-outline" @click="router.push('/thanks')">
            <i class="ri-heart-3-line"></i>
            {{ $t('settings.about.thanks') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 插件更新对话框 -->
    <AddonUpdateDialog ref="addonUpdateDialogRef" @installed="checkAddonInstalled" />
    <RModal v-model="showChangelogModal" :title="$t('settings.about.clientChangelog')" width="640px">
      <div class="changelog-modal">
        <div v-for="entry in changelogEntries" :key="entry.title" class="changelog-entry">
          <div class="changelog-header">
            <span class="changelog-tag">{{ entry.title }}</span>
          </div>
          <div class="changelog-content">{{ entry.content }}</div>
        </div>
      </div>
    </RModal>
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
  background-color: var(--color-panel-bg);
  border-radius: 16px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: var(--shadow-md);
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.breadcrumbs i {
  font-size: 18px;
  color: var(--icon-color);
}

.breadcrumbs .current {
  color: var(--icon-color);
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
  background: var(--color-panel-bg);
  border-radius: 16px;
  padding: 24px;
  box-shadow: var(--shadow-md);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border);
}

.card-icon {
  width: 48px;
  height: 48px;
  background: var(--icon-bg);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon i {
  font-size: 24px;
  color: var(--icon-color);
}

.card-title h3 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main);
}

.card-title p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
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
  background: linear-gradient(135deg, var(--color-accent), var(--color-text-secondary));
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
  color: var(--color-text-main);
}

.avatar-info p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

/* 路径输入 */
.path-input-group {
  display: flex;
  gap: 12px;
}

.path-input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid var(--input-border);
  border-radius: 10px;
  background: var(--input-bg);
  color: var(--color-text-main);
  font-size: 14px;
  transition: all 0.2s;
}

.path-input:focus {
  outline: none;
  border-color: var(--input-focus);
  box-shadow: 0 0 0 3px rgba(var(--shadow-base), 0.1);
}

.path-input::placeholder {
  color: var(--input-placeholder);
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
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.path-item:hover {
  background: var(--color-card-bg-hover);
  border-color: var(--color-border-hover);
}

.path-item.selected {
  background: var(--color-primary-light);
  border-color: var(--color-secondary);
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
  color: var(--icon-color);
}

.path-text {
  font-size: 13px;
  color: var(--color-text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.flavor-tag {
  font-size: 12px;
  color: var(--color-text-secondary);
  background: var(--tag-bg);
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
  background: var(--color-card-bg);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 10px;
}

.switch-item:last-child {
  margin-bottom: 0;
}

.switch-item:hover {
  background: var(--color-card-bg-hover);
}

.switch-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.switch-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
}

.switch-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* 开关组件 */
.switch {
  width: 44px;
  height: 24px;
  background: var(--switch-inactive);
  border-radius: 12px;
  position: relative;
  transition: all 0.3s;
  flex-shrink: 0;
}

.switch.active {
  background: var(--switch-active);
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
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
}

.btn-primary:hover:not(:disabled) {
  background: var(--btn-primary-hover);
  box-shadow: 0 4px 12px rgba(var(--shadow-base), 0.3);
}

.btn-secondary {
  background: var(--btn-secondary-bg);
  color: var(--btn-secondary-text);
}

.btn-secondary:hover {
  background: var(--btn-secondary-hover);
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--btn-outline-border);
  color: var(--btn-outline-text);
}

.btn-outline:hover {
  background: var(--btn-outline-hover);
  border-color: var(--color-border-hover);
}

.btn-danger {
  background: var(--btn-danger-bg);
  color: #FFFFFF;
}

.btn-danger:hover {
  background: var(--btn-danger-hover);
  box-shadow: 0 4px 12px rgba(220, 53, 69, 0.3);
}

/* 关于卡片 */
.about-card {
  background: linear-gradient(135deg, var(--gradient-start) 0%, var(--gradient-end) 100%);
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
  background: var(--color-accent);
  color: var(--color-text-main);
}

.btn-update:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.update-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.new-version {
  font-size: 13px;
  color: var(--color-accent);
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

/* 底部按钮区 */
.extra-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

/* 更新日志弹窗 */
.changelog-modal {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.changelog-entry {
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 16px;
}

.changelog-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.changelog-tag {
  background: var(--badge-bg);
  color: #FFFFFF;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 999px;
}

.changelog-content {
  color: var(--color-primary);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
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

/* 插件管理样式 */
.addon-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: var(--color-card-bg);
  border-radius: 10px;
}

.addon-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.status-icon.installed {
  color: #4CAF50;
}

.status-icon.not-installed {
  color: #FF9800;
}

.status-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.status-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-main);
}

.status-version {
  font-size: 13px;
  color: var(--icon-color);
  font-weight: 500;
}

.status-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* 主题设置样式 */
.sponsor-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: linear-gradient(135deg, #FFD700, #FFA500);
  color: #4B3621;
  font-size: 11px;
  font-weight: 600;
  border-radius: 999px;
  margin-left: auto;
}

.sponsor-badge i {
  font-size: 12px;
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 16px;
}

.theme-item {
  position: relative;
  cursor: pointer;
  border-radius: 12px;
  padding: 8px;
  background: #FDFBF9;
  border: 2px solid transparent;
  transition: all 0.2s;
}

.theme-item:hover {
  background: rgba(128, 64, 48, 0.05);
  border-color: var(--color-border);
}

.theme-item.active {
  border-color: var(--color-secondary);
  background: rgba(128, 64, 48, 0.08);
}

.theme-preview {
  width: 100%;
  aspect-ratio: 16 / 10;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  background: var(--preview-bg);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.preview-sidebar {
  width: 25%;
  background: var(--preview-sidebar);
}

.preview-content {
  flex: 1;
  padding: 8%;
  display: flex;
  flex-direction: column;
  gap: 8%;
}

.preview-header {
  height: 15%;
  background: var(--preview-accent);
  border-radius: 2px;
  opacity: 0.8;
}

.preview-card {
  flex: 1;
  background: var(--preview-panel);
  border-radius: 3px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.theme-name {
  display: block;
  text-align: center;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-main);
}

.theme-check {
  position: absolute;
  top: 12px;
  right: 12px;
  font-size: 18px;
  color: var(--color-secondary);
}

.theme-locked {
  padding: 32px 16px;
  text-align: center;
}

.locked-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.locked-content i {
  font-size: 40px;
  color: #D4D4D4;
}

.locked-content p {
  font-size: 14px;
  color: var(--color-text-secondary);
  max-width: 280px;
}

/* 语言设置样式 */
.language-options {
  display: flex;
  gap: 12px;
}

.language-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--color-card-bg);
  border: 2px solid var(--color-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.language-item:hover {
  background: var(--color-card-bg-hover);
  border-color: var(--color-border-hover);
}

.language-item.active {
  border-color: var(--color-secondary);
  background: var(--color-primary-light);
}

.language-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
}

.language-item i {
  font-size: 18px;
  color: var(--color-secondary);
}
</style>
