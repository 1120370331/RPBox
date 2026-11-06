import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import GuildVersionSelect from '@shared/components/GuildVersionSelect.vue'
import type { GuildGameFlavor } from '@shared/utils/guildGameFlavors'

describe('guild version dropdown', () => {
  it('keeps the choices inside a dropdown and allows both versions', async () => {
    const Host = defineComponent({
      components: { GuildVersionSelect },
      setup() { return { selected: ref<GuildGameFlavor[]>([]) } },
      template: '<GuildVersionSelect v-model="selected" label="版本" all-label="全部" retail-label="正式服" forever-label="魔兽：无限" allow-all />',
    })
    const wrapper = mount(Host, { attachTo: document.body })
    try {
      expect(document.querySelector('.guild-version-menu')).toBeNull()
      await wrapper.get('.guild-version-trigger').trigger('click')
      await nextTick()
      const menu = document.querySelector('.guild-version-menu')!
      expect(menu).not.toBeNull()
      menu.querySelector<HTMLInputElement>('input[value="_retail_"]')!.click()
      menu.querySelector<HTMLInputElement>('input[value="_forever_"]')!.click()
      await nextTick()
      expect((wrapper.vm as unknown as { selected: GuildGameFlavor[] }).selected).toEqual(['_retail_', '_forever_'])
      expect(wrapper.get('.guild-version-trigger').text()).toContain('正式服、魔兽：无限')
      menu.querySelector<HTMLButtonElement>('.guild-version-all')!.click()
      await nextTick()
      expect((wrapper.vm as unknown as { selected: GuildGameFlavor[] }).selected).toEqual([])
      expect(document.querySelector('.guild-version-menu')).toBeNull()
    } finally {
      wrapper.unmount()
    }
  })

  it('lets an existing guild select both without allowing an empty assignment', async () => {
    const Host = defineComponent({
      components: { GuildVersionSelect },
      setup() { return { selected: ref<GuildGameFlavor[]>(['_retail_']) } },
      template: '<GuildVersionSelect v-model="selected" label="版本" all-label="全部" retail-label="正式服" forever-label="魔兽：无限" />',
    })
    const wrapper = mount(Host, { attachTo: document.body })
    try {
      await wrapper.get('.guild-version-trigger').trigger('click')
      await nextTick()
      const menu = document.querySelector('.guild-version-menu')!
      expect(menu.querySelector<HTMLInputElement>('input[value="_retail_"]')!.disabled).toBe(true)
      menu.querySelector<HTMLInputElement>('input[value="_forever_"]')!.click()
      await nextTick()
      expect((wrapper.vm as unknown as { selected: GuildGameFlavor[] }).selected).toEqual(['_retail_', '_forever_'])
    } finally {
      wrapper.unmount()
    }
  })
})
