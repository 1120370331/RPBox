import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CharacterCardPicker from './CharacterCardPicker.vue'

const mocks = vi.hoisted(() => ({ sources: vi.fn(), invoke: vi.fn() }))
vi.mock('@/api/characterCard', () => ({ getCharacterCardSources: mocks.sources }))
vi.mock('@tauri-apps/api/core', () => ({ invoke: mocks.invoke }))
const modal = defineComponent({ template:'<div><slot /></div>' })
afterEach(() => { delete (window as unknown as Record<string, unknown>).__TAURI_INTERNALS__; localStorage.clear(); vi.clearAllMocks() })

describe('CharacterCardPicker', () => {
  it('searches names without categorizing cards by source or matching their profession', async () => {
    mocks.sources.mockResolvedValue({ sources:[{ backup_id:1, account_id:'A', profile_id:'P', display_name:'艾琳晨星', class:'游侠' }] })
    const wrapper = mount(CharacterCardPicker,{ props:{ modelValue:true,cards:[] },global:{stubs:{RModal:modal,RouterLink:true}} })
    await flushPromises()
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('.character-choice__source').exists()).toBe(false)
    await wrapper.get('[aria-label="人物卡名称"]').setValue('晨星')
    expect(wrapper.findAll('.character-choice')).toHaveLength(1)
    await wrapper.get('[aria-label="人物卡名称"]').setValue('游侠')
    expect(wrapper.findAll('.character-choice')).toHaveLength(0)
    expect(wrapper.text()).toContain('需要在 RPBox 桌面客户端读取')
    expect(mocks.invoke).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('reads the configured local source and emits only the selected profile', async () => {
    (window as unknown as Record<string, unknown>).__TAURI_INTERNALS__ = {}
    localStorage.setItem('wow_path','C:/WoW/_retail_')
    mocks.sources.mockResolvedValue({sources:[]})
    mocks.invoke.mockResolvedValue({accounts:[{account_id:'A',raw_trp3_lua:'entire account must not be submitted',profiles:[
      {id:'one',name:'人物一',raw_lua:JSON.stringify({player:{characteristics:{FN:'米拉',CL:'猎人'}}})},
      {id:'two',name:'人物二',raw_lua:JSON.stringify({player:{characteristics:{FN:'艾琳',CL:'法师'}}})},
    ]}]})
    const wrapper=mount(CharacterCardPicker,{props:{modelValue:true,cards:[]},global:{stubs:{RModal:modal,RouterLink:true}}})
    await flushPromises()
    expect(mocks.invoke).toHaveBeenCalledWith('scan_profiles',{wowPath:'C:/WoW/_retail_'})
    await wrapper.get('[aria-label="人物卡名称"]').setValue('艾琳')
    await wrapper.get('.character-choice').trigger('click')
    const choice=wrapper.emitted('select')?.[0]?.[0] as {kind:string;source:{profile_id:string;profile:unknown}}
    expect(choice.kind).toBe('local');expect(choice.source.profile_id).toBe('two')
    expect(JSON.stringify(choice)).not.toContain('entire account')
    expect(JSON.stringify(choice)).not.toContain('米拉')
    wrapper.unmount()
  })
})
