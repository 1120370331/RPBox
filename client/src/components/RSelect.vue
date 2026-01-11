<script setup lang="ts">
import { ref, computed } from 'vue'

interface Option {
  label: string
  value: string | number
}

interface Props {
  modelValue?: string | number
  options: Option[]
  placeholder?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '请选择',
})

const emit = defineEmits<{ 'update:modelValue': [value: string | number] }>()

const visible = ref(false)
const selected = computed(() => props.options.find(o => o.value === props.modelValue))

function select(opt: Option) {
  emit('update:modelValue', opt.value)
  visible.value = false
}
</script>

<template>
  <div class="r-select" :class="{ 'r-select--disabled': disabled }">
    <div class="r-select__trigger" @click="!disabled && (visible = !visible)">
      <span :class="{ 'r-select__placeholder': !selected }">
        {{ selected?.label || placeholder }}
      </span>
      <span class="r-select__arrow">▼</span>
    </div>
    <Transition name="r-select">
      <div v-if="visible" class="r-select__dropdown">
        <div
          v-for="opt in options"
          :key="opt.value"
          class="r-select__option"
          :class="{ 'r-select__option--active': opt.value === modelValue }"
          @click="select(opt)"
        >
          {{ opt.label }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.r-select { position: relative; }
.r-select__trigger {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: #fff;
  border: 2px solid rgba(75,54,33,0.2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 14px;
}
.r-select__placeholder { color: rgba(75,54,33,0.4); }
.r-select__arrow { font-size: 10px; color: var(--color-secondary); }
.r-select__dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: #fff;
  border: 1px solid rgba(75,54,33,0.1);
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 16px rgba(0,0,0,0.1);
  z-index: 100;
  max-height: 200px;
  overflow-y: auto;
}
.r-select__option {
  padding: 10px 14px;
  cursor: pointer;
  font-size: 14px;
}
.r-select__option:hover { background: rgba(75,54,33,0.05); }
.r-select__option--active { color: var(--color-accent); background: rgba(184,115,51,0.1); }
.r-select--disabled { opacity: 0.5; pointer-events: none; }
.r-select-enter-active, .r-select-leave-active { transition: all 0.2s; }
.r-select-enter-from, .r-select-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
