<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { dialog } from '@/composables/useDialog'
import { useToast } from '@/composables/useToast'
import {
  getPendingRPDBMedia,
  getPendingRPDBRevisions,
  getPendingRPDBWorks,
  reviewRPDBMedia,
  reviewRPDBRevision,
  reviewRPDBWork,
  type ModeratorStats,
  type RPDBMediaReviewItem,
  type RPDBRevisionReviewItem,
  type ReviewRequest,
} from '@/api/moderator'
import { resolveRPDBMediaURL, type RPDBWork } from '@/api/rpdb'

type ReviewQueue = 'works' | 'media' | 'revisions'

const props = defineProps<{ stats?: ModeratorStats | null }>()
const emit = defineEmits<{ reviewed: [] }>()
const router = useRouter()
const toast = useToast()

const activeQueue = ref<ReviewQueue>('works')
const loading = ref(false)
const works = ref<RPDBWork[]>([])
const media = ref<RPDBMediaReviewItem[]>([])
const revisions = ref<RPDBRevisionReviewItem[]>([])
const totals = reactive<Record<ReviewQueue, number>>({ works: 0, media: 0, revisions: 0 })
const commentDrafts = reactive<Record<string, string>>({})

const queueTabs = computed(() => [
  { value: 'works' as const, label: '作品投稿', icon: 'ri-archive-drawer-line', count: props.stats?.pending_rpdb_works ?? totals.works },
  { value: 'media' as const, label: '媒体素材', icon: 'ri-image-2-line', count: props.stats?.pending_rpdb_media ?? totals.media },
  { value: 'revisions' as const, label: '内容修订', icon: 'ri-draft-line', count: props.stats?.pending_rpdb_revisions ?? totals.revisions },
])

onMounted(loadActiveQueue)

async function selectQueue(queue: ReviewQueue) {
  if (activeQueue.value === queue) return
  activeQueue.value = queue
  await loadActiveQueue()
}

async function loadActiveQueue() {
  loading.value = true
  try {
    if (activeQueue.value === 'works') {
      const response = await getPendingRPDBWorks({ page: 1, page_size: 50 })
      works.value = response.works || []
      totals.works = response.total || 0
    } else if (activeQueue.value === 'media') {
      const response = await getPendingRPDBMedia({ page: 1, page_size: 50 })
      media.value = response.media || []
      totals.media = response.total || 0
    } else {
      const response = await getPendingRPDBRevisions({ page: 1, page_size: 50 })
      revisions.value = response.revisions || []
      totals.revisions = response.total || 0
    }
  } catch (error) {
    console.error('加载 RP 数据库审核队列失败:', error)
    toast.error('加载 RP 数据库审核队列失败')
  } finally {
    loading.value = false
  }
}

async function reviewTarget(queue: ReviewQueue, id: number, action: ReviewRequest['action']) {
  const actionText = action === 'approve' ? '通过' : '拒绝'
  const subject = queue === 'works' ? '作品投稿' : queue === 'media' ? '媒体素材' : '内容修订'
  const confirmed = await dialog.confirm({
    title: `${actionText}${subject}`,
    message: action === 'approve' && queue === 'revisions'
      ? '通过后将立即覆盖当前公开版本并提升版本号，确定继续吗？'
      : `确定要${actionText}这条${subject}吗？`,
    type: action === 'approve' ? 'success' : 'warning',
    confirmText: actionText,
    cancelText: '取消',
  })
  if (!confirmed) return

  const key = `${queue}-${id}`
  const payload: ReviewRequest = { action, comment: commentDrafts[key]?.trim() || '' }
  try {
    if (queue === 'works') await reviewRPDBWork(id, payload)
    else if (queue === 'media') await reviewRPDBMedia(id, payload)
    else await reviewRPDBRevision(id, payload)
    delete commentDrafts[key]
    await loadActiveQueue()
    emit('reviewed')
    toast.success(`${actionText}成功`)
  } catch (error) {
    console.error('RP 数据库审核失败:', error)
    toast.error(`审核失败: ${(error as Error).message}`)
  }
}

