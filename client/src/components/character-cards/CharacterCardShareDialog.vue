<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { toBlob } from 'html-to-image'
import { useI18n } from 'vue-i18n'
import { getCharacterCardShare, type CharacterCard, type CharacterCardShare } from '@/api/characterCard'
import { useToastStore } from '@/stores/toast'
import { getCharacterCardDisplayName } from '@/utils/characterCardDraft'
import { buildPublicSitePathUrl } from '@/utils/desktopDeepLink'
import { saveBlobAsFile } from '@/utils/saveBlob'
import CharacterCardSharePoster from './CharacterCardSharePoster.vue'

type ShareMode = 'link' | 'poster'
type PosterExpose = { getElement: () => HTMLElement | null }
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api/v1'

const props = withDefaults(defineProps<{
  modelValue: boolean
  card: CharacterCard
  canShareLink?: boolean
  linkUnavailableReason?: string
}>(), {
  canShareLink: true,
  linkUnavailableReason: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t } = useI18n()
const toast = useToastStore()
const dialogRef = ref<HTMLElement | null>(null)
const posterRef = ref<PosterExpose | null>(null)
const mode = ref<ShareMode | null>(null)
const linkLoading = ref(false)
const posterSaving = ref(false)
const shareData = ref<CharacterCardShare | null>(null)
const linkError = ref('')
const displayName = computed(() => getCharacterCardDisplayName(props.card))
const shareUrl = computed(() => shareData.value?.path ? buildPublicSitePathUrl(shareData.value.path) : '')
const modeTitle = computed(() => {
  if (mode.value === 'link') return t('characterCards.share.linkTitle')
  if (mode.value === 'poster') return t('characterCards.share.posterTitle')
  return t('characterCards.share.chooseTitle')
})
let previousBodyOverflow = ''

watch(() => props.modelValue, async (visible) => {
  if (!visible) {
    releaseDialogEffects()
    mode.value = null
    linkError.value = ''
    return
  }
  previousBodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  document.addEventListener('keydown', handleKeydown)
  await nextTick()
  dialogRef.value?.focus()
})

watch(() => props.card.id, () => {
  shareData.value = null
  linkError.value = ''
})

onBeforeUnmount(() => releaseDialogEffects())

function releaseDialogEffects() {
  document.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = previousBodyOverflow
}

function close() {
  releaseDialogEffects()
  emit('update:modelValue', false)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

async function selectMode(nextMode: ShareMode) {
  if (nextMode === 'link' && !props.canShareLink) return
  mode.value = nextMode
  if (nextMode === 'link') await loadShareLink()
}

async function loadShareLink() {
  if (shareData.value || linkLoading.value || !props.canShareLink) return
  linkLoading.value = true
  linkError.value = ''
  try {
    shareData.value = await getCharacterCardShare(props.card.id)
  } catch (error: unknown) {
    console.error('加载人物卡分享链接失败:', error)
    linkError.value = t('characterCards.share.linkFailed')
  } finally {
    linkLoading.value = false
  }
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('Clipboard copy failed')
}

async function copyShareLink() {
  if (!shareUrl.value) await loadShareLink()
  if (!shareUrl.value) return
  try {
    await copyText(shareUrl.value)
    toast.success(t('characterCards.share.linkCopied'))
  } catch (error: unknown) {
    console.error('复制人物卡分享链接失败:', error)
    toast.error(t('characterCards.share.linkFailed'))
  }
}

async function shareLink() {
  if (!shareUrl.value) await loadShareLink()
  if (!shareUrl.value) return
  if (!navigator.share) {
    await copyShareLink()
    return
  }
  try {
    await navigator.share({
      title: shareData.value?.title || displayName.value,
      text: t('characterCards.detail.shareText', { name: shareData.value?.title || displayName.value }),
      url: shareUrl.value,
    })
  } catch (error: any) {
    if (error?.name !== 'AbortError') {
      console.error('分享人物卡链接失败:', error)
      toast.error(t('characterCards.share.linkFailed'))
    }
  }
}

function wait(milliseconds: number) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds))
}

