import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it } from 'vitest'
import TiptapEditor from './TiptapEditor.vue'

let wrapper: VueWrapper | undefined

function mountEditor(content = '<p>正文</p>') {
  wrapper = mount(TiptapEditor, {
    props: { modelValue: content },
    global: { plugins: [createPinia()] },
    attachTo: document.body,
  })
  return wrapper
}

function lastHtml(target: VueWrapper) {
  const events = target.emitted('update:modelValue') ?? []
  return String(events.at(-1)?.[0] ?? '')
}

function findCollapseButton(target: VueWrapper) {
  return target.findAll('.toolbar button').find(button => button.attributes('title') === '折叠区块')
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('TiptapEditor collapsible block', () => {
  it('inserts a reader-toggleable block from the toolbar', async () => {
    const editor = mountEditor()
    await flushPromises()

    const button = findCollapseButton(editor)
    expect(button).toBeTruthy()

    await button!.trigger('click')
    await flushPromises()

    expect(editor.findAll('.collapse-editor')).toHaveLength(1)
    expect((editor.get('.collapse-editor__title').element as HTMLInputElement).value).toBe('折叠内容')

    const html = lastHtml(editor)
    expect(html).toContain('<details')
    expect(html).toContain('<summary')
    expect(html).toContain('折叠内容')
    expect(html).toContain('rpbox-collapse__body')
    expect(html).toContain('正文')
  })

  it('lets the author rename the block header', async () => {
    const editor = mountEditor()
    await flushPromises()
    await findCollapseButton(editor)!.trigger('click')
    await flushPromises()

    const input = editor.get('.collapse-editor__title')
    await input.setValue('剧透警告')
    await flushPromises()

    expect(lastHtml(editor)).toContain('<summary class="rpbox-collapse__summary">剧透警告</summary>')
  })

  it('falls back to the default title when the author clears the header', async () => {
    const editor = mountEditor()
    await flushPromises()
    await findCollapseButton(editor)!.trigger('click')
    await flushPromises()

    const input = editor.get('.collapse-editor__title')
    await input.setValue('')
    await flushPromises()

    // 输入框可以临时为空，写出的正文仍然带默认标题
    expect((input.element as HTMLInputElement).value).toBe('')
    expect(lastHtml(editor)).toContain('折叠内容')

    await input.trigger('blur')
    await flushPromises()
    expect((input.element as HTMLInputElement).value).toBe('折叠内容')
  })

  it('defaults to collapsed for readers and can be switched to expanded', async () => {
    const editor = mountEditor()
    await flushPromises()
    await findCollapseButton(editor)!.trigger('click')
    await flushPromises()

    expect(lastHtml(editor)).not.toMatch(/<details[^>]*\bopen\b/)

    await editor.get('.collapse-editor__toggle').trigger('click')
    await flushPromises()

    expect(lastHtml(editor)).toMatch(/<details[^>]*\bopen\b/)
  })

  it('renders an existing stored block back into the editor', async () => {
    const editor = mountEditor(
      '<details class="rpbox-collapse"><summary class="rpbox-collapse__summary">旧内容</summary><div class="rpbox-collapse__body"><p>已保存的正文</p></div></details>',
    )
    await flushPromises()

    expect(editor.findAll('.collapse-editor')).toHaveLength(1)
    expect((editor.get('.collapse-editor__title').element as HTMLInputElement).value).toBe('旧内容')
    expect(editor.get('.collapse-editor__body').text()).toContain('已保存的正文')
  })
})
