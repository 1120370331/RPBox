<script setup lang="ts">
import { ref, provide } from 'vue'

interface Props {
  modelValue?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const activeTab = ref(props.modelValue || '')

function setActive(name: string) {
  activeTab.value = name
  emit('update:modelValue', name)
}

provide('tabs', { activeTab, setActive })
</script>

<template>
  <div class="r-tabs">
    <div class="r-tabs__header"><slot name="tabs" /></div>
    <div class="r-tabs__content"><slot /></div>
  </div>
</template>

<style scoped>
.r-tabs__header {
  display: flex;
  gap: 4px;
  border-bottom: 2px solid rgba(75, 54, 33, 0.1);
  margin-bottom: 16px;
}
</style>
