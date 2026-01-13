<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'

type TemplateOption = '1' | '2' | '3'

interface ProfileForm {
  firstName: string
  lastName: string
  title: string
  fullTitle: string
  race: string
  class: string
  age: string
  relationshipStatus: string
  eyeColor: string
  eyeColorHex: string
  classColorHex: string
  height: string
  weight: string
  birthplace: string
  residence: string
  icon: string
  aboutTitle: string
  aboutText: string
  template: TemplateOption
  t2Blocks: { title: string; text: string; icon: string; background: string }[]
  t3Physical: string
  t3Personality: string
  t3History: string
  misc: { name: string; value: string; icon: string; preset?: string }[]
  personality: { left: string; right: string; value: number }[]
}

const route = useRoute()
const router = useRouter()
const profileId = route.params.id as string

const form = ref<ProfileForm>({
  firstName: '',
  lastName: '',
  title: '',
  fullTitle: '',
  race: '',
  class: '',
  age: '',
  relationshipStatus: '0',
  eyeColor: '',
  eyeColorHex: '#442211',
  classColorHex: '#804030',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  icon: '',
  aboutTitle: '',
  aboutText: '',
  template: '3',
  t2Blocks: [
    { title: '框架1', text: '', icon: '', background: '' },
    { title: '框架2', text: '', icon: '', background: '' }
  ],
  t3Physical: '',
  t3Personality: '',
  t3History: '',
  misc: [{ name: '昵称', value: '', icon: '', preset: '3' }],
  personality: [
    { left: '理性', right: '感性', value: 10 },
    { left: '外向', right: '内向', value: 10 },
    { left: '冷静', right: '冲动', value: 10 }
  ]
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
      form.value.fullTitle = profile.characteristics.fullTitle || ''
      form.value.race = profile.characteristics.race || ''
      form.value.class = profile.characteristics.class || ''
      form.value.age = profile.characteristics.age || ''
      form.value.eyeColor = profile.characteristics.eyeColor || ''
      form.value.eyeColorHex = profile.characteristics.eyeColorHex || form.value.eyeColorHex
      form.value.classColorHex = profile.characteristics.classColorHex || form.value.classColorHex
      form.value.height = profile.characteristics.height || ''
      form.value.weight = profile.characteristics.weight || ''
      form.value.birthplace = profile.characteristics.birthplace || ''
      form.value.residence = profile.characteristics.residence || ''
      form.value.relationshipStatus = profile.characteristics.relationshipStatus || '0'
      form.value.icon = profile.characteristics.icon || ''
    }
    if (profile.about) {
      form.value.aboutTitle = (profile.about as any).title || ''
      form.value.aboutText = (profile.about as any).text || ''
      const about = profile.about as any
      if (about.template) form.value.template = String(about.template) as TemplateOption
      if (about.t2?.length) {
        form.value.t2Blocks = about.t2.map((b: any, idx: number) => ({
          title: b.title || `框架${idx + 1}`,
          text: b.text || b.TX || '',
          icon: b.icon || b.IC || '',
          background: b.background || b.BK || ''
        }))
      }
      if (about.t3) {
        form.value.t3Physical = about.t3.PH?.TX || ''
        form.value.t3Personality = about.t3.PS?.TX || ''
        form.value.t3History = about.t3.HI?.TX || ''
      }
    }
    if ((profile as any).misc) {
      form.value.misc = (profile as any).misc
    }
    if ((profile as any).personality) {
      form.value.personality = (profile as any).personality
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
          fullTitle: form.value.fullTitle,
          race: form.value.race,
          class: form.value.class,
          age: form.value.age,
          relationshipStatus: form.value.relationshipStatus,
          eyeColor: form.value.eyeColor,
          eyeColorHex: form.value.eyeColorHex,
          classColorHex: form.value.classColorHex,
          height: form.value.height,
          weight: form.value.weight,
          birthplace: form.value.birthplace,
          residence: form.value.residence,
          icon: form.value.icon
        },
        about: {
          template: form.value.template,
          title: form.value.aboutTitle,
          text: form.value.aboutText,
          t1: { text: form.value.aboutText },
          t2: form.value.t2Blocks.map(b => ({
            title: b.title,
            text: b.text,
            icon: b.icon,
            background: b.background
          })),
          t3: {
            PH: { TX: form.value.t3Physical },
            PS: { TX: form.value.t3Personality },
            HI: { TX: form.value.t3History }
          }
        },
        misc: form.value.misc,
        personality: form.value.personality
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
        <h3>基本信息（characteristics）</h3>
        <div class="form-grid">
          <div class="form-item">
            <label>名 (FN)</label>
            <input v-model="form.firstName" type="text" placeholder="角色名" />
          </div>
          <div class="form-item">
            <label>姓 (LN)</label>
            <input v-model="form.lastName" type="text" placeholder="角色姓氏" />
          </div>
          <div class="form-item">
            <label>头衔 (TI)</label>
            <input v-model="form.title" type="text" placeholder="称号或头衔" />
          </div>
          <div class="form-item">
            <label>全名/英文 (FT)</label>
            <input v-model="form.fullTitle" type="text" placeholder="Full Title" />
          </div>
          <div class="form-item">
            <label>种族 (RA)</label>
            <input v-model="form.race" type="text" placeholder="种族" />
          </div>
          <div class="form-item">
            <label>职业 (CL)</label>
            <input v-model="form.class" type="text" placeholder="职业" />
          </div>
          <div class="form-item">
            <label>年龄 (AG)</label>
            <input v-model="form.age" type="text" placeholder="年龄" />
          </div>
          <div class="form-item">
            <label>身高 (HE)</label>
            <input v-model="form.height" type="text" placeholder="身高" />
          </div>
          <div class="form-item">
            <label>体重 (WE)</label>
            <input v-model="form.weight" type="text" placeholder="体重" />
          </div>
          <div class="form-item">
            <label>眼睛颜色 (EC)</label>
            <input v-model="form.eyeColor" type="text" placeholder="如：琥珀色" />
          </div>
          <div class="form-item">
            <label>眼睛颜色Hex (EH)</label>
            <input v-model="form.eyeColorHex" type="color" />
          </div>
          <div class="form-item">
            <label>职业颜色Hex (CH)</label>
            <input v-model="form.classColorHex" type="color" />
          </div>
          <div class="form-item">
            <label>出生地 (BP)</label>
            <input v-model="form.birthplace" type="text" placeholder="出生地" />
          </div>
          <div class="form-item">
            <label>居住地 (RE)</label>
            <input v-model="form.residence" type="text" placeholder="现居住地" />
          </div>
          <div class="form-item">
            <label>感情状态 (RS)</label>
            <select v-model="form.relationshipStatus">
              <option value="0">未知</option>
              <option value="1">单身</option>
              <option value="2">恋爱中</option>
              <option value="3">已婚</option>
              <option value="4">离异</option>
              <option value="5">丧偶</option>
            </select>
          </div>
          <div class="form-item">
            <label>头像图标 (IC)</label>
            <input v-model="form.icon" type="text" placeholder="游戏图标路径" />
          </div>
        </div>
      </div>

      <div class="section">
        <h3>关于 (about)</h3>
        <div class="form-grid template-grid">
          <div class="form-item">
            <label>模板选择 (TE)</label>
            <div class="radio-row">
              <label><input type="radio" value="1" v-model="form.template" /> 模板1</label>
              <label><input type="radio" value="2" v-model="form.template" /> 模板2</label>
              <label><input type="radio" value="3" v-model="form.template" /> 模板3</label>
            </div>
          </div>
          <div class="form-item">
            <label>标题</label>
            <input v-model="form.aboutTitle" type="text" placeholder="关于标题" />
          </div>
        </div>

        <div v-if="form.template === '1'" class="form-item">
          <label>模板1 文本 (T1.TX)</label>
          <textarea v-model="form.aboutText" rows="6" placeholder="角色描述..."></textarea>
        </div>

        <div v-if="form.template === '2'" class="t2-grid">
          <div
            v-for="(block, idx) in form.t2Blocks"
            :key="idx"
            class="t2-card"
          >
            <div class="t2-head">
              <strong>框架 {{ idx + 1 }}</strong>
              <input v-model="block.title" type="text" placeholder="标题" />
            </div>
            <textarea v-model="block.text" rows="4" placeholder="内容 (T2[].TX)"></textarea>
            <div class="t2-row">
              <input v-model="block.icon" type="text" placeholder="图标 (IC)" />
              <input v-model="block.background" type="text" placeholder="背景 (BK)" />
            </div>
          </div>
          <button class="btn-secondary ghost" type="button" @click="form.t2Blocks.push({ title: '新框架', text: '', icon: '', background: '' })">+ 添加框架</button>
        </div>

        <div v-if="form.template === '3'" class="t3-grid">
          <div class="form-item">
            <label>外貌 (T3.PH.TX)</label>
            <textarea v-model="form.t3Physical" rows="4" placeholder="外貌描述"></textarea>
          </div>
          <div class="form-item">
            <label>性格 (T3.PS.TX)</label>
            <textarea v-model="form.t3Personality" rows="4" placeholder="性格描述"></textarea>
          </div>
          <div class="form-item">
            <label>历史 (T3.HI.TX)</label>
            <textarea v-model="form.t3History" rows="4" placeholder="历史描述"></textarea>
          </div>
        </div>
      </div>

      <div class="section">
        <h3>性格特征 (PS 数值)</h3>
        <div class="traits-grid">
          <div class="trait" v-for="(trait, idx) in form.personality" :key="idx">
            <div class="trait-head">
              <input v-model="trait.left" type="text" placeholder="左侧特征" />
              <input v-model="trait.right" type="text" placeholder="右侧特征" />
            </div>
            <input v-model.number="trait.value" type="range" min="0" max="20" />
            <div class="trait-value">当前值: {{ trait.value }}</div>
          </div>
        </div>
      </div>

      <div class="section">
        <h3>其他信息 (MI)</h3>
        <div class="misc-grid">
          <div class="misc-row" v-for="(item, idx) in form.misc" :key="idx">
            <input v-model="item.name" type="text" placeholder="字段名称 (NA)" />
            <input v-model="item.value" type="text" placeholder="字段值 (VA)" />
            <input v-model="item.icon" type="text" placeholder="图标 (IC)" />
            <input v-model="item.preset" type="text" placeholder="预设ID (可选)" />
          </div>
          <button class="btn-secondary ghost" type="button" @click="form.misc.push({ name: '', value: '', icon: '' })">+ 添加字段</button>
        </div>
      </div>
    </form>
  </div>
</template>

<style scoped>
.profile-edit {
  padding: 1.5rem;
  max-width: 1000px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.header {
  display: flex;
  align-items: center;
  gap: 1rem;
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
  background: #fff;
  border-radius: 12px;
  padding: 1.25rem;
  border: 1px solid var(--color-border, #E8DCCF);
  box-shadow: var(--shadow-sm, 0 4px 12px rgba(75,54,33,0.05));
}

.section h3 {
  margin: 0 0 1rem 0;
  color: var(--color-primary);
  font-size: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 0.75rem;
}

.template-grid {
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-item label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.form-item input,
.form-item textarea,
.form-item select {
  padding: 0.55rem 0.65rem;
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 8px;
  background: #fffcf9;
  color: var(--color-text-main);
  font-family: inherit;
}

.form-item textarea {
  resize: vertical;
}

.radio-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.t2-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 12px;
  margin-top: 10px;
}

.t2-card {
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 10px;
  padding: 10px;
  background: #fffdfb;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.t2-head {
  display: flex;
  gap: 8px;
  align-items: center;
}

.t2-head input {
  flex: 1;
}

.t2-row {
  display: flex;
  gap: 8px;
}

.t3-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 10px;
  margin-top: 10px;
}

.traits-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 10px;
}

.trait {
  border: 1px solid var(--color-border, #E8DCCF);
  border-radius: 10px;
  padding: 10px;
  background: #fffdfb;
}

.trait-head {
  display: flex;
  gap: 8px;
}

.trait-head input {
  flex: 1;
}

.trait-value {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 6px;
}

.misc-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.misc-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 8px;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 8px;
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

.btn-secondary {
  padding: 10px 14px;
  background: rgba(128, 64, 48, 0.1);
  color: var(--color-primary);
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}

.ghost {
  background: rgba(128, 64, 48, 0.06);
  border: 1px solid rgba(128, 64, 48, 0.15);
}

.loading {
  text-align: center;
  padding: 3rem;
  color: var(--color-text-secondary);
}
</style>