async function inlineRemoteImages(root: HTMLElement) {
  const token = localStorage.getItem('token')
  await Promise.all(Array.from(root.querySelectorAll<HTMLImageElement>('img')).map(async (image) => {
    const source = image.currentSrc || image.src
    if (!source || /^data:/i.test(source)) return
    try {
      const target = new URL(source, window.location.href)
      const apiOrigin = new URL(API_BASE, window.location.href).origin
      const headers = token && (target.origin === window.location.origin || target.origin === apiOrigin)
        ? { Authorization: `Bearer ${token}` }
        : undefined
      const response = await fetch(target.toString(), { headers })
      if (!response.ok) return
      const blob = await response.blob()
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(String(reader.result || ''))
        reader.onerror = () => reject(reader.error)
        reader.readAsDataURL(blob)
      })
      if (!dataUrl) return
      image.src = dataUrl
      await image.decode().catch(() => undefined)
    } catch {
      // Keep the visible source when an external host does not allow inlining.
    }
  }))
}

async function waitForPoster(root: HTMLElement) {
  const deadline = Date.now() + 12000
  while (root.querySelector('[aria-busy="true"]') && Date.now() < deadline) {
    await wait(80)
  }
  await inlineRemoteImages(root)
  await Promise.all(Array.from(root.querySelectorAll<HTMLImageElement>('img')).map((image) => {
    if (image.complete) return Promise.resolve()
    return new Promise<void>((resolve) => {
      image.addEventListener('load', () => resolve(), { once: true })
      image.addEventListener('error', () => resolve(), { once: true })
    })
  }))
  await new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
}

