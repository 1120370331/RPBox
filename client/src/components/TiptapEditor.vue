<script setup lang="ts">
import { ref, onBeforeUnmount, watch } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import Mention from '@tiptap/extension-mention'
import TextAlign from '@tiptap/extension-text-align'
import { mergeAttributes, Node } from '@tiptap/core'
import { uploadAttachment, uploadImage } from '@/api/item'
import { useToast } from '@/composables/useToast'
import { searchUsers, type UserMentionItem } from '@/api/user'

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const imageInputRef = ref<HTMLInputElement | null>(null)
const attachmentInputRef = ref<HTMLInputElement | null>(null)
const toast = useToast()
const uploadCacheKey = 'tiptap_image_upload_cache'
const uploadCache = new Map<string, string>()
const maxAttachmentBytes = 25 * 1024 * 1024

function loadUploadCache() {
  try {
    const cached = sessionStorage.getItem(uploadCacheKey)
    if (!cached) return
    const parsed = JSON.parse(cached) as Record<string, string>
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'string') {
        uploadCache.set(key, value)
      }
    }
  } catch (error) {
    console.warn('Failed to load upload cache:', error)
  }
}

function persistUploadCache() {
  try {
    const data: Record<string, string> = {}
    uploadCache.forEach((value, key) => {
      data[key] = value
    })
    sessionStorage.setItem(uploadCacheKey, JSON.stringify(data))
  } catch (error) {
    console.warn('Failed to persist upload cache:', error)
  }
}

function getFileCacheKey(file: File) {
  return `${file.name}-${file.size}-${file.lastModified}`
}

loadUploadCache()

type MentionSuggestionItem = Omit<UserMentionItem, 'id'> & {
  id: string
  label: string
}

let mentionPopup: HTMLDivElement | null = null

async function fetchMentionItems(query: string): Promise<MentionSuggestionItem[]> {
  try {
    const res = await searchUsers(query, 8)
    return (res.users || []).map((user) => ({
      ...user,
      id: String(user.id),
      label: user.username,
    }))
  } catch (error) {
    console.error('加载提及用户失败:', error)
    return []
  }
}

const MentionExtension = Mention.extend({
  renderHTML({ node, HTMLAttributes }) {
    return [
      'span',
      mergeAttributes(HTMLAttributes, {
        'data-mention-id': node.attrs.id,
        'data-mention-name': node.attrs.label,
      }),
      `@${node.attrs.label}`,
    ]
  },
  renderText({ node }) {
    return `@${node.attrs.label}`
  },
})

type JumpVariant = 'story-mine' | 'story-guild' | 'post-public' | 'guild-home'

function normalizeJumpText(value: unknown) {
  return String(value || '').trim()
}

function formatJumpStatus(value: string) {
  if (value === 'draft') return '草稿'
  if (value === 'published') return '已发布'
  return value
}

function formatJumpVisibility(value: string) {
  if (value === 'public') return '公开'
  if (value === 'private') return '私密'
  return value
}

function resolveJumpVariant(attrs: Record<string, any>): JumpVariant | '' {
  const variant = normalizeJumpText(attrs.variant)
  if (variant) return variant as JumpVariant
  const type = normalizeJumpText(attrs.type)
  const label = normalizeJumpText(attrs.label)
  if (type === 'guild') return 'guild-home'
  if (type === 'post') return 'post-public'
  if (type === 'story') {
    return label.includes('公会') ? 'story-guild' : 'story-mine'
  }
  return ''
}

function buildJumpBaseAttrs(attrs: Record<string, any>, variant: string, styleVariant?: string) {
  const href = normalizeJumpText(attrs.href)
  const label = normalizeJumpText(attrs.label)
  const title = normalizeJumpText(attrs.title)
  const type = normalizeJumpText(attrs.type)
  const status = normalizeJumpText(attrs.status)
  const visibility = normalizeJumpText(attrs.visibility)
  const guild = normalizeJumpText(attrs.guild)
  const guildId = normalizeJumpText(attrs.guildId)
  const author = normalizeJumpText(attrs.author)
  const avatar = normalizeJumpText(attrs.avatar)
  const members = normalizeJumpText(attrs.members)
  const image = normalizeJumpText(attrs.image)
  const classes = ['jump-card']
  const appliedVariant = styleVariant || variant
  if (appliedVariant) classes.push(`jump-card--${appliedVariant}`)
  return {
    class: classes.join(' '),
    role: 'link',
    tabindex: '0',
    'data-jump-href': href,
    'data-jump-type': type,
    'data-jump-label': label,
    'data-jump-title': title,
    'data-jump-variant': variant,
    'data-jump-status': status,
    'data-jump-visibility': visibility,
    'data-jump-guild': guild,
    'data-jump-guild-id': guildId,
    'data-jump-author': author,
    'data-jump-avatar': avatar,
    'data-jump-members': members,
    'data-jump-image': image,
  }
}

