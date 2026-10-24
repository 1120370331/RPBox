<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { filterRPTools, rpTools } from '../../data/rpTools'
const route = useRoute(), router = useRouter()
const tools = rpTools.filter(tool => !tool.desktopOnly || (typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window))
function updateFilters(values: Record<string, string | string[] | undefined>) {
  void router.replace({ path: route.path, query: { ...route.query, ...values } })
}
const search = computed({ get: () => typeof route.query.q === 'string' ? route.query.q : '', set: value => updateFilters({ q: value || undefined }) })
const feature = computed({ get: () => typeof route.query.feature === 'string' ? route.query.feature : '', set: value => updateFilters({ feature: value || undefined }) })
const tags = computed(() => [route.query.tag].flat().filter((value): value is string => typeof value === 'string'))
const features = [...new Set(tools.flatMap(tool => tool.features))]
const availableTags = [...new Set(tools.flatMap(tool => tool.tags))]
const filtered = computed(() => filterRPTools(tools, search.value, feature.value, tags.value))
function toggleTag(tag: string) { updateFilters({ tag: tags.value.includes(tag) ? tags.value.filter(value => value !== tag) : [...tags.value, tag] }) }
function reset() { updateFilters({ q: undefined, feature: undefined, tag: undefined }) }
</script>

<template>
  <main class="toolbox-page">
    <header class="toolbox-heading"><span>为角色与故事准备</span><h1>工具箱</h1><p>发现创作、记录与扮演过程中用得上的 RP 工具。</p></header>
    <section class="toolbox-filters" aria-label="筛选 RP 工具">
      <div class="toolbox-search-row">
        <label class="toolbox-search"><i class="ri-search-line" aria-hidden="true"></i><input v-model="search" type="search" aria-label="搜索 RP 工具" placeholder="搜索工具、功能或标签…" /></label>
        <label class="toolbox-feature"><span>功能</span><select v-model="feature" aria-label="筛选功能"><option value="">全部功能</option><option v-for="value in features" :key="value">{{ value }}</option></select></label>
      </div>
      <div class="toolbox-tags" role="group" aria-label="筛选标签"><span>标签</span><button v-for="tag in availableTags" :key="tag" type="button" :aria-pressed="tags.includes(tag)" :class="{ selected: tags.includes(tag) }" @click="toggleTag(tag)">{{ tag }}</button><button v-if="search || feature || tags.length" class="reset" type="button" @click="reset">清除筛选</button></div>
    </section>
    <p class="toolbox-count" role="status">{{ filtered.length }} 个工具</p>
    <div v-if="filtered.length" class="toolbox-grid">
      <RouterLink v-for="tool in filtered" :key="tool.id" :to="tool.route" class="toolbox-card">
        <div class="toolbox-card__top"><span class="toolbox-card__icon"><i :class="tool.icon" aria-hidden="true"></i></span><span class="toolbox-card__open">打开工具 <i class="ri-arrow-right-up-line" aria-hidden="true"></i></span></div>
        <h2>{{ tool.name }}</h2><strong>{{ tool.subtitle }}</strong><p>{{ tool.description }}</p>
        <div class="toolbox-card__features"><span v-for="value in tool.features" :key="value"><i class="ri-check-line" aria-hidden="true"></i>{{ value }}</span></div>
        <footer><span v-for="tag in tool.tags" :key="tag">{{ tag }}</span></footer>
      </RouterLink>
    </div>
    <div v-else class="toolbox-empty"><i class="ri-search-2-line" aria-hidden="true"></i><h2>没有找到匹配的工具</h2><p>试试其他关键词，或减少功能和标签限制。</p><button type="button" @click="reset">查看全部工具</button></div>
  </main>
</template>

