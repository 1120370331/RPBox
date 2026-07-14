<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@shared/stores/user'
import { listPosts, POST_CATEGORIES, type PostWithAuthor } from '@/api/post'

type SaveState = 'idle' | 'saving' | 'saved' | 'error'

const props = defineProps<{
  currentDraftId?: number | null
  saveState?: SaveState
  refreshKey?: number
}>()

const emit = defineEmits<{
  select: [id: number]
  create: []
  delete: [post: PostWithAuthor]
}>()

const { t, locale } = useI18n()
const userStore = useUserStore()
const open = ref(false)
const loading = ref(false)
const drafts = ref<PostWithAuthor[]>([])
const saveLabel = computed(() => t(`community.editor.drafts.${props.saveState || 'idle'}`))

function formatDate(value: string) {
  return new Intl.DateTimeFormat(locale.value, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function categoryLabel(category: string) {
  return t(`community.categories.${category}`, POST_CATEGORIES.find((item) => item.value === category)?.label || category)
}

async function loadDrafts() {
  const authorId = userStore.user?.id
  if (!authorId) return
  loading.value = true
  try {
    const result = await listPosts({ author_id: authorId, status: 'draft', sort: 'updated_at', order: 'desc', page: 1, page_size: 100 })
    drafts.value = result.posts || []
  } catch (error) {
    console.error('Failed to load post drafts', error)
  } finally {
    loading.value = false
  }
}

function selectDraft(id: number) {
  open.value = false
  emit('select', id)
}

watch(open, (value) => { if (value) void loadDrafts() })
watch(() => props.refreshKey, loadDrafts)
</script>

<template>
  <button type="button" class="draft-header-btn" :aria-label="$t('community.editor.drafts.title')" @click="open = true">
    <i class="ri-draft-line" />
    <span>{{ saveLabel }}</span>
  </button>

  <div v-if="open" class="draft-mask" @click.self="open = false">
    <section class="draft-sheet">
      <header>
        <div>
          <h2>{{ $t('community.editor.drafts.title') }}</h2>
          <p>{{ $t('community.editor.drafts.count', { count: drafts.length }) }}</p>
        </div>
        <button type="button" class="icon-btn" :aria-label="$t('community.editor.drafts.close')" @click="open = false"><i class="ri-close-line" /></button>
      </header>

      <button type="button" class="new-draft" @click="open = false; emit('create')">
        <i class="ri-add-line" />
        {{ $t('community.editor.drafts.new') }}
      </button>

      <div class="draft-list">
        <div v-if="loading" class="draft-empty">{{ $t('community.editor.drafts.loading') }}</div>
        <div v-else-if="drafts.length === 0" class="draft-empty">{{ $t('community.editor.drafts.empty') }}</div>
        <template v-else>
          <article v-for="draft in drafts" :key="draft.id" :class="{ active: draft.id === currentDraftId }">
            <button type="button" class="draft-main" @click="selectDraft(draft.id)">
              <strong>{{ draft.title || $t('community.editor.drafts.untitled') }}</strong>
              <span>{{ categoryLabel(draft.category) }} · {{ formatDate(draft.updated_at) }}</span>
            </button>
            <button type="button" class="delete-btn" :aria-label="$t('community.editor.drafts.delete')" @click="open = false; emit('delete', draft)"><i class="ri-delete-bin-line" /></button>
          </article>
        </template>
      </div>
    </section>
  </div>
</template>

<style scoped>
.draft-header-btn{display:flex;align-items:center;gap:5px;min-width:76px;min-height:36px;padding:0 8px;border:0;background:transparent;color:var(--color-text-secondary);font-size:11px}.draft-header-btn i{font-size:18px}.draft-header-btn span{max-width:70px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.draft-mask{position:fixed;inset:0;z-index:1000;display:flex;align-items:flex-end;background:rgba(0,0,0,.46)}
.draft-sheet{width:100%;max-height:min(76vh,620px);padding:16px calc(14px + var(--safe-right,0px)) calc(18px + var(--safe-bottom,0px)) calc(14px + var(--safe-left,0px));overflow:hidden;border-radius:14px 14px 0 0;background:var(--color-panel-bg)}
.draft-sheet header{display:flex;align-items:center;justify-content:space-between}.draft-sheet h2{margin:0;color:var(--text-dark);font-size:18px}.draft-sheet p{margin:3px 0 0;color:var(--color-text-secondary);font-size:12px}.icon-btn{width:36px;height:36px;border:0;border-radius:50%;background:var(--input-bg);color:var(--text-dark);font-size:20px}
.new-draft{display:flex;width:100%;min-height:42px;align-items:center;justify-content:center;gap:6px;margin:14px 0 8px;border:1px solid var(--color-primary);border-radius:var(--radius-sm);background:transparent;color:var(--color-primary);font-weight:700}
.draft-list{max-height:calc(76vh - 150px);overflow-y:auto}.draft-list article{display:grid;grid-template-columns:minmax(0,1fr) 42px;align-items:center;border-bottom:1px solid var(--input-border)}.draft-list article.active{background:color-mix(in srgb,var(--color-primary) 9%,transparent)}
.draft-main{display:flex;min-width:0;flex-direction:column;align-items:flex-start;gap:4px;padding:12px 8px;border:0;background:transparent;text-align:left}.draft-main strong{max-width:100%;overflow:hidden;color:var(--text-dark);font-size:14px;text-overflow:ellipsis;white-space:nowrap}.draft-main span{color:var(--color-text-secondary);font-size:11px}.delete-btn{width:38px;height:38px;border:0;background:transparent;color:#b64b4b;font-size:18px}.draft-empty{display:flex;min-height:130px;align-items:center;justify-content:center;color:var(--color-text-secondary);font-size:13px}
</style>
