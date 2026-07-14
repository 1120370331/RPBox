<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  createRPDBList,
  exportRPDBList,
  getRPDBWork,
  listRPDBLists,
  removeRPDBListEntry,
  resolveRPDBMediaURL,
  updateRPDBListEntry,
  type RPDBWork,
  type RPDBList,
  type RPDBListEntry,
  type RPDBListStatus,
} from '@/api/rpdb'
import RModal from '@/components/RModal.vue'
import RPDBGuideSection from '@/components/rpdb/RPDBGuideSection.vue'
import { useToastStore } from '@/stores/toast'
import { useDialog } from '@/composables/useDialog'

const toast = useToastStore()
const dialog = useDialog()
const lists = ref<RPDBList[]>([])
const active = ref(0)
const showCreateModal = ref(false)
const newListName = ref('')
const newListDescription = ref('')
const creating = ref(false)
const guideModalOpen = ref(false)
const guideLoading = ref(false)
const guideError = ref('')
const guideEntry = ref<RPDBListEntry | null>(null)
const guideWork = ref<RPDBWork | null>(null)

const activeList = computed(() => lists.value[active.value])
const guideSteps = computed(() => guideWork.value?.guide_steps || [])
const guideModalTitle = computed(() => guideWork.value?.title || guideEntry.value?.work.title || '获取攻略')
const guideSectionTitle = computed(() => guideWork.value?.type === 'transmog' ? '部件获取攻略' : '获取攻略')
const statusLabels: Record<RPDBListStatus, string> = {
  wanted: '未收集',
  farming: '未收集',
  owned: '已收集',
  paused: '未收集',
}
const collectedCount = computed(() => activeList.value?.entries.filter(entry => entry.status === 'owned').length || 0)
const pendingCount = computed(() => activeList.value?.entries.filter(entry => entry.status !== 'owned').length || 0)

async function load() {
  try {
    const result = await listRPDBLists()
    lists.value = result.lists || []
    if (active.value >= lists.value.length) active.value = 0
  } catch (error) {
    toast.error((error as Error).message)
  }
}

function openCreateModal() {
  newListName.value = ''
  newListDescription.value = ''
  showCreateModal.value = true
}

async function create() {
  const nextName = newListName.value.trim()
  if (!nextName || creating.value) return
  creating.value = true
  try {
    const result = await createRPDBList(nextName, newListDescription.value.trim())
    newListName.value = ''
    newListDescription.value = ''
    showCreateModal.value = false
    toast.success('清单已创建')
    await load()
    const createdIndex = lists.value.findIndex(list => list.id === result.list?.id)
    if (createdIndex >= 0) active.value = createdIndex
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    creating.value = false
  }
}

async function status(entry: RPDBListEntry, value: RPDBListStatus) {
  await updateRPDBListEntry(entry.list_id, entry.work_id, { status: value })
  entry.status = value
}

async function toggleCollected(entry: RPDBListEntry) {
  await status(entry, entry.status === 'owned' ? 'wanted' : 'owned')
}

async function openGuide(entry: RPDBListEntry) {
  guideEntry.value = entry
  guideWork.value = null
  guideError.value = ''
  guideModalOpen.value = true
  guideLoading.value = true
  try {
    const result = await getRPDBWork(entry.work_id)
    guideWork.value = result.work
  } catch (error) {
    guideError.value = (error as Error).message
    toast.error(guideError.value)
  } finally {
    guideLoading.value = false
  }
}

async function remove(entry: RPDBListEntry) {
  const confirmed = await dialog.confirm({
    title: '移出清单',
    message: `确定移出「${entry.work.title}」吗？`,
    type: 'warning',
  })
  if (!confirmed) return
  await removeRPDBListEntry(entry.list_id, entry.work_id)
  await load()
}

async function exportList(format: 'json' | 'csv' | 'tomtom') {
  const list = activeList.value
  if (!list) return
  const result = await exportRPDBList(list.id, format)
  const content = result.content ?? (result.list ? JSON.stringify(result.list, null, 2) : '')
  if (format === 'tomtom' && !content.trim()) {
    toast.warning('当前清单没有可导出的 TomTom 坐标')
    return
  }
  await navigator.clipboard?.writeText(content)
  if (format === 'tomtom') {
    const missingCount = result.missing_coordinates?.length || 0
    if (missingCount) {
      toast.warning(`TomTom 路线已复制，${missingCount} 项没有坐标`)
    } else {
      toast.success('TomTom 路线已复制，可在游戏内使用 /ttpaste')
    }
    return
  }
  toast.success(`${format.toUpperCase()} 已复制到剪贴板`)
}

onMounted(load)
</script>

