<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { ensureAccountBackup, listAccountBackups, type AccountBackup } from '@/api/accountBackup'
import { getCharacterCardTargets, getCharacterCardSources, type CharacterCard, type CharacterCardSource, type CharacterCardSaveTarget } from '@/api/characterCard'
import { gameFlavors, gameFlavorLabel, getGameInstallations } from '../../utils/gameInstallations'

const props = defineProps<{ card: CharacterCard; busy: boolean }>()
const emit = defineEmits<{ change: [targets: CharacterCardSaveTarget[]]; save: [] }>()
const accounts = ref<AccountBackup[]>([]), sources = ref<CharacterCardSource[]>([])
const selected = ref<number[]>([]), profiles = ref<Record<number, string>>({})
const loading = ref(false), error = ref(''), adding = ref(false)
const accountName = ref(''), flavor = ref('_retail_')
const desktop = '__TAURI_INTERNALS__' in window
let generation = 0
let initializedCardId: number | null = null
const selection = computed(() => loading.value || adding.value ? [] : selected.value.flatMap(id => {
  const account = accounts.value.find(item => item.id === id)
  return account ? [{ backup_id: id, profile_id: profiles.value[id] || `rpbox-${props.card.id}`, expected_version: account.version }] : []
}))
watch(selection, value => emit('change', value), { deep: true })
watch(() => props.card.id, () => void refresh(false), { immediate: true })
async function refresh(keepSelection = true) {
  const token = ++generation
  const preserveSelection = keepSelection && initializedCardId === props.card.id
  const previous = preserveSelection ? [...selected.value] : []
  const previousProfiles = preserveSelection ? { ...profiles.value } : {}
  loading.value = true; selected.value = []; error.value = ''; emit('change', [])
  try {
    const [backups, bindings, cards] = await Promise.all([listAccountBackups(), getCharacterCardTargets(props.card.id), getCharacterCardSources()])
    if (token !== generation) return
    accounts.value = backups; sources.value = cards.sources
    const boundProfiles = new Map(bindings.targets.filter(item => item.profile_id?.trim()).map(item => [item.backup_id, item.profile_id]))
    // A backup ID identifies the versioned account exactly. Local provenance may
    // only resolve by both account and edition; never guess from its name alone.
    const sourceBackup = props.card.source_backup_id
      ? backups.find(backup => backup.id === props.card.source_backup_id)
      : props.card.source_account_id && props.card.source_game_flavor
        ? backups.find(backup => backup.account_id === props.card.source_account_id && backup.game_flavor === props.card.source_game_flavor)
        : undefined
    if (sourceBackup && props.card.source_profile_id?.trim() && !boundProfiles.has(sourceBackup.id)) {
      boundProfiles.set(sourceBackup.id, props.card.source_profile_id)
    }
    profiles.value = Object.fromEntries(backups.map(backup => [backup.id,
      previousProfiles[backup.id] || boundProfiles.get(backup.id) || `rpbox-${props.card.id}`]))
    selected.value = (preserveSelection ? previous : [...boundProfiles.keys()]).filter(id => backups.some(item => item.id === id))
    initializedCardId = props.card.id
    emit('change', selection.value)
  } catch (e) { if (token === generation) { selected.value = []; error.value = e instanceof Error ? e.message : '加载保存目标失败' } }
  finally { if (token === generation) loading.value = false }
}
async function addAccount() {
  if (!accountName.value.trim() || adding.value) return
  adding.value = true; error.value = ''
  try {
    const account = await ensureAccountBackup(accountName.value.trim(), flavor.value)
    await refresh(true)
    if (accounts.value.some(item => item.id === account.id)) selected.value = [...new Set([...selected.value, account.id])]
    accountName.value = ''
  } catch (e) { error.value = e instanceof Error ? e.message : '添加子账号失败' }
  finally { adding.value = false }
}
async function scanLocalAccounts() {
  const installations = getGameInstallations()
  if (!installations.length) { error.value = '请先在插件管理中设置至少一个游戏目录。'; return }
  adding.value = true; error.value = ''
  const failures: string[] = []
  for (const installation of installations) {
    try {
      if (!installation.flavor) throw new Error('目录版本未识别，请在插件管理中重新添加此目录')
      const scan = await invoke<{ accounts: Array<{ account_id: string }> }>('scan_profiles', { wowPath: installation.path })
      for (const account of scan.accounts) await ensureAccountBackup(account.account_id, installation.flavor)
    } catch (e) { failures.push(`${installation.path}：${e instanceof Error ? e.message : String(e)}`) }
  }
  await refresh(true)
  if (failures.length) error.value = ['部分目录未能读取；成功识别的子账号已保留。', ...failures].join(' ')
  adding.value = false
}
defineExpose({ refresh })
</script>

