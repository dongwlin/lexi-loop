import type { ReviewMode, ReviewResult } from './review-state-machine'

export type ReviewKeyAction =
  'next' | 'answerForgotten' | 'answerRemembered' | 'correctForgotten'

export interface ReviewKeyEvent {
  key: string
  code?: string
}

/** 控件上的 Space / Enter 保留原生激活；释义阶段不允许向上修正。 */
export function resolveReviewKeyAction(
  event: ReviewKeyEvent,
  mode: ReviewMode,
  onControl: boolean,
  pendingResult: ReviewResult | null = null,
): ReviewKeyAction | null {
  if (mode === 'idle') return null
  if (event.key === ' ' || event.code === 'Space' || event.key === 'Enter') {
    return mode === 'revealed' && !onControl ? 'next' : null
  }
  if (event.key === '1' || event.key === 'ArrowLeft') {
    if (mode === 'recalling') return 'answerForgotten'
    return pendingResult === 'remembered' ? 'correctForgotten' : null
  }
  if (mode === 'recalling' && (event.key === '2' || event.key === 'ArrowRight'))
    return 'answerRemembered'
  return null
}
