<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { useRoute, useRouter } from 'vue-router'
import {
  getCharacterCard,
  syncCharacterCardFromTRP3,
  updateCharacterCard,
  uploadCharacterCardPortrait,
  type CharacterCard,
} from '@/api/characterCard'
import ImageCropperDialog from '@/components/ImageCropperDialog.vue'
import CharacterCardPortrait from '@/components/character-cards/CharacterCardPortrait.vue'
import PostQuickJump from '@/components/PostQuickJump.vue'
import RModal from '@/components/RModal.vue'
import TiptapEditor from '@/components/TiptapEditor.vue'
import { useDialog } from '@/composables/useDialog'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'
import {
  createCharacterCardDraft,
  createEmptyCharacterCardDraft,
  getCharacterCardDisplayName,
  type CharacterCardEditorTab,
} from '@/utils/characterCardDraft'
import { normalizeCharacterCardHexForTRP3 } from '@/utils/characterCardColor'

interface EditorHandle {
  insertContent: (html: string) => void
}

type RichTextTab = Exclude<CharacterCardEditorTab, 'basic'>

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const dialog = useDialog()
const userStore = useUserStore()

const cardId = computed(() => Number(route.params.id))
const card = ref<CharacterCard | null>(null)
const form = reactive(createEmptyCharacterCardDraft())
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const activeTab = ref<CharacterCardEditorTab>('basic')
const originalSnapshot = ref('')

const portraitInput = ref<HTMLInputElement | null>(null)
const cropperOpen = ref(false)
const cropperFile = ref<File | null>(null)
const portraitUploading = ref(false)
const syncing = ref(false)
const writingBack = ref(false)
const uploadedPortraitPreview = ref('')
const portraitPreviewObjectUrls = new Set<string>()
const portraitPreviewOpen = ref(false)

const quickJumpOpen = ref(false)
const quickJumpTarget = ref<RichTextTab>('background')
const backgroundEditor = ref<EditorHandle | null>(null)
const impressionEditor = ref<EditorHandle | null>(null)
const otherEditor = ref<EditorHandle | null>(null)

const tabs: Array<{ id: CharacterCardEditorTab; label: string; icon: string }> = [
  { id: 'basic', label: '基础信息', icon: 'ri-id-card-line' },
  { id: 'background', label: '背景故事', icon: 'ri-book-open-line' },
  { id: 'impression', label: '第一印象', icon: 'ri-eye-2-line' },
  { id: 'other', label: '其他', icon: 'ri-archive-stack-line' },
]

const displayName = computed(() => getCharacterCardDisplayName(form))
const isDirty = computed(() => originalSnapshot.value !== JSON.stringify(form))
const hasPortrait = computed(() => Boolean(form.portrait_image_url))

const isTauriRuntime = computed(() => typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window)
const wowPath = computed(() => localStorage.getItem('wow_path')?.trim() || '')
const writeBackDisabledReason = computed(() => {
  if (!isTauriRuntime.value) return '仅 RPBox 桌面客户端可以写入本地 TRP3 文件。'
  if (!wowPath.value) return '请先在设置中选择有效的魔兽世界 WTF 路径。'
  if (!card.value?.source_account_id || !card.value?.source_profile_id) {
    return '这张人物卡没有可靠的来源账号和本地 profile 标识，无法安全定位。'
  }
  return ''
})
const canSyncFromBackup = computed(() => Boolean(card.value?.source_backup_id && card.value?.source_profile_id))

watch(cardId, () => void loadCard(), { immediate: true })
onBeforeUnmount(revokeAllPortraitPreviewObjectUrls)

async function loadCard() {
  if (!Number.isFinite(cardId.value) || cardId.value <= 0) {
    loadError.value = '人物卡地址无效'
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const result = await getCharacterCard(cardId.value)
    if (!userStore.user?.id || userStore.user.id !== result.user_id) {
      card.value = null
      loadError.value = '这张人物卡不存在，或你没有编辑权限。'
      return
    }
    applyCard(result)
  } catch (error: unknown) {
    loadError.value = error instanceof Error ? error.message : '人物卡暂不可用'
  } finally {
    loading.value = false
  }
}

function applyCard(nextCard: CharacterCard) {
  card.value = nextCard
  Object.assign(form, createCharacterCardDraft(nextCard))
  revokeUploadedPortraitPreview()
  originalSnapshot.value = JSON.stringify(form)
}

