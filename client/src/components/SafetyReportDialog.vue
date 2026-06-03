<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  visible: boolean
  title?: string
  targetLabel?: string
  targetType?: 'post' | 'item' | 'comment' | 'item_comment' | 'user' | 'story'
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [{ reason: string; detail: string; hideTarget: boolean; blockAuthor: boolean; submitReport: boolean }]
}>()

const reason = ref('spam')
const detail = ref('')
const hideTarget = ref(true)
const blockAuthor = ref(false)
const submitReport = ref(false)
const canSubmit = computed(() => {
  if (submitReport.value) {
    return detail.value.trim().length > 0
  }
  return hideTarget.value || blockAuthor.value
})
const hideTargetLabel = computed(() => {
  if (props.targetType === 'comment' || props.targetType === 'item_comment') {
    return '隐藏这条评论'
  }
  if (props.targetType === 'item') {
    return '隐藏这个道具'
  }
  if (props.targetType === 'user') {
    return '隐藏该用户相关内容'
  }
  if (props.targetType === 'story') {
    return '隐藏这个剧情'
  }
  return '隐藏这条内容'
})
const blockAuthorLabel = computed(() => {
  if (props.targetType === 'user') return '屏蔽该用户'
  return '同时屏蔽该作者'
})
const hintText = computed(() => {
  if (submitReport.value) return '同时举报需要选择原因并填写备注说明。'
  if (!canSubmit.value) return '请选择至少一个本地处理动作。'
  return '默认只对你隐藏或屏蔽，不进入版主审核队列。'
})

const reasonOptions = [
  { value: 'spam', label: '垃圾信息或刷屏' },
  { value: 'abuse', label: '辱骂、人身攻击' },
  { value: 'fraud', label: '诈骗或恶意引流' },
  { value: 'sexual', label: '色情或不适内容' },
  { value: 'illegal', label: '违法违规内容' },
  { value: 'other', label: '其他问题' },
]

function resetForm() {
  reason.value = 'spam'
  detail.value = ''
  submitReport.value = false
  hideTarget.value = props.targetType !== 'user'
  blockAuthor.value = props.targetType === 'user'
}

watch(() => props.visible, (visible) => {
  if (visible) {
    resetForm()
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
    submitReport: submitReport.value,
  })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="r-dialog">
      <div v-if="visible" class="report-dialog-mask" @click.self="close">
        <div class="report-dialog">
          <div class="report-dialog__header">
            <div>
              <h3>{{ title || '举报内容' }}</h3>
              <p v-if="targetLabel">{{ targetLabel }}</p>
            </div>
            <button class="report-dialog__close" type="button" @click="close">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="report-dialog__body">
            <div class="report-dialog__local-actions">
              <label class="report-dialog__check">
                <input v-model="hideTarget" type="checkbox">
                <span>{{ hideTargetLabel }}</span>
              </label>
              <label class="report-dialog__check">
                <input v-model="blockAuthor" type="checkbox">
                <span>{{ blockAuthorLabel }}</span>
              </label>
            </div>
            <label class="report-dialog__check report-dialog__report-check">
              <input v-model="submitReport" type="checkbox">
              <span>同时提交给版主审核</span>
            </label>
            <label v-if="submitReport" class="report-dialog__label">
              <span>举报原因</span>
              <select v-model="reason">
                <option v-for="option in reasonOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label v-if="submitReport" class="report-dialog__label">
              <span>补充说明</span>
              <textarea
                v-model="detail"
                rows="4"
                maxlength="500"
                placeholder="请填写备注说明，帮助版主判断处理"
              ></textarea>
            </label>
            <p class="report-dialog__hint" :class="{ error: !canSubmit }">{{ hintText }}</p>
          </div>
          <div class="report-dialog__footer">
            <button class="report-dialog__btn report-dialog__btn--ghost" type="button" @click="close">
              取消
            </button>
            <button class="report-dialog__btn report-dialog__btn--primary" type="button" :disabled="submitting || !canSubmit" @click="submit">
              {{ submitting ? '提交中...' : (submitReport ? '提交举报' : '确认处理') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.report-dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(18, 24, 38, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2200;
  padding: 20px;
}

.report-dialog {
  width: min(520px, 100%);
  background: var(--color-panel-bg);
  border-radius: 16px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.18);
  overflow: hidden;
}

.report-dialog__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 24px 12px;
}

.report-dialog__header h3 {
  margin: 0;
  font-size: 20px;
  color: var(--color-text-main);
}

.report-dialog__header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--color-text-muted);
}

.report-dialog__close {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: var(--color-card-bg);
  color: var(--color-text-muted);
  cursor: pointer;
}

.report-dialog__body {
  padding: 8px 24px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-dialog__local-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-card-bg);
}

.report-dialog__label {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.report-dialog__label span {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-main);
}

.report-dialog__label select,
.report-dialog__label textarea {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-card-bg);
  color: var(--color-text-main);
  padding: 12px 14px;
  font: inherit;
}

.report-dialog__label textarea {
  resize: vertical;
}

.report-dialog__check {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-main);
}

.report-dialog__check input {
  width: 16px;
  height: 16px;
  accent-color: var(--color-secondary);
}

.report-dialog__report-check {
  padding-top: 2px;
}

.report-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 24px 24px;
}

.report-dialog__hint {
  margin: -4px 0 0;
  font-size: 12px;
  color: var(--color-text-muted);
}

.report-dialog__hint.error {
  color: #c2410c;
}

.report-dialog__btn {
  min-width: 104px;
  height: 42px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: none;
}

.report-dialog__btn--ghost {
  background: var(--color-card-bg);
  color: var(--color-text-main);
  border: 1px solid var(--color-border);
}

.report-dialog__btn--primary {
  background: var(--color-secondary);
  color: var(--btn-primary-text, #fff);
}

.report-dialog__btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
