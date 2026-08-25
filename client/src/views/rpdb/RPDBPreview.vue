<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { resolveRPDBMediaURL, type RPDBWorkPayload } from '@/api/rpdb'
import RPDBWorkContent from '@/components/rpdb/RPDBWorkContent.vue'
import { useRPDBOptionLabels } from '@/composables/useRPDBOptionLabels'

const router = useRouter()
const { availabilityLabel, bindTypeLabel, factionLabel } = useRPDBOptionLabels()
const viewport = ref<'desktop' | 'mobile'>('desktop')
const work = computed<RPDBWorkPayload>(() => {
  try {
    return JSON.parse(sessionStorage.getItem('rpdb-preview') || '{}')
  } catch {
    return {} as RPDBWorkPayload
  }
})
const coverURL = computed(() => resolveRPDBMediaURL(work.value.cover_image))
const typeLabel = computed(() => ({
  item_showcase: '魔兽物品',
  transmog: '幻化方案',
  home_showcase: '家宅分享',
  musician_midi: 'Musician MIDI',
}[work.value.type] || '玩家作品'))
const homeDetails = computed<Record<string, string>>(() => {
  const value = work.value.extra
  if (!value) return {}
  if (typeof value === 'object') return value as Record<string, string>
  try {
    return JSON.parse(value)
  } catch {
    return {}
  }
})
const checks = computed(() => [
  { label: '标题与摘要', done: Boolean(work.value.title?.trim() && work.value.summary?.trim()) },
  { label: '封面与媒体', done: Boolean(work.value.cover_image || work.value.media?.length) },
  { label: '正文内容', done: Boolean(work.value.content?.trim()) },
  {
    label: work.value.type === 'home_showcase' ? '家宅资料' : work.value.type === 'musician_midi' ? 'MIDI 文件' : '关联游戏物品',
    done: work.value.type === 'home_showcase'
      ? Boolean(homeDetails.value.share_code)
      : work.value.type === 'musician_midi'
        ? Boolean(homeDetails.value.midi_url)
      : Boolean(work.value.references?.length),
  },
  {
    label: work.value.type === 'home_showcase' ? '参观说明' : work.value.type === 'musician_midi' ? '乐曲介绍' : '获取攻略',
    done: work.value.type === 'home_showcase'
      ? Boolean(homeDetails.value.visit_notes)
      : work.value.type === 'musician_midi'
        ? Boolean(work.value.content?.trim())
        : Boolean(work.value.guide_steps?.length),
  },
])
const completedChecks = computed(() => checks.value.filter(item => item.done).length)
</script>

