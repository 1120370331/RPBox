<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  createCharacterCard,
  getCharacterCardSources,
  type CharacterCardSource,
  type CharacterCardSourceType,
} from '@/api/characterCard'
import { useToastStore } from '@/stores/toast'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const toast = useToastStore()
const userStore = useUserStore()

const sources = ref<CharacterCardSource[]>([])
const selectedSourceKey = ref('')
const loading = ref(true)
const loadError = ref('')
const creatingType = ref<CharacterCardSourceType | null>(null)

const sourceGroups = computed(() => {
  const groups = new Map<string, CharacterCardSource[]>()
  for (const source of sources.value) {
    const key = source.account_id || '未知账号'
    const group = groups.get(key) || []
    group.push(source)
    groups.set(key, group)
  }
  return Array.from(groups, ([accountId, profiles]) => ({ accountId, profiles }))
})

const selectedSource = computed(() => (
  sources.value.find((source) => sourceKey(source) === selectedSourceKey.value) || null
))

onMounted(() => void loadSources())

function sourceKey(source: CharacterCardSource) {
  return `${source.backup_id}:${source.profile_id}`
}

function sourceName(source: CharacterCardSource) {
  if (source.display_name?.trim()) return source.display_name
  const name = [source.first_name, source.last_name].map((part) => part?.trim()).filter(Boolean).join(' ')
  return name || source.profile_name || '未命名人物'
}

function sourceMeta(source: CharacterCardSource) {
  return [source.race, source.class, source.title].map((part) => part?.trim()).filter(Boolean).join(' · ') || '基础身份信息待补充'
}

function formatDate(value?: string) {
  if (!value) return '时间未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '时间未知'
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

async function loadSources() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getCharacterCardSources()
    sources.value = response.sources || []
  } catch (error: unknown) {
    sources.value = []
    loadError.value = error instanceof Error ? error.message : '无法读取云备份来源'
  } finally {
    loading.value = false
  }
}

async function createFromBlank() {
  await createDraft('blank')
}

async function createFromBackup() {
  if (!selectedSource.value) {
    toast.warning('请先选择一份备份人物资料')
    return
  }
  await createDraft('backup')
}

async function createDraft(sourceType: CharacterCardSourceType) {
  if (creatingType.value) return
  creatingType.value = sourceType
  try {
    const source = selectedSource.value
    const card = await createCharacterCard(sourceType === 'backup' && source
      ? {
          source_type: 'backup',
          source_backup_id: source.backup_id,
          source_profile_id: source.profile_id,
        }
      : { source_type: 'blank' })
    toast.success(sourceType === 'backup' ? '已从备份创建人物卡草稿' : '空白人物卡草稿已建立')
    await router.replace(`/character-cards/${card.id}/edit`)
  } catch (error: unknown) {
    toast.error(error instanceof Error ? error.message : '人物卡创建失败')
  } finally {
    creatingType.value = null
  }
}

function goBack() {
  if (userStore.user?.id) {
    void router.push(`/user/${userStore.user.id}`)
    return
  }
  router.back()
}
</script>

