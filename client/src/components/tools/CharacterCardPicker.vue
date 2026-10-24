<script setup lang="ts">
import { configuredGameFlavor } from '../../utils/gameInstallations'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import RModal from '../RModal.vue'
import { getCharacterCardSources, type CharacterCardSource, type CharacterCardSummary } from '@/api/characterCard'
import { buildCharacterChoices, matchesCharacterKeywords, type CharacterCardChoice, type LocalCharacterSource } from '../../utils/characterCardSelection'

const props = defineProps<{ modelValue: boolean; cards: CharacterCardSummary[]; busy?: boolean; currentCardId?: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; select: [choice: CharacterCardChoice] }>()
const query = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const cloud = ref<CharacterCardSource[]>([]), local = ref<LocalCharacterSource[]>([])
const cloudLoading = ref(false), localLoading = ref(false), cloudError = ref(''), localError = ref('')
const desktop = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
const hasWowPath = ref(false)
let generation = 0
const page = ref(1), pageSize = 30
const choices = computed(() => buildCharacterChoices(props.cards, cloud.value, local.value))
const filtered = computed(() => choices.value.filter(choice => matchesCharacterKeywords(choice.searchText, query.value)))
const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))
const visible = computed(() => filtered.value.slice((page.value - 1) * pageSize, page.value * pageSize))
const loading = computed(() => cloudLoading.value || localLoading.value)
watch(query, () => { page.value = 1 })
watch(totalPages, value => { page.value = Math.min(page.value, value) })

interface LocalScan { accounts: Array<{ account_id: string; profiles: Array<{ id: string; name: string; raw_lua: string }> }> }
async function loadCloud(token: number) {
  cloudLoading.value = true; cloudError.value = ''
  try { const response = await getCharacterCardSources(); if (token === generation) cloud.value = response.sources }
  catch (error) { if (token === generation) cloudError.value = error instanceof Error ? error.message : '云端来源加载失败' }
  finally { if (token === generation) cloudLoading.value = false }
}
async function loadLocal(token: number) {
  const wowPath = localStorage.getItem('wow_path')?.trim() || ''
  hasWowPath.value = !!wowPath
  if (!desktop || !wowPath) return
  localLoading.value = true; localError.value = ''
  try {
    const result = await invoke<LocalScan>('scan_profiles', { wowPath })
    const entries: LocalCharacterSource[] = []
    let invalid = 0
    for (const account of result.accounts || []) for (const profile of account.profiles || []) {
      try {
        const data: unknown = JSON.parse(profile.raw_lua)
        if (!data || typeof data !== 'object' || Array.isArray(data)) throw new Error('invalid profile')
        entries.push({ game_flavor: configuredGameFlavor(wowPath), account_id: account.account_id, profile_id: profile.id, profile_name: profile.name, profile: data as Record<string, unknown> })
      } catch { invalid++ }
    }
    if (token !== generation) return
    local.value = entries
    if (invalid) localError.value = `${invalid} 张本地人物卡数据无法解析，已保留其他可用结果。`
  } catch (error) { if (token === generation) localError.value = error instanceof Error ? error.message : String(error) }
  finally { if (token === generation) localLoading.value = false }
}
async function load() {
  const token = ++generation
  cloud.value = []; local.value = []; cloudLoading.value = false; localLoading.value = false; cloudError.value = ''; localError.value = ''
  await Promise.allSettled([loadCloud(token), loadLocal(token)])
}
watch(() => props.modelValue, async open => {
  if (!open) { generation++; return }
  query.value = ''; page.value = 1
  void load()
  await nextTick(); searchInput.value?.focus()
}, { immediate: true })
onBeforeUnmount(() => { generation++ })
</script>

