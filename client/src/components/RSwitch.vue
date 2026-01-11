<script setup lang="ts">
interface Props {
  modelValue?: boolean
  disabled?: boolean
  size?: 'sm' | 'md'
}

withDefaults(defineProps<Props>(), {
  size: 'md',
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <button
    class="r-switch"
    :class="[`r-switch--${size}`, { 'r-switch--on': modelValue, 'r-switch--disabled': disabled }]"
    :disabled="disabled"
    @click="emit('update:modelValue', !modelValue)"
  >
    <span class="r-switch__dot"></span>
  </button>
</template>

<style scoped>
.r-switch {
  position: relative;
  border: none;
  border-radius: 20px;
  background: rgba(75,54,33,0.2);
  cursor: pointer;
  transition: all 0.2s;
  padding: 2px;
}

.r-switch--sm { width: 36px; height: 20px; }
.r-switch--md { width: 44px; height: 24px; }

.r-switch__dot {
  display: block;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.2s;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
}

.r-switch--sm .r-switch__dot { width: 16px; height: 16px; }
.r-switch--md .r-switch__dot { width: 20px; height: 20px; }

.r-switch--on { background: var(--color-accent); }
.r-switch--on.r-switch--sm .r-switch__dot { transform: translateX(16px); }
.r-switch--on.r-switch--md .r-switch__dot { transform: translateX(20px); }

.r-switch--disabled { opacity: 0.5; cursor: not-allowed; }
</style>
