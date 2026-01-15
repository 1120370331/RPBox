<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { createGuild } from '@/api/guild'
import RButton from '@/components/RButton.vue'
import RInput from '@/components/RInput.vue'

const router = useRouter()
const name = ref('')
const description = ref('')
const color = ref('B87333')
const creating = ref(false)

async function handleCreate() {
  if (!name.value.trim()) return
  creating.value = true
  try {
    const guild = await createGuild({
      name: name.value,
      description: description.value,
      color: color.value.replace('#', '')
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
    <div class="form">
      <div class="field">
        <label>公会名称</label>
        <RInput v-model="name" placeholder="输入公会名称" />
      </div>
      <div class="field">
        <label>公会描述</label>
        <textarea v-model="description" placeholder="简要描述公会..." rows="3"></textarea>
      </div>
      <div class="field">
        <label>主题色</label>
        <input v-model="color" type="color" />
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
  max-width: 500px;
  margin: 0 auto;
  padding: 24px;
}

.create-page h1 {
  font-size: 24px;
  color: #4B3621;
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

.field textarea {
  padding: 10px 12px;
  border: 1px solid #d1bfa8;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
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
