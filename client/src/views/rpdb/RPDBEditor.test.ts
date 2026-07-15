import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createPinia } from 'pinia'
import RPDBEditor from './RPDBEditor.vue'
import { createRPDBDraft } from '@/api/rpdb'
import { uploadImage } from '@/api/item'
import i18n from '@/i18n'

enableAutoUnmount(afterEach)

const getPresetTags = vi.hoisted(() => vi.fn())

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return {
    ...actual,
    listRPDBDrafts: vi.fn().mockResolvedValue({ drafts: [] }),
    createRPDBDraft: vi.fn().mockImplementation(async (payload) => ({
      draft: {
        id: 999,
        author_id: 1,
        type: payload?.type || 'item_showcase',
        title: payload?.title || '',
        payload: payload || {},
        base_version: 0,
        status: 'active',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    })),
    updateRPDBDraft: vi.fn().mockImplementation(async (id, payload) => ({
      draft: {
        id,
        author_id: 1,
        type: payload?.type || 'item_showcase',
        title: payload?.title || '',
        payload: payload || {},
        base_version: 0,
        status: 'active',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    })),
    getRPDBDraft: vi.fn(),
    deleteRPDBDraft: vi.fn(),
    publishRPDBDraft: vi.fn().mockResolvedValue({ work: { id: 999 } }),
  }
})

vi.mock('@/api/tag', () => ({
  getPresetTags,
}))

vi.mock('@/api/item', () => ({
  uploadImage: vi.fn(),
}))

vi.mock('@/api/guild', () => ({
  listGuilds: vi.fn().mockResolvedValue({
    guilds: [
      { id: 7, name: '银月议会', my_role: 'member' },
      { id: 8, name: '远行者公会', my_role: 'admin' },
    ],
  }),
}))

function mountEditor() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'rpdb-create', component: RPDBEditor },
      { path: '/drafts/:draftId/edit', name: 'rpdb-draft-edit', component: RPDBEditor },
    ],
  })
  return router.push('/').then(() => mount(RPDBEditor, {
    global: {
      plugins: [createPinia(), router, i18n],
      stubs: {
        TiptapEditor: {
          props: ['modelValue'],
          template: '<div data-testid="rpdb-rich-editor">富文本编辑器<slot name="toolbar" /></div>',
        },
        PostQuickJump: {
          props: ['modelValue', 'onInsert'],
          template: `<div v-if="modelValue" data-testid="post-quick-jump"><button type="button" @click="onInsert('<p>内部链接卡片</p>')">插入内部链接</button></div>`,
        },
        RPDBMediaGallery: {
          props: ['cover', 'media', 'title'],
          template: '<div data-testid="rpdb-preview-gallery">预览图相册</div>',
        },
        ImageViewer: {
          props: ['modelValue', 'images', 'startIndex'],
          template: '<div v-if="modelValue" data-testid="image-viewer"></div>',
        },
      },
    },
  }))
}

