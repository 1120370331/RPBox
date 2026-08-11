import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import i18n from '@/i18n'
import CharacterCardColorField from './CharacterCardColorField.vue'

i18n.global.locale.value = 'zh-CN'

describe('CharacterCardColorField', () => {
  it('keeps the keyboard color input and exact HEX value synchronized', async () => {
    const wrapper = mount(CharacterCardColorField, {
      props: { modelValue: '80C9D5E7', label: '名字颜色' },
      global: { plugins: [i18n] },
    })

    expect(wrapper.get<HTMLInputElement>('.character-dye__hex').element.value).toBe('#C9D5E780')
    await wrapper.get<HTMLInputElement>('input[type="color"]').setValue('#336699')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['#336699'])

    await wrapper.setProps({ modelValue: '#336699' })
    await wrapper.get<HTMLInputElement>('.character-dye__hex').setValue('#12zz99')
    expect(wrapper.get('.character-dye__hex').attributes('aria-invalid')).toBe('true')
    expect(wrapper.text()).toContain('#RRGGBB')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['#336699'])
  })
})
