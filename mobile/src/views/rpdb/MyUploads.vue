<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  deleteRPDBDraft,
  deleteRPDBWork,
  listMyRPDBWorks,
  listRPDBDrafts,
  resolveRPDBMediaUrl,
  updateRPDBWorkVisibility,
  type RPDBDraft,
  type RPDBVisibility,
  type RPDBWork,
} from '@/api/rpdb'
import { listGuilds, type Guild } from '@/api/guild'
import CachedImage from '@/components/CachedImage.vue'
import { getRPDBTypeIcon, getRPDBTypeLabel } from '@/utils/rpdb'

const router = useRouter()
const toast = useToastStore()
const loading = ref(true)
const works = ref<RPDBWork[]>([])
const drafts = ref<RPDBDraft[]>([])
const guilds = ref<Guild[]>([])
const activeStatus = ref<'all' | 'published' | 'pending' | 'draft' | 'rejected'>('all')
const busyId = ref('')
const deleteTarget = ref<{ kind: 'work' | 'draft'; id: number; title: string } | null>(null)

const statusTabs = computed(() => [
  { id: 'all' as const, label: '全部', count: works.value.length + drafts.value.length },
  { id: 'published' as const, label: '已发布', count: works.value.filter(item => item.status === 'published' && item.review_status !== 'rejected').length },
  { id: 'pending' as const, label: '审核中', count: works.value.filter(item => item.status === 'pending' || item.review_status === 'pending').length },
  { id: 'draft' as const, label: '草稿', count: drafts.value.length },
  { id: 'rejected' as const, label: '需修改', count: works.value.filter(item => item.review_status === 'rejected').length },
])
const visibleDrafts = computed(() => activeStatus.value === 'all' || activeStatus.value === 'draft' ? drafts.value : [])
const visibleWorks = computed(() => {
  if (activeStatus.value === 'all') return works.value
  if (activeStatus.value === 'draft') return []
  if (activeStatus.value === 'rejected') return works.value.filter(item => item.review_status === 'rejected')
  if (activeStatus.value === 'pending') return works.value.filter(item => item.status === 'pending' || item.review_status === 'pending')
  return works.value.filter(item => item.status === 'published' && item.review_status !== 'rejected')
})
const hasContent = computed(() => visibleDrafts.value.length > 0 || visibleWorks.value.length > 0)

function resolveVisibility(work: RPDBWork): RPDBVisibility {
  if (['public', 'guild', 'private'].includes(work.visibility)) return work.visibility
  return work.is_public ? 'public' : 'private'
}

function selectedGuildIds(work: RPDBWork) {
  return work.guild_ids?.length ? work.guild_ids : work.guild_id ? [work.guild_id] : []
}

