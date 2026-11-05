import { mergeAttributes, Node } from '@tiptap/core'
import { VueNodeViewRenderer } from '@tiptap/vue-3'
import CollapsibleBlockView from './CollapsibleBlockView.vue'
import { COLLAPSE_BODY_CLASS, DEFAULT_COLLAPSE_SUMMARY } from './collapseBlockConstants'

export { COLLAPSE_BODY_CLASS, DEFAULT_COLLAPSE_SUMMARY }

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    collapsibleBlock: {
      /** 在光标处插入一个折叠区块，读者可点击标题展开/收起。 */
      insertCollapsibleBlock: (attrs?: { summary?: string; open?: boolean }) => ReturnType
    }
  }
}

/**
 * 读者可展开/收起的折叠区块，落库为原生 `<details>`，
 * 因此客户端、Web 和手机端都能在没有脚本的情况下正常工作。
 */
export const CollapsibleBlockExtension = Node.create({
  name: 'collapsibleBlock',
  group: 'block',
  content: 'block+',
  defining: true,

  addAttributes() {
    return {
      summary: {
        default: DEFAULT_COLLAPSE_SUMMARY,
        parseHTML: (element: HTMLElement) => element.querySelector('summary')?.textContent?.trim() || DEFAULT_COLLAPSE_SUMMARY,
        renderHTML: () => ({}),
      },
      open: {
        default: false,
        parseHTML: (element: HTMLElement) => element.hasAttribute('open'),
        renderHTML: (attributes: Record<string, unknown>) => (attributes.open ? { open: '' } : {}),
      },
    }
  },

  parseHTML() {
    return [
      {
        tag: 'details',
        contentElement: `div.${COLLAPSE_BODY_CLASS}`,
      },
    ]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'details',
      mergeAttributes(HTMLAttributes, { class: 'rpbox-collapse' }),
      ['summary', { class: 'rpbox-collapse__summary' }, String(node.attrs.summary || DEFAULT_COLLAPSE_SUMMARY)],
      ['div', { class: COLLAPSE_BODY_CLASS }, 0],
    ]
  },

  addNodeView() {
    return VueNodeViewRenderer(CollapsibleBlockView)
  },

  addCommands() {
    return {
      insertCollapsibleBlock:
        (attrs = {}) =>
        ({ chain, state }) => {
          const { $from } = state.selection
          // 插到当前段落之后，这样有选中文字时也不会被覆盖
          const insertPos = $from.depth > 0 ? $from.after($from.depth) : state.doc.content.size
          return chain()
            .insertContentAt(insertPos, {
              type: 'collapsibleBlock',
              attrs: { summary: DEFAULT_COLLAPSE_SUMMARY, ...attrs },
              content: [{ type: 'paragraph' }],
            })
            // 光标落到新折叠区块的正文里，作者可以直接输入
            .setTextSelection(insertPos + 2)
            .focus()
            .run()
        },
    }
  },
})
