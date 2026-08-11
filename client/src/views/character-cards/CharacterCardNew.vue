<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  createCharacterCard,
  getCharacterCardSources,
  type CharacterCardSource,
  type CharacterCardSourceType,
} from '@/api/characterCard'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const { t, locale } = useI18n()
const toast = useToastStore()
const userStore = useUserStore()

const sources = ref<CharacterCardSource[]>([])
const selectedSourceKey = ref('')
const loading = ref(true)
const loadError = ref('')
const creatingType = ref<CharacterCardSourceType | null>(null)

const sourceGroups = computed(() => {
  const groups = new Map<string, CharacterCardSource[]>()
  for (const source of sources.value) {
    const key = source.account_id || t('characterCards.newPage.unknownAccount')
    const group = groups.get(key) || []
    group.push(source)
    groups.set(key, group)
  }
  return Array.from(groups, ([accountId, profiles]) => ({ accountId, profiles }))
})

const selectedSource = computed(() => (
  sources.value.find((source) => sourceKey(source) === selectedSourceKey.value) || null
))

onMounted(() => void loadSources())

function sourceKey(source: CharacterCardSource) {
  return `${source.backup_id}:${source.profile_id}`
}

function sourceName(source: CharacterCardSource) {
  if (source.display_name?.trim()) return source.display_name
  const name = [source.first_name, source.last_name].map((part) => part?.trim()).filter(Boolean).join(' ')
  return name || source.profile_name || t('characterCards.newPage.unnamed')
}

function sourceMeta(source: CharacterCardSource) {
  return [source.race, source.class, source.title].map((part) => part?.trim()).filter(Boolean).join(' · ') || t('characterCards.newPage.identityPending')
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
    sources.value = response.sources || []
  } catch (error: unknown) {
    sources.value = []
    loadError.value = error instanceof Error ? error.message : t('characterCards.newPage.loadFailed')
  } finally {
    loading.value = false
  }
}

async function createFromBlank() {
  await createDraft('blank')
}

async function createFromBackup() {
  if (!selectedSource.value) {
    toast.warning(t('characterCards.newPage.selectBackup'))
    return
  }
  await createDraft('backup')
}

async function createDraft(sourceType: CharacterCardSourceType) {
  if (creatingType.value) return
  creatingType.value = sourceType
  try {
    const source = selectedSource.value
    const card = await createCharacterCard(sourceType === 'backup' && source
      ? {
          source_type: 'backup',
          source_backup_id: source.backup_id,
          source_profile_id: source.profile_id,
        }
      : { source_type: 'blank' })
    toast.success(t(sourceType === 'backup' ? 'characterCards.newPage.createdFromBackup' : 'characterCards.newPage.createdBlank'))
    await router.replace(`/character-cards/${card.id}/edit`)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : t('characterCards.newPage.createFailed'))
  } finally {
    creatingType.value = null
  }
}

function goBack() {
  if (userStore.user?.id) {
    void router.push(`/user/${userStore.user.id}`)
    return
  }
  router.back()
}
</script>

