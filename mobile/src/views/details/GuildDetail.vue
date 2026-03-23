<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { applyGuild, getGuild, leaveGuild, listGuildMembers, type Guild, type GuildMember } from '@/api/guild'
import { resolveApiUrl } from '@/api/image'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const applying = ref(false)
const leaving = ref(false)
const showLeaveConfirm = ref(false)
const guild = ref<Guild | null>(null)
const myRole = ref<'' | 'owner' | 'admin' | 'member'>('')
const members = ref<GuildMember[]>([])

const guildId = computed(() => Number(route.params.id))
const guildAvatar = computed(() => resolveApiUrl(guild.value?.avatar_url || guild.value?.avatar || ''))
const guildBanner = computed(() => resolveApiUrl(guild.value?.banner_url || guild.value?.banner || ''))
const canLeave = computed(() => myRole.value === 'admin' || myRole.value === 'member')
const isAdmin = computed(() => myRole.value === 'owner' || myRole.value === 'admin')

function roleLabel(role?: Guild['my_role'] | '') {
  if (!role) return ''
  return t(`guild.role.${role}`)
}

function factionLabel(name?: Guild['faction']) {
  if (!name) return ''
  return t(`guild.faction.${name}`)
}

async function loadDetail() {
  if (!guildId.value) return
  loading.value = true
  try {
    const res = await getGuild(guildId.value)
    guild.value = res.guild
    myRole.value = res.my_role || ''
    if (res.my_role) {
      const membersRes = await listGuildMembers(guildId.value)
      members.value = membersRes.members || []
    } else {
      members.value = []
    }
  } catch (error) {
    toast.error((error as Error)?.message || t('common.status.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleApply() {
  if (!guild.value || applying.value) return
  applying.value = true
  try {
    const res = await applyGuild(guild.value.id)
    toast.success(res.auto_approved ? t('guild.apply.autoApproved') : t('guild.apply.success'))
    await loadDetail()
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.apply.failed'))
  } finally {
    applying.value = false
  }
}

async function confirmLeave() {
  if (!guild.value || leaving.value) return
  leaving.value = true
  try {
    await leaveGuild(guild.value.id)
    toast.success(t('guild.leave.success'))
    showLeaveConfirm.value = false
    router.replace({ name: 'guild' })
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.leave.failed'))
  } finally {
    leaving.value = false
  }
}

function goGuildHome() {
  router.push({ name: 'guild' })
}

onMounted(loadDetail)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="goGuildHome"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('guild.detailTitle') }}</h1>
    </header>

    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('guild.loading') }}</div>
      <div v-else-if="!guild" class="hint">{{ $t('guild.empty') }}</div>

      <template v-else>
        <section class="hero-card">
          <div class="hero-banner" :style="guildBanner ? { backgroundImage: `url(${guildBanner})` } : undefined">
            <span v-if="guild.faction" class="faction-chip">{{ factionLabel(guild.faction) }}</span>
          </div>

          <div class="hero-main">
            <div class="avatar-wrap" :style="{ background: `#${guild.color || 'B87333'}` }">
              <img v-if="guildAvatar" :src="guildAvatar" alt="" loading="lazy" />
              <span v-else>{{ guild.name.slice(0, 1) }}</span>
            </div>
            <div class="guild-text">
              <h2>{{ guild.name }}</h2>
              <p>{{ guild.slogan || guild.description || $t('guild.info.noDescription') }}</p>
              <div class="meta">
                <span><i class="ri-user-line" /> {{ guild.member_count }} {{ $t('guild.info.members') }}</span>
                <span><i class="ri-book-open-line" /> {{ guild.story_count }} {{ $t('guild.info.stories') }}</span>
              </div>
            </div>
          </div>

          <div class="hero-actions">
            <button v-if="!myRole" class="action-btn primary" :disabled="applying" @click="handleApply">
              {{ $t('guild.actions.join') }}
            </button>
            <button v-else-if="canLeave" class="action-btn" @click="showLeaveConfirm = true">{{ $t('guild.actions.leave') }}</button>
            <span v-else class="role-badge">{{ roleLabel(myRole) }}</span>
          </div>
        </section>

        <section class="info-card">
          <div class="row">
            <span class="label">{{ $t('guild.info.myRole') }}</span>
            <span class="value">{{ myRole ? roleLabel(myRole) : '-' }}</span>
          </div>
          <div v-if="isAdmin" class="row">
            <span class="label">{{ $t('guild.info.inviteCode') }}</span>
            <span class="value code">{{ guild.invite_code || '-' }}</span>
          </div>
        </section>

        <section class="nav-card">
          <button class="nav-action" @click="router.push({ name: 'guild-posts', params: { id: guild.id } })">
            <i class="ri-article-line" />
            <div>
              <strong>{{ $t('guild.posts.guildPosts') }}</strong>
              <span>{{ $t('guild.detailNav.postsDesc') }}</span>
            </div>
            <i class="ri-arrow-right-s-line arrow" />
          </button>
          <button class="nav-action" @click="router.push({ name: 'guild-stories', params: { id: guild.id } })">
            <i class="ri-book-2-line" />
            <div>
              <strong>{{ $t('guild.stories.guildStories') }}</strong>
              <span>{{ $t('guild.detailNav.storiesDesc') }}</span>
            </div>
            <i class="ri-arrow-right-s-line arrow" />
          </button>
        </section>

        <section v-if="members.length" class="members-card">
          <h3>{{ $t('guild.info.members') }}</h3>
          <ul>
            <li v-for="member in members" :key="member.id">
              <div class="member-main">
                <img v-if="member.avatar" :src="resolveApiUrl(member.avatar)" alt="" loading="lazy" />
                <span v-else class="fallback">{{ member.username.slice(0, 1) }}</span>
                <span
                  class="name"
                  :style="{ color: member.name_color || undefined, fontWeight: member.name_bold ? '700' : undefined }"
                >{{ member.username }}</span>
              </div>
              <span class="member-role">{{ roleLabel(member.role) }}</span>
            </li>
          </ul>
        </section>
      </template>
    </div>

    <div v-if="showLeaveConfirm" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('guild.leave.title') }}</h3>
        <p>{{ $t('guild.leave.message') }}</p>
        <div class="dialog-actions">
          <button class="action-btn" @click="showLeaveConfirm = false">{{ $t('guild.actions.cancel') }}</button>
          <button class="action-btn danger" :disabled="leaving" @click="confirmLeave">{{ $t('guild.actions.confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hero-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
  margin-bottom: 12px;
}

.hero-banner {
  height: 120px;
  background: linear-gradient(135deg, #7b5f45, #4b3621);
  background-size: cover;
  background-position: center;
  padding: 10px;
  display: flex;
  justify-content: flex-end;
}

.faction-chip {
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 11px;
  height: fit-content;
}

.hero-main {
  display: flex;
  gap: 10px;
  padding: 12px;
}

.avatar-wrap {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #fff;
  font-weight: 700;
}

.avatar-wrap img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.guild-text { flex: 1; min-width: 0; }
.guild-text h2 { font-size: 17px; }
.guild-text p {
  margin-top: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.6;
  white-space: pre-wrap;
}

.meta {
  margin-top: 8px;
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.hero-actions {
  padding: 0 12px 12px;
  display: flex;
  justify-content: flex-end;
}

.action-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 13px;
  color: var(--text-dark);
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.action-btn.danger {
  border-color: var(--btn-danger-bg);
  background: var(--btn-danger-bg);
  color: #fff;
}

.role-badge {
  font-size: 12px;
  color: var(--color-text-secondary);
  background: var(--color-primary-light);
  border-radius: 999px;
  padding: 4px 10px;
}

.info-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 12px;
  margin-bottom: 12px;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 0;
}

.label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.value {
  font-size: 13px;
  color: var(--text-dark);
}

.value.code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.members-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 12px;
}

.nav-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 8px;
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-action {
  border: 1px solid var(--color-border-light);
  background: var(--color-panel-bg);
  border-radius: 10px;
  padding: 10px;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-action > i:first-child {
  font-size: 18px;
  color: var(--color-primary);
}

.nav-action div {
  flex: 1;
  min-width: 0;
}

.nav-action strong {
  display: block;
  font-size: 13px;
}

.nav-action span {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.nav-action .arrow {
  color: var(--color-text-secondary);
}

.members-card h3 {
  font-size: 14px;
  margin-bottom: 10px;
}

.members-card ul {
  list-style: none;
}

.members-card li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border-light);
}

.members-card li:last-child {
  border-bottom: none;
}

.member-main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.member-main img,
.fallback {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--icon-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--icon-color);
  font-size: 12px;
}

.name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-role {
  font-size: 11px;
  color: var(--color-text-secondary);
}

.dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.48);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  z-index: 1000;
}

.dialog {
  width: 100%;
  max-width: 340px;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 14px;
}

.dialog h3 {
  font-size: 16px;
}

.dialog p {
  margin-top: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
