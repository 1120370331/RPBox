import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RPDBWorkCard from './RPDBWorkCard.vue'
import type { RPDBWork } from '@/api/rpdb'

const work: RPDBWork = {
  id: 7,
  author_id: 2,
  author_name: '守望者',
  type: 'item_showcase',
  title: '月光灯笼',
  slug: 'moon-lantern',
  summary: '适合夜间巡逻与酒馆场景。',
  content: '',
  content_type: 'html',
  cover_image: '',
  rp_use_cases: '酒馆,巡逻',
  effect_description: '发出柔和蓝光',
  restrictions: '{}',
  extra: '{}',
  game_version: '11.2.7',
  expansion: '至暗之夜',
  availability_status: 'available',
  bind_type: 'yes',
  item_type: 'toy',
  faction: 'neutral',
  armor_type: '',
  verification_status: 'verified',
  verified_count: 12,
  outdated_count: 1,
  status: 'published',
  is_public: true,
  review_status: 'approved',
  version: 2,
  view_count: 188,
  like_count: 24,
  favorite_count: 31,
  comment_count: 6,
  list_count: 18,
  media_count: 3,
  is_liked: false,
  is_favorited: true,
  in_collection_list: false,
  tags: [
    { id: 101, name: '联盟风格', color: '2F66C8' },
    { id: 103, name: '库尔提拉斯风格', color: '356A8A' },
  ],
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-09T00:00:00Z',
}

describe('RPDBWorkCard', () => {
  it('renders the UGC identity and RP style tags', () => {
    const wrapper = mount(RPDBWorkCard, { props: { work } })

    expect(wrapper.text()).toContain('月光灯笼')
    expect(wrapper.text()).toContain('守望者')
    expect(wrapper.text()).toContain('魔兽物品')
    const itemTraits = wrapper.get('[data-testid="rpdb-item-traits"]')
    expect(itemTraits.text()).toContain('玩具')
    expect(itemTraits.text()).toContain('已绑定')
    expect(itemTraits.find('[title="物品类型"]').attributes('aria-label')).toBe('物品类型 玩具')
    expect(itemTraits.find('[title="是否绑定"]').attributes('aria-label')).toBe('是否绑定 已绑定')
    expect(wrapper.text()).toContain('联盟风格')
    expect(wrapper.text()).toContain('库尔提拉斯风格')
    expect(wrapper.text()).not.toContain('社区已验证')
    expect(wrapper.text()).toContain('31')
    const metrics = wrapper.get('[data-testid="rpdb-work-metrics"]')
    expect(metrics.find('small').exists()).toBe(false)
    expect(metrics.findAll('span').map(metric => metric.attributes('title'))).toEqual(['浏览', '点赞', '收藏', '加入清单'])
    expect(metrics.findAll('b').map(metric => metric.text())).toEqual(['188', '24', '31', '18'])
  })

  it('compacts long metric values to keep the stats on one row', () => {
    const wrapper = mount(RPDBWorkCard, {
      props: { work: { ...work, view_count: 12345 } },
    })

    expect(wrapper.get('[data-testid="rpdb-work-metrics"] b').text()).toBe('1.2万')
  })

  it('shows quest items that do not bind as explicit card traits', () => {
    const wrapper = mount(RPDBWorkCard, {
      props: { work: { ...work, item_type: 'quest_item', bind_type: 'no' } },
    })

    expect(wrapper.get('[data-testid="rpdb-item-traits"]').text()).toContain('任务道具')
    expect(wrapper.get('[data-testid="rpdb-item-traits"]').text()).toContain('不绑定')
  })

  it('emits open when the card is activated', async () => {
    const wrapper = mount(RPDBWorkCard, { props: { work } })
    await wrapper.get('[data-testid="rpdb-work-card"]').trigger('click')
    expect(wrapper.emitted('open')).toHaveLength(1)
  })
})
