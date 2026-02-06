<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getGuild, leaveGuild, deleteGuild, listGuildMembers, updateGuild, uploadGuildBanner, uploadGuildAvatar, listGuildApplications, applyGuild, type Guild, type GuildMember } from '@/api/guild'
import { getImageUrl } from '@/api/item'
import { useDialog } from '@/composables/useDialog'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import LazyBgImage from '@/components/LazyBgImage.vue'
import { buildNameStyle } from '@/utils/userNameStyle'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { confirm, alert } = useDialog()

const loading = ref(true)
const guild = ref<Guild | null>(null)
const myRole = ref('')
const members = ref<GuildMember[]>([])
const pendingApplicationCount = ref(0)

// 设置相关
const showSettingsModal = ref(false)
const showLoreModal = ref(false)
const saving = ref(false)
const editForm = ref({
  name: '',
  slogan: '',
  description: '',
  lore: '',
  faction: '',
  color: '',
  visitor_can_view_stories: false,
  visitor_can_view_posts: false,
  member_can_view_stories: true,
  member_can_view_posts: true,
  auto_approve: false
})
const bannerFile = ref<File | null>(null)
const bannerPreview = ref('')
const heroBannerInput = ref<HTMLInputElement | null>(null)
const avatarFile = ref<File | null>(null)
const avatarPreview = ref('')
const heroAvatarInput = ref<HTMLInputElement | null>(null)

const guildId = Number(route.params.id)

const isAdmin = computed(() => myRole.value === 'owner' || myRole.value === 'admin')
const guildAvatarUrl = computed(() => {
  if (!guild.value || (!guild.value.avatar_url && !guild.value.avatar)) return ''
  return getImageUrl('guild-avatar', guild.value.id, {
    w: 160,
    q: 80,
    v: guild.value.avatar_updated_at || guild.value.updated_at,
  })
})

async function loadGuild() {
  loading.value = true
  try {
    const res = await getGuild(guildId)
    guild.value = res.guild
    myRole.value = res.my_role

    // 只有成员才能查看成员列表
    if (myRole.value) {
      const membersRes = await listGuildMembers(guildId)
      members.value = membersRes.members || []
    }

    // 如果是管理员，加载待处理申请数量
    if (myRole.value === 'owner' || myRole.value === 'admin') {
      const appsRes = await listGuildApplications(guildId, 'pending')
      pendingApplicationCount.value = appsRes.applications?.length || 0
    }
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    loading.value = false
  }
}

async function handleLeave() {
  const confirmed = await confirm({
    title: t('guild.leave.title'),
    message: t('guild.leave.message'),
    type: 'warning'
  })
  if (!confirmed) return
  try {
    await leaveGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    await alert({ title: t('guild.leave.failed'), message: e.message || t('guild.leave.failed'), type: 'error' })
  }
}

async function handleApply() {
  try {
    const res = await applyGuild(guildId)
    // 检查是否为自动加入
    if ((res as any).auto_approved) {
      await alert({
        title: t('guild.apply.joinSuccess'),
        message: t('guild.apply.joinSuccessMessage'),
        type: 'success'
      })
    } else {
      await alert({
        title: t('guild.apply.submitted'),
        message: t('guild.apply.submittedMessage'),
        type: 'success'
      })
    }
    await loadGuild() // 重新加载公会信息
  } catch (e: any) {
    await alert({ title: t('guild.apply.failed'), message: e.message || t('guild.apply.failed'), type: 'error' })
  }
}

async function handleDelete() {
  const confirmed = await confirm({
    title: t('guild.disband.title'),
    message: t('guild.disband.message'),
    type: 'error'
  })
  if (!confirmed) return
  try {
    await deleteGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    await alert({ title: t('guild.disband.failed'), message: e.message || t('guild.disband.failed'), type: 'error' })
  }
}

