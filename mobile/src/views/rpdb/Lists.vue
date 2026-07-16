<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  createRPDBList,
  exportRPDBList,
  getRPDBWork,
  listRPDBLists,
  removeRPDBListEntry,
  resolveRPDBMediaUrl,
  updateRPDBListEntry,
  type RPDBGuideStep,
  type RPDBList,
  type RPDBListEntry,
  type RPDBListStatus,
} from '@/api/rpdb'
import CachedImage from '@/components/CachedImage.vue'
import { buildTomTomCommand, getRPDBTypeIcon } from '@/utils/rpdb'
import { copyTextToClipboard, shareTextFile } from '@/utils/mobileShare'

const router = useRouter()
const toast = useToastStore()
const loading = ref(true)
const lists = ref<RPDBList[]>([])
const activeIndex = ref(0)
const createOpen = ref(false)
const creating = ref(false)
const createForm = ref({ name: '', description: '', is_public: false })
const editingEntry = ref<RPDBListEntry | null>(null)
const entryForm = ref({ status: 'wanted' as RPDBListStatus, priority: 0, quantity: 1, note: '' })
const savingEntry = ref(false)
const removingEntry = ref<RPDBListEntry | null>(null)
const guideOpen = ref(false)
const guideLoading = ref(false)
const guideTitle = ref('')
const guideSteps = ref<RPDBGuideStep[]>([])

const activeList = computed(() => lists.value[activeIndex.value])
const entries = computed(() => activeList.value?.entries || [])
const ownedCount = computed(() => entries.value.filter(item => item.status === 'owned').length)
const completion = computed(() => entries.value.length ? Math.round((ownedCount.value / entries.value.length) * 100) : 0)

function statusLabel(status: RPDBListStatus) {
  return ({ wanted: '未收集', farming: '收集中', owned: '已收集', paused: '已暂停' } as const)[status]
}

async function load() {
  loading.value = true
  try {
    const result = await listRPDBLists()
    lists.value = result.lists || []
    if (activeIndex.value >= lists.value.length) activeIndex.value = 0
  } catch (error) {
    toast.error((error as Error).message || '清单加载失败')
  } finally {
    loading.value = false
  }
}

async function createList() {
  const name = createForm.value.name.trim()
  if (!name || creating.value) return
  creating.value = true
  try {
    const result = await createRPDBList(name, createForm.value.description.trim(), createForm.value.is_public)
    createOpen.value = false
    createForm.value = { name: '', description: '', is_public: false }
    await load()
    const index = lists.value.findIndex(item => item.id === result.list.id)
    if (index >= 0) activeIndex.value = index
    toast.success('清单已创建')
  } catch (error) {
    toast.error((error as Error).message || '创建失败')
  } finally {
    creating.value = false
  }
}

function openEntryEditor(entry: RPDBListEntry) {
  editingEntry.value = entry
  entryForm.value = {
    status: entry.status,
    priority: entry.priority || 0,
    quantity: Math.max(1, entry.quantity || 1),
    note: entry.note || '',
  }
}

async function saveEntry() {
  if (!editingEntry.value || savingEntry.value) return
  savingEntry.value = true
  try {
    await updateRPDBListEntry(editingEntry.value.list_id, editingEntry.value.work_id, entryForm.value)
    Object.assign(editingEntry.value, entryForm.value)
    editingEntry.value = null
    toast.success('收集进度已更新')
  } catch (error) {
    toast.error((error as Error).message || '保存失败')
  } finally {
    savingEntry.value = false
  }
}

async function quickToggle(entry: RPDBListEntry) {
  const status: RPDBListStatus = entry.status === 'owned' ? 'wanted' : 'owned'
  try {
    await updateRPDBListEntry(entry.list_id, entry.work_id, { status })
    entry.status = status
  } catch (error) {
    toast.error((error as Error).message || '更新失败')
  }
}

async function removeEntry() {
  const entry = removingEntry.value
  if (!entry) return
  try {
    await removeRPDBListEntry(entry.list_id, entry.work_id)
    const list = activeList.value
    if (list?.entries) {
      list.entries = list.entries.filter(item => item.id !== entry.id)
      list.item_count = list.entries.length
    }
    removingEntry.value = null
    toast.success('已移出清单')
  } catch (error) {
    toast.error((error as Error).message || '移除失败')
  }
}

