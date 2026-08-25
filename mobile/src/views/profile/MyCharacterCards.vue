<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  deleteCharacterCard,
  listMyCharacterCards,
  type CharacterCardSummary,
} from '@/api/characterCard'
import { resolveApiUrl } from '@/api/image'
import CachedImage from '@/components/CachedImage.vue'

type CardState = 'published' | 'pending' | 'draft' | 'private' | 'rejected' | 'unsubmitted'

const router = useRouter()
const { locale, t } = useI18n()
const toast = useToastStore()

const cards = ref<CharacterCardSummary[]>([])
const loading = ref(true)
const loadFailed = ref(false)
const deleteDialogOpen = ref(false)
const deleting = ref(false)
const deleteTarget = ref<CharacterCardSummary | null>(null)

function classifyCard(card: CharacterCardSummary): CardState {
  if (card.status === 'draft') return 'draft'
  if (card.review_status === 'rejected') return 'rejected'
  if (card.review_status === 'pending') return 'pending'
  if (card.visibility === 'private') return 'private'
  if (card.status === 'published' && card.visibility === 'public' && card.review_status === 'approved') {
    return 'published'
  }
  return 'unsubmitted'
}

const counts = computed<Record<CardState, number>>(() => {
  const result: Record<CardState, number> = {
    published: 0,
    pending: 0,
    draft: 0,
    private: 0,
    rejected: 0,
    unsubmitted: 0,
  }
  for (const card of cards.value) result[classifyCard(card)] += 1
  return result
})

const statusLedger = computed(() => (
  (['published', 'pending', 'draft', 'private', 'rejected', 'unsubmitted'] as CardState[]).map(status => ({
    status,
    count: counts.value[status],
    label: t(`characterCards.management.status.${status}`),
  }))
))

function displayName(card: CharacterCardSummary) {
  return card.display_name?.trim()
    || [card.first_name, card.last_name].filter(Boolean).join(' ')
    || t('characterCards.common.unnamed')
}

function identityLine(card: CharacterCardSummary) {
  return [card.race, card.class].map(value => value?.trim()).filter(Boolean).join(' · ')
    || t('characterCards.common.noIdentity')
}

function cardTitle(card: CharacterCardSummary) {
  return card.title?.trim() || card.full_title?.trim() || t('characterCards.management.titleMissing')
}

function portraitUrl(card: CharacterCardSummary) {
  const portraits = [...(card.portraits || [])].sort((left, right) => left.sort_order - right.sort_order)
  const cover = portraits.find(portrait => portrait.is_cover) || portraits[0]
  return resolveApiUrl(cover?.image_url || card.portrait_image_url || '')
}