function buildJumpTag(label: string, variant: string) {
  if (!label) return null
  return ['span', { class: `jump-tag ${variant}`.trim() }, label]
}

function buildJumpCard(attrs: Record<string, any>) {
  const label = normalizeJumpText(attrs.label) || '跳转'
  const title = normalizeJumpText(attrs.title)
  const status = normalizeJumpText(attrs.status)
  const visibility = normalizeJumpText(attrs.visibility)
  const guild = normalizeJumpText(attrs.guild)
  const author = normalizeJumpText(attrs.author)
  const avatar = normalizeJumpText(attrs.avatar)
  const members = normalizeJumpText(attrs.members)
  const image = normalizeJumpText(attrs.image)
  const type = normalizeJumpText(attrs.type)
  const variant = resolveJumpVariant(attrs)
  const baseAttrs = buildJumpBaseAttrs(attrs, variant)

  if (!variant) {
    return [
      'span',
      {
        class: 'jump-link',
        role: 'link',
        tabindex: '0',
        'data-jump-href': normalizeJumpText(attrs.href),
        'data-jump-type': type,
        'data-jump-label': label,
        'data-jump-title': title,
      },
      title ? `${label}：${title}` : label,
    ]
  }

  if (variant === 'story-mine') {
    const tags: any[] = []
    const statusLabel = formatJumpStatus(status)
    const visibilityLabel = formatJumpVisibility(visibility)
    const statusTag = buildJumpTag(statusLabel, 'jump-tag--status')
    if (statusTag) tags.push(statusTag)
    const visibilityTag = buildJumpTag(visibilityLabel, 'jump-tag--privacy')
    if (visibilityTag) tags.push(visibilityTag)
    const hint = status === 'draft' ? '点击继续编写' : '查看剧情详情'
    const meta = [
      'span',
      { class: 'jump-card__meta' },
      ['span', { class: 'jump-card__label' }, label || '我的剧情'],
      tags.length ? ['span', { class: 'jump-card__tags' }, ...tags] : ['span', { class: 'jump-card__tags' }],
    ]
    return [
      'span',
      baseAttrs,
      meta,
      ['span', { class: 'jump-card__title' }, title || '未命名剧情'],
      ['span', { class: 'jump-card__hint' }, hint],
    ]
  }

  if (variant === 'story-guild') {
    const statusLabel = formatJumpStatus(status) || '已发布'
    const mediaChildren: any[] = []
    if (image) {
      mediaChildren.push(['img', { class: 'jump-card__image', src: image, alt: '' }])
    } else {
      mediaChildren.push(['span', { class: 'jump-card__media-fallback' }, (guild || label || '公会').slice(0, 1)])
    }
    mediaChildren.push(['span', { class: 'jump-card__media-overlay' }])
    return [
      'span',
      baseAttrs,
      ['span', { class: 'jump-card__media' }, ...mediaChildren],
      [
        'span',
        { class: 'jump-card__content' },
        [
          'span',
          { class: 'jump-card__meta' },
          ['span', { class: 'jump-card__label' }, label || '公会剧情'],
          ['span', { class: 'jump-card__dot' }],
          ['span', { class: 'jump-card__sub' }, guild || '未知公会'],
        ],
        ['span', { class: 'jump-card__title' }, title || '未命名剧情'],
        [
          'span',
          { class: 'jump-card__footer' },
          buildJumpTag(statusLabel, 'jump-tag--status') || ['span', { class: 'jump-tag jump-tag--status' }, statusLabel],
          ['span', { class: 'jump-card__arrow' }, '→'],
        ],
      ],
    ]
  }

  if (variant === 'post-public') {
    const authorName = author || '未知作者'
    const postTitle = title || '未命名帖子'
    const logoChildren: any[] = []
    if (image) {
      logoChildren.push(['img', { class: 'jump-card__logo-image', src: image, alt: '' }])
    } else {
      logoChildren.push(['span', { class: 'jump-card__logo-fallback' }, postTitle.slice(0, 1)])
    }
    return [
      'span',
      buildJumpBaseAttrs(attrs, variant, 'guild-home'),
      ['span', { class: 'jump-card__logo' }, ...logoChildren],
      [
        'span',
        { class: 'jump-card__content' },
        ['span', { class: 'jump-card__label' }, label || '公开帖子'],
        ['span', { class: 'jump-card__title' }, postTitle],
      ],
      [
        'span',
        { class: 'jump-card__stat' },
        ['span', { class: 'jump-card__stat-value' }, authorName],
        ['span', { class: 'jump-card__stat-label' }, '作者'],
      ],
      ['span', { class: 'jump-card__action' }, '→'],
    ]
  }

  const memberLabel = members || '0'
  const guildName = title || label || '公会主页'
  const bgChildren: any[] = []
  if (image) {
    bgChildren.push(['img', { class: 'jump-card__bg-image', src: image, alt: '' }])
  } else {
    bgChildren.push(['span', { class: 'jump-card__bg-fallback' }])
  }
  const guildInitial = guildName.slice(0, 1)
  const authorAvatar: any[] = []
  if (avatar) {
    authorAvatar.push(['img', { src: avatar, alt: '' }])
  } else {
    authorAvatar.push(guildInitial)
  }
  return [
    'span',
    buildJumpBaseAttrs(attrs, variant, 'post-public'),
    ['span', { class: 'jump-card__bg' }, ...bgChildren],
    ['span', { class: 'jump-card__overlay' }],
    [
      'span',
      { class: 'jump-card__content' },
      ['span', { class: 'jump-card__label' }, label || '公会主页'],
      ['span', { class: 'jump-card__title' }, guildName],
      [
        'span',
        { class: 'jump-card__footer' },
        [
          'span',
          { class: 'jump-card__author' },
          ['span', { class: 'jump-card__author-avatar' }, ...authorAvatar],
          ['span', { class: 'jump-card__author-name' }, `成员：${memberLabel}`],
        ],
        ['span', { class: 'jump-card__cta' }, '进入公会 →'],
      ],
    ],
  ]
}

