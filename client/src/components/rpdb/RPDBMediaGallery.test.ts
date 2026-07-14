import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RPDBMediaGallery from './RPDBMediaGallery.vue'

describe('RPDBMediaGallery', () => {
  const media = [
    { type: 'image' as const, url: '/uploads/preview-1.jpg', thumbnail_url: '/uploads/thumb-1.jpg', caption: '角色正面' },
    { type: 'video' as const, url: '/uploads/preview.mp4', caption: '动作展示' },
  ]

  it('combines the cover and media into a clickable thumbnail list', async () => {
    const wrapper = mount(RPDBMediaGallery, {
      props: {
        cover: '/uploads/cover.jpg',
        media,
        title: '测试作品',
      },
    })

    const thumbnails = wrapper.findAll('[data-testid="gallery-thumbnails"] button')
    expect(thumbnails).toHaveLength(3)
    expect(thumbnails[0].attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="gallery-stage"] .stage-image img').attributes('src')).toContain('/uploads/cover.jpg')

    await thumbnails[1].trigger('click')

    expect(thumbnails[1].attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('[data-testid="gallery-stage"] .stage-image img').attributes('src')).toContain('/uploads/preview-1.jpg')
    expect(wrapper.text()).toContain('角色正面')
    expect(wrapper.text()).toContain('2 / 3')
  })

  it('opens the selected image at its correct position in the image viewer list', async () => {
    const wrapper = mount(RPDBMediaGallery, {
      props: {
        cover: '/uploads/cover.jpg',
        media,
        title: '测试作品',
      },
    })

    await wrapper.findAll('[data-testid="gallery-thumbnails"] button')[1].trigger('click')
    await wrapper.find('[data-testid="gallery-stage"] .stage-image').trigger('click')

    expect(wrapper.emitted('openImage')).toEqual([[['/uploads/cover.jpg', '/uploads/preview-1.jpg'], 1]])
  })

  it('moves through media with the stage navigation controls', async () => {
    const wrapper = mount(RPDBMediaGallery, {
      props: {
        cover: '/uploads/cover.jpg',
        media,
        title: '测试作品',
      },
    })

    await wrapper.find('button[aria-label="下一张预览"]').trigger('click')
    await wrapper.find('button[aria-label="下一张预览"]').trigger('click')

    expect(wrapper.find('video').exists()).toBe(true)
    expect(wrapper.text()).toContain('3 / 3')

    await wrapper.find('button[aria-label="上一张预览"]').trigger('click')
    expect(wrapper.find('[data-testid="gallery-stage"] .stage-image img').attributes('src')).toContain('/uploads/preview-1.jpg')
  })

  it('keeps the thumbnail reel visible for a single preview image', () => {
    const wrapper = mount(RPDBMediaGallery, {
      props: {
        media: [{ type: 'image', url: '/uploads/only-preview.jpg', caption: '唯一预览' }],
        title: '单图作品',
      },
    })

    expect(wrapper.find('[data-testid="gallery-reel"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="gallery-thumbnails"] button')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('预览素材')
  })
})
