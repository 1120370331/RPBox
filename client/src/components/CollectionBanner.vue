<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  getPostCollection,
  getItemCollection,
  listUserCollections,
  addPostToCollection,
  removePostFromCollection,
  addItemToCollection,
  removeItemFromCollection,
  createCollection,
  type CollectionInfo,
  type Collection,
} from '../api/collection'
import { useToast } from '../composables/useToast'

const props = defineProps<{
  type: 'post' | 'item'
  contentId: number
  isAuthor?: boolean
}>()

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const router = useRouter()
const { t } = useI18n()
const toast = useToast()

const collectionInfo = ref<CollectionInfo | null>(null)
const loading = ref(false)

// 编辑模式相关
const editing = ref(false)
const userCollections = ref<Collection[]>([])
const selectedCollectionId = ref<number | null>(null)
const loadingCollections = ref(false)
const saving = ref(false)

// 新建合集弹窗
const showCreateModal = ref(false)
const newName = ref('')
const newDesc = ref('')
const creating = ref(false)

const collection = computed(() => collectionInfo.value?.collection)

async function loadCollection() {
  loading.value = true
  try {
    if (props.type === 'post') {
      collectionInfo.value = await getPostCollection(props.contentId)
    } else {
      collectionInfo.value = await getItemCollection(props.contentId)
    }
  } catch (e) {
    console.error('Failed to load collection:', e)
    collectionInfo.value = null
  } finally {
    loading.value = false
  }
}

function goToCollection() {
  if (collection.value) {
    router.push({ name: 'collection-detail', params: { id: collection.value.id } })
  }
}

// 进入编辑模式
async function startEditing() {
  editing.value = true
  selectedCollectionId.value = collection.value?.id ?? null
  await loadUserCollections()
}

// 取消编辑
function cancelEditing() {
  editing.value = false
  selectedCollectionId.value = null
}

// 加载用户的合集列表
async function loadUserCollections() {
  loadingCollections.value = true
  try {
    const res = await listUserCollections(props.type)
    userCollections.value = res.collections || []
  } catch (e) {
    console.error('Failed to load user collections:', e)
  } finally {
    loadingCollections.value = false
  }
}

// 保存合集变更
async function saveCollection() {
  saving.value = true
  try {
    const oldCollectionId = collection.value?.id
    const newCollectionId = selectedCollectionId.value

    // 如果没有变化，直接关闭
    if (oldCollectionId === newCollectionId) {
      editing.value = false
      return
    }

    // 从旧合集移除
    if (oldCollectionId) {
      if (props.type === 'post') {
        await removePostFromCollection(oldCollectionId, props.contentId)
      } else {
        await removeItemFromCollection(oldCollectionId, props.contentId)
      }
    }

    // 添加到新合集
    if (newCollectionId) {
      if (props.type === 'post') {
        await addPostToCollection(newCollectionId, props.contentId)
      } else {
        await addItemToCollection(newCollectionId, props.contentId)
      }
    }

    toast.success(t('collection.banner.updateSuccess'))
    editing.value = false
    await loadCollection()
    emit('updated')
  } catch (e: any) {
    console.error('Failed to update collection:', e)
    toast.error(e?.response?.data?.error || t('collection.banner.updateFailed'))
  } finally {
    saving.value = false
  }
}

// 新建合集
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
      content_type: props.type,
      is_public: true,
    })
    userCollections.value.unshift(col)
    selectedCollectionId.value = col.id
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

watch(() => props.contentId, loadCollection)
onMounted(loadCollection)
</script>

