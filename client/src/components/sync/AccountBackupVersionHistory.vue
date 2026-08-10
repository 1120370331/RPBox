<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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

const dialog = useDialog()
const toast = useToastStore()
const versions = ref<AccountBackupVersion[]>([])
const loading = ref(false)
const busyId = ref<number | null>(null)
const detail = ref<AccountBackupVersion | null>(null)
const expandedId = ref<number | null>(null)
const renameId = ref<number | null>(null)
const renameDraft = ref('')

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
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function versionName(version: AccountBackupVersion) {
  return version.name?.trim() || `${formatDate(version.created_at)} · ${version.checksum.slice(0, 8)}`
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
    toast.error(error instanceof Error ? error.message : '云端版本历史加载失败')
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
    toast.error(error instanceof Error ? error.message : '云端版本详情加载失败')
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
    toast.success('云端版本已命名')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '云端版本命名失败')
  } finally {
    busyId.value = null
  }
}

async function restoreVersion(version: AccountBackupVersion) {
  const confirmed = await dialog.confirm({
    title: '回退云端账号备份',
    message: `将把账号「${props.accountId}」回退到“${versionName(version)}”。系统会先把当前云端数据自动保存为新版本；本地文件不会在此步骤中被直接覆盖。`,
    type: 'error',
    confirmText: '建立快照并回退',
  })
  if (!confirmed) return
  busyId.value = version.id
  try {
    const result = await restoreAccountBackupVersion(
      props.accountId,
      version.id,
      `云端回退前 · ${new Date().toLocaleString('zh-CN', { hour12: false })}`,
    )
    emit('restored', result.backup)
    await loadVersions()
    toast.success('云端备份已回退；回退前状态已自动保留', 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '云端版本回退失败')
  } finally {
    busyId.value = null
  }
}

