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
  <div class="layout4">
    <!-- Header -->
    <header class="bento-header">
      <div class="breadcrumb">
        <span class="server">SERVER: {{ guild.server || 'UNKNOWN' }}</span>
        <span class="sep">/</span>
        <span class="name">{{ guild.name.toUpperCase() }}</span>
      </div>
      <div class="header-actions">
        <button class="neo-btn icon-btn" @click="emit('settings')" v-if="myRole === 'owner' || myRole === 'admin'">
          <i class="ri-settings-3-line"></i>
        </button>
        <button class="primary-btn" v-if="myRole !== 'owner'" @click="emit('leave')">
          <i class="ri-logout-box-line"></i> 退出
        </button>
        <button class="danger-btn" v-if="myRole === 'owner'" @click="emit('delete')">
          <i class="ri-delete-bin-line"></i> 解散
        </button>
      </div>
    </header>

    <!-- Bento Grid -->
    <div class="bento-grid">
      <!-- Hero Card -->
      <div class="neo-panel hero-panel" :style="bannerStyle">
        <div class="hero-overlay"></div>
        <div class="hero-content">
          <div class="hero-badges">
            <span v-if="guild.faction" class="badge faction" :class="guild.faction">{{ factionLabel }}</span>
            <span class="badge date">Founded 2024</span>
          </div>
          <h1>{{ guild.name }}</h1>
          <p>{{ guild.slogan || guild.description || '暂无描述' }}</p>
        </div>
        <div class="hero-level">
          <div class="level-label">GUILD LEVEL</div>
          <div class="level-value">{{ guild.story_count || 1 }}</div>
        </div>
      </div>

      <!-- Status Card -->
      <div class="neo-panel status-panel">
        <div class="panel-header">
          <h3>Status</h3>
          <div class="status-dot"></div>
        </div>
        <div class="status-content">
          <div class="status-title">{{ guild.status === 'approved' ? '正常运营' : '审核中' }}</div>
          <div class="status-sub">邀请码: {{ guild.invite_code }}</div>
        </div>
        <div class="capacity-bar">
          <div class="bar-fill" :style="{ width: Math.min(guild.member_count / 60 * 100, 100) + '%' }"></div>
        </div>
        <div class="capacity-info">
          <span>CAPACITY</span>
          <span>{{ guild.member_count }}/60</span>
        </div>
      </div>

      <!-- Event Card -->
      <div class="neo-panel event-panel">
        <div class="event-date">
          <span class="month">DEC</span>
          <span class="day">24</span>
          <span class="time">20:00</span>
        </div>
        <div class="event-content">
          <h3>公会活动</h3>
          <p>{{ guild.description ? guild.description.slice(0, 50) + '...' : '暂无活动安排' }}</p>
          <button class="text-btn">查看详情 →</button>
        </div>
      </div>

      <!-- Contributors Card -->
      <div class="neo-panel contributors-panel">
        <div class="panel-header">
          <h3>贡献榜单</h3>
          <button class="icon-btn-sm"><i class="ri-more-fill"></i></button>
        </div>
        <div class="contributors-list">
          <div v-for="(m, i) in members.slice(0, 4)" :key="m.id" class="contributor-item">
            <span class="rank" :class="{ gold: i === 0 }">{{ i + 1 }}</span>
            <div class="avatar">
              <img v-if="m.avatar" :src="m.avatar" alt="" />
              <span v-else>{{ m.username?.charAt(0) }}</span>
            </div>
            <div class="info">
              <div class="name" :style="buildNameStyle(m.name_color, m.name_bold)">{{ m.username }}</div>
              <div class="role">{{ getRoleLabel(m.role) }} • {{ 9850 - i * 1430 }} pts</div>
            </div>
            <i v-if="i === 0" class="ri-vip-crown-fill crown"></i>
          </div>
        </div>
      </div>

      <!-- Resources Card -->
      <div class="neo-panel resources-panel">
        <h3>Guild Bank</h3>
        <div class="resource-icons">
          <div class="resource-icon"><i class="ri-copper-coin-line"></i></div>
          <div class="resource-icon"><i class="ri-flask-line"></i></div>
          <div class="resource-icon"><i class="ri-scroll-to-bottom-line"></i></div>
          <div class="resource-icon"><i class="ri-diamond-line"></i></div>
        </div>
        <div class="resource-stats">
          <div class="stat-main">
            <span class="value">{{ guild.story_count * 1000 + 500 }}</span>
            <span class="label">GOLD RESERVES</span>
          </div>
          <div class="stat-trend up">
            <i class="ri-arrow-up-line"></i> +12%
          </div>
        </div>
      </div>

      <!-- Activity Log -->
      <div class="neo-panel activity-panel">
        <div class="activity-icon"><i class="ri-flashlight-line"></i></div>
        <h3>动态日志</h3>
        <div class="activity-list">
          <div class="activity-item featured">
            <span class="time">10m ago</span>
            <p><span class="highlight">Admin</span> 更新了公会公告</p>
          </div>
          <div class="activity-item">
            <span class="time">2h ago</span>
            <p><span class="highlight" :style="buildNameStyle(members[0]?.name_color, members[0]?.name_bold)">{{ members[0]?.username || 'Member' }}</span> 获得成就</p>
          </div>
          <div class="activity-item">
            <span class="time">5h ago</span>
            <p>新成员加入公会</p>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="neo-panel actions-panel">
        <div class="action-buttons">
          <button class="action-btn discord">
            <i class="ri-discord-fill"></i> Discord
          </button>
          <button class="action-btn neo">
            <i class="ri-quill-pen-line"></i> Log
          </button>
        </div>
        <button class="checkin-btn">
          <i class="ri-calendar-check-line"></i> 签到领取奖励
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.layout4 {
  min-height: 100vh;
  background: #EED9C4;
  background-image: repeating-linear-gradient(45deg, transparent, transparent 10px, rgba(128, 64, 48, 0.03) 10px, rgba(128, 64, 48, 0.03) 12px);
  padding: 40px;
}