function workStatus(work: RPDBWork) {
  if (work.review_status === 'rejected') return '需修改'
  if (work.status === 'pending' || work.review_status === 'pending') return '审核中'
  return work.status === 'published' ? '已发布' : '草稿'
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

async function load() {
  loading.value = true
  try {
    const [workResult, draftResult, guildResult] = await Promise.all([
      listMyRPDBWorks(),
      listRPDBDrafts(),
      listGuilds().catch(() => ({ guilds: [] })),
    ])
    works.value = workResult.works || []
    drafts.value = draftResult.drafts || []
    guilds.value = guildResult.guilds || []
  } catch (error) {
    toast.error((error as Error).message || '上传内容加载失败')
  } finally {
    loading.value = false
  }
}

async function changeVisibility(work: RPDBWork, visibility: RPDBVisibility) {
  let guildIds = selectedGuildIds(work)
  if (visibility === 'guild') {
    guildIds = guildIds.length ? guildIds : guilds.value[0]?.id ? [guilds.value[0].id] : []
    if (!guildIds.length) {
      toast.warning('需要先加入公会')
      return
    }
  }
  busyId.value = `work-${work.id}`
  try {
    const result = await updateRPDBWorkVisibility(work.id, visibility, guildIds)
    Object.assign(work, result.work)
    toast.success('可见范围已更新')
  } catch (error) {
    toast.error((error as Error).message || '更新失败')
  } finally {
    busyId.value = ''
  }
}

async function toggleGuild(work: RPDBWork, guildId: number) {
  const next = new Set(selectedGuildIds(work))
  if (next.has(guildId)) next.delete(guildId)
  else next.add(guildId)
  if (!next.size) {
    toast.warning('至少选择一个公会')
    return
  }
  busyId.value = `work-${work.id}`
  try {
    const result = await updateRPDBWorkVisibility(work.id, 'guild', [...next])
    Object.assign(work, result.work)
  } catch (error) {
    toast.error((error as Error).message || '更新失败')
  } finally {
    busyId.value = ''
  }
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target || busyId.value) return
  busyId.value = `${target.kind}-${target.id}`
  try {
    if (target.kind === 'draft') {
      await deleteRPDBDraft(target.id)
      drafts.value = drafts.value.filter(item => item.id !== target.id)
    } else {
      await deleteRPDBWork(target.id)
      works.value = works.value.filter(item => item.id !== target.id)
    }
    toast.success('内容已删除')
    deleteTarget.value = null
  } catch (error) {
    toast.error((error as Error).message || '删除失败')
  } finally {
    busyId.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="sub-page uploads-page">
    <header class="sub-header uploads-header">
      <button class="back-btn" type="button" aria-label="返回 RP 数据库" @click="router.push({ name: 'rpdb' })"><i class="ri-arrow-left-line" /></button>
      <div><small>创作工作区</small><h1>我的上传</h1></div>
      <button class="add-button" type="button" aria-label="发布内容" @click="router.push({ name: 'rpdb-create' })"><i class="ri-add-line" /></button>
    </header>

    <main class="sub-body">
      <section class="summary-grid">
        <div><b>{{ statusTabs[0].count }}</b><span>全部</span></div>
        <div><b>{{ statusTabs[1].count }}</b><span>已发布</span></div>
        <div><b>{{ statusTabs[2].count }}</b><span>审核中</span></div>
        <div><b>{{ statusTabs[3].count }}</b><span>草稿</span></div>
      </section>

      <nav class="status-tabs" aria-label="内容状态">
        <button v-for="tab in statusTabs" :key="tab.id" type="button" :class="{ active: activeStatus === tab.id }" @click="activeStatus = tab.id">
          {{ tab.label }}<span>{{ tab.count }}</span>
        </button>
      </nav>

      <div v-if="loading" class="page-state"><i class="ri-loader-4-line spin" /><span>正在整理上传内容</span></div>
      <div v-else-if="!hasContent" class="page-state">
        <i class="ri-draft-line" />
        <b>这里还没有内容</b>
        <button type="button" @click="router.push({ name: 'rpdb-create' })">发布第一份作品</button>
      </div>

      <section v-if="visibleDrafts.length" class="content-list">
        <article v-for="draft in visibleDrafts" :key="`draft-${draft.id}`" class="content-card">
          <div class="cover">
            <CachedImage v-if="draft.cover_image" :src="resolveRPDBMediaUrl(draft.cover_image)" :alt="draft.title" />
            <i v-else class="ri-draft-line" />
          </div>
          <div class="card-copy">
            <div class="eyebrow"><span>草稿</span><time>{{ formatDate(draft.updated_at) }}</time></div>
            <h2>{{ draft.title || '未命名草稿' }}</h2>
            <p>{{ draft.work_id ? '正式作品的修改草稿，发布前不会影响线上版本。' : '尚未发布的新作品。' }}</p>
          </div>
          <footer>
            <button type="button" @click="router.push({ name: 'rpdb-draft-edit', params: { draftId: draft.id } })"><i class="ri-edit-line" />继续编辑</button>
            <button type="button" class="danger" @click="deleteTarget = { kind: 'draft', id: draft.id, title: draft.title || '未命名草稿' }"><i class="ri-delete-bin-line" /></button>
          </footer>
        </article>
      </section>

      <section v-if="visibleWorks.length" class="content-list">
        <article v-for="work in visibleWorks" :key="work.id" class="content-card">
          <div class="cover">
            <CachedImage v-if="work.cover_image" :src="resolveRPDBMediaUrl(work.cover_image)" :alt="work.title" />
            <i v-else :class="getRPDBTypeIcon(work.type)" />
          </div>
          <div class="card-copy">
            <div class="eyebrow">
              <span :class="{ rejected: work.review_status === 'rejected' }">{{ workStatus(work) }}</span>
              <time>{{ formatDate(work.updated_at) }}</time>
            </div>
            <h2>{{ work.title }}</h2>
            <p>{{ work.summary || getRPDBTypeLabel(work.type) }}</p>
            <div class="metrics">
              <span><i class="ri-eye-line" />{{ work.view_count }}</span>
              <span><i class="ri-heart-3-line" />{{ work.like_count }}</span>
              <span><i class="ri-bookmark-3-line" />{{ work.favorite_count }}</span>
            </div>
          </div>
          <div class="visibility">
            <label>
              <span>可见范围</span>
              <select :value="resolveVisibility(work)" :disabled="busyId === `work-${work.id}`" @change="changeVisibility(work, ($event.target as HTMLSelectElement).value as RPDBVisibility)">
                <option value="public">公开</option>
                <option value="guild">公会可见</option>
                <option value="private">仅自己</option>
              </select>
            </label>
            <div v-if="resolveVisibility(work) === 'guild'" class="guild-options">
              <label v-for="guild in guilds" :key="guild.id" :class="{ selected: selectedGuildIds(work).includes(guild.id) }">
                <input type="checkbox" :checked="selectedGuildIds(work).includes(guild.id)" @change="toggleGuild(work, guild.id)">
                {{ guild.name }}
              </label>
            </div>
          </div>
          <footer>
            <button type="button" @click="router.push({ name: 'rpdb-detail', params: { id: work.id } })"><i class="ri-eye-line" />查看</button>
            <button type="button" @click="router.push({ name: 'rpdb-edit', params: { id: work.id } })"><i class="ri-edit-line" />编辑</button>
            <button type="button" class="danger" @click="deleteTarget = { kind: 'work', id: work.id, title: work.title }"><i class="ri-delete-bin-line" /></button>
          </footer>
        </article>
      </section>
    </main>

    <div v-if="deleteTarget" class="dialog-mask">
      <section class="dialog" role="dialog" aria-modal="true">
        <h2>删除内容</h2>
        <p>确定删除“{{ deleteTarget.title }}”吗？删除后无法恢复。</p>
        <footer>
          <button type="button" @click="deleteTarget = null">取消</button>
          <button type="button" class="danger" :disabled="Boolean(busyId)" @click="confirmDelete">确认删除</button>
        </footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.uploads-page{padding-bottom:calc(24px + var(--safe-bottom,0px))}
.uploads-header{grid-template-columns:44px minmax(0,1fr) 40px}.uploads-header>div{min-width:0}.uploads-header small{color:var(--color-accent);font-size:9px;font-weight:800;letter-spacing:.06em}.uploads-header h1{margin-top:2px;font-size:18px}.add-button{display:grid;width:40px;height:40px;place-items:center;border:0;border-radius:8px;background:var(--color-primary);color:#fff;font-size:19px}
.summary-grid{display:grid;grid-template-columns:repeat(4,1fr);overflow:hidden;margin-bottom:12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-panel-bg)}.summary-grid div{display:grid;gap:2px;padding:10px 4px;border-right:1px solid var(--color-border);text-align:center}.summary-grid div:last-child{border-right:0}.summary-grid b{font-size:17px}.summary-grid span{color:var(--color-text-secondary);font-size:9px}
.status-tabs{display:flex;gap:5px;margin-bottom:12px;overflow-x:auto;scrollbar-width:none}.status-tabs::-webkit-scrollbar{display:none}.status-tabs button{display:inline-flex;min-height:34px;flex:0 0 auto;align-items:center;gap:5px;padding:0 10px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-secondary);font-size:11px}.status-tabs button.active{border-color:var(--color-primary);background:var(--color-primary);color:#fff}.status-tabs span{display:grid;min-width:17px;height:17px;place-items:center;border-radius:4px;background:rgba(255,255,255,.16);font-size:8px}
.content-list{display:grid;gap:10px;margin-bottom:12px}.content-card{display:grid;grid-template-columns:92px minmax(0,1fr);gap:10px;overflow:hidden;padding:10px;border:1px solid rgba(75,54,33,.1);border-radius:8px;background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.cover{display:grid;width:92px;aspect-ratio:1/1;place-items:center;overflow:hidden;border-radius:6px;background:var(--color-primary-light);color:var(--color-secondary);font-size:26px}.card-copy{min-width:0}.eyebrow{display:flex;align-items:center;justify-content:space-between;gap:7px}.eyebrow span{padding:2px 5px;border-radius:3px;background:var(--tag-bg);color:var(--tag-text);font-size:9px;font-weight:800}.eyebrow span.rejected{background:#fff0ed;color:#b6382d}.eyebrow time{color:var(--color-text-muted);font-size:9px}.card-copy h2{overflow:hidden;margin:7px 0 4px;font-size:15px;text-overflow:ellipsis;white-space:nowrap}.card-copy p{display:-webkit-box;overflow:hidden;color:var(--color-text-secondary);font-size:11px;line-height:1.45;-webkit-box-orient:vertical;-webkit-line-clamp:2}.metrics{display:flex;gap:10px;margin-top:7px;color:var(--color-text-secondary);font-size:9px}.metrics span{display:inline-flex;align-items:center;gap:2px}
.visibility{display:grid;grid-column:1/-1;gap:8px;padding-top:9px;border-top:1px solid var(--color-border-light)}.visibility>label{display:grid;grid-template-columns:auto minmax(120px,1fr);align-items:center;gap:8px}.visibility span{color:var(--color-text-secondary);font-size:10px}.visibility select{height:36px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--color-text-main);padding:0 8px}.guild-options{display:flex;flex-wrap:wrap;gap:5px}.guild-options label{display:inline-flex;align-items:center;gap:4px;padding:5px 7px;border:1px solid var(--color-border);border-radius:5px;font-size:9px}.guild-options label.selected{border-color:var(--color-accent);background:var(--tag-bg)}.guild-options input{width:auto}
.content-card>footer{display:grid;grid-column:1/-1;grid-template-columns:1fr 1fr 42px;gap:6px}.content-card>footer button{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:5px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-main);font-size:11px}.content-card>footer .danger{color:#b6382d}.page-state{display:grid;min-height:300px;place-items:center;align-content:center;gap:9px;color:var(--color-text-secondary);text-align:center}.page-state>i{color:var(--color-secondary);font-size:36px}.page-state button{min-height:38px;padding:0 13px;border:0;border-radius:6px;background:var(--color-primary);color:#fff}
.dialog-mask{position:fixed;inset:0;z-index:2200;display:grid;place-items:center;padding:16px;background:rgba(44,24,16,.48)}.dialog{width:min(100%,360px);padding:16px;border-radius:8px;background:var(--color-panel-bg)}.dialog h2{font-size:18px}.dialog p{margin-top:8px;color:var(--color-text-secondary);font-size:13px;line-height:1.6}.dialog footer{display:flex;justify-content:flex-end;gap:8px;margin-top:16px}.dialog button{min-height:38px;padding:0 13px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg)}.dialog button.danger{border-color:#b6382d;background:#b6382d;color:#fff}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:350px){.content-card{grid-template-columns:76px minmax(0,1fr)}.cover{width:76px}}
</style>