<template>
  <div class="preview-page minimal-preview-shell">
    <header class="preview-toolbar" data-testid="preview-toolbar">
      <div class="toolbar-left">
        <button type="button" @click="router.back()">
          <i class="ri-arrow-left-line"></i>
          返回编辑
        </button>
        <span class="draft-status"><i class="ri-draft-line"></i>草稿预览，不会公开</span>
      </div>
      <div class="toolbar-actions">
        <div class="viewport-switch" aria-label="预览尺寸">
          <button type="button" :class="{ active: viewport === 'desktop' }" @click="viewport = 'desktop'">
            <i class="ri-computer-line"></i>
            桌面
          </button>
          <button type="button" :class="{ active: viewport === 'mobile' }" @click="viewport = 'mobile'">
            <i class="ri-smartphone-line"></i>
            窄屏
          </button>
        </div>
        <button type="button" class="submit" @click="router.back()">
          <i class="ri-edit-line"></i>
          返回补充
        </button>
      </div>
    </header>

    <div class="preview-layout" data-testid="preview-workspace">
      <main class="preview-stage">
        <article class="preview-document" :class="`is-${viewport}`">
          <section class="work-hero" data-testid="work-hero">
            <div class="cover-stage">
              <img v-if="coverURL" :src="coverURL" :alt="work.title || '作品封面'">
              <div v-else class="cover-empty">
                <i class="ri-image-line"></i>
                <span>尚未上传封面</span>
              </div>
              <div v-if="work.media?.length" class="media-count">
                <i class="ri-gallery-line"></i>
                {{ work.media.length + (work.cover_image ? 1 : 0) }} 张媒体
              </div>
            </div>
            <div class="hero-copy">
              <span class="work-type">{{ typeLabel }} · 内部草稿</span>
              <h1>{{ work.title || '未命名作品' }}</h1>
              <p>{{ work.summary || '填写摘要后，浏览者会在这里快速了解作品价值。' }}</p>
              <div class="author-row">
                <span class="avatar">R</span>
                <span>
                  当前作者
                  <small>提交前预览</small>
                </span>
              </div>
              <div class="preview-stats">
                <span><b>{{ work.references?.length || 0 }}</b>关联对象</span>
                <span><b>{{ work.media?.length || 0 }}</b>媒体</span>
                <span><b>{{ work.guide_steps?.length || 0 }}</b>攻略步骤</span>
              </div>
            </div>
          </section>

          <div class="preview-body">
            <RPDBWorkContent :work="work" :home-details="homeDetails" />
            <aside class="work-metadata">
              <section>
                <h3>作品资料</h3>
                <dl>
                  <div v-if="work.type !== 'musician_midi'"><dt>获取状态</dt><dd>{{ availabilityLabel(work.availability_status, work.type) }}</dd></div>
                  <div v-if="work.type === 'item_showcase'"><dt>是否绑定</dt><dd>{{ bindTypeLabel(work.bind_type) }}</dd></div>
                  <div v-if="work.type !== 'musician_midi'"><dt>阵营</dt><dd>{{ factionLabel(work.faction) }}</dd></div>
                  <div v-if="work.type === 'transmog'"><dt>护甲类型</dt><dd>{{ work.armor_type || '不限' }}</dd></div>
                </dl>
              </section>
              <section v-if="work.references?.length">
                <h3>引用物品</h3>
                <div v-for="item in work.references" :key="item.id || item.external_id" class="reference">
                  <i class="ri-external-link-line"></i>
                  <span><b>{{ item.name || '未命名物品' }}</b><small>{{ item.source || '外部资料' }}</small></span>
                </div>
              </section>
              <section class="preview-actions">
                <h3>发布后的操作</h3>
                <button type="button"><i class="ri-bookmark-3-line"></i>收藏作品</button>
                <button type="button"><i class="ri-add-circle-line"></i>加入收集清单</button>
              </section>
            </aside>
          </div>
        </article>
      </main>

    </div>

    <aside class="quality-panel" data-testid="preview-quality">
      <header>
        <span>发布质量检查</span>
        <h2>发布质量检查</h2>
        <p>{{ completedChecks }} / {{ checks.length }} 项已经完成</p>
      </header>
      <div class="quality-list">
        <div v-for="item in checks" :key="item.label" :class="{ done: item.done }">
          <i :class="item.done ? 'ri-checkbox-circle-fill' : 'ri-error-warning-line'"></i>
          <span>{{ item.label }}</span>
          <b>{{ item.done ? '通过' : '待补充' }}</b>
        </div>
      </div>
      <p class="quality-note">
        <i class="ri-information-line"></i>
        预览页与公开详情使用相同的正文和攻略组件。
      </p>
      <button type="button" class="quality-return" @click="router.back()">返回编辑器</button>
    </aside>
  </div>
</template>

