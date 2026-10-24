<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { open } from '@tauri-apps/plugin-dialog'
import RDialog from '@/components/RDialog.vue'
import { dialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { safeNavigationPath } from '@/utils/navigationTrail'
import {
  gameFlavors, gameFlavorLabel, getGameInstallations, registerGameInstallation,
  removeGameInstallation, sameGamePath, selectGameInstallation, type GameInstallation,
} from '@/utils/gameInstallations'

interface InspectedDirectory { path: string; flavor: string | null; accounts: string[] }
const route = useRoute()
const toast = useToastStore()
const desktop = '__TAURI_INTERNALS__' in window
const installations = ref(getGameInstallations())
const activePath = ref(localStorage.getItem('wow_path') || '')
const choosing = ref(false)
const error = ref('')
const confirming = ref(false)
const editing = ref(false)
const candidates = ref<InspectedDirectory[]>([])
const draftPath = ref('')
const draftFlavor = ref('')
let request = 0
onBeforeUnmount(() => { request++ })
const candidate = computed(() => candidates.value.find(item => item.path === draftPath.value))
const canSave = computed(() => !!candidate.value && gameFlavors.some(item => item.value === draftFlavor.value))
const returnTo = computed(() => safeNavigationPath(route.query.redirect) ? route.query.redirect : '')
const recognitionHint = computed(() => {
  if (installations.value.some(item => sameGamePath(item.path, draftPath.value) && item.flavor)) return '已保留此目录保存的游戏版本，可修改。'
  return candidate.value?.flavor ? `自动识别为${gameFlavorLabel(candidate.value.flavor)}，可自行修改。` : '未能识别游戏版本，请手动选择。'
})

function refresh() {
  installations.value = getGameInstallations()
  activePath.value = localStorage.getItem('wow_path') || ''
}
function chooseCandidate() {
  const registered = installations.value.find(item => sameGamePath(item.path, draftPath.value))
  draftFlavor.value = registered?.flavor || candidate.value?.flavor || ''
}
async function addDirectory() {
  if (!desktop || choosing.value) return
  const current = ++request
  choosing.value = true
  error.value = ''
  try {
    const path = await open({ directory: true, multiple: false, title: '选择魔兽世界目录' })
    if (!path || Array.isArray(path) || current !== request) return
    const result = await invoke<InspectedDirectory[]>('inspect_game_directories', { path })
    if (current !== request) return
    if (!result.length) {
      error.value = '未找到游戏目录，请选择包含游戏程序或 WTF/Account 的目录。'
      return
    }
    candidates.value = result
    draftPath.value = result.length === 1 ? result[0]!.path : ''
    editing.value = false
    chooseCandidate()
    confirming.value = true
  } catch (reason) {
    if (current === request) error.value = reason instanceof Error ? reason.message : String(reason)
  } finally {
    if (current === request) choosing.value = false
  }
}
function editVersion(item: GameInstallation) {
  candidates.value = [{ path: item.path, flavor: null, accounts: [] }]
  draftPath.value = item.path
  draftFlavor.value = item.flavor
  editing.value = true
  error.value = ''
  confirming.value = true
}
function saveDirectory() {
  if (!canSave.value) return
  try {
    registerGameInstallation({ path: draftPath.value, flavor: draftFlavor.value })
    refresh()
    confirming.value = false
    toast.success(editing.value ? '游戏版本已更新' : '游戏目录已登记')
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : String(reason))
  }
}
function activate(item: GameInstallation) {
  try { selectGameInstallation(item); refresh(); toast.success('已切换游戏目录') }
  catch (reason) { toast.error(reason instanceof Error ? reason.message : String(reason)) }
}
async function remove(item: GameInstallation) {
  if (!(await dialog.confirm({ title: '移除游戏目录', message: `移除 ${item.path} 的登记？\n游戏文件和备份不会删除。`, type: 'warning', confirmText: '移除登记' }))) return
  try { removeGameInstallation(item.path); refresh(); toast.success('目录登记已移除') }
  catch (reason) { toast.error(reason instanceof Error ? reason.message : String(reason)) }
}
</script>