function parseJumpAttrs(node: HTMLElement) {
  const href = node.getAttribute('data-jump-href') || node.getAttribute('href') || ''
  let label = node.getAttribute('data-jump-label') || ''
  let title = node.getAttribute('data-jump-title') || ''
  const type = node.getAttribute('data-jump-type') || ''
  const variant = node.getAttribute('data-jump-variant') || ''
  const status = node.getAttribute('data-jump-status') || ''
  const visibility = node.getAttribute('data-jump-visibility') || ''
  const guild = node.getAttribute('data-jump-guild') || ''
  const guildId = node.getAttribute('data-jump-guild-id') || ''
  const author = node.getAttribute('data-jump-author') || ''
  let avatar = node.getAttribute('data-jump-avatar') || ''
  const members = node.getAttribute('data-jump-members') || ''
  let image = node.getAttribute('data-jump-image') || ''

  if (!image) {
    const img = node.querySelector('.jump-card__bg-image, .jump-card__image, .jump-card__logo-image, .jump-card__thumb img')
    if (img) {
      image = img.getAttribute('src') || ''
    }
  }

  if (!avatar) {
    const authorImg = node.querySelector('.jump-card__author-avatar img')
    if (authorImg) {
      avatar = authorImg.getAttribute('src') || ''
    }
  }

  if (!label) {
    label = node.querySelector('.jump-card__label')?.textContent || ''
  }
  if (!title) {
    title = node.querySelector('.jump-card__title')?.textContent || ''
  }

  const text = (node.textContent || '').trim()
  if (!label && text) {
    const parts = text.split('：')
    if (parts.length > 1) {
      label = parts[0].trim()
      title = parts.slice(1).join('：').trim()
    } else {
      label = text
    }
  }

  return {
    href,
    label: label.trim(),
    title: title.trim(),
    type: type.trim(),
    variant: variant.trim(),
    status: status.trim(),
    visibility: visibility.trim(),
    guild: guild.trim(),
    guildId: guildId.trim(),
    author: author.trim(),
    avatar: avatar.trim(),
    members: members.trim(),
    image: image.trim(),
  }
}

