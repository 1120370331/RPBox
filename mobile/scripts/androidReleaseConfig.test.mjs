import assert from 'node:assert/strict'
import test from 'node:test'

import {
  checkApiReachability,
  resolveAndroidVersion,
  resolveHealthUrl,
  resolveRpdbProbeUrl,
  validateApiBase,
  validateReleaseBase,
} from './androidReleaseConfig.mjs'

test('accepts verified production URLs and derives the root health endpoint', () => {
  assert.equal(validateApiBase('https://api.rpbox.app/api/v1'), 'https://api.rpbox.app/api/v1')
  assert.equal(
    validateApiBase('https://ksxvodevhonx.sealosbja.site/api/v1'),
    'https://ksxvodevhonx.sealosbja.site/api/v1',
  )
  assert.equal(
    validateReleaseBase('https://ksxvodevhonx.sealosbja.site/releases/mobile/'),
    'https://ksxvodevhonx.sealosbja.site/releases/mobile',
  )
  assert.equal(resolveHealthUrl('https://api.rpbox.app/api/v1'), 'https://api.rpbox.app/health')
  assert.equal(
    resolveRpdbProbeUrl('https://api.rpbox.app/api/v1'),
    'https://api.rpbox.app/api/v1/rpdb/works?page=1&page_size=1',
  )
})

test('rejects non-production API and release targets', () => {
  const rejectedApiBases = [
    'http://api.rpbox.app/api/v1',
    'https://localhost/api/v1',
    'https://127.0.0.1/api/v1',
    'https://10.1.2.3/api/v1',
    'https://staging.rpbox.app/api/v1',
    'https://api-staging.rpbox.app/api/v1',
    'https://user:pass@api.rpbox.app/api/v1',
    'https://api.rpbox.app:8443/api/v1',
    'https://api.rpbox.app/api/v2',
    'https://api.rpbox.app/api/v1?debug=1',
  ]

  for (const value of rejectedApiBases) {
    assert.throws(() => validateApiBase(value), /\[Android Release\]/)
  }

  assert.throws(
    () => validateReleaseBase('https://api.rpbox.app/releases'),
    /must end with \/mobile/,
  )
  assert.throws(
    () => validateReleaseBase('https://preview.rpbox.app/releases/mobile'),
    /public production hostname/,
  )
})

test('maps strict semver to a bounded monotonic Android versionCode', () => {
  assert.deepEqual(resolveAndroidVersion('2.0.3'), { version: '2.0.3', versionCode: 2_000_003 })
  assert.deepEqual(resolveAndroidVersion('2147.483.647'), {
    version: '2147.483.647',
    versionCode: 2_147_483_647,
  })
  assert.throws(() => resolveAndroidVersion('2.0'), /strict major\.minor\.patch/)
  assert.throws(() => resolveAndroidVersion('2.1000.0'), /between 0 and 999/)
  assert.throws(() => resolveAndroidVersion('2148.0.0'), /between 1 and 2147483647/)
})

test('requires successful production health and RPDB data payloads', async () => {
  const requestedUrls = []
  await checkApiReachability('https://api.rpbox.app/api/v1', {
    attempts: 1,
    fetchImpl: async (url) => {
      requestedUrls.push(url)
      return {
        ok: true,
        json: async () => url.endsWith('/health') ? { status: 'ok' } : { works: [] },
      }
    },
  })
  assert.deepEqual(requestedUrls, [
    'https://api.rpbox.app/health',
    'https://api.rpbox.app/api/v1/rpdb/works?page=1&page_size=1',
  ])

  await assert.rejects(
    checkApiReachability('https://api.rpbox.app/api/v1', {
      attempts: 2,
      fetchImpl: async () => {
        throw new Error('dns unavailable')
      },
      wait: async () => {},
    }),
    /refusing to build a release/,
  )

  await assert.rejects(
    checkApiReachability('https://api.rpbox.app/api/v1', {
      attempts: 1,
      fetchImpl: async (url) => ({
        ok: true,
        json: async () => url.endsWith('/health') ? { status: 'ok' } : { works: null },
      }),
    }),
    /health\/data preflight failed/,
  )
})
