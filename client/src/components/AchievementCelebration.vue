<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

interface CelebrationItem {
  id: number
  title: string
  message: string
  icon?: string
  rarity?: string
  completedAt: string
}

const props = defineProps<{
  celebration: CelebrationItem | null
}>()

const emit = defineEmits<{
  dismiss: []
}>()

const overlayRef = ref<HTMLElement | null>(null)
let previousBodyOverflow = ''
let bodyLocked = false

const rarityTheme = computed(() => {
  const themes: Record<string, { color: string; glow: string; name: string }> = {
    common: { color: '#D8C4A3', glow: 'rgba(216, 196, 163, 0.34)', name: '普通成就' },
    rare: { color: '#68C3FF', glow: 'rgba(90, 183, 255, 0.42)', name: '稀有成就' },
    fine: { color: '#65DF7A', glow: 'rgba(85, 214, 107, 0.4)', name: '精良成就' },
    epic: { color: '#C184FF', glow: 'rgba(178, 108, 255, 0.46)', name: '史诗成就' },
    legendary: { color: '#FFBC50', glow: 'rgba(255, 178, 62, 0.52)', name: '传说成就' },
  }
  return themes[props.celebration?.rarity || 'common'] || themes.common
})

const completedAtLabel = computed(() => {
  if (!props.celebration?.completedAt) return ''
  const date = new Date(props.celebration.completedAt)
  if (Number.isNaN(date.getTime())) return props.celebration.completedAt
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
})

function dismiss() {
  emit('dismiss')
}

watch(() => props.celebration, async (celebration) => {
  if (celebration) {
    if (!bodyLocked) {
      previousBodyOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      bodyLocked = true
    }
    await nextTick()
    overlayRef.value?.focus({ preventScroll: true })
    return
  }
  if (bodyLocked) {
    document.body.style.overflow = previousBodyOverflow
    bodyLocked = false
  }
}, { immediate: true })

onBeforeUnmount(() => {
  if (bodyLocked) document.body.style.overflow = previousBodyOverflow
})
</script>

<template>
  <Teleport to="body">
    <Transition name="achievement-screen" mode="out-in">
      <div
        v-if="celebration"
        :key="celebration.id"
        ref="overlayRef"
        class="achievement-screen"
        :style="{
          '--achievement-color': rarityTheme.color,
          '--achievement-glow': rarityTheme.glow,
        }"
        role="dialog"
        aria-modal="true"
        :aria-label="`成就完成：${celebration.title}。点击任意区域关闭`"
        tabindex="0"
        @click="dismiss"
        @keydown.esc.prevent="dismiss"
        @keydown.enter.prevent="dismiss"
        @keydown.space.prevent="dismiss"
      >
        <div class="achievement-screen__veil" aria-hidden="true"></div>
        <div class="achievement-screen__rays" aria-hidden="true"></div>
        <div class="achievement-screen__stage">
          <p class="achievement-screen__kicker">Achievement complete</p>
          <h2>恭喜你已完成</h2>

          <div class="achievement-screen__seal" aria-hidden="true">
            <span class="achievement-screen__orbit"></span>
            <span class="achievement-screen__icon">
              <i :class="celebration.icon || 'ri-medal-line'"></i>
            </span>
          </div>

          <div class="achievement-screen__copy">
            <span class="achievement-screen__rarity">{{ rarityTheme.name }}</span>
            <h1>{{ celebration.title }}</h1>
            <p>{{ celebration.message }}</p>
          </div>

          <div class="achievement-screen__time">
            <span>完成时间</span>
            <time :datetime="celebration.completedAt">{{ completedAtLabel }}</time>
          </div>

          <p class="achievement-screen__dismiss-hint">
            <i class="ri-mouse-line" aria-hidden="true"></i>
            点击任意区域关闭
          </p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.achievement-screen {
  --achievement-color: #D8C4A3;
  --achievement-glow: rgba(216, 196, 163, 0.34);
  position: fixed;
  inset: 0;
  z-index: 12000;
  display: grid;
  overflow: hidden;
  place-items: center;
  padding: max(28px, env(safe-area-inset-top)) max(24px, env(safe-area-inset-right)) max(28px, env(safe-area-inset-bottom)) max(24px, env(safe-area-inset-left));
  color: #FFF7E7;
  cursor: pointer;
  isolation: isolate;
  outline: none;
}