function applySyncedCard(nextCard: CharacterCard) {
  const preserved = {
    display_name: form.display_name,
    summary: form.summary,
    background_story: form.background_story,
    first_impression: form.first_impression,
    other_content: form.other_content,
    portrait_image_url: form.portrait_image_url,
    status: form.status,
    visibility: form.visibility,
    sort_order: form.sort_order,
  }
  const preservedPortraitPreview = uploadedPortraitPreview.value

  card.value = nextCard
  Object.assign(form, createCharacterCardDraft(nextCard))
  originalSnapshot.value = JSON.stringify(form)
  Object.assign(form, preserved)
  uploadedPortraitPreview.value = preservedPortraitPreview
}

function selectTab(tab: CharacterCardEditorTab) {
  activeTab.value = tab
}

function handleTabKeydown(event: KeyboardEvent, currentTab: CharacterCardEditorTab) {
  const currentIndex = tabs.findIndex((tab) => tab.id === currentTab)
  if (currentIndex < 0) return

  let nextIndex: number | null = null
  switch (event.key) {
    case 'ArrowRight':
    case 'ArrowDown':
      nextIndex = (currentIndex + 1) % tabs.length
      break
    case 'ArrowLeft':
    case 'ArrowUp':
      nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = tabs.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  const nextTab = tabs[nextIndex]
  activeTab.value = nextTab.id
  void nextTick(() => {
    document.getElementById(`character-tab-${nextTab.id}`)?.focus()
  })
}

function openQuickJump(tab: RichTextTab) {
  quickJumpTarget.value = tab
  quickJumpOpen.value = true
}

function insertQuickJump(html: string) {
  const editors: Record<RichTextTab, EditorHandle | null> = {
    background: backgroundEditor.value,
    impression: impressionEditor.value,
    other: otherEditor.value,
  }
  editors[quickJumpTarget.value]?.insertContent(html)
}

function triggerPortraitUpload() {
  portraitInput.value?.click()
}

function handlePortraitFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择 PNG、JPG、WebP 或 GIF 图片')
    return
  }
  if (file.size > 20 * 1024 * 1024) {
    toast.warning('角色大图不能超过 20MB')
    return
  }
  cropperFile.value = file
  cropperOpen.value = true
}

async function handlePortraitCropped(file: File) {
  const previousPreview = uploadedPortraitPreview.value
  const nextPreview = URL.createObjectURL(file)
  portraitPreviewObjectUrls.add(nextPreview)
  uploadedPortraitPreview.value = nextPreview
  portraitUploading.value = true
  try {
    form.portrait_image_url = await uploadCharacterCardPortrait(file)
    revokeObjectUrl(previousPreview)
    toast.success('角色大图已上传，保存人物卡后生效')
  } catch (error: unknown) {
    revokeObjectUrl(nextPreview)
    uploadedPortraitPreview.value = previousPreview
    toast.error(error instanceof Error ? error.message : '角色大图上传失败')
  } finally {
    portraitUploading.value = false
    cropperFile.value = null
  }
}

function handleCropperError(error: Error) {
  toast.error(error.message || '图片处理失败')
}

function removePortrait() {
  form.portrait_image_url = ''
  revokeUploadedPortraitPreview()
  portraitPreviewOpen.value = false
}

function revokeObjectUrl(url: string) {
  if (!url.startsWith('blob:')) return
  URL.revokeObjectURL(url)
  portraitPreviewObjectUrls.delete(url)
}

function revokeUploadedPortraitPreview() {
  revokeObjectUrl(uploadedPortraitPreview.value)
  uploadedPortraitPreview.value = ''
}

function revokeAllPortraitPreviewObjectUrls() {
  for (const url of portraitPreviewObjectUrls) URL.revokeObjectURL(url)
  portraitPreviewObjectUrls.clear()
  uploadedPortraitPreview.value = ''
}

async function saveCard(returnToDetail = false) {
  if (saving.value || !card.value) return
  saving.value = true
  try {
    const result = await updateCharacterCard(card.value.id, { ...form })
    applyCard(result)
    toast.success('人物卡全部分栏已保存')
    if (returnToDetail) await router.push(`/character-cards/${result.id}`)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '人物卡保存失败')
  } finally {
    saving.value = false
  }
}

