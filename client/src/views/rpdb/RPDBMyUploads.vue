<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  deleteRPDBDraft,
  deleteRPDBWork,
  listMyRPDBWorks,
  listRPDBDrafts,
  resolveRPDBMediaURL,
  updateRPDBWorkVisibility,
  type RPDBDraft,
  type RPDBVisibility,
  type RPDBWork,
} from '@/api/rpdb'
import { listGuilds, type Guild } from '@/api/guild'
import { useToastStore } from '@/stores/toast'
import { dialog } from '@/composables/useDialog'

const router = useRouter()
const toast = useToastStore()
const works = ref<RPDBWork[]>([])
const drafts = ref<RPDBDraft[]>([])
const guilds = ref<Guild[]>([])
const loading = ref(true)
const busyWorkId = ref<number | null>(null)
const statusFilter = ref<'all' | 'draft' | 'pending' | 'published' | 'rejected'>('all')

const filteredWorks = computed(() => {
  if (statusFilter.value === 'all') return works.value
  if (statusFilter.value === 'draft') return []
  if (statusFilter.value === 'rejected') return works.value.filter(work => work.review_status === 'rejected')
  return works.value.filter(work => work.status === statusFilter.value)
})
const filteredDrafts = computed(() => statusFilter.value === 'all' || statusFilter.value === 'draft' ? drafts.value : [])
const hasFilteredContent = computed(() => filteredWorks.value.length > 0 || filteredDrafts.value.length > 0)
const counts = computed(() => ({
  all: works.value.length + drafts.value.length,
  published: works.value.filter(work => work.status === 'published').length,
  pending: works.value.filter(work => work.status === 'pending').length,
  draft: drafts.value.length,
  private: works.value.filter(work => resolveVisibility(work) === 'private').length,
}))
const tabs = computed(() => [
  { id: 'all' as const, label: '全部', count: counts.value.all },
  { id: 'published' as const, label: '已发布', count: counts.value.published },
  { id: 'pending' as const, label: '审核中', count: counts.value.pending },
  { id: 'draft' as const, label: '草稿', count: counts.value.draft },
  { id: 'rejected' as const, label: '需修改', count: works.value.filter(work => work.review_status === 'rejected').length },
])

function resolveVisibility(work: RPDBWork): RPDBVisibility {
  if (work.visibility === 'public' || work.visibility === 'guild' || work.visibility === 'private') return work.visibility
  return work.is_public ? 'public' : 'private'
}

function statusLabel(work: RPDBWork) {
  if (work.review_status === 'rejected') return '需修改'
  if (work.status === 'published') return '已发布'
  if (work.status === 'pending') return '审核中'
  return '草稿'
}

function typeLabel(work: RPDBWork) {
  if (work.type === 'transmog') return '幻化方案'
  if (work.type === 'home_showcase') return '家宅分享'
  if (work.type === 'musician_midi') return 'Musician MIDI'
  return '魔兽物品'
}