.achievement-screen__veil {
  position: absolute;
  inset: 0;
  z-index: -3;
  background:
    radial-gradient(circle at 50% 44%, color-mix(in srgb, var(--achievement-color) 17%, transparent), transparent 32%),
    linear-gradient(180deg, rgba(18, 12, 8, 0.9), rgba(8, 6, 5, 0.96));
  backdrop-filter: blur(9px) saturate(0.72);
  -webkit-backdrop-filter: blur(9px) saturate(0.72);
}

.achievement-screen__rays {
  position: absolute;
  top: 50%;
  left: 50%;
  z-index: -2;
  width: min(780px, 88vmin);
  aspect-ratio: 1;
  border-radius: 50%;
  background: repeating-conic-gradient(
    from 0deg,
    color-mix(in srgb, var(--achievement-color) 18%, transparent) 0deg 1deg,
    transparent 1deg 12deg
  );
  mask-image: radial-gradient(circle, transparent 0 20%, #000 34%, transparent 72%);
  opacity: 0.65;
  transform: translate(-50%, -50%);
  animation: achievement-rays 24s linear infinite;
}

.achievement-screen__stage {
  position: relative;
  width: min(680px, 92vw);
  padding: clamp(34px, 6vh, 62px) clamp(26px, 7vw, 72px) 32px;
  text-align: center;
  filter: drop-shadow(0 26px 60px rgba(0, 0, 0, 0.45));
  animation: achievement-stage-in 680ms cubic-bezier(0.18, 0.9, 0.24, 1.15) both;
}

.achievement-screen__stage::before,
.achievement-screen__stage::after {
  position: absolute;
  content: '';
  pointer-events: none;
}

.achievement-screen__stage::before {
  inset: 0;
  z-index: -1;
  border: 1px solid color-mix(in srgb, var(--achievement-color) 58%, transparent);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.07), transparent 38%),
    linear-gradient(160deg, rgba(68, 43, 24, 0.96), rgba(29, 20, 14, 0.98));
  clip-path: polygon(26px 0, calc(100% - 26px) 0, 100% 26px, 100% calc(100% - 26px), calc(100% - 26px) 100%, 26px 100%, 0 calc(100% - 26px), 0 26px);
  box-shadow: inset 0 0 0 5px rgba(255, 255, 255, 0.025), inset 0 0 70px var(--achievement-glow);
}

.achievement-screen__stage::after {
  inset: 12px 32px auto;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--achievement-color), transparent);
  box-shadow: 0 calc(clamp(310px, 56vh, 470px)) 0 color-mix(in srgb, var(--achievement-color) 70%, transparent);
  opacity: 0.78;
}

.achievement-screen__kicker {
  margin: 0 0 7px;
  color: var(--achievement-color);
  font: 700 11px/1.2 Georgia, 'Times New Roman', serif;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  animation: achievement-copy-in 450ms 160ms both;
}

.achievement-screen h2 {
  margin: 0;
  color: rgba(255, 247, 231, 0.78);
  font: 600 clamp(16px, 2vw, 20px)/1.3 'Microsoft YaHei', sans-serif;
  letter-spacing: 0.16em;
  animation: achievement-copy-in 450ms 210ms both;
}

.achievement-screen__seal {
  position: relative;
  display: grid;
  width: clamp(112px, 18vw, 150px);
  aspect-ratio: 1;
  margin: clamp(24px, 4vh, 36px) auto 24px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--achievement-color) 72%, transparent);
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--achievement-color) 20%, #4B2B16) 0 42%, #24170F 43% 100%);
  box-shadow:
    0 0 0 8px rgba(255, 255, 255, 0.025),
    0 0 0 10px color-mix(in srgb, var(--achievement-color) 34%, transparent),
    0 0 48px var(--achievement-glow);
  animation: achievement-seal-in 820ms 120ms cubic-bezier(0.16, 1.08, 0.32, 1.22) both;
}

