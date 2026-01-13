<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'

// 感情状态枚举
const RelationshipStatusMap: Record<number, string> = {
  1: '单身', 2: '恋爱中', 3: '已婚', 4: '离异', 5: '丧偶'
}

// 性格特征预设
const PersonalityPresets: Record<number, { left: string; right: string }> = {
  1: { left: '混乱', right: '守序' },
  2: { left: '贞洁', right: '好色' },
  3: { left: '宽容', right: '记仇' },
  4: { left: '利他', right: '自私' },
  5: { left: '诚实', right: '欺骗' },
  6: { left: '温和', right: '残暴' },
  7: { left: '迷信', right: '理性' },
  8: { left: '叛逆', right: '典范' },
  9: { left: '谨慎', right: '冲动' },
  10: { left: '禁欲', right: '享乐' },
  11: { left: '勇敢', right: '懦弱' }
}

// 其他信息预设类型
const MiscInfoPresets: Record<number, string> = {
  1: '自定义', 2: '家族/家名', 3: '昵称', 4: '座右铭', 5: '面部特征',
  6: '穿孔', 7: '代词', 8: 'RP公会名', 9: 'RP公会头衔', 10: '纹身', 11: '声音参考'
}

interface PersonalityTrait {
  ID?: number; LT?: string; RT?: string; V2?: number
}

interface MiscInfo {
  ID?: number; NA?: string; VA?: string; IC?: string
}

interface LocalProfile {
  id: string
  name: string
  icon?: string
  checksum: string
  characteristics?: {
    FN?: string; LN?: string; TI?: string; FT?: string
    RA?: string; CL?: string; CH?: string; AG?: string
    EC?: string; EH?: string; HE?: string; WE?: string
    BP?: string; RE?: string; RS?: number; IC?: string
    MI?: MiscInfo[]; PS?: PersonalityTrait[]
  }
  about?: {
    TE?: number
    T1?: { TX?: string }
    T2?: Array<{ TX?: string; IC?: string }>
    T3?: {
      PH?: { TX?: string }
      PS?: { TX?: string }
      HI?: { TX?: string }
    }
  }
  character?: {
    RP?: number; WU?: number; CU?: string; CO?: string
  }
}

const route = useRoute()
const router = useRouter()
const profileId = computed(() => route.params.id as string)

const profile = ref<LocalProfile | null>(null)
const isLoading = ref(false)
const mounted = ref(false)

// 构建显示名称
const displayName = computed(() => {
  const c = profile.value?.characteristics
  if (c?.FN && c?.LN) return `${c.FN} ${c.LN}`
  if (c?.FN) return c.FN
  return profile.value?.name || '未命名角色'
})

// 构建标签
const metaBadges = computed(() => {
  const badges: string[] = []
  const c = profile.value?.characteristics
  if (c?.RA) badges.push(c.RA)
  if (c?.CL) badges.push(c.CL)
  if (c?.RS && RelationshipStatusMap[c.RS]) badges.push(RelationshipStatusMap[c.RS])
  return badges
})

// 获取性格特征标签
function getTraitLabels(trait: PersonalityTrait) {
  if (trait.ID && PersonalityPresets[trait.ID]) return PersonalityPresets[trait.ID]
  return { left: trait.LT || '左', right: trait.RT || '右' }
}

// 获取其他信息名称
function getMiscName(misc: MiscInfo): string {
  if (misc.NA) return misc.NA
  if (misc.ID && MiscInfoPresets[misc.ID]) return MiscInfoPresets[misc.ID]
  return '其他'
}

// 检查各部分是否有内容 - 改为始终显示
const hasBasicInfo = computed(() => true)

const hasAbout = computed(() => true)

const hasCharacter = computed(() => true)

const hasPersonality = computed(() => {
  const ps = profile.value?.characteristics?.PS
  return Array.isArray(ps) && ps.length > 0
})

const hasMiscInfo = computed(() => {
  const mi = profile.value?.characteristics?.MI
  return Array.isArray(mi) && mi.length > 0
})

onMounted(async () => {
  await loadProfile()
  setTimeout(() => mounted.value = true, 50)
})

