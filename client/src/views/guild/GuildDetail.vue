<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGuild, leaveGuild, deleteGuild, listGuildMembers, updateGuild, uploadGuildBanner, listGuildApplications, applyGuild, type Guild, type GuildMember } from '@/api/guild'
import { useDialog } from '@/composables/useDialog'
import RModal from '@/components/RModal.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import LazyBgImage from '@/components/LazyBgImage.vue'

const route = useRoute()
const router = useRouter()
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
  show_to_visitors: true,
  show_to_members: true,
  auto_approve: false
})
const bannerFile = ref<File | null>(null)
const bannerPreview = ref('')
const heroBannerInput = ref<HTMLInputElement | null>(null)

const guildId = Number(route.params.id)

const isAdmin = computed(() => myRole.value === 'owner' || myRole.value === 'admin')

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
    title: '退出公会',
    message: '确定要退出公会吗？',
    type: 'warning'
  })
  if (!confirmed) return
  try {
    await leaveGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    await alert({ title: '退出失败', message: e.message || '退出失败', type: 'error' })
  }
}

async function handleApply() {
  try {
    const res = await applyGuild(guildId)
    // 检查是否为自动加入
    if ((res as any).auto_approved) {
      await alert({
        title: '加入成功',
        message: '您已成功加入公会！',
        type: 'success'
      })
    } else {
      await alert({
        title: '申请已提交',
        message: '您的入会申请已提交，请等待管理员审核',
        type: 'success'
      })
    }
    await loadGuild() // 重新加载公会信息
  } catch (e: any) {
    await alert({ title: '申请失败', message: e.message || '申请失败', type: 'error' })
  }
}

async function handleDelete() {
  const confirmed = await confirm({
    title: '解散公会',
    message: '确定要解散公会吗？此操作不可恢复！',
    type: 'error'
  })
  if (!confirmed) return
  try {
    await deleteGuild(guildId)
    router.push('/guild')
  } catch (e: any) {
    await alert({ title: '解散失败', message: e.message || '解散失败', type: 'error' })
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
    show_to_visitors: guild.value.show_to_visitors ?? true,
    show_to_members: guild.value.show_to_members ?? true,
    auto_approve: guild.value.auto_approve ?? false
  }
  bannerPreview.value = guild.value.banner || ''
  bannerFile.value = null
  showSettingsModal.value = true
}

function handleBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    const file = input.files[0]
    if (file.size > 20 * 1024 * 1024) {
      alert({ title: '文件过大', message: '头图文件不能超过20MB', type: 'error' })
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

function triggerBannerUpload() {
  heroBannerInput.value?.click()
}

async function handleHeroBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || !input.files[0] || !guild.value) return

  const file = input.files[0]
  if (file.size > 20 * 1024 * 1024) {
    await alert({ title: '文件过大', message: '头图文件不能超过20MB', type: 'error' })
    return
  }

  try {
    const res = await uploadGuildBanner(guildId, file)
    guild.value.banner = res.banner
  } catch (e: any) {
    await alert({ title: '上传失败', message: e.message || '上传失败', type: 'error' })
  }
  input.value = ''
}

async function saveSettings() {
  if (!guild.value) return
  saving.value = true
  try {
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
      show_to_visitors: editForm.value.show_to_visitors,
      show_to_members: editForm.value.show_to_members,
      auto_approve: editForm.value.auto_approve
    })
    // 刷新数据
    await loadGuild()
    showSettingsModal.value = false
  } catch (e: any) {
    await alert({ title: '保存失败', message: e.message || '保存失败', type: 'error' })
  } finally {
    saving.value = false
  }
}

function getRoleLabel(role: string): string {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role] || role
}