function openWork(workID: number) {
  router.push({ name: 'rpdb-detail', params: { id: workID } })
}

function formatDate(value?: string) {
  if (!value) return '未知时间'
  return new Date(value).toLocaleString('zh-CN')
}

function typeLabel(type: RPDBWork['type']) {
  return type === 'item_showcase' ? '魔兽物品' : type === 'transmog' ? '幻化方案' : type === 'musician_midi' ? 'Musician MIDI' : '家宅分享'
}

function parseRevisionPayload(revision: RPDBRevisionReviewItem): Record<string, unknown> {
  try {
    return JSON.parse(revision.payload) as Record<string, unknown>
  } catch {
    return {}
  }
}

function revisionTitle(revision: RPDBRevisionReviewItem) {
  const title = parseRevisionPayload(revision).title
  return typeof title === 'string' && title.trim() ? title : `作品 #${revision.work_id} 修订`
}

function revisionFields(revision: RPDBRevisionReviewItem) {
  return Object.keys(parseRevisionPayload(revision)).filter(field => field !== 'change_summary')
}
</script>

<template>
  <section data-testid="rpdb-moderation-panel" class="rpdb-review-panel">
    <header class="rpdb-review-header">
      <div>
        <span class="rpdb-review-kicker">RP DATABASE MODERATION</span>
        <h3>RP 数据库审核</h3>
        <p>集中处理作品投稿、媒体证据和已发布内容修订。</p>
      </div>
      <div class="rpdb-review-tabs" role="tablist" aria-label="RP 数据库审核队列">
        <button
          v-for="tab in queueTabs"
          :key="tab.value"
          :data-testid="`rpdb-moderation-${tab.value}`"
          :class="{ active: activeQueue === tab.value }"
          type="button"
          @click="selectQueue(tab.value)"
        >
          <i :class="tab.icon"></i>
          <span>{{ tab.label }}</span>
          <b v-if="tab.count > 0">{{ tab.count }}</b>
        </button>
      </div>
    </header>

    <div v-if="loading" class="rpdb-review-state">
      <i class="ri-loader-4-line loading-spinner"></i>
      <span>加载审核队列...</span>
    </div>

    <template v-else-if="activeQueue === 'works'">
      <div v-if="works.length === 0" class="rpdb-review-state">
        <i class="ri-checkbox-circle-line"></i>
        <span>暂无待审核作品</span>
      </div>
      <div v-else class="rpdb-review-list">
        <article v-for="work in works" :key="work.id" class="rpdb-review-card">
          <div class="rpdb-review-card__main">
            <div class="rpdb-review-card__eyebrow">
              <span>{{ typeLabel(work.type) }}</span>
              <span>版本 {{ work.version }}</span>
            </div>
            <h4>{{ work.title }}</h4>
            <p>{{ work.summary || work.effect_description || '作者未填写摘要。' }}</p>
            <div class="rpdb-review-meta">
              <span><i class="ri-user-line"></i>{{ work.author_name || `用户 #${work.author_id}` }}</span>
              <span><i class="ri-time-line"></i>{{ formatDate(work.created_at) }}</span>
            </div>
          </div>
          <div class="rpdb-review-card__aside">
            <button class="review-link" type="button" @click="openWork(work.id)">
              <i class="ri-external-link-line"></i>查看详情
            </button>
            <input v-model="commentDrafts[`works-${work.id}`]" type="text" placeholder="审核备注（可选）" />
            <div class="rpdb-review-actions">
              <button class="approve" type="button" @click="reviewTarget('works', work.id, 'approve')"><i class="ri-check-line"></i>通过</button>
              <button class="reject" type="button" @click="reviewTarget('works', work.id, 'reject')"><i class="ri-close-line"></i>拒绝</button>
            </div>
          </div>
        </article>
      </div>
    </template>

    <template v-else-if="activeQueue === 'media'">
      <div v-if="media.length === 0" class="rpdb-review-state">
        <i class="ri-checkbox-circle-line"></i>
        <span>暂无待审核媒体</span>
      </div>
      <div v-else class="rpdb-review-list">
        <article v-for="item in media" :key="item.id" class="rpdb-review-card media-card">
          <div class="rpdb-media-preview">
            <img v-if="item.type === 'image' || item.type === 'gif'" :src="resolveRPDBMediaURL(item.thumbnail_url || item.url)" :alt="item.caption || '待审核媒体'" />
            <i v-else :class="item.type === 'video' ? 'ri-video-line' : 'ri-links-line'"></i>
          </div>
          <div class="rpdb-review-card__main">
            <div class="rpdb-review-card__eyebrow">
              <span>{{ item.type.toUpperCase() }}</span>
              <span>作品 #{{ item.work_id }}</span>
            </div>
            <h4>{{ item.caption || '未命名媒体素材' }}</h4>
            <p class="media-url">{{ item.url }}</p>
            <div class="rpdb-review-meta">
              <span><i class="ri-user-line"></i>上传者 #{{ item.author_id || '未知' }}</span>
              <span><i class="ri-time-line"></i>{{ formatDate(item.created_at) }}</span>
            </div>
          </div>
          <div class="rpdb-review-card__aside">
            <button class="review-link" type="button" @click="openWork(item.work_id)">
              <i class="ri-external-link-line"></i>查看原作
            </button>
            <input v-model="commentDrafts[`media-${item.id}`]" type="text" placeholder="审核备注（可选）" />
            <div class="rpdb-review-actions">
              <button class="approve" type="button" @click="reviewTarget('media', item.id, 'approve')"><i class="ri-check-line"></i>通过</button>
              <button class="reject" type="button" @click="reviewTarget('media', item.id, 'reject')"><i class="ri-close-line"></i>拒绝</button>
            </div>
          </div>
        </article>
      </div>
    </template>

    <template v-else>
      <div v-if="revisions.length === 0" class="rpdb-review-state">
        <i class="ri-checkbox-circle-line"></i>
        <span>暂无待审核修订</span>
      </div>
      <div v-else class="rpdb-review-list">
        <article v-for="revision in revisions" :key="revision.id" class="rpdb-review-card">
          <div class="rpdb-review-card__main">
            <div class="rpdb-review-card__eyebrow">
              <span>作品 #{{ revision.work_id }}</span>
              <span>基于版本 {{ revision.base_version }}</span>
            </div>
            <h4>{{ revisionTitle(revision) }}</h4>
            <p>{{ revision.change_summary || '作者未填写修订说明。' }}</p>
            <div class="revision-fields">
              <span v-for="field in revisionFields(revision)" :key="field">{{ field }}</span>
            </div>
            <div class="rpdb-review-meta">
              <span><i class="ri-user-line"></i>提交者 #{{ revision.proposer_id }}</span>
              <span><i class="ri-time-line"></i>{{ formatDate(revision.created_at) }}</span>
            </div>
          </div>
          <div class="rpdb-review-card__aside">
            <button class="review-link" type="button" @click="openWork(revision.work_id)">
              <i class="ri-external-link-line"></i>查看当前版本
            </button>
            <input v-model="commentDrafts[`revisions-${revision.id}`]" type="text" placeholder="审核备注（可选）" />
            <div class="rpdb-review-actions">
              <button class="approve" type="button" @click="reviewTarget('revisions', revision.id, 'approve')"><i class="ri-check-line"></i>应用修订</button>
              <button class="reject" type="button" @click="reviewTarget('revisions', revision.id, 'reject')"><i class="ri-close-line"></i>拒绝</button>
            </div>
          </div>
        </article>
      </div>
    </template>
  </section>