async function syncFromBackup() {
  if (!card.value || !canSyncFromBackup.value || syncing.value) return
  const confirmed = await dialog.confirm({
    title: '从云备份刷新基础信息',
    message: '这会用记录的 TRP3 备份来源覆盖名姓、称号、种族、职业等基础字段。角色大图、摘要和三个富文本分栏不会被覆盖。',
    type: 'warning',
    confirmText: '刷新基础信息',
  })
  if (!confirmed) return

  syncing.value = true
  try {
    const result = await syncCharacterCardFromTRP3(card.value.id)
    applySyncedCard(result)
    toast.success('已从云备份刷新基础信息')
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '备份来源同步失败')
  } finally {
    syncing.value = false
  }
}

async function writeBackToLocalTRP3() {
  if (!card.value || writeBackDisabledReason.value || writingBack.value) return
  const confirmed = await dialog.confirm({
    title: '写回本地 TRP3 基础信息',
    message: `将把当前人物卡的共用基础字段写入本机 TRP3 profile「${card.value.source_profile_id}」。不会写入角色大图、摘要或富文本。请确认魔兽世界已经关闭。`,
    type: 'warning',
    confirmText: '写入本地文件',
  })
  if (!confirmed) return

  writingBack.value = true
  try {
    await invoke('update_profile', {
      wowPath: wowPath.value,
      accountId: card.value.source_account_id,
      profileId: card.value.source_profile_id,
      updates: {
        characteristics: {
          firstName: form.first_name,
          lastName: form.last_name,
          title: form.title,
          fullTitle: form.full_title,
          race: form.race,
          class: form.class,
          eyeColor: form.eye_color,
          eyeColorHex: normalizeCharacterCardHexForTRP3(form.eye_color_hex),
          age: form.age,
          height: form.height,
          weight: form.weight,
          birthplace: form.birthplace,
          residence: form.residence,
          relationshipStatus: form.relationship_status,
          icon: form.icon,
          nameColor: normalizeCharacterCardHexForTRP3(form.name_color),
        },
      },
    })
    toast.success('已写入本地 TRP3 基础信息；请回到同步页重新上传账号备份', 6000)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '本地 TRP3 写入失败')
  } finally {
    writingBack.value = false
  }
}

async function goBack() {
  if (isDirty.value) {
    const confirmed = await dialog.confirm({
      title: '离开人物卡编辑器',
      message: '尚有未保存的更改。离开后，这些更改不会保留。',
      type: 'warning',
      confirmText: '放弃更改',
    })
    if (!confirmed) return
  }
  if (card.value) {
    await router.push(`/character-cards/${card.value.id}`)
    return
  }
  router.back()
}
</script>

