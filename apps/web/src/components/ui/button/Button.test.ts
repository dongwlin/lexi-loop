import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import Button from './Button.vue'

describe('Button', () => {
  it('渲染默认 primary 变体与 slot 文案，type 默认 button', () => {
    render(Button, { slots: { default: '开始复习' } })

    const button = screen.getByRole('button', { name: '开始复习' })
    expect(button).toBeInTheDocument()
    expect(button).toHaveAttribute('type', 'button')
    expect(button).toBeEnabled()
  })

  it('点击透传 onClick 回调', async () => {
    const onClick = vi.fn()
    render(Button, {
      attrs: { onClick },
      slots: { default: '保存' },
    })

    await userEvent.click(screen.getByRole('button', { name: '保存' }))
    expect(onClick).toHaveBeenCalledOnce()
  })

  it('pending 态标记 aria-busy 并禁用按钮阻止点击', async () => {
    const onClick = vi.fn()
    render(Button, {
      props: { pending: true },
      attrs: { onClick },
      slots: { default: '提交' },
    })

    const button = screen.getByRole('button', { name: /提交/ })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('aria-busy', 'true')
    // Pending 只因原生 disabled 被禁用，视觉上必须保留可读文案：整体降透明度会把白字
    // 一起合成掉（浅色实心主按钮仅 1.51:1），因此 opacity-50 只作用于真正 disabled 的
    // 按钮（设计系统方案 §8.1「Pending 保留按钮宽度与可读文案」）。
    expect(button.className).toContain('disabled:not-aria-busy:opacity-50')
    expect(button.className).not.toMatch(/(?:^|\s)disabled:opacity-50(?:\s|$)/)

    await userEvent.click(button)
    expect(onClick).not.toHaveBeenCalled()
  })

  it('disabled 态同样阻止点击且不带 aria-busy', async () => {
    const onClick = vi.fn()
    render(Button, {
      props: { disabled: true },
      attrs: { onClick },
      slots: { default: '删除' },
    })

    const button = screen.getByRole('button', { name: '删除' })
    expect(button).toBeDisabled()
    expect(button).not.toHaveAttribute('aria-busy')

    await userEvent.click(button)
    expect(onClick).not.toHaveBeenCalled()
  })
})
