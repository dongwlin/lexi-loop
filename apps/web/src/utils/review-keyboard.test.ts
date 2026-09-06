import { describe, expect, it } from 'vitest'

import { resolveReviewKeyAction } from './review-keyboard'

describe('resolveReviewKeyAction', () => {
  it('idle 无任何快捷键（数量选择阶段不拦截按键）', () => {
    expect(resolveReviewKeyAction({ key: ' ' }, 'idle', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: '1' }, 'idle', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'ArrowRight' }, 'idle', false)).toBeNull()
  })

  it('recalling 下 Space 揭示释义', () => {
    expect(resolveReviewKeyAction({ key: ' ' }, 'recalling', false)).toBe('reveal')
  })

  it('key 非 " " 时以 code === "Space" 兜底', () => {
    expect(resolveReviewKeyAction({ key: 'x', code: 'Space' }, 'recalling', false)).toBe('reveal')
  })

  it('焦点在控件上时 Space 让位原生激活，不映射', () => {
    expect(resolveReviewKeyAction({ key: ' ' }, 'recalling', true)).toBeNull()
    expect(resolveReviewKeyAction({ key: ' ', code: 'Space' }, 'revealed', true)).toBeNull()
  })

  it('revealed 下 1 / ← 不记得，2 / → 记得', () => {
    expect(resolveReviewKeyAction({ key: '1' }, 'revealed', false)).toBe('answerForgotten')
    expect(resolveReviewKeyAction({ key: 'ArrowLeft' }, 'revealed', false)).toBe('answerForgotten')
    expect(resolveReviewKeyAction({ key: '2' }, 'revealed', false)).toBe('answerRemembered')
    expect(resolveReviewKeyAction({ key: 'ArrowRight' }, 'revealed', false)).toBe('answerRemembered')
  })

  it('1 / 2 不是原生激活键，焦点在控件上仍作答', () => {
    expect(resolveReviewKeyAction({ key: '1' }, 'revealed', true)).toBe('answerForgotten')
    expect(resolveReviewKeyAction({ key: '2' }, 'revealed', true)).toBe('answerRemembered')
  })

  it('recalling 下不能未揭示就作答', () => {
    expect(resolveReviewKeyAction({ key: '1' }, 'recalling', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'ArrowLeft' }, 'recalling', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: '2' }, 'recalling', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'ArrowRight' }, 'recalling', false)).toBeNull()
  })

  it('revealed 下 Space 仍映射为 reveal（揭示迁移无效由页面守卫吸收，拦截避免滚动）', () => {
    expect(resolveReviewKeyAction({ key: ' ' }, 'revealed', false)).toBe('reveal')
  })

  it('其它按键不映射（Enter / Tab / 方向键上下 / 普通字符）', () => {
    expect(resolveReviewKeyAction({ key: 'Enter' }, 'recalling', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'Tab' }, 'revealed', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'ArrowUp' }, 'revealed', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: '3' }, 'revealed', false)).toBeNull()
    expect(resolveReviewKeyAction({ key: 'a' }, 'revealed', false)).toBeNull()
  })
})