function getFactionLabel(f: string): string {
  const map: Record<string, string> = { alliance: '联盟', horde: '部落', neutral: '中立' }
  return map[f] || ''
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
    <div v-if="loading" class="loading">加载中...</div>

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
              <i class="ri-team-line"></i> 管理公会
              <span v-if="pendingApplicationCount > 0" class="pending-badge">{{ pendingApplicationCount }}</span>
            </button>
            <button v-if="isAdmin" class="primary-btn" @click="openSettings">
              <i class="ri-settings-3-line"></i> 设置
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
                  <div class="badges">
                    <span v-if="guild.faction" class="badge faction" :class="guild.faction">
                      {{ getFactionLabel(guild.faction) }}
                    </span>
                    <span class="badge members">
                      <i class="ri-user-line"></i> {{ guild.member_count }} 成员
                    </span>
                  </div>
                  <h1>{{ guild.name }}</h1>
                  <p class="slogan">{{ guild.slogan || guild.description || '暂无描述' }}</p>

                  <div class="hero-actions">
                    <div v-if="!myRole" class="apply-action">
                      <button class="btn-outline" @click="handleApply">
                        申请入会
                      </button>
                      <span v-if="guild.auto_approve" class="auto-approve-hint">
                        <i class="ri-check-line"></i> 无需审核
                      </span>
                    </div>
                    <button v-else-if="myRole !== 'owner'" class="btn-outline" @click="handleLeave">
                      退出公会
                    </button>
                    <button v-if="isAdmin" class="btn-outline" @click="triggerBannerUpload">
                      <i class="ri-image-edit-line"></i> 更换头图
                    </button>
                  </div>
                  <input ref="heroBannerInput" type="file" accept="image/*" hidden @change="handleHeroBannerSelect" />
                </div>
              </div>
            </div>

            <!-- 侧边公告卡片 -->
            <div class="announcement-card">
              <div class="card-header">
                <h3>公会信息</h3>
                <span class="tag">INFO</span>
              </div>
              <ul class="info-list">
                <li v-if="isAdmin">
                  <span class="label">邀请码</span>
                  <span class="value code">{{ guild.invite_code }}</span>
                </li>
                <li>
                  <span class="label">剧情数</span>
                  <span class="value">{{ guild.story_count }}</span>
                </li>
                <li v-if="myRole">
                  <span class="label">我的身份</span>
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
              <h3>公会介绍</h3>
              <p>{{ guild.description || '暂无详细介绍，会长可以在设置中添加。' }}</p>
            </div>

            <!-- 公会设定 -->
            <div class="bento-card lore-card" :class="{ clickable: guild.lore }" @click="guild.lore && (showLoreModal = true)">
              <div class="card-icon"><i class="ri-quill-pen-line"></i></div>
              <h3>公会设定 <i v-if="guild.lore" class="ri-arrow-right-s-line"></i></h3>
              <div v-if="guild.lore" class="lore-preview" v-html="guild.lore"></div>
              <p v-else class="empty-hint">暂无公会设定，会长可以在设置中添加。</p>
            </div>

            <!-- 成员列表 -->
            <div v-if="myRole" class="bento-card members-card">
              <div class="card-header">
                <h3>成员列表</h3>
                <span class="count">{{ members.length }}人</span>
              </div>
              <div class="member-list">
                <div v-for="m in members" :key="m.id" class="member-item">
                  <div class="avatar">
                    <img v-if="m.avatar" :src="m.avatar" alt="" loading="lazy" />
                    <span v-else>{{ m.username?.charAt(0) || '?' }}</span>
                  </div>
                  <div class="info">
                    <span class="name">{{ m.username }}</span>
                    <span class="role" :class="m.role">{{ getRoleLabel(m.role) }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 快捷操作 -->
            <div class="bento-card actions-card">
              <h3>快捷操作</h3>
              <div class="quick-actions">
                <button class="action-btn" @click="goToEvents">
                  <i class="ri-calendar-event-line"></i>
                  <span>公会帖子</span>
                </button>
                <button class="action-btn" @click="goToStories">
                  <i class="ri-book-2-line"></i>
                  <span>过往剧情</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 公会设定详情弹窗 -->
    <RModal v-model="showLoreModal" title="公会设定" width="800px">
      <div class="article-content" v-html="guild?.lore"></div>
    </RModal>

    <!-- 设置弹窗 -->
    <RModal v-model="showSettingsModal" title="公会设置" width="560px">
      <div class="settings-form">
        <!-- 头图上传 -->
        <div class="form-section">
          <label>公会头图</label>
          <div
            class="banner-upload"
            :style="bannerPreview
              ? { backgroundImage: `url(${bannerPreview})` }
              : { background: `linear-gradient(135deg, #${editForm.color || 'B87333'}, #4B3621)` }"
            @click="($refs.bannerInput as HTMLInputElement).click()"
          >
            <div class="upload-hint">
              <i class="ri-image-add-line"></i>
              <span>点击上传头图</span>
            </div>
          </div>
          <input ref="bannerInput" type="file" accept="image/*" hidden @change="handleBannerSelect" />
        </div>

        <!-- 基本信息 -->
        <div class="form-section">
          <label>公会名称</label>
          <RInput v-model="editForm.name" placeholder="输入公会名称" />
        </div>

        <div class="form-section">
          <label>公会标语</label>
          <RInput v-model="editForm.slogan" placeholder="一句话介绍公会" />
        </div>

        <div class="form-section">
          <label>公会介绍</label>
          <textarea v-model="editForm.description" placeholder="详细介绍公会..." rows="3"></textarea>
        </div>

        <div class="form-row">
          <div class="form-section">
            <label>阵营</label>
            <select v-model="editForm.faction">
              <option value="">请选择</option>
              <option value="alliance">联盟</option>
              <option value="horde">部落</option>
              <option value="neutral">中立</option>
            </select>
          </div>
          <div class="form-section">
            <label>主题色</label>
            <input
              type="color"
              :value="'#' + editForm.color"
              @input="editForm.color = ($event.target as HTMLInputElement).value.replace('#', '')"
            />
          </div>
        </div>

        <div class="form-section lore-section">
          <label>公会设定</label>
          <TiptapEditor v-model="editForm.lore" placeholder="编写公会的背景设定、历史故事..." />
        </div>

        <!-- 隐私设置 -->
        <div class="form-section privacy-section">
          <label>隐私设置</label>
          <div class="privacy-toggles">
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">向访客展示公会内容</span>
                <span class="toggle-hint">非公会成员可以查看公会剧情和帖子</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="editForm.show_to_visitors" />
                <span class="slider"></span>
              </label>
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">向普通成员展示公会内容</span>
                <span class="toggle-hint">普通成员可以查看公会剧情和帖子（管理员始终可见）</span>
              </div>
              <label class="switch">
                <input type="checkbox" v-model="editForm.show_to_members" />
                <span class="slider"></span>
              </label>
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">自动审核</span>
                <span class="toggle-hint">开启后新成员无需审核即可直接加入公会</span>
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
        <RButton @click="showSettingsModal = false">取消</RButton>
        <RButton type="primary" :loading="saving" @click="saveSettings">保存</RButton>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.guild-detail {
  min-height: 100vh;
  background: #EED9C4;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  font-size: 16px;
  color: #8C7B70;
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
  border-bottom: 1px solid #E8DCCF;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.breadcrumb .path {
  color: #8C7B70;
}

.breadcrumb .sep {
  color: #D4A373;
}

.breadcrumb .current {
  color: #2C1810;
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
  color: #8C7B70;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  transition: all 0.2s;
}

.icon-btn:hover {
  background: rgba(128, 64, 48, 0.1);
  color: #804030;
}

.primary-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.primary-btn:hover {
  background: #6B3626;
}

.manage-btn {
  position: relative;
}

.pending-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  background: #FF9800;
  color: #fff;
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
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.4);
  color: #fff;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-outline:hover {
  background: rgba(255, 255, 255, 0.1);
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
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 12px rgba(44, 24, 16, 0.08);
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
  color: #2C1810;
  margin: 0;
}

