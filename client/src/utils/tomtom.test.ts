import { describe, expect, it } from 'vitest'
import { formatTomTomCommand, parseTomTomCommand, parseTomTomCommands } from './tomtom'

describe('TomTom command compatibility', () => {
  it('formats portable map IDs with the hash prefix used by TomTom exports', () => {
    expect(formatTomTomCommand({
      map_id: '47',
      x: 42.6,
      y: 71.3,
      title: '前往守夜营地',
    })).toBe('/way #47 42.60 71.30 前往守夜营地')
  })

  it('parses map IDs, localized zone names and current-zone aliases', () => {
    expect(parseTomTomCommand('/way #47 42.60 71.30 守夜营地')).toEqual({
      map_id: '47',
      zone: '',
      x: 42.6,
      y: 71.3,
      label: '守夜营地',
    })
    expect(parseTomTomCommand('/tway 暮色森林 33.20 44.10 林地入口')).toMatchObject({
      map_id: '',
      zone: '暮色森林',
      x: 33.2,
      y: 44.1,
    })
    expect(parseTomTomCommand('/tomtomway 14.78 23.90 临时坐标')).toMatchObject({
      map_id: '',
      zone: '',
      x: 14.78,
      y: 23.9,
    })
  })

  it('keeps compatibility with the former RPBox numeric-map format', () => {
    expect(parseTomTomCommand('/way 47 73.80 44.50 夜色镇集合')).toMatchObject({
      map_id: '47',
      x: 73.8,
      y: 44.5,
      label: '夜色镇集合',
    })
    expect(parseTomTomCommand('/way 2022 42.10 65.30 巨龙群岛')).toMatchObject({
      map_id: '2022',
      x: 42.1,
      y: 65.3,
    })
  })

  it('preserves route order in multi-point titles and reports rejected lines', () => {
    expect(formatTomTomCommand({ map_id: '#47', x: 10, y: 20, title: '入口' }, { sequence: 1, total: 2 }))
      .toBe('/way #47 10.00 20.00 [1/2] 入口')

    const result = parseTomTomCommands('/way #47 10 20 入口\ninvalid\n/way #47 30 40 终点')
    expect(result.waypoints).toHaveLength(2)
    expect(result.rejected).toEqual(['invalid'])
  })
})
