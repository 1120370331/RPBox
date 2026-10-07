<script setup lang="ts">
import { computed } from 'vue'
import { guildColorStyle } from '../guildColorStyle'
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
        <div class="guild-icon" :style="guildColorStyle(guild.color || 'B87333')">
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
      <div class="guild-panel bulletin-panel">
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
      <div class="guild-panel members-panel">
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
              <div class="member-name" :style="buildNameStyle(m.name_color, m.name_bold)">{{ m.username }}</div>
              <div class="member-role">{{ getRoleLabel(m.role) }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Leaderboard -->
      <div class="guild-panel leaderboard-panel">
        <h2>本周贡献榜</h2>
        <ul class="leaderboard">
          <li v-for="(m, i) in members.slice(0, 3)" :key="m.id">
            <span class="rank" :class="{ gold: i === 0 }">{{ i + 1 }}</span>
            <span class="name" :style="buildNameStyle(m.name_color, m.name_bold, ['gradientStart', 'gradientEnd'])">{{ m.username }}</span>
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
  background: var(--color-background);
  padding: 32px;
  color: var(--color-text-main);
}

/* Header */
.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--color-border);
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
  box-shadow: var(--shadow-md);
  border: 3px solid var(--color-panel-bg);
}

.guild-info h1 {
  font-size: 36px;
  color: var(--color-text-main);
  margin: 8px 0;
  font-weight: 700;
}

.guild-info p {
  color: var(--color-text-secondary);
  font-size: 14px;
  max-width: 400px;
}

.badges {
  display: flex;
  gap: 12px;
  align-items: center;
}

.level-badge {
  background: var(--badge-bg);
  color: var(--btn-primary-text);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.server-info {
  font-size: 12px;
  color: var(--color-text-secondary);
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
  background: var(--btn-secondary-bg);
  color: var(--btn-secondary-text);
  border: 1px solid var(--btn-outline-border);
}

.btn-secondary:hover {
  background: var(--btn-secondary-hover);
}

.btn-danger {
  background: var(--btn-danger-bg);
  color: var(--btn-danger-text);
}

.btn-danger:hover {
  background: var(--btn-danger-hover);
}

/* Stats Row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--color-panel-bg);
  padding: 20px;
  border-radius: 4px;
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
  position: relative;
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text-main);
}

.stat-sub {
  font-size: 16px;
  color: var(--color-text-secondary);
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

.guild-panel {
  background: var(--color-panel-bg);
  border-radius: 4px;
  padding: 24px;
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.panel-header h2 {
  font-size: 16px;
  color: var(--color-text-main);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.callout {
  background: var(--color-primary-light);
  border-left: 4px solid var(--color-primary);
  padding: 16px;
  border-radius: 0 4px 4px 0;
}

.callout-header {
  margin-bottom: 8px;
}

.callout-title {
  font-weight: 600;
  color: var(--color-primary);
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.callout p {
  font-size: 14px;
  color: var(--color-text-main);
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
  background: var(--color-card-bg-hover);
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-card-bg);
  color: var(--color-text-main);
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
  color: var(--color-text-main);
}

.member-role {
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* Leaderboard */
.leaderboard-panel {
  background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
  color: var(--gradient-text);
}

.leaderboard-panel h2 {
  font-size: 14px;
  color: var(--gradient-text);
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
  border-bottom: 1px solid var(--gradient-border);
}

.leaderboard li:last-child {
  border-bottom: none;
}

.leaderboard .rank {
  width: 24px;
  font-weight: 700;
  color: var(--gradient-text-muted);
}

.leaderboard .rank.gold {
  color: var(--gradient-text);
}

.leaderboard .name {
  flex: 1;
  color: var(--gradient-text);
}

.leaderboard .score {
  font-family: monospace;
  color: var(--gradient-text);
}

/* Quick Links */
.quick-links h3 {
  font-size: 12px;
  color: var(--color-text-secondary);
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
  background: var(--color-panel-bg);
  border: 1px dashed var(--color-border);
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
  border-color: var(--color-border-hover);
  background: var(--color-card-bg-hover);
}

.link-card i {
  font-size: 24px;
  color: var(--color-text-main);
}

.link-card span {
  font-size: 14px;
  color: var(--color-text-main);
  font-weight: 500;
}
</style>
