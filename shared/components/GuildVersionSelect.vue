<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import type { GuildGameFlavor } from '../utils/guildGameFlavors'

const props = withDefaults(defineProps<{
  modelValue: GuildGameFlavor[]
  label: string
  allLabel: string
  retailLabel: string
  foreverLabel: string
  allowAll?: boolean
}>(), { allowAll: false })

const emit = defineEmits<{
  'update:modelValue': [flavors: GuildGameFlavor[]]
  change: [flavors: GuildGameFlavor[]]
}>()

const menuId = useId()
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const open = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selected = ref<GuildGameFlavor[]>([...props.modelValue])
watch(() => props.modelValue, value => { selected.value = [...(value || [])] }, { deep: true })
const summary = computed(() => {
  if (!selected.value.length) return props.allLabel
  return selected.value.map(flavor => flavor === '_forever_' ? props.foreverLabel : props.retailLabel).join('、')
})

function updatePosition() {
  const button = trigger.value
  if (!button) return
  const rect = button.getBoundingClientRect()
  const edge = 8
  const gap = 5
  const height = menu.value?.offsetHeight || 140
  const width = Math.min(Math.max(rect.width, 190), window.innerWidth - edge * 2)
  const left = Math.max(edge, Math.min(rect.left, window.innerWidth - width - edge))
  const above = window.innerHeight - rect.bottom < height + edge && rect.top > height + edge
  menuStyle.value = {
    top: `${above ? rect.top - height - gap : rect.bottom + gap}px`,
    left: `${left}px`,
    width: `${width}px`,
  }
}

async function toggle() {
  open.value = !open.value
  if (open.value) {
    await nextTick()
    updatePosition()
  }
}

function close() {
  open.value = false
  trigger.value?.focus()
}

function onOutsidePointer(event: PointerEvent) {
  const target = event.target as Node | null
  if (!target || root.value?.contains(target) || menu.value?.contains(target)) return
  open.value = false
}

function updateSelection(flavors: GuildGameFlavor[]) {
  selected.value = flavors
  emit('update:modelValue', flavors)
  emit('change', flavors)
}

function toggleFlavor(flavor: GuildGameFlavor) {
  const next = selected.value.includes(flavor)
    ? selected.value.filter(value => value !== flavor)
    : [...selected.value, flavor]
  if (!props.allowAll && next.length === 0) return
  updateSelection(next)
}

function selectAll() {
  updateSelection([])
  close()
}

watch(open, value => {
  if (value) {
    document.addEventListener('pointerdown', onOutsidePointer, true)
    document.addEventListener('scroll', updatePosition, true)
    window.addEventListener('resize', updatePosition)
  } else {
    document.removeEventListener('pointerdown', onOutsidePointer, true)
    document.removeEventListener('scroll', updatePosition, true)
    window.removeEventListener('resize', updatePosition)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onOutsidePointer, true)
  document.removeEventListener('scroll', updatePosition, true)
  window.removeEventListener('resize', updatePosition)
})
</script>

<template>
  <div ref="root" class="guild-version-select" @keydown.esc="close">
    <button
      ref="trigger"
      type="button"
      class="guild-version-trigger"
      :aria-label="`${label}：${summary}`"
      :aria-expanded="open"
      :aria-controls="menuId"
      aria-haspopup="true"
      @click="toggle"
    >
      <span>{{ summary }}</span><i class="ri-arrow-down-s-line" aria-hidden="true" />
    </button>
    <Teleport to="body">
      <div v-if="open" :id="menuId" ref="menu" class="guild-version-menu" :style="menuStyle" role="group" :aria-label="label" @keydown.esc="close">
        <button v-if="allowAll" type="button" class="guild-version-all" :class="{ active: !selected.length }" @click="selectAll">{{ allLabel }}</button>
        <label class="guild-version-option"><input type="checkbox" value="_retail_" :checked="selected.includes('_retail_')" :disabled="!allowAll && selected.length === 1 && selected.includes('_retail_')" @change="toggleFlavor('_retail_')" />{{ retailLabel }}</label>
        <label class="guild-version-option"><input type="checkbox" value="_forever_" :checked="selected.includes('_forever_')" :disabled="!allowAll && selected.length === 1 && selected.includes('_forever_')" @change="toggleFlavor('_forever_')" />{{ foreverLabel }}</label>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.guild-version-select { position: relative; min-width: 0; width: 100%; }
.guild-version-trigger { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; min-height: 40px; padding: 8px 11px; border: 1px solid var(--input-border, var(--color-border)); border-radius: 8px; background: var(--input-bg, var(--color-panel-bg)); color: var(--color-text-main); text-align: left; cursor: pointer; }
.guild-version-trigger span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.guild-version-trigger i { flex: none; color: var(--color-text-secondary); font-size: 17px; }
.guild-version-trigger[aria-expanded="true"] { border-color: var(--input-focus, var(--color-accent)); }
.guild-version-menu { position: fixed; z-index: 4000; display: grid; gap: 2px; padding: 5px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-panel-bg); color: var(--color-text-main); box-shadow: 0 12px 28px rgba(var(--shadow-base, 44, 24, 16), 0.2); }
.guild-version-all, .guild-version-option { display: flex; align-items: center; gap: 9px; width: 100%; min-height: 38px; padding: 8px 10px; border: 0; border-radius: 7px; background: transparent; color: inherit; font-size: 13px; text-align: left; cursor: pointer; }
.guild-version-all.active, .guild-version-all:hover, .guild-version-option:hover { background: var(--btn-secondary-bg, var(--color-card-bg)); }
.guild-version-option input { width: 16px; height: 16px; accent-color: var(--color-accent); }
.guild-version-option input:disabled { opacity: 0.75; }
</style>
