#!/usr/bin/env node

import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const DEFAULT_API_BASE = 'https://ksxvodevhonx.sealosbja.site/api/v1'

function printUsage() {
  console.log(`Usage:
  node scripts/rpbox-rpdb.mjs --work-file <file> [--status published] [--dry-run]

Environment:
  RPBOX_API_BASE   API base URL, defaults to ${DEFAULT_API_BASE}
  RPBOX_USERNAME   RPBox username or email
  RPBOX_PASSWORD   RPBox password
`)
}

function parseArgs(argv) {
  const args = {
    apiBase: process.env.RPBOX_API_BASE || DEFAULT_API_BASE,
    username: process.env.RPBOX_USERNAME,
    password: process.env.RPBOX_PASSWORD,
    workFile: undefined,
    status: undefined,
    dryRun: false,
  }

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--help' || arg === '-h') {
      args.help = true
    } else if (arg === '--api-base') {
      args.apiBase = argv[++i]
    } else if (arg === '--work-file') {
      args.workFile = argv[++i]
    } else if (arg === '--status') {
      args.status = argv[++i]
    } else if (arg === '--dry-run') {
      args.dryRun = true
    } else {
      throw new Error(`Unknown argument: ${arg}`)
    }
  }

  return args
}

function requireValue(value, name) {
  if (!value || String(value).trim() === '') {
    throw new Error(`${name} is required`)
  }
  return String(value).trim()
}

async function requestJson(url, options) {
  const res = await fetch(url, options)
  let data = null
  try {
    data = await res.json()
  } catch {
    // 204 responses and proxy errors may not include JSON.
  }

  if (!res.ok) {
    const message = data?.error || data?.message || res.statusText
    throw new Error(`${res.status} ${message}`)
  }

  return data
}

async function login(apiBase, username, password) {
  const data = await requestJson(`${apiBase}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })

  if (!data?.token) {
    throw new Error('Login response did not include a token')
  }

  return data.token
}

async function createWork(apiBase, token, work) {
  return requestJson(`${apiBase}/rpdb/works`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(work),
  })
}

async function loadWorks(workFile, status) {
  const text = await readFile(resolve(workFile), 'utf8')
  const parsed = JSON.parse(text)
  const works = Array.isArray(parsed) ? parsed : parsed.works

  if (!Array.isArray(works) || works.length === 0) {
    throw new Error('Work file must contain a non-empty works array')
  }

  return works.map((work, index) => {
    const title = requireValue(work.title, `works[${index}].title`)
    return {
      type: 'item_showcase',
      content_type: 'html',
      summary: '',
      content: '',
      cover_image: '',
      rp_use_cases: '',
      effect_description: '',
      restrictions: {},
      extra: {},
      game_version: '',
      expansion: '',
      availability_status: 'available',
      bind_type: 'no',
      faction: 'neutral',
      armor_type: '',
      visibility: 'public',
      is_public: true,
      references: [],
      media: [],
      guide_steps: [],
      tag_names: [],
      ...work,
      title,
      status: status || work.status || 'published',
    }
  })
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  if (args.help) {
    printUsage()
    return
  }

  const workFile = requireValue(args.workFile, '--work-file')
  const works = await loadWorks(workFile, args.status)

  if (args.dryRun) {
    console.log(JSON.stringify({ apiBase: args.apiBase, dryRun: true, works }, null, 2))
    return
  }

  const username = requireValue(args.username, 'RPBOX_USERNAME')
  const password = requireValue(args.password, 'RPBOX_PASSWORD')
  const token = await login(args.apiBase, username, password)

  const created = []
  for (const work of works) {
    const result = await createWork(args.apiBase, token, work)
    created.push({
      id: result.work?.id,
      title: result.work?.title,
      status: result.work?.status,
      review_status: result.work?.review_status,
    })
  }

  console.log(JSON.stringify({ created }, null, 2))
}

main().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
