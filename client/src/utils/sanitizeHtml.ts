import createDOMPurify, {
  type Config,
  type UponSanitizeAttributeHookEvent,
  type WindowLike,
} from 'dompurify'

const RICH_TEXT_TAGS = [
  'a', 'b', 'blockquote', 'br', 'code', 'del', 'div', 'em', 'figcaption', 'figure',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'li', 'mark', 'ol', 'p',
  'pre', 's', 'small', 'span', 'strike', 'strong', 'sub', 'sup', 'u', 'ul',
]

const RICH_TEXT_ATTRIBUTES = [
  'alt', 'aria-disabled', 'aria-label', 'class', 'contenteditable', 'download', 'draggable',
  'height', 'href', 'loading', 'rel', 'role', 'src', 'style', 'tabindex', 'target',
  'title', 'width',
]

const FORBIDDEN_TAGS = [
  'button', 'embed', 'form', 'iframe', 'input', 'math', 'object', 'script', 'select',
  'style', 'svg', 'textarea',
]
const RICH_TEXT_TAG_SET = new Set(RICH_TEXT_TAGS)
const RICH_TEXT_ATTRIBUTE_SET = new Set(RICH_TEXT_ATTRIBUTES)
const FORBIDDEN_TAG_SET = new Set(FORBIDDEN_TAGS)

const CONTROL_OR_SPACE_RE = /[\u0000-\u0020\u007f-\u009f]/g
const SCHEME_RE = /^([a-z][a-z\d+.-]*):/i
const SAFE_PROTOCOLS = new Set(['http', 'https'])
const SAFE_LINK_PROTOCOLS = new Set(['http', 'https', 'mailto'])
const RESOURCE_DATA_ATTRIBUTES = new Set([
  'data-href',
  'data-jump-avatar',
  'data-jump-image',
])

function normalizeURL(value: string) {
  return value.trim().replace(CONTROL_OR_SPACE_RE, '')
}

function isSafeURL(value: string, protocols: Set<string>) {
  const normalized = normalizeURL(value)
  if (!normalized || normalized.startsWith('//')) return false

  const scheme = normalized.match(SCHEME_RE)?.[1]?.toLowerCase()
  return !scheme || protocols.has(scheme)
}

function isSafeInternalJumpURL(value: string) {
  const normalized = normalizeURL(value)
  if (!normalized || normalized.startsWith('//') || SCHEME_RE.test(normalized)) return false
  return normalized.startsWith('/')
    || normalized.startsWith('./')
    || normalized.startsWith('../')
    || normalized.startsWith('#')
    || normalized.startsWith('?')
}

function sanitizeInlineStyle(value: string) {
  for (const declaration of value.split(';')) {
    const match = declaration.match(/^\s*text-align\s*:\s*(left|right|center|justify|start|end)\s*$/i)
    if (match) return `text-align: ${match[1].toLowerCase()}`
  }
  return ''
}

function sanitizeAttribute(_node: Element, hook: UponSanitizeAttributeHookEvent) {
  const name = hook.attrName.toLowerCase()

  if (name.startsWith('on') || name === 'srcdoc') {
    hook.keepAttr = false
    return
  }

  if (name === 'style') {
    hook.attrValue = sanitizeInlineStyle(hook.attrValue)
    hook.keepAttr = Boolean(hook.attrValue)
    return
  }

  if (name === 'href') {
    hook.keepAttr = isSafeURL(hook.attrValue, SAFE_LINK_PROTOCOLS)
    return
  }

  if (name === 'src' || RESOURCE_DATA_ATTRIBUTES.has(name)) {
    hook.keepAttr = isSafeURL(hook.attrValue, SAFE_PROTOCOLS)
    return
  }

  if (name === 'data-jump-href') {
    hook.keepAttr = isSafeInternalJumpURL(hook.attrValue)
  }
}

function hardenLinks(node: Element) {
  if (node.tagName.toLowerCase() !== 'a') return

  const target = node.getAttribute('target')
  if (target && target !== '_blank' && target !== '_self') {
    node.removeAttribute('target')
  }

  const href = node.getAttribute('href') || ''
  const scheme = normalizeURL(href).match(SCHEME_RE)?.[1]?.toLowerCase()
  if (target === '_blank' || (scheme && SAFE_LINK_PROTOCOLS.has(scheme))) {
    node.setAttribute('rel', 'noopener noreferrer')
  }
}

