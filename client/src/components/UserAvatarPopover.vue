<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { resolveApiUrl } from '@/api/item'
import { getCharacterCardPortraitUrl, type CharacterCardSummary } from '@/api/characterCard'
import i18n from '@/i18n'
import { buildNameStyle } from '@/utils/userNameStyle'
import { getCharacterCardDisplayColor } from '@/utils/characterCardColor'
import { getCharacterCardDisplayName } from '@/utils/characterCardDraft'
import { loadUserPopoverData, type UserPopoverData } from '@/utils/userPopoverData'
import UserLevelBadge from './UserLevelBadge.vue'

defineOptions({ inheritAttrs: false })

interface Props {
  userId?: number
  username?: string
  avatarUrl?: string
  nameColor?: string
  nameBold?: boolean
  size?: number
  radius?: string
  showPopover?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  userId: 0,
  username: '',
  avatarUrl: '',
  nameColor: '',
  nameBold: false,
  size: 32,
  radius: '50%',
  showPopover: true,
})

const t = i18n.global.t
const triggerRef = ref<InstanceType<typeof RouterLink> | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const visible = ref(false)
const loading = ref(false)
const loadFailed = ref(false)
const avatarFailed = ref(false)
const data = ref<UserPopoverData | null>(null)
const position = ref({ left: 12, top: 12 })
let closeTimer: ReturnType<typeof setTimeout> | undefined

const canOpen = computed(() => Number.isInteger(props.userId) && props.userId > 0)
const profile = computed(() => data.value?.profile)
const characterCards = computed(() => data.value?.characterCards.slice(0, 3) || [])
const displayName = computed(() => profile.value?.username || props.username || t('common.userPopover.unknownUser'))
const avatarSource = computed(() => resolveApiUrl(profile.value?.avatar || props.avatarUrl))
const displayInitial = computed(() => displayName.value.trim().charAt(0).toUpperCase() || 'U')
const displayNameStyle = computed(() => buildNameStyle(
  profile.value?.name_color || props.nameColor,
  profile.value?.name_bold ?? props.nameBold,
))
const triggerStyle = computed(() => {
  const size = `${Math.max(1, props.size)}px`
  return {
    width: size,
    height: size,
    minWidth: size,
    maxWidth: size,
    minHeight: size,
    maxHeight: size,
    borderRadius: props.radius,
  }
})

function triggerElement(): HTMLElement | null {
  return (triggerRef.value as any)?.$el || null
}

function clearCloseTimer() {
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = undefined
  }
}

async function loadData() {
  if (!canOpen.value || data.value || loading.value) return
  loading.value = true
  loadFailed.value = false
  try {
    data.value = await loadUserPopoverData(props.userId)
  } catch (error) {
    console.error('加载用户悬浮信息失败:', error)
    loadFailed.value = true
  } finally {
    loading.value = false
    await nextTick()
    updatePosition()
  }
}

async function openPopover() {
  if (!props.showPopover || !canOpen.value) return
  clearCloseTimer()
  visible.value = true
  await nextTick()
  updatePosition()
  void loadData()
}

function closePopover() {
  clearCloseTimer()
  visible.value = false
}

function scheduleClose() {
  if (!props.showPopover) return
  clearCloseTimer()
  closeTimer = setTimeout(closePopover, 140)
}

function updatePosition() {
  if (!visible.value) return
  const trigger = triggerElement()
  if (!trigger) return

  const rect = trigger.getBoundingClientRect()
  const popoverWidth = Math.min(360, window.innerWidth - 24)
  const popoverHeight = popoverRef.value?.offsetHeight || 320
  const edge = 12
  const gap = 10
  const left = Math.min(
    Math.max(edge, rect.left + (rect.width - popoverWidth) / 2),
    window.innerWidth - popoverWidth - edge,
  )
  const hasRoomBelow = rect.bottom + gap + popoverHeight <= window.innerHeight - edge
  const top = hasRoomBelow
    ? rect.bottom + gap
    : Math.max(edge, rect.top - gap - popoverHeight)
  position.value = { left, top }
}

function handleWindowChange() {
  if (visible.value) updatePosition()
}

function portraitUrl(card: CharacterCardSummary) {
  return resolveApiUrl(getCharacterCardPortraitUrl(card, { w: 112, q: 76 }))
}

watch(() => props.avatarUrl, () => {
  avatarFailed.value = false
})

watch(() => props.userId, () => {
  data.value = null
  loadFailed.value = false
  closePopover()
})

onMounted(() => {
  window.addEventListener('resize', handleWindowChange)
  window.addEventListener('scroll', handleWindowChange, true)
})

onBeforeUnmount(() => {
  clearCloseTimer()
  window.removeEventListener('resize', handleWindowChange)
  window.removeEventListener('scroll', handleWindowChange, true)
})
</script>

