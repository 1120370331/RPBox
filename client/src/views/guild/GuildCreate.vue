<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { createGuild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
const { t } = useI18n()
const name = ref('')
const description = ref('')
const slogan = ref('')
const server = ref('')
const faction = ref('')
const color = ref('B87333')
const bannerPreview = ref('')
const bannerFile = ref<File | null>(null)
const creating = ref(false)

function handleBannerSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    const file = input.files[0]
    if (file.size > 20 * 1024 * 1024) {
      alert(t('guild.create.fileTooLarge'))
      return
    }
    bannerFile.value = file
    const reader = new FileReader()
    reader.onload = (e) => {
      bannerPreview.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

async function handleCreate() {
  if (!name.value.trim()) return
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
          <RInput v-model="server" :placeholder="t('guild.create.serverPlaceholder')" />
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
}

.create-page h1 {
  font-size: 24px;
  color: #4B3621;
  margin-bottom: 8px;
}

.tip {
  font-size: 13px;
  color: #856a52;
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
  color: #4B3621;
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
  color: rgba(255,255,255,0.8);
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
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  background: #fff;
  color: #4B3621;
}

.field textarea {
  resize: vertical;
}

.field input[type="color"] {
  width: 60px;
  height: 36px;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  cursor: pointer;
}

.actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