<style scoped>
.toolbox-page{max-width:1180px;margin:0 auto;padding:36px 16px 60px;color:var(--color-text-main)}.toolbox-heading>span{font-size:12px;color:var(--color-accent);letter-spacing:.08em}.toolbox-heading h1{font-size:34px;letter-spacing:-.04em;margin:10px 0}.toolbox-heading p{color:var(--color-text-secondary);font-size:14px;margin:0}.toolbox-filters{margin-top:30px;padding:20px;background:var(--color-panel-bg);border:1px solid var(--color-border);border-radius:16px}.toolbox-search-row{display:flex;gap:14px}.toolbox-search{display:flex;flex:1;align-items:center;gap:10px;border:1px solid var(--input-border);border-radius:10px;padding:0 14px;background:var(--input-bg)}.toolbox-search input{width:100%;min-width:0;height:44px;border:0;background:transparent;color:var(--input-text);outline:none;font:inherit}.toolbox-search:focus-within{outline:2px solid var(--color-accent)}.toolbox-feature{display:flex;align-items:center;gap:10px;color:var(--color-text-secondary);font-size:12px}.toolbox-feature select{height:44px;min-width:150px;padding:0 12px;border:1px solid var(--input-border);border-radius:10px;background:var(--input-bg);color:var(--input-text)}.toolbox-tags{display:flex;align-items:center;flex-wrap:wrap;gap:8px;margin-top:16px;font-size:12px}.toolbox-tags>span{margin-right:6px;color:var(--color-text-secondary)}.toolbox-tags button{padding:6px 12px;border:1px solid var(--color-border);border-radius:999px;background:var(--color-card-bg);color:var(--color-text-secondary);cursor:pointer}.toolbox-tags button.selected{background:var(--btn-primary-bg);color:var(--btn-primary-text);border-color:var(--btn-primary-bg)}.toolbox-tags .reset{margin-left:auto;border:0;background:transparent;color:var(--link-color)}.toolbox-count{margin:24px 0 14px;font-size:12px;color:var(--color-text-secondary)}.toolbox-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:22px}.toolbox-card{display:flex;flex-direction:column;padding:26px;border:1px solid var(--color-border);border-radius:18px;background:var(--color-panel-bg);color:inherit;text-decoration:none;transition:transform .18s,border-color .18s,box-shadow .18s}.toolbox-card:hover{transform:translateY(-3px);border-color:var(--color-accent);box-shadow:var(--shadow-md)}.toolbox-card__top{display:flex;align-items:center;justify-content:space-between;margin-bottom:22px}.toolbox-card__icon{display:grid;place-items:center;width:60px;height:60px;border-radius:15px;background:var(--gradient-start);color:var(--gradient-text);font-size:30px}.toolbox-card__open{font-size:12px;color:var(--color-text-secondary)}.toolbox-card h2{font-size:23px;margin:0 0 8px;letter-spacing:-.02em}.toolbox-card>strong{font-size:12px;font-weight:500;color:var(--color-accent)}.toolbox-card>p{margin:18px 0;min-height:50px;font-size:13px;line-height:1.9;color:var(--color-text-secondary)}.toolbox-card__features{display:flex;flex-wrap:wrap;gap:12px;margin-bottom:24px;font-size:12px}.toolbox-card__features i{color:var(--color-accent);margin-right:4px}.toolbox-card footer{display:flex;flex-wrap:wrap;gap:8px;margin-top:auto;padding-top:18px;border-top:1px solid var(--color-border)}.toolbox-card footer span{font-size:10px;padding:4px 8px;border-radius:6px;background:var(--color-card-bg);color:var(--color-text-secondary)}.toolbox-empty{text-align:center;padding:70px 20px;color:var(--color-text-secondary)}.toolbox-empty>i{font-size:38px}.toolbox-empty h2{font-size:20px;color:var(--color-text-main)}.toolbox-empty p{font-size:13px}.toolbox-empty button{padding:10px 18px;border:0;border-radius:8px;background:var(--btn-primary-bg);color:var(--btn-primary-text);cursor:pointer}.toolbox-page :is(a,button,select):focus-visible{outline:3px solid var(--color-accent);outline-offset:3px}@media(max-width:900px){.toolbox-grid{grid-template-columns:1fr}.toolbox-search-row{flex-direction:column}}@media(prefers-reduced-motion:reduce){.toolbox-card{transition:none}}
</style>

<style scoped>

@media(max-width:600px){.toolbox-page{padding:22px 16px 32px}.toolbox-heading h1{font-size:28px}.toolbox-heading p{line-height:1.8}.toolbox-filters{padding:14px;margin-top:20px}.toolbox-search{min-width:0}.toolbox-search input{font-size:16px}.toolbox-feature select{flex:1;min-width:0;font-size:16px}.toolbox-tags button{min-height:40px}.toolbox-card{padding:20px}.toolbox-grid{gap:14px}.toolbox-card__top{gap:10px}.toolbox-card h2{font-size:21px}}

</style>
