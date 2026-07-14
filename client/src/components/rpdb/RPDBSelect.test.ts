import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import RPDBSelect from './RPDBSelect.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('RPDBSelect', () => {
  it('teleports options outside clipped editor cards and updates the selection', async () => {
    const wrapper = mount(RPDBSelect, {
      attachTo: document.body,
      props: {
        modelValue: 'available',
        options: [
          { value: 'available', label: '可获取' },
          { value: 'limited', label: '限时获取' },
          { value: 'removed', label: '已绝版' },
        ],
      },
    })

    await wrapper.find('.rpdb-select__trigger').trigger('click')
    await flushPromises()

    const menu = document.body.querySelector<HTMLElement>('.rpdb-select__menu')
    expect(menu).not.toBeNull()
    expect(wrapper.element.contains(menu)).toBe(false)
    expect(menu?.parentElement).toBe(document.body)
    expect(menu?.style.top).not.toBe('')

    const options = document.body.querySelectorAll<HTMLButtonElement>('.rpdb-select__option')
    expect(options).toHaveLength(3)
    options[1].click()
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['limited'])
    expect(document.body.querySelector('.rpdb-select__menu')).toBeNull()
    wrapper.unmount()
  })
})
