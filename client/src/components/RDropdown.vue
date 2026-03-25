<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  trigger?: 'click' | 'hover'
}

withDefaults(defineProps<Props>(), {
  trigger: 'click',
})

const visible = ref(false)

function toggle() { visible.value = !visible.value }
function show() { visible.value = true }
function hide() { visible.value = false }
</script>

<template>
  <div
    class="r-dropdown"
    @click="trigger === 'click' && toggle()"
    @mouseenter="trigger === 'hover' && show()"
    @mouseleave="trigger === 'hover' && hide()"
  >
    <div class="r-dropdown__trigger"><slot /></div>
    <Transition name="r-dropdown">
      <div v-if="visible" class="r-dropdown__menu" @click="hide">
        <slot name="menu" />
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.r-dropdown { position: relative; display: inline-block; }

.r-dropdown__menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  min-width: 120px;
  background: var(--color-panel-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 16px rgba(var(--shadow-base), 0.2);
  z-index: 100;
  padding: 4px 0;
}

.r-dropdown-enter-active, .r-dropdown-leave-active { transition: all 0.2s; }
.r-dropdown-enter-from, .r-dropdown-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
