<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    modelValue: number
    totalPages: number
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
  change: [value: number]
}>()

const { t } = useI18n()
const expanded = ref(false)

const current = computed(() => Math.max(1, Math.min(props.modelValue, props.totalPages || 1)))
const total = computed(() => Math.max(1, props.totalPages || 1))
const canPrev = computed(() => !props.disabled && current.value > 1)
const canNext = computed(() => !props.disabled && current.value < total.value)

type PageToken = number | 'dots-left' | 'dots-right'

const pageTokens = computed<PageToken[]>(() => {
  if (total.value <= 9) {
    return Array.from({ length: total.value }, (_, i) => i + 1)
  }

  const tokens: PageToken[] = [1]
  const start = Math.max(2, current.value - 2)
  const end = Math.min(total.value - 1, current.value + 2)

  if (start > 2) tokens.push('dots-left')
  for (let page = start; page <= end; page += 1) tokens.push(page)
  if (end < total.value - 1) tokens.push('dots-right')
  tokens.push(total.value)
  return tokens
})

function goTo(page: number) {
  if (props.disabled) return
  const safePage = Math.max(1, Math.min(page, total.value))
  if (safePage === current.value) return
  emit('update:modelValue', safePage)
  emit('change', safePage)
  expanded.value = false
}

function toggleExpanded() {
  if (total.value <= 1 || props.disabled) return
  expanded.value = !expanded.value
}
</script>

<template>
  <nav v-if="total > 1" class="mobile-pagination" aria-label="Pagination">
    <div class="pager-shell">
      <button class="icon-btn" :disabled="!canPrev" :aria-label="t('common.pagination.prev')" @click="goTo(current - 1)">
        <i class="ri-arrow-left-s-line" />
      </button>

      <button
        class="page-pill"
        :class="{ open: expanded }"
        :disabled="disabled"
        @click="toggleExpanded"
      >
        <span class="page-current">{{ current }}</span>
        <span class="page-divider">/</span>
        <span class="page-total">{{ total }}</span>
        <i class="ri-arrow-down-s-line arrow" />
      </button>

      <button class="icon-btn" :disabled="!canNext" :aria-label="t('common.pagination.next')" @click="goTo(current + 1)">
        <i class="ri-arrow-right-s-line" />
      </button>
    </div>

    <Transition name="jump-fade">
      <div v-if="expanded" class="jump-panel">
        <button
          v-for="token in pageTokens"
          :key="String(token)"
          class="jump-chip"
          :class="{ active: token === current }"
          :disabled="typeof token !== 'number'"
          @click="typeof token === 'number' ? goTo(token) : undefined"
        >
          <template v-if="token === 'dots-left' || token === 'dots-right'">…</template>
          <template v-else>{{ token }}</template>
        </button>
      </div>
    </Transition>
  </nav>
</template>

<style scoped>
.mobile-pagination {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pager-shell {
  min-height: 52px;
  border-radius: 999px;
  border: 1px solid rgba(75, 54, 33, 0.14);
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 10px 22px rgba(44, 24, 16, 0.12);
  backdrop-filter: blur(10px);
  display: grid;
  grid-template-columns: 44px 1fr 44px;
  align-items: center;
  gap: 6px;
  padding: 4px;
}

.icon-btn {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-dark);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.icon-btn:active:not(:disabled) {
  transform: scale(0.95);
}

.icon-btn:disabled {
  opacity: 0.35;
}

.page-pill {
  height: 40px;
  border: none;
  border-radius: 999px;
  background: rgba(128, 64, 48, 0.08);
  color: var(--text-dark);
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.01em;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.page-pill .arrow {
  font-size: 18px;
  transition: transform 0.2s ease;
}

.page-pill.open .arrow {
  transform: rotate(180deg);
}

.page-current {
  min-width: 20px;
  text-align: right;
}

.page-divider,
.page-total {
  opacity: 0.75;
}

.jump-panel {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 4px 2px 0;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.jump-panel::-webkit-scrollbar {
  display: none;
}

.jump-chip {
  min-width: 34px;
  height: 34px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-panel-bg);
  color: var(--text-dark);
  font-size: 12px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.jump-chip.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.jump-chip:disabled {
  opacity: 0.6;
}

.jump-fade-enter-active,
.jump-fade-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
  transform-origin: top center;
}

.jump-fade-enter-from,
.jump-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