/* Header */
.bento-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-family: monospace;
}

.breadcrumb .server { color: #8C7B70; }
.breadcrumb .sep { color: #804030; }
.breadcrumb .name { color: #2C1810; font-weight: 600; }

.header-actions {
  display: flex;
  gap: 12px;
}

/* Neo Buttons */
.neo-btn {
  background: #EED9C4;
  box-shadow: 5px 5px 10px #cabaa8, -5px -5px 10px #ffffff;
  border: none;
  border-radius: 2px;
  cursor: pointer;
  transition: all 0.2s;
}

.neo-btn:active {
  box-shadow: inset 3px 3px 6px #cabaa8, inset -3px -3px 6px #ffffff;
}

.icon-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #804030;
  font-size: 18px;
}

.primary-btn {
  padding: 10px 24px;
  background: #804030;
  color: #fff;
  border: none;
  border-radius: 2px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 4px 12px rgba(128, 64, 48, 0.3);
}

.danger-btn {
  padding: 10px 24px;
  background: #dc2626;
  color: #fff;
  border: none;
  border-radius: 2px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Bento Grid */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  grid-template-rows: repeat(auto-fill, minmax(60px, auto));
  gap: 24px;
}

/* Neo Panel Base */
.neo-panel {
  background: linear-gradient(145deg, #fff3e3, #d6c3b0);
  box-shadow: 8px 8px 20px #cabaa8, -8px -8px 20px #ffffff;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.4);
  transition: all 0.3s;
  position: relative;
  overflow: hidden;
}

.neo-panel:hover {
  transform: translateY(-4px);
  box-shadow: 12px 12px 30px #cabaa8, -12px -12px 30px #ffffff;
}

/* Hero Panel */
.hero-panel {
  grid-column: span 8;
  grid-row: span 4;
  min-height: 280px;
  background-size: cover;
  background-position: center;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  padding: 32px;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(75, 54, 33, 0.9), rgba(75, 54, 33, 0.4), transparent);
}

.hero-content {
  position: relative;
  z-index: 10;
}

.hero-badges {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.badge {
  padding: 4px 12px;
  border-radius: 2px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.badge.faction {
  background: #D4A373;
  color: #4B3621;
}

.badge.faction.alliance { background: #3b82f6; color: #fff; }
.badge.faction.horde { background: #dc2626; color: #fff; }

.badge.date {
  color: rgba(255, 255, 255, 0.8);
  font-family: monospace;
  letter-spacing: 1px;
}

.hero-content h1 {
  font-size: 42px;
  color: #fff;
  margin: 0 0 8px 0;
  text-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.hero-content p {
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  max-width: 500px;
  margin: 0;
}

.hero-level {
  position: absolute;
  top: 32px;
  right: 32px;
  text-align: right;
  z-index: 10;
}

.level-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);
  font-family: monospace;
}

.level-value {
  font-size: 48px;
  font-weight: 800;
  color: #D4A373;
}

/* Status Panel */
.status-panel {
  grid-column: span 4;
  grid-row: span 2;
  padding: 24px;
  background: #fff;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.panel-header h3 {
  font-size: 11px;
  color: #804030;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin: 0;
}

.status-dot {
  width: 12px;
  height: 12px;
  background: #4ade80;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 10px rgba(74, 222, 128, 0); }
}

.status-title {
  font-size: 28px;
  font-weight: 700;
  color: #2C1810;
}

.status-sub {
  font-size: 13px;
  color: #8C7B70;
  margin-top: 4px;
}

.capacity-bar {
  height: 4px;
  background: #e5e7eb;
  border-radius: 2px;
  overflow: hidden;
  margin-top: 16px;
}

.bar-fill {
  height: 100%;
  background: #804030;
}

.capacity-info {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #8C7B70;
  font-family: monospace;
  margin-top: 4px;
}

/* Event Panel */
.event-panel {
  grid-column: span 4;
  grid-row: span 2;
  display: flex;
  overflow: hidden;
  padding: 0;
}

.event-date {
  width: 96px;
  background: #804030;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}

.event-date .month {
  font-size: 11px;
  font-weight: 600;
  opacity: 0.6;
}

.event-date .day {
  font-size: 32px;
  font-weight: 800;
}

.event-date .time {
  font-size: 11px;
  font-weight: 600;
  opacity: 0.6;
}

.event-content {
  flex: 1;
  padding: 20px;
  background: #FDFBF9;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.event-content h3 {
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 8px 0;
}

.event-content p {
  font-size: 12px;
  color: #8C7B70;
  margin: 0;
  line-height: 1.5;
}

.text-btn {
  background: none;
  border: none;
  color: #804030;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  padding: 0;
  margin-top: 12px;
  text-align: left;
}

/* Contributors Panel */
.contributors-panel {
  grid-column: span 4;
  grid-row: span 4;
  padding: 24px;
  background: #F8F4F0;
}

.contributors-panel .panel-header {
  margin-bottom: 20px;
}

.contributors-panel h3 {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
  margin: 0;
}

.icon-btn-sm {
  background: none;
  border: none;
  color: #804030;
  cursor: pointer;
  padding: 4px;
}

.contributors-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.contributor-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.contributor-item .rank {
  width: 16px;
  font-size: 16px;
  font-weight: 800;
  color: #8C7B70;
}

.contributor-item .rank.gold {
  color: #D4A373;
}

.contributor-item .avatar {
  width: 40px;
  height: 40px;
  border-radius: 2px;
  background: linear-gradient(135deg, #B87333, #4B3621);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  border: 1px solid #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  overflow: hidden;
}

.contributor-item .avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.contributor-item .info {
  flex: 1;
}

.contributor-item .name {
  font-size: 13px;
  font-weight: 600;
  color: #2C1810;
}

.contributor-item .role {
  font-size: 11px;
  color: #8C7B70;
}

.contributor-item .crown {
  color: #eab308;
  font-size: 12px;
}

/* Resources Panel */
.resources-panel {
  grid-column: span 4;
  grid-row: span 2;
  padding: 24px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.resources-panel h3 {
  font-size: 11px;
  color: #8C7B70;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin: 0 0 16px 0;
}

.resource-icons {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.resource-icon {
  aspect-ratio: 1;
  background: rgba(75, 54, 33, 0.05);
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #804030;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.resource-icon:hover {
  background: #804030;
  color: #fff;
  border-color: rgba(75, 54, 33, 0.2);
}

.resource-stats {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-top: 16px;
}

.stat-main .value {
  font-size: 24px;
  font-weight: 800;
  color: #2C1810;
  display: block;
}

.stat-main .label {
  font-size: 11px;
  color: #8C7B70;
  font-family: monospace;
}

.stat-trend {
  font-size: 12px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat-trend.up {
  color: #16a34a;
}

/* Activity Panel */
.activity-panel {
  grid-column: span 4;
  grid-row: span 4;
  padding: 24px;
  position: relative;
}

.activity-icon {
  position: absolute;
  top: 24px;
  right: 24px;
  font-size: 48px;
  color: #804030;
  opacity: 0.1;
}

.activity-panel h3 {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 16px 0;
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.activity-item {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding-left: 12px;
  border-left: 2px solid #e5e7eb;
}

.activity-item.featured {
  border-left-color: #D4A373;
}

.activity-item .time {
  font-size: 11px;
  color: #8C7B70;
  font-family: monospace;
  min-width: 60px;
}

.activity-item p {
  font-size: 13px;
  color: #2C1810;
  margin: 0;
  line-height: 1.4;
}

.activity-item .highlight {
  color: #804030;
  font-weight: 600;
}

/* Actions Panel */
.actions-panel {
  grid-column: span 4;
  grid-row: span 2;
  padding: 24px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 12px;
}

.action-buttons {
  display: flex;
  gap: 16px;
}

.action-btn {
  flex: 1;
  height: 48px;
  border-radius: 2px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.2s;
  border: none;
}

.action-btn.discord {
  background: #4B3621;
  color: #fff;
  box-shadow: 0 4px 12px rgba(75, 54, 33, 0.3);
}

.action-btn.discord:hover {
  background: #804030;
}

.action-btn.neo {
  background: #EED9C4;
  color: #2C1810;
  box-shadow: 5px 5px 10px #cabaa8, -5px -5px 10px #ffffff;
}

.checkin-btn {
  width: 100%;
  height: 48px;
  border: 2px dashed rgba(128, 64, 48, 0.3);
  background: transparent;
  color: #804030;
  font-weight: 600;
  font-size: 13px;
  border-radius: 2px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.2s;
}

.checkin-btn:hover {
  background: rgba(128, 64, 48, 0.05);
}
</style>
