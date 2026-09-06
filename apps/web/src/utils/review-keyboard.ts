// 复习区域键盘映射（review-flow.md §10）：先看释义、后作答——
// recalling 下 Space 揭示释义；revealed 下 1 / ← 不记得、2 / → 记得；idle 无快捷键。
// 纯映射不触碰 DOM 与 preventDefault：页面（pages/review/ReviewPage.vue）把事件归约为
// {key, code} 与「焦点是否在控件上」的布尔后调用，返回非 null 才拦截默认行为并执行操作。

import type { ReviewMode } from './review-state-machine'

/** 键盘事件映射出的复习操作。 */
export type ReviewKeyAction = 'reveal' | 'answerForgotten' | 'answerRemembered'

/** 键盘映射关心的事件字段（event.key / event.code 的归约）。 */
export interface ReviewKeyEvent {
  key: string
  code?: string
}

/**
 * 把复习区域内的按键解析为操作；返回 null 表示非快捷键（或焦点在控件上的 Space），
 * 页面不拦截默认行为。
 * 焦点落在按钮 / 链接等控件上时只有 Space 让位原生激活（1 / 2 不是原生激活键，仍作答）；
 * revealed 下 Space 映射为 reveal 但揭示迁移无效，由页面的迁移守卫吸收（仍拦截避免滚动）。
 */
export function resolveReviewKeyAction(
  event: ReviewKeyEvent,
  mode: ReviewMode,
  onControl: boolean,
): ReviewKeyAction | null {
  if (mode === 'idle') return null

  if (event.key === ' ' || event.code === 'Space') {
    return onControl ? null : 'reveal'
  }

  if (mode !== 'revealed') return null
  if (event.key === '1' || event.key === 'ArrowLeft') return 'answerForgotten'
  if (event.key === '2' || event.key === 'ArrowRight') return 'answerRemembered'
  return null
}