async function deleteVersion(version: AccountBackupVersion) {
  const confirmed = await dialog.confirm({
    title: '删除云端历史版本',
    message: `确定永久删除“${versionName(version)}”吗？当前云端备份不会改变，但该历史版本将无法恢复。`,
    type: 'error',
    confirmText: '永久删除版本',
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
    toast.success('云端历史版本已删除')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '云端历史版本删除失败')
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <section class="cloud-history" aria-label="云端账号备份版本">
    <header class="cloud-history__header">
      <div><span>Cloud archive reel</span><h3>版本历史</h3></div>
      <button type="button" :disabled="loading" @click="loadVersions"><i class="ri-refresh-line"></i>刷新</button>
    </header>
    <div v-if="loading" class="cloud-history__state" role="status"><i class="ri-loader-4-line cloud-history__spin"></i>正在读取云端版本…</div>
    <div v-else-if="!versions.length" class="cloud-history__state"><i class="ri-history-line"></i>尚无历史版本；下一次备份或写回会建立第一条。</div>
    <ol v-else class="cloud-history__list">
      <li v-for="version in versions" :key="version.id" class="cloud-version">
        <span class="cloud-version__number">V{{ version.version }}</span>
        <div class="cloud-version__copy">
          <label v-if="renameId === version.id" class="cloud-version__rename">
            <span class="sr-only">版本名称</span>
            <input v-model="renameDraft" maxlength="120" @keydown.enter.prevent="saveRename(version)" @keydown.esc="renameId = null" />
            <button type="button" :disabled="!renameDraft.trim() || busyId === version.id" @click="saveRename(version)">保存</button>
            <button type="button" @click="renameId = null">取消</button>
          </label>
          <template v-else>
            <strong>{{ versionName(version) }}</strong>
            <span>{{ formatDate(version.created_at) }} · {{ version.checksum.slice(0, 10) }}</span>
          </template>
        </div>
        <div class="cloud-version__actions">
          <button type="button" :disabled="busyId !== null" @click="toggleDetail(version)">{{ expandedId === version.id ? '收起' : '查看' }}</button>
          <button type="button" :disabled="busyId !== null" @click="startRename(version)">命名</button>
          <button type="button" class="restore" :disabled="busyId !== null" @click="restoreVersion(version)">回退</button>
          <button type="button" class="danger" :disabled="busyId !== null" @click="deleteVersion(version)">删除</button>
        </div>
        <div v-if="expandedId === version.id && detail" class="cloud-version__detail">
          <span>人物卡 {{ detailProfiles.length || detail.profiles_count || 0 }}</span>
          <span>道具 {{ detail.tools_count || 0 }}</span>
          <span>运行时 {{ detail.runtime_size_kb || 0 }} KB</span>
          <div v-if="detailProfiles.length" class="cloud-version__profiles">
            <b v-for="profile in detailProfiles" :key="profile.id">{{ profile.name }} <small>{{ profile.id.slice(0, 8) }}</small></b>
          </div>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.cloud-history { --copper: #B87333; --rust: #804030; --ink: #2C1810; --muted: #8C7B70; display: grid; gap: 12px; color: var(--ink); }
.cloud-history__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.cloud-history__header > div { display: grid; gap: 2px; }
.cloud-history__header span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: .13em; text-transform: uppercase; }
.cloud-history__header h3 { margin: 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 17px; }
.cloud-history button { padding: 6px 9px; border: 1px solid #D8B898; border-radius: 6px; background: #FFF; color: var(--rust); cursor: pointer; font: inherit; font-size: 10px; font-weight: 700; }
.cloud-history button:disabled { cursor: not-allowed; opacity: .45; }
.cloud-history__header button { display: inline-flex; align-items: center; gap: 4px; }
.cloud-history__state { display: flex; min-height: 140px; align-items: center; justify-content: center; gap: 7px; color: var(--muted); font-size: 11px; }
.cloud-history__list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
.cloud-version { display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 10px; align-items: center; padding: 11px; border: 1px solid #E0CCB7; border-radius: 9px; background: #FCF8F3; }
.cloud-version__number { display: grid; width: 38px; height: 38px; place-items: center; border: 1px solid #C7986A; border-radius: 50%; color: var(--rust); font: 800 10px/1 ui-monospace, Consolas, monospace; }
.cloud-version__copy { display: grid; min-width: 0; gap: 3px; }
.cloud-version__copy strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: Georgia, 'Noto Serif SC', serif; font-size: 12px; }
.cloud-version__copy > span { color: var(--muted); font: 9px/1.4 ui-monospace, Consolas, monospace; }
.cloud-version__actions { display: flex; flex-wrap: wrap; gap: 4px; }
.cloud-version__actions .restore { border-color: #9D5E3D; background: var(--rust); color: #FFF8EF; }
.cloud-version__actions .danger { color: #9B4136; }
.cloud-version__rename { display: grid; grid-template-columns: minmax(120px,1fr) auto auto; gap: 4px; }
.cloud-version__rename input { min-width: 0; padding: 7px 8px; border: 1px solid #C99B70; border-radius: 6px; outline: none; font: inherit; }
.cloud-version__detail { grid-column: 2 / -1; display: flex; flex-wrap: wrap; gap: 6px; padding-top: 8px; border-top: 1px dashed #DCC4AA; color: var(--muted); font-size: 10px; }
.cloud-version__detail > span { padding: 4px 7px; border-radius: 999px; background: #F2E7DC; }
.cloud-version__profiles { display: flex; width: 100%; flex-wrap: wrap; gap: 5px; }
.cloud-version__profiles b { padding: 5px 7px; border: 1px solid #E0CCB7; border-radius: 5px; background: #FFF; color: var(--ink); font-size: 10px; }
.cloud-version__profiles small { color: var(--muted); font-family: ui-monospace, Consolas, monospace; }
.cloud-history button:focus-visible, .cloud-version__rename input:focus-visible { outline: 3px solid rgba(184,115,51,.28); outline-offset: 2px; }
.cloud-history__spin { animation: cloud-history-spin 900ms linear infinite; }
@keyframes cloud-history-spin { to { transform: rotate(360deg); } }
@media (max-width: 680px) { .cloud-version { grid-template-columns: 42px minmax(0,1fr); } .cloud-version__actions { grid-column: 2; } .cloud-version__detail { grid-column: 1 / -1; } }
@media (prefers-reduced-motion: reduce) { .cloud-history__spin { animation: none; } }
</style>
