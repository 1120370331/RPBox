<script setup lang="ts">
import { computed } from 'vue'
import {
  ACHIEVEMENT_CATEGORY_META,
  ACHIEVEMENT_RARITY_META,
  type AchievementDefinition,
} from '@/data/achievements'

const props = withDefaults(defineProps<{
  achievement: AchievementDefinition
  earned?: boolean
  size?: 'sm' | 'md' | 'lg'
}>(), {
  earned: false,
  size: 'md',
})

const rarityMeta = computed(() => ACHIEVEMENT_RARITY_META[props.achievement.rarity])
const categoryMeta = computed(() => ACHIEVEMENT_CATEGORY_META[props.achievement.category])
</script>

<template>
  <span
    class="achievement-medal"
    :class="[
      `achievement-medal--${size}`,
      `achievement-medal--${achievement.rarity}`,
      `achievement-medal--shape-${categoryMeta.shape}`,
      { 'achievement-medal--locked': !earned },
    ]"
    :style="{
      '--achievement-edge': rarityMeta.edge,
      '--achievement-glow': rarityMeta.glow,
      '--achievement-text': rarityMeta.text,
    }"
    :title="`${achievement.title} · ${rarityMeta.label}`"
  >
    <span class="achievement-medal__edge"></span>
    <span class="achievement-medal__core">
      <i :class="achievement.icon"></i>
    </span>
    <span class="achievement-medal__shine"></span>
  </span>
</template>

<style scoped>
.achievement-medal {
  --achievement-size: 58px;
  width: var(--achievement-size);
  height: var(--achievement-size);
  position: relative;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  filter: drop-shadow(0 8px 14px var(--achievement-glow));
}

.achievement-medal--sm {
  --achievement-size: 38px;
}

.achievement-medal--lg {
  --achievement-size: 70px;
}

.achievement-medal__edge,
.achievement-medal__core,
.achievement-medal__shine {
  position: absolute;
  inset: 0;
  clip-path: inherit;
}

.achievement-medal__edge {
  background:
    radial-gradient(circle at 28% 22%, rgba(255, 255, 255, 0.86), transparent 22%),
    linear-gradient(145deg, rgba(255, 255, 255, 0.5), transparent 28%),
    linear-gradient(135deg, var(--achievement-edge), color-mix(in srgb, var(--achievement-edge) 52%, #1D1612));
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.42),
    inset 0 -6px 14px rgba(29, 22, 18, 0.28);
}

.achievement-medal__core {
  inset: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 35% 28%, rgba(255, 255, 255, 0.7), transparent 22%),
    linear-gradient(145deg, #FFF4DF, #DCC39A 62%, #7A5433);
  color: var(--achievement-text);
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.5),
    inset 0 -5px 12px rgba(77, 48, 28, 0.22);
}

.achievement-medal__core i {
  font-size: calc(var(--achievement-size) * 0.36);
  position: relative;
  z-index: 2;
}

.achievement-medal__shine {
  background:
    linear-gradient(120deg, transparent 20%, rgba(255, 255, 255, 0.46) 42%, transparent 58%);
  mix-blend-mode: screen;
  opacity: 0;
  transform: translateX(-28%);
  transition:
    opacity 0.22s ease,
    transform 0.35s ease;
}

.achievement-medal:active .achievement-medal__shine {
  opacity: 0.72;
  transform: translateX(18%);
}

.achievement-medal--locked {
  opacity: 0.48;
  filter: grayscale(0.84) drop-shadow(0 6px 12px rgba(44, 34, 26, 0.1));
}

.achievement-medal--locked .achievement-medal__core {
  background:
    radial-gradient(circle at 35% 28%, rgba(255, 255, 255, 0.58), transparent 22%),
    linear-gradient(145deg, #E5DDD2, #A89C8D);
  color: #6C6258;
}

.achievement-medal--shape-round {
  clip-path: circle(50% at 50% 50%);
}

.achievement-medal--shape-calendar {
  clip-path: polygon(18% 8%, 82% 8%, 92% 26%, 86% 88%, 50% 100%, 14% 88%, 8% 26%);
}

.achievement-medal--shape-hex {
  clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0% 50%);
}

.achievement-medal--shape-shield {
  clip-path: polygon(50% 2%, 92% 16%, 86% 66%, 50% 100%, 14% 66%, 8% 16%);
}

.achievement-medal--shape-diamond {
  clip-path: polygon(50% 0%, 96% 50%, 50% 100%, 4% 50%);
}

.achievement-medal--shape-heart {
  clip-path: polygon(50% 92%, 12% 56%, 6% 30%, 22% 10%, 44% 18%, 50% 30%, 56% 18%, 78% 10%, 94% 30%, 88% 56%);
}

.achievement-medal--shape-scroll {
  clip-path: polygon(18% 8%, 82% 8%, 94% 20%, 84% 34%, 92% 50%, 84% 66%, 94% 80%, 82% 92%, 18% 92%, 6% 80%, 16% 66%, 8% 50%, 16% 34%, 6% 20%);
}

.achievement-medal--shape-oval {
  clip-path: ellipse(42% 50% at 50% 50%);
}

.achievement-medal--shape-crown {
  clip-path: polygon(8% 28%, 26% 48%, 38% 12%, 50% 44%, 62% 12%, 74% 48%, 92% 28%, 84% 92%, 16% 92%);
}

.achievement-medal--shape-star {
  clip-path: polygon(50% 0%, 62% 32%, 96% 32%, 69% 52%, 80% 90%, 50% 68%, 20% 90%, 31% 52%, 4% 32%, 38% 32%);
}
</style>
