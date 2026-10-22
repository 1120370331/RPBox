import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { parse, compileScript, compileTemplate, compileStyle } from '@vue/compiler-sfc'
import { themes } from '../../../stores/theme.ts'
import { buildNameStyle } from '../../../utils/userNameStyle.ts'
import { guildColorStyle, guildTagStyle } from '../guildColorStyle.ts'

const dir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const repo = path.resolve(dir, '../../../..')
const files = [
  ...fs.readdirSync(dir).filter(f => f.endsWith('.vue')).map(f => path.join(dir, f)),
  ...fs.readdirSync(path.join(dir, 'layouts')).filter(f => f.endsWith('.vue')).map(f => path.join(dir, 'layouts', f)),
]
for (const file of files) {
  const source = fs.readFileSync(file, 'utf8').replace(/\r\n/g, '\n')
  const { descriptor, errors } = parse(source, { filename: file })
  const script = compileScript(descriptor, { id: 'data-v-guild-check' })
  const template = compileTemplate({ source: descriptor.template.content, filename: file, id: 'data-v-guild-check', compilerOptions: { bindingMetadata: script.bindings } })
  const style = compileStyle({ source: descriptor.styles[0].content, filename: file, id: 'data-v-guild-check', scoped: true })
  assert.deepEqual([...errors, ...template.errors, ...style.errors], [])
  const themeRules = style.code.split('\n').filter(line => line.includes('[data-theme="black-gold"]'))
  assert(themeRules.every(line => line.includes('[data-v-guild-check]')), 'Native-control rules must remain scoped to guild pages')

  const relative = path.relative(repo, file).replaceAll('\\', '/')
  const baseline = execFileSync('git', ['show', `HEAD:${relative}`], { cwd: repo, encoding: 'utf8' }).replace(/\r\n/g, '\n')
  const originalScript = parse(baseline).descriptor.scriptSetup.content
  const currentScript = descriptor.scriptSetup.content
    .replace(/^import \{ guild(?:Color|Tag)Style \} from ['"].*['"]\n/gm, '')
    .replace('return guildColorStyle(guild.color)', 'return guild.color ? { background: `#${guild.color}` } : {}')
  assert.equal(currentScript, originalScript, 'API payloads, saved colors and public script contracts must remain unchanged')
  assert.deepEqual(source.match(/buildNameStyle\([^)]*\)/g), baseline.match(/buildNameStyle\([^)]*\)/g))
  console.log(`PASS compile, scoped theme rules and script contracts: ${path.relative(dir, file)}`)
}

function luminance(hex) {
  const rgb = hex.replace('#', '').match(/../g).map(x => parseInt(x, 16) / 255).map(c => c <= .04045 ? c / 12.92 : ((c + .055) / 1.055) ** 2.4)
  return rgb[0] * .2126 + rgb[1] * .7152 + rgb[2] * .0722
}
function contrast(a, b) {
  const x = luminance(a), y = luminance(b)
  return (Math.max(x, y) + .05) / (Math.min(x, y) + .05)
}
for (const value of ['000000', 'FFFFFF', 'B87333', 'D4AF37', '8FD0FF', '126260', '804030', '777777']) {
  const style = guildColorStyle(value)
  assert.equal(style.background, '#' + value)
  assert(contrast(style.background, style.color) >= 4.5)
  assert.deepEqual(guildTagStyle(value), { '--guild-tag-color': '#' + value })
  assert.deepEqual(guildTagStyle(value, true), { '--guild-tag-color': '#' + value, ...style })
  console.log(`PASS saved brand ${value}, foreground contrast ${contrast(style.background, style.color).toFixed(2)}`)
}
for (const value of [undefined, '', 'bad color', '12345678']) assert.deepEqual(guildColorStyle(value), {})
assert.deepEqual(guildColorStyle('#000000'), { background: '#000000', color: '#ffffff' })
console.log('PASS missing/invalid colors and optional prefix')

for (const theme of themes) {
  const c = theme.colors
  const pairs = [[c.panelBg, c.textMain], [c.cardBg, c.textMain], [c.btnPrimaryBg, c.btnPrimaryText], [c.gradientStart, c.gradientText], [c.gradientEnd, c.gradientText]]
  assert(pairs.every(([a, b]) => contrast(a, b) >= 4.5))
  console.log(`PASS theme palette ${theme.id}: ${pairs.map(([a, b]) => contrast(a, b).toFixed(2)).join(', ')}`)
}

const dark = themes.find(theme => theme.id === 'black-gold')
const dyedName = buildNameStyle('#000000', true)
console.log(`PUBLIC DEPENDENCY: buildNameStyle black name on dark panel, contrast ${contrast(dyedName.color, dark.colors.panelBg).toFixed(2)}; requires W1/integrated acceptance`)
console.log('Static checks do not certify browser rendering, beta installation or real API persistence.')
