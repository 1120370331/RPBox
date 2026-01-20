<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '../../stores/user'
import { useToastStore } from '../../stores/toast'
import { uploadAvatar, bindEmail } from '../../api/user'
import { sendVerificationCode } from '../../api/auth'
import request from '../../api/request'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const toast = useToastStore()

const userId = computed(() => route.params.id as string)
const isOwnProfile = computed(() => userStore.user?.id === Number(userId.value))

const userProfile = ref<any>(null)
const userGuilds = ref<any[]>([])
const loading = ref(true)
const editMode = ref(false)
const avatarUploading = ref(false)
const avatarInputRef = ref<HTMLInputElement | null>(null)

// 邮箱绑定相关
const showEmailBinding = ref(false)
const newEmail = ref('')
const emailCode = ref('')
const sendingEmailCode = ref(false)
const emailCountdown = ref(0)

let emailCountdownTimer: number | null = null

// 表单数据
const formData = ref({
  username: '',
  bio: '',
  location: '',
  website: ''
})

onMounted(async () => {
  await loadUserProfile()
  await loadUserGuilds()
})

async function loadUserProfile() {
  try {
    loading.value = true
    const res = await request.get(`/users/${userId.value}`)
    userProfile.value = res
    formData.value = {
      username: res.username || '',
      bio: res.bio || '',
      location: res.location || '',
      website: res.website || ''
    }
  } catch (error: any) {
    console.error('加载用户信息失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadUserGuilds() {
  try {
    const res = await request.get(`/users/${userId.value}/guilds`)
    userGuilds.value = res.guilds || []
  } catch (error: any) {
    console.error('加载公会列表失败:', error)
  }
}

async function saveProfile() {
  try {
    await request.put('/user/info', formData.value)
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
}

function triggerAvatarUpload() {
  if (!isOwnProfile.value) return
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
    userProfile.value.avatar = res.avatar
    toast.success('头像更新成功')
  } catch (error: any) {
    toast.error(error.message || '上传失败')
  } finally {
    avatarUploading.value = false
    input.value = ''
  }
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

function goBack() {
  router.back()
}

async function handleSendEmailCode() {
  if (!newEmail.value || !newEmail.value.includes('@')) {
    toast.error('请输入有效的邮箱地址')
    return
  }

  sendingEmailCode.value = true
  try {
    await sendVerificationCode(newEmail.value)
    toast.success('验证码已发送到您的邮箱')

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

          <h1 class="username">{{ userProfile.username }}</h1>
          <div class="user-meta">
            <span class="role-badge" :class="userProfile.role">
              {{ userProfile.role === 'admin' ? '管理员' : userProfile.role === 'moderator' ? '版主' : '用户' }}
            </span>
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
              <div class="stat-value">{{ userProfile.profile_count || 0 }}</div>
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
              </div>

              <!-- 邮箱绑定区域 -->
              <div class="form-group email-section">
                <div class="email-header">
                  <label>邮箱</label>
                  <span v-if="userProfile.email" class="email-status">
                    <i class="ri-checkbox-circle-fill"></i>
                    已绑定
                  </span>
                  <span v-else class="email-status warning">
                    <i class="ri-error-warning-fill"></i>
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
                  {{ userProfile.email ? '更换邮箱' : '绑定邮箱' }}
                </button>

                <div v-if="showEmailBinding" class="email-binding-form">
                  <div class="form-group">
                    <input v-model="newEmail" type="email" placeholder="新邮箱地址" />
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
                    <button type="button" class="bind-btn" @click="handleBindEmail">确认绑定</button>
                    <button type="button" class="cancel-bind-btn" @click="showEmailBinding = false; newEmail = ''; emailCode = ''">取消</button>
                  </div>
                </div>

                <p v-if="!userProfile.email" class="email-tip">
                  <i class="ri-information-line"></i>
                  绑定邮箱后可用于找回密码和账号安全验证
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

        <!-- 3. 公会卡片 -->
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
                <div class="guild-icon" :style="{ background: guild.color || '#D4A373' }">
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
      </div>
    </div>
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
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  border: 1px solid #E8DCC8;
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
    grid-column: span 4;
  }
}

.card-accent {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 4px;
  background: linear-gradient(to right, #D4A373, #B87333);
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
  border: 4px solid #fff;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.avatar-wrapper .avatar-letter {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  font-weight: 700;
  color: #fff;
  border: 4px solid #fff;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.avatar-wrapper.clickable {
  cursor: pointer;
}

.avatar-overlay {
  position: absolute;
  inset: 4px;
  background: rgba(75, 54, 33, 0.6);
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
  color: #fff;
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
  color: #4B3621;
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.user-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 24px;
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
  background: rgba(140, 123, 112, 0.1);
  color: #8C7B70;
  border: 1px solid rgba(140, 123, 112, 0.2);
}

.role-badge.moderator {
  background: rgba(184, 115, 51, 0.1);
  color: #B87333;
  border: 1px solid rgba(184, 115, 51, 0.2);
}

.role-badge.admin {
  background: rgba(128, 64, 48, 0.1);
  color: #804030;
  border: 1px solid rgba(128, 64, 48, 0.2);
}

/* 统计数据 */
.stats-row {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  border-top: 1px solid #F2E6D8;
  padding-top: 24px;
  margin-top: auto;
}

.stat-item {
  text-align: center;
}

.stat-item.bordered {
  border-left: 1px solid #F2E6D8;
  border-right: 1px solid #F2E6D8;
}

.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: #B87333;
}

.stat-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: #8C7B70;
  margin-top: 4px;
}

.edit-profile-btn {
  width: 100%;
  margin-top: 24px;
  padding: 10px;
  background: #FBF5EF;
  border: 1px solid #E8DCC8;
  border-radius: 4px;
  color: #B87333;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
}

.edit-profile-btn:hover {
  background: #F2E6D8;
  border-color: #D4A373;
}

/* 2. 简介卡片 */
.bio-card {
  grid-column: span 12;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  border: 1px solid #E8DCC8;
  padding: 24px 32px;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

@media (min-width: 768px) {
  .bio-card {
    grid-column: span 8;
  }
}

.card-icon {
  position: absolute;
  top: 16px;
  right: 16px;
  font-size: 24px;
  color: #D4A373;
  opacity: 0.2;
}

.card-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 2px;
  color: #8C7B70;
  margin: 0 0 16px 0;
}

.bio-text {
  font-size: 15px;
  font-weight: 500;
  color: #4B3621;
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
  border-top: 1px solid #F2E6D8;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #4B3621;
}

.info-item i {
  color: #D4A373;
}

.info-item a {
  color: #B87333;
  text-decoration: underline;
  text-decoration-color: rgba(212, 163, 115, 0.5);
  text-underline-offset: 2px;
  transition: all 0.2s;
}

.info-item a:hover {
  color: #4B3621;
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
  color: #4ade80;
}

.email-status.warning {
  color: #FF9800;
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
  color: #FF9800;
  line-height: 1.4;
}

.email-tip i {
  margin-top: 2px;
  flex-shrink: 0;
}

/* 3. 公会卡片 */
.guilds-card {
  grid-column: span 12;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.05);
  border: 1px solid #E8DCC8;
  padding: 24px;
  display: flex;
  flex-direction: column;
}

@media (min-width: 768px) {
  .guilds-card {
    grid-column: span 8;
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
  color: #B87333;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-decoration: none;
  transition: color 0.2s;
}

.create-btn:hover {
  color: #4B3621;
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
  color: #B87333;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-decoration: none;
  transition: color 0.2s;
}

.join-btn:hover {
  color: #4B3621;
}

.guilds-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-guilds {
  text-align: center;
  padding: 40px 20px;
  color: #8C7B70;
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
  border: 1px solid #F2E6D8;
  background: rgba(251, 245, 239, 0.3);
  text-decoration: none;
  transition: all 0.2s;
}

.guild-item:hover {
  background: #fff;
  border-color: #D4A373;
  box-shadow: 0 4px 12px rgba(0,0,0,0.05);
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
  color: #fff;
  margin-right: 16px;
  flex-shrink: 0;
  border: 1px solid #E8DCC8;
}

.guild-info {
  flex: 1;
  min-width: 0;
}

.guild-info h3 {
  font-size: 14px;
  font-weight: 700;
  color: #4B3621;
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.guild-info p {
  font-size: 12px;
  color: #8C7B70;
  margin: 0;
}

.guild-badge {
  margin-left: 16px;
  flex-shrink: 0;
}

.role-tag {
  display: inline-flex;
  padding: 4px 8px;
  background: #F2E6D8;
  color: #4B3621;
  border: 1px solid #E8DCC8;
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
  background: #4B3621;
  border-radius: 12px;
  box-shadow: 0 4px 20px -2px rgba(75, 54, 33, 0.2);
  padding: 24px;
  color: #fff;
  position: relative;
  overflow: hidden;
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
  background: #B87333;
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
  background: #4ade80;
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
  color: #D4A373;
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
  color: #fff;
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
</style>