<template>
  <section class="save-targets" aria-label="云端保存目标" :aria-busy="loading">
    <header><div><h2>保存到子账号</h2><p>选择一个或多个子账号。保存会更新对应云端人物卡，并保留存档历史。</p></div><button v-if="desktop" type="button" :disabled="busy || loading || adding" @click="scanLocalAccounts">识别本机子账号</button><button type="button" :disabled="busy || loading || adding" @click="refresh()">刷新目标</button></header>
    <p v-if="error" role="alert" class="target-error">{{ error }}</p>
    <p v-if="loading" role="status">正在读取子账号…</p>
    <fieldset :disabled="busy || loading || adding">
      <div v-if="accounts.length" class="target-list">
        <div v-for="account in accounts" :key="account.id" class="target-row" :class="{ selected: selected.includes(account.id) }">
          <label><input v-model="selected" type="checkbox" :value="account.id" :aria-label="`${gameFlavorLabel(account.game_flavor)} ${account.account_id}`" /><strong>{{ account.account_id }}</strong><span class="edition">{{ gameFlavorLabel(account.game_flavor) }}</span></label>
          <select v-model="profiles[account.id]" :disabled="!selected.includes(account.id)" :aria-label="`${gameFlavorLabel(account.game_flavor)} ${account.account_id} 保存人物卡`">
            <option :value="`rpbox-${card.id}`">独立人物卡 · {{ card.display_name || card.first_name || '未命名' }}</option>
            <option v-if="profiles[account.id] !== `rpbox-${card.id}` && !sources.some(item => item.backup_id === account.id && item.profile_id === profiles[account.id])" :value="profiles[account.id]">已关联人物卡 · {{ profiles[account.id] }}</option>
            <option v-for="source in sources.filter(item => item.backup_id === account.id && item.profile_id !== `rpbox-${card.id}`)" :key="source.profile_id" :value="source.profile_id">{{ source.display_name || source.profile_name }} · {{ source.profile_id }}</option>
          </select>
        </div>
      </div>
      <p v-else-if="!loading">还没有云端子账号。添加子账号，或前往人物卡备份扫描本机账号。</p>
      <details><summary>添加保存子账号</summary><div class="add-target"><input v-model="accountName" aria-label="子账号目录名" placeholder="WTF/Account 下的子账号目录名" maxlength="32" /><select v-model="flavor" aria-label="子账号游戏版本"><option v-for="item in gameFlavors" :key="item.value" :value="item.value">{{ item.label }}</option></select><button type="button" :disabled="!accountName.trim()" @click="addAccount">添加</button></div><p>在其他设备上，可以先建立云端子账号；本地写回时会核对游戏版本与账号目录。</p></details>
    </fieldset>
    <footer><span>{{ selected.length ? `已选择 ${selected.length} 个子账号` : '请选择至少一个保存子账号' }} · 此处保存不会写入本地游戏文件</span><button class="mobile-target-save" type="button" :disabled="busy || loading || adding || !selection.length" @click="emit('save')">{{ busy ? '保存中…' : '保存到所选云端存档' }}</button><RouterLink :to="{ path: '/sync', query: { mode: 'cloud', flavor: accounts.find(item => item.id === selected[0])?.game_flavor || '', account: accounts.find(item => item.id === selected[0])?.account_id || '' } }">前往人物卡备份 / 同步到本地 <i class="ri-arrow-right-line" /></RouterLink></footer>
  </section>
</template>

<style scoped>
.save-targets{margin:18px 0;padding:22px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-card-bg);color:var(--color-text-main)}header{display:flex;align-items:start;justify-content:space-between;gap:16px}h2{margin:0;font-size:17px}p,footer{font-size:12px;line-height:1.7;color:var(--color-text-secondary)}fieldset{border:0;padding:0;margin:12px 0}.target-list{display:grid;gap:8px;max-height:300px;overflow:auto}.target-row{display:flex;gap:16px;align-items:center;justify-content:space-between;padding:12px;border:1px solid var(--color-border);border-radius:8px}.target-row.selected{border-color:var(--color-accent)}label{display:flex;align-items:center;gap:10px;flex-wrap:wrap}.edition{font-size:11px;color:var(--color-text-secondary);border:1px solid var(--color-border);border-radius:4px;padding:2px 6px}select,input:not([type=checkbox]),button{background:var(--input-bg);color:var(--color-text-main);border:1px solid var(--input-border);border-radius:6px;padding:8px;max-width:100%;font:inherit;font-size:12px}button,summary{cursor:pointer}button:disabled{opacity:.5;cursor:default}select{max-width:48%}footer{display:flex;flex-wrap:wrap;justify-content:space-between;gap:10px}a{color:var(--color-accent)}details{margin-top:16px;font-size:12px}.add-target{display:flex;flex-wrap:wrap;gap:10px;margin-top:12px}.add-target input{flex:1;min-width:210px}.target-error{color:var(--color-danger)}@media(max-width:650px){.target-row{align-items:stretch;flex-direction:column}.target-row select{max-width:100%}}
</style>

<style scoped>

@media(max-width:600px){.save-targets{padding:16px 12px;margin:16px 0}header{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}header>div{grid-column:1/-1}header>button{min-height:44px}.target-list{max-height:none;overflow:visible}.target-row{padding:12px;gap:12px}.target-row label{min-width:0;line-height:1.6}.target-row strong{overflow-wrap:anywhere;min-width:0}.target-row input[type=checkbox]{width:22px;height:22px;flex-shrink:0}.target-row select{width:100%;min-width:0;min-height:44px;font-size:16px}summary{min-height:44px;align-content:center}.add-target{display:grid;grid-template-columns:minmax(0,1fr)}.add-target input,.add-target select{min-width:0;max-width:100%;width:100%;min-height:44px;font-size:16px}.add-target button{min-height:44px}footer{line-height:1.8}footer>a{min-height:44px;display:flex;align-items:center;gap:6px;flex-wrap:wrap}}

</style>

<style scoped>
.mobile-target-save{display:none}@media(max-width:600px){.mobile-target-save{display:flex;align-items:center;justify-content:center;width:100%;min-height:48px;background:var(--btn-primary-bg);color:var(--btn-primary-text);font-size:15px;font-weight:600;border:0}}
</style>
