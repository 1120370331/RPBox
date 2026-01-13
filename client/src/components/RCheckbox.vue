<script setup lang="ts">
interface Props {
  modelValue?: boolean
  disabled?: boolean
  label?: string
  indeterminate?: boolean
}

withDefaults(defineProps<Props>(), {})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <label class="r-checkbox" :class="{ 'r-checkbox--disabled': disabled, 'r-checkbox--indeterminate': indeterminate }">
    <input
      type="checkbox"
      :checked="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', ($event.target as HTMLInputElement).checked)"
    />
    <span class="r-checkbox__box">
      <span class="r-checkbox__check">✓</span>
      <span class="r-checkbox__indeterminate">−</span>
    </span>
    <span v-if="label" class="r-checkbox__label">{{ label }}</span>
  </label>
</template>

<style scoped>
.r-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.r-checkbox input { display: none; }

.r-checkbox__box {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(75,54,33,0.3);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  position: relative;
}

.r-checkbox__check,
.r-checkbox__indeterminate {
  position: absolute;
  color: #fff;
  opacity: 0;
  transform: scale(0);
  transition: all 0.2s;
}

.r-checkbox__check {
  font-size: 12px;
}

.r-checkbox__indeterminate {
  font-size: 14px;
  font-weight: bold;
}

.r-checkbox input:checked + .r-checkbox__box {
  background: var(--color-accent);
  border-color: var(--color-accent);
}

.r-checkbox input:checked + .r-checkbox__box .r-checkbox__check {
  opacity: 1;
  transform: scale(1);
}

/* Indeterminate 状态 */
.r-checkbox--indeterminate .r-checkbox__box {
  background: var(--color-accent);
  border-color: var(--color-accent);
}

.r-checkbox--indeterminate .r-checkbox__box .r-checkbox__check {
  opacity: 0;
  transform: scale(0);
}

.r-checkbox--indeterminate .r-checkbox__box .r-checkbox__indeterminate {
  opacity: 1;
  transform: scale(1);
}

.r-checkbox__label { font-size: 14px; color: var(--color-primary); }
.r-checkbox--disabled { opacity: 0.5; cursor: not-allowed; }
</style>
