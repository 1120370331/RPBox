<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '../../stores/user'
import { useToastStore } from '../../stores/toast'
import { uploadAvatar, bindEmail, type UserInfo } from '../../api/user'
import { sendVerificationCode } from '../../api/auth'
import request from '../../api/request'
import RColorPicker from '@/components/RColorPicker.vue'
import RCheckbox from '@/components/RCheckbox.vue'
import ImageCropperDialog from '@/components/ImageCropperDialog.vue'
import RModal from '@/components/RModal.vue'
import UserLevelBadge from '@/components/UserLevelBadge.vue'
import AchievementMedal from '@/components/AchievementMedal.vue'
import RPDBContributionSection from '@/components/rpdb/RPDBContributionSection.vue'
import CharacterCardWall from '@/components/character-cards/CharacterCardWall.vue'
import { buildForumLevelGuide, computeLevelProgressPercent } from '@/utils/forumLevel'
import { buildNameStyle } from '@/utils/userNameStyle'
import {
  ACHIEVEMENT_CATEGORY_META,
  ACHIEVEMENT_RARITY_META,
  type AchievementDefinition,
} from '@/data/achievements'
import {
  buildAchievementEntries,
  buildAchievementProgressContext,
  buildAchievementWallEntries,
  pickFeaturedAchievement,
  summarizeAchievementRarities,
} from '@/utils/achievementProgress'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const toast = useToastStore()

const userId = computed(() => route.params.id as string)
const isOwnProfile = computed(() => userStore.user?.id === Number(userId.value))
const sponsorLevel = computed(() => {
  const level = userProfile.value?.sponsor_level
  if (typeof level === 'number') return level
  return userProfile.value?.is_sponsor ? 2 : 0
})
const isSponsor = computed(() => sponsorLevel.value > 0)
const canEditSponsorStyle = computed(() => sponsorLevel.value >= 2)
const sponsorStyleTip = computed(() => {
  if (canEditSponsorStyle.value) return ''
  if (sponsorLevel.value === 1) {
    return '当前为 Lv1 仅鸣谢，升级到 Lv2 可解锁昵称样式。'
  }
  return '成为赞助者以获得自定义昵称样式权限！'
})

const userProfile = ref<any>(null)
const userGuilds = ref<any[]>([])
const loading = ref(true)
const editMode = ref(false)
const showLevelGuide = ref(false)
const showAchievementDetail = ref(false)
const selectedAchievementId = ref<string | null>(null)
const avatarUploading = ref(false)
const avatarInputRef = ref<HTMLInputElement | null>(null)
const avatarCropperOpen = ref(false)
const avatarCropperFile = ref<File | null>(null)
const sponsorColor = ref('')
const sponsorBold = ref(false)
const nameStylePreference = ref<'default' | 'sponsor'>('default')

// 邮箱绑定相关
const showEmailBinding = ref(false)
const newEmail = ref('')
const emailCode = ref('')
const sendingEmailCode = ref(false)
const emailCountdown = ref(0)

let emailCountdownTimer: number | null = null

function normalizeNameStylePreference(value?: string): 'default' | 'sponsor' {
  return value === 'sponsor' ? 'sponsor' : 'default'
}

// 表单数据
const formData = ref({
  username: '',
  bio: '',
  location: '',
  website: ''
})

const nameStyleOptions = computed(() => [
  { value: 'default' as const, label: '默认颜色', disabled: false },
  { value: 'sponsor' as const, label: '赞助者颜色', disabled: !canEditSponsorStyle.value },
])

const activityExpRules = [
  '每日签到 +10',
  '每日首次点赞 +5',
  '发表评论 +3，收到评论 +3（含社区、道具与 RP 数据库）',
  '帖子或 RP 数据库作品获赞 +5，道具获赞 +10',
  '帖子或 RP 数据库作品审核通过 +30，道具审核通过 +50',
  '道具被下载 +10',
  '剧情归档每累计 10 条 +1，每日最多 +50',
]

const forumLevelDefinitions = [
  { level: 1, name: '新人', color: '#403B33', bold: false },
  { level: 2, name: '启源', color: '#808080', bold: false },
  { level: 3, name: '常态', color: '#FFFFFF', bold: false },
  { level: 4, name: '优秀', color: '#00C100', bold: false },
  { level: 5, name: '精良', color: '#0080FF', bold: false },
  { level: 6, name: '史诗', color: '#800080', bold: false },
  { level: 7, name: '传奇', color: '#F59B00', bold: true },
  { level: 8, name: '传承', color: '#0080C0', bold: true },
  { level: 9, name: '神话', color: '#EBD7A7', bold: true },
  { level: 10, name: '顶级', color: '#8E1027', bold: true },
] as const

const forumLevelGuide = computed(() => buildForumLevelGuide(forumLevelDefinitions))

const activityProgressPercent = computed(() => {
  const apiValue = Number(userProfile.value?.level_progress_percent)
  if (Number.isFinite(apiValue)) {
    return Math.max(0, Math.min(100, Math.round(apiValue)))
  }
  return computeLevelProgressPercent(userProfile.value?.current_level_exp, userProfile.value?.next_level_exp)
})

const achievementProgressContext = computed(() => buildAchievementProgressContext({
  profile: userProfile.value,
  guilds: userGuilds.value,
  sponsorLevel: sponsorLevel.value,
}))

const achievementEntries = computed(() => buildAchievementEntries(achievementProgressContext.value))

const earnedAchievementCount = computed(() => achievementEntries.value.filter((entry) => entry.progress.earned).length)

const achievementRaritySummary = computed(() => summarizeAchievementRarities(achievementEntries.value))
const achievementWallEntries = computed(() => buildAchievementWallEntries(achievementEntries.value, 6))

const selectedAchievementEntry = computed(() => {
  if (!selectedAchievementId.value) return null
  return achievementEntries.value.find((entry) => entry.definition.id === selectedAchievementId.value) || null
})

const featuredAchievementEntry = computed(() => pickFeaturedAchievement(achievementEntries.value))

onMounted(async () => {
  await loadUserProfile()
  await loadUserGuilds()
})

