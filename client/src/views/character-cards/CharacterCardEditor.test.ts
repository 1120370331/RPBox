import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import CharacterCardEditor from './CharacterCardEditor.vue'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  addCharacterCardPortrait: vi.fn(),
  deleteCharacterCardPortrait: vi.fn(),
  getCharacterCard: vi.fn(),
  getCharacterCardTRP3Lua: vi.fn(),
  listAccountBackups: vi.fn(),
  publishCharacterCard: vi.fn(),
  reorderCharacterCardPortraits: vi.fn(),
  setCharacterCardPortraitCover: vi.fn(),
  updateCharacterCard: vi.fn(),
  syncCharacterCardFromTRP3: vi.fn(),
  uploadCharacterCardImpressionImage: vi.fn(),
  uploadCharacterCardPortrait: vi.fn(),
  writeBackCharacterCardToTRP3: vi.fn(),
  confirm: vi.fn(),
  invoke: vi.fn(),
  toastError: vi.fn(),
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
}))

vi.mock('@/api/characterCard', () => ({
  addCharacterCardPortrait: mocks.addCharacterCardPortrait,
  deleteCharacterCardPortrait: mocks.deleteCharacterCardPortrait,
  getCharacterCard: mocks.getCharacterCard,
  getCharacterCardTRP3Lua: mocks.getCharacterCardTRP3Lua,
  publishCharacterCard: mocks.publishCharacterCard,
  reorderCharacterCardPortraits: mocks.reorderCharacterCardPortraits,
  setCharacterCardPortraitCover: mocks.setCharacterCardPortraitCover,
  updateCharacterCard: mocks.updateCharacterCard,
  syncCharacterCardFromTRP3: mocks.syncCharacterCardFromTRP3,
  uploadCharacterCardImpressionImage: mocks.uploadCharacterCardImpressionImage,
  uploadCharacterCardPortrait: mocks.uploadCharacterCardPortrait,
  writeBackCharacterCardToTRP3: mocks.writeBackCharacterCardToTRP3,
  getCharacterCardPortraitUrl: vi.fn(() => ''),
}))

vi.mock('@/api/accountBackup', () => ({
  listAccountBackups: mocks.listAccountBackups,
}))

vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ confirm: mocks.confirm }),
}))

vi.mock('@/stores/toast', () => ({
  useToastStore: () => ({
    error: mocks.toastError,
    success: mocks.toastSuccess,
    warning: mocks.toastWarning,
  }),
}))

const editorStub = defineComponent({
  name: 'TiptapEditor',
  inheritAttrs: false,
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, expose }) {
    expose({ insertContent: (html: string) => emit('update:modelValue', `${props.modelValue}${html}`) })
    return () => h('textarea', {
      class: 'editor-stub',
      placeholder: props.placeholder,
      value: props.modelValue,
      onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
    })
  },
})

const cropperStub = defineComponent({
  name: 'ImageCropperDialog',
  props: { modelValue: { type: Boolean, default: false } },
  emits: ['update:modelValue', 'cropped', 'error'],
  setup(_props, { emit }) {
    return () => h('button', {
      class: 'cropper-stub',
      onClick: () => emit('cropped', new File(['portrait'], 'portrait.webp', { type: 'image/webp' })),
    }, 'emit cropped portrait')
  },
})

const modalStub = defineComponent({
  name: 'RModal',
  props: { modelValue: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => props.modelValue
      ? h('div', { class: 'modal-stub' }, slots.default?.())
      : null
  },
})

const galleryImageStub = defineComponent({
  name: 'CharacterCardGalleryImage',
  props: {
    portrait: { type: Object, default: null },
    alt: { type: String, default: '' },
  },
  setup(props) {
    return () => h('img', {
      src: (props.portrait as { image_url?: string } | null)?.image_url || '',
      alt: props.alt,
    })
  },
})

