<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  open: boolean
  title?: string
  targetLabel?: string
  targetType?: 'post' | 'item' | 'comment' | 'item_comment' | 'user'
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [{ reason: string; detail: string; hideTarget: boolean; blockAuthor: boolean }]
}>()

const reason = ref('spam')
const detail = ref('')
const hideTarget = ref(false)
const blockAuthor = ref(false)
const canSubmit = computed(() => detail.value.trim().length > 0)
const hideTargetLabel = computed(() => {
  if (props.targetType === 'comment' || props.targetType === 'item_comment') {
    return '提交后同时隐藏这条评论'
  }
  if (props.targetType === 'item') {
    return '提交后同时隐藏这个道具'
  }
  if (props.targetType === 'user') {
    return '提交后同时隐藏该用户相关内容'
  }
  return '提交后同时隐藏这条内容'
})

const reasonOptions = [
  { value: 'spam', label: '垃圾信息或刷屏' },
  { value: 'abuse', label: '辱骂、人身攻击' },
  { value: 'fraud', label: '诈骗或恶意引流' },
  { value: 'sexual', label: '色情或不适内容' },
  { value: 'illegal', label: '违法违规内容' },
  { value: 'other', label: '其他问题' },
]

watch(() => props.open, (open) => {
  if (!open) {
    reason.value = 'spam'
    detail.value = ''
    hideTarget.value = false
    blockAuthor.value = false
  }
})

function close() {
  emit('close')
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    reason: reason.value,
    detail: detail.value.trim(),
    hideTarget: hideTarget.value,
    blockAuthor: blockAuthor.value,
  })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet-fade">
      <div v-if="open" class="sheet-mask" @click.self="close">
        <Transition name="sheet-slide">
          <div class="sheet-panel">
            <div class="sheet-handle"></div>
            <div class="sheet-header">
              <div>
                <h3>{{ title || '举报内容' }}</h3>
                <p v-if="targetLabel">{{ targetLabel }}</p>
              </div>
              <button type="button" class="sheet-close" @click="close">
                <i class="ri-close-line" />
              </button>
            </div>
            <label class="sheet-field">
              <span>举报原因</span>
              <select v-model="reason">
                <option v-for="option in reasonOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="sheet-field">
              <span>补充说明</span>
              <textarea
                v-model="detail"
                rows="4"
                maxlength="500"
                placeholder="请填写备注说明"
              />
            </label>
            <label class="sheet-check">
              <input v-model="hideTarget" type="checkbox">
              <span>{{ hideTargetLabel }}</span>
            </label>
            <label class="sheet-check">
              <input v-model="blockAuthor" type="checkbox">
              <span>提交后同时屏蔽该作者</span>
            </label>
            <p class="sheet-hint" :class="{ error: !canSubmit }">举报需选择原因并填写备注说明。</p>
            <div class="sheet-actions">
              <button type="button" class="sheet-btn ghost" @click="close">取消</button>
              <button type="button" class="sheet-btn primary" :disabled="submitting || !canSubmit" @click="submit">
                {{ submitting ? '提交中...' : '提交举报' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.sheet-mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.52);
  z-index: 2400;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-panel {
  width: 100%;
  max-width: 640px;
  background: var(--color-card-bg);
  border-radius: 22px 22px 0 0;
  padding: 10px 16px calc(20px + var(--safe-bottom, 0px));
  box-shadow: 0 -18px 40px rgba(0, 0, 0, 0.18);
}

.sheet-handle {
  width: 54px;
  height: 5px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.45);
  margin: 0 auto 14px;
}

.sheet-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 14px;
}

.sheet-header h3 {
  margin: 0;
  font-size: 17px;
}

.sheet-header p {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.sheet-close {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  border: 1px solid var(--color-border);
  background: #fff;
  color: var(--color-text-secondary);
}

.sheet-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 14px;
}

.sheet-field span {
  font-size: 13px;
  font-weight: 600;
}

.sheet-field select,
.sheet-field textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 12px;
  background: var(--input-bg);
  font: inherit;
}

.sheet-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 6px;
}

.sheet-check {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-main);
  margin-bottom: 6px;
}

.sheet-check input {
  width: 16px;
  height: 16px;
  accent-color: var(--color-secondary);
}

.sheet-hint {
  margin: -4px 0 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.sheet-hint.error {
  color: #c2410c;
}

.sheet-btn {
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: none;
  font-size: 14px;
  font-weight: 600;
}

.sheet-btn.ghost {
  border: 1px solid var(--color-border);
  background: #fff;
}

.sheet-btn.primary {
  background: var(--color-secondary);
  color: var(--btn-primary-text);
}

.sheet-btn:disabled {
  opacity: 0.6;
}

.sheet-fade-enter-active,
.sheet-fade-leave-active {
  transition: opacity 0.2s ease;
}

.sheet-fade-enter-from,
.sheet-fade-leave-to {
  opacity: 0;
}

.sheet-slide-enter-active,
.sheet-slide-leave-active {
  transition: transform 0.2s ease;
}

.sheet-slide-enter-from,
.sheet-slide-leave-to {
  transform: translateY(100%);
}
</style>
