<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  createCharacterCard,
  getCharacterCardSources,
  type CharacterCardSource,
  type CharacterCardSourceType,
} from '@/api/characterCard'

const router = useRouter()
const { t, locale } = useI18n()
const toast = useToastStore()

const sources = ref<CharacterCardSource[]>([])
const selectedSourceKey = ref('')
const loading = ref(true)
const loadError = ref('')
const creatingType = ref<CharacterCardSourceType | null>(null)

const sourceGroups = computed(() => {
  const groups = new Map<string, CharacterCardSource[]>()
  for (const source of sources.value) {
    const accountId = source.account_id?.trim() || t('characterCards.newPage.unknownAccount')
    const group = groups.get(accountId) || []
    group.push(source)
    groups.set(accountId, group)
  }
  return Array.from(groups, ([accountId, profiles]) => ({ accountId, profiles }))
})

const selectedSource = computed(() => (
  sources.value.find((source) => sourceKey(source) === selectedSourceKey.value) || null
))

function sourceKey(source: CharacterCardSource) {
  return `${source.backup_id}:${source.profile_id}`
}

function sourceName(source: CharacterCardSource) {
  if (source.display_name?.trim()) return source.display_name.trim()
  const composed = [source.first_name, source.last_name]
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(' ')
  return composed || source.profile_name?.trim() || t('characterCards.newPage.unnamed')
}

function sourceMeta(source: CharacterCardSource) {
  return [source.race, source.class, source.title]
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(' · ') || t('characterCards.newPage.identityPending')
}

function formatDate(value?: string) {
  if (!value) return t('characterCards.newPage.unknownTime')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('characterCards.newPage.unknownTime')
  return date.toLocaleDateString(locale.value, { year: 'numeric', month: '2-digit', day: '2-digit' })
}

async function loadSources() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getCharacterCardSources()
    sources.value = response.sources
  } catch (error: unknown) {
    sources.value = []
    loadError.value = error instanceof Error
      ? error.message
      : t('characterCards.newPage.loadFailed')
  } finally {
    loading.value = false
  }
}

async function createDraft(sourceType: CharacterCardSourceType) {
  if (creatingType.value) return
  const source = selectedSource.value
  if (sourceType === 'backup' && !source) {
    toast.warning(t('characterCards.newPage.selectBackup'))
    return
  }

  creatingType.value = sourceType
  try {
    const card = await createCharacterCard(sourceType === 'backup' && source
      ? {
          source_type: 'backup',
          source_backup_id: source.backup_id,
          source_profile_id: source.profile_id,
        }
      : { source_type: 'blank' })
    toast.success(t(sourceType === 'backup'
      ? 'characterCards.newPage.createdFromBackup'
      : 'characterCards.newPage.createdBlank'))
    await router.replace({ name: 'character-card-edit', params: { id: card.id } })
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.newPage.createFailed'))
  } finally {
    creatingType.value = null
  }
}

onMounted(() => void loadSources())
</script>