async function exportList(format: 'json' | 'csv' | 'tomtom') {
  const list = activeList.value
  if (!list) return
  try {
    const result = await exportRPDBList(list.id, format)
    const content = result.content ?? (result.list ? JSON.stringify(result.list, null, 2) : '')
    if (!content.trim()) {
      toast.warning(format === 'tomtom' ? '清单中没有可用坐标' : '没有可导出的内容')
      return
    }
    if (format === 'tomtom') {
      await copyTextToClipboard(content, `${list.name} TomTom`)
      const missing = result.missing_coordinates?.length || 0
      toast.success(missing ? `路线已复制，${missing} 项没有坐标` : 'TomTom 路线已复制')
      return
    }
    await shareTextFile({
      filename: `${list.name}.${format}`,
      content,
      title: `导出 ${list.name}`,
      dialogTitle: '分享收集清单',
    })
  } catch (error) {
    toast.error((error as Error).message || '导出失败')
  }
}

async function openGuide(entry: RPDBListEntry) {
  guideOpen.value = true
  guideLoading.value = true
  guideTitle.value = entry.work.title
  guideSteps.value = []
  try {
    const result = await getRPDBWork(entry.work_id)
    guideSteps.value = [...(result.work.guide_steps || [])].sort((a, b) => a.sort_order - b.sort_order)
  } catch (error) {
    toast.error((error as Error).message || '攻略加载失败')
  } finally {
    guideLoading.value = false
  }
}

async function copyStep(step: RPDBGuideStep) {
  const command = buildTomTomCommand(step)
  if (!command) return
  await copyTextToClipboard(command, step.title)
  toast.success('坐标已复制')
}

onMounted(load)
</script>