<template>
  <main class="editor-page">
    <div v-if="loading" class="editor-state" role="status">
      <i class="ri-loader-4-line spin" aria-hidden="true"></i>
      <span>正在展开人物档案…</span>
    </div>

    <div v-else-if="loadError" class="editor-state editor-state--error" role="alert">
      <i class="ri-file-warning-line" aria-hidden="true"></i>
      <h1>无法编辑这张人物卡</h1>
      <p>{{ loadError }}</p>
      <button type="button" @click="goBack">返回</button>
    </div>

    <template v-else-if="card">
      <header class="editor-header">
        <button type="button" class="editor-header__back" @click="goBack">
          <i class="ri-arrow-left-line" aria-hidden="true"></i>
          返回人物卡
        </button>
        <div class="editor-header__identity">
          <span>Character file · {{ card.id }}</span>
          <h1>{{ displayName }}</h1>
        </div>
        <div class="editor-header__actions">
          <span v-if="isDirty" class="unsaved-mark"><i class="ri-circle-fill" aria-hidden="true"></i>未保存</span>
          <button type="button" class="button button--quiet" :disabled="saving" @click="saveCard(false)">
            {{ saving ? '保存中…' : '保存' }}
          </button>
          <button type="button" class="button button--primary" :disabled="saving" @click="saveCard(true)">
            保存并查看
          </button>
        </div>
      </header>

      <div class="editor-layout">
        <aside class="portrait-editor" aria-label="角色肖像与身份预览">
          <div class="portrait-editor__rail" aria-hidden="true"></div>
          <div class="portrait-editor__frame">
            <img
              v-if="uploadedPortraitPreview"
              class="portrait-editor__image"
              :src="uploadedPortraitPreview"
              :alt="`${displayName}的角色肖像预览`"
            />
            <CharacterCardPortrait
              v-else-if="form.portrait_image_url"
              class="portrait-editor__image"
              :card="card"
              :alt="`${displayName}的角色肖像预览`"
              :width="900"
              :quality="90"
            />
            <div v-else class="portrait-editor__empty">
              <i class="ri-user-star-line" aria-hidden="true"></i>
              <strong>角色大图</strong>
              <span>建议使用 3:4 竖幅构图</span>
            </div>
            <div class="portrait-editor__tools">
              <button type="button" :disabled="portraitUploading" @click="triggerPortraitUpload">
                <i :class="portraitUploading ? 'ri-loader-4-line spin' : 'ri-camera-line'" aria-hidden="true"></i>
                {{ hasPortrait ? '替换' : '上传' }}
              </button>
              <button v-if="hasPortrait" type="button" @click="portraitPreviewOpen = true">
                <i class="ri-focus-mode" aria-hidden="true"></i>预览
              </button>
              <button v-if="hasPortrait" type="button" class="danger" @click="removePortrait">
                <i class="ri-delete-bin-line" aria-hidden="true"></i>移除
              </button>
            </div>
          </div>
          <input ref="portraitInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="handlePortraitFile" />

          <div class="portrait-editor__plaque">
            <strong>{{ displayName }}</strong>
            <span>{{ form.title || form.full_title || '称号待补充' }}</span>
            <small>{{ [form.race, form.class].filter(Boolean).join(' · ') || '身份待补充' }}</small>
          </div>

          <div class="source-info">
            <span><i class="ri-link-m" aria-hidden="true"></i>来源</span>
            <strong v-if="card.source_profile_id">{{ card.source_account_id || '云备份' }} · {{ card.source_profile_id }}</strong>
            <strong v-else>RPBox 独立人物卡</strong>
            <button v-if="canSyncFromBackup" type="button" :disabled="syncing" @click="syncFromBackup">
              <i :class="syncing ? 'ri-loader-4-line spin' : 'ri-refresh-line'" aria-hidden="true"></i>
              {{ syncing ? '正在刷新基础信息…' : '从备份刷新基础信息' }}
            </button>
          </div>

          <div class="local-writeback" :class="{ disabled: Boolean(writeBackDisabledReason) }">
            <span><i class="ri-hard-drive-3-line" aria-hidden="true"></i>桌面端本地互通</span>
            <p>{{ writeBackDisabledReason || '只写入共用基础字段；不会把 RPBox 专属内容写入 Lua。' }}</p>
            <button type="button" :disabled="Boolean(writeBackDisabledReason) || writingBack" @click="writeBackToLocalTRP3">
              {{ writingBack ? '正在写入本地文件…' : '写回本地 TRP3 基础信息' }}
            </button>
          </div>
        </aside>

        <section class="editor-ledger">
          <nav class="editor-tabs" role="tablist" aria-label="人物卡编辑分栏">
            <button
              v-for="tab in tabs"
              :id="`character-tab-${tab.id}`"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              :aria-controls="`character-panel-${tab.id}`"
              :tabindex="activeTab === tab.id ? 0 : -1"
              :class="{ active: activeTab === tab.id }"
              @click="selectTab(tab.id)"
              @keydown="handleTabKeydown($event, tab.id)"
            >
              <i :class="tab.icon" aria-hidden="true"></i>
              {{ tab.label }}
            </button>
          </nav>

          <section
            v-show="activeTab === 'basic'"
            id="character-panel-basic"
            class="editor-panel"
            role="tabpanel"
            aria-labelledby="character-tab-basic"
          >
            <header class="panel-heading">
              <div><span>Identity registry</span><h2>基础身份与展示方式</h2></div>
              <p>此处的共用字段可以由你主动从 TRP3 导入或写回；摘要和展示状态只属于 RPBox。</p>
            </header>

            <div class="form-section">
              <h3>姓名与称号</h3>
              <div class="form-grid form-grid--three">
                <label><span>名</span><input v-model="form.first_name" maxlength="128" autocomplete="off" /></label>
                <label><span>姓</span><input v-model="form.last_name" maxlength="128" autocomplete="off" /></label>
                <label><span>展示名</span><input v-model="form.display_name" maxlength="256" placeholder="留空时自动组合名与姓" /></label>
                <label><span>称号</span><input v-model="form.title" maxlength="128" /></label>
                <label class="form-span-two"><span>完整头衔</span><input v-model="form.full_title" maxlength="256" /></label>
              </div>
            </div>

            <div class="form-section">
              <h3>身份特征</h3>
              <div class="form-grid form-grid--three">
                <label><span>种族</span><input v-model="form.race" maxlength="64" /></label>
                <label><span>职业</span><input v-model="form.class" maxlength="64" /></label>
                <label><span>年龄</span><input v-model="form.age" maxlength="64" /></label>
                <label><span>身高</span><input v-model="form.height" maxlength="64" /></label>
                <label><span>体重</span><input v-model="form.weight" maxlength="64" /></label>
                <label><span>关系状态</span><input v-model="form.relationship_status" maxlength="64" placeholder="保留 TRP3 原值" /></label>
                <label><span>出生地</span><input v-model="form.birthplace" maxlength="256" /></label>
                <label><span>居所</span><input v-model="form.residence" maxlength="256" /></label>
                <label><span>TRP3 图标</span><input v-model="form.icon" maxlength="128" /></label>
              </div>
            </div>

            <div class="form-section">
              <h3>颜色记录</h3>
              <div class="form-grid form-grid--three">
                <label><span>眼睛颜色</span><input v-model="form.eye_color" maxlength="64" /></label>
                <label><span>眼睛颜色值</span><input v-model="form.eye_color_hex" maxlength="16" placeholder="#6F8CA3" /></label>
                <label><span>名字颜色</span><input v-model="form.name_color" maxlength="16" placeholder="#E8DCCF" /></label>
              </div>
            </div>

            <div class="form-section form-section--rpbox">
              <h3>RPBox 展示</h3>
              <label class="summary-field">
                <span>人物摘要</span>
                <textarea v-model="form.summary" rows="4" maxlength="1000" placeholder="用几句话介绍这位角色；它会显示在展示墙和帖子预览中。"></textarea>
                <small>{{ form.summary.length }} / 1000</small>
              </label>
              <div class="visibility-grid">
                <label>
                  <span>制作状态</span>
                  <select v-model="form.status">
                    <option value="draft">草稿</option>
                    <option value="published">已发布</option>
                  </select>
                  <small>草稿不会出现在访客展示墙或帖子选择器中。</small>
                </label>
                <label>
                  <span>可见范围</span>
                  <select v-model="form.visibility">
                    <option value="private">仅自己可见</option>
                    <option value="public">公开可见</option>
                  </select>
                  <small>只有“已发布 + 公开可见”才会向其他用户展示。</small>
                </label>
              </div>
            </div>
          </section>

          <section
            v-show="activeTab === 'background'"
            id="character-panel-background"
            class="editor-panel editor-panel--rich"
            role="tabpanel"
            aria-labelledby="character-tab-background"
          >
            <header class="panel-heading">
              <div><span>Chronicle</span><h2>背景故事</h2></div>
              <p>记录角色的经历、转折与仍未揭开的线索。</p>
            </header>
            <TiptapEditor ref="backgroundEditor" v-model="form.background_story" placeholder="从角色最重要的一段过去开始…">
              <template #toolbar>
                <button type="button" class="internal-link-button" title="插入站内内容" @mousedown.prevent @click="openQuickJump('background')">
                  <i class="ri-links-line"></i><span>站内链接</span>
                </button>
              </template>
            </TiptapEditor>
          </section>

          <section
            v-show="activeTab === 'impression'"
            id="character-panel-impression"
            class="editor-panel editor-panel--rich"
            role="tabpanel"
            aria-labelledby="character-tab-impression"
          >
            <header class="panel-heading">
              <div><span>At first sight</span><h2>第一印象</h2></div>
              <p>写下陌生人最先注意到的气质、外貌和习惯。</p>
            </header>
            <TiptapEditor ref="impressionEditor" v-model="form.first_impression" placeholder="第一次见到这位角色时，人们会注意到…">
              <template #toolbar>
                <button type="button" class="internal-link-button" title="插入站内内容" @mousedown.prevent @click="openQuickJump('impression')">
                  <i class="ri-links-line"></i><span>站内链接</span>
                </button>
              </template>
            </TiptapEditor>
          </section>

          <section
            v-show="activeTab === 'other'"
            id="character-panel-other"
            class="editor-panel editor-panel--rich"
            role="tabpanel"
            aria-labelledby="character-tab-other"
          >
            <header class="panel-heading">
              <div><span>Filed notes</span><h2>其他资料</h2></div>
              <p>收纳关系、传闻、创作约定或任何不适合放进前述分栏的内容。</p>
            </header>
            <TiptapEditor ref="otherEditor" v-model="form.other_content" placeholder="补充关系、传闻或创作约定…">
              <template #toolbar>
                <button type="button" class="internal-link-button" title="插入站内内容" @mousedown.prevent @click="openQuickJump('other')">
                  <i class="ri-links-line"></i><span>站内链接</span>
                </button>
              </template>
            </TiptapEditor>
          </section>
        </section>
      </div>

      <footer class="save-dock">
        <div>
          <strong>{{ isDirty ? '有尚未保存的更改' : '所有分栏均已保存' }}</strong>
          <span>保存操作会整体提交基础信息和三个富文本分栏。</span>
        </div>
        <button type="button" class="button button--primary" :disabled="saving || !isDirty" @click="saveCard(false)">
          <i class="ri-save-3-line" aria-hidden="true"></i>{{ saving ? '保存中…' : '保存整张人物卡' }}
        </button>
      </footer>

      <ImageCropperDialog
        v-model="cropperOpen"
        :file="cropperFile"
        :aspect-ratio="3 / 4"
        :output-width="1200"
        :output-height="1600"
        :max-size-k-b="2048"
        title="调整角色大图"
        @cropped="handlePortraitCropped"
        @error="handleCropperError"
      />

      <RModal v-model="portraitPreviewOpen" :title="`${displayName} · 角色大图`" width="680px">
        <div class="portrait-lightbox">
          <img v-if="uploadedPortraitPreview" :src="uploadedPortraitPreview" :alt="`${displayName}的角色大图`" />
          <CharacterCardPortrait
            v-else-if="form.portrait_image_url"
            :card="card"
            :alt="`${displayName}的角色大图`"
            :width="1200"
            :quality="92"
          />
        </div>
      </RModal>

      <PostQuickJump v-model="quickJumpOpen" :on-insert="insertQuickJump" />
    </template>
  </main>
