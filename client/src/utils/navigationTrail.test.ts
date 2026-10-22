import { describe, expect, it } from 'vitest'
import { advanceNavigationTrail, defaultNavigationTrail, safeNavigationPath } from './navigationTrail'

describe('application navigation hierarchy', () => {
  it('keeps the RPBox source across modules and trims ancestors without return loops', () => {
    let trail = defaultNavigationTrail('/tools?q=RPBox&tag=RPBox')
    for (const path of ['/tools/rpbox', '/sync', '/sync/profile/12', '/sync/profile/12/edit']) trail = advanceNavigationTrail(trail, path)
    expect(trail.map(step => step.path)).toEqual(['/', '/tools?q=RPBox&tag=RPBox', '/tools/rpbox', '/sync', '/sync/profile/12', '/sync/profile/12/edit'])
    trail = advanceNavigationTrail(trail, '/sync')
    expect(trail.at(-2)?.path).toBe('/tools/rpbox')
    trail = advanceNavigationTrail(trail, '/tools/rpbox')
    expect(trail.at(-2)?.path).toBe('/tools?q=RPBox&tag=RPBox')
  })
  it('retains a source editor when opening its archive, then removes the cycle on return', () => {
    const editor = defaultNavigationTrail('/tools/trp3/9')
    const archive = advanceNavigationTrail(editor, '/character-cards/9')
    expect(archive.at(-2)?.path).toBe('/tools/trp3/9')
    expect(advanceNavigationTrail(archive, '/tools/trp3/9')).toEqual(editor)
    expect(advanceNavigationTrail(editor, '/tools/trp3/10').map(step => step.path)).not.toContain('/tools/trp3/9')
  })
  it('provides explicit parents for deep links and refresh without relying on browser history', () => {
    expect(defaultNavigationTrail('/tools/rpbox/install').at(-2)?.path).toBe('/tools/rpbox')
    expect(defaultNavigationTrail('/sync/profile/12/edit').at(-2)?.path).toBe('/sync/profile/12')
    expect(defaultNavigationTrail('/archives/story/5').at(-2)?.path).toBe('/archives')
    expect(safeNavigationPath('//evil.example/path')).toBe(false)
    expect(safeNavigationPath('/\\evil.example')).toBe(false)
    expect(safeNavigationPath('/login?redirect=/tools')).toBe(false)
  })
  it('does not return to a backup page that immediately redirects to required setup', () => {
    const previous = advanceNavigationTrail(defaultNavigationTrail('/tools/rpbox'), '/sync')
    const setup = advanceNavigationTrail(previous, '/sync/setup?redirect=/sync&required=1')
    expect(setup.at(-2)?.path).toBe('/tools/rpbox')
    expect(setup.map(step => step.path)).not.toContain('/sync')
    expect(advanceNavigationTrail(setup, '/sync').at(-2)?.path).toBe('/tools/rpbox')
    expect(defaultNavigationTrail('/sync/setup').at(-2)?.path).toBe('/tools/rpbox')
  })
})
