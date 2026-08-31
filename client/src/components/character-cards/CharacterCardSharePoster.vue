<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CharacterCard } from '@/api/characterCard'
import { resolveApiUrl } from '@/api/item'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import CharacterCardGalleryImage from '@/components/character-cards/CharacterCardGalleryImage.vue'
import CharacterCardImpressionMark from '@/components/character-cards/CharacterCardImpressionMark.vue'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { getCharacterCardDisplayName, normalizeCharacterCardImpressions } from '@/utils/characterCardDraft'
import { getCharacterCardCoverPortrait, normalizeCharacterCardPortraits } from '@/utils/characterCardPortraits'
import { sanitizeRichHtml } from '@/utils/sanitizeHtml'

const props = defineProps<{
  card: CharacterCard
}>()

const { t, locale } = useI18n()
const posterRef = ref<HTMLElement | null>(null)
const displayName = computed(() => getCharacterCardDisplayName(props.card))
const sanitizedBackgroundStory = computed(() => sanitizeRichHtml(props.card.background_story || ''))
const sanitizedFirstImpression = computed(() => sanitizeRichHtml(props.card.first_impression || ''))
const sanitizedOtherContent = computed(() => sanitizeRichHtml(props.card.other_content || ''))
const displayColor = computed(() => getCharacterCardDisplayColor(props.card))
const posterNameColor = computed(() => displayColor.value
  ? `color-mix(in srgb, ${displayColor.value} 55%, #211b18)`
  : '')
const identityLine = computed(() => [props.card.race, props.card.class].filter(Boolean).join(' · '))
const portraits = computed(() => normalizeCharacterCardPortraits(props.card))
const coverPortrait = computed(() => getCharacterCardCoverPortrait(portraits.value))
const impressions = computed(() => normalizeCharacterCardImpressions(props.card.impressions).filter((item) => item.active))
const details = computed(() => [
  { label: t('characterCards.detail.fields.firstName'), value: props.card.first_name },
  { label: t('characterCards.detail.fields.lastName'), value: props.card.last_name },
  { label: t('characterCards.detail.fields.fullTitle'), value: props.card.full_title },
  { label: t('characterCards.detail.fields.race'), value: props.card.race },
  { label: t('characterCards.detail.fields.class'), value: props.card.class },
  { label: t('characterCards.detail.fields.age'), value: props.card.age },
  { label: t('characterCards.detail.fields.height'), value: props.card.height },
  { label: t('characterCards.detail.fields.weight'), value: props.card.weight },
  { label: t('characterCards.detail.fields.eyes'), value: props.card.eye_color },
  { label: t('characterCards.detail.fields.birthplace'), value: props.card.birthplace },
  { label: t('characterCards.detail.fields.residence'), value: props.card.residence },
  { label: t('characterCards.detail.fields.relationship'), value: props.card.relationship_status },
].filter((item) => Boolean(item.value)))
const generatedAt = computed(() => new Intl.DateTimeFormat(locale.value, {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
}).format(new Date()))

function getElement() {
  return posterRef.value
}

defineExpose({ getElement })
</script>

