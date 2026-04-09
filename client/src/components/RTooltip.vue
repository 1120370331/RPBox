<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  content: string
  position?: 'top' | 'bottom' | 'left' | 'right'
}

withDefaults(defineProps<Props>(), {
  position: 'top',
})

const visible = ref(false)
</script>

<template>
  <div class="r-tooltip" @mouseenter="visible = true" @mouseleave="visible = false">
    <slot />
    <Transition name="r-tooltip">
      <div v-if="visible" class="r-tooltip__content" :class="`r-tooltip--${position}`">
        {{ content }}
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.r-tooltip { position: relative; display: inline-block; }
.r-tooltip__content {
  position: absolute;
  padding: 8px 10px;
  width: 260px;
  max-width: min(260px, calc(100vw - 32px));
  background: var(--color-primary, #4B3621);
  color: var(--color-text-light, #FBF5EF);
  font-size: 12px;
  line-height: 1.5;
  text-align: left;
  border-radius: 8px;
  white-space: pre-line;
  overflow-wrap: anywhere;
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.18);
  z-index: 100;
}
.r-tooltip--top { bottom: 100%; left: 50%; transform: translateX(-50%); margin-bottom: 6px; }
.r-tooltip--bottom { top: 100%; left: 50%; transform: translateX(-50%); margin-top: 6px; }
.r-tooltip--left { right: 100%; top: 50%; transform: translateY(-50%); margin-right: 6px; }
.r-tooltip--right { left: 100%; top: 50%; transform: translateY(-50%); margin-left: 6px; }
.r-tooltip-enter-active, .r-tooltip-leave-active { transition: all 0.2s; }
.r-tooltip-enter-from, .r-tooltip-leave-to { opacity: 0; }
</style>
