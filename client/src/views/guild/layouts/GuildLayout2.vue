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

const guildMaster = computed(() => {
  return props.members.find(m => m.role === 'owner')
})
</script>

<template>
  <div class="layout2">
    <!-- Decorative Background -->
    <div class="bg-decoration"></div>
    <div class="bg-symbol">G</div>

    <div class="content-wrapper">
      <!-- Editorial Header -->
      <header class="editorial-header">
        <div class="header-left">
          <div class="header-badges">
            <span class="rank-badge">Guild Rank #{{ guild.story_count || 1 }}</span>
            <span class="divider"></span>
            <span class="est-date">Est. 2024</span>
          </div>
          <h1 class="guild-title">
            {{ guild.name.split(' ')[0] || guild.name }}<br>
            <span class="accent">{{ guild.name.split(' ').slice(1).join(' ') || '' }}</span>
          </h1>
        </div>
        <div class="header-right">
          <p class="guild-motto">{{ guild.slogan || guild.description || '暂无描述' }}</p>
          <div class="header-actions">
            <button v-if="myRole === 'owner' || myRole === 'admin'" class="btn-primary" @click="emit('settings')">
              <span>设置</span>
            </button>
            <button v-if="myRole !== 'owner'" class="btn-outline" @click="emit('leave')">退出公会</button>
            <button v-if="myRole === 'owner'" class="btn-danger" @click="emit('delete')">解散公会</button>
          </div>
        </div>
      </header>

      <!-- Hero Section -->
      <section class="hero-section">
        <div class="hero-image" :style="bannerStyle">
          <div class="hero-overlay"></div>
          <div class="hero-caption">
            <div class="caption-label">Current Campaign</div>
            <h2>{{ guild.slogan || '探索未知的边境' }}</h2>
          </div>
        </div>

        <div class="hero-stats">
          <div class="vertical-text">名誉と栄光</div>
          <div class="stats-content">
            <div class="stat-item">
              <span class="stat-label">Members</span>
              <div class="stat-value">
                <span class="number">{{ guild.member_count }}</span>
                <span class="sub">/ 150</span>
              </div>
            </div>
            <div class="stat-item">
              <span class="stat-label">Stories</span>
              <div class="stat-value">
                <span class="number">{{ guild.story_count }}</span>
                <span class="sub">篇</span>
              </div>
            </div>
            <div class="stat-item" v-if="guildMaster">
              <span class="stat-label">Master</span>
              <div class="master-info">
                <div class="master-avatar">
                  <img v-if="guildMaster.avatar" :src="guildMaster.avatar" alt="" />
                  <span v-else>{{ guildMaster.username?.charAt(0) }}</span>
                </div>
                <div>
                  <div class="master-name">{{ guildMaster.username }}</div>
                  <div class="master-role">会长</div>
                </div>
              </div>
            </div>
            <div class="stat-footer">
              <span>邀请码</span>
              <span class="invite-code">{{ guild.invite_code }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Content Grid -->
      <div class="content-grid">
        <!-- Notice Board -->
        <div class="notice-column">
          <h3 class="section-title">Notice Board</h3>

          <article class="notice-item featured">
            <div class="notice-meta">
              <span class="dot"></span>
              <span>公会介绍</span>
            </div>
            <h4>{{ guild.name }}</h4>
            <p>{{ guild.description || '暂无详细介绍' }}</p>
          </article>

          <div class="hairline"></div>

          <article v-if="guild.server" class="notice-item">
            <div class="notice-meta">
              <span class="dot gray"></span>
              <span>服务器</span>
            </div>
            <h4>{{ guild.server }}</h4>
            <p v-if="guild.faction">阵营: {{ factionLabel }}</p>
          </article>

          <a href="#" class="view-archive">
            View Archive <span>→</span>
          </a>
        </div>

        <!-- Members Column -->
        <div class="members-column">
          <div class="column-header">
            <h3>Members</h3>
          </div>

          <div class="members-grid">
            <div v-for="m in members" :key="m.id" class="member-card">
              <div class="member-badge">{{ getRoleLabel(m.role).charAt(0) }}</div>
              <div class="member-avatar">
                <img v-if="m.avatar" :src="m.avatar" alt="" />
                <span v-else>{{ m.username?.charAt(0) }}</span>
              </div>
              <div class="member-details">
                <h4>{{ m.username }}</h4>
                <p>{{ getRoleLabel(m.role) }}</p>
              </div>
              <div class="member-progress">
                <div class="progress-bar">
                  <div class="progress-fill" :style="{ width: m.role === 'owner' ? '100%' : m.role === 'admin' ? '75%' : '50%' }"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <footer class="editorial-footer">
        <div class="footer-brand">{{ guild.name }} © 2024</div>
        <div class="footer-links">
          <a href="#">Manifesto</a>
          <a href="#">Roster</a>
          <a href="#">Diplomacy</a>
        </div>
      </footer>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,400;0,700;1,400&display=swap');

.layout2 {
  min-height: 100vh;
  background: #F2EBE5;
  color: #1A110D;
  position: relative;
  overflow-x: hidden;
}

.bg-decoration {
  position: absolute;
  top: 0;
  right: 0;
  width: 33%;
  height: 100%;
  background: #E8DCCF;
  opacity: 0.3;
  pointer-events: none;
}

.bg-symbol {
  position: absolute;
  top: 80px;
  right: 80px;
  font-size: 200px;
  font-family: 'Playfair Display', serif;
  color: #804030;
  opacity: 0.05;
  pointer-events: none;
  user-select: none;
}

.content-wrapper {
  position: relative;
  z-index: 10;
  max-width: 1200px;
  margin: 0 auto;
  padding: 48px 32px;
}

/* Editorial Header */
.editorial-header {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 48px;
  align-items: flex-end;
  margin-bottom: 48px;
}

.header-badges {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.rank-badge {
  padding: 4px 12px;
  border: 1px solid #804030;
  color: #804030;
  font-size: 11px;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.divider {
  width: 48px;
  height: 1px;
  background: rgba(128, 64, 48, 0.3);
}

.est-date {
  font-size: 14px;
  color: #8C7B70;
  font-style: italic;
  font-family: 'Playfair Display', serif;
}

.guild-title {
  font-size: 64px;
  font-family: 'Playfair Display', serif;
  font-weight: 400;
  line-height: 0.9;
  margin: 0;
  color: #2C1810;
}

.guild-title .accent {
  color: #804030;
  font-style: italic;
}

.header-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 24px;
}

.guild-motto {
  text-align: right;
  font-size: 14px;
  color: #8C7B70;
  max-width: 200px;
  line-height: 1.6;
  border-right: 1px solid rgba(128, 64, 48, 0.3);
  padding-right: 16px;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.header-actions button {
  padding: 12px 32px;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-primary {
  background: #2C1810;
  color: #fff;
}

.btn-primary:hover {
  background: #804030;
}

.btn-outline {
  background: transparent;
  border: 1px solid #2C1810;
  color: #2C1810;
}

.btn-danger {
  background: #804030;
  color: #fff;
}

/* Hero Section */
.hero-section {
  display: grid;
  grid-template-columns: 2fr 1fr;
  border-top: 1px solid rgba(26, 17, 13, 0.1);
  border-bottom: 1px solid rgba(26, 17, 13, 0.1);
  margin-bottom: 48px;
}

.hero-image {
  height: 400px;
  background-size: cover;
  background-position: center;
  position: relative;
  filter: grayscale(100%);
  transition: filter 0.5s;
}

.hero-image:hover {
  filter: grayscale(0%);
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0,0,0,0.8), transparent);
}

.hero-caption {
  position: absolute;
  bottom: 32px;
  left: 32px;
  color: #fff;
}

.caption-label {
  font-size: 11px;
  letter-spacing: 2px;
  text-transform: uppercase;
  color: #D4A373;
  margin-bottom: 8px;
}

.hero-caption h2 {
  font-size: 32px;
  font-family: 'Playfair Display', serif;
  font-style: italic;
  margin: 0;
}

.hero-stats {
  background: #fff;
  padding: 32px;
  border-left: 1px solid rgba(26, 17, 13, 0.1);
  position: relative;
}

.vertical-text {
  position: absolute;
  top: 32px;
  right: 24px;
  writing-mode: vertical-rl;
  font-size: 32px;
  font-family: 'Playfair Display', serif;
  color: rgba(140, 123, 112, 0.2);
  user-select: none;
}

.stats-content {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 11px;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: #8C7B70;
}

.stat-value {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.stat-value .number {
  font-size: 48px;
  font-family: 'Playfair Display', serif;
  color: #2C1810;
}

.stat-value .sub {
  font-size: 16px;
  color: #804030;
}

.master-info {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
}

.master-avatar {
  width: 40px;
  height: 40px;
  background: #2C1810;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  overflow: hidden;
}

.master-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.master-name {
  font-size: 14px;
  font-weight: 600;
  color: #2C1810;
}

.master-role {
  font-size: 12px;
  color: #8C7B70;
}

.stat-footer {
  padding-top: 24px;
  border-top: 1px dashed rgba(26, 17, 13, 0.2);
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #8C7B70;
}

.invite-code {
  font-weight: 600;
  color: #2C1810;
  font-family: monospace;
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 48px;
  margin-bottom: 48px;
}

.section-title {
  font-size: 18px;
  font-family: 'Playfair Display', serif;
  font-style: italic;
  border-bottom: 2px solid #804030;
  display: inline-block;
  padding-right: 32px;
  padding-bottom: 8px;
  margin-bottom: 24px;
}

.notice-item {
  margin-bottom: 24px;
  cursor: pointer;
}

.notice-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #8C7B70;
  margin-bottom: 8px;
}

.dot {
  width: 8px;
  height: 8px;
  background: #804030;
}

.dot.gray {
  background: #ccc;
}

.notice-item h4 {
  font-size: 18px;
  font-weight: 600;
  color: #2C1810;
  margin: 0 0 8px 0;
  transition: color 0.2s;
}

.notice-item:hover h4 {
  color: #804030;
}

.notice-item p {
  font-size: 14px;
  color: #8C7B70;
  line-height: 1.6;
  margin: 0;
}

.hairline {
  height: 1px;
  background: linear-gradient(90deg, transparent, #8C7B70, transparent);
  margin: 24px 0;
}

.view-archive {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: #804030;
  text-decoration: none;
  transition: color 0.2s;
}

.view-archive:hover {
  color: #2C1810;
}

.view-archive span {
  font-size: 16px;
}

/* Members Column */
.column-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 24px;
  border-bottom: 1px solid rgba(26, 17, 13, 0.1);
  padding-bottom: 8px;
}

.column-header h3 {
  font-size: 28px;
  font-family: 'Playfair Display', serif;
  margin: 0;
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.member-card {
  background: #fff;
  padding: 20px;
  box-shadow: 4px 4px 0 rgba(75, 54, 33, 0.1);
  border: 1px solid transparent;
  transition: all 0.2s;
  cursor: pointer;
  position: relative;
}

.member-card:hover {
  box-shadow: 6px 6px 0 rgba(128, 64, 48, 0.2);
  border-color: rgba(128, 64, 48, 0.2);
}

.member-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  background: #2C1810;
  color: #fff;
  font-size: 10px;
  padding: 2px 8px;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.member-card .member-avatar {
  width: 40px;
  height: 40px;
  background: #E8DCCF;
  color: #4B3621;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  filter: grayscale(100%);
  transition: filter 0.2s;
  margin-bottom: 12px;
  overflow: hidden;
}

.member-card .member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-card:hover .member-avatar {
  filter: grayscale(0%);
}

.member-details h4 {
  font-size: 16px;
  font-family: 'Playfair Display', serif;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: #2C1810;
  transition: color 0.2s;
}

.member-card:hover .member-details h4 {
  color: #804030;
}

.member-details p {
  font-size: 12px;
  color: #8C7B70;
  margin: 0 0 12px 0;
}

.member-progress {
  margin-top: 8px;
}

.progress-bar {
  height: 4px;
  background: #f0f0f0;
}

.progress-fill {
  height: 100%;
  background: #804030;
}

/* Footer */
.editorial-footer {
  border-top: 1px solid rgba(26, 17, 13, 0.1);
  padding-top: 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: #8C7B70;
}

.footer-brand {
  font-family: 'Playfair Display', serif;
  font-style: italic;
}

.footer-links {
  display: flex;
  gap: 24px;
}

.footer-links a {
  color: #8C7B70;
  text-decoration: none;
  transition: color 0.2s;
}

.footer-links a:hover {
  color: #804030;
}
</style>
