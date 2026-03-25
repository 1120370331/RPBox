<script setup lang="ts">
import { ref, provide, watch, reactive, onMounted } from 'vue'

interface Props {
  modelValue?: string
}

interface TabInfo {
  name: string
  label: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const activeTab = ref(props.modelValue || '')
const tabs = reactive<TabInfo[]>([])

watch(() => props.modelValue, (val) => {
  if (val) activeTab.value = val
})

function setActive(name: string) {
  activeTab.value = name
  emit('update:modelValue', name)
}

function registerTab(info: TabInfo) {
  if (!tabs.find(t => t.name === info.name)) {
    tabs.push(info)
    // 如果没有激活的 tab，激活第一个
    if (!activeTab.value) {
      activeTab.value = info.name
      emit('update:modelValue', info.name)
    }
  }
}

provide('tabs', { activeTab, setActive, registerTab })
</script>

<template>
  <div class="r-tabs">
    <div class="r-tabs__header">
      <button
        v-for="tab in tabs"
        :key="tab.name"
        class="r-tabs__tab"
        :class="{ 'r-tabs__tab--active': activeTab === tab.name }"
        @click="setActive(tab.name)"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="r-tabs__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.r-tabs {
  background: var(--color-panel-bg);
  border-radius: 12px;
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

.r-tabs__header {
  display: flex;
  border-bottom: 2px solid var(--color-border-light);
}

.r-tabs__tab {
  padding: 14px 24px;
  border: none;
  background: transparent;
  font-size: 15px;
  color: var(--color-secondary);
  cursor: pointer;
  position: relative;
  transition: all 0.2s;
  font-family: inherit;
  font-weight: 500;
}

.r-tabs__tab:hover {
  color: var(--color-primary);
}

.r-tabs__tab--active {
  color: var(--color-accent);
  font-weight: 600;
}

.r-tabs__tab--active::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--color-accent);
  border-radius: 3px 3px 0 0;
}

.r-tabs__content {
  padding: 20px;
}
</style>