<style scoped>
.preview-page{max-width:1380px;margin:auto;color:var(--color-text-main)}
.minimal-preview-shell{--rpdb-surface:color-mix(in srgb,var(--color-panel-bg) 88%,#fff 12%);--rpdb-muted:color-mix(in srgb,var(--color-card-bg) 84%,#fff 16%);--rpdb-line:color-mix(in srgb,var(--color-border) 72%,transparent);--rpdb-soft:color-mix(in srgb,var(--color-accent) 8%,transparent)}
.preview-toolbar{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-bottom:16px;padding:10px 0;border-bottom:1px solid var(--rpdb-line)}
.toolbar-left,.toolbar-actions,.viewport-switch{display:flex;align-items:center;gap:8px}
.preview-toolbar button{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:34px;padding:0 12px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--rpdb-surface);color:var(--color-text-main)}
.draft-status{display:inline-flex;align-items:center;gap:6px;color:var(--color-text-secondary);font-size:12px}
.viewport-switch{gap:4px;padding:3px;border:1px solid var(--rpdb-line);border-radius:12px}
.viewport-switch button{min-height:30px;border:0;background:transparent}
.viewport-switch button.active,.preview-toolbar .submit{background:var(--color-accent);color:#fff}
.preview-layout{display:block}
.preview-stage{min-width:0}
.preview-document,.quality-panel{overflow:hidden;border:1px solid var(--rpdb-line);border-radius:14px;background:var(--rpdb-surface)}
.preview-document{margin:auto}
.preview-document.is-mobile{max-width:520px}
.work-hero{display:grid;grid-template-columns:minmax(0,1.3fr) minmax(320px,.7fr);min-height:330px;border-bottom:1px solid var(--rpdb-line)}
.cover-stage{position:relative;display:grid;min-height:330px;place-items:center;overflow:hidden;background:#18130f}
.cover-stage img{width:100%;height:100%;object-fit:cover}
.cover-empty{display:grid;place-items:center;align-content:center;gap:8px;color:#b9a89a}
.cover-empty i{font-size:44px}
.media-count{position:absolute;right:12px;bottom:12px;padding:7px 10px;border-radius:999px;background:rgba(26,18,13,.74);color:#fff;font-size:11px}
.hero-copy{display:flex;flex-direction:column;padding:26px}
.work-type{align-self:flex-start;padding:5px 9px;border-radius:999px;background:var(--rpdb-soft);color:var(--color-accent);font-size:11px;font-weight:800}
.hero-copy h1{margin:16px 0 10px;color:var(--color-text-main);font:700 32px/1.2 system-ui,'Microsoft YaHei',sans-serif}
.hero-copy>p{margin:0;color:var(--color-text-secondary);font-size:14px;line-height:1.8}
.author-row{display:flex;align-items:center;gap:9px;margin-top:20px;color:var(--color-text-main)}
.avatar{display:grid;width:34px;height:34px;place-items:center;border-radius:50%;background:var(--color-secondary);color:#fff;font-weight:800}
.author-row>span:last-child{display:flex;flex-direction:column}
.author-row small{margin-top:3px;color:var(--color-text-secondary)}
.preview-stats{display:grid;grid-template-columns:repeat(3,1fr);margin-top:auto;padding-top:20px;border-top:1px solid var(--rpdb-line)}
.preview-stats span{text-align:center;color:var(--color-text-secondary);font-size:11px}
.preview-stats b{display:block;margin-bottom:3px;color:var(--color-accent);font-size:18px}
.preview-body{display:grid;grid-template-columns:minmax(0,1fr) 250px}
.work-metadata{padding:22px 18px;border-left:1px solid var(--rpdb-line);background:var(--rpdb-muted)}
.work-metadata section{padding:0 0 15px;margin-bottom:15px;border-bottom:1px solid var(--rpdb-line)}
.work-metadata section:last-child{margin-bottom:0;border-bottom:0}
.work-metadata h3{margin:0 0 10px;color:var(--color-text-main);font-size:14px}
.work-metadata dl{display:grid;gap:8px;margin:0}
.work-metadata dl div{display:flex;justify-content:space-between;gap:10px}
.work-metadata dt{color:var(--color-text-secondary)}
.work-metadata dd{margin:0;text-align:right}
.reference{display:flex;gap:8px;padding:8px 0;border-bottom:1px solid var(--rpdb-line)}
.reference i{color:var(--color-accent)}
.reference span{display:flex;min-width:0;flex-direction:column}
.reference small{color:var(--color-text-secondary)}
.preview-actions{display:grid;gap:7px}
.preview-actions h3{margin-bottom:3px}
.preview-actions button,.quality-return{display:inline-flex;align-items:center;justify-content:center;gap:6px;min-height:34px;border:1px solid var(--rpdb-line);border-radius:10px;background:var(--color-panel-bg);color:var(--color-text-main)}
.quality-panel{position:fixed;top:86px;right:max(18px,calc((100vw - 1380px) / 2 + 18px));z-index:20;width:270px;box-shadow:0 16px 36px rgba(0,0,0,.14);backdrop-filter:blur(14px)}
.quality-panel header{padding:16px;border-bottom:1px solid var(--rpdb-line)}
.quality-panel header span{color:var(--color-accent);font-size:10px;font-weight:800;letter-spacing:.06em}
.quality-panel h2{margin:5px 0;color:var(--color-text-main);font:700 20px/1.25 system-ui,'Microsoft YaHei',sans-serif}
.quality-panel header p{margin:0;color:var(--color-text-secondary);font-size:12px}
.quality-list{padding:8px 14px}
.quality-list>div{display:grid;grid-template-columns:20px 1fr auto;gap:7px;align-items:center;padding:10px 0;color:var(--color-text-secondary)}
.quality-list i{color:#b65a4f}
.quality-list b{color:#b65a4f;font-size:11px}
.quality-list .done i,.quality-list .done b{color:#4d7a4c}
.quality-note{display:flex;gap:7px;margin:0;padding:12px 14px;background:var(--rpdb-muted);color:var(--color-text-secondary);font-size:11px;line-height:1.6}
.quality-return{width:calc(100% - 28px);margin:14px;color:var(--color-accent)}
.preview-document.is-mobile .work-hero,.preview-document.is-mobile .preview-body{grid-template-columns:1fr}
.preview-document.is-mobile .work-metadata{border-top:1px solid var(--rpdb-line);border-left:0}
@media(max-width:1100px){.quality-panel{position:static;width:auto;margin-top:14px;box-shadow:none}.quality-list{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}}
@media(max-width:760px){.preview-toolbar{align-items:stretch;flex-direction:column}.toolbar-left,.toolbar-actions{justify-content:space-between}.work-hero,.preview-body{grid-template-columns:1fr}.cover-stage{min-height:250px}.hero-copy{padding:22px}.work-metadata{border-top:1px solid var(--rpdb-line);border-left:0}.quality-list{grid-template-columns:1fr}}
@media(max-width:520px){.toolbar-left,.toolbar-actions{align-items:stretch;flex-direction:column}.viewport-switch button{flex:1}.hero-copy h1{font-size:27px}}
</style>