<template>
  <div class="lists-page minimal-lists-shell">
    <header class="lists-heading">
      <div>
        <span>收集清单</span>
        <h1>RP 收集清单</h1>
        <p>把内容加入清单后，在这里追踪是否已收集，并随时进入帖子查看攻略。</p>
      </div>
      <router-link to="/rpdb">
        <i class="ri-compass-3-line"></i>
        返回发现
      </router-link>
    </header>

    <div class="lists-layout">
      <section class="list-switcher" aria-label="收集清单列表">
        <div class="switcher-head">
          <div>
            <b>清单列表</b>
            <span>{{ lists.length }} 份清单</span>
          </div>
          <button type="button" class="create-open" data-testid="rpdb-list-create-open" @click="openCreateModal">
            <i class="ri-add-line"></i>
            新建清单
          </button>
        </div>

        <div class="list-select-row">
          <label class="list-select">
            <span>当前清单</span>
            <select
              :value="active"
              data-testid="rpdb-list-select"
              aria-label="选择收集清单"
              @change="active = Number(($event.target as HTMLSelectElement).value)"
            >
              <option
                v-for="(list, index) in lists"
                :key="list.id"
                :value="index"
              >
                {{ list.name }} · {{ list.item_count }} 项内容{{ list.is_default ? ' · 默认' : '' }}
              </option>
            </select>
          </label>
          <div v-if="activeList" class="selected-list-card">
            <i :class="activeList.is_default ? 'ri-star-fill' : 'ri-list-check-3'"></i>
            <span>
              <b>{{ activeList.name }}</b>
              <small>{{ activeList.description || `${activeList.item_count} 项内容` }}</small>
            </span>
          </div>
        </div>
      </section>

      <main v-if="activeList" class="list-workspace">
        <div class="toolbar">
          <div>
            <h2>{{ activeList.name }}</h2>
            <p>{{ activeList.description || '追踪这份清单里的 RP 内容收集进度' }}</p>
          </div>
          <div>
            <button type="button" @click="exportList('json')">JSON</button>
            <button type="button" @click="exportList('csv')">CSV</button>
            <button type="button" class="tomtom-export" data-testid="tomtom-list-export" @click="exportList('tomtom')">
              <i class="ri-route-line"></i>
              TomTom /ttpaste
            </button>
          </div>
        </div>

        <div class="summary">
          <span>
            <b>{{ pendingCount }}</b>
            未收集
          </span>
          <span>
            <b>{{ collectedCount }}</b>
            已收集
          </span>
          <span>
            <b>{{ activeList.entries.length }}</b>
            清单内容
          </span>
        </div>

        <div v-if="activeList.entries.length" class="entries">
          <article v-for="entry in activeList.entries" :key="entry.id" :class="{ collected: entry.status === 'owned' }">
            <div class="cover">
              <img v-if="entry.work.cover_image" :src="resolveRPDBMediaURL(entry.work.cover_image)" alt="">
              <i v-else class="ri-archive-line"></i>
            </div>
            <div class="entry-copy">
              <router-link :to="`/rpdb/${entry.work_id}`">{{ entry.work.title }}</router-link>
              <p>{{ entry.work.summary }}</p>
            </div>
            <button
              type="button"
              class="collect-toggle"
              data-testid="collection-owned-toggle"
              :class="{ collected: entry.status === 'owned' }"
              @click="toggleCollected(entry)"
            >
              <i :class="entry.status === 'owned' ? 'ri-checkbox-circle-fill' : 'ri-checkbox-blank-circle-line'"></i>
              {{ entry.status === 'owned' ? '已收集' : '未收集' }}
            </button>
            <button type="button" class="open-guide" data-testid="open-guide-link" @click="openGuide(entry)">
              <i class="ri-route-line"></i>
              查攻略
            </button>
            <router-link class="open-work" data-testid="open-work-link" :to="{ path: `/rpdb/${entry.work_id}`, query: { from: 'collection' } }">
              <i class="ri-external-link-line"></i>
              帖子
            </router-link>
            <button type="button" class="remove" data-testid="remove-collection-entry" @click="remove(entry)">
              <i class="ri-delete-bin-line"></i>
            </button>
          </article>
        </div>

        <div v-else class="empty">
          <i class="ri-bookmark-3-line"></i>
          <p>清单还是空的，从发现页加入作品。</p>
        </div>
      </main>

      <main v-else class="empty list-workspace">
        <i class="ri-list-check-3"></i>
        <p>创建第一份 RP 收集清单。</p>
      </main>
    </div>

    <RModal v-model="showCreateModal" title="新建收集清单" width="520px">
      <form class="create-modal" data-testid="rpdb-list-create-form" @submit.prevent="create">
        <label>
          <span>清单名称</span>
          <input v-model="newListName" data-testid="rpdb-list-create-name" maxlength="128" placeholder="例如：夜巡道具收集">
        </label>
        <label>
          <span>详情</span>
          <textarea v-model="newListDescription" data-testid="rpdb-list-create-description" rows="4" maxlength="512" placeholder="写清这份清单要收集什么，供自己之后查看。"></textarea>
        </label>
      </form>
      <template #footer>
        <button type="button" class="modal-secondary" @click="showCreateModal = false">取消</button>
        <button type="button" class="modal-primary" data-testid="rpdb-list-create-submit" :disabled="creating || !newListName.trim()" @click="create">
          {{ creating ? '创建中' : '创建清单' }}
        </button>
      </template>
    </RModal>

    <RModal v-model="guideModalOpen" :title="guideModalTitle" width="880px">
      <div class="guide-modal" data-testid="collection-guide-modal">
        <div v-if="guideLoading" class="guide-modal-state">
          <i class="ri-loader-4-line spin"></i>
          <span>正在加载攻略</span>
        </div>
        <div v-else-if="guideError" class="guide-modal-state error">
          <i class="ri-error-warning-line"></i>
          <span>{{ guideError }}</span>
        </div>
        <RPDBGuideSection
          v-else-if="guideSteps.length"
          :steps="guideSteps"
          :title="guideSectionTitle"
        />
        <div v-else class="guide-modal-state">
          <i class="ri-route-line"></i>
          <span>作者暂未填写攻略步骤。</span>
        </div>
      </div>
      <template #footer>
        <router-link
          v-if="guideEntry"
          class="modal-secondary open-detail"
          data-testid="guide-modal-open-detail"
          :to="{ path: `/rpdb/${guideEntry.work_id}`, query: { from: 'collection' } }"
        >
          打开帖子
        </router-link>
        <button type="button" class="modal-primary" @click="guideModalOpen = false">关闭</button>
      </template>
    </RModal>
  </div>
