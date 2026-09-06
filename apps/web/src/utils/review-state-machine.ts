// 复习页状态机的纯迁移逻辑（review-flow.md §10）：
// idle → recalling → revealed → answered → recalling → ... → finished（跳转结果页）。
// 页面（pages/review/ReviewPage.vue）持有 ref 状态并承担副作用（提交 API / 焦点 / 路由），
// 迁移判定与推进计算收敛在这里，非法迁移（recalling 直接作答、revealed 重复揭示等）不发生。

/** 复习页闪卡模式：idle 为数量选择阶段，recalling 只见单词，revealed 显示释义待作答。 */
export type ReviewMode = 'idle' | 'recalling' | 'revealed'

/** 进入一轮复习的起点：模式与当前卡下标。 */
export interface ReviewCardEntry {
  mode: ReviewMode
  currentIndex: number
}

/**
 * 进入一轮复习：有卡时从 resumeIndex（钳制到 [0, itemCount - 1]，缺省 0）以 recalling 进入；
 * 空轮（防御分支）保持 idle、currentIndex -1（无当前卡）。
 * 开始复习与「继续复习」恢复（D010）共用：恢复进度对齐失败回退第 1 题（review-flow.md §6）。
 */
export function enterReview(
  itemCount: number,
  resumeIndex = 0,
): ReviewCardEntry {
  if (itemCount <= 0) return { mode: 'idle', currentIndex: -1 }
  return {
    mode: 'recalling',
    currentIndex: Math.min(Math.max(resumeIndex, 0), itemCount - 1),
  }
}

/**
 * 揭示释义迁移：仅 recalling 且当前卡在界时迁入 revealed；
 * 其余（idle、已 revealed、无当前卡）返回 null 表示不迁移，页面保持原状态。
 */
export function revealCard(
  mode: ReviewMode,
  hasCurrentCard: boolean,
): 'revealed' | null {
  return mode === 'recalling' && hasCurrentCard ? 'revealed' : null
}

/** 作答成功后的推进：进入下一张卡（recalling），或最后一张答完完成本轮（页面跳转结果页）。 */
export type ReviewAdvance =
  | { outcome: 'next'; nextIndex: number; mode: 'recalling' }
  | { outcome: 'finished' }

/**
 * 第 currentIndex 张卡作答成功后的推进（review-flow.md §10：成功后 index++ 进入下一个
 * recalling，最后一个提交完 → finished）。前提是调用方已确认处于 revealed 且提交成功；
 * 作答的入口守卫（revealed / 非提交中 / 有 session）在页面侧。
 */
export function advanceAfterAnswer(
  currentIndex: number,
  itemCount: number,
): ReviewAdvance {
  const nextIndex = currentIndex + 1
  if (nextIndex < itemCount) {
    return { outcome: 'next', nextIndex, mode: 'recalling' }
  }
  return { outcome: 'finished' }
}