function selectedGuildIDs(work: RPDBWork) {
  return work.guild_ids?.length ? work.guild_ids : work.guild_id ? [work.guild_id] : []
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

async function load() {
  loading.value = true
  try {
    const [workResult, draftResult, guildResult] = await Promise.all([listMyRPDBWorks(), listRPDBDrafts(), listGuilds()])
    works.value = workResult.works || []
    drafts.value = draftResult.drafts || []
    guilds.value = guildResult.guilds || []
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

async function removeDraft(draft: RPDBDraft) {
  const confirmed = await dialog.confirm({
    title: '删除草稿',
    message: `确定永久删除「${draft.title || '未命名草稿'}」吗？关联的正式作品不会受到影响。`,
    type: 'warning',
    confirmText: '删除',
  })
  if (!confirmed) return
  try {
    await deleteRPDBDraft(draft.id)
    drafts.value = drafts.value.filter(item => item.id !== draft.id)
    toast.success('草稿已删除')
  } catch (error) {
    toast.error((error as Error).message)
  }
}

async function changeVisibility(work: RPDBWork, visibility: RPDBVisibility) {
  const previousVisibility = resolveVisibility(work)
  let guildIDs = selectedGuildIDs(work)
  if (visibility === 'guild') {
    guildIDs = guildIDs.length ? guildIDs : guilds.value[0]?.id ? [guilds.value[0].id] : []
    if (!guildIDs.length) {
      toast.error('你还没有加入公会，无法设置为公会可见')
      return
    }
  }
  busyWorkId.value = work.id
  try {
    const result = await updateRPDBWorkVisibility(work.id, visibility, guildIDs)
    Object.assign(work, result.work)
    toast.success('可见范围已更新')
  } catch (error) {
    work.visibility = previousVisibility
    toast.error((error as Error).message)
  } finally {
    busyWorkId.value = null
  }
}

async function toggleGuild(work: RPDBWork, guildID: number) {
  const previousGuildIDs = selectedGuildIDs(work)
  const nextGuildIDs = new Set(previousGuildIDs)
  if (nextGuildIDs.has(guildID)) nextGuildIDs.delete(guildID)
  else nextGuildIDs.add(guildID)
  if (!nextGuildIDs.size) {
    toast.error('公会可见至少需要选择一个公会')
    return
  }
  busyWorkId.value = work.id
  try {
    const result = await updateRPDBWorkVisibility(work.id, 'guild', Array.from(nextGuildIDs))
    Object.assign(work, result.work)
    toast.success('可见公会已更新')
  } catch (error) {
    work.guild_ids = previousGuildIDs
    toast.error((error as Error).message)
  } finally {
    busyWorkId.value = null
  }
}

async function removeWork(work: RPDBWork) {
  const confirmed = await dialog.confirm({
    title: '删除上传内容',
    message: `确定删除「${work.title}」吗？草稿会永久删除，已发布内容会从所有列表中归档。`,
    type: 'warning',
    confirmText: '删除',
  })
  if (!confirmed) return
  busyWorkId.value = work.id
  try {
    await deleteRPDBWork(work.id)
    works.value = works.value.filter(item => item.id !== work.id)
    toast.success('上传内容已删除')
  } catch (error) {
    toast.error((error as Error).message)
  } finally {
    busyWorkId.value = null
  }
}

onMounted(load)
</script>

<template>
  <main class="uploads-page">
    <header class="uploads-header">
      <div>
        <button type="button" class="back-link" @click="router.push('/rpdb')"><i class="ri-arrow-left-line"></i>返回 RP 数据库</button>
        <h1>我的上传</h1>
        <p>管理作品状态、可见范围和后续修订。</p>
      </div>
      <button type="button" class="create-button" @click="router.push('/rpdb/create')"><i class="ri-add-line"></i>发布内容</button>
    </header>

    <section class="summary-strip" aria-label="上传统计">
      <div><b>{{ counts.all }}</b><span>全部上传</span></div>
      <div><b>{{ counts.published }}</b><span>已发布</span></div>
      <div><b>{{ counts.pending }}</b><span>审核中</span></div>
      <div><b>{{ counts.draft }}</b><span>草稿</span></div>
      <div><b>{{ counts.private }}</b><span>仅自己可见</span></div>
    </section>

    <nav class="status-tabs" aria-label="上传状态筛选">
      <button v-for="tab in tabs" :key="tab.id" type="button" :class="{ active: statusFilter === tab.id }" @click="statusFilter = tab.id">
        {{ tab.label }}<b>{{ tab.count }}</b>
      </button>
    </nav>

    <div v-if="loading" class="page-state"><i class="ri-loader-4-line spin"></i>正在加载上传内容</div>
    <div v-else-if="!hasFilteredContent" class="page-state empty">
      <i class="ri-upload-cloud-2-line"></i>
      <h2>这个分类还没有内容</h2>
      <button type="button" @click="router.push('/rpdb/create')">发布第一份作品</button>
    </div>
    <section v-if="!loading && filteredWorks.length" class="uploads-list" data-testid="rpdb-my-uploads-list">
      <article v-for="work in filteredWorks" :key="work.id" class="upload-row" data-testid="rpdb-upload-row">
        <div class="upload-cover" :class="{ empty: !work.cover_image }">
          <img v-if="work.cover_image" :src="resolveRPDBMediaURL(work.cover_image)" :alt="work.title">
          <i v-else class="ri-image-line"></i>
        </div>
        <div class="upload-main">
          <div class="upload-heading">
            <div>
              <span class="type-label">{{ typeLabel(work) }}</span>
              <span class="status-label" :class="[work.status, work.review_status]">{{ statusLabel(work) }}</span>
            </div>
            <time>{{ formatDate(work.updated_at) }}</time>
          </div>
          <h2>{{ work.title }}</h2>
          <p>{{ work.summary || '尚未填写作品摘要。' }}</p>
          <div class="upload-metrics" aria-label="作品数据">
            <span title="浏览"><i class="ri-eye-line"></i>{{ work.view_count }}</span>
            <span title="点赞"><i class="ri-heart-3-line"></i>{{ work.like_count }}</span>
            <span title="收藏"><i class="ri-bookmark-3-line"></i>{{ work.favorite_count }}</span>
            <span title="清单"><i class="ri-list-check-3"></i>{{ work.list_count }}</span>
          </div>
        </div>
        <div class="visibility-control">
          <label>
            <span>可见范围</span>
            <select :value="resolveVisibility(work)" :disabled="busyWorkId === work.id" @change="changeVisibility(work, ($event.target as HTMLSelectElement).value as RPDBVisibility)">
              <option value="public">公开</option>
              <option value="guild">公会可见</option>
              <option value="private">仅自己</option>
            </select>
          </label>
          <label v-if="resolveVisibility(work) === 'guild'">
            <span>查看公会（可多选）</span>
            <span class="visibility-guilds">
              <label v-for="guild in guilds" :key="guild.id" :class="{ selected: selectedGuildIDs(work).includes(guild.id) }">
                <input type="checkbox" :checked="selectedGuildIDs(work).includes(guild.id)" :disabled="busyWorkId === work.id" @change="toggleGuild(work, guild.id)">
                {{ guild.name }}
              </label>
            </span>
          </label>
        </div>
        <div class="row-actions">
          <button type="button" title="查看" @click="router.push(`/rpdb/${work.id}`)"><i class="ri-eye-line"></i><span>查看</span></button>
          <button type="button" title="编辑" @click="router.push(`/rpdb/${work.id}/edit`)"><i class="ri-edit-line"></i><span>编辑</span></button>
          <button type="button" class="danger" title="删除" :disabled="busyWorkId === work.id" @click="removeWork(work)"><i class="ri-delete-bin-line"></i><span>删除</span></button>
        </div>
      </article>
    </section>
    <section v-if="!loading && filteredDrafts.length" class="uploads-list draft-list" data-testid="rpdb-draft-list">
      <article v-for="draft in filteredDrafts" :key="`draft-${draft.id}`" class="upload-row" data-testid="rpdb-draft-row">
        <div class="upload-cover" :class="{ empty: !draft.cover_image }">
          <img v-if="draft.cover_image" :src="resolveRPDBMediaURL(draft.cover_image)" :alt="draft.title">
          <i v-else class="ri-draft-line"></i>
        </div>
        <div class="upload-main">
          <div class="upload-heading">
            <div>
              <span class="type-label">{{ draft.type === 'transmog' ? '幻化方案' : draft.type === 'home_showcase' ? '家宅分享' : draft.type === 'musician_midi' ? 'Musician MIDI' : '魔兽物品' }}</span>
              <span class="status-label draft">草稿</span>
            </div>
            <time>{{ formatDate(draft.updated_at) }}</time>
          </div>
          <h2>{{ draft.title || '未命名草稿' }}</h2>
          <p>{{ draft.work_id ? '这是正式作品的关联修改草稿，发布前不会改动线上内容。' : '尚未发布的新内容草稿。' }}</p>
        </div>
        <div class="visibility-control">
          <span>草稿归属</span>
          <b>{{ draft.work_id ? `正式作品 #${draft.work_id}` : '新内容' }}</b>
        </div>
        <div class="row-actions">
          <button type="button" title="继续编辑" @click="router.push(`/rpdb/drafts/${draft.id}/edit`)"><i class="ri-edit-line"></i><span>编辑</span></button>
          <button type="button" class="danger" title="删除" @click="removeDraft(draft)"><i class="ri-delete-bin-line"></i><span>删除</span></button>
        </div>
      </article>
    </section>
  </main>
</template>

<style scoped>
.uploads-page{max-width:1260px;margin:auto;color:var(--color-text-main)}.uploads-header{display:flex;align-items:flex-end;justify-content:space-between;gap:20px;padding-bottom:18px;border-bottom:1px solid var(--color-border)}.back-link{display:inline-flex;align-items:center;gap:5px;padding:0;border:0;background:transparent;color:var(--color-accent);font-size:12px}.uploads-header h1{margin:10px 0 4px;font-size:28px}.uploads-header p{margin:0;color:var(--color-text-secondary);font-size:13px}.create-button{display:inline-flex;min-height:40px;align-items:center;gap:6px;padding:0 15px;border:0;border-radius:7px;background:var(--btn-primary-bg);color:var(--btn-primary-text);font-weight:800}.summary-strip{display:grid;grid-template-columns:repeat(5,1fr);margin:16px 0;border:1px solid var(--color-border);border-radius:8px;background:var(--color-panel-bg)}.summary-strip div{display:grid;justify-items:center;gap:3px;padding:14px;border-right:1px solid var(--color-border)}.summary-strip div:last-child{border-right:0}.summary-strip b{font-size:20px}.summary-strip span{color:var(--color-text-secondary);font-size:10px}.status-tabs{display:flex;gap:6px;margin-bottom:12px;border-bottom:1px solid var(--color-border)}.status-tabs button{display:inline-flex;min-height:38px;align-items:center;gap:7px;padding:0 12px;border:0;border-bottom:2px solid transparent;background:transparent;color:var(--color-text-secondary);font-weight:700}.status-tabs button.active{border-bottom-color:var(--color-accent);color:var(--color-text-main)}.status-tabs b{display:grid;min-width:18px;height:18px;place-items:center;border-radius:9px;background:var(--tag-bg);font-size:9px}.uploads-list{display:grid;border:1px solid var(--color-border);border-radius:8px;background:var(--color-panel-bg)}.upload-row{display:grid;grid-template-columns:150px minmax(260px,1fr) minmax(150px,190px) auto;gap:16px;align-items:center;padding:14px}.upload-row+ .upload-row{border-top:1px solid var(--color-border)}.upload-cover{display:grid;width:150px;aspect-ratio:16/10;place-items:center;overflow:hidden;border-radius:6px;background:var(--color-card-bg);color:var(--icon-color);font-size:28px}.upload-cover img{width:100%;height:100%;object-fit:cover}.upload-main{min-width:0}.upload-heading{display:flex;align-items:center;justify-content:space-between;gap:10px}.upload-heading>div{display:flex;gap:6px}.type-label,.status-label{padding:3px 6px;border-radius:4px;background:var(--tag-bg);color:var(--tag-text);font-size:9px;font-weight:800}.status-label.published,.status-label.approved{color:var(--color-success,#2f855a)}.status-label.pending{color:var(--color-warning,#a06010)}.status-label.rejected{color:var(--color-danger,#b83232)}.upload-heading time{color:var(--color-text-muted);font-size:10px}.upload-main h2{margin:8px 0 4px;overflow:hidden;font-size:17px;text-overflow:ellipsis;white-space:nowrap}.upload-main p{display:-webkit-box;overflow:hidden;margin:0;color:var(--color-text-secondary);font-size:11px;line-height:1.5;-webkit-box-orient:vertical;-webkit-line-clamp:2}.upload-metrics{display:flex;gap:12px;margin-top:9px;color:var(--color-text-secondary);font-size:10px}.upload-metrics span{display:inline-flex;align-items:center;gap:3px}.visibility-control{display:grid;gap:8px}.visibility-control label{display:grid;gap:5px}.visibility-control label>span{color:var(--color-text-secondary);font-size:10px;font-weight:700}.visibility-control select{width:100%;height:34px;padding:0 8px;border:1px solid var(--input-border);border-radius:6px;background:var(--input-bg);color:var(--color-text-main)}.row-actions{display:flex;gap:5px}.row-actions button{display:grid;width:44px;height:42px;place-items:center;gap:1px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);color:var(--color-text-secondary)}.row-actions button:hover{border-color:var(--color-accent);color:var(--color-accent)}.row-actions button.danger:hover{border-color:var(--color-danger,#b83232);color:var(--color-danger,#b83232)}.row-actions button:disabled{opacity:.5}.row-actions i{font-size:15px}.row-actions span{font-size:8px}.page-state{display:grid;min-height:280px;place-items:center;align-content:center;gap:10px;color:var(--color-text-secondary)}.page-state>i{font-size:36px}.page-state h2{margin:0;font-size:18px}.page-state button{padding:9px 12px;border:0;border-radius:6px;background:var(--btn-primary-bg);color:var(--btn-primary-text)}.spin{animation:spin 1s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}@media(max-width:980px){.upload-row{grid-template-columns:120px minmax(220px,1fr) 160px}.upload-cover{width:120px}.row-actions{grid-column:2/-1;justify-content:flex-end}}@media(max-width:700px){.uploads-header{align-items:flex-start;flex-direction:column}.summary-strip{grid-template-columns:repeat(3,1fr)}.summary-strip div:nth-child(3){border-right:0}.upload-row{grid-template-columns:90px minmax(0,1fr)}.upload-cover{width:90px}.visibility-control,.row-actions{grid-column:1/-1}.status-tabs{overflow-x:auto}}
.visibility-guilds{display:flex!important;flex-wrap:wrap;gap:5px}.visibility-guilds label{display:inline-flex;align-items:center;gap:4px;padding:5px 7px;border:1px solid var(--color-border);border-radius:6px;background:var(--color-card-bg);font-size:9px;cursor:pointer}.visibility-guilds label.selected{border-color:var(--color-accent);background:var(--tag-bg)}.visibility-guilds input{width:auto}
.draft-list{margin-bottom:12px}.draft-list .visibility-control>span{color:var(--color-text-secondary);font-size:10px;font-weight:700}.draft-list .visibility-control>b{font-size:12px}
</style>
