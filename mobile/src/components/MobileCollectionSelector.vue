<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@shared/stores/toast'
import { createCollection, listUserCollections, type Collection } from '@/api/collection'

const props = withDefaults(defineProps<{
  modelValue?: number | null
  contentType?: 'post' | 'item' | 'mixed'
}>(), {
  modelValue: null,
  contentType: 'mixed',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: number | null): void
}>()

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const creating = ref(false)
const collections = ref<Collection[]>([])
const selectedId = ref<number | null>(props.modelValue)
const showCreate = ref(false)
const form = ref({
  name: '',
  description: '',
  is_public: true,
})

async function loadCollections() {
  loading.value = true
  try {
    const res = await listUserCollections(props.contentType)
    collections.value = res.collections || []
  } catch (error) {
    console.error('Failed to load collections', error)
  } finally {
    loading.value = false
  }
}

function onSelectChange() {
  emit('update:modelValue', selectedId.value)
}

function openCreateDialog() {
  showCreate.value = true
}

function closeCreateDialog() {
  showCreate.value = false
  form.value = {
    name: '',
    description: '',
    is_public: true,
  }
}

async function submitCreate() {
  const name = form.value.name.trim()
  if (!name) {
    toast.warning(t('collection.selector.nameRequired'))
    return
  }
  creating.value = true
  try {
    const collection = await createCollection({
      name,
      description: form.value.description.trim(),
      content_type: props.contentType,
      is_public: form.value.is_public,
    })
    collections.value.unshift(collection)
    selectedId.value = collection.id
    emit('update:modelValue', collection.id)
    toast.success(t('collection.selector.createSuccess'))
    closeCreateDialog()
  } catch (error) {
    console.error('Failed to create collection', error)
    toast.error((error as Error)?.message || t('collection.selector.createFailed'))
  } finally {
    creating.value = false
  }
}

function goManageCollections() {
  router.push({ name: 'my-collections' })
}

watch(() => props.modelValue, (value) => {
  selectedId.value = value ?? null
})

onMounted(loadCollections)
</script>

<template>
  <section class="collection-selector">
    <div class="selector-head">
      <span>{{ $t('collection.selector.label') }}</span>
      <button type="button" class="link-btn" @click="goManageCollections">
        {{ $t('collection.selector.manage') }}
      </button>
    </div>

    <div class="selector-row">
      <select v-model="selectedId" :disabled="loading" @change="onSelectChange">
        <option :value="null">{{ $t('collection.selector.none') }}</option>
        <option v-for="collection in collections" :key="collection.id" :value="collection.id">
          {{ collection.name }} ({{ collection.item_count }})
        </option>
      </select>
      <button type="button" class="icon-btn" :disabled="loading" @click="openCreateDialog">
        <i class="ri-add-line" />
      </button>
    </div>

    <div v-if="showCreate" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('collection.selector.createTitle') }}</h3>
        <div class="dialog-form">
          <label>
            <span>{{ $t('collection.selector.name') }}</span>
            <input v-model="form.name" type="text" maxlength="128">
          </label>
          <label>
            <span>{{ $t('collection.selector.description') }}</span>
            <textarea v-model="form.description" rows="3" maxlength="500" />
          </label>
          <label class="switch-row">
            <span>{{ $t('collection.selector.public') }}</span>
            <input v-model="form.is_public" type="checkbox">
          </label>
        </div>
        <div class="dialog-actions">
          <button type="button" class="action-btn" @click="closeCreateDialog">
            {{ $t('collection.selector.cancel') }}
          </button>
          <button type="button" class="action-btn primary" :disabled="creating" @click="submitCreate">
            {{ creating ? $t('common.status.loading') : $t('collection.selector.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.collection-selector {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selector-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.link-btn {
  border: none;
  background: transparent;
  color: var(--color-secondary);
  font-size: 12px;
}

.selector-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.selector-row select {
  flex: 1;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  min-height: 40px;
  padding: 0 10px;
}

.icon-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  font-size: 18px;
}

.dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.48);
}

.dialog {
  width: 100%;
  max-width: 360px;
  border-radius: var(--radius-md);
  background: var(--color-panel-bg);
  padding: 14px;
}

.dialog h3 {
  font-size: 16px;
}

.dialog-form {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-form label {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dialog-form span {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.dialog-form input,
.dialog-form textarea {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--input-bg);
}

.switch-row {
  flex-direction: row !important;
  justify-content: space-between;
  align-items: center;
}

.switch-row input {
  width: auto;
}

.dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.action-btn {
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 8px 14px;
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}
</style>