<template>
  <RModal :model-value="modelValue" title="搜索 / 选择人物卡" width="760px" :closable="!busy" :mask-closable="!busy" @update:model-value="emit('update:modelValue', $event)">
    <div class="character-picker">
      <div class="character-picker__filters"><label class="character-picker__search"><i class="ri-search-line" aria-hidden="true"></i><input ref="searchInput" v-model="query" type="search" aria-label="人物卡名称" placeholder="按人物卡名称搜索…" /></label></div>
      <p class="character-picker__count" role="status">{{ filtered.length }} 张人物卡<span v-if="loading"> · 正在加载人物卡…</span></p>
      <p v-if="!desktop" class="character-picker__note">本地游戏人物卡需要在 RPBox 桌面客户端读取。已导入的本地档案仍可在这里选择。</p>
      <p v-else-if="!hasWowPath" class="character-picker__note">请先设置游戏目录，再读取本地人物卡。<RouterLink to="/tools/rpbox/install">设置游戏目录</RouterLink></p>
      <p v-if="cloudError" class="character-picker__error" role="alert">云端：{{ cloudError }} <button type="button" @click="loadCloud(generation)">重试</button></p>
      <p v-if="localError" class="character-picker__error" role="alert">本地：{{ localError }} <button type="button" @click="loadLocal(generation)">重新读取</button></p>
      <div class="character-picker__list">
        <button v-for="choice in visible" :key="choice.key" type="button" class="character-choice" :class="{ current: choice.kind === 'archive' && choice.card.id === currentCardId }" :disabled="busy" @click="emit('select', choice)">
          <span class="character-choice__copy"><strong>{{ choice.name }}</strong><small>{{ choice.detail }}</small></span><span class="character-choice__action">{{ choice.kind === 'archive' ? choice.card.id === currentCardId ? '当前档案' : '编辑原档案' : '导入并编辑' }}<i class="ri-arrow-right-s-line" aria-hidden="true"></i></span>
        </button>
        <div v-if="!filtered.length && !loading" class="character-picker__empty"><i class="ri-search-2-line" aria-hidden="true"></i><strong>没有匹配的人物卡</strong><span>试试完整名称，或输入名称中的一部分。</span><button v-if="query" type="button" @click="query = ''">清除筛选</button></div>
      </div>
      <nav v-if="totalPages > 1" class="character-picker__pages" aria-label="人物卡搜索分页"><button type="button" :disabled="page === 1" @click="page--">上一页</button><span>{{ page }} / {{ totalPages }}</span><button type="button" :disabled="page === totalPages" @click="page++">下一页</button></nav>
      <p class="character-picker__note">选择已有人物卡继续编辑，或导入选中的人物卡。</p>
    </div>
  </RModal>
</template>

<style scoped>
.character-picker{color:var(--color-text-main)}.character-picker__filters{display:flex;align-items:center;gap:14px}.character-picker__search{display:flex;align-items:center;gap:8px;flex:1;min-width:0;height:40px;padding:0 12px;border:1px solid var(--input-border);border-radius:8px;background:var(--input-bg)}.character-picker__search input{min-width:0;flex:1;border:0;outline:none;background:transparent;color:var(--input-text);font:inherit;font-size:13px}.character-picker__count{margin:16px 0 12px;color:var(--color-text-secondary);font-size:12px}.character-picker__list{display:grid;gap:8px;max-height:400px;overflow:auto;padding:2px}.character-choice{display:flex;align-items:center;gap:12px;width:100%;padding:14px;text-align:left;border:1px solid var(--color-border);border-radius:10px;background:var(--color-card-bg);color:var(--color-text-main);cursor:pointer}.character-choice:hover,.character-choice.current{border-color:var(--color-accent);background:var(--color-card-bg-hover)}.character-choice:disabled{opacity:.5;cursor:default}.character-choice__copy{display:grid;flex:1;min-width:0;gap:6px}.character-choice__copy strong{font-size:14px;overflow-wrap:anywhere}.character-choice__copy small{font-size:11px;color:var(--color-text-secondary);overflow-wrap:anywhere}.character-choice__action{font-size:11px;color:var(--color-accent);white-space:nowrap}.character-picker__note{font-size:11px;color:var(--color-text-secondary);line-height:1.8;margin:14px 0 0}.character-picker__note a{margin-left:6px;color:var(--link-color)}.character-picker__error{font-size:12px;color:var(--color-danger)}.character-picker__error button{border:0;background:transparent;color:var(--link-color);cursor:pointer}.character-picker__empty{display:grid;justify-items:center;gap:12px;padding:45px 16px;color:var(--color-text-secondary);font-size:12px}.character-picker__empty>i{font-size:32px}.character-picker__empty strong{color:var(--color-text-main);font-size:15px}.character-picker__empty button,.character-picker__pages button{border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);padding:7px 12px;cursor:pointer}.character-picker__pages{display:flex;justify-content:center;align-items:center;gap:20px;margin-top:16px;font-size:12px}.character-picker__pages button:disabled{opacity:.4;cursor:default}.character-picker button:focus-visible,.character-picker__search:focus-within{outline:2px solid var(--color-accent);outline-offset:2px}@media(max-width:650px){.character-picker__filters{align-items:stretch;flex-direction:column}.character-picker__search{flex:auto}.character-choice{gap:8px;flex-wrap:wrap}.character-choice__copy{flex-basis:60%}}
</style>

<style scoped>

@media(max-width:600px){.character-picker__search{height:46px}.character-picker__search input{font-size:16px;min-height:44px}.character-picker__list{max-height:48dvh}.character-choice{min-height:64px;padding:12px;align-items:flex-start}.character-choice__copy{flex-basis:100%}.character-choice__action{margin-left:auto}.character-picker__pages{gap:12px}.character-picker__pages button,.character-picker__empty button{min-height:44px}}

</style>