function openSettings() {
  if (!guild.value) return
  editForm.value = {
    name: guild.value.name,
    slogan: guild.value.slogan || '',
    description: guild.value.description || '',
    lore: guild.value.lore || '',
    faction: guild.value.faction || '',
    color: guild.value.color || 'B87333',
    visitor_can_view_stories: guild.value.visitor_can_view_stories ?? true,
    visitor_can_view_posts: guild.value.visitor_can_view_posts ?? true,
    member_can_view_stories: guild.value.member_can_view_stories ?? true,
    member_can_view_posts: guild.value.member_can_view_posts ?? true,
    auto_approve: guild.value.auto_approve ?? false
  }
  bannerPreview.value = guild.value.banner || ''
  bannerFile.value = null
  avatarPreview.value = guildAvatarUrl.value || ''
  avatarFile.value = null
  showSettingsModal.value = true
}

function handleBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    const file = input.files[0]
    if (file.size > 20 * 1024 * 1024) {
      alert({ title: t('guild.settings.fileTooLarge'), message: t('guild.settings.bannerSizeLimit'), type: 'error' })
      return
    }
    bannerFile.value = file
    const reader = new FileReader()
    reader.onload = (e) => {
      bannerPreview.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

function handleAvatarSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    const file = input.files[0]
    if (file.size > 10 * 1024 * 1024) {
      alert({ title: t('guild.settings.fileTooLarge'), message: t('guild.settings.avatarSizeLimit'), type: 'error' })
      return
    }
    avatarFile.value = file
    const reader = new FileReader()
    reader.onload = (e) => {
      avatarPreview.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

function triggerBannerUpload() {
  heroBannerInput.value?.click()
}

function triggerAvatarUpload() {
  if (!isAdmin.value) return
  heroAvatarInput.value?.click()
}

async function handleHeroBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || !input.files[0] || !guild.value) return

  const file = input.files[0]
  if (file.size > 20 * 1024 * 1024) {
    await alert({ title: t('guild.settings.fileTooLarge'), message: t('guild.settings.bannerSizeLimit'), type: 'error' })
    return
  }

  try {
    const res = await uploadGuildBanner(guildId, file)
    guild.value.banner = res.banner
  } catch (e: any) {
    await alert({ title: t('guild.settings.uploadFailed'), message: e.message || t('guild.settings.uploadFailed'), type: 'error' })
  }
  input.value = ''
}

async function handleHeroAvatarSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || !input.files[0] || !guild.value) return

  const file = input.files[0]
  if (file.size > 10 * 1024 * 1024) {
    await alert({ title: t('guild.settings.fileTooLarge'), message: t('guild.settings.avatarSizeLimit'), type: 'error' })
    return
  }

  try {
    const res = await uploadGuildAvatar(guildId, file)
    guild.value.avatar = res.avatar
    if (res.avatar_updated_at) {
      guild.value.avatar_updated_at = res.avatar_updated_at
    }
  } catch (e: any) {
    await alert({ title: t('guild.settings.uploadFailed'), message: e.message || t('guild.settings.uploadFailed'), type: 'error' })
  }
  input.value = ''
}

async function saveSettings() {
  if (!guild.value) return
  saving.value = true
  try {
    if (avatarFile.value) {
      const res = await uploadGuildAvatar(guildId, avatarFile.value)
      guild.value.avatar = res.avatar
      if (res.avatar_updated_at) {
        guild.value.avatar_updated_at = res.avatar_updated_at
      }
    }
    // 如果有新头图，先上传
    if (bannerFile.value) {
      const res = await uploadGuildBanner(guildId, bannerFile.value)
      guild.value.banner = res.banner
    }
    // 更新其他信息
    await updateGuild(guildId, {
      name: editForm.value.name,
      slogan: editForm.value.slogan,
      description: editForm.value.description,
      lore: editForm.value.lore,
      faction: editForm.value.faction,
      color: editForm.value.color.replace('#', ''),
      visitor_can_view_stories: editForm.value.visitor_can_view_stories,
      visitor_can_view_posts: editForm.value.visitor_can_view_posts,
      member_can_view_stories: editForm.value.member_can_view_stories,
      member_can_view_posts: editForm.value.member_can_view_posts,
      auto_approve: editForm.value.auto_approve
    })
    // 刷新数据
    await loadGuild()
    showSettingsModal.value = false
  } catch (e: any) {
    await alert({ title: t('guild.settings.saveFailed'), message: e.message || t('guild.settings.saveFailed'), type: 'error' })
  } finally {
    saving.value = false
  }
}

