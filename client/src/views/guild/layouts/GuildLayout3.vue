<script setup lang="ts">
import { computed } from 'vue'
import type { Guild, GuildMember } from '@/api/guild'
import { buildNameStyle } from '@/utils/userNameStyle'

const props = defineProps<{
  guild: Guild
  members: GuildMember[]
  myRole: string
}>()

const emit = defineEmits<{
  (e: 'leave'): void
  (e: 'delete'): void
  (e: 'settings'): void
}>()

const bannerStyle = computed(() => {
  if (props.guild.banner) {
    return { backgroundImage: `url(${props.guild.banner})` }
  }
  return { background: `linear-gradient(135deg, #${props.guild.color || 'B87333'}, #4B3621)` }
})

const factionLabel = computed(() => {
  const map: Record<string, string> = { alliance: '联盟', horde: '部落', neutral: '中立' }
  return map[props.guild.faction] || ''
})

function getRoleLabel(role: string): string {
  const map: Record<string, string> = { owner: '会长', admin: '管理员', member: '成员' }
  return map[role] || role
}
</script>

<template>
  <div class="layout3">
    <!-- Hero Section -->
    <div class="hero-section">
      <div class="hero-image" :style="bannerStyle">
        <div class="hero-overlay"></div>
      </div>

      <div class="hero-content">
        <div class="content-card">
          <div class="badges">
            <span v-if="guild.faction" class="faction-badge" :class="guild.faction">
              {{ factionLabel }}
            </span>
            <span class="online-badge">
              <span class="dot"></span>
              {{ guild.member_count }} 成员
            </span>
          </div>

          <h1 class="guild-name">{{ guild.name }}</h1>
          <p class="guild-desc">{{ guild.slogan || guild.description || '暂无描述' }}</p>

          <div class="actions">
            <button v-if="myRole === 'owner' || myRole === 'admin'" class="btn-settings" @click="emit('settings')">
              <i class="ri-settings-3-line"></i>
              设置
            </button>
            <button v-if="myRole !== 'owner'" class="btn-leave" @click="emit('leave')">退出公会</button>
            <button v-if="myRole === 'owner'" class="btn-danger" @click="emit('delete')">解散公会</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Info Cards -->
    <div class="info-grid">
      <!-- Announcement Card -->
      <div class="info-card announcement-card">
        <div class="card-header">
          <h3>公会信息</h3>
          <span class="invite-code">邀请码: {{ guild.invite_code }}</span>
        </div>
        <div class="info-list">
          <div v-if="guild.server" class="info-item">
            <i class="ri-server-line"></i>
            <span>服务器: {{ guild.server }}</span>
          </div>
          <div class="info-item">
            <i class="ri-book-line"></i>
            <span>剧情数: {{ guild.story_count }}</span>
          </div>
          <div class="info-item">
            <i class="ri-user-line"></i>
            <span>成员数: {{ guild.member_count }}</span>
          </div>
        </div>
      </div>

      <!-- Members Card -->
      <div class="info-card members-card">
        <div class="card-header">
          <h3>核心成员</h3>
        </div>
        <div class="member-avatars">
          <div v-for="m in members.slice(0, 4)" :key="m.id" class="member-avatar" :title="m.username">
            <img v-if="m.avatar" :src="m.avatar" alt="" />
            <span v-else>{{ m.username?.charAt(0) || '?' }}</span>
          </div>
          <div v-if="members.length > 4" class="member-avatar more">
            +{{ members.length - 4 }}
          </div>
        </div>
      </div>

      <!-- Description Card -->
      <div class="info-card desc-card">
        <div class="card-header">
          <h3>公会介绍</h3>
        </div>
        <p class="description">{{ guild.description || '暂无详细介绍' }}</p>
      </div>
    </div>

    <!-- Members List -->
    <div class="members-section">
      <h3>成员列表</h3>
      <div class="members-list">
        <div v-for="m in members" :key="m.id" class="member-row">
          <div class="member-info">
            <div class="avatar">
              <img v-if="m.avatar" :src="m.avatar" alt="" />
              <span v-else>{{ m.username?.charAt(0) || '?' }}</span>
            </div>
            <span class="name" :style="buildNameStyle(m.name_color, m.name_bold)">{{ m.username }}</span>
          </div>
          <span class="role" :class="m.role">{{ getRoleLabel(m.role) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.layout3 {
  min-height: 100vh;
  background: #f8f1eb;
  padding: 24px;
}

/* Hero Section */
.hero-section {
  position: relative;
  margin-bottom: 24px;
}

.hero-image {
  height: 380px;
  border-radius: 32px;
  background-size: cover;
  background-position: center;
  position: relative;
  overflow: hidden;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(75, 54, 33, 0.9), rgba(75, 54, 33, 0.2), transparent);
}

.hero-content {
  position: absolute;
  bottom: 32px;
  left: 32px;
  right: 32px;
}

.content-card {
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 24px;
  border-radius: 24px;
  max-width: 600px;
}

.badges {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.faction-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
}

.faction-badge.alliance { background: linear-gradient(135deg, #1e5aa8, #3b82f6); }
.faction-badge.horde { background: linear-gradient(135deg, #991b1b, #dc2626); }
.faction-badge.neutral { background: linear-gradient(135deg, #6b7280, #9ca3af); }

.online-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
}

.online-badge .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #4ade80;
}

.guild-name {
  font-size: 42px;
  font-weight: 700;
  color: #fff;
  margin: 0 0 12px 0;
  text-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.guild-desc {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.8);
  margin: 0 0 20px 0;
  line-height: 1.6;
}

.actions {
  display: flex;
  gap: 12px;
}

.actions button {
  padding: 12px 24px;
  border-radius: 24px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-settings {
  background: #fff;
  color: #4B3621;
}

.btn-settings:hover {
  transform: scale(1.05);
}

.btn-leave {
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.btn-danger {
  background: #dc2626;
  color: #fff;
}

/* Info Grid */
.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.info-card {
  background: #fff;
  border-radius: 28px;
  padding: 24px;
  box-shadow: 0 8px 16px -4px rgba(44, 24, 16, 0.1);
  border: 1px solid #E8DCCF;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.card-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: #2C1810;
  margin: 0;
}

.invite-code {
  font-size: 12px;
  color: #8C7B70;
  background: #f8f1eb;
  padding: 4px 12px;
  border-radius: 8px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #4B3621;
}

.info-item i {
  color: #B87333;
}

/* Members Card */
.member-avatars {
  display: flex;
  gap: -8px;
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  border: 2px solid #fff;
  margin-left: -8px;
  transition: transform 0.2s;
  overflow: hidden;
}

.member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-avatar:first-child {
  margin-left: 0;
}

.member-avatar:hover {
  transform: translateY(-4px) scale(1.1);
  z-index: 10;
}

.member-avatar.more {
  background: #f8f1eb;
  color: #8C7B70;
  font-size: 12px;
}

/* Description */
.description {
  font-size: 14px;
  color: #4B3621;
  line-height: 1.8;
  margin: 0;
}

/* Members Section */
.members-section {
  background: #fff;
  border-radius: 28px;
  padding: 24px;
  box-shadow: 0 8px 16px -4px rgba(44, 24, 16, 0.1);
  border: 1px solid #E8DCCF;
}

.members-section h3 {
  font-size: 18px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 16px 0;
}

.members-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.member-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-radius: 12px;
  transition: background 0.2s;
}

.member-row:hover {
  background: #f8f1eb;
}

.member-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.member-info .avatar {
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

.member-info .avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-info .name {
  font-size: 14px;
  font-weight: 500;
  color: #2C1810;
}

.role {
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 12px;
  font-weight: 500;
}

.role.owner {
  background: #fef3c7;
  color: #92400e;
}

.role.admin {
  background: #dbeafe;
  color: #1e40af;
}

.role.member {
  background: #f3f4f6;
  color: #6b7280;
}
</style>
