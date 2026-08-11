import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import i18n from '@/i18n'
import type { Character } from '@/api/character'
import type { CharacterCardSummary } from '@/api/characterCard'
import CharacterCard from './CharacterCard.vue'

const rpboxCard: CharacterCardSummary = {
  id: 41,
  user_id: 7,
  first_name: '伊莱娅',
  last_name: '雾桥',
  display_name: '伊莱娅·雾桥',
  title: '暮色信使',
  full_title: '',
  race: '人类',
  class: '潜行者',
  eye_color: '灰绿色',
  eye_color_hex: '76917D',
  age: '29',
  height: '168 cm',
  weight: '',
  birthplace: '',
  residence: '伯拉勒斯',
  relationship_status: '',
  icon: 'ability_stealth',
  class_color: 'FFF468',
  name_color: 'FFF468',
  summary: '负责把不该留在纸面上的消息送到正确的人手里。',
  portrait_image_url: '/api/v1/images/character-card-portrait/41',
  portrait_image_updated_at: '2026-08-11T08:00:00Z',
  impressions: [{
    slot: 1,
    active: true,
    title: '总在观察出口',
    text: '进入陌生房间时，她会先确认每一条退路。',
    trp3_icon: 'ability_rogue_sprint',
    icon_image_url: '',
    icon_image_updated_at: null,
    image_url: '',
    image_updated_at: null,
  }],
  status: 'draft',
  visibility: 'private',
  created_at: '2026-08-11T08:00:00Z',
  updated_at: '2026-08-11T08:00:00Z',
}

const legacyCharacter: Character = {
  id: 9,
  user_id: 7,
  ref_id: 'legacy-profile',
  game_id: 'Legacy-Realm',
  is_npc: false,
  created_at: '2026-08-11T08:00:00Z',
  updated_at: '2026-08-11T08:00:00Z',
  trp3_version: 1,
  race: '暗夜精灵',
  class: '德鲁伊',
  first_name: '莱格',
  last_name: '',
  full_title: '',
  title: '林地守望者',
  icon: 'ability_druid_catform',
  color: 'FF7C0A',
  eye_color: '琥珀色',
  age: '',
  height: '',
  residence: '',
  birthplace: '',
  misc_info: '',
  psycho: '',
  about_text: '',
  custom_avatar: '',
  custom_name: '',
  custom_color: '',
  raw_trp3_data: '',
}

function mountCard(props: Record<string, unknown>) {
  return mount(CharacterCard, {
    attachTo: document.body,
    props: { visible: true, position: { x: 100, y: 80 }, ...props },
    global: {
      plugins: [i18n],
      stubs: {
        CharacterCardPortrait: {
          template: '<img data-testid="portrait" alt="" />',
        },
        CharacterCardImpressionMark: {
          template: '<span data-testid="impression-mark" />',
        },
        WowIcon: {
          template: '<span data-testid="wow-icon" />',
        },
      },
    },
  })
}

describe('CharacterCard story popover', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    i18n.global.locale.value = 'zh-CN'
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    document.body.innerHTML = ''
  })

  it('renders an RPBox card with its portrait, system dossier fields and first impression', () => {
    wrapper = mountCard({ characterCard: rpboxCard, speaker: '历史名字' })

    const popover = document.body.querySelector('[data-testid="story-character-card"]')
    expect(popover?.textContent).toContain('RPBox 人物卡')
    expect(popover?.textContent).toContain('伊莱娅·雾桥')
    expect(popover?.textContent).toContain('暮色信使')
    expect(popover?.textContent).toContain('灰绿色')
    expect(popover?.textContent).toContain('总在观察出口')
    expect(popover?.querySelector('[data-testid="portrait"]')).not.toBeNull()
    expect(popover?.querySelector('[data-testid="impression-mark"]')).not.toBeNull()
    expect(popover?.textContent).not.toContain('编辑本剧情角色')
  })

  it('keeps the legacy TRP3 card editable and emits the existing edit contract', async () => {
    wrapper = mountCard({ character: legacyCharacter })

    expect(document.body.textContent).toContain('TRP3 剧情角色')
    expect(document.body.textContent).toContain('莱格')
    const editButton = Array.from(document.body.querySelectorAll('button'))
      .find(button => button.textContent?.includes('编辑本剧情角色'))
    expect(editButton).toBeDefined()
    editButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('edit')?.[0]).toEqual([legacyCharacter])
  })
})