<template>
  <main class="source-page">
    <header class="source-page__header">
      <button type="button" class="source-page__back" @click="goBack">
        <i class="ri-arrow-left-line" aria-hidden="true"></i>
        返回个人中心
      </button>
      <div class="source-page__title-row">
        <div>
          <span class="source-page__kicker">New character record</span>
          <h1>选择人物卡的起点</h1>
          <p>导入只会复制可以共用的基础身份字段，不会修改原始备份，也不会建立自动覆盖关系。</p>
        </div>
        <div class="source-page__seal" aria-hidden="true"><i class="ri-quill-pen-line"></i></div>
      </div>
    </header>

    <aside class="sync-note">
      <i class="ri-git-merge-line" aria-hidden="true"></i>
      <div>
        <strong>TRP3 与 RPBox 只显式交换基础信息</strong>
        <p>背景故事、第一印象、其他富文本和角色大图始终属于 RPBox 人物卡，不会写入或被备份导入覆盖。</p>
      </div>
    </aside>

    <div class="source-grid">
      <section class="source-panel source-panel--backup" aria-labelledby="backup-source-title">
        <header class="source-panel__header">
          <span class="source-panel__number">A</span>
          <div>
            <h2 id="backup-source-title">从已备份的人物卡开始</h2>
            <p>选择当前账号拥有的云备份 profile，复制基础身份信息。</p>
          </div>
        </header>

        <div v-if="loading" class="source-state" role="status">
          <i class="ri-loader-4-line spin" aria-hidden="true"></i>
          正在整理备份档案…
        </div>

        <div v-else-if="loadError" class="source-state source-state--error" role="alert">
          <i class="ri-file-warning-line" aria-hidden="true"></i>
          <div><strong>无法读取备份来源</strong><span>{{ loadError }}</span></div>
          <button type="button" @click="loadSources">重新加载</button>
        </div>

        <div v-else-if="sourceGroups.length === 0" class="source-state source-state--empty">
          <i class="ri-archive-drawer-line" aria-hidden="true"></i>
          <div>
            <strong>还没有可导入的云备份</strong>
            <span>可以先从零创建；之后仍可独立编辑 RPBox 人物卡。</span>
          </div>
        </div>

        <div v-else class="account-groups">
          <section v-for="group in sourceGroups" :key="group.accountId" class="account-group">
            <header>
              <span><i class="ri-folder-user-line" aria-hidden="true"></i>{{ group.accountId }}</span>
              <small>{{ group.profiles.length }} 份人物资料</small>
            </header>
            <div class="profile-sources" role="radiogroup" :aria-label="`${group.accountId} 的备份人物`">
              <button
                v-for="source in group.profiles"
                :key="sourceKey(source)"
                type="button"
                class="profile-source"
                :class="{ selected: selectedSourceKey === sourceKey(source) }"
                role="radio"
                :aria-checked="selectedSourceKey === sourceKey(source)"
                @click="selectedSourceKey = sourceKey(source)"
              >
                <span class="profile-source__icon">
                  <i :class="source.icon ? 'ri-user-star-line' : 'ri-user-line'" aria-hidden="true"></i>
                </span>
                <span class="profile-source__copy">
                  <strong>{{ sourceName(source) }}</strong>
                  <span>{{ sourceMeta(source) }}</span>
                  <small>备份更新：{{ formatDate(source.backup_updated_at) }}</small>
                </span>
                <span class="profile-source__check" aria-hidden="true">
                  <i :class="selectedSourceKey === sourceKey(source) ? 'ri-check-line' : 'ri-arrow-right-s-line'"></i>
                </span>
              </button>
            </div>
          </section>
        </div>

        <footer class="source-panel__footer">
          <button
            type="button"
            class="source-action source-action--primary"
            :disabled="!selectedSource || Boolean(creatingType)"
            @click="createFromBackup"
          >
            <i :class="creatingType === 'backup' ? 'ri-loader-4-line spin' : 'ri-file-copy-2-line'" aria-hidden="true"></i>
            {{ creatingType === 'backup' ? '正在建立草稿…' : '从所选备份创建' }}
          </button>
        </footer>
      </section>

      <section class="source-panel source-panel--blank" aria-labelledby="blank-source-title">
        <header class="source-panel__header">
          <span class="source-panel__number">B</span>
          <div>
            <h2 id="blank-source-title">从零创建 RPBox 人物卡</h2>
            <p>不依赖 TRP3，直接开启一份独立的角色档案。</p>
          </div>
        </header>

        <div class="blank-ledger" aria-hidden="true">
          <span class="blank-ledger__compass"><i class="ri-compass-3-line"></i></span>
          <span class="blank-ledger__line"></span>
          <span class="blank-ledger__line short"></span>
          <span class="blank-ledger__line"></span>
          <span class="blank-ledger__stamp">RPB · NEW</span>
        </div>

        <div class="blank-copy">
          <strong>一张干净的档案页</strong>
          <p>适合原创角色、未使用 TRP3 的设定，或希望与游戏内资料完全分开维护的人物。</p>
          <ul>
            <li><i class="ri-check-line" aria-hidden="true"></i>独立的大图与展示摘要</li>
            <li><i class="ri-check-line" aria-hidden="true"></i>四个分栏整体保存</li>
            <li><i class="ri-check-line" aria-hidden="true"></i>默认保存为私密草稿</li>
          </ul>
        </div>

        <footer class="source-panel__footer">
          <button
            type="button"
            class="source-action source-action--blank"
            :disabled="Boolean(creatingType)"
            @click="createFromBlank"
          >
            <i :class="creatingType === 'blank' ? 'ri-loader-4-line spin' : 'ri-add-line'" aria-hidden="true"></i>
            {{ creatingType === 'blank' ? '正在建立草稿…' : '从零创建' }}
          </button>
        </footer>
      </section>
    </div>
  </main>
