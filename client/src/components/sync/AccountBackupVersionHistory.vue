<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  deleteAccountBackupVersion,
  getAccountBackupVersion,
  getAccountBackupVersions,
  renameAccountBackupVersion,
  restoreAccountBackupVersion,
  type AccountBackup,
  type AccountBackupVersion,
} from '@/api/accountBackup'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'

const props = defineProps<{ accountId: string }>()
const emit = defineEmits<{
  restored: [backup: AccountBackup]
}>()

const { t, locale } = useI18n()
const dialog = useDialog()
const toast = useToastStore()
const versions = ref<AccountBackupVersion[]>([])
const loading = ref(false)
const busyId = ref<number | null>(null)
const detail = ref<AccountBackupVersion | null>(null)
const expandedId = ref<number | null>(null)
const renameId = ref<number | null>(null)
const renameDraft = ref('')

const sourceReasons = new Set([
  'before_manual_backup',
  'before_restore',
  'before_character_card_writeback',
  'restore_sync',
  'sync',
  'manual',
])

const detailProfiles = computed(() => {
  if (!detail.value?.profiles_data) return []
  try {
    const profiles = JSON.parse(detail.value.profiles_data) as Record<string, Record<string, unknown>>
    return Object.entries(profiles).map(([id, profile]) => ({
      id,
      name: String(profile.profileName || id),
    }))
  } catch {
    return []
  }
})

watch(() => props.accountId, () => void loadVersions())
onMounted(() => void loadVersions())

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale.value, { hour12: false })
}

function versionName(version: AccountBackupVersion) {
  return version.name?.trim() || `${formatDate(version.created_at)} · ${version.checksum.slice(0, 8)}`
}

function sourceReason(version: AccountBackupVersion) {
  const reason = version.change_log?.trim() || ''
  return sourceReasons.has(reason) ? reason : 'unknown'
}

function sourceLabel(version: AccountBackupVersion) {
  return t(`characterCards.versions.common.sources.${sourceReason(version)}`)
}

function sourceIcon(version: AccountBackupVersion) {
  switch (sourceReason(version)) {
    case 'before_manual_backup': return 'ri-save-3-line'
    case 'before_restore': return 'ri-shield-flash-line'
    case 'before_character_card_writeback': return 'ri-file-transfer-line'
    case 'restore_sync': return 'ri-loop-left-line'
    case 'manual': return 'ri-user-line'
    case 'sync': return 'ri-refresh-line'
    default: return 'ri-archive-line'
  }
}

