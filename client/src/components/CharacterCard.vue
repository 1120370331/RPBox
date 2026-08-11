<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Character } from '@/api/character'
import type { CharacterCardSummary } from '@/api/characterCard'
import { getCharacterCardDisplayColor, normalizeCharacterCardHexForCSS } from '@/utils/characterCardColor'
import { getCharacterCardDisplayName } from '@/utils/characterCardDraft'
import CharacterCardImpressionMark from './character-cards/CharacterCardImpressionMark.vue'
import CharacterCardPortrait from './character-cards/CharacterCardPortrait.vue'
import RButton from './RButton.vue'
import WowIcon from './WowIcon.vue'

interface Props {
  visible: boolean
  character?: Character
  characterCard?: CharacterCardSummary
  speaker?: string
  position?: { x: number; y: number }
  editable?: boolean
}

interface InfoItem {
  label: string
  value: string
}

interface GlanceSlot {
  slot: number
  icon: string
  iconImageUrl: string
  title: string
  text: string
  rpbox: boolean
}

const CARD_WIDTH = 340
const VIEWPORT_GAP = 12

const props = withDefaults(defineProps<Props>(), {
  editable: true,
})
const emit = defineEmits<{
  'update:visible': [value: boolean]
  edit: [character: Character]
}>()
const { t } = useI18n()
let clickListenerTimer: number | null = null

const isVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})
const isRPBoxCard = computed(() => Boolean(props.characterCard))
const displayName = computed(() => {
  if (props.characterCard) return getCharacterCardDisplayName(props.characterCard)
  if (!props.character) return props.speaker || t('archives.characterPopover.unknown')
  return props.character.custom_name
    || [props.character.first_name, props.character.last_name].filter(Boolean).join(' ')
    || props.speaker
    || t('archives.characterPopover.unknown')
})
const displayColor = computed(() => {
  if (props.characterCard) return getCharacterCardDisplayColor(props.characterCard)
  return normalizeCharacterCardHexForCSS(props.character?.custom_color || props.character?.color)
})
const displayIcon = computed(() => {
  if (props.characterCard) return props.characterCard.icon || ''
  return props.character?.custom_avatar || props.character?.icon || ''
})
const title = computed(() => props.characterCard?.title || props.character?.title || '')
const fullTitle = computed(() => props.characterCard?.full_title || props.character?.full_title || '')
const summary = computed(() => props.characterCard?.summary?.trim() || '')
const portraitAvailable = computed(() => Boolean(props.characterCard?.portrait_image_url))

const infoItems = computed<InfoItem[]>(() => {
  const source = props.characterCard || props.character
  if (!source) return []
  const values: Array<[string, string | undefined | null]> = [
    [t('archives.characterPopover.race'), source.race],
    [t('archives.characterPopover.class'), source.class],
    [t('archives.characterPopover.age'), source.age],
    [t('archives.characterPopover.height'), source.height],
    [t('archives.characterPopover.eyes'), source.eye_color],
  ]
  if (props.characterCard) {
    values.push(
      [t('archives.characterPopover.weight'), props.characterCard.weight],
      [t('archives.characterPopover.birthplace'), props.characterCard.birthplace],
      [t('archives.characterPopover.residence'), props.characterCard.residence],
    )
  }
  return values
    .filter((item): item is [string, string] => Boolean(item[1]?.trim()))
    .map(([label, value]) => ({ label, value }))
})

const glanceSlots = computed<GlanceSlot[]>(() => {
  if (props.characterCard) {
    return (props.characterCard.impressions || [])
      .filter((slot) => slot.active)
      .sort((a, b) => a.slot - b.slot)
      .map((slot) => ({
        slot: slot.slot,
        icon: slot.trp3_icon || '',
        iconImageUrl: slot.icon_image_url || '',
        title: slot.title || t('archives.characterPopover.unnamedImpression'),
        text: slot.text || '',
        rpbox: true,
      }))
  }
  if (!props.character?.misc_info) return []
  try {
    const misc = JSON.parse(props.character.misc_info)
    if (!misc.PE) return []
    const slots: GlanceSlot[] = []
    for (let slot = 1; slot <= 5; slot += 1) {
      const value = misc.PE[String(slot)]
      if (!value?.AC) continue
      slots.push({
        slot,
        icon: value.IC || '',
        iconImageUrl: '',
        title: value.TI || t('archives.characterPopover.unnamedImpression'),
        text: value.TX || '',
        rpbox: false,
      })
    }
    return slots
  } catch {
    return []
  }
})

