<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '@shared/stores/toast'
import {
  addItemToCollection,
  getItemCollection,
  removeItemFromCollection,
} from '@/api/collection'
import {
  createItem,
  deleteItem,
  deleteItemImage,
  getItem,
  type ItemImage,
  type UpdateItemRequest,
  updateItem,
  uploadImage,
  uploadItemImages,
} from '@/api/item'
import { resolveApiUrl } from '@/api/image'
import MobileCollectionSelector from '@/components/MobileCollectionSelector.vue'
import MobileRichEditor from '@/components/MobileRichEditor.vue'

interface ItemEditorForm {
  name: string
  type: 'item' | 'campaign' | 'artwork'
  description: string
  detail_content: string
  import_code: string
  preview_image: string
  status: 'draft' | 'published'
  is_public: boolean
  enable_watermark: boolean
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToastStore()

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const coverUploading = ref(false)
const showDeleteDialog = ref(false)
const coverInput = ref<HTMLInputElement | null>(null)
const artworkInput = ref<HTMLInputElement | null>(null)
const coverPreview = ref('')
const existingImages = ref<ItemImage[]>([])
const deletedImageIds = ref<number[]>([])
const newImages = ref<Array<{ file: File; preview: string }>>([])
const selectedCollectionId = ref<number | null>(null)
const originalCollectionId = ref<number | null>(null)

const form = ref<ItemEditorForm>({
  name: '',
  type: 'item',
  description: '',
  detail_content: '',
  import_code: '',
  preview_image: '',
  status: 'published',
  is_public: true,
  enable_watermark: true,
})

const itemId = computed(() => Number(route.params.id))
const isEdit = computed(() => Number.isFinite(itemId.value) && itemId.value > 0)
const isArtwork = computed(() => form.value.type === 'artwork')
const pageTitle = computed(() => isEdit.value ? t('market.editor.editTitle') : t('market.editor.createTitle'))
const visibleExistingImages = computed(() => existingImages.value.filter((img) => !deletedImageIds.value.includes(img.id)))
const totalArtworkImages = computed(() => visibleExistingImages.value.length + newImages.value.length)

async function loadItemForEdit() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res = await getItem(itemId.value)
    form.value = {
      name: res.item.name || '',
      type: res.item.type || 'item',
      description: res.item.description || '',
      detail_content: res.item.detail_content || '',
      import_code: res.item.import_code || '',
      preview_image: res.item.preview_image || '',
      status: res.item.status === 'draft' ? 'draft' : 'published',
      is_public: res.item.is_public ?? true,
      enable_watermark: res.item.enable_watermark ?? true,
    }
    coverPreview.value = resolveApiUrl(res.item.preview_image || res.item.preview_image_url || '')
    existingImages.value = res.images || []

    const collectionRes = await getItemCollection(itemId.value)
    const collectionId = collectionRes.collection?.id ?? null
    originalCollectionId.value = collectionId
    selectedCollectionId.value = collectionId
  } catch (error) {
    console.error('Failed to load item detail', error)
    toast.error((error as Error)?.message || t('market.editor.loadFailed'))
    router.replace({ name: 'my-items' })
  } finally {
    loading.value = false
  }
}

async function onCoverFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning(t('market.editor.invalidImage'))
    input.value = ''
    return
  }

  coverUploading.value = true
  try {
    const res = await uploadImage(file)
    const url = res.url || ''
    if (!url) throw new Error(t('market.editor.uploadFailed'))
    form.value.preview_image = url
    coverPreview.value = resolveApiUrl(url)
    toast.success(t('market.editor.uploadSuccess'))
  } catch (error) {
    console.error('Failed to upload preview image', error)
    toast.error((error as Error)?.message || t('market.editor.uploadFailed'))
  } finally {
    coverUploading.value = false
    input.value = ''
  }
}

function removeCoverImage() {
  form.value.preview_image = ''
  coverPreview.value = ''
}

function onArtworkFilesSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  if (files.length === 0) return

  const maxAllowed = 20 - totalArtworkImages.value
  if (maxAllowed <= 0) {
    toast.warning(t('market.editor.maxArtworkImages'))
    input.value = ''
    return
  }

  files.slice(0, maxAllowed).forEach((file) => {
    if (!file.type.startsWith('image/')) return
    const reader = new FileReader()
    reader.onload = () => {
      newImages.value.push({
        file,
        preview: String(reader.result || ''),
      })
    }
    reader.readAsDataURL(file)
  })

  input.value = ''
}

function removeNewImage(index: number) {
  newImages.value.splice(index, 1)
}

function markExistingImageForDelete(imageId: number) {
  if (!deletedImageIds.value.includes(imageId)) {
    deletedImageIds.value.push(imageId)
  }
}

function restoreExistingImage(imageId: number) {
  deletedImageIds.value = deletedImageIds.value.filter((id) => id !== imageId)
}