<template>
  <RouterLink
    v-if="canOpen"
    ref="triggerRef"
    v-bind="$attrs"
    class="user-avatar-popover__trigger"
    :style="triggerStyle"
    :to="`/user/${userId}`"
    :aria-label="t('common.userPopover.openProfile', { name: displayName })"
    :aria-expanded="showPopover ? visible : undefined"
    :aria-haspopup="showPopover ? 'dialog' : undefined"
    @click.stop
    @mouseenter="openPopover"
    @mouseleave="scheduleClose"
    @focus="openPopover"
    @blur="scheduleClose"
    @keydown.esc="closePopover"
  >
    <img
      v-if="avatarSource && !avatarFailed"
      :src="avatarSource"
      :alt="t('common.userPopover.avatarAlt', { name: displayName })"
      loading="lazy"
      @error="avatarFailed = true"
    />
    <span v-else>{{ displayInitial }}</span>
  </RouterLink>
  <span v-else v-bind="$attrs" class="user-avatar-popover__trigger user-avatar-popover__trigger--disabled" :style="triggerStyle">
    <img
      v-if="avatarSource && !avatarFailed"
      :src="avatarSource"
      :alt="t('common.userPopover.avatarAlt', { name: displayName })"
      loading="lazy"
      @error="avatarFailed = true"
    />
    <span v-else>{{ displayInitial }}</span>
  </span>

  <Teleport to="body">
    <Transition name="user-popover">
      <aside
        v-if="showPopover && visible"
        ref="popoverRef"
        class="user-avatar-popover"
        :style="{ left: `${position.left}px`, top: `${position.top}px` }"
        role="dialog"
        :aria-label="t('common.userPopover.aria', { name: displayName })"
        @click.stop
        @mouseenter="clearCloseTimer"
        @mouseleave="scheduleClose"
      >
        <div class="user-avatar-popover__banner">
          <div class="user-avatar-popover__portrait">
            <img
              v-if="avatarSource && !avatarFailed"
              :src="avatarSource"
              :alt="t('common.userPopover.avatarAlt', { name: displayName })"
              @error="avatarFailed = true"
            />
            <span v-else>{{ displayInitial }}</span>
          </div>
          <div class="user-avatar-popover__identity">
            <strong :style="displayNameStyle">{{ displayName }}</strong>
            <UserLevelBadge
              :level="profile?.forum_level"
              :name="profile?.forum_level_name"
              :color="profile?.forum_level_color"
              :bold="profile?.forum_level_bold"
              size="xs"
            />
          </div>
        </div>

        <div v-if="loading" class="user-avatar-popover__state">
          <i class="ri-loader-4-line user-avatar-popover__spinner" aria-hidden="true"></i>
          {{ t('common.userPopover.loading') }}
        </div>
        <div v-else-if="loadFailed" class="user-avatar-popover__state user-avatar-popover__state--error">
          <i class="ri-error-warning-line" aria-hidden="true"></i>
          {{ t('common.userPopover.failed') }}
        </div>
        <template v-else-if="profile">
          <p class="user-avatar-popover__bio">
            {{ profile.bio || t('common.userPopover.noBio') }}
          </p>
          <dl class="user-avatar-popover__stats">
            <div>
              <dt>{{ t('common.userPopover.posts') }}</dt>
              <dd>{{ profile.post_count || 0 }}</dd>
            </div>
            <div>
              <dt>{{ t('common.userPopover.items') }}</dt>
              <dd>{{ profile.item_count || 0 }}</dd>
            </div>
            <div>
              <dt>{{ t('common.userPopover.guilds') }}</dt>
              <dd>{{ profile.guild_count || 0 }}</dd>
            </div>
          </dl>

          <section class="user-avatar-popover__cards">
            <div class="user-avatar-popover__section-title">
              <span>{{ t('common.userPopover.publicCards') }}</span>
              <b>{{ data?.characterCards.length || 0 }}</b>
            </div>
            <div v-if="characterCards.length" class="user-avatar-popover__card-list">
              <RouterLink
                v-for="characterCard in characterCards"
                :key="characterCard.id"
                class="user-avatar-popover__card"
                :to="`/character-cards/${characterCard.id}`"
                @click.stop="closePopover"
              >
                <span class="user-avatar-popover__card-art">
                  <img
                    v-if="portraitUrl(characterCard)"
                    :src="portraitUrl(characterCard)"
                    :alt="t('common.userPopover.cardAlt', { name: getCharacterCardDisplayName(characterCard) })"
                    loading="lazy"
                  />
                  <i v-else class="ri-user-star-line" aria-hidden="true"></i>
                </span>
                <span class="user-avatar-popover__card-copy">
                  <strong :style="{ color: getCharacterCardDisplayColor(characterCard) }">
                    {{ getCharacterCardDisplayName(characterCard) }}
                  </strong>
                  <small>{{ characterCard.title || characterCard.race || t('common.userPopover.cardFallback') }}</small>
                </span>
                <i class="ri-arrow-right-up-line" aria-hidden="true"></i>
              </RouterLink>
            </div>
            <p v-else class="user-avatar-popover__cards-empty">{{ t('common.userPopover.noCards') }}</p>
          </section>

          <RouterLink class="user-avatar-popover__profile-link" :to="`/user/${userId}`" @click.stop="closePopover">
            {{ t('common.userPopover.viewProfile') }}
            <i class="ri-arrow-right-line" aria-hidden="true"></i>
          </RouterLink>
        </template>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.user-avatar-popover__trigger {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  overflow: hidden;
  background: linear-gradient(135deg, var(--color-accent), var(--color-secondary));
  color: inherit;
  cursor: pointer;
  font-weight: 700;
  line-height: 1;
  text-decoration: none;
}