function formatUpdatedAt(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('characterCards.management.timeUnknown')
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

async function loadCards() {
  loading.value = true
  loadFailed.value = false
  try {
    const response = await listMyCharacterCards()
    cards.value = response.character_cards || []
  } catch (error) {
    console.error('Failed to load RPBox character cards', error)
    cards.value = []
    loadFailed.value = true
    toast.error((error as Error)?.message || t('characterCards.management.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openDeleteDialog(card: CharacterCardSummary) {
  deleteTarget.value = card
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  if (deleting.value) return
  deleteDialogOpen.value = false
  deleteTarget.value = null
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target || deleting.value) return
  deleting.value = true
  try {
    await deleteCharacterCard(target.id)
    cards.value = cards.value.filter(card => card.id !== target.id)
    toast.success(t('characterCards.management.deleteSuccess'))
    deleteDialogOpen.value = false
    deleteTarget.value = null
  } catch (error) {
    console.error('Failed to delete RPBox character card', error)
    toast.error((error as Error)?.message || t('characterCards.management.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

onMounted(loadCards)
</script>

<template>
  <div class="sub-page character-ledger-page">
    <header class="sub-header ledger-header">
      <button type="button" class="back-btn" :aria-label="t('common.button.back')" @click="router.back()">
        <i class="ri-arrow-left-line" aria-hidden="true" />
      </button>
      <div class="ledger-heading">
        <span>RPBOX · DOSSIERS</span>
        <h1>{{ t('characterCards.management.title') }}</h1>
      </div>
      <button
        type="button"
        class="new-card-btn"
        data-testid="character-card-new"
        @click="router.push({ name: 'character-card-new' })"
      >
        <i class="ri-add-line" aria-hidden="true" />
        <span>{{ t('characterCards.management.new') }}</span>
      </button>
    </header>

    <main class="sub-body ledger-body">
      <section class="status-ledger" :aria-label="t('characterCards.management.statusSummary')">
        <div v-for="entry in statusLedger" :key="entry.status" :data-status-count="entry.status">
          <strong>{{ entry.count }}</strong>
          <span>{{ entry.label }}</span>
        </div>
      </section>

      <section v-if="loading" class="ledger-state" data-testid="character-card-loading">
        <i class="ri-loader-4-line spin" aria-hidden="true" />
        <h2>{{ t('characterCards.management.loading') }}</h2>
        <p>{{ t('characterCards.management.loadingBody') }}</p>
      </section>

      <section v-else-if="loadFailed" class="ledger-state ledger-state--error">
        <i class="ri-file-warning-line" aria-hidden="true" />
        <h2>{{ t('characterCards.management.loadErrorTitle') }}</h2>
        <p>{{ t('characterCards.management.loadErrorBody') }}</p>
        <button type="button" @click="loadCards">{{ t('characterCards.common.reload') }}</button>
      </section>

      <section v-else-if="cards.length === 0" class="ledger-state ledger-state--empty">
        <i class="ri-quill-pen-line" aria-hidden="true" />
        <span>FILE · 000</span>
        <h2>{{ t('characterCards.management.emptyTitle') }}</h2>
        <p>{{ t('characterCards.management.emptyBody') }}</p>
        <button type="button" @click="router.push({ name: 'character-card-new' })">
          {{ t('characterCards.management.createFirst') }}
        </button>
      </section>

      <section v-else class="dossier-list" :aria-label="t('characterCards.management.listAria')">
        <article
          v-for="(card, index) in cards"
          :key="card.id"
          class="dossier-card"
          :data-card-state="classifyCard(card)"
        >
          <div class="dossier-index">FILE · {{ String(index + 1).padStart(3, '0') }}</div>
          <div class="dossier-main">
            <div class="portrait-frame">
              <CachedImage
                v-if="portraitUrl(card)"
                class="portrait-image"
                :src="portraitUrl(card)"
                :alt="t('characterCards.common.portraitAlt', { name: displayName(card) })"
                :auth-fetch="true"
              />
              <div v-else class="portrait-fallback" aria-hidden="true">
                <i class="ri-user-star-line" />
              </div>
            </div>

            <div class="dossier-copy">
              <h2>{{ displayName(card) }}</h2>
              <p class="character-title">{{ cardTitle(card) }}</p>
              <p class="identity-line">{{ identityLine(card) }}</p>
              <time :datetime="card.updated_at">
                {{ t('characterCards.management.updatedAt', { time: formatUpdatedAt(card.updated_at) }) }}
              </time>
            </div>

            <span class="status-seal" :class="`status-seal--${classifyCard(card)}`">
              {{ t(`characterCards.management.status.${classifyCard(card)}`) }}
            </span>
          </div>

          <div class="dossier-actions">
            <button
              type="button"
              data-testid="character-card-view"
              @click="router.push({ name: 'character-card-detail', params: { id: card.id } })"
            >
              <i class="ri-book-open-line" aria-hidden="true" />
              {{ t('characterCards.common.view') }}
            </button>
            <button
              type="button"
              data-testid="character-card-edit"
              @click="router.push({ name: 'character-card-edit', params: { id: card.id } })"
            >
              <i class="ri-quill-pen-line" aria-hidden="true" />
              {{ t('characterCards.management.edit') }}
            </button>
            <button
              type="button"
              class="danger-action"
              data-testid="character-card-delete"
              @click="openDeleteDialog(card)"
            >
              <i class="ri-delete-bin-line" aria-hidden="true" />
              {{ t('characterCards.common.delete') }}
            </button>
          </div>
        </article>
      </section>
    </main>

    <div
      v-if="deleteDialogOpen && deleteTarget"
      class="dialog-mask"
      data-testid="character-card-delete-dialog"
      @click.self="closeDeleteDialog"
    >
      <section class="delete-dialog" role="dialog" aria-modal="true" :aria-labelledby="`delete-card-${deleteTarget.id}`">
        <span class="dialog-kicker">IRREVERSIBLE · ACTION</span>
        <h2 :id="`delete-card-${deleteTarget.id}`">{{ t('characterCards.management.deleteTitle') }}</h2>
        <p>{{ t('characterCards.management.deleteMessage', { name: displayName(deleteTarget) }) }}</p>
        <div class="dialog-actions">
          <button type="button" :disabled="deleting" @click="closeDeleteDialog">
            {{ t('characterCards.common.cancel') }}
          </button>
          <button
            type="button"
            class="confirm-delete"
            data-testid="character-card-delete-confirm"
            :disabled="deleting"
            @click="confirmDelete"
          >
            {{ deleting ? t('characterCards.management.deleting') : t('characterCards.management.deleteConfirm') }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.character-ledger-page {
  background: var(--color-background);
}

.ledger-header {
  min-height: 64px;
}

.ledger-heading {
  min-width: 0;
  flex: 1;
}

.ledger-heading > span,
.dossier-index,
.dialog-kicker,
.status-ledger span,
.status-seal {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ledger-heading > span {
  display: block;
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
}

.ledger-heading h1,
.dossier-copy h2,
.ledger-state h2,
.delete-dialog h2 {
  font-family: Georgia, 'Times New Roman', serif;
}

.ledger-heading h1 {
  margin-top: 2px;
  font-size: 19px;
}

.new-card-btn {
  min-height: 44px;
  padding: 0 10px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: var(--text-light);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 700;
}

.ledger-body {
  padding-bottom: calc(28px + var(--safe-bottom, 0px));
}

.status-ledger {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  margin-bottom: 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
}

.status-ledger div {
  min-width: 0;
  padding: 9px 6px;
  border-right: 1px solid var(--color-border-light);
  border-bottom: 1px solid var(--color-border-light);
  text-align: center;
}

.status-ledger div:nth-child(3n) {
  border-right: none;
}

.status-ledger div:nth-last-child(-n + 3) {
  border-bottom: none;
}

.status-ledger strong {
  display: block;
  color: var(--color-primary);
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 20px;
  line-height: 1;
}

.status-ledger span {
  display: block;
  overflow: hidden;
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ledger-state {
  min-height: 320px;
  padding: 34px 24px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  box-shadow: var(--shadow-sm);
}

.ledger-state > i {
  color: var(--color-accent);
  font-size: 36px;
}

.ledger-state > span {
  margin-top: 10px;
  color: var(--color-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 10px;
  letter-spacing: 0.12em;
}

.ledger-state h2 {
  margin-top: 10px;
  color: var(--color-text-main);
  font-size: 20px;
}

.ledger-state p {
  max-width: 310px;
  margin-top: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.ledger-state button {
  min-height: 44px;
  margin-top: 18px;
  padding: 0 16px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: var(--text-light);
  font-weight: 700;
}

.dossier-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dossier-card {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-card-bg);
  box-shadow: var(--shadow-sm);
}

.dossier-card::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: var(--color-accent);
  content: '';
}

.dossier-index {
  padding: 8px 12px 6px 16px;
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-secondary);
  font-size: 8px;
  font-weight: 700;
}

.dossier-main {
  display: grid;
  grid-template-columns: 66px minmax(0, 1fr) 58px;
  align-items: center;
  gap: 11px;
  padding: 12px 12px 12px 16px;
}

.portrait-frame,
.portrait-fallback {
  width: 66px;
  height: 84px;
  border-radius: 6px;
}

.portrait-frame {
  overflow: hidden;
  border: 1px solid var(--color-border);
  background: var(--color-border-light);
}

.portrait-image {
  width: 100%;
  height: 100%;
}

.portrait-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(145deg, var(--color-border), var(--color-border-light));
  color: var(--color-text-muted);
}

.portrait-fallback i {
  font-size: 28px;
}

.dossier-copy {
  min-width: 0;
}

.dossier-copy h2 {
  overflow: hidden;
  color: var(--color-text-main);
  font-size: 18px;
  line-height: 1.15;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.character-title,
.identity-line,
.dossier-copy time {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.character-title {
  margin-top: 5px;
  color: var(--color-secondary);
  font-size: 12px;
  font-weight: 700;
}

.identity-line {
  margin-top: 3px;
  color: var(--color-text-secondary);
  font-size: 11px;
}

.dossier-copy time {
  display: block;
  margin-top: 10px;
  color: var(--color-text-muted);
  font-size: 9px;
}

.status-seal {
  width: 56px;
  min-height: 56px;
  padding: 6px;
  border: 2px solid var(--color-accent);
  border-radius: 50%;
  color: var(--color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  font-weight: 900;
  line-height: 1.25;
  text-align: center;
  box-shadow: inset 0 0 0 2px var(--color-card-bg);
  transform: rotate(-6deg);
}

.status-seal--published {
  border-color: var(--color-success);
  color: var(--color-success);
}

.status-seal--pending,
.status-seal--unsubmitted {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.status-seal--draft,
.status-seal--private {
  border-color: var(--color-text-muted);
  color: var(--color-text-secondary);
}

.status-seal--rejected {
  border-color: var(--btn-danger-bg);
  color: var(--btn-danger-bg);
}

.dossier-actions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding-left: 4px;
  border-top: 1px solid var(--color-border-light);
}

.dossier-actions button {
  min-height: 44px;
  border: none;
  border-right: 1px solid var(--color-border-light);
  background: var(--color-panel-bg);
  color: var(--color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 700;
}

.dossier-actions button:last-child {
  border-right: none;
}

.dossier-actions button.danger-action {
  color: var(--btn-danger-bg);
}

.dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  padding: 16px;
  background: var(--overlay-bg);
  display: flex;
  align-items: center;
  justify-content: center;
}

.delete-dialog {
  width: 100%;
  max-width: 370px;
  padding: 18px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  box-shadow: var(--shadow-sm);
}

.dialog-kicker {
  color: var(--btn-danger-bg);
  font-size: 9px;
  font-weight: 800;
}

.delete-dialog h2 {
  margin-top: 6px;
  color: var(--color-text-main);
  font-size: 20px;
}

.delete-dialog p {
  margin-top: 10px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

.dialog-actions button {
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--color-primary);
  font-weight: 700;
}

.dialog-actions button.confirm-delete {
  border-color: var(--btn-danger-bg);
  background: var(--btn-danger-bg);
  color: var(--btn-primary-text);
}

.dialog-actions button:disabled {
  opacity: 0.55;
}

.spin {
  animation: ledger-spin 0.8s linear infinite;
}

@keyframes ledger-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .spin {
    animation: none;
  }
}

@media (max-width: 380px) {
  .new-card-btn span {
    display: none;
  }

  .new-card-btn {
    width: 44px;
    padding: 0;
    justify-content: center;
  }

  .dossier-main {
    grid-template-columns: 58px minmax(0, 1fr) 50px;
    gap: 8px;
  }

  .portrait-frame,
  .portrait-fallback {
    width: 58px;
    height: 74px;
  }

  .status-seal {
    width: 50px;
    min-height: 50px;
    font-size: 7px;
  }
}
</style>