const card: CharacterCard = {
  id: 12,
  user_id: 3,
  first_name: '伊莉娅',
  last_name: '星语',
  display_name: '伊莉娅·星语',
  title: '月之女祭司',
  full_title: '',
  race: '暗夜精灵',
  class: '牧师',
  eye_color: '银色',
  eye_color_hex: '#C9D5E7',
  age: '',
  height: '',
  weight: '',
  birthplace: '',
  residence: '',
  relationship_status: '',
  icon: '',
  name_color: '',
  summary: '',
  background_story: '<p>旧背景</p>',
  first_impression: '<p>旧印象</p>',
  impressions: [
    {
      slot: 1,
      active: true,
      title: '淡淡的草药香',
      text: '靠近时能闻到晒干草叶的气息。',
      trp3_icon: 'INV_Misc_Herb_19',
      icon_image_url: '',
      icon_image_updated_at: null,
      image_url: '',
      image_updated_at: null,
    },
    ...Array.from({ length: 4 }, (_, index) => ({
      slot: index + 2,
      active: false,
      title: '',
      text: '',
      trp3_icon: '',
      icon_image_url: '',
      icon_image_updated_at: null,
      image_url: '',
      image_updated_at: null,
    })),
  ],
  other_content: '<p>旧资料</p>',
  portrait_image_url: '',
  status: 'draft',
  visibility: 'private',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

beforeEach(() => {
  i18n.global.locale.value = 'zh-CN'
  localStorage.setItem('user', JSON.stringify({ id: 3, username: '人物卡所有者' }))
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.clearAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
  delete (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__
  document.body.innerHTML = ''
})

describe('CharacterCardEditor tabs', () => {
  it('keeps the four color controls aligned in two columns without helper captions', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const colorGrid = wrapper.get('.character-color-grid')
    expect(colorGrid.findAll('.character-dye')).toHaveLength(3)
    expect(colorGrid.findAll('.character-dye__hint')).toHaveLength(0)
    expect(colorGrid.text()).not.toContain('承接 TRP3')
    expect(colorGrid.text()).not.toContain('修改任一项会同步')
    wrapper.unmount()
  })

  it('automatically saves the complete card to cloud after a short editing pause', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
      updated_at: '2026-08-10T09:00:00Z',
    }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()
    vi.useFakeTimers()

    await wrapper.findAll<HTMLInputElement>('#character-panel-basic input')[0].setValue('自动保存的伊莉娅')
    expect(wrapper.get('.save-sync').text()).toContain('等待自动保存')
    expect(mocks.updateCharacterCard).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1199)
    expect(mocks.updateCharacterCard).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledTimes(1)
    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      first_name: '自动保存的伊莉娅',
      background_story: '<p>旧背景</p>',
      impressions: expect.arrayContaining([expect.objectContaining({ slot: 1 })]),
    }))
    expect(wrapper.get('.save-sync').text()).toContain('已自动保存到云端')
    expect(mocks.publishCharacterCard).not.toHaveBeenCalled()
    expect(mocks.toastSuccess).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('submits only after the explicit publish button saves the latest working copy', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
      updated_at: '2026-08-10T09:00:00Z',
    }))
    mocks.publishCharacterCard.mockResolvedValue({
      ...card,
      status: 'published',
      visibility: 'public',
      review_status: 'pending',
      updated_at: '2026-08-10T09:00:00Z',
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/character-cards/:id/edit', component: CharacterCardEditor },
        { path: '/character-cards/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    await wrapper.findAll<HTMLInputElement>('#character-panel-basic input')[0].setValue('点击发布的伊莉娅')
    const publishButton = wrapper.findAll('button').find((button) => button.text().includes('发布并提交审核'))
    expect(publishButton).toBeTruthy()
    await publishButton!.trigger('click')
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      first_name: '点击发布的伊莉娅',
      status: 'published',
      visibility: 'public',
    }))
    expect(mocks.publishCharacterCard).toHaveBeenCalledWith(12)
    expect(mocks.updateCharacterCard.mock.invocationCallOrder[0])
      .toBeLessThan(mocks.publishCharacterCard.mock.invocationCallOrder[0])
    expect(mocks.toastSuccess).toHaveBeenCalledWith(expect.stringContaining('最后一次点击发布'), 6000)
    wrapper.unmount()
  })

  it('queues edits made during an active cloud save without overwriting the newer form', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    let resolveFirstSave: ((saved: CharacterCard) => void) | undefined
    let firstPayload: Partial<CharacterCard> | undefined
    mocks.updateCharacterCard
      .mockImplementationOnce((_id: number, payload: Partial<CharacterCard>) => {
        firstPayload = payload
        return new Promise<CharacterCard>((resolve) => {
          resolveFirstSave = resolve
        })
      })
      .mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
        ...card,
        ...payload,
        updated_at: '2026-08-10T10:00:00Z',
      }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()
    vi.useFakeTimers()

    const nameFields = wrapper.findAll<HTMLInputElement>('#character-panel-basic input')
    await nameFields[0].setValue('第一次提交的名字')
    await vi.advanceTimersByTimeAsync(1200)
    expect(mocks.updateCharacterCard).toHaveBeenCalledTimes(1)
    expect(wrapper.get('.save-sync').text()).toContain('正在同步到云端')

    await nameFields[1].setValue('请求期间新增的姓氏')
    resolveFirstSave?.({
      ...card,
      ...firstPayload,
      updated_at: '2026-08-10T09:00:00Z',
    } as CharacterCard)
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledTimes(2)
    expect(mocks.updateCharacterCard.mock.calls[1][1]).toEqual(expect.objectContaining({
      first_name: '第一次提交的名字',
      last_name: '请求期间新增的姓氏',
    }))
    expect(nameFields[0].element.value).toBe('第一次提交的名字')
    expect(nameFields[1].element.value).toBe('请求期间新增的姓氏')
    expect(wrapper.get('.save-sync').text()).toContain('已自动保存到云端')
    wrapper.unmount()
  })

  it('does not render editing controls for a public card owned by another user', async () => {
    localStorage.setItem('user', JSON.stringify({ id: 99, username: '访客' }))
    mocks.getCharacterCard.mockResolvedValue(card)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('没有编辑权限')
    expect(wrapper.text()).not.toContain('伊莉娅·星语')
    expect(wrapper.find('.editor-header').exists()).toBe(false)
    expect(wrapper.find('.editor-layout').exists()).toBe(false)
    expect(wrapper.find('form, input, textarea, [contenteditable="true"]').exists()).toBe(false)
    expect(mocks.updateCharacterCard).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps the active ARIA tab and keyboard focus synchronized', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const tabs = wrapper.findAll<HTMLButtonElement>('.editor-tabs [role="tab"]')
    const basic = tabs[0]
    const background = tabs[1]
    const impression = tabs[2]
    const other = tabs[3]

    basic.element.focus()
    await basic.trigger('keydown', { key: 'ArrowRight' })
    await wrapper.vm.$nextTick()
    expect(background.attributes('aria-selected')).toBe('true')
    expect(background.attributes('tabindex')).toBe('0')
    expect(document.activeElement).toBe(background.element)

    await background.trigger('keydown', { key: 'End' })
    await wrapper.vm.$nextTick()
    expect(other.attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(other.element)

    await other.trigger('keydown', { key: 'ArrowDown' })
    await wrapper.vm.$nextTick()
    expect(basic.attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(basic.element)

    await basic.trigger('keydown', { key: 'ArrowUp' })
    await wrapper.vm.$nextTick()
    expect(other.attributes('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(other.element)

    await other.trigger('keydown', { key: 'Home' })
    await wrapper.vm.$nextTick()
    expect(basic.attributes('aria-selected')).toBe('true')
    expect(impression.attributes('tabindex')).toBe('-1')
    expect(document.activeElement).toBe(basic.element)
    wrapper.unmount()
  })

  it('keeps edits from every rich-text tab and saves the whole card at once', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
      updated_at: '2026-08-10T09:00:00Z',
    }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const tabByText = (text: string) => wrapper.findAll<HTMLButtonElement>('.editor-tabs button')
      .find((button) => button.text().includes(text))!

    await tabByText('背景故事').trigger('click')
    const background = wrapper.find<HTMLTextAreaElement>('textarea[placeholder="从角色最重要的一段过去开始…"]')
    await background.setValue('<p>未保存的新背景</p>')

    await tabByText('第一印象').trigger('click')
    const impression = wrapper.find<HTMLTextAreaElement>('textarea[placeholder^="补充不适合放进五条观察记录"]')
    await impression.setValue('<p>未保存的新印象</p>')

    await tabByText('背景故事').trigger('click')
    expect(background.element.value).toBe('<p>未保存的新背景</p>')

    const saveButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('保存整张人物卡'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      background_story: '<p>未保存的新背景</p>',
      first_impression: '<p>未保存的新印象</p>',
      impressions: expect.arrayContaining([expect.objectContaining({ slot: 1, title: '淡淡的草药香' })]),
      other_content: '<p>旧资料</p>',
    }))
    wrapper.unmount()
  })

  it('preserves unsaved RPBox-only content when refreshing TRP3 basic fields', async () => {
    const importedCard: CharacterCard = {
      ...card,
      source_backup_id: 7,
      source_account_id: 'WOW-ACCOUNT',
      source_profile_id: 'profile-7',
      summary: '服务端旧摘要',
    }
    const syncedCard: CharacterCard = {
      ...importedCard,
      first_name: '刷新后的名字',
      race: '高等精灵',
      impressions: importedCard.impressions.map((impression) => ({
        ...impression,
        title: impression.slot === 1 ? '服务端刷新值' : impression.title,
      })),
      updated_at: '2026-08-10T09:00:00Z',
    }
    mocks.getCharacterCard.mockResolvedValue(importedCard)
    mocks.syncCharacterCardFromTRP3.mockResolvedValue(syncedCard)
    mocks.confirm.mockResolvedValue(true)
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...syncedCard,
      ...payload,
    }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const summary = wrapper.get<HTMLTextAreaElement>('textarea[placeholder^="用几句话介绍"]')
    await summary.setValue('尚未保存的新摘要')
    const impressionTab = wrapper.findAll<HTMLButtonElement>('.editor-tabs button')
      .find((button) => button.text().includes('第一印象'))!
    await impressionTab.trigger('click')
    const impressionTitle = wrapper.get<HTMLInputElement>('input[placeholder^="例如：总是带着"]')
    await impressionTitle.setValue('尚未保存的观察')
    const refreshButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('从备份刷新基础信息'))!
    await refreshButton.trigger('click')
    await flushPromises()

    expect(mocks.syncCharacterCardFromTRP3).toHaveBeenCalledWith(12)
    expect(summary.element.value).toBe('尚未保存的新摘要')
    expect(impressionTitle.element.value).toBe('尚未保存的观察')
    expect(wrapper.findAll<HTMLInputElement>('.form-grid input')[0].element.value).toBe('刷新后的名字')

    const saveButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('保存整张人物卡'))!
    await saveButton.trigger('click')
    await flushPromises()
    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      first_name: '刷新后的名字',
      summary: '尚未保存的新摘要',
      impressions: expect.arrayContaining([expect.objectContaining({ slot: 1, title: '尚未保存的观察' })]),
    }))
    wrapper.unmount()
  })

  it('keeps exactly five slots and uploads a custom icon with a local preview', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.uploadCharacterCardImpressionImage.mockResolvedValue('character-card-impression-pending://icon-token')
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
      impressions: (payload.impressions || card.impressions).map((impression) => ({
        ...impression,
        icon_image_url: impression.slot === 1
          ? '/api/v1/images/character-card-impression-icon/12-1?v=saved'
          : impression.icon_image_url,
      })),
    }))
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:local-impression-icon')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      blob: vi.fn().mockResolvedValue(new Blob(['saved-icon'], { type: 'image/png' })),
    }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const impressionTab = wrapper.findAll<HTMLButtonElement>('.editor-tabs button')
      .find((button) => button.text().includes('第一印象'))!
    await impressionTab.trigger('click')
    const records = wrapper.findAll('.observation-record')
    expect(records).toHaveLength(5)

    const uploadButton = records[0].findAll<HTMLButtonElement>('.observation-media-actions button')
      .find((button) => button.text().includes('自定义图标'))!
    await uploadButton.trigger('click')
    const uploadInput = wrapper.get<HTMLInputElement>('.editor-panel--impression input[type="file"]')
    const file = new File(['icon'], 'impression-icon.png', { type: 'image/png' })
    Object.defineProperty(uploadInput.element, 'files', { value: [file], configurable: true })
    await uploadInput.trigger('change')
    await flushPromises()

    expect(createObjectURL).toHaveBeenCalledWith(file)
    expect(mocks.uploadCharacterCardImpressionImage).toHaveBeenCalledWith(file, 'icon')
    expect(records[0].get('.impression-mark__custom img').attributes('src')).toBe('blob:local-impression-icon')

    const saveButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('保存整张人物卡'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      impressions: expect.arrayContaining([
        expect.objectContaining({ slot: 1, icon_image_url: 'character-card-impression-pending://icon-token' }),
      ]),
    }))
    const updateCalls = mocks.updateCharacterCard.mock.calls
    const savedImpression = updateCalls[updateCalls.length - 1][1].impressions[0]
    expect(savedImpression).not.toHaveProperty('icon_image_updated_at')
    expect(savedImpression).not.toHaveProperty('image_updated_at')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:local-impression-icon')
    wrapper.unmount()
  })

  it('writes a blank card to a local profile without requiring or mutating a cloud backup', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
    }))
    mocks.listAccountBackups.mockResolvedValue([])
    mocks.getCharacterCardTRP3Lua.mockResolvedValue({
      profile_id: 'rpbox-12',
      profile: {
        profileName: '伊莉娅·星语',
        player: {
          characteristics: { FN: '伊莉娅', LN: '星语', EC: '银色' },
          misc: { PE: { 1: { TX: '仅由 Tauri 丢弃' } } },
        },
      },
      lua: 'TRP3_Profiles = { ["rpbox-12"] = {} }',
    })
    mocks.confirm.mockResolvedValue(true)
    mocks.invoke.mockImplementation(async (command: string) => {
      if (command === 'scan_profiles') {
        return { accounts: [{ account_id: 'WOW-ACCOUNT-B' }] }
      }
      return undefined
    })
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft\\_retail_\\WTF')
    ;(window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ = {}
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: modalStub,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    await wrapper.findAll<HTMLInputElement>('#character-panel-basic input')[0].setValue('已保存的伊莉娅')
    const writeBackButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('写回本地 TRP3'))!
    expect(writeBackButton.attributes('disabled')).toBeUndefined()
    await writeBackButton.trigger('click')
    await flushPromises()

    expect(mocks.invoke).toHaveBeenCalledWith('scan_profiles', {
      wowPath: 'C:\\Games\\World of Warcraft\\_retail_\\WTF',
    })
    expect(wrapper.text()).toContain('此账号暂无云端备份')
    const profileField = wrapper.findAll('label')
      .find((label) => label.text().includes('TRP3 profile ID'))!
      .get('input')
    await profileField.setValue('new-local-profile')
    const continueButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('继续写回'))!
    await continueButton.trigger('click')
    await flushPromises()

    expect(mocks.getCharacterCardTRP3Lua).toHaveBeenCalledWith(12)
    expect(mocks.invoke).toHaveBeenCalledWith('write_character_card_profile', expect.objectContaining({
      wowPath: 'C:\\Games\\World of Warcraft\\_retail_\\WTF',
      accountId: 'WOW-ACCOUNT-B',
      profileId: 'new-local-profile',
      profile: expect.objectContaining({ profileName: '伊莉娅·星语' }),
    }))
    expect(mocks.writeBackCharacterCardToTRP3).not.toHaveBeenCalled()
    const localWriteCallIndex = mocks.invoke.mock.calls.findIndex(([command]) => command === 'write_character_card_profile')
    expect(mocks.updateCharacterCard.mock.invocationCallOrder[0])
      .toBeLessThan(mocks.getCharacterCardTRP3Lua.mock.invocationCallOrder[0])
    expect(mocks.getCharacterCardTRP3Lua.mock.invocationCallOrder[0])
      .toBeLessThan(mocks.invoke.mock.invocationCallOrder[localWriteCallIndex])
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      expect.stringContaining('本次未修改云端账号备份'),
      7000,
    )
    wrapper.unmount()
  })

  it('keeps a successful local write when the optional cloud sync fails', async () => {
    const importedCard: CharacterCard = {
      ...card,
      source_backup_id: 7,
      source_account_id: 'WOW-ACCOUNT-B',
      source_profile_id: 'shared-profile',
    }
    mocks.getCharacterCard.mockResolvedValue(importedCard)
    mocks.listAccountBackups.mockResolvedValue([{ id: 7, account_id: 'WOW-ACCOUNT-B', version: 4 }])
    mocks.getCharacterCardTRP3Lua.mockResolvedValue({
      profile_id: 'shared-profile',
      profile: { profileName: '伊莉娅·星语', player: { characteristics: { FN: '伊莉娅' } } },
      lua: 'TRP3_Profiles = {}',
    })
    mocks.writeBackCharacterCardToTRP3.mockRejectedValue(new Error('云端暂时不可用'))
    mocks.confirm.mockResolvedValue(true)
    mocks.invoke.mockImplementation(async (command: string) => {
      if (command === 'scan_profiles') return { accounts: [{ account_id: 'WOW-ACCOUNT-B' }] }
      return { snapshot: { id: 'local-snapshot' } }
    })
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft\\_retail_\\WTF')
    ;(window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ = {}
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: true,
          RModal: modalStub,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    const writeBackButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('写回本地 TRP3'))!
    await writeBackButton.trigger('click')
    await flushPromises()

    const cloudOption = wrapper.get<HTMLInputElement>('.writeback-sheet__cloud-option input')
    expect(cloudOption.element.checked).toBe(true)
    expect(wrapper.text()).toContain('本地写入成功后，同步更新云端账号备份（可选）')
    const continueButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('继续写回'))!
    await continueButton.trigger('click')
    await flushPromises()

    const localWriteCallIndex = mocks.invoke.mock.calls.findIndex(([command]) => command === 'write_character_card_profile')
    expect(localWriteCallIndex).toBeGreaterThanOrEqual(0)
    expect(mocks.writeBackCharacterCardToTRP3).toHaveBeenCalledWith(12, expect.objectContaining({
      backup_id: 7,
      profile_id: 'shared-profile',
    }))
    expect(mocks.invoke.mock.invocationCallOrder[localWriteCallIndex])
      .toBeLessThan(mocks.writeBackCharacterCardToTRP3.mock.invocationCallOrder[0])
    expect(mocks.toastWarning).toHaveBeenCalledWith(
      '本地已成功、云端未更新：云端暂时不可用',
      8000,
    )
    expect(mocks.toastError).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('attaches a protected portrait immediately and revokes its temporary preview URL', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.uploadCharacterCardPortrait.mockResolvedValue('character-card-pending://portrait-token')
    mocks.addCharacterCardPortrait.mockResolvedValue({
      ...card,
      portrait_image_url: 'stored/character-card-portrait.webp',
      portrait_image_updated_at: '2026-08-10T10:00:00Z',
      portraits: [{
        id: 41,
        image_url: '/api/v1/images/character-card-portrait/41',
        image_updated_at: '2026-08-10T10:00:00Z',
        sort_order: 0,
        is_cover: true,
      }],
    })
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:local-portrait')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/character-cards/:id/edit', component: CharacterCardEditor }],
    })
    await router.push('/character-cards/12/edit')
    await router.isReady()

    const wrapper = mount(CharacterCardEditor, {
      attachTo: document.body,
      global: {
        plugins: [createPinia(), router, i18n],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: cropperStub,
          RModal: true,
          CharacterCardPortrait: true,
          CharacterCardGalleryImage: galleryImageStub,
        },
      },
    })
    await flushPromises()

    await wrapper.get('.cropper-stub').trigger('click')
    await flushPromises()

    expect(createObjectURL).toHaveBeenCalledWith(expect.any(File))
    expect(mocks.uploadCharacterCardPortrait).toHaveBeenCalledWith(expect.any(File))
    expect(mocks.addCharacterCardPortrait).toHaveBeenCalledWith(12, 'character-card-pending://portrait-token')
    expect(wrapper.findAll('.portrait-film__cell')).toHaveLength(1)
    expect(wrapper.get('.portrait-editor__image').attributes('src')).toBe('/api/v1/images/character-card-portrait/41')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:local-portrait')
    expect(mocks.updateCharacterCard).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
