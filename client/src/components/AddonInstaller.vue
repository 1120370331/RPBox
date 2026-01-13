<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { getAddonLatest, getAddonDownloadUrl } from '@/api/addon'
import RModal from './RModal.vue'
import RButton from './RButton.vue'
import RSelect from './RSelect.vue'

interface Props {
  modelValue: boolean
  wowPath: string
}

interface InstalledAddonInfo {
  installed: boolean
  version: string | null
  path: string | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  installed: []
}>()

const loading = ref(false)
const checking = ref(false)
const error = ref('')
const installedInfo = ref<InstalledAddonInfo | null>(null)
const latestVersion = ref('')
const selectedFlavor = ref('_retail_')

const flavorOptions = [
  { label: '正式服 (_retail_)', value: '_retail_' },
  { label: '怀旧服 (_classic_)', value: '_classic_' },
  { label: '经典旧世 (_classic_era_)', value: '_classic_era_' },
]

const needsUpdate = computed(() => {
  if (!installedInfo.value?.installed || !latestVersion.value) return false
  return installedInfo.value.version !== latestVersion.value
})

const statusText = computed(() => {
  if (checking.value) return '检测中...'
  if (!installedInfo.value) return ''
  if (!installedInfo.value.installed) return '未安装'
  if (needsUpdate.value) return `已安装 v${installedInfo.value.version}，可更新到 v${latestVersion.value}`
  return `已安装 v${installedInfo.value.version}（最新）`
})

async function checkInstalled() {
  checking.value = true
  error.value = ''
  try {
    installedInfo.value = await invoke<InstalledAddonInfo>('check_addon_installed', {
      wowPath: props.wowPath,
      flavor: selectedFlavor.value,
    })
    const latest = await getAddonLatest()
    latestVersion.value = latest.version
  } catch (e: any) {
    error.value = e.message || '检测失败'
  } finally {
    checking.value = false
  }
}

async function installAddon() {
  loading.value = true
  error.value = ''
  try {
    const url = getAddonDownloadUrl(latestVersion.value)
    const response = await fetch(url)
    if (!response.ok) throw new Error('下载失败')
    const arrayBuffer = await response.arrayBuffer()
    const zipData = Array.from(new Uint8Array(arrayBuffer))

    await invoke('install_addon', {
      wowPath: props.wowPath,
      flavor: selectedFlavor.value,
      zipData,
    })

    await checkInstalled()
    emit('installed')
  } catch (e: any) {
    error.value = e.message || '安装失败'
  } finally {
    loading.value = false
  }
}

function close() {
  emit('update:modelValue', false)
}

onMounted(() => {
  if (props.wowPath) checkInstalled()
})
</script>

<template>
  <RModal :model-value="modelValue" title="安装 RPBox 插件" width="480px" @update:model-value="close">
    <div class="addon-installer">
      <p class="addon-installer__desc">需要安装配套插件才能使用剧情记录功能</p>

      <div class="addon-installer__field">
        <label>游戏版本</label>
        <RSelect v-model="selectedFlavor" :options="flavorOptions" @change="checkInstalled" />
      </div>

      <div class="addon-installer__status" :class="{ 'addon-installer__status--error': error }">
        {{ error || statusText }}
      </div>
    </div>

    <template #footer>
      <RButton @click="close">稍后再说</RButton>
      <RButton
        v-if="!installedInfo?.installed || needsUpdate"
        type="primary"
        :loading="loading"
        :disabled="checking || !latestVersion"
        @click="installAddon"
      >
        {{ installedInfo?.installed ? '更新插件' : '安装插件' }}
      </RButton>
    </template>
  </RModal>
</template>

<style scoped>
.addon-installer {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.addon-installer__desc {
  color: var(--color-secondary);
  margin: 0;
}

.addon-installer__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.addon-installer__field label {
  font-weight: 500;
  color: var(--color-primary);
}

.addon-installer__status {
  padding: 12px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  color: var(--color-secondary);
}

.addon-installer__status--error {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}
</style>
