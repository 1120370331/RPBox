import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import type { AccountChatLogs, ChatRecord } from '@/types/chatLog'
import StagingPool from './StagingPool.vue'

const invoke = vi.hoisted(() => vi.fn())

vi.mock('@tauri-apps/api/core', () => ({ invoke }))

function makeRecord(overrides: Partial<ChatRecord>): ChatRecord {
  return {
    record_key: 'rpbox-default',
    account_id: 'ACCOUNT-A',
    timestamp: 1_753_200_000,
    channel: 'SAY',
    sender: { gameID: 'Player-Realm' },
    content: 'line',
    identity_source: 'game_id',
    ...overrides,
  }
}

function logs(records: ChatRecord[]): AccountChatLogs[] {
  return [{
    account_id: 'ACCOUNT-A',
    last_update: records.at(-1)?.timestamp || null,
    record_count: records.length,
    records,
  }]
}

async function mountPool(records: ChatRecord[]) {
  invoke.mockResolvedValue(logs(records))
  const wrapper = mount(StagingPool, {
    global: {
      plugins: [i18n],
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('StagingPool', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('wow_path', 'C:\\Games\\World of Warcraft')
    invoke.mockReset()
  })

  it('selects and archives same-second records by stable key', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-first', content: 'first line' }),
      makeRecord({ record_key: 'rpbox-second', content: 'second line' }),
    ])
    const items = wrapper.findAll('.record-item')
    expect(items).toHaveLength(2)
    const firstLine = items.find(item => item.text().includes('first line'))
    expect(firstLine).toBeTruthy()

    await firstLine!.trigger('click')
    expect(wrapper.find('.staging-footer').text()).toContain('1')
    await wrapper.find('.staging-footer button').trigger('click')

    const archived = wrapper.emitted<[ChatRecord[]]>('archive')?.[0]?.[0]
    expect(archived).toHaveLength(1)
    expect(archived?.[0].record_key).toBe('rpbox-first')

    ;(wrapper.vm as unknown as { removeArchivedRecords: (keys: string[]) => void })
      .removeArchivedRecords(['rpbox-first'])
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('second line')
  })

  it('archives only selected records that remain visible after filtering', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-visible', content: 'visible needle' }),
      makeRecord({ record_key: 'rpbox-hidden', content: 'hidden haystack' }),
    ])
    for (const item of wrapper.findAll('.record-item')) await item.trigger('click')
    expect(wrapper.find('.staging-footer').text()).toContain('2')

    await wrapper.get('.search-field input').setValue('visible needle')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.find('.staging-footer').text()).toContain('1')

    await wrapper.find('.staging-footer button').trigger('click')
    const archived = wrapper.emitted<[ChatRecord[]]>('archive')?.[0]?.[0]
    expect(archived).toHaveLength(1)
    expect(archived?.[0].record_key).toBe('rpbox-visible')
  })

  it('combines speaker and listener profile filters with strict legacy semantics', async () => {
    const wrapper = await mountPool([
      makeRecord({
        record_key: 'rpbox-a-viewer',
        content: 'matching line',
        profile_snapshot_id: 'speaker-a',
        profile_snapshot: { n: 'Speaker A', pn: 'Card A' },
        identity_source: 'snapshot',
        listeners: [{
          gameID: 'Viewer-Realm',
          snapshot_id: 'viewer-a',
          snapshot: { n: 'Viewer A', pn: 'View Card A' },
        }],
      }),
      makeRecord({
        record_key: 'rpbox-b-viewer',
        content: 'other line',
        profile_snapshot_id: 'speaker-b',
        profile_snapshot: { n: 'Speaker B', pn: 'Card B' },
        identity_source: 'snapshot',
        listeners: [{
          gameID: 'Other-Realm',
          snapshot_id: 'viewer-b',
          snapshot: { n: 'Viewer B', pn: 'View Card B' },
        }],
      }),
      makeRecord({
        record_key: 'rpbox-legacy',
        content: 'legacy without listener metadata',
      }),
    ])

    const profileButtons = wrapper.findAll('.profile-option')
    const speakerA = profileButtons.find(button => button.text().includes('Speaker A'))
    const viewerA = profileButtons.find(button => button.text().includes('Viewer A'))
    expect(speakerA).toBeTruthy()
    expect(viewerA).toBeTruthy()

    await speakerA!.trigger('click')
    await viewerA!.trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('matching line')
    expect(wrapper.text()).not.toContain('legacy without listener metadata')
  })

  it('renders profile switches as a dedicated identity timeline node', async () => {
    const wrapper = await mountPool([
      makeRecord({
        record_key: 'rpbox-switch',
        mark: 'S',
        channel: 'SYSTEM',
        event: {
          kind: 'profile_switch',
          certainty: 'exact',
          from: { snapshot_id: 'before', display_name: 'Before Name' },
          to: { snapshot_id: 'after', display_name: 'After Name' },
        },
      }),
    ])

    const node = wrapper.get('.record-item.identity-event')
    expect(node.text()).toContain('Before Name')
    expect(node.text()).toContain('After Name')
    expect(node.text()).toContain('Profile switched')
  })

  it('normalizes inbound and outbound whispers and exposes the guild channel', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-whisper', channel: 'WHISPER_IN', content: 'private line' }),
      makeRecord({ record_key: 'rpbox-guild', channel: 'GUILD', content: 'guild line' }),
    ])
    const chips = wrapper.findAll('.filter-chip')
    const whisper = chips.find(button => button.text() === 'Whisper')
    const guild = chips.find(button => button.text() === 'Guild')
    expect(whisper).toBeTruthy()
    expect(guild).toBeTruthy()

    await whisper!.trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('private line')
    expect(wrapper.text()).not.toContain('guild line')
  })

  it('groups snapshot revisions by TRP3 profile ref and matches every historical name', async () => {
    const wrapper = await mountPool([
      makeRecord({
        record_key: 'rpbox-card-rev-1',
        ref_id: 'profile-a',
        profile_snapshot_id: 'snapshot-a-1',
        profile_snapshot: { ref: 'profile-a', n: 'Before Rename', pn: 'Main Card', rev: 1 },
        identity_source: 'snapshot',
        content: 'before revision',
      }),
      makeRecord({
        record_key: 'rpbox-card-rev-2',
        ref_id: 'profile-a',
        profile_snapshot_id: 'snapshot-a-2',
        profile_snapshot: { ref: 'profile-a', n: 'After Rename', pn: 'Main Card', rev: 2 },
        identity_source: 'snapshot',
        content: 'after revision',
      }),
    ])

    const profileButtons = wrapper.findAll('.profile-option')
    expect(profileButtons).toHaveLength(1)
    expect(profileButtons[0].text()).toContain('Main Card')

    await profileButtons[0].trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(2)
    expect(wrapper.text()).toContain('Before Rename')
    expect(wrapper.text()).toContain('After Rename')
  })
})
