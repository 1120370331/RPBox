// One-shot Worker diagnostics; run from client/. This is not installed-app acceptance.
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'
import { parse, compileScript, compileTemplate, compileStyle } from '@vue/compiler-sfc'

const directory = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(directory, '../../../..')
const manifest = JSON.parse(fs.readFileSync(path.join(directory, 'manifest.json'), 'utf8'))
for (const file of manifest.sourceFiles) {
  const bytes = fs.readFileSync(path.join(root, file.path))
  assert.equal(createHash('sha256').update(bytes).digest('hex'), file.sha256, `Candidate changed: ${file.path}`)
}
let count = 0
for (const file of manifest.sourceFiles.filter(file => file.path.endsWith('.vue'))) {
  const absolute = path.join(root, file.path)
  const { descriptor, errors } = parse(fs.readFileSync(absolute, 'utf8'), { filename: absolute })
  assert.deepEqual(errors, [])
  const script = compileScript(descriptor, {
    id: 'r52-worker', fs: { fileExists: fs.existsSync, readFile: file => fs.readFileSync(file, 'utf8') },
  })
  const template = compileTemplate({
    source: descriptor.template.content, filename: absolute, id: 'r52-worker',
    compilerOptions: { bindingMetadata: script.bindings },
  })
  assert.deepEqual(template.errors, [])
  for (const style of descriptor.styles) {
    assert.deepEqual(compileStyle({ source: style.content, filename: absolute, id: 'r52-worker', scoped: style.scoped }).errors, [])
  }
  count++
}
console.log(`PASS ${count} changed Vue SFC scripts, templates and styles; source hashes match`)

// Execute the real API source with a mocked request adapter to check response shape preservation.
let response
const exports = {}
const source = fs.readFileSync(path.join(root, 'client/src/api/accountBackup.ts'), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 },
}).outputText
vm.runInNewContext(compiled, {
  exports, require: specifier => {
    assert.equal(specifier, './request')
    return { request: { get: async () => response, put: async () => response } }
  },
})
const version = Object.freeze({ id: 1, backup_id: 2, version: 3, checksum: 'saved', change_log: '', created_at: '2026-10-06T00:00:00Z' })
for (const data of [version, { version }, { data: version }, { data: { version } }]) {
  response = data
  assert.equal(await exports.getAccountBackupVersion('a', 1), version)
  assert.equal(await exports.renameAccountBackupVersion('a', 1, 'display'), version)
}
assert.equal(version.version, 3)
console.log('PASS 8 mocked-adapter source-module response assertions; saved version object unchanged')
