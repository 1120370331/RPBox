function normalizeVersion(version: string | null | undefined) {
  return (version || '').trim().replace(/^v/i, '')
}

interface ParsedVersion {
  core: number[]
  prerelease: string[] | null
}

function parseVersion(version: string | null | undefined): ParsedVersion | null {
  const normalized = normalizeVersion(version).split('+', 1)[0]
  const match = /^(\d+(?:\.\d+)*)(?:-([0-9A-Za-z.-]+))?$/.exec(normalized)
  if (!match) return null

  const core = match[1].split('.').map(Number)
  if (core.some(part => !Number.isSafeInteger(part))) return null

  return {
    core,
    prerelease: match[2] ? match[2].split('.') : null,
  }
}

function comparePrereleaseIdentifier(a: string, b: string) {
  const numeric = /^\d+$/
  const aIsNumeric = numeric.test(a)
  const bIsNumeric = numeric.test(b)

  if (aIsNumeric && bIsNumeric) return Number(a) - Number(b)
  if (aIsNumeric) return -1
  if (bIsNumeric) return 1
  return a.localeCompare(b)
}

/** Returns true only when candidate is a semantically newer add-on version than current. */
export function isAddonVersionNewer(candidate: string | null | undefined, current: string | null | undefined) {
  const latest = parseVersion(candidate)
  if (!latest) return false

  const installed = parseVersion(current)
  if (!installed) return true

  const coreLength = Math.max(latest.core.length, installed.core.length)
  for (let index = 0; index < coreLength; index += 1) {
    const difference = (latest.core[index] || 0) - (installed.core[index] || 0)
    if (difference !== 0) return difference > 0
  }

  if (latest.prerelease === null && installed.prerelease !== null) return true
  if (latest.prerelease !== null && installed.prerelease === null) return false
  if (latest.prerelease === null || installed.prerelease === null) return false

  const prereleaseLength = Math.max(latest.prerelease.length, installed.prerelease.length)
  for (let index = 0; index < prereleaseLength; index += 1) {
    if (index >= latest.prerelease.length) return false
    if (index >= installed.prerelease.length) return true
    const difference = comparePrereleaseIdentifier(latest.prerelease[index], installed.prerelease[index])
    if (difference !== 0) return difference > 0
  }

  return false
}
