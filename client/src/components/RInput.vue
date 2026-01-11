<script setup lang="ts">
interface Props {
  modelValue?: string
  type?: 'text' | 'password' | 'email' | 'number' | 'search' | 'textarea'
  placeholder?: string
  disabled?: boolean
  readonly?: boolean
  error?: string
  size?: 'sm' | 'md' | 'lg'
  prefix?: string
  suffix?: string
  clearable?: boolean
  rows?: number
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  type: 'text',
  size: 'md',
  rows: 3,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  focus: [e: FocusEvent]
  blur: [e: FocusEvent]
}>()

function handleInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
}

function handleClear() {
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="r-input" :class="[`r-input--${size}`, { 'r-input--error': error, 'r-input--disabled': disabled }]">
    <span v-if="prefix" class="r-input__prefix">{{ prefix }}</span>
    <textarea
      v-if="type === 'textarea'"
      class="r-input__inner r-input__textarea"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :rows="rows"
      @input="handleInput"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />
    <input
      v-else
      class="r-input__inner"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      @input="handleInput"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />
    <span v-if="clearable && modelValue" class="r-input__clear" @click="handleClear">×</span>
    <span v-if="suffix" class="r-input__suffix">{{ suffix }}</span>
  </div>
  <p v-if="error" class="r-input__error">{{ error }}</p>
</template>

<style scoped>
.r-input {
  display: flex;
  align-items: center;
  background: #fff;
  border: 2px solid rgba(75, 54, 33, 0.2);
  border-radius: var(--radius-sm);
  transition: border-color 0.2s;
}

.r-input:focus-within { border-color: var(--color-accent); }
.r-input--error { border-color: #c41e3a; }
.r-input--disabled { opacity: 0.6; background: #f5f5f5; }

.r-input--sm { padding: 6px 12px; font-size: 12px; }
.r-input--md { padding: 10px 14px; font-size: 14px; }
.r-input--lg { padding: 14px 18px; font-size: 16px; }

.r-input__inner {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: inherit;
  color: var(--text-dark);
  font-family: inherit;
}

.r-input__inner::placeholder { color: rgba(75, 54, 33, 0.4); }
.r-input__textarea { resize: vertical; min-height: 80px; }

.r-input__prefix, .r-input__suffix {
  color: var(--color-secondary);
  font-size: 14px;
}
.r-input__prefix { margin-right: 8px; }
.r-input__suffix { margin-left: 8px; }

.r-input__clear {
  margin-left: 8px;
  cursor: pointer;
  color: var(--color-secondary);
  font-size: 16px;
}
.r-input__clear:hover { color: var(--color-primary); }

.r-input__error {
  color: #c41e3a;
  font-size: 12px;
  margin-top: 4px;
}
</style>