<template>
  <article ref="posterRef" class="share-poster" data-testid="character-share-poster">
    <header class="poster-masthead">
      <div class="poster-brand">
        <span class="poster-brand__mark">R</span>
        <div>
          <strong>RPBOX</strong>
          <small>{{ t('characterCards.share.poster.archive') }}</small>
        </div>
      </div>
      <span class="poster-record">CHARACTER / {{ String(card.id).padStart(4, '0') }}</span>
    </header>

    <section class="poster-hero">
      <figure class="poster-cover">
        <CharacterCardGalleryImage
          v-if="coverPortrait"
          :card="card"
          :portrait="coverPortrait"
          :alt="t('characterCards.common.portraitAlt', { name: displayName })"
          :width="900"
          :quality="94"
        />
        <div v-else class="poster-cover__empty">
          <span>{{ displayName.charAt(0) }}</span>
          <small>{{ t('characterCards.detail.portraitMissing') }}</small>
        </div>
        <figcaption>{{ t('characterCards.share.poster.cover') }}</figcaption>
      </figure>

      <div class="poster-identity">
        <span class="poster-eyebrow">{{ t('characterCards.share.poster.identity') }}</span>
        <h1 :style="posterNameColor ? { color: posterNameColor } : undefined">{{ displayName }}</h1>
        <p class="poster-title">{{ card.title || card.full_title || t('characterCards.detail.titleMissing') }}</p>
        <p class="poster-line">{{ identityLine || t('characterCards.detail.identityMissing') }}</p>
        <blockquote>{{ card.summary || t('characterCards.detail.summaryMissing') }}</blockquote>
        <div class="poster-seal" aria-hidden="true">
          <span>RP</span>
          <small>OPEN ARCHIVE</small>
        </div>
      </div>
    </section>

    <section v-if="portraits.length > 1" class="poster-section poster-gallery-section">
      <header class="poster-section__header">
        <span>01 / IMAGES</span>
        <h2>{{ t('characterCards.share.poster.gallery') }}</h2>
        <small>{{ t('characterCards.share.poster.galleryCount', { count: portraits.length }) }}</small>
      </header>
      <div class="poster-gallery">
        <figure v-for="(portrait, index) in portraits" :key="portrait.id">
          <CharacterCardGalleryImage
            :card="card"
            :portrait="portrait"
            :alt="t('characterCards.common.portraitAlt', { name: displayName })"
            :width="720"
            :quality="92"
          />
          <figcaption>{{ String(index + 1).padStart(2, '0') }}<span v-if="portrait.is_cover">COVER</span></figcaption>
        </figure>
      </div>
    </section>

    <section class="poster-section">
      <header class="poster-section__header">
        <span>02 / PROFILE</span>
        <h2>{{ t('characterCards.share.poster.details') }}</h2>
        <small>{{ t('characterCards.share.poster.detailsCaption') }}</small>
      </header>
      <dl v-if="details.length" class="poster-ledger">
        <div v-for="item in details" :key="item.label">
          <dt>{{ item.label }}</dt>
          <dd>{{ item.value }}</dd>
        </div>
      </dl>
      <p v-else class="poster-empty">{{ t('characterCards.detail.moreBasicMissing') }}</p>
    </section>

    <section class="poster-section poster-story-section">
      <header class="poster-section__header">
        <span>03 / CHRONICLE</span>
        <h2>{{ t('characterCards.tabs.background') }}</h2>
        <small>{{ t('characterCards.share.poster.chronicleCaption') }}</small>
      </header>
      <div v-if="card.background_story" class="poster-rich" v-html="sanitizedBackgroundStory"></div>
      <p v-else class="poster-empty">{{ t('characterCards.detail.backgroundMissing') }}</p>
    </section>

    <section class="poster-section poster-impression-section">
      <header class="poster-section__header">
        <span>04 / FIRST SIGHT</span>
        <h2>{{ t('characterCards.tabs.impression') }}</h2>
        <small>{{ t('characterCards.share.poster.impressionCaption') }}</small>
      </header>
      <div v-if="impressions.length" class="poster-impressions">
        <article v-for="impression in impressions" :key="impression.slot" class="poster-impression">
          <div class="poster-impression__mark">
            <CharacterCardImpressionMark
              :icon-image-url="impression.icon_image_url"
              :trp3-icon="impression.trp3_icon"
              :fallback-label="String(impression.slot)"
              :size="76"
            />
            <span>{{ String(impression.slot).padStart(2, '0') }}</span>
          </div>
          <div class="poster-impression__copy">
            <h3>{{ impression.title || t('characterCards.detail.observationUnnamed') }}</h3>
            <p>{{ impression.text || t('characterCards.detail.observationEmpty') }}</p>
          </div>
          <figure v-if="impression.image_url" class="poster-impression__image">
            <AuthenticatedImage
              :src="resolveApiUrl(impression.image_url)"
              :alt="t('characterCards.common.impressionImageAlt', { title: impression.title })"
            />
          </figure>
        </article>
      </div>
      <p v-else class="poster-empty">{{ t('characterCards.detail.impressionsMissing') }}</p>

      <div v-if="card.first_impression" class="poster-supplement">
        <span>{{ t('characterCards.detail.supplement') }}</span>
        <div class="poster-rich" v-html="sanitizedFirstImpression"></div>
      </div>
    </section>

    <section class="poster-section poster-other-section">
      <header class="poster-section__header">
        <span>05 / FILED NOTES</span>
        <h2>{{ t('characterCards.tabs.other') }}</h2>
        <small>{{ t('characterCards.share.poster.otherCaption') }}</small>
      </header>
      <div v-if="card.other_content" class="poster-rich" v-html="sanitizedOtherContent"></div>
      <p v-else class="poster-empty">{{ t('characterCards.detail.otherMissing') }}</p>
    </section>

    <footer class="poster-footer">
      <div>
        <strong>RPBOX</strong>
        <span>totalrpbox.com/character-cards/{{ card.id }}</span>
      </div>
      <p>{{ t('characterCards.share.poster.generatedAt', { date: generatedAt }) }}</p>
    </footer>
  </article>
</template>