<template>
  <main class="directories-page">
    <header class="directories-heading">
      <div><h1>游戏目录</h1><p>先选择目录，再确认对应的游戏版本。</p></div>
      <button class="directory-primary" type="button" :disabled="!desktop || choosing" @click="addDirectory"><i :class="choosing ? 'ri-loader-4-line' : 'ri-folder-add-line'" aria-hidden="true" />{{ choosing ? '正在识别…' : '添加目录' }}</button>
    </header>
    <p v-if="!desktop" class="directory-notice">请在桌面客户端选择本机游戏目录。</p>
    <p v-if="error" class="directory-error" role="alert">{{ error }}</p>
    <div v-if="installations.length" class="directories-list">
      <article v-for="item in installations" :key="item.path" class="directory-entry" :class="{ current: sameGamePath(activePath, item.path) }">
        <div class="directory-details">
          <div class="directory-version"><i class="ri-folder-3-line" aria-hidden="true" /><strong>{{ item.flavor ? gameFlavorLabel(item.flavor) : '待确认游戏版本' }}</strong><span v-if="sameGamePath(activePath, item.path)" class="current-label">当前使用</span></div>
          <code>{{ item.path }}</code>
        </div>
        <div class="directory-entry-actions">
          <button v-if="!sameGamePath(activePath, item.path) && item.flavor" type="button" @click="activate(item)">设为当前</button>
          <button type="button" @click="editVersion(item)">修改版本</button>
          <button class="directory-remove" type="button" @click="remove(item)">移除</button>
        </div>
      </article>
    </div>
    <section v-else class="directories-empty"><i class="ri-folder-open-line" aria-hidden="true" /><h2>还没有登记游戏目录</h2><p>添加目录后，备份和插件安装会使用这里保存的游戏版本。</p></section>
    <footer class="directories-footer">
      <RouterLink v-if="returnTo" :to="returnTo">完成并返回 <i class="ri-arrow-right-line" aria-hidden="true" /></RouterLink>
      <RouterLink v-else to="/tools/rpbox/install">前往插件安装 <i class="ri-arrow-right-line" aria-hidden="true" /></RouterLink>
    </footer>

    <RDialog v-model="confirming" :title="editing ? '修改游戏版本' : '确认游戏目录'" :confirm-text="editing ? '保存修改' : '确认登记'" :confirm-disabled="!canSave" width="560px" @confirm="saveDirectory">
      <div class="directory-confirmation">
        <label v-if="candidates.length > 1">游戏目录<select v-model="draftPath" aria-label="确认游戏目录" @change="chooseCandidate"><option value="" disabled>请选择具体目录</option><option v-for="item in candidates" :key="item.path" :value="item.path">{{ item.path }}</option></select></label>
        <div v-else class="confirmed-path"><span>游戏路径</span><code>{{ draftPath }}</code></div>
        <label>游戏版本<select v-model="draftFlavor" aria-label="游戏版本" :disabled="!draftPath"><option value="" disabled>请选择游戏版本</option><option v-for="item in gameFlavors" :key="item.value" :value="item.value">{{ item.label }}</option></select></label>
        <p v-if="draftPath && !editing">{{ recognitionHint }}</p>
      </div>
    </RDialog>
  </main>
</template>

<style scoped>
.directories-page { max-width: 1080px; margin: 0 auto; padding: 28px 20px 48px; color: var(--color-text-main); }
.directories-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 18px; margin-bottom: 24px; }
.directories-heading h1 { margin: 0 0 8px; font-size: 28px; }
.directories-heading p, .directories-empty p { margin: 0; color: var(--color-text-secondary); font-size: 14px; line-height: 1.7; }
.directories-page button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 40px; padding: 8px 14px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-panel-bg); color: var(--color-text-main); font: inherit; font-size: 13px; cursor: pointer; }
.directories-page button:disabled { opacity: .55; cursor: not-allowed; }
.directories-page .directory-primary { background: var(--btn-primary-bg); border-color: var(--btn-primary-bg); color: var(--btn-primary-text); }
.directories-list { display: grid; gap: 12px; }
.directory-entry { display: flex; flex-wrap: wrap; align-items: center; gap: 20px; padding: 20px; border: 1px solid var(--color-border); border-left: 4px solid var(--color-border); border-radius: 12px; background: var(--color-panel-bg); }
.directory-entry.current { border-left-color: var(--color-accent); }
.directory-details { flex: 1 1 320px; min-width: 0; }
.directory-version { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 10px; font-size: 15px; }
.directory-version > i { color: var(--color-accent); font-size: 22px; }
.current-label { font-size: 12px; color: var(--color-accent); }
.directory-details code, .confirmed-path code { display: block; font-family: Consolas, monospace; font-size: 13px; line-height: 1.8; overflow-wrap: anywhere; color: var(--color-text-secondary); }
.directory-entry-actions { display: flex; flex-wrap: wrap; gap: 8px; }
.directories-page .directory-remove { border-color: transparent; color: var(--btn-danger-bg); }
.directories-empty { text-align: center; padding: 56px 20px; border: 1px dashed var(--color-border); border-radius: 12px; }
.directories-empty > i { font-size: 40px; color: var(--color-accent); }
.directories-empty h2 { font-size: 18px; margin: 16px 0 8px; }
.directories-footer { display: flex; justify-content: flex-end; margin-top: 20px; }
.directories-footer a { color: var(--link-color); text-decoration: none; min-height: 40px; display: inline-flex; align-items: center; gap: 8px; }
.directory-notice { color: var(--color-text-secondary); font-size: 13px; }
.directory-error { color: var(--btn-danger-bg); font-size: 13px; }
.directory-confirmation { display: grid; gap: 18px; white-space: normal; }
.directory-confirmation label, .confirmed-path { display: grid; gap: 8px; min-width: 0; font-size: 14px; }
.directory-confirmation select { width: 100%; min-width: 0; min-height: 44px; padding: 8px 10px; border: 1px solid var(--input-border); border-radius: 8px; background: var(--input-bg); color: var(--input-text); font: inherit; }
.directory-confirmation p { margin: 0; font-size: 13px; color: var(--color-text-secondary); line-height: 1.6; }
.directories-page :is(button,a):focus-visible, .directory-confirmation select:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 3px; }
@media(max-width:600px) { .directories-page { padding: 20px 12px; } .directory-entry { padding: 16px; gap: 14px; } .directory-entry-actions { width: 100%; } .directories-page button { min-height: 44px; } }
</style>