async function syncItemCollection(targetItemId: number) {
  if (selectedCollectionId.value === originalCollectionId.value) return

  if (originalCollectionId.value) {
    try {
      await removeItemFromCollection(originalCollectionId.value, targetItemId)
    } catch (error) {
      console.warn('Failed to remove item from old collection', error)
    }
  }

  if (selectedCollectionId.value) {
    await addItemToCollection(selectedCollectionId.value, targetItemId)
  }

  originalCollectionId.value = selectedCollectionId.value
}

function validateForm() {
  if (!form.value.name.trim()) {
    toast.warning(t('market.editor.nameRequired'))
    return false
  }
  if (!isArtwork.value && !form.value.import_code.trim()) {
    toast.warning(t('market.editor.importCodeRequired'))
    return false
  }
  if (isArtwork.value && totalArtworkImages.value === 0) {
    toast.warning(t('market.editor.artworkImageRequired'))
    return false
  }
  return true
}

function buildPayload(status: 'draft' | 'published'): UpdateItemRequest {
  return {
    name: form.value.name.trim(),
    description: form.value.description.trim(),
    detail_content: form.value.detail_content.trim(),
    import_code: form.value.import_code,
    preview_image: form.value.preview_image,
    status,
    is_public: form.value.is_public,
    enable_watermark: form.value.enable_watermark,
  }
}

function handleDetailContentChange(value: string) {
  form.value.detail_content = value
}

async function submit(status: 'draft' | 'published') {
  if (!validateForm()) return

  saving.value = true
  try {
    let targetId = itemId.value
    if (isEdit.value) {
      await updateItem(targetId, buildPayload(status))
    } else {
      const created = await createItem({
        ...buildPayload(status),
        type: form.value.type,
      })
      targetId = created.id
    }

    if (isArtwork.value) {
      if (isEdit.value && deletedImageIds.value.length > 0) {
        await Promise.all(deletedImageIds.value.map((imageId) => deleteItemImage(targetId, imageId)))
      }
      if (newImages.value.length > 0) {
        await uploadItemImages(targetId, newImages.value.map((img) => img.file))
      }
    }

    await syncItemCollection(targetId)

    toast.success(status === 'published' ? t('market.editor.publishSuccess') : t('market.editor.draftSuccess'))
    router.replace(isEdit.value
      ? { name: 'item-detail', params: { id: targetId } }
      : { name: 'my-items' })
  } catch (error) {
    console.error('Failed to submit item', error)
    toast.error((error as Error)?.message || t('market.editor.submitFailed'))
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!isEdit.value || deleting.value) return
  deleting.value = true
  try {
    await deleteItem(itemId.value)
    toast.success(t('market.editor.deleteSuccess'))
    router.replace({ name: 'my-items' })
  } catch (error) {
    console.error('Failed to delete item', error)
    toast.error((error as Error)?.message || t('market.editor.deleteFailed'))
  } finally {
    deleting.value = false
    showDeleteDialog.value = false
  }
}

