import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import AchievementCelebration from './AchievementCelebration.vue'

describe('AchievementCelebration', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('renders the completion details and closes from any click', async () => {
    const wrapper = mount(AchievementCelebration, {
      props: {
        celebration: {
          id: 1,
          title: '亮出名牌',
          message: '发布第一张 RPBox 人物卡',
          icon: 'ri-id-card-line',
          rarity: 'common',
          completedAt: '2026-08-31T09:00:00Z',
        },
      },
    })

    const overlay = document.body.querySelector<HTMLElement>('.achievement-screen')
    expect(overlay?.textContent).toContain('恭喜你已完成')
    expect(overlay?.textContent).toContain('亮出名牌')
    expect(overlay?.textContent).toContain('完成时间')
    expect(overlay?.textContent).toContain('点击任意区域关闭')
    expect(overlay?.querySelector('.ri-id-card-line')).not.toBeNull()
    expect(document.body.style.overflow).toBe('hidden')

    overlay?.click()
    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    wrapper.unmount()
    expect(document.body.style.overflow).toBe('')
  })
})