.announcement-card .tag {
  font-size: 10px;
  font-weight: 700;
  color: #804030;
  background: rgba(128, 64, 48, 0.1);
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
  color: #8C7B70;
}

.info-list .value {
  color: #2C1810;
  font-weight: 500;
}

.info-list .value.code {
  font-family: monospace;
  background: #f5f0eb;
  padding: 4px 8px;
  border-radius: 4px;
}

.info-list .value.role {
  color: #804030;
}

/* Bento Grid */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.bento-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 4px 12px rgba(44, 24, 16, 0.08);
}

.bento-card .card-icon {
  width: 40px;
  height: 40px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #804030;
  font-size: 20px;
  margin-bottom: 16px;
}

.bento-card h3 {
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 12px 0;
}

.bento-card p {
  font-size: 14px;
  color: #8C7B70;
  line-height: 1.6;
  margin: 0;
}

/* Lore Card */
.lore-card {
  grid-column: span 2;
}

.lore-content {
  font-size: 14px;
  color: #4B3621;
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
  color: #8C7B70;
  font-style: italic;
}

/* Clickable Card */
.bento-card.clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.bento-card.clickable:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(44, 24, 16, 0.12);
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
  color: #4B3621;
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
  background: linear-gradient(transparent, #fff);
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
  color: #8C7B70;
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
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
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
  color: #2C1810;
}