<template>
  <div class="sub-page lists-page">
    <header class="sub-header lists-header">
      <button class="back-btn" type="button" aria-label="返回 RP 数据库" @click="router.push({ name: 'rpdb' })"><i class="ri-arrow-left-line" /></button>
      <div><small>收集助手</small><h1>RP 收集清单</h1></div>
      <button class="add-button" type="button" aria-label="新建清单" @click="createOpen = true"><i class="ri-add-line" /></button>
    </header>

    <main class="sub-body">
      <div v-if="loading" class="page-state"><i class="ri-loader-4-line spin" />正在加载清单</div>
      <template v-else-if="lists.length">
        <section class="list-selector">
          <label>
            <span>当前清单</span>
            <select v-model.number="activeIndex">
              <option v-for="(list, index) in lists" :key="list.id" :value="index">{{ list.name }} · {{ list.item_count }} 项</option>
            </select>
          </label>
          <div class="progress">
            <span><b>{{ ownedCount }}</b> / {{ entries.length }} 已收集</span>
            <strong>{{ completion }}%</strong>
            <div><i :style="{ width: `${completion}%` }" /></div>
          </div>
        </section>

        <section class="export-row">
          <button type="button" @click="exportList('json')"><i class="ri-braces-line" />JSON</button>
          <button type="button" @click="exportList('csv')"><i class="ri-file-excel-2-line" />CSV</button>
          <button type="button" class="tomtom" @click="exportList('tomtom')"><i class="ri-route-line" />TomTom /ttpaste</button>
        </section>

        <section v-if="entries.length" class="entry-list">
          <article v-for="entry in entries" :key="entry.id" :class="{ owned: entry.status === 'owned' }">
            <button class="entry-main" type="button" @click="router.push({ name: 'rpdb-detail', params: { id: entry.work_id }, query: { from: 'collection' } })">
              <span class="cover">
                <CachedImage v-if="entry.work.cover_image" :src="resolveRPDBMediaUrl(entry.work.cover_image)" :alt="entry.work.title" />
                <i v-else :class="getRPDBTypeIcon(entry.work.type)" />
              </span>
              <span class="copy">
                <span class="status" :class="entry.status">{{ statusLabel(entry.status) }}</span>
                <b>{{ entry.work.title }}</b>
                <small>{{ entry.note || entry.work.summary || '尚未填写备注' }}</small>
              </span>
            </button>
            <div class="entry-meta">
              <span>优先级 {{ entry.priority || 0 }}</span>
              <span>数量 {{ entry.quantity || 1 }}</span>
            </div>
            <footer>
              <button type="button" :class="{ active: entry.status === 'owned' }" @click="quickToggle(entry)">
                <i :class="entry.status === 'owned' ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'" />
                {{ entry.status === 'owned' ? '已收集' : '标记完成' }}
              </button>
              <button type="button" @click="openGuide(entry)"><i class="ri-route-line" />攻略</button>
              <button type="button" aria-label="编辑清单项" @click="openEntryEditor(entry)"><i class="ri-edit-line" /></button>
              <button type="button" class="danger" aria-label="移出清单" @click="removingEntry = entry"><i class="ri-delete-bin-line" /></button>
            </footer>
          </article>
        </section>
        <div v-else class="page-state"><i class="ri-list-check-3" /><b>这份清单还是空的</b><button type="button" @click="router.push({ name: 'rpdb' })">去发现作品</button></div>
      </template>
      <div v-else class="page-state"><i class="ri-list-check-3" /><b>创建第一份收集清单</b><button type="button" @click="createOpen = true">新建清单</button></div>
    </main>

    <div v-if="createOpen" class="dialog-mask">
      <section class="dialog" role="dialog" aria-modal="true">
        <h2>新建收集清单</h2>
        <label><span>清单名称</span><input v-model="createForm.name" maxlength="128" placeholder="例如：夜巡道具收集"></label>
        <label><span>说明</span><textarea v-model="createForm.description" rows="3" maxlength="512" /></label>
        <label class="switch-row"><span>公开清单</span><input v-model="createForm.is_public" type="checkbox"></label>
        <footer><button type="button" @click="createOpen = false">取消</button><button type="button" class="primary" :disabled="creating || !createForm.name.trim()" @click="createList">创建</button></footer>
      </section>
    </div>

    <div v-if="editingEntry" class="dialog-mask">
      <section class="dialog" role="dialog" aria-modal="true">
        <h2>编辑收集进度</h2>
        <label><span>状态</span><select v-model="entryForm.status"><option value="wanted">未收集</option><option value="farming">收集中</option><option value="owned">已收集</option><option value="paused">已暂停</option></select></label>
        <div class="two-fields"><label><span>优先级</span><input v-model.number="entryForm.priority" type="number" min="0" max="99"></label><label><span>数量</span><input v-model.number="entryForm.quantity" type="number" min="1" max="999"></label></div>
        <label><span>备注</span><textarea v-model="entryForm.note" rows="3" maxlength="500" /></label>
        <footer><button type="button" @click="editingEntry = null">取消</button><button type="button" class="primary" :disabled="savingEntry" @click="saveEntry">保存</button></footer>
      </section>
    </div>

    <div v-if="removingEntry" class="dialog-mask">
      <section class="dialog" role="dialog" aria-modal="true"><h2>移出清单</h2><p>确定移出“{{ removingEntry.work.title }}”吗？</p><footer><button type="button" @click="removingEntry = null">取消</button><button type="button" class="danger" @click="removeEntry">移出</button></footer></section>
    </div>

    <div v-if="guideOpen" class="sheet-mask" @click.self="guideOpen = false">
      <section class="guide-sheet" role="dialog" aria-modal="true">
        <header><div><small>获取攻略</small><h2>{{ guideTitle }}</h2></div><button type="button" @click="guideOpen = false"><i class="ri-close-line" /></button></header>
        <div v-if="guideLoading" class="page-state"><i class="ri-loader-4-line spin" />加载攻略</div>
        <ol v-else-if="guideSteps.length">
          <li v-for="(step, index) in guideSteps" :key="step.id || index">
            <span>{{ index + 1 }}</span><div><b>{{ step.title }}</b><p v-if="step.body">{{ step.body }}</p><small v-if="step.zone"><i class="ri-map-pin-line" />{{ step.zone }}</small><button v-if="buildTomTomCommand(step)" type="button" @click="copyStep(step)">复制坐标</button></div>
          </li>
        </ol>
        <div v-else class="page-state"><i class="ri-route-line" />作者暂未填写攻略</div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.lists-header{display:grid;grid-template-columns:44px minmax(0,1fr) 40px}.lists-header>div{min-width:0}.lists-header small{color:var(--color-accent);font-size:9px;font-weight:800}.lists-header h1{margin-top:2px;font-size:18px}.add-button{display:grid;width:40px;height:40px;place-items:center;border:0;border-radius:8px;background:var(--color-primary);color:#fff;font-size:19px}
.list-selector{display:grid;gap:12px;padding:12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-panel-bg)}.list-selector label{display:grid;gap:6px}.list-selector label>span{color:var(--color-text-secondary);font-size:10px;font-weight:700}.list-selector select{height:42px;padding:0 10px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--color-text-main);font-weight:700}.progress{display:grid;grid-template-columns:1fr auto;gap:5px;color:var(--color-text-secondary);font-size:10px}.progress b,.progress strong{color:var(--color-secondary)}.progress>div{grid-column:1/-1;height:6px;overflow:hidden;border-radius:3px;background:var(--color-border)}.progress i{display:block;height:100%;background:var(--color-accent)}
.export-row{display:grid;grid-template-columns:1fr 1fr 1.5fr;gap:6px;margin:10px 0}.export-row button{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:4px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);font-size:10px}.export-row .tomtom{border-color:rgba(184,115,51,.4);color:var(--color-secondary);font-weight:800}
.entry-list{display:grid;gap:10px}.entry-list article{overflow:hidden;border:1px solid rgba(75,54,33,.1);border-radius:8px;background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.entry-main{display:grid;width:100%;grid-template-columns:78px minmax(0,1fr);gap:10px;padding:10px;border:0;background:transparent;color:var(--color-text-main);text-align:left}.cover{display:grid;width:78px;aspect-ratio:1/1;place-items:center;overflow:hidden;border-radius:6px;background:var(--color-primary-light);color:var(--color-secondary);font-size:24px}.copy{display:flex;min-width:0;flex-direction:column}.copy .status{align-self:flex-start;padding:2px 5px;border-radius:3px;background:var(--tag-bg);color:var(--tag-text);font-size:8px}.copy .status.owned{background:var(--color-success-light);color:var(--color-success)}.copy b{overflow:hidden;margin-top:6px;font-size:14px;text-overflow:ellipsis;white-space:nowrap}.copy small{display:-webkit-box;overflow:hidden;margin-top:5px;color:var(--color-text-secondary);font-size:10px;line-height:1.45;-webkit-box-orient:vertical;-webkit-line-clamp:2}.entry-list article.owned .copy b{text-decoration:line-through;color:var(--color-text-secondary)}.entry-meta{display:flex;gap:10px;padding:0 10px 8px;color:var(--color-text-secondary);font-size:9px}.entry-list footer{display:grid;grid-template-columns:1.3fr 1fr 40px 40px;gap:5px;padding:8px;border-top:1px solid var(--color-border-light)}.entry-list footer button{display:inline-flex;min-height:36px;align-items:center;justify-content:center;gap:4px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);font-size:9px}.entry-list footer button.active{color:var(--color-success)}.entry-list footer .danger{color:#b6382d}
.page-state{display:grid;min-height:280px;place-items:center;align-content:center;gap:9px;color:var(--color-text-secondary);text-align:center}.page-state>i{color:var(--color-secondary);font-size:34px}.page-state button{min-height:38px;padding:0 13px;border:0;border-radius:6px;background:var(--color-primary);color:#fff}
.dialog-mask{position:fixed;inset:0;z-index:2200;display:grid;place-items:center;padding:16px;background:rgba(44,24,16,.5)}.dialog{display:grid;width:min(100%,370px);gap:11px;padding:16px;border-radius:8px;background:var(--color-panel-bg)}.dialog h2{font-size:18px}.dialog p{color:var(--color-text-secondary);font-size:13px}.dialog label{display:grid;gap:6px}.dialog label>span{color:var(--color-text-secondary);font-size:10px;font-weight:700}.dialog input,.dialog textarea,.dialog select{width:100%;min-height:40px;padding:9px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--color-text-main)}.switch-row{display:flex!important;align-items:center;justify-content:space-between}.switch-row input{width:auto}.two-fields{display:grid;grid-template-columns:1fr 1fr;gap:8px}.dialog footer{display:flex;justify-content:flex-end;gap:7px}.dialog footer button{min-height:38px;padding:0 13px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg)}.dialog footer .primary{border-color:var(--color-primary);background:var(--color-primary);color:#fff}.dialog footer .danger{border-color:#b6382d;background:#b6382d;color:#fff}
.sheet-mask{position:fixed;inset:0;z-index:2200;display:flex;align-items:flex-end;background:rgba(44,24,16,.5)}.guide-sheet{width:100%;max-height:78vh;overflow:auto;padding:16px var(--page-gutter) calc(18px + var(--safe-bottom,0px));border-radius:14px 14px 0 0;background:var(--color-panel-bg)}.guide-sheet header{display:flex;align-items:flex-start;justify-content:space-between;gap:10px}.guide-sheet header small{color:var(--color-accent);font-size:9px;font-weight:800}.guide-sheet h2{margin-top:3px;font-size:18px}.guide-sheet header button{display:grid;width:38px;height:38px;place-items:center;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg)}.guide-sheet ol{display:grid;gap:0;margin-top:15px;padding:0;list-style:none}.guide-sheet li{display:grid;grid-template-columns:30px minmax(0,1fr);gap:9px;padding-bottom:16px}.guide-sheet li>span{display:grid;width:30px;height:30px;place-items:center;border-radius:5px;background:var(--color-primary);color:#edbf84;font-weight:800}.guide-sheet li p{margin-top:6px;color:var(--color-text-secondary);font-size:11px;line-height:1.6}.guide-sheet li small{display:block;margin-top:6px;color:var(--color-text-secondary)}.guide-sheet li button{min-height:32px;margin-top:7px;padding:0 9px;border:1px solid var(--color-border);border-radius:5px;background:var(--color-card-bg);color:var(--color-secondary);font-size:10px}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
</style>
