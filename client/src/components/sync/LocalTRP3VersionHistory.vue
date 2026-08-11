<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { useI18n } from 'vue-i18n'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'

export interface LocalTRP3Snapshot {
  id: string
  name: string
  account_id: string
  reason?: string
  checksum: string
  created_at: string
  size_bytes: number
}

interface LocalTRP3SnapshotDetail extends LocalTRP3Snapshot {
  content: string
}

const props = defineProps<{
  wowPath: string
  accountId: string
}>()

const emit = defineEmits<{
  restored: [snapshot: LocalTRP3Snapshot]
}>()

const { t, locale } = useI18n()
const dialog = useDialog()
const toast = useToastStore()
const snapshots = ref<LocalTRP3Snapshot[]>([])
const loading = ref(false)
const busyId = ref('')
const expandedId = ref('')
const detail = ref<LocalTRP3SnapshotDetail | null>(null)
const renameId = ref('')
const renameDraft = ref('')

const sourceReasons = new Set([
  'before_manual_backup',
  'before_restore',
  'before_character_card_writeback',
  'restore_sync',
  'sync',
  'manual',
])

watch(() => [props.wowPath, props.accountId] as const, () => void loadSnapshots())
onMounted(() => void loadSnapshots())

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale.value, { hour12: false })
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${new Intl.NumberFormat(locale.value).format(bytes)} B`
  const value = bytes / 1024
  return `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: value < 10 ? 1 : 0 }).format(value)} KB`
}

function sourceReason(snapshot: LocalTRP3Snapshot) {
  const reason = snapshot.reason?.trim() || ''
  return sourceReasons.has(reason) ? reason : 'unknown'
}

function sourceLabel(snapshot: LocalTRP3Snapshot) {
  return t(`characterCards.versions.common.sources.${sourceReason(snapshot)}`)
}

function sourceIcon(snapshot: LocalTRP3Snapshot) {
  switch (sourceReason(snapshot)) {
    case 'before_restore': return 'ri-shield-flash-line'
    case 'before_character_card_writeback': return 'ri-file-transfer-line'
    case 'before_manual_backup': return 'ri-save-3-line'
    case 'restore_sync': return 'ri-loop-left-line'
    case 'manual': return 'ri-user-line'
    case 'sync': return 'ri-refresh-line'
    default: return 'ri-archive-line'
  }
}

async function loadSnapshots() {
  if (!props.wowPath || !props.accountId) {
    snapshots.value = []
    return
  }
  loading.value = true
  try {
    snapshots.value = await invoke<LocalTRP3Snapshot[]>('list_local_trp3_snapshots', {
      wowPath: props.wowPath,
      accountId: props.accountId,
    })
  } catch (error: unknown) {
    snapshots.value = []
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.local.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function toggleDetail(snapshot: LocalTRP3Snapshot) {
  if (expandedId.value === snapshot.id) {
    expandedId.value = ''
    detail.value = null
    return
  }
  busyId.value = snapshot.id
  try {
    detail.value = await invoke<LocalTRP3SnapshotDetail>('read_local_trp3_snapshot', {
      wowPath: props.wowPath,
      accountId: props.accountId,
      snapshotId: snapshot.id,
    })
    expandedId.value = snapshot.id
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.local.detailFailed'))
  } finally {
    busyId.value = ''
  }
}

function startRename(snapshot: LocalTRP3Snapshot) {
  renameId.value = snapshot.id
  renameDraft.value = snapshot.name
}

async function saveRename(snapshot: LocalTRP3Snapshot) {
  const name = renameDraft.value.trim()
  if (!name || busyId.value) return
  busyId.value = snapshot.id
  try {
    const updated = await invoke<LocalTRP3Snapshot>('rename_local_trp3_snapshot', {
      wowPath: props.wowPath,
      accountId: props.accountId,
      snapshotId: snapshot.id,
      name,
    })
    snapshots.value = snapshots.value.map((item) => item.id === updated.id ? updated : item)
    renameId.value = ''
    toast.success(t('characterCards.versions.local.renamed'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.local.renameFailed'))
  } finally {
    busyId.value = ''
  }
}

async function restoreSnapshot(snapshot: LocalTRP3Snapshot) {
  const confirmed = await dialog.confirm({
    title: t('characterCards.versions.local.restoreTitle'),
    message: t('characterCards.versions.local.restoreMessage', { account: props.accountId, name: snapshot.name }),
    type: 'error',
    confirmText: t('characterCards.versions.local.restoreConfirm'),
  })
  if (!confirmed) return
  busyId.value = snapshot.id
  try {
    await invoke('restore_local_trp3_snapshot', {
      wowPath: props.wowPath,
      accountId: props.accountId,
      snapshotId: snapshot.id,
      safetySnapshotName: t('characterCards.versions.local.restoreSnapshotName', { time: formatDate(new Date().toISOString()) }),
    })
    await loadSnapshots()
    emit('restored', snapshot)
    toast.success(t('characterCards.versions.local.restored'), 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.local.restoreFailed'))
  } finally {
    busyId.value = ''
  }
}

async function deleteSnapshot(snapshot: LocalTRP3Snapshot) {
  const confirmed = await dialog.confirm({
    title: t('characterCards.versions.local.deleteTitle'),
    message: t('characterCards.versions.local.deleteMessage', { name: snapshot.name }),
    type: 'error',
    confirmText: t('characterCards.versions.local.deleteConfirm'),
  })
  if (!confirmed) return
  busyId.value = snapshot.id
  try {
    await invoke('delete_local_trp3_snapshot', {
      wowPath: props.wowPath,
      accountId: props.accountId,
      snapshotId: snapshot.id,
    })
    snapshots.value = snapshots.value.filter((item) => item.id !== snapshot.id)
    if (expandedId.value === snapshot.id) {
      expandedId.value = ''
      detail.value = null
    }
    toast.success(t('characterCards.versions.local.deleted'))
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.versions.local.deleteFailed'))
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <section class="local-history" :aria-label="t('characterCards.versions.local.aria')">
    <header class="local-history__header">
      <div><span>{{ t('characterCards.versions.local.kicker') }}</span><h3>{{ accountId }}</h3></div>
      <button type="button" :disabled="loading" @click="loadSnapshots"><i class="ri-refresh-line"></i>{{ t('characterCards.versions.common.refresh') }}</button>
    </header>

    <div v-if="loading" class="local-history__state" role="status"><i class="ri-loader-4-line local-history__spin"></i>{{ t('characterCards.versions.local.loading') }}</div>
    <div v-else-if="!snapshots.length" class="local-history__state"><i class="ri-history-line"></i>{{ t('characterCards.versions.local.empty') }}</div>
    <ol v-else class="local-history__list">
      <li v-for="snapshot in snapshots" :key="snapshot.id" class="local-version">
        <div class="local-version__rail" aria-hidden="true"></div>
        <div class="local-version__main">
          <div class="local-version__source" :data-reason="sourceReason(snapshot)">
            <i :class="sourceIcon(snapshot)" aria-hidden="true"></i>{{ sourceLabel(snapshot) }}
          </div>
          <template v-if="renameId === snapshot.id">
            <label class="local-version__rename">
              <span class="sr-only">{{ t('characterCards.versions.common.versionName') }}</span>
              <input v-model="renameDraft" maxlength="120" @keydown.enter.prevent="saveRename(snapshot)" @keydown.esc="renameId = ''" />
              <button type="button" :disabled="!renameDraft.trim() || busyId === snapshot.id" @click="saveRename(snapshot)">{{ t('characterCards.versions.common.save') }}</button>
              <button type="button" @click="renameId = ''">{{ t('characterCards.versions.common.cancel') }}</button>
            </label>
          </template>
          <strong v-else>{{ snapshot.name }}</strong>
          <div class="local-version__facts">
            <time :datetime="snapshot.created_at"><i class="ri-time-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.savedAt', { time: formatDate(snapshot.created_at) }) }}</time>
            <span><i class="ri-fingerprint-line" aria-hidden="true"></i>{{ t('characterCards.versions.common.hash', { hash: snapshot.checksum.slice(0, 10) }) }}</span>
            <span><i class="ri-file-code-line" aria-hidden="true"></i>{{ t('characterCards.versions.local.size', { size: formatSize(snapshot.size_bytes) }) }}</span>
          </div>
        </div>
        <div class="local-version__actions">
          <button type="button" :disabled="Boolean(busyId)" @click="toggleDetail(snapshot)">{{ expandedId === snapshot.id ? t('characterCards.versions.common.collapse') : t('characterCards.versions.common.view') }}</button>
          <button type="button" :disabled="Boolean(busyId)" @click="startRename(snapshot)">{{ t('characterCards.versions.common.rename') }}</button>
          <button type="button" class="restore" :disabled="Boolean(busyId)" @click="restoreSnapshot(snapshot)">{{ t('characterCards.versions.common.restore') }}</button>
          <button type="button" class="danger" :disabled="Boolean(busyId)" @click="deleteSnapshot(snapshot)">{{ t('characterCards.versions.common.delete') }}</button>
        </div>
        <pre v-if="expandedId === snapshot.id && detail" class="local-version__preview">{{ detail.content }}</pre>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.local-history { display: grid; gap: 14px; color: var(--color-text-main); }
.local-history__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--color-border); }
.local-history__header > div { display: grid; gap: 2px; }
.local-history__header span { color: var(--color-accent); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .14em; text-transform: uppercase; }
.local-history__header h3 { margin: 0; font-family: Georgia, 'Noto Serif SC', serif; }
.local-history button { border: 1px solid var(--btn-outline-border); border-radius: 6px; background: var(--btn-outline-hover); color: var(--btn-outline-text); cursor: pointer; font: inherit; font-size: 10px; font-weight: 700; }
.local-history button:disabled { cursor: not-allowed; opacity: .45; }
.local-history__header button { display: inline-flex; gap: 5px; align-items: center; padding: 7px 10px; }
.local-history__state { display: flex; min-height: 180px; align-items: center; justify-content: center; gap: 8px; color: var(--color-text-secondary); font-size: 11px; }
.local-history__list { display: grid; max-height: 58vh; gap: 9px; margin: 0; padding: 0 3px 0 0; overflow-y: auto; list-style: none; }
.local-version { position: relative; display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 10px 14px; overflow: hidden; padding: 13px 14px 13px 22px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-card-bg); box-shadow: var(--shadow-sm); }
.local-version__rail { position: absolute; inset: 0 auto 0 0; width: 8px; background: repeating-linear-gradient(to bottom, var(--gradient-start) 0 8px, var(--color-accent) 8px 12px); }
.local-version__main { display: grid; min-width: 0; gap: 7px; }
.local-version__main > strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: Georgia, 'Noto Serif SC', serif; font-size: 12px; }
.local-version__source { display: inline-flex; width: max-content; max-width: 100%; align-items: center; gap: 5px; padding: 4px 7px; border-radius: 999px; background: var(--tag-bg); color: var(--tag-text); font-size: 9px; font-weight: 800; }
.local-version__source[data-reason="before_restore"],
.local-version__source[data-reason="before_character_card_writeback"] { background: var(--color-warning-light); color: var(--color-warning-dark); }
.local-version__source[data-reason="manual"] { background: var(--btn-secondary-bg); color: var(--btn-secondary-text); }
.local-version__facts { display: flex; flex-wrap: wrap; gap: 5px 12px; color: var(--color-text-muted); font: 9px/1.4 ui-monospace, Consolas, monospace; }
.local-version__facts time, .local-version__facts span { display: inline-flex; align-items: center; gap: 4px; }
.local-version__actions { display: flex; max-width: 150px; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 4px; }
.local-version__actions button { padding: 6px 8px; }
.local-version__actions .restore { border-color: var(--btn-primary-bg); background: var(--btn-primary-bg); color: var(--btn-primary-text); }
.local-version__actions .danger { color: var(--btn-danger-bg); }
.local-version__rename { display: grid; grid-template-columns: minmax(120px, 1fr) auto auto; gap: 5px; }
.local-version__rename input { min-width: 0; padding: 7px 8px; border: 1px solid var(--input-border); border-radius: 6px; outline: none; background: var(--input-bg); color: var(--color-text-main); font: inherit; }
.local-version__rename input:focus { box-shadow: 0 0 0 3px color-mix(in srgb, var(--input-focus) 14%, transparent); }
.local-version__preview { grid-column: 1 / -1; max-height: 250px; margin: 0; padding: 12px; overflow: auto; border-radius: 6px; background: var(--gradient-end); color: var(--gradient-text); font: 9px/1.6 ui-monospace, Consolas, monospace; white-space: pre-wrap; word-break: break-word; }
.local-history button:focus-visible, .local-version__rename input:focus-visible { outline: 3px solid color-mix(in srgb, var(--input-focus) 28%, transparent); outline-offset: 2px; }
.local-history__spin { animation: local-history-spin 900ms linear infinite; }
@keyframes local-history-spin { to { transform: rotate(360deg); } }
@media (max-width: 620px) { .local-version { grid-template-columns: 1fr; } .local-version__actions { max-width: none; justify-content: flex-start; } }
@media (prefers-reduced-motion: reduce) { .local-history__spin { animation: none; } }
</style>
