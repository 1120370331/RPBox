#!/usr/bin/env node

import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const DEFAULT_API_BASE = 'https://ksxvodevhonx.sealosbja.site/api/v1'

function printUsage() {
  console.log(`Usage:
  node scripts/rpbox-post.mjs --post-file <file> [--status draft|published] [--dry-run]

Environment:
  RPBOX_API_BASE   API base URL, defaults to ${DEFAULT_API_BASE}
  RPBOX_USERNAME   RPBox username or email
  RPBOX_PASSWORD   RPBox password

JSON shape:
  { "posts": [{ "title": "...", "content": "...", "category": "other" }] }
  or an array of post objects
`)
}

function parseArgs(argv) {
  const args = {
    apiBase: process.env.RPBOX_API_BASE || DEFAULT_API_BASE,
    username: process.env.RPBOX_USERNAME,
    password: process.env.RPBOX_PASSWORD,
    postFile: undefined,
    status: undefined,
    dryRun: false,
  }

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--help' || arg === '-h') {
      args.help = true
    } else if (arg === '--api-base') {
      args.apiBase = argv[++i]
    } else if (arg === '--post-file') {
      args.postFile = argv[++i]
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
    // Keep the original HTTP status if the body is empty or not JSON.
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

async function createPost(apiBase, token, post) {
  return requestJson(`${apiBase}/posts`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(post),
  })
}

async function loadPosts(postFile, status) {
  const text = await readFile(resolve(postFile), 'utf8')
  const parsed = JSON.parse(text)
  const posts = Array.isArray(parsed) ? parsed : parsed.posts

  if (!Array.isArray(posts) || posts.length === 0) {
    throw new Error('Post file must contain a non-empty posts array')
  }

  return posts.map((post, index) => {
    const title = requireValue(post.title, `posts[${index}].title`)
    const content = requireValue(post.content, `posts[${index}].content`)
    return {
      content_type: 'markdown',
      category: 'other',
      is_public: true,
      ...post,
      title,
      content,
      status: status || post.status || 'draft',
    }
  })
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  if (args.help) {
    printUsage()
    return
  }

  const postFile = requireValue(args.postFile, '--post-file')
  const posts = await loadPosts(postFile, args.status)

  if (args.dryRun) {
    console.log(JSON.stringify({ apiBase: args.apiBase, dryRun: true, posts }, null, 2))
    return
  }

  const username = requireValue(args.username, 'RPBOX_USERNAME')
  const password = requireValue(args.password, 'RPBOX_PASSWORD')
  const token = await login(args.apiBase, username, password)

  const created = []
  for (const post of posts) {
    const result = await createPost(args.apiBase, token, post)
    created.push({ id: result.id, title: result.title, status: result.status, review_status: result.review_status })
  }

  console.log(JSON.stringify({ created }, null, 2))
}

main().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