.member-item .role {
  font-size: 12px;
  color: #8C7B70;
}

.member-item .role.owner {
  color: #D4A373;
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
  background: #f5f0eb;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: #ebe3db;
}

.action-btn i {
  font-size: 24px;
  color: #804030;
}

.action-btn span {
  font-size: 12px;
  color: #2C1810;
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
  color: #2C1810;
}

.form-section textarea {
  padding: 12px;
  border: 1px solid #E8DCCF;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.form-section textarea:focus {
  outline: none;
  border-color: #804030;
}

.form-section select {
  padding: 12px;
  border: 1px solid #E8DCCF;
  border-radius: 8px;
  font-size: 14px;
  background: #fff;
}

.form-section select:focus {
  outline: none;
  border-color: #804030;
}

.form-section input[type="color"] {
  width: 60px;
  height: 36px;
  border: 1px solid #E8DCCF;
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
  border: 1px solid #E8DCCF;
  border-radius: 8px;
}

.lore-section :deep(.tiptap-editor:focus-within) {
  border-color: #804030;
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
  border: 2px dashed #E8DCCF;
  transition: all 0.2s;
}

.banner-upload:hover {
  border-color: #804030;
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
  color: #fff;
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

/* Article Content - 富文本渲染 */
.article-content {
  font-size: 15px;
  line-height: 1.8;
  color: #2C1810;
}

.article-content :deep(h1),
.article-content :deep(h2),
.article-content :deep(h3) {
  margin: 24px 0 12px 0;
  font-weight: 600;
  color: #2C1810;
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
  border-left: 4px solid #D4A373;
  background: #f5f0eb;
  color: #4B3621;
}

.article-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 16px 0;
}

.article-content :deep(a) {
  color: #804030;
  text-decoration: underline;
}

.article-content :deep(code) {
  background: #f5f0eb;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
}

.article-content :deep(pre) {
  background: #2C1810;
  color: #EED9C4;
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
  border-top: 1px solid #E8DCCF;
  margin: 24px 0;
}

/* Privacy Settings */
.privacy-section {
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px solid #E8DCCF;
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
  background: #f5f0eb;
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
  color: #2C1810;
}

.toggle-hint {
  font-size: 12px;
  color: #8C7B70;
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
  background-color: #E8DCCF;
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
  background-color: #804030;
}

input:checked + .slider:before {
  transform: translateX(22px);
}
</style>
