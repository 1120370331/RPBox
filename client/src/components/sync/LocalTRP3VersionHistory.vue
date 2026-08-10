<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'

export interface LocalTRP3Snapshot {
  id: string
  name: string
  account_id: string
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

const dialog = useDialog()
const toast = useToastStore()
const snapshots = ref<LocalTRP3Snapshot[]>([])
const loading = ref(false)
const busyId = ref('')
const expandedId = ref('')
const detail = ref<LocalTRP3SnapshotDetail | null>(null)
const renameId = ref('')
const renameDraft = ref('')

watch(() => [props.wowPath, props.accountId] as const, () => void loadSnapshots())
onMounted(() => void loadSnapshots())

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  return `${(bytes / 1024).toFixed(bytes < 10 * 1024 ? 1 : 0)} KB`
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
    toast.error(error instanceof Error ? error.message : '本地版本读取失败')
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
    toast.error(error instanceof Error ? error.message : '本地版本内容读取失败')
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
    toast.success('本地版本已命名')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '本地版本命名失败')
  } finally {
    busyId.value = ''
  }
}

async function restoreSnapshot(snapshot: LocalTRP3Snapshot) {
  const confirmed = await dialog.confirm({
    title: '回退本地 totalRP3.lua',
    message: `将把账号「${props.accountId}」恢复到“${snapshot.name}”。恢复前会先为当前文件自动建立一份新的安全快照；请确认魔兽世界已经关闭。`,
    type: 'error',
    confirmText: '建立快照并回退',
  })
  if (!confirmed) return
  busyId.value = snapshot.id
  try {
    await invoke('restore_local_trp3_snapshot', {
      wowPath: props.wowPath,
      accountId: props.accountId,
      snapshotId: snapshot.id,
      safetySnapshotName: `回退前 · ${new Date().toLocaleString('zh-CN', { hour12: false })}`,
    })
    await loadSnapshots()
    emit('restored', snapshot)
    toast.success('本地 Lua 已回退；回退前内容已自动保存为新版本', 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '本地版本回退失败')
  } finally {
    busyId.value = ''
  }
}

