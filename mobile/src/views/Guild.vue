<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  applyGuild,
  cancelApplication,
  joinGuild,
  listGuilds,
  listMyApplications,
  listPublicGuilds,
  type Guild,
  type GuildApplication,
} from '@/api/guild'
import { resolveApiUrl } from '@/api/image'

const { t } = useI18n()
const router = useRouter()
const toast = useToastStore()

const loading = ref(false)
const activeTab = ref<'public' | 'my'>('public')
const keyword = ref('')
const faction = ref('')
const inviteCode = ref('')
const joining = ref(false)

const myGuilds = ref<Guild[]>([])
const publicGuilds = ref<Guild[]>([])
const myApplications = ref<GuildApplication[]>([])

const displayGuilds = computed(() => (activeTab.value === 'my' ? myGuilds.value : publicGuilds.value))

function getGuildAvatar(guild: Guild) {
  return resolveApiUrl(guild.avatar_url || guild.avatar || '')
}

function getGuildBanner(guild: Guild) {
  return resolveApiUrl(guild.banner_url || guild.banner || '')
}

function roleLabel(role?: Guild['my_role']) {
  if (!role) return ''
  return t(`guild.role.${role}`)
}

function factionLabel(name?: Guild['faction']) {
  if (!name) return ''
  return t(`guild.faction.${name}`)
}

function isMember(guildId: number) {
  return myGuilds.value.some((g) => g.id === guildId)
}

function pendingApplication(guildId: number) {
  return myApplications.value.find((app) => app.guild_id === guildId && app.status === 'pending')
}

async function loadMyGuilds() {
  const res = await listGuilds()
  myGuilds.value = res.guilds || []
}

async function loadPublicGuilds() {
  const res = await listPublicGuilds({
    keyword: keyword.value.trim() || undefined,
    faction: faction.value || undefined,
  })
  publicGuilds.value = res.guilds || []
}

async function loadMyApplications() {
  const res = await listMyApplications()
  myApplications.value = res.applications || []
}

