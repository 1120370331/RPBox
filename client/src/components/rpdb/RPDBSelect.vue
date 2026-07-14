<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

interface RPDBSelectOption {
  value: string
  label: string
  hint?: string
}

const props = defineProps<{
  modelValue?: string
  options: RPDBSelectOption[]
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const open = ref(false)
const menuStyle = ref<Record<string, string>>({})

const selectedOption = computed(() => props.options?.find(option => option.value === props.modelValue))

function choose(value: string) {
  emit('update:modelValue', value)
  open.value = false
}

async function toggle() {
  open.value = !open.value
  if (!open.value) return
  await nextTick()
  updateMenuPosition()
}

function updateMenuPosition() {
  const trigger = triggerRef.value
  if (!trigger) return
  const rect = trigger.getBoundingClientRect()
  const edge = 8
  const gap = 6
  const desiredHeight = Math.min(menuRef.value?.scrollHeight || 238, 238)
  const spaceBelow = window.innerHeight - rect.bottom - edge - gap
  const spaceAbove = rect.top - edge - gap
  const openAbove = spaceBelow < Math.min(desiredHeight, 140) && spaceAbove > spaceBelow
  const availableHeight = Math.max(96, Math.min(desiredHeight, openAbove ? spaceAbove : spaceBelow))
  const width = Math.min(Math.max(rect.width, 180), window.innerWidth - edge * 2)
  const left = Math.max(edge, Math.min(rect.left, window.innerWidth - width - edge))
  const top = openAbove
    ? Math.max(edge, rect.top - availableHeight - gap)
    : Math.min(window.innerHeight - availableHeight - edge, rect.bottom + gap)

  menuStyle.value = {
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${availableHeight}px`,
  }
}

function handleOutsidePointer(event: MouseEvent) {
  const target = event.target as Node | null
  if (!target || rootRef.value?.contains(target) || menuRef.value?.contains(target)) return
  open.value = false
}

function close() {
  open.value = false
}

function addFloatingListeners() {
  document.addEventListener('mousedown', handleOutsidePointer, true)
  document.addEventListener('scroll', updateMenuPosition, true)
  window.addEventListener('resize', updateMenuPosition)
}

function removeFloatingListeners() {
  document.removeEventListener('mousedown', handleOutsidePointer, true)
  document.removeEventListener('scroll', updateMenuPosition, true)
  window.removeEventListener('resize', updateMenuPosition)
}

watch(open, async (isOpen) => {
  removeFloatingListeners()
  if (!isOpen) return
  addFloatingListeners()
  await nextTick()
  updateMenuPosition()
})

onBeforeUnmount(removeFloatingListeners)
</script>

<template>
  <div ref="rootRef" class="rpdb-select" data-testid="rpdb-custom-select" @keydown.esc="close">
    <button
      ref="triggerRef"
      type="button"
      class="rpdb-select__trigger"
      :class="{ active: open }"
      aria-haspopup="listbox"
      :aria-expanded="open"
      @click="toggle"
    >
      <span>
        <b>{{ selectedOption?.label || placeholder || '请选择' }}</b>
        <small v-if="selectedOption?.hint">{{ selectedOption.hint }}</small>
      </span>
      <i class="ri-arrow-down-s-line"></i>
    </button>
    <Teleport to="body">
      <div
        v-if="open"
        ref="menuRef"
        class="rpdb-select__menu"
        role="listbox"
        :style="menuStyle"
        @keydown.esc="close"
      >
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          role="option"
          class="rpdb-select__option"
          :class="{ selected: option.value === modelValue }"
          :aria-selected="option.value === modelValue"
          @click="choose(option.value)"
        >
          <span>
            <b>{{ option.label }}</b>
            <small v-if="option.hint">{{ option.hint }}</small>
          </span>
          <i v-if="option.value === modelValue" class="ri-check-line"></i>
        </button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.rpdb-select {
  position: relative;
  min-width: 0;
}

.rpdb-select__trigger,
.rpdb-select__option {
  width: 100%;
  border: 1px solid var(--rpdb-line, var(--color-border));
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  font: inherit;
  text-align: left;
}

.rpdb-select__trigger {
  display: flex;
  min-height: 34px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 9px 7px 10px;
  border-radius: 7px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-panel-bg) 97%, #fff 3%), color-mix(in srgb, var(--color-card-bg) 82%, #fff 18%));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .35);
  cursor: pointer;
}

.rpdb-select__trigger:hover,
.rpdb-select__trigger.active {
  border-color: var(--rpdb-focus, var(--color-accent));
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-panel-bg) 98%, #fff 2%), color-mix(in srgb, var(--color-accent) 6%, var(--color-card-bg)));
}

.rpdb-select__trigger.active {
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-accent) 14%, transparent), inset 0 1px 0 rgba(255, 255, 255, .35);
}

.rpdb-select__trigger > span,
.rpdb-select__option > span {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.rpdb-select__trigger b,
.rpdb-select__option b {
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rpdb-select__trigger small,
.rpdb-select__option small {
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: 10px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rpdb-select__trigger i {
  flex: 0 0 auto;
  color: var(--color-accent);
  font-size: 17px;
  transition: transform .16s ease;
}

.rpdb-select__trigger.active i {
  transform: rotate(180deg);
}

.rpdb-select__menu {
  position: fixed;
  z-index: 2600;
  display: grid;
  overflow: auto;
  padding: 5px;
  border: 1px solid var(--rpdb-line, var(--color-border));
  border-radius: 9px;
  background: color-mix(in srgb, var(--color-panel-bg) 96%, transparent);
  box-shadow: 0 16px 34px rgba(0, 0, 0, .18);
  backdrop-filter: blur(14px);
}

.rpdb-select__option {
  display: flex;
  min-height: 32px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 8px;
  border-color: transparent;
  border-radius: 7px;
  background: transparent;
  cursor: pointer;
}

.rpdb-select__option:hover {
  background: color-mix(in srgb, var(--color-accent) 8%, transparent);
}

.rpdb-select__option.selected {
  background: color-mix(in srgb, var(--color-accent) 13%, transparent);
  color: var(--color-accent);
}

.rpdb-select__option i {
  flex: 0 0 auto;
  color: var(--color-accent);
  font-size: 16px;
}
</style>
