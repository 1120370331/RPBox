import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import RPDBPreview from './RPDBPreview.vue'
import i18n from '@/i18n'

const resolveRPDBMediaURL = vi.hoisted(() => vi.fn((value?: string) => value ? `resolved:${value}` : ''))

vi.mock('@/api/rpdb', async () => {
  const actual = await vi.importActual<typeof import('@/api/rpdb')>('@/api/rpdb')
  return { ...actual, resolveRPDBMediaURL }
})

describe('RPDBPreview', () => {
  it('renders the internal review toolbar, work preview and quality checks', async () => {
    i18n.global.locale.value = 'zh-CN'
    sessionStorage.setItem('rpdb-preview', JSON.stringify({
      type: 'item_showcase',
      title: '测试作品',
      cover_image: '/uploads/rpdb/demo/item-01.jpg',
      summary: '测试摘要',
      content: '<p>测试正文</p>',
      availability_status: 'available',
      bind_type: 'no',
      faction: 'neutral',
      guide_steps: [{
        sort_order: 1,
        title: '前往测试地点',
        body: '完成测试步骤',
        zone: '暮色森林',
        map_id: '47',
        x: 42.6,
        y: 71.3,
      }],
    }))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/preview', component: RPDBPreview }],
    })
    await router.push('/preview')
    const wrapper = mount(RPDBPreview, { global: { plugins: [router] } })

    expect(resolveRPDBMediaURL).toHaveBeenCalledWith('/uploads/rpdb/demo/item-01.jpg')
    expect(wrapper.find('[data-testid="preview-toolbar"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preview-workspace"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preview-quality"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="preview-workspace"] [data-testid="preview-quality"]').exists()).toBe(false)
    expect(wrapper.find('.minimal-preview-shell').exists()).toBe(true)
    expect(wrapper.find('[data-testid="work-hero"] img').attributes('src')).toBe('resolved:/uploads/rpdb/demo/item-01.jpg')
    expect(wrapper.text()).toContain('/ttpaste')
    expect(wrapper.find('.work-metadata').text()).toContain('获取状态可获取')
    expect(wrapper.find('.work-metadata').text()).toContain('是否绑定否')
    expect(wrapper.find('.work-metadata').text()).toContain('阵营不限')
    expect(wrapper.find('.work-metadata').text()).not.toContain('available')
    expect(wrapper.find('.work-metadata').text()).not.toContain('neutral')
    expect(wrapper.text()).not.toContain('资料片')
    expect(wrapper.text()).not.toContain('版本未标注')
  })
})
