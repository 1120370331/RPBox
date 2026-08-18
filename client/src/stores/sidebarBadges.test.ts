import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSidebarBadgesStore } from './sidebarBadges'

const mocks = vi.hoisted(() => ({
  listPosts: vi.fn(),
  listEvents: vi.fn(),
  listItems: vi.fn(),
  listRPDBWorks: vi.fn(),
  getAddonLatest: vi.fn(),
  getTRP3Latest: vi.fn(),
  invoke: vi.fn(),
}))

vi.mock('@/api/post', () => ({
  listPosts: mocks.listPosts,
  listEvents: mocks.listEvents,
}))

vi.mock('@/api/item', () => ({ listItems: mocks.listItems }))
vi.mock('@/api/rpdb', () => ({ listRPDBWorks: mocks.listRPDBWorks }))
vi.mock('@/api/addon', () => ({
  getAddonLatest: mocks.getAddonLatest,
  getTRP3Latest: mocks.getTRP3Latest,
}))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))

function mockContentTotals(community: number, events: number, market: number, rpdb: number) {
  mocks.listPosts.mockResolvedValue({ posts: [], total: community })
  mocks.listEvents.mockResolvedValue({ events: Array.from({ length: events }, (_, id) => ({ id })) })
  mocks.listItems.mockResolvedValue({ code: 0, data: { items: [], total: market } })
  mocks.listRPDBWorks.mockResolvedValue({ works: [], total: rpdb, page: 1, page_size: 1 })
}

describe('sidebar badge store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
    mockContentTotals(12, 3, 7, 9)
  })

  it('creates a quiet first-use baseline, then counts new publications per section', async () => {
    const store = useSidebarBadgesStore()
    await store.initialize(42)

    expect(store.unreadCounts).toEqual({ community: 0, events: 0, market: 0, rpdb: 0 })

    mockContentTotals(15, 4, 9, 14)
    await store.refreshContent()

    expect(store.unreadCounts).toEqual({ community: 3, events: 1, market: 2, rpdb: 5 })
  })

  it('marks both community publications and events as read when the community menu opens', async () => {
    const store = useSidebarBadgesStore()
    await store.initialize(42)
    mockContentTotals(14, 5, 7, 9)
    await store.refreshContent()

    store.markMenuRead('community')

    expect(store.unreadCounts.community).toBe(0)
    expect(store.unreadCounts.events).toBe(0)
  })

  it('shows installed add-on updates at startup and keeps the current version signature dismissed', async () => {
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft')
    mocks.invoke.mockImplementation((command: string) => {
      if (command === 'check_trp3_addons') {
        return Promise.resolve({
          addons: [
            { id: 'total-rp-3', installed: true, version: '2.0.0' },
            { id: 'total-rp-3-extended', installed: true, version: '1.0.0' },
          ],
        })
      }
      return Promise.resolve({ installed: true, version: '1.0.0' })
    })
    mocks.getTRP3Latest.mockResolvedValue({
      addons: [
        { id: 'total-rp-3', latestVersion: '2.1.0' },
        { id: 'total-rp-3-extended', latestVersion: '1.0.0' },
      ],
    })
    mocks.getAddonLatest.mockResolvedValue({ version: '1.1.0' })

    const store = useSidebarBadgesStore()
    await store.initialize(42)
    expect(store.addonUpdateCount).toBe(2)

    store.markMenuRead('warcraft')
    expect(store.addonUpdateCount).toBe(0)

    await store.refreshAddonUpdates()
    expect(store.addonUpdateCount).toBe(0)

    mocks.getAddonLatest.mockResolvedValue({ version: '1.2.0' })
    await store.refreshAddonUpdates()
    expect(store.addonUpdateCount).toBe(2)
  })

  it('does not create an update badge when server metadata is older than the installed add-on', async () => {
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft')
    mocks.invoke.mockImplementation((command: string) => {
      if (command === 'check_trp3_addons') {
        return Promise.resolve({ addons: [{ id: 'total-rp-3', installed: true, version: '3.4.1' }] })
      }
      return Promise.resolve({ installed: true, version: '1.0.14' })
    })
    mocks.getTRP3Latest.mockResolvedValue({
      addons: [{ id: 'total-rp-3', latestVersion: '3.3.6' }],
    })
    mocks.getAddonLatest.mockResolvedValue({ version: '1.0.14' })

    const store = useSidebarBadgesStore()
    await store.initialize(42)

    expect(store.addonUpdateCount).toBe(0)
  })

  it('shows each RPBox system update once until a newer version is available', () => {
    const store = useSidebarBadgesStore()

    store.syncSystemUpdate('v1.2.0')
    expect(store.systemUpdateAvailable).toBe(true)
    expect(store.systemUpdateVersion).toBe('1.2.0')

    store.markMenuRead('settings')
    expect(store.systemUpdateAvailable).toBe(false)

    store.syncSystemUpdate('1.2.0')
    expect(store.systemUpdateAvailable).toBe(false)

    store.syncSystemUpdate('1.3.0')
    expect(store.systemUpdateAvailable).toBe(true)
  })
})
