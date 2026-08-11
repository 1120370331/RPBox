<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { normalizeCharacterCardHexForCSS } from '@/utils/characterCardColor'

let colorFieldSequence = 0
const { t } = useI18n()

const props = withDefaults(defineProps<{
  modelValue: string
  label: string
  fieldId?: string
  hint?: string
  presets?: string[]
}>(), {
  fieldId: '',
  hint: '',
  presets: () => ['#C79C6E', '#69CCF0', '#9482C9', '#FFF468', '#ABD473', '#F58CBA'],
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const generatedId = `character-dye-${++colorFieldSequence}`
const inputId = computed(() => props.fieldId || generatedId)
const errorId = computed(() => `${inputId.value}-error`)
const hintId = computed(() => `${inputId.value}-hint`)
const textValue = ref('')

function normalizeEditableHex(value?: string | null) {
  const normalized = normalizeCharacterCardHexForCSS(value).trim().toUpperCase()
  if (!normalized) return ''
  return normalized.startsWith('#') ? normalized : `#${normalized}`
}

function isValidHex(value: string) {
  return /^#[0-9A-F]{6}(?:[0-9A-F]{2})?$/.test(value)
}

const invalid = computed(() => Boolean(textValue.value) && !isValidHex(textValue.value))
const currentColor = computed(() => isValidHex(textValue.value) ? textValue.value : '#E8DCCF')
const pickerColor = computed(() => currentColor.value.slice(0, 7))
const describedBy = computed(() => invalid.value ? errorId.value : (props.hint ? hintId.value : undefined))

watch(
  () => props.modelValue,
  (value) => {
    const normalized = normalizeEditableHex(value)
    if (normalized !== textValue.value) textValue.value = normalized
  },
  { immediate: true },
)

function commitText(rawValue: string) {
  const normalized = normalizeEditableHex(rawValue)
  textValue.value = normalized
  if (!normalized || isValidHex(normalized)) emit('update:modelValue', normalized)
}

function selectColor(value: string) {
  const normalized = normalizeEditableHex(value)
  textValue.value = normalized
  emit('update:modelValue', normalized)
}
</script>

<template>
  <div class="character-dye" :class="{ 'character-dye--invalid': invalid }">
    <label class="character-dye__label" :for="inputId">{{ label }}</label>
    <div class="character-dye__well">
      <label class="character-dye__swatch" :style="{ '--dye-color': currentColor }">
        <span class="sr-only">{{ t('characterCards.colorField.openPicker', { label }) }}</span>
        <input
          type="color"
          :value="pickerColor"
          :aria-label="t('characterCards.colorField.picker', { label })"
          @input="selectColor(($event.target as HTMLInputElement).value)"
        />
      </label>
      <span class="character-dye__prefix" aria-hidden="true">HEX</span>
      <input
        :id="inputId"
        class="character-dye__hex"
        type="text"
        :value="textValue"
        maxlength="9"
        spellcheck="false"
        autocomplete="off"
        placeholder="#E8DCCF"
        :aria-invalid="invalid || undefined"
        :aria-describedby="describedBy"
        @input="commitText(($event.target as HTMLInputElement).value)"
      />
    </div>
    <div class="character-dye__presets" :aria-label="t('characterCards.colorField.presets')">
      <button
        v-for="preset in presets"
        :key="preset"
        type="button"
        :class="{ active: currentColor === preset.toUpperCase() }"
        :style="{ '--preset-color': preset }"
        :aria-label="t('characterCards.colorField.useColor', { color: preset })"
        :aria-pressed="currentColor === preset.toUpperCase()"
        @click="selectColor(preset)"
      ></button>
    </div>
    <small v-if="invalid" :id="errorId" class="character-dye__error" role="alert">
      {{ t('characterCards.colorField.invalid') }}
    </small>
    <small v-else-if="hint" :id="hintId" class="character-dye__hint">{{ hint }}</small>
  </div>
</template>

<style scoped>
.character-dye { display: grid; min-width: 0; gap: 6px; }
.character-dye__label { color: var(--color-text-secondary); font-size: 10px; font-weight: 700; }
.character-dye__well {
  display: grid;
  min-height: 42px;
  grid-template-columns: 42px auto minmax(0, 1fr);
  align-items: center;
  overflow: hidden;
  border: 1px solid var(--input-border);
  border-radius: 8px;
  background: var(--input-bg);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--color-text-light) 18%, transparent);
}
.character-dye__swatch {
  position: relative;
  display: grid;
  width: 42px;
  height: 100%;
  min-height: 42px;
  place-items: center;
  overflow: hidden;
  border-right: 1px solid var(--color-border);
  background:
    radial-gradient(circle at 35% 30%, rgba(255, 255, 255, 0.62), transparent 26%),
    var(--dye-color);
  cursor: pointer;
}
.character-dye__swatch::after {
  width: 21px;
  height: 9px;
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 50%;
  box-shadow: 0 7px 14px color-mix(in srgb, var(--color-primary) 24%, transparent);
  content: '';
}
.character-dye__swatch input { position: absolute; width: 1px; height: 1px; opacity: 0; }
.character-dye__swatch:focus-within { outline: 3px solid color-mix(in srgb, var(--color-accent) 30%, transparent); outline-offset: -3px; }
.character-dye__prefix { padding-left: 10px; color: var(--color-text-muted); font: 800 8px/1 ui-monospace, Consolas, monospace; letter-spacing: 0.08em; }
.character-dye__hex { min-width: 0; padding: 10px 10px 10px 7px; border: 0; outline: none; color: var(--color-text-main); font: 700 11px/1.2 ui-monospace, Consolas, monospace; text-transform: uppercase; }
.character-dye__well:focus-within { border-color: var(--input-focus); box-shadow: 0 0 0 3px color-mix(in srgb, var(--input-focus) 14%, transparent); }
.character-dye--invalid .character-dye__well { border-color: var(--btn-danger-bg); box-shadow: 0 0 0 3px color-mix(in srgb, var(--btn-danger-bg) 12%, transparent); }
.character-dye__presets { display: flex; flex-wrap: wrap; gap: 5px; }
.character-dye__presets button { width: 18px; height: 18px; padding: 0; border: 2px solid var(--color-panel-bg); border-radius: 50%; outline: 1px solid var(--color-border); background: var(--preset-color); cursor: pointer; }
.character-dye__presets button.active { outline: 2px solid var(--color-secondary); outline-offset: 1px; }
.character-dye__presets button:focus-visible { outline: 3px solid var(--color-accent); outline-offset: 2px; }
.character-dye__error { color: var(--btn-danger-bg); }
.character-dye__hint { color: var(--color-text-secondary); }
.character-dye__error, .character-dye__hint { font-size: 9px; line-height: 1.45; }

@media (prefers-reduced-motion: reduce) {
  .character-dye__presets button { transition: none; }
}
</style>