</template>

<style scoped>
.rpdb-review-panel{display:grid;gap:16px}.rpdb-review-header{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;padding:20px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.rpdb-review-kicker{color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.14em}.rpdb-review-header h3{margin:5px 0 4px;color:var(--color-text-main);font-size:21px}.rpdb-review-header p{margin:0;color:var(--color-text-secondary);font-size:13px}.rpdb-review-tabs{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:8px}.rpdb-review-tabs button{display:inline-flex;align-items:center;gap:6px;min-height:38px;padding:0 12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-card-bg);color:var(--color-text-secondary);cursor:pointer}.rpdb-review-tabs button.active{border-color:var(--color-accent);background:var(--btn-secondary-bg);color:var(--btn-secondary-text)}.rpdb-review-tabs b{display:grid;min-width:19px;height:19px;place-items:center;border-radius:10px;background:var(--color-accent);color:var(--color-accent-contrast);font-size:10px}.rpdb-review-state{display:flex;min-height:180px;align-items:center;justify-content:center;gap:8px;border:1px dashed var(--color-border);border-radius:12px;color:var(--color-text-secondary)}.rpdb-review-state i{font-size:22px}.rpdb-review-list{display:grid;gap:12px}.rpdb-review-card{display:grid;grid-template-columns:minmax(0,1fr) minmax(220px,280px);gap:18px;padding:18px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-panel-bg);box-shadow:var(--shadow-sm)}.rpdb-review-card.media-card{grid-template-columns:120px minmax(0,1fr) minmax(220px,280px)}.rpdb-review-card__main{min-width:0}.rpdb-review-card__eyebrow,.rpdb-review-meta,.revision-fields{display:flex;flex-wrap:wrap;align-items:center;gap:8px}.rpdb-review-card__eyebrow span,.revision-fields span{padding:3px 7px;border-radius:4px;background:var(--tag-bg);color:var(--tag-text);font-size:10px;font-weight:700}.rpdb-review-card h4{margin:9px 0 5px;color:var(--color-text-main);font-size:17px}.rpdb-review-card p{display:-webkit-box;overflow:hidden;margin:0;color:var(--color-text-secondary);font-size:13px;line-height:1.6;-webkit-box-orient:vertical;-webkit-line-clamp:2}.rpdb-review-meta{margin-top:12px;color:var(--color-text-muted);font-size:11px}.rpdb-review-meta span{display:inline-flex;align-items:center;gap:4px}.revision-fields{margin-top:10px}.rpdb-review-card__aside{display:flex;flex-direction:column;justify-content:center;gap:10px}.rpdb-review-card__aside input{width:100%;min-height:38px;padding:0 10px;border:1px solid var(--color-border);border-radius:7px;background:var(--input-bg);color:var(--color-text-main);outline:none}.rpdb-review-card__aside input:focus{border-color:var(--color-accent)}.review-link{align-self:flex-start;border:0;background:transparent;color:var(--color-accent);font-size:12px;font-weight:700;cursor:pointer}.rpdb-review-actions{display:flex;gap:8px}.rpdb-review-actions button{display:inline-flex;min-height:36px;flex:1;align-items:center;justify-content:center;gap:5px;border:0;border-radius:7px;color:#fff;font-weight:700;cursor:pointer}.rpdb-review-actions .approve{background:var(--color-success,#2f855a)}.rpdb-review-actions .reject{background:var(--btn-danger-bg,#c0392b)}.rpdb-media-preview{display:grid;min-height:100px;place-items:center;overflow:hidden;border:1px solid var(--color-border);border-radius:9px;background:var(--color-card-bg);color:var(--icon-color);font-size:30px}.rpdb-media-preview img{width:100%;height:100%;object-fit:cover}.media-url{word-break:break-all}@media(max-width:900px){.rpdb-review-header{flex-direction:column}.rpdb-review-tabs{justify-content:flex-start}.rpdb-review-card,.rpdb-review-card.media-card{grid-template-columns:1fr}.rpdb-media-preview{height:180px}.rpdb-review-card__aside{max-width:none}}
</style>