function getRoleLabel(role: string): string {
  if (!role) return ''
  return t(`guild.role.${role}`)
}

function getFactionLabel(f: string): string {
  if (!f) return ''
  return t(`guild.info.${f}`)
}

function goToEvents() {
  // 跳转到公会帖子页面
  router.push({ name: 'guild-posts', params: { id: guildId } })
}

function goToStories() {
  // 跳转到公会剧情页面
  router.push({ name: 'guild-stories', params: { id: guildId } })
}

onMounted(loadGuild)
</script>

<template>
  <div class="guild-detail">
    <div v-if="loading" class="loading">{{ t('guild.loading') }}</div>

    <template v-else-if="guild">
      <!-- 主内容区 -->
      <div class="main-content">
        <!-- 顶部导航 -->
        <header class="top-bar">
          <div class="breadcrumb">
            <span class="path">RPBox</span>
            <span class="sep">/</span>
            <span class="current">{{ guild.name }}</span>
          </div>
          <div class="top-actions">
            <button class="icon-btn" @click="loadGuild"><i class="ri-refresh-line"></i></button>
            <button v-if="isAdmin" class="primary-btn manage-btn" @click="router.push(`/guild/${guildId}/manage`)">
              <i class="ri-team-line"></i> {{ t('guild.action.manageGuild') }}
              <span v-if="pendingApplicationCount > 0" class="pending-badge">{{ pendingApplicationCount }}</span>
            </button>
            <button v-if="isAdmin" class="primary-btn" @click="openSettings">
              <i class="ri-settings-3-line"></i> {{ t('guild.action.settings') }}
            </button>
          </div>
        </header>

        <!-- 滚动内容区 -->
        <div class="scroll-content">
          <!-- Hero 区域 -->
          <div class="hero-section">
            <div class="hero-card">
              <!-- 背景图 -->
              <LazyBgImage
                class="hero-bg"
                :src="guild.banner"
                :fallback-gradient="`linear-gradient(135deg, #${guild.color || 'B87333'}, #4B3621)`"
              >
                <div class="hero-overlay"></div>
              </LazyBgImage>

              <!-- 内容 -->
              <div class="hero-content">
                <div class="content-box">
                  <div class="guild-avatar" :class="{ editable: isAdmin }" @click="triggerAvatarUpload">
                    <img v-if="guildAvatarUrl" :src="guildAvatarUrl" alt="" />
                    <span v-else>{{ guild.name.charAt(0) }}</span>
                    <div v-if="isAdmin" class="avatar-edit">
                      <i class="ri-camera-line"></i>
                    </div>
                  </div>
                  <div class="badges">
                    <span v-if="guild.faction" class="badge faction" :class="guild.faction">
                      {{ getFactionLabel(guild.faction) }}
                    </span>
                    <span class="badge members">
                      <i class="ri-user-line"></i> {{ guild.member_count }} {{ t('guild.info.members') }}
                    </span>
                  </div>
                  <h1>{{ guild.name }}</h1>
                  <p class="slogan">{{ guild.slogan || guild.description || t('guild.info.noDescription') }}</p>

                  <div class="hero-actions">
                    <div v-if="!myRole" class="apply-action">
                      <button class="btn-outline" @click="handleApply">
                        {{ t('guild.action.applyJoin') }}
                      </button>
                      <span v-if="guild.auto_approve" class="auto-approve-hint">
                        <i class="ri-check-line"></i> {{ t('guild.detail.autoApprove') }}
                      </span>
                    </div>
                    <button v-else-if="myRole !== 'owner'" class="btn-outline" @click="handleLeave">
                      {{ t('guild.action.leave') }}
                    </button>
                    <button v-if="isAdmin" class="btn-outline" @click="triggerBannerUpload">
                      <i class="ri-image-edit-line"></i> {{ t('guild.action.changeBanner') }}
                    </button>
                  </div>
                  <input ref="heroBannerInput" type="file" accept="image/*" hidden @change="handleHeroBannerSelect" />
                  <input ref="heroAvatarInput" type="file" accept="image/*" hidden @change="handleHeroAvatarSelect" />
                </div>
              </div>
            </div>

            <!-- 侧边公告卡片 -->
            <div class="announcement-card">
              <div class="card-header">
                <h3>{{ t('guild.info.guildInfo') }}</h3>
                <span class="tag">INFO</span>
              </div>
              <ul class="info-list">
                <li v-if="isAdmin">
                  <span class="label">{{ t('guild.info.inviteCode') }}</span>
                  <span class="value code">{{ guild.invite_code }}</span>
                </li>
                <li>
                  <span class="label">{{ t('guild.info.storyCount') }}</span>
                  <span class="value">{{ guild.story_count }}</span>
                </li>
                <li v-if="myRole">
                  <span class="label">{{ t('guild.info.myRole') }}</span>
                  <span class="value role">{{ getRoleLabel(myRole) }}</span>
                </li>
              </ul>
            </div>
          </div>

          <!-- Bento Grid -->
          <div class="bento-grid">
            <!-- 公会介绍 -->
            <div class="bento-card desc-card">
              <div class="card-icon"><i class="ri-file-text-line"></i></div>
              <h3>{{ t('guild.detail.guildIntro') }}</h3>
              <p>{{ guild.description || t('guild.detail.noIntro') }}</p>
            </div>

            <!-- 公会设定 -->
            <div class="bento-card lore-card" :class="{ clickable: guild.lore }" @click="guild.lore && (showLoreModal = true)">
              <div class="card-icon"><i class="ri-quill-pen-line"></i></div>
              <h3>{{ t('guild.detail.guildLore') }} <i v-if="guild.lore" class="ri-arrow-right-s-line"></i></h3>
              <div v-if="guild.lore" class="lore-preview" v-html="guild.lore"></div>
              <p v-else class="empty-hint">{{ t('guild.detail.noLore') }}</p>
            </div>

            <!-- 成员列表 -->
            <div v-if="myRole" class="bento-card members-card">
              <div class="card-header">
                <h3>{{ t('guild.detail.memberList') }}</h3>
                <span class="count">{{ members.length }}{{ t('guild.detail.memberCountUnit') }}</span>
              </div>
              <div class="member-list">
                <div v-for="m in members" :key="m.id" class="member-item">
                  <div class="avatar">
                    <img v-if="m.avatar" :src="m.avatar" alt="" loading="lazy" />
                    <span v-else>{{ m.username?.charAt(0) || '?' }}</span>
                  </div>
                  <div class="info">
                    <span class="name" :style="buildNameStyle(m.name_color, m.name_bold)">{{ m.username }}</span>
                    <span class="role" :class="m.role">{{ getRoleLabel(m.role) }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 快捷操作 -->
            <div class="bento-card actions-card">
              <h3>{{ t('guild.detail.quickActions') }}</h3>
              <div class="quick-actions">
                <button class="action-btn" @click="goToEvents">
                  <i class="ri-calendar-event-line"></i>
                  <span>{{ t('guild.detail.guildPosts') }}</span>
                </button>
                <button class="action-btn" @click="goToStories">
                  <i class="ri-book-2-line"></i>
                  <span>{{ t('guild.detail.pastStories') }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 公会设定详情弹窗 -->
    <RModal v-model="showLoreModal" :title="t('guild.detail.guildLore')" width="800px">
      <div class="article-content" v-html="guild?.lore"></div>
    </RModal>

    <!-- 设置弹窗 -->
    <RModal v-model="showSettingsModal" :title="t('guild.settings.title')" width="560px">
      <div class="settings-form">
        <!-- 头像上传 -->
        <div class="form-section">
          <label>{{ t('guild.settings.avatar') }}</label>
          <div
            class="avatar-upload"
            @click="($refs.avatarInput as HTMLInputElement).click()"
          >
            <img v-if="avatarPreview" :src="avatarPreview" alt="" />
            <div v-else class="upload-hint">
              <i class="ri-camera-line"></i>
              <span>{{ t('guild.settings.uploadAvatar') }}</span>
            </div>
          </div>
          <input ref="avatarInput" type="file" accept="image/*" hidden @change="handleAvatarSelect" />
        </div>

        <!-- 头图上传 -->
        <div class="form-section">
          <label>{{ t('guild.settings.banner') }}</label>
          <div
            class="banner-upload"
            :style="bannerPreview
              ? { backgroundImage: `url(${bannerPreview})` }
              : { background: `linear-gradient(135deg, #${editForm.color || 'B87333'}, #4B3621)` }"
            @click="($refs.bannerInput as HTMLInputElement).click()"
          >
            <div class="upload-hint">
              <i class="ri-image-add-line"></i>
              <span>{{ t('guild.settings.uploadBanner') }}</span>
            </div>
          </div>
          <input ref="bannerInput" type="file" accept="image/*" hidden @change="handleBannerSelect" />
        </div>

        <!-- 基本信息 -->
        <div class="form-section">
          <label>{{ t('guild.settings.name') }}</label>
          <RInput v-model="editForm.name" :placeholder="t('guild.settings.namePlaceholder')" />
        </div>

        <div class="form-section">
          <label>{{ t('guild.settings.slogan') }}</label>
          <RInput v-model="editForm.slogan" :placeholder="t('guild.settings.sloganPlaceholder')" />
        </div>

        <div class="form-section">
          <label>{{ t('guild.settings.intro') }}</label>
          <textarea v-model="editForm.description" :placeholder="t('guild.settings.introPlaceholder')" rows="3"></textarea>
        </div>

        <div class="form-row">
          <div class="form-section">
            <label>{{ t('guild.info.faction') }}</label>
            <select v-model="editForm.faction">
              <option value="">{{ t('guild.settings.selectFaction') }}</option>
              <option value="alliance">{{ t('guild.info.alliance') }}</option>
              <option value="horde">{{ t('guild.info.horde') }}</option>
              <option value="neutral">{{ t('guild.info.neutral') }}</option>
            </select>
          </div>
          <div class="form-section">
            <label>{{ t('guild.settings.themeColor') }}</label>
            <input
              type="color"
              :value="'#' + editForm.color"
              @input="editForm.color = ($event.target as HTMLInputElement).value.replace('#', '')"
            />
          </div>
        </div>

        <div class="form-section lore-section">
          <label>{{ t('guild.settings.lore') }}</label>
          <TiptapEditor v-model="editForm.lore" :placeholder="t('guild.settings.lorePlaceholder')" />
        </div>

        <!-- 隐私设置 -->
        <div class="form-section privacy-section">
          <label>{{ t('guild.privacy.title') }}</label>
          <div class="privacy-toggles">
            <div class="toggle-group">
              <div class="toggle-group-label">{{ t('guild.privacy.visitorVisible') }}</div>
              <div class="toggle-item">
                <div class="toggle-info">
                  <span class="toggle-label">{{ t('guild.privacy.storyArchive') }}</span>
                  <span class="toggle-hint">{{ t('guild.privacy.visitorStoryHint') }}</span>
                </div>
                <label class="switch">
                  <input type="checkbox" v-model="editForm.visitor_can_view_stories" />
                  <span class="slider"></span>
                </label>
              </div>
              <div class="toggle-item">
                <div class="toggle-info">
                  <span class="toggle-label">{{ t('guild.privacy.guildPosts') }}</span>
                  <span class="toggle-hint">{{ t('guild.privacy.visitorPostHint') }}</span>
                </div>
                <label class="switch">
                  <input type="checkbox" v-model="editForm.visitor_can_view_posts" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
            <div class="toggle-group">
              <div class="toggle-group-label">{{ t('guild.privacy.memberVisible') }}</div>
              <div class="toggle-item">
                <div class="toggle-info">
                  <span class="toggle-label">{{ t('guild.privacy.storyArchive') }}</span>
                  <span class="toggle-hint">{{ t('guild.privacy.memberStoryHint') }}</span>
                </div>
                <label class="switch">
                  <input type="checkbox" v-model="editForm.member_can_view_stories" />
                  <span class="slider"></span>
                </label>
              </div>
              <div class="toggle-item">
                <div class="toggle-info">
                  <span class="toggle-label">{{ t('guild.privacy.guildPosts') }}</span>
                  <span class="toggle-hint">{{ t('guild.privacy.memberPostHint') }}</span>
                </div>
                <label class="switch">
                  <input type="checkbox" v-model="editForm.member_can_view_posts" />
                  <span class="slider"></span>
                </label>
              </div>
            </div>
            <div class="toggle-item standalone">
              <div class="toggle-info">
                <span class="toggle-label">{{ t('guild.privacy.autoApprove') }}</span>
                <span class="toggle-hint">{{ t('guild.privacy.autoApproveHint') }}</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="editForm.auto_approve" />
                <span class="slider"></span>
              </label>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <RButton @click="showSettingsModal = false">{{ t('guild.action.cancel') }}</RButton>
        <RButton type="primary" :loading="saving" @click="saveSettings">{{ t('guild.action.save') }}</RButton>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.guild-detail {
  min-height: 100vh;
  background: var(--color-background, #EED9C4);
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  font-size: 16px;
  color: var(--color-text-secondary, #8C7B70);
}

.main-content {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* Top Bar */
.top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--color-border, #E8DCCF);
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.breadcrumb .path {
  color: var(--color-text-secondary, #8C7B70);
}

.breadcrumb .sep {
  color: var(--color-accent, #D4A373);
}

.breadcrumb .current {
  color: var(--color-text-main, #2C1810);
  font-weight: 600;
}

.top-actions {
  display: flex;
  gap: 12px;
}

.icon-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary, #8C7B70);
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: all 0.2s;
}

.icon-btn:hover {
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
  color: var(--color-secondary, #804030);
}

.primary-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--color-secondary, #804030);
  color: var(--color-text-light, #fff);
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.primary-btn:hover {
  background: var(--color-secondary-hover, #6B3626);
}

.manage-btn {
  position: relative;
}

.pending-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  background: #FF9800;
  color: var(--color-text-light, #fff);
  font-size: 11px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

/* Scroll Content */
.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

/* Hero Section */
.hero-section {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
  margin-bottom: 24px;
}

.hero-card {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  min-height: 280px;
}

.hero-bg {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(44, 24, 16, 0.9), rgba(44, 24, 16, 0.4), transparent);
}

.hero-content {
  position: relative;
  z-index: 10;
  height: 100%;
  display: flex;
  align-items: flex-end;
  padding: 32px;
}

.content-box {
  max-width: 500px;
}

.guild-avatar {
  width: 72px;
  height: 72px;
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 16px;
  position: relative;
  border: 2px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 6px 16px rgba(44, 24, 16, 0.25);
}

.guild-avatar.editable {
  cursor: pointer;
}

.guild-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-edit {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  color: #fff;
  font-size: 18px;
}

.guild-avatar.editable:hover .avatar-edit {
  opacity: 1;
}

.badges {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.badge {
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
}

.badge.faction {
  background: #D4A373;
  color: #4B3621;
}

.badge.faction.alliance {
  background: #3b82f6;
  color: #fff;
}

.badge.faction.horde {
  background: #dc2626;
  color: #fff;
}

.badge.members {
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  display: flex;
  align-items: center;
  gap: 4px;
}

.hero-content h1 {
  font-size: 36px;
  font-weight: 700;
  color: #fff;
  margin: 0 0 8px 0;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.slogan {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.8);
  margin: 0 0 20px 0;
  line-height: 1.6;
}

.hero-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.apply-action {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auto-approve-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
}

.auto-approve-hint i {
  font-size: 14px;
}

.btn-outline {
  padding: 10px 20px;
  background: rgba(0, 0, 0, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.4);
  color: #fff;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.btn-outline:hover {
  background: rgba(0, 0, 0, 0.65);
}

.btn-danger {
  padding: 10px 20px;
  background: #dc2626;
  border: none;
  color: #fff;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-danger:hover {
  background: #b91c1c;
}

/* Announcement Card */
.announcement-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  padding: 24px;
  box-shadow: var(--shadow-md, 0 4px 12px rgba(44, 24, 16, 0.08));
}

.announcement-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.announcement-card h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  margin: 0;
}

.announcement-card .tag {
  font-size: 10px;
  font-weight: 700;
  color: var(--color-secondary, #804030);
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
  padding: 4px 8px;
  border-radius: 4px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-list li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
}

.info-list .label {
  color: var(--color-text-secondary, #8C7B70);
}

.info-list .value {
  color: var(--color-text-main, #2C1810);
  font-weight: 500;
}

.info-list .value.code {
  font-family: monospace;
  background: var(--color-card-bg, #f5f0eb);
  padding: 4px 8px;
  border-radius: 4px;
}

.info-list .value.role {
  color: var(--color-secondary, #804030);
}

/* Bento Grid */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.bento-card {
  background: var(--color-panel-bg, #fff);
  border-radius: 16px;
  padding: 24px;
  box-shadow: var(--shadow-md, 0 4px 12px rgba(44, 24, 16, 0.08));
}

.bento-card .card-icon {
  width: 40px;
  height: 40px;
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-secondary, #804030);
  font-size: 20px;
  margin-bottom: 16px;
}

.bento-card h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  margin: 0 0 12px 0;
}

.bento-card p {
  font-size: 14px;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.6;
  margin: 0;
}

/* Lore Card */
.lore-card {
  grid-column: span 2;
}

.lore-content {
  font-size: 14px;
  color: var(--color-primary, #4B3621);
  line-height: 1.8;
  max-height: 200px;
  overflow-y: auto;
}

.lore-content :deep(p) {
  margin: 0 0 12px 0;
}

.lore-content :deep(p:last-child) {
  margin-bottom: 0;
}

.empty-hint {
  color: var(--color-text-secondary, #8C7B70);
  font-style: italic;
}

/* Clickable Card */
.bento-card.clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.bento-card.clickable:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg, 0 8px 24px rgba(44, 24, 16, 0.12));
}

.bento-card.clickable h3 i {
  opacity: 0;
  transition: opacity 0.2s;
}

.bento-card.clickable:hover h3 i {
  opacity: 1;
}

/* Lore Preview */
.lore-preview {
  font-size: 14px;
  color: var(--color-primary, #4B3621);
  line-height: 1.8;
  max-height: 120px;
  overflow: hidden;
  position: relative;
}

.lore-preview::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(transparent, var(--color-panel-bg, #fff));
}

.lore-preview :deep(p) {
  margin: 0 0 8px 0;
}

.lore-preview :deep(img) {
  display: none;
}

/* Members Card */
.members-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.members-card .count {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.member-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.member-item .avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-accent, #B87333), var(--color-primary, #4B3621));
  color: var(--color-text-light, #fff);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
}

.member-item .avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-item .info {
  flex: 1;
}

.member-item .name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main, #2C1810);
}

.member-item .role {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.member-item .role.owner {
  color: var(--color-accent, #D4A373);
}

.member-item .role.admin {
  color: #3b82f6;
}

/* Actions Card */
.actions-card h3 {
  margin-bottom: 16px;
}

.quick-actions {
  display: flex;
  gap: 12px;
}

.action-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: var(--color-card-bg, #f5f0eb);
  border: none;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--color-border, #ebe3db);
}

.action-btn i {
  font-size: 24px;
  color: var(--color-secondary, #804030);
}

.action-btn span {
  font-size: 12px;
  color: var(--color-text-main, #2C1810);
}

/* Settings Form */
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-section label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main, #2C1810);
}

.form-section textarea {
  padding: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.form-section textarea:focus {
  outline: none;
  border-color: var(--color-secondary, #804030);
}

.form-section select {
  padding: 12px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 8px;
  font-size: 14px;
  background: var(--color-panel-bg, #fff);
}

.form-section select:focus {
  outline: none;
  border-color: var(--color-secondary, #804030);
}

.form-section input[type="color"] {
  width: 60px;
  height: 36px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 8px;
  cursor: pointer;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

/* Lore Section */
.lore-section {
  margin-top: 8px;
}

.lore-section :deep(.tiptap-editor) {
  min-height: 200px;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 8px;
}

.lore-section :deep(.tiptap-editor:focus-within) {
  border-color: var(--color-secondary, #804030);
}

/* Banner Upload */
.banner-upload {
  height: 160px;
  border-radius: 12px;
  background-size: cover;
  background-position: center;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  border: 2px dashed var(--color-border, #E8DCCF);
  transition: all 0.2s;
}

.banner-upload:hover {
  border-color: var(--color-secondary, #804030);
}

.upload-hint {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(0, 0, 0, 0.4);
  color: var(--color-text-light, #fff);
  opacity: 0;
  transition: opacity 0.2s;
}

.banner-upload:hover .upload-hint {
  opacity: 1;
}

.upload-hint i {
  font-size: 32px;
}

.upload-hint span {
  font-size: 14px;
}

/* Avatar Upload */
.avatar-upload {
  width: 120px;
  height: 120px;
  border-radius: 12px;
  background: var(--color-card-bg, #FBF5EF);
  cursor: pointer;
  position: relative;
  overflow: hidden;
  border: 2px dashed var(--color-border, #E8DCCF);
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-upload:hover {
  border-color: var(--color-secondary, #804030);
}

.avatar-upload img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-upload .upload-hint {
  opacity: 1;
  background: var(--color-primary-light, rgba(128, 64, 48, 0.08));
  color: var(--color-secondary, #804030);
}

.avatar-upload img + .upload-hint {
  opacity: 0;
  background: rgba(0, 0, 0, 0.4);
  color: var(--color-text-light, #fff);
}

.avatar-upload:hover img + .upload-hint {
  opacity: 1;
}

/* Article Content - 富文本渲染 */
.article-content {
  font-size: 15px;
  line-height: 1.8;
  color: var(--color-text-main, #2C1810);
}

.article-content :deep(h1),
.article-content :deep(h2),
.article-content :deep(h3) {
  margin: 24px 0 12px 0;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
}

.article-content :deep(h1) { font-size: 24px; }
.article-content :deep(h2) { font-size: 20px; }
.article-content :deep(h3) { font-size: 18px; }

.article-content :deep(p) {
  margin: 0 0 16px 0;
}

.article-content :deep(ul),
.article-content :deep(ol) {
  margin: 0 0 16px 0;
  padding-left: 24px;
}

.article-content :deep(li) {
  margin-bottom: 8px;
}

.article-content :deep(blockquote) {
  margin: 16px 0;
  padding: 12px 20px;
  border-left: 4px solid var(--color-accent, #D4A373);
  background: var(--color-card-bg, #f5f0eb);
  color: var(--color-primary, #4B3621);
}

.article-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 16px 0;
}

.article-content :deep(a) {
  color: var(--color-secondary, #804030);
  text-decoration: underline;
}

.article-content :deep(code) {
  background: var(--color-card-bg, #f5f0eb);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
}

.article-content :deep(pre) {
  background: var(--color-text-main, #2C1810);
  color: var(--color-background, #EED9C4);
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 16px 0;
}

.article-content :deep(pre code) {
  background: transparent;
  padding: 0;
}

.article-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border, #E8DCCF);
  margin: 24px 0;
}

/* Privacy Settings */
.privacy-section {
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border, #E8DCCF);
}

.privacy-toggles {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toggle-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--color-card-bg, #f5f0eb);
  border-radius: 8px;
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toggle-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main, #2C1810);
}

.toggle-hint {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

/* Toggle Group */
.toggle-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: var(--color-card-bg, #f9f6f3);
  border-radius: 12px;
}

.toggle-group-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-secondary, #804030);
  margin-bottom: 4px;
}

.toggle-group .toggle-item {
  background: var(--color-panel-bg, #fff);
}

.toggle-item.standalone {
  margin-top: 4px;
}

/* Switch Toggle */
.switch {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--color-border, #E8DCCF);
  transition: 0.3s;
  border-radius: 26px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

input:checked + .slider {
  background-color: var(--color-secondary, #804030);
}

input:checked + .slider:before {
  transform: translateX(22px);
}
</style>