function posterFilename() {
  const safeName = displayName.value
    .replace(/[<>:"/\\|?*\u0000-\u001f]/g, '-')
    .replace(/\s+/g, '-')
    .slice(0, 64) || `character-${props.card.id}`
  return `RPBox-${safeName}-人物卡.png`
}

async function savePoster() {
  if (posterSaving.value) return
  const poster = posterRef.value?.getElement()
  if (!poster) return
  posterSaving.value = true
  try {
    await waitForPoster(poster)
    const outputScale = Math.max(1, Math.min(2, 28000 / Math.max(poster.scrollHeight, 1)))
    const blob = await toBlob(poster, {
      backgroundColor: '#f1e7d4',
      cacheBust: true,
      pixelRatio: outputScale,
      skipFonts: true,
    })
    if (!blob) throw new Error('Poster image generation returned no data')
    const saved = await saveBlobAsFile(blob, posterFilename())
    if (saved) toast.success(t('characterCards.share.posterSaved'))
  } catch (error: unknown) {
    console.error('保存人物卡长图失败:', error)
    toast.error(t('characterCards.share.posterFailed'))
  } finally {
    posterSaving.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="share-dialog">
      <div v-if="modelValue" class="share-dialog-mask" @click.self="close">
        <section
          ref="dialogRef"
          class="share-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="character-share-title"
          tabindex="-1"
          data-testid="character-share-dialog"
        >
          <header class="share-dialog__header">
            <div>
              <span>{{ t('characterCards.share.kicker') }}</span>
              <h2 id="character-share-title">{{ modeTitle }}</h2>
              <p>{{ displayName }}</p>
            </div>
            <button type="button" :aria-label="t('characterCards.share.close')" @click="close">
              <i class="ri-close-line" aria-hidden="true"></i>
            </button>
          </header>

          <main v-if="!mode" class="share-chooser">
            <div class="share-chooser__intro">
              <span class="share-chooser__number">02</span>
              <div>
                <h3>{{ t('characterCards.share.chooseHeading') }}</h3>
                <p>{{ t('characterCards.share.chooseBody') }}</p>
              </div>
            </div>
            <div class="share-choices">
              <button
                type="button"
                class="share-choice share-choice--link"
                :disabled="!canShareLink"
                data-testid="share-choice-link"
                @click="selectMode('link')"
              >
                <span class="share-choice__icon"><i class="ri-link-m" aria-hidden="true"></i></span>
                <span class="share-choice__copy">
                  <small>{{ t('characterCards.share.linkBadge') }}</small>
                  <strong>{{ t('characterCards.share.linkTitle') }}</strong>
                  <span>{{ canShareLink ? t('characterCards.share.linkBody') : linkUnavailableReason }}</span>
                </span>
                <i class="ri-arrow-right-up-line share-choice__arrow" aria-hidden="true"></i>
              </button>

              <button
                type="button"
                class="share-choice share-choice--poster"
                data-testid="share-choice-poster"
                @click="selectMode('poster')"
              >
                <span class="share-choice__icon"><i class="ri-image-2-line" aria-hidden="true"></i></span>
                <span class="share-choice__copy">
                  <small>{{ t('characterCards.share.posterBadge') }}</small>
                  <strong>{{ t('characterCards.share.posterTitle') }}</strong>
                  <span>{{ t('characterCards.share.posterBody') }}</span>
                </span>
                <i class="ri-arrow-right-up-line share-choice__arrow" aria-hidden="true"></i>
              </button>
            </div>
            <p class="share-chooser__note"><i class="ri-shield-check-line" aria-hidden="true"></i>{{ t('characterCards.share.localNote') }}</p>
          </main>

          <main v-else-if="mode === 'link'" class="share-link-view">
            <button type="button" class="share-back" @click="mode = null">
              <i class="ri-arrow-left-line" aria-hidden="true"></i>{{ t('characterCards.share.back') }}
            </button>
            <div class="share-link-card">
              <span class="share-link-card__glyph"><i class="ri-links-line" aria-hidden="true"></i></span>
              <small>{{ t('characterCards.share.linkBadge') }}</small>
              <h3>{{ shareData?.title || displayName }}</h3>
              <p>{{ shareData?.summary || card.summary || t('characterCards.detail.summaryMissing') }}</p>
              <div class="share-url" :class="{ loading: linkLoading, error: linkError }">
                <i :class="linkLoading ? 'ri-loader-4-line spin' : linkError ? 'ri-error-warning-line' : 'ri-global-line'" aria-hidden="true"></i>
                <span>{{ linkLoading ? t('characterCards.share.loadingLink') : linkError || shareUrl }}</span>
                <button v-if="linkError" type="button" @click="loadShareLink">{{ t('characterCards.share.retry') }}</button>
              </div>
              <div class="share-link-actions">
                <button type="button" class="share-action share-action--secondary" :disabled="!shareUrl || linkLoading" data-testid="copy-share-link" @click="copyShareLink">
                  <i class="ri-file-copy-line" aria-hidden="true"></i>{{ t('characterCards.share.copyLink') }}
                </button>
                <button type="button" class="share-action share-action--primary" :disabled="!shareUrl || linkLoading" @click="shareLink">
                  <i class="ri-share-forward-line" aria-hidden="true"></i>{{ t('characterCards.share.shareNow') }}
                </button>
              </div>
            </div>
          </main>

          <main v-else class="share-poster-view">
            <aside class="poster-preview-rail">
              <button type="button" class="share-back" @click="mode = null">
                <i class="ri-arrow-left-line" aria-hidden="true"></i>{{ t('characterCards.share.back') }}
              </button>
              <div>
                <span>{{ t('characterCards.share.posterBadge') }}</span>
                <h3>{{ t('characterCards.share.previewHeading') }}</h3>
                <p>{{ t('characterCards.share.previewBody') }}</p>
              </div>
              <ul>
                <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.share.includesPortraits') }}</li>
                <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.share.includesDetails') }}</li>
                <li><i class="ri-check-line" aria-hidden="true"></i>{{ t('characterCards.share.includesStories') }}</li>
              </ul>
              <button type="button" class="share-action share-action--primary poster-save" :disabled="posterSaving" data-testid="save-share-poster" @click="savePoster">
                <i :class="posterSaving ? 'ri-loader-4-line spin' : 'ri-download-2-line'" aria-hidden="true"></i>
                {{ posterSaving ? t('characterCards.share.savingPoster') : t('characterCards.share.savePoster') }}
              </button>
              <small class="poster-save-hint">{{ t('characterCards.share.posterSaveHint') }}</small>
            </aside>
            <div class="poster-preview-stage">
              <div class="poster-preview-stage__label"><span></span>{{ t('characterCards.share.previewLabel') }}</div>
              <div class="poster-preview-scroll">
                <CharacterCardSharePoster ref="posterRef" :card="card" />
              </div>
            </div>
          </main>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.share-dialog-mask { position: fixed; z-index: 2800; inset: 0; display: grid; place-items: center; padding: 18px; background: color-mix(in srgb, var(--gradient-end) 76%, transparent); backdrop-filter: blur(10px); }
.share-dialog { display: flex; width: min(1180px, calc(100vw - 36px)); height: min(850px, calc(100vh - 36px)); flex-direction: column; overflow: hidden; border: 1px solid var(--gradient-border); border-radius: 18px; background: var(--color-panel-bg); color: var(--color-text-main); box-shadow: 0 28px 80px color-mix(in srgb, var(--gradient-end) 48%, transparent); outline: none; }
.share-dialog__header { display: flex; min-height: 90px; align-items: center; justify-content: space-between; padding: 18px 24px 18px 30px; border-bottom: 1px solid var(--color-border); background: linear-gradient(100deg, color-mix(in srgb, var(--color-accent) 9%, transparent), transparent 42%), var(--color-panel-bg); }
.share-dialog__header > div { display: grid; grid-template-columns: auto auto; align-items: baseline; gap: 4px 14px; }
.share-dialog__header span { grid-column: 1 / -1; color: var(--color-accent); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .18em; text-transform: uppercase; }
.share-dialog__header h2 { margin: 0; color: var(--color-primary); font: 600 23px/1.2 Georgia, 'Noto Serif SC', serif; }
.share-dialog__header p { margin: 0; color: var(--color-text-secondary); font-size: 12px; }
.share-dialog__header > button { display: grid; width: 38px; height: 38px; place-items: center; border: 1px solid var(--color-border); border-radius: 50%; background: var(--color-card-bg); color: var(--color-text-secondary); cursor: pointer; font-size: 20px; }
.share-dialog__header > button:hover { border-color: var(--color-border-hover); color: var(--color-primary); }

.share-chooser { display: grid; flex: 1; align-content: center; gap: 32px; padding: 50px clamp(30px, 7vw, 94px); overflow-y: auto; background: radial-gradient(circle at 50% 0, color-mix(in srgb, var(--color-accent) 10%, transparent), transparent 42%); }
.share-chooser__intro { display: grid; grid-template-columns: 58px minmax(0, 1fr); align-items: start; gap: 22px; max-width: 660px; }
.share-chooser__number { color: color-mix(in srgb, var(--color-accent) 65%, transparent); font: 500 42px/1 Georgia, serif; }
.share-chooser__intro h3 { margin: 0 0 7px; color: var(--color-primary); font: 600 28px/1.25 Georgia, 'Noto Serif SC', serif; }
.share-chooser__intro p { margin: 0; color: var(--color-text-secondary); line-height: 1.65; }
.share-choices { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
.share-choice { position: relative; display: grid; min-height: 238px; grid-template-columns: 64px minmax(0, 1fr) 28px; align-content: center; align-items: center; gap: 20px; padding: 30px; overflow: hidden; border: 1px solid var(--color-border); border-radius: 14px; background: var(--color-card-bg); color: var(--color-text-main); cursor: pointer; text-align: left; transition: transform .2s ease, border-color .2s ease, box-shadow .2s ease; }
.share-choice::before { position: absolute; inset: 0; content: ''; opacity: .42; pointer-events: none; }
.share-choice--link::before { background: linear-gradient(145deg, color-mix(in srgb, var(--color-accent) 14%, transparent), transparent 48%); }
.share-choice--poster { background: var(--gradient-end); color: var(--gradient-text); }
.share-choice--poster::before { background: radial-gradient(circle at 86% 12%, color-mix(in srgb, var(--color-accent) 40%, transparent), transparent 45%); }
.share-choice:hover:not(:disabled) { transform: translateY(-4px); border-color: var(--color-border-hover); box-shadow: 0 16px 34px color-mix(in srgb, var(--color-primary) 16%, transparent); }
.share-choice:disabled { cursor: not-allowed; filter: grayscale(.35); opacity: .56; }
.share-choice__icon { position: relative; z-index: 1; display: grid; width: 64px; height: 64px; place-items: center; border: 1px solid var(--color-border-hover); border-radius: 50%; background: var(--color-panel-bg); color: var(--color-accent); font-size: 28px; }
.share-choice--poster .share-choice__icon { border-color: var(--gradient-border); background: var(--gradient-surface); color: #e6b071; }
.share-choice__copy { position: relative; z-index: 1; display: grid; gap: 8px; }
.share-choice__copy small { color: var(--color-accent); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .15em; text-transform: uppercase; }
.share-choice__copy strong { color: inherit; font: 600 23px/1.15 Georgia, 'Noto Serif SC', serif; }
.share-choice__copy > span { max-width: 310px; color: var(--color-text-secondary); font-size: 12px; line-height: 1.65; }
.share-choice--poster .share-choice__copy > span { color: var(--gradient-text-muted); }
.share-choice__arrow { position: relative; z-index: 1; align-self: start; color: var(--color-text-secondary); font-size: 20px; }
.share-choice--poster .share-choice__arrow { color: var(--gradient-text-muted); }
.share-chooser__note { display: flex; align-items: center; gap: 7px; margin: 0; color: var(--color-text-secondary); font-size: 11px; }
.share-chooser__note i { color: var(--color-accent); font-size: 15px; }

.share-back { display: inline-flex; align-items: center; gap: 6px; width: fit-content; padding: 5px 0; border: 0; background: transparent; color: var(--color-text-secondary); cursor: pointer; font: inherit; font-size: 11px; }
.share-link-view { display: grid; flex: 1; align-content: center; justify-items: center; gap: 20px; padding: 34px; overflow-y: auto; background: linear-gradient(160deg, color-mix(in srgb, var(--color-accent) 8%, transparent), transparent 42%); }
.share-link-view > .share-back { width: min(680px, 100%); }
.share-link-card { display: grid; width: min(680px, 100%); justify-items: center; padding: 50px; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-card-bg); box-shadow: var(--shadow-md); text-align: center; }
.share-link-card__glyph { display: grid; width: 78px; height: 78px; margin-bottom: 20px; place-items: center; border: 1px solid var(--color-border-hover); border-radius: 50%; background: var(--tag-bg); color: var(--tag-text); font-size: 32px; outline: 1px dashed var(--color-border); outline-offset: 7px; }
.share-link-card > small { color: var(--color-accent); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .16em; text-transform: uppercase; }
.share-link-card h3 { margin: 12px 0 8px; color: var(--color-primary); font: 600 31px/1.25 Georgia, 'Noto Serif SC', serif; }
.share-link-card > p { max-width: 500px; margin: 0; color: var(--color-text-secondary); font-size: 13px; line-height: 1.7; }
.share-url { display: grid; width: 100%; min-height: 54px; grid-template-columns: 20px minmax(0, 1fr) auto; align-items: center; gap: 10px; margin-top: 30px; padding: 0 15px; border: 1px solid var(--input-border); border-radius: 9px; background: var(--input-bg); color: var(--color-text-main); text-align: left; }
.share-url > span { overflow: hidden; font: 12px/1.4 ui-monospace, Consolas, monospace; text-overflow: ellipsis; white-space: nowrap; }
.share-url > i { color: var(--color-accent); }
.share-url.error { border-color: color-mix(in srgb, var(--btn-danger-bg) 48%, var(--color-border)); color: var(--btn-danger-bg); }
.share-url > button { border: 0; background: transparent; color: var(--link-color); cursor: pointer; }
.share-link-actions { display: flex; width: 100%; gap: 10px; margin-top: 14px; }
.share-action { display: inline-flex; min-height: 44px; flex: 1; align-items: center; justify-content: center; gap: 7px; padding: 0 18px; border: 1px solid var(--btn-outline-border); border-radius: 8px; cursor: pointer; font: inherit; font-size: 12px; font-weight: 700; }
.share-action--secondary { background: var(--color-panel-bg); color: var(--btn-outline-text); }
.share-action--primary { border-color: var(--btn-primary-bg); background: var(--btn-primary-bg); color: var(--btn-primary-text); }
.share-action:disabled { cursor: wait; opacity: .6; }

.share-poster-view { display: grid; min-height: 0; flex: 1; grid-template-columns: 230px minmax(0, 1fr); }
.poster-preview-rail { display: flex; min-height: 0; flex-direction: column; gap: 24px; padding: 28px 24px; border-right: 1px solid var(--color-border); background: var(--color-card-bg); }
.poster-preview-rail > div > span { color: var(--color-accent); font: 700 9px/1 ui-monospace, Consolas, monospace; letter-spacing: .16em; text-transform: uppercase; }
.poster-preview-rail h3 { margin: 12px 0 9px; color: var(--color-primary); font: 600 23px/1.25 Georgia, 'Noto Serif SC', serif; }
.poster-preview-rail p { margin: 0; color: var(--color-text-secondary); font-size: 11px; line-height: 1.65; }
.poster-preview-rail ul { display: grid; gap: 9px; margin: 0; padding: 18px 0; border-top: 1px solid var(--color-border); border-bottom: 1px solid var(--color-border); list-style: none; color: var(--color-text-secondary); font-size: 10px; }
.poster-preview-rail li { display: flex; gap: 7px; }
.poster-preview-rail li i { color: var(--color-accent); }
.poster-save { flex: 0 0 auto; margin-top: auto; }
.poster-save-hint { color: var(--color-text-muted); font-size: 9px; line-height: 1.5; text-align: center; }
.poster-preview-stage { display: flex; min-width: 0; min-height: 0; flex-direction: column; padding: 16px; background: color-mix(in srgb, var(--gradient-end) 92%, black); }
.poster-preview-stage__label { display: flex; align-items: center; gap: 8px; height: 30px; color: var(--gradient-text-muted); font: 700 8px/1 ui-monospace, Consolas, monospace; letter-spacing: .14em; text-transform: uppercase; }
.poster-preview-stage__label span { width: 6px; height: 6px; border-radius: 50%; background: #68ad8c; box-shadow: 0 0 0 4px rgba(104,173,140,.12); }
.poster-preview-scroll { flex: 1; min-height: 0; overflow: auto; border: 1px solid var(--gradient-border); border-radius: 7px; background: #10171d; }
.poster-preview-scroll :deep(.share-poster) { margin: 0 auto; box-shadow: 0 18px 44px rgba(0,0,0,.32); }

.share-dialog button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-accent) 32%, transparent); outline-offset: 2px; }
.spin { animation: share-spin 900ms linear infinite; }
@keyframes share-spin { to { transform: rotate(360deg); } }
.share-dialog-enter-active, .share-dialog-leave-active { transition: opacity .22s ease; }
.share-dialog-enter-active .share-dialog, .share-dialog-leave-active .share-dialog { transition: transform .22s ease, opacity .22s ease; }
.share-dialog-enter-from, .share-dialog-leave-to { opacity: 0; }
.share-dialog-enter-from .share-dialog, .share-dialog-leave-to .share-dialog { opacity: 0; transform: translateY(14px) scale(.985); }

@media (max-width: 800px) {
  .share-dialog-mask { padding: 8px; }
  .share-dialog { width: calc(100vw - 16px); height: calc(100vh - 16px); border-radius: 14px; }
  .share-choices { grid-template-columns: 1fr; }
  .share-choice { min-height: 174px; }
  .share-poster-view { grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); }
  .poster-preview-rail { display: grid; grid-template-columns: 1fr auto; gap: 8px 16px; padding: 14px 16px; border-right: 0; border-bottom: 1px solid var(--color-border); }
  .poster-preview-rail > div, .poster-preview-rail ul, .poster-save-hint { display: none; }
  .poster-preview-rail .share-back { align-self: center; }
  .poster-save { width: auto; margin: 0; }
  .share-link-card { padding: 34px 22px; }
}

@media (max-width: 520px) {
  .share-dialog__header { min-height: 76px; padding: 14px 16px; }
  .share-dialog__header h2 { font-size: 19px; }
  .share-dialog__header p { display: none; }
  .share-chooser { align-content: start; padding: 28px 18px; }
  .share-chooser__intro { grid-template-columns: 1fr; gap: 8px; }
  .share-chooser__number { font-size: 28px; }
  .share-choice { grid-template-columns: 48px minmax(0, 1fr); gap: 14px; padding: 22px 18px; }
  .share-choice__icon { width: 48px; height: 48px; font-size: 22px; }
  .share-choice__arrow { display: none; }
  .share-link-view { padding: 22px 14px; }
  .share-link-actions { flex-direction: column; }
}

@media (prefers-reduced-motion: reduce) {
  .share-choice, .share-dialog-enter-active, .share-dialog-leave-active, .share-dialog-enter-active .share-dialog, .share-dialog-leave-active .share-dialog { transition: none; }
  .spin { animation: none; }
}
</style>