<template>
  <main class="sub-page character-source-page">
    <header class="sub-header">
      <button
        type="button"
        class="back-btn"
        :aria-label="t('characterCards.newPage.backProfile')"
        @click="router.push({ name: 'my-character-cards' })"
      >
        <i class="ri-arrow-left-line" aria-hidden="true" />
      </button>
      <div class="header-copy">
        <span>{{ t('characterCards.newPage.kicker') }}</span>
        <h1>{{ t('characterCards.newPage.title') }}</h1>
      </div>
    </header>

    <div class="sub-body source-body">
      <p class="page-lead">{{ t('characterCards.newPage.subtitle') }}</p>

      <aside class="source-note">
        <i class="ri-cloud-line" aria-hidden="true" />
        <div>
          <strong>{{ t('characterCards.newPage.syncTitle') }}</strong>
          <p>{{ t('characterCards.newPage.syncBody') }}</p>
        </div>
      </aside>

      <section class="source-card" aria-labelledby="backup-source-heading">
        <header class="section-heading">
          <span class="section-index">A</span>
          <div>
            <h2 id="backup-source-heading">{{ t('characterCards.newPage.backupTitle') }}</h2>
            <p>{{ t('characterCards.newPage.backupBody') }}</p>
          </div>
        </header>

        <div v-if="loading" class="source-state" role="status">
          <i class="ri-loader-4-line spin" aria-hidden="true" />
          {{ t('characterCards.newPage.loading') }}
        </div>

        <div v-else-if="loadError" class="source-state source-state--error" role="alert">
          <i class="ri-error-warning-line" aria-hidden="true" />
          <div>
            <strong>{{ t('characterCards.newPage.loadErrorTitle') }}</strong>
            <span>{{ loadError }}</span>
          </div>
          <button type="button" @click="loadSources">{{ t('characterCards.common.reload') }}</button>
        </div>

        <div v-else-if="sourceGroups.length === 0" class="source-state">
          <i class="ri-archive-drawer-line" aria-hidden="true" />
          <div>
            <strong>{{ t('characterCards.newPage.emptyTitle') }}</strong>
            <span>{{ t('characterCards.newPage.emptyBody') }}</span>
          </div>
        </div>

        <div v-else class="account-groups">
          <section v-for="group in sourceGroups" :key="group.accountId" class="account-group">
            <header>
              <strong>{{ group.accountId }}</strong>
              <span>{{ t('characterCards.newPage.profileCount', { count: group.profiles.length }) }}</span>
            </header>
            <div
              class="profile-options"
              role="radiogroup"
              :aria-label="t('characterCards.newPage.groupAria', { account: group.accountId })"
            >
              <button
                v-for="source in group.profiles"
                :key="sourceKey(source)"
                type="button"
                class="profile-option"
                :class="{ selected: selectedSourceKey === sourceKey(source) }"
                role="radio"
                :aria-checked="selectedSourceKey === sourceKey(source)"
                data-testid="character-card-source"
                @click="selectedSourceKey = sourceKey(source)"
              >
                <span class="profile-icon"><i class="ri-user-star-line" aria-hidden="true" /></span>
                <span class="profile-copy">
                  <strong>{{ sourceName(source) }}</strong>
                  <span>{{ sourceMeta(source) }}</span>
                  <small>{{ t('characterCards.newPage.backupUpdated', { date: formatDate(source.backup_updated_at) }) }}</small>
                </span>
                <i class="ri-check-line option-check" aria-hidden="true" />
              </button>
            </div>
          </section>
        </div>

        <button
          type="button"
          class="source-action source-action--primary"
          data-testid="create-from-backup"
          :disabled="!selectedSource || Boolean(creatingType)"
          @click="createDraft('backup')"
        >
          <i :class="creatingType === 'backup' ? 'ri-loader-4-line spin' : 'ri-file-copy-2-line'" aria-hidden="true" />
          {{ creatingType === 'backup'
            ? t('characterCards.newPage.creating')
            : t('characterCards.newPage.createFromSelected') }}
        </button>
      </section>

      <section class="source-card source-card--blank" aria-labelledby="blank-source-heading">
        <header class="section-heading">
          <span class="section-index">B</span>
          <div>
            <h2 id="blank-source-heading">{{ t('characterCards.newPage.blankTitle') }}</h2>
            <p>{{ t('characterCards.newPage.blankBody') }}</p>
          </div>
        </header>
        <div class="blank-sheet">
          <i class="ri-quill-pen-line" aria-hidden="true" />
          <div>
            <strong>{{ t('characterCards.newPage.blankSheetTitle') }}</strong>
            <p>{{ t('characterCards.newPage.blankSheetBody') }}</p>
          </div>
        </div>
        <button
          type="button"
          class="source-action"
          data-testid="create-blank-card"
          :disabled="Boolean(creatingType)"
          @click="createDraft('blank')"
        >
          <i :class="creatingType === 'blank' ? 'ri-loader-4-line spin' : 'ri-add-line'" aria-hidden="true" />
          {{ creatingType === 'blank'
            ? t('characterCards.newPage.creating')
            : t('characterCards.newPage.createBlank') }}
        </button>
      </section>
    </div>
  </main>
</template>

