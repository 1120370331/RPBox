<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPosts, POST_CATEGORIES, type PostWithAuthor } from '@/api/post'
import { useUserStore } from '@/stores/user'

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
const root = ref<HTMLElement | null>(null)
const open = ref(false)
const loading = ref(false)
const drafts = ref<PostWithAuthor[]>([])

const saveLabel = computed(() => t(`community.drafts.${props.saveState || 'idle'}`))

function formatDate(value: string) {
  return new Intl.DateTimeFormat(locale.value, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function categoryLabel(category: string) {
  const item = POST_CATEGORIES.find((entry) => entry.value === category)
  return item ? t(`community.categories.${item.value}`, item.label) : category
}

async function loadDrafts() {
  const authorId = userStore.user?.id
  if (!authorId) return
  loading.value = true
  try {
    const result = await listPosts({
      author_id: authorId,
      status: 'draft',
      sort: 'updated_at',
      order: 'desc',
      page: 1,
      page_size: 100,
    })
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

function createDraft() {
  open.value = false
  emit('create')
}

function handleDocumentClick(event: MouseEvent) {
  if (open.value && root.value && !root.value.contains(event.target as Node)) {
    open.value = false
  }
}

watch(() => props.refreshKey, loadDrafts)
watch(open, (value) => {
  if (value) void loadDrafts()
})

onMounted(() => document.addEventListener('mousedown', handleDocumentClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', handleDocumentClick))
</script>

<template>
  <div ref="root" class="draft-box">
    <button
      type="button"
      class="draft-trigger"
      :class="`is-${saveState || 'idle'}`"
      :aria-expanded="open"
      @click="open = !open"
    >
      <i class="ri-draft-line" />
      <span>{{ t('community.drafts.title') }}</span>
      <small>{{ saveLabel }}</small>
      <i class="ri-arrow-down-s-line caret" />
    </button>

    <div v-if="open" class="draft-popover">
      <header>
        <div>
          <strong>{{ t('community.drafts.title') }}</strong>
          <span>{{ t('community.drafts.count', { count: drafts.length }) }}</span>
        </div>
        <button type="button" class="new-draft" @click="createDraft">
          <i class="ri-add-line" />
          {{ t('community.drafts.new') }}
        </button>
      </header>

      <div class="draft-list">
        <div v-if="loading" class="draft-empty">{{ t('community.drafts.loading') }}</div>
        <div v-else-if="drafts.length === 0" class="draft-empty">
          <i class="ri-file-edit-line" />
          <span>{{ t('community.drafts.empty') }}</span>
        </div>
        <template v-else>
          <div
            v-for="draft in drafts"
            :key="draft.id"
            class="draft-row"
            :class="{ active: draft.id === currentDraftId }"
          >
            <button type="button" class="draft-main" @click="selectDraft(draft.id)">
              <strong>{{ draft.title || t('community.drafts.untitled') }}</strong>
              <span>{{ categoryLabel(draft.category) }} · {{ formatDate(draft.updated_at) }}</span>
            </button>
            <button
              type="button"
              class="delete-draft"
              :title="t('community.drafts.delete')"
              :aria-label="t('community.drafts.delete')"
            @click.stop="open = false; emit('delete', draft)"
            >
              <i class="ri-delete-bin-line" />
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.draft-box{position:relative;z-index:30;flex:0 0 auto}
.draft-trigger{
  display:inline-flex;
  align-items:center;
  gap:8px;
  min-height:40px;
  padding:0 14px;
  border:1px solid var(--color-border, #E5D4C1);
  border-radius:8px;
  background:var(--color-panel-bg, #fff);
  color:var(--color-primary, #4B3621);
  box-shadow:0 2px 10px rgba(75,54,33,.06);
  cursor:pointer;
  white-space:nowrap;
}
.draft-trigger>i:first-child{font-size:16px;color:var(--color-accent, #B87333)}
.draft-trigger>span{font-size:13px;font-weight:700}
.draft-trigger small{color:var(--color-text-secondary, #8D7B68);font-size:11px}
.draft-trigger.is-saving small{color:#b26a00}
.draft-trigger.is-saved small{color:#2d7a52}
.draft-trigger.is-error small{color:#b23a3a}
.draft-trigger .caret{margin-left:2px;color:var(--color-text-secondary, #8D7B68)}
.draft-popover{
  position:absolute;
  top:46px;
  right:0;
  width:min(400px, calc(100vw - 32px));
  overflow:hidden;
  border:1px solid var(--color-border, #E5D4C1);
  border-radius:10px;
  background:var(--color-panel-bg, #fff);
  box-shadow:0 18px 45px rgba(44,24,16,.16);
}
.draft-popover header{
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:12px;
  padding:14px;
  border-bottom:1px solid var(--color-border-light, #F1E4D7);
  background:var(--color-card-bg, #F5EFE7);
}
.draft-popover header>div{display:flex;flex-direction:column;gap:2px}
.draft-popover header strong{color:var(--color-text-main, #2C1810);font-size:14px}
.draft-popover header span{color:var(--color-text-secondary, #8D7B68);font-size:11px}
.new-draft{
  display:inline-flex;
  align-items:center;
  gap:4px;
  min-height:32px;
  padding:0 10px;
  border:1px solid var(--color-accent, #B87333);
  border-radius:6px;
  background:var(--color-panel-bg, #fff);
  color:var(--color-secondary, #804030);
  font-size:12px;
  font-weight:700;
  cursor:pointer;
}
.draft-list{max-height:340px;overflow-y:auto;padding:6px}
.draft-row{display:grid;grid-template-columns:minmax(0,1fr) 34px;align-items:center;border-radius:6px}
.draft-row:hover{background:var(--color-card-bg-hover, #FFF5E6)}
.draft-row.active{background:color-mix(in srgb, var(--color-accent, #B87333) 12%, #fff)}
.draft-row.active .draft-main strong:before{content:'';display:inline-block;width:6px;height:6px;margin-right:7px;border-radius:50%;background:var(--color-accent, #B87333);vertical-align:2px}
.draft-main{display:flex;min-width:0;flex-direction:column;align-items:flex-start;gap:4px;padding:10px;border:0;background:transparent;color:var(--color-text-main, #2C1810);text-align:left;cursor:pointer}
.draft-main strong{max-width:100%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px}
.draft-main span{color:var(--color-text-secondary, #8D7B68);font-size:11px}
.delete-draft{width:30px;height:30px;border:0;border-radius:5px;background:transparent;color:#a48d77;cursor:pointer}
.delete-draft:hover{background:#f7e6e3;color:#b23a3a}
.draft-empty{display:flex;min-height:110px;align-items:center;justify-content:center;flex-direction:column;gap:7px;color:var(--color-text-secondary, #8D7B68);font-size:12px}
.draft-empty i{font-size:22px}
@media(max-width:620px){
  .draft-trigger small{display:none}
  .draft-popover{right:-8px}
}
</style>