.achievement-screen__orbit {
  position: absolute;
  inset: -18px;
  border: 1px dashed color-mix(in srgb, var(--achievement-color) 56%, transparent);
  border-radius: 50%;
  animation: achievement-orbit 18s linear infinite;
}

.achievement-screen__icon {
  display: grid;
  width: 70%;
  aspect-ratio: 1;
  place-items: center;
  border-radius: 50%;
  color: #26160C;
  background: linear-gradient(145deg, #FFF2BF, var(--achievement-color) 58%, #8B5528);
  box-shadow: inset 0 2px 1px rgba(255, 255, 255, 0.62), inset 0 -5px 12px rgba(63, 31, 10, 0.34);
}

.achievement-screen__icon i {
  font-size: clamp(42px, 7vw, 58px);
  filter: drop-shadow(0 1px 0 rgba(255, 255, 255, 0.24));
}

.achievement-screen__copy {
  animation: achievement-copy-in 520ms 360ms both;
}

.achievement-screen__rarity {
  display: inline-block;
  margin-bottom: 7px;
  color: var(--achievement-color);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
}

.achievement-screen h1 {
  margin: 0;
  color: #FFF7E7;
  font: 700 clamp(32px, 6vw, 52px)/1.15 Georgia, 'Microsoft YaHei', serif;
  letter-spacing: 0.05em;
  text-wrap: balance;
  text-shadow: 0 3px 24px var(--achievement-glow);
}

.achievement-screen__copy p {
  max-width: 480px;
  margin: 13px auto 0;
  color: rgba(255, 247, 231, 0.72);
  font-size: clamp(13px, 2vw, 15px);
  line-height: 1.7;
  text-wrap: balance;
}

.achievement-screen__time {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-top: 25px;
  padding: 8px 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 247, 231, 0.58);
  font-size: 11px;
  animation: achievement-copy-in 480ms 450ms both;
}

.achievement-screen__time time {
  color: rgba(255, 247, 231, 0.9);
  font-family: ui-monospace, 'Cascadia Code', Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.achievement-screen__dismiss-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  margin: 27px 0 0;
  color: rgba(255, 247, 231, 0.48);
  font-size: 12px;
  letter-spacing: 0.08em;
  animation: achievement-hint 2.2s 1s ease-in-out infinite;
}

.achievement-screen-enter-active,
.achievement-screen-leave-active {
  transition: opacity 240ms ease;
}

.achievement-screen-enter-from,
.achievement-screen-leave-to {
  opacity: 0;
}

@keyframes achievement-stage-in {
  from { opacity: 0; transform: translateY(28px) scale(0.82); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@keyframes achievement-seal-in {
  from { opacity: 0; transform: scale(0.28) rotate(-18deg); }
  72% { opacity: 1; transform: scale(1.08) rotate(2deg); }
  to { opacity: 1; transform: scale(1) rotate(0); }
}

@keyframes achievement-copy-in {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes achievement-rays {
  to { transform: translate(-50%, -50%) rotate(360deg); }
}

@keyframes achievement-orbit {
  to { transform: rotate(-360deg); }
}

@keyframes achievement-hint {
  0%, 100% { opacity: 0.46; }
  50% { opacity: 0.88; }
}

@media (max-width: 560px) {
  .achievement-screen {
    padding: max(18px, env(safe-area-inset-top)) 14px max(18px, env(safe-area-inset-bottom));
  }

  .achievement-screen__stage {
    width: min(100%, 440px);
    padding: 34px 22px 25px;
  }

  .achievement-screen__stage::after {
    inset-inline: 22px;
  }

  .achievement-screen__dismiss-hint i {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .achievement-screen *,
  .achievement-screen,
  .achievement-screen-enter-active,
  .achievement-screen-leave-active {
    animation: none !important;
    transition-duration: 1ms !important;
  }
}
</style>