async function loadProfile() {
  isLoading.value = true
  try {
    const wowPath = localStorage.getItem('wow_path') || ''
    profile.value = await invoke<LocalProfile>('get_profile_detail', {
      wowPath, profileId: profileId.value
    })
    console.log('[ProfileDetail] 返回数据:', JSON.stringify(profile.value, null, 2))
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="detail-page" :class="{ 'animate-in': mounted }">
    <!-- 顶部工具栏 -->
    <header class="top-toolbar anim-item" style="--delay: 0">
      <div class="breadcrumbs">
        <i class="ri-home-4-line"></i>
        <span class="separator">/</span>
        <span>人物卡</span>
        <span class="separator">/</span>
        <span class="current">{{ displayName }}</span>
      </div>
      <div class="toolbar-actions">
        <button class="btn btn-secondary" @click="router.back()">
          <i class="ri-arrow-left-line"></i> 返回
        </button>
      </div>
    </header>

    <!-- 加载状态 -->
    <div v-if="isLoading" class="loading-state">
      <div class="loader"></div>
      <p>正在加载人物卡...</p>
    </div>

    <!-- 主内容 -->
    <template v-else-if="profile">
      <!-- 角色头部 -->
      <div class="char-header anim-item" style="--delay: 1">
        <div class="char-avatar">
          <i class="ri-user-3-line"></i>
        </div>
        <div class="char-info">
          <h1>{{ displayName }}</h1>
          <p class="char-title" v-if="profile.characteristics?.TI">{{ profile.characteristics.TI }}</p>
          <div class="char-meta" v-if="metaBadges.length">
            <span class="meta-badge" v-for="badge in metaBadges" :key="badge">{{ badge }}</span>
          </div>
        </div>
      </div>

      <!-- 基本信息 -->
      <section class="panel anim-item" style="--delay: 2" v-if="hasBasicInfo">
        <div class="panel-header">
          <div class="panel-title"><i class="ri-profile-line"></i> 基本信息</div>
        </div>
        <div class="panel-body">
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label">头衔</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.TI }">{{ profile.characteristics?.TI || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">全名</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.FT }">{{ profile.characteristics?.FT || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">年龄</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.AG }">{{ profile.characteristics?.AG || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">身高</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.HE }">{{ profile.characteristics?.HE || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">体重</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.WE }">{{ profile.characteristics?.WE || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">眼睛颜色</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.EC }" :style="profile.characteristics?.EH ? { color: '#' + profile.characteristics.EH } : {}">
                {{ profile.characteristics?.EC || '-' }}
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">出生地</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.BP }">{{ profile.characteristics?.BP || '-' }}</div>
            </div>
            <div class="form-group">
              <label class="form-label">居住地</label>
              <div class="form-value" :class="{ empty: !profile.characteristics?.RE }">{{ profile.characteristics?.RE || '-' }}</div>
            </div>
          </div>
        </div>
      </section>

      <!-- 当前状态 -->
      <section class="panel anim-item" style="--delay: 3" v-if="hasCharacter">
        <div class="panel-header">
          <div class="panel-title"><i class="ri-emotion-line"></i> 当前状态</div>
          <div class="rp-badge" :class="profile.character?.RP === 1 ? 'ic' : 'ooc'">
            {{ profile.character?.RP === 1 ? 'IC 角色扮演中' : 'OOC 非角色扮演' }}
          </div>
        </div>
        <div class="panel-body">
          <div class="status-content">
            <label class="form-label">当前状态</label>
            <p :class="{ empty: !profile.character?.CU }">{{ profile.character?.CU || '-' }}</p>
          </div>
          <div class="status-ooc">
            <label class="form-label">OOC 备注</label>
            <p :class="{ empty: !profile.character?.CO }">{{ profile.character?.CO || '-' }}</p>
          </div>
        </div>
      </section>

      <!-- 关于 -->
      <section class="panel anim-item" style="--delay: 4" v-if="hasAbout">
        <div class="panel-header">
          <div class="panel-title"><i class="ri-book-open-line"></i> 关于</div>
          <span class="template-badge">模板 {{ profile.about?.TE || 1 }}</span>
        </div>
        <div class="panel-body">
          <!-- 模板1：单一文本 -->
          <div v-if="(profile.about?.TE || 1) === 1" class="about-text" :class="{ empty: !profile.about?.T1?.TX }">
            {{ profile.about?.T1?.TX || '暂无内容' }}
          </div>
          <!-- 模板2：多框架 -->
          <div v-else-if="profile.about?.TE === 2" class="about-blocks">
            <template v-if="profile.about?.T2?.length">
              <div class="about-block" v-for="(block, idx) in profile.about.T2" :key="idx">
                <p>{{ block.TX || '' }}</p>
              </div>
            </template>
            <div v-else class="empty">暂无内容</div>
          </div>
          <!-- 模板3：外貌/性格/历史 -->
          <div v-else-if="profile.about?.TE === 3" class="about-sections">
            <div class="about-section">
              <h4>外貌</h4>
              <p :class="{ empty: !profile.about?.T3?.PH?.TX }">{{ profile.about?.T3?.PH?.TX || '-' }}</p>
            </div>
            <div class="about-section">
              <h4>性格</h4>
              <p :class="{ empty: !profile.about?.T3?.PS?.TX }">{{ profile.about?.T3?.PS?.TX || '-' }}</p>
            </div>
            <div class="about-section">
              <h4>历史</h4>
              <p :class="{ empty: !profile.about?.T3?.HI?.TX }">{{ profile.about?.T3?.HI?.TX || '-' }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- 性格特征 -->
      <section class="panel anim-item" style="--delay: 5" v-if="hasPersonality">
        <div class="panel-header">
          <div class="panel-title"><i class="ri-mental-health-line"></i> 性格特征</div>
        </div>
        <div class="panel-body">
          <div class="traits-grid">
            <div class="trait-item" v-for="(t, idx) in profile.characteristics?.PS" :key="idx">
              <div class="trait-labels">
                <span>{{ getTraitLabels(t).left }}</span>
                <span>{{ getTraitLabels(t).right }}</span>
              </div>
              <div class="trait-bar">
                <div class="trait-fill" :style="{ width: `${((t.V2 || 10) / 20) * 100}%` }"></div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- 其他信息 -->
      <section class="panel anim-item" style="--delay: 6" v-if="hasMiscInfo">
        <div class="panel-header">
          <div class="panel-title"><i class="ri-file-list-3-line"></i> 其他信息</div>
        </div>
        <div class="panel-body">
          <div class="misc-grid">
            <div class="misc-item" v-for="(m, idx) in profile.characteristics?.MI" :key="idx">
              <label>{{ getMiscName(m) }}</label>
              <span>{{ m.VA || '-' }}</span>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.detail-page {
  padding: 24px;
  min-height: 100vh;
  background: var(--color-main-bg, #EED9C4);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 顶部工具栏 */
.top-toolbar {
  background: #fff;
  border-radius: 16px;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.breadcrumbs {
  display: flex;
  align-items: center;
  font-size: 14px;
  color: #8C7B70;
  gap: 8px;
}

.breadcrumbs .separator { color: #D4A373; }
.breadcrumbs .current { color: #804030; font-weight: 600; }

.toolbar-actions { display: flex; gap: 12px; }

.btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: transform 0.2s;
}

.btn:hover { transform: translateY(-2px); }
.btn-secondary { background: rgba(128, 64, 48, 0.1); color: #804030; }

/* 角色头部 */
.char-header {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  display: flex;
  align-items: flex-start;
  gap: 24px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
}

.char-avatar {
  width: 100px;
  height: 100px;
  border-radius: 16px;
  background: linear-gradient(135deg, #D4A373, #8C7B70);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
  color: #fff;
  flex-shrink: 0;
}

.char-info h1 {
  font-size: 32px;
  color: #2C1810;
  margin: 0 0 8px 0;
  font-weight: 700;
}

.char-title {
  font-size: 16px;
  color: #8C7B70;
  margin: 0 0 12px 0;
}

.char-meta { display: flex; gap: 8px; flex-wrap: wrap; }

.meta-badge {
  background: rgba(128, 64, 48, 0.1);
  color: #804030;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
}

/* 面板 */
.panel {
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 4px 20px rgba(75, 54, 33, 0.05);
  overflow: hidden;
}

.panel-header {
  height: 60px;
  padding: 0 24px;
  border-bottom: 1px solid #E8DCCF;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #FFFCF9;
}

.panel-title {
  font-size: 16px;
  color: #2C1810;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-title i { color: #804030; }

.panel-body { padding: 24px; }

/* 表单网格 */
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.form-group { display: flex; flex-direction: column; gap: 6px; }

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: #8C7B70;
}

.form-value {
  font-size: 15px;
  color: #2C1810;
  padding: 12px 16px;
  background: #FFFCF9;
  border: 1px solid #E8DCCF;
  border-radius: 8px;
}

/* RP状态标签 */
.rp-badge {
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.rp-badge.ic { background: #e8f5e9; color: #2e7d32; }
.rp-badge.ooc { background: #fff3e0; color: #ed6c02; }

/* 模板标签 */
.template-badge {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  background: rgba(128, 64, 48, 0.1);
  color: #804030;
}

/* 空值占位样式 */
.empty {
  color: #C4B5A8 !important;
  font-style: italic;
}

/* 状态内容 */
.status-content p {
  font-size: 15px;
  color: #2C1810;
  line-height: 1.7;
  margin: 0;
  white-space: pre-wrap;
}

.status-ooc {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px dashed #E8DCCF;
}

.status-ooc label {
  font-size: 12px;
  color: #8C7B70;
  display: block;
  margin-bottom: 6px;
}

.status-ooc p {
  font-size: 14px;
  color: #8C7B70;
  margin: 0;
}

/* 关于文本 */
.about-text {
  font-size: 15px;
  color: #2C1810;
  line-height: 1.8;
  white-space: pre-wrap;
}

.about-blocks {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.about-block {
  padding: 16px;
  background: #FFFCF9;
  border: 1px solid #E8DCCF;
  border-radius: 10px;
}

.about-block p {
  margin: 0;
  font-size: 15px;
  color: #2C1810;
  line-height: 1.7;
  white-space: pre-wrap;
}

.about-sections {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.about-section h4 {
  font-size: 14px;
  color: #804030;
  margin: 0 0 8px 0;
  font-weight: 600;
}

.about-section p {
  margin: 0;
  font-size: 15px;
  color: #2C1810;
  line-height: 1.7;
  white-space: pre-wrap;
}

/* 性格特征 */
.traits-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.trait-item {
  padding: 14px 16px;
  background: #FFFCF9;
  border: 1px solid #E8DCCF;
  border-radius: 10px;
}

.trait-labels {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #8C7B70;
  margin-bottom: 8px;
}

.trait-bar {
  height: 8px;
  background: rgba(128, 64, 48, 0.1);
  border-radius: 8px;
  overflow: hidden;
}

.trait-fill {
  height: 100%;
  background: linear-gradient(90deg, #D4A373, #804030);
  border-radius: 8px;
}

/* 其他信息 */
.misc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.misc-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #FFFCF9;
  border: 1px solid #E8DCCF;
  border-radius: 8px;
}

.misc-item label {
  font-size: 13px;
  color: #8C7B70;
  font-weight: 600;
}

.misc-item span {
  font-size: 14px;
  color: #2C1810;
}

/* 加载状态 */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px;
  color: #8C7B70;
}

.loader {
  width: 40px;
  height: 40px;
  border: 3px solid #E8DCCF;
  border-top-color: #804030;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* 动画 */
.anim-item {
  opacity: 0;
  transform: translateY(20px);
}

.animate-in .anim-item {
  animation: fadeUp 0.4s ease forwards;
  animation-delay: calc(var(--delay) * 0.08s);
}

@keyframes fadeUp {
  to { opacity: 1; transform: translateY(0); }
}

/* 响应式 */
@media (max-width: 768px) {
  .form-grid { grid-template-columns: 1fr; }
  .char-header { flex-direction: column; align-items: center; text-align: center; }
  .char-meta { justify-content: center; }
}
</style>
