<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { createGuild } from '@/api/guild'
import { useDialog } from '@/composables/useDialog'
import ImageCropperDialog from '@/components/ImageCropperDialog.vue'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
const { t } = useI18n()
const { alert } = useDialog()
const name = ref('')
const description = ref('')
const slogan = ref('')
const server = ref('')
const serverError = computed(() => Array.from(server.value.trim()).length > 128
  ? t('common.validation.maxLength', { n: 128 }) : '')
const faction = ref('')
const color = ref('B87333')
const bannerPreview = ref('')
const bannerCropperOpen = ref(false)
const bannerCropperFile = ref<File | null>(null)
const creating = ref(false)

function handleBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    const file = input.files[0]
    if (!file.type.startsWith('image/')) {
      void alert({ title: t('guild.settings.uploadFailed'), message: t('guild.settings.uploadFailed'), type: 'error' })
      input.value = ''
      return
    }
    if (file.size > 20 * 1024 * 1024) {
      void alert({ title: t('guild.create.fileTooLarge'), message: t('guild.settings.bannerSizeLimit'), type: 'error' })
      input.value = ''
      return
    }
    bannerCropperFile.value = file
    bannerCropperOpen.value = true
    input.value = ''
  }
}

function handleBannerCropped(file: File) {
  const reader = new FileReader()
  reader.onload = (event) => {
    bannerPreview.value = event.target?.result as string
    bannerCropperFile.value = null
  }
  reader.readAsDataURL(file)
}

function handleBannerCropperError(error: Error) {
  void alert({ title: t('guild.settings.uploadFailed'), message: error.message || t('guild.settings.uploadFailed'), type: 'error' })
}

async function handleCreate() {
  if (!name.value.trim() || serverError.value) return
  creating.value = true
  try {
    const guild = await createGuild({
      name: name.value,
      description: description.value,
      slogan: slogan.value,
      server: server.value,
      faction: faction.value,
      color: color.value.replace('#', ''),
      banner: bannerPreview.value || undefined
    })
    router.push(`/guild/${guild.id}`)
  } catch (e) {
    console.error('创建失败:', e)
    await alert({
      title: t('common.status.failed'),
      message: e instanceof Error ? e.message : t('common.status.failed'),
      type: 'error'
    })
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="create-page">
    <h1>{{ t('guild.create.title') }}</h1>
    <p class="tip">{{ t('guild.create.tip') }}</p>

    <div class="form">
      <!-- 头图上传 -->
      <div class="field banner-field">
        <label>{{ t('guild.create.banner') }}</label>
        <div
          class="banner-upload"
          :style="{ background: bannerPreview ? `url(${bannerPreview}) center/cover` : `linear-gradient(135deg, #${color}, #4B3621)` }"
          @click="($refs.bannerInput as HTMLInputElement).click()"
        >
          <div class="upload-hint" v-if="!bannerPreview">
            <i class="ri-image-add-line"></i>
            <span>{{ t('guild.create.uploadBanner') }}</span>
          </div>
        </div>
        <input ref="bannerInput" type="file" accept="image/*" hidden @change="handleBannerSelect" />
        <ImageCropperDialog
          v-model="bannerCropperOpen"
          :file="bannerCropperFile"
          :aspect-ratio="3"
          :output-width="1920"
          :output-height="640"
          :max-size-k-b="1536"
          title="调整公会头图"
          @cropped="handleBannerCropped"
          @error="handleBannerCropperError"
        />
      </div>

      <div class="field">
        <label>{{ t('guild.create.nameRequired') }}</label>
        <RInput v-model="name" :placeholder="t('guild.create.namePlaceholder')" />
      </div>

      <div class="field">
        <label>{{ t('guild.create.slogan') }}</label>
        <RInput v-model="slogan" :placeholder="t('guild.create.sloganPlaceholder')" />
      </div>

      <div class="field">
        <label>{{ t('guild.create.description') }}</label>
        <textarea v-model="description" :placeholder="t('guild.create.descriptionPlaceholder')" rows="3"></textarea>
      </div>

      <div class="row">
        <div class="field">
          <label>{{ t('guild.create.server') }}</label>
          <RInput v-model="server" :placeholder="t('guild.create.serverPlaceholder')" :error="serverError" clearable />
        </div>
        <div class="field">
          <label>{{ t('guild.info.faction') }}</label>
          <select v-model="faction">
            <option value="">{{ t('guild.settings.selectFaction') }}</option>
            <option value="alliance">{{ t('guild.info.alliance') }}</option>
            <option value="horde">{{ t('guild.info.horde') }}</option>
            <option value="neutral">{{ t('guild.info.neutral') }}</option>
          </select>
        </div>
      </div>

      <div class="field">
        <label>{{ t('guild.create.themeColor') }}</label>
        <input type="color" :value="'#' + color" @input="color = ($event.target as HTMLInputElement).value.replace('#', '')" />
      </div>

      <div class="actions">
        <RButton @click="router.back()">{{ t('guild.action.cancel') }}</RButton>
        <RButton type="primary" :loading="creating" @click="handleCreate">{{ t('guild.create.create') }}</RButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.create-page {
  max-width: 600px;
  margin: 0 auto;
  padding: 24px;
  color: var(--color-text-main);
}

.create-page h1 {
  font-size: 24px;
  color: var(--color-text-main);
  margin-bottom: 8px;
}

.tip {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 24px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
}

.banner-upload {
  height: 160px;
  border-radius: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
}

.banner-upload:hover {
  opacity: 0.9;
}

.upload-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  padding: 12px 20px;
  border-radius: 8px;
}

.upload-hint i {
  font-size: 32px;
}

.row {
  display: flex;
  gap: 16px;
}

.row .field {
  flex: 1;
}

.field textarea,
.field select {
  padding: 10px 12px;
  border: 1px solid var(--input-border);
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  background: var(--input-bg);
  color: var(--color-text-main);
}

.field textarea {
  resize: vertical;
}

.field input[type="color"] {
  width: 60px;
  height: 36px;
  border: 1px solid var(--input-border);
  border-radius: 8px;
  cursor: pointer;
  background: var(--input-bg);
}

.actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 16px;
}

.field textarea:focus,
.field select:focus {
  outline: none;
  border-color: var(--input-focus);
}

.field textarea::placeholder {
  color: var(--input-placeholder);
}

.field select {
  color-scheme: light;
}
[data-theme="black-gold"] .field select {
  color-scheme: dark;
}
</style>
