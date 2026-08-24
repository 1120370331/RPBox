<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { useUserStore } from '@shared/stores/user'
import { createContentReport } from '@/api/safety'
import SafetyReportSheet from '@/components/SafetyReportSheet.vue'
import {
  applyGuild,
  cancelApplication,
  deleteGuild,
  getGuild,
  leaveGuild,
  listGuildMembers,
  listMyApplications,
  type Guild,
  type GuildApplication,
  type GuildMember,
} from '@/api/guild'
import { resolveApiUrl } from '@/api/image'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const loading = ref(false)
const membersLoading = ref(false)
const applying = ref(false)
const cancelingApplication = ref(false)
const leaving = ref(false)
const disbanding = ref(false)
const safetySubmitting = ref(false)
const showLeaveConfirm = ref(false)
const showDisbandConfirm = ref(false)
const showSafetySheet = ref(false)
const guild = ref<Guild | null>(null)
const myRole = ref<'' | 'owner' | 'admin' | 'member'>('')
const members = ref<GuildMember[]>([])
const pendingApplication = ref<GuildApplication | null>(null)

const guildId = computed(() => Number(route.params.id))
const guildAvatar = computed(() => resolveApiUrl(guild.value?.avatar_url || guild.value?.avatar || ''))
const guildBanner = computed(() => resolveApiUrl(guild.value?.banner_url || guild.value?.banner || ''))
const canLeave = computed(() => myRole.value === 'admin' || myRole.value === 'member')
const isAdmin = computed(() => myRole.value === 'owner' || myRole.value === 'admin')
const isOwner = computed(() => myRole.value === 'owner')
const canUseGuildSafetyAction = computed(() => {
  const currentUserId = userStore.user?.id
  const ownerId = guild.value?.owner_id
  return Boolean(currentUserId && ownerId && currentUserId !== ownerId && !isOwner.value)
})
const guildSafetyLabel = computed(() => guild.value
  ? t('guild.safety.targetLabel', { id: guild.value.id, name: guild.value.name })
  : '')
const relationshipLabel = computed(() => {
  if (myRole.value) return roleLabel(myRole.value)
  if (pendingApplication.value) return t('guild.apply.pending')
  return '-'
})

const guildRoleOrder: Record<GuildMember['role'], number> = {
  owner: 0,
  admin: 1,
  member: 2,
}

function hasValidMemberRole(member: GuildMember): boolean {
  return member.role === 'owner' || member.role === 'admin' || member.role === 'member'
}

function hasMemberIdentity(member: GuildMember): boolean {
  return Boolean(member.id && member.guild_id && member.user_id && member.username)
}

function sortGuildMembers(memberList: GuildMember[]): GuildMember[] {
  return memberList
    .map((member, index) => ({ member, index }))
    .sort((a, b) => {
      const aActive = hasValidMemberRole(a.member) && hasMemberIdentity(a.member)
      const bActive = hasValidMemberRole(b.member) && hasMemberIdentity(b.member)
      if (aActive !== bActive) return aActive ? -1 : 1

      const roleDiff = (guildRoleOrder[a.member.role] ?? 99) - (guildRoleOrder[b.member.role] ?? 99)
      if (roleDiff !== 0) return roleDiff

      return a.index - b.index
    })
    .map(({ member }) => member)
}

function roleLabel(role?: Guild['my_role'] | '') {
  if (!role) return ''
  return t(`guild.role.${role}`)
}

function factionLabel(name?: Guild['faction']) {
  if (!name) return ''
  return t(`guild.faction.${name}`)
}

async function loadPendingApplication() {
  const res = await listMyApplications()
  pendingApplication.value = res.applications?.find((app) => (
    app.guild_id === guildId.value && app.status === 'pending'
  )) || null
}