function getFileIcon(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, string> = {
    pdf: 'ri-file-pdf-2-line',
    doc: 'ri-file-word-line',
    docx: 'ri-file-word-line',
    xls: 'ri-file-excel-line',
    xlsx: 'ri-file-excel-line',
    ppt: 'ri-file-ppt-line',
    pptx: 'ri-file-ppt-line',
    zip: 'ri-file-zip-line',
    rar: 'ri-file-zip-line',
    '7z': 'ri-file-zip-line',
    txt: 'ri-file-text-line',
    mp3: 'ri-file-music-line',
    wav: 'ri-file-music-line',
    mp4: 'ri-file-video-line',
    avi: 'ri-file-video-line',
    mkv: 'ri-file-video-line',
    lua: 'ri-file-code-line',
    js: 'ri-file-code-line',
    ts: 'ri-file-code-line',
    json: 'ri-file-code-line',
  }
  return iconMap[ext] || 'ri-file-line'
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const AttachmentCardExtension = Node.create({
  name: 'attachmentCard',
  group: 'block',
  atom: true,
  selectable: true,
  draggable: true,
  addAttributes() {
    return {
      href: { default: '' },
      filename: { default: '' },
      filesize: { default: 0 },
      uploading: { default: false },
      uploadId: { default: '' },
    }
  },
  parseHTML() {
    return [
      {
        tag: 'div.attachment-card',
        getAttrs: (node) => {
          const el = node as HTMLElement
          return {
            href: el.getAttribute('data-href') || '',
            filename: el.getAttribute('data-filename') || '',
            filesize: parseInt(el.getAttribute('data-filesize') || '0', 10),
            uploading: el.getAttribute('data-uploading') === 'true',
            uploadId: el.getAttribute('data-upload-id') || '',
          }
        },
      },
    ]
  },
  renderHTML({ HTMLAttributes }) {
    const href = HTMLAttributes.href || ''
    const filename = HTMLAttributes.filename || '未知文件'
    const filesize = HTMLAttributes.filesize || 0
    const uploading = HTMLAttributes.uploading
    const uploadId = HTMLAttributes.uploadId || ''
    const icon = uploading ? 'ri-loader-4-line' : getFileIcon(filename)
    const sizeText = filesize > 0 ? formatFileSize(filesize) : ''
    const cardClass = uploading ? 'attachment-card attachment-card--uploading' : 'attachment-card'

    if (uploading) {
      return [
        'div',
        {
          class: cardClass,
          'data-href': href,
          'data-filename': filename,
          'data-filesize': String(filesize),
          'data-uploading': 'true',
          'data-upload-id': uploadId,
        },
        ['div', { class: 'attachment-card__icon attachment-card__icon--spin' }, ['i', { class: icon }]],
        [
          'div',
          { class: 'attachment-card__info' },
          ['span', { class: 'attachment-card__name' }, filename],
          ['span', { class: 'attachment-card__size' }, '上传中...'],
        ],
        ['div', { class: 'attachment-card__progress' }],
      ]
    }

    return [
      'div',
      {
        class: cardClass,
        'data-href': href,
        'data-filename': filename,
        'data-filesize': String(filesize),
      },
      ['div', { class: 'attachment-card__icon' }, ['i', { class: icon }]],
      [
        'div',
        { class: 'attachment-card__info' },
        ['span', { class: 'attachment-card__name' }, filename],
        sizeText ? ['span', { class: 'attachment-card__size' }, sizeText] : ['span', { class: 'attachment-card__size' }],
      ],
      [
        'a',
        {
          class: 'attachment-card__download',
          href: href,
          download: filename,
        },
        ['i', { class: 'ri-download-line' }],
        '下载',
      ],
    ]
  },
})

const JumpLinkExtension = Node.create({
  name: 'jumpLink',
  inline: true,
  group: 'inline',
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      href: { default: '' },
      label: { default: '' },
      title: { default: '' },
      type: { default: '' },
      variant: { default: '' },
      status: { default: '' },
      visibility: { default: '' },
      guild: { default: '' },
      guildId: { default: '' },
      author: { default: '' },
      avatar: { default: '' },
      members: { default: '' },
      image: { default: '' },
    }
  },
  parseHTML() {
    return [
      {
        tag: 'a.jump-link',
        getAttrs: (node) => parseJumpAttrs(node as HTMLElement),
      },
      {
        tag: 'span.jump-link',
        getAttrs: (node) => parseJumpAttrs(node as HTMLElement),
      },
      {
        tag: 'span.jump-card',
        getAttrs: (node) => parseJumpAttrs(node as HTMLElement),
      },
      {
        tag: 'a.jump-card',
        getAttrs: (node) => parseJumpAttrs(node as HTMLElement),
      },
    ]
  },
  renderHTML({ HTMLAttributes }) {
    return buildJumpCard(HTMLAttributes as Record<string, any>)
  },
})

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Placeholder.configure({
      placeholder: props.placeholder || '开始写作...',
    }),
    Image.configure({
      inline: true,
      allowBase64: true,
    }),
    Link.configure({
      openOnClick: false,
      autolink: true,
      linkOnPaste: true,
      HTMLAttributes: {
        rel: 'noopener noreferrer',
        target: '_blank',
      },
    }),
    TextAlign.configure({
      types: ['heading', 'paragraph'],
    }),
    JumpLinkExtension,
    AttachmentCardExtension,
    MentionExtension.configure({
      HTMLAttributes: {
        class: 'mention',
      },
      suggestion: {
        items: ({ query }) => fetchMentionItems(query),
        render: () => {
          let selectedIndex = 0
          let root: HTMLDivElement | null = null

          const update = (props: any) => {
            if (!root) return
            root.innerHTML = ''

            const list = document.createElement('div')
            list.className = 'mention-suggestion__list'

            if (!props.items.length) {
              const empty = document.createElement('div')
              empty.className = 'mention-suggestion__empty'
              empty.textContent = '未找到用户'
              list.appendChild(empty)
            } else {
              props.items.forEach((item: MentionSuggestionItem, index: number) => {
                const button = document.createElement('button')
                button.type = 'button'
                button.className = 'mention-suggestion__item'
                if (index === selectedIndex) {
                  button.classList.add('is-active')
                }

                if (item.avatar) {
                  const img = document.createElement('img')
                  img.src = item.avatar
                  img.alt = item.username
                  img.className = 'mention-suggestion__avatar'
                  button.appendChild(img)
                } else {
                  const avatar = document.createElement('div')
                  avatar.className = 'mention-suggestion__avatar mention-suggestion__avatar--fallback'
                  avatar.textContent = item.username.charAt(0).toUpperCase()
                  button.appendChild(avatar)
                }

                const name = document.createElement('span')
                name.className = 'mention-suggestion__name'
                name.textContent = item.username
                if (item.name_color) {
                  name.style.color = item.name_color
                }
                if (item.name_bold) {
                  name.style.fontWeight = '700'
                }
                button.appendChild(name)

                button.addEventListener('click', () => {
                  props.command(item)
                })

                list.appendChild(button)
              })
            }

            root.appendChild(list)

            if (props.clientRect) {
              const rect = props.clientRect()
              if (rect) {
                root.style.display = 'block'
                root.style.top = `${rect.bottom + window.scrollY + 6}px`
                root.style.left = `${rect.left + window.scrollX}px`
              } else {
                root.style.display = 'none'
              }
            } else {
              root.style.display = 'none'
            }
          }

          return {
            onStart: (props: any) => {
              root = document.createElement('div')
              root.className = 'mention-suggestion'
              mentionPopup = root
              document.body.appendChild(root)
              selectedIndex = 0
              update(props)
            },
            onUpdate: (props: any) => {
              if (!props.items.length) {
                selectedIndex = 0
              } else if (selectedIndex >= props.items.length) {
                selectedIndex = props.items.length - 1
              }
              update(props)
            },
            onKeyDown: (props: any) => {
              if (props.event.key === 'Escape') {
                if (root) root.remove()
                return true
              }
              if (!props.items.length) {
                return false
              }
              if (props.event.key === 'ArrowDown') {
                selectedIndex = (selectedIndex + 1) % props.items.length
                update(props)
                return true
              }
              if (props.event.key === 'ArrowUp') {
                selectedIndex = (selectedIndex - 1 + props.items.length) % props.items.length
                update(props)
                return true
              }
              if (props.event.key === 'Enter' || props.event.key === 'Tab') {
                props.command(props.items[selectedIndex])
                return true
              }
              return false
            },
            onExit: () => {
              if (root) {
                root.remove()
              }
              if (mentionPopup === root) {
                mentionPopup = null
              }
            },
          }
        },
      },
    }),
  ],
  onUpdate: ({ editor }) => {
    emit('update:modelValue', editor.getHTML())
  },
  editorProps: {
    handleDOMEvents: {
      click: (_view, event) => {
        const element = event.target instanceof Element ? event.target : null
        if (!element) return false
        const link = element.closest('.jump-link, a.jump-card, [data-jump-href], [data-jump-type]')
        if (link) {
          event.preventDefault()
        }
        return false
      },
    },
  },
})

