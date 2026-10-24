<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import RModal from '../RModal.vue'
import WowIcon from '../WowIcon.vue'
import { normalizeTRP3Icon } from '../../utils/trp3Markup'
import { useIconPageImages } from '../../composables/useIconPageImages'
const props = defineProps<{ modelValue: boolean; selected?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; select: [value: string] }>()
const icons = ref<Array<{ id: number; name: string }>>([])
const search = ref(''), category = ref(''), custom = ref(''), error = ref('')
const loading = ref(false), page = ref(1)
const pageSize = 80
const aliases: Record<string, string> = { 剑: 'sword', 书: 'book', 法术: 'spell', 盾: 'shield', 药水: 'potion', 猫: 'cat', 骷髅: 'skull', 花: 'flower', 地图: 'map', 头盔: 'helmet', 卷轴: 'scroll', 羽毛: 'feather', 德鲁伊: 'druid', 法师: 'mage', 牧师: 'priest', 战士: 'warrior', 盗贼: 'rogue', 圣骑士: 'paladin', 猎人: 'hunter', 萨满: 'shaman', 术士: 'warlock' }
const filtered = computed(() => {
  const query = (aliases[search.value.trim()] || search.value).trim().toLowerCase()
  return icons.value.filter(icon => (!category.value || icon.name.startsWith(category.value)) && (!query || icon.name.includes(query) || String(icon.id) === query))
})
const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))
const visible = computed(() => filtered.value.slice((page.value - 1) * pageSize, page.value * pageSize))
const visibleNames = computed(() => props.modelValue ? visible.value.map(icon => icon.name) : [])
const upcomingNames = computed(() => props.modelValue ? filtered.value.slice(page.value * pageSize, (page.value + 1) * pageSize).map(icon => icon.name) : [])
const { images, failed, retry } = useIconPageImages(visibleNames, upcomingNames)
const loadedCount = computed(() => visibleNames.value.filter(name => images.has(name)).length)
const hasFailed = computed(() => visibleNames.value.some(name => failed.has(name)))
watch([search, category], () => { page.value = 1 })
watch(() => props.modelValue, async open => {
  if (!open) return
  custom.value = props.selected || ''; error.value = ''; search.value = ''; category.value = ''; page.value = 1
  if (icons.value.length) return
  loading.value = true
  try {
    const data = (await import('../../data/trp3Icons.json')).default
    icons.value = data.map(row => ({ id: Number(row[0]), name: String(row[1]).toLowerCase() }))
  } catch { error.value = '图标目录加载失败，请关闭后重新打开。' } finally { loading.value = false }
})
function select(name: string) { emit('select', name); emit('update:modelValue', false) }
function applyCustom() {
  const value = normalizeTRP3Icon(custom.value)
  const match = icons.value.find(icon => icon.name === value || String(icon.id) === value)
  if (!match) { error.value = '未找到这个图标，请输入图标名称、Interface\\Icons 路径或 File ID。'; return }
  select(match.name)
}
</script>
<template>
  <RModal :model-value="modelValue" title="图标浏览器" width="760px" @update:model-value="emit('update:modelValue', $event)">
    <div class="trp3-icon-picker">
      <div class="icon-search"><input v-model="search" type="search" aria-label="搜索魔兽图标" placeholder="搜索图标名称、File ID，或剑、书、地图…" /><select v-model="category" aria-label="图标分类"><option value="">全部图标</option><option value="inv_">物品</option><option value="spell_">法术</option><option value="ability_">技能</option><option value="achievement_">成就</option><option value="racial_">种族</option></select></div>
      <p class="icon-count" role="status">{{ loading ? '加载图标目录…' : `${filtered.length.toLocaleString()} 个图标` }}<span v-if="!loading && loadedCount < visible.length"> · 已加载 {{ loadedCount }} / {{ visible.length }}</span><button v-if="hasFailed" type="button" @click="retry">重试未加载图标</button></p>
      <div class="icon-grid"><button v-for="icon in visible" :key="icon.id" type="button" :title="`${icon.name} · ${icon.id}`" :aria-label="icon.name" :class="{ selected: selected?.toLowerCase() === icon.name }" @click="select(icon.name)"><img v-if="images.has(icon.name)" :src="images.get(icon.name)" :alt="icon.name" width="42" height="42" decoding="async" /><span v-else class="icon-placeholder" :class="{ failed: failed.has(icon.name) }" aria-hidden="true">{{ failed.has(icon.name) ? '?' : '' }}</span></button></div>
      <p v-if="!loading && !filtered.length" class="icon-empty">没有匹配的图标，试试英文名称或其他分类。</p>
      <nav class="icon-pagination" aria-label="图标分页"><button type="button" :disabled="page === 1" @click="page--">上一页</button><span>{{ page }} / {{ pages }}</span><button type="button" :disabled="page === pages" @click="page++">下一页</button></nav>
      <form class="icon-custom" @submit.prevent="applyCustom"><WowIcon :icon="normalizeTRP3Icon(custom)" :size="36" /><input v-model="custom" aria-label="图标名称或路径" placeholder="Interface\Icons\INV_Misc_Book_09" /><button type="submit" :disabled="loading">使用图标</button></form>
      <p v-if="error" class="icon-error" role="alert">{{ error }}</p>
    </div>
  </RModal>