async function loadData() {
  loading.value = true
  try {
    await Promise.all([loadMyGuilds(), loadPublicGuilds(), loadMyApplications()])
  } catch (error) {
    toast.error((error as Error)?.message || t('common.status.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleSearch() {
  loading.value = true
  try {
    await loadPublicGuilds()
  } catch (error) {
    toast.error((error as Error)?.message || t('common.status.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleJoinByCode() {
  if (!inviteCode.value.trim() || joining.value) return
  joining.value = true
  try {
    await joinGuild(inviteCode.value.trim())
    inviteCode.value = ''
    toast.success(t('guild.joinByCode.success'))
    await loadData()
    activeTab.value = 'my'
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.joinByCode.failed'))
  } finally {
    joining.value = false
  }
}

async function handleApply(guild: Guild) {
  try {
    const res = await applyGuild(guild.id)
    toast.success(res.auto_approved ? t('guild.apply.autoApproved') : t('guild.apply.success'))
    await Promise.all([loadMyGuilds(), loadMyApplications()])
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.apply.failed'))
  }
}

async function handleCancelApplication(guildId: number) {
  const app = pendingApplication(guildId)
  if (!app) return
  try {
    await cancelApplication(guildId, app.id)
    toast.success(t('guild.apply.canceled'))
    await loadMyApplications()
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.apply.cancelFailed'))
  }
}

onMounted(loadData)
</script>

<template>
  <div class="page guild-page">
    <header class="page-header">
      <h1>{{ $t('guild.title') }}</h1>
    </header>

    <div class="tab-row">
      <button :class="['tab-btn', { active: activeTab === 'public' }]" @click="activeTab = 'public'">{{ $t('guild.tabs.public') }}</button>
      <button :class="['tab-btn', { active: activeTab === 'my' }]" @click="activeTab = 'my'">{{ $t('guild.tabs.my') }}</button>
    </div>

    <div v-if="activeTab === 'public'" class="search-area">
      <div class="search-bar">
        <i class="ri-search-line" />
        <input v-model="keyword" :placeholder="$t('guild.searchPlaceholder')" @keyup.enter="handleSearch" />
      </div>
      <div class="search-actions">
        <select v-model="faction" class="faction-select">
          <option value="">{{ $t('guild.allFactions') }}</option>
          <option value="alliance">{{ $t('guild.faction.alliance') }}</option>
          <option value="horde">{{ $t('guild.faction.horde') }}</option>
          <option value="neutral">{{ $t('guild.faction.neutral') }}</option>
        </select>
        <button class="small-btn" @click="handleSearch">{{ $t('guild.actions.search') }}</button>
      </div>
    </div>

    <div class="join-card">
      <div class="join-label">{{ $t('guild.actions.joinByCode') }}</div>
      <div class="join-row">
        <input v-model="inviteCode" :placeholder="$t('guild.joinByCode.placeholder')" @keyup.enter="handleJoinByCode" />
        <button class="small-btn primary" :disabled="joining" @click="handleJoinByCode">{{ $t('guild.actions.confirm') }}</button>
      </div>
    </div>

    <div class="page-body">
      <div v-if="loading" class="empty-hint">{{ $t('guild.loading') }}</div>
      <div v-else-if="displayGuilds.length === 0" class="empty-hint">{{ activeTab === 'my' ? $t('guild.emptyMy') : $t('guild.empty') }}</div>

      <div v-else class="guild-list">
        <article
          v-for="guild in displayGuilds"
          :key="guild.id"
          class="guild-card"
          @click="router.push({ name: 'guild-detail', params: { id: guild.id } })"
        >
          <div class="guild-banner" :style="getGuildBanner(guild) ? { backgroundImage: `url(${getGuildBanner(guild)})` } : undefined">
            <span v-if="guild.faction" class="faction-chip">{{ factionLabel(guild.faction) }}</span>
          </div>
          <div class="guild-main">
            <div class="avatar-wrap" :style="{ background: `#${guild.color || 'B87333'}` }">
              <img v-if="getGuildAvatar(guild)" :src="getGuildAvatar(guild)" alt="" loading="lazy" />
              <span v-else>{{ guild.name.slice(0, 1) }}</span>
            </div>
            <div class="guild-info">
              <div class="title-row">
                <h3>{{ guild.name }}</h3>
                <span v-if="guild.my_role" class="role-badge">{{ roleLabel(guild.my_role) }}</span>
              </div>
              <p>{{ guild.slogan || guild.description || $t('guild.info.noDescription') }}</p>
              <div class="meta-row">
                <span><i class="ri-user-line" /> {{ guild.member_count }} {{ $t('guild.info.members') }}</span>
                <span><i class="ri-book-open-line" /> {{ guild.story_count }} {{ $t('guild.info.stories') }}</span>
              </div>
            </div>
          </div>

          <div v-if="activeTab === 'public' && !isMember(guild.id)" class="action-row">
            <button
              v-if="!pendingApplication(guild.id)"
              class="small-btn primary"
              @click.stop="handleApply(guild)"
            >{{ $t('guild.actions.join') }}</button>
            <button
              v-else
              class="small-btn"
              @click.stop="handleCancelApplication(guild.id)"
            >{{ $t('guild.actions.cancelApplication') }}</button>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 0 16px 16px; }
.page-header { padding: 12px 0 8px; }
.page-header h1 { font-size: 22px; }

.tab-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 10px;
}

.tab-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 12px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.tab-btn.active {
  background: var(--color-primary);
  color: var(--text-light);
  border-color: var(--color-primary);
}

.search-area {
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  border-radius: var(--radius-md);
  padding: 12px;
  margin-bottom: 10px;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--input-bg);
  border-radius: 20px;
  padding: 8px 12px;
}

.search-bar i { font-size: 16px; color: var(--input-placeholder); }

.search-bar input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 14px;
  color: var(--text-dark);
}

.search-actions {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}

.faction-select {
  flex: 1;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 8px;
  background: var(--input-bg);
  color: var(--text-dark);
}

.join-card {
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
  border-radius: var(--radius-md);
  padding: 12px;
  margin-bottom: 10px;
}

.join-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.join-row {
  display: flex;
  gap: 8px;
}

.join-row input {
  flex: 1;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 8px 10px;
  background: var(--input-bg);
  color: var(--text-dark);
}

.small-btn {
  border: 1px solid var(--color-border);
  background: var(--color-panel-bg);
  border-radius: 10px;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--text-dark);
}

.small-btn.primary {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--text-light);
}

.small-btn:disabled {
  opacity: 0.6;
}

.empty-hint {
  text-align: center;
  padding: 60px 0;
  color: var(--color-accent);
  font-size: 14px;
}

.guild-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.guild-card {
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
}

.guild-banner {
  height: 84px;
  background: linear-gradient(135deg, #7b5f45, #4b3621);
  background-size: cover;
  background-position: center;
  display: flex;
  justify-content: flex-end;
  padding: 8px;
}

.faction-chip {
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 11px;
  height: fit-content;
}

.guild-main {
  display: flex;
  gap: 10px;
  padding: 12px;
}

.avatar-wrap {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  flex-shrink: 0;
}

.avatar-wrap img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.guild-info {
  flex: 1;
  min-width: 0;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-row h3 {
  font-size: 15px;
  font-weight: 600;
  flex: 1;
}

.role-badge {
  font-size: 11px;
  color: var(--color-text-secondary);
  background: var(--color-primary-light);
  border-radius: 8px;
  padding: 2px 6px;
}

.guild-info p {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.meta-row {
  margin-top: 8px;
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.meta-row i { margin-right: 2px; }

.action-row {
  padding: 0 12px 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
