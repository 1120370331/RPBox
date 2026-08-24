import { createApp, defineComponent, h, nextTick, reactive, type App } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it } from 'vitest'
import enCommon from '@/i18n/locales/en-US/common'
import zhCommon from '@/i18n/locales/zh-CN/common'
import SafetyReportSheet from './SafetyReportSheet.vue'

type TargetType =
  | 'post'
  | 'item'
  | 'user'
  | 'comment'
  | 'item_comment'
  | 'rpdb_comment'
  | 'story'
  | 'rpdb_work'
  | 'character_card'
  | 'guild'

interface SheetProps {
  open: boolean
  targetType?: TargetType
  initialAction?: 'default' | 'report' | 'block'
  submitting?: boolean
}

interface SubmitPayload {
  reason: string
  detail: string
  hideTarget: boolean
  blockAuthor: boolean
  submitReport: boolean
}

let app: App<Element> | null = null

async function flushUi() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

function mountSheet(initialProps: SheetProps, initialLocale: 'zh-CN' | 'en-US' = 'zh-CN') {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const props = reactive<SheetProps>({ ...initialProps })
  const submissions: SubmitPayload[] = []
  const i18n = createI18n({
    legacy: false,
    locale: initialLocale,
    fallbackLocale: 'zh-CN',
    messages: {
      'zh-CN': { common: zhCommon },
      'en-US': { common: enCommon },
    },
  })

  const Root = defineComponent({
    setup() {
      return () => h(SafetyReportSheet, {
        ...props,
        onSubmit: (payload: SubmitPayload) => submissions.push(payload),
      })
    },
  })

  app = createApp(Root)
  app.use(i18n)
  app.mount(host)

  return {
    props,
    submissions,
    setLocale(locale: 'zh-CN' | 'en-US') {
      i18n.global.locale.value = locale
    },
  }
}

afterEach(() => {
  app?.unmount()
  app = null
  document.body.innerHTML = ''
})

describe('SafetyReportSheet localization and behavior', () => {
  it('reactively localizes dialog semantics, reasons, actions, hints, and new target labels', async () => {
    const harness = mountSheet({
      open: true,
      targetType: 'character_card',
      initialAction: 'report',
    })
    await flushUi()

    const dialog = document.querySelector<HTMLElement>('.sheet-panel')!
    const title = dialog.querySelector<HTMLHeadingElement>('h3')!
    const close = dialog.querySelector<HTMLButtonElement>('.sheet-close')!
    expect(dialog.getAttribute('role')).toBe('dialog')
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(dialog.getAttribute('aria-labelledby')).toBe(title.id)
    expect(title.textContent).toBe('举报人物卡')
    expect(close.getAttribute('aria-label')).toBe('关闭举报与屏蔽面板')
    expect(dialog.textContent).toContain('隐藏这张人物卡')
    expect(dialog.textContent).toContain('同时提交给版主审核')
    expect(dialog.textContent).toContain('垃圾信息或刷屏')
    expect(dialog.textContent).toContain('同时举报需要选择原因并填写备注说明。')
    expect(dialog.textContent).toContain('提交举报')
    expect(dialog.querySelector('textarea')?.placeholder).toBe('请填写备注说明')

    harness.setLocale('en-US')
    await flushUi()

    expect(title.textContent).toBe('Report character card')
    expect(close.getAttribute('aria-label')).toBe('Close report and block panel')
    expect(dialog.textContent).toContain('Hide this character card')
    expect(dialog.textContent).toContain('Also submit to moderators for review')
    expect(dialog.textContent).toContain('Spam or flooding')
    expect(dialog.textContent).toContain('To also report this content, select a reason and add details.')
    expect(dialog.textContent).toContain('Submit report')
    expect(dialog.querySelector('textarea')?.placeholder).toBe('Describe the issue')
    expect(dialog.textContent).not.toContain('垃圾信息或刷屏')

    harness.props.targetType = 'guild'
    await flushUi()
    expect(title.textContent).toBe('Report guild')
    expect(dialog.textContent).toContain('Hide this guild')
  })

  it('requires report details and preserves the trimmed moderator payload', async () => {
    const harness = mountSheet({
      open: true,
      targetType: 'post',
      initialAction: 'report',
    }, 'en-US')
    await flushUi()

    const submitButton = document.querySelector<HTMLButtonElement>('.sheet-btn.primary')!
    expect(submitButton.disabled).toBe(true)
    submitButton.click()
    expect(harness.submissions).toEqual([])

    const reason = document.querySelector<HTMLSelectElement>('.sheet-field select')!
    reason.value = 'abuse'
    reason.dispatchEvent(new Event('change', { bubbles: true }))
    const detail = document.querySelector<HTMLTextAreaElement>('.sheet-field textarea')!
    detail.value = '  Repeated personal attacks  '
    detail.dispatchEvent(new Event('input', { bubbles: true }))
    await flushUi()

    expect(submitButton.disabled).toBe(false)
    submitButton.click()
    submitButton.click()
    expect(harness.submissions).toEqual([{
      reason: 'abuse',
      detail: 'Repeated personal attacks',
      hideTarget: false,
      blockAuthor: false,
      submitReport: true,
    }])

    harness.props.submitting = true
    await flushUi()
    submitButton.click()
    expect(harness.submissions).toHaveLength(1)
  })

  it('keeps default local hide and user block actions available without a report', async () => {
    const harness = mountSheet({ open: true, targetType: 'item' }, 'en-US')
    await flushUi()

    const submitButton = document.querySelector<HTMLButtonElement>('.sheet-btn.primary')!
    expect(submitButton.disabled).toBe(false)
    expect(document.querySelector('.sheet-hint')?.textContent)
      .toBe('Hiding and blocking apply only to you and are not sent to moderators.')
    submitButton.click()
    expect(harness.submissions).toEqual([{
      reason: 'spam',
      detail: '',
      hideTarget: true,
      blockAuthor: false,
      submitReport: false,
    }])

    harness.props.open = false
    await flushUi()
    harness.props.targetType = 'user'
    harness.props.open = true
    await flushUi()
    document.querySelector<HTMLButtonElement>('.sheet-btn.primary')!.click()
    expect(harness.submissions[1]).toEqual({
      reason: 'spam',
      detail: '',
      hideTarget: false,
      blockAuthor: true,
      submitReport: false,
    })
  })
})
