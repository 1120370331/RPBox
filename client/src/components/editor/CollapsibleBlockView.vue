<script setup lang="ts">
import { computed } from 'vue'
import { NodeViewContent, NodeViewWrapper, nodeViewProps } from '@tiptap/vue-3'
import { DEFAULT_COLLAPSE_SUMMARY } from './collapseBlockConstants'

const props = defineProps(nodeViewProps)

// 编辑时允许标题临时为空，失焦后再回填默认标题，
// 否则清空输入框会被立刻写回默认值，打断输入。
const summary = computed({
  get: () => String(props.node.attrs.summary ?? ''),
  set: (value: string) => props.updateAttributes({ summary: value }),
})

function normalizeSummary() {
  if (!String(props.node.attrs.summary || '').trim()) {
    props.updateAttributes({ summary: DEFAULT_COLLAPSE_SUMMARY })
  }
}

function toggleOpen() {
  props.updateAttributes({ open: !props.node.attrs.open })
}
</script>

<template>
  <NodeViewWrapper class="collapse-editor" data-type="collapsible-block">
    <div class="collapse-editor__head" contenteditable="false">
      <i class="ri-arrow-down-s-line collapse-editor__chevron" aria-hidden="true"></i>
      <input
        v-model="summary"
        class="collapse-editor__title"
        type="text"
        maxlength="40"
        placeholder="折叠区块标题"
        aria-label="折叠区块标题"
        @keydown.stop
        @mousedown.stop
        @click.stop
        @blur="normalizeSummary"
      />
      <button
        type="button"
        class="collapse-editor__toggle"
        :class="{ 'is-open': node.attrs.open }"
        :title="node.attrs.open ? '读者看到时默认展开，点击改为默认收起' : '读者看到时默认收起，点击改为默认展开'"
        @click.stop.prevent="toggleOpen"
      >
        <i :class="node.attrs.open ? 'ri-eye-line' : 'ri-eye-off-line'" aria-hidden="true"></i>
        {{ node.attrs.open ? '默认展开' : '默认收起' }}
      </button>
    </div>
    <NodeViewContent class="collapse-editor__body" />
  </NodeViewWrapper>
</template>

<style scoped>
.collapse-editor {
  margin: 14px 0;
  border: 1px dashed color-mix(in srgb, var(--color-secondary) 45%, var(--color-border));
  border-radius: 10px;
  background: color-mix(in srgb, var(--color-secondary) 5%, transparent);
  overflow: hidden;
}

.collapse-editor__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px dashed var(--color-border);
  background: color-mix(in srgb, var(--color-secondary) 8%, transparent);
}

.collapse-editor__chevron {
  flex: 0 0 auto;
  color: var(--color-secondary);
  font-size: 16px;
}

.collapse-editor__title {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--color-text-main);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.4;
  outline: none;
}

.collapse-editor__toggle {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-panel-bg);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s, background 0.2s;
}

.collapse-editor__toggle:hover,
.collapse-editor__toggle.is-open {
  border-color: var(--color-secondary);
  background: color-mix(in srgb, var(--color-secondary) 12%, transparent);
  color: var(--color-secondary);
}

.collapse-editor__body {
  padding: 6px 14px 10px;
}
</style>
