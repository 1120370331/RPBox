import { defineComponent, h } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import type { ChatRecord } from '@/types/chatLog'
import ArchivesMain from './ArchivesMain.vue'

const mocks = vi.hoisted(() => ({
  invoke: vi.fn(),
  createStory: vi.fn(),
  addStoryEntries: vi.fn(),
  getStory: vi.fn(),
  listStories: vi.fn(),
  listTags: vi.fn(),
  addStoryTag: vi.fn(),
  getAddonManifest: vi.fn(),
  getGuild: vi.fn(),
  removeArchivedRecords: vi.fn(),
  reloadStories: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  toastInfo: vi.fn(),
}))

vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
vi.mock('@/api/story', () => ({
  createStory: mocks.createStory,
  addStoryEntries: mocks.addStoryEntries,
  getStory: mocks.getStory,
  listStories: mocks.listStories,
}))
vi.mock('@/api/tag', () => ({
  listTags: mocks.listTags,
  addStoryTag: mocks.addStoryTag,
}))
vi.mock('@/api/addon', () => ({ getAddonManifest: mocks.getAddonManifest }))
vi.mock('@/api/guild', () => ({ getGuild: mocks.getGuild }))
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
    info: mocks.toastInfo,
  }),
}))

const archiveRecords: ChatRecord[] = [{
  record_key: 'rpbox-archive-1',
  account_id: 'ACCOUNT-A',
  timestamp: 1_753_200_000,
  channel: 'SAY',
  sender: { gameID: 'Player-Realm' },
  content: 'A line worth remembering',
  ref_id: 'profile-a',
  profile_snapshot_id: 'snapshot-a',
  profile_snapshot: { ref: 'profile-a', n: 'Historic Name', pn: 'Main Card' },
  identity_source: 'snapshot',
}]

const StagingPoolStub = defineComponent({
  name: 'StagingPool',
  emits: ['archive'],
  setup(_, { emit, expose }) {
    expose({ removeArchivedRecords: mocks.removeArchivedRecords })
    return () => h('button', { class: 'staging-test-trigger', onClick: () => emit('archive', archiveRecords) }, 'archive')
  },
})

const StoryListStub = defineComponent({
  name: 'StoryList',
  setup(_, { expose }) {
    expose({ loadStories: mocks.reloadStories })
    return () => null
  },
})

const mountedWrappers: VueWrapper[] = []
let consoleErrorSpy: ReturnType<typeof vi.spyOn>

async function mountArchives() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/archives', name: 'archives', component: { template: '<div />' } },
      { path: '/stories/:id', name: 'story-detail', component: { template: '<div />' } },
    ],
  })
  await router.push('/archives')
  await router.isReady()

  const wrapper = mount(ArchivesMain, {
    attachTo: document.body,
    global: {
      plugins: [router, i18n],
      stubs: {
        Teleport: true,
        RTabs: { template: '<div><slot /></div>' },
        RTabPane: { template: '<section><slot /></section>' },
        StagingPool: StagingPoolStub,
        StoryList: StoryListStub,
        AddonInstaller: true,
        AddonUpdateDialog: true,
      },
    },
  })
  mountedWrappers.push(wrapper)
  await flushPromises()
  return wrapper
}

describe('ArchivesMain archive workbench', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    localStorage.clear()
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft')
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    Object.values(mocks).forEach(mock => mock.mockReset())
    mocks.invoke.mockResolvedValue({ installed: false, version: null })
    mocks.listTags.mockResolvedValue({ tags: [] })
    mocks.listStories.mockResolvedValue({
      stories: [
        { id: 11, title: 'Moonlit Patrol', description: 'North road', tags: '', updated_at: '2026-07-20T00:00:00Z', entry_count: 12 },
        { id: 12, title: 'Harbor Meeting', description: 'Docks', tags: '', updated_at: '2026-07-21T00:00:00Z', entry_count: 4 },
      ],
    })
    mocks.createStory.mockResolvedValue({ id: 77, title: 'Suggested story' })
    mocks.addStoryEntries.mockResolvedValue(undefined)
    mocks.getStory.mockResolvedValue({ story: { id: 77 }, entries: [] })
  })

  afterEach(() => {
    mountedWrappers.splice(0).forEach(wrapper => wrapper.unmount())
    consoleErrorSpy.mockRestore()
  })

  it('opens a compact archive manifest and searches existing story targets', async () => {
    const wrapper = await mountArchives()
    await wrapper.get('.staging-test-trigger').trigger('click')
    await flushPromises()

    expect(wrapper.get('.archive-manifest').text()).toContain('1')
    expect((wrapper.get('.form-field input').element as HTMLInputElement).value).toContain('Historic Name')

    await wrapper.findAll('.mode-btn')[1].trigger('click')
    await wrapper.get('input[type="search"]').setValue('Moonlit')
    expect(wrapper.findAll('.story-option')).toHaveLength(1)
    expect(wrapper.get('.story-option').text()).toContain('Moonlit Patrol')
  })

  it('reuses a story created by a failed attempt and reconciles before retrying', async () => {
    mocks.addStoryEntries
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockResolvedValueOnce(undefined)

    const wrapper = await mountArchives()
    await wrapper.get('.staging-test-trigger').trigger('click')
    await flushPromises()

    const submit = wrapper.get('.r-modal__footer .r-button--primary')
    expect(submit.attributes('disabled')).toBeUndefined()
    await submit.trigger('click')
    await flushPromises()

    expect(mocks.createStory).toHaveBeenCalledTimes(1)
    expect(wrapper.get('.archive-recovery-note').text()).toContain('#77')
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('.r-modal__close').exists()).toBe(false)
    expect(wrapper.get('.r-modal__footer .r-button--outline').attributes('disabled')).toBeDefined()

    await wrapper.get('.r-modal__footer .r-button--primary').trigger('click')
    await flushPromises()

    expect(mocks.createStory).toHaveBeenCalledTimes(1)
    expect(mocks.getStory).toHaveBeenCalledTimes(1)
    expect(mocks.addStoryEntries).toHaveBeenCalledTimes(2)
    expect(mocks.removeArchivedRecords).toHaveBeenCalledWith(['rpbox-archive-1'])
    expect(localStorage.getItem('rpbox_recent_archive_story_id')).toBe('77')
    expect(mocks.toastSuccess).toHaveBeenCalledTimes(1)
  })

  it('does not submit the same source twice when the first response was ambiguous', async () => {
    mocks.addStoryEntries.mockRejectedValueOnce(new Error('response lost'))
    mocks.getStory.mockResolvedValue({
      story: { id: 77 },
      entries: [{ source_id: 'chat_rpbox-archive-1' }],
    })

    const wrapper = await mountArchives()
    await wrapper.get('.staging-test-trigger').trigger('click')
    await flushPromises()
    await wrapper.get('.r-modal__footer .r-button--primary').trigger('click')
    await flushPromises()
    await wrapper.get('.r-modal__footer .r-button--primary').trigger('click')
    await flushPromises()

    expect(mocks.addStoryEntries).toHaveBeenCalledTimes(1)
    expect(mocks.getStory).toHaveBeenCalledWith(77)
    expect(mocks.removeArchivedRecords).toHaveBeenCalledWith(['rpbox-archive-1'])
    expect(mocks.toastInfo).toHaveBeenCalledTimes(1)
  })
})