<template>
  <div class="collection-banner-wrapper">
    <!-- 显示模式 -->
    <div v-if="collection && !editing" class="collection-banner" @click="goToCollection">
      <div class="banner-icon">
        <i class="ri-book-2-line"></i>
      </div>
      <div class="banner-content">
        <span class="banner-label">{{ t('collection.banner.belongsTo') }}：</span>
        <span class="collection-name">《{{ collection.name }}》</span>
        <span class="item-count">{{ t('collection.banner.totalCount', { count: collection.item_count }) }}</span>
      </div>
      <button v-if="isAuthor" class="edit-btn" @click.stop="startEditing" :title="t('collection.banner.edit')">
        <i class="ri-edit-line"></i>
      </button>
      <div v-else class="banner-arrow">
        <i class="ri-arrow-right-s-line"></i>
      </div>
    </div>

    <!-- 作者添加合集入口（当前无合集时显示） -->
    <div v-else-if="!collection && !editing && isAuthor" class="collection-banner add-mode" @click="startEditing">
      <div class="banner-icon add">
        <i class="ri-add-line"></i>
      </div>
      <div class="banner-content">
        <span class="banner-label">{{ t('collection.banner.addToCollection') }}</span>
      </div>
    </div>

    <!-- 编辑模式 -->
    <div v-if="editing" class="collection-banner edit-mode">
      <div class="banner-icon">
        <i class="ri-book-2-line"></i>
      </div>
      <div class="edit-content">
        <select v-model="selectedCollectionId" class="collection-select" :disabled="loadingCollections || saving">
          <option :value="null">{{ t('collection.selector.none') }}</option>
          <option v-for="c in userCollections" :key="c.id" :value="c.id">
            {{ c.name }} ({{ c.item_count }})
          </option>
        </select>
        <button type="button" class="create-btn" @click="showCreateModal = true" :title="t('collection.create.title')">
          <i class="ri-add-line"></i>
        </button>
      </div>
      <div class="edit-actions">
        <button class="cancel-btn" @click="cancelEditing" :disabled="saving">
          {{ t('common.button.cancel') }}
        </button>
        <button class="save-btn" @click="saveCollection" :disabled="saving || loadingCollections">
          {{ saving ? t('common.status.loading') : t('common.button.save') }}
        </button>
      </div>
    </div>

    <!-- 新建合集弹窗 -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showCreateModal" class="modal-mask" @click.self="showCreateModal = false">
          <div class="modal-content">
            <div class="modal-header">
              <h3>{{ t('collection.create.title') }}</h3>
              <button class="modal-close" @click="showCreateModal = false">
                <i class="ri-close-line"></i>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label>{{ t('collection.create.name') }}</label>
                <input v-model="newName" type="text" :placeholder="t('collection.create.namePlaceholder')" maxlength="128" />
              </div>
              <div class="form-group">
                <label>{{ t('collection.create.description') }}</label>
                <textarea v-model="newDesc" :placeholder="t('collection.create.descPlaceholder')" rows="3"></textarea>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn-cancel" @click="showCreateModal = false">{{ t('common.button.cancel') }}</button>
              <button class="btn-confirm" @click="handleCreate" :disabled="creating">
                {{ creating ? t('common.status.loading') : t('common.button.create') }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.collection-banner-wrapper {
  margin: 16px 32px;
}

.collection-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.collection-banner:hover {
  border-color: var(--color-accent);
  background: var(--color-primary-light);
}

.collection-banner.edit-mode {
  cursor: default;
  flex-wrap: wrap;
}

.collection-banner.add-mode {
  border-style: dashed;
}

.banner-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-accent);
  border-radius: 8px;
  color: var(--color-text-light);
  font-size: 18px;
  flex-shrink: 0;
}

.banner-icon.add {
  background: var(--color-border);
}

.banner-content {
  flex: 1;
  display: flex;
  align-items: baseline;
  gap: 4px;
  flex-wrap: wrap;
}

.banner-label {
  font-size: 14px;
  color: var(--color-text-muted);
}

.collection-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-secondary);
}

.item-count {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-left: 4px;
}

.banner-arrow {
  color: var(--color-text-muted);
  font-size: 20px;
  flex-shrink: 0;
}

.collection-banner:hover .banner-arrow {
  color: var(--color-accent);
}

/* 编辑按钮 */
.edit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.edit-btn:hover {
  background: var(--color-panel-bg);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

/* 编辑模式 */
.edit-content {
  flex: 1;
  display: flex;
  gap: 8px;
  min-width: 200px;
}

.collection-select {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  font-size: 14px;
  cursor: pointer;
}

.collection-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.create-btn {
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-panel-bg);
  color: var(--color-text-main);
  cursor: pointer;
  transition: all 0.2s;
}

.create-btn:hover {
  background: var(--color-accent);
  color: var(--color-text-light);
  border-color: var(--color-accent);
}

.edit-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.cancel-btn,
.save-btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}

.cancel-btn:hover:not(:disabled) {
  border-color: var(--color-text-muted);
}

.save-btn {
  background: var(--color-accent);
  border: none;
  color: var(--color-text-light);
}

.save-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.save-btn:disabled,
.cancel-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  border-color: var(--color-accent);
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
  background: var(--color-accent);
  color: var(--color-text-light);
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