async function deleteSnapshot(snapshot: LocalTRP3Snapshot) {
  const confirmed = await dialog.confirm({
    title: '删除本地 Lua 版本',
    message: `确定永久删除“${snapshot.name}”吗？删除版本不会改变当前 totalRP3.lua，但该快照无法恢复。`,
    type: 'error',
    confirmText: '永久删除版本',
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
    toast.success('本地版本已删除')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '本地版本删除失败')
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <section class="local-history" aria-label="本地 TRP3 Lua 版本">
    <header class="local-history__header">
      <div><span>Local safety reel</span><h3>{{ accountId }}</h3></div>
      <button type="button" :disabled="loading" @click="loadSnapshots"><i class="ri-refresh-line"></i>刷新</button>
    </header>

    <div v-if="loading" class="local-history__state" role="status"><i class="ri-loader-4-line local-history__spin"></i>正在读取本地版本…</div>
    <div v-else-if="!snapshots.length" class="local-history__state"><i class="ri-history-line"></i>尚无独立版本；第一次写回会自动创建。</div>
    <ol v-else class="local-history__list">
      <li v-for="snapshot in snapshots" :key="snapshot.id" class="local-version">
        <div class="local-version__rail" aria-hidden="true"></div>
        <div class="local-version__copy">
          <template v-if="renameId === snapshot.id">
            <label class="local-version__rename">
              <span class="sr-only">版本名称</span>
              <input v-model="renameDraft" maxlength="120" @keydown.enter.prevent="saveRename(snapshot)" @keydown.esc="renameId = ''" />
              <button type="button" :disabled="!renameDraft.trim() || busyId === snapshot.id" @click="saveRename(snapshot)">保存</button>
              <button type="button" @click="renameId = ''">取消</button>
            </label>
          </template>
          <template v-else>
            <strong>{{ snapshot.name }}</strong>
            <span>{{ formatDate(snapshot.created_at) }} · {{ snapshot.checksum.slice(0, 10) }} · {{ formatSize(snapshot.size_bytes) }}</span>
          </template>
        </div>
        <div class="local-version__actions">
          <button type="button" :disabled="Boolean(busyId)" @click="toggleDetail(snapshot)">{{ expandedId === snapshot.id ? '收起' : '查看' }}</button>
          <button type="button" :disabled="Boolean(busyId)" @click="startRename(snapshot)">命名</button>
          <button type="button" class="restore" :disabled="Boolean(busyId)" @click="restoreSnapshot(snapshot)">回退</button>
          <button type="button" class="danger" :disabled="Boolean(busyId)" @click="deleteSnapshot(snapshot)">删除</button>
        </div>
        <pre v-if="expandedId === snapshot.id && detail" class="local-version__preview">{{ detail.content }}</pre>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.local-history { --copper: #B87333; --rust: #804030; --ink: #2C1810; --muted: #8C7B70; display: grid; gap: 14px; color: var(--ink); }
.local-history__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 10px; border-bottom: 1px solid #E3D4C5; }
.local-history__header > div { display: grid; gap: 2px; }
.local-history__header span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .14em; text-transform: uppercase; }
.local-history__header h3 { margin: 0; font-family: Georgia, 'Noto Serif SC', serif; }
.local-history button { border: 1px solid #D5B797; border-radius: 6px; background: #FFF; color: var(--rust); cursor: pointer; font: inherit; font-size: 10px; font-weight: 700; }
.local-history button:disabled { cursor: not-allowed; opacity: .45; }
.local-history__header button { display: inline-flex; gap: 5px; align-items: center; padding: 7px 10px; }
.local-history__state { display: flex; min-height: 180px; align-items: center; justify-content: center; gap: 8px; color: var(--muted); font-size: 11px; }
.local-history__list { display: grid; max-height: 58vh; gap: 9px; margin: 0; padding: 0 3px 0 0; overflow-y: auto; list-style: none; }
.local-version { position: relative; display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 10px 14px; overflow: hidden; padding: 13px 14px 13px 22px; border: 1px solid #E1CEBA; border-radius: 9px; background: #FCF8F3; }
.local-version__rail { position: absolute; inset: 0 auto 0 0; width: 8px; background: repeating-linear-gradient(to bottom, #4A3023 0 8px, #C89058 8px 12px); }
.local-version__copy { display: grid; min-width: 0; gap: 3px; }
.local-version__copy strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: Georgia, 'Noto Serif SC', serif; font-size: 12px; }
.local-version__copy > span { color: var(--muted); font: 9px/1.4 ui-monospace, Consolas, monospace; }
.local-version__actions { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 4px; }
.local-version__actions button { padding: 6px 8px; }
.local-version__actions .restore { border-color: #B87333; background: #804030; color: #FFF7EF; }
.local-version__actions .danger { color: #9E4136; }
.local-version__rename { display: grid; grid-template-columns: minmax(120px, 1fr) auto auto; gap: 5px; }
.local-version__rename input { min-width: 0; padding: 7px 8px; border: 1px solid #C99B70; border-radius: 6px; outline: none; font: inherit; }
.local-version__rename input:focus { box-shadow: 0 0 0 3px rgba(184,115,51,.13); }
.local-version__preview { grid-column: 1 / -1; max-height: 250px; margin: 0; padding: 12px; overflow: auto; border-radius: 6px; background: #261810; color: #EBD7C2; font: 9px/1.6 ui-monospace, Consolas, monospace; white-space: pre-wrap; word-break: break-word; }
.local-history button:focus-visible, .local-version__rename input:focus-visible { outline: 3px solid rgba(184,115,51,.28); outline-offset: 2px; }
.local-history__spin { animation: local-history-spin 900ms linear infinite; }
@keyframes local-history-spin { to { transform: rotate(360deg); } }
@media (max-width: 620px) { .local-version { grid-template-columns: 1fr; } .local-version__actions { justify-content: flex-start; } }
@media (prefers-reduced-motion: reduce) { .local-history__spin { animation: none; } }
</style>
