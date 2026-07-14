import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import RPDBContributionSection from './RPDBContributionSection.vue'

const listRPDBWorks = vi.hoisted(() => vi.fn())

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return { ...actual, listRPDBWorks }
})

describe('RPDBContributionSection', () => {
  it('loads public works for the profile owner', async () => {
    listRPDBWorks.mockResolvedValue({
      works: [{
        id: 9,
        author_id: 7,
        author_name: '雾灯',
        type: 'item_showcase',
        title: '调查员的旧怀表',
        slug: 'old-watch',
        summary: '适合悬疑剧情的可交互道具',
        content: '',
        content_type: 'html',
        cover_image: '',
        rp_use_cases: '',
        effect_description: '',
        restrictions: '{}',
        extra: '{}',
        game_version: '',
        expansion: '',
        availability_status: 'available',
        bind_type: '',
        faction: '',
        armor_type: '',
        verification_status: 'verified',
        verified_count: 1,
        outdated_count: 0,
        status: 'published',
        is_public: true,
        review_status: 'approved',
        version: 1,
        view_count: 10,
        like_count: 3,
        favorite_count: 2,
        comment_count: 1,
        list_count: 0,
        media_count: 0,
        is_liked: false,
        is_favorited: false,
        in_collection_list: false,
        created_at: '2026-07-10T00:00:00Z',
        updated_at: '2026-07-10T00:00:00Z',
      }],
      total: 1,
      page: 1,
      page_size: 6,
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/rpdb', component: { template: '<div />' } },
        { path: '/rpdb/:id', name: 'rpdb-detail', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    const wrapper = mount(RPDBContributionSection, {
      props: { userId: 7, isOwnProfile: false },
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(listRPDBWorks).toHaveBeenCalledWith(expect.objectContaining({ author_id: 7, page_size: 6 }))
    expect(wrapper.text()).toContain('调查员的旧怀表')
  })
})
