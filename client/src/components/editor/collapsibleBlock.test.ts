import { describe, expect, it } from 'vitest'
import { Editor, generateHTML, generateJSON } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { COLLAPSE_BODY_CLASS, CollapsibleBlockExtension, DEFAULT_COLLAPSE_SUMMARY } from './collapsibleBlock'

const extensions = [StarterKit, CollapsibleBlockExtension]

const stored = [
  '<details class="rpbox-collapse">',
  '<summary class="rpbox-collapse__summary">剧透警告</summary>',
  `<div class="${COLLAPSE_BODY_CLASS}"><p>隐藏的正文</p></div>`,
  '</details>',
].join('')

describe('collapsibleBlock extension', () => {
  it('parses stored details markup into a titled block', () => {
    const json = generateJSON(stored, extensions) as {
      content: Array<{ type: string; attrs: Record<string, unknown>; content?: unknown[] }>
    }

    expect(json.content).toHaveLength(1)
    expect(json.content[0].type).toBe('collapsibleBlock')
    expect(json.content[0].attrs.summary).toBe('剧透警告')
    expect(json.content[0].attrs.open).toBe(false)
    expect(json.content[0].content).toEqual([
      { type: 'paragraph', content: [{ type: 'text', text: '隐藏的正文' }] },
    ])
  })

  it('serializes back to native details/summary markup that readers can toggle', () => {
    const html = generateHTML(generateJSON(stored, extensions), extensions)

    expect(html).toContain('<details')
    expect(html).toContain('rpbox-collapse')
    expect(html).toContain('<summary')
    expect(html).toContain('剧透警告')
    expect(html).toContain(COLLAPSE_BODY_CLASS)
    expect(html).toContain('隐藏的正文')
    expect(html).not.toMatch(/<details[^>]*\bopen\b/)
  })

  it('round-trips a block that readers should see expanded by default', () => {
    const json = generateJSON(
      `<details class="rpbox-collapse" open><summary>配置说明</summary><div class="${COLLAPSE_BODY_CLASS}"><p>默认可见</p></div></details>`,
      extensions,
    ) as { content: Array<{ attrs: Record<string, unknown> }> }
    expect(json.content[0].attrs.open).toBe(true)

    const html = generateHTML(json, extensions)
    expect(html).toMatch(/<details[^>]*\bopen\b/)
    expect(html).toContain('默认可见')
  })

  it('falls back to the default title when the summary is missing', () => {
    const json = generateJSON(
      `<details><div class="${COLLAPSE_BODY_CLASS}"><p>没有标题</p></div></details>`,
      extensions,
    ) as { content: Array<{ attrs: Record<string, unknown> }> }
    expect(json.content[0].attrs.summary).toBe(DEFAULT_COLLAPSE_SUMMARY)
  })

  it('inserts after the current paragraph and puts the caret inside the new body', () => {
    const editor = new Editor({ extensions, content: '<p>正文</p>' })
    const paragraphEnd = editor.state.doc.firstChild!.nodeSize
    editor.commands.setTextSelection(paragraphEnd - 1)

    const inserted = editor.commands.insertCollapsibleBlock()
    expect(inserted).toBe(true)

    const names = Array.from(
      { length: editor.state.doc.childCount },
      (_, index) => editor.state.doc.child(index).type.name,
    )
    expect(names.slice(0, 2)).toEqual(['paragraph', 'collapsibleBlock'])

    const { $from } = editor.state.selection
    expect($from.parent.type.name).toBe('paragraph')
    expect($from.parent.textContent).toBe('')
    expect($from.node($from.depth - 1).type.name).toBe('collapsibleBlock')

    editor.destroy()
  })
})