<template>
  <main class="source-page">
    <header class="source-page__header">
      <button type="button" class="source-page__back" @click="goBack">
        <i class="ri-arrow-left-line" aria-hidden="true"></i>
        {{ t('characterCards.newPage.backProfile') }}
      </button>
      <div class="source-page__title-row">
        <div>
          <span class="source-page__kicker">{{ t('characterCards.newPage.kicker') }}</span>
          <h1>{{ t('characterCards.newPage.title') }}</h1>
          <p>{{ t('characterCards.newPage.subtitle') }}</p>
        </div>
        <div class="source-page__seal" aria-hidden="true"><i class="ri-quill-pen-line"></i></div>
      </div>
    </header>

    <aside class="sync-note">
      <i class="ri-git-merge-line" aria-hidden="true"></i>
      <div>
        <strong>{{ t('characterCards.newPage.syncTitle') }}</strong>
        <p>{{ t('characterCards.newPage.syncBody') }}</p>
      </div>
    </aside>

    <div class="source-grid">
      <section class="source-panel source-panel--backup" aria-labelledby="backup-source-title">
        <header class="source-panel__header">
          <span class="source-panel__number">A</span>
          <div>
            <h2 id="backup-source-title">{{ t('characterCards.newPage.backupTitle') }}</h2>
            <p>{{ t('characterCards.newPage.backupBody') }}</p>
          </div>
        </header>

        <div v-if="loading" class="source-state" role="status">
          <i class="ri-loader-4-line spin" aria-hidden="true"></i>
          {{ t('characterCards.newPage.loading') }}
        </div>

        <div v-else-if="loadError" class="source-state source-state--error" role="alert">
          <i class="ri-file-warning-line" aria-hidden="true"></i>
          <div><strong>{{ t('characterCards.newPage.loadErrorTitle') }}</strong><span>{{ loadError }}</span></div>
          <button type="button" @click="loadSources">{{ t('characterCards.common.reload') }}</button>
        </div>

        <div v-else-if="sourceGroups.length === 0" class="source-state source-state--empty">
          <i class="ri-archive-drawer-line" aria-hidden="true"></i>
          <div>
            <strong>{{ t('characterCards.newPage.emptyTitle') }}</strong>
            <span>{{ t('characterCards.newPage.emptyBody') }}</span>
          </div>
        </div>

        <div v-else class="account-groups">
          <section v-for="group in sourceGroups" :key="group.accountId" class="account-group">
            <header>
              <span><i class="ri-folder-user-line" aria-hidden="true"></i>{{ group.accountId }}</span>
              <small>{{ t('characterCards.newPage.profileCount', { count: group.profiles.length }) }}</small>
            </header>
            <div class="profile-sources" role="radiogroup" :aria-label="t('characterCards.newPage.groupAria', { account: group.accountId })">
              <button
                v-for="source in group.profiles"
                :key="sourceKey(source)"
                type="button"
                class="profile-source"
                :class="{ selected: selectedSourceKey === sourceKey(source) }"
                role="radio"
                :aria-checked="selectedSourceKey === sourceKey(source)"
                @click="selectedSourceKey = sourceKey(source)"
              >
                <span class="profile-source__icon">
                  <i :class="source.icon ? 'ri-user-star-line' : 'ri-user-line'" aria-hidden="true"></i>
                </span>
                <span class="profile-source__copy">
                  <strong>{{ sourceName(source) }}</strong>
                  <span>{{ sourceMeta(source) }}</span>
                  <small>{{ t('characterCards.newPage.backupUpdated', { date: formatDate(source.backup_updated_at) }) }}</small>
                </span>
                <span class="profile-source__check" aria-hidden="true">
                  <i :class="selectedSourceKey === sourceKey(source) ? 'ri-check-line' : 'ri-arrow-right-s-line'"></i>
                </span>
              </button>
            </div>
          </section>
        </div>

        <footer class="source-panel__footer">
          <button
            type="button"
            class="source-action source-action--primary"
            :disabled="!selectedSource || Boolean(creatingType)"
            @click="createFromBackup"
          >
            <i :class="creatingType === 'backup' ? 'ri-loader-4-line spin' : 'ri-file-copy-2-line'" aria-hidden="true"></i>
            {{ creatingType === 'backup' ? t('characterCards.newPage.creating') : t('characterCards.newPage.createFromSelected') }}
          </button>
        </footer>
      </section>

      <section class="source-panel source-panel--blank" aria-labelledby="blank-source-title">
        <header class="source-panel__header">
          <span class="source-panel__number">B</span>
          <div>
            <h2 id="blank-source-title">{{ t('characterCards.newPage.blankTitle') }}</h2>
            <p>{{ t('characterCards.newPage.blankBody') }}</p>
          </div>
        </header>

        <div class="blank-ledger" aria-hidden="true">
          <span class="blank-ledger__compass"><i class="ri-compass-3-line"></i></span>
          <span class="blank-ledger__line"></span>
          <span class="blank-ledger__line short"></span>
          <span class="blank-ledger__line"></span>
          <span class="blank-ledger__stamp">RPB · NEW</span>
        </div>

        <div class="blank-copy">
          <strong>{{ t('characterCards.newPage.blankSheetTitle') }}</strong>
          <p>{{ t('characterCards.newPage.blankSheetBody') }}</p>
          <ul>
            <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.newPage.blankFeaturePortrait') }}</li>
            <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.newPage.blankFeatureTabs') }}</li>
            <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.newPage.blankFeaturePrivate') }}</li>
          </ul>
        </div>

        <footer class="source-panel__footer">
          <button
            type="button"
            class="source-action source-action--blank"
            :disabled="Boolean(creatingType)"
            @click="createFromBlank"
          >
            <i :class="creatingType === 'blank' ? 'ri-loader-4-line spin' : 'ri-add-line'" aria-hidden="true"></i>
            {{ creatingType === 'blank' ? t('characterCards.newPage.creating') : t('characterCards.newPage.createBlank') }}
          </button>
        </footer>
      </section>
    </div>
  </main>