const cardStyle = computed(() => {
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  if (!props.position) {
    return {
      top: `${VIEWPORT_GAP}px`,
      left: `${Math.max(VIEWPORT_GAP, (viewportWidth - CARD_WIDTH) / 2)}px`,
    }
  }

  let left = props.position.x + VIEWPORT_GAP
  if (left + CARD_WIDTH > viewportWidth - VIEWPORT_GAP) {
    left = props.position.x - CARD_WIDTH - VIEWPORT_GAP
  }
  left = Math.max(VIEWPORT_GAP, Math.min(left, viewportWidth - CARD_WIDTH - VIEWPORT_GAP))
  const maxTop = Math.max(VIEWPORT_GAP, viewportHeight - Math.min(560, viewportHeight - VIEWPORT_GAP * 2) - VIEWPORT_GAP)
  const top = Math.max(VIEWPORT_GAP, Math.min(props.position.y, maxTop))
  return { top: `${top}px`, left: `${left}px` }
})

function close() {
  isVisible.value = false
}

function handleEdit() {
  if (props.character) emit('edit', props.character)
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target
  if (target instanceof Element && !target.closest('.story-character-card')) close()
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

watch(isVisible, (visible) => {
  if (clickListenerTimer !== null) {
    window.clearTimeout(clickListenerTimer)
    clickListenerTimer = null
  }
  if (visible) {
    clickListenerTimer = window.setTimeout(() => {
      clickListenerTimer = null
      document.addEventListener('click', handleClickOutside)
    }, 0)
    document.addEventListener('keydown', handleKeydown)
    return
  }
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
}, { immediate: true })

onBeforeUnmount(() => {
  if (clickListenerTimer !== null) window.clearTimeout(clickListenerTimer)
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="story-character-card">
      <aside
        v-if="isVisible"
        class="story-character-card"
        :style="cardStyle"
        role="dialog"
        :aria-label="displayName"
        data-testid="story-character-card"
      >
        <span class="card-rail" aria-hidden="true"></span>
        <header class="card-header">
          <div class="avatar" :class="{ portrait: portraitAvailable }">
            <CharacterCardPortrait
              v-if="props.characterCard && portraitAvailable"
              :card="props.characterCard"
              :alt="t('archives.characterPopover.portraitAlt', { name: displayName })"
              :width="180"
              :quality="76"
            >
              <template #fallback>
                <span class="avatar-text">{{ displayName.charAt(0) }}</span>
              </template>
            </CharacterCardPortrait>
            <WowIcon v-else-if="displayIcon" :icon="displayIcon" :size="68" :fallback="displayName.charAt(0)" />
            <span v-else class="avatar-text">{{ displayName.charAt(0) }}</span>
          </div>

          <div class="identity">
            <span class="source-badge">
              <i :class="isRPBoxCard ? 'ri-id-card-line' : 'ri-gamepad-line'" aria-hidden="true"></i>
              {{ t(isRPBoxCard ? 'archives.characterPopover.rpbox' : 'archives.characterPopover.trp3') }}
            </span>
            <h3 :style="displayColor ? { color: displayColor } : undefined">{{ displayName }}</h3>
            <p v-if="title">{{ title }}</p>
            <small v-if="fullTitle">{{ fullTitle }}</small>
          </div>

          <button
            type="button"
            class="close-button"
            :aria-label="t('archives.characterPopover.close')"
            @click="close"
          >
            <i class="ri-close-line" aria-hidden="true"></i>
          </button>
        </header>

        <div class="card-body">
          <dl v-if="infoItems.length" class="info-grid">
            <div v-for="item in infoItems" :key="item.label" class="info-item">
              <dt>{{ item.label }}</dt>
              <dd>{{ item.value }}</dd>
            </div>
          </dl>

          <section v-if="summary" class="summary-section">
            <h4>{{ t('archives.characterPopover.summary') }}</h4>
            <p>{{ summary }}</p>
          </section>

          <section v-if="glanceSlots.length" class="glance-section">
            <h4>{{ t('archives.characterPopover.firstImpression') }}</h4>
            <div class="glance-list">
              <article v-for="slot in glanceSlots" :key="slot.slot" class="glance-item">
                <CharacterCardImpressionMark
                  v-if="slot.rpbox"
                  :icon-image-url="slot.iconImageUrl"
                  :trp3-icon="slot.icon"
                  :fallback-label="String(slot.slot)"
                  :size="30"
                />
                <WowIcon v-else :icon="slot.icon" :size="30" :fallback="String(slot.slot)" />
                <div>
                  <strong>{{ slot.title }}</strong>
                  <p v-if="slot.text">{{ slot.text }}</p>
                </div>
              </article>
            </div>
          </section>

          <p v-if="!infoItems.length && !summary && !glanceSlots.length" class="empty-state">
            {{ t('archives.characterPopover.empty') }}
          </p>
        </div>

        <footer v-if="editable && character && !characterCard" class="card-footer">
          <RButton size="small" @click="handleEdit">
            <i class="ri-edit-line" aria-hidden="true"></i>
            {{ t('archives.characterPopover.editStoryCharacter') }}
          </RButton>
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.story-character-card {
  position: fixed;
  z-index: 2500;
  display: flex;
  width: min(340px, calc(100vw - 24px));
  max-height: min(560px, calc(100vh - 24px));
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  box-shadow: var(--shadow-lg);
}

.card-rail {
  position: absolute;
  z-index: 2;
  top: 0;
  bottom: 0;
  left: 0;
  width: 4px;
  background: linear-gradient(var(--gradient-start), var(--color-accent) 50%, var(--gradient-end));
}

.card-header {
  position: relative;
  display: grid;
  grid-template-columns: 68px minmax(0, 1fr) 28px;
  align-items: center;
  gap: 12px;
  padding: 16px 12px 16px 18px;
  border-bottom: 1px solid var(--color-border);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--color-accent) 9%, transparent), transparent 58%),
    var(--color-card-bg);
}