async function loadUserProfile() {
  try {
    loading.value = true
    const res = await request.get<UserInfo>(`/users/${userId.value}`)
    userProfile.value = res
    formData.value = {
      username: res.username || '',
      bio: res.bio || '',
      location: res.location || '',
      website: res.website || ''
    }
    sponsorColor.value = res.sponsor_color || ''
    sponsorBold.value = !!res.sponsor_bold
    nameStylePreference.value = normalizeNameStylePreference(res.name_style_preference)
    if (isOwnProfile.value && userStore.user) {
      const level = typeof res.sponsor_level === 'number' ? res.sponsor_level : (res.is_sponsor ? 2 : 0)
      userStore.mergeUser({
        ...res,
        username: res.username,
        name_color: res.name_color,
        name_bold: res.name_bold,
        is_sponsor: level > 0,
        sponsor_level: level,
        sponsor_color: res.sponsor_color,
        sponsor_bold: res.sponsor_bold,
        total_sign_in_days: res.total_sign_in_days,
        consecutive_sign_in_days: res.consecutive_sign_in_days,
        post_count: res.post_count,
        guild_count: res.guild_count,
        item_count: res.item_count,
        story_count: res.story_count,
        story_entry_count: res.story_entry_count,
        profile_count: res.profile_count,
        character_card_count: res.character_card_count,
        max_post_views: res.max_post_views,
        max_item_downloads: res.max_item_downloads,
        total_likes: res.total_likes,
        total_item_downloads: res.total_item_downloads,
      })
    }
  } catch (error: any) {
    console.error('加载用户信息失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadUserGuilds() {
  try {
    const res = await request.get<{ guilds: any[] }>(`/users/${userId.value}/guilds`)
    userGuilds.value = res.guilds || []
  } catch (error: any) {
    console.error('加载公会列表失败:', error)
  }
}

async function saveProfile() {
  try {
    const payload: Record<string, any> = { ...formData.value }
    if (canEditSponsorStyle.value) {
      payload.sponsor_color = sponsorColor.value
      payload.sponsor_bold = sponsorBold.value
    }
    payload.name_style_preference = nameStylePreference.value
    await request.put('/user/info', payload)
    await loadUserProfile()
    editMode.value = false
    toast.success('保存成功')
  } catch (error: any) {
    console.error('保存失败:', error)
    toast.error('保存失败')
  }
}

function cancelEdit() {
  editMode.value = false
  formData.value = {
    username: userProfile.value?.username || '',
    bio: userProfile.value?.bio || '',
    location: userProfile.value?.location || '',
    website: userProfile.value?.website || ''
  }
  sponsorColor.value = userProfile.value?.sponsor_color || ''
  sponsorBold.value = !!userProfile.value?.sponsor_bold
  nameStylePreference.value = normalizeNameStylePreference(userProfile.value?.name_style_preference)
}

function triggerAvatarUpload() {
  if (!isOwnProfile.value) return
  avatarInputRef.value?.click()
}

function handleAvatarChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    input.value = ''
    return
  }
  if (file.size > 20 * 1024 * 1024) {
    toast.warning('头像文件不能超过20MB')
    input.value = ''
    return
  }

  avatarCropperFile.value = file
  avatarCropperOpen.value = true
  input.value = ''
}

async function handleAvatarCropped(file: File) {
  avatarUploading.value = true
  try {
    const res = await uploadAvatar(file)
    userStore.mergeUser(res)
    userProfile.value = {
      ...userProfile.value,
      ...res,
      avatar: res.avatar,
    }
    toast.success(res.message || '头像更新成功')
  } catch (error: any) {
    toast.error(error.message || '上传失败')
  } finally {
    avatarUploading.value = false
    avatarCropperFile.value = null
  }
}

function handleAvatarCropperError(error: Error) {
  toast.error(error.message || '头像处理失败')
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short' })
}

function getRoleLabel(role: string) {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role] || role
}

function formatCostLabel(cost?: number) {
  return cost && cost > 0 ? `${cost} 积分` : '免费'
}

function goBack() {
  router.back()
}

async function handleSendEmailCode() {
  // 如果用户有未验证的邮箱，且没有输入新邮箱，则使用当前邮箱
  const emailToVerify = newEmail.value || userProfile.value?.email

  if (!emailToVerify || !emailToVerify.includes('@')) {
    toast.error('请输入有效的邮箱地址')
    return
  }

  sendingEmailCode.value = true
  try {
    await sendVerificationCode(emailToVerify)
    toast.success('验证码已发送到您的邮箱')

    // 如果使用的是当前邮箱，自动填充
    if (!newEmail.value && userProfile.value?.email) {
      newEmail.value = userProfile.value.email
    }

    // 开始60秒倒计时
    emailCountdown.value = 60
    emailCountdownTimer = setInterval(() => {
      emailCountdown.value--
      if (emailCountdown.value <= 0 && emailCountdownTimer) {
        clearInterval(emailCountdownTimer)
        emailCountdownTimer = null
      }
    }, 1000) as unknown as number
  } catch (error: any) {
    toast.error(error.message || '发送验证码失败')
  } finally {
    sendingEmailCode.value = false
  }
}

async function handleBindEmail() {
  if (!newEmail.value || !emailCode.value) {
    toast.error('请填写邮箱和验证码')
    return
  }

  try {
    await bindEmail(newEmail.value, emailCode.value)
    toast.success('邮箱绑定成功')
    showEmailBinding.value = false
    newEmail.value = ''
    emailCode.value = ''
    await loadUserProfile()
  } catch (error: any) {
    toast.error(error.message || '绑定失败')
  }
}

function openAchievementDetail(achievement: AchievementDefinition) {
  selectedAchievementId.value = achievement.id
  showAchievementDetail.value = true
}

function openAchievementList() {
  router.push({ name: 'user-achievements', params: { id: userId.value } })
}

function previewAchievementNotification() {
  const entry = featuredAchievementEntry.value
  if (!entry) return
  toast.achievement(
    entry.definition.title,
    entry.definition.condition,
    {
      icon: entry.definition.icon,
      rarity: entry.definition.rarity,
    },
  )
}
</script>