</template>

<style scoped>
.source-page {
  --ink: var(--color-text-main);
  --walnut: var(--color-primary);
  --copper: var(--color-accent);
  --rust: var(--color-secondary);
  --muted: var(--color-text-secondary);
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
  padding: 30px 0 54px;
  color: var(--ink);
}

.source-page__header { margin-bottom: 18px; }

.source-page__back {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 7px 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
}

.source-page__back:hover { color: var(--rust); }

.source-page__title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 30px;
  margin-top: 22px;
}

.source-page__kicker,
.source-panel__number,
.blank-ledger__stamp {
  color: var(--copper);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.source-page h1 {
  margin: 5px 0 8px;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: clamp(30px, 4vw, 46px);
  font-weight: 600;
  letter-spacing: 0.01em;
}

.source-page__title-row p {
  max-width: 720px;
  margin: 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.7;
}

.source-page__seal {
  display: grid;
  width: 76px;
  height: 76px;
  flex: 0 0 76px;
  place-items: center;
  border: 1px solid var(--color-border-hover);
  border-radius: 50%;
  color: var(--rust);
  font-size: 30px;
  outline: 1px dashed var(--color-border);
  outline-offset: 7px;
}

.sync-note {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 13px;
  margin-bottom: 20px;
  padding: 14px 16px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-card-bg);
}

.sync-note > i { color: var(--copper); font-size: 21px; }
.sync-note strong { color: var(--walnut); font-size: 13px; }
.sync-note p { margin: 3px 0 0; color: var(--muted); font-size: 12px; line-height: 1.6; }

.source-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(300px, 0.8fr);
  gap: 18px;
  align-items: stretch;
}

.source-panel {
  display: flex;
  min-width: 0;
  min-height: 560px;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-panel-bg);
  box-shadow: var(--shadow-md);
}

.source-panel--backup { border-top: 4px solid var(--copper); }
.source-panel--blank { border-top: 4px solid var(--walnut); }

.source-panel__header {
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr);
  gap: 11px;
  padding: 22px 24px 18px;
  border-bottom: 1px solid var(--color-border-light);
}

.source-panel__number {
  display: grid;
  width: 26px;
  height: 26px;
  place-items: center;
  border: 1px solid currentColor;
  border-radius: 50%;
}

.source-panel h2 {
  margin: 0 0 5px;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 20px;
  font-weight: 600;
}

.source-panel__header p {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.55;
}

.source-state {
  display: flex;
  min-height: 280px;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 30px;
  color: var(--muted);
  text-align: center;
}

.source-state--empty,
.source-state--error { flex-direction: column; }
.source-state > i { color: var(--copper); font-size: 34px; }
.source-state div { display: grid; gap: 4px; }
.source-state strong { color: var(--walnut); }
.source-state span { font-size: 12px; }
.source-state button {
  margin-top: 5px;
  padding: 7px 12px;
  border: 1px solid var(--copper);
  border-radius: 7px;
  background: transparent;
  color: var(--rust);
  cursor: pointer;
}

.account-groups {
  display: flex;
  max-height: 420px;
  flex: 1;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
  padding: 18px 20px;
}

.account-group > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--walnut);
  font-size: 12px;
  font-weight: 700;
}