onBeforeUnmount(() => {
  if (mentionPopup) {
    mentionPopup.remove()
    mentionPopup = null
  }
})

watch(() => props.modelValue, (value) => {
  if (editor.value && editor.value.getHTML() !== value) {
    editor.value.commands.setContent(value, false)
  }
})

// 图片上传
function triggerImageUpload() {
  imageInputRef.value?.click()
}

async function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  if (files.length === 0) return

  for (const file of files) {
    if (file.size > 20 * 1024 * 1024) {
      toast.info(`图片 ${file.name} 不能超过20MB`)
      continue
    }

    const cacheKey = getFileCacheKey(file)
    const cachedUrl = uploadCache.get(cacheKey)
    if (cachedUrl) {
      editor.value?.chain().focus().setImage({ src: cachedUrl }).run()
      continue
    }

    try {
      const res: any = await uploadImage(file)
      const url = res?.data?.url || res?.url
      if (!url) {
        throw new Error('未获取到图片地址')
      }
      editor.value?.chain().focus().setImage({ src: url }).run()
      uploadCache.set(cacheKey, url)
      persistUploadCache()
    } catch (error: any) {
      console.error('图片上传失败:', error)
      toast.error(error.message || '图片上传失败')
    }
  }

  input.value = ''
}

// 通过URL插入图片
function insertImageByUrl() {
  const url = prompt('请输入图片URL:')
  if (url) {
    editor.value?.chain().focus().setImage({ src: url }).run()
  }
}