</template>

<style scoped>
.source-page {
  --ink: #2C1810;
  --walnut: #4B3621;
  --copper: #B87333;
  --rust: #804030;
  --muted: #8C7B70;
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
  padding: 30px 0 54px;
  color: var(--ink);
}

.source-page__header { margin-bottom: 18px; }

.source-page__back {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 7px 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
}

.source-page__back:hover { color: var(--rust); }

.source-page__title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 30px;
  margin-top: 22px;
}

.source-page__kicker,
.source-panel__number,
.blank-ledger__stamp {
  color: var(--copper);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.source-page h1 {
  margin: 5px 0 8px;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: clamp(30px, 4vw, 46px);
  font-weight: 600;
  letter-spacing: 0.01em;
}

.source-page__title-row p {
  max-width: 720px;
  margin: 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.7;
}

.source-page__seal {
  display: grid;
  width: 76px;
  height: 76px;
  flex: 0 0 76px;
  place-items: center;
  border: 1px solid #CDA57C;
  border-radius: 50%;
  color: var(--rust);
  font-size: 30px;
  outline: 1px dashed #DCC7B1;
  outline-offset: 7px;
}

.sync-note {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 13px;
  margin-bottom: 20px;
  padding: 14px 16px;
  border: 1px solid #E0CDB9;
  border-radius: 10px;
  background: #FBF6F0;
}

.sync-note > i { color: var(--copper); font-size: 21px; }
.sync-note strong { color: var(--walnut); font-size: 13px; }
.sync-note p { margin: 3px 0 0; color: var(--muted); font-size: 12px; line-height: 1.6; }

.source-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(300px, 0.8fr);
  gap: 18px;
  align-items: stretch;
}

.source-panel {
  display: flex;
  min-width: 0;
  min-height: 560px;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #E0D0C0;
  border-radius: 14px;
  background: #FDFBF9;
  box-shadow: 0 10px 28px rgba(75, 54, 33, 0.07);
}

.source-panel--backup { border-top: 4px solid var(--copper); }
.source-panel--blank { border-top: 4px solid var(--walnut); }

.source-panel__header {
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr);
  gap: 11px;
  padding: 22px 24px 18px;
  border-bottom: 1px solid #EEE3D8;
}

.source-panel__number {
  display: grid;
  width: 26px;
  height: 26px;
  place-items: center;
  border: 1px solid currentColor;
  border-radius: 50%;
}

.source-panel h2 {
  margin: 0 0 5px;
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 20px;
  font-weight: 600;
}

.source-panel__header p {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.55;
}

.source-state {
  display: flex;
  min-height: 280px;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 30px;
  color: var(--muted);
  text-align: center;
}

.source-state--empty,
.source-state--error { flex-direction: column; }
.source-state > i { color: var(--copper); font-size: 34px; }
.source-state div { display: grid; gap: 4px; }
.source-state strong { color: var(--walnut); }
.source-state span { font-size: 12px; }
.source-state button {
  margin-top: 5px;
  padding: 7px 12px;
  border: 1px solid var(--copper);
  border-radius: 7px;
  background: transparent;
  color: var(--rust);
  cursor: pointer;
}

.account-groups {
  display: flex;
  max-height: 420px;
  flex: 1;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
  padding: 18px 20px;
}