.user-avatar-popover__trigger > img,
.user-avatar-popover__trigger > span {
  width: 100%;
  height: 100%;
}

.user-avatar-popover__trigger > img {
  display: block;
  object-fit: cover;
}

.user-avatar-popover__trigger > span {
  display: grid;
  place-items: center;
}

.user-avatar-popover__trigger:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 3px;
}

.user-avatar-popover__trigger--disabled {
  cursor: default;
}

.user-avatar-popover {
  position: fixed;
  z-index: 1800;
  width: min(360px, calc(100vw - 24px));
  overflow: hidden;
  border: 1px solid var(--color-border-hover);
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  box-shadow: var(--shadow-lg);
}

.user-avatar-popover__banner {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 88px;
  padding: 16px;
  border-bottom: 1px solid var(--gradient-border);
  background:
    radial-gradient(circle at 88% 0%, var(--gradient-surface-hover), transparent 42%),
    linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
  color: var(--gradient-text);
}

.user-avatar-popover__portrait {
  display: grid;
  width: 54px;
  height: 54px;
  flex: 0 0 54px;
  place-items: center;
  overflow: hidden;
  border: 2px solid var(--gradient-border);
  border-radius: 50%;
  background: var(--gradient-surface);
  font-size: 21px;
  font-weight: 800;
}

.user-avatar-popover__portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-avatar-popover__identity {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.user-avatar-popover__identity strong {
  overflow: hidden;
  color: inherit;
  font-size: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-avatar-popover__state {
  display: flex;
  min-height: 160px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-secondary);
  font-size: 12px;
}

.user-avatar-popover__state--error {
  color: var(--btn-danger-bg);
}

.user-avatar-popover__spinner {
  animation: user-popover-spin 0.8s linear infinite;
}

.user-avatar-popover__bio {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  padding: 14px 16px 10px;
  color: var(--color-text-secondary);
  font-size: 11px;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.user-avatar-popover__stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  margin: 0 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-card-bg);
}

.user-avatar-popover__stats div {
  display: grid;
  justify-items: center;
  gap: 1px;
  padding: 8px;
  border-right: 1px solid var(--color-border);
}

.user-avatar-popover__stats div:last-child {
  border-right: 0;
}

.user-avatar-popover__stats dt {
  color: var(--color-text-muted);
  font-size: 9px;
}

.user-avatar-popover__stats dd {
  margin: 0;
  font-size: 13px;
  font-weight: 800;
}

.user-avatar-popover__cards {
  padding: 13px 16px 10px;
}

.user-avatar-popover__section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 7px;
  color: var(--color-text-secondary);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.04em;
}

.user-avatar-popover__section-title b {
  display: grid;
  min-width: 18px;
  height: 18px;
  place-items: center;
  border-radius: 9px;
  background: var(--tag-bg);
  color: var(--tag-text);
  font-size: 9px;
}

.user-avatar-popover__card-list {
  display: grid;
  gap: 5px;
}

.user-avatar-popover__card {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 16px;
  align-items: center;
  gap: 8px;
  min-height: 42px;
  padding: 4px 7px 4px 4px;
  border: 1px solid transparent;
  border-radius: 7px;
  color: var(--color-text-main);
  text-decoration: none;
}

.user-avatar-popover__card:hover,
.user-avatar-popover__card:focus-visible {
  border-color: var(--color-border-hover);
  background: var(--color-card-bg-hover);
  outline: none;
}

.user-avatar-popover__card-art {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  overflow: hidden;
  border-radius: 6px;
  background: var(--icon-bg);
  color: var(--icon-color);
}

.user-avatar-popover__card-art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-avatar-popover__card-copy {
  display: grid;
  min-width: 0;
  gap: 1px;
}

.user-avatar-popover__card-copy strong,
.user-avatar-popover__card-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-avatar-popover__card-copy strong {
  font-size: 11px;
}

.user-avatar-popover__card-copy small,
.user-avatar-popover__card > i {
  color: var(--color-text-muted);
  font-size: 9px;
}

.user-avatar-popover__cards-empty {
  margin: 0;
  padding: 9px;
  border-radius: 7px;
  background: var(--color-card-bg);
  color: var(--color-text-muted);
  font-size: 10px;
  text-align: center;
}

.user-avatar-popover__profile-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 38px;
  padding: 0 16px;
  border-top: 1px solid var(--color-border);
  background: var(--color-card-bg);
  color: var(--link-color);
  font-size: 11px;
  font-weight: 800;
  text-decoration: none;
}

.user-avatar-popover__profile-link:hover {
  background: var(--color-card-bg-hover);
  color: var(--link-hover);
}

.user-popover-enter-active,
.user-popover-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.user-popover-enter-from,
.user-popover-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

@keyframes user-popover-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .user-popover-enter-active,
  .user-popover-leave-active {
    transition: none;
  }
}
</style>
