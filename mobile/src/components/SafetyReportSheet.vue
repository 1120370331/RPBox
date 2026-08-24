<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

type SafetyTargetType =
  | 'post'
  | 'item'
  | 'user'
  | 'comment'
  | 'item_comment'
  | 'rpdb_comment'
  | 'story'
  | 'rpdb_work'
  | 'character_card'
  | 'guild'

const props = defineProps<{
  open: boolean
  title?: string
  targetLabel?: string
  targetType?: SafetyTargetType
  initialAction?: 'default' | 'report' | 'block'
  submitting?: boolean
}>()

const { t } = useI18n()
const titleId = `safety-report-sheet-title-${getCurrentInstance()?.uid ?? 0}`

const emit = defineEmits<{
  close: []
  submit: [{ reason: string; detail: string; hideTarget: boolean; blockAuthor: boolean; submitReport: boolean }]
}>()

const reason = ref('spam')
const detail = ref('')
const hideTarget = ref(true)
const blockAuthor = ref(false)
const submitReport = ref(false)
const submitGuard = ref(false)
const canSubmit = computed(() => {
  if (submitReport.value) return detail.value.trim().length > 0
  return hideTarget.value || blockAuthor.value
})
const targetTypeKey = computed(() => props.targetType ?? 'content')
const dialogTitle = computed(() => props.title || t(`common.safetyReport.title.${targetTypeKey.value}`))
const hideTargetLabel = computed(() => t(`common.safetyReport.hide.${targetTypeKey.value}`))
const blockAuthorLabel = computed(() => {
  const key = props.targetType === 'user' ? 'user' : 'author'
  return t(`common.safetyReport.block.${key}`)
})
const hintText = computed(() => {
  if (submitReport.value) return t('common.safetyReport.hint.report')
  if (!canSubmit.value) return t('common.safetyReport.hint.selectLocalAction')
  return t('common.safetyReport.hint.localOnly')
})

const reasonOptions = computed(() => [
  { value: 'spam', label: t('common.safetyReport.reason.spam') },
  { value: 'abuse', label: t('common.safetyReport.reason.abuse') },
  { value: 'fraud', label: t('common.safetyReport.reason.fraud') },
  { value: 'sexual', label: t('common.safetyReport.reason.sexual') },
  { value: 'illegal', label: t('common.safetyReport.reason.illegal') },
  { value: 'other', label: t('common.safetyReport.reason.other') },
])

function resetForm() {
  submitGuard.value = false
  reason.value = 'spam'
  detail.value = ''
  if (props.initialAction === 'report') {
    submitReport.value = true
    hideTarget.value = false
    blockAuthor.value = false
    return
  }
  if (props.initialAction === 'block') {
    submitReport.value = false
    hideTarget.value = false
    blockAuthor.value = true
    return
  }
  submitReport.value = false
  hideTarget.value = props.targetType !== 'user'
  blockAuthor.value = props.targetType === 'user'
}

watch(() => props.open, (open) => {
  if (open) resetForm()
}, { immediate: true })

watch(() => props.submitting, (submitting) => {
  if (!submitting) submitGuard.value = false
})

function close() {
  emit('close')
}

function submit() {
  if (props.submitting || submitGuard.value || !canSubmit.value) return
  submitGuard.value = true
  emit('submit', {
    reason: reason.value,
    detail: detail.value.trim(),
    hideTarget: hideTarget.value,
    blockAuthor: blockAuthor.value,
    submitReport: submitReport.value,
  })
  void nextTick(() => {
    if (!props.submitting) submitGuard.value = false
  })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet-fade">
      <div v-if="open" class="sheet-mask" @click.self="close">
        <Transition name="sheet-slide">
          <div
            class="sheet-panel"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="titleId"
          >
            <div class="sheet-handle"></div>
            <div class="sheet-header">
              <div>
                <h3 :id="titleId">{{ dialogTitle }}</h3>
                <p v-if="targetLabel">{{ targetLabel }}</p>
              </div>
              <button
                type="button"
                class="sheet-close"
                :aria-label="t('common.safetyReport.close')"
                @click="close"
              >
                <i class="ri-close-line" aria-hidden="true" />
              </button>
            </div>
            <div class="sheet-local-actions">
              <label class="sheet-check">
                <input v-model="hideTarget" type="checkbox">
                <span>{{ hideTargetLabel }}</span>
              </label>
              <label class="sheet-check">
                <input v-model="blockAuthor" type="checkbox">
                <span>{{ blockAuthorLabel }}</span>
              </label>
            </div>
            <label class="sheet-check report-check">
              <input v-model="submitReport" type="checkbox">
              <span>{{ t('common.safetyReport.submitToModerator') }}</span>
            </label>
            <label v-if="submitReport" class="sheet-field">
              <span>{{ t('common.safetyReport.reasonLabel') }}</span>
              <select v-model="reason">
                <option v-for="option in reasonOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label v-if="submitReport" class="sheet-field">
              <span>{{ t('common.safetyReport.detailLabel') }}</span>
              <textarea
                v-model="detail"
                rows="4"
                maxlength="500"
                :placeholder="t('common.safetyReport.detailPlaceholder')"
              />
            </label>
            <p class="sheet-hint" :class="{ error: !canSubmit }">{{ hintText }}</p>
            <div class="sheet-actions">
              <button type="button" class="sheet-btn ghost" @click="close">
                {{ t('common.safetyReport.cancel') }}
              </button>
              <button
                type="button"
                class="sheet-btn primary"
                :disabled="submitting || submitGuard || !canSubmit"
                @click="submit"
              >
                {{ submitting
                  ? t('common.safetyReport.submitting')
                  : (submitReport ? t('common.safetyReport.submitReport') : t('common.safetyReport.confirm')) }}
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

.sheet-local-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-panel-bg);
}

.report-check {
  margin-bottom: 12px;
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
