<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '../../stores/user'
import request from '../../api/request'

const route = useRoute()
const userStore = useUserStore()

const userId = computed(() => route.params.id as string)
const isOwnProfile = computed(() => userStore.user?.id === Number(userId.value))

const userProfile = ref<any>(null)
const userGuilds = ref<any[]>([])
const loading = ref(true)
const editMode = ref(false)

// 表单数据
const formData = ref({
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
  } catch (error: any) {
    console.error('保存失败:', error)
  }
}

function cancelEdit() {
  editMode.value = false
  formData.value = {
    bio: userProfile.value?.bio || '',
    location: userProfile.value?.location || '',
    website: userProfile.value?.website || ''
  }
}
</script>

<template>
  <div class="user-profile">
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="userProfile" class="profile-container">
      <!-- 用户信息卡片 -->
      <div class="profile-card">
        <div class="profile-header">
          <div class="avatar">
            {{ userProfile.username?.charAt(0)?.toUpperCase() || 'U' }}
          </div>
          <div class="user-info">
            <h2>{{ userProfile.username }}</h2>
            <span class="role-badge" :class="userProfile.role">{{ userProfile.role }}</span>
          </div>
          <button v-if="isOwnProfile" @click="editMode = !editMode" class="edit-btn">
            <i :class="editMode ? 'ri-close-line' : 'ri-edit-line'"></i>
            {{ editMode ? '取消' : '编辑资料' }}
          </button>
        </div>

        <!-- 统计数据 -->
        <div class="stats">
          <div class="stat-item">
            <div class="stat-value">{{ userProfile.post_count || 0 }}</div>
            <div class="stat-label">帖子</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ userProfile.story_count || 0 }}</div>
            <div class="stat-label">剧情</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ userProfile.profile_count || 0 }}</div>
            <div class="stat-label">人物卡</div>
          </div>
        </div>

        <!-- 个人资料 -->
        <div class="profile-details">
          <div v-if="!editMode" class="view-mode">
            <div v-if="userProfile.bio" class="detail-item">
              <i class="ri-file-text-line"></i>
              <span>{{ userProfile.bio }}</span>
            </div>
            <div v-if="userProfile.location" class="detail-item">
              <i class="ri-map-pin-line"></i>
              <span>{{ userProfile.location }}</span>
            </div>
            <div v-if="userProfile.website" class="detail-item">
              <i class="ri-link"></i>
              <a :href="userProfile.website" target="_blank">{{ userProfile.website }}</a>
            </div>
          </div>

          <div v-else class="edit-mode">
            <div class="form-group">
              <label>个人简介</label>
              <textarea v-model="formData.bio" placeholder="介绍一下自己..." maxlength="500"></textarea>
            </div>
            <div class="form-group">
              <label>地区</label>
              <input v-model="formData.location" type="text" placeholder="你的地区" maxlength="100">
            </div>
            <div class="form-group">
              <label>个人网站</label>
              <input v-model="formData.website" type="url" placeholder="https://..." maxlength="256">
            </div>
            <div class="form-actions">
              <button @click="saveProfile" class="save-btn">保存</button>
              <button @click="cancelEdit" class="cancel-btn">取消</button>
            </div>
          </div>
        </div>
      </div>

      <!-- 公会列表 -->
      <div class="guilds-section">
        <div class="section-header">
          <h3>加入的公会</h3>
          <router-link v-if="isOwnProfile" to="/guild/create" class="create-guild-btn">
            <i class="ri-add-line"></i>
            创建公会
          </router-link>
        </div>
        <p class="section-tip" v-if="isOwnProfile">创建公会需要版主审核通过后才能显示</p>
        <div v-if="userGuilds.length === 0" class="empty">
          <i class="ri-shield-line"></i>
          <p>还没有加入任何公会</p>
        </div>
        <div v-else class="guilds-list">
          <router-link
            v-for="guild in userGuilds"
            :key="guild.id"
            :to="`/guild/${guild.id}`"
            class="guild-card"
            :class="{ pending: guild.status === 'pending' }"
          >
            <div class="guild-icon" :style="{ background: guild.color || '#D4A373' }">
              {{ guild.name?.charAt(0) || 'G' }}
            </div>
            <div class="guild-info">
              <h4>
                {{ guild.name }}
                <span v-if="guild.status === 'pending'" class="pending-badge">待审核</span>
              </h4>
              <p>{{ guild.member_count }} 成员 · {{ guild.role === 'owner' ? '会长' : guild.role }}</p>
            </div>
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.user-profile {
  max-width: 1000px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #8C7B70;
}

