<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  createCollection,
  deleteCollection,
  listUserCollections,
  updateCollection,
  type Collection,
} from '@/api/collection'

const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const collections = ref<Collection[]>([])
const showEditor = ref(false)
const showDeleteDialog = ref(false)
const editingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const form = ref({
  name: '',
  description: '',
  content_type: 'mixed' as 'post' | 'item' | 'mixed',
  is_public: true,
})

const typeLabels: Record<'post' | 'item' | 'mixed', string> = {
  post: t('collection.types.post'),
  item: t('collection.types.item'),
  mixed: t('collection.types.mixed'),
}

async function loadCollections() {
  loading.value = true
  try {
    const res = await listUserCollections()
    collections.value = res.collections || []
  } catch (error) {
    console.error('Failed to load collections', error)
    toast.error((error as Error)?.message || t('collection.manage.loadFailed'))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    name: '',
    description: '',
    content_type: 'mixed',
    is_public: true,
  }
}

function openCreate() {
  editingId.value = null
  resetForm()
  showEditor.value = true
}

function openEdit(collection: Collection) {
  editingId.value = collection.id
  form.value = {
    name: collection.name,
    description: collection.description || '',
    content_type: collection.content_type,
    is_public: collection.is_public,
  }
  showEditor.value = true
}

function openDelete(collectionId: number) {
  deletingId.value = collectionId
  showDeleteDialog.value = true
}

async function submitEditor() {
  const name = form.value.name.trim()
  if (!name) {
    toast.warning(t('collection.manage.nameRequired'))
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      const updated = await updateCollection(editingId.value, {
        name,
        description: form.value.description.trim(),
        content_type: form.value.content_type,
        is_public: form.value.is_public,
      })
      const index = collections.value.findIndex((collection) => collection.id === updated.id)
      if (index >= 0) {
        collections.value.splice(index, 1, updated)
      }
      toast.success(t('collection.manage.updateSuccess'))
    } else {
      const created = await createCollection({
        name,
        description: form.value.description.trim(),
        content_type: form.value.content_type,
        is_public: form.value.is_public,
      })
      collections.value.unshift(created)
      toast.success(t('collection.manage.createSuccess'))
    }
    showEditor.value = false
  } catch (error) {
    console.error('Failed to save collection', error)
    toast.error((error as Error)?.message || t('collection.manage.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingId.value || deleting.value) return
  deleting.value = true
  try {
    await deleteCollection(deletingId.value)
    collections.value = collections.value.filter((collection) => collection.id !== deletingId.value)
    toast.success(t('collection.manage.deleteSuccess'))
    showDeleteDialog.value = false
    deletingId.value = null
  } catch (error) {
    console.error('Failed to delete collection', error)
    toast.error((error as Error)?.message || t('collection.manage.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

onMounted(loadCollections)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ $t('collection.manage.title') }}</h1>
      <button class="add-btn" @click="openCreate"><i class="ri-add-line" /></button>
    </header>

    <div class="sub-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <div v-else-if="collections.length === 0" class="hint">{{ $t('collection.manage.empty') }}</div>
      <div v-else class="collection-list">
        <article v-for="collection in collections" :key="collection.id" class="collection-card">
          <div class="card-main">
            <h3>{{ collection.name }}</h3>
            <p v-if="collection.description">{{ collection.description }}</p>
            <div class="meta-row">
              <span>{{ typeLabels[collection.content_type] }}</span>
              <span>{{ collection.item_count }} {{ $t('collection.manage.items') }}</span>
              <span>{{ collection.is_public ? $t('collection.manage.public') : $t('collection.manage.private') }}</span>
            </div>
          </div>
          <div class="card-actions">
            <button class="action-btn" @click="openEdit(collection)">{{ $t('collection.manage.edit') }}</button>
            <button class="action-btn danger" @click="openDelete(collection.id)">{{ $t('collection.manage.delete') }}</button>
          </div>
        </article>
      </div>
    </div>

    <div v-if="showEditor" class="dialog-mask">
      <div class="dialog">
        <h3>{{ editingId ? $t('collection.manage.editTitle') : $t('collection.manage.createTitle') }}</h3>
        <div class="dialog-form">
          <label>
            <span>{{ $t('collection.manage.name') }}</span>
            <input v-model="form.name" type="text" maxlength="128">
          </label>
          <label>
            <span>{{ $t('collection.manage.description') }}</span>
            <textarea v-model="form.description" rows="3" maxlength="500" />
          </label>
          <label>
            <span>{{ $t('collection.manage.contentType') }}</span>
            <select v-model="form.content_type">
              <option value="post">{{ $t('collection.types.post') }}</option>
              <option value="item">{{ $t('collection.types.item') }}</option>
              <option value="mixed">{{ $t('collection.types.mixed') }}</option>
            </select>
          </label>
          <label class="switch-row">
            <span>{{ $t('collection.manage.public') }}</span>
            <input v-model="form.is_public" type="checkbox">
          </label>
        </div>
        <div class="dialog-actions">
          <button class="action-btn" @click="showEditor = false">{{ $t('collection.manage.cancel') }}</button>
          <button class="action-btn primary" :disabled="saving" @click="submitEditor">
            {{ saving ? $t('common.status.loading') : $t('collection.manage.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showDeleteDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('collection.manage.deleteTitle') }}</h3>
        <p>{{ $t('collection.manage.deleteMessage') }}</p>
        <div class="dialog-actions">
          <button class="action-btn" @click="showDeleteDialog = false">{{ $t('collection.manage.cancel') }}</button>
          <button class="action-btn danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? $t('common.status.loading') : $t('collection.manage.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.add-btn {
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--text-light);
}

.collection-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.collection-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 12px;
}

.card-main h3 {
  font-size: 16px;
}

.card-main p {
  margin-top: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.meta-row {
  margin-top: 8px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.card-actions {
  margin-top: 10px;
  display: flex;
  gap: 8px;
}

.action-btn {
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 8px 12px;
  font-size: 13px;
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.action-btn.danger {
  border-color: #c44747;
  color: #c44747;
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

.dialog p {
  margin-top: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
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
.dialog-form textarea,
.dialog-form select {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--input-bg);
}

.switch-row {
  flex-direction: row !important;
  align-items: center;
  justify-content: space-between;
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
</style>