function sanitizeDOMAttributes(element: Element) {
  for (const attribute of Array.from(element.attributes)) {
    const name = attribute.name.toLowerCase()
    const isAllowed = RICH_TEXT_ATTRIBUTE_SET.has(name)
      || name.startsWith('data-')
      || name.startsWith('aria-')

    if (!isAllowed || name.startsWith('on') || name === 'srcdoc') {
      element.removeAttribute(attribute.name)
      continue
    }

    if (name === 'style') {
      const style = sanitizeInlineStyle(attribute.value)
      if (style) element.setAttribute('style', style)
      else element.removeAttribute(attribute.name)
      continue
    }

    if (name === 'href' && !isSafeURL(attribute.value, SAFE_LINK_PROTOCOLS)) {
      element.removeAttribute(attribute.name)
      continue
    }

    if ((name === 'src' || RESOURCE_DATA_ATTRIBUTES.has(name))
      && !isSafeURL(attribute.value, SAFE_PROTOCOLS)) {
      element.removeAttribute(attribute.name)
      continue
    }

    if (name === 'data-jump-href' && !isSafeInternalJumpURL(attribute.value)) {
      element.removeAttribute(attribute.name)
    }
  }

  hardenLinks(element)
}

function sanitizeDOMTree(parent: ParentNode) {
  for (const child of Array.from(parent.childNodes)) {
    if (child.nodeType === Node.COMMENT_NODE) {
      child.remove()
      continue
    }
    if (child.nodeType !== Node.ELEMENT_NODE) continue

    const element = child as Element
    const tagName = element.tagName.toLowerCase()
    if (FORBIDDEN_TAG_SET.has(tagName)) {
      element.remove()
      continue
    }

    sanitizeDOMTree(element)
    if (!RICH_TEXT_TAG_SET.has(tagName)) {
      element.replaceWith(...Array.from(element.childNodes))
      continue
    }

    sanitizeDOMAttributes(element)
  }
}

function enforceRichTextPolicy(html: string) {
  const template = document.createElement('template')
  template.innerHTML = html
  sanitizeDOMTree(template.content)
  return template.innerHTML
}

const purifier = createDOMPurify(window as unknown as WindowLike)

purifier.addHook('uponSanitizeAttribute', sanitizeAttribute)
purifier.addHook('afterSanitizeAttributes', hardenLinks)

const SANITIZE_CONFIG: Config = {
  ALLOWED_TAGS: RICH_TEXT_TAGS,
  ALLOWED_ATTR: RICH_TEXT_ATTRIBUTES,
  ALLOWED_NAMESPACES: ['http://www.w3.org/1999/xhtml'],
  ALLOW_ARIA_ATTR: true,
  ALLOW_DATA_ATTR: true,
  ALLOW_UNKNOWN_PROTOCOLS: false,
  FORBID_ATTR: ['srcdoc'],
  FORBID_TAGS: FORBIDDEN_TAGS,
  KEEP_CONTENT: true,
  RETURN_TRUSTED_TYPE: false,
}

// Some lightweight DOM test implementations claim DOMPurify compatibility but do not
// traverse sibling elements correctly. Browsers pass this probe; the DOM allowlist below
// remains a fail-closed fallback and a defense-in-depth pass in every environment.
const purifierIsReliable = (() => {
  if (!purifier.isSupported) return false
  const probe = purifier.sanitize('<p>probe</p><img src="/probe.png" onerror="alert(1)">', SANITIZE_CONFIG)
  return probe.includes('<p>probe</p>') && !probe.includes('onerror')
})()

/** Sanitizes Tiptap and RPBox rich text before it is assigned to v-html. */
export function sanitizeRichHtml(raw: string): string {
  if (!raw) return ''
  const purified = purifierIsReliable ? purifier.sanitize(raw, SANITIZE_CONFIG) : raw
  return enforceRichTextPolicy(purified)
}