.profile-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.profile-card {
  background: #FBF5EF;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 24px;
}

.avatar {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: bold;
  color: #FFF;
  border: 3px solid rgba(255,255,255,0.3);
}

.user-info {
  flex: 1;
}

.user-info h2 {
  margin: 0 0 8px 0;
  color: #4B3621;
  font-size: 24px;
}

.role-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.role-badge.user {
  background: #E8DCC8;
  color: #8C7B70;
}

.role-badge.moderator {
  background: #B87333;
  color: #FFF;
}

.role-badge.admin {
  background: #804030;
  color: #FFF;
}

.edit-btn {
  padding: 10px 20px;
  background: #D4A373;
  color: #FFF;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  transition: all 0.3s;
}

.edit-btn:hover {
  background: #B87333;
}

.stats {
  display: flex;
  gap: 32px;
  padding: 24px 0;
  border-top: 1px solid #E8DCC8;
  border-bottom: 1px solid #E8DCC8;
}

.stat-item {
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #4B3621;
}

.stat-label {
  font-size: 14px;
  color: #8C7B70;
  margin-top: 4px;
}

.profile-details {
  margin-top: 24px;
}

.view-mode {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #4B3621;
}

.detail-item i {
  font-size: 18px;
  color: #D4A373;
}

.detail-item a {
  color: #D4A373;
  text-decoration: none;
}

.detail-item a:hover {
  text-decoration: underline;
}

.edit-mode {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  font-weight: 600;
  color: #4B3621;
}

.form-group input,
.form-group textarea {
  padding: 12px;
  border: 1px solid #E8DCC8;
  border-radius: 8px;
  font-size: 14px;
  background: #FFF;
  color: #4B3621;
}

.form-group textarea {
  min-height: 100px;
  resize: vertical;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.save-btn,
.cancel-btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s;
}

.save-btn {
  background: #D4A373;
  color: #FFF;
}

.save-btn:hover {
  background: #B87333;
}

.cancel-btn {
  background: #E8DCC8;
  color: #4B3621;
}

.cancel-btn:hover {
  background: #D4C4B0;
}

.guilds-section {
  background: #FBF5EF;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.guilds-section h3 {
  margin: 0;
  color: #4B3621;
  font-size: 20px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.create-guild-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #D4A373;
  color: #FFF;
  border-radius: 8px;
  text-decoration: none;
  font-size: 14px;
  transition: all 0.3s;
}

.create-guild-btn:hover {
  background: #B87333;
}

.section-tip {
  font-size: 12px;
  color: #8C7B70;
  margin: 0 0 16px 0;
}

.empty {
  text-align: center;
  padding: 40px;
  color: #8C7B70;
}

.empty i {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.guilds-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.guild-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #FFF;
  border-radius: 12px;
  text-decoration: none;
  transition: all 0.3s;
  border: 1px solid #E8DCC8;
}

.guild-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.guild-card.pending {
  background: #FFF8E1;
  border-color: #FFB74D;
}

.pending-badge {
  display: inline-block;
  padding: 2px 8px;
  background: #FF9800;
  color: #FFF;
  font-size: 10px;
  border-radius: 4px;
  margin-left: 8px;
  font-weight: normal;
}

.guild-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  color: #FFF;
}

.guild-info h4 {
  margin: 0 0 4px 0;
  color: #4B3621;
  font-size: 16px;
}

.guild-info p {
  margin: 0;
  font-size: 12px;
  color: #8C7B70;
}
</style>
