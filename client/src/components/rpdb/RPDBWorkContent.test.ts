import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RPDBWorkContent from './RPDBWorkContent.vue'
import type { RPDBWorkPayload } from '@/api/rpdb'

describe('RPDBWorkContent Musician MIDI', () => {
  it('sanitizes server-provided rich HTML before Vue inserts it', () => {
    const wrapper = mount(RPDBWorkContent, {
      props: {
        work: {
          type: 'item',
          title: 'Unsafe work',
          content: '<p>Safe copy</p><img src="/safe.png" onerror="alert(1)"><svg onload="alert(2)"></svg><a href="javascript:alert(3)">unsafe link</a>',
        },
      },
    })

    const rich = wrapper.get('.rich-content')
    expect(rich.html()).toContain('<p>Safe copy</p>')
    expect(rich.find('img').attributes('src')).toBe('/safe.png')
    expect(rich.html()).not.toMatch(/onerror|onload|javascript:|<svg/i)
  })

  it('renders the uploaded MIDI as a downloadable Musician file', () => {
    const work: RPDBWorkPayload = {
      type: 'musician_midi',
      title: 'Moonlight Sonata',
      summary: 'A Musician arrangement.',
      content: '<p>Use piano for the lead.</p>',
      extra: {
        midi_url: '/uploads/rpdb/musician-midi/moonlight.mid',
        midi_name: 'Moonlight.mid',
        midi_size: 2048,
      },
    }

    const wrapper = mount(RPDBWorkContent, { props: { work } })

    const download = wrapper.get('[data-testid="musician-midi-download"] a')
    expect(download.attributes('href')).toContain('/uploads/rpdb/musician-midi/moonlight.mid')
    expect(download.attributes('download')).toBe('Moonlight.mid')
    expect(wrapper.text()).toContain('2.0 KB')
    expect(wrapper.text()).toContain('原始 MIDI 文件')
    expect(wrapper.find('#rpdb-section-guide').exists()).toBe(false)
  })

  it('copies the Musician import code for direct in-game paste', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    const code = btoa(String.fromCharCode(77, 85, 83, 56, 0, 0, 16, 0, 0, 0, 0))
    const wrapper = mount(RPDBWorkContent, {
      props: {
        work: {
          type: 'musician_midi',
          title: 'Direct import song',
          extra: { musician_code: code },
        },
      },
    })

    await wrapper.get('[data-testid="copy-musician-code"]').trigger('click')

    expect(writeText).toHaveBeenCalledWith(code)
    expect(wrapper.text()).toContain('已复制')
  })
})
