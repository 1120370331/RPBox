import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CharacterCardEditor from './CharacterCardEditor.vue'
import type { CharacterCard } from '@/api/characterCard'

const mocks = vi.hoisted(() => ({
  getCharacterCard: vi.fn(),
  updateCharacterCard: vi.fn(),
  syncCharacterCardFromTRP3: vi.fn(),
  uploadCharacterCardPortrait: vi.fn(),
  confirm: vi.fn(),
  invoke: vi.fn(),
}))

vi.mock('@/api/characterCard', () => ({
  getCharacterCard: mocks.getCharacterCard,
  updateCharacterCard: mocks.updateCharacterCard,
  syncCharacterCardFromTRP3: mocks.syncCharacterCardFromTRP3,
  uploadCharacterCardPortrait: mocks.uploadCharacterCardPortrait,
  getCharacterCardPortraitUrl: vi.fn(() => ''),
}))

vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ confirm: mocks.confirm }),
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
  other_content: '<p>旧资料</p>',
  portrait_image_url: '',
  status: 'draft',
  visibility: 'private',
  created_at: '2026-08-10T08:00:00Z',
  updated_at: '2026-08-10T08:00:00Z',
}

beforeEach(() => {
  localStorage.setItem('user', JSON.stringify({ id: 3, username: '人物卡所有者' }))
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
  localStorage.clear()
  delete (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__
  document.body.innerHTML = ''
})

describe('CharacterCardEditor tabs', () => {
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
        plugins: [createPinia(), router],
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
        plugins: [createPinia(), router],
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
        plugins: [createPinia(), router],
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
    const impression = wrapper.find<HTMLTextAreaElement>('textarea[placeholder="第一次见到这位角色时，人们会注意到…"]')
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
        plugins: [createPinia(), router],
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
    const refreshButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('从备份刷新基础信息'))!
    await refreshButton.trigger('click')
    await flushPromises()

    expect(mocks.syncCharacterCardFromTRP3).toHaveBeenCalledWith(12)
    expect(summary.element.value).toBe('尚未保存的新摘要')
    expect(wrapper.findAll<HTMLInputElement>('.form-grid input')[0].element.value).toBe('刷新后的名字')

    const saveButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('保存整张人物卡'))!
    await saveButton.trigger('click')
    await flushPromises()
    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      first_name: '刷新后的名字',
      summary: '尚未保存的新摘要',
    }))
    wrapper.unmount()
  })

  it('scopes desktop TRP3 write-back to the source account', async () => {
    const importedCard: CharacterCard = {
      ...card,
      source_backup_id: 7,
      source_account_id: 'WOW-ACCOUNT-B',
      source_profile_id: 'shared-profile',
    }
    mocks.getCharacterCard.mockResolvedValue(importedCard)
    mocks.confirm.mockResolvedValue(true)
    mocks.invoke.mockResolvedValue(undefined)
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
        plugins: [createPinia(), router],
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

    const writeBackButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('写回本地 TRP3 基础信息'))!
    expect(writeBackButton.attributes('disabled')).toBeUndefined()
    await writeBackButton.trigger('click')
    await flushPromises()

    expect(mocks.invoke).toHaveBeenCalledWith('update_profile', expect.objectContaining({
      wowPath: 'C:\\Games\\World of Warcraft\\_retail_\\WTF',
      accountId: 'WOW-ACCOUNT-B',
      profileId: 'shared-profile',
      updates: expect.objectContaining({
        characteristics: expect.objectContaining({
          eyeColorHex: 'C9D5E7',
        }),
      }),
    }))
    wrapper.unmount()
  })

  it('previews a protected portrait locally and revokes its object URL after save', async () => {
    mocks.getCharacterCard.mockResolvedValue(card)
    mocks.uploadCharacterCardPortrait.mockResolvedValue('character-card-pending://portrait-token')
    mocks.updateCharacterCard.mockImplementation(async (_id: number, payload: Partial<CharacterCard>) => ({
      ...card,
      ...payload,
      portrait_image_url: 'stored/character-card-portrait.webp',
      portrait_image_updated_at: '2026-08-10T10:00:00Z',
    }))
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
        plugins: [createPinia(), router],
        stubs: {
          TiptapEditor: editorStub,
          PostQuickJump: true,
          ImageCropperDialog: cropperStub,
          RModal: true,
          CharacterCardPortrait: true,
        },
      },
    })
    await flushPromises()

    await wrapper.get('.cropper-stub').trigger('click')
    await flushPromises()

    expect(createObjectURL).toHaveBeenCalledWith(expect.any(File))
    expect(mocks.uploadCharacterCardPortrait).toHaveBeenCalledWith(expect.any(File))
    expect(wrapper.get('.portrait-editor__image').attributes('src')).toBe('blob:local-portrait')

    const saveButton = wrapper.findAll<HTMLButtonElement>('button')
      .find((button) => button.text().includes('保存整张人物卡'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(mocks.updateCharacterCard).toHaveBeenCalledWith(12, expect.objectContaining({
      portrait_image_url: 'character-card-pending://portrait-token',
    }))
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:local-portrait')
    wrapper.unmount()
  })
})