</template>

<style scoped>
.editor-page {
  --ink: #2C1810;
  --walnut: #4B3621;
  --copper: #B87333;
  --rust: #804030;
  --paper: #FDFBF9;
  --line: #E3D4C5;
  --muted: #8C7B70;
  width: min(1320px, calc(100% - 36px));
  margin: 0 auto;
  padding: 20px 0 86px;
  color: var(--ink);
}

.editor-state {
  display: grid;
  min-height: 60vh;
  place-content: center;
  gap: 12px;
  color: var(--muted);
  text-align: center;
}
.editor-state > i { color: var(--copper); font-size: 36px; }
.editor-state h1 { margin: 0; color: var(--ink); font-family: Georgia, 'Noto Serif SC', serif; }
.editor-state p { margin: 0; }
.editor-state button { justify-self: center; padding: 9px 18px; border: 1px solid var(--rust); border-radius: 7px; background: var(--rust); color: #fff; cursor: pointer; }

.editor-header {
  display: grid;
  grid-template-columns: minmax(170px, 1fr) auto minmax(270px, 1fr);
  align-items: center;
  gap: 18px;
  margin-bottom: 16px;
  padding: 12px 16px;
  border: 1px solid var(--line);
  border-radius: 12px;
  background: rgba(253, 251, 249, 0.92);
  box-shadow: 0 5px 16px rgba(75, 54, 33, 0.06);
  backdrop-filter: blur(12px);
}

.editor-header__back {
  display: inline-flex;
  justify-self: start;
  align-items: center;
  gap: 7px;
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.editor-header__identity { min-width: 0; text-align: center; }
.editor-header__identity span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: 0.16em; text-transform: uppercase; }
.editor-header__identity h1 { overflow: hidden; margin: 3px 0 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 21px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.editor-header__actions { display: flex; justify-content: flex-end; align-items: center; gap: 8px; }

.unsaved-mark { display: inline-flex; align-items: center; gap: 5px; color: #9A5A2D; font-size: 10px; }
.unsaved-mark i { font-size: 6px; }

.button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 15px;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: #FFF;
  color: var(--walnut);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}
.button--primary { border-color: var(--rust); background: var(--rust); color: #FFF9F2; }
.button:disabled { cursor: not-allowed; opacity: 0.5; }

.editor-layout {
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.portrait-editor,
.editor-ledger {
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--paper);
  box-shadow: 0 9px 28px rgba(75, 54, 33, 0.07);
}

.portrait-editor {
  position: sticky;
  top: 14px;
  overflow: hidden;
  padding: 18px;
}

.portrait-editor__rail {
  position: absolute;
  top: 0;
  left: 18px;
  right: 18px;
  height: 5px;
  border-radius: 0 0 5px 5px;
  background: linear-gradient(90deg, var(--rust), #C58D59, var(--rust));
}

.portrait-editor__frame {
  position: relative;
  overflow: hidden;
  aspect-ratio: 3 / 4;
  border: 6px solid #352219;
  border-radius: 6px 6px 0 0;
  background: #271A14;
  box-shadow: 0 9px 20px rgba(44, 24, 16, 0.22);
}
.portrait-editor__image { width: 100%; height: 100%; display: block; object-fit: cover; }
.portrait-editor__empty { display: grid; width: 100%; height: 100%; place-content: center; gap: 7px; background: radial-gradient(circle at center, rgba(184, 115, 51, 0.18), transparent 48%), #2C1D16; color: #D4AF8A; text-align: center; }
.portrait-editor__empty i { font-size: 50px; }
.portrait-editor__empty strong { font-family: Georgia, 'Noto Serif SC', serif; }
.portrait-editor__empty span { color: #BCA997; font-size: 10px; }

.portrait-editor__tools {
  position: absolute;
  right: 8px;
  bottom: 8px;
  left: 8px;
  display: flex;
  justify-content: center;
  gap: 5px;
  padding: 7px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 7px;
  background: rgba(31, 19, 13, 0.78);
  backdrop-filter: blur(8px);
}
.portrait-editor__tools button { display: inline-flex; align-items: center; gap: 4px; padding: 6px 7px; border: 0; border-radius: 5px; background: rgba(255, 255, 255, 0.09); color: #F5E6D7; cursor: pointer; font-size: 10px; }
.portrait-editor__tools button.danger { color: #F0B4A7; }

.portrait-editor__plaque {
  display: grid;
  gap: 3px;
  padding: 13px 12px;
  border-top: 1px solid #B87333;
  border-radius: 0 0 6px 6px;
  background: linear-gradient(90deg, #281912, #40271B 50%, #281912);
  color: #F1D7BB;
  text-align: center;
}
.portrait-editor__plaque strong { font-family: Georgia, 'Noto Serif SC', serif; font-size: 19px; font-weight: 600; }
.portrait-editor__plaque span { color: #D4B697; font-size: 11px; }
.portrait-editor__plaque small { color: #AFA093; font-size: 9px; }

.source-info,
.local-writeback { display: grid; gap: 6px; margin-top: 14px; padding: 12px; border: 1px solid var(--line); border-radius: 8px; background: #FBF6F0; }
.source-info > span,
.local-writeback > span { display: inline-flex; align-items: center; gap: 5px; color: var(--copper); font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: 0.08em; }
.source-info strong { overflow: hidden; color: var(--walnut); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.source-info button,
.local-writeback button { padding: 7px 9px; border: 1px solid #C99769; border-radius: 6px; background: #FFF; color: var(--rust); cursor: pointer; font-size: 10px; font-weight: 700; }
.local-writeback p { margin: 0; color: var(--muted); font-size: 10px; line-height: 1.55; }
.local-writeback.disabled { background: #F5F1ED; }
.local-writeback button:disabled { cursor: not-allowed; opacity: 0.45; }

.editor-ledger { min-width: 0; overflow: hidden; }

.editor-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  padding: 0 18px;
  border-bottom: 1px solid var(--line);
  background: #F9F4EE;
}
.editor-tabs button { position: relative; display: inline-flex; min-height: 58px; align-items: center; justify-content: center; gap: 7px; border: 0; background: transparent; color: var(--muted); cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.editor-tabs button::after { position: absolute; right: 18%; bottom: -1px; left: 18%; height: 3px; border-radius: 3px 3px 0 0; background: var(--copper); content: ''; opacity: 0; transform: scaleX(0.4); transition: opacity 150ms ease, transform 150ms ease; }
.editor-tabs button.active { color: var(--rust); }
.editor-tabs button.active::after { opacity: 1; transform: scaleX(1); }

.editor-panel { min-height: 650px; padding: 26px; }
.editor-panel--rich :deep(.rich-editor) { min-height: 500px; }
.editor-panel--rich :deep(.editor-content) { min-height: 390px; }

.panel-heading { display: grid; grid-template-columns: minmax(0, 1fr) minmax(220px, 0.8fr); align-items: end; gap: 24px; margin-bottom: 26px; padding-bottom: 16px; border-bottom: 1px solid var(--line); }
.panel-heading span { color: var(--copper); font: 800 9px/1.2 ui-monospace, Consolas, monospace; letter-spacing: 0.16em; text-transform: uppercase; }
.panel-heading h2 { margin: 4px 0 0; font-family: Georgia, 'Noto Serif SC', serif; font-size: 25px; font-weight: 600; }
.panel-heading p { margin: 0; color: var(--muted); font-size: 11px; line-height: 1.65; }

.form-section { margin-bottom: 26px; }
.form-section h3 { margin: 0 0 12px; color: var(--walnut); font-size: 12px; letter-spacing: 0.06em; }
.form-section--rpbox { margin-bottom: 0; padding: 18px; border: 1px solid #DEC7AF; border-radius: 10px; background: #FCF7F2; }
.form-grid { display: grid; gap: 12px; }
.form-grid--three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.form-span-two { grid-column: span 2; }
.form-grid label,
.summary-field,
.visibility-grid label { display: grid; min-width: 0; gap: 6px; color: var(--muted); font-size: 10px; font-weight: 700; }
.form-grid input,
.summary-field textarea,
.visibility-grid select { width: 100%; box-sizing: border-box; padding: 10px 11px; border: 1px solid var(--line); border-radius: 7px; outline: none; background: #FFF; color: var(--ink); font: inherit; font-size: 12px; }
.form-grid input:focus,
.summary-field textarea:focus,
.visibility-grid select:focus { border-color: var(--copper); box-shadow: 0 0 0 3px rgba(184, 115, 51, 0.1); }
.summary-field { position: relative; }
.summary-field textarea { resize: vertical; line-height: 1.65; }
.summary-field small { position: absolute; right: 9px; bottom: 8px; color: #AA9481; font-weight: 400; }
.visibility-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 14px; }
.visibility-grid small { color: var(--muted); font-weight: 400; line-height: 1.45; }

.internal-link-button { display: inline-flex; align-items: center; gap: 5px; width: auto !important; padding: 0 8px !important; color: var(--rust) !important; }

.save-dock {
  position: fixed;
  z-index: 40;
  right: 24px;
  bottom: 18px;
  display: flex;
  max-width: min(600px, calc(100vw - 48px));
  align-items: center;
  gap: 22px;
  padding: 11px 12px 11px 16px;
  border: 1px solid #D2B99F;
  border-radius: 12px;
  background: rgba(253, 251, 249, 0.94);
  box-shadow: 0 14px 35px rgba(44, 24, 16, 0.2);
  backdrop-filter: blur(14px);
}
.save-dock > div { display: grid; gap: 2px; }
.save-dock strong { color: var(--walnut); font-size: 11px; }
.save-dock span { color: var(--muted); font-size: 9px; }

.portrait-lightbox { display: grid; max-height: 72vh; place-items: center; overflow: hidden; border-radius: 8px; background: #21150F; }
.portrait-lightbox img { max-width: 100%; max-height: 72vh; object-fit: contain; }

.editor-header__back:focus-visible,
.button:focus-visible,
.editor-tabs button:focus-visible,
.portrait-editor button:focus-visible { outline: 3px solid rgba(184, 115, 51, 0.3); outline-offset: 2px; }

.spin { animation: editor-spin 900ms linear infinite; }
@keyframes editor-spin { to { transform: rotate(360deg); } }

@media (max-width: 980px) {
  .editor-header { grid-template-columns: 1fr auto; }
  .editor-header__identity { grid-row: 2; grid-column: 1 / -1; text-align: left; }
  .editor-layout { grid-template-columns: 240px minmax(0, 1fr); }
  .form-grid--three { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 760px) {
  .editor-page { width: min(100% - 20px, 1320px); padding-top: 10px; }
  .editor-header { grid-template-columns: 1fr; }
  .editor-header__actions { justify-content: flex-start; flex-wrap: wrap; }
  .editor-header__identity { grid-row: auto; grid-column: auto; }
  .editor-layout { grid-template-columns: 1fr; }
  .portrait-editor { position: relative; top: auto; }
  .portrait-editor__frame { max-width: 360px; margin: 0 auto; }
  .portrait-editor__plaque { max-width: 336px; margin: 0 auto; }
  .editor-tabs { padding: 0 5px; overflow-x: auto; }
  .editor-tabs button { min-width: 105px; }
  .editor-panel { min-height: 540px; padding: 20px 14px; }
  .panel-heading { grid-template-columns: 1fr; gap: 8px; }
  .form-grid--three,
  .visibility-grid { grid-template-columns: 1fr; }
  .form-span-two { grid-column: auto; }
  .save-dock { right: 10px; bottom: 10px; left: 10px; max-width: none; justify-content: space-between; }
}

@media (max-width: 480px) {
  .save-dock > div { display: none; }
  .save-dock .button { width: 100%; }
}

@media (prefers-reduced-motion: reduce) {
  .editor-tabs button::after { transition: none; }
  .spin { animation: none; }
}
</style>
