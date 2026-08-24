import { isIP } from 'node:net'
import { pathToFileURL } from 'node:url'

const FORBIDDEN_ENVIRONMENT_LABELS = new Set([
  'dev',
  'development',
  'local',
  'localhost',
  'preview',
  'qa',
  'sandbox',
  'stage',
  'staging',
  'test',
  'testing',
])

function fail(message) {
  throw new Error(`[Android Release] ${message}`)
}

function normalizeHostname(hostname) {
  return hostname.toLowerCase().replace(/^\[|\]$/g, '').replace(/\.+$/, '')
}

function isForbiddenIpv4(hostname) {
  const octets = hostname.split('.').map(Number)
  if (octets.length !== 4 || octets.some((value) => !Number.isInteger(value) || value < 0 || value > 255)) {
    return true
  }

  const [first, second] = octets
  return (
    first === 0 ||
    first === 10 ||
    first === 127 ||
    (first === 100 && second >= 64 && second <= 127) ||
    (first === 169 && second === 254) ||
    (first === 172 && second >= 16 && second <= 31) ||
    (first === 192 && second === 168) ||
    first >= 224
  )
}

function isForbiddenIpv6(hostname) {
  const normalized = hostname.toLowerCase()
  return (
    normalized === '::' ||
    normalized === '::1' ||
    normalized.startsWith('fc') ||
    normalized.startsWith('fd') ||
    /^fe[89ab]/.test(normalized) ||
    normalized.startsWith('::ffff:127.') ||
    normalized.startsWith('::ffff:7f')
  )
}

function isForbiddenHostname(hostname) {
  if (!hostname) return true
  if (hostname === 'localhost' || hostname.endsWith('.localhost') || hostname === 'localhost.localdomain') return true

  const ipVersion = isIP(hostname)
  if (ipVersion === 4) return isForbiddenIpv4(hostname)
  if (ipVersion === 6) return isForbiddenIpv6(hostname)

  const labels = hostname.split('.')
  return labels.some((label) =>
    label.split('-').some((part) => FORBIDDEN_ENVIRONMENT_LABELS.has(part)),
  )
}

function validatePublicHttpsUrl(rawValue, label) {
  const raw = String(rawValue ?? '')
  if (!raw || raw !== raw.trim() || /[\r\n]/.test(raw)) {
    fail(`${label} must be a non-empty URL without surrounding whitespace`)
  }

  let parsed
  try {
    parsed = new URL(raw)
  } catch {
    fail(`${label} must be an absolute URL`)
  }

  const hostname = normalizeHostname(parsed.hostname)
  if (parsed.protocol !== 'https:') fail(`${label} must use HTTPS`)
  if (parsed.username || parsed.password) fail(`${label} must not contain credentials`)
  if (parsed.port && parsed.port !== '443') fail(`${label} must use the standard HTTPS port`)
  if (parsed.search || parsed.hash) fail(`${label} must not contain a query string or fragment`)
  if (isForbiddenHostname(hostname)) fail(`${label} must target a public production hostname`)

  parsed.pathname = parsed.pathname.replace(/\/{2,}/g, '/').replace(/\/$/, '') || '/'
  return parsed
}

export function validateApiBase(rawValue) {
  const parsed = validatePublicHttpsUrl(rawValue, 'VITE_API_BASE')
  if (parsed.pathname !== '/api/v1') {
    fail('VITE_API_BASE must end with the exact /api/v1 path')
  }
  return parsed.toString().replace(/\/$/, '')
}

export function validateReleaseBase(rawValue) {
  const parsed = validatePublicHttpsUrl(rawValue, 'Android release base URL')
  if (!parsed.pathname.endsWith('/mobile')) {
    fail('Android release base URL must end with /mobile')
  }
  return parsed.toString().replace(/\/$/, '')
}

export function resolveHealthUrl(rawApiBase) {
  const apiBase = new URL(validateApiBase(rawApiBase))
  return new URL('/health', apiBase.origin).toString()
}

export function resolveRpdbProbeUrl(rawApiBase) {
  const apiBase = validateApiBase(rawApiBase)
  return `${apiBase}/rpdb/works?page=1&page_size=1`
}

export function resolveAndroidVersion(rawVersion) {
  const version = String(rawVersion ?? '').trim()
  const match = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.exec(version)
  if (!match) fail('release version must be strict major.minor.patch semver')

  const [, rawMajor, rawMinor, rawPatch] = match
  const major = Number(rawMajor)
  const minor = Number(rawMinor)
  const patch = Number(rawPatch)
  if (minor > 999 || patch > 999) {
    fail('minor and patch versions must be between 0 and 999')
  }

  const versionCode = major * 1_000_000 + minor * 1_000 + patch
  if (!Number.isSafeInteger(versionCode) || versionCode < 1 || versionCode > 2_147_483_647) {
    fail('calculated Android versionCode must be between 1 and 2147483647')
  }

  return { version, versionCode }
}

export async function checkApiReachability(rawApiBase, options = {}) {
  const {
    attempts = 3,
    fetchImpl = globalThis.fetch,
    timeoutMs = 15_000,
    wait = (delayMs) => new Promise((resolve) => setTimeout(resolve, delayMs)),
  } = options

  if (typeof fetchImpl !== 'function') fail('fetch is unavailable for the production API preflight')
  const healthUrl = resolveHealthUrl(rawApiBase)
  const rpdbProbeUrl = resolveRpdbProbeUrl(rawApiBase)

  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), timeoutMs)
    try {
      const response = await fetchImpl(healthUrl, {
        headers: { accept: 'application/json' },
        redirect: 'error',
        signal: controller.signal,
      })
      if (!response.ok) throw new Error('non-success response')
      const body = await response.json()
      if (body?.status !== 'ok') throw new Error('unexpected health payload')

      const rpdbResponse = await fetchImpl(rpdbProbeUrl, {
        headers: { accept: 'application/json' },
        redirect: 'error',
        signal: controller.signal,
      })
      if (!rpdbResponse.ok) throw new Error('non-success RPDB response')
      const rpdbBody = await rpdbResponse.json()
      if (!rpdbBody || typeof rpdbBody !== 'object' || !Array.isArray(rpdbBody.works)) {
        throw new Error('unexpected RPDB payload')
      }
      return
    } catch {
      if (attempt === attempts) {
        fail('production API health/data preflight failed; refusing to build a release')
      }
      await wait(1_000 * attempt)
    } finally {
      clearTimeout(timeout)
    }
  }
}

async function runCli() {
  const command = process.argv[2]
  if (command === 'validate-api-base') {
    process.stdout.write(validateApiBase(process.env.VITE_API_BASE))
    return
  }
  if (command === 'validate-release-base') {
    process.stdout.write(validateReleaseBase(process.env.ANDROID_RELEASE_BASE_URL))
    return
  }
  if (command === 'version-code') {
    process.stdout.write(String(resolveAndroidVersion(process.env.ANDROID_RELEASE_VERSION).versionCode))
    return
  }
  if (command === 'check-api-reachability') {
    await checkApiReachability(process.env.VITE_API_BASE)
    process.stdout.write('[Android Release] production API preflight passed\n')
    return
  }
  fail(`unknown command: ${command || '(missing)'}`)
}

const isDirectRun = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href
if (isDirectRun) {
  runCli().catch((error) => {
    console.error(error instanceof Error ? error.message : '[Android Release] validation failed')
    process.exitCode = 1
  })
}
