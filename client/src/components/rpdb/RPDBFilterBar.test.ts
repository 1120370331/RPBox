import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RPDBFilterBar from './RPDBFilterBar.vue'
import type { ListRPDBWorksParams } from '@/api/rpdb'

function mountFilter(modelValue: ListRPDBWorksParams) {
  return mount(RPDBFilterBar, {
    props: {
      modelValue,
      styleTags: [
        { id: 101, name: '联盟风格' },
        { id: 103, name: '库尔提拉斯风格' },
      ],
    },
  })
}

describe('RPDBFilterBar', () => {
  it('shows different filters for each RPDB work type', () => {
    const item = mountFilter({ search: '', type: 'item_showcase', sort: 'updated_at' })
    expect(item.find('[aria-label="获取状态"]').exists()).toBe(true)
    expect(item.find('[data-testid="rpdb-tag-filter-input"]').exists()).toBe(true)
    expect(item.find('[aria-label="护甲类型"]').exists()).toBe(false)
    expect(item.text()).not.toContain('全部可信度')

    const transmog = mountFilter({ search: '', type: 'transmog', sort: 'updated_at' })
    expect(transmog.find('[aria-label="护甲类型"]').exists()).toBe(true)
    expect(transmog.find('[aria-label="阵营"]').exists()).toBe(true)
    expect(transmog.find('[aria-label="参观状态"]').exists()).toBe(false)

    const home = mountFilter({ search: '', type: 'home_showcase', sort: 'updated_at' })
    expect(home.find('[aria-label="参观状态"]').exists()).toBe(true)
    expect(home.find('[data-testid="rpdb-tag-filter-input"]').exists()).toBe(true)
    expect(home.find('[aria-label="护甲类型"]').exists()).toBe(false)
  })

  it('emits fuzzy RP style filters from typed text and suggestions', async () => {
    const wrapper = mountFilter({ search: '', type: '', sort: 'updated_at' })

    expect(wrapper.find('[data-testid="rpdb-tag-filter-input"]').attributes('placeholder')).toContain('风格')
    expect(wrapper.find('[data-testid="rpdb-tag-filter-suggestion"]').exists()).toBe(false)
    await wrapper.find('[data-testid="rpdb-tag-filter-input"]').trigger('click')
    expect(wrapper.findAll('[data-testid="rpdb-tag-filter-suggestion"]')).toHaveLength(2)
    expect(wrapper.findAll('[data-testid="rpdb-tag-filter-suggestion"]')[0].text()).toBe('联盟风格')

    await wrapper.find('[data-testid="rpdb-tag-filter-input"]').setValue('库尔')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({
      tag_search: '库尔',
      page: 1,
    })

    await wrapper.find('[data-testid="rpdb-tag-filter-input"]').trigger('keyup.enter')
    expect(wrapper.emitted('search')).toBeTruthy()

    await wrapper.setProps({ modelValue: { search: '', type: '', sort: 'updated_at', tag_search: '库尔' } })
    await wrapper.findAll('[data-testid="rpdb-tag-filter-suggestion"]')[0].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({
      tag_search: '库尔提拉斯风格',
      page: 1,
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="rpdb-tag-filter-suggestion"]').exists()).toBe(false)
  })

  it('uses a usable search action button instead of a passive filter trigger', async () => {
    const wrapper = mountFilter({ search: '月光灯笼', type: '', sort: 'updated_at' })

    expect(wrapper.find('.filter-trigger').exists()).toBe(false)
    const searchButton = wrapper.find('[data-testid="rpdb-search-button"]')
    const searchRow = wrapper.get('[data-testid="rpdb-search-row"]')
    const filterRow = wrapper.get('[data-testid="rpdb-filter-row"]')
    expect(searchButton.exists()).toBe(true)
    expect(searchButton.text()).toContain('搜索')
    expect(searchRow.element.children[0].classList.contains('search-box')).toBe(true)
    expect(searchRow.element.children[1]).toBe(searchButton.element)
    expect(filterRow.findAll('select')).toHaveLength(3)
    expect(filterRow.find('[data-testid="rpdb-tag-filter-input"]').exists()).toBe(true)

    await searchButton.trigger('click')

    expect(wrapper.emitted('search')).toBeTruthy()
  })
})