</template>
<style scoped>
.icon-grid img{display:block;object-fit:cover;border-radius:2px}.icon-placeholder{display:block;width:42px;height:42px;border-radius:2px;background:linear-gradient(120deg,#24221e,#484033,#24221e);animation:icon-pulse 1.2s ease-in-out infinite alternate}.icon-placeholder.failed{display:grid;place-items:center;animation:none;color:#d4c7b0;font-size:20px}.icon-count button{margin-left:12px;border:0;background:transparent;color:var(--link-color);cursor:pointer;font-size:11px}@keyframes icon-pulse{to{opacity:.45}}@media(prefers-reduced-motion:reduce){.icon-placeholder{animation:none}}
.trp3-icon-picker{padding:4px;color:var(--color-text-main)}.icon-search{display:flex;gap:10px}.icon-search input,.icon-custom input{flex:1;min-width:0}.icon-search :is(input,select),.icon-custom input{height:38px;padding:0 10px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--input-text)}.icon-count{font-size:12px;color:var(--color-text-secondary);margin:14px 0}.icon-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(48px,1fr));gap:6px;min-height:190px;max-height:380px;overflow-y:auto;padding:3px}.icon-grid button{display:grid;place-items:center;padding:3px;border:1px solid #5b5140;border-radius:4px;background:#151515;cursor:pointer;min-height:48px}.icon-grid button:hover,.icon-grid button.selected{outline:2px solid #e8bd57;outline-offset:0}.icon-pagination{display:flex;justify-content:center;align-items:center;gap:18px;margin:16px 0;font-size:12px}.icon-pagination button,.icon-custom button{padding:7px 12px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);cursor:pointer}.icon-pagination button:disabled{opacity:.4;cursor:default}.icon-custom{display:flex;align-items:center;gap:10px;border-top:1px solid var(--color-border);padding-top:16px}.icon-error{color:var(--color-danger);font-size:12px}.icon-empty{text-align:center;color:var(--color-text-secondary)}button:focus-visible{outline:3px solid var(--color-accent)}
</style>

<style scoped>

@media(max-width:600px){.icon-search{flex-direction:column;gap:8px}.icon-search :is(input,select),.icon-custom input{min-height:44px;font-size:16px}.icon-grid{max-height:42dvh}.icon-custom{flex-wrap:wrap}.icon-custom input{flex-basis:calc(100% - 60px)}.icon-custom button{width:100%;min-height:44px}.icon-pagination{gap:12px}.icon-pagination button{min-height:44px}.icon-count{line-height:1.8}}

</style>