async function loadDetail() {
  if (!guildId.value) return
  loading.value = true
  membersLoading.value = false
  pendingApplication.value = null
  try {
    const res = await getGuild(guildId.value)
    guild.value = res.guild
    myRole.value = res.my_role || ''
    members.value = []
    if (res.my_role) {
      membersLoading.value = true
      void listGuildMembers(guildId.value)
        .then((membersRes) => {
          members.value = sortGuildMembers(membersRes.members || [])
        })
        .catch(() => {
          members.value = []
        })
        .finally(() => {
          membersLoading.value = false
        })
    } else {
      await loadPendingApplication().catch(() => {
        pendingApplication.value = null
      })
    }
  } catch (error) {
    toast.error((error as Error)?.message || t('common.status.loadFailed'))
    members.value = []
    pendingApplication.value = null
    membersLoading.value = false
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
    if (!res.auto_approved && res.application?.status === 'pending') {
      pendingApplication.value = res.application
    }
    await loadDetail()
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.apply.failed'))
  } finally {
    applying.value = false
  }
}

async function handleCancelApplication() {
  if (!guild.value || !pendingApplication.value || cancelingApplication.value) return
  cancelingApplication.value = true
  try {
    await cancelApplication(guild.value.id, pendingApplication.value.id)
    pendingApplication.value = null
    toast.success(t('guild.apply.canceled'))
    await loadDetail()
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.apply.cancelFailed'))
  } finally {
    cancelingApplication.value = false
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

function openDisbandConfirm() {
  if (!guild.value || !isOwner.value || disbanding.value) return
  showDisbandConfirm.value = true
}

function closeDisbandConfirm() {
  if (disbanding.value) return
  showDisbandConfirm.value = false
}

async function confirmDisband() {
  if (!guild.value || !isOwner.value || disbanding.value) return
  disbanding.value = true
  try {
    await deleteGuild(guild.value.id)
    showDisbandConfirm.value = false
    toast.success(t('guild.disband.success'))
    await router.replace({ name: 'guild' })
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.disband.failed'))
  } finally {
    disbanding.value = false
  }
}

function openSafetySheet() {
  if (!canUseGuildSafetyAction.value || safetySubmitting.value) return
  showSafetySheet.value = true
}

function closeSafetySheet() {
  if (safetySubmitting.value) return
  showSafetySheet.value = false
}

async function submitGuildSafety(payload: {
  reason: string
  detail: string
  hideTarget: boolean
  blockAuthor: boolean
  submitReport: boolean
}) {
  if (!guild.value || !canUseGuildSafetyAction.value || safetySubmitting.value) return

  const currentGuild = guild.value
  const currentUserId = userStore.user?.id
  if (!currentUserId || currentGuild.owner_id === currentUserId) return

  const hasLocalSafetyAction = payload.hideTarget || payload.blockAuthor
  safetySubmitting.value = true
  try {
    await createContentReport({
      target_type: 'guild',
      target_id: currentGuild.id,
      reason: payload.reason,
      detail: payload.detail,
      hide_target: payload.hideTarget,
      block_author: payload.blockAuthor,
      submit_report: payload.submitReport,
    })
    showSafetySheet.value = false
    const successKey = payload.hideTarget && payload.blockAuthor
      ? 'guild.safety.hiddenAndBlockedSuccess'
      : payload.hideTarget
        ? 'guild.safety.hiddenSuccess'
        : payload.blockAuthor
          ? 'guild.safety.blockedSuccess'
          : 'guild.safety.reportSuccess'
    toast.success(t(successKey))
    if (hasLocalSafetyAction) {
      await router.replace({ name: 'guild' })
    }
  } catch (error) {
    toast.error((error as Error)?.message || t('guild.safety.failed'))
  } finally {
    safetySubmitting.value = false
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
            <button v-if="!myRole && !pendingApplication" class="action-btn primary" :disabled="applying" @click="handleApply">
              {{ $t('guild.actions.join') }}
            </button>
            <button v-else-if="!myRole && pendingApplication" class="action-btn" :disabled="cancelingApplication" @click="handleCancelApplication">
              {{ $t('guild.actions.cancelApplication') }}
            </button>
            <button v-else-if="canLeave" class="action-btn" @click="showLeaveConfirm = true">{{ $t('guild.actions.leave') }}</button>
            <span v-else class="role-badge">{{ roleLabel(myRole) }}</span>
          </div>
        </section>

        <section class="info-card">
          <div class="row">
            <span class="label">{{ $t('guild.info.myRole') }}</span>
            <span class="value" :class="{ pending: pendingApplication }">{{ relationshipLabel }}</span>
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

        <section v-if="canUseGuildSafetyAction" class="safety-card">
          <div>
            <h3>{{ $t('guild.safety.title') }}</h3>
            <p>{{ $t('guild.safety.description') }}</p>
          </div>
          <button
            type="button"
            class="safety-action"
            data-testid="guild-safety-open"
            :disabled="safetySubmitting"
            @click="openSafetySheet"
          >
            <i class="ri-alarm-warning-line" aria-hidden="true" />
            {{ $t('guild.safety.action') }}
          </button>
        </section>

        <section v-if="membersLoading || members.length" class="members-card">
          <h3>{{ $t('guild.info.members') }}</h3>
          <div v-if="membersLoading" class="hint">{{ $t('common.status.loading') }}</div>
          <ul v-else>
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

        <section v-if="isOwner" class="danger-zone" aria-labelledby="guild-danger-title">
          <div class="danger-zone__heading">
            <span class="danger-zone__eyebrow">{{ $t('guild.disband.eyebrow') }}</span>
            <h3 id="guild-danger-title">{{ $t('guild.disband.title') }}</h3>
          </div>
          <p>{{ $t('guild.disband.description') }}</p>
          <button
            type="button"
            class="action-btn danger danger-zone__action"
            data-testid="guild-disband-open"
            :disabled="disbanding"
            @click="openDisbandConfirm"
          >
            <i class="ri-delete-bin-6-line" aria-hidden="true" />
            {{ $t('guild.disband.action') }}
          </button>
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

    <div v-if="showDisbandConfirm" class="dialog-mask" @click.self="closeDisbandConfirm">
      <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="guild-disband-dialog-title">
        <h3 id="guild-disband-dialog-title">{{ $t('guild.disband.confirmTitle') }}</h3>
        <p>{{ $t('guild.disband.confirmMessage', { name: guild?.name || '' }) }}</p>
        <div class="dialog-actions">
          <button
            type="button"
            class="action-btn"
            data-testid="guild-disband-cancel"
            :disabled="disbanding"
            @click="closeDisbandConfirm"
          >
            {{ $t('guild.actions.cancel') }}
          </button>
          <button
            type="button"
            class="action-btn danger"
            data-testid="guild-disband-confirm"
            :disabled="disbanding"
            @click="confirmDisband"
          >
            {{ $t('guild.disband.confirmAction') }}
          </button>
        </div>
      </div>
    </div>

    <SafetyReportSheet
      :open="showSafetySheet"
      :submitting="safetySubmitting"
      :title="$t('guild.safety.sheetTitle')"
      :target-label="guildSafetyLabel"
      target-type="guild"
      initial-action="report"
      @close="closeSafetySheet"
      @submit="submitGuildSafety"
    />
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

.value.pending {
  color: #B87333;
  font-weight: 600;
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

.safety-card {
  display: grid;
  gap: 10px;
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
}

.safety-card h3 {
  font-size: 13px;
}

.safety-card p {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.safety-action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 40px;
  border: 1px solid color-mix(in srgb, var(--btn-danger-bg) 45%, var(--color-border));
  border-radius: 10px;
  background: color-mix(in srgb, var(--btn-danger-bg) 7%, var(--color-panel-bg));
  color: var(--btn-danger-bg);
  font-size: 13px;
  font-weight: 600;
}

.safety-action:disabled {
  opacity: 0.6;
}

.danger-zone {
  margin-top: 12px;
  padding: 14px;
  border: 1px solid color-mix(in srgb, var(--btn-danger-bg) 42%, var(--color-border));
  border-left: 4px solid var(--btn-danger-bg);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--btn-danger-bg) 7%, var(--color-card-bg));
  box-shadow: var(--shadow-sm);
}

.danger-zone__heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.danger-zone__eyebrow {
  color: var(--btn-danger-bg);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.danger-zone h3 {
  flex: 1;
  font-size: 14px;
}

.danger-zone p {
  margin-top: 7px;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.danger-zone__action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-height: 42px;
  margin-top: 12px;
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
