import { afterEach, describe, expect, it, vi } from 'vitest'
import { getExistingStoryEntrySourceIds } from './story'

describe('story source id lookup', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('deduplicates source ids and queries the bounded endpoint in batches of 500', async () => {
    const fetchMock = vi.fn(async (_url: string, options: RequestInit) => {
      const body = JSON.parse(String(options.body)) as { source_ids: string[] }
      return {
        ok: true,
        status: 200,
        json: async () => ({ source_ids: [body.source_ids[0]] }),
      }
    })
    vi.stubGlobal('fetch', fetchMock)
    const sourceIds = [
      ...Array.from({ length: 501 }, (_, index) => `source-${index}`),
      ' source-0 ',
      '',
    ]

    const existing = await getExistingStoryEntrySourceIds(42, sourceIds)

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock.mock.calls.map(([, options]) => (
      (JSON.parse(String(options.body)) as { source_ids: string[] }).source_ids.length
    ))).toEqual([500, 1])
    expect(fetchMock.mock.calls[0][0]).toContain('/stories/42/entries/existing-source-ids')
    expect(existing).toEqual(['source-0', 'source-500'])
  })
})