.account-group > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--walnut);
  font-size: 12px;
  font-weight: 700;
}

.account-group > header span { display: inline-flex; align-items: center; gap: 6px; }
.account-group > header i { color: var(--copper); }
.account-group > header small { color: var(--muted); font-weight: 400; }

.profile-sources { display: grid; gap: 7px; }

.profile-source {
  display: grid;
  width: 100%;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 10px 11px;
  border: 1px solid #E9DDD1;
  border-radius: 9px;
  background: #FFF;
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms ease;
}

.profile-source:hover,
.profile-source.selected {
  border-color: #BF7E45;
  background: #FFF9F3;
  transform: translateX(2px);
}

.profile-source:focus-visible,
.source-action:focus-visible,
.source-page__back:focus-visible,
.source-state button:focus-visible {
  outline: 3px solid rgba(184, 115, 51, 0.3);
  outline-offset: 2px;
}

.profile-source__icon {
  display: grid;
  width: 42px;
  height: 48px;
  place-items: center;
  border-radius: 5px;
  background: linear-gradient(145deg, #4B3621, #2C1810);
  color: #E9C7A4;
  font-size: 19px;
}

.profile-source__copy { display: grid; min-width: 0; gap: 2px; }
.profile-source__copy strong { overflow: hidden; color: var(--ink); font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.profile-source__copy > span { overflow: hidden; color: var(--muted); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.profile-source__copy small { color: #AD927A; font-size: 9px; }
.profile-source__check { color: var(--copper); font-size: 17px; }

.source-panel__footer {
  margin-top: auto;
  padding: 16px 20px;
  border-top: 1px solid #EEE3D8;
  background: rgba(250, 245, 239, 0.74);
}

.source-action {
  display: inline-flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--rust);
  border-radius: 8px;
  background: var(--rust);
  color: #FFF8F1;
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  font-weight: 700;
}

.source-action--blank { background: var(--walnut); border-color: var(--walnut); }
.source-action:disabled { cursor: not-allowed; opacity: 0.48; }

.blank-ledger {
  position: relative;
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin: 24px 24px 16px;
  border: 1px solid #DCC9B6;
  background:
    linear-gradient(90deg, transparent 28px, rgba(184, 115, 51, 0.15) 29px, transparent 30px),
    repeating-linear-gradient(#FFFDFB, #FFFDFB 28px, #EDE2D8 29px);
  box-shadow: inset 0 0 30px rgba(75, 54, 33, 0.035);
}

.blank-ledger__compass {
  display: grid;
  width: 70px;
  height: 70px;
  place-items: center;
  border: 1px solid #CEAE8F;
  border-radius: 50%;
  color: #B87333;
  font-size: 36px;
}

.blank-ledger__line { width: 52%; height: 1px; background: #DCCBBC; }
.blank-ledger__line.short { width: 34%; }
.blank-ledger__stamp { position: absolute; right: 14px; bottom: 12px; transform: rotate(-4deg); }

.blank-copy { padding: 0 26px 22px; }
.blank-copy > strong { font-family: Georgia, 'Noto Serif SC', serif; font-size: 18px; }
.blank-copy p { margin: 8px 0 14px; color: var(--muted); font-size: 12px; line-height: 1.65; }
.blank-copy ul { display: grid; gap: 7px; margin: 0; padding: 0; list-style: none; color: var(--walnut); font-size: 12px; }
.blank-copy li { display: flex; align-items: center; gap: 7px; }
.blank-copy li i { color: var(--copper); }

.spin { animation: source-spin 900ms linear infinite; }
@keyframes source-spin { to { transform: rotate(360deg); } }

@media (max-width: 860px) {
  .source-grid { grid-template-columns: 1fr; }
  .source-panel { min-height: auto; }
}

@media (max-width: 560px) {
  .source-page { width: min(100% - 24px, 1180px); padding-top: 18px; }
  .source-page__seal { display: none; }
  .source-panel__header { padding-right: 16px; padding-left: 16px; }
  .account-groups { padding-right: 12px; padding-left: 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .profile-source { transition: none; }
  .spin { animation: none; }
}
</style>
