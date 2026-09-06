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