async function loadVersions() {
  if (!props.accountId) {
    versions.value = []
    return
  }
  loading.value = true
  try {
    versions.value = await getAccountBackupVersions(props.accountId)
  } catch (error: unknown) {
    versions.value = []
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.cloud.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function toggleDetail(version: AccountBackupVersion) {
  if (expandedId.value === version.id) {
    expandedId.value = null
    detail.value = null
    return
  }
  busyId.value = version.id
  try {
    detail.value = await getAccountBackupVersion(props.accountId, version.id)
    expandedId.value = version.id
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.cloud.detailFailed'))
  } finally {
    busyId.value = null
  }
}

function startRename(version: AccountBackupVersion) {
  renameId.value = version.id
  renameDraft.value = versionName(version)
}

async function saveRename(version: AccountBackupVersion) {
  const name = renameDraft.value.trim()
  if (!name || busyId.value !== null) return
  busyId.value = version.id
  try {
    const updated = await renameAccountBackupVersion(props.accountId, version.id, name)
    versions.value = versions.value.map((item) => item.id === updated.id ? { ...item, ...updated } : item)
    renameId.value = null
    toast.success(t('characterCards.versions.cloud.renamed'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.cloud.renameFailed'))
  } finally {
    busyId.value = null
  }
}

async function restoreVersion(version: AccountBackupVersion) {
  const confirmed = await dialog.confirm({
    title: t('characterCards.versions.cloud.restoreTitle'),
    message: t('characterCards.versions.cloud.restoreMessage', {
      account: props.accountId,
      name: versionName(version),
    }),
    type: 'error',
    confirmText: t('characterCards.versions.cloud.restoreConfirm'),
  })
  if (!confirmed) return
  busyId.value = version.id
  try {
    const result = await restoreAccountBackupVersion(
      props.accountId,
      version.id,
      t('characterCards.versions.cloud.restoreSnapshotName', { time: formatDate(new Date().toISOString()) }),
    )
    emit('restored', result.backup)
    await loadVersions()
    toast.success(t('characterCards.versions.cloud.restored'), 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.cloud.restoreFailed'))
  } finally {
    busyId.value = null
  }
}

async function deleteVersion(version: AccountBackupVersion) {
  const confirmed = await dialog.confirm({
    title: t('characterCards.versions.cloud.deleteTitle'),
    message: t('characterCards.versions.cloud.deleteMessage', { name: versionName(version) }),
    type: 'error',
    confirmText: t('characterCards.versions.cloud.deleteConfirm'),
  })
  if (!confirmed) return
  busyId.value = version.id
  try {
    await deleteAccountBackupVersion(props.accountId, version.id)
    versions.value = versions.value.filter((item) => item.id !== version.id)
    if (expandedId.value === version.id) {
      expandedId.value = null
      detail.value = null
    }
    toast.success(t('characterCards.versions.cloud.deleted'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.cloud.deleteFailed'))
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <section class="cloud-history" :aria-label="t('characterCards.versions.cloud.aria')">
    <header class="cloud-history__header">
      <div><span>{{ t('characterCards.versions.cloud.kicker') }}</span><h3>{{ t('characterCards.versions.cloud.title') }}</h3></div>
      <button type="button" :disabled="loading" @click="loadVersions"><i class="ri-refresh-line"></i>{{ t('characterCards.versions.common.refresh') }}</button>
    </header>
    <div v-if="loading" class="cloud-history__state" role="status"><i class="ri-loader-4-line cloud-history__spin"></i>{{ t('characterCards.versions.cloud.loading') }}</div>
    <div v-else-if="!versions.length" class="cloud-history__state"><i class="ri-history-line"></i>{{ t('characterCards.versions.cloud.empty') }}</div>
    <ol v-else class="cloud-history__list">
      <li v-for="version in versions" :key="version.id" class="cloud-version">
        <span class="cloud-version__number">V{{ version.version }}</span>
        <div class="cloud-version__main">
          <div class="cloud-version__source" :data-reason="sourceReason(version)">
            <i :class="sourceIcon(version)" aria-hidden="true"></i>
            {{ sourceLabel(version) }}
          </div>
          <label v-if="renameId === version.id" class="cloud-version__rename">
            <span class="sr-only">{{ t('characterCards.versions.common.versionName') }}</span>
            <input v-model="renameDraft" maxlength="120" @keydown.enter.prevent="saveRename(version)" @keydown.esc="renameId = null" />
            <button type="button" :disabled="!renameDraft.trim() || busyId === version.id" @click="saveRename(version)">{{ t('characterCards.versions.common.save') }}</button>
            <button type="button" @click="renameId = null">{{ t('characterCards.versions.common.cancel') }}</button>
          </label>
          <strong v-else class="cloud-version__name">{{ versionName(version) }}</strong>
          <div class="cloud-version__facts">
            <time :datetime="version.created_at"><i class="ri-time-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.savedAt', { time: formatDate(version.created_at) }) }}</time>
            <span><i class="ri-fingerprint-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.hash', { hash: version.checksum.slice(0, 10) }) }}</span>
          </div>
          <div class="cloud-version__metrics">
            <span><i class="ri-id-card-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.profiles', { count: version.profiles_count || 0 }) }}</span>
            <span><i class="ri-treasure-map-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.tools', { count: version.tools_count || 0 }) }}</span>
            <span><i class="ri-database-2-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.runtime', { size: version.runtime_size_kb || 0 }) }}</span>
          </div>
        </div>
        <div class="cloud-version__actions">
          <button type="button" :disabled="busyId !== null" @click="toggleDetail(version)">{{ expandedId === version.id ? t('characterCards.versions.common.collapse') : t('characterCards.versions.common.view') }}</button>
          <button type="button" :disabled="busyId !== null" @click="startRename(version)">{{ t('characterCards.versions.common.rename') }}</button>
          <button type="button" class="restore" :disabled="busyId !== null" @click="restoreVersion(version)">{{ t('characterCards.versions.common.restore') }}</button>
          <button type="button" class="danger" :disabled="busyId !== null" @click="deleteVersion(version)">{{ t('characterCards.versions.common.delete') }}</button>
        </div>
        <div v-if="expandedId === version.id && detail" class="cloud-version__detail">
          <div v-if="detailProfiles.length" class="cloud-version__profiles">
            <b v-for="profile in detailProfiles" :key="profile.id">{{ profile.name }} <small>{{ profile.id.slice(0, 8) }}</small></b>
          </div>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.cloud-history { display: grid; gap: 14px; color: var(--color-text-main); }
.cloud-history__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--color-border); }
.cloud-history__header > div { display: grid; gap: 2px; }
.cloud-history__header span { color: var(--color-accent); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .13em; text-transform: uppercase; }
.cloud-history__header h3 { margin: 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 17px; }
.cloud-history button { padding: 6px 9px; border: 1px solid var(--btn-outline-border); border-radius: 6px; background: var(--btn-outline-hover); color: var(--btn-outline-text); cursor: pointer; font: inherit; font-size: 10px; font-weight: 700; }
.cloud-history button:disabled { cursor: not-allowed; opacity: .45; }
.cloud-history__header button { display: inline-flex; align-items: center; gap: 4px; }
.cloud-history__state { display: flex; min-height: 140px; align-items: center; justify-content: center; gap: 7px; color: var(--color-text-secondary); font-size: 11px; }
.cloud-history__list { display: grid; max-height: 58vh; gap: 9px; margin: 0; padding: 0 3px 0 0; overflow-y: auto; list-style: none; }
.cloud-version { display: grid; grid-template-columns: 44px minmax(0, 1fr) auto; gap: 10px 12px; align-items: center; padding: 12px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-card-bg); box-shadow: var(--shadow-sm); }
.cloud-version__number { display: grid; width: 40px; height: 40px; place-items: center; border: 1px solid var(--color-border-hover); border-radius: 50%; background: var(--color-panel-bg); color: var(--color-accent); font: 800 10px/1 ui-monospace, Consolas, monospace; }
.cloud-version__main { display: grid; min-width: 0; gap: 7px; }
.cloud-version__source { display: inline-flex; width: max-content; max-width: 100%; align-items: center; gap: 5px; padding: 4px 7px; border-radius: 999px; background: var(--tag-bg); color: var(--tag-text); font-size: 9px; font-weight: 800; }
.cloud-version__source[data-reason="before_restore"],
.cloud-version__source[data-reason="before_character_card_writeback"] { background: var(--color-warning-light); color: var(--color-warning-dark); }
.cloud-version__source[data-reason="before_manual_backup"],
.cloud-version__source[data-reason="manual"] { background: var(--btn-secondary-bg); color: var(--btn-secondary-text); }
.cloud-version__name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: Georgia, 'Noto Serif SC', serif; font-size: 13px; }
.cloud-version__facts { display: flex; flex-wrap: wrap; gap: 5px 12px; color: var(--color-text-muted); font: 9px/1.4 ui-monospace, Consolas, monospace; }
.cloud-version__facts time, .cloud-version__facts span { display: inline-flex; align-items: center; gap: 4px; }
.cloud-version__metrics { display: flex; flex-wrap: wrap; gap: 6px; }
.cloud-version__metrics span { display: inline-flex; align-items: center; gap: 4px; padding: 5px 8px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-panel-bg); color: var(--color-text-secondary); font-size: 10px; font-weight: 700; }
.cloud-version__metrics i { color: var(--color-accent); }
.cloud-version__actions { display: flex; max-width: 150px; flex-wrap: wrap; justify-content: flex-end; gap: 4px; }
.cloud-version__actions .restore { border-color: var(--btn-primary-bg); background: var(--btn-primary-bg); color: var(--btn-primary-text); }
.cloud-version__actions .danger { color: var(--btn-danger-bg); }
.cloud-version__rename { display: grid; grid-template-columns: minmax(120px, 1fr) auto auto; gap: 4px; }
.cloud-version__rename input { min-width: 0; padding: 7px 8px; border: 1px solid var(--input-border); border-radius: 6px; outline: none; background: var(--input-bg); color: var(--color-text-main); font: inherit; }
.cloud-version__detail { grid-column: 2 / -1; padding-top: 9px; border-top: 1px dashed var(--color-border-hover); }
.cloud-version__profiles { display: flex; width: 100%; flex-wrap: wrap; gap: 5px; }
.cloud-version__profiles b { padding: 5px 7px; border: 1px solid var(--color-border); border-radius: 5px; background: var(--color-panel-bg); color: var(--color-text-main); font-size: 10px; }
.cloud-version__profiles small { color: var(--color-text-muted); font-family: ui-monospace, Consolas, monospace; }
.cloud-history button:focus-visible, .cloud-version__rename input:focus-visible { outline: 3px solid color-mix(in srgb, var(--input-focus) 28%, transparent); outline-offset: 2px; }
.cloud-history__spin { animation: cloud-history-spin 900ms linear infinite; }
@keyframes cloud-history-spin { to { transform: rotate(360deg); } }
@media (max-width: 680px) { .cloud-version { grid-template-columns: 42px minmax(0, 1fr); } .cloud-version__actions { grid-column: 2; max-width: none; justify-content: flex-start; } .cloud-version__detail { grid-column: 1 / -1; } }
@media (prefers-reduced-motion: reduce) { .cloud-history__spin { animation: none; } }
</style>