<style scoped>
.character-source-page { min-height: 100dvh; }
.sub-header { align-items: flex-start; }
.header-copy { display: grid; gap: 2px; min-width: 0; padding-top: 3px; }
.header-copy span { color: var(--color-accent); font: 800 9px/1.2 ui-monospace, monospace; letter-spacing: .13em; text-transform: uppercase; }
.header-copy h1 { font-family: Georgia, 'Noto Serif SC', serif; font-size: 19px; line-height: 1.25; }
.source-body { display: grid; gap: 14px; max-width: 720px; margin: 0 auto; }
.page-lead { color: var(--color-text-secondary); font-size: 13px; line-height: 1.65; }
.source-note { display: flex; gap: 11px; padding: 13px; border: 1px solid color-mix(in srgb, var(--color-accent) 35%, var(--color-border)); border-radius: 14px; background: color-mix(in srgb, var(--color-accent) 8%, var(--color-card-bg)); }
.source-note > i { margin-top: 2px; color: var(--color-accent); font-size: 21px; }
.source-note strong { color: var(--color-primary); font-size: 13px; }
.source-note p { margin-top: 4px; color: var(--color-text-secondary); font-size: 11px; line-height: 1.55; }
.source-card { display: grid; gap: 14px; padding: 16px; border: 1px solid var(--color-border); border-radius: 18px; background: var(--color-card-bg); box-shadow: var(--shadow-sm); }
.section-heading { display: grid; grid-template-columns: 32px minmax(0, 1fr); gap: 10px; }
.section-index { display: grid; width: 32px; height: 32px; place-items: center; border-radius: 50%; background: var(--color-primary); color: var(--text-light); font: 800 12px/1 ui-monospace, monospace; }
.section-heading h2 { color: var(--color-primary); font-family: Georgia, 'Noto Serif SC', serif; font-size: 17px; }
.section-heading p { margin-top: 4px; color: var(--color-text-secondary); font-size: 12px; line-height: 1.5; }
.source-state { display: flex; min-height: 72px; align-items: center; gap: 10px; padding: 12px; border: 1px dashed var(--color-border); border-radius: 12px; color: var(--color-text-secondary); font-size: 12px; }
.source-state > i { font-size: 20px; }
.source-state > div { display: grid; flex: 1; gap: 3px; }
.source-state button { min-height: 44px; padding: 0 12px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--input-bg); color: var(--color-primary); }
.source-state--error { color: var(--btn-danger-bg); }
.account-groups { display: grid; gap: 14px; }
.account-group { display: grid; gap: 8px; }
.account-group > header { display: flex; align-items: center; justify-content: space-between; gap: 10px; color: var(--color-primary); font-size: 11px; }
.account-group > header span { color: var(--color-text-secondary); font-weight: 500; }
.profile-options { display: grid; gap: 8px; }
.profile-option { display: grid; width: 100%; min-height: 72px; grid-template-columns: 40px minmax(0, 1fr) 20px; align-items: center; gap: 10px; padding: 10px; border: 1px solid var(--color-border); border-radius: 12px; background: var(--input-bg); color: var(--color-text-main); text-align: left; }
.profile-option.selected { border-color: var(--color-accent); background: color-mix(in srgb, var(--color-accent) 9%, var(--input-bg)); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 12%, transparent); }
.profile-icon { display: grid; width: 40px; height: 40px; place-items: center; border-radius: 10px; background: var(--color-primary-light); color: var(--color-secondary); font-size: 20px; }
.profile-copy { display: grid; min-width: 0; gap: 2px; }
.profile-copy strong, .profile-copy span, .profile-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.profile-copy strong { font-size: 13px; }
.profile-copy span { color: var(--color-text-secondary); font-size: 11px; }
.profile-copy small { color: var(--color-text-muted); font-size: 10px; }
.option-check { color: var(--color-accent); opacity: 0; }
.selected .option-check { opacity: 1; }
.source-action { display: inline-flex; width: 100%; min-height: 48px; align-items: center; justify-content: center; gap: 7px; padding: 0 14px; border: 1px solid var(--color-primary); border-radius: 12px; background: var(--input-bg); color: var(--color-primary); font-weight: 800; }
.source-action--primary { background: var(--color-primary); color: var(--text-light); }
.source-action:disabled { cursor: not-allowed; opacity: .52; }
.blank-sheet { display: flex; gap: 12px; align-items: flex-start; padding: 14px; border: 1px dashed var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-background) 45%, var(--color-card-bg)); }
.blank-sheet > i { color: var(--color-accent); font-size: 25px; }
.blank-sheet strong { color: var(--color-primary); font-size: 13px; }
.blank-sheet p { margin-top: 4px; color: var(--color-text-secondary); font-size: 11px; line-height: 1.55; }
.spin { animation: source-spin .9s linear infinite; }
@keyframes source-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .spin { animation: none; } }
</style>
