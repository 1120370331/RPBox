<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ListRPDBWorksParams, RPDBWorkType } from '@/api/rpdb'
import type { Tag } from '@/api/tag'

const props = defineProps<{ modelValue: ListRPDBWorksParams; styleTags?: Array<Pick<Tag, 'id' | 'name'>> }>()
const emit = defineEmits<{ 'update:modelValue': [value: ListRPDBWorksParams]; search: [] }>()
const tagSuggestionsOpen = ref(false)

function patch(key: keyof ListRPDBWorksParams, value: string | number | undefined) {
  emit('update:modelValue', { ...props.modelValue, [key]: value, page: 1 })
}

const normalizedTagSearch = computed(() => normalizeTopicSearch(props.modelValue.tag_search || ''))
const suggestedTags = computed(() => {
  const term = normalizedTagSearch.value.toLowerCase()
  const tags = props.styleTags || []
  if (!term) return tags.slice(0, 5)
  return tags.filter(tag => tag.name.toLowerCase().includes(term)).slice(0, 5)
})
const showTagSuggestions = computed(() => tagSuggestionsOpen.value && suggestedTags.value.length > 0)

function normalizeTopicSearch(value: string) {
  return value.trim().replace(/^#+/, '').trim()
}

function patchTagSearch(value: string) {
  tagSuggestionsOpen.value = true
  patch('tag_search', normalizeTopicSearch(value) || undefined)
}

function selectTagSuggestion(name: string) {
  patch('tag_search', name)
  tagSuggestionsOpen.value = false
  emit('search')
}

function closeTagSuggestions(event: FocusEvent) {
  const current = event.currentTarget as HTMLElement
  const next = event.relatedTarget as Node | null
  if (next && current.contains(next)) return
  tagSuggestionsOpen.value = false
}

function patchType(value: RPDBWorkType | '') {
  emit('update:modelValue', {
    ...props.modelValue,
    type: value,
    availability_status: '',
    faction: '',
    armor_type: '',
    tag_id: undefined,
    tag_search: undefined,
    page: 1,
  })
}
</script>

<template>
  <div class="filter-bar">
    <div class="search-row" data-testid="rpdb-search-row">
      <label class="search-box">
        <i class="ri-search-2-line"></i>
        <input :value="modelValue.search" placeholder="搜索道具、幻化、家宅、作者或标签" @input="patch('search', ($event.target as HTMLInputElement).value)" @keyup.enter="$emit('search')" />
      </label>
      <button type="button" class="search-trigger" data-testid="rpdb-search-button" title="搜索" @click="$emit('search')"><i class="ri-search-2-line"></i><span>搜索</span></button>
    </div>

    <div class="filter-row" data-testid="rpdb-filter-row">
      <select :value="modelValue.type" aria-label="作品类型" @change="patchType(($event.target as HTMLSelectElement).value as RPDBWorkType | '')">
        <option value="">全部分类</option><option value="item_showcase">魔兽物品</option><option value="transmog">幻化方案</option><option value="home_showcase">家宅分享</option>
      </select>

      <select v-if="modelValue.type !== 'home_showcase'" :value="modelValue.availability_status" aria-label="获取状态" @change="patch('availability_status', ($event.target as HTMLSelectElement).value)">
        <option value="">全部获取状态</option><option value="available">可获取</option><option value="limited">限时获取</option><option value="removed">已绝版</option><option value="unknown">未知</option>
      </select>
      <select v-else :value="modelValue.availability_status" aria-label="参观状态" @change="patch('availability_status', ($event.target as HTMLSelectElement).value)">
        <option value="">全部参观状态</option><option value="available">可参观</option><option value="limited">需预约</option><option value="removed">暂不开放</option><option value="unknown">未知</option>
      </select>

      <select v-if="modelValue.type === 'transmog'" :value="modelValue.armor_type" aria-label="护甲类型" @change="patch('armor_type', ($event.target as HTMLSelectElement).value)">
        <option value="">全部护甲</option><option value="cloth">布甲</option><option value="leather">皮甲</option><option value="mail">锁甲</option><option value="plate">板甲</option><option value="cosmetic">装饰</option>
      </select>

      <select v-if="modelValue.type === 'transmog' || modelValue.type === 'item_showcase'" :value="modelValue.faction" aria-label="阵营" @change="patch('faction', ($event.target as HTMLSelectElement).value)">
        <option value="">全部阵营</option><option value="alliance">联盟</option><option value="horde">部落</option><option value="neutral">不限阵营</option>
      </select>

      <div class="tag-filter" @focusout="closeTagSuggestions">
        <i class="ri-price-tag-3-line"></i>
        <input
          :value="modelValue.tag_search || ''"
          data-testid="rpdb-tag-filter-input"
          placeholder="输入风格标签"
          @click="tagSuggestionsOpen = true"
          @input="patchTagSearch(($event.target as HTMLInputElement).value)"
          @keyup.enter="$emit('search')"
        >
        <div v-if="showTagSuggestions" class="tag-suggestions">
          <button
            v-for="tag in suggestedTags"
            :key="tag.id"
            type="button"
            data-testid="rpdb-tag-filter-suggestion"
            @click.stop="selectTagSuggestion(tag.name)"
          >
            {{ tag.name }}
          </button>
        </div>
      </div>

      <select :value="modelValue.sort" aria-label="排序" @change="patch('sort', ($event.target as HTMLSelectElement).value)">
        <option value="updated_at">最近更新</option><option value="created_at">最新发布</option><option value="popular">热门浏览</option><option value="favorite">收藏最多</option><option value="comments">讨论最多</option>
      </select>
    </div>
  </div>
</template>

<style scoped>
.filter-bar{display:grid;gap:8px}.search-row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:8px}.filter-row{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:8px}.search-box,.tag-filter{display:flex;min-width:0;height:40px;align-items:center;gap:8px;padding-left:12px;border:1px solid var(--input-border);border-radius:var(--radius-sm);background:var(--input-bg);color:var(--color-text-secondary)}.search-box:focus-within,.tag-filter:focus-within{border-color:var(--input-focus);box-shadow:0 0 0 3px rgba(var(--shadow-base),.1)}.search-box input,.tag-filter input{min-width:0;flex:1;border:0!important;outline:0;background:transparent!important;color:var(--color-text-main)!important;font:inherit;box-shadow:none!important}.search-box input::placeholder,.tag-filter input::placeholder{color:var(--input-placeholder)!important}.tag-filter{position:relative;padding-right:10px}.tag-filter>i{color:var(--icon-color)}.tag-suggestions{position:absolute;top:44px;right:0;left:0;z-index:25;display:grid;gap:4px;padding:6px;border:1px solid var(--color-border);border-radius:var(--radius-sm);background:var(--color-panel-bg);box-shadow:var(--shadow-md)}.tag-suggestions button{min-height:28px;border:0;border-radius:var(--radius-sm);background:transparent;color:var(--color-text-main);text-align:left;font:inherit}.tag-suggestions button:hover{background:var(--btn-secondary-bg);color:var(--btn-secondary-text)}select{width:100%;height:40px;padding:0 10px;border:1px solid var(--input-border);border-radius:var(--radius-sm);background:var(--input-bg);color:var(--color-text-main);font:inherit}.search-trigger{display:inline-flex;height:40px;align-items:center;justify-content:center;gap:6px;padding:0 20px;border:1px solid var(--btn-primary-bg);border-radius:var(--radius-sm);background:var(--btn-primary-bg);color:var(--btn-primary-text);font-weight:800;box-shadow:var(--shadow-sm);cursor:pointer}.search-trigger:hover{background:var(--btn-primary-hover);color:var(--btn-primary-text)}.search-trigger i{font-size:16px}
@media(max-width:1050px){.filter-row{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:620px){.filter-row{grid-template-columns:1fr}.search-trigger{padding:0 14px}.search-trigger span{display:none}}
</style>
