<script setup lang="ts">
import { inject, computed } from 'vue'

interface Props {
  name: string
  label: string
}

const props = defineProps<Props>()
const tabs = inject<any>('tabs')

const isActive = computed(() => tabs?.activeTab.value === props.name)
</script>

<template>
  <button
    class="r-tab-pane"
    :class="{ 'r-tab-pane--active': isActive }"
    @click="tabs?.setActive(name)"
  >
    {{ label }}
  </button>
</template>

<style scoped>
.r-tab-pane {
  padding: 12px 20px;
  border: none;
  background: transparent;
  font-size: 14px;
  color: var(--color-secondary);
  cursor: pointer;
  position: relative;
  transition: all 0.2s;
  font-family: inherit;
}

.r-tab-pane:hover { color: var(--color-primary); }

.r-tab-pane--active {
  color: var(--color-accent);
  font-weight: 600;
}

.r-tab-pane--active::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--color-accent);
}
</style>