</template>

<style scoped>
.lists-page{max-width:1280px;margin:auto;color:var(--color-text-main)}
.minimal-lists-shell{--rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 88%,#fff 12%);--rpdb-muted:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%);--rpdb-line:color-mix(in srgb,var(--color-border) 72%,transparent);--rpdb-soft:color-mix(in srgb,var(--color-accent) 8%,transparent)}
.lists-heading{display:flex;justify-content:space-between;align-items:flex-end;gap:18px;padding-bottom:16px;border-bottom:1px solid var(--rpdb-line)}
.lists-heading span{color:var(--color-accent);font-size:11px;font-weight:800;letter-spacing:.06em}
.lists-heading h1{margin:6px 0 4px;font:700 30px/1.2 system-ui,'Microsoft YaHei',sans-serif}
.lists-heading p{margin:0;color:var(--color-text-secondary)}
.lists-heading a{display:inline-flex;align-items:center;gap:6px;color:var(--color-accent);text-decoration:none}
.lists-layout{display:grid;gap:14px;margin-top:16px}
.list-switcher,.list-workspace{border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.list-switcher{display:grid;gap:12px;padding:14px}
.switcher-head{display:flex;align-items:center;justify-content:space-between;gap:12px}
.switcher-head>div{display:flex;min-width:0;flex-direction:column;gap:3px}
.switcher-head b{font-size:15px}
.switcher-head span{color:var(--color-text-secondary);font-size:12px}
.create-open{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:6px;padding:0 14px;border:0;border-radius:10px;background:var(--color-accent);color:#fff;font-weight:800;box-shadow:var(--shadow-sm)}
.list-select-row{display:grid;grid-template-columns:minmax(260px,380px) minmax(0,1fr);align-items:end;gap:12px}
.list-select{display:grid;gap:7px;color:var(--color-text-secondary);font-size:13px;font-weight:800}
.list-select select{height:42px;padding:0 38px 0 12px;border:1px solid var(--input-border);border-radius:10px;background:var(--input-bg);color:var(--color-text-main);font:inherit;font-weight:700}
.list-select select:focus{border-color:var(--input-focus);outline:0;box-shadow:0 0 0 3px rgba(var(--shadow-base),.1)}
.selected-list-card{display:flex;min-height:42px;align-items:center;gap:10px;padding:9px 12px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--color-panel-bg)}
.selected-list-card>i{color:var(--color-accent);font-size:18px}
.selected-list-card span{display:flex;min-width:0;flex-direction:column}
.selected-list-card b,.selected-list-card small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.selected-list-card small{margin-top:3px;color:var(--color-text-secondary);font-size:12px}
.list-workspace{min-height:540px;padding:20px}
.toolbar{display:flex;justify-content:space-between;align-items:center;gap:16px}
.toolbar h2{margin:0;font:700 22px/1.25 system-ui,'Microsoft YaHei',sans-serif}
.toolbar p{margin:5px 0 0;color:var(--color-text-secondary)}
.toolbar>div:last-child{display:flex;flex-wrap:wrap;gap:7px}
.toolbar button{min-height:34px;padding:0 11px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main)}
.toolbar .tomtom-export{display:inline-flex;align-items:center;gap:6px;border-color:color-mix(in srgb,var(--color-accent) 48%,var(--rpdb-line));color:var(--color-accent);font-weight:800}.toolbar .tomtom-export i{font-size:15px}
.summary{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:16px 0}
.summary span{padding:13px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--rpdb-muted);color:var(--color-text-secondary)}
.summary b{display:block;margin-bottom:3px;font-size:22px;color:var(--color-accent)}
.entries{display:grid;gap:8px}
.entries article{display:grid;grid-template-columns:68px minmax(0,1fr) 118px 88px 78px 38px;align-items:center;gap:10px;padding:9px;border:1px solid var(--rpdb-line);border-radius:12px;background:var(--color-panel-bg)}
.cover{display:grid;width:68px;height:52px;place-items:center;overflow:hidden;border-radius:10px;background:#211914;color:#fff}
.cover img{width:100%;height:100%;object-fit:cover}
.entry-copy{min-width:0}
.entries a{font-weight:800;color:var(--color-text-main);text-decoration:none}
.entries p{margin:4px 0 0;overflow:hidden;color:var(--color-text-secondary);font-size:12px;text-overflow:ellipsis;white-space:nowrap}
.entries article.collected .entry-copy a,.entries article.collected .entry-copy p{color:var(--color-text-tertiary,var(--color-text-secondary));text-decoration:line-through;text-decoration-thickness:1px;text-decoration-color:currentColor}
.collect-toggle{display:inline-flex;min-height:36px;align-items:center;justify-content:center;gap:6px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-muted);color:var(--color-text-main);font-weight:800}
.collect-toggle.collected{border-color:var(--color-accent);background:var(--rpdb-soft);color:var(--color-accent)}
.remove{height:36px;border:1px solid var(--rpdb-line);border-radius:10px;background:transparent;color:#a33}
.open-guide,.open-work{display:inline-flex;min-height:34px;align-items:center;justify-content:center;gap:5px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-muted);color:var(--color-accent)!important;text-decoration:none;font-weight:800}
.open-guide{cursor:pointer;font:inherit}
.empty{display:grid;min-height:360px;place-items:center;align-content:center;color:var(--color-text-secondary);text-align:center}
.empty i{font-size:40px;color:var(--color-accent)}
.create-modal{display:grid;gap:16px}
.create-modal label{display:grid;gap:7px;color:var(--color-text-secondary);font-size:13px;font-weight:700}
.create-modal input,.create-modal textarea{width:100%;box-sizing:border-box;border:1px solid var(--input-border);border-radius:10px;background:var(--input-bg);color:var(--color-text-main);font:inherit}
.create-modal input{height:40px;padding:0 12px}
.create-modal textarea{resize:vertical;min-height:100px;padding:10px 12px}
.create-modal input:focus,.create-modal textarea:focus{border-color:var(--input-focus);outline:0;box-shadow:0 0 0 3px rgba(var(--shadow-base),.1)}
.modal-secondary,.modal-primary{min-height:36px;padding:0 14px;border-radius:10px;font-weight:800}
.modal-secondary{border:1px solid var(--rpdb-line);background:var(--color-panel-bg);color:var(--color-text-main)}
.modal-primary{border:0;background:var(--color-accent);color:#fff}
.modal-primary:disabled{cursor:not-allowed;opacity:.45}
.open-detail{display:inline-flex;align-items:center;justify-content:center;text-decoration:none}
.guide-modal :deep(.guide-section){padding:0;border-top:0}
.guide-modal :deep(.guide-heading){padding-bottom:14px}
.guide-modal-state{display:grid;min-height:220px;place-items:center;align-content:center;gap:10px;color:var(--color-text-secondary);text-align:center}
.guide-modal-state i{color:var(--color-accent);font-size:34px}
.guide-modal-state.error i{color:#b94a48}
.spin{animation:spin 1s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:800px){.lists-heading,.switcher-head{align-items:flex-start;flex-direction:column}.switcher-head .create-open{width:100%}.list-select-row{grid-template-columns:1fr}.summary{grid-template-columns:1fr}.entries article{grid-template-columns:60px 1fr}.collect-toggle,.open-guide,.open-work,.remove{grid-column:auto}.toolbar{align-items:flex-start;flex-direction:column}.toolbar>div:last-child{width:100%}.toolbar button{flex:1}}
</style>