.account-group > header span { display: inline-flex; align-items: center; gap: 6px; }
.account-group > header i { color: var(--copper); }
.account-group > header small { color: var(--muted); font-weight: 400; }

.profile-sources { display: grid; gap: 7px; }

.profile-source {
  display: grid;
  width: 100%;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 10px 11px;
  border: 1px solid var(--color-border);
  border-radius: 9px;
  background: var(--color-card-bg);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms ease;
}

.profile-source:hover,
.profile-source.selected {
  border-color: var(--color-border-hover);
  background: var(--color-card-bg-hover);
  transform: translateX(2px);
}

.profile-source:focus-visible,
.source-action:focus-visible,
.source-page__back:focus-visible,
.source-state button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--color-accent) 34%, transparent);
  outline-offset: 2px;
}

.profile-source__icon {
  display: grid;
  width: 42px;
  height: 48px;
  place-items: center;
  border-radius: 5px;
  background: linear-gradient(145deg, var(--gradient-start), var(--gradient-end));
  color: var(--gradient-text);
  font-size: 19px;
}

.profile-source__copy { display: grid; min-width: 0; gap: 2px; }
.profile-source__copy strong { overflow: hidden; color: var(--ink); font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.profile-source__copy > span { overflow: hidden; color: var(--muted); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.profile-source__copy small { color: var(--color-text-muted); font-size: 9px; }
.profile-source__check { color: var(--copper); font-size: 17px; }

.source-panel__footer {
  margin-top: auto;
  padding: 16px 20px;
  border-top: 1px solid var(--color-border-light);
  background: color-mix(in srgb, var(--color-card-bg) 82%, transparent);
}

.source-action {
  display: inline-flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--btn-primary-bg);
  border-radius: 8px;
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  font-weight: 700;
}

.source-action--blank { background: var(--color-primary); border-color: var(--color-primary); color: var(--color-text-light); }
.source-action:disabled { cursor: not-allowed; opacity: 0.48; }

.blank-ledger {
  position: relative;
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin: 24px 24px 16px;
  border: 1px solid var(--color-border);
  background:
    linear-gradient(90deg, transparent 28px, color-mix(in srgb, var(--color-accent) 18%, transparent) 29px, transparent 30px),
    repeating-linear-gradient(var(--color-card-bg), var(--color-card-bg) 28px, var(--color-border-light) 29px);
  box-shadow: inset 0 0 30px color-mix(in srgb, var(--color-primary) 5%, transparent);
}

.blank-ledger__compass {
  display: grid;
  width: 70px;
  height: 70px;
  place-items: center;
  border: 1px solid var(--color-border-hover);
  border-radius: 50%;
  color: var(--color-accent);
  font-size: 36px;
}

.blank-ledger__line { width: 52%; height: 1px; background: var(--color-border); }
.blank-ledger__line.short { width: 34%; }
.blank-ledger__stamp { position: absolute; right: 14px; bottom: 12px; transform: rotate(-4deg); }

.blank-copy { padding: 0 26px 22px; }
.blank-copy > strong { font-family: Georgia, 'Noto Serif SC', serif; font-size: 18px; }
.blank-copy p { margin: 8px 0 14px; color: var(--muted); font-size: 12px; line-height: 1.65; }
.blank-copy ul { display: grid; gap: 7px; margin: 0; padding: 0; list-style: none; color: var(--walnut); font-size: 12px; }
.blank-copy li { display: flex; align-items: center; gap: 7px; }
.blank-copy li i { color: var(--copper); }

.spin { animation: source-spin 900ms linear infinite; }
@keyframes source-spin { to { transform: rotate(360deg); } }

@media (max-width: 860px) {
  .source-grid { grid-template-columns: 1fr; }
  .source-panel { min-height: auto; }
}

@media (max-width: 560px) {
  .source-page { width: min(100% - 24px, 1180px); padding-top: 18px; }
  .source-page__seal { display: none; }
  .source-panel__header { padding-right: 16px; padding-left: 16px; }
  .account-groups { padding-right: 12px; padding-left: 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .profile-source { transition: none; }
  .spin { animation: none; }
}
</style>