<template>
  <div class="user-profile">
    <!-- 背景装饰 -->
    <div class="bg-pattern"></div>
    <div class="bg-gradient"></div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="userProfile" class="profile-container">
      <!-- 顶部导航 -->
      <header class="page-header">
        <div class="header-label">个人主页</div>
        <button class="back-btn" @click="goBack">
          <i class="ri-arrow-left-s-line"></i>
          返回
        </button>
      </header>

      <!-- Bento Grid 布局 -->
      <div class="bento-grid">
        <!-- 1. 身份卡片 -->
        <div class="identity-card">
          <div class="card-accent"></div>

          <!-- 头像 -->
          <div class="avatar-wrapper" :class="{ clickable: isOwnProfile }" @click="triggerAvatarUpload">
            <img v-if="userProfile.avatar" :src="userProfile.avatar" alt="头像" />
            <span v-else class="avatar-letter">{{ userProfile.username?.charAt(0)?.toUpperCase() || 'U' }}</span>
            <div v-if="isOwnProfile" class="avatar-overlay">
              <i :class="avatarUploading ? 'ri-loader-4-line spin' : 'ri-camera-line'"></i>
            </div>
          </div>
          <input ref="avatarInputRef" type="file" accept="image/*" style="display: none" @change="handleAvatarChange" />
          <ImageCropperDialog
            v-model="avatarCropperOpen"
            :file="avatarCropperFile"
            :aspect-ratio="1"
            :output-width="512"
            :output-height="512"
            :max-size-k-b="512"
            title="调整头像"
            round-preview
            @cropped="handleAvatarCropped"
            @error="handleAvatarCropperError"
          />

          <h1 class="username" :style="buildNameStyle(userProfile.name_color, userProfile.name_bold)">{{ userProfile.username }}</h1>
          <div class="user-meta">
            <span v-if="isSponsor" class="sponsor-badge">赞助 Lv{{ sponsorLevel }}</span>
            <span class="role-badge" :class="userProfile.role">
              {{ userProfile.role === 'admin' ? '管理员' : userProfile.role === 'moderator' ? '版主' : '用户' }}
            </span>
          </div>

          <div class="activity-panel">
            <div class="activity-header">
              <span class="activity-title">社区活跃度</span>
              <UserLevelBadge
                :level="userProfile.forum_level"
                :name="userProfile.forum_level_name"
                :color="userProfile.forum_level_color"
                :bold="userProfile.forum_level_bold"
                size="md"
              />
            </div>
            <div class="activity-progress">
              <div class="progress-meta">
                <span>等级进度</span>
                <span>{{ activityProgressPercent }}%</span>
              </div>
              <div class="progress-track">
                <div
                  class="progress-fill"
                  :style="{
                    width: `${activityProgressPercent}%`,
                    background: userProfile.forum_level_color || '#B87333',
                  }"
                ></div>
              </div>
              <div class="progress-foot">
                <span>{{ userProfile.current_level_exp || 0 }} / {{ userProfile.next_level_exp || userProfile.current_level_exp || 0 }}</span>
                <span class="progress-foot-note">
                  <span>经验</span>
                  <button type="button" class="progress-help" aria-label="经验与等级说明" @click="showLevelGuide = true">
                      <i class="ri-question-line"></i>
                  </button>
                </span>
              </div>
            </div>
            <div v-if="isOwnProfile" class="activity-stats">
              <div class="activity-stat">
                <span>积分</span>
                <strong>{{ userProfile.activity_points || 0 }}</strong>
              </div>
              <div class="activity-stat">
                <span>累计经验</span>
                <strong>{{ userProfile.activity_experience || 0 }}</strong>
              </div>
            </div>
          </div>

          <!-- 统计数据 -->
          <div class="stats-row">
            <div class="stat-item">
              <div class="stat-value">{{ userProfile.post_count || 0 }}</div>
              <div class="stat-label">帖子</div>
            </div>
            <div class="stat-item bordered">
              <div class="stat-value">{{ userProfile.story_count || 0 }}</div>
              <div class="stat-label">剧情</div>
            </div>
            <div class="stat-item">
              <div class="stat-value">{{ userProfile.character_card_count ?? userProfile.profile_count ?? 0 }}</div>
              <div class="stat-label">人物卡</div>
            </div>
          </div>

          <button v-if="isOwnProfile" class="edit-profile-btn" @click="editMode = !editMode">
            {{ editMode ? '取消编辑' : '编辑资料' }}
          </button>
        </div>

        <!-- 2. 简介卡片 -->
        <div class="bio-card">
          <div class="card-icon">
            <i class="ri-user-heart-line"></i>
          </div>

          <template v-if="!editMode">
            <h2 class="card-title">个人简介</h2>
            <p class="bio-text">{{ userProfile.bio || '这个人很懒，什么都没写...' }}</p>

            <div class="info-row">
              <div v-if="userProfile.location" class="info-item">
                <i class="ri-map-pin-line"></i>
                <span>{{ userProfile.location }}</span>
              </div>
              <div v-if="userProfile.website" class="info-item">
                <i class="ri-global-line"></i>
                <a :href="userProfile.website" target="_blank">{{ userProfile.website }}</a>
              </div>
            </div>
          </template>

          <!-- 编辑模式 -->
          <template v-else>
            <h2 class="card-title">编辑资料</h2>
            <div class="edit-form">
              <div class="form-group">
                <label>用户名</label>
                <input v-model="formData.username" type="text" placeholder="你的用户名" maxlength="50">
                <p class="field-hint">本次修改费用：{{ formatCostLabel(userProfile.next_username_change_cost) }}</p>
              </div>

              <div class="form-group sponsor-style" :class="{ locked: !canEditSponsorStyle }">
                <div class="sponsor-style-header">
                  <label>赞助者昵称样式</label>
                  <span class="sponsor-badge" :class="{ locked: !canEditSponsorStyle }">
                    {{ isSponsor ? `Lv${sponsorLevel} 赞助` : '赞助者权限' }}
                  </span>
                </div>
                <div class="sponsor-style-controls" :class="{ disabled: !canEditSponsorStyle }">
                  <RColorPicker v-model="sponsorColor" />
                  <RCheckbox v-model="sponsorBold" label="加粗显示" />
                </div>
                <p v-if="!canEditSponsorStyle" class="sponsor-style-tip locked">{{ sponsorStyleTip }}</p>
              </div>

              <div class="form-group name-style-section">
                <label>昵称展示</label>
                <div class="name-style-options">
                  <button
                    v-for="option in nameStyleOptions"
                    :key="option.value"
                    type="button"
                    class="name-style-btn"
                    :class="{ active: nameStylePreference === option.value }"
                    :disabled="option.disabled"
                    @click="nameStylePreference = option.value"
                  >
                    {{ option.label }}
                  </button>
                </div>
                <p class="field-hint">普通用户统一使用默认昵称颜色，赞助者可切换为赞助者颜色。</p>
              </div>

              <!-- 邮箱绑定区域 -->
              <div class="form-group email-section">
                <div class="email-header">
                  <label>邮箱</label>
                  <span v-if="userProfile.email && userProfile.email_verified" class="email-status verified">
                    <i class="ri-checkbox-circle-fill"></i>
                    已验证
                  </span>
                  <span v-else-if="userProfile.email && !userProfile.email_verified" class="email-status warning">
                    <i class="ri-error-warning-fill"></i>
                    未验证
                  </span>
                  <span v-else class="email-status error">
                    <i class="ri-close-circle-fill"></i>
                    未绑定
                  </span>
                </div>
                <div class="current-email">
                  {{ userProfile.email || '未绑定邮箱' }}
                </div>
                <button
                  v-if="!showEmailBinding"
                  type="button"
                  class="change-email-btn"
                  @click="showEmailBinding = true"
                >
                  {{ userProfile.email ? (userProfile.email_verified ? '更换邮箱' : '验证邮箱') : '绑定邮箱' }}
                </button>

                <div v-if="showEmailBinding" class="email-binding-form">
                  <div class="form-group">
                    <input
                      v-model="newEmail"
                      type="email"
                      :placeholder="userProfile.email && !userProfile.email_verified ? '验证当前邮箱或输入新邮箱' : '新邮箱地址'"
                    />
                  </div>
                  <div class="verification-group">
                    <input v-model="emailCode" placeholder="验证码" maxlength="6" />
                    <button
                      type="button"
                      class="btn-send-code"
                      @click="handleSendEmailCode"
                      :disabled="!newEmail || emailCountdown > 0"
                    >
                      <span v-if="emailCountdown > 0">{{ emailCountdown }}s</span>
                      <span v-else-if="sendingEmailCode">发送中...</span>
                      <span v-else>获取验证码</span>
                    </button>
                  </div>
                  <div class="email-actions">
                    <button type="button" class="bind-btn" @click="handleBindEmail">确认{{ userProfile.email && !userProfile.email_verified ? '验证' : '绑定' }}</button>
                    <button type="button" class="cancel-bind-btn" @click="showEmailBinding = false; newEmail = ''; emailCode = ''">取消</button>
                  </div>
                </div>

                <p v-if="!userProfile.email" class="email-tip error-tip">
                  <i class="ri-information-line"></i>
                  绑定邮箱后可用于找回密码和账号安全验证
                </p>
                <p v-else-if="!userProfile.email_verified" class="email-tip warning-tip">
                  <i class="ri-information-line"></i>
                  邮箱未验证，验证后可用于找回密码和账号安全验证
                </p>
              </div>

              <div class="form-group">
                <label>个人简介</label>
                <textarea v-model="formData.bio" placeholder="介绍一下自己..." maxlength="500" rows="4"></textarea>
              </div>
              <div class="form-row">
                <div class="form-group">
                  <label>地区</label>
                  <input v-model="formData.location" type="text" placeholder="你的地区" maxlength="100">
                </div>
                <div class="form-group">
                  <label>个人网站</label>
                  <input v-model="formData.website" type="url" placeholder="https://...">
                </div>
              </div>
              <div class="form-actions">
                <button class="save-btn" @click="saveProfile">保存</button>
                <button class="cancel-btn" @click="cancelEdit">取消</button>
              </div>
            </div>
          </template>
        </div>

        <!-- 3. 公会卡片：桌面端承接身份卡右侧第二行 -->
        <div class="guilds-card">
          <div class="card-header">
            <h2 class="card-title">加入的公会</h2>
            <div v-if="isOwnProfile" class="header-actions">
              <router-link to="/guild" class="join-btn">
                <i class="ri-shield-line"></i>
                加入公会
              </router-link>
              <router-link to="/guild/create" class="create-btn">
                <i class="ri-add-line"></i>
                创建
              </router-link>
            </div>
          </div>

          <div class="guilds-list">
            <template v-if="userGuilds.length === 0">
              <div class="empty-guilds">
                <i class="ri-shield-line"></i>
                <p>还没有加入任何公会</p>
              </div>
            </template>
            <template v-else>
              <router-link
                v-for="guild in userGuilds"
                :key="guild.id"
                :to="`/guild/${guild.id}`"
                class="guild-item"
                :class="{ pending: guild.status === 'pending' }"
              >
                <div class="guild-icon" :style="{ background: guild.color || 'var(--color-accent, #D4A373)' }">
                  {{ guild.name?.charAt(0) || 'G' }}
                </div>
                <div class="guild-info">
                  <h3>{{ guild.name }}</h3>
                  <p>{{ guild.member_count }} 成员 · {{ getRoleLabel(guild.role) }}</p>
                </div>
                <div class="guild-badge">
                  <span v-if="guild.status === 'pending'" class="pending-tag">
                    <i class="ri-time-line"></i>
                    待审核
                  </span>
                  <span v-else class="role-tag">{{ getRoleLabel(guild.role) }}</span>
                </div>
              </router-link>
            </template>
          </div>
        </div>

        <CharacterCardWall
          :user-id="Number(userId)"
          :is-own-profile="isOwnProfile"
        />

        <RPDBContributionSection
          :user-id="Number(userId)"
          :is-own-profile="isOwnProfile"
        />

        <!-- 4. 账户状态卡片 -->
        <div class="status-card">
          <div class="status-bg"></div>
          <div class="status-content">
            <h2 class="status-title">账户状态</h2>
            <div class="status-indicator">
              <span class="status-dot"></span>
              <span>正常</span>
            </div>

            <div class="status-info">
              <div class="status-row">
                <span>角色</span>
                <span class="mono">{{ userProfile.role === 'admin' ? '管理员' : userProfile.role === 'moderator' ? '版主' : '用户' }}</span>
              </div>
              <div class="status-row">
                <span>注册时间</span>
                <span class="mono">{{ formatDate(userProfile.created_at) }}</span>
              </div>
            </div>
          </div>

          <button v-if="isOwnProfile" class="settings-btn" @click="$router.push('/settings')">
            设置
          </button>
        </div>

        <!-- 5. 成就墙 -->
        <div class="achievements-card achievement-wall-card">
          <div class="achievements-hero achievement-wall-hero">
            <div>
              <span class="achievements-kicker">Achievement Wall</span>
              <h2>{{ isOwnProfile ? '我的成就墙' : '成就墙' }}</h2>
              <p>展示已解锁的代表勋章和下一阶段目标，完整成就列表可单独查看。</p>
            </div>
            <div class="achievements-score">
              <strong>{{ earnedAchievementCount }}</strong>
              <span>/ {{ achievementEntries.length }}</span>
              <small>已获得</small>
            </div>
          </div>

          <button
            v-if="featuredAchievementEntry"
            type="button"
            class="achievement-featured"
            :class="{ earned: featuredAchievementEntry.progress.earned }"
            @click="openAchievementDetail(featuredAchievementEntry.definition)"
          >
            <AchievementMedal
              :achievement="featuredAchievementEntry.definition"
              :earned="featuredAchievementEntry.progress.earned"
              size="lg"
            />
            <span class="achievement-featured__copy">
              <small>{{ featuredAchievementEntry.progress.earned ? '代表成就' : '首个目标' }}</small>
              <strong>{{ featuredAchievementEntry.definition.title }}</strong>
              <span>{{ featuredAchievementEntry.definition.condition }}</span>
            </span>
            <i class="ri-arrow-right-s-line"></i>
          </button>

          <div class="achievement-wall-strip">
            <button
              v-for="entry in achievementWallEntries"
              :key="entry.definition.id"
              type="button"
              class="achievement-wall-medal"
              :class="{ earned: entry.progress.earned }"
              @click="openAchievementDetail(entry.definition)"
            >
              <AchievementMedal
                :achievement="entry.definition"
                :earned="entry.progress.earned"
                size="sm"
              />
              <span>{{ entry.definition.title }}</span>
            </button>
          </div>

          <div class="achievement-rarity-strip achievement-wall-rarity-strip">
            <div
              v-for="summary in achievementRaritySummary"
              :key="summary.rarity"
              class="achievement-rarity-pill"
              :style="{
                '--rarity-edge': ACHIEVEMENT_RARITY_META[summary.rarity].edge,
                '--rarity-glow': ACHIEVEMENT_RARITY_META[summary.rarity].glow,
              }"
            >
              <span>{{ summary.label }}</span>
              <strong>{{ summary.earned }}/{{ summary.total }}</strong>
            </div>
          </div>

          <div class="achievement-card-foot">
            <span>
              <i class="ri-medal-line"></i>
              成就按签到、社区、道具、剧情、赞助与等级进度自动点亮。
            </span>
            <div class="achievement-card-actions">
              <button v-if="isOwnProfile" type="button" class="achievement-preview-btn ghost" @click="previewAchievementNotification">
                预览通知
              </button>
              <button type="button" class="achievement-preview-btn" @click="openAchievementList">
                查看全部成就
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <RModal v-model="showLevelGuide" title="经验与等级说明" width="720px">
      <div class="level-guide">
        <section class="guide-section">
          <h3>经验获取方式</h3>
          <ul class="guide-rule-list">
            <li v-for="rule in activityExpRules" :key="rule">{{ rule }}</li>
          </ul>
        </section>

        <section class="guide-section">
          <div class="guide-section-head">
            <h3>等级列表</h3>
            <span v-if="userProfile?.forum_level" class="guide-current-text">当前等级：Lv{{ userProfile.forum_level }} {{ userProfile.forum_level_name }}</span>
          </div>
          <div class="level-guide-list">
            <div
              v-for="level in forumLevelGuide"
              :key="level.level"
              class="level-guide-item"
              :class="{ current: userProfile?.forum_level === level.level }"
            >
              <div class="level-guide-main">
                <UserLevelBadge
                  :level="level.level"
                  :name="level.name"
                  :color="level.color"
                  :bold="level.bold"
                  size="sm"
                />
                <span v-if="userProfile?.forum_level === level.level" class="level-guide-current-badge">当前</span>
              </div>
              <div class="level-guide-meta">
                <span>等级门槛：{{ level.currentBase }} 总经验</span>
                <span v-if="level.nextBase !== null">下一级：{{ level.nextBase }} 经验</span>
                <span v-else>已到最高等级</span>
              </div>
            </div>
          </div>
        </section>
      </div>
    </RModal>

    <RModal
      v-model="showAchievementDetail"
      :title="selectedAchievementEntry ? selectedAchievementEntry.definition.title : '成就详情'"
      width="560px"
    >
      <div v-if="selectedAchievementEntry" class="achievement-detail">
        <AchievementMedal
          :achievement="selectedAchievementEntry.definition"
          :earned="selectedAchievementEntry.progress.earned"
          size="lg"
        />
        <div class="achievement-detail__body">
          <div class="achievement-detail__tags">
            <span
              class="achievement-detail__rarity"
              :style="{ '--rarity-edge': ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].edge }"
            >
              {{ ACHIEVEMENT_RARITY_META[selectedAchievementEntry.definition.rarity].label }}
            </span>
            <span>{{ ACHIEVEMENT_CATEGORY_META[selectedAchievementEntry.definition.category].label }}</span>
          </div>
          <h3>{{ selectedAchievementEntry.definition.title }}</h3>
          <p>{{ selectedAchievementEntry.definition.condition }}</p>
          <div class="achievement-detail__progress">
            <div class="achievement-detail__progress-meta">
              <span>{{ selectedAchievementEntry.progress.earned ? '已获得' : '进度' }}</span>
              <strong>{{ selectedAchievementEntry.progress.label }}</strong>
            </div>
            <div class="achievement-detail__track">
              <div
                class="achievement-detail__fill"
                :style="{ width: `${selectedAchievementEntry.progress.percent}%` }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </RModal>
  </div>
