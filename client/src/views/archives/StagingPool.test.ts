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

async function mountPool(records: ChatRecord[], accountLogs = logs(records)) {
  invoke.mockResolvedValue(accountLogs)
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
    await wrapper.find('.staging-footer .r-button').trigger('click')

    const archived = wrapper.emitted<[ChatRecord[]]>('archive')?.[0]?.[0]
    expect(archived).toHaveLength(1)
    expect(archived?.[0].record_key).toBe('rpbox-first')

    ;(wrapper.vm as unknown as { removeArchivedRecords: (keys: string[]) => void })
      .removeArchivedRecords(['rpbox-first'])
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('second line')
  })

  it('drops selections hidden by filtering before archive', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-visible', content: 'visible needle' }),
      makeRecord({ record_key: 'rpbox-hidden', content: 'hidden haystack' }),
    ])
    for (const item of wrapper.findAll('.record-item')) await item.trigger('click')
    expect(wrapper.find('.staging-footer').text()).toContain('2')

    await wrapper.get('.search-field input').setValue('visible needle')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.find('.staging-footer').text()).toContain('1')

    await wrapper.find('.staging-footer .r-button').trigger('click')
    const archived = wrapper.emitted<[ChatRecord[]]>('archive')?.[0]?.[0]
    expect(archived).toHaveLength(1)
    expect(archived?.[0].record_key).toBe('rpbox-visible')

    await wrapper.get('.rail-heading .clear-button').trigger('click')
    expect(wrapper.findAll('.record-item.selected')).toHaveLength(1)
    expect(wrapper.find('.staging-footer').text()).toContain('1')
  })

  it('bulk-selects, inverts, and clears only the current matches', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-one', content: 'one' }),
      makeRecord({ record_key: 'rpbox-two', content: 'two' }),
      makeRecord({ record_key: 'rpbox-three', content: 'three' }),
    ])
    const actions = wrapper.findAll('.bulk-actions button')

    await actions[0].trigger('click')
    expect(wrapper.findAll('.record-item.selected')).toHaveLength(3)

    await wrapper.findAll('.record-item')[0].trigger('click')
    await actions[1].trigger('click')
    expect(wrapper.findAll('.record-item.selected')).toHaveLength(1)

    await actions[2].trigger('click')
    expect(wrapper.findAll('.record-item.selected')).toHaveLength(0)
    expect(wrapper.find('.staging-footer').exists()).toBe(false)
  })

  it('selects a contiguous visible range with shift click', async () => {
    const wrapper = await mountPool([
      makeRecord({ record_key: 'rpbox-range-a', timestamp: 1_753_200_000, content: 'range a' }),
      makeRecord({ record_key: 'rpbox-range-b', timestamp: 1_753_200_001, content: 'range b' }),
      makeRecord({ record_key: 'rpbox-range-c', timestamp: 1_753_200_002, content: 'range c' }),
    ])
    const items = wrapper.findAll('.record-item')

    await items[0].trigger('click')
    await items[2].trigger('click', { shiftKey: true })

    expect(wrapper.findAll('.record-item.selected')).toHaveLength(3)
    expect(wrapper.find('.staging-footer').text()).toContain('3')
  })

  it('pages long hours without growing the rendered record budget', async () => {
    const records = Array.from({ length: 150 }, (_, index) => makeRecord({
      record_key: `rpbox-long-hour-${index}`,
      timestamp: 1_753_200_000 + index,
      sequence: index,
      content: `line ${index}`,
    }))
    const wrapper = await mountPool(records)

    expect(wrapper.findAll('.record-item')).toHaveLength(120)
    expect(wrapper.get('.hour-pagination').text()).toContain('120')
    expect(wrapper.get('.hour-pagination').text()).toContain('150')

    await wrapper.get('.next-hour-page').trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(30)
    expect(wrapper.get('.hour-pagination').text()).toContain('121')
    expect(wrapper.get('.previous-hour-page').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('.next-hour-page').attributes('disabled')).toBeDefined()
  })

  it('clamps the active hour page after archived records shrink the result set', async () => {
    const records = Array.from({ length: 250 }, (_, index) => makeRecord({
      record_key: `rpbox-clamp-hour-${index}`,
      timestamp: 1_753_200_000 + index,
      sequence: index,
      content: `clamp line ${index}`,
    }))
    const wrapper = await mountPool(records)

    await wrapper.get('.next-hour-page').trigger('click')
    await wrapper.get('.next-hour-page').trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(10)

    ;(wrapper.vm as unknown as { removeArchivedRecords: (keys: string[]) => void })
      .removeArchivedRecords(Array.from({ length: 10 }, (_, index) => `rpbox-clamp-hour-${index}`))
    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('.record-item')).toHaveLength(120)
    expect(wrapper.get('.hour-pagination').text()).toContain('121')
    expect(wrapper.get('.hour-pagination').text()).toContain('240')
  })

  it('keeps one paged hour open so total rendered records stay globally bounded', async () => {
    const firstHour = Array.from({ length: 150 }, (_, index) => makeRecord({
      record_key: `rpbox-hour-a-${index}`,
      timestamp: 1_753_200_000 + index,
      sequence: index,
      content: `hour a ${index}`,
    }))
    const secondHour = Array.from({ length: 150 }, (_, index) => makeRecord({
      record_key: `rpbox-hour-b-${index}`,
      timestamp: 1_753_203_600 + index,
      sequence: index,
      content: `hour b ${index}`,
    }))
    const wrapper = await mountPool([...firstHour, ...secondHour])

    expect(wrapper.findAll('.record-item').length).toBeLessThanOrEqual(120)
    const closedHour = wrapper.findAll('.hour-header')
      .find(header => header.find('.ri-arrow-right-s-line').exists())
    await closedHour!.findAll('button')[1].trigger('click')

    expect(wrapper.findAll('.record-item').length).toBeLessThanOrEqual(120)
    expect(wrapper.findAll('.records')).toHaveLength(1)
  })

  it('limits shift-range selection to the currently rendered records', async () => {
    const records = Array.from({ length: 150 }, (_, index) => makeRecord({
      record_key: `rpbox-shift-batch-${index}`,
      timestamp: 1_753_210_000 + index,
      sequence: index,
      content: `range line ${index}`,
    }))
    const wrapper = await mountPool(records)
    const items = wrapper.findAll('.record-item')

    await items[0].trigger('click')
    await items[119].trigger('click', { shiftKey: true })

    expect(wrapper.findAll('.record-item.selected')).toHaveLength(120)
    expect(wrapper.find('.staging-footer').text()).toContain('120')
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

  it('filters by account, record type, and a quick date range', async () => {
    const now = new Date()
    now.setHours(12, 0, 0, 0)
    const old = new Date(now)
    old.setDate(old.getDate() - 10)
    const recentNpc = makeRecord({
      record_key: 'rpbox-account-b-npc',
      account_id: 'ACCOUNT-B',
      timestamp: Math.floor(now.getTime() / 1000),
      mark: 'N',
      npc: 'Innkeeper',
      content: 'recent npc line',
    })
    const oldBackground = makeRecord({
      record_key: 'rpbox-account-b-old',
      account_id: 'ACCOUNT-B',
      timestamp: Math.floor(old.getTime() / 1000),
      mark: 'B',
      content: 'old scene line',
    })
    const accountA = makeRecord({
      record_key: 'rpbox-account-a',
      timestamp: Math.floor(now.getTime() / 1000),
      content: 'account a line',
    })
    const accountLogs: AccountChatLogs[] = [
      { account_id: 'ACCOUNT-A', last_update: accountA.timestamp, record_count: 1, records: [accountA] },
      {
        account_id: 'ACCOUNT-B',
        last_update: recentNpc.timestamp,
        record_count: 2,
        records: [recentNpc, oldBackground],
      },
    ]
    const wrapper = await mountPool([accountA, recentNpc, oldBackground], accountLogs)
    const accountB = wrapper.findAll('.account-chip')
      .find(button => button.text().includes('ACCOUNT-B'))

    await accountB!.trigger('click')
    expect(wrapper.get('.selection-ruler').text()).toContain('2')

    const recentWeek = wrapper.findAll('.date-presets button')
      .find(button => button.text().includes('Last 7 days'))
    await recentWeek!.trigger('click')
    expect(wrapper.get('.selection-ruler').text()).toContain('1')
    expect(wrapper.text()).toContain('recent npc line')
    expect(wrapper.text()).not.toContain('old scene line')

    const npcKind = wrapper.findAll('.filter-chip')
      .find(button => button.text() === 'NPC dialogue')
    await npcKind!.trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('Innkeeper')
  })

  it('searches profile options, shows unarchived counts, and removes archived-only options', async () => {
    const wrapper = await mountPool([
      makeRecord({
        record_key: 'rpbox-profile-a-1',
        ref_id: 'profile-a',
        profile_snapshot_id: 'snapshot-a-1',
        profile_snapshot: { ref: 'profile-a', n: 'Alice One', pn: 'Alice Card' },
        identity_source: 'snapshot',
      }),
      makeRecord({
        record_key: 'rpbox-profile-a-2',
        ref_id: 'profile-a',
        profile_snapshot_id: 'snapshot-a-2',
        profile_snapshot: { ref: 'profile-a', n: 'Alice Two', pn: 'Alice Card' },
        identity_source: 'snapshot',
      }),
      makeRecord({
        record_key: 'rpbox-profile-b',
        ref_id: 'profile-b',
        profile_snapshot_id: 'snapshot-b',
        profile_snapshot: { ref: 'profile-b', n: 'Bram', pn: 'Bram Card' },
        identity_source: 'snapshot',
      }),
    ])
    const speakerSection = wrapper.findAll('.filter-section')
      .find(section => section.text().includes('Speaker profile'))!
    const search = speakerSection.get('.profile-search input')

    await search.setValue('Alice')
    expect(speakerSection.findAll('.profile-option')).toHaveLength(1)
    expect(speakerSection.get('.profile-option').text()).toContain('2')

    ;(wrapper.vm as unknown as { removeArchivedRecords: (keys: string[]) => void })
      .removeArchivedRecords(['rpbox-profile-a-1', 'rpbox-profile-a-2'])
    await wrapper.vm.$nextTick()
    expect(speakerSection.findAll('.profile-option')).toHaveLength(0)
    expect(speakerSection.text()).toContain('No profile options match')

    await search.setValue('')
    expect(speakerSection.findAll('.profile-option')).toHaveLength(1)
    expect(speakerSection.text()).toContain('Bram Card')
  })

  it('keeps a selected profile visible when the bounded option list is restored', async () => {
    const records = Array.from({ length: 161 }, (_, index) => {
      const suffix = String(index).padStart(3, '0')
      return makeRecord({
        record_key: `rpbox-profile-${suffix}`,
        ref_id: `profile-${suffix}`,
        profile_snapshot_id: `snapshot-${suffix}`,
        profile_snapshot: {
          ref: `profile-${suffix}`,
          n: `Character ${suffix}`,
          pn: `Card ${suffix}`,
        },
        identity_source: 'snapshot',
      })
    })
    const wrapper = await mountPool(records)
    const speakerSection = wrapper.findAll('.filter-section')
      .find(section => section.text().includes('Speaker profile'))!
    const search = speakerSection.get('.profile-search input')

    await search.setValue('Card 160')
    await speakerSection.get('.profile-option').trigger('click')
    await search.setValue('')

    expect(speakerSection.findAll('.profile-option')).toHaveLength(160)
    expect(speakerSection.text()).toContain('Card 160')
  })

  it('keeps groups collapsed until a filter opens only its first match', async () => {
    const today = new Date()
    today.setHours(12, 0, 0, 0)
    const yesterday = new Date(today)
    yesterday.setDate(yesterday.getDate() - 1)
    const wrapper = await mountPool([
      makeRecord({
        record_key: 'rpbox-new-day',
        timestamp: Math.floor(today.getTime() / 1000),
        content: 'newest unique line',
      }),
      makeRecord({
        record_key: 'rpbox-old-day',
        timestamp: Math.floor(yesterday.getTime() / 1000),
        content: 'old unique line',
      }),
    ])

    const sort = wrapper.findAll('.view-actions button')[0]
    await sort.trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('old unique line')

    const collapse = wrapper.findAll('.view-actions button')[1]
    await collapse.trigger('click')
    expect(wrapper.findAll('.record-item')).toHaveLength(0)

    await wrapper.get('.search-field input').setValue('old unique line')
    expect(wrapper.findAll('.record-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('old unique line')
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
