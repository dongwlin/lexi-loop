import { describe, expect, it } from 'vitest'
import { resolveReviewKeyAction } from './review-keyboard'

describe('resolveReviewKeyAction', () => {
  it.each(['1', '2', 'ArrowLeft', 'ArrowRight', ' ', 'Enter'])(
    'idle 不处理 %s',
    (key) => {
      expect(resolveReviewKeyAction({ key }, 'idle', false)).toBeNull()
    },
  )
  it.each([
    ['1', 'answerForgotten'],
    ['ArrowLeft', 'answerForgotten'],
    ['2', 'answerRemembered'],
    ['ArrowRight', 'answerRemembered'],
  ])('未揭示时 %s 初选', (key, action) => {
    expect(resolveReviewKeyAction({ key }, 'recalling', false)).toBe(action)
    expect(resolveReviewKeyAction({ key }, 'recalling', true)).toBe(action)
  })
  it.each([' ', 'Enter'])('揭示后 %s 下一词，控件保留原生激活', (key) => {
    expect(resolveReviewKeyAction({ key }, 'revealed', false)).toBe('next')
    expect(resolveReviewKeyAction({ key }, 'revealed', true)).toBeNull()
    expect(resolveReviewKeyAction({ key }, 'recalling', false)).toBeNull()
  })
  it('Space code 兜底', () => {
    expect(
      resolveReviewKeyAction({ key: 'x', code: 'Space' }, 'revealed', false),
    ).toBe('next')
  })
  it.each(['1', 'ArrowLeft'])('%s 仅向下修正认识', (key) => {
    expect(
      resolveReviewKeyAction({ key }, 'revealed', false, 'remembered'),
    ).toBe('correctForgotten')
    expect(
      resolveReviewKeyAction({ key }, 'revealed', false, 'forgotten'),
    ).toBeNull()
    expect(resolveReviewKeyAction({ key }, 'revealed', false)).toBeNull()
  })
  it.each(['2', 'ArrowRight', 'Tab', 'ArrowUp', 'a'])(
    '揭示后 %s 不映射',
    (key) => {
      expect(
        resolveReviewKeyAction({ key }, 'revealed', false, 'remembered'),
      ).toBeNull()
    },
  )
})
