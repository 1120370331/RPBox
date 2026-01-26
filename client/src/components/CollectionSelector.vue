<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listUserCollections, createCollection, type Collection } from '../api/collection'
import { useToast } from '../composables/useToast'

const props = defineProps<{
  modelValue?: number | null
  contentType?: 'post' | 'item' | 'mixed'
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: number | null): void
}>()

const { t } = useI18n()
const toast = useToast()

const collections = ref<Collection[]>([])
const selectedId = ref<number | null>(props.modelValue ?? null)
const loading = ref(false)
const showCreateModal = ref(false)
const newName = ref('')
const newDesc = ref('')
const creating = ref(false)

async function loadCollections() {
  loading.value = true
  try {
    const res = await listUserCollections(props.contentType)
    collections.value = res.collections || []
  } catch (e) {
    console.error('Failed to load collections:', e)
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!newName.value.trim()) {
    toast.warning(t('collection.create.nameRequired'))
    return
  }
  creating.value = true
  try {
    const col = await createCollection({
      name: newName.value.trim(),
      description: newDesc.value.trim(),
      content_type: props.contentType || 'mixed',
      is_public: true,
    })
    collections.value.unshift(col)
    selectedId.value = col.id
    emit('update:modelValue', col.id)
    showCreateModal.value = false
    newName.value = ''
    newDesc.value = ''
    toast.success(t('collection.create.success'))
  } catch (e) {
    toast.error(t('collection.create.failed'))
  } finally {
    creating.value = false
  }
}

function onSelect() {
  emit('update:modelValue', selectedId.value)
}

watch(() => props.modelValue, (val) => {
  selectedId.value = val ?? null
})

onMounted(loadCollections)
</script>

<template>
  <div class="collection-selector">
    <label class="selector-label">
      <i class="ri-folder-line"></i>
      {{ $t('collection.selector.label') }}
    </label>
    <div class="selector-row">
      <select v-model="selectedId" class="selector-select" @change="onSelect" :disabled="loading">
        <option :value="null">{{ $t('collection.selector.none') }}</option>
        <option v-for="c in collections" :key="c.id" :value="c.id">
          {{ c.name }} ({{ c.item_count }})
        </option>
      </select>
      <button type="button" class="selector-btn" @click="showCreateModal = true" :title="$t('collection.create.title')">
        <i class="ri-add-line"></i>
      </button>
    </div>

    <!-- Create Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showCreateModal" class="modal-mask" @click.self="showCreateModal = false">
          <div class="modal-content">
            <div class="modal-header">
              <h3>{{ $t('collection.create.title') }}</h3>
              <button class="modal-close" @click="showCreateModal = false">
                <i class="ri-close-line"></i>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label>{{ $t('collection.create.name') }}</label>
                <input
                  v-model="newName"
                  type="text"
                  :placeholder="$t('collection.create.namePlaceholder')"
                  maxlength="128"
                />
              </div>
              <div class="form-group">
                <label>{{ $t('collection.create.description') }}</label>
                <textarea
                  v-model="newDesc"
                  :placeholder="$t('collection.create.descPlaceholder')"
                  rows="3"
                ></textarea>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn-cancel" @click="showCreateModal = false">
                {{ $t('common.button.cancel') }}
              </button>
              <button class="btn-confirm" @click="handleCreate" :disabled="creating">
                {{ creating ? $t('common.status.loading') : $t('common.button.create') }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.collection-selector {
  margin-top: 16px;
}

.selector-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.selector-row {
  display: flex;
  gap: 8px;
}

.selector-select {
  flex: 1;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  font-size: 14px;
  cursor: pointer;
}

.selector-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.selector-btn {
  padding: 10px 14px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  cursor: pointer;
  transition: all 0.2s;
}

.selector-btn:hover {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}

/* Modal styles */
.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.modal-content {
  background: var(--color-panel-bg);
  border-radius: 12px;
  width: 400px;
  max-width: 90vw;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--color-text-main);
}

.modal-close {
  background: none;
  border: none;
  font-size: 20px;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
  margin-bottom: 8px;
}

.form-group input,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-input-bg, #fff);
  color: var(--color-text-main);
  font-size: 14px;
  box-sizing: border-box;
}

.form-group input:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-group textarea {
  resize: vertical;
  min-height: 80px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--color-border);
}

.btn-cancel,
.btn-confirm {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn-cancel {
  background: rgba(128, 64, 48, 0.08);
  color: var(--color-text-main);
}

.btn-cancel:hover {
  background: rgba(128, 64, 48, 0.15);
}

.btn-confirm {
  background: var(--color-primary);
  color: #fff;
}

.btn-confirm:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-confirm:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.9);
}
</style>