<style scoped>
.share-poster {
  --poster-ink: #211b18;
  --poster-muted: #74685e;
  --poster-paper: #f1e7d4;
  --poster-paper-deep: #e3d2b8;
  --poster-night: #17202a;
  --poster-copper: #ad6c3f;
  width: 900px;
  overflow: hidden;
  background:
    linear-gradient(90deg, rgba(82, 55, 38, .035) 1px, transparent 1px),
    linear-gradient(rgba(82, 55, 38, .028) 1px, transparent 1px),
    var(--poster-paper);
  background-size: 28px 28px;
  color: var(--poster-ink);
  font-family: Inter, 'Noto Sans SC', 'Microsoft YaHei', sans-serif;
  text-align: left;
}

.poster-masthead { display: flex; align-items: center; justify-content: space-between; padding: 24px 34px; background: var(--poster-night); color: #f7eddd; }
.poster-brand { display: flex; align-items: center; gap: 12px; }
.poster-brand__mark { display: grid; width: 42px; height: 42px; place-items: center; border: 1px solid #c29166; border-radius: 50%; color: #dfb07d; font: 600 22px/1 Georgia, serif; }
.poster-brand div { display: grid; gap: 2px; }
.poster-brand strong { font: 700 14px/1 ui-monospace, Consolas, monospace; letter-spacing: .22em; }
.poster-brand small { color: #b8afa6; font-size: 9px; letter-spacing: .1em; text-transform: uppercase; }
.poster-record { color: #c8bdaf; font: 700 10px/1 ui-monospace, Consolas, monospace; letter-spacing: .14em; }

.poster-hero { display: grid; grid-template-columns: 330px minmax(0, 1fr); min-height: 440px; }
.poster-cover { position: relative; min-height: 440px; margin: 0; overflow: hidden; background: #222a32; }
.poster-cover :deep(.authenticated-image), .poster-cover :deep(img) { width: 100%; height: 100%; object-fit: cover; }
.poster-cover::after { position: absolute; inset: 0; content: ''; background: linear-gradient(to top, rgba(18, 25, 31, .55), transparent 45%); pointer-events: none; }
.poster-cover figcaption { position: absolute; z-index: 1; right: 18px; bottom: 16px; color: #f5eadb; font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .14em; }
.poster-cover__empty { display: grid; width: 100%; height: 100%; place-content: center; justify-items: center; gap: 10px; background: radial-gradient(circle at 50% 40%, #4c3c35, #17202a 62%); color: #f1dfc7; }
.poster-cover__empty span { font: 500 84px/1 Georgia, 'Noto Serif SC', serif; }
.poster-cover__empty small { color: #b7aa9d; }
.poster-identity { position: relative; padding: 66px 56px 48px; border-bottom: 1px solid #bda98e; background: linear-gradient(135deg, rgba(255,255,255,.32), transparent 58%); }
.poster-eyebrow { display: block; margin-bottom: 20px; color: var(--poster-copper); font: 700 10px/1 ui-monospace, Consolas, monospace; letter-spacing: .2em; text-transform: uppercase; }
.poster-identity h1 { max-width: 420px; margin: 0; font: 600 48px/1.12 Georgia, 'Noto Serif SC', serif; letter-spacing: -.035em; }
.poster-title { margin: 16px 0 0; color: #4b3c32; font: 600 17px/1.4 Georgia, 'Noto Serif SC', serif; }
.poster-line { margin: 8px 0 0; color: var(--poster-muted); font-size: 12px; letter-spacing: .04em; }
.poster-identity blockquote { margin: 34px 0 0; padding: 0 0 0 18px; border-left: 3px solid var(--poster-copper); color: #4b433c; font: 15px/1.8 Georgia, 'Noto Serif SC', serif; }
.poster-seal { position: absolute; right: 34px; bottom: 30px; display: grid; width: 88px; height: 88px; place-content: center; justify-items: center; border: 1px solid rgba(111, 74, 49, .42); border-radius: 50%; color: rgba(111, 74, 49, .55); transform: rotate(-8deg); }
.poster-seal span { font: 600 28px/1 Georgia, serif; }
.poster-seal small { margin-top: 5px; font: 700 6px/1 ui-monospace, monospace; letter-spacing: .09em; }

.poster-section { padding: 46px 48px 52px; border-bottom: 1px solid #bfae97; }
.poster-section__header { display: grid; grid-template-columns: 125px minmax(0, 1fr) auto; align-items: end; gap: 18px; margin-bottom: 30px; padding-bottom: 15px; border-bottom: 1px solid #aa9277; }
.poster-section__header > span { color: var(--poster-copper); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .14em; }
.poster-section__header h2 { margin: 0; font: 600 28px/1.1 Georgia, 'Noto Serif SC', serif; letter-spacing: -.02em; }
.poster-section__header small { color: var(--poster-muted); font-size: 9px; letter-spacing: .06em; }

.poster-gallery-section { background: #202a34; color: #f4eadc; }
.poster-gallery-section .poster-section__header { border-color: #51606a; }
.poster-gallery-section .poster-section__header small { color: #aeb8bd; }
.poster-gallery { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.poster-gallery figure { position: relative; min-height: 280px; margin: 0; overflow: hidden; border: 1px solid #4b5961; background: #111922; }
.poster-gallery :deep(.authenticated-image), .poster-gallery :deep(img) { width: 100%; height: 100%; object-fit: cover; }
.poster-gallery figcaption { position: absolute; right: 0; bottom: 0; left: 0; display: flex; justify-content: space-between; padding: 28px 12px 10px; background: linear-gradient(transparent, rgba(8,13,18,.82)); color: #f4eadc; font: 700 9px/1 ui-monospace, monospace; letter-spacing: .12em; }
.poster-gallery figcaption span { color: #e6aa73; }

.poster-ledger { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); margin: 0; border-top: 1px solid #c2af95; border-left: 1px solid #c2af95; }
.poster-ledger div { min-height: 82px; padding: 16px 18px; border-right: 1px solid #c2af95; border-bottom: 1px solid #c2af95; }
.poster-ledger dt { margin-bottom: 7px; color: var(--poster-muted); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .08em; text-transform: uppercase; }
.poster-ledger dd { margin: 0; font: 600 15px/1.45 Georgia, 'Noto Serif SC', serif; }

.poster-story-section { background: rgba(255,255,255,.22); }
.poster-rich { color: #342e2a; font: 14px/1.85 Georgia, 'Noto Serif SC', serif; overflow-wrap: anywhere; }
.poster-rich :deep(h1), .poster-rich :deep(h2), .poster-rich :deep(h3) { margin: 1.25em 0 .55em; color: #241e1a; line-height: 1.35; }
.poster-rich :deep(h1:first-child), .poster-rich :deep(h2:first-child), .poster-rich :deep(h3:first-child), .poster-rich :deep(p:first-child) { margin-top: 0; }
.poster-rich :deep(p) { margin: .7em 0; }
.poster-rich :deep(img) { display: block; max-width: 100%; height: auto; margin: 18px auto; border: 1px solid #b79d7e; }
.poster-rich :deep(blockquote) { margin: 18px 0; padding: 4px 0 4px 18px; border-left: 3px solid var(--poster-copper); color: #5c5047; }
.poster-rich :deep(a) { color: #805034; text-decoration: none; }

.poster-impressions { display: grid; gap: 14px; }
.poster-impression { display: grid; grid-template-columns: 90px minmax(0, 1fr); gap: 18px; padding: 20px; border: 1px solid #bda98f; background: rgba(255,255,255,.26); }
.poster-impression__mark { display: grid; align-content: start; justify-items: center; gap: 8px; }
.poster-impression__mark > span { color: var(--poster-copper); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .12em; }
.poster-impression__copy h3 { margin: 4px 0 8px; font: 600 20px/1.3 Georgia, 'Noto Serif SC', serif; }
.poster-impression__copy p { margin: 0; color: #554a42; font: 13px/1.7 Georgia, 'Noto Serif SC', serif; }
.poster-impression__image { grid-column: 1 / -1; height: 300px; margin: 0; overflow: hidden; border: 1px solid #ab9579; }
.poster-impression__image :deep(.authenticated-image), .poster-impression__image :deep(img) { width: 100%; height: 100%; object-fit: cover; }
.poster-supplement { margin-top: 28px; padding: 24px; border: 1px dashed #ad9478; }
.poster-supplement > span { display: block; margin-bottom: 16px; color: var(--poster-copper); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .15em; text-transform: uppercase; }
.poster-other-section { background: rgba(225, 207, 180, .45); }
.poster-empty { margin: 0; padding: 28px; border: 1px dashed #bda98f; color: var(--poster-muted); font: 13px/1.6 Georgia, 'Noto Serif SC', serif; text-align: center; }

.poster-footer { display: flex; align-items: flex-end; justify-content: space-between; padding: 30px 42px; background: var(--poster-night); color: #f1e7d8; }
.poster-footer div { display: grid; gap: 5px; }
.poster-footer strong { font: 700 13px/1 ui-monospace, Consolas, monospace; letter-spacing: .18em; }
.poster-footer span, .poster-footer p { margin: 0; color: #afa69c; font-size: 9px; }
</style>
