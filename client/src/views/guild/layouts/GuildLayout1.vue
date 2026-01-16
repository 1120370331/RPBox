<script setup lang="ts">
import { computed } from 'vue'
import type { Guild, GuildMember } from '@/api/guild'

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
  <div class="layout1">
    <!-- Header -->
    <header class="header">
      <div class="guild-identity">
        <div class="guild-icon" :style="{ background: '#' + (guild.color || 'B87333') }">
          {{ guild.name.charAt(0) }}
        </div>
        <div class="guild-info">
          <div class="badges">
            <span class="level-badge">LV. {{ guild.story_count || 1 }}</span>
            <span v-if="guild.server" class="server-info">
              <i class="ri-map-pin-line"></i> {{ guild.server }}
            </span>
          </div>
          <h1>{{ guild.name }}</h1>
          <p>{{ guild.slogan || guild.description || '暂无描述' }}</p>
        </div>
      </div>
      <div class="header-actions">
        <button v-if="myRole === 'owner' || myRole === 'admin'" class="btn-secondary" @click="emit('settings')">
          公会设置
        </button>
        <button v-if="myRole !== 'owner'" class="btn-danger" @click="emit('leave')">退出</button>
        <button v-if="myRole === 'owner'" class="btn-danger" @click="emit('delete')">解散</button>
      </div>
    </header>

    <!-- Stats Row -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-label">活跃成员</div>
        <div class="stat-value">{{ guild.member_count }}<span class="stat-sub">/ 200</span></div>
        <div v-if="guild.faction" class="stat-badge" :class="guild.faction">{{ factionLabel }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">剧情归档</div>
        <div class="stat-value">{{ guild.story_count }}<span class="stat-sub">篇</span></div>
      </div>
      <div class="stat-card">
        <div class="stat-label">邀请码</div>
        <div class="stat-value invite-code">{{ guild.invite_code }}</div>
      </div>
    </div>

    <!-- Main Grid -->
    <div class="notion-grid">
      <!-- Bulletin Board -->
      <div class="panel bulletin-panel">
        <div class="panel-header">
          <h2><i class="ri-newspaper-line"></i> 公告板</h2>
        </div>
        <div class="callout">
          <div class="callout-header">
            <span class="callout-title"><i class="ri-star-fill"></i> 公会介绍</span>
          </div>
          <p>{{ guild.description || '暂无详细介绍，会长可以在设置中添加公会介绍。' }}</p>
        </div>
      </div>

      <!-- Members Panel -->
      <div class="panel members-panel">
        <div class="panel-header">
          <h2>成员列表</h2>
        </div>
        <div class="member-list">
          <div v-for="m in members" :key="m.id" class="member-item">
            <div class="member-avatar">
              <img v-if="m.avatar" :src="m.avatar" alt="" />
              <span v-else>{{ m.username?.charAt(0) || '?' }}</span>
            </div>
            <div class="member-info">
              <div class="member-name">{{ m.username }}</div>
              <div class="member-role">{{ getRoleLabel(m.role) }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Leaderboard -->
      <div class="panel leaderboard-panel">
        <h2>本周贡献榜</h2>
        <ul class="leaderboard">
          <li v-for="(m, i) in members.slice(0, 3)" :key="m.id">
            <span class="rank" :class="{ gold: i === 0 }">{{ i + 1 }}</span>
            <span class="name">{{ m.username }}</span>
            <span class="score">{{ 2400 - i * 450 }}</span>
          </li>
        </ul>
      </div>
    </div>

    <!-- Quick Links -->
    <div class="quick-links">
      <h3>资源导航</h3>
      <div class="links-grid">
        <a href="#" class="link-card">
          <i class="ri-book-2-line"></i>
          <span>攻略百科</span>
        </a>
        <a href="#" class="link-card">
          <i class="ri-team-line"></i>
          <span>成员名册</span>
        </a>
        <a href="#" class="link-card">
          <i class="ri-money-cny-box-line"></i>
          <span>财务报表</span>
        </a>
        <a href="#" class="link-card">
          <i class="ri-calendar-event-line"></i>
          <span>活动日历</span>
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.layout1 {
  min-height: 100vh;
  background: #EED9C4;
  padding: 32px;
}

/* Header */
.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #DCC8B8;
}

.guild-identity {
  display: flex;
  gap: 24px;
  align-items: center;
}

.guild-icon {
  width: 96px;
  height: 96px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 40px;
  font-weight: 700;
  color: #fff;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.2);
  border: 3px solid #fff;
}

.guild-info h1 {
  font-size: 36px;
  color: #2C1810;
  margin: 8px 0;
  font-weight: 700;
}

.guild-info p {
  color: #8C7B70;
  font-size: 14px;
  max-width: 400px;
}

.badges {
  display: flex;
  gap: 12px;
  align-items: center;
}

.level-badge {
  background: #804030;
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.server-info {
  font-size: 12px;
  color: #8C7B70;
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.header-actions button {
  padding: 10px 20px;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-secondary {
  background: #fff;
  color: #2C1810;
  border: 1px solid #E8DCCF;
}

.btn-secondary:hover {
  background: #FAF6F2;
}

.btn-danger {
  background: #804030;
  color: #fff;
}

.btn-danger:hover {
  background: #6B3626;
}

/* Stats Row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: #fff;
  padding: 20px;
  border-radius: 4px;
  border: 1px solid rgba(75, 54, 33, 0.08);
  box-shadow: 0 1px 2px rgba(75, 54, 33, 0.05);
  position: relative;
}

.stat-label {
  font-size: 12px;
  color: #8C7B70;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #2C1810;
}

.stat-sub {
  font-size: 16px;
  color: #8C7B70;
  margin-left: 4px;
}

.stat-badge {
  position: absolute;
  top: 16px;
  right: 16px;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
}

.stat-badge.alliance { background: #1e5aa8; }
.stat-badge.horde { background: #991b1b; }
.stat-badge.neutral { background: #6b7280; }

.invite-code {
  font-family: monospace;
  font-size: 24px;
  letter-spacing: 2px;
}

/* Notion Grid */
.notion-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 24px;
  margin-bottom: 24px;
}

.panel {
  background: #fff;
  border-radius: 4px;
  padding: 24px;
  border: 1px solid rgba(75, 54, 33, 0.08);
  box-shadow: 0 1px 2px rgba(75, 54, 33, 0.05);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.panel-header h2 {
  font-size: 16px;
  color: #2C1810;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.callout {
  background: rgba(212, 163, 115, 0.15);
  border-left: 4px solid #804030;
  padding: 16px;
  border-radius: 0 4px 4px 0;
}

.callout-header {
  margin-bottom: 8px;
}

.callout-title {
  font-weight: 600;
  color: #804030;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.callout p {
  font-size: 14px;
  color: #2C1810;
  line-height: 1.6;
  margin: 0;
}

/* Members Panel */
.members-panel {
  grid-row: span 2;
}

.member-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 4px;
  transition: background 0.2s;
}

.member-item:hover {
  background: #F9F4EF;
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #EED9C4;
  color: #4B3621;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  overflow: hidden;
}

.member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-info {
  flex: 1;
}

.member-name {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
}

.member-role {
  font-size: 12px;
  color: #8C7B70;
}

/* Leaderboard */
.leaderboard-panel {
  background: #4B3621;
  color: #fff;
}

.leaderboard-panel h2 {
  font-size: 14px;
  color: #EED9C4;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin: 0 0 16px 0;
}

.leaderboard {
  list-style: none;
  padding: 0;
  margin: 0;
}

.leaderboard li {
  display: flex;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.leaderboard li:last-child {
  border-bottom: none;
}

.leaderboard .rank {
  width: 24px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.6);
}

.leaderboard .rank.gold {
  color: #D4A373;
}

.leaderboard .name {
  flex: 1;
  color: rgba(255, 255, 255, 0.9);
}

.leaderboard .score {
  font-family: monospace;
  color: #EED9C4;
}

/* Quick Links */
.quick-links h3 {
  font-size: 12px;
  color: #8C7B70;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin: 0 0 16px 4px;
}

.links-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.link-card {
  background: #fff;
  border: 1px dashed #E8DCCF;
  border-radius: 4px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-decoration: none;
  transition: all 0.2s;
}

.link-card:hover {
  border-color: #D4A373;
  background: #fff;
}

.link-card i {
  font-size: 24px;
  color: #2C1810;
}

.link-card span {
  font-size: 14px;
  color: #2C1810;
  font-weight: 500;
}
</style>