.avatar {
  display: grid;
  width: 68px;
  height: 68px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-sm);
  background:
    radial-gradient(circle, color-mix(in srgb, var(--color-accent) 24%, transparent), transparent 58%),
    var(--gradient-end);
  color: var(--gradient-text);
  box-shadow: var(--shadow-sm);
}

.avatar :deep(.wow-icon),
.avatar :deep(.character-portrait-state),
.avatar :deep(img) {
  width: 100%;
  height: 100%;
  border-radius: 0;
  object-fit: cover;
}

.avatar-text {
  color: var(--gradient-text);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 25px;
  font-weight: 600;
}

.identity { min-width: 0; }

.source-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--color-accent);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: .1em;
  text-transform: uppercase;
}

.identity h3 {
  overflow: hidden;
  margin: 5px 0 1px;
  color: var(--color-text-main);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 19px;
  font-weight: 600;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity p,
.identity small {
  display: block;
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity p { color: var(--color-secondary); font-weight: 600; }

.close-button {
  display: grid;
  width: 28px;
  height: 28px;
  padding: 0;
  place-items: center;
  align-self: start;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.close-button:hover,
.close-button:focus-visible {
  border-color: var(--color-border-hover);
  outline: none;
  background: var(--btn-outline-hover);
  color: var(--color-text-main);
}

.card-body {
  min-height: 0;
  overflow-y: auto;
  padding: 15px 18px 17px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px 14px;
  margin: 0;
}

.info-item { min-width: 0; }
.info-item dt,
.summary-section h4,
.glance-section h4 {
  color: var(--color-text-secondary);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: .09em;
  text-transform: uppercase;
}
.info-item dd {
  overflow: hidden;
  margin: 2px 0 0;
  color: var(--color-text-main);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-section,
.glance-section {
  margin-top: 14px;
  padding-top: 13px;
  border-top: 1px solid var(--color-border);
}

.summary-section p {
  display: -webkit-box;
  overflow: hidden;
  margin: 7px 0 0;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
}

.glance-list {
  display: grid;
  gap: 9px;
  margin-top: 9px;
}

.glance-item {
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr);
  align-items: start;
  gap: 9px;
}

.glance-item :deep(.wow-icon) { border-radius: 7px; }
.glance-item strong {
  display: block;
  overflow: hidden;
  color: var(--color-text-main);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.glance-item p {
  display: -webkit-box;
  overflow: hidden;
  margin: 2px 0 0;
  color: var(--color-text-secondary);
  font-size: 10px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.empty-state {
  margin: 0;
  padding: 12px;
  border-radius: var(--radius-sm);
  background: var(--color-card-bg);
  color: var(--color-text-secondary);
  font-size: 11px;
  text-align: center;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  padding: 10px 16px;
  border-top: 1px solid var(--color-border);
  background: var(--color-card-bg);
}

.story-character-card-enter-active,
.story-character-card-leave-active { transition: opacity 140ms ease, transform 140ms ease; }
.story-character-card-enter-from,
.story-character-card-leave-to { opacity: 0; transform: translateY(3px); }

@media (max-width: 420px) {
  .card-header { grid-template-columns: 58px minmax(0, 1fr) 28px; }
  .avatar { width: 58px; height: 58px; }
}

@media (prefers-reduced-motion: reduce) {
  .story-character-card-enter-active,
  .story-character-card-leave-active { transition: none; }
}
</style>