</template>

<style scoped>
.user-profile {
  position: relative;
  min-height: 100vh;
  padding: 24px 0;
  width: 100%;
  box-sizing: border-box;
}

/* 背景装饰 */
.bg-pattern {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  opacity: 0.4;
  background-image: radial-gradient(#D4A373 0.5px, transparent 0.5px);
  background-size: 24px 24px;
}

.bg-gradient {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 256px;
  background: linear-gradient(to bottom, #fff, transparent);
  pointer-events: none;
  z-index: 0;
}

.loading {
  text-align: center;
  padding: 80px;
  color: #8C7B70;
  font-size: 16px;
}

.profile-container {
  position: relative;
  z-index: 10;
  width: 100%;
}

/* 顶部导航 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 40px;
}

.header-label {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 2px;
  color: #8C7B70;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: #B87333;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.2s;
}

.back-btn:hover {
  color: #4B3621;
}

.back-btn i {
  font-size: 18px;
}

/* Bento Grid 布局 */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  grid-auto-rows: minmax(100px, auto);
  gap: 24px;
  width: 100%;
}

/* 1. 身份卡片 */
.identity-card {
  grid-column: span 12;
  grid-row: span 2;
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  box-shadow: var(--shadow-md, 0 4px 20px -2px rgba(75, 54, 33, 0.05));
  border: 1px solid var(--color-border, #E8DCC8);
  padding: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  position: relative;
  overflow: hidden;
}

@media (min-width: 768px) {
  .identity-card {
    grid-column: 1 / span 4;
    grid-row: 1 / span 2;
  }
}

.card-accent {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 4px;
  background: linear-gradient(to right, var(--color-accent, #D4A373), var(--color-secondary, #B87333));
}

/* 头像 */
.avatar-wrapper {
  position: relative;
  width: 96px;
  height: 96px;
  margin-bottom: 16px;
}

.avatar-wrapper img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  border: 4px solid var(--color-panel-bg, #fff);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.avatar-wrapper .avatar-letter {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background:
    radial-gradient(circle at 30% 24%, rgba(255, 255, 255, 0.72), transparent 34%),
    linear-gradient(135deg, var(--gradient-start, #D4A373), var(--gradient-end, #8C7B70));
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  font-weight: 700;
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  border: 4px solid var(--color-panel-bg, #fff);
  box-shadow: 0 8px 22px rgba(var(--shadow-base, 75, 54, 33), 0.16);
}

.avatar-wrapper.clickable {
  cursor: pointer;
}

.avatar-overlay {
  position: absolute;
  inset: 4px;
  background: rgba(var(--shadow-base, 75, 54, 33), 0.6);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: all 0.3s;
  backdrop-filter: blur(4px);
}

.avatar-wrapper.clickable:hover .avatar-overlay {
  opacity: 1;
}

.avatar-overlay i {
  font-size: 32px;
  color: var(--color-text-light, #fff);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.username {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-main, #4B3621);
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.user-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 24px;
}

.activity-panel {
  width: 100%;
  margin-bottom: 24px;
  padding: 16px;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--color-card-bg, #FBF5EF), var(--color-panel-bg, #F2E6D8));
  border: 1px solid var(--color-border, #E8DCC8);
  box-sizing: border-box;
  overflow: visible;
}

.activity-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.activity-title {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text-secondary, #8C7B70);
}

.activity-progress {
  margin-top: 14px;
}

.progress-meta,
.progress-foot {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.progress-foot-note {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.progress-help {
  width: 18px;
  height: 18px;
  padding: 0;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-light, rgba(75, 54, 33, 0.08));
  border: 1px solid var(--color-border, rgba(140, 123, 112, 0.26));
  color: var(--color-text-secondary, #8C7B70);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}

.progress-help:hover {
  background: var(--color-card-bg-hover, rgba(184, 115, 51, 0.12));
  border-color: var(--color-border-hover, rgba(184, 115, 51, 0.32));
  color: var(--color-accent, #B87333);
}

.progress-help i {
  font-size: 12px;
  line-height: 1;
}

.level-guide {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.guide-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.guide-section h3 {
  margin: 0;
  font-size: 16px;
  color: var(--color-text-main, #4B3621);
}

.guide-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.guide-current-text {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
}

.guide-rule-list {
  margin: 0;
  padding-left: 18px;
  color: var(--color-text-secondary, #8C7B70);
  display: flex;
  flex-direction: column;
  gap: 8px;
  line-height: 1.6;
}

.level-guide-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.level-guide-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid var(--color-border, rgba(232, 220, 200, 0.92));
  background: var(--color-card-bg, rgba(255, 250, 245, 0.82));
}

.level-guide-item.current {
  border-color: var(--color-border-hover, rgba(184, 115, 51, 0.32));
  background: linear-gradient(135deg, var(--color-card-bg, rgba(255, 249, 240, 0.96)), var(--color-panel-bg, rgba(246, 233, 214, 0.96)));
  box-shadow: var(--shadow-sm, 0 10px 24px -20px rgba(128, 64, 48, 0.42));
}

.level-guide-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.level-guide-current-badge {
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--color-primary-light, rgba(184, 115, 51, 0.14));
  color: var(--color-accent, #B87333);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.level-guide-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 12px;
  text-align: right;
}

@media (max-width: 767px) {
  .progress-meta,
  .progress-foot {
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 8px;
  }

  .progress-help {
    width: 22px;
    height: 22px;
  }

  .guide-section {
    gap: 10px;
  }

  .guide-section-head {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
  }

  .guide-rule-list {
    padding-left: 16px;
    font-size: 13px;
  }

  .level-guide-item {
    flex-direction: column;
    align-items: flex-start;
    padding: 12px 13px;
  }

  .level-guide-main {
    width: 100%;
    justify-content: space-between;
    gap: 8px;
  }

  .level-guide-meta {
    align-items: flex-start;
    text-align: left;
    width: 100%;
    font-size: 12px;
  }
}

.progress-track {
  margin: 8px 0;
  height: 10px;
  border-radius: 999px;
  background: var(--color-card-bg-hover, rgba(75, 54, 33, 0.08));
  overflow: visible;
}

.progress-fill {
  height: 100%;
  border-radius: inherit;
  box-shadow: 0 0 14px rgba(255, 255, 255, 0.35);
}

.activity-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.activity-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border-radius: 10px;
  background: var(--color-panel-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--color-border-light, rgba(232, 220, 200, 0.9));
}

.activity-stat span {
  font-size: 11px;
  color: var(--color-text-secondary, #8C7B70);
}

.activity-stat strong {
  font-size: 22px;
  line-height: 1;
  color: var(--color-text-main, #4B3621);
}

.sponsor-badge {
  display: inline-flex;
  align-items: center;
  padding: 5px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-accent, #E7C67D), var(--color-secondary, #D6A645));
  color: var(--color-accent-contrast, var(--color-primary, #4B3621));
  border: 1px solid rgba(214, 166, 69, 0.4);
}

.sponsor-badge.locked {
  background: var(--color-card-bg, #F2E6D8);
  color: var(--color-text-secondary, #8C7B70);
  border-color: var(--color-border, #E0D2C1);
}

.role-badge {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.role-badge.user {
  background: var(--color-primary-light, rgba(140, 123, 112, 0.1));
  color: var(--color-text-secondary, #8C7B70);
  border: 1px solid rgba(var(--shadow-base, 75, 54, 33), 0.2);
}

.role-badge.moderator {
  background: var(--color-primary-light, rgba(184, 115, 51, 0.1));
  color: var(--color-accent, #B87333);
  border: 1px solid rgba(var(--shadow-base, 75, 54, 33), 0.2);
}

.role-badge.admin {
  background: var(--color-primary-light, rgba(128, 64, 48, 0.1));
  color: var(--color-secondary, #804030);
  border: 1px solid rgba(var(--shadow-base, 75, 54, 33), 0.2);
}

/* 统计数据 */
.stats-row {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  border-top: 1px solid var(--color-border-light, #F2E6D8);
  padding-top: 24px;
  margin-top: auto;
}

.stat-item {
  text-align: center;
}

.stat-item.bordered {
  border-left: 1px solid var(--color-border-light, #F2E6D8);
  border-right: 1px solid var(--color-border-light, #F2E6D8);
}

.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--color-accent, #B87333);
}

.stat-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: var(--color-text-secondary, #8C7B70);
  margin-top: 4px;
}

.edit-profile-btn {
  width: 100%;
  margin-top: 24px;
  padding: 10px;
  background: var(--color-card-bg, #FBF5EF);
  border: 1px solid var(--color-border, #E8DCC8);
  border-radius: 4px;
  color: var(--color-accent, #B87333);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
}

.edit-profile-btn:hover {
  background: var(--color-card-bg-hover, #F2E6D8);
  border-color: var(--color-border-hover, #D4A373);
}

/* 2. 简介卡片 */
.bio-card {
  grid-column: span 12;
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  box-shadow: var(--shadow-md, 0 4px 20px -2px rgba(75, 54, 33, 0.05));
  border: 1px solid var(--color-border, #E8DCC8);
  padding: 24px 32px;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

@media (min-width: 768px) {
  .bio-card {
    grid-column: 5 / -1;
    grid-row: 1;
  }
}

.card-icon {
  position: absolute;
  top: 16px;
  right: 16px;
  font-size: 24px;
  color: var(--icon-color, #D4A373);
  opacity: 0.2;
}

.card-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 2px;
  color: var(--color-text-secondary, #8C7B70);
  margin: 0 0 16px 0;
}

.bio-text {
  font-size: 15px;
  font-weight: 500;
  color: var(--color-text-main, #4B3621);
  line-height: 1.7;
  margin: 0;
  max-width: 600px;
}

.info-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--color-border-light, #F2E6D8);
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-main, #4B3621);
}

.info-item i {
  color: var(--icon-color, #D4A373);
}

.info-item a {
  color: var(--link-color, #B87333);
  text-decoration: underline;
  text-decoration-color: rgba(212, 163, 115, 0.5);
  text-underline-offset: 2px;
  transition: all 0.2s;
}

.info-item a:hover {
  color: var(--link-hover, #4B3621);
}

/* 编辑表单 */
.edit-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 12px;
  font-weight: 600;
  color: #8C7B70;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.field-hint {
  margin: 0;
  font-size: 12px;
  color: #8C7B70;
}

.sponsor-style {
  border: 1px dashed #E8DCC8;
  border-radius: 8px;
  padding: 12px;
  background: rgba(251, 245, 239, 0.6);
}

.sponsor-style.locked {
  border-color: #E0D2C1;
  background: rgba(244, 238, 230, 0.8);
}

.sponsor-style-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.sponsor-style-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.sponsor-style-controls.disabled {
  pointer-events: none;
  opacity: 0.6;
  filter: grayscale(1);
}

.sponsor-style-tip {
  margin: 0;
  font-size: 12px;
  color: #8C7B70;
}

.sponsor-style-tip.locked {
  color: #9C8E82;
}

.name-style-section {
  border: 1px solid #E8DCC8;
  border-radius: 8px;
  padding: 12px;
  background: rgba(251, 245, 239, 0.55);
}

.name-style-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.name-style-btn {
  padding: 8px 12px;
  border: 1px solid #E8DCC8;
  border-radius: 999px;
  background: #fff;
  color: #6F5B4B;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.name-style-btn:hover:not(:disabled) {
  border-color: #B87333;
  color: #B87333;
}

.name-style-btn.active {
  background: #4B3621;
  border-color: #4B3621;
  color: #fff;
}

.name-style-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.form-group input,
.form-group textarea {
  padding: 12px;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 14px;
  background: #FBF5EF;
  color: #4B3621;
  transition: all 0.2s;
}

.form-group input:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #B87333;
  background: #fff;
}

.form-group textarea {
  resize: vertical;
  font-family: inherit;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.save-btn {
  padding: 10px 24px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
}

.save-btn:hover {
  background: #4B3621;
}

.cancel-btn {
  padding: 10px 24px;
  background: #FBF5EF;
  color: #8C7B70;
  border: 1px solid #E8DCC8;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn:hover {
  background: #F2E6D8;
}

/* 邮箱绑定样式 */
.email-section {
  border: 1px solid #E8DCC8;
  border-radius: 8px;
  padding: 16px;
  background: rgba(251, 245, 239, 0.5);
}

.email-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.email-status {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
}

.email-status.verified {
  color: #4ade80;
}

.email-status.warning {
  color: #FF9800;
}

.email-status.error {
  color: #f87171;
}

.email-status i {
  font-size: 14px;
}

.current-email {
  padding: 10px 12px;
  background: #fff;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 14px;
  color: #4B3621;
  margin-bottom: 12px;
}

.change-email-btn {
  width: 100%;
  padding: 8px;
  background: #FBF5EF;
  border: 1px solid #D4A373;
  border-radius: 6px;
  color: #B87333;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.change-email-btn:hover {
  background: #F2E6D8;
  border-color: #B87333;
}

.email-binding-form {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.verification-group {
  display: flex;
  gap: 8px;
}

.verification-group input {
  flex: 1;
  padding: 10px 12px;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 14px;
  background: #fff;
}

.btn-send-code {
  padding: 10px 16px;
  border: 1px solid #B87333;
  border-radius: 6px;
  background: #fff;
  color: #B87333;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}

.btn-send-code:hover:not(:disabled) {
  background: #B87333;
  color: #fff;
}

.btn-send-code:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.email-actions {
  display: flex;
  gap: 8px;
}

.bind-btn {
  flex: 1;
  padding: 8px;
  background: #B87333;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.bind-btn:hover {
  background: #4B3621;
}

.cancel-bind-btn {
  flex: 1;
  padding: 8px;
  background: #FBF5EF;
  color: #8C7B70;
  border: 1px solid #E8DCC8;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-bind-btn:hover {
  background: #F2E6D8;
}

.email-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 12px;
  font-size: 11px;
  line-height: 1.4;
}

.email-tip.error-tip {
  color: #f87171;
}

.email-tip.warning-tip {
  color: #FF9800;
}

.email-tip i {
  margin-top: 2px;
  flex-shrink: 0;
}

/* 3. 公会卡片 */
.guilds-card {
  grid-column: span 12;
  background: var(--color-panel-bg, #fff);
  border-radius: 12px;
  box-shadow: var(--shadow-md, 0 4px 20px -2px rgba(75, 54, 33, 0.05));
  border: 1px solid var(--color-border, #E8DCC8);
  padding: 24px;
  display: flex;
  flex-direction: column;
}

@media (min-width: 768px) {
  .guilds-card {
    grid-column: 5 / -1;
    grid-row: 2;
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--color-accent, #B87333);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-decoration: none;
  transition: color 0.2s;
}

.create-btn:hover {
  color: var(--color-primary, #4B3621);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.join-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--color-accent, #B87333);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-decoration: none;
  transition: color 0.2s;
}

.join-btn:hover {
  color: var(--color-primary, #4B3621);
}

.guilds-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-guilds {
  text-align: center;
  padding: 40px 20px;
  color: var(--color-text-secondary, #8C7B70);
}

.empty-guilds i {
  font-size: 48px;
  opacity: 0.3;
  margin-bottom: 12px;
  display: block;
}

.guild-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--color-border-light, #F2E6D8);
  background: var(--color-card-bg, rgba(251, 245, 239, 0.3));
  text-decoration: none;
  transition: all 0.2s;
}

.guild-item:hover {
  background: var(--color-panel-bg, #fff);
  border-color: var(--color-border-hover, #D4A373);
  box-shadow: var(--shadow-sm, 0 4px 12px rgba(0,0,0,0.05));
}

.guild-item.pending {
  border-style: dashed;
  border-color: rgba(255, 152, 0, 0.3);
  background: rgba(255, 152, 0, 0.05);
}

.guild-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-light, #fff);
  margin-right: 16px;
  flex-shrink: 0;
  border: 1px solid var(--color-border, #E8DCC8);
}

.guild-info {
  flex: 1;
  min-width: 0;
}

.guild-info h3 {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-main, #4B3621);
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.guild-info p {
  font-size: 12px;
  color: var(--color-text-secondary, #8C7B70);
  margin: 0;
}

.guild-badge {
  margin-left: 16px;
  flex-shrink: 0;
}

.role-tag {
  display: inline-flex;
  padding: 4px 8px;
  background: var(--color-card-bg, #F2E6D8);
  color: var(--color-text-main, #4B3621);
  border: 1px solid var(--color-border, #E8DCC8);
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.pending-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: rgba(255, 152, 0, 0.1);
  color: #FF9800;
  border: 1px solid rgba(255, 152, 0, 0.2);
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
}

.pending-tag i {
  font-size: 12px;
}

/* 4. 账户状态卡片 */
.status-card {
  grid-column: span 12;
  background: var(--color-primary, #4B3621);
  border-radius: 12px;
  box-shadow: var(--shadow-md, 0 4px 20px -2px rgba(75, 54, 33, 0.2));
  padding: 24px;
  color: var(--color-text-light, #fff);
  position: relative;
  overflow: visible;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

@media (min-width: 768px) {
  .status-card {
    grid-column: span 4;
  }
}

.status-bg {
  position: absolute;
  top: -32px;
  right: -32px;
  width: 128px;
  height: 128px;
  background: var(--color-accent, #B87333);
  border-radius: 50%;
  opacity: 0.2;
  filter: blur(40px);
}

.status-content {
  position: relative;
  z-index: 10;
}

.status-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 2px;
  color: rgba(212, 163, 115, 0.8);
  margin: 0 0 16px 0;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  font-size: 14px;
  font-weight: 500;
  letter-spacing: 0.5px;
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-success, #4ade80);
  box-shadow: 0 0 8px rgba(74, 222, 128, 0.6);
}

.status-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.status-row:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.status-row .mono {
  font-family: monospace;
  color: var(--color-accent, #D4A373);
}

.settings-btn {
  position: relative;
  z-index: 10;
  width: 100%;
  margin-top: 24px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  color: var(--color-text-light, #fff);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  backdrop-filter: blur(4px);
  transition: all 0.2s;
}

.settings-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* 5. 成就陈列 */
.achievements-card {
  grid-column: span 12;
  position: relative;
  overflow: hidden;
  border-radius: 18px;
  padding: 28px;
  border: 1px solid color-mix(in srgb, var(--color-border, #E8DCC8) 72%, transparent);
  background:
    radial-gradient(circle at 12% 8%, color-mix(in srgb, var(--color-accent, #B87333) 18%, transparent), transparent 28%),
    radial-gradient(circle at 88% 18%, rgba(255, 178, 62, 0.16), transparent 30%),
    linear-gradient(135deg, var(--color-panel-bg, #FFF9F0), var(--color-card-bg, #F8EEDF));
  box-shadow: var(--shadow-md, 0 20px 42px -30px rgba(75, 54, 33, 0.32));
}

.achievements-card::before {
  content: '';
  position: absolute;
  inset: 16px;
  pointer-events: none;
  border: 1px solid rgba(184, 115, 51, 0.12);
  border-radius: 14px;
}

.achievements-hero,
.achievement-card-foot {
  position: relative;
  z-index: 1;
}

.achievements-hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: flex-start;
  margin-bottom: 20px;
}

.achievements-kicker {
  display: inline-flex;
  margin-bottom: 8px;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 2px;
  text-transform: uppercase;
  color: var(--color-accent, #B87333);
}

.achievements-hero h2 {
  margin: 0;
  font-size: 28px;
  color: var(--color-text-main, #4B3621);
  letter-spacing: -0.02em;
}

.achievements-hero p {
  max-width: 700px;
  margin: 10px 0 0;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 14px;
  line-height: 1.7;
}

.achievements-score {
  min-width: 132px;
  padding: 14px 16px;
  border-radius: 16px;
  text-align: right;
  background: rgba(75, 54, 33, 0.08);
  border: 1px solid rgba(184, 115, 51, 0.16);
}

.achievements-score strong {
  font-size: 36px;
  line-height: 1;
  color: var(--color-accent, #B87333);
}

.achievements-score span {
  color: var(--color-text-secondary, #8C7B70);
  font-size: 16px;
  font-weight: 700;
}

.achievements-score small {
  display: block;
  margin-top: 5px;
  color: var(--color-text-muted, #9C8E82);
  font-size: 11px;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.achievement-wall-card {
  background:
    radial-gradient(circle at 10% 10%, color-mix(in srgb, var(--color-accent, #B87333) 20%, transparent), transparent 26%),
    radial-gradient(circle at 84% 22%, rgba(255, 190, 219, 0.26), transparent 28%),
    linear-gradient(135deg, var(--color-panel-bg, #FFF9F0), var(--color-card-bg, #F8EEDF));
}

.achievement-wall-hero {
  margin-bottom: 14px;
}

.achievement-featured {
  position: relative;
  z-index: 1;
  width: 100%;
  border: 1px solid rgba(184, 115, 51, 0.16);
  border-radius: 20px;
  padding: 16px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.64), rgba(255, 255, 255, 0.28)),
    radial-gradient(circle at 0% 50%, rgba(255, 178, 62, 0.14), transparent 34%);
  color: var(--color-text-main, #4B3621);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  text-align: left;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    background 0.2s ease;
}

.achievement-featured:hover {
  transform: translateY(-2px);
  border-color: rgba(184, 115, 51, 0.32);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.78), rgba(255, 255, 255, 0.38)),
    radial-gradient(circle at 0% 50%, rgba(255, 178, 62, 0.2), transparent 36%);
}

.achievement-featured__copy {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.achievement-featured__copy small {
  color: var(--color-accent, #B87333);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.achievement-featured__copy strong {
  color: var(--color-text-main, #4B3621);
  font-size: 18px;
}

.achievement-featured__copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 13px;
}

.achievement-featured > i {
  color: var(--color-accent, #B87333);
  font-size: 22px;
}

.achievement-wall-strip {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.achievement-wall-medal {
  min-width: 0;
  border: 1px solid rgba(184, 115, 51, 0.12);
  border-radius: 16px;
  padding: 12px 8px 10px;
  background: rgba(255, 255, 255, 0.38);
  color: var(--color-text-main, #4B3621);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    background 0.2s ease;
}

.achievement-wall-medal:hover {
  transform: translateY(-2px);
  border-color: rgba(184, 115, 51, 0.28);
  background: rgba(255, 255, 255, 0.64);
}

.achievement-wall-medal.earned {
  background:
    radial-gradient(circle at 50% 0%, rgba(255, 214, 135, 0.18), transparent 44%),
    rgba(255, 255, 255, 0.54);
}

.achievement-wall-medal span {
  width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 12px;
  text-align: center;
}

.achievement-wall-rarity-strip {
  margin: 14px 0 0;
}

.achievement-rarity-strip {
  position: relative;
  z-index: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
}

.achievement-rarity-pill {
  --rarity-edge: #B87333;
  --rarity-glow: rgba(184, 115, 51, 0.18);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--rarity-edge) 12%, rgba(255, 255, 255, 0.72));
  border: 1px solid color-mix(in srgb, var(--rarity-edge) 42%, transparent);
  color: var(--color-text-main, #4B3621);
  box-shadow: 0 8px 18px -16px var(--rarity-glow);
  font-size: 12px;
}

.achievement-rarity-pill strong {
  color: var(--rarity-edge);
}

.achievement-grid {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.achievement-tile {
  min-width: 0;
  border: 1px solid rgba(184, 115, 51, 0.12);
  border-radius: 16px;
  padding: 16px 10px 12px;
  background: rgba(255, 255, 255, 0.42);
  color: var(--color-text-main, #4B3621);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  transition:
    transform 0.2s ease,
    background 0.2s ease,
    border-color 0.2s ease;
}

.achievement-tile:hover {
  transform: translateY(-3px);
  border-color: rgba(184, 115, 51, 0.28);
  background: rgba(255, 255, 255, 0.7);
}

.achievement-tile.earned {
  border-color: rgba(184, 115, 51, 0.26);
  background:
    radial-gradient(circle at 50% 0%, rgba(255, 214, 135, 0.18), transparent 42%),
    rgba(255, 255, 255, 0.64);
}

.achievement-tile__meta {
  display: flex;
  flex-direction: column;
  gap: 3px;
  text-align: center;
  min-width: 0;
}

.achievement-tile__meta strong {
  font-size: 13px;
  color: var(--color-text-main, #4B3621);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.achievement-tile__meta span {
  font-size: 11px;
  color: var(--color-text-secondary, #8C7B70);
}

.achievement-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid rgba(184, 115, 51, 0.14);
  color: var(--color-text-secondary, #8C7B70);
  font-size: 12px;
  line-height: 1.6;
}

.achievement-card-foot span {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.achievement-card-foot i {
  color: var(--color-accent, #B87333);
}

.achievement-preview-btn {
  flex: 0 0 auto;
  padding: 9px 13px;
  border-radius: 999px;
  border: 1px solid rgba(184, 115, 51, 0.28);
  background: rgba(75, 54, 33, 0.08);
  color: var(--color-accent, #B87333);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
}

.achievement-preview-btn:hover {
  background: var(--color-accent, #B87333);
  color: var(--color-accent-contrast, #fff);
}

.achievement-preview-btn.ghost {
  background: transparent;
}

.achievement-card-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.achievement-detail {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 22px;
  align-items: start;
}

.achievement-detail__body {
  min-width: 0;
}

.achievement-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.achievement-detail__tags span {
  padding: 5px 9px;
  border-radius: 999px;
  background: var(--color-card-bg, #F2E6D8);
  color: var(--color-text-secondary, #8C7B70);
  font-size: 11px;
  font-weight: 700;
}

.achievement-detail__rarity {
  --rarity-edge: #B87333;
  background: color-mix(in srgb, var(--rarity-edge) 14%, #fff) !important;
  color: var(--rarity-edge) !important;
}

.achievement-detail h3 {
  margin: 0 0 8px;
  font-size: 22px;
  color: var(--color-text-main, #4B3621);
}

.achievement-detail p {
  margin: 0;
  color: var(--color-text-secondary, #8C7B70);
  line-height: 1.7;
}

.achievement-detail__progress {
  margin: 16px 0;
}

.achievement-detail__progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--color-text-secondary, #8C7B70);
  font-size: 12px;
}

.achievement-detail__progress-meta strong {
  color: var(--color-accent, #B87333);
}

.achievement-detail__track {
  height: 9px;
  border-radius: 999px;
  background: rgba(75, 54, 33, 0.1);
  overflow: hidden;
}

.achievement-detail__fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #D4A373, #FFB23E);
}

@media (max-width: 980px) {
  .achievement-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .achievement-wall-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .achievements-card {
    padding: 20px 14px;
  }

  .achievements-hero,
  .achievement-card-foot,
  .achievement-detail {
    grid-template-columns: 1fr;
    flex-direction: column;
    align-items: flex-start;
  }

  .achievement-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .achievement-featured {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .achievement-featured > i {
    display: none;
  }

  .achievement-wall-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .achievements-score {
    width: 100%;
    text-align: left;
  }

  .achievement-card-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