describe('RPDBEditor', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
    localStorage.clear()
    sessionStorage.clear()
    vi.clearAllMocks()
    getPresetTags.mockResolvedValue({
      tags: [
        { id: 105, name: '炼金工坊风格', color: '6F8F46', type: 'preset', is_public: true, usage_count: 0, creator_id: 1, created_at: '', updated_at: '' },
        { id: 104, name: '洛丹伦风格', color: '6E6A85', type: 'preset', is_public: true, usage_count: 0, creator_id: 1, created_at: '', updated_at: '' },
        { id: 102, name: '部落风格', color: 'B83030', type: 'preset', is_public: true, usage_count: 0, creator_id: 1, created_at: '', updated_at: '' },
        { id: 101, name: '联盟风格', color: '2F66C8', type: 'preset', is_public: true, usage_count: 0, creator_id: 1, created_at: '', updated_at: '' },
        { id: 103, name: '库尔提拉斯风格', color: '356A8A', type: 'preset', is_public: true, usage_count: 0, creator_id: 1, created_at: '', updated_at: '' },
      ],
    })
  })

  it('renders an upper publishing setup and lower writing workspace', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.find('[data-testid="editor-upper"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="editor-lower"]').exists()).toBe(true)
    expect(wrapper.find('.minimal-editor-shell').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-rich-editor"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('帖子编辑')
    expect(wrapper.text()).not.toContain('新媒体正文与攻略编辑')
    expect(wrapper.text()).not.toContain('发布检查')
    expect(wrapper.text()).toContain('保存草稿')
    expect(wrapper.find('[data-testid="rpdb-draft-box-button"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('游戏版本')
    expect(wrapper.text()).not.toContain('资料片')
    expect(wrapper.text()).not.toContain('获取难度')
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('[data-testid="rpdb-custom-select"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="floating-submit-toolbar"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="publish-work"]').text()).toContain('发布')
    expect(wrapper.find('[data-testid="rpdb-authoring-steps"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="rpdb-internal-link-button"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('内部链接')
  })

  it('opens the visible draft box and keeps saved drafts separate from formal works', async () => {
    const wrapper = await mountEditor()
    await wrapper.find('#rpdb-title').setValue('未发布的独立草稿')
    await wrapper.find('[data-testid="save-rpdb-draft"]').trigger('click')

    await vi.waitFor(() => expect(vi.mocked(createRPDBDraft)).toHaveBeenCalled())
    await wrapper.find('[data-testid="rpdb-draft-box-button"]').trigger('click')

    const draftBox = wrapper.find('[data-testid="rpdb-draft-box"]')
    expect(draftBox.exists()).toBe(true)
    expect(draftBox.text()).toContain('未发布的独立草稿')
    expect(draftBox.text()).toContain('新内容')
  })

  it('places required visibility above content type and defaults to public', async () => {
    const wrapper = await mountEditor()
    const visibilityField = wrapper.find('[data-testid="visibility-field"]')
    const typeCards = wrapper.find('.type-cards')

    expect(visibilityField.exists()).toBe(true)
    expect(visibilityField.text()).toContain('可见度')
    expect(visibilityField.text()).toContain('公开可见')
    expect(visibilityField.find('.required-mark').exists()).toBe(true)
    expect(visibilityField.element.compareDocumentPosition(typeCards.element)).toBe(Node.DOCUMENT_POSITION_FOLLOWING)

    await visibilityField.find('.rpdb-select__trigger').trigger('click')
    await wrapper.vm.$nextTick()
    const labels = Array.from(document.body.querySelectorAll('.rpdb-select__option')).map(option => option.textContent)
    expect(labels.some(label => label?.includes('公开可见'))).toBe(true)
    expect(labels.some(label => label?.includes('公会可见'))).toBe(true)
    expect(labels.some(label => label?.includes('仅自己'))).toBe(true)
  })

  it('submits multiple guilds selected for guild visibility', async () => {
    localStorage.setItem('token', 'test-token')
    const wrapper = await mountEditor()
    const visibilityField = wrapper.find('[data-testid="visibility-field"]')

    await visibilityField.find('.rpdb-select__trigger').trigger('click')
    await wrapper.vm.$nextTick()
    const guildOption = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.rpdb-select__option'))
      .find(option => option.textContent?.includes('公会可见'))
    expect(guildOption).toBeDefined()
    guildOption!.click()
    await wrapper.vm.$nextTick()

    const guildSelect = wrapper.find('[data-testid="visibility-guild-select"]')
    expect(guildSelect.text()).toContain('可多选')
    const guildCheckboxes = guildSelect.findAll('input[type="checkbox"]')
    expect(guildCheckboxes).toHaveLength(2)
    await guildCheckboxes[0].setValue(true)
    await guildCheckboxes[1].setValue(true)
    await wrapper.find('input[placeholder="例如：月光灯笼的巡夜用法"]').setValue('银月议会收藏')
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    await vi.waitFor(() => expect(vi.mocked(createRPDBDraft)).toHaveBeenCalled())
    expect(vi.mocked(createRPDBDraft).mock.calls.at(-1)?.[0]).toMatchObject({
      visibility: 'guild',
      guild_id: 7,
      guild_ids: [7, 8],
      is_public: false,
    })
  })

  it('validates the required title inline without an editor navigation bar', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.find('[data-testid="rpdb-authoring-steps"]').exists()).toBe(false)
    expect(wrapper.find('#rpdb-title').attributes('required')).toBeDefined()

    vi.mocked(createRPDBDraft).mockClear()
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    expect(wrapper.find('.field-control').classes()).toContain('invalid')
    expect(wrapper.text()).toContain('请填写内容标题后再发布')
    expect(createRPDBDraft).not.toHaveBeenCalled()

    await wrapper.find('#rpdb-title').setValue('暮色森林巡林灯')
    expect(wrapper.find('.field-control').classes()).not.toContain('invalid')
  })

  it('shows preview gallery only after preview upload and lets cover be removed', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.find('[data-testid="rpdb-media-strip"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="cover-upload"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preview-upload"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preview-gallery"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="rpdb-preview-gallery"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('封面图')
    expect(wrapper.text()).toContain('预览图')
    expect(wrapper.find('.generated-cover').exists()).toBe(false)
    expect(wrapper.text()).toContain('可不填，发布后会根据标题自动生成默认封面')

    const coverInput = wrapper.find('[data-testid="cover-upload"] input')
    const coverClick = vi.spyOn(coverInput.element, 'click')
    await wrapper.find('[data-testid="cover-upload"]').trigger('click')
    expect(coverClick).toHaveBeenCalledOnce()
    coverClick.mockRestore()

    vi.mocked(uploadImage).mockReset()
    vi.mocked(uploadImage)
      .mockResolvedValueOnce({ url: '/uploads/rpdb/test-cover.jpg' })
      .mockResolvedValueOnce({ url: '/uploads/rpdb/test-preview.jpg' })
      .mockResolvedValueOnce({ url: '/uploads/rpdb/test-preview-2.jpg' })

    const files = [new File(['x'], 'rpdb.png', { type: 'image/png' })]
    Object.defineProperty(coverInput.element, 'files', { value: files, configurable: true })
    await coverInput.trigger('change')

    expect(wrapper.find('[data-testid="cover-remove"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="cover-upload"] img').attributes('src')).toContain('/uploads/rpdb/test-cover.jpg')

    await wrapper.find('[data-testid="cover-remove"]').trigger('click')
    expect(wrapper.find('[data-testid="cover-remove"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="cover-upload"] img').exists()).toBe(false)

    const previewFiles = [
      new File(['x'], 'rpdb-preview-1.png', { type: 'image/png' }),
      new File(['y'], 'rpdb-preview-2.png', { type: 'image/png' }),
    ]
    const previewInput = wrapper.find('[data-testid="preview-upload"] input')
    expect(previewInput.attributes('multiple')).toBeDefined()
    Object.defineProperty(previewInput.element, 'files', { value: previewFiles, configurable: true })
    await previewInput.trigger('change')

    expect(uploadImage).toHaveBeenCalledTimes(3)
    expect(wrapper.find('[data-testid="preview-gallery"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-preview-gallery"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('预览图相册')
  })

  it('opens the reused RPBox internal link picker from the post editor toolbar', async () => {
    const wrapper = await mountEditor()

    await wrapper.find('[data-testid="rpdb-internal-link-button"]').trigger('click')

    expect(wrapper.find('[data-testid="post-quick-jump"]').exists()).toBe(true)
  })

  it('offers item, transmog and housing while keeping guides inside item and transmog', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.findAll('.type-cards button')).toHaveLength(3)
    expect(wrapper.find('[data-testid="item-editor-fields"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="home-editor-fields"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('获取攻略')
    expect(wrapper.text()).toContain('添加攻略步骤')
    expect(wrapper.find('[data-testid="add-guide-step-bottom"]').exists()).toBe(true)

    await wrapper.findAll('.type-cards button')[1].trigger('click')
    expect(wrapper.find('[data-testid="item-editor-fields"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('护甲类型')
    expect(wrapper.text()).toContain('幻化部位')
    expect(wrapper.text()).toContain('幻化分享代码')
    expect(wrapper.find('[data-testid="transmog-share-code-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).not.toContain('外观主题')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).not.toContain('主体来源')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).not.toContain('套装链接')
    expect(wrapper.find('[data-testid="add-guide-step-bottom"]').exists()).toBe(true)

    await wrapper.findAll('.type-cards button')[2].trigger('click')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').exists()).toBe(false)
    const homeFields = wrapper.find('[data-testid="home-editor-fields"]')
    expect(homeFields.exists()).toBe(true)
    expect(homeFields.text()).not.toContain('服务器')
    expect(homeFields.text()).not.toContain('所在区域')
    expect(homeFields.text()).not.toContain('布置风格')
    expect(wrapper.text()).toContain('家宅资料')
    expect(wrapper.text()).toContain('住宅分享代码')
    expect(wrapper.text()).toContain('加好友后参观')
    expect(wrapper.text()).toContain('室内外')
    expect(wrapper.text()).not.toContain('开放参观')
    expect(wrapper.text()).not.toContain('混合空间')
    expect(wrapper.text()).not.toContain('可获取 / 可参观')
    expect(wrapper.find('[data-testid="home-share-code-input"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('添加攻略步骤')
    expect(wrapper.text()).not.toContain('RP 用途')
    expect(wrapper.text()).not.toContain('空间亮点')
    expect(wrapper.text()).not.toContain('家宅代码预览')
    expect(wrapper.find('[data-testid="add-guide-step-bottom"]').exists()).toBe(false)
  })

  it('uses type-specific content checklists for item, transmog and housing', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.find('[data-testid="rpdb-content-checklist"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('内容清单')
    expect(wrapper.text()).not.toContain('效果媒体')
    expect(wrapper.find('[data-testid="item-content-checklist"]').exists()).toBe(true)
    const itemPanels = wrapper.findAll('[data-testid="item-reference-panel"]')
    expect(itemPanels.length).toBeGreaterThan(0)
    expect(itemPanels[0].attributes('open')).toBeUndefined()
    expect(itemPanels[0].find('summary').text()).toContain('物品')
    await itemPanels[0].find('summary').trigger('click')
    expect(itemPanels[0].attributes('open')).toBeDefined()
    expect(itemPanels[0].find('.slot-row__body').isVisible()).toBe(true)
    expect(itemPanels[0].text()).toContain('物品名称')
    expect(itemPanels[0].text()).toContain('物品描述')
    expect(itemPanels[0].text()).toContain('物品类型')
    expect(itemPanels[0].text()).toContain('物品来源')
    expect(itemPanels[0].text()).not.toContain('Wowhead 地址')
    expect(itemPanels[0].text()).not.toContain('道具 ID')
    expect(itemPanels[0].text()).not.toContain('图标')
    expect(itemPanels[0].text()).not.toContain('品质')
    expect(wrapper.find('[data-testid="item-reference-more-options"]').exists()).toBe(false)

    await wrapper.findAll('.type-cards button')[1].trigger('click')
    expect(wrapper.find('[data-testid="transmog-content-checklist"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('头部')
    expect(wrapper.text()).toContain('主手')
    expect(wrapper.text()).toContain('槽位状态')
    const transmogSlotPanels = wrapper.findAll('[data-testid="transmog-slot-panel"]')
    expect(transmogSlotPanels.length).toBeGreaterThan(0)
    expect(transmogSlotPanels[0].attributes('open')).toBeUndefined()
    expect(transmogSlotPanels[0].find('summary').text()).toContain('头部')
    await transmogSlotPanels[0].find('summary').trigger('click')
    expect(transmogSlotPanels[0].attributes('open')).toBeDefined()
    expect(transmogSlotPanels[0].find('.slot-row__body').isVisible()).toBe(true)
    const transmogExtraOptions = wrapper.find('[data-testid="transmog-slot-more-options"]')
    expect(transmogExtraOptions.exists()).toBe(true)
    expect(transmogExtraOptions.attributes('open')).toBeUndefined()
    expect(transmogExtraOptions.text()).toContain('其他选项')
    expect(transmogExtraOptions.text()).toContain('Wowhead 地址')
    expect(transmogExtraOptions.text()).toContain('替代件')

    await wrapper.findAll('.type-cards button')[2].trigger('click')
    const homeChecklist = wrapper.find('[data-testid="home-content-checklist"]')
    expect(homeChecklist.exists()).toBe(true)
    const furniturePanels = wrapper.findAll('[data-testid="furniture-reference-panel"]')
    expect(furniturePanels.length).toBeGreaterThan(0)
    expect(furniturePanels[0].attributes('open')).toBeUndefined()
    expect(furniturePanels[0].find('summary').text()).toContain('家具 1')
    await furniturePanels[0].find('summary').trigger('click')
    expect(furniturePanels[0].attributes('open')).toBeDefined()
    expect(furniturePanels[0].find('.slot-row__body').isVisible()).toBe(true)
    expect(wrapper.text()).toContain('家具名称')
    expect(homeChecklist.text()).toContain('图标')
    expect(homeChecklist.text()).not.toContain('放置位置')
    expect(wrapper.text()).toContain('获取途径')
  })

  it('switches all editor-specific labels to English', async () => {
    i18n.global.locale.value = 'en-US'
    const wrapper = await mountEditor()

    expect(wrapper.text()).toContain('Publish content')
    expect(wrapper.text()).toContain('WoW Item')
    expect(wrapper.find('[data-testid="publish-work"]').text()).toContain('Publish')

    await wrapper.findAll('.type-cards button')[1].trigger('click')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).toContain('Armor type')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).toContain('Transmog share code')
    expect(wrapper.find('[data-testid="transmog-content-checklist"]').text()).toContain('Head')

    await wrapper.findAll('.type-cards button')[2].trigger('click')
    const furniturePanel = wrapper.find('[data-testid="furniture-reference-panel"]')
    expect(furniturePanel.find('summary').text()).toContain('Furniture 1')
    await furniturePanel.find('summary').trigger('click')
    expect(furniturePanel.text()).toContain('Furniture name')
    expect(furniturePanel.find('[data-testid="furniture-icon-url"]').attributes('placeholder')).toBe('Paste an image URL')
  })

  it('lets housing authors paste, upload and clear furniture icons', async () => {
    vi.mocked(uploadImage).mockReset()
    vi.mocked(uploadImage).mockResolvedValue({ url: '/uploads/rpdb/furniture-icon.png' })
    const wrapper = await mountEditor()

    await wrapper.findAll('.type-cards button')[2].trigger('click')
    const checklist = wrapper.find('[data-testid="home-content-checklist"]')
    await checklist.find('[data-testid="furniture-reference-panel"] summary').trigger('click')
    const iconURL = checklist.find('[data-testid="furniture-icon-url"]')
    expect(iconURL.attributes('placeholder')).toBe('粘贴图片链接')
    expect(checklist.text()).not.toContain('放置位置')

    await iconURL.setValue('https://example.com/furniture.png')
    expect((iconURL.element as HTMLInputElement).value).toBe('https://example.com/furniture.png')

    const uploadInput = checklist.find('[data-testid="furniture-icon-upload"]')
    const file = new File(['icon'], 'furniture.png', { type: 'image/png' })
    Object.defineProperty(uploadInput.element, 'files', { value: [file], configurable: true })
    await uploadInput.trigger('change')

    await vi.waitFor(() => expect(uploadImage).toHaveBeenCalledWith(file))
    await vi.waitFor(() => expect((iconURL.element as HTMLInputElement).value).toBe('/uploads/rpdb/furniture-icon.png'))
    expect(checklist.find('button[aria-label="清除图标"]').exists()).toBe(true)

    await checklist.find('button[aria-label="清除图标"]').trigger('click')
    expect((iconURL.element as HTMLInputElement).value).toBe('')
  })

  it('keeps usage restrictions in the main type form instead of the side inspector', async () => {
    const wrapper = await mountEditor()

    expect(wrapper.find('[data-testid="item-editor-fields"]').text()).toContain('是否绑定')
    expect(wrapper.find('[data-testid="item-editor-fields"]').text()).toContain('阵营')
    expect(wrapper.find('.content-inspector').text()).not.toContain('适用限制')

    await wrapper.findAll('.type-cards button')[1].trigger('click')

    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).toContain('护甲类型')
    expect(wrapper.find('[data-testid="transmog-editor-fields"]').text()).toContain('阵营')
    expect(wrapper.find('.content-inspector').text()).not.toContain('适用限制')
  })

  it('uses the requested item fields and generates the hidden reference id on submit', async () => {
    const wrapper = await mountEditor()
    const fields = wrapper.find('[data-testid="item-editor-fields"]')

    expect(fields.text()).toContain('物品名称')
    expect(fields.text()).toContain('物品描述')
    expect(fields.text()).toContain('物品类型')
    expect(fields.text()).toContain('物品来源')
    expect(fields.text()).toContain('阵营')
    expect(fields.text()).toContain('是否绑定')
    expect(fields.text()).toContain('物品')
    expect(fields.text()).toContain('否')
    expect(fields.text()).not.toContain('来源类型')
    expect(fields.text()).not.toContain('资料链接')
    expect(fields.text()).not.toContain('道具 ID')

    await wrapper.find('input[placeholder="例如：月光灯笼的巡夜用法"]').setValue('暮色森林巡林灯')
    await fields.find('input[placeholder="例如：月光灯笼"]').setValue('巡林灯')
    await fields.find('textarea[placeholder="描述外观、效果或 RP 使用方式"]').setValue('夜间巡逻使用的暖色提灯。')
    await fields.find('input[placeholder="例如：任务奖励、商人购买或公会活动产出"]').setValue('公会巡夜活动')
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    await vi.waitFor(() => expect(vi.mocked(createRPDBDraft)).toHaveBeenCalled())
    expect(vi.mocked(createRPDBDraft).mock.calls.at(-1)?.[0]).toMatchObject({
      bind_type: 'no',
      faction: 'neutral',
      visibility: 'public',
      is_public: true,
      references: [{
        external_type: 'item',
        external_id: 'rpbox-1',
        name: '巡林灯',
        description: '夜间巡逻使用的暖色提灯。',
        acquisition_method: '公会巡夜活动',
      }],
    })
  })

  it('imports ordered TomTom /ttpaste routes into guide steps', async () => {
    const wrapper = await mountEditor()

    await wrapper.find('[data-testid="tomtom-import-input"]').setValue([
      '/way #47 73.80 44.50 夜色镇集合',
      '/tway 暮色森林 68.20 51.40 林地入口',
    ].join('\n'))
    await wrapper.find('[data-testid="tomtom-import-button"]').trigger('click')

    expect(wrapper.text()).toContain('夜色镇集合')
    expect(wrapper.text()).toContain('林地入口')
    expect(wrapper.text()).toContain('/way #47 73.80 44.50 [1/2] 夜色镇集合')
    expect(wrapper.text()).toContain('/way 暮色森林 68.20 51.40 [2/2] 林地入口')
  })

  it('uses selectable RP style presets instead of manual tag IDs', async () => {
    const wrapper = await mountEditor()

    await vi.waitFor(() => expect(wrapper.text()).toContain('联盟风格'))

    expect(wrapper.text()).toContain('RP 风格标签')
    expect(wrapper.find('[data-testid="rpdb-topic-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rpdb-style-candidates"]').element.compareDocumentPosition(wrapper.find('[data-testid="rpdb-topic-custom"]').element)).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
    expect(wrapper.text()).toContain('库尔提拉斯风格')
    expect(wrapper.text()).toContain('洛丹伦风格')
    expect(wrapper.text()).not.toContain('标签 ID')

    const styleButtons = wrapper.findAll('[data-testid="rpdb-style-option"]')
    expect(styleButtons.slice(0, 5).map(button => button.text())).toEqual([
      '#联盟风格',
      '#部落风格',
      '#库尔提拉斯风格',
      '#洛丹伦风格',
      '#炼金工坊风格',
    ])
    await styleButtons[0].trigger('click')
    await styleButtons[2].trigger('click')

    const selectedTopics = wrapper.find('[data-testid="rpdb-selected-topics"]')
    expect(selectedTopics.text()).toContain('#联盟风格')
    expect(selectedTopics.text()).toContain('#库尔提拉斯风格')
    expect(wrapper.findAll('[data-testid="rpdb-selected-topic"]')).toHaveLength(2)

    await wrapper.findAll('[data-testid="remove-rpdb-style-tag"]')[0].trigger('click')
    expect(wrapper.find('[data-testid="rpdb-selected-topics"]').text()).not.toContain('#联盟风格')
  })

  it('keeps custom RP topic tags local until review submission', async () => {
    const wrapper = await mountEditor()

    await wrapper.find('[data-testid="rpdb-topic-input"]').setValue('#暮色森林风格')
    await wrapper.find('[data-testid="rpdb-topic-input"]').trigger('keydown.enter')

    expect(wrapper.find('[data-testid="rpdb-selected-topics"]').text()).toContain('#暮色森林风格')
    expect(wrapper.findAll('[data-testid="rpdb-style-option"]').map(button => button.text())).not.toContain('#暮色森林风格')

    await wrapper.find('input[placeholder="例如：月光灯笼的巡夜用法"]').setValue('暮色森林巡林灯')
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    await vi.waitFor(() => {
      expect(vi.mocked(createRPDBDraft)).toHaveBeenCalled()
    })
    const customTopicSubmission = vi.mocked(createRPDBDraft).mock.calls.find(([payload]) => payload?.tag_names?.includes('暮色森林风格'))
    expect(customTopicSubmission?.[0]).toMatchObject({
      tag_names: ['暮色森林风格'],
    })
    expect(customTopicSubmission?.[0]).not.toHaveProperty('game_version')
    expect(customTopicSubmission?.[0]).not.toHaveProperty('expansion')
  })

  it('keeps built-in style tags selectable when the preset API is unavailable', async () => {
    getPresetTags.mockRejectedValueOnce(new Error('network unavailable'))
    const wrapper = await mountEditor()

    await vi.waitFor(() => expect(wrapper.text()).toContain('联盟风格'))
    expect(wrapper.text()).not.toContain('正在加载系统风格标签')

    await wrapper.findAll('[data-testid="rpdb-style-option"]')[0].trigger('click')
    expect(wrapper.find('[data-testid="rpdb-selected-topics"]').text()).toContain('#联盟风格')

    await wrapper.find('input[placeholder="例如：月光灯笼的巡夜用法"]').setValue('线上风格标签回退测试')
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    await vi.waitFor(() => {
      const fallbackSubmission = vi.mocked(createRPDBDraft).mock.calls.find(([payload]) => payload?.tag_names?.includes('联盟风格'))
      expect(fallbackSubmission?.[0]).toMatchObject({ tag_names: ['联盟风格'] })
    })
  })

  it('publishes the pasted transmog share code with the post', async () => {
    const wrapper = await mountEditor()

    await wrapper.findAll('.type-cards button')[1].trigger('click')
    await wrapper.find('#rpdb-title').setValue('银月秘法使')
    await wrapper.find('[data-testid="transmog-share-code-input"]').setValue('TRANSMOG:HEAD=34339;CHEST=34202')
    await wrapper.find('[data-testid="publish-work"]').trigger('click')

    await vi.waitFor(() => expect(vi.mocked(createRPDBDraft)).toHaveBeenCalled())
    expect(vi.mocked(createRPDBDraft).mock.calls.at(-1)?.[0]).toMatchObject({
      type: 'transmog',
      extra: { share_code: 'TRANSMOG:HEAD=34339;CHEST=34202' },
    })
  })
})
