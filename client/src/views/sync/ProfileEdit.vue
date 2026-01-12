<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'

interface ProfileForm {
  firstName: string
  lastName: string
  title: string
  race: string
  class: string
  age: string
  eyeColor: string
  height: string
  weight: string
  aboutTitle: string
  aboutText: string
}

const route = useRoute()
const router = useRouter()
const profileId = route.params.id as string

const form = ref<ProfileForm>({
  firstName: '',
  lastName: '',
  title: '',
  race: '',
  class: '',
  age: '',
  eyeColor: '',
  height: '',
  weight: '',
  aboutTitle: '',
  aboutText: ''
})

const isLoading = ref(false)
const isSaving = ref(false)
const profileName = ref('')

onMounted(async () => {
  await loadProfile()
})

async function loadProfile() {
  isLoading.value = true
  try {
    const wowPath = localStorage.getItem('wow_path') || ''
    const profile = await invoke<any>('get_profile_detail', { wowPath: wowPath, profileId: profileId })
    profileName.value = profile.name
    if (profile.characteristics) {
      form.value.firstName = profile.characteristics.firstName || ''
      form.value.lastName = profile.characteristics.lastName || ''
      form.value.title = profile.characteristics.title || ''
      form.value.race = profile.characteristics.race || ''
      form.value.class = profile.characteristics.class || ''
      form.value.age = profile.characteristics.age || ''
      form.value.eyeColor = profile.characteristics.eyeColor || ''
      form.value.height = profile.characteristics.height || ''
      form.value.weight = profile.characteristics.weight || ''
    }
    if (profile.about) {
      form.value.aboutTitle = profile.about.title || ''
      form.value.aboutText = profile.about.text || ''
    }
  } finally {
    isLoading.value = false
  }
}

async function saveProfile() {
  isSaving.value = true
  try {
    const wowPath = localStorage.getItem('wow_path') || ''
    await invoke('update_profile', {
      wowPath: wowPath,
      profileId: profileId,
      updates: {
        characteristics: {
          firstName: form.value.firstName,
          lastName: form.value.lastName,
          title: form.value.title,
          race: form.value.race,
          class: form.value.class,
          age: form.value.age,
          eyeColor: form.value.eyeColor,
          height: form.value.height,
          weight: form.value.weight
        },
        about: {
          title: form.value.aboutTitle,
          text: form.value.aboutText
        }
      }
    })
    router.back()
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <div class="profile-edit">
    <div class="header">
      <button class="back-btn" @click="router.back()">&larr; 返回</button>
      <h2>编辑 {{ profileName }}</h2>
      <button class="btn btn-primary" @click="saveProfile" :disabled="isSaving">
        {{ isSaving ? '保存中...' : '保存' }}
      </button>
    </div>

    <div v-if="isLoading" class="loading">加载中...</div>

    <form v-else @submit.prevent="saveProfile">
      <div class="section">
        <h3>基本信息</h3>
        <div class="form-grid">
          <div class="form-item">
            <label>名</label>
            <input v-model="form.firstName" type="text" placeholder="角色名" />
          </div>
          <div class="form-item">
            <label>姓</label>
            <input v-model="form.lastName" type="text" placeholder="角色姓氏" />
          </div>
          <div class="form-item">
            <label>头衔</label>
            <input v-model="form.title" type="text" placeholder="称号或头衔" />
          </div>
          <div class="form-item">
            <label>种族</label>
            <input v-model="form.race" type="text" placeholder="种族" />
          </div>
          <div class="form-item">
            <label>职业</label>
            <input v-model="form.class" type="text" placeholder="职业" />
          </div>
          <div class="form-item">
            <label>年龄</label>
            <input v-model="form.age" type="text" placeholder="年龄" />
          </div>
          <div class="form-item">
            <label>眼睛颜色</label>
            <input v-model="form.eyeColor" type="text" placeholder="眼睛颜色" />
          </div>
          <div class="form-item">
            <label>身高</label>
            <input v-model="form.height" type="text" placeholder="身高" />
          </div>
          <div class="form-item">
            <label>体重</label>
            <input v-model="form.weight" type="text" placeholder="体重" />
          </div>
        </div>
      </div>

      <div class="section">
        <h3>关于</h3>
        <div class="form-item">
          <label>标题</label>
          <input v-model="form.aboutTitle" type="text" placeholder="关于标题" />
        </div>
        <div class="form-item">
          <label>描述</label>
          <textarea v-model="form.aboutText" rows="8" placeholder="角色描述..."></textarea>
        </div>
      </div>
    </form>
  </div>
</template>

<style scoped>
.profile-edit {
  padding: 1.5rem;
  max-width: 800px;
  margin: 0 auto;
}

.header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.header h2 {
  flex: 1;
  margin: 0;
}

.back-btn {
  background: none;
  border: none;
  font-size: 1rem;
  cursor: pointer;
  color: var(--color-primary);
}

.section {
  background: var(--color-bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1rem;
}

.section h3 {
  margin: 0 0 1rem 0;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.5rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.form-item label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.form-item input,
.form-item textarea {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg);
  color: var(--color-text);
  font-family: inherit;
}

.form-item textarea {
  resize: vertical;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.loading {
  text-align: center;
  padding: 3rem;
  color: var(--color-text-secondary);
}
</style>
