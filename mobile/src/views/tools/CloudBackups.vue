<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listAccountBackups, getAccountBackup, type AccountBackup } from '@/api/accountBackup'
import { gameFlavorLabel } from '../../../../client/src/utils/gameInstallations'
import AccountBackupVersionHistory from '../../../../client/src/components/sync/AccountBackupVersionHistory.vue'
const route = useRoute()
const accounts = ref<AccountBackup[]>([]), selected = ref(0), backup = ref<AccountBackup | null>(null)
const loading = ref(false), error = ref('')
let generation = 0
const profiles = computed(() => {
  try {
    const value = JSON.parse(backup.value?.profiles_data || '{}')
    return Object.entries(value.profiles || value).map(([id, profile]) => ({ id, name: String((profile as { profileName?: string })?.profileName || id) }))
  } catch { return [] }
})
async function load() {
  loading.value = true; error.value = ''
  try {
    accounts.value = await listAccountBackups()
    const preferred = accounts.value.find(item => item.account_id === route.query.account && (item.game_flavor || '') === (route.query.flavor || ''))
    const next = preferred?.id || accounts.value.find(item => item.id === selected.value)?.id || accounts.value[0]?.id || 0
    if (selected.value === next) await loadBackup()
    else selected.value = next
  } catch (e) { error.value = e instanceof Error ? e.message : '读取存档失败'; loading.value = false }
}
async function loadBackup() {
  const token = ++generation
  const account = accounts.value.find(item => item.id === selected.value)
  backup.value = null; error.value = ''; loading.value = true
  try {
    if (account) { const result = await getAccountBackup(account.account_id, account.game_flavor); if (token === generation) backup.value = result }
  } catch (e) { if (token === generation) error.value = e instanceof Error ? e.message : '读取存档失败' }
  finally { if (token === generation) loading.value = false }
}
watch(selected, () => void loadBackup())
onMounted(() => void load())
async function restored(value: AccountBackup) { if (value.id === selected.value) { backup.value = value; const index = accounts.value.findIndex(item => item.id === value.id); if (index >= 0) accounts.value[index] = value } }
</script>
<template>
  <div class="cloud-backups">
    <header><h1>人物卡云端备份</h1><p>手机上查看人物卡和历史版本。需要写回游戏时，在桌面端选择对应版本与子账号同步。</p></header>
    <label v-if="accounts.length" class="account-label">游戏版本与子账号<select v-model="selected" aria-label="云端子账号" :disabled="loading"><option v-for="item in accounts" :key="item.id" :value="item.id">{{ item.account_id }} · {{ gameFlavorLabel(item.game_flavor) }}</option></select></label>
    <p v-if="error" role="alert">{{ error }} <button @click="load">重试</button></p><p v-if="loading" role="status">正在读取存档…</p>
    <section v-else-if="backup" class="backup-content"><div class="backup-summary"><strong>{{ backup.profiles_count }} 张人物卡</strong><span>云端版本 v{{ backup.version }}</span></div><ul><li v-for="profile in profiles" :key="profile.id"><strong>{{ profile.name }}</strong><small>{{ profile.id }}</small></li></ul><RouterLink class="editor-link" to="/tools/trp3">前往在线编辑器 <i class="ri-arrow-right-line" /></RouterLink><AccountBackupVersionHistory :key="backup.id" :account-id="backup.account_id" :game-flavor="backup.game_flavor" @restored="restored" /></section>
    <section v-else-if="!error" class="empty"><h2>还没有云端存档</h2><p>在在线编辑器中添加保存子账号，或通过桌面端备份人物卡。</p><RouterLink to="/tools/trp3">打开在线编辑器</RouterLink></section>
  </div>
</template>
<style scoped>
.cloud-backups{padding:20px 16px 32px;color:var(--color-text-main)}h1{margin:0;font-size:24px}header p,.empty p{font-size:13px;line-height:1.8;color:var(--color-text-secondary)}.account-label{display:grid;gap:8px;margin:20px 0;font-size:13px}select{width:100%;min-width:0;min-height:46px;border:1px solid var(--color-border);border-radius:8px;padding:8px;background:var(--input-bg);color:var(--color-text-main);font-size:16px}.backup-summary{display:flex;flex-wrap:wrap;gap:10px;justify-content:space-between;font-size:13px}ul{list-style:none;padding:0;display:grid;gap:8px}li{display:grid;gap:5px;padding:14px;border:1px solid var(--color-border);border-radius:10px;background:var(--color-panel-bg);overflow-wrap:anywhere}li strong{font-size:14px}small{font-size:11px;color:var(--color-text-secondary)}a{color:var(--color-accent)}.editor-link{display:flex;align-items:center;justify-content:space-between;min-height:44px;margin-bottom:20px;text-decoration:none}.empty{padding:30px 0}button{min-height:44px}
</style>