onMounted(loadItemForEdit)
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ pageTitle }}</h1>
    </header>

    <div class="sub-body editor-body">
      <div v-if="loading" class="hint">{{ $t('common.status.loading') }}</div>
      <template v-else>
        <section class="editor-card">
          <label class="field">
            <span>{{ $t('market.editor.name') }}</span>
            <input v-model="form.name" type="text" maxlength="120">
          </label>

          <label class="field">
            <span>{{ $t('market.editor.type') }}</span>
            <select v-model="form.type" :disabled="isEdit">
              <option value="item">{{ $t('market.types.item') }}</option>
              <option value="campaign">{{ $t('market.types.campaign') }}</option>
              <option value="artwork">{{ $t('market.types.artwork') }}</option>
            </select>
          </label>

          <label class="switch-field">
            <span>{{ $t('market.editor.publicVisible') }}</span>
            <input v-model="form.is_public" type="checkbox">
          </label>

          <label v-if="isArtwork" class="switch-field">
            <span>{{ $t('market.editor.enableWatermark') }}</span>
            <input v-model="form.enable_watermark" type="checkbox">
          </label>

          <div class="field">
            <span>{{ $t('market.editor.cover') }}</span>
            <div class="cover-box">
              <img v-if="coverPreview" :src="coverPreview" alt="">
              <button v-if="coverPreview" type="button" class="inline-btn" @click="removeCoverImage">
                {{ $t('market.editor.removeCover') }}
              </button>
              <button type="button" class="inline-btn" :disabled="coverUploading" @click="coverInput?.click()">
                {{ coverUploading ? $t('common.status.loading') : $t('market.editor.uploadCover') }}
              </button>
              <input ref="coverInput" type="file" accept="image/*" hidden @change="onCoverFileChange">
            </div>
          </div>

          <label class="field">
            <span>{{ $t('market.editor.description') }}</span>
            <textarea v-model="form.description" rows="4" />
          </label>

          <label class="field">
            <span>{{ $t('market.editor.detailContent') }}</span>
            <MobileRichEditor
              :model-value="form.detail_content"
              :placeholder="$t('market.editor.detailContent')"
              @update:modelValue="handleDetailContentChange"
            />
          </label>

          <label v-if="!isArtwork" class="field">
            <span>{{ $t('market.editor.importCode') }}</span>
            <textarea v-model="form.import_code" rows="6" />
          </label>

          <div v-if="isArtwork" class="field">
            <span>{{ $t('market.editor.artworkImages') }}</span>
            <div class="artwork-grid">
              <div
                v-for="image in existingImages"
                :key="image.id"
                class="artwork-item"
                :class="{ removed: deletedImageIds.includes(image.id) }"
              >
                <img :src="resolveApiUrl(image.image_url)" alt="">
                <button
                  v-if="!deletedImageIds.includes(image.id)"
                  type="button"
                  class="mini-btn"
                  @click="markExistingImageForDelete(image.id)"
                >
                  <i class="ri-delete-bin-line" />
                </button>
                <button
                  v-else
                  type="button"
                  class="mini-btn"
                  @click="restoreExistingImage(image.id)"
                >
                  <i class="ri-arrow-go-back-line" />
                </button>
              </div>

              <div v-for="(image, index) in newImages" :key="`new-${index}`" class="artwork-item new">
                <img :src="image.preview" alt="">
                <button type="button" class="mini-btn" @click="removeNewImage(index)">
                  <i class="ri-close-line" />
                </button>
              </div>

              <button v-if="totalArtworkImages < 20" type="button" class="add-artwork-btn" @click="artworkInput?.click()">
                <i class="ri-add-line" />
              </button>
              <input ref="artworkInput" type="file" accept="image/*" multiple hidden @change="onArtworkFilesSelected">
            </div>
          </div>

          <MobileCollectionSelector
            v-model="selectedCollectionId"
            content-type="item"
          />
        </section>

        <section class="action-bar">
          <button type="button" class="action-btn" :disabled="saving" @click="submit('draft')">
            {{ $t('market.editor.saveDraft') }}
          </button>
          <button type="button" class="action-btn primary" :disabled="saving" @click="submit('published')">
            {{ $t('market.editor.publish') }}
          </button>
          <button v-if="isEdit" type="button" class="action-btn danger" :disabled="saving" @click="showDeleteDialog = true">
            {{ $t('market.editor.delete') }}
          </button>
        </section>
      </template>
    </div>

    <div v-if="showDeleteDialog" class="dialog-mask">
      <div class="dialog">
        <h3>{{ $t('market.editor.deleteTitle') }}</h3>
        <p>{{ $t('market.editor.deleteMessage') }}</p>
        <div class="dialog-actions">
          <button type="button" class="action-btn" @click="showDeleteDialog = false">{{ $t('market.editor.cancel') }}</button>
          <button type="button" class="action-btn danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? $t('common.status.loading') : $t('market.editor.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor-body {
  padding-bottom: calc(88px + var(--safe-bottom, 0px));
}

.editor-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field > span {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.field input,
.field textarea,
.field select {
  width: 100%;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--input-bg);
  color: var(--text-dark);
}

.field textarea {
  resize: vertical;
}

.switch-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
}

.switch-field input {
  width: auto;
}

.cover-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cover-box img {
  width: 100%;
  border-radius: var(--radius-sm);
  max-height: 220px;
  object-fit: cover;
}

.inline-btn {
  width: fit-content;
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  padding: 8px 12px;
  font-size: 13px;
}

.artwork-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.artwork-item {
  position: relative;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  aspect-ratio: 1 / 1;
}

.artwork-item.removed {
  opacity: 0.4;
}

.artwork-item.new {
  border-color: #5b8c5a;
}

.artwork-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-btn {
  position: absolute;
  right: 6px;
  top: 6px;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
}

.add-artwork-btn {
  border: 1px dashed var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--color-text-secondary);
  font-size: 22px;
  min-height: 88px;
}

.action-bar {
  position: fixed;
  left: calc(var(--safe-left, 0px) + 10px);
  right: calc(var(--safe-right, 0px) + 10px);
  bottom: calc(var(--safe-bottom, 0px) + 8px);
  z-index: 110;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 8px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(75, 54, 33, 0.12);
  box-shadow: 0 8px 20px rgba(44, 24, 16, 0.15);
}

.action-btn {
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text-dark);
  min-height: 40px;
  font-size: 13px;
  font-weight: 600;
}

.action-btn.primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--text-light);
}

.action-btn.danger {
  grid-column: span 2;
  border-color: #c44747;
  color: #c44747;
}

.dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.48);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 16px;
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

.dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
