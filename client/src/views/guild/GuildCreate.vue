<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { createGuild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
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
      alert('头图文件不能超过20MB')
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
    <h1>创建公会</h1>
    <p class="tip">创建公会需要版主审核通过后才能在公会广场显示</p>

    <div class="form">
      <!-- 头图上传 -->
      <div class="field banner-field">
        <label>公会头图</label>
        <div
          class="banner-upload"
          :style="{ background: bannerPreview ? `url(${bannerPreview}) center/cover` : `linear-gradient(135deg, #${color}, #4B3621)` }"
          @click="($refs.bannerInput as HTMLInputElement).click()"
        >
          <div class="upload-hint" v-if="!bannerPreview">
            <i class="ri-image-add-line"></i>
            <span>点击上传头图</span>
          </div>
        </div>
        <input ref="bannerInput" type="file" accept="image/*" hidden @change="handleBannerSelect" />
      </div>

      <div class="field">
        <label>公会名称 *</label>
        <RInput v-model="name" placeholder="输入公会名称" />
      </div>

      <div class="field">
        <label>公会标语</label>
        <RInput v-model="slogan" placeholder="一句话介绍公会" />
      </div>

      <div class="field">
        <label>公会描述</label>
        <textarea v-model="description" placeholder="详细描述公会..." rows="3"></textarea>
      </div>

      <div class="row">
        <div class="field">
          <label>所在服务器</label>
          <RInput v-model="server" placeholder="如：暗影之月" />
        </div>
        <div class="field">
          <label>阵营</label>
          <select v-model="faction">
            <option value="">请选择</option>
            <option value="alliance">联盟</option>
            <option value="horde">部落</option>
            <option value="neutral">中立</option>
          </select>
        </div>
      </div>

      <div class="field">
        <label>主题色</label>
        <input v-model="color" type="color" :value="'#' + color" @input="color = ($event.target as HTMLInputElement).value.replace('#', '')" />
      </div>

      <div class="actions">
        <RButton @click="router.back()">取消</RButton>
        <RButton type="primary" :loading="creating" @click="handleCreate">创建</RButton>
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