function insertAttachmentLink(url: string, name: string, size: number = 0) {
  if (!url) return
  const filename = name.trim() || url.split('/').pop() || '未知文件'
  editor.value?.chain().focus().insertContent({
    type: 'attachmentCard',
    attrs: {
      href: url,
      filename: filename,
      filesize: size,
    },
  }).run()
}

function triggerAttachmentUpload() {
  attachmentInputRef.value?.click()
}

function generateUploadId() {
  return `upload_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

function findAndReplaceUploadingCard(uploadId: string, newAttrs: Record<string, any> | null) {
  if (!editor.value) return
  const { state } = editor.value
  let targetPos: number | null = null

  state.doc.descendants((node, pos) => {
    if (node.type.name === 'attachmentCard' && node.attrs.uploadId === uploadId) {
      targetPos = pos
      return false
    }
    return true
  })

  if (targetPos !== null) {
    if (newAttrs) {
      editor.value.chain().focus()
        .setNodeSelection(targetPos)
        .deleteSelection()
        .insertContent({
          type: 'attachmentCard',
          attrs: newAttrs,
        })
        .run()
    } else {
      editor.value.chain().focus()
        .setNodeSelection(targetPos)
        .deleteSelection()
        .run()
    }
  }
}

async function handleAttachmentUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  if (files.length === 0) return

  for (const file of files) {
    if (file.size > maxAttachmentBytes) {
      toast.info(`附件 ${file.name} 不能超过25MB`)
      continue
    }

    const uploadId = generateUploadId()

    // 先插入上传中的卡片
    editor.value?.chain().focus().insertContent({
      type: 'attachmentCard',
      attrs: {
        href: '',
        filename: file.name,
        filesize: file.size,
        uploading: true,
        uploadId: uploadId,
      },
    }).run()

    try {
      const res: any = await uploadAttachment(file)
      const url = res?.data?.url || res?.url
      if (!url) {
        throw new Error('未获取到附件地址')
      }
      const name = res?.data?.name || file.name
      // 替换为完成的卡片
      findAndReplaceUploadingCard(uploadId, {
        href: url,
        filename: name,
        filesize: file.size,
        uploading: false,
        uploadId: '',
      })
    } catch (error: any) {
      console.error('附件上传失败:', error)
      toast.error(error.message || '附件上传失败')
      // 移除上传中的卡片
      findAndReplaceUploadingCard(uploadId, null)
    }
  }

  input.value = ''
}

function insertContent(html: string) {
  if (!html) return
  editor.value?.chain().focus().insertContent(html).run()
}

defineExpose({
  insertContent,
})
</script>

<template>
  <div class="rich-editor">
    <div class="toolbar">
      <button
        type="button"
        :class="{ active: editor?.isActive('bold') }"
        @click="editor?.chain().focus().toggleBold().run()"
        title="粗体"
      >
        <i class="ri-bold"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('italic') }"
        @click="editor?.chain().focus().toggleItalic().run()"
        title="斜体"
      >
        <i class="ri-italic"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('strike') }"
        @click="editor?.chain().focus().toggleStrike().run()"
        title="删除线"
      >
        <i class="ri-strikethrough"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 1 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 1 }).run()"
        title="标题1"
      >
        <i class="ri-h-1"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 2 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 2 }).run()"
        title="标题2"
      >
        <i class="ri-h-2"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('heading', { level: 3 }) }"
        @click="editor?.chain().focus().toggleHeading({ level: 3 }).run()"
        title="标题3"
      >
        <i class="ri-h-3"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('bulletList') }"
        @click="editor?.chain().focus().toggleBulletList().run()"
        title="无序列表"
      >
        <i class="ri-list-unordered"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('orderedList') }"
        @click="editor?.chain().focus().toggleOrderedList().run()"
        title="有序列表"
      >
        <i class="ri-list-ordered"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive({ textAlign: 'left' }) }"
        @click="editor?.chain().focus().setTextAlign('left').run()"
        title="左对齐"
      >
        <i class="ri-align-left"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive({ textAlign: 'center' }) }"
        @click="editor?.chain().focus().setTextAlign('center').run()"
        title="居中"
      >
        <i class="ri-align-center"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive({ textAlign: 'right' }) }"
        @click="editor?.chain().focus().setTextAlign('right').run()"
        title="右对齐"
      >
        <i class="ri-align-right"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        :class="{ active: editor?.isActive('blockquote') }"
        @click="editor?.chain().focus().toggleBlockquote().run()"
        title="引用"
      >
        <i class="ri-double-quotes-l"></i>
      </button>
      <button
        type="button"
        :class="{ active: editor?.isActive('codeBlock') }"
        @click="editor?.chain().focus().toggleCodeBlock().run()"
        title="代码块"
      >
        <i class="ri-code-box-line"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        @click="editor?.chain().focus().undo().run()"
        :disabled="!editor?.can().undo()"
        title="撤销"
      >
        <i class="ri-arrow-go-back-line"></i>
      </button>
      <button
        type="button"
        @click="editor?.chain().focus().redo().run()"
        :disabled="!editor?.can().redo()"
        title="重做"
      >
        <i class="ri-arrow-go-forward-line"></i>
      </button>
      <span class="divider"></span>
      <button
        type="button"
        @click="triggerImageUpload"
        title="上传图片"
      >
        <i class="ri-image-add-line"></i>
      </button>
      <button
        type="button"
        @click="insertImageByUrl"
        title="图片链接"
      >
        <i class="ri-link"></i>
      </button>
      <button
        type="button"
        @click="triggerAttachmentUpload"
        title="上传附件"
      >
        <i class="ri-attachment-2"></i>
      </button>
      <template v-if="$slots.toolbar">
        <span class="divider"></span>
        <slot name="toolbar" />
      </template>
    </div>
    <input
      ref="imageInputRef"
      type="file"
      accept="image/*"
      multiple
      style="display: none"
      @change="handleImageUpload"
    />
    <input
      ref="attachmentInputRef"
      type="file"
      multiple
      style="display: none"
      @change="handleAttachmentUpload"
    />
    <EditorContent :editor="editor" class="editor-content" />
  </div>
</template>

<style scoped>
.rich-editor {
  border: 2px solid var(--color-border);
  border-radius: 12px;
  overflow: hidden;
  background: var(--color-panel-bg);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 12px;
  background: var(--color-card-bg);
  border-bottom: 2px solid var(--color-border);
}

.toolbar button {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--color-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.toolbar button:hover {
  background: var(--color-border);
}

.toolbar button.active {
  background: var(--color-secondary);
  color: var(--color-text-light);
}

.toolbar button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.toolbar button i {
  font-size: 18px;
}

.toolbar :deep(.toolbar-slot) {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--color-primary);
  cursor: pointer;
  transition: all 0.2s;
  padding: 0;
  outline: none;
}

.toolbar :deep(.toolbar-slot:hover) {
  background: var(--color-border);
}

.toolbar :deep(.toolbar-slot.active) {
  background: var(--color-secondary);
  color: var(--color-text-light);
}

.toolbar :deep(.toolbar-slot:disabled) {
  opacity: 0.4;
  cursor: not-allowed;
}

.toolbar :deep(.toolbar-slot i) {
  font-size: 18px;
}

.divider {
  width: 1px;
  height: 24px;
  background: var(--color-border);
  margin: 6px 8px;
}

.editor-content {
  min-height: 300px;
  padding: 24px 32px;
}

.editor-content :deep(.tiptap) {
  outline: none;
  min-height: 280px;
  font-family: 'Merriweather', serif;
  font-size: 16px;
  line-height: 1.9;
  color: var(--color-primary);
}

.editor-content :deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: var(--color-text-muted);
  pointer-events: none;
  height: 0;
}

.editor-content :deep(h1) {
  font-size: 28px;
  font-weight: 700;
  margin: 16px 0 8px;
  color: var(--color-text-main);
}

.editor-content :deep(h2) {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-main);
  margin: 1.5em 0 0.8em;
}

.editor-content :deep(h3) {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-main);
  margin: 1.2em 0 0.6em;
}

.editor-content :deep(p) {
  margin-bottom: 1.5em;
}

.editor-content :deep(ul),
.editor-content :deep(ol) {
  padding-left: 24px;
  margin: 8px 0;
}

.editor-content :deep(li) {
  margin: 4px 0;
}

.editor-content :deep(blockquote) {
  border-left: 4px solid var(--color-accent);
  padding-left: 20px;
  margin: 1.5em 0;
  color: var(--color-text-muted);
  font-style: italic;
}

.editor-content :deep(pre) {
  background: var(--color-text-main);
  color: var(--color-border);
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 12px 0;
}

.editor-content :deep(code) {
  font-family: 'Fira Code', monospace;
  font-size: 14px;
}

.editor-content :deep(img) {
  max-width: 100%;
  height: auto;
  display: inline-block;
  border-radius: 4px;
  margin: 0.6em 0.6em;
  vertical-align: middle;
  cursor: pointer;
  transition: transform 0.2s;
}

.editor-content :deep(img:hover) {
  transform: scale(1.02);
}

.editor-content :deep(img.ProseMirror-selectednode) {
  outline: 3px solid var(--color-secondary);
  outline-offset: 2px;
}

.editor-content :deep(strong) {
  color: var(--color-secondary);
}

.editor-content :deep(.mention) {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-primary-light);
  color: var(--color-secondary);
  font-weight: 600;
  font-size: 0.9em;
}

.editor-content :deep(.jump-link) {
  cursor: text;
}

:global(.mention-suggestion) {
  position: absolute;
  z-index: 2000;
  min-width: 200px;
  background: var(--color-panel-bg);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  box-shadow: 0 12px 24px rgba(var(--shadow-base), 0.12);
  padding: 8px;
}

:global(.mention-suggestion__list) {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

:global(.mention-suggestion__item) {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s;
}

:global(.mention-suggestion__item.is-active),
:global(.mention-suggestion__item:hover) {
  background: var(--color-primary-light);
}

:global(.mention-suggestion__avatar) {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--color-card-bg);
  color: var(--color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

:global(.mention-suggestion__avatar--fallback) {
  border: 1px solid var(--color-border);
}

:global(.mention-suggestion__name) {
  font-size: 13px;
  color: var(--color-text-main);
}

:global(.mention-suggestion__empty) {
  padding: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
  text-align: center;
}

/* Attachment Card */
.editor-content :deep(.attachment-card) {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin: 12px 0;
  background: var(--color-card-bg);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  transition: all 0.2s;
}

.editor-content :deep(.attachment-card:hover) {
  border-color: var(--color-secondary);
  box-shadow: 0 2px 8px rgba(var(--shadow-base), 0.08);
}

.editor-content :deep(.attachment-card.ProseMirror-selectednode) {
  outline: 2px solid var(--color-secondary);
  outline-offset: 2px;
}

.editor-content :deep(.attachment-card__icon) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--color-primary-light);
  border-radius: 8px;
  flex-shrink: 0;
}

.editor-content :deep(.attachment-card__icon i) {
  font-size: 20px;
  color: var(--color-secondary);
}

.editor-content :deep(.attachment-card__info) {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.editor-content :deep(.attachment-card__name) {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.editor-content :deep(.attachment-card__size) {
  font-size: 12px;
  color: var(--color-text-muted);
}

.editor-content :deep(.attachment-card__download) {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  background: var(--color-secondary);
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.2s;
  flex-shrink: 0;
}

.editor-content :deep(.attachment-card__download:hover) {
  background: var(--color-secondary-dark, #6B3528);
  transform: translateY(-1px);
}

.editor-content :deep(.attachment-card__download i) {
  font-size: 16px;
}

/* Attachment Card Uploading State */
.editor-content :deep(.attachment-card--uploading) {
  background: var(--color-primary-light);
  border-style: dashed;
}

.editor-content :deep(.attachment-card__icon--spin i) {
  animation: spin 1s linear infinite;
}

.editor-content :deep(.attachment-card__progress) {
  width: 60px;
  height: 4px;
  background: var(--color-border);
  border-radius: 2px;
  overflow: hidden;
  position: relative;
}

.editor-content :deep(.attachment-card__progress::after) {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  width: 30%;
  background: var(--color-secondary);
  border-radius: 2px;
  animation: progress 1.2s ease-in-out infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes progress {
  0% { left: -30%; }
  100% { left: 100%; }
}
</style>
