import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RPDBGuideSection from './RPDBGuideSection.vue'

describe('RPDBGuideSection', () => {
  it('renders acquisition steps and copyable TomTom commands inside a work', () => {
    const wrapper = mount(RPDBGuideSection, {
      props: {
        steps: [
          {
            sort_order: 1,
            title: '前往守夜营地',
            body: '找到废弃哨塔附近的任务 NPC。',
            zone: '暮色森林',
            map_id: '47',
            x: 42.6,
            y: 71.3,
          },
          {
            sort_order: 2,
            title: '完成补给任务',
            body: '完成前置任务并领取奖励。',
            zone: '暮色森林',
            map_id: '47',
            x: 48.2,
            y: 63.4,
          },
        ],
      },
    })

    expect(wrapper.find('[data-testid="guide-reading"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="guide-step"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('/ttpaste')
    expect(wrapper.text()).toContain('/way #47 42.60 71.30 [1/2] 前往守夜营地')
    expect(wrapper.text()).toContain('/way #47 48.20 63.40 [2/2] 完成补给任务')
    expect(wrapper.text()).toContain('复制 2 个坐标')
  })
})
