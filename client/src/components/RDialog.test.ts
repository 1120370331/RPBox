import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import i18n from '@/i18n'
import { dialog } from '@/composables/useDialog'
import RDialog from './RDialog.vue'

afterEach(() => dialog.close(false))
describe('RDialog confirmation modes', () => {
  it('preserves the singleton confirmation API', async () => {
    const wrapper = mount(RDialog, { global: { plugins: [i18n], stubs: { teleport: true } } })
    const result = dialog.confirm({ title: '确认操作', message: '确认继续' })
    await flushPromises()
    expect(wrapper.get('[role="dialog"]').attributes('aria-label')).toBe('确认操作')
    await wrapper.get('.r-dialog__btn--confirm').trigger('click')
    await expect(result).resolves.toBe(true)
    expect(dialog.state.value.visible).toBe(false)
    wrapper.unmount()
  })
  it('keeps controlled forms open until the parent accepts and prevents invalid confirmation', async () => {
    const wrapper = mount(RDialog, { props: { modelValue: true, title: '确认游戏目录', confirmDisabled: true }, slots: { default: '<select><option>正式服</option></select>' }, global: { plugins: [i18n], stubs: { teleport: true } } })
    await wrapper.get('.r-dialog__btn--confirm').trigger('click')
    expect(wrapper.emitted('confirm')).toBeUndefined()
    await wrapper.setProps({ confirmDisabled: false })
    await wrapper.get('.r-dialog__btn--confirm').trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
    await wrapper.get('.r-dialog__btn--cancel').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
    wrapper.unmount()
  })
})
